package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validWorkshopState(title string) State {
	return State{
		Version:    2,
		Identities: []Identity{{ID: "waldo", Name: "Waldo", Model: "waldo"}},
		Sessions: []Session{{
			ID:                "session-1",
			IdentityID:        "waldo",
			Title:             title,
			Heartbeat:         "paused",
			RuntimeMode:       "paused",
			HeartbeatEverySec: 300,
			CreatedAt:         "2026-09-09T06:00:00Z",
			Messages:          []Message{},
		}},
		Memories: []Memory{},
		Consents: []Consent{},
		Media:    []Media{},
	}
}

func TestWorkshopStateEnvelopeRoundTrip(t *testing.T) {
	want := validWorkshopState("round trip")
	encoded, err := encodeWorkshopState(want)
	if err != nil {
		t.Fatal(err)
	}
	got, legacy, err := decodeWorkshopState(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if legacy {
		t.Fatal("new envelope was reported as legacy state")
	}
	if got.Sessions[0].Title != want.Sessions[0].Title {
		t.Fatalf("title=%q, want %q", got.Sessions[0].Title, want.Sessions[0].Title)
	}
	var envelope workshopStateEnvelope
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Kind != workshopStateKind || envelope.Schema != workshopStateSchema || len(envelope.StateSHA256) != 64 {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}

func TestWorkshopStateEnvelopeRejectsTampering(t *testing.T) {
	encoded, err := encodeWorkshopState(validWorkshopState("before"))
	if err != nil {
		t.Fatal(err)
	}
	tampered := bytes.Replace(encoded, []byte("before"), []byte("after!"), 1)
	if _, _, err := decodeWorkshopState(tampered); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch, got %v", err)
	}
}

func TestLegacyStateMigratesIntoCheckpointEnvelope(t *testing.T) {
	dir := t.TempDir()
	legacy := validWorkshopState("legacy")
	legacy.Version = 1
	legacy.Sessions[0].RuntimeMode = ""
	legacy.Sessions[0].HeartbeatEverySec = 0
	legacyBytes, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, workshopStateFile), legacyBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := loadWorkshopState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Found || !loaded.NeedsCheckpoint || loaded.State.Version != 2 {
		t.Fatalf("unexpected legacy load: %+v", loaded)
	}
	if err := saveWorkshopState(dir, loaded.State); err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(filepath.Join(dir, workshopStateFile))
	if err != nil {
		t.Fatal(err)
	}
	if _, legacy, err := decodeWorkshopState(current); err != nil || legacy {
		t.Fatalf("migrated current state: legacy=%v err=%v", legacy, err)
	}
	backup, err := os.ReadFile(filepath.Join(dir, workshopStateBackupFile))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(backup, legacyBytes) {
		t.Fatal("legacy bytes were not retained as the first backup")
	}
}

func TestWorkshopStateRecoversVerifiedBackupAndQuarantinesRejectedBytes(t *testing.T) {
	dir := t.TempDir()
	first := validWorkshopState("first")
	second := validWorkshopState("second")
	if err := saveWorkshopState(dir, first); err != nil {
		t.Fatal(err)
	}
	if err := saveWorkshopState(dir, second); err != nil {
		t.Fatal(err)
	}
	rejected := []byte(`{"kind":"axm.workshop.state","schema":1,"state_sha256":"cut off"`)
	if err := os.WriteFile(filepath.Join(dir, workshopStateFile), rejected, 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := loadWorkshopState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.RecoveredBackup || !loaded.NeedsCheckpoint || loaded.State.Sessions[0].Title != "first" {
		t.Fatalf("unexpected recovery load: %+v", loaded)
	}
	if loaded.RejectedStateRef == "" {
		t.Fatal("rejected state was not quarantined")
	}
	preserved, err := os.ReadFile(loaded.RejectedStateRef)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(preserved, rejected) {
		t.Fatal("quarantine did not preserve the exact rejected bytes")
	}
	if err := saveWorkshopState(dir, loaded.State); err != nil {
		t.Fatal(err)
	}
	reloaded, err := loadWorkshopState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.RecoveredBackup || reloaded.State.Sessions[0].Title != "first" {
		t.Fatalf("recovered state was not re-established as current: %+v", reloaded)
	}
}

func TestWorkshopStateRecoversWhenCurrentCheckpointIsMissing(t *testing.T) {
	dir := t.TempDir()
	state := validWorkshopState("backup only")
	encoded, err := encodeWorkshopState(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, workshopStateBackupFile), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadWorkshopState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.RecoveredBackup || loaded.RejectedStateRef != "" || loaded.State.Sessions[0].Title != "backup only" {
		t.Fatalf("unexpected missing-current recovery: %+v", loaded)
	}
}

func TestWorkshopStateFailsClosedWhenCurrentAndBackupAreInvalid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, workshopStateFile), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, workshopStateBackupFile), []byte("{also broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadWorkshopState(dir); err == nil || !strings.Contains(err.Error(), "verify backup") {
		t.Fatalf("expected fail-closed backup error, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "recovery")); !os.IsNotExist(err) {
		t.Fatalf("invalid state was moved without a usable backup: %v", err)
	}
}

func TestWorkshopStateRejectsBrokenReferencesAndUnknownFields(t *testing.T) {
	state := validWorkshopState("invalid")
	state.Sessions[0].IdentityID = "missing"
	if _, err := encodeWorkshopState(state); err == nil || !strings.Contains(err.Error(), "unknown identity") {
		t.Fatalf("expected broken reference error, got %v", err)
	}
	encoded, err := encodeWorkshopState(validWorkshopState("strict"))
	if err != nil {
		t.Fatal(err)
	}
	withUnknown := bytes.Replace(encoded, []byte(`"schema": 1,`), []byte(`"schema": 1, "surprise": true,`), 1)
	if _, _, err := decodeWorkshopState(withUnknown); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field error, got %v", err)
	}
}
