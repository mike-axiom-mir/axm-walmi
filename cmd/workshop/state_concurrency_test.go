package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const (
	workshopStateHelperModeEnv  = "AXM_WORKSHOP_STATE_HELPER"
	workshopStateHelperDirEnv   = "AXM_WORKSHOP_STATE_DIR"
	workshopStateHelperTitleEnv = "AXM_WORKSHOP_STATE_TITLE"
)

type workshopStateProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	stderr *bytes.Buffer
	waited bool
}

func TestWorkshopStateRejectsStaleWriter(t *testing.T) {
	dir := t.TempDir()
	initial := validWorkshopState("initial")
	if _, err := saveWorkshopState(dir, initial, workshopCheckpointToken{}); err != nil {
		t.Fatal(err)
	}

	first, err := loadWorkshopState(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := loadWorkshopState(dir)
	if err != nil {
		t.Fatal(err)
	}

	first.State.Sessions[0].Title = "first writer"
	if _, err := saveWorkshopState(dir, first.State, first.Checkpoint); err != nil {
		t.Fatalf("first writer: %v", err)
	}
	second.State.Sessions[0].Title = "stale writer"
	if _, err := saveWorkshopState(dir, second.State, second.Checkpoint); err == nil || !strings.Contains(err.Error(), "changed since it was loaded") {
		t.Fatalf("stale writer was not rejected: %v", err)
	}

	loaded, err := loadWorkshopState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.State.Sessions[0].Title; got != "first writer" {
		t.Fatalf("current title=%q, want first writer", got)
	}
}

func TestWorkshopStateRejectsConcurrentWriterWhileClaimHeld(t *testing.T) {
	dir := t.TempDir()
	claim, err := lockWorkshopState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := saveWorkshopState(dir, validWorkshopState("held"), workshopCheckpointToken{}); err == nil || !strings.Contains(err.Error(), "busy in another process") {
		unlockWorkshopState(claim)
		t.Fatalf("concurrent writer was not rejected: %v", err)
	}
	unlockWorkshopState(claim)

	if _, err := saveWorkshopState(dir, validWorkshopState("released"), workshopCheckpointToken{}); err != nil {
		t.Fatalf("writer could not retry after claim release: %v", err)
	}
}

func TestWorkshopStateRejectsStaleWriterAcrossProcesses(t *testing.T) {
	dir := t.TempDir()
	if _, err := saveWorkshopState(dir, validWorkshopState("initial"), workshopCheckpointToken{}); err != nil {
		t.Fatal(err)
	}

	first := startWorkshopStateProcess(t, "stale-save", dir, "first process")
	second := startWorkshopStateProcess(t, "stale-save", dir, "stale process")

	firstReady := first.readLine(t)
	secondReady := second.readLine(t)
	if !strings.HasPrefix(firstReady, "READY sha256:") {
		t.Fatalf("first process did not report a checkpoint: %q", firstReady)
	}
	if firstReady != secondReady {
		t.Fatalf("processes did not load the same checkpoint: first=%q second=%q", firstReady, secondReady)
	}

	first.signal(t)
	if got := first.readLine(t); got != "SAVED" {
		t.Fatalf("first process result=%q, want SAVED", got)
	}
	first.wait(t)

	second.signal(t)
	if got := second.readLine(t); got != "CONFLICT" {
		t.Fatalf("stale process result=%q, want CONFLICT", got)
	}
	second.wait(t)

	loaded, err := loadWorkshopState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.State.Sessions[0].Title; got != "first process" {
		t.Fatalf("current title=%q, want first process", got)
	}
}

func TestWorkshopStateReleasesLockAfterHolderProcessDeath(t *testing.T) {
	dir := t.TempDir()
	holder := startWorkshopStateProcess(t, "hold-lock", dir, "")
	if got := holder.readLine(t); got != "LOCKED" {
		t.Fatalf("holder result=%q, want LOCKED", got)
	}

	claim, err := lockWorkshopState(dir)
	if err == nil {
		unlockWorkshopState(claim)
		t.Fatal("parent acquired Workshop lock while helper process still held it")
	}
	if !strings.Contains(err.Error(), "busy in another process") {
		t.Fatalf("lock contention returned unexpected error: %v", err)
	}

	holder.killAndWait(t)

	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		claim, err = lockWorkshopState(dir)
		if err == nil {
			unlockWorkshopState(claim)
			return
		}
		lastErr = err
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("Workshop lock was not released after holder process death: %v", lastErr)
}

// TestWorkshopStateProcessHelper is launched only by the process-level tests
// above. It deliberately uses the production load/save/lock primitives from a
// separate OS process so in-memory state cannot satisfy the evidence contract.
func TestWorkshopStateProcessHelper(t *testing.T) {
	mode := os.Getenv(workshopStateHelperModeEnv)
	if mode == "" {
		return
	}
	dir := os.Getenv(workshopStateHelperDirEnv)
	if dir == "" {
		t.Fatal("helper state directory is required")
	}

	switch mode {
	case "stale-save":
		loaded, err := loadWorkshopState(dir)
		if err != nil {
			t.Fatalf("helper load: %v", err)
		}
		fmt.Printf("READY %s\n", formatWorkshopCheckpointToken(loaded.Checkpoint))
		if err := waitForWorkshopStateProcessSignal(); err != nil {
			t.Fatal(err)
		}
		loaded.State.Sessions[0].Title = os.Getenv(workshopStateHelperTitleEnv)
		_, err = saveWorkshopState(dir, loaded.State, loaded.Checkpoint)
		if err == nil {
			fmt.Println("SAVED")
			return
		}
		if errors.Is(err, errWorkshopStateConflict) {
			fmt.Println("CONFLICT")
			return
		}
		t.Fatalf("helper save: %v", err)
	case "hold-lock":
		claim, err := lockWorkshopState(dir)
		if err != nil {
			t.Fatalf("helper lock: %v", err)
		}
		defer unlockWorkshopState(claim)
		fmt.Println("LOCKED")
		if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
			t.Fatalf("helper wait: %v", err)
		}
	default:
		t.Fatalf("unknown helper mode %q", mode)
	}
}

