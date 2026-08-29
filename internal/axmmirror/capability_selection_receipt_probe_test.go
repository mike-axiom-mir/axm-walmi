package axmmirror

import (
	"encoding/json"
	"os"
	"testing"
)

func loadSelectionReceiptV018(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/capability-selection-receipt-v0.18.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSelectionReceiptV018Fixture(t *testing.T) {
	b := loadSelectionReceiptV018(t)
	c, err := VerifySelectionReceiptV018(b)
	if err != nil {
		t.Fatal(err)
	}
	if c.ReceiptDigest != "sha256:768d32a9bc59008065f09b31e8b09be6b1fd20ce77dd9d578c1b76af2474b06f" {
		t.Fatalf("receipt %s", c.ReceiptDigest)
	}
	w, err := WitnessSelectionReceiptV018(b)
	if err != nil {
		t.Fatal(err)
	}
	if w.WitnessDigest != "1bbb03715c787a397987d50fb0629c928bb4108f773e48cced8838ec1adcfe57" {
		t.Fatalf("witness %s", w.WitnessDigest)
	}
}

func TestSelectionReceiptV018DuplicateSameSelectionIsNotConflict(t *testing.T) {
	b := loadSelectionReceiptV018(t)
	var c SelectionReceiptV018Contract
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	duplicate := c.SelectionReceipt
	duplicate.ReceiptID = c.SelectionReceipt.ReceiptID
	outcome, reason := selectionReceiptV018EvaluateReceiptSet([]SelectionReceiptV018Receipt{c.SelectionReceipt, duplicate})
	if outcome != "SELECTED_PROPOSAL_ONLY" || reason != "EXECUTION_AUTHORITY_ABSENT" {
		t.Fatalf("duplicate same selection = %s/%s", outcome, reason)
	}
}

func TestSelectionReceiptV018SemanticTamperRejectedAfterReseal(t *testing.T) {
	cases := map[string]func(*SelectionReceiptV018Contract){
		"claim real governance": func(c *SelectionReceiptV018Contract) { c.RealGovernanceDecisionObserved = true },
		"waldo issued":          func(c *SelectionReceiptV018Contract) { c.WaldoIssuedSelection = true },
		"build start":           func(c *SelectionReceiptV018Contract) { c.BuildStarted = true },
		"execution authority":   func(c *SelectionReceiptV018Contract) { c.ExecutionAuthority = "BUILD" },
		"last writer wins":      func(c *SelectionReceiptV018Contract) { c.ConflictCase.LastWriterWins = true },
		"scope escalation":      func(c *SelectionReceiptV018Contract) { c.SelectionReceipt.Scope = []string{"SELECT_PROPOSAL", "BUILD"} },
		"source rewrite":        func(c *SelectionReceiptV018Contract) { c.SourceV017Head = "rewritten" },
	}
	b := loadSelectionReceiptV018(t)
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			var c SelectionReceiptV018Contract
			if err := json.Unmarshal(b, &c); err != nil {
				t.Fatal(err)
			}
			mutate(&c)
			sealed, err := SealSelectionReceiptV018(c)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := VerifySelectionReceiptV018(sealed); err == nil {
				t.Fatal("expected semantic rejection")
			}
		})
	}
}

func TestSelectionReceiptV018RejectsStaleReceiptAndUnknownField(t *testing.T) {
	b := loadSelectionReceiptV018(t)
	var c SelectionReceiptV018Contract
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	c.ReceiptDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	bad, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifySelectionReceiptV018(bad); err == nil {
		t.Fatal("expected stale receipt rejection")
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	raw["mystery"] = true
	unknown, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifySelectionReceiptV018(unknown); err == nil {
		t.Fatal("expected unknown field rejection")
	}
}
