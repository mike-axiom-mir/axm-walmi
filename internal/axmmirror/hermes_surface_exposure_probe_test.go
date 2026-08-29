package axmmirror

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestHermesExposureV021ActualFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/hermes-surface-capability-exposure-v0.21.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := VerifyHermesExposureV021Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	if c.ReceiptDigest != "sha256:e0bc916d45cdfd5702067984b69482de451e8843b1729351199aeb777e0f955c" {
		t.Fatalf("unexpected receipt digest %s", c.ReceiptDigest)
	}
	w, err := WitnessHermesExposureV021Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	if w.WitnessDigest != "143a9f3c06371c519df8fff8a528a44f31af21438aa565d899326ec3d4360e72" {
		t.Fatalf("unexpected witness digest %s", w.WitnessDigest)
	}
	if w.DonorFileCount != 3 || w.EvaluatedCases != 5 || w.RefusalCases != 4 {
		t.Fatalf("unexpected witness counts %+v", w)
	}
}

func TestHermesExposureV021CaseSemantics(t *testing.T) {
	data, err := os.ReadFile("testdata/hermes-surface-capability-exposure-v0.21.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := VerifyHermesExposureV021Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	profiles := map[string]HermesExposureV021SurfaceProfile{}
	for _, p := range c.Profiles {
		profiles[p.ProfileID] = p
	}
	for _, tc := range c.Cases {
		outcome, reason := EvaluateHermesExposureV021Case(profiles[tc.SurfaceProfileID], tc)
		if outcome != tc.ExpectedOutcome || reason != tc.ExpectedReason {
			t.Fatalf("case %s got %s/%s", tc.CaseID, outcome, reason)
		}
	}
}

func TestHermesExposureV021RejectsSemanticTamperingAfterReseal(t *testing.T) {
	data, err := os.ReadFile("testdata/hermes-surface-capability-exposure-v0.21.json")
	if err != nil {
		t.Fatal(err)
	}
	var base HermesExposureV021Contract
	if err := json.Unmarshal(data, &base); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		mutate func(*HermesExposureV021Contract)
	}{
		{"reviewed donor commit drift", func(c *HermesExposureV021Contract) { c.Source.ReviewedCommit = "deadbeef" }},
		{"donor blob continuity false", func(c *HermesExposureV021Contract) { c.Source.DonorFiles[0].Unchanged = false }},
		{"claim Hermes runtime executed", func(c *HermesExposureV021Contract) { c.Boundary.HermesRuntimeExecuted = true }},
		{"widen webhook safe tools", func(c *HermesExposureV021Contract) {
			for i := range c.Profiles {
				if c.Profiles[i].ProfileID == "webhook-untrusted-safe" {
					c.Profiles[i].ExposedTools = []string{"clarify", "terminal", "vision_analyze", "web_extract", "web_search"}
				}
			}
		}},
		{"claim surface grants authority", func(c *HermesExposureV021Contract) { c.Truth.SurfaceExposureGrantsAuthority = true }},
		{"claim silent context pruning", func(c *HermesExposureV021Contract) { c.Truth.ContextSilentlyPruned = true }},
		{"make WALDO owner", func(c *HermesExposureV021Contract) { c.Truth.WaldoOwnsSurfaceExposure = true }},
		{"rewrite v020 history", func(c *HermesExposureV021Contract) { c.Truth.HistoricalV020ReceiptRewritten = true }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			copy := base
			copy.Source.DonorFiles = append([]HermesExposureV021DonorFile(nil), base.Source.DonorFiles...)
			copy.Profiles = append([]HermesExposureV021SurfaceProfile(nil), base.Profiles...)
			for i := range copy.Profiles {
				copy.Profiles[i].ExposedTools = append([]string(nil), base.Profiles[i].ExposedTools...)
				copy.Profiles[i].ExposedToolsets = append([]string(nil), base.Profiles[i].ExposedToolsets...)
			}
			tc.mutate(&copy)
			sealed, err := SealHermesExposureV021Contract(copy)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := VerifyHermesExposureV021Contract(sealed); err == nil {
				t.Fatal("expected semantic tamper rejection")
			}
		})
	}
}

func TestHermesExposureV021RejectsStaleReceiptAndUnknownFields(t *testing.T) {
	data, err := os.ReadFile("testdata/hermes-surface-capability-exposure-v0.21.json")
	if err != nil {
		t.Fatal(err)
	}
	var c HermesExposureV021Contract
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	c.ReceiptDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	stale, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyHermesExposureV021Contract(stale); err == nil {
		t.Fatal("expected stale receipt rejection")
	}
	unknown := bytes.Replace(data, []byte(`"authority": "NONE"`), []byte(`"unexpected": true, "authority": "NONE"`), 1)
	if _, err := VerifyHermesExposureV021Contract(unknown); err == nil {
		t.Fatal("expected unknown-field rejection")
	}
}
