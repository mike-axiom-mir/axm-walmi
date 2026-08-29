package inference

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/openwaldo/waldo/internal/axmmirror"
	"github.com/openwaldo/waldo/internal/model"
)

const (
	AXMHybridRuntimeVersion = "v0.35"
	AXMHybridModeEnv        = "WALDO_AXM_HYBRID"
	AXMHybridGroundingEnv   = "WALDO_AXM_GROUNDING_FILE"
	AXMHybridTraceEnv       = "WALDO_AXM_HYBRID_TRACE"
)

type AXMHybridTrace struct {
	Schema                string      `json:"schema"`
	Runtime               string      `json:"runtime"`
	Mode                  string      `json:"mode"`
	Model                 string      `json:"model"`
	Backend               string      `json:"backend"`
	GroundingSHA256       string      `json:"grounding_sha256"`
	PromptSHA256          string      `json:"prompt_sha256"`
	NeuralOutputSHA256    string      `json:"neural_output_sha256,omitempty"`
	DelegateCalled        bool        `json:"delegate_called"`
	PromptPreservedExact  bool        `json:"prompt_preserved_exact"`
	GroundingAppliedAt    string      `json:"grounding_applied_at"`
	DeterministicFallback string      `json:"deterministic_fallback"`
	Authority             string      `json:"authority"`
	Timestamp             time.Time   `json:"timestamp"`
	Source                Description `json:"source"`
}

type AXMHybridSession struct {
	delegate    Session
	description Description
	grounding   axmmirror.LiveNeuralGrounding
	tracePath   string
	mu          sync.Mutex
	lastTrace   AXMHybridTrace
}

func OpenAXMHybrid(ctx context.Context, inspection model.Inspection) (Opened, error) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv(AXMHybridModeEnv)))
	if mode == "raw" || mode == "off" || mode == "0" || mode == "false" {
		return Open(ctx, inspection)
	}
	grounding, err := loadAXMHybridGrounding(os.Getenv(AXMHybridGroundingEnv))
	if err != nil {
		return Opened{}, err
	}
	opened, err := Open(ctx, inspection)
	if err != nil {
		return Opened{}, err
	}
	wrapped := NewAXMHybridSession(opened.Session, opened.Description, grounding, os.Getenv(AXMHybridTraceEnv))
	opened.Session = wrapped
	opened.Description.Backend += "+axm-hybrid-v0.35"
	return opened, nil
}

func NewAXMHybridSession(delegate Session, description Description, grounding axmmirror.LiveNeuralGrounding, tracePath string) *AXMHybridSession {
	return &AXMHybridSession{delegate: delegate, description: description, grounding: grounding, tracePath: strings.TrimSpace(tracePath)}
}

func (session *AXMHybridSession) Generate(ctx context.Context, prompt string, options Options, emit func(Token) error) (Result, error) {
	if session.delegate == nil {
		return Result{}, fmt.Errorf("AXM hybrid session has no neural delegate")
	}
	if err := options.Validate(); err != nil {
		return Result{}, err
	}
	_, groundingDigest, err := session.grounding.Capsule()
	if err != nil {
		return Result{}, fmt.Errorf("build AXM grounding capsule: %w", err)
	}
	trace := AXMHybridTrace{
		Schema:                "axm.waldo.live-neural-hybrid-trace/v0.35",
		Runtime:               AXMHybridRuntimeVersion,
		Mode:                  "NEURAL_PRIMARY_DETERMINISTIC_RESPONSE_GROUNDING",
		Model:                 session.description.Model,
		Backend:               session.description.Backend,
		GroundingSHA256:       groundingDigest,
		PromptSHA256:          digestText(prompt),
		DelegateCalled:        true,
		PromptPreservedExact:  true,
		GroundingAppliedAt:    "RESPONSE_BOUNDARY",
		DeterministicFallback: "NONE",
		Authority:             "NONE",
		Timestamp:             time.Now().UTC(),
		Source:                session.description,
	}

	if session.grounding.ShouldDeterministicallyHold() {
		result, err := session.delegate.Generate(ctx, prompt, options, nil)
		if err != nil {
			return Result{}, err
		}
		trace.NeuralOutputSHA256 = digestText(result.Text)
		trace.Mode = "NEURAL_CANDIDATE_WITH_DETERMINISTIC_FALLBACK_HOLD"
		trace.DeterministicFallback = "HOLD_UNRESOLVED_HIGH_CONSEQUENCE"
		if err := session.recordTrace(trace); err != nil {
			return Result{}, err
		}
		message := "AXM grounding hold: HIGH-consequence response withheld because grounding is unresolved; the neural draft remains a non-authoritative candidate."
		if emit != nil {
			if err := emit(Token{Bytes: []byte(message)}); err != nil {
				return Result{}, err
			}
		}
		return Result{Text: message, Tokens: 0, FinishReason: "axm-grounding-hold", Duration: result.Duration, DurationMS: result.DurationMS}, nil
	}

	result, err := session.delegate.Generate(ctx, prompt, options, func(token Token) error {
		if emit == nil {
			return nil
		}
		return emit(token)
	})
	if err != nil {
		return Result{}, err
	}
	trace.NeuralOutputSHA256 = digestText(result.Text)
	if err := session.recordTrace(trace); err != nil {
		return Result{}, err
	}
	return result, nil
}

func (session *AXMHybridSession) Close() error {
	if session.delegate == nil {
		return nil
	}
	return session.delegate.Close()
}

func (session *AXMHybridSession) LastTrace() AXMHybridTrace {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.lastTrace
}

func (session *AXMHybridSession) recordTrace(trace AXMHybridTrace) error {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.lastTrace = trace
	if session.tracePath == "" {
		return nil
	}
	payload, err := json.Marshal(trace)
	if err != nil {
		return fmt.Errorf("marshal AXM hybrid trace: %w", err)
	}
	payload = append(payload, '\n')
	file, err := os.OpenFile(session.tracePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open AXM hybrid trace %s: %w", session.tracePath, err)
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return fmt.Errorf("secure AXM hybrid trace %s: %w", session.tracePath, err)
	}
	if _, err := file.Write(payload); err != nil {
		_ = file.Close()
		return fmt.Errorf("append AXM hybrid trace %s: %w", session.tracePath, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close AXM hybrid trace %s: %w", session.tracePath, err)
	}
	return nil
}

func loadAXMHybridGrounding(path string) (axmmirror.LiveNeuralGrounding, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return axmmirror.DefaultLiveNeuralGrounding(), nil
	}
	file, err := os.Open(path)
	if err != nil {
		return axmmirror.LiveNeuralGrounding{}, fmt.Errorf("open AXM grounding file %s: %w", path, err)
	}
	defer file.Close()
	grounding, err := axmmirror.LoadLiveNeuralGrounding(file)
	if err != nil {
		return axmmirror.LiveNeuralGrounding{}, fmt.Errorf("load AXM grounding file %s: %w", path, err)
	}
	return grounding, nil
}

func digestText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
