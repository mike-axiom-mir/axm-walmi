package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	workshopStateKind       = "axm.workshop.state"
	workshopStateSchema     = 1
	workshopStateFile       = "state.json"
	workshopStateBackupFile = "state.json.backup"
)

type workshopStateEnvelope struct {
	Kind        string `json:"kind"`
	Schema      int    `json:"schema"`
	StateSHA256 string `json:"state_sha256"`
	State       State  `json:"state"`
}

type workshopStateLoad struct {
	State            State
	Found            bool
	NeedsCheckpoint  bool
	RecoveredBackup  bool
	RejectedStateRef string
}

func loadWorkshopState(dir string) (workshopStateLoad, error) {
	currentPath := filepath.Join(dir, workshopStateFile)
	current, err := os.ReadFile(currentPath)
	if err == nil {
		state, legacy, decodeErr := decodeWorkshopState(current)
		if decodeErr == nil {
			return workshopStateLoad{State: state, Found: true, NeedsCheckpoint: legacy}, nil
		}
		return recoverWorkshopState(dir, current, fmt.Errorf("verify %s: %w", currentPath, decodeErr))
	}
	if !errors.Is(err, os.ErrNotExist) {
		return workshopStateLoad{}, fmt.Errorf("read %s: %w", currentPath, err)
	}

	backupPath := filepath.Join(dir, workshopStateBackupFile)
	if _, backupErr := os.Stat(backupPath); errors.Is(backupErr, os.ErrNotExist) {
		return workshopStateLoad{}, nil
	} else if backupErr != nil {
		return workshopStateLoad{}, fmt.Errorf("inspect %s: %w", backupPath, backupErr)
	}
	return recoverWorkshopState(dir, nil, fmt.Errorf("%s is missing", currentPath))
}

func recoverWorkshopState(dir string, rejected []byte, currentErr error) (workshopStateLoad, error) {
	backupPath := filepath.Join(dir, workshopStateBackupFile)
	backup, err := os.ReadFile(backupPath)
	if err != nil {
		return workshopStateLoad{}, fmt.Errorf("workshop state is unavailable (%v); read backup %s: %w", currentErr, backupPath, err)
	}
	state, _, err := decodeWorkshopState(backup)
	if err != nil {
		return workshopStateLoad{}, fmt.Errorf("workshop state is unavailable (%v); verify backup %s: %w", currentErr, backupPath, err)
	}

	ref := ""
	if rejected != nil {
		ref, err = quarantineRejectedState(dir, rejected)
		if err != nil {
			return workshopStateLoad{}, fmt.Errorf("preserve rejected workshop state: %w", err)
		}
	}
	return workshopStateLoad{
		State:            state,
		Found:            true,
		NeedsCheckpoint:  true,
		RecoveredBackup:  true,
		RejectedStateRef: ref,
	}, nil
}

func decodeWorkshopState(data []byte) (State, bool, error) {
	var shape map[string]json.RawMessage
	if err := decodeStrictJSON(data, &shape); err != nil {
		return State{}, false, err
	}
	if _, ok := shape["kind"]; !ok {
		var legacy State
		if err := decodeStrictJSON(data, &legacy); err != nil {
			return State{}, false, fmt.Errorf("decode legacy state: %w", err)
		}
		if legacy.Version < 0 || legacy.Version > 2 {
			return State{}, false, fmt.Errorf("unsupported legacy state version %d", legacy.Version)
		}
		normalizeWorkshopState(&legacy)
		if err := validateWorkshopState(legacy); err != nil {
			return State{}, false, fmt.Errorf("validate legacy state: %w", err)
		}
		return legacy, true, nil
	}

	var envelope workshopStateEnvelope
	if err := decodeStrictJSON(data, &envelope); err != nil {
		return State{}, false, fmt.Errorf("decode state envelope: %w", err)
	}
	if envelope.Kind != workshopStateKind {
		return State{}, false, fmt.Errorf("state kind %q is not %q", envelope.Kind, workshopStateKind)
	}
	if envelope.Schema != workshopStateSchema {
		return State{}, false, fmt.Errorf("unsupported state envelope schema %d", envelope.Schema)
	}
	stateBytes, err := json.Marshal(envelope.State)
	if err != nil {
		return State{}, false, fmt.Errorf("encode state for verification: %w", err)
	}
	want := sha256Hex(stateBytes)
	if envelope.StateSHA256 != want {
		return State{}, false, fmt.Errorf("state digest mismatch: recorded %q, computed %q", envelope.StateSHA256, want)
	}
	if err := validateWorkshopState(envelope.State); err != nil {
		return State{}, false, err
	}
	return envelope.State, false, nil
}

func encodeWorkshopState(state State) ([]byte, error) {
	if err := validateWorkshopState(state); err != nil {
		return nil, err
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("encode workshop state: %w", err)
	}
	envelope := workshopStateEnvelope{
		Kind:        workshopStateKind,
		Schema:      workshopStateSchema,
		StateSHA256: sha256Hex(stateBytes),
		State:       state,
	}
	encoded, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode workshop state envelope: %w", err)
	}
	return append(encoded, '\n'), nil
}

