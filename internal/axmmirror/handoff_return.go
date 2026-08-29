package axmmirror

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

const (
	CapabilityReturnDraftSchema   = "axm.waldo-witness.capability-return-draft/v0.1"
	CapabilityReturnReceiptSchema = "axm.waldo-witness.capability-return-receipt/v0.1"

	CapabilityReturnVerified        = "RETURN_BINDINGS_VERIFIED"
	CapabilityReturnBindingHold     = "HOLD_RETURN_BINDING"
	CapabilityReturnPlanHold        = "HOLD_RETURN_PLAN"
	CapabilityReturnTranslationHold = "HOLD_RETURN_TRANSLATION"
	CapabilityReturnExpiredHold     = "HOLD_RETURN_EXPIRED"
	CapabilityReturnRuntimeHold     = "HOLD_RETURN_RUNTIME"
	CapabilityReturnPermissionHold  = "HOLD_RETURN_PERMISSION"
	CapabilityReturnCleanupHold     = "HOLD_RETURN_CLEANUP"
	CapabilityReturnOutputHold      = "HOLD_RETURN_OUTPUT"

	CapabilityRuntimeComplete  = "OBSERVED_COMPLETE"
	CapabilityRuntimeFailed    = "OBSERVED_FAILED"
	CapabilityRuntimeUnknown   = "UNKNOWN"
	CapabilityRuntimeSimulated = "SIMULATED"

	CapabilityPermissionConfirmed = "EXTERNAL_CONFIRMED"
	CapabilityPermissionDenied    = "DENIED"
	CapabilityPermissionUnknown   = "UNKNOWN"

	CapabilityCleanupConfirmed     = "CONFIRMED"
	CapabilityCleanupUnconfirmed   = "UNCONFIRMED"
	CapabilityCleanupNotApplicable = "NOT_APPLICABLE"

	MaxCapabilityReturnOutputs = 256
)

type CapabilityOutputArtifact struct {
	ID string `json:"id"`
	CapabilityArtifactRef
}

type CapabilityReturnDraft struct {
	Schema                   string                     `json:"schema"`
	ReturnID                 string                     `json:"return_id"`
	PlanSHA256               string                     `json:"plan_sha256"`
	TranslationReceiptSHA256 string                     `json:"translation_receipt_sha256"`
	ProviderID               string                     `json:"provider_id"`
	Input                    CapabilityArtifactRef      `json:"input"`
	Outputs                  []CapabilityOutputArtifact `json:"outputs"`
	ProviderReceiptSHA256    string                     `json:"provider_receipt_sha256"`
	StartedAt                string                     `json:"started_at"`
	CompletedAt              string                     `json:"completed_at"`
	AssessedAt               string                     `json:"assessed_at"`
	RuntimeState             string                     `json:"runtime_state"`
	PermissionState          string                     `json:"permission_state"`
	CleanupState             string                     `json:"cleanup_state"`
	RetainedInputBytes       int64                      `json:"retained_input_bytes"`
	Authority                Authority                  `json:"authority"`
}

type CapabilityReturnCheck struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

// CapabilityReturnReceipt checks digest and contract bindings only. It does
// not fetch output bytes or establish semantic, visual, safety, or legal quality.
type CapabilityReturnReceipt struct {
	Schema          string                  `json:"schema"`
	State           string                  `json:"state"`
	Draft           CapabilityReturnDraft   `json:"draft"`
	DraftSHA256     string                  `json:"draft_sha256"`
	OutputSetSHA256 string                  `json:"output_set_sha256"`
	Checks          []CapabilityReturnCheck `json:"checks"`
	Holds           []string                `json:"holds,omitempty"`
	Notices         []string                `json:"notices"`
	Authority       Authority               `json:"authority"`
	ReceiptSHA256   string                  `json:"receipt_sha256,omitempty"`
}

