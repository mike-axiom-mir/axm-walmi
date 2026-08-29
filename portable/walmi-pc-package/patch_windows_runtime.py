from __future__ import annotations

import json
from pathlib import Path

BASE_COMMIT = "c6011afbb456b8fe1e6c7c6eda7857aec681cb8f"
changes: list[str] = []


def replace_once(path: str, old: str, new: str, label: str) -> None:
    file = Path(path)
    text = file.read_text(encoding="utf-8")
    if old not in text:
        raise SystemExit(f"patch seam missing for {label}: {path}")
    file.write_text(text.replace(old, new, 1), encoding="utf-8")
    changes.append(label)


replace_once(
    "internal/training/pytorch.go",
    'if hostOS != "linux" {\n\t\treturn Selection{}, fmt.Errorf("PyTorch training currently requires Linux; this host is %s/%s", hostOS, hostArch)\n\t}',
    'if hostOS != "linux" && hostOS != "windows" {\n\t\treturn Selection{}, fmt.Errorf("PyTorch training currently requires Linux or Windows; this host is %s/%s", hostOS, hostArch)\n\t}',
    "PyTorch resolver permits the packaged Windows runtime",
)

replace_once(
    "internal/training/environment.go",
    '\tcase "linux":\n\t\tfor _, candidate := range []struct {',
    '\tcase "windows":\n\t\tinstallation, err := resolver.probe()(ctx, resolver.candidates(), "torch", "torch")\n\t\tif err == nil {\n\t\t\treturn BackendPyTorch, &installation, nil\n\t\t}\n\t\tif !errors.Is(err, errPythonPackageNotFound) {\n\t\t\treturn "", nil, fmt.Errorf("probe PyTorch: %w", err)\n\t\t}\n\t\treturn "", nil, fmt.Errorf("model.backend=auto found no usable bundled PyTorch runtime on Windows")\n\tcase "linux":\n\t\tfor _, candidate := range []struct {',
    "model.backend=auto resolves bundled PyTorch on Windows",
)

# Python workers use Unix process groups in the research source. The Windows
# package keeps cancellation bounded by killing only the worker process.
p = Path("internal/training/python_worker.go")
s = p.read_text(encoding="utf-8")
s = s.replace('\n\t"syscall"', "")
start = s.index("func terminateWorkerGroup(command *exec.Cmd) string {")
end = s.index("\n}\n\nfunc writeStoppedByWorkerExit", start) + 2
replacement = '''func terminateWorkerGroup(command *exec.Cmd) string {
\tif command.Process == nil {
\t\treturn "left running"
\t}
\tif err := command.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
\t\treturn fmt.Sprintf("could not be killed: %v", err)
\t}
\treturn "killed"
}'''
s = s[:start] + replacement + s[end:]
s = s.replace(
    "return errors.Is(err, os.ErrClosed) || errors.Is(err, io.ErrClosedPipe) || errors.Is(err, syscall.EPIPE)",
    "return errors.Is(err, os.ErrClosed) || errors.Is(err, io.ErrClosedPipe)",
)
s = s.replace(
    '\n\tif command.SysProcAttr == nil {\n\t\tcommand.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}\n\t}',
    "",
)
p.write_text(s, encoding="utf-8")
changes.append("Python worker cancellation uses portable Process.Kill")

# TorchTitan remains in the exact archived source and Git history. It is a
# Linux/distributed lane, so the Windows package explicitly routes to PyTorch.
p = Path("internal/training/torchtitan.go")
s = p.read_text(encoding="utf-8")
s = s.replace('\n\t"syscall"', "")
run_start = "func (backend TorchTitan) Run(ctx context.Context, request Request) (Observation, error) {\n"
if run_start not in s:
    raise SystemExit("TorchTitan Run seam missing")
s = s.replace(
    run_start,
    run_start
    + '\tif runtime.GOOS == "windows" {\n'
    + '\t\treturn Observation{}, fmt.Errorf("TorchTitan is retained in source but not enabled by the WALMI Windows host; use bundled PyTorch")\n'
    + "\t}\n",
    1,
)
old_cancel = '''\tcommand.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
\tcommand.Cancel = func() error {
\t\treturn syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
\t}'''
new_cancel = '''\tcommand.Cancel = func() error {
\t\tif command.Process == nil {
\t\t\treturn nil
\t\t}
\t\treturn command.Process.Kill()
\t}'''
if old_cancel not in s:
    raise SystemExit("TorchTitan Unix cancellation seam missing")
p.write_text(s.replace(old_cancel, new_cancel, 1), encoding="utf-8")
changes.append("TorchTitan source retained; Windows runtime explicitly uses PyTorch")

