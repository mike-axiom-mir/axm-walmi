package axmmirror

import (
	"errors"
	"fmt"
)

func validateExecutionRevocationV020Refusals(cases []ExecutionRevocationV020RefusalCase) error {
	expected := []ExecutionRevocationV020RefusalCase{
		{CaseID: "source-mismatch", ExpectedOutcome: "REFUSED", ExpectedReason: "REVOCATION_SOURCE_MISMATCH"},
		{CaseID: "target-mismatch", ExpectedOutcome: "REFUSED", ExpectedReason: "REVOCATION_TARGET_MISMATCH"},
		{CaseID: "scope-escalation", ExpectedOutcome: "REFUSED", ExpectedReason: "REVOCATION_SCOPE_EXCEEDED"},
		{CaseID: "waldo-self-revocation", ExpectedOutcome: "REFUSED", ExpectedReason: "WALDO_CANNOT_SELF_ISSUE_REVOCATION"},
		{CaseID: "delete-prior-permit", ExpectedOutcome: "REFUSED", ExpectedReason: "HISTORY_REWRITE_FORBIDDEN"},
		{CaseID: "revocation-grants-authority", ExpectedOutcome: "REFUSED", ExpectedReason: "REVOCATION_CANNOT_GRANT_AUTHORITY"},
		{CaseID: "order-unresolved", ExpectedOutcome: "HOLD", ExpectedReason: "REVOCATION_ORDER_UNRESOLVED"},
	}
	if len(cases) != len(expected) {
		return errors.New("v0.20 refusal case count drifted")
	}
	seen := map[string]bool{}
	for i, c := range cases {
		if seen[c.CaseID] {
			return fmt.Errorf("v0.20 duplicate refusal case %s", c.CaseID)
		}
		seen[c.CaseID] = true
		if c != expected[i] {
			return fmt.Errorf("v0.20 refusal case %d drifted", i)
		}
	}
	return nil
}
