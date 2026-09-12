package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const workshopHTTPServerHelperEnv = "AXM_WORKSHOP_HTTP_SERVER_HELPER"

type workshopHTTPServerProcess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Scanner
	stderr  *bytes.Buffer
	baseURL string
	waited  bool
}

func TestWorkshopHTTPServersRejectStaleWriterAcrossProcesses(t *testing.T) {
	dir := t.TempDir()
	if _, err := saveWorkshopState(dir, validWorkshopState("shared server state"), workshopCheckpointToken{}); err != nil {
		t.Fatal(err)
	}

	first := startWorkshopHTTPServerProcess(t, dir)
	second := startWorkshopHTTPServerProcess(t, dir)

	firstState := getWorkshopHTTPState(t, first.baseURL)
	secondState := getWorkshopHTTPState(t, second.baseURL)
	if firstState.Sessions[0].Goal != "" || secondState.Sessions[0].Goal != "" {
		t.Fatalf("servers did not begin from the same goal: first=%q second=%q", firstState.Sessions[0].Goal, secondState.Sessions[0].Goal)
	}

	if status, body := postWorkshopHTTPGoal(t, first.baseURL, "first server committed"); status != http.StatusOK {
		t.Fatalf("first server goal status=%d body=%q", status, body)
	}
	if status, body := postWorkshopHTTPGoal(t, second.baseURL, "stale server overwrite"); status != http.StatusInternalServerError || !strings.Contains(body, errWorkshopStateConflict.Error()) {
		t.Fatalf("stale server was not rejected: status=%d body=%q", status, body)
	}

	persisted, err := loadWorkshopState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := persisted.State.Sessions[0].Goal; got != "first server committed" {
		t.Fatalf("persisted goal=%q, want first server committed", got)
	}
	staleView := getWorkshopHTTPState(t, second.baseURL)
	if got := staleView.Sessions[0].Goal; got != "" {
		t.Fatalf("failed stale mutation leaked into second server live state: goal=%q", got)
	}

	second.stop(t)
	restarted := startWorkshopHTTPServerProcess(t, dir)
	restartedState := getWorkshopHTTPState(t, restarted.baseURL)
	if got := restartedState.Sessions[0].Goal; got != "first server committed" {
		t.Fatalf("restarted server goal=%q, want first server committed", got)
	}
	if status, body := postWorkshopHTTPGoal(t, restarted.baseURL, "restart accepted current checkpoint"); status != http.StatusOK {
		t.Fatalf("restarted server goal status=%d body=%q", status, body)
	}

	finalState, err := loadWorkshopState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := finalState.State.Sessions[0].Goal; got != "restart accepted current checkpoint" {
		t.Fatalf("final persisted goal=%q, want restart accepted current checkpoint", got)
	}
}

// TestWorkshopHTTPServerProcessHelper runs the production Workshop App and
// routes behind a real loopback HTTP server in a separate OS process. The
// parent test controls only the process lifetime and HTTP requests; it does not
// call the storage primitives on behalf of the child server.
func TestWorkshopHTTPServerProcessHelper(t *testing.T) {
	if os.Getenv(workshopHTTPServerHelperEnv) == "" {
		return
	}

	a, err := newApp()
	if err != nil {
		t.Fatalf("start Workshop app: %v", err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.runtimeLoop(ctx)

	srv := &http.Server{Handler: a.routes(), ReadHeaderTimeout: 5 * time.Second}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(listener)
	}()

	fmt.Printf("READY http://%s\n", listener.Addr().String())
	if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
		t.Fatalf("wait for parent: %v", err)
	}
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown Workshop server: %v", err)
	}
	if err := <-serveErr; err != nil && err != http.ErrServerClosed {
		t.Fatalf("serve Workshop: %v", err)
	}
}

func startWorkshopHTTPServerProcess(t *testing.T, dir string) *workshopHTTPServerProcess {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("locate test executable: %v", err)
	}
	cmd := exec.Command(executable, "-test.run=^TestWorkshopHTTPServerProcessHelper$", "-test.count=1")
	cmd.Env = append(os.Environ(),
		workshopHTTPServerHelperEnv+"=1",
		"AXM_WORKSHOP_DATA="+dir,
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
		t.Fatalf("start Workshop HTTP helper: %v", err)
	}
	process := &workshopHTTPServerProcess{
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

	if !process.stdout.Scan() {
		process.waited = true
		_ = process.cmd.Wait()
		t.Fatalf("Workshop HTTP helper stopped before READY: scan=%v stderr=%q", process.stdout.Err(), process.stderr.String())
	}
	line := strings.TrimSpace(process.stdout.Text())
	if !strings.HasPrefix(line, "READY http://") {
		t.Fatalf("Workshop HTTP helper first marker=%q, want READY URL", line)
	}
	process.baseURL = strings.TrimPrefix(line, "READY ")
	return process
}

func (process *workshopHTTPServerProcess) stop(t *testing.T) {
	t.Helper()
	if process.waited {
		return
	}
	if err := process.stdin.Close(); err != nil {
		t.Fatalf("close Workshop HTTP helper stdin: %v", err)
	}
	if err := process.cmd.Wait(); err != nil {
		t.Fatalf("Workshop HTTP helper failed: %v stderr=%q", err, process.stderr.String())
	}
	process.waited = true
}

func getWorkshopHTTPState(t *testing.T, baseURL string) State {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/api/state")
	if err != nil {
		t.Fatalf("GET /api/state: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		t.Fatalf("GET /api/state status=%d body=%q", resp.StatusCode, body)
	}
	var state State
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&state); err != nil {
		t.Fatalf("decode /api/state: %v", err)
	}
	if len(state.Sessions) == 0 {
		t.Fatal("GET /api/state returned no sessions")
	}
	return state
}

func postWorkshopHTTPGoal(t *testing.T, baseURL, goal string) (int, string) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"op":        "goal",
		"sessionID": "session-1",
		"goal":      goal,
	})
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(baseURL+"/api/op", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST /api/op: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		t.Fatalf("read /api/op response: %v", err)
	}
	return resp.StatusCode, strings.TrimSpace(string(body))
}
