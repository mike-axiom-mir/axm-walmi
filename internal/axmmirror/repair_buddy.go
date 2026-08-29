package axmmirror

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	RepairBuddyRequestSchema = "axm.waldo-witness.repair-buddy-request/v0.1"
	RepairBuddyPlanSchema    = "axm.waldo-witness.repair-buddy-plan/v0.1"

	RepairBuddyCandidateReady = "REPAIR_CANDIDATE_READY"
	RepairBuddyNotNeededHold  = "HOLD_REPAIR_NOT_NEEDED"

	RepairStageComplete  = "COMPLETE"
	RepairStageCandidate = "CANDIDATE"
	RepairStageRequired  = "REQUIRED"
	RepairStageBlocked   = "BLOCKED"
)

type RepairBuddyRequest struct {
	Schema                    string    `json:"schema"`
	RepairID                  string    `json:"repair_id"`
	FailedChangeReceiptSHA256 string    `json:"failed_change_receipt_sha256"`
	IncidentEvidenceSHA256    string    `json:"incident_evidence_sha256"`
	ObservedAt                string    `json:"observed_at"`
	MaxAttempts               int       `json:"max_attempts"`
	Authority                 Authority `json:"authority"`
}

type RepairBuddyStage struct {
	Stage            string `json:"stage"`
	State            string `json:"state"`
	Action           string `json:"action"`
	EvidenceRequired string `json:"evidence_required"`
}

// RepairBuddyPlan is a bounded recovery proposal. It may diagnose, contain,
// and name a next candidate, but deliberately carries no patch, command, path,
// activation, promotion, or CANON mechanism.
type RepairBuddyPlan struct {
	Schema                    string                `json:"schema"`
	State                     string                `json:"state"`
	Request                   RepairBuddyRequest    `json:"request"`
	RequestSHA256             string                `json:"request_sha256"`
	FailedChange              VerifierChangeReceipt `json:"failed_change"`
	FailedChangeReceiptSHA256 string                `json:"failed_change_receipt_sha256"`
	ActiveRegistrySHA256      string                `json:"active_registry_sha256"`
	RollbackTargetSHA256      string                `json:"rollback_target_sha256"`
	BaselineRetained          bool                  `json:"baseline_retained"`
	Diagnosis                 string                `json:"diagnosis"`
	Disposition               string                `json:"disposition"`
	Stages                    []RepairBuddyStage    `json:"stages"`
	ReentryRequirements       []string              `json:"reentry_requirements"`
	CandidateOnly             bool                  `json:"candidate_only"`
	AutomaticRepair           bool                  `json:"automatic_repair"`
	HumanReviewRequired       bool                  `json:"human_review_required"`
	Notices                   []string              `json:"notices"`
	Authority                 Authority             `json:"authority"`
	PlanSHA256                string                `json:"plan_sha256,omitempty"`
}

func PlanVerifierRepair(current VerifierRegistry, failed VerifierChangeReceipt, request RepairBuddyRequest) (RepairBuddyPlan, error) {
	if err := current.Validate(); err != nil {
		return RepairBuddyPlan{}, fmt.Errorf("current verifier registry: %w", err)
	}
	if err := failed.Validate(); err != nil {
		return RepairBuddyPlan{}, fmt.Errorf("failed verifier change receipt: %w", err)
	}
	canonicalRequest, err := canonicalizeRepairBuddyRequest(request)
	if err != nil {
		return RepairBuddyPlan{}, err
	}
	recomputed, err := AssessVerifierChange(current, failed.Request)
	if err != nil {
		return RepairBuddyPlan{}, err
	}
	if recomputed.ReceiptSHA256 != failed.ReceiptSHA256 {
		return RepairBuddyPlan{}, errors.New("failed verifier change receipt does not match a fresh assessment")
	}
	if canonicalRequest.FailedChangeReceiptSHA256 != failed.ReceiptSHA256 {
		return RepairBuddyPlan{}, errors.New("repair request does not bind the supplied failed verifier change receipt")
	}
	requestDigest, err := digestJSON(canonicalRequest, "repair buddy request")
	if err != nil {
		return RepairBuddyPlan{}, err
	}
	diagnosis, disposition, needed, err := repairDiagnosis(failed.State)
	if err != nil {
		return RepairBuddyPlan{}, err
	}
	plan := RepairBuddyPlan{
		Schema: RepairBuddyPlanSchema, State: RepairBuddyCandidateReady,
		Request: canonicalRequest, RequestSHA256: requestDigest,
		FailedChange: failed, FailedChangeReceiptSHA256: failed.ReceiptSHA256,
		ActiveRegistrySHA256: current.RegistrySHA256, RollbackTargetSHA256: current.RegistrySHA256,
		BaselineRetained: true, Diagnosis: diagnosis, Disposition: disposition,
		CandidateOnly: true, AutomaticRepair: false, HumanReviewRequired: true,
		Notices: []string{
			"Repair Buddy retains the current trusted registry and produces only a review candidate; it does not patch, install, activate, promote, or execute a verifier",
			"any revised verifier must re-enter through a new sealed intent, frozen shadow replay, independent peer quorum, and the same invariant kernel",
			"technical repair evidence is not promotion, safety, usefulness, legal approval, or CANON authority",
		},
		Authority: Authority{},
	}
	if !needed {
		plan.State = RepairBuddyNotNeededHold
		plan.Diagnosis = "NO_FAILED_OR_HELD_CHANGE"
		plan.Disposition = "NO_REPAIR_CANDIDATE"
	}
	plan.Stages = repairStages(needed, disposition)
	plan.ReentryRequirements = []string{
		"a new intent id and source-evidence digest",
		"the unchanged compiled invariant kernel and exact active-registry digest",
		"a frozen replay over the same declared fixture set or an explicitly versioned replacement fixture set",
		"at least two passing current peers from two independent groups with no target self-review",
		"explicit human review before any candidate generation is activated",
	}
	plan.PlanSHA256, err = repairBuddyPlanDigest(plan)
	if err != nil {
		return RepairBuddyPlan{}, err
	}
	if err := plan.Validate(); err != nil {
		return RepairBuddyPlan{}, fmt.Errorf("generated Repair Buddy plan: %w", err)
	}
	return plan, nil
}

