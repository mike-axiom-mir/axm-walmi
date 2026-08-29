package axmmirror

import (
	"encoding/json"
	"testing"
)

func TestAnchorModelBOMSelectsCompleteRealRun(t *testing.T) {
	anchor := anchorModelForTest(t, validModelBOM())
	if anchor.State != AnchorStateAnchored {
		t.Fatalf("State = %q, want %q", anchor.State, AnchorStateAnchored)
	}
	if anchor.SourceType != "run" || anchor.SourceID != "run-0001" || anchor.RunID != "run-0001" {
		t.Fatalf("selected source = %q/%q run %q", anchor.SourceType, anchor.SourceID, anchor.RunID)
	}
	if len(anchor.Artifacts) != 3 || !hasArtifactRole(anchor.Artifacts, "weights") {
		t.Fatalf("Artifacts = %+v", anchor.Artifacts)
	}
	if err := anchor.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestAnchorModelBOMHoldsWithoutSelectedSource(t *testing.T) {
	bom := validModelBOM()
	bom.CurrentRunID = ""
	anchor := anchorModelForTest(t, bom)
	if anchor.State != AnchorStateHold || len(anchor.Holds) == 0 {
		t.Fatalf("anchor = %+v", anchor)
	}
}

func TestAnchorModelBOMRejectsSelectedSimulation(t *testing.T) {
	bom := validModelBOM()
	bom.Runs[0].Simulated = true
	data, err := json.Marshal(bom)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AnchorWALDOBOM(data); err == nil {
		t.Fatal("AnchorWALDOBOM() accepted a simulated selected run")
	}
}

func TestAnchorModelBOMRejectsIncompleteAnsweringArtifacts(t *testing.T) {
	bom := validModelBOM()
	bom.Runs[0].Artifacts = bom.Runs[0].Artifacts[:1]
	data, err := json.Marshal(bom)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AnchorWALDOBOM(data); err == nil {
		t.Fatal("AnchorWALDOBOM() accepted selected weights without configuration and tokenizer")
	}
}

func TestAnchorReleaseBOM(t *testing.T) {
	anchor := anchorReleaseForTest(t, validReleaseBOM())
	if anchor.State != AnchorStateAnchored || anchor.Subject != "model-release" || anchor.Format != "huggingface" {
		t.Fatalf("anchor = %+v", anchor)
	}
	if err := anchor.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestAnchorReleaseBOMRejectsMismatchedRunIdentity(t *testing.T) {
	bom := validReleaseBOM()
	bom.RunID = "different-run"
	data, err := json.Marshal(bom)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AnchorWALDOBOM(data); err == nil {
		t.Fatal("AnchorWALDOBOM() accepted mismatched release source_id and run_id")
	}
}

func TestCompareIdentityAllowsAppendOnlyHistory(t *testing.T) {
	expected := anchorModelForTest(t, validModelBOM())
	observedBOM := validModelBOM()
	observedBOM.Runs = append(observedBOM.Runs, waldoModelRun{
		ID: "run-0002", Stage: "finetune", Ordinal: 2,
		RunBOM: "runs/0002-finetune-run-0002/RUN-BOM.json", BOMSHA256: repeatHex("6"),
		State: "failed", Backend: waldoIdentity{Name: "torch", Revision: "2.9.0"},
	})
	observedBOM.Generated = "2026-08-15T03:05:00Z"
	observed := anchorModelForTest(t, observedBOM)

	receipt, err := CompareIdentity(expected, observed)
	if err != nil {
		t.Fatalf("CompareIdentity() error = %v", err)
	}
	if receipt.State != IdentityStateLocked || !receipt.ProvenanceChanged || len(receipt.Notices) == 0 {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestCompareIdentityDetectsSelectedRunDrift(t *testing.T) {
	expected := anchorModelForTest(t, validModelBOM())
	observedBOM := validModelBOM()
	observedBOM.Runs = append(observedBOM.Runs, waldoModelRun{
		ID: "run-0002", Stage: "finetune", Ordinal: 2,
		RunBOM: "runs/0002-finetune-run-0002/RUN-BOM.json", BOMSHA256: repeatHex("6"),
		State: "complete", Backend: waldoIdentity{Name: "torch", Revision: "2.9.0"},
		Artifacts: []AnchoredArtifact{{
			Role: "weights", Path: "runs/0002-finetune-run-0002/artifacts/model.safetensors",
			SHA256: repeatHex("7"), Bytes: 2048,
		}, {
			Role: "configuration", Path: "runs/0002-finetune-run-0002/artifacts/config.json",
			SHA256: repeatHex("8"), Bytes: 256,
		}, {
			Role: "tokenizer", Path: "runs/0002-finetune-run-0002/artifacts/tokenizer.json",
			SHA256: repeatHex("9"), Bytes: 512,
		}},
	})
	observedBOM.CurrentRunID = "run-0002"
	observed := anchorModelForTest(t, observedBOM)

	receipt, err := CompareIdentity(expected, observed)
	if err != nil {
		t.Fatalf("CompareIdentity() error = %v", err)
	}
	if receipt.State != IdentityStateDrift || len(receipt.Changes) == 0 {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestCompareIdentityTreatsReleaseBOMAsImmutableIdentity(t *testing.T) {
	expected := anchorReleaseForTest(t, validReleaseBOM())
	observedBOM := validReleaseBOM()
	observedBOM.Generated = "2026-08-15T03:10:00Z"
	observed := anchorReleaseForTest(t, observedBOM)

	receipt, err := CompareIdentity(expected, observed)
	if err != nil {
		t.Fatalf("CompareIdentity() error = %v", err)
	}
	if receipt.State != IdentityStateDrift || !contains(receipt.Changes, "release_bom_sha256") {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestSealAnchoredRequiresMatchingReleaseBOM(t *testing.T) {
	anchor := anchorReleaseForTest(t, validReleaseBOM())
	draft := validDraft()
	draft.WALDO.ModelBOMSHA256 = ""
	draft.WALDO.ReleaseBOMSHA256 = anchor.BOMSHA256
	sealed, err := SealAnchored(draft, anchor)
	if err != nil {
		t.Fatalf("SealAnchored() error = %v", err)
	}
	if err := sealed.Verify(); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	draft.WALDO.ReleaseBOMSHA256 = repeatHex("8")
	if _, err := SealAnchored(draft, anchor); err == nil {
		t.Fatal("SealAnchored() accepted a mismatched release BOM digest")
	}
}

func validModelBOM() waldoModelBOM {
	return waldoModelBOM{
		Kind: "openwaldo-bom", Schema: 1, Subject: "model",
		ModelID: "model-001", Name: "mirror-smoke",
		PlanSHA256: repeatHex("1"), ArchitectureSHA256: repeatHex("2"), PathBase: "model-root",
		CurrentRunID: "run-0001",
		Runs: []waldoModelRun{{
			ID: "run-0001", Stage: "pretrain", Ordinal: 1,
			RunBOM: "runs/0001-pretrain-run-0001/RUN-BOM.json", BOMSHA256: repeatHex("3"),
			State: "complete", Backend: waldoIdentity{Name: "torch", Revision: "2.9.0"},
			Artifacts: []AnchoredArtifact{{
				Role: "weights", Path: "runs/0001-pretrain-run-0001/artifacts/model.safetensors",
				SHA256: repeatHex("4"), Bytes: 1024,
			}, {
				Role: "configuration", Path: "runs/0001-pretrain-run-0001/artifacts/config.json",
				SHA256: repeatHex("5"), Bytes: 256,
			}, {
				Role: "tokenizer", Path: "runs/0001-pretrain-run-0001/artifacts/tokenizer.json",
				SHA256: repeatHex("6"), Bytes: 512,
			}},
		}},
		Generated: "2026-08-15T03:00:00Z",
	}
}

func validReleaseBOM() waldoReleaseBOM {
	return waldoReleaseBOM{
		Kind: "openwaldo-bom", Schema: 1, Subject: "model-release", Format: "huggingface",
		ModelID: "model-001", Name: "mirror-smoke", SourceType: "run", SourceID: "run-0001",
		RunID: "run-0001", SourceBOM: repeatHex("5"),
		Artifacts: []AnchoredArtifact{{
			Role: "weights", Path: "model.safetensors", SHA256: repeatHex("4"), Bytes: 1024,
		}},
		Generated: "2026-08-15T03:00:00Z",
	}
}

func anchorModelForTest(t *testing.T, bom waldoModelBOM) OriginAnchor {
	t.Helper()
	data, err := json.Marshal(bom)
	if err != nil {
		t.Fatal(err)
	}
	anchor, err := AnchorWALDOBOM(data)
	if err != nil {
		t.Fatalf("AnchorWALDOBOM() error = %v", err)
	}
	return anchor
}

func anchorReleaseForTest(t *testing.T, bom waldoReleaseBOM) OriginAnchor {
	t.Helper()
	data, err := json.Marshal(bom)
	if err != nil {
		t.Fatal(err)
	}
	anchor, err := AnchorWALDOBOM(data)
	if err != nil {
		t.Fatalf("AnchorWALDOBOM() error = %v", err)
	}
	return anchor
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
