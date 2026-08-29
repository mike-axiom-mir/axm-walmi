package axmmirror

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestCoupledReasoningV024Fixture(t *testing.T) {
	data, err := os.ReadFile("testdata/coupled-reasoning-v0.24.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := VerifyCoupledReasoningV024Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	if c.ReceiptDigest == "" {
		t.Fatal("missing receipt")
	}
	w, err := WitnessCoupledReasoningV024Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	if w.CaseCount != 11 || w.EventCount == 0 || w.ReplanCaseCount == 0 || w.ExecutionRequestCount == 0 || w.ReadyCaseCount != 1 || len(w.WitnessDigest) != 64 {
		t.Fatalf("unexpected witness %+v", w)
	}
}

func TestCoupledReasoningV024Cases(t *testing.T) {
	data, _ := os.ReadFile("testdata/coupled-reasoning-v0.24.json")
	c, err := VerifyCoupledReasoningV024Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range c.Cases {
		o, r := EvaluateCoupledReasoningV024Case(tc)
		if o != tc.ExpectedOutcome || r != tc.ExpectedReason {
			t.Fatalf("%s got %s/%s", tc.CaseID, o, r)
		}
	}
}

func TestCoupledReasoningV024SemanticTamperResealed(t *testing.T) {
	data, _ := os.ReadFile("testdata/coupled-reasoning-v0.24.json")
	var base CoupledReasoningV024Contract
	if err := json.Unmarshal(data, &base); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*CoupledReasoningV024Contract)
	}{
		{"parent receipt", func(c *CoupledReasoningV024Contract) {
			c.ParentReceipt = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
		}},
		{"live runtime claim", func(c *CoupledReasoningV024Contract) { c.Boundary.LiveCoupledRuntimeObserved = true }},
		{"live execution claim", func(c *CoupledReasoningV024Contract) { c.Boundary.LiveEnvironmentExecutionSeen = true }},
		{"quantum overclaim", func(c *CoupledReasoningV024Contract) { c.Boundary.QuantumEntanglementClaimed = true }},
		{"coupling authority", func(c *CoupledReasoningV024Contract) { c.Truth.CouplingGrantsExecutionAuthority = true }},
		{"permit generalizes", func(c *CoupledReasoningV024Contract) { c.Truth.PermitDoesNotGeneralize = false }},
		{"zero latency truth", func(c *CoupledReasoningV024Contract) { c.Truth.ZeroLatencyNotClaimed = false }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copy := base
			copy.Cases = append([]CoupledReasoningV024Case(nil), base.Cases...)
			tt.mutate(&copy)
			sealed, err := SealCoupledReasoningV024Contract(copy)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := VerifyCoupledReasoningV024Contract(sealed); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestCoupledReasoningV024CausalAndReceiptGuards(t *testing.T) {
	data, _ := os.ReadFile("testdata/coupled-reasoning-v0.24.json")
	var c CoupledReasoningV024Contract
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	c.Cases[0].Events[1].CausalSeq = []int{99}
	sealed, err := SealCoupledReasoningV024Contract(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyCoupledReasoningV024Contract(sealed); err == nil {
		t.Fatal("expected forward/unknown causal reference rejection")
	}

	json.Unmarshal(data, &c)
	c.ReceiptDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	stale, _ := json.Marshal(c)
	if _, err := VerifyCoupledReasoningV024Contract(stale); err == nil {
		t.Fatal("expected stale receipt rejection")
	}

	// Inject an unknown field without depending on the fixture's whitespace.
	unknown := bytes.Replace(data, []byte(`"authority":`), []byte(`"unexpected":true,"authority":`), 1)
	if bytes.Equal(unknown, data) {
		t.Fatal("failed to construct unknown-field mutation")
	}
	if _, err := VerifyCoupledReasoningV024Contract(unknown); err == nil {
		t.Fatal("expected unknown field rejection")
	}
}
