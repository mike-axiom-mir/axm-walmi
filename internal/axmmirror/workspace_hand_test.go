package axmmirror

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceHandReadsOnlyInsideSelectedRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "app.ts"), []byte("export const x = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hand, err := NewWorkspaceHand(root)
	if err != nil {
		t.Fatal(err)
	}
	list, err := hand.List(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Entries) != 2 || list.Entries[1].Path != "src/app.ts" {
		t.Fatalf("unexpected entries %#v", list.Entries)
	}
	read, err := hand.Read("src/app.ts", 1024)
	if err != nil {
		t.Fatal(err)
	}
	if read.Content != "export const x = 1;\n" || read.WorkspaceMutation {
		t.Fatalf("unexpected read %#v", read)
	}
	if _, err := hand.Read("../secret", 1024); err == nil {
		t.Fatal("expected root escape refusal")
	}
}

func TestWorkspaceHandHashesAndExplicitlyGatesWrites(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.txt")
	if err := os.WriteFile(path, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}
	hand, err := NewWorkspaceHand(root)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := hand.Stat("a.txt", true)
	if err != nil {
		t.Fatal(err)
	}
	before := sha256.Sum256([]byte("before"))
	after := sha256.Sum256([]byte("after"))
	if hash.Entry == nil || hash.Entry.SHA256 != hex.EncodeToString(before[:]) {
		t.Fatalf("unexpected hash %#v", hash.Entry)
	}
	request := CandidateWriteRequest{Schema: CandidateWriteRequestSchema, CandidateID: "test", SourceSHA256: hex.EncodeToString(before[:]), Operations: []CandidateWriteOperation{{Action: "update", Path: "a.txt", Content: "after", ContentSHA256: hex.EncodeToString(after[:]), ExpectedSHA256: hex.EncodeToString(before[:])}}}
	if _, err := hand.Apply(request, false); err == nil {
		t.Fatal("expected explicit write opt-in")
	}
	receipt, err := hand.Apply(request, true)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.WorkspaceMutation || receipt.WriteReceipt == nil || receipt.WriteReceipt.FilesChanged != 1 {
		t.Fatalf("unexpected receipt %#v", receipt)
	}
}