func VerifyCapabilityReturn(plan CapabilityHandoffPlan, translation TranslationLossReceipt, draft CapabilityReturnDraft) (CapabilityReturnReceipt, error) {
	if err := plan.Validate(); err != nil {
		return CapabilityReturnReceipt{}, fmt.Errorf("capability handoff plan: %w", err)
	}
	if err := translation.Validate(); err != nil {
		return CapabilityReturnReceipt{}, fmt.Errorf("translation loss receipt: %w", err)
	}
	canonical, startedAt, completedAt, assessedAt, err := canonicalizeCapabilityReturnDraft(draft)
	if err != nil {
		return CapabilityReturnReceipt{}, err
	}
	draftDigest, err := digestJSON(canonical, "capability return draft")
	if err != nil {
		return CapabilityReturnReceipt{}, err
	}
	outputDigest, err := digestJSON(canonical.Outputs, "capability return output set")
	if err != nil {
		return CapabilityReturnReceipt{}, err
	}
	checks := buildCapabilityReturnChecks(plan, translation, canonical, startedAt, completedAt, assessedAt)
	state, holds := capabilityReturnState(checks)
	receipt := CapabilityReturnReceipt{
		Schema: CapabilityReturnReceiptSchema, State: state, Draft: canonical,
		DraftSHA256: draftDigest, OutputSetSHA256: outputDigest, Checks: checks, Holds: holds,
		Notices: []string{
			"verification binds exact declared digests, schemas, sizes, times, provider receipt, cleanup, and externally reported state; it does not retrieve or rehash artifact bytes",
			"RETURN_BINDINGS_VERIFIED does not establish content correctness, semantic or visual quality, safety, legality, usefulness, provider identity, or successful permission enforcement",
			"the return receipt cannot install, publish, promote, train, alter CANON, grant permission, or act in the world",
		},
		Authority: Authority{},
	}
	receipt.ReceiptSHA256, err = capabilityReturnReceiptDigest(receipt)
	if err != nil {
		return CapabilityReturnReceipt{}, err
	}
	if err := receipt.Validate(); err != nil {
		return CapabilityReturnReceipt{}, fmt.Errorf("generated capability return receipt: %w", err)
	}
	return receipt, nil
}

