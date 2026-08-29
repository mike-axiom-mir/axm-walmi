package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

//go:embed ui.html
var web embed.FS

type Identity struct{ ID, Name, Model string }

type Message struct {
	ID, Role, Text, CreatedAt string
	MediaIDs                  []string `json:"media_ids,omitempty"`
}

type Session struct {
	ID, IdentityID, Title, Goal, Heartbeat, LastHeartbeat, CreatedAt string
	RuntimeMode, NextHeartbeat, LastRuntimeError                     string
	HeartbeatEverySec, PulseCount                                    int
	Messages                                                         []Message
}

type Memory struct{ ID, Scope, SessionID, IdentityID, Text, CreatedAt string }
type Consent struct{ ID, SessionID, Action, Reason, Status, CreatedAt, DecidedAt string }
type Media struct{ ID, SessionID, IdentityID, Name, MIME, FileName, URL, CreatedAt string }

type State struct {
	Version    int
	Identities []Identity
	Sessions   []Session
	Memories   []Memory
	Consents   []Consent
	Media      []Media
}

type App struct {
	mu      sync.Mutex
	state   State
	dir     string
	client  *http.Client
	running map[string]bool
}

type Op struct {
	Op, SessionID, IdentityID, Title, Goal, Status, Scope, Text, ID, Action, Reason, Decision string
	EverySeconds                                                                              int
}

func timestamp() string { return time.Now().UTC().Format(time.RFC3339) }

func id(prefix string) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b)
}

func env(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}

func heartbeatInterval(seconds int) int {
	if seconds < 30 {
		return 30
	}
	if seconds > 86400 {
		return 86400
	}
	return seconds
}

func newApp() (*App, error) {
	dir := os.Getenv("AXM_WORKSHOP_DATA")
	if dir == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(h, ".axm-workshop")
	}
	if err := os.MkdirAll(filepath.Join(dir, "media"), 0o700); err != nil {
		return nil, err
	}
	a := &App{dir: dir, client: &http.Client{Timeout: 120 * time.Second}, running: map[string]bool{}}
	if b, err := os.ReadFile(filepath.Join(dir, "state.json")); err == nil {
		if err := json.Unmarshal(b, &a.state); err != nil {
			return nil, err
		}
		if a.normalizeState() {
			if err := a.save(); err != nil {
				return nil, err
			}
		}
	} else if errors.Is(err, os.ErrNotExist) {
		identity := Identity{ID: "waldo", Name: "Waldo", Model: env("AXM_AI_MODEL", "waldo")}
		a.state = State{
			Version:    2,
			Identities: []Identity{identity},
			Sessions: []Session{{
				ID:                id("session"),
				IdentityID:        "waldo",
				Title:             "Waldo Mirror",
				Heartbeat:         "paused",
				RuntimeMode:       "paused",
				HeartbeatEverySec: 300,
				CreatedAt:         timestamp(),
				Messages:          []Message{},
			}},
			Memories: []Memory{}, Consents: []Consent{}, Media: []Media{},
		}
		if err := a.save(); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}
	return a, nil
}

func (a *App) normalizeState() bool {
	changed := false
	if a.state.Version < 2 {
		a.state.Version = 2
		changed = true
	}
	for i := range a.state.Sessions {
		s := &a.state.Sessions[i]
		if s.RuntimeMode != "active" && s.RuntimeMode != "paused" {
			s.RuntimeMode = "paused"
			changed = true
		}
		if s.HeartbeatEverySec <= 0 {
			s.HeartbeatEverySec = 300
			changed = true
		}
		s.HeartbeatEverySec = heartbeatInterval(s.HeartbeatEverySec)
		if s.RuntimeMode == "paused" && s.NextHeartbeat != "" {
			s.NextHeartbeat = ""
			changed = true
		}
		if s.Heartbeat == "" || (s.RuntimeMode == "paused" && s.Heartbeat == "idle") {
			s.Heartbeat = s.RuntimeMode
			changed = true
		}
	}
	return changed
}

