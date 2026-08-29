package axmmirror

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	EvaluationProtocolDraftSchema = "axm.waldo-witness.evaluation-protocol-draft/v0.1"
	EvaluationProtocolSealSchema  = "axm.waldo-witness.evaluation-protocol-seal/v0.1"

	ProtocolStateReady            = "SEALED_READY_FOR_EXACT_EVALUATION"
	ProtocolStateContamination    = "HOLD_CONTAMINATION"
	ProtocolStateIndependence     = "HOLD_INDEPENDENCE_UNPROVEN"
	ProtocolStateAnswerKeyLeak    = "HOLD_ANSWER_KEY_LEAK"
	ProtocolStateTargetMismatch   = "HOLD_TARGET_IDENTITY_MISMATCH"
	ProtocolStatePermission       = "HOLD_PERMISSION_NOT_ALLOWED"
	ProtocolStateAuthorityRefused = "REFUSED_AUTHORITY_GROWTH"

	AnswerKeyWithheld      = "withheld-from-target"
	AnswerKeyVisible       = "visible-to-target"
	AnswerKeyUnknown       = "unknown"
	AnswerKeyNotApplicable = "not-applicable"
)

type EvaluationPackIdentity struct {
	SHA256                   string `json:"sha256"`
	DocumentSHA256           string `json:"document_sha256"`
	CaseCount                int    `json:"case_count"`
	CaseOrderSHA256          string `json:"case_order_sha256"`
	AuthorshipState          string `json:"authorship_state"`
	AuthorshipEvidenceSHA256 string `json:"authorship_evidence_sha256,omitempty"`
	AnswerKeySHA256          string `json:"answer_key_sha256,omitempty"`
}

type EvaluationExecutionLimits struct {
	MaxCases       int   `json:"max_cases"`
	MaxOutputBytes int64 `json:"max_output_bytes"`
	TimeoutMillis  int64 `json:"timeout_millis"`
}

// EvaluationProtocolDraft deliberately has no output or metric-result field.
// Strict decoding therefore rejects attempts to let later model behavior alter
// the precommitted pack, target, metrics, or comparison dimensions.
type EvaluationProtocolDraft struct {
	Schema                        string                    `json:"schema"`
	ProtocolID                    string                    `json:"protocol_id"`
	Pack                          EvaluationPackIdentity    `json:"pack"`
	TargetAnsweringIdentitySHA256 string                    `json:"target_answering_identity_sha256"`
	Anchor                        OriginAnchor              `json:"anchor"`
	Context                       ProvenanceContextPacket   `json:"context"`
	Contamination                 ContaminationReport       `json:"contamination"`
	RequestSetSHA256              string                    `json:"request_set_sha256"`
	AllowedMetrics                []string                  `json:"allowed_metrics"`
	ComparisonDimensions          []string                  `json:"comparison_dimensions"`
	AnswerKeyVisibility           string                    `json:"answer_key_visibility"`
	Limits                        EvaluationExecutionLimits `json:"limits"`
	PermissionState               string                    `json:"permission_state"`
	RequestedAuthority            Authority                 `json:"requested_authority"`
}

type EvaluationTargetBinding struct {
	Subject                 string `json:"subject"`
	ModelID                 string `json:"model_id"`
	RunID                   string `json:"run_id,omitempty"`
	AnsweringIdentitySHA256 string `json:"answering_identity_sha256"`
	AnchorReceiptSHA256     string `json:"anchor_receipt_sha256"`
	ContextPacketSHA256     string `json:"context_packet_sha256"`
}

// EvaluationProtocolSeal is a deterministic precommitment. A later behavior
// receipt must name SealSHA256; the seal itself cannot prove wall-clock order.
type EvaluationProtocolSeal struct {
	Schema                    string                    `json:"schema"`
	State                     string                    `json:"state"`
	ProtocolID                string                    `json:"protocol_id"`
	DraftSHA256               string                    `json:"draft_sha256"`
	Pack                      EvaluationPackIdentity    `json:"pack"`
	Target                    EvaluationTargetBinding   `json:"target"`
	TargetMatched             bool                      `json:"target_matched"`
	ContaminationReportSHA256 string                    `json:"contamination_report_sha256"`
	ContaminationState        string                    `json:"contamination_state"`
	RequestSetSHA256          string                    `json:"request_set_sha256"`
	AllowedMetrics            []string                  `json:"allowed_metrics"`
	ComparisonDimensions      []string                  `json:"comparison_dimensions"`
	AnswerKeyVisibility       string                    `json:"answer_key_visibility"`
	Limits                    EvaluationExecutionLimits `json:"limits"`
	PermissionState           string                    `json:"permission_state"`
	RequestedAuthority        Authority                 `json:"requested_authority"`
	SealSHA256                string                    `json:"seal_sha256,omitempty"`
	Holds                     []string                  `json:"holds,omitempty"`
	Notices                   []string                  `json:"notices"`
	Authority                 Authority                 `json:"authority"`
}

