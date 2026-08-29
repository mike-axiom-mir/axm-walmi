package axmmirror

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTrainingRunWitnessReadyForCompleteRealRun(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	witness := witnessFixture(t, bom, run)
	if witness.State != RunWitnessStateReady || witness.RunState != "complete" || witness.Simulated {
		t.Fatalf("witness state = %+v", witness)
	}
	if witness.Corpus.State != CorpusEvidenceStateReady || witness.ObservationSHA256 == "" || witness.ReceiptSHA256 == "" || len(witness.Artifacts) != 1 {
		t.Fatalf("witness evidence = %+v", witness)
	}
	if err := witness.Validate(); err != nil {
		t.Fatalf("witness.Validate() error = %v", err)
	}
}

func TestTrainingRunWitnessDetectsReceiptTamper(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	witness := witnessFixture(t, bom, run)
	witness.Execution.Runtime = "rewritten runtime"
	if err := witness.Validate(); err == nil || !strings.Contains(err.Error(), "receipt digest mismatch") {
		t.Fatalf("witness.Validate() tamper error = %v", err)
	}
}

func TestTrainingRunWitnessHoldsSimulatedRun(t *testing.T) {
	bom, run := validRunFixture(t, "fake", true, "complete")
	witness := witnessFixture(t, bom, run)
	if witness.State != RunWitnessStateHold || !witness.Simulated || len(witness.Holds) != 1 || !strings.Contains(witness.Holds[0], "simulated") {
		t.Fatalf("simulated witness = %+v", witness)
	}
}

func TestTrainingRunWitnessHoldsNonCompleteLifecycle(t *testing.T) {
	for _, state := range []string{"planned", "running", "failed", "interrupted"} {
		t.Run(state, func(t *testing.T) {
			bom, run := validRunFixture(t, "pytorch", false, state)
			witness := witnessFixture(t, bom, run)
			if witness.State != RunWitnessStateHold || len(witness.Holds) == 0 || !strings.Contains(witness.Holds[0], state) {
				t.Fatalf("%s witness = %+v", state, witness)
			}
		})
	}
}

func TestTrainingRunWitnessRejectsCorpusDigestMismatch(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	bom.CorpusBOMSHA256 = repeatHex("f")
	repinRun(t, bom, &run)
	if _, err := encodeAndWitness(bom, run); err == nil || !strings.Contains(err.Error(), "embedded corpus BOM digest mismatch") {
		t.Fatalf("WitnessTrainingRun() corpus mismatch error = %v", err)
	}
}

func TestTrainingRunWitnessRejectsRunRecordPinMismatch(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	run.BOMSHA256 = repeatHex("f")
	if _, err := encodeAndWitness(bom, run); err == nil || !strings.Contains(err.Error(), "record BOM digest mismatch") {
		t.Fatalf("WitnessTrainingRun() run pin error = %v", err)
	}
}

func TestTrainingRunWitnessRejectsInvalidObservationArtifact(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	run.Observation.Artifacts[0].Path = "../model.safetensors"
	if _, err := encodeAndWitness(bom, run); err == nil || !strings.Contains(err.Error(), "portable relative path") {
		t.Fatalf("WitnessTrainingRun() artifact error = %v", err)
	}
}