func (a *App) save() error {
	b, err := json.MarshalIndent(a.state, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(a.dir, "state.json.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(a.dir, "state.json"))
}

func out(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, status int, err error) {
	out(w, status, map[string]string{"error": err.Error()})
}
func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(v)
}
func (a *App) sessionIndex(id string) int {
	for i := range a.state.Sessions {
		if a.state.Sessions[i].ID == id {
			return i
		}
	}
	return -1
}
func (a *App) identity(id string) (Identity, bool) {
	for _, x := range a.state.Identities {
		if x.ID == id {
			return x, true
		}
	}
	return Identity{}, false
}

func (a *App) routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		b, _ := web.ReadFile("ui.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	})
	m.HandleFunc("GET /api/state", func(w http.ResponseWriter, _ *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		out(w, 200, a.state)
	})
	m.HandleFunc("POST /api/op", a.handleOp)
	m.HandleFunc("POST /api/upload", a.upload)
	m.HandleFunc("GET /media/{id}", a.media)
	m.HandleFunc("POST /api/chat", a.chat)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		m.ServeHTTP(w, r)
	})
}

func (a *App) handleOp(w http.ResponseWriter, r *http.Request) {
	var x Op
	if err := readJSON(r, &x); err != nil {
		fail(w, 400, err)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	si := a.sessionIndex(x.SessionID)
	switch x.Op {
	case "new_session":
		if _, ok := a.identity(x.IdentityID); !ok {
			fail(w, 400, errors.New("unknown identity"))
			return
		}
		title := strings.TrimSpace(x.Title)
		if title == "" {
			title = "New session"
		}
		s := Session{ID: id("session"), IdentityID: x.IdentityID, Title: title, Heartbeat: "paused", RuntimeMode: "paused", HeartbeatEverySec: 300, CreatedAt: timestamp(), Messages: []Message{}}
		a.state.Sessions = append(a.state.Sessions, s)
		if err := a.save(); err != nil {
			fail(w, 500, err)
			return
		}
		out(w, 201, s)
	case "goal":
		if si < 0 {
			fail(w, 404, errors.New("session not found"))
			return
		}
		a.state.Sessions[si].Goal = strings.TrimSpace(x.Goal)
		if err := a.save(); err != nil {
			fail(w, 500, err)
			return
		}
		out(w, 200, a.state.Sessions[si])
	case "pulse":
		if si < 0 {
			fail(w, 404, errors.New("session not found"))
			return
		}
		if strings.TrimSpace(x.Status) == "" {
			x.Status = "pulse"
		}
		a.state.Sessions[si].Heartbeat, a.state.Sessions[si].LastHeartbeat = x.Status, timestamp()
		if err := a.save(); err != nil {
			fail(w, 500, err)
			return
		}
		out(w, 200, a.state.Sessions[si])
	case "runtime":
		if si < 0 {
			fail(w, 404, errors.New("session not found"))
			return
		}
		if x.Status != "active" && x.Status != "paused" {
			fail(w, 400, errors.New("runtime status must be active or paused"))
			return
		}
		s := &a.state.Sessions[si]
		if x.Status == "active" && strings.TrimSpace(s.Goal) == "" {
			fail(w, 400, errors.New("set a goal before starting background runtime"))
			return
		}
		if x.EverySeconds > 0 {
			s.HeartbeatEverySec = heartbeatInterval(x.EverySeconds)
		} else if s.HeartbeatEverySec <= 0 {
			s.HeartbeatEverySec = 300
		}
		s.RuntimeMode = x.Status
		s.LastRuntimeError = ""
		if x.Status == "active" {
			s.Heartbeat = "active"
			s.NextHeartbeat = time.Now().UTC().Add(2 * time.Second).Format(time.RFC3339)
		} else {
			s.Heartbeat = "paused"
			s.NextHeartbeat = ""
		}
		if err := a.save(); err != nil {
			fail(w, 500, err)
			return
		}
		out(w, 200, *s)
	case "memory_add":
		if x.Scope != "session" && x.Scope != "identity" && x.Scope != "vault" {
			fail(w, 400, errors.New("bad memory scope"))
			return
		}
		if strings.TrimSpace(x.Text) == "" {
			fail(w, 400, errors.New("memory is empty"))
			return
		}
		m := Memory{ID: id("memory"), Scope: x.Scope, Text: strings.TrimSpace(x.Text), CreatedAt: timestamp()}
		if x.Scope == "session" {
			m.SessionID = x.SessionID
		}
		if x.Scope == "identity" {
			m.IdentityID = x.IdentityID
		}
		a.state.Memories = append(a.state.Memories, m)
		if err := a.save(); err != nil {
			fail(w, 500, err)
			return
		}
		out(w, 201, m)
	case "memory_delete":
		for i, m := range a.state.Memories {
			if m.ID == x.ID {
				a.state.Memories = append(a.state.Memories[:i], a.state.Memories[i+1:]...)
				if err := a.save(); err != nil {
					fail(w, 500, err)
					return
				}
				out(w, 200, map[string]bool{"deleted": true})
				return
			}
		}
		fail(w, 404, errors.New("memory not found"))
	case "consent_add":
		if si < 0 {
			fail(w, 404, errors.New("session not found"))
			return
		}
		if strings.TrimSpace(x.Action) == "" {
			fail(w, 400, errors.New("action is empty"))
			return
		}
		c := Consent{ID: id("consent"), SessionID: x.SessionID, Action: strings.TrimSpace(x.Action), Reason: strings.TrimSpace(x.Reason), Status: "pending", CreatedAt: timestamp()}
		a.state.Consents = append(a.state.Consents, c)
		if err := a.save(); err != nil {
			fail(w, 500, err)
			return
		}
		out(w, 201, c)
	case "consent_decide":
		if x.Decision != "approved" && x.Decision != "rejected" {
			fail(w, 400, errors.New("decision must be approved or rejected"))
			return
		}
		for i := range a.state.Consents {
			if a.state.Consents[i].ID == x.ID {
				a.state.Consents[i].Status, a.state.Consents[i].DecidedAt = x.Decision, timestamp()
				if err := a.save(); err != nil {
					fail(w, 500, err)
					return
				}
				out(w, 200, a.state.Consents[i])
				return
			}
		}
		fail(w, 404, errors.New("consent request not found"))
	default:
		fail(w, 400, errors.New("unknown operation"))
	}
}

func (a *App) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 25<<20)
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		fail(w, 400, err)
		return
	}
	sid := r.FormValue("session_id")
	f, h, err := r.FormFile("file")
	if err != nil {
		fail(w, 400, err)
		return
	}
	defer f.Close()
	a.mu.Lock()
	si := a.sessionIndex(sid)
	if si < 0 {
		a.mu.Unlock()
		fail(w, 404, errors.New("session not found"))
		return
	}
	iid := a.state.Sessions[si].IdentityID
	a.mu.Unlock()
	head := make([]byte, 512)
	n, _ := f.Read(head)
	mime := http.DetectContentType(head[:n])
	if !strings.HasPrefix(mime, "image/") {
		fail(w, 400, errors.New("creative room accepts images only"))
		return
	}
	mid := id("media")
	ext := filepath.Ext(h.Filename)
	if ext == "" {
		ext = ".img"
	}
	name := mid + ext
	p := filepath.Join(a.dir, "media", name)
	o, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		fail(w, 500, err)
		return
	}
	_, e1 := o.Write(head[:n])
	if e1 == nil {
		_, e1 = io.Copy(o, f)
	}
	e2 := o.Close()
	if e1 != nil {
		_ = os.Remove(p)
		fail(w, 500, e1)
		return
	}
	if e2 != nil {
		_ = os.Remove(p)
		fail(w, 500, e2)
		return
	}
	m := Media{ID: mid, SessionID: sid, IdentityID: iid, Name: filepath.Base(h.Filename), MIME: mime, FileName: name, URL: "/media/" + mid, CreatedAt: timestamp()}
	a.mu.Lock()
	a.state.Media = append(a.state.Media, m)
	err = a.save()
	a.mu.Unlock()
	if err != nil {
		fail(w, 500, err)
		return
	}
	out(w, 201, m)
}

