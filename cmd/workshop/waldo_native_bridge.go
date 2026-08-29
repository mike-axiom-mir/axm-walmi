package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/openwaldo/waldo/internal/config"
	"github.com/openwaldo/waldo/internal/inference"
	"github.com/openwaldo/waldo/internal/model"
)

// The workshop UI already speaks a tiny OpenAI-compatible chat shape. When no
// external AXM_AI_BASE_URL is configured, this file supplies that shape locally
// and routes it into WALDO's real model artifacts and AXM hybrid inference
// session. It is a compatibility seam inside the same workshop process, not a
// second AI service the user has to install or manage.
const nativeWaldoDefaultPort = "7789"

type nativeWaldoBridge struct {
	mu        sync.Mutex
	opened    map[string]inference.Opened
	modelRoot string
}

type waldoBridgeChatRequest struct {
	Model    string                   `json:"model"`
	Messages []waldoBridgeChatMessage `json:"messages"`
}

type waldoBridgeChatMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

func init() {
	if strings.TrimSpace(os.Getenv("AXM_AI_BASE_URL")) != "" {
		return
	}
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("AXM_WORKSHOP_NATIVE_WALDO")))
	if mode == "0" || mode == "false" || mode == "off" || mode == "disabled" {
		return
	}
	port := env("AXM_WALDO_BRIDGE_PORT", nativeWaldoDefaultPort)
	base := "http://127.0.0.1:" + port + "/v1"
	_ = os.Setenv("AXM_AI_BASE_URL", base)
	go runNativeWaldoBridge("127.0.0.1:" + port)
}

func runNativeWaldoBridge(addr string) {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("native Waldo bridge unavailable: load WALDO config: %v", err)
		return
	}
	root, err := config.EffectiveModelRoot(cfg)
	if err != nil {
		log.Printf("native Waldo bridge unavailable: resolve model root: %v", err)
		return
	}
	bridge := &nativeWaldoBridge{opened: map[string]inference.Opened{}, modelRoot: root}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "backend": "waldo-native-axm-hybrid", "model_root": root})
	})
	mux.HandleFunc("POST /v1/chat/completions", bridge.chat)
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("native Waldo bridge: http://%s/v1", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("native Waldo bridge stopped: %v", err)
	}
}

func (b *nativeWaldoBridge) chat(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	defer r.Body.Close()
	var request waldoBridgeChatRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		waldoBridgeError(w, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(request.Model)
	if name == "" {
		name = env("AXM_AI_MODEL", "waldo")
	}
	prompt, err := waldoBridgePrompt(request.Messages)
	if err != nil {
		waldoBridgeError(w, http.StatusBadRequest, err)
		return
	}
	opened, err := b.session(name)
	if err != nil {
		waldoBridgeError(w, http.StatusBadGateway, err)
		return
	}
	maxTokens := waldoBridgeEnvInt("AXM_WALDO_MAX_TOKENS", 512, 1, 65536)
	result, err := opened.Session.Generate(r.Context(), prompt, inference.Options{
		MaxTokens:   maxTokens,
		Temperature: 0.4,
		TopP:        0.95,
	}, nil)
	if err != nil {
		waldoBridgeError(w, http.StatusBadGateway, fmt.Errorf("Waldo generation failed: %w", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"model": name,
		"choices": []any{map[string]any{
			"index":         0,
			"message":       map[string]any{"role": "assistant", "content": result.Text},
			"finish_reason": result.FinishReason,
		}},
	})
}

func (b *nativeWaldoBridge) session(name string) (inference.Opened, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if opened, ok := b.opened[name]; ok && opened.Session != nil {
		return opened, nil
	}
	inspection, err := model.Inspect(b.modelRoot, name)
	if err != nil {
		return inference.Opened{}, fmt.Errorf("open local Waldo model %q: %w", name, err)
	}
	// Use a process-lifetime context to open the worker. Individual Generate
	// calls still receive the request context, so cancelling a chat cancels the
	// generation without destroying the cached model session.
	opened, err := inference.OpenAXMHybrid(context.Background(), inspection)
	if err != nil {
		return inference.Opened{}, fmt.Errorf("open Waldo AXM hybrid inference for %q: %w", name, err)
	}
	b.opened[name] = opened
	return opened, nil
}

func waldoBridgePrompt(messages []waldoBridgeChatMessage) (string, error) {
	if len(messages) == 0 {
		return "", errors.New("chat request has no messages")
	}
	var builder strings.Builder
	for _, message := range messages {
		text, err := waldoBridgeMessageText(message.Content)
		if err != nil {
			return "", err
		}
		role := strings.ToUpper(strings.TrimSpace(message.Role))
		if role == "" {
			role = "MESSAGE"
		}
		builder.WriteString("[")
		builder.WriteString(role)
		builder.WriteString("]\n")
		builder.WriteString(strings.TrimSpace(text))
		builder.WriteString("\n\n")
	}
	builder.WriteString("[ASSISTANT]\n")
	return builder.String(), nil
}

func waldoBridgeMessageText(raw json.RawMessage) (string, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", errors.New("unsupported chat message content")
	}
	var builder strings.Builder
	for _, part := range parts {
		switch part.Type {
		case "text":
			if strings.TrimSpace(part.Text) != "" {
				builder.WriteString(part.Text)
				builder.WriteByte('\n')
			}
		case "image_url":
			// The Creative Room still preserves and shares the image. Current WALDO
			// native inference is text generation, so we mark the image visibly
			// instead of pretending the model inspected its pixels.
			builder.WriteString("[Image attached in Creative Room; native Waldo text inference cannot inspect its pixels.]\n")
		}
	}
	return strings.TrimSpace(builder.String()), nil
}

func waldoBridgeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
}

func waldoBridgeEnvInt(name string, fallback, minimum, maximum int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum || parsed > maximum {
		return fallback
	}
	return parsed
}
