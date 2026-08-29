package model

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestWALMIWindowsComposeLockExclusiveAndReleasable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "compose.lock")
	first, err := lockComposeTransaction(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := lockComposeTransaction(path)
	if err == nil {
		unlockComposeTransaction(second)
		t.Fatal("second owner unexpectedly acquired compose lock")
	}
	if !strings.Contains(err.Error(), "another process owns this compose") {
		t.Fatalf("error = %v", err)
	}
	unlockComposeTransaction(first)
	third, err := lockComposeTransaction(path)
	if err != nil {
		t.Fatalf("lock did not release: %v", err)
	}
	unlockComposeTransaction(third)
}