func (a *App) media(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a.mu.Lock()
	var m *Media
	for _, x := range a.state.Media {
		if x.ID == id {
			y := x
			m = &y
			break
		}
	}
	a.mu.Unlock()
	if m == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", m.MIME)
	http.ServeFile(w, r, filepath.Join(a.dir, "media", m.FileName))
}

func (a *App) chat(w http.ResponseWriter, r *http.Request) {
	var x struct {
		SessionID, Text string
		MediaIDs        []string `json:"media_ids"`
	}
	if err := readJSON(r, &x); err != nil {
		fail(w, 400, err)
		return
	}
	if strings.TrimSpace(x.Text) == "" && len(x.MediaIDs) == 0 {
		fail(w, 400, errors.New("message is empty"))
		return
	}
	a.mu.Lock()
	si := a.sessionIndex(x.SessionID)
	if si < 0 {
		a.mu.Unlock()
		fail(w, 404, errors.New("session not found"))
		return
	}
	u := Message{ID: id("msg"), Role: "user", Text: strings.TrimSpace(x.Text), MediaIDs: x.MediaIDs, CreatedAt: timestamp()}
	a.state.Sessions[si].Messages = append(a.state.Sessions[si].Messages, u)
	if err := a.save(); err != nil {
		a.mu.Unlock()
		fail(w, 500, err)
		return
	}
	s := a.state.Sessions[si]
	ident, ok := a.identity(s.IdentityID)
	memories := append([]Memory(nil), a.state.Memories...)
	media := append([]Media(nil), a.state.Media...)
	a.mu.Unlock()
	if !ok {
		fail(w, 500, errors.New("identity missing"))
		return
	}
	reply, err := a.generate(r.Context(), ident, s, memories, media)
	if err != nil {
		fail(w, 502, err)
		return
	}
	m := Message{ID: id("msg"), Role: "assistant", Text: reply, CreatedAt: timestamp()}
	a.mu.Lock()
	si = a.sessionIndex(x.SessionID)
	if si < 0 {
		a.mu.Unlock()
		fail(w, 409, errors.New("session disappeared"))
		return
	}
	a.state.Sessions[si].Messages = append(a.state.Sessions[si].Messages, m)
	err = a.save()
	a.mu.Unlock()
	if err != nil {
		fail(w, 500, err)
		return
	}
	out(w, 200, map[string]any{"assistant": m})
}

