package axmmirror

import (
	"encoding/json"
	"os"
	"testing"
)

func loadCapabilityCompositionFixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/capability-composition-contract-v0.16.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCapabilityCompositionContractActualFixture(t *testing.T) {
	data := loadCapabilityCompositionFixture(t)
	c, err := VerifyCapabilityCompositionContract(data)
	if err != nil {
		t.Fatal(err)
	}
	if c.ReceiptDigest != "sha256:7e8188eb87772ae6ae47ab67bd1dbf158008513ae43c63da680cfb388ca5f81d" {
		t.Fatalf("unexpected receipt digest %s", c.ReceiptDigest)
	}
	w1, err := WitnessCapabilityCompositionContract(data)
	if err != nil {
		t.Fatal(err)
	}
	w2, err := WitnessCapabilityCompositionContract(data)
	if err != nil {
		t.Fatal(err)
	}
	if w1.WitnessSHA256 == "" || w1.WitnessSHA256 != w2.WitnessSHA256 {
		t.Fatalf("witness not deterministic: %q vs %q", w1.WitnessSHA256, w2.WitnessSHA256)
	}
	if w1.MissingLinkHolds != 3 || w1.AuthorityRefusals != 1 || w1.Authority != "NONE" {
		t.Fatalf("unexpected witness summary: %+v", w1)
	}
}

func TestCapabilityCompositionContractRejectsResealedSemanticTampering(t *testing.T) {
	original := loadCapabilityCompositionFixture(t)
	var base CapabilityCompositionContract
	if err := json.Unmarshal(original, &base); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		mutate func(*CapabilityCompositionContract)
	}{
		{"claim real local execution", func(c *CapabilityCompositionContract) { c.ProbeBoundary.ActualCapabilityFabricExecutionObserved = true }},
		{"remove required hand", func(c *CapabilityCompositionContract) {
			c.Chain.Links = removeLink(c.Chain.Links, "public-artifact-reader-hand")
		}},
		{"widen child authority", func(c *CapabilityCompositionContract) {
			c.Chain.Links[1].RequiresAuthority = append(c.Chain.Links[1].RequiresAuthority, "WRITE_WORKSPACE")
		}},
		{"invent missing replacement", func(c *CapabilityCompositionContract) { c.MissingLinkCases[1].InventedReplacement = true }},
		{"start build", func(c *CapabilityCompositionContract) { c.Decision.BuildStarted = true }},
		{"make WALDO composition owner", func(c *CapabilityCompositionContract) { c.Truth.WaldoOwnsComposition = true }},
		{"rewrite v0.15 history", func(c *CapabilityCompositionContract) { c.Truth.HistoricalV015ReceiptRewritten = true }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := base
			c.Chain.Links = append([]CapabilityCompositionLink(nil), base.Chain.Links...)
			c.MissingLinkCases = append([]CapabilityCompositionMissingLinkCase(nil), base.MissingLinkCases...)
			tc.mutate(&c)
			resealed, err := SealCapabilityCompositionContract(c)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := VerifyCapabilityCompositionContract(resealed); err == nil {
				t.Fatal("expected resealed semantic tampering to be rejected")
			}
		})
	}
}

func TestCapabilityCompositionContractRejectsStaleReceipt(t *testing.T) {
	data := loadCapabilityCompositionFixture(t)
	var c CapabilityCompositionContract
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	c.ReceiptDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyCapabilityCompositionContract(raw); err == nil {
		t.Fatal("expected stale receipt to be rejected")
	}
}
