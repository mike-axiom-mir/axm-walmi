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

func VerifyExecutionPermitV019Contract(data []byte) (ExecutionPermitV019Contract, error) {
	var c ExecutionPermitV019Contract
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("decode v0.19 execution-permit contract: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("v0.19 execution-permit contract has trailing JSON")
		}
		return ExecutionPermitV019Contract{}, err
	}
	if err := c.Validate(); err != nil {
		return ExecutionPermitV019Contract{}, err
	}
	expected, err := executionPermitV019ExternalDigest(c)
	if err != nil {
		return ExecutionPermitV019Contract{}, err
	}
	if expected != c.ReceiptDigest {
		return ExecutionPermitV019Contract{}, fmt.Errorf("v0.19 receipt digest mismatch: expected %s, got %s", expected, c.ReceiptDigest)
	}
	return c, nil
}

func (c ExecutionPermitV019Contract) Validate() error {
	if c.Schema != ExecutionPermitV019Schema || c.Status != ExecutionPermitV019Status || c.Challenge != ExecutionPermitV019Challenge {
		return errors.New("v0.19 schema/status/challenge mismatch")
	}
	if c.SourceV018Receipt != ExecutionPermitV019SourceReceipt || c.SourceV018Head != ExecutionPermitV019SourceHead || c.CandidateSetDigest != ExecutionPermitV019CandidateSetDigest || c.SelectedCandidateID != ExecutionPermitV019CandidateID || c.SelectedCandidateDigest != ExecutionPermitV019CandidateDigest {
		return errors.New("v0.19 frozen source binding mismatch")
	}
	b := c.ProbeBoundary
	if b.LocalCapabilityFabricImplementationIncluded || b.ExecutableCandidateBodyIncluded || b.ProbeExecutesCandidate || b.LiveAIProviderCalled || b.RealExecutionApprovalObserved {
		return errors.New("v0.19 probe truth boundary drifted")
	}
	if err := validateExecutionPermitV019Permit(c.ExecutionPermit); err != nil {
		return err
	}
	if err := validateExecutionPermitV019Request(c.ExecutionPermit, c.ExecutionRequest); err != nil {
		return err
	}
	decision, reason := EvaluateExecutionPermitV019Request(c.ExecutionPermit, c.ExecutionRequest)
	if c.PermitValidation != "PERMIT_ACCEPTED" || c.RequestValidation != "REQUEST_BOUND" || c.OverallDecision != decision || c.OverallReason != reason || decision != "HOLD" || reason != "EXECUTABLE_BODY_ABSENT" {
		return errors.New("v0.19 expected missing-body HOLD drifted")
	}
	if err := validateExecutionPermitV019Refusals(c.RefusalCases); err != nil {
		return err
	}
	if c.DuplicateCase.DuplicatePermitID != c.ExecutionPermit.PermitID || c.DuplicateCase.ExpectedOutcome != "DEDUPLICATED" || c.DuplicateCase.ExtraAuthority || c.DuplicateCase.ExtraAction {
		return errors.New("v0.19 duplicate permit case drifted")
	}
	t := c.Truth
	if !t.PermissionAccepted || t.PermissionTreatedAsInstruction || !t.MissingBodyReturnsHold || t.MissingBodyAutoBuilt || t.MissingBodyInvented || t.ExecutionObserved || t.InstallAuthorityGranted || t.PromotionAuthorityGranted || t.CanonAuthorityGranted || t.WaldoOwnsExecution {
		return errors.New("v0.19 truth claims drifted")
	}
	if c.BuildStarted || c.Installed || c.Promoted || c.CanonChanged {
		return errors.New("v0.19 contract unexpectedly acted beyond verification")
	}
	if !executionPermitV019ValidSHA(c.ReceiptDigest) {
		return errors.New("v0.19 receipt digest must be sha256:<64 hex>")
	}
	return nil
}