func saveWorkshopState(dir string, state State) error {
	encoded, err := encodeWorkshopState(state)
	if err != nil {
		return err
	}
	currentPath := filepath.Join(dir, workshopStateFile)
	if current, readErr := os.ReadFile(currentPath); readErr == nil {
		if _, _, verifyErr := decodeWorkshopState(current); verifyErr == nil {
			if err := writeAtomicFile(filepath.Join(dir, workshopStateBackupFile), current); err != nil {
				return fmt.Errorf("checkpoint previous workshop state: %w", err)
			}
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("read previous workshop state: %w", readErr)
	}
	if err := writeAtomicFile(currentPath, encoded); err != nil {
		return fmt.Errorf("commit workshop state: %w", err)
	}
	return nil
}

func quarantineRejectedState(dir string, rejected []byte) (string, error) {
	digest := sha256Hex(rejected)
	recoveryDir := filepath.Join(dir, "recovery")
	if err := os.MkdirAll(recoveryDir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(recoveryDir, "state-sha256-"+digest+".rejected.json")
	if existing, err := os.ReadFile(path); err == nil {
		if !bytes.Equal(existing, rejected) {
			return "", fmt.Errorf("recovery object %s does not match its content identity", path)
		}
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := writeAtomicFile(path, rejected); err != nil {
		return "", err
	}
	return path, nil
}

func writeAtomicFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporary)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	removeTemporary = false
	return syncDirectory(dir)
}

func syncDirectory(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	file, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

func validateWorkshopState(state State) error {
	if state.Version != 2 {
		return fmt.Errorf("workshop state version %d is not supported", state.Version)
	}
	identities := make(map[string]struct{}, len(state.Identities))
	if len(state.Identities) == 0 {
		return errors.New("workshop state has no identities")
	}
	for _, identity := range state.Identities {
		if err := addUniqueID("identity", identity.ID, identities); err != nil {
			return err
		}
		if strings.TrimSpace(identity.Name) == "" {
			return fmt.Errorf("identity %q has no name", identity.ID)
		}
	}
	sessions := make(map[string]struct{}, len(state.Sessions))
	for _, session := range state.Sessions {
		if err := addUniqueID("session", session.ID, sessions); err != nil {
			return err
		}
		if _, ok := identities[session.IdentityID]; !ok {
			return fmt.Errorf("session %q references unknown identity %q", session.ID, session.IdentityID)
		}
		if session.RuntimeMode != "active" && session.RuntimeMode != "paused" {
			return fmt.Errorf("session %q has invalid runtime mode %q", session.ID, session.RuntimeMode)
		}
		if session.HeartbeatEverySec < 30 || session.HeartbeatEverySec > 86400 {
			return fmt.Errorf("session %q has invalid heartbeat interval %d", session.ID, session.HeartbeatEverySec)
		}
	}
	memories := make(map[string]struct{}, len(state.Memories))
	for _, memory := range state.Memories {
		if err := addUniqueID("memory", memory.ID, memories); err != nil {
			return err
		}
		switch memory.Scope {
		case "session":
			if _, ok := sessions[memory.SessionID]; !ok {
				return fmt.Errorf("memory %q references unknown session %q", memory.ID, memory.SessionID)
			}
		case "identity":
			if _, ok := identities[memory.IdentityID]; !ok {
				return fmt.Errorf("memory %q references unknown identity %q", memory.ID, memory.IdentityID)
			}
		case "vault":
		default:
			return fmt.Errorf("memory %q has invalid scope %q", memory.ID, memory.Scope)
		}
	}
	consents := make(map[string]struct{}, len(state.Consents))
	for _, consent := range state.Consents {
		if err := addUniqueID("consent", consent.ID, consents); err != nil {
			return err
		}
		if _, ok := sessions[consent.SessionID]; !ok {
			return fmt.Errorf("consent %q references unknown session %q", consent.ID, consent.SessionID)
		}
		if consent.Status != "pending" && consent.Status != "approved" && consent.Status != "rejected" {
			return fmt.Errorf("consent %q has invalid status %q", consent.ID, consent.Status)
		}
	}
	mediaIDs := make(map[string]struct{}, len(state.Media))
	for _, media := range state.Media {
		if err := addUniqueID("media", media.ID, mediaIDs); err != nil {
			return err
		}
		if _, ok := sessions[media.SessionID]; !ok {
			return fmt.Errorf("media %q references unknown session %q", media.ID, media.SessionID)
		}
		if _, ok := identities[media.IdentityID]; !ok {
			return fmt.Errorf("media %q references unknown identity %q", media.ID, media.IdentityID)
		}
		if media.FileName == "" || filepath.Base(media.FileName) != media.FileName {
			return fmt.Errorf("media %q has unsafe file name %q", media.ID, media.FileName)
		}
	}
	return nil
}

func addUniqueID(kind, value string, seen map[string]struct{}) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s has an empty ID", kind)
	}
	if _, ok := seen[value]; ok {
		return fmt.Errorf("duplicate %s ID %q", kind, value)
	}
	seen[value] = struct{}{}
	return nil
}

func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return fmt.Errorf("trailing JSON data: %w", err)
	}
	return nil
}

func sha256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
