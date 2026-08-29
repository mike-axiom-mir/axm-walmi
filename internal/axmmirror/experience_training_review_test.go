package axmmirror

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMirrorExperienceTrainingPromotionRequiresIndependentEvidenceReview(t *testing.T) {
	captured := time.Date(2026, 8, 24, 16, 30, 0, 0, time.UTC)
	learning := reviewedExperienceTestLearningRecord(t, captured)

	circular := MirrorExperienceTrainingReview{
		Schema:                MirrorExperienceTrainingReviewSchema,
		LearningRecordSHA256:  learning.LearningRecordSHA256,
		Decision:              MirrorExperienceReviewApprove,
		EvidenceClass:         MirrorExperienceEvidenceUserConfirmed,
		EvidenceReceiptSHA256: learning.LearningRecordSHA256,
		Rationale:             "This deliberately tries to use the candidate as its own evidence.",
		ReviewedAt:            captured.Add(time.Minute),
		Authority:             Authority{},
	}
	if err := SealMirrorExperienceTrainingReview(&circular); err == nil || !strings.Contains(err.Error(), "independent") {
		t.Fatalf("circular review error = %v", err)
	}

	review := MirrorExperienceTrainingReview{
		Schema:                MirrorExperienceTrainingReviewSchema,
		LearningRecordSHA256:  learning.LearningRecordSHA256,
		Decision:              MirrorExperienceReviewApprove,
		EvidenceClass:         MirrorExperienceEvidenceUserConfirmed,
		EvidenceReceiptSHA256: strings.Repeat("b", 64),
		Rationale:             "A separate explicit user-confirmation receipt agrees with the successful outcome.",
		ReviewedAt:            captured.Add(time.Minute),
		Authority:             Authority{},
	}
	if err := SealMirrorExperienceTrainingReview(&review); err != nil {
		t.Fatal(err)
	}
	promotion, err := PromoteMirrorExperienceLearning(learning, review)
	if err != nil {
		t.Fatal(err)
	}
	if promotion == nil || promotion.SourceLearningRecordSHA256 != learning.LearningRecordSHA256 || promotion.ReviewSHA256 != review.ReviewSHA256 || promotion.EvidenceReceiptSHA256 != review.EvidenceReceiptSHA256 {
		t.Fatalf("promotion = %+v", promotion)
	}
	if promotion.PromotionSHA256 == "" || promotion.TrainingText != learning.TrainingText {
		t.Fatalf("promotion digest/text = %+v", promotion)
	}
	if _, err := promotion.JSONLine(); err != nil {
		t.Fatal(err)
	}
}

func TestMirrorExperienceTrainingPromotionRejectDecisionProducesNoPositiveTarget(t *testing.T) {
	captured := time.Date(2026, 8, 24, 16, 35, 0, 0, time.UTC)
	learning := reviewedExperienceTestLearningRecord(t, captured)
	review := MirrorExperienceTrainingReview{
		Schema:                MirrorExperienceTrainingReviewSchema,
		LearningRecordSHA256:  learning.LearningRecordSHA256,
		Decision:              MirrorExperienceReviewReject,
		EvidenceClass:         MirrorExperienceEvidenceIndependentReview,
		EvidenceReceiptSHA256: strings.Repeat("c", 64),
		Rationale:             "Independent review found insufficient support for positive weight training.",
		ReviewedAt:            captured.Add(time.Minute),
		Authority:             Authority{},
	}
	if err := SealMirrorExperienceTrainingReview(&review); err != nil {
		t.Fatal(err)
	}
	promotion, err := PromoteMirrorExperienceLearning(learning, review)
	if err != nil {
		t.Fatal(err)
	}
	if promotion != nil {
		t.Fatalf("rejected review produced training promotion: %+v", promotion)
	}
}

func TestMirrorExperienceTrainingPromotionRejectsReflectionAsEvidence(t *testing.T) {
	captured := time.Date(2026, 8, 24, 16, 40, 0, 0, time.UTC)
	learning := reviewedExperienceTestLearningRecord(t, captured)
	review := MirrorExperienceTrainingReview{
		Schema:                MirrorExperienceTrainingReviewSchema,
		LearningRecordSHA256:  learning.LearningRecordSHA256,
		Decision:              MirrorExperienceReviewApprove,
		EvidenceClass:         MirrorExperienceEvidenceToolVerified,
		EvidenceReceiptSHA256: learning.ReflectionEventSHA256,
		Rationale:             "This deliberately tries to recycle the reflection as verification evidence.",
		ReviewedAt:            captured.Add(time.Minute),
		Authority:             Authority{},
	}
	if err := SealMirrorExperienceTrainingReview(&review); err != nil {
		t.Fatal(err)
	}
	if _, err := PromoteMirrorExperienceLearning(learning, review); err == nil || !strings.Contains(err.Error(), "reflection") {
		t.Fatalf("reflection-evidence error = %v", err)
	}
}