func canonicalizeCapabilityReturnDraft(draft CapabilityReturnDraft) (CapabilityReturnDraft, time.Time, time.Time, time.Time, error) {
	if draft.Schema != CapabilityReturnDraftSchema {
		return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("capability return draft schema must be %q", CapabilityReturnDraftSchema)
	}
	for name, value := range map[string]string{"return_id": draft.ReturnID, "provider_id": draft.ProviderID} {
		if err := requireTrimmed("capability return "+name, value); err != nil {
			return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, err
		}
	}
	for name, value := range map[string]string{
		"plan_sha256": draft.PlanSHA256, "translation_receipt_sha256": draft.TranslationReceiptSHA256,
		"provider_receipt_sha256": draft.ProviderReceiptSHA256,
	} {
		if err := validateSHA256("capability return "+name, value); err != nil {
			return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, err
		}
	}
	if err := validateCapabilityArtifact("capability return input", draft.Input, MaxCapabilityArtifactBytes); err != nil {
		return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	if len(draft.Outputs) == 0 || len(draft.Outputs) > MaxCapabilityReturnOutputs {
		return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("capability return requires between 1 and %d output artifacts", MaxCapabilityReturnOutputs)
	}
	canonical := draft
	canonical.Outputs = append([]CapabilityOutputArtifact(nil), draft.Outputs...)
	for i, artifact := range canonical.Outputs {
		if err := requireTrimmed(fmt.Sprintf("capability return output %d id", i), artifact.ID); err != nil {
			return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, err
		}
		if err := validateCapabilityArtifact(fmt.Sprintf("capability return output %d", i), artifact.CapabilityArtifactRef, MaxCapabilityArtifactBytes); err != nil {
			return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, err
		}
	}
	sort.Slice(canonical.Outputs, func(i, j int) bool { return canonical.Outputs[i].ID < canonical.Outputs[j].ID })
	for i := 1; i < len(canonical.Outputs); i++ {
		if canonical.Outputs[i-1].ID == canonical.Outputs[i].ID {
			return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("duplicate capability return output id %q", canonical.Outputs[i].ID)
		}
	}
	startedAt, err := parseCapabilityTime("capability return started_at", draft.StartedAt)
	if err != nil {
		return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	completedAt, err := parseCapabilityTime("capability return completed_at", draft.CompletedAt)
	if err != nil {
		return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	assessedAt, err := parseCapabilityTime("capability return assessed_at", draft.AssessedAt)
	if err != nil {
		return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	canonical.StartedAt = startedAt.UTC().Format(time.RFC3339Nano)
	canonical.CompletedAt = completedAt.UTC().Format(time.RFC3339Nano)
	canonical.AssessedAt = assessedAt.UTC().Format(time.RFC3339Nano)
	if !oneOf(draft.RuntimeState, CapabilityRuntimeComplete, CapabilityRuntimeFailed, CapabilityRuntimeUnknown, CapabilityRuntimeSimulated) {
		return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("unsupported capability return runtime_state %q", draft.RuntimeState)
	}
	if !oneOf(draft.PermissionState, CapabilityPermissionConfirmed, CapabilityPermissionDenied, CapabilityPermissionUnknown) {
		return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("unsupported capability return permission_state %q", draft.PermissionState)
	}
	if !oneOf(draft.CleanupState, CapabilityCleanupConfirmed, CapabilityCleanupUnconfirmed, CapabilityCleanupNotApplicable) {
		return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("unsupported capability return cleanup_state %q", draft.CleanupState)
	}
	if draft.RetainedInputBytes < 0 {
		return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, errors.New("capability return retained_input_bytes must not be negative")
	}
	if !draft.Authority.closed() {
		return CapabilityReturnDraft{}, time.Time{}, time.Time{}, time.Time{}, errors.New("capability return draft must carry closed authority")
	}
	return canonical, startedAt, completedAt, assessedAt, nil
}

func buildCapabilityReturnChecks(plan CapabilityHandoffPlan, translation TranslationLossReceipt, draft CapabilityReturnDraft, startedAt, completedAt, assessedAt time.Time) []CapabilityReturnCheck {
	bindingPass := draft.PlanSHA256 == plan.PlanSHA256 &&
		draft.TranslationReceiptSHA256 == translation.ReceiptSHA256 &&
		translation.Declaration.PlanSHA256 == plan.PlanSHA256 &&
		translation.PlanState == plan.State &&
		draft.ProviderID == translation.Declaration.ProviderID &&
		capabilityArtifactsEqual(draft.Input, plan.Request.Input)
	bindingDetail := "return, translation, provider, and input artifact bind the supplied plan"
	if !bindingPass {
		bindingDetail = "return, translation, provider, or input artifact does not bind the supplied plan"
	}

	planPass := plan.State == HandoffStateReady && plan.UniqueCandidateProviderID == draft.ProviderID && !plan.ProviderSelected && !plan.ExternalExecutionAllowed
	planDetail := "plan has one exact unselected candidate and carries no execution authorization"
	if !planPass {
		planDetail = "plan is not READY for the returned provider"
	}

	translationPass := translation.State == TranslationStateNotRequired || translation.State == TranslationStateNoDeclaredLoss
	translationDetail := "translation declares no loss requiring review"
	if !translationPass {
		translationDetail = fmt.Sprintf("translation state %s requires review or blocks return", translation.State)
	}

	expiresAt, _ := parseCapabilityTime("capability gap expires_at", plan.Request.ExpiresAt)
	createdAt, _ := parseCapabilityTime("capability gap created_at", plan.Request.CreatedAt)
	expiryPass := !startedAt.Before(createdAt) && !assessedAt.After(expiresAt) && !completedAt.Before(startedAt) && !assessedAt.Before(completedAt)
	expiryDetail := "return timestamps are ordered and assessed before plan expiry"
	if !expiryPass {
		expiryDetail = "return timestamps are out of order, precede plan creation, or exceed plan expiry"
	}

	runtimePass := draft.RuntimeState == CapabilityRuntimeComplete
	runtimeDetail := fmt.Sprintf("runtime state is %s", draft.RuntimeState)
	permissionPass := draft.PermissionState == CapabilityPermissionConfirmed
	permissionDetail := fmt.Sprintf("externally reported permission state is %s", draft.PermissionState)
	cleanupPass := draft.CleanupState == CapabilityCleanupConfirmed && draft.RetainedInputBytes == 0
	cleanupDetail := fmt.Sprintf("cleanup state is %s with %d retained input bytes", draft.CleanupState, draft.RetainedInputBytes)

	outputPass := true
	totalBytes := int64(0)
	for _, artifact := range draft.Outputs {
		totalBytes += artifact.Bytes
		if artifact.Schema != plan.Request.RequiredOutputSchema {
			outputPass = false
		}
	}
	if totalBytes > plan.Request.MaxOutputBytes {
		outputPass = false
	}
	outputDetail := fmt.Sprintf("%d output artifact(s), %d bytes, required schema %s", len(draft.Outputs), totalBytes, plan.Request.RequiredOutputSchema)

	return []CapabilityReturnCheck{
		{Name: "plan-and-input-binding", Pass: bindingPass, Detail: bindingDetail},
		{Name: "plan-readiness", Pass: planPass, Detail: planDetail},
		{Name: "translation-loss", Pass: translationPass, Detail: translationDetail},
		{Name: "expiry-and-timestamps", Pass: expiryPass, Detail: expiryDetail},
		{Name: "runtime-observation", Pass: runtimePass, Detail: runtimeDetail},
		{Name: "external-permission-report", Pass: permissionPass, Detail: permissionDetail},
		{Name: "cleanup-and-retention", Pass: cleanupPass, Detail: cleanupDetail},
		{Name: "output-contract", Pass: outputPass, Detail: outputDetail},
	}
}

func capabilityReturnState(checks []CapabilityReturnCheck) (string, []string) {
	states := []string{
		CapabilityReturnBindingHold, CapabilityReturnPlanHold, CapabilityReturnTranslationHold,
		CapabilityReturnExpiredHold, CapabilityReturnRuntimeHold, CapabilityReturnPermissionHold,
		CapabilityReturnCleanupHold, CapabilityReturnOutputHold,
	}
	for i, check := range checks {
		if !check.Pass {
			return states[i], []string{check.Detail}
		}
	}
	return CapabilityReturnVerified, nil
}

func (receipt CapabilityReturnReceipt) Validate() error {
	if receipt.Schema != CapabilityReturnReceiptSchema || !oneOf(receipt.State,
		CapabilityReturnVerified, CapabilityReturnBindingHold, CapabilityReturnPlanHold,
		CapabilityReturnTranslationHold, CapabilityReturnExpiredHold, CapabilityReturnRuntimeHold,
		CapabilityReturnPermissionHold, CapabilityReturnCleanupHold, CapabilityReturnOutputHold) {
		return fmt.Errorf("unsupported capability return receipt identity %q state %q", receipt.Schema, receipt.State)
	}
	canonical, _, _, _, err := canonicalizeCapabilityReturnDraft(receipt.Draft)
	if err != nil {
		return err
	}
	if !capabilityReturnDraftsEqual(canonical, receipt.Draft) {
		return errors.New("capability return draft is not canonical")
	}
	for name, value := range map[string]string{
		"draft_sha256": receipt.DraftSHA256, "output_set_sha256": receipt.OutputSetSHA256, "receipt_sha256": receipt.ReceiptSHA256,
	} {
		if err := validateSHA256("capability return receipt "+name, value); err != nil {
			return err
		}
	}
	draftDigest, err := digestJSON(receipt.Draft, "capability return draft")
	if err != nil {
		return err
	}
	outputDigest, err := digestJSON(receipt.Draft.Outputs, "capability return output set")
	if err != nil {
		return err
	}
	if draftDigest != receipt.DraftSHA256 || outputDigest != receipt.OutputSetSHA256 {
		return errors.New("capability return draft or output-set digest does not match")
	}
	expectedNames := []string{
		"plan-and-input-binding", "plan-readiness", "translation-loss", "expiry-and-timestamps",
		"runtime-observation", "external-permission-report", "cleanup-and-retention", "output-contract",
	}
	if len(receipt.Checks) != len(expectedNames) {
		return errors.New("capability return receipt has an incomplete check set")
	}
	for i, check := range receipt.Checks {
		if check.Name != expectedNames[i] || check.Detail == "" {
			return fmt.Errorf("capability return check %d is invalid or out of order", i)
		}
	}
	expectedState, expectedHolds := capabilityReturnState(receipt.Checks)
	if receipt.State != expectedState || !stringSlicesEqual(receipt.Holds, expectedHolds) {
		return errors.New("capability return state or holds do not match its checks")
	}
	if len(receipt.Notices) == 0 || !receipt.Authority.closed() {
		return errors.New("capability return receipt requires notices and closed authority")
	}
	digest, err := capabilityReturnReceiptDigest(receipt)
	if err != nil {
		return err
	}
	if digest != receipt.ReceiptSHA256 {
		return errors.New("capability return receipt digest does not match")
	}
	return nil
}

func capabilityArtifactsEqual(left, right CapabilityArtifactRef) bool {
	return left.Schema == right.Schema && left.SHA256 == right.SHA256 && left.Bytes == right.Bytes
}

func capabilityReturnReceiptDigest(receipt CapabilityReturnReceipt) (string, error) {
	receipt.ReceiptSHA256 = ""
	return digestJSON(receipt, "capability return receipt")
}

func capabilityReturnDraftsEqual(left, right CapabilityReturnDraft) bool {
	leftDigest, err := digestJSON(left, "left capability return draft")
	if err != nil {
		return false
	}
	rightDigest, err := digestJSON(right, "right capability return draft")
	return err == nil && leftDigest == rightDigest
}