func validateExecutionPermitV019Permit(p ExecutionPermitV019Permit) error {
	if p.PermitID != "external-disposable-execution-permit-direct-read" || p.IssuerClass != "EXTERNAL_EXECUTION_AUTHORITY_CONTRACT_FIXTURE" || p.Authenticity != "FIXTURE_ONLY_NOT_REAL_APPROVAL" {
		return errors.New("v0.19 execution permit identity drifted")
	}
	if p.SourceSelectionContractReceipt != ExecutionPermitV019SourceReceipt || p.SourceCandidateSetDigest != ExecutionPermitV019CandidateSetDigest || p.SelectedCandidateID != ExecutionPermitV019CandidateID || p.SelectedCandidateDigest != ExecutionPermitV019CandidateDigest {
		return errors.New("v0.19 execution permit binding mismatch")
	}
	if !executionPermitV019StringSliceExact(p.Scope, []string{"EXECUTE_DISPOSABLE_PROBE"}) || p.GrantsInstall || p.GrantsPromotion || p.GrantsCanon {
		return errors.New("v0.19 execution permit scope exceeds disposable execution")
	}
	if p.WaldoIssued {
		return errors.New("v0.19 WALDO cannot self-issue execution authority")
	}
	return nil
}

func validateExecutionPermitV019Request(p ExecutionPermitV019Permit, r ExecutionPermitV019Request) error {
	if r.RequestID != "execute-direct-read-chain-disposable" || r.RequestedByClass != "EXTERNAL_EXECUTION_REQUEST_FIXTURE" || r.Action != "EXECUTE_DISPOSABLE_PROBE" {
		return errors.New("v0.19 execution request identity/action drifted")
	}
	if r.PermitID != p.PermitID || r.SelectedCandidateID != p.SelectedCandidateID || r.SelectedCandidateDigest != p.SelectedCandidateDigest {
		return errors.New("v0.19 execution request is not bound to the accepted permit selection")
	}
	if r.ExecutableBodyPresent || r.AutoBuildMissingBody || r.InventedBody || len(r.RequestedSideEffects) != 0 {
		return errors.New("v0.19 baseline request must preserve absent body with no auto-build, invention, or side effects")
	}
	return nil
}

func EvaluateExecutionPermitV019Request(p ExecutionPermitV019Permit, r ExecutionPermitV019Request) (string, string) {
	if p.WaldoIssued {
		return "REFUSED", "WALDO_CANNOT_SELF_ISSUE_EXECUTION"
	}
	if p.SourceSelectionContractReceipt != ExecutionPermitV019SourceReceipt || p.SourceCandidateSetDigest != ExecutionPermitV019CandidateSetDigest {
		return "REFUSED", "SELECTION_SOURCE_MISMATCH"
	}
	if p.SelectedCandidateID != ExecutionPermitV019CandidateID || p.SelectedCandidateDigest != ExecutionPermitV019CandidateDigest {
		return "REFUSED", "EXECUTION_SELECTION_MISMATCH"
	}
	if !executionPermitV019StringSliceExact(p.Scope, []string{"EXECUTE_DISPOSABLE_PROBE"}) || p.GrantsInstall || p.GrantsPromotion || p.GrantsCanon {
		return "REFUSED", "EXECUTION_SCOPE_EXCEEDED"
	}
	if r.PermitID != p.PermitID {
		return "REFUSED", "EXECUTION_REQUEST_PERMIT_MISMATCH"
	}
	if r.SelectedCandidateID != p.SelectedCandidateID || r.SelectedCandidateDigest != p.SelectedCandidateDigest {
		return "REFUSED", "EXECUTION_REQUEST_SELECTION_MISMATCH"
	}
	if r.Action != "EXECUTE_DISPOSABLE_PROBE" || len(r.RequestedSideEffects) != 0 {
		return "REFUSED", "EXECUTION_REQUEST_SCOPE_EXCEEDED"
	}
	if r.AutoBuildMissingBody {
		return "REFUSED", "AUTO_BUILD_FORBIDDEN"
	}
	if r.InventedBody {
		return "REFUSED", "MISSING_BODY_INVENTION_FORBIDDEN"
	}
	if !r.ExecutableBodyPresent {
		return "HOLD", "EXECUTABLE_BODY_ABSENT"
	}
	return "READY", "EXECUTION_AUTHORIZED_NOT_EXECUTED"
}

func SealExecutionPermitV019Contract(c ExecutionPermitV019Contract) ([]byte, error) {
	digest, err := executionPermitV019ExternalDigest(c)
	if err != nil {
		return nil, err
	}
	c.ReceiptDigest = digest
	return json.MarshalIndent(c, "", "  ")
}

func executionPermitV019ExternalDigest(c ExecutionPermitV019Contract) (string, error) {
	c.ReceiptDigest = ""
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func executionPermitV019StringSliceExact(got, want []string) bool {
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

func executionPermitV019ValidSHA(v string) bool {
	if !strings.HasPrefix(v, "sha256:") || len(v) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(v, "sha256:"))
	return err == nil
}