# These original tests create Unix fake executables/process groups. The exact
# unmodified suite is already preserved and Linux-PASS in the source audit.
# mlx_test.go also contains recordSourceFunc, a generic RecordSource test adapter;
# the Windows package re-declares only that helper below rather than re-enabling
# the Unix shell-based MLX tests.
for name in ["python_worker_test.go", "mlx_test.go", "pytorch_test.go", "torchtitan_test.go"]:
    p = Path("internal/training") / name
    s = p.read_text(encoding="utf-8")
    if not s.startswith("//go:build !windows"):
        p.write_text("//go:build !windows\n\n" + s, encoding="utf-8")
changes.append("Unix host-specific training tests excluded only from Windows package tests")

# The inference suite has two shell-script fake Python workers and one POSIX
# permission-bit assertion. Keep all production inference code and the useful
# tests; skip only those fake-worker executions on Windows and keep the
# append-only/privacy trace assertions while omitting a Unix mode-bit check
# that Windows filesystems cannot represent through Go's permission bits.
replace_once(
    "internal/inference/inference_test.go",
    '\t"path/filepath"\n\t"strings"',
    '\t"path/filepath"\n\t"runtime"\n\t"strings"',
    "Windows inference tests import runtime for fixture gating",
)
replace_once(
    "internal/inference/inference_test.go",
    'func TestMLXSessionConsumesStreamingProtocol(t *testing.T) {\n',
    'func TestMLXSessionConsumesStreamingProtocol(t *testing.T) {\n\tif runtime.GOOS == "windows" {\n\t\tt.Skip("Unix shell-worker fixture; MLX is not the WALMI Windows backend")\n\t}\n',
    "Windows skips Unix MLX shell-worker fixture",
)
replace_once(
    "internal/inference/inference_test.go",
    'func TestPyTorchSessionConsumesStreamingProtocol(t *testing.T) {\n',
    'func TestPyTorchSessionConsumesStreamingProtocol(t *testing.T) {\n\tif runtime.GOOS == "windows" {\n\t\tt.Skip("Unix shell-worker fixture; packaged PyTorch is verified with the real portable Python runtime")\n\t}\n',
    "Windows skips Unix PyTorch shell-worker fixture",
)
replace_once(
    "internal/inference/axm_hybrid_test.go",
    '\t"path/filepath"\n\t"strings"',
    '\t"path/filepath"\n\t"runtime"\n\t"strings"',
    "Windows hybrid trace test imports runtime for permission semantics",
)
replace_once(
    "internal/inference/axm_hybrid_test.go",
    'if info.Mode().Perm() != 0o600 {\n\t\tt.Fatalf("trace mode = %o, want 600", info.Mode().Perm())\n\t}',
    'if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {\n\t\tt.Fatalf("trace mode = %o, want 600", info.Mode().Perm())\n\t}',
    "Windows preserves trace checks without Unix permission-bit assertion",
)

# Model compose locking used direct Unix flock. Move that behind a platform
# seam and use Windows LockFileEx with FAIL_IMMEDIATELY to preserve exclusive,
# non-blocking transaction ownership and automatic release on handle close.
p = Path("internal/model/build.go")
s = p.read_text(encoding="utf-8")
s = s.replace('\n\t"syscall"', "")
old_lock = '''func lockComposeTransaction(path string) (*os.File, error) {
\tfile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
\tif err != nil {
\t\treturn nil, err
\t}
\tif err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
\t\t_ = file.Close()
\t\treturn nil, fmt.Errorf("another process owns this compose; wait for it to finish")
\t}
\treturn file, nil
}

func unlockComposeTransaction(file *os.File) {
\tif file == nil {
\t\treturn
\t}
\t_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
\t_ = file.Close()
}'''
new_lock = '''func lockComposeTransaction(path string) (*os.File, error) {
\tfile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
\tif err != nil {
\t\treturn nil, err
\t}
\tif err := lockComposeFile(file); err != nil {
\t\t_ = file.Close()
\t\treturn nil, fmt.Errorf("another process owns this compose; wait for it to finish")
\t}
\treturn file, nil
}

func unlockComposeTransaction(file *os.File) {
\tif file == nil {
\t\treturn
\t}
\t_ = unlockComposeFile(file)
\t_ = file.Close()
}'''
if old_lock not in s:
    raise SystemExit("model flock seam missing")
p.write_text(s.replace(old_lock, new_lock, 1), encoding="utf-8")