// SealEvaluationProtocol freezes the exact evaluation contract before model
// execution. Valid but unsafe declarations produce typed HOLD/REFUSED seals so
// the attempted protocol remains inspectable without becoming executable.
func SealEvaluationProtocol(draft EvaluationProtocolDraft) (EvaluationProtocolSeal, error) {
	canonical, err := validateAndCanonicalizeProtocolDraft(draft)
	if err != nil {
		return EvaluationProtocolSeal{}, err
	}
	anchorDigest, err := digestJSON(canonical.Anchor, "evaluation protocol anchor")
	if err != nil {
		return EvaluationProtocolSeal{}, err
	}
	contaminationDigest, err := digestJSON(canonical.Contamination, "evaluation protocol contamination report")
	if err != nil {
		return EvaluationProtocolSeal{}, err
	}
	draftDigest, err := digestJSON(canonical, "evaluation protocol draft")
	if err != nil {
		return EvaluationProtocolSeal{}, err
	}
	targetHolds := evaluationTargetHolds(canonical, contaminationDigest)
	seal := EvaluationProtocolSeal{
		Schema: EvaluationProtocolSealSchema, State: ProtocolStateReady,
		ProtocolID: canonical.ProtocolID, DraftSHA256: draftDigest, Pack: canonical.Pack,
		Target: EvaluationTargetBinding{
			Subject: canonical.Anchor.Subject, ModelID: canonical.Anchor.ModelID, RunID: canonical.Anchor.RunID,
			AnsweringIdentitySHA256: canonical.TargetAnsweringIdentitySHA256,
			AnchorReceiptSHA256:     anchorDigest, ContextPacketSHA256: canonical.Context.PacketSHA256,
		},
		TargetMatched:             len(targetHolds) == 0,
		ContaminationReportSHA256: contaminationDigest, ContaminationState: canonical.Contamination.State,
		RequestSetSHA256: canonical.RequestSetSHA256, AllowedMetrics: canonical.AllowedMetrics,
		ComparisonDimensions: canonical.ComparisonDimensions, AnswerKeyVisibility: canonical.AnswerKeyVisibility,
		Limits: canonical.Limits, PermissionState: canonical.PermissionState, RequestedAuthority: canonical.RequestedAuthority,
		Notices: []string{
			"the seal freezes evaluation identity, order, target, context, metrics, dimensions, limits, permissions, and answer-key boundary before outputs are accepted",
			"a self-digest detects mutation but does not by itself prove wall-clock chronology, authorship, or independent execution",
			"CLEAR contamination is bounded to exact declared inventories and does not prove semantic independence or absence of memorization",
		},
		Authority: Authority{},
	}

	switch {
	case !canonical.RequestedAuthority.closed():
		seal.State = ProtocolStateAuthorityRefused
		seal.Holds = []string{"the draft requested authority beyond evidence-only sealing; no authority was granted"}
	case len(targetHolds) != 0:
		seal.State = ProtocolStateTargetMismatch
		seal.Holds = targetHolds
	case canonical.PermissionState != "allowed":
		seal.State = ProtocolStatePermission
		seal.Holds = []string{fmt.Sprintf("evaluation permission_state is %q, not allowed", canonical.PermissionState)}
	case canonical.Contamination.State == ContaminationStateContaminated:
		seal.State = ProtocolStateContamination
		seal.Holds = []string{"the declared training and evaluation inventories overlap"}
	case canonical.Contamination.State != ContaminationStateClear:
		seal.State = ProtocolStateIndependence
		seal.Holds = []string{"the contamination comparison is not CLEAR within its exact declared dimensions"}
	case canonical.Pack.AuthorshipState != "outside-authored":
		seal.State = ProtocolStateIndependence
		seal.Holds = []string{fmt.Sprintf("evaluation pack authorship is %q, not outside-authored", canonical.Pack.AuthorshipState)}
	case canonical.AnswerKeyVisibility == AnswerKeyVisible:
		seal.State = ProtocolStateAnswerKeyLeak
		seal.Holds = []string{"the evaluation answer key is declared visible to the target"}
	case canonical.AnswerKeyVisibility == AnswerKeyUnknown:
		seal.State = ProtocolStateIndependence
		seal.Holds = []string{"the evaluation answer-key visibility boundary is unknown"}
	}
	seal.SealSHA256, err = evaluationProtocolSealDigest(seal)
	if err != nil {
		return EvaluationProtocolSeal{}, err
	}
	if err := seal.Validate(); err != nil {
		return EvaluationProtocolSeal{}, fmt.Errorf("generated evaluation protocol seal: %w", err)
	}
	return seal, nil
}

