package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHeartbeatIntervalBounds(t *testing.T) {
	cases := []struct {
		in, want int
	}{{0, 30}, {5, 30}, {30, 30}, {300, 300}, {86400, 86400}, {999999, 86400}}
	for _, tc := range cases {
		if got := heartbeatInterval(tc.in); got != tc.want {
			t.Fatalf("heartbeatInterval(%d)=%d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestVisibleDeliveryErrorIsBounded(t *testing.T) {
	got := visibleDeliveryError(&fixedError{message: string(make([]rune, 400))})
	if len([]rune(got)) != 361 || []rune(got)[360] != '…' {
		t.Fatalf("visible delivery error was not bounded: %d runes", len([]rune(got)))
	}
}

type fixedError struct{ message string }

func (e *fixedError) Error() string { return e.message }

func TestNormalizeStateMigratesExistingSession(t *testing.T) {
	a := &App{state: State{Version: 1, Sessions: []Session{{
		ID:        "s1",
		Heartbeat: "idle",
		Messages:  []Message{{ID: "m1", Role: "user", Text: "unfinished", Delivery: "pending"}},
	}}}}
	if !a.normalizeState() {
		t.Fatal("expected migration to change state")
	}
	s := a.state.Sessions[0]
	if a.state.Version != 2 || s.RuntimeMode != "paused" || s.HeartbeatEverySec != 300 || s.Heartbeat != "paused" {
		t.Fatalf("unexpected migrated state: version=%d session=%+v", a.state.Version, s)
	}
	if s.Messages[0].Delivery != "failed" || s.Messages[0].DeliveryError == "" {
		t.Fatalf("pending message did not become visibly retryable after restart: %+v", s.Messages[0])
	}
}

func TestFailedChatPersistsDeliveryErrorAndRetriesWithoutDuplicate(t *testing.T) {
	upstreamCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		upstreamCalls++
		w.Header().Set("Content-Type", "application/json")
		if upstreamCalls == 1 {
			http.Error(w, "local model is starting", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Recovered answer"}}]}`))
	}))
	t.Cleanup(upstream.Close)
	t.Setenv("AXM_AI_BASE_URL", upstream.URL)

	a := &App{
		dir:     t.TempDir(),
		client:  upstream.Client(),
		running: map[string]bool{},
		state: State{Version: 2,
			Identities: []Identity{{ID: "waldo", Name: "Waldo", Model: "waldo"}},
			Sessions:   []Session{{ID: "s1", IdentityID: "waldo", Title: "Test", Messages: []Message{}}},
			Memories:   []Memory{}, Consents: []Consent{}, Media: []Media{},
		},
	}
	server := httptest.NewServer(a.routes())
	t.Cleanup(server.Close)

	post := func(body any) *http.Response {
		t.Helper()
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.Post(server.URL+"/api/chat", "application/json", bytes.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	failed := post(map[string]any{"SessionID": "s1", "Text": "Try the local model"})
	if failed.StatusCode != http.StatusBadGateway {
		t.Fatalf("first chat status=%d, want %d", failed.StatusCode, http.StatusBadGateway)
	}
	_ = failed.Body.Close()
	if len(a.state.Sessions[0].Messages) != 1 {
		t.Fatalf("messages=%d, want one retained user turn", len(a.state.Sessions[0].Messages))
	}
	messageID := a.state.Sessions[0].Messages[0].ID
	if got := a.state.Sessions[0].Messages[0]; got.Delivery != "failed" || got.DeliveryError == "" {
		t.Fatalf("failed delivery was not retained: %+v", got)
	}

	retried := post(map[string]any{"SessionID": "s1", "RetryMessageID": messageID})
	if retried.StatusCode != http.StatusOK {
		t.Fatalf("retry status=%d, want %d", retried.StatusCode, http.StatusOK)
	}
	_ = retried.Body.Close()
	got := a.state.Sessions[0].Messages
	if len(got) != 2 || got[0].ID != messageID || got[0].Delivery != "answered" || got[1].Role != "assistant" || got[1].Text != "Recovered answer" {
		t.Fatalf("unexpected retry conversation: %+v", got)
	}
}
