package axmmirror

import (
	"errors"
	"fmt"
)

func validateHermesExposureV021Refusals(cases []HermesExposureV021RefusalCase) error {
	expected := map[string]string{
		"surface-source-mismatch":   "SURFACE_SOURCE_MISMATCH",
		"exposure-grants-authority": "EXPOSURE_CANNOT_GRANT_AUTHORITY",
		"silent-context-pruning":    "CONTEXT_PRUNING_WITHOUT_RECEIPT_FORBIDDEN",
		"webhook-safe-set-widening": "WEBHOOK_SAFE_SET_WIDENING_FORBIDDEN",
	}
	if len(cases) != len(expected) {
		return fmt.Errorf("hermes exposure refusal case count mismatch: got %d", len(cases))
	}
	seen := map[string]bool{}
	for _, rc := range cases {
		want, ok := expected[rc.CaseID]
		if !ok || seen[rc.CaseID] {
			return errors.New("hermes exposure refusal case id unexpected or duplicated")
		}
		seen[rc.CaseID] = true
		if rc.ExpectedOutcome != "REFUSED" || rc.ExpectedReason != want {
			return fmt.Errorf("hermes exposure refusal case %s drifted", rc.CaseID)
		}
	}
	return nil
}
