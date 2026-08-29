package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/openwaldo/waldo/internal/axmmirror"
)

func TestReviewedIntakeApprovesIntoReviewedAndGroundRecords(t *testing.T) {
	directory := t.TempDir()
	learning := makeLearningCandidate(t, "review-intake-approved")
	learningPath := filepath.Join(directory, "learning.json")
	writeJSONLine(t, learningPath, learning)

	review := axmmirror.MirrorExperienceTrainingReview{
		Schema:                axmmirror.MirrorExperienceTrainingReviewSchema,
		LearningRecordSHA256:  learning.LearningRecordSHA256,
		Decision:              axmmirror.MirrorExperienceReviewApprove,
		EvidenceClass:         axmmirror.MirrorExperienceEvidenceUserConfirmed,
		EvidenceReceiptSHA256: strings.Repeat("a", 64),
		Rationale:             "The user confirmed the observed response solved the task.",
		ReviewedAt:            learning.CapturedAt.Add(time.Minute),
		Authority:             axmmirror.Authority{},
	}
	if err := axmmirror.SealMirrorExperienceTrainingReview(&review); err != nil {
		t.Fatal(err)
	}
	reviewPath := filepath.Join(directory, "review.json")
	writeJSONLine(t, reviewPath, review)
	reviewedPath := filepath.Join(directory, "reviewed.jsonl")
	groundPath := filepath.Join(directory, "ground.jsonl")

	var output bytes.Buffer
	if err := run([]string{
		"--reviewed-to", reviewedPath,
		"--ground-to", groundPath,
		"--data-class", axmmirror.MirrorGroundObservedChat,
		"--root", "truth-before-story",
		"--challenge", "OBSERVED_HELPFUL_RESPONSE",
		"--json",
		learningPath,
		reviewPath,
	}, &output); err != nil {
		t.Fatal(err)
	}
	var receipt reviewedIntakeReceipt
	if err := json.Unmarshal(output.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Decision != axmmirror.MirrorExperienceReviewApprove || !receipt.ReviewedTrainingMutation || !receipt.GroundRecordMutation || receipt.ModelWeightMutation || receipt.IdentityMutation || receipt.MemoryDeletion {
		t.Fatalf("receipt = %+v", receipt)
	}
	if receipt.ReviewedTrainingSHA256 == "" || receipt.GroundRecordSHA256 == "" || receipt.EvidenceReceiptSHA256 != review.EvidenceReceiptSHA256 {
		t.Fatalf("receipt provenance = %+v", receipt)
	}

	for _, path := range []string{reviewedPath, groundPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != fs.FileMode(0o600) {
			t.Fatalf("%s mode = %o", path, info.Mode().Perm())
		}
	}
	groundFile, err := os.Open(groundPath)
	if err != nil {
		t.Fatal(err)
	}
	defer groundFile.Close()
	dataset, err := axmmirror.LoadMirrorGroundDataset(groundFile)
	if err != nil {
		t.Fatal(err)
	}
	if dataset.Observed != 1 || dataset.TrainingTargets != 1 || dataset.Records[0].SourceReceiptSHA256 != receipt.ReviewedTrainingSHA256 {
		t.Fatalf("ground dataset = %+v", dataset)
	}
}

func TestReviewedIntakeRejectKeepsExperienceMemoryOnly(t *testing.T) {
	directory := t.TempDir()
	learning := makeLearningCandidate(t, "review-intake-rejected")
	learningPath := filepath.Join(directory, "learning.json")
	writeJSONLine(t, learningPath, learning)
	review := axmmirror.MirrorExperienceTrainingReview{
		Schema:                axmmirror.MirrorExperienceTrainingReviewSchema,
		LearningRecordSHA256:  learning.LearningRecordSHA256,
		Decision:              axmmirror.MirrorExperienceReviewReject,
		EvidenceClass:         axmmirror.MirrorExperienceEvidenceIndependentReview,
		EvidenceReceiptSHA256: strings.Repeat("b", 64),
		Rationale:             "Independent review found the episode useful as context but insufficient for weight training.",
		ReviewedAt:            learning.CapturedAt.Add(time.Minute),
		Authority:             axmmirror.Authority{},
	}
	if err := axmmirror.SealMirrorExperienceTrainingReview(&review); err != nil {
		t.Fatal(err)
	}
	reviewPath := filepath.Join(directory, "review.json")
	writeJSONLine(t, reviewPath, review)

	var output bytes.Buffer
	if err := run([]string{"--json", learningPath, reviewPath}, &output); err != nil {
		t.Fatal(err)
	}
	var receipt reviewedIntakeReceipt
	if err := json.Unmarshal(output.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Decision != axmmirror.MirrorExperienceReviewReject || receipt.ReviewedTrainingMutation || receipt.GroundRecordMutation || receipt.MemoryDeletion || receipt.ModelWeightMutation {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestReviewedIntakeRejectsAliasedOutputsAndMissingGroundEvidence(t *testing.T) {
	directory := t.TempDir()
	learning := makeLearningCandidate(t, "review-intake-invalid")
	learningPath := filepath.Join(directory, "learning.json")
	writeJSONLine(t, learningPath, learning)
	review := axmmirror.MirrorExperienceTrainingReview{
		Schema:                axmmirror.MirrorExperienceTrainingReviewSchema,
		LearningRecordSHA256:  learning.LearningRecordSHA256,
		Decision:              axmmirror.MirrorExperienceReviewApprove,
		EvidenceClass:         axmmirror.MirrorExperienceEvidenceToolVerified,
		EvidenceReceiptSHA256: strings.Repeat("c", 64),
		Rationale:             "A tool receipt independently verifies the outcome.",
		ReviewedAt:            learning.CapturedAt.Add(time.Minute),
		Authority:             axmmirror.Authority{},
	}
	if err := axmmirror.SealMirrorExperienceTrainingReview(&review); err != nil {
		t.Fatal(err)
	}
	reviewPath := filepath.Join(directory, "review.json")
	writeJSONLine(t, reviewPath, review)
	if err := run([]string{"--reviewed-to", learningPath, learningPath, reviewPath}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "distinct") {
		t.Fatalf("alias error = %v", err)
	}
	reviewedPath := filepath.Join(directory, "reviewed.jsonl")
	groundPath := filepath.Join(directory, "ground.jsonl")
	if err := run([]string{"--reviewed-to", reviewedPath, "--ground-to", groundPath, learningPath, reviewPath}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "--root") {
		t.Fatalf("missing root error = %v", err)
	}
	if _, err := os.Stat(reviewedPath); !os.IsNotExist(err) {
		t.Fatalf("reviewed output mutated before validation: %v", err)
	}
}

func makeLearningCandidate(t *testing.T, episodeID string) axmmirror.MirrorExperienceLearningRecord {
	t.Helper()
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	request := axmmirror.MirrorNeuralEscalationRequest{
		Schema:                axmmirror.MirrorNeuralEscalationRequestSchema,
		Prompt:                "What should the machine retain from this result?",
		EscalationReason:      axmmirror.MirrorEscalationNone,
		DeterministicResponse: "Retain the verified outcome and its evidence boundary.",
		Grounding: axmmirror.LiveNeuralGrounding{
			Schema:      axmmirror.LiveNeuralGroundingSchema,
			State:       axmmirror.GroundingStable,
			Consequence: axmmirror.ConsequenceLow,
			Authority:   axmmirror.Authority{},
		},
	}
	receipt, err := axmmirror.RunMirrorNeuralEscalation(context.Background(), request, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	events, err := axmmirror.BuildMirrorExperienceEpisode(receipt, request.Prompt, episodeID, now)
	if err != nil {
		t.Fatal(err)
	}
	ledger := axmmirror.MirrorExperienceLedger{Episodes: map[string][]axmmirror.MirrorExperienceEvent{episodeID: events}}
	closure, err := axmmirror.CloseMirrorExperienceEpisode(ledger, axmmirror.MirrorExperienceOutcomeRequest{
		Schema:    axmmirror.MirrorExperienceOutcomeRequestSchema,
		EpisodeID: episodeID,
		Signal:    axmmirror.MirrorExperienceHelpful,
		Feedback:  "The verified outcome matched what happened.",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if closure.LearningRecord == nil {
		t.Fatal("helpful experience did not produce a learning candidate")
	}
	return *closure.LearningRecord
}

func writeJSONLine(t *testing.T, path string, value interface{ JSONLine() ([]byte, error) }) {
	t.Helper()
	line, err := value.JSONLine()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, line, 0o600); err != nil {
		t.Fatal(err)
	}
}
