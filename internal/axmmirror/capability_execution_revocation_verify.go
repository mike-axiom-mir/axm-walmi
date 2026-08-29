package axmmirror

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

func VerifyExecutionRevocationV020Contract(data []byte) (ExecutionRevocationV020Contract, error) {
	var c ExecutionRevocationV020Contract
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("decode v0.20 execution-revocation contract: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("v0.20 execution-revocation contract has trailing JSON")
		}
		return ExecutionRevocationV020Contract{}, err
	}
	if err := c.Validate(); err != nil {
		return ExecutionRevocationV020Contract{}, err
	}
	expected, err := executionRevocationV020ExternalDigest(c)
	if err != nil {
		return ExecutionRevocationV020Contract{}, err
	}
	if expected != c.ReceiptDigest {
		return ExecutionRevocationV020Contract{}, fmt.Errorf("v0.20 receipt digest mismatch: expected %s, got %s", expected, c.ReceiptDigest)
	}
	return c, nil
}

func (c ExecutionRevocationV020Contract) Validate() error {
	if c.Schema != ExecutionRevocationV020Schema || c.Status != ExecutionRevocationV020Status || c.Challenge != ExecutionRevocationV020Challenge {
		return errors.New("v0.20 schema/status/challenge mismatch")
	}
	if c.SourceV019Receipt != ExecutionRevocationV020SourceReceipt || c.SourceV019Head != ExecutionRevocationV020SourceHead || c.SourceV019Witness != ExecutionRevocationV020SourceWitness || c.PermitID != ExecutionRevocationV020PermitID || c.SelectedCandidateID != ExecutionRevocationV020CandidateID || c.SelectedCandidateDigest != ExecutionRevocationV020CandidateSHA {
		return errors.New("v0.20 frozen source binding mismatch")
	}
	b := c.ProbeBoundary
	if b.LocalCapabilityFabricImplementationIncluded || b.ExecutableCandidateBodyIncluded || b.ProbeExecutesCandidate || b.LiveAIProviderCalled || b.RealRevocationObserved {
		return errors.New("v0.20 probe truth boundary drifted")
	}
	if err := validateExecutionRevocationV020Ledger(c.Ledger); err != nil {
		return err
	}
	if err := validateExecutionRevocationV020Receipt(c.Revocation); err != nil {
		return err
	}
	if err := validateExecutionRevocationV020Scenarios(c.Revocation, c.Scenarios); err != nil {
		return err
	}
	if err := validateExecutionRevocationV020Refusals(c.RefusalCases); err != nil {
		return err
	}
	if c.DuplicateCase.DuplicateReceiptID != c.Revocation.ReceiptID || c.DuplicateCase.ExpectedOutcome != "DEDUPLICATED" || c.DuplicateCase.ExtraAuthority || c.DuplicateCase.HistoryRewritten {
		return errors.New("v0.20 duplicate revocation case drifted")
	}
	if c.CurrentState != "REVOKED_BEFORE_EXECUTION" {
		return errors.New("v0.20 current authority state drifted")
	}
	t := c.Truth
	if !t.PriorPermitAcceptancePreserved || !t.RevocationAppendOnly || t.PriorPermitDeleted || t.PastAuthorizationRewritten || !t.FutureActionBlocked || t.ExecutionObserved || t.InstallAuthorityGranted || t.PromotionAuthorityGranted || t.CanonAuthorityGranted || t.WaldoOwnsRevocation {
		return errors.New("v0.20 truth claims drifted")
	}
	if c.Installed || c.Promoted || c.CanonChanged {
		return errors.New("v0.20 contract unexpectedly changed durable authority state")
	}
	if !executionRevocationV020ValidSHA(c.ReceiptDigest) {
		return errors.New("v0.20 receipt digest must be sha256:<64 hex>")
	}
	return nil
}

func validateExecutionRevocationV020Ledger(entries []ExecutionRevocationV020LedgerEntry) error {
	if len(entries) != 2 {
		return errors.New("v0.20 ledger must contain exactly permit acceptance and later revocation")
	}
	first, second := entries[0], entries[1]
	if first.Sequence != 1 || first.EventID != "v019-permit-accepted" || first.EventType != "PERMIT_ACCEPTED" || first.PermitID != ExecutionRevocationV020PermitID || first.Outcome != "ACCEPTED" || first.Reason != "V019_CONTRACT_FIXTURE_ACCEPTED" || first.ReferencesEventID != "" || first.RewritesPriorEntry {
		return errors.New("v0.20 retained permit-acceptance ledger entry drifted")
	}
	if second.Sequence != 2 || second.EventID != "v020-permit-revoked" || second.EventType != "PERMIT_REVOKED" || second.PermitID != ExecutionRevocationV020PermitID || second.Outcome != "REVOKED" || second.Reason != "EXTERNAL_REVOCATION_FIXTURE" || second.ReferencesEventID != first.EventID || second.RewritesPriorEntry {
		return errors.New("v0.20 revocation ledger entry drifted")
	}
	return nil
}

