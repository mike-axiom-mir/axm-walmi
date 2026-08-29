package axmmirror

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	pathpkg "path"
	"reflect"
	"sort"
	"strings"
)

const (
	OriginAnchorSchema = "axm.waldo-witness.origin-anchor/v0.1"
	IdentityLockSchema = "axm.waldo-witness.release-identity-lock/v0.1"

	AnchorStateAnchored = "ANCHORED"
	AnchorStateHold     = "HOLD"

	IdentityStateLocked = "LOCKED"
	IdentityStateDrift  = "DRIFT"
	IdentityStateHold   = "HOLD"
)

// AnchoredArtifact is the bounded artifact identity carried from the selected
// WALDO origin, run, or release. Paths remain relative to their BOM root.
type AnchoredArtifact struct {
	Role   string `json:"role"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

// OriginAnchor is a read-only projection of one validated WALDO model or
// model-release BOM. BOMSHA256 uses WALDO's canonical JSON representation;
// DocumentSHA256 additionally binds the exact file bytes that were observed.
type OriginAnchor struct {
	Schema                  string             `json:"schema"`
	State                   string             `json:"state"`
	Subject                 string             `json:"subject"`
	BOMSHA256               string             `json:"bom_sha256"`
	DocumentSHA256          string             `json:"document_sha256"`
	AnsweringIdentitySHA256 string             `json:"answering_identity_sha256"`
	ModelID                 string             `json:"model_id"`
	Name                    string             `json:"name"`
	ArchitectureSHA256      string             `json:"architecture_sha256,omitempty"`
	PlanSHA256              string             `json:"plan_sha256,omitempty"`
	Format                  string             `json:"format,omitempty"`
	SourceType              string             `json:"source_type,omitempty"`
	SourceID                string             `json:"source_id,omitempty"`
	RunID                   string             `json:"run_id,omitempty"`
	SourceBOMSHA256         string             `json:"source_bom_sha256,omitempty"`
	Simulated               bool               `json:"simulated"`
	Artifacts               []AnchoredArtifact `json:"artifacts,omitempty"`
	Holds                   []string           `json:"holds,omitempty"`
	Authority               Authority          `json:"authority"`
}

// IdentityAnchorRef keeps an identity-lock receipt compact while retaining
// both the selected answering identity and its exact provenance document.
type IdentityAnchorRef struct {
	Subject                 string `json:"subject"`
	ModelID                 string `json:"model_id"`
	BOMSHA256               string `json:"bom_sha256"`
	AnsweringIdentitySHA256 string `json:"answering_identity_sha256"`
}

// IdentityLockReceipt compares two anchors without mutating either one.
// ProvenanceChanged may be true while State remains LOCKED when append-only
// history changed but the selected answering model did not.
type IdentityLockReceipt struct {
	Schema            string            `json:"schema"`
	State             string            `json:"state"`
	Expected          IdentityAnchorRef `json:"expected"`
	Observed          IdentityAnchorRef `json:"observed"`
	Changes           []string          `json:"changes,omitempty"`
	Notices           []string          `json:"notices,omitempty"`
	ProvenanceChanged bool              `json:"provenance_changed"`
	Authority         Authority         `json:"authority"`
}

type waldoIdentity struct {
	Name     string `json:"name"`
	Revision string `json:"revision"`
}

type waldoModelOrigin struct {
	BOM       string             `json:"bom"`
	SHA256    string             `json:"sha256"`
	Artifacts []AnchoredArtifact `json:"artifacts"`
}

type waldoModelRun struct {
	ID                string             `json:"id"`
	Stage             string             `json:"stage"`
	Ordinal           int                `json:"ordinal"`
	RunBOM            string             `json:"run_bom"`
	BOMSHA256         string             `json:"bom_sha256"`
	State             string             `json:"state"`
	Backend           waldoIdentity      `json:"backend"`
	Simulated         bool               `json:"simulated"`
	ObservationSHA256 string             `json:"observation_sha256,omitempty"`
	Artifacts         []AnchoredArtifact `json:"artifacts,omitempty"`
}

// waldoModelBOM mirrors the documented schema-1 managed/native model BOM.
// It is deliberately local to this fork and does not import WALDO internals.
type waldoModelBOM struct {
	Kind                string            `json:"kind"`
	Schema              int               `json:"schema"`
	Subject             string            `json:"subject"`
	ModelID             string            `json:"model_id"`
	Name                string            `json:"name"`
	PlanSHA256          string            `json:"plan_sha256"`
	ArchitectureSHA256  string            `json:"architecture_sha256"`
	PathBase            string            `json:"path_base"`
	CurrentRunID        string            `json:"current_run_id,omitempty"`
	CurrentOriginSHA256 string            `json:"current_origin_sha256,omitempty"`
	Origin              *waldoModelOrigin `json:"origin,omitempty"`
	Runs                []waldoModelRun   `json:"runs"`
	Generated           string            `json:"generated"`
}

type waldoReleaseBOM struct {
	Kind         string             `json:"kind"`
	Schema       int                `json:"schema"`
	Subject      string             `json:"subject"`
	Format       string             `json:"format"`
	ModelID      string             `json:"model_id"`
	Name         string             `json:"name"`
	SourceType   string             `json:"source_type"`
	SourceID     string             `json:"source_id"`
	RunID        string             `json:"run_id,omitempty"`
	SourceBOM    string             `json:"source_bom_sha256"`
	Artifacts    []AnchoredArtifact `json:"artifacts"`
	Quantization json.RawMessage    `json:"quantization,omitempty"`
	Generated    string             `json:"generated"`
}

type answeringIdentity struct {
	Subject            string             `json:"subject"`
	ModelID            string             `json:"model_id"`
	ArchitectureSHA256 string             `json:"architecture_sha256,omitempty"`
	Format             string             `json:"format,omitempty"`
	SourceType         string             `json:"source_type,omitempty"`
	SourceID           string             `json:"source_id,omitempty"`
	RunID              string             `json:"run_id,omitempty"`
	Artifacts          []AnchoredArtifact `json:"artifacts,omitempty"`
}

// AnchorWALDOBOM validates and projects a documented WALDO schema-1 model or
// model-release BOM. It never follows artifact paths or grants execution.
func AnchorWALDOBOM(data []byte) (OriginAnchor, error) {
	var envelope struct {
		Kind    string `json:"kind"`
		Schema  int    `json:"schema"`
		Subject string `json:"subject"`
	}
	if err := decodeJSON(data, &envelope, false); err != nil {
		return OriginAnchor{}, fmt.Errorf("decode WALDO BOM envelope: %w", err)
	}
	if envelope.Kind != "openwaldo-bom" || envelope.Schema != 1 {
		return OriginAnchor{}, fmt.Errorf("unsupported WALDO BOM identity %q schema %d", envelope.Kind, envelope.Schema)
	}

	documentDigest := digestBytes(data)
	switch envelope.Subject {
	case "model":
		var bom waldoModelBOM
		if err := decodeJSON(data, &bom, true); err != nil {
			return OriginAnchor{}, fmt.Errorf("decode model BOM: %w", err)
		}
		return anchorModelBOM(bom, documentDigest)
	case "model-release":
		var bom waldoReleaseBOM
		if err := decodeJSON(data, &bom, true); err != nil {
			return OriginAnchor{}, fmt.Errorf("decode model-release BOM: %w", err)
		}
		return anchorReleaseBOM(bom, documentDigest)
	default:
		return OriginAnchor{}, fmt.Errorf("unsupported WALDO BOM subject %q; expected model or model-release", envelope.Subject)
	}
}

func anchorModelBOM(bom waldoModelBOM, documentDigest string) (OriginAnchor, error) {
	if bom.Kind != "openwaldo-bom" || bom.Schema != 1 || bom.Subject != "model" {
		return OriginAnchor{}, errors.New("invalid model BOM identity")
	}
	if strings.TrimSpace(bom.ModelID) == "" || strings.TrimSpace(bom.Name) == "" {
		return OriginAnchor{}, errors.New("model BOM requires model_id and name")
	}
	if err := validateSHA256("model.plan_sha256", bom.PlanSHA256); err != nil {
		return OriginAnchor{}, err
	}
	if err := validateSHA256("model.architecture_sha256", bom.ArchitectureSHA256); err != nil {
		return OriginAnchor{}, err
	}
	if bom.PathBase != "model-root" {
		return OriginAnchor{}, fmt.Errorf("model path_base must be %q", "model-root")
	}
	if strings.TrimSpace(bom.Generated) == "" {
		return OriginAnchor{}, errors.New("model generated timestamp is required")
	}
	if bom.CurrentRunID != "" && bom.CurrentOriginSHA256 != "" {
		return OriginAnchor{}, errors.New("model BOM cannot select both current_run_id and current_origin_sha256")
	}

	runByID := make(map[string]waldoModelRun, len(bom.Runs))
	runOrdinals := make(map[int]struct{}, len(bom.Runs))
	for i, run := range bom.Runs {
		if strings.TrimSpace(run.ID) == "" || strings.TrimSpace(run.Stage) == "" || run.Ordinal < 1 {
			return OriginAnchor{}, fmt.Errorf("model runs[%d] requires id, stage, and positive ordinal", i)
		}
		if _, exists := runByID[run.ID]; exists {
			return OriginAnchor{}, fmt.Errorf("duplicate model run id %q", run.ID)
		}
		runByID[run.ID] = run
		if _, exists := runOrdinals[run.Ordinal]; exists {
			return OriginAnchor{}, fmt.Errorf("duplicate model run ordinal %d", run.Ordinal)
		}
		runOrdinals[run.Ordinal] = struct{}{}
		if err := validateRelativePath(fmt.Sprintf("model runs[%d].run_bom", i), run.RunBOM); err != nil {
			return OriginAnchor{}, err
		}
		if err := validateSHA256(fmt.Sprintf("model runs[%d].bom_sha256", i), run.BOMSHA256); err != nil {
			return OriginAnchor{}, err
		}
		if !oneOf(run.State, "planned", "running", "complete", "failed", "interrupted") {
			return OriginAnchor{}, fmt.Errorf("model runs[%d] has unsupported state %q", i, run.State)
		}
		if strings.TrimSpace(run.Backend.Name) == "" || strings.TrimSpace(run.Backend.Revision) == "" {
			return OriginAnchor{}, fmt.Errorf("model runs[%d] requires backend name and revision", i)
		}
		if run.ObservationSHA256 != "" {
			if err := validateSHA256(fmt.Sprintf("model runs[%d].observation_sha256", i), run.ObservationSHA256); err != nil {
				return OriginAnchor{}, err
			}
		}
		if err := validateArtifacts(fmt.Sprintf("model runs[%d].artifacts", i), run.Artifacts, false); err != nil {
			return OriginAnchor{}, err
		}
	}

	if bom.Origin != nil {
		if err := validateRelativePath("model origin.bom", bom.Origin.BOM); err != nil {
			return OriginAnchor{}, err
		}
		if err := validateSHA256("model origin.sha256", bom.Origin.SHA256); err != nil {
			return OriginAnchor{}, err
		}
		if err := validateArtifacts("model origin.artifacts", bom.Origin.Artifacts, true); err != nil {
			return OriginAnchor{}, err
		}
	}

	anchor := OriginAnchor{
		Schema:             OriginAnchorSchema,
		State:              AnchorStateAnchored,
		Subject:            bom.Subject,
		DocumentSHA256:     documentDigest,
		ModelID:            bom.ModelID,
		Name:               bom.Name,
		ArchitectureSHA256: bom.ArchitectureSHA256,
		PlanSHA256:         bom.PlanSHA256,
		Authority:          Authority{},
	}
	canonical, err := json.Marshal(bom)
	if err != nil {
		return OriginAnchor{}, fmt.Errorf("canonicalize model BOM: %w", err)
	}
	anchor.BOMSHA256 = digestBytes(canonical)

	switch {
	case bom.CurrentRunID != "":
		run, exists := runByID[bom.CurrentRunID]
		if !exists {
			return OriginAnchor{}, fmt.Errorf("current_run_id %q is missing from model runs", bom.CurrentRunID)
		}
		if run.State != "complete" || run.Simulated {
			return OriginAnchor{}, fmt.Errorf("current_run_id %q must select a complete non-simulated run", bom.CurrentRunID)
		}
		if err := validateArtifacts("selected model run artifacts", run.Artifacts, true); err != nil {
			return OriginAnchor{}, err
		}
		if err := requireArtifactRoles(fmt.Sprintf("current_run_id %q", bom.CurrentRunID), run.Artifacts, "weights", "configuration", "tokenizer"); err != nil {
			return OriginAnchor{}, err
		}
		anchor.SourceType = "run"
		anchor.SourceID = run.ID
		anchor.RunID = run.ID
		anchor.Artifacts = cloneSortedArtifacts(run.Artifacts)
	case bom.CurrentOriginSHA256 != "":
		if err := validateSHA256("model.current_origin_sha256", bom.CurrentOriginSHA256); err != nil {
			return OriginAnchor{}, err
		}
		if bom.Origin == nil || bom.Origin.SHA256 != bom.CurrentOriginSHA256 {
			return OriginAnchor{}, errors.New("current_origin_sha256 does not match model origin")
		}
		if err := requireArtifactRoles("current model origin", bom.Origin.Artifacts, "weights", "configuration", "tokenizer"); err != nil {
			return OriginAnchor{}, err
		}
		anchor.SourceType = "origin"
		anchor.SourceID = bom.CurrentOriginSHA256
		anchor.Artifacts = cloneSortedArtifacts(bom.Origin.Artifacts)
	default:
		anchor.State = AnchorStateHold
		anchor.Holds = []string{"model BOM has no selected complete real run or verified origin"}
	}

	anchor.AnsweringIdentitySHA256, err = answeringIdentityDigest(anchor)
	if err != nil {
		return OriginAnchor{}, err
	}
	return anchor, nil
}

func anchorReleaseBOM(bom waldoReleaseBOM, documentDigest string) (OriginAnchor, error) {
	if bom.Kind != "openwaldo-bom" || bom.Schema != 1 || bom.Subject != "model-release" {
		return OriginAnchor{}, errors.New("invalid model-release BOM identity")
	}
	if strings.TrimSpace(bom.ModelID) == "" || strings.TrimSpace(bom.Name) == "" || strings.TrimSpace(bom.Format) == "" {
		return OriginAnchor{}, errors.New("model-release BOM requires model_id, name, and format")
	}
	if !oneOf(bom.SourceType, "run", "origin") || strings.TrimSpace(bom.SourceID) == "" {
		return OriginAnchor{}, fmt.Errorf("model-release source_type must be run or origin with source_id")
	}
	if bom.SourceType == "run" && (strings.TrimSpace(bom.RunID) == "" || bom.RunID != bom.SourceID) {
		return OriginAnchor{}, errors.New("run-backed model-release requires run_id equal to source_id")
	}
	if bom.SourceType == "origin" && bom.RunID != "" {
		return OriginAnchor{}, errors.New("origin-backed model-release must not carry run_id")
	}
	if err := validateSHA256("model-release.source_bom_sha256", bom.SourceBOM); err != nil {
		return OriginAnchor{}, err
	}
	if err := validateArtifacts("model-release.artifacts", bom.Artifacts, true); err != nil {
		return OriginAnchor{}, err
	}
	if err := requireArtifactRoles("model-release", bom.Artifacts, "weights"); err != nil {
		return OriginAnchor{}, err
	}
	if len(bom.Quantization) > 0 {
		quantization := bytes.TrimSpace(bom.Quantization)
		if len(quantization) < 2 || quantization[0] != '{' || quantization[len(quantization)-1] != '}' {
			return OriginAnchor{}, errors.New("model-release quantization must be an object when present")
		}
	}
	if strings.TrimSpace(bom.Generated) == "" {
		return OriginAnchor{}, errors.New("model-release generated timestamp is required")
	}

	canonical, err := json.Marshal(bom)
	if err != nil {
		return OriginAnchor{}, fmt.Errorf("canonicalize model-release BOM: %w", err)
	}
	anchor := OriginAnchor{
		Schema:          OriginAnchorSchema,
		State:           AnchorStateAnchored,
		Subject:         bom.Subject,
		BOMSHA256:       digestBytes(canonical),
		DocumentSHA256:  documentDigest,
		ModelID:         bom.ModelID,
		Name:            bom.Name,
		Format:          bom.Format,
		SourceType:      bom.SourceType,
		SourceID:        bom.SourceID,
		RunID:           bom.RunID,
		SourceBOMSHA256: bom.SourceBOM,
		Artifacts:       cloneSortedArtifacts(bom.Artifacts),
		Authority:       Authority{},
	}
	anchor.AnsweringIdentitySHA256, err = answeringIdentityDigest(anchor)
	if err != nil {
		return OriginAnchor{}, err
	}
	return anchor, nil
}

func (a OriginAnchor) Validate() error {
	if a.Schema != OriginAnchorSchema {
		return fmt.Errorf("anchor schema must be %q", OriginAnchorSchema)
	}
	if !oneOf(a.State, AnchorStateAnchored, AnchorStateHold) {
		return fmt.Errorf("unsupported anchor state %q", a.State)
	}
	if !oneOf(a.Subject, "model", "model-release") {
		return fmt.Errorf("unsupported anchor subject %q", a.Subject)
	}
	for _, item := range []struct {
		name  string
		value string
	}{
		{"anchor.bom_sha256", a.BOMSHA256},
		{"anchor.document_sha256", a.DocumentSHA256},
		{"anchor.answering_identity_sha256", a.AnsweringIdentitySHA256},
	} {
		if err := validateSHA256(item.name, item.value); err != nil {
			return err
		}
	}
	for _, item := range []struct {
		name  string
		value string
	}{
		{"anchor.architecture_sha256", a.ArchitectureSHA256},
		{"anchor.plan_sha256", a.PlanSHA256},
		{"anchor.source_bom_sha256", a.SourceBOMSHA256},
	} {
		if item.value != "" {
			if err := validateSHA256(item.name, item.value); err != nil {
				return err
			}
		}
	}
	if strings.TrimSpace(a.ModelID) == "" || strings.TrimSpace(a.Name) == "" {
		return errors.New("anchor requires model_id and name")
	}
	if a.Simulated {
		return errors.New("origin anchor cannot select a simulated model")
	}
	if !a.Authority.closed() {
		return errors.New("origin anchor must carry closed authority")
	}
	switch a.Subject {
	case "model":
		if a.ArchitectureSHA256 == "" || a.PlanSHA256 == "" {
			return errors.New("model anchor requires architecture and plan digests")
		}
		if a.Format != "" || a.SourceBOMSHA256 != "" {
			return errors.New("model anchor must not carry release format or source BOM fields")
		}
	case "model-release":
		if strings.TrimSpace(a.Format) == "" || a.SourceBOMSHA256 == "" {
			return errors.New("model-release anchor requires format and source BOM digest")
		}
	}
	if a.State == AnchorStateAnchored {
		if !oneOf(a.SourceType, "run", "origin") || strings.TrimSpace(a.SourceID) == "" {
			return errors.New("anchored origin requires a run or origin source identity")
		}
		if a.SourceType == "run" && (a.RunID == "" || a.RunID != a.SourceID) {
			return errors.New("run-backed anchor requires run_id equal to source_id")
		}
		if a.SourceType == "origin" && a.RunID != "" {
			return errors.New("origin-backed anchor must not carry run_id")
		}
		if err := validateArtifacts("anchor.artifacts", a.Artifacts, true); err != nil {
			return err
		}
		if len(a.Holds) != 0 {
			return errors.New("anchored origin must not carry hold reasons")
		}
	} else {
		if a.Subject != "model" {
			return errors.New("only a model anchor may be HOLD")
		}
		if a.SourceType != "" || a.SourceID != "" || a.RunID != "" || len(a.Artifacts) != 0 {
			return errors.New("held origin anchor must not carry a selected source")
		}
		if len(a.Holds) == 0 {
			return errors.New("held origin anchor requires at least one hold reason")
		}
	}
	actual, err := answeringIdentityDigest(a)
	if err != nil {
		return err
	}
	if actual != a.AnsweringIdentitySHA256 {
		return fmt.Errorf("answering identity digest mismatch: expected %s, got %s", a.AnsweringIdentitySHA256, actual)
	}
	return nil
}

// CompareIdentity reports whether two provenance anchors select the same
// answering model identity. It does not promote, replace, or mutate either.
func CompareIdentity(expected, observed OriginAnchor) (IdentityLockReceipt, error) {
	if err := expected.Validate(); err != nil {
		return IdentityLockReceipt{}, fmt.Errorf("expected anchor: %w", err)
	}
	if err := observed.Validate(); err != nil {
		return IdentityLockReceipt{}, fmt.Errorf("observed anchor: %w", err)
	}
	receipt := IdentityLockReceipt{
		Schema: IdentityLockSchema,
		State:  IdentityStateLocked,
		Expected: IdentityAnchorRef{
			Subject: expected.Subject, ModelID: expected.ModelID, BOMSHA256: expected.BOMSHA256,
			AnsweringIdentitySHA256: expected.AnsweringIdentitySHA256,
		},
		Observed: IdentityAnchorRef{
			Subject: observed.Subject, ModelID: observed.ModelID, BOMSHA256: observed.BOMSHA256,
			AnsweringIdentitySHA256: observed.AnsweringIdentitySHA256,
		},
		ProvenanceChanged: expected.BOMSHA256 != observed.BOMSHA256,
		Authority:         Authority{},
	}
	if expected.Subject != observed.Subject {
		receipt.Changes = append(receipt.Changes, "subject")
	}
	if expected.ModelID != observed.ModelID {
		receipt.Changes = append(receipt.Changes, "model_id")
	}
	if expected.ArchitectureSHA256 != observed.ArchitectureSHA256 {
		receipt.Changes = append(receipt.Changes, "architecture_sha256")
	}
	if expected.Format != observed.Format {
		receipt.Changes = append(receipt.Changes, "format")
	}
	if expected.SourceType != observed.SourceType {
		receipt.Changes = append(receipt.Changes, "source_type")
	}
	if expected.SourceID != observed.SourceID {
		receipt.Changes = append(receipt.Changes, "source_id")
	}
	if expected.RunID != observed.RunID {
		receipt.Changes = append(receipt.Changes, "run_id")
	}
	if expected.SourceBOMSHA256 != observed.SourceBOMSHA256 {
		receipt.Changes = append(receipt.Changes, "source_bom_sha256")
	}
	if !reflect.DeepEqual(cloneSortedArtifacts(expected.Artifacts), cloneSortedArtifacts(observed.Artifacts)) {
		receipt.Changes = append(receipt.Changes, "artifacts")
	}
	if expected.Subject == "model-release" && expected.BOMSHA256 != observed.BOMSHA256 {
		receipt.Changes = append(receipt.Changes, "release_bom_sha256")
	}
	if expected.AnsweringIdentitySHA256 != observed.AnsweringIdentitySHA256 && len(receipt.Changes) == 0 {
		receipt.Changes = append(receipt.Changes, "answering_identity_sha256")
	}
	if len(receipt.Changes) > 0 {
		receipt.State = IdentityStateDrift
	} else if expected.State == AnchorStateHold || observed.State == AnchorStateHold {
		receipt.State = IdentityStateHold
	}
	if receipt.ProvenanceChanged && receipt.State == IdentityStateLocked {
		receipt.Notices = []string{"provenance document changed while selected answering identity remained stable"}
	}
	return receipt, nil
}

// SealAnchored refuses a decorative or stale lineage digest before delegating
// to the existing behavior-evidence seal.
func SealAnchored(draft BehaviorEvidenceDraft, anchor OriginAnchor) (SealedBehaviorEvidence, error) {
	if err := anchor.Validate(); err != nil {
		return SealedBehaviorEvidence{}, fmt.Errorf("origin anchor: %w", err)
	}
	if anchor.State != AnchorStateAnchored {
		return SealedBehaviorEvidence{}, errors.New("origin anchor is HOLD; a complete answering identity is required")
	}
	switch anchor.Subject {
	case "model":
		if draft.WALDO.ModelBOMSHA256 != anchor.BOMSHA256 {
			return SealedBehaviorEvidence{}, errors.New("draft model_bom_sha256 does not match origin anchor")
		}
	case "model-release":
		if draft.WALDO.ReleaseBOMSHA256 != anchor.BOMSHA256 {
			return SealedBehaviorEvidence{}, errors.New("draft release_bom_sha256 does not match origin anchor")
		}
	}
	return Seal(draft)
}

func answeringIdentityDigest(anchor OriginAnchor) (string, error) {
	projection := answeringIdentity{
		Subject:            anchor.Subject,
		ModelID:            anchor.ModelID,
		ArchitectureSHA256: anchor.ArchitectureSHA256,
		Format:             anchor.Format,
		SourceType:         anchor.SourceType,
		SourceID:           anchor.SourceID,
		RunID:              anchor.RunID,
		Artifacts:          cloneSortedArtifacts(anchor.Artifacts),
	}
	data, err := json.Marshal(projection)
	if err != nil {
		return "", fmt.Errorf("encode answering identity: %w", err)
	}
	return digestBytes(data), nil
}

func validateArtifacts(name string, artifacts []AnchoredArtifact, required bool) error {
	if required && len(artifacts) == 0 {
		return fmt.Errorf("%s requires at least one artifact", name)
	}
	seen := map[string]struct{}{}
	for i, artifact := range artifacts {
		if strings.TrimSpace(artifact.Role) == "" {
			return fmt.Errorf("%s[%d].role is required", name, i)
		}
		if err := validateRelativePath(fmt.Sprintf("%s[%d].path", name, i), artifact.Path); err != nil {
			return err
		}
		if _, exists := seen[artifact.Path]; exists {
			return fmt.Errorf("%s has duplicate path %q", name, artifact.Path)
		}
		seen[artifact.Path] = struct{}{}
		if err := validateSHA256(fmt.Sprintf("%s[%d].sha256", name, i), artifact.SHA256); err != nil {
			return err
		}
		if artifact.Bytes < 0 {
			return fmt.Errorf("%s[%d].bytes must not be negative", name, i)
		}
	}
	return nil
}

func hasArtifactRole(artifacts []AnchoredArtifact, role string) bool {
	for _, artifact := range artifacts {
		if artifact.Role == role {
			return true
		}
	}
	return false
}

func requireArtifactRoles(name string, artifacts []AnchoredArtifact, roles ...string) error {
	for _, role := range roles {
		if !hasArtifactRole(artifacts, role) {
			return fmt.Errorf("%s has no %s artifact", name, role)
		}
	}
	return nil
}

func validateRelativePath(name, value string) error {
	if strings.TrimSpace(value) == "" || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") {
		return fmt.Errorf("%s must be a non-empty portable relative path", name)
	}
	clean := pathpkg.Clean(value)
	if clean != value || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("%s must be a clean portable relative path", name)
	}
	return nil
}

func cloneSortedArtifacts(input []AnchoredArtifact) []AnchoredArtifact {
	if len(input) == 0 {
		return nil
	}
	output := append([]AnchoredArtifact(nil), input...)
	sort.Slice(output, func(i, j int) bool {
		if output[i].Path != output[j].Path {
			return output[i].Path < output[j].Path
		}
		return output[i].Role < output[j].Role
	})
	return output
}

func decodeJSON(data []byte, target any, strict bool) error {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if strict {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("trailing JSON content")
		}
		return fmt.Errorf("trailing JSON content: %w", err)
	}
	return nil
}

// DecodeStrictJSON decodes one JSON value, rejecting unknown fields, duplicate
// object keys, and trailing content. It is shared by the fork CLI so every
// durable witness input has the same unambiguous decoding rules.
func DecodeStrictJSON(data []byte, target any) error {
	return decodeJSON(data, target, true)
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanJSONValue(decoder, "$"); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return errors.New("trailing JSON content")
		}
		return fmt.Errorf("trailing JSON content: %w", err)
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder, location string) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("JSON object key at %s is not a string", location)
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate JSON key %q at %s", key, location)
			}
			seen[key] = struct{}{}
			if err := scanJSONValue(decoder, location+"."+key); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return fmt.Errorf("JSON object at %s has invalid closing token", location)
		}
	case '[':
		index := 0
		for decoder.More() {
			if err := scanJSONValue(decoder, fmt.Sprintf("%s[%d]", location, index)); err != nil {
				return err
			}
			index++
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return fmt.Errorf("JSON array at %s has invalid closing token", location)
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q at %s", delimiter, location)
	}
	return nil
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