func TestReviewedExperienceBridgesIntoGroundWithTransitiveReceipt(t *testing.T) {
	captured := time.Date(2026, 8, 24, 16, 45, 0, 0, time.UTC)
	learning := reviewedExperienceTestLearningRecord(t, captured)
	review := MirrorExperienceTrainingReview{
		Schema:                MirrorExperienceTrainingReviewSchema,
		LearningRecordSHA256:  learning.LearningRecordSHA256,
		Decision:              MirrorExperienceReviewApprove,
		EvidenceClass:         MirrorExperienceEvidenceExecutionVerified,
		EvidenceReceiptSHA256: strings.Repeat("d", 64),
		Rationale:             "A separate execution receipt verifies the observed result.",
		ReviewedAt:            captured.Add(time.Minute),
		Authority:             Authority{},
	}
	if err := SealMirrorExperienceTrainingReview(&review); err != nil {
		t.Fatal(err)
	}
	promotion, err := PromoteMirrorExperienceLearning(learning, review)
	if err != nil {
		t.Fatal(err)
	}
	ground, err := promotion.GroundRecord(
		MirrorGroundObservedExecutionTrace,
		[]string{"evidence-before-authority", "truth-before-story"},
		"REVIEWED_EXECUTION_OUTCOME",
		review.ReviewedAt.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	if ground.SourceReceiptSHA256 != promotion.PromotionSHA256 || ground.EvidenceSignal != MirrorGroundHelpful || ground.TrainingDisposition != MirrorGroundPositiveTarget {
		t.Fatalf("ground = %+v", ground)
	}
	line, err := ground.JSONLine()
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := LoadMirrorGroundDataset(bytes.NewReader(line))
	if err != nil {
		t.Fatal(err)
	}
	projection, err := dataset.TrainingProjectionJSONL()
	if err != nil {
		t.Fatal(err)
	}
	if dataset.Observed != 1 || dataset.TrainingTargets != 1 || !bytes.Contains(projection, []byte(`"data_class":"OBSERVED_EXECUTION_TRACE"`)) {
		t.Fatalf("dataset = %+v; projection = %s", dataset, projection)
	}
}

func TestMirrorExperienceTrainingReviewAndPromotionRejectTamperAndUnknownFields(t *testing.T) {
	captured := time.Date(2026, 8, 24, 16, 50, 0, 0, time.UTC)
	learning := reviewedExperienceTestLearningRecord(t, captured)
	review := MirrorExperienceTrainingReview{
		Schema:                MirrorExperienceTrainingReviewSchema,
		LearningRecordSHA256:  learning.LearningRecordSHA256,
		Decision:              MirrorExperienceReviewApprove,
		EvidenceClass:         MirrorExperienceEvidenceUserConfirmed,
		EvidenceReceiptSHA256: strings.Repeat("e", 64),
		Rationale:             "Separate confirmation supports the promotion.",
		ReviewedAt:            captured.Add(time.Minute),
		Authority:             Authority{},
	}
	if err := SealMirrorExperienceTrainingReview(&review); err != nil {
		t.Fatal(err)
	}
	line, err := review.JSONLine()
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err != nil {
		t.Fatal(err)
	}
	raw["hiddenApproval"] = true
	unknown, _ := json.Marshal(raw)
	if _, err := LoadMirrorExperienceTrainingReview(bytes.NewReader(unknown)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field error = %v", err)
	}

	promotion, err := PromoteMirrorExperienceLearning(learning, review)
	if err != nil {
		t.Fatal(err)
	}
	promotion.TargetResponse = "silently rewritten target"
	if err := promotion.Validate(); err == nil {
		t.Fatal("tampered reviewed training promotion was accepted")
	}
}

func reviewedExperienceTestLearningRecord(t *testing.T, captured time.Time) MirrorExperienceLearningRecord {
	t.Helper()
	record := MirrorExperienceLearningRecord{
		Schema:                MirrorExperienceLearningSchema,
		EpisodeID:             "episode-reviewed-training",
		OutcomeSignal:         MirrorExperienceHelpful,
		TrainingReady:         true,
		Prompt:                "Did the bounded change solve the observed task?",
		TargetResponse:        "Yes. The verified result matches the requested bounded change.",
		Feedback:              "The observed task completed successfully.",
		Lesson:                "Retain the successful pattern only when the evidence remains comparable.",
		TrainingText:          "User: Did the bounded change solve the observed task?\n\nAssistant: Yes. The verified result matches the requested bounded change.",
		ReflectionEventSHA256: strings.Repeat("a", 64),
		Authority:             Authority{},
		CapturedAt:            captured,
	}
	if err := record.seal(); err != nil {
		t.Fatal(err)
	}
	return record
}
