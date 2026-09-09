package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func appWithBlockedPersistence(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("block persistence"), 0o600); err != nil {
		t.Fatal(err)
	}
	return &App{
		dir:     blocker,
		state:   validWorkshopState("original"),
		running: map[string]bool{},
	}
}

func TestGoalMutationRollsBackWhenCheckpointCommitFails(t *testing.T) {
	a := appWithBlockedPersistence(t)
	request := httptest.NewRequest("POST", "/api/op", strings.NewReader(`{
		"op":"goal",
		"SessionID":"session-1",
		"goal":"must not remain live"
	}`))
	response := httptest.NewRecorder()

	a.handleOp(response, request)

	if response.Code != 500 {
		t.Fatalf("status=%d, want 500", response.Code)
	}
	if got := a.state.Sessions[0].Goal; got != "" {
		t.Fatalf("failed mutation remained live with goal %q", got)
	}
}

func TestDeleteMutationRollsBackWhenCheckpointCommitFails(t *testing.T) {
	a := appWithBlockedPersistence(t)
	a.state.Memories = []Memory{{ID: "memory-1", Scope: "vault", Text: "keep me"}}
	request := httptest.NewRequest("POST", "/api/op", strings.NewReader(`{
		"op":"memory_delete",
		"id":"memory-1"
	}`))
	response := httptest.NewRecorder()

	a.handleOp(response, request)

	if response.Code != 500 {
		t.Fatalf("status=%d, want 500", response.Code)
	}
	if len(a.state.Memories) != 1 || a.state.Memories[0].ID != "memory-1" {
		t.Fatalf("failed delete remained live: %+v", a.state.Memories)
	}
}
