package axmmirror

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestOpinionArtifactV023Fixture(t *testing.T) {
	data, err := os.ReadFile("testdata/opinion-artifact-v0.23.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := VerifyOpinionArtifactV023Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	if c.ReceiptDigest == "" {
		t.Fatal("missing receipt")
	}
	w, err := WitnessOpinionArtifactV023Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	if w.CaseCount != 10 || w.EvidenceRefCount == 0 || w.DissentRefCount == 0 || w.UncertaintyMarkerCount == 0 || len(w.WitnessDigest) != 64 {
		t.Fatalf("unexpected witness %+v", w)
	}
}

func TestOpinionArtifactV023Cases(t *testing.T) {
	data, _ := os.ReadFile("testdata/opinion-artifact-v0.23.json")
	c, err := VerifyOpinionArtifactV023Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range c.Cases {
		o, r := EvaluateOpinionArtifactV023Case(tc)
		if o != tc.ExpectedOutcome || r != tc.ExpectedReason {
			t.Fatalf("%s got %s/%s", tc.CaseID, o, r)
		}
	}
}

func TestOpinionArtifactV023SemanticTamperResealed(t *testing.T) {
	data, _ := os.ReadFile("testdata/opinion-artifact-v0.23.json")
	var base OpinionArtifactV023Contract
	if err := json.Unmarshal(data, &base); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*OpinionArtifactV023Contract)
	}{
		{"live opinion claim", func(c *OpinionArtifactV023Contract) { c.Boundary.LiveMachineOpinionObserved = true }},
		{"consciousness claim", func(c *OpinionArtifactV023Contract) { c.Boundary.ConsciousnessObserved = true }},
		{"internal state access", func(c *OpinionArtifactV023Contract) { c.Boundary.ModelInternalStateAccessed = true }},
		{"execution boundary", func(c *OpinionArtifactV023Contract) { c.Boundary.OpinionExecutionAllowed = true }},
		{"authority claim", func(c *OpinionArtifactV023Contract) { c.Truth.OpinionGrantsAuthority = true }},
		{"promotion claim", func(c *OpinionArtifactV023Contract) { c.Truth.OpinionGrantsPromotion = true }},
		{"parent receipt", func(c *OpinionArtifactV023Contract) {
			c.ParentReceipt = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copy := base
			copy.Cases = append([]OpinionArtifactV023Case(nil), base.Cases...)
			tt.mutate(&copy)
			sealed, err := SealOpinionArtifactV023Contract(copy)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := VerifyOpinionArtifactV023Contract(sealed); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestOpinionArtifactV023StaleAndUnknown(t *testing.T) {
	data, _ := os.ReadFile("testdata/opinion-artifact-v0.23.json")
	var c OpinionArtifactV023Contract
	json.Unmarshal(data, &c)
	c.ReceiptDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	stale, _ := json.Marshal(c)
	if _, err := VerifyOpinionArtifactV023Contract(stale); err == nil {
		t.Fatal("expected stale receipt rejection")
	}
	unknown := bytes.Replace(data, []byte(`"authority": "NONE"`), []byte(`"unexpected": true, "authority": "NONE"`), 1)
	if _, err := VerifyOpinionArtifactV023Contract(unknown); err == nil {
		t.Fatal("expected unknown field rejection")
	}
}
