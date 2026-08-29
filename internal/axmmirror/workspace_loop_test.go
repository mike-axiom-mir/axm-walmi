package axmmirror

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type scriptedWorkspaceModel struct {
	responses []string
	index     int
}

func (model *scriptedWorkspaceModel) Generate(_ context.Context, _ string) (string, error) {
	response := model.responses[model.index]
	model.index++
	return response, nil
}

func TestWorkspaceLoopReadsWritesVerifiesAndFinishes(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "app.txt")
	if err := os.WriteFile(path, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := sha256.Sum256([]byte("before"))
	after := sha256.Sum256([]byte("after"))
	model := &scriptedWorkspaceModel{responses: []string{
		`{"tool":"LIST_FILES","path":"."}`,
		`{"tool":"READ_FILE","path":"app.txt","max_bytes":1024}`,
		`{"tool":"APPLY_WRITES","write_request":{"schema":"` + CandidateWriteRequestSchema + `","candidate_id":"workspace-test","operations":[{"action":"update","path":"app.txt","content":"after","content_sha256":"` + hex.EncodeToString(after[:]) + `","expected_sha256":"` + hex.EncodeToString(before[:]) + `"}],"source_sha256":"` + hex.EncodeToString(before[:]) + `"}}`,
		`{"tool":"RUN_VERIFY","command_index":0}`,
		`{"final":"updated and verified"}`,
	}}
	hand, err := NewWorkspaceHand(root)
	if err != nil {
		t.Fatal(err)
	}
	request := WorkspaceLoopRequest{Schema: WorkspaceLoopRequestSchema, Prompt: "Update app.txt", MaxTurns: 8, VerifyCommands: [][]string{{"go", "version"}}}
	receipt, err := RunWorkspaceLoop(context.Background(), hand, model, request, true)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != "COMPLETED" || !receipt.WorkspaceMutation || !receipt.VerificationRun || !receipt.VerificationPassed || receipt.Final == "" {
		t.Fatalf("unexpected receipt %#v", receipt)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "after" {
		t.Fatalf("unexpected file %q err=%v", data, err)
	}
}

func TestWorkspaceLoopRetainsWriteBoundary(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.txt"), []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}
	model := &scriptedWorkspaceModel{responses: []string{`{"tool":"APPLY_WRITES","write_request":{"schema":"` + CandidateWriteRequestSchema + `","candidate_id":"blocked","operations":[],"source_sha256":"` + strings.Repeat("0", 64) + `"}}`, `{"final":"write was refused"}`}}
	hand, err := NewWorkspaceHand(root)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := RunWorkspaceLoop(context.Background(), hand, model, WorkspaceLoopRequest{Schema: WorkspaceLoopRequestSchema, Prompt: "Try a write", MaxTurns: 2}, false)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.WorkspaceMutation || receipt.State != "COMPLETED" || receipt.Turns[0].Error == "" {
		t.Fatalf("unexpected receipt %#v", receipt)
	}
}
