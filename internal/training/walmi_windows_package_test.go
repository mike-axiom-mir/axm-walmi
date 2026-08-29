//go:build windows

package training

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWALMIPackageWindowsAcceptsPythonWithoutUnixExecuteBits(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows executable permission semantics")
	}
	directory := t.TempDir()
	python := filepath.Join(directory, "python.exe")
	if err := os.WriteFile(python, []byte("fixture"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", directory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	candidates := pythonCandidates()
	if len(candidates) != 1 || candidates[0] != python {
		t.Fatalf("Python candidates = %v, want %s", candidates, python)
	}
}

type recordSourceFunc func(context.Context, func(Record) error) error

func (function recordSourceFunc) Stream(ctx context.Context, consume func(Record) error) error {
	return function(ctx, consume)
}

func TestWALMIPackageWindowsPyTorch(t *testing.T) {
	architecture := json.RawMessage(`{"family":"decoder-transformer","vocabulary_size":259,"tokenizer":{"name":"byte","revision":"builtin-byte-schema-1"}}`)
	resolver := PyTorchResolver{
		OS: "windows", Arch: "amd64", Candidates: []string{"python"},
		Probe: func(context.Context, string) (pyTorchProbe, error) {
			return pyTorchProbe{PythonVersion: "3.12", TorchVersion: "package", Device: "cpu", Manufacturer: "CPU", Accelerator: "x64"}, nil
		},
	}
	selection, err := resolver.Resolve(context.Background(), ResolveRequest{Architecture: architecture})
	if err != nil {
		t.Fatal(err)
	}
	if selection.Execution.Host.OS != "windows" {
		t.Fatalf("host = %+v", selection.Execution.Host)
	}
}

func TestWALMIPackageTorchTitanRefusesWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows package check")
	}
	_, err := (TorchTitan{Python: "python", LocalProcs: 1, Nodes: 1}).Run(context.Background(), Request{})
	if err == nil || !strings.Contains(err.Error(), "not enabled") {
		t.Fatalf("error = %v", err)
	}
}
