package axmmirror

import (
	"encoding/json"
	"os"
	"testing"
)

func loadChoiceV017(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/capability-choice-contract-v0.17.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCapabilityChoiceV017Fixture(t *testing.T) {
	b := loadChoiceV017(t)
	c, err := VerifyCapabilityChoiceV017(b)
	if err != nil {
		t.Fatal(err)
	}
	if c.ReceiptDigest != "sha256:62d12d9ef02d3527878ba5e1c9baee6e79dc973d3617caf03796170fc5787dbe" {
		t.Fatalf("receipt %s", c.ReceiptDigest)
	}
	w, err := WitnessCapabilityChoiceV017(b)
	if err != nil {
		t.Fatal(err)
	}
	if w.CandidateSetDigest != "sha256:d396b84ddaa15f230273e96cb7b6c35dfece8d4d81ffbb46a127cccb2e0b8aa0" {
		t.Fatalf("candidate set %s", w.CandidateSetDigest)
	}
	if w.WitnessDigest != "06d3528871b5e3c243af8eac7462a16b919c8d3258bf9a5c171f22dabcd4daf7" {
		t.Fatalf("witness %s", w.WitnessDigest)
	}
}

func TestCapabilityChoiceV017OrderInvariant(t *testing.T) {
	b := loadChoiceV017(t)
	var c CapabilityChoiceV017Contract
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	original, err := capChoiceV017CandidateSetDigest(c.Candidates)
	if err != nil {
		t.Fatal(err)
	}
	c.Candidates[0], c.Candidates[1] = c.Candidates[1], c.Candidates[0]
	sealed, err := SealCapabilityChoiceV017(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = VerifyCapabilityChoiceV017(sealed); err != nil {
		t.Fatal(err)
	}
	w, err := WitnessCapabilityChoiceV017(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if w.CandidateSetDigest != original {
		t.Fatalf("order changed set identity: %s != %s", w.CandidateSetDigest, original)
	}
	if w.Decision != "HOLD" || w.Reason != "MULTIPLE_VALID_COMPOSITIONS" {
		t.Fatalf("order changed decision: %s/%s", w.Decision, w.Reason)
	}
}

func TestCapabilityChoiceV017SingleSurvivorStillHolds(t *testing.T) {
	b := loadChoiceV017(t)
	var c CapabilityChoiceV017Contract
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	d, r := capChoiceV017EvaluateSet([]CapabilityChoiceV017Chain{capChoiceV017Find(c.Candidates, "direct-read-chain")}, "NONE")
	if d != "HOLD" || r != "SELECTION_AUTHORITY_ABSENT" {
		t.Fatalf("got %s/%s", d, r)
	}
}

func TestCapabilityChoiceV017TamperRejections(t *testing.T) {
	cases := map[string]func(*CapabilityChoiceV017Contract){
		"silent selection":     func(c *CapabilityChoiceV017Contract) { c.SelectionPerformed = true },
		"waldo owns selection": func(c *CapabilityChoiceV017Contract) { c.WaldoOwnsSelection = true },
		"real execution":       func(c *CapabilityChoiceV017Contract) { c.LocalCapabilityFabricExecuted = true },
		"priority smuggling":   func(c *CapabilityChoiceV017Contract) { n := 1; c.AdvisoryPriority["direct-read-chain"] = &n },
		"write authority": func(c *CapabilityChoiceV017Contract) {
			c.Candidates[0].Links[1].Authority = append(c.Candidates[0].Links[1].Authority, "WRITE_WORKSPACE")
		},
		"source rewrite": func(c *CapabilityChoiceV017Contract) { c.SourceHead = "rewritten" },
		"cycle erased":   func(c *CapabilityChoiceV017Contract) { c.CycleEdges = nil },
	}
	b := loadChoiceV017(t)
	for name, mut := range cases {
		t.Run(name, func(t *testing.T) {
			var c CapabilityChoiceV017Contract
			if err := json.Unmarshal(b, &c); err != nil {
				t.Fatal(err)
			}
			mut(&c)
			sealed, err := SealCapabilityChoiceV017(c)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = VerifyCapabilityChoiceV017(sealed); err == nil {
				t.Fatal("expected semantic rejection")
			}
		})
	}
}

func TestCapabilityChoiceV017RejectsStaleReceiptAndUnknownFields(t *testing.T) {
	b := loadChoiceV017(t)
	var c CapabilityChoiceV017Contract
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	c.ReceiptDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	bad, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = VerifyCapabilityChoiceV017(bad); err == nil {
		t.Fatal("expected stale receipt rejection")
	}
	var raw map[string]any
	if err = json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	raw["mystery"] = true
	unknown, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = VerifyCapabilityChoiceV017(unknown); err == nil {
		t.Fatal("expected unknown field rejection")
	}
}
