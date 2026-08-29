package axmmirror

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const BehaviorEvidenceSchema = "axm.mirror.behavior-evidence/v0.1"

// Authority is deliberately explicit. The first AXM/WALDO research slice is
// evidence-only: a record must not grant execution, training, promotion, canon,
// or world-action authority merely because it was produced by an AI system.
type Authority struct {
	ToolExecution bool `json:"tool_execution"`
	Training      bool `json:"training"`
	Promotion     bool `json:"promotion"`
	Canon         bool `json:"canon"`
	WorldAction   bool `json:"world_action"`
}

func (a Authority) closed() bool {
	return !a.ToolExecution && !a.Training && !a.Promotion && !a.Canon && !a.WorldAction
}

// WALDOLineage binds a downstream behavior record to the WALDO provenance
// objects that existed before the observed interaction. Empty optional fields
// remain explicit; at least a model or release BOM digest is required.
type WALDOLineage struct {
	CorpusBOMSHA256  string `json:"corpus_bom_sha256,omitempty"`
	RunBOMSHA256     string `json:"run_bom_sha256,omitempty"`
	ModelBOMSHA256   string `json:"model_bom_sha256,omitempty"`
	ReleaseBOMSHA256 string `json:"release_bom_sha256,omitempty"`
}

// Verification is a named check over the observed output or action. State is
// intentionally not collapsed into one score.
type Verification struct {
	Name           string `json:"name"`
	State          string `json:"state"`
	EvidenceSHA256 string `json:"evidence_sha256,omitempty"`
}

// Dissent preserves a disagreement or unresolved interpretation. Note is human
// readable and has no authority field by design.
type Dissent struct {
	Source string `json:"source"`
	State  string `json:"state"`
	Note   string `json:"note"`
}

// BehaviorEvidenceDraft is the public-safe bridge between WALDO model lineage
// and AXM/Mirror downstream behavior evidence. It stores digests, states and
// boundaries; it does not store hidden reasoning, secrets, private memory, or
// raw prompts by default.
type BehaviorEvidenceDraft struct {
	Schema          string         `json:"schema"`
	ExperimentID    string         `json:"experiment_id"`
	WALDO           WALDOLineage   `json:"waldo"`
	ContextSHA256   string         `json:"context_sha256"`
	RequestSHA256   string         `json:"request_sha256"`
	OutputSHA256    string         `json:"output_sha256"`
	PermissionState string         `json:"permission_state"`
	OutcomeState    string         `json:"outcome_state"`
	Verification    []Verification `json:"verification,omitempty"`
	Dissent         []Dissent      `json:"dissent,omitempty"`
	Authority       Authority      `json:"authority"`
}

// SealedBehaviorEvidence binds the exact JSON representation of Record to a
// SHA-256 digest. Sealing does not make the claims true; it only makes later
// mutation detectable.
type SealedBehaviorEvidence struct {
	Record BehaviorEvidenceDraft `json:"record"`
	SHA256 string                `json:"sha256"`
}

func (d BehaviorEvidenceDraft) Validate() error {
	if d.Schema != BehaviorEvidenceSchema {
		return fmt.Errorf("schema must be %q", BehaviorEvidenceSchema)
	}
	if strings.TrimSpace(d.ExperimentID) == "" {
		return errors.New("experiment_id is required")
	}
	if d.WALDO.ModelBOMSHA256 == "" && d.WALDO.ReleaseBOMSHA256 == "" {
		return errors.New("waldo lineage requires model_bom_sha256 or release_bom_sha256")
	}

	for _, field := range []struct {
		name   string
		digest string
	}{
		{"waldo.corpus_bom_sha256", d.WALDO.CorpusBOMSHA256},
		{"waldo.run_bom_sha256", d.WALDO.RunBOMSHA256},
		{"waldo.model_bom_sha256", d.WALDO.ModelBOMSHA256},
		{"waldo.release_bom_sha256", d.WALDO.ReleaseBOMSHA256},
		{"context_sha256", d.ContextSHA256},
		{"request_sha256", d.RequestSHA256},
		{"output_sha256", d.OutputSHA256},
	} {
		if field.digest == "" && strings.HasPrefix(field.name, "waldo.") {
			continue
		}
		if err := validateSHA256(field.name, field.digest); err != nil {
			return err
		}
	}

	if !oneOf(d.PermissionState, "allowed", "denied", "observed", "unknown") {
		return fmt.Errorf("unsupported permission_state %q", d.PermissionState)
	}
	if !oneOf(d.OutcomeState, "pass", "fail", "hold", "unknown") {
		return fmt.Errorf("unsupported outcome_state %q", d.OutcomeState)
	}
	if !d.Authority.closed() {
		return errors.New("behavior evidence v0.1 must carry closed authority")
	}

	seenVerification := map[string]struct{}{}
	for i, check := range d.Verification {
		name := strings.TrimSpace(check.Name)
		if name == "" {
			return fmt.Errorf("verification[%d].name is required", i)
		}
		if _, exists := seenVerification[name]; exists {
			return fmt.Errorf("duplicate verification name %q", name)
		}
		seenVerification[name] = struct{}{}
		if !oneOf(check.State, "pass", "fail", "hold", "unknown") {
			return fmt.Errorf("verification[%d] has unsupported state %q", i, check.State)
		}
		if check.EvidenceSHA256 != "" {
			if err := validateSHA256(fmt.Sprintf("verification[%d].evidence_sha256", i), check.EvidenceSHA256); err != nil {
				return err
			}
		}
	}

	for i, item := range d.Dissent {
		if strings.TrimSpace(item.Source) == "" {
			return fmt.Errorf("dissent[%d].source is required", i)
		}
		if !oneOf(item.State, "open", "resolved", "superseded", "unknown") {
			return fmt.Errorf("dissent[%d] has unsupported state %q", i, item.State)
		}
		if strings.TrimSpace(item.Note) == "" {
			return fmt.Errorf("dissent[%d].note is required", i)
		}
	}
	return nil
}

func Seal(d BehaviorEvidenceDraft) (SealedBehaviorEvidence, error) {
	if err := d.Validate(); err != nil {
		return SealedBehaviorEvidence{}, err
	}
	payload, err := json.Marshal(d)
	if err != nil {
		return SealedBehaviorEvidence{}, fmt.Errorf("marshal behavior evidence: %w", err)
	}
	digest := sha256.Sum256(payload)
	return SealedBehaviorEvidence{Record: d, SHA256: hex.EncodeToString(digest[:])}, nil
}

func (s SealedBehaviorEvidence) Verify() error {
	if err := s.Record.Validate(); err != nil {
		return fmt.Errorf("record validation failed: %w", err)
	}
	if err := validateSHA256("sha256", s.SHA256); err != nil {
		return err
	}
	payload, err := json.Marshal(s.Record)
	if err != nil {
		return fmt.Errorf("marshal sealed record: %w", err)
	}
	digest := sha256.Sum256(payload)
	actual := hex.EncodeToString(digest[:])
	if actual != strings.ToLower(s.SHA256) {
		return fmt.Errorf("sealed digest mismatch: expected %s, got %s", strings.ToLower(s.SHA256), actual)
	}
	return nil
}

func validateSHA256(name, value string) error {
	if len(value) != 64 {
		return fmt.Errorf("%s must be a 64-character SHA-256 hex digest", name)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != sha256.Size {
		return fmt.Errorf("%s must be valid SHA-256 hex", name)
	}
	if value != strings.ToLower(value) {
		return fmt.Errorf("%s must use lowercase hex", name)
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
