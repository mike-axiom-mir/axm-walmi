package main

import (
	"strings"
	"testing"
)

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
