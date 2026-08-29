package axmmirror

import (
	"errors"
	"fmt"
)

func validateExecutionPermitV019Refusals(cases []ExecutionPermitV019RefusalCase) error {
	expected := []ExecutionPermitV019RefusalCase{
		{CaseID: "selection-source-mismatch", ExpectedOutcome: "REFUSED", ExpectedReason: "SELECTION_SOURCE_MISMATCH"},
		{CaseID: "execution-selection-mismatch", ExpectedOutcome: "REFUSED", ExpectedReason: "EXECUTION_SELECTION_MISMATCH"},
		{CaseID: "execution-scope-escalation", ExpectedOutcome: "REFUSED", ExpectedReason: "EXECUTION_SCOPE_EXCEEDED"},
		{CaseID: "waldo-self-issued-execution", ExpectedOutcome: "REFUSED", ExpectedReason: "WALDO_CANNOT_SELF_ISSUE_EXECUTION"},
		{CaseID: "request-permit-mismatch", ExpectedOutcome: "REFUSED", ExpectedReason: "EXECUTION_REQUEST_PERMIT_MISMATCH"},
		{CaseID: "auto-build-missing-body", ExpectedOutcome: "REFUSED", ExpectedReason: "AUTO_BUILD_FORBIDDEN"},
		{CaseID: "invent-missing-body", ExpectedOutcome: "REFUSED", ExpectedReason: "MISSING_BODY_INVENTION_FORBIDDEN"},
		{CaseID: "request-side-effect-escalation", ExpectedOutcome: "REFUSED", ExpectedReason: "EXECUTION_REQUEST_SCOPE_EXCEEDED"},
	}
	if len(cases) != len(expected) {
		return errors.New("v0.19 refusal case count drifted")
	}
	seen := map[string]bool{}
	for i, c := range cases {
		if seen[c.CaseID] {
			return fmt.Errorf("v0.19 duplicate refusal case %s", c.CaseID)
		}
		seen[c.CaseID] = true
		if c != expected[i] {
			return fmt.Errorf("v0.19 refusal case %d drifted", i)
		}
	}
	return nil
}
