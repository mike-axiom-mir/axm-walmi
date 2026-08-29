package axmmirror

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestHermesRunCapsuleV022Fixture(t *testing.T) {
	data, err := os.ReadFile("testdata/hermes-run-capsule-evidence-v0.22.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := VerifyHermesRunCapsuleV022Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	if c.ReceiptDigest == "" {
		t.Fatal("missing receipt")
	}
	w, err := WitnessHermesRunCapsuleV022Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	if w.DonorFileCount != 7 || w.CaseCount != 7 || len(w.WitnessDigest) != 64 {
		t.Fatalf("unexpected witness %+v", w)
	}
}

func TestHermesRunCapsuleV022Cases(t *testing.T) {
	data, _ := os.ReadFile("testdata/hermes-run-capsule-evidence-v0.22.json")
	c, err := VerifyHermesRunCapsuleV022Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range c.Cases {
		o, r := EvaluateHermesRunCapsuleV022Case(tc)
		if o != tc.ExpectedOutcome || r != tc.ExpectedReason {
			t.Fatalf("%s got %s/%s", tc.CaseID, o, r)
		}
	}
}

func TestHermesRunCapsuleV022SemanticTamperResealed(t *testing.T) {
	data, _ := os.ReadFile("testdata/hermes-run-capsule-evidence-v0.22.json")
	var base HermesRunCapsuleV022Contract
	if err := json.Unmarshal(data, &base); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*HermesRunCapsuleV022Contract)
	}{
		{"platform head", func(c *HermesRunCapsuleV022Contract) { c.Source.PlatformHead = "deadbeef" }},
		{"gate evidence", func(c *HermesRunCapsuleV022Contract) { c.Source.RuntimeGateConclusion = "failure" }},
		{"copy runtime", func(c *HermesRunCapsuleV022Contract) { c.Boundary.PlatformHermesRuntimeCopied = true }},
		{"live run claim", func(c *HermesRunCapsuleV022Contract) { c.Boundary.LiveHermesRunObserved = true }},
		{"promotion claim", func(c *HermesRunCapsuleV022Contract) { c.Truth.ReturnPacketGrantsPromotion = true }},
		{"waldo ownership", func(c *HermesRunCapsuleV022Contract) { c.Truth.WaldoOwnsHermesRuntime = true }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copy := base
			copy.Source.DonorFiles = append([]HermesRunCapsuleV022DonorFile(nil), base.Source.DonorFiles...)
			copy.Cases = append([]HermesRunCapsuleV022Case(nil), base.Cases...)
			tt.mutate(&copy)
			sealed, err := SealHermesRunCapsuleV022Contract(copy)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := VerifyHermesRunCapsuleV022Contract(sealed); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestHermesRunCapsuleV022StaleAndUnknown(t *testing.T) {
	data, _ := os.ReadFile("testdata/hermes-run-capsule-evidence-v0.22.json")
	var c HermesRunCapsuleV022Contract
	json.Unmarshal(data, &c)
	c.ReceiptDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	stale, _ := json.Marshal(c)
	if _, err := VerifyHermesRunCapsuleV022Contract(stale); err == nil {
		t.Fatal("expected stale receipt rejection")
	}
	unknown := bytes.Replace(data, []byte(`"authority": "NONE"`), []byte(`"unexpected": true, "authority": "NONE"`), 1)
	if _, err := VerifyHermesRunCapsuleV022Contract(unknown); err == nil {
		t.Fatal("expected unknown field rejection")
	}
}
