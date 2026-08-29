package axmmirror

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func VerifySelectionReceiptV018(data []byte) (SelectionReceiptV018Contract, error) {
	var c SelectionReceiptV018Contract
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("decode v0.18 selection contract: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("v0.18 selection contract has trailing JSON")
		}
		return SelectionReceiptV018Contract{}, err
	}
	if err := c.Validate(); err != nil {
		return SelectionReceiptV018Contract{}, err
	}
	expected, err := selectionReceiptV018ExternalDigest(c)
	if err != nil {
		return SelectionReceiptV018Contract{}, err
	}
	if c.ReceiptDigest != expected {
		return SelectionReceiptV018Contract{}, fmt.Errorf("v0.18 receipt mismatch: expected %s got %s", expected, c.ReceiptDigest)
	}
	return c, nil
}

func (c SelectionReceiptV018Contract) Validate() error {
	if c.Schema != SelectionReceiptV018Schema || c.Status != SelectionReceiptV018Status || c.Challenge != SelectionReceiptV018Challenge {
		return errors.New("v0.18 schema/status/challenge drifted")
	}
	if c.SourceV017Receipt != SelectionReceiptV018SourceReceipt || c.SourceV017Head != SelectionReceiptV018SourceHead || c.CandidateSetDigest != SelectionReceiptV018CandidateSetDigest {
		return errors.New("v0.18 source binding drifted")
	}
	if err := selectionReceiptV018ValidateFrozenCandidates(c.FrozenCandidates); err != nil {
		return err
	}
	validation, reason := selectionReceiptV018ValidateReceipt(c.SelectionReceipt, c.FrozenCandidates, c.CandidateSetDigest)
	if validation != "CONTRACT_SELECTION_ACCEPTED" || reason != "" || c.SelectionValidation != validation {
		return fmt.Errorf("v0.18 primary selection receipt invalid: %s/%s", validation, reason)
	}
	if c.SelectedProposalState != "SELECTED_PROPOSAL_ONLY" || c.OverallDecision != "HOLD" || c.OverallReason != "EXECUTION_AUTHORITY_ABSENT" || c.ExecutionAuthority != "NONE" {
		return errors.New("v0.18 selected-vs-execution boundary drifted")
	}
	conflictValidation, conflictReason := selectionReceiptV018ValidateReceipt(c.ConflictReceipt, c.FrozenCandidates, c.CandidateSetDigest)
	if conflictValidation != "CONTRACT_SELECTION_ACCEPTED" || conflictReason != "" {
		return errors.New("v0.18 conflict receipt must be individually valid")
	}
	outcome, conflict := selectionReceiptV018EvaluateReceiptSet([]SelectionReceiptV018Receipt{c.SelectionReceipt, c.ConflictReceipt})
	if outcome != c.ConflictCase.ExpectedOutcome || conflict != c.ConflictCase.ExpectedReason || outcome != "HOLD" || conflict != "CONFLICTING_SELECTION_RECEIPTS" || c.ConflictCase.LastWriterWins {
		return errors.New("v0.18 conflict behavior drifted")
	}
	if c.ConflictCase.FirstReceiptID != c.SelectionReceipt.ReceiptID || c.ConflictCase.SecondReceiptID != c.ConflictReceipt.ReceiptID {
		return errors.New("v0.18 conflict receipt lineage drifted")
	}
	if err := selectionReceiptV018ValidateRefusalCases(c); err != nil {
		return err
	}
	if c.RealGovernanceDecisionObserved || c.LiveAIProviderCalled || c.WaldoIssuedSelection || c.BuildStarted || c.Installed || c.Promoted || c.CanonChanged {
		return errors.New("v0.18 truth/authority boundary drifted")
	}
	if !selectionReceiptV018ValidSHA(c.ReceiptDigest) {
		return errors.New("v0.18 receipt must be sha256:<64 hex>")
	}
	return nil
}

func selectionReceiptV018ValidateFrozenCandidates(candidates []SelectionReceiptV018Candidate) error {
	if len(candidates) != 2 {
		return errors.New("v0.18 must bind exactly two v0.17 candidates")
	}
	seen := map[string]string{}
	for _, c := range candidates {
		if c.ID == "" || seen[c.ID] != "" || !selectionReceiptV018ValidSHA(c.Digest) {
			return errors.New("v0.18 frozen candidate identity invalid or duplicated")
		}
		seen[c.ID] = c.Digest
	}
	if seen["direct-read-chain"] != SelectionReceiptV018DirectDigest || seen["projected-read-chain"] != SelectionReceiptV018ProjectedDigest {
		return errors.New("v0.18 frozen candidate digests drifted")
	}
	return nil
}

func selectionReceiptV018ValidateReceipt(r SelectionReceiptV018Receipt, candidates []SelectionReceiptV018Candidate, setDigest string) (string, string) {
	if r.SourceCandidateSetDigest != setDigest {
		return "REFUSED", "CANDIDATE_SET_MISMATCH"
	}
	if r.IssuerClass == "WALDO" {
		return "REFUSED", "WALDO_CANNOT_SELF_ISSUE_SELECTION"
	}
	if r.IssuerClass != "EXTERNAL_GOVERNANCE_CONTRACT_FIXTURE" || r.Authenticity != "FIXTURE_ONLY_NOT_REAL_APPROVAL" {
		return "REFUSED", "SELECTION_ISSUER_OR_AUTHENTICITY_INVALID"
	}
	if !selectionReceiptV018SliceExact(r.Scope, []string{"SELECT_PROPOSAL"}) || r.GrantsExecution || r.GrantsInstall || r.GrantsPromotion || r.GrantsCanon {
		return "REFUSED", "SELECTION_SCOPE_EXCEEDED"
	}
	candidateDigest := ""
	for _, c := range candidates {
		if c.ID == r.SelectedCandidateID {
			candidateDigest = c.Digest
			break
		}
	}
	if candidateDigest == "" {
		return "REFUSED", "CANDIDATE_NOT_IN_FROZEN_SET"
	}
	if r.SelectedCandidateDigest != candidateDigest {
		return "REFUSED", "CANDIDATE_DIGEST_MISMATCH"
	}
	return "CONTRACT_SELECTION_ACCEPTED", ""
}

func selectionReceiptV018EvaluateReceiptSet(receipts []SelectionReceiptV018Receipt) (string, string) {
	selected := map[string]bool{}
	seenReceipt := map[string]bool{}
	for _, r := range receipts {
		if seenReceipt[r.ReceiptID] {
			continue
		}
		seenReceipt[r.ReceiptID] = true
		selected[r.SelectedCandidateID] = true
	}
	if len(selected) > 1 {
		return "HOLD", "CONFLICTING_SELECTION_RECEIPTS"
	}
	if len(selected) == 1 {
		return "SELECTED_PROPOSAL_ONLY", "EXECUTION_AUTHORITY_ABSENT"
	}
	return "HOLD", "NO_SELECTION_RECEIPT"
}