func (plan RepairBuddyPlan) Validate() error {
	if plan.Schema != RepairBuddyPlanSchema {
		return fmt.Errorf("Repair Buddy plan schema must be %q", RepairBuddyPlanSchema)
	}
	request, err := canonicalizeRepairBuddyRequest(plan.Request)
	if err != nil {
		return err
	}
	requestDigest, err := digestJSON(request, "repair buddy request")
	if err != nil {
		return err
	}
	if requestDigest != plan.RequestSHA256 {
		return errors.New("Repair Buddy request digest mismatch")
	}
	if err := plan.FailedChange.Validate(); err != nil {
		return fmt.Errorf("Repair Buddy failed change: %w", err)
	}
	if plan.FailedChange.ReceiptSHA256 != plan.FailedChangeReceiptSHA256 || plan.Request.FailedChangeReceiptSHA256 != plan.FailedChangeReceiptSHA256 {
		return errors.New("Repair Buddy failed change binding mismatch")
	}
	for name, digest := range map[string]string{
		"active_registry_sha256": plan.ActiveRegistrySHA256,
		"rollback_target_sha256": plan.RollbackTargetSHA256,
		"plan_sha256":            plan.PlanSHA256,
	} {
		if err := validateSHA256("Repair Buddy "+name, digest); err != nil {
			return err
		}
	}
	if plan.ActiveRegistrySHA256 != plan.FailedChange.ActiveRegistrySHA256 || plan.RollbackTargetSHA256 != plan.ActiveRegistrySHA256 || !plan.BaselineRetained {
		return errors.New("Repair Buddy must retain the exact active registry as rollback baseline")
	}
	if !oneOf(plan.State, RepairBuddyCandidateReady, RepairBuddyNotNeededHold) {
		return fmt.Errorf("unsupported Repair Buddy state %q", plan.State)
	}
	if !plan.CandidateOnly || plan.AutomaticRepair || !plan.HumanReviewRequired {
		return errors.New("Repair Buddy must remain candidate-only, non-automatic, and human-reviewed")
	}
	if len(plan.Stages) != 10 || len(plan.ReentryRequirements) < 5 || len(plan.Notices) < 3 {
		return errors.New("Repair Buddy plan requires the complete bounded lifecycle, re-entry requirements, and notices")
	}
	expectedStages := []string{"OBSERVE", "DETECT", "CONTAIN", "DIAGNOSE", "PROPOSE", "SIMULATE", "AUTHORIZE", "REPAIR", "VERIFY", "LEARN"}
	for i, stage := range plan.Stages {
		if stage.Stage != expectedStages[i] || !oneOf(stage.State, RepairStageComplete, RepairStageCandidate, RepairStageRequired, RepairStageBlocked) || strings.TrimSpace(stage.Action) == "" || strings.TrimSpace(stage.EvidenceRequired) == "" {
			return fmt.Errorf("Repair Buddy stage %d is invalid", i)
		}
	}
	if strings.TrimSpace(plan.Diagnosis) == "" || strings.TrimSpace(plan.Disposition) == "" {
		return errors.New("Repair Buddy plan requires diagnosis and disposition")
	}
	if !plan.Authority.closed() {
		return errors.New("Repair Buddy plan must carry closed authority")
	}
	expected, err := repairBuddyPlanDigest(plan)
	if err != nil {
		return err
	}
	if expected != plan.PlanSHA256 {
		return errors.New("Repair Buddy plan digest mismatch")
	}
	return nil
}

