package main

import "testing"

func TestHeartbeatIntervalBounds(t *testing.T) {
	cases := []struct {
		in, want int
	}{{0, 30}, {5, 30}, {30, 30}, {300, 300}, {86400, 86400}, {999999, 86400}}
	for _, tc := range cases {
		if got := heartbeatInterval(tc.in); got != tc.want {
			t.Fatalf("heartbeatInterval(%d)=%d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeStateMigratesExistingSession(t *testing.T) {
	a := &App{state: State{Version: 1, Sessions: []Session{{ID: "s1", Heartbeat: "idle"}}}}
	if !a.normalizeState() {
		t.Fatal("expected migration to change state")
	}
	s := a.state.Sessions[0]
	if a.state.Version != 2 || s.RuntimeMode != "paused" || s.HeartbeatEverySec != 300 || s.Heartbeat != "paused" {
		t.Fatalf("unexpected migrated state: version=%d session=%+v", a.state.Version, s)
	}
}