func validateExecutionRevocationV020Receipt(r ExecutionRevocationV020Receipt) error {
	if r.ReceiptID != "external-revocation-direct-read-v020" || r.IssuerClass != "EXTERNAL_EXECUTION_AUTHORITY_REVOCATION_FIXTURE" || r.Authenticity != "FIXTURE_ONLY_NOT_REAL_REVOCATION" {
		return errors.New("v0.20 revocation receipt identity drifted")
	}
	if r.SourceV019Receipt != ExecutionRevocationV020SourceReceipt || r.TargetPermitID != ExecutionRevocationV020PermitID || r.TargetCandidateID != ExecutionRevocationV020CandidateID || r.TargetCandidateSHA != ExecutionRevocationV020CandidateSHA {
		return errors.New("v0.20 revocation target binding mismatch")
	}
	if !executionRevocationV020StringSliceExact(r.Scope, []string{"REVOKE_EXECUTE_DISPOSABLE_PROBE"}) || r.EffectiveSequence != 2 || r.WaldoIssued || r.DeletePriorPermit || r.GrantsExecution || r.GrantsInstall || r.GrantsPromotion || r.GrantsCanon {
		return errors.New("v0.20 revocation scope/authority drifted")
	}
	return nil
}

func validateExecutionRevocationV020Scenarios(r ExecutionRevocationV020Receipt, scenarios []ExecutionRevocationV020Scenario) error {
	expected := []ExecutionRevocationV020Scenario{
		{ScenarioID: "before-revocation-body-present", RequestSequence: 1, BodyPresent: true, ExpectedDecision: "READY", ExpectedReason: "EXECUTION_AUTHORIZED_NOT_EXECUTED"},
		{ScenarioID: "before-revocation-body-absent", RequestSequence: 1, BodyPresent: false, ExpectedDecision: "HOLD", ExpectedReason: "EXECUTABLE_BODY_ABSENT"},
		{ScenarioID: "after-revocation-body-present", RequestSequence: 3, BodyPresent: true, ExpectedDecision: "REFUSED", ExpectedReason: "PERMIT_REVOKED"},
		{ScenarioID: "after-revocation-body-absent", RequestSequence: 3, BodyPresent: false, ExpectedDecision: "REFUSED", ExpectedReason: "PERMIT_REVOKED"},
	}
	if len(scenarios) != len(expected) {
		return errors.New("v0.20 scenario count drifted")
	}
	for i, s := range scenarios {
		if s != expected[i] {
			return fmt.Errorf("v0.20 scenario %d declaration drifted", i)
		}
		decision, reason := EvaluateExecutionRevocationV020(r, s.RequestSequence, s.BodyPresent)
		if decision != s.ExpectedDecision || reason != s.ExpectedReason {
			return fmt.Errorf("v0.20 scenario %s evaluated as %s/%s", s.ScenarioID, decision, reason)
		}
	}
	return nil
}

func EvaluateExecutionRevocationV020(r ExecutionRevocationV020Receipt, requestSequence int, bodyPresent bool) (string, string) {
	if r.WaldoIssued {
		return "REFUSED", "WALDO_CANNOT_SELF_ISSUE_REVOCATION"
	}
	if r.SourceV019Receipt != ExecutionRevocationV020SourceReceipt {
		return "REFUSED", "REVOCATION_SOURCE_MISMATCH"
	}
	if r.TargetPermitID != ExecutionRevocationV020PermitID || r.TargetCandidateID != ExecutionRevocationV020CandidateID || r.TargetCandidateSHA != ExecutionRevocationV020CandidateSHA {
		return "REFUSED", "REVOCATION_TARGET_MISMATCH"
	}
	if !executionRevocationV020StringSliceExact(r.Scope, []string{"REVOKE_EXECUTE_DISPOSABLE_PROBE"}) {
		return "REFUSED", "REVOCATION_SCOPE_EXCEEDED"
	}
	if r.DeletePriorPermit {
		return "REFUSED", "HISTORY_REWRITE_FORBIDDEN"
	}
	if r.GrantsExecution || r.GrantsInstall || r.GrantsPromotion || r.GrantsCanon {
		return "REFUSED", "REVOCATION_CANNOT_GRANT_AUTHORITY"
	}
	if r.EffectiveSequence <= 1 {
		return "HOLD", "REVOCATION_ORDER_UNRESOLVED"
	}
	if requestSequence >= r.EffectiveSequence {
		return "REFUSED", "PERMIT_REVOKED"
	}
	if !bodyPresent {
		return "HOLD", "EXECUTABLE_BODY_ABSENT"
	}
	return "READY", "EXECUTION_AUTHORIZED_NOT_EXECUTED"
}

func SealExecutionRevocationV020Contract(c ExecutionRevocationV020Contract) ([]byte, error) {
	digest, err := executionRevocationV020ExternalDigest(c)
	if err != nil {
		return nil, err
	}
	c.ReceiptDigest = digest
	return json.MarshalIndent(c, "", "  ")
}

func executionRevocationV020ExternalDigest(c ExecutionRevocationV020Contract) (string, error) {
	c.ReceiptDigest = ""
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func executionRevocationV020StringSliceExact(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func executionRevocationV020ValidSHA(v string) bool {
	if !strings.HasPrefix(v, "sha256:") || len(v) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(v, "sha256:"))
	return err == nil
}