func TestSealWitnessedBindsAnchorRunAndCorpus(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	witness := witnessFixture(t, bom, run)
	release := waldoReleaseBOM{
		Kind: "openwaldo-bom", Schema: 1, Subject: "model-release", Format: "huggingface",
		ModelID: bom.ModelID, Name: "witness-test", SourceType: "run", SourceID: bom.ID, RunID: bom.ID,
		SourceBOM: repeatHex("1"), Artifacts: []AnchoredArtifact{{Role: "weights", Path: "model.safetensors", SHA256: repeatHex("2"), Bytes: 128}},
		Generated: "2026-08-15T00:00:00Z",
	}
	releaseData, err := json.Marshal(release)
	if err != nil {
		t.Fatal(err)
	}
	anchor, err := AnchorWALDOBOM(releaseData)
	if err != nil {
		t.Fatal(err)
	}
	draft := validDraft()
	draft.WALDO = WALDOLineage{ReleaseBOMSHA256: anchor.BOMSHA256, RunBOMSHA256: witness.RunBOMSHA256, CorpusBOMSHA256: witness.Corpus.BOMSHA256}
	sealed, err := SealWitnessed(draft, anchor, witness)
	if err != nil {
		t.Fatalf("SealWitnessed() error = %v", err)
	}
	if err := sealed.Verify(); err != nil {
		t.Fatalf("sealed.Verify() error = %v", err)
	}
	foundWitnessBinding := false
	for _, verification := range sealed.Record.Verification {
		if verification.Name == "waldo-training-run-witness" && verification.State == "pass" && verification.EvidenceSHA256 == witness.ReceiptSHA256 {
			foundWitnessBinding = true
		}
	}
	if !foundWitnessBinding {
		t.Fatalf("sealed behavior evidence does not bind witness receipt: %+v", sealed.Record.Verification)
	}

	draft.WALDO.CorpusBOMSHA256 = repeatHex("9")
	if _, err := SealWitnessed(draft, anchor, witness); err == nil || !strings.Contains(err.Error(), "corpus_bom_sha256") {
		t.Fatalf("SealWitnessed() corpus mismatch error = %v", err)
	}
}

func TestSealWitnessedRefusesHeldRun(t *testing.T) {
	bom, run := validRunFixture(t, "fake", true, "complete")
	witness := witnessFixture(t, bom, run)
	anchor := OriginAnchor{
		Schema: OriginAnchorSchema, State: AnchorStateAnchored, Subject: "model-release", BOMSHA256: repeatHex("1"), DocumentSHA256: repeatHex("2"),
		AnsweringIdentitySHA256: repeatHex("3"), ModelID: bom.ModelID, Name: "test", Format: "huggingface", SourceType: "run", SourceID: bom.ID,
		RunID: bom.ID, SourceBOMSHA256: repeatHex("4"), Artifacts: []AnchoredArtifact{{Role: "weights", Path: "model.safetensors", SHA256: repeatHex("5"), Bytes: 1}},
	}
	anchor.AnsweringIdentitySHA256, _ = answeringIdentityDigest(anchor)
	draft := validDraft()
	draft.WALDO = WALDOLineage{ReleaseBOMSHA256: anchor.BOMSHA256, RunBOMSHA256: witness.RunBOMSHA256, CorpusBOMSHA256: witness.Corpus.BOMSHA256}
	if _, err := SealWitnessed(draft, anchor, witness); err == nil || !strings.Contains(err.Error(), "HOLD") {
		t.Fatalf("SealWitnessed() held error = %v", err)
	}
}