func canonicalizeRepairBuddyRequest(request RepairBuddyRequest) (RepairBuddyRequest, error) {
	if request.Schema != RepairBuddyRequestSchema {
		return RepairBuddyRequest{}, fmt.Errorf("Repair Buddy request schema must be %q", RepairBuddyRequestSchema)
	}
	if err := requireTrimmed("Repair Buddy repair_id", request.RepairID); err != nil {
		return RepairBuddyRequest{}, err
	}
	if err := validateSHA256("Repair Buddy failed_change_receipt_sha256", request.FailedChangeReceiptSHA256); err != nil {
		return RepairBuddyRequest{}, err
	}
	if err := validateSHA256("Repair Buddy incident_evidence_sha256", request.IncidentEvidenceSHA256); err != nil {
		return RepairBuddyRequest{}, err
	}
	if _, err := time.Parse(time.RFC3339Nano, request.ObservedAt); err != nil {
		return RepairBuddyRequest{}, fmt.Errorf("parse Repair Buddy observed_at: %w", err)
	}
	if request.MaxAttempts < 1 || request.MaxAttempts > 3 {
		return RepairBuddyRequest{}, errors.New("Repair Buddy max_attempts must be between 1 and 3")
	}
	if !request.Authority.closed() {
		return RepairBuddyRequest{}, errors.New("Repair Buddy request must carry closed authority")
	}
	return request, nil
}

func repairDiagnosis(state string) (string, string, bool, error) {
	switch state {
	case VerifierChangeReady:
		return "NO_FAILED_OR_HELD_CHANGE", "NO_REPAIR_CANDIDATE", false, nil
	case VerifierChangeRollbackSelf:
		return "SELF_APPROVAL_PATH", "QUARANTINE_SELF_APPROVAL_PATH", true, nil
	case VerifierChangeRollbackPeerReject:
		return "CANDIDATE_REGRESSION", "REVISE_OR_WITHDRAW_CANDIDATE", true, nil
	case VerifierChangeRollbackBinding, VerifierChangeRollbackDrift, VerifierChangeRollbackShadow:
		return "EVIDENCE_BINDING_FAILURE", "REBUILD_EVIDENCE_FROM_FROZEN_BASELINE", true, nil
	case VerifierChangeHoldQuorum, VerifierChangeHoldEvidence:
		return "INCOMPLETE_INDEPENDENT_EVIDENCE", "COLLECT_MISSING_INDEPENDENT_REVIEWS", true, nil
	case VerifierChangeHoldMergeGate:
		return "PROTECTED_CHANGE_REQUIRES_MERGE_GATE", "ESCALATE_TO_EXPLICIT_MERGE_GATE", true, nil
	default:
		return "", "", false, fmt.Errorf("unsupported failed verifier change state %q", state)
	}
}

func repairStages(needed bool, disposition string) []RepairBuddyStage {
	proposeState := RepairStageCandidate
	if !needed {
		proposeState = RepairStageBlocked
	}
	return []RepairBuddyStage{
		{Stage: "OBSERVE", State: RepairStageComplete, Action: "bind the exact failed-or-held change receipt", EvidenceRequired: "failed change receipt digest"},
		{Stage: "DETECT", State: RepairStageComplete, Action: "classify the typed verifier-change state", EvidenceRequired: "fresh deterministic reassessment"},
		{Stage: "CONTAIN", State: RepairStageComplete, Action: "retain the current registry and quarantine the candidate", EvidenceRequired: "active and rollback registry digests"},
		{Stage: "DIAGNOSE", State: RepairStageComplete, Action: "map failed checks to a bounded diagnosis", EvidenceRequired: "peer checks and frozen replay bindings"},
		{Stage: "PROPOSE", State: proposeState, Action: disposition, EvidenceRequired: "new candidate intent and source evidence"},
		{Stage: "SIMULATE", State: RepairStageRequired, Action: "rerun the frozen shadow protocol", EvidenceRequired: "baseline and candidate result-set digests"},
		{Stage: "AUTHORIZE", State: RepairStageRequired, Action: "obtain independent peer quorum and explicit human review", EvidenceRequired: "two current peer receipts from two groups"},
		{Stage: "REPAIR", State: RepairStageBlocked, Action: "materialize only after a new verifier-change receipt is READY", EvidenceRequired: "new READY receipt"},
		{Stage: "VERIFY", State: RepairStageBlocked, Action: "verify the new generation and retained rollback chain", EvidenceRequired: "new registry self-digest and previous-generation digest"},
		{Stage: "LEARN", State: RepairStageBlocked, Action: "record outcome only after post-repair verification", EvidenceRequired: "independent post-repair evidence receipt"},
	}
}

func repairBuddyPlanDigest(plan RepairBuddyPlan) (string, error) {
	plan.PlanSHA256 = ""
	return digestJSON(plan, "Repair Buddy plan")
}
