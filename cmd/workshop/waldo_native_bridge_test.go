package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNativePromptPreservesRoles(t *testing.T) {
	messages := []waldoBridgeChatMessage{
		{Role: "system", Content: json.RawMessage(`"You are Waldo."`)},
		{Role: "user", Content: json.RawMessage(`"Continue the goal."`)},
	}
	prompt, err := waldoBridgePrompt(messages)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"[SYSTEM]", "You are Waldo.", "[USER]", "Continue the goal.", "[ASSISTANT]"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q: %s", want, prompt)
		}
	}
}

func TestNativeMessageTextMarksImageWithoutClaimingVision(t *testing.T) {
	raw := json.RawMessage(`[
		{"type":"text","text":"What do you notice?"},
		{"type":"image_url","image_url":{"url":"data:image/png;base64,AA=="}}
	]`)
	text, err := waldoBridgeMessageText(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "What do you notice?") {
		t.Fatalf("text part missing: %s", text)
	}
	if !strings.Contains(text, "cannot inspect its pixels") {
		t.Fatalf("image limitation must stay explicit: %s", text)
	}
}