func validRunFixture(t *testing.T, backend string, simulated bool, state string) (waldoRunBOM, waldoRunRecord) {
	t.Helper()
	corpus := validCorpusBOM()
	corpusData, err := json.Marshal(corpus)
	if err != nil {
		t.Fatal(err)
	}
	framework := "pytorch"
	runtime := "python test runtime"
	if backend == "fake" {
		framework = "fake"
		runtime = "explicit-test-simulation"
	}
	bom := waldoRunBOM{
		Kind: "openwaldo-bom", Schema: 1, Subject: "training-run", ID: "run-0001", ModelID: "model-0001",
		Stage: "pretrain", StageType: "pre-training", Ordinal: 1, Objective: "causal-language-modeling",
		Execution: RunExecutionEvidence{
			Backend: RunBackendEvidence{Name: backend, Revision: "test-r1"}, Framework: framework, Runtime: runtime,
			Host: RunHostEvidence{OS: "linux", Architecture: "amd64"}, Nodes: 1, WorldSize: 1,
		},
		ArchitectureSHA256: repeatHex("e"), CorpusBOMSHA256: digestBytes(corpusData), CorpusBOM: corpus,
		Parameters: waldoResolvedParameters{
			Profile: "causal-pretrain-v1", ProfileSchema: 1, Epochs: 1, Steps: 2, BatchSize: 1, SequenceLength: 4,
			LearningRate: 0.001, Seed: 7,
			Optimizer:       waldoOptimizer{Name: "adamw", WeightDecay: 0.1, Beta1: 0.9, Beta2: 0.95, Epsilon: 1e-8},
			Schedule:        waldoSchedule{Name: "cosine", WarmupSteps: 1, MinimumRateRatio: 0.1},
			Data:            waldoDataPlan{Order: "bounded-shuffle-v1", ShuffleBufferRecords: 16, ShuffleBufferBytes: 1024, Packing: "continuous-eos-v1"},
			Evaluation:      &waldoEvaluationPolicy{Selection: "lowest-sha256-v1", Fraction: 0.5, MaxRecords: 1, MaxBytes: 1024},
			CheckpointEvery: 2, EvaluateEvery: 2, PlannedTokenCapacity: 8,
		},
		EvaluationSet: &waldoEvaluationSet{Selection: "lowest-sha256-v1", Seed: 7, Records: 1, TokenTargets: 4, TextBytes: 16, SHA256: repeatHex("7")},
	}
	run := waldoRunRecord{Kind: "waldo-training-run", Schema: 1, ID: bom.ID, State: state, Planned: "2026-08-15T00:00:00Z"}
	switch state {
	case "running":
		run.Started = "2026-08-15T00:01:00Z"
		run.Attempts = []waldoRunAttempt{{Ordinal: 1, Started: run.Started, State: "running"}}
		run.Progress = &waldoRunProgress{Steps: 1, ConsumedTokens: 4}
	case "complete":
		loss := 1.25
		run.Started, run.Finished = "2026-08-15T00:01:00Z", "2026-08-15T00:02:00Z"
		run.Attempts = []waldoRunAttempt{{Ordinal: 1, Started: run.Started, Finished: run.Finished, State: "complete"}}
		run.Observation = &waldoRunObservation{
			Simulated: simulated, Steps: 2, ConsumedTokens: 8, FinalLoss: &loss,
			Evaluations: []waldoRunEvaluation{{Step: 2, Tokens: 8, Metrics: map[string]float64{"artifact_heldout_loss": 1.5, "heldout_loss": 1.5}}},
			Artifacts:   []waldoRunArtifact{{Path: "artifacts/model.safetensors", SHA256: repeatHex("8"), Bytes: 128}},
		}
	case "failed", "interrupted":
		run.Started, run.Finished, run.Error = "2026-08-15T00:01:00Z", "2026-08-15T00:02:00Z", "bounded test failure"
		run.Attempts = []waldoRunAttempt{{Ordinal: 1, Started: run.Started, Finished: run.Finished, State: state, Error: run.Error}}
	}
	repinRun(t, bom, &run)
	return bom, run
}

func repinRun(t *testing.T, bom waldoRunBOM, run *waldoRunRecord) {
	t.Helper()
	data, err := json.Marshal(bom)
	if err != nil {
		t.Fatal(err)
	}
	run.BOMSHA256 = digestBytes(data)
}

func witnessFixture(t *testing.T, bom waldoRunBOM, run waldoRunRecord) TrainingRunWitness {
	t.Helper()
	witness, err := encodeAndWitness(bom, run)
	if err != nil {
		t.Fatalf("WitnessTrainingRun() error = %v", err)
	}
	return witness
}

func encodeAndWitness(bom waldoRunBOM, run waldoRunRecord) (TrainingRunWitness, error) {
	bomData, err := json.MarshalIndent(bom, "", "  ")
	if err != nil {
		return TrainingRunWitness{}, err
	}
	runData, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return TrainingRunWitness{}, err
	}
	return WitnessTrainingRun(bomData, runData)
}