func waitForWorkshopStateProcessSignal() error {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return nil
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read helper signal: %w", err)
	}
	return errors.New("helper signal stream closed before release")
}

func startWorkshopStateProcess(t *testing.T, mode, dir, title string) *workshopStateProcess {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("locate test executable: %v", err)
	}
	cmd := exec.Command(executable, "-test.run=^TestWorkshopStateProcessHelper$", "-test.count=1")
	cmd.Env = append(os.Environ(),
		workshopStateHelperModeEnv+"="+mode,
		workshopStateHelperDirEnv+"="+dir,
		workshopStateHelperTitleEnv+"="+title,
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("helper stdin: %v", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("helper stdout: %v", err)
	}
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper process: %v", err)
	}

	process := &workshopStateProcess{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdoutPipe),
		stderr: stderr,
	}
	t.Cleanup(func() {
		if process.waited {
			return
		}
		_ = process.stdin.Close()
		if process.cmd.Process != nil {
			_ = process.cmd.Process.Kill()
		}
		_ = process.cmd.Wait()
		process.waited = true
	})
	return process
}

func (process *workshopStateProcess) readLine(t *testing.T) string {
	t.Helper()
	if process.stdout.Scan() {
		return strings.TrimSpace(process.stdout.Text())
	}
	t.Fatalf("helper stopped before emitting expected marker: scan=%v stderr=%q", process.stdout.Err(), process.stderr.String())
	return ""
}

func (process *workshopStateProcess) signal(t *testing.T) {
	t.Helper()
	if _, err := io.WriteString(process.stdin, "continue\n"); err != nil {
		t.Fatalf("release helper process: %v", err)
	}
}

func (process *workshopStateProcess) wait(t *testing.T) {
	t.Helper()
	_ = process.stdin.Close()
	err := process.cmd.Wait()
	process.waited = true
	if err != nil {
		t.Fatalf("helper process failed: %v stderr=%q", err, process.stderr.String())
	}
}

func (process *workshopStateProcess) killAndWait(t *testing.T) {
	t.Helper()
	if err := process.cmd.Process.Kill(); err != nil {
		t.Fatalf("kill helper process: %v", err)
	}
	_ = process.stdin.Close()
	err := process.cmd.Wait()
	process.waited = true
	if err == nil {
		t.Fatal("helper process exited cleanly after kill request; process-death evidence is ambiguous")
	}
}
