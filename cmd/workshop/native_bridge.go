package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/openwaldo/waldo/internal/config"
	"github.com/openwaldo/waldo/internal/inference"
	"github.com/openwaldo/waldo/internal/model"
)

const nativeWaldoBackend = "native-waldo"

type nativeChatMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type nativeChatRequest struct {
	Model    string              `json:"model"`
	Messages []nativeChatMessage `json:"messages"`
}

type nativeWaldoSession struct {
	mu     sync.Mutex
	opened inference.Opened
}

var nativeWaldoSessions = struct {
	sync.Mutex
	values map[string]*nativeWaldoSession
}{values: map[string]*nativeWaldoSession{}}

func init() {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("AXM_AI_BACKEND")))
	if mode == "" {
		if strings.TrimSpace(os.Getenv("AXM_AI_BASE_URL")) != "" {
			return
		}
		mode = nativeWaldoBackend
	}
	if mode != nativeWaldoBackend {
		return
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("start native WALDO workshop bridge: %v", err))
	}
	base := "http://" + listener.Addr().String() + "/v1"
	_ = os.Setenv("AXM_AI_BASE_URL", base)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", nativeWaldoChat)
	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
}

func nativeWaldoChat(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request nativeChatRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&request); err != nil {
		nativeBridgeError(w, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(request.Model)
	if override := strings.TrimSpace(os.Getenv("AXM_WALDO_MODEL")); override != "" {
		name = override
	}
	if name == "" {
		name = "waldo"
	}
	prompt, err := nativePrompt(request.Messages)
	if err != nil {
		nativeBridgeError(w, http.StatusBadRequest, err)
		return
	}
	opened, err := nativeSession(name)
	if err != nil {
		nativeBridgeError(w, http.StatusBadGateway, err)
		return
	}

	opened.mu.Lock()
	result, err := opened.opened.Session.Generate(r.Context(), prompt, inference.Options{
		MaxTokens:   512,
		Temperature: 0.7,
		TopP:        0.95,
	}, nil)
	opened.mu.Unlock()
	if err != nil {
		nativeBridgeError(w, http.StatusBadGateway, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"choices": []map[string]any{{
			"message":       map[string]any{"role": "assistant", "content": result.Text},
			"finish_reason": result.FinishReason,
		}},
	})
}

func nativeSession(name string) (*nativeWaldoSession, error) {
	nativeWaldoSessions.Lock()
	defer nativeWaldoSessions.Unlock()
	if existing := nativeWaldoSessions.values[name]; existing != nil {
		return existing, nil
	}
	configuration, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load WALDO config: %w", err)
	}
	root, err := config.EffectiveModelRoot(configuration)
	if err != nil {
		return nil, fmt.Errorf("resolve WALDO model root: %w", err)
	}
	inspection, err := model.Inspect(root, name)
	if err != nil {
		return nil, fmt.Errorf("inspect WALDO model %q: %w", name, err)
	}
	opened, err := inference.OpenAXMHybrid(context.Background(), inspection)
	if err != nil {
		return nil, fmt.Errorf("open WALDO native inference %q: %w", name, err)
	}
	value := &nativeWaldoSession{opened: opened}
	nativeWaldoSessions.values[name] = value
	return value, nil
}

func nativePrompt(messages []nativeChatMessage) (string, error) {
	if len(messages) == 0 {
		return "", errors.New("native WALDO request has no messages")
	}
	var out strings.Builder
	for _, message := range messages {
		text, hasImage, err := nativeMessageText(message.Content)
		if err != nil {
			return "", err
		}
		if hasImage {
			return "", errors.New("native WALDO bridge is text-only today; the image remains in the Creative Room, but WALDO cannot truthfully inspect it yet")
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		role := strings.ToLower(strings.TrimSpace(message.Role))
		if role == "" {
			role = "user"
		}
		fmt.Fprintf(&out, "[%s]\n%s\n\n", role, text)
	}
	prompt := strings.TrimSpace(out.String())
	if prompt == "" {
		return "", errors.New("native WALDO request contains no text")
	}
	return prompt, nil
}

func nativeMessageText(raw json.RawMessage) (string, bool, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, false, nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", false, errors.New("unsupported native WALDO message content")
	}
	var out strings.Builder
	hasImage := false
	for _, part := range parts {
		switch part.Type {
		case "text":
			if strings.TrimSpace(part.Text) != "" {
				if out.Len() > 0 {
					out.WriteByte('\n')
				}
				out.WriteString(part.Text)
			}
		case "image_url":
			hasImage = true
		}
	}
	return out.String(), hasImage, nil
}

func nativeBridgeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"message": err.Error()},
	})
}