func (seal EvaluationProtocolSeal) Validate() error {
	if seal.Schema != EvaluationProtocolSealSchema || !oneOf(seal.State,
		ProtocolStateReady, ProtocolStateContamination, ProtocolStateIndependence,
		ProtocolStateAnswerKeyLeak, ProtocolStateTargetMismatch, ProtocolStatePermission,
		ProtocolStateAuthorityRefused) {
		return fmt.Errorf("unsupported evaluation protocol seal identity %q state %q", seal.Schema, seal.State)
	}
	if strings.TrimSpace(seal.ProtocolID) == "" || strings.TrimSpace(seal.Target.Subject) == "" || strings.TrimSpace(seal.Target.ModelID) == "" {
		return errors.New("evaluation protocol seal has incomplete protocol or target identity")
	}
	for _, item := range []struct{ name, value string }{
		{"evaluation protocol.draft_sha256", seal.DraftSHA256},
		{"evaluation protocol.target.answering_identity_sha256", seal.Target.AnsweringIdentitySHA256},
		{"evaluation protocol.target.anchor_receipt_sha256", seal.Target.AnchorReceiptSHA256},
		{"evaluation protocol.target.context_packet_sha256", seal.Target.ContextPacketSHA256},
		{"evaluation protocol.contamination_report_sha256", seal.ContaminationReportSHA256},
		{"evaluation protocol.request_set_sha256", seal.RequestSetSHA256},
		{"evaluation protocol.seal_sha256", seal.SealSHA256},
	} {
		if err := validateSHA256(item.name, item.value); err != nil {
			return err
		}
	}
	if err := validateEvaluationPack(seal.Pack, seal.AnswerKeyVisibility); err != nil {
		return err
	}
	if !oneOf(seal.ContaminationState, ContaminationStateClear, ContaminationStateHold, ContaminationStateContaminated) {
		return fmt.Errorf("evaluation protocol has unsupported contamination state %q", seal.ContaminationState)
	}
	if !oneOf(seal.PermissionState, "allowed", "denied", "unknown") {
		return fmt.Errorf("evaluation protocol has unsupported permission_state %q", seal.PermissionState)
	}
	if err := validateProtocolNameSet("allowed_metrics", seal.AllowedMetrics); err != nil {
		return err
	}
	if err := validateProtocolNameSet("comparison_dimensions", seal.ComparisonDimensions); err != nil {
		return err
	}
	if seal.Limits.MaxCases != seal.Pack.CaseCount || seal.Limits.MaxOutputBytes < 1 || seal.Limits.TimeoutMillis < 1 {
		return errors.New("evaluation protocol limits are incomplete or do not bind the exact case count")
	}
	if seal.State == ProtocolStateReady {
		if len(seal.Holds) != 0 || !seal.TargetMatched || seal.ContaminationState != ContaminationStateClear || seal.Pack.AuthorshipState != "outside-authored" || seal.PermissionState != "allowed" || !seal.RequestedAuthority.closed() || !oneOf(seal.AnswerKeyVisibility, AnswerKeyWithheld, AnswerKeyNotApplicable) {
			return errors.New("ready evaluation protocol does not satisfy its precommit gates")
		}
	} else if len(seal.Holds) == 0 {
		return errors.New("held or refused evaluation protocol requires a reason")
	}
	if seal.State == ProtocolStateAuthorityRefused && seal.RequestedAuthority.closed() {
		return errors.New("authority-refused evaluation protocol records no requested authority growth")
	}
	if seal.State != ProtocolStateAuthorityRefused && !seal.RequestedAuthority.closed() {
		return errors.New("evaluation protocol with requested authority growth must be refused")
	}
	switch seal.State {
	case ProtocolStateTargetMismatch:
		if seal.TargetMatched {
			return errors.New("target-mismatch evaluation protocol records a matched target")
		}
	case ProtocolStatePermission:
		if !seal.TargetMatched || seal.PermissionState == "allowed" {
			return errors.New("permission-held evaluation protocol has inconsistent target or permission state")
		}
	case ProtocolStateContamination:
		if !seal.TargetMatched || seal.PermissionState != "allowed" || seal.ContaminationState != ContaminationStateContaminated {
			return errors.New("contamination-held evaluation protocol has inconsistent gate state")
		}
	case ProtocolStateAnswerKeyLeak:
		if !seal.TargetMatched || seal.PermissionState != "allowed" || seal.ContaminationState != ContaminationStateClear || seal.Pack.AuthorshipState != "outside-authored" || seal.AnswerKeyVisibility != AnswerKeyVisible {
			return errors.New("answer-key-held evaluation protocol has inconsistent gate state")
		}
	case ProtocolStateIndependence:
		unresolved := seal.ContaminationState != ContaminationStateClear || seal.Pack.AuthorshipState != "outside-authored" || seal.AnswerKeyVisibility == AnswerKeyUnknown
		if !seal.TargetMatched || seal.PermissionState != "allowed" || !unresolved {
			return errors.New("independence-held evaluation protocol has inconsistent gate state")
		}
	}
	if len(seal.Notices) == 0 || !seal.Authority.closed() {
		return errors.New("evaluation protocol seal must state its boundaries and carry closed authority")
	}
	expected, err := evaluationProtocolSealDigest(seal)
	if err != nil {
		return err
	}
	if expected != seal.SealSHA256 {
		return fmt.Errorf("evaluation protocol seal digest mismatch: expected %s, got %s", seal.SealSHA256, expected)
	}
	return nil
}