func (a *App) generate(ctx context.Context, ident Identity, s Session, memories []Memory, media []Media) (string, error) {
	model := env("AXM_AI_MODEL", ident.Model)
	base := strings.TrimRight(env("AXM_AI_BASE_URL", "http://127.0.0.1:11434/v1"), "/")
	system := "You are " + ident.Name + " inside a local user-controlled AI workshop."
	if s.Goal != "" {
		system += "\nCurrent goal: " + s.Goal
	}
	system += "\nVisible memory is explicitly stored by the user. Background goal heartbeat may ask you to continue thinking or planning toward the goal. Consent, memory, app, and file actions are controlled by the host; never claim they happened unless the host reports success."
	for _, m := range memories {
		if m.Scope == "vault" || (m.Scope == "identity" && m.IdentityID == s.IdentityID) || (m.Scope == "session" && m.SessionID == s.ID) {
			system += "\n- [" + m.Scope + "] " + m.Text
		}
	}
	msgs := []map[string]any{{"role": "system", "content": system}}
	byID := map[string]Media{}
	for _, m := range media {
		byID[m.ID] = m
	}
	start := 0
	if len(s.Messages) > 30 {
		start = len(s.Messages) - 30
	}
	for _, m := range s.Messages[start:] {
		if len(m.MediaIDs) == 0 {
			msgs = append(msgs, map[string]any{"role": m.Role, "content": m.Text})
			continue
		}
		parts := []map[string]any{{"type": "text", "text": m.Text}}
		for _, mid := range m.MediaIDs {
			x, ok := byID[mid]
			if !ok {
				continue
			}
			b, err := os.ReadFile(filepath.Join(a.dir, "media", x.FileName))
			if err != nil {
				continue
			}
			parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]string{"url": "data:" + x.MIME + ";base64," + base64.StdEncoding.EncodeToString(b)}})
		}
		msgs = append(msgs, map[string]any{"role": m.Role, "content": parts})
	}
	body, _ := json.Marshal(map[string]any{"model": model, "messages": msgs, "stream": false})
	req, err := http.NewRequestWithContext(ctx, "POST", base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if key := os.Getenv("AXM_AI_KEY"); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("AI endpoint unavailable: %w", err)
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("AI endpoint returned %s: %s", resp.Status, strings.TrimSpace(string(rb)))
	}
	var result struct {
		Choices []struct {
			Message struct{ Content json.RawMessage }
		}
	}
	if err := json.Unmarshal(rb, &result); err != nil || len(result.Choices) == 0 {
		return "", errors.New("invalid AI response")
	}
	var text string
	if json.Unmarshal(result.Choices[0].Message.Content, &text) != nil {
		var p []struct{ Text string }
		_ = json.Unmarshal(result.Choices[0].Message.Content, &p)
		for _, x := range p {
			if text != "" {
				text += "\n"
			}
			text += x.Text
		}
	}
	if strings.TrimSpace(text) == "" {
		return "", errors.New("AI returned an empty message")
	}
	return text, nil
}