Path("internal/model/compose_lock_windows.go").write_text(
    '''//go:build windows

package model

import (
\t"os"

\t"golang.org/x/sys/windows"
)

func lockComposeFile(file *os.File) error {
\tvar overlapped windows.Overlapped
\treturn windows.LockFileEx(
\t\twindows.Handle(file.Fd()),
\t\twindows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
\t\t0,
\t\t1,
\t\t0,
\t\t&overlapped,
\t)
}

func unlockComposeFile(file *os.File) error {
\tvar overlapped windows.Overlapped
\treturn windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &overlapped)
}
''',
    encoding="utf-8",
)
Path("internal/model/compose_lock_nonwindows.go").write_text(
    '''//go:build !windows

package model

import (
\t"os"
\t"syscall"
)

func lockComposeFile(file *os.File) error {
\treturn syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

func unlockComposeFile(file *os.File) error {
\treturn syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
''',
    encoding="utf-8",
)
changes.append("model compose ownership uses Windows LockFileEx; Unix keeps flock")

# Windows package tests replace the Unix-host fixtures with relevant Windows
# assertions while the original suite remains in the archived exact source.
# recordSourceFunc is copied from mlx_test.go because it is a generic test-only
# RecordSource adapter, not an MLX or Unix runtime behavior.
Path("internal/training/walmi_windows_package_test.go").write_text(
    '''package training

import (
\t"context"
\t"encoding/json"
\t"runtime"
\t"strings"
\t"testing"
)

type recordSourceFunc func(context.Context, func(Record) error) error

func (function recordSourceFunc) Stream(ctx context.Context, consume func(Record) error) error {
\treturn function(ctx, consume)
}

func TestWALMIPackageWindowsPyTorch(t *testing.T) {
\tarchitecture := json.RawMessage(`{"family":"decoder-transformer","vocabulary_size":259,"tokenizer":{"name":"byte","revision":"builtin-byte-schema-1"}}`)
\tresolver := PyTorchResolver{
\t\tOS: "windows", Arch: "amd64", Candidates: []string{"python"},
\t\tProbe: func(context.Context, string) (pyTorchProbe, error) {
\t\t\treturn pyTorchProbe{PythonVersion: "3.12", TorchVersion: "package", Device: "cpu", Manufacturer: "CPU", Accelerator: "x64"}, nil
\t\t},
\t}
\tselection, err := resolver.Resolve(context.Background(), ResolveRequest{Architecture: architecture})
\tif err != nil { t.Fatal(err) }
\tif selection.Execution.Host.OS != "windows" { t.Fatalf("host = %+v", selection.Execution.Host) }
}

func TestWALMIPackageTorchTitanRefusesWindows(t *testing.T) {
\tif runtime.GOOS != "windows" { t.Skip("Windows package check") }
\t_, err := (TorchTitan{Python: "python", LocalProcs: 1, Nodes: 1}).Run(context.Background(), Request{})
\tif err == nil || !strings.Contains(err.Error(), "not enabled") { t.Fatalf("error = %v", err) }
}
''',
    encoding="utf-8",
)
changes.append("Windows package preserves the generic recordSourceFunc test adapter")

Path("internal/model/walmi_windows_lock_test.go").write_text(
    '''package model

import (
\t"path/filepath"
\t"strings"
\t"testing"
)

func TestWALMIWindowsComposeLockExclusiveAndReleasable(t *testing.T) {
\tpath := filepath.Join(t.TempDir(), "compose.lock")
\tfirst, err := lockComposeTransaction(path)
\tif err != nil { t.Fatal(err) }
\tsecond, err := lockComposeTransaction(path)
\tif err == nil {
\t\tunlockComposeTransaction(second)
\t\tt.Fatal("second owner unexpectedly acquired compose lock")
\t}
\tif !strings.Contains(err.Error(), "another process owns this compose") { t.Fatalf("error = %v", err) }
\tunlockComposeTransaction(first)
\tthird, err := lockComposeTransaction(path)
\tif err != nil { t.Fatalf("lock did not release: %v", err) }
\tunlockComposeTransaction(third)
}
''',
    encoding="utf-8",
)

Path("dist").mkdir(exist_ok=True)
Path("dist/WINDOWS_RUNTIME_PATCH_RECEIPT.json").write_text(
    json.dumps(
        {
            "schema": "walmi.windows-runtime-patch/v7",
            "baseCommit": BASE_COMMIT,
            "scope": "PACKAGE_BUILD_ONLY_NOT_MERGED",
            "changes": changes,
            "torchTitanWindows": "SOURCE_RETAINED_WINDOWS_DISABLED_USE_PYTORCH",
            "authority": "NONE",
        },
        indent=2,
    ),
    encoding="utf-8",
)
print("WALMI Windows package membrane applied:", len(changes), "recorded changes")
