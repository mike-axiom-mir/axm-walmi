package axmmirror

import (
	"errors"
	"fmt"
	"strings"
)

func selectionReceiptV018ValidateRefusalCases(c SelectionReceiptV018Contract) error {
	want := map[string]string{
		"stale-candidate-set":       "CANDIDATE_SET_MISMATCH",
		"unknown-candidate":         "CANDIDATE_NOT_IN_FROZEN_SET",
		"candidate-digest-mismatch": "CANDIDATE_DIGEST_MISMATCH",
		"scope-escalation":          "SELECTION_SCOPE_EXCEEDED",
		"waldo-self-selection":      "WALDO_CANNOT_SELF_ISSUE_SELECTION",
	}
	if len(c.RefusalCases) != len(want) {
		return errors.New("v0.18 refusal case count drifted")
	}
	seen := map[string]bool{}
	for _, x := range c.RefusalCases {
		if seen[x.CaseID] || x.ExpectedOutcome != "REFUSED" || want[x.CaseID] != x.ExpectedReason {
			return fmt.Errorf("v0.18 refusal case %s drifted", x.CaseID)
		}
		seen[x.CaseID] = true
	}
	for id := range want {
		if !seen[id] {
			return fmt.Errorf("v0.18 refusal case %s missing", id)
		}
	}

	mutations := map[string]func(*SelectionReceiptV018Receipt){
		"stale-candidate-set":       func(r *SelectionReceiptV018Receipt) { r.SourceCandidateSetDigest = "sha256:" + strings.Repeat("0", 64) },
		"unknown-candidate":         func(r *SelectionReceiptV018Receipt) { r.SelectedCandidateID = "invented-chain" },
		"candidate-digest-mismatch": func(r *SelectionReceiptV018Receipt) { r.SelectedCandidateDigest = SelectionReceiptV018ProjectedDigest },
		"scope-escalation": func(r *SelectionReceiptV018Receipt) {
			r.Scope = []string{"SELECT_PROPOSAL", "BUILD"}
			r.GrantsExecution = true
		},
		"waldo-self-selection": func(r *SelectionReceiptV018Receipt) { r.IssuerClass = "WALDO" },
	}
	for _, x := range c.RefusalCases {
		r := c.SelectionReceipt
		mutations[x.CaseID](&r)
		outcome, reason := selectionReceiptV018ValidateReceipt(r, c.FrozenCandidates, c.CandidateSetDigest)
		if outcome != x.ExpectedOutcome || reason != x.ExpectedReason {
			return fmt.Errorf("v0.18 refusal case %s evaluation mismatch: %s/%s", x.CaseID, outcome, reason)
		}
	}
	return nil
}