func validateAndCanonicalizeProtocolDraft(draft EvaluationProtocolDraft) (EvaluationProtocolDraft, error) {
	if draft.Schema != EvaluationProtocolDraftSchema {
		return EvaluationProtocolDraft{}, fmt.Errorf("evaluation protocol draft schema must be %q", EvaluationProtocolDraftSchema)
	}
	if strings.TrimSpace(draft.ProtocolID) == "" || draft.ProtocolID != strings.TrimSpace(draft.ProtocolID) {
		return EvaluationProtocolDraft{}, errors.New("evaluation protocol_id is required and must be trimmed")
	}
	if err := draft.Anchor.Validate(); err != nil {
		return EvaluationProtocolDraft{}, fmt.Errorf("evaluation protocol anchor: %w", err)
	}
	if err := draft.Context.Validate(); err != nil {
		return EvaluationProtocolDraft{}, fmt.Errorf("evaluation protocol context: %w", err)
	}
	if err := validateContaminationReport(draft.Contamination); err != nil {
		return EvaluationProtocolDraft{}, fmt.Errorf("evaluation protocol contamination: %w", err)
	}
	if err := validateSHA256("evaluation protocol target_answering_identity_sha256", draft.TargetAnsweringIdentitySHA256); err != nil {
		return EvaluationProtocolDraft{}, err
	}
	if err := validateSHA256("evaluation protocol request_set_sha256", draft.RequestSetSHA256); err != nil {
		return EvaluationProtocolDraft{}, err
	}
	if err := validateEvaluationPack(draft.Pack, draft.AnswerKeyVisibility); err != nil {
		return EvaluationProtocolDraft{}, err
	}
	if !oneOf(draft.PermissionState, "allowed", "denied", "unknown") {
		return EvaluationProtocolDraft{}, fmt.Errorf("unsupported evaluation permission_state %q", draft.PermissionState)
	}
	if draft.Limits.MaxCases != draft.Pack.CaseCount || draft.Limits.MaxOutputBytes < 1 || draft.Limits.TimeoutMillis < 1 {
		return EvaluationProtocolDraft{}, errors.New("evaluation limits must bind the exact case count and positive output/time bounds")
	}
	canonical := draft
	allowedMetrics, err := canonicalProtocolNameSet(draft.AllowedMetrics)
	if err != nil {
		return EvaluationProtocolDraft{}, err
	}
	comparisonDimensions, err := canonicalProtocolNameSet(draft.ComparisonDimensions)
	if err != nil {
		return EvaluationProtocolDraft{}, err
	}
	canonical.AllowedMetrics = allowedMetrics
	canonical.ComparisonDimensions = comparisonDimensions
	if err := validateProtocolNameSet("allowed_metrics", canonical.AllowedMetrics); err != nil {
		return EvaluationProtocolDraft{}, err
	}
	if err := validateProtocolNameSet("comparison_dimensions", canonical.ComparisonDimensions); err != nil {
		return EvaluationProtocolDraft{}, err
	}
	return canonical, nil
}

