package inference

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/openwaldo/waldo/internal/axmmirror"
)

type axmHybridFakeSession struct {
	prompts []string
	result  Result
	closed  bool
}

func (session *axmHybridFakeSession) Generate(_ context.Context, prompt string, _ Options, emit func(Token) error) (Result, error) {
	session.prompts = append(session.prompts, prompt)
	if emit != nil && session.result.Text != "" {
		if err := emit(Token{Bytes: []byte(session.result.Text)}); err != nil {
			return Result{}, err
		}
	}
	return session.result, nil
}

func (session *axmHybridFakeSession) Close() error {
	session.closed = true
	return nil
}

func testHybridOptions() Options {
	return Options{MaxTokens: 64, Temperature: 0, TopP: 1}
}

func TestAXMHybridSessionUsesRealSessionInterfaceWithoutPromptRewrite(t *testing.T) {
	delegate := &axmHybridFakeSession{result: Result{Text: "candidate", Tokens: 1, FinishReason: "stop"}}
	description := Description{Model: "fixture", Backend: "pytorch"}
	session := NewAXMHybridSession(delegate, description, axmmirror.DefaultLiveNeuralGrounding(), "")
	var streamed strings.Builder
	result, err := session.Generate(context.Background(), "rendered-model-prompt", testHybridOptions(), func(token Token) error {
		streamed.Write(token.Bytes)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(delegate.prompts) != 1 || delegate.prompts[0] != "rendered-model-prompt" {
		t.Fatalf("hybrid wrapper rewrote the rendered prompt: %#v", delegate.prompts)
	}
	if result.Text != "candidate" || streamed.String() != "candidate" {
		t.Fatalf("unexpected neural result/stream: %#v %q", result, streamed.String())
	}
	trace := session.LastTrace()
	if !trace.DelegateCalled || !trace.PromptPreservedExact || trace.GroundingAppliedAt != "RESPONSE_BOUNDARY" || trace.NeuralOutputSHA256 == "" || trace.Authority != "NONE" {
		t.Fatalf("unexpected hybrid trace: %#v", trace)
	}
}

func TestAXMHybridSessionHighUnresolvedUsesNeuralCandidateThenDeterministicFallback(t *testing.T) {
	delegate := &axmHybridFakeSession{result: Result{Text: "withheld neural draft", Tokens: 3, FinishReason: "stop"}}
	grounding := axmmirror.DefaultLiveNeuralGrounding()
	grounding.State = axmmirror.GroundingConflict
	grounding.Consequence = axmmirror.ConsequenceHigh
	session := NewAXMHybridSession(delegate, Description{Model: "fixture", Backend: "mlx"}, grounding, "")
	var streamed strings.Builder
	result, err := session.Generate(context.Background(), "rendered-high-consequence-prompt", testHybridOptions(), func(token Token) error {
		streamed.Write(token.Bytes)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(delegate.prompts) != 1 || delegate.prompts[0] != "rendered-high-consequence-prompt" {
		t.Fatalf("neural delegate was not called exactly once with original prompt: %#v", delegate.prompts)
	}
	if strings.Contains(streamed.String(), "withheld neural draft") || strings.Contains(result.Text, "withheld neural draft") {
		t.Fatal("withheld neural candidate leaked through deterministic fallback")
	}
	if result.FinishReason != "axm-grounding-hold" || !strings.Contains(streamed.String(), "grounding hold") {
		t.Fatalf("unexpected deterministic fallback: %#v %q", result, streamed.String())
	}
	trace := session.LastTrace()
	if !trace.DelegateCalled || trace.NeuralOutputSHA256 == "" || trace.DeterministicFallback != "HOLD_UNRESOLVED_HIGH_CONSEQUENCE" {
		t.Fatalf("unexpected hold trace: %#v", trace)
	}
}

func TestAXMHybridSessionLowUncertaintyKeepsNeuralPrimary(t *testing.T) {
	delegate := &axmHybridFakeSession{result: Result{Text: "uncertain candidate", Tokens: 2, FinishReason: "stop"}}
	grounding := axmmirror.DefaultLiveNeuralGrounding()
	grounding.State = axmmirror.GroundingUncertain
	grounding.Consequence = axmmirror.ConsequenceLow
	session := NewAXMHybridSession(delegate, Description{Model: "fixture", Backend: "pytorch"}, grounding, "")
	if _, err := session.Generate(context.Background(), "prompt", testHybridOptions(), nil); err != nil {
		t.Fatal(err)
	}
	if len(delegate.prompts) != 1 || delegate.prompts[0] != "prompt" {
		t.Fatalf("low-consequence uncertainty must keep neural primary without prompt rewrite: %#v", delegate.prompts)
	}
}

func TestAXMHybridTraceIsAppendOnlyJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hybrid-trace.jsonl")
	delegate := &axmHybridFakeSession{result: Result{Text: "candidate", Tokens: 1, FinishReason: "stop"}}
	session := NewAXMHybridSession(delegate, Description{Model: "fixture", Backend: "pytorch"}, axmmirror.DefaultLiveNeuralGrounding(), path)
	for _, prompt := range []string{"first-private-prompt", "second-private-prompt"} {
		if _, err := session.Generate(context.Background(), prompt, testHybridOptions(), nil); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.TrimSpace(string(data))
	if strings.Contains(text, "first-private-prompt") || strings.Contains(text, "second-private-prompt") || strings.Contains(text, "candidate") {
		t.Fatal("trace leaked raw prompt or neural output")
	}
	lines := strings.Split(text, "\n")
	if len(lines) != 2 {
		t.Fatalf("trace lines = %d, want 2: %q", len(lines), text)
	}
	var traces [2]AXMHybridTrace
	for index, line := range lines {
		if err := json.Unmarshal([]byte(line), &traces[index]); err != nil {
			t.Fatalf("decode trace line %d: %v", index, err)
		}
		if traces[index].NeuralOutputSHA256 == "" || traces[index].Authority != "NONE" {
			t.Fatalf("unexpected trace line %d: %#v", index, traces[index])
		}
	}
	if traces[0].PromptSHA256 == traces[1].PromptSHA256 {
		t.Fatal("distinct prompts collapsed to the same trace identity")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("trace mode = %o, want 600", info.Mode().Perm())
	}
}

func TestAXMHybridSessionCloseDelegates(t *testing.T) {
	delegate := &axmHybridFakeSession{}
	session := NewAXMHybridSession(delegate, Description{}, axmmirror.DefaultLiveNeuralGrounding(), "")
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if !delegate.closed {
		t.Fatal("close was not delegated")
	}
}
