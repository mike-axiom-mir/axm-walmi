package axmmirror

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateWriterFocusedChecks(t *testing.T) {
	root := t.TempDir()
	source := strings.Repeat("a", 64)
	request := func(operations ...CandidateWriteOperation) CandidateWriteRequest {
		return CandidateWriteRequest{Schema: CandidateWriteRequestSchema, CandidateID: "website-1", SourceSHA256: source, Operations: operations}
	}
	content := func(action, path, body string) CandidateWriteOperation {
		return CandidateWriteOperation{Action: action, Path: path, Content: body, ContentSHA256: candidateBytesSHA256([]byte(body))}
	}

	t.Run("create", func(t *testing.T) {
		receipt, err := ApplyCandidateWrites(root, request(content("create", "index.html", "<h1>WALDO</h1>\n")))
		if err != nil || receipt.State != "APPLIED" || receipt.WriterAuthoredCode || !receipt.WorkspaceMutation {
			t.Fatalf("create receipt = %+v, err = %v", receipt, err)
		}
	})
	t.Run("update", func(t *testing.T) {
		before, _ := os.ReadFile(filepath.Join(root, "index.html"))
		op := content("update", "index.html", "<h1>WALDO website</h1>\n")
		op.ExpectedSHA256 = candidateBytesSHA256(before)
		if _, err := ApplyCandidateWrites(root, request(op)); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("delete", func(t *testing.T) {
		path := filepath.Join(root, "delete.txt")
		_ = os.WriteFile(path, []byte("remove"), 0o644)
		op := CandidateWriteOperation{Action: "delete", Path: "delete.txt", ExpectedSHA256: candidateBytesSHA256([]byte("remove"))}
		if _, err := ApplyCandidateWrites(root, request(op)); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("deleted path still exists: %v", err)
		}
	})

	refused := []struct {
		name string
		op   CandidateWriteOperation
		want string
	}{
		{"stale hash", CandidateWriteOperation{Action: "update", Path: "index.html", Content: "x", ContentSHA256: candidateBytesSHA256([]byte("x")), ExpectedSHA256: strings.Repeat("b", 64)}, "stale"},
		{"content hash", CandidateWriteOperation{Action: "create", Path: "hash.txt", Content: "x", ContentSHA256: strings.Repeat("c", 64)}, "mismatch"},
		{"path traversal", content("create", "../escape.txt", "x"), "refused segment"},
		{"absolute path", content("create", "/tmp/escape.txt", "x"), "clean relative"},
		{"backslash path", content("create", `sub\\escape.txt`, "x"), "invalid"},
		{"git path", content("create", ".git/config", "x"), "refused segment"},
		{"duplicate path", content("create", "same.txt", "x"), "duplicate"},
		{"unsupported action", content("chmod", "mode.txt", "x"), "unsupported action"},
		{"delete content", CandidateWriteOperation{Action: "delete", Path: "index.html", Content: "x", ContentSHA256: candidateBytesSHA256([]byte("x")), ExpectedSHA256: candidateBytesSHA256([]byte("<h1>WALDO website</h1>\n"))}, "must not carry content"},
	}
	for _, test := range refused {
		t.Run(test.name, func(t *testing.T) {
			operations := []CandidateWriteOperation{test.op}
			if test.name == "duplicate path" {
				operations = append(operations, test.op)
			}
			if _, err := ApplyCandidateWrites(root, request(operations...)); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}

	t.Run("file limit", func(t *testing.T) {
		body := strings.Repeat("x", CandidateWriteMaxFileBytes+1)
		if _, err := ApplyCandidateWrites(root, request(content("create", "large.txt", body))); err == nil || !strings.Contains(err.Error(), "file limit") {
			t.Fatalf("file limit error = %v", err)
		}
	})
	t.Run("operation limit", func(t *testing.T) {
		operations := make([]CandidateWriteOperation, CandidateWriteMaxOperations+1)
		if _, err := ApplyCandidateWrites(root, request(operations...)); err == nil || !strings.Contains(err.Error(), "1-64") {
			t.Fatalf("operation limit error = %v", err)
		}
	})
	t.Run("transaction preflight", func(t *testing.T) {
		operations := []CandidateWriteOperation{content("create", "preflight-ok.txt", "x"), content("create", "../preflight-bad.txt", "x")}
		if _, err := ApplyCandidateWrites(root, request(operations...)); err == nil {
			t.Fatal("invalid transaction was accepted")
		}
		if _, err := os.Stat(filepath.Join(root, "preflight-ok.txt")); !os.IsNotExist(err) {
			t.Fatalf("preflight partially wrote: %v", err)
		}
	})
	t.Run("outside root protection", func(t *testing.T) {
		outside := filepath.Join(filepath.Dir(root), "outside.txt")
		if _, err := os.Stat(outside); !os.IsNotExist(err) {
			t.Skip("outside fixture path already exists")
		}
		if _, err := ApplyCandidateWrites(root, request(content("create", "../outside.txt", "x"))); err == nil {
			t.Fatal("outside-root write accepted")
		}
		if _, err := os.Stat(outside); !os.IsNotExist(err) {
			t.Fatalf("outside path changed: %v", err)
		}
	})
	t.Run("symlink escape", func(t *testing.T) {
		outside := t.TempDir()
		link := filepath.Join(root, "link")
		if err := os.Symlink(outside, link); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if _, err := ApplyCandidateWrites(root, request(content("create", "link/escape.txt", "x"))); err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("symlink escape error = %v", err)
		}
	})
	t.Run("root symlink", func(t *testing.T) {
		parent := t.TempDir()
		link := filepath.Join(parent, "root-link")
		if err := os.Symlink(root, link); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if _, err := ApplyCandidateWrites(link, request(content("create", "root-link.txt", "x"))); err == nil || !strings.Contains(err.Error(), "non-symlink") {
			t.Fatalf("root symlink error = %v", err)
		}
	})
	t.Run("receipt binds source", func(t *testing.T) {
		receipt, err := ApplyCandidateWrites(root, request(content("create", "receipt.txt", "bound")))
		if err != nil {
			t.Fatal(err)
		}
		if receipt.SourceSHA256 != source || len(receipt.ReceiptSHA256) != 64 || receipt.NetworkUsed || receipt.Installed || receipt.Promoted || receipt.CanonChanged {
			t.Fatalf("receipt truth = %+v", receipt)
		}
	})
}