func validateEvaluationPack(pack EvaluationPackIdentity, answerKeyVisibility string) error {
	for _, item := range []struct{ name, value string }{
		{"evaluation pack.sha256", pack.SHA256},
		{"evaluation pack.document_sha256", pack.DocumentSHA256},
		{"evaluation pack.case_order_sha256", pack.CaseOrderSHA256},
	} {
		if err := validateSHA256(item.name, item.value); err != nil {
			return err
		}
	}
	if pack.CaseCount < 1 || !oneOf(pack.AuthorshipState, "outside-authored", "same-author", "unknown") {
		return errors.New("evaluation pack requires a positive case count and supported authorship state")
	}
	if pack.AuthorshipEvidenceSHA256 != "" {
		if err := validateSHA256("evaluation pack.authorship_evidence_sha256", pack.AuthorshipEvidenceSHA256); err != nil {
			return err
		}
	}
	if !oneOf(answerKeyVisibility, AnswerKeyWithheld, AnswerKeyVisible, AnswerKeyUnknown, AnswerKeyNotApplicable) {
		return fmt.Errorf("unsupported answer_key_visibility %q", answerKeyVisibility)
	}
	if answerKeyVisibility == AnswerKeyNotApplicable {
		if pack.AnswerKeySHA256 != "" {
			return errors.New("not-applicable answer-key boundary must not carry an answer key digest")
		}
	} else {
		if err := validateSHA256("evaluation pack.answer_key_sha256", pack.AnswerKeySHA256); err != nil {
			return err
		}
	}
	return nil
}

func evaluationTargetHolds(draft EvaluationProtocolDraft, contaminationDigest string) []string {
	var holds []string
	if draft.Anchor.State != AnchorStateAnchored || draft.TargetAnsweringIdentitySHA256 != draft.Anchor.AnsweringIdentitySHA256 {
		holds = append(holds, "declared target does not match the ANCHORED answering identity")
	}
	if draft.Context.State != ContextStateReady {
		holds = append(holds, "provenance context packet is not ready")
		return holds
	}
	identity, ok := findProvenanceFact(draft.Context, "answering.identity_sha256")
	if !ok {
		holds = append(holds, "provenance context omits answering.identity_sha256")
	} else {
		var value string
		if err := json.Unmarshal(identity.Value, &value); err != nil || value != draft.TargetAnsweringIdentitySHA256 {
			holds = append(holds, "provenance context answering identity does not match the protocol target")
		}
	}
	comparison, ok := findProvenanceFact(draft.Context, "evaluation.comparison_sha256")
	if !ok {
		holds = append(holds, "provenance context omits evaluation.comparison_sha256")
	} else {
		var value string
		if err := json.Unmarshal(comparison.Value, &value); err != nil || value != draft.Contamination.ComparisonSHA256 || comparison.SourceReceiptSHA256 != contaminationDigest {
			holds = append(holds, "provenance context does not bind the supplied contamination comparison")
		}
	}
	state, ok := findProvenanceFact(draft.Context, "evaluation.contamination_state")
	if !ok {
		holds = append(holds, "provenance context omits evaluation.contamination_state")
	} else {
		var value string
		if err := json.Unmarshal(state.Value, &value); err != nil || value != draft.Contamination.State || state.SourceReceiptSHA256 != contaminationDigest {
			holds = append(holds, "provenance context contamination state does not match the supplied report")
		}
	}
	return holds
}

func validateProtocolNameSet(name string, values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("evaluation protocol %s requires at least one value", name)
	}
	if !sort.StringsAreSorted(values) {
		return fmt.Errorf("evaluation protocol %s must be sorted", name)
	}
	for i, value := range values {
		if strings.TrimSpace(value) == "" || value != strings.TrimSpace(value) || len(value) > 128 {
			return fmt.Errorf("evaluation protocol %s[%d] is invalid", name, i)
		}
		if i > 0 && values[i-1] == value {
			return fmt.Errorf("evaluation protocol %s has duplicate value %q", name, value)
		}
	}
	return nil
}

func canonicalProtocolNameSet(values []string) ([]string, error) {
	result := append([]string(nil), values...)
	sort.Strings(result)
	for i := 1; i < len(result); i++ {
		if result[i] == result[i-1] {
			return nil, fmt.Errorf("duplicate evaluation protocol value %q", result[i])
		}
	}
	return result, nil
}

func evaluationProtocolSealDigest(seal EvaluationProtocolSeal) (string, error) {
	seal.SealSHA256 = ""
	return digestJSON(seal, "evaluation protocol seal")
}