func (a *App) runtimeLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.startDueHeartbeats(ctx)
		}
	}
}

func (a *App) startDueHeartbeats(ctx context.Context) {
	now := time.Now().UTC()
	due := []string{}
	changed := false
	a.mu.Lock()
	for i := range a.state.Sessions {
		s := &a.state.Sessions[i]
		if s.RuntimeMode != "active" || strings.TrimSpace(s.Goal) == "" || a.running[s.ID] {
			continue
		}
		if s.NextHeartbeat == "" {
			s.NextHeartbeat = now.Add(2 * time.Second).Format(time.RFC3339)
			changed = true
			continue
		}
		next, err := time.Parse(time.RFC3339, s.NextHeartbeat)
		if err != nil {
			s.NextHeartbeat = now.Add(2 * time.Second).Format(time.RFC3339)
			changed = true
			continue
		}
		if next.After(now) {
			continue
		}
		a.running[s.ID] = true
		s.Heartbeat = "working"
		s.LastHeartbeat = timestamp()
		due = append(due, s.ID)
		changed = true
	}
	if changed {
		_ = a.save()
	}
	a.mu.Unlock()
	for _, sid := range due {
		go a.backgroundHeartbeat(ctx, sid)
	}
}

func (a *App) backgroundHeartbeat(ctx context.Context, sessionID string) {
	a.mu.Lock()
	si := a.sessionIndex(sessionID)
	if si < 0 {
		delete(a.running, sessionID)
		a.mu.Unlock()
		return
	}
	s := a.state.Sessions[si]
	ident, ok := a.identity(s.IdentityID)
	memories := append([]Memory(nil), a.state.Memories...)
	media := append([]Media(nil), a.state.Media...)
	a.mu.Unlock()
	if !ok {
		a.finishHeartbeat(sessionID, "", errors.New("identity missing"))
		return
	}
	s.Messages = append(s.Messages, Message{Role: "user", Text: "[Background goal heartbeat] Continue pursuing the current goal from the existing conversation and explicit memory. Produce a concise progress note: what matters now, the best next step, and any question or consent you need from the user. Do not claim any tool, app, file, or external action happened.", CreatedAt: timestamp()})
	reply, err := a.generate(ctx, ident, s, memories, media)
	a.finishHeartbeat(sessionID, reply, err)
}

func (a *App) finishHeartbeat(sessionID, reply string, runErr error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	defer delete(a.running, sessionID)
	si := a.sessionIndex(sessionID)
	if si < 0 {
		return
	}
	s := &a.state.Sessions[si]
	s.LastHeartbeat = timestamp()
	if runErr != nil {
		s.LastRuntimeError = runErr.Error()
		if s.RuntimeMode == "active" {
			s.Heartbeat = "error"
			s.NextHeartbeat = time.Now().UTC().Add(time.Duration(heartbeatInterval(s.HeartbeatEverySec)) * time.Second).Format(time.RFC3339)
		} else {
			s.Heartbeat = "paused"
			s.NextHeartbeat = ""
		}
		_ = a.save()
		return
	}
	s.PulseCount++
	s.LastRuntimeError = ""
	s.Messages = append(s.Messages, Message{ID: id("msg"), Role: "assistant", Text: "⏱ Goal heartbeat\n" + strings.TrimSpace(reply), CreatedAt: timestamp()})
	if s.RuntimeMode == "active" {
		s.Heartbeat = "active"
		s.NextHeartbeat = time.Now().UTC().Add(time.Duration(heartbeatInterval(s.HeartbeatEverySec)) * time.Second).Format(time.RFC3339)
	} else {
		s.Heartbeat = "paused"
		s.NextHeartbeat = ""
	}
	_ = a.save()
}

func main() {
	a, err := newApp()
	if err != nil {
		log.Fatal(err)
	}
	addr := "127.0.0.1:" + env("AXM_WORKSHOP_PORT", "7788")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go a.runtimeLoop(ctx)

	srv := &http.Server{Addr: addr, Handler: a.routes(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("AXM Local Workshop: http://%s", addr)
	log.Printf("data: %s", a.dir)
	log.Printf("AI: %s · %s", env("AXM_AI_BASE_URL", "http://127.0.0.1:11434/v1"), env("AXM_AI_MODEL", "waldo"))
	log.Printf("background goal runtime: enabled for sessions explicitly set active")
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
