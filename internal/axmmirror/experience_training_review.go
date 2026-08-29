package axmmirror

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	MirrorExperienceTrainingReviewSchema   = "axm.waldo.mirror-experience-training-review/v0.40"
	MirrorExperienceReviewedTrainingSchema = "axm.waldo.mirror-experience-reviewed-training/v0.40"

	MirrorExperienceReviewApprove = "APPROVE_TRAINING"
	MirrorExperienceReviewReject  = "REJECT_TRAINING"

	MirrorExperienceEvidenceUserConfirmed     = "USER_CONFIRMED"
	MirrorExperienceEvidenceToolVerified      = "TOOL_VERIFIED"
	MirrorExperienceEvidenceExecutionVerified = "EXECUTION_VERIFIED"
	MirrorExperienceEvidenceIndependentReview = "INDEPENDENT_REVIEW"
)

// MirrorExperienceTrainingReview is an explicit promotion decision over an
// already sealed experience-learning candidate. The evidence receipt must come
// from outside the candidate itself so a model cannot turn its own output hash
// into evidence that the output was correct.
type MirrorExperienceTrainingReview struct {
	Schema                string    `json:"schema"`
	LearningRecordSHA256  string    `json:"learningRecordSha256"`
	Decision              string    `json:"decision"`
	EvidenceClass         string    `json:"evidenceClass"`
	EvidenceReceiptSHA256 string    `json:"evidenceReceiptSha256"`
	Rationale             string    `json:"rationale"`
	ReviewedAt            time.Time `json:"reviewedAt"`
	Authority             Authority `json:"authority"`
	ReviewSHA256          string    `json:"reviewSha256"`
}

// MirrorExperienceReviewedTrainingRecord is the only v0.40 object produced by
// this gate that represents an evidence-reviewed positive training target. It
// still carries no training or execution authority; a separate WALDO training
// command remains necessary for any weight mutation.
type MirrorExperienceReviewedTrainingRecord struct {
	Schema                      string    `json:"schema"`
	EpisodeID                   string    `json:"episodeId"`
	OutcomeSignal               string    `json:"outcomeSignal"`
	Prompt                      string    `json:"prompt"`
	TargetResponse              string    `json:"targetResponse"`
	TrainingText                string    `json:"text"`
	SourceLearningRecordSHA256  string    `json:"sourceLearningRecordSha256"`
	SourceReflectionEventSHA256 string    `json:"sourceReflectionEventSha256"`
	ReviewSHA256                string    `json:"reviewSha256"`
	EvidenceClass               string    `json:"evidenceClass"`
	EvidenceReceiptSHA256       string    `json:"evidenceReceiptSha256"`
	ExperienceCapturedAt        time.Time `json:"experienceCapturedAt"`
	ReviewedAt                  time.Time `json:"reviewedAt"`
	Authority                   Authority `json:"authority"`
	PromotionSHA256             string    `json:"promotionSha256"`
}

func SealMirrorExperienceTrainingReview(review *MirrorExperienceTrainingReview) error {
	if review == nil {
		return errors.New("Mirror experience training review is required")
	}
	review.ReviewSHA256 = ""
	payload, err := json.Marshal(review)
	if err != nil {
		return err
	}
	review.ReviewSHA256 = digestMirrorText(string(payload))
	return review.Validate()
}

func (review MirrorExperienceTrainingReview) Validate() error {
	if review.Schema != MirrorExperienceTrainingReviewSchema {
		return fmt.Errorf("schema must be %q", MirrorExperienceTrainingReviewSchema)
	}
	if !review.Authority.closed() {
		return errors.New("experience training review carries no execution, training, promotion, canon, or world-action authority")
	}
	if !validMirrorSHA(review.LearningRecordSHA256) || !validMirrorSHA(review.EvidenceReceiptSHA256) {
		return errors.New("experience training review requires valid learning and evidence receipt digests")
	}
	if review.LearningRecordSHA256 == review.EvidenceReceiptSHA256 {
		return errors.New("evidence receipt must be independent of the learning candidate")
	}
	if review.ReviewedAt.IsZero() {
		return errors.New("reviewedAt is required")
	}
	if !validMirrorExperienceEvidenceClass(review.EvidenceClass) {
		return errors.New("evidenceClass must be USER_CONFIRMED, TOOL_VERIFIED, EXECUTION_VERIFIED, or INDEPENDENT_REVIEW")
	}
	switch review.Decision {
	case MirrorExperienceReviewApprove, MirrorExperienceReviewReject:
	default:
		return errors.New("decision must be APPROVE_TRAINING or REJECT_TRAINING")
	}
	if strings.TrimSpace(review.Rationale) == "" || len(review.Rationale) > MaxMirrorEscalationTextBytes {
		return fmt.Errorf("review rationale is required and limited to %d bytes", MaxMirrorEscalationTextBytes)
	}
	copy := review
	copy.ReviewSHA256 = ""
	payload, err := json.Marshal(copy)
	if err != nil {
		return err
	}
	if review.ReviewSHA256 != digestMirrorText(string(payload)) {
		return errors.New("experience training review digest mismatch")
	}
	return nil
}

func (review MirrorExperienceTrainingReview) JSONLine() ([]byte, error) {
	if err := review.Validate(); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(review)
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}

func LoadMirrorExperienceTrainingReview(reader io.Reader) (MirrorExperienceTrainingReview, error) {
	var review MirrorExperienceTrainingReview
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&review); err != nil {
		return review, fmt.Errorf("decode Mirror experience training review: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("trailing JSON")
		}
		return MirrorExperienceTrainingReview{}, fmt.Errorf("decode Mirror experience training review: %w", err)
	}
	if err := review.Validate(); err != nil {
		return MirrorExperienceTrainingReview{}, err
	}
	return review, nil
}

func LoadMirrorExperienceLearningRecord(reader io.Reader) (MirrorExperienceLearningRecord, error) {
	var record MirrorExperienceLearningRecord
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return record, fmt.Errorf("decode Mirror experience learning record: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("trailing JSON")
		}
		return MirrorExperienceLearningRecord{}, fmt.Errorf("decode Mirror experience learning record: %w", err)
	}
	if err := record.Validate(); err != nil {
		return MirrorExperienceLearningRecord{}, err
	}
	return record, nil
}

// PromoteMirrorExperienceLearning performs the evidence gate. A rejection is a
// successful review with no positive training record, not an error; the source
// experience remains available to the existing visible-memory path.
func PromoteMirrorExperienceLearning(record MirrorExperienceLearningRecord, review MirrorExperienceTrainingReview) (*MirrorExperienceReviewedTrainingRecord, error) {
	if err := record.Validate(); err != nil {
		return nil, fmt.Errorf("validate experience learning candidate: %w", err)
	}
	if err := review.Validate(); err != nil {
		return nil, fmt.Errorf("validate experience training review: %w", err)
	}
	if review.LearningRecordSHA256 != record.LearningRecordSHA256 {
		return nil, errors.New("training review does not bind the supplied learning candidate")
	}
	if review.ReviewedAt.Before(record.CapturedAt) {
		return nil, errors.New("training review cannot predate the captured experience")
	}
	if review.EvidenceReceiptSHA256 == record.ReflectionEventSHA256 {
		return nil, errors.New("evidence receipt must be independent of the experience reflection")
	}
	if review.Decision == MirrorExperienceReviewReject {
		return nil, nil
	}
	promotion := MirrorExperienceReviewedTrainingRecord{
		Schema:                      MirrorExperienceReviewedTrainingSchema,
		EpisodeID:                   record.EpisodeID,
		OutcomeSignal:               record.OutcomeSignal,
		Prompt:                      record.Prompt,
		TargetResponse:              record.TargetResponse,
		TrainingText:                record.TrainingText,
		SourceLearningRecordSHA256:  record.LearningRecordSHA256,
		SourceReflectionEventSHA256: record.ReflectionEventSHA256,
		ReviewSHA256:                review.ReviewSHA256,
		EvidenceClass:               review.EvidenceClass,
		EvidenceReceiptSHA256:       review.EvidenceReceiptSHA256,
		ExperienceCapturedAt:        record.CapturedAt,
		ReviewedAt:                  review.ReviewedAt,
		Authority:                   Authority{},
	}
	if err := promotion.seal(); err != nil {
		return nil, err
	}
	return &promotion, nil
}

func (record *MirrorExperienceReviewedTrainingRecord) seal() error {
	if record == nil {
		return errors.New("reviewed training record is required")
	}
	record.PromotionSHA256 = ""
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	record.PromotionSHA256 = digestMirrorText(string(payload))
	return record.Validate()
}

func (record MirrorExperienceReviewedTrainingRecord) Validate() error {
	if record.Schema != MirrorExperienceReviewedTrainingSchema {
		return fmt.Errorf("schema must be %q", MirrorExperienceReviewedTrainingSchema)
	}
	if !record.Authority.closed() {
		return errors.New("reviewed training record carries no execution, training, promotion, canon, or world-action authority")
	}
	if err := validateMirrorEpisodeID(record.EpisodeID); err != nil {
		return err
	}
	if record.OutcomeSignal != MirrorExperienceHelpful && record.OutcomeSignal != MirrorExperienceCorrected {
		return errors.New("reviewed training record requires a helpful or corrected outcome")
	}
	if strings.TrimSpace(record.Prompt) == "" || strings.TrimSpace(record.TargetResponse) == "" {
		return errors.New("reviewed training record requires visible prompt and target response")
	}
	if len(record.Prompt) > MaxMirrorEscalationTextBytes || len(record.TargetResponse) > MaxMirrorEscalationTextBytes {
		return fmt.Errorf("reviewed training fields are limited to %d bytes", MaxMirrorEscalationTextBytes)
	}
	if record.TrainingText != "User: "+record.Prompt+"\n\nAssistant: "+record.TargetResponse {
		return errors.New("reviewed training text does not bind its prompt and target")
	}
	for _, digest := range []string{record.SourceLearningRecordSHA256, record.SourceReflectionEventSHA256, record.ReviewSHA256, record.EvidenceReceiptSHA256} {
		if !validMirrorSHA(digest) {
			return errors.New("reviewed training provenance contains an invalid digest")
		}
	}
	if record.EvidenceReceiptSHA256 == record.SourceLearningRecordSHA256 || record.EvidenceReceiptSHA256 == record.SourceReflectionEventSHA256 || record.EvidenceReceiptSHA256 == record.ReviewSHA256 {
		return errors.New("reviewed training evidence receipt must remain independent of generated learning artifacts")
	}
	if !validMirrorExperienceEvidenceClass(record.EvidenceClass) {
		return errors.New("reviewed training evidence class is invalid")
	}
	if record.ExperienceCapturedAt.IsZero() || record.ReviewedAt.IsZero() || record.ReviewedAt.Before(record.ExperienceCapturedAt) {
		return errors.New("reviewed training timestamps are invalid")
	}
	copy := record
	copy.PromotionSHA256 = ""
	payload, err := json.Marshal(copy)
	if err != nil {
		return err
	}
	if record.PromotionSHA256 != digestMirrorText(string(payload)) {
		return errors.New("reviewed training promotion digest mismatch")
	}
	return nil
}

func (record MirrorExperienceReviewedTrainingRecord) JSONLine() ([]byte, error) {
	if err := record.Validate(); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}

// GroundRecord bridges a reviewed experience into the existing v0.39 ground
// verifier without discarding provenance. The source receipt becomes the
// reviewed-promotion digest, which transitively binds the learning candidate,
// reflection, independent evidence receipt, and review decision.
func (record MirrorExperienceReviewedTrainingRecord) GroundRecord(dataClass string, rootIDs []string, challengeKind string, createdAt time.Time) (MirrorGroundRecord, error) {
	if err := record.Validate(); err != nil {
		return MirrorGroundRecord{}, err
	}
	if dataClass != MirrorGroundObservedChat && dataClass != MirrorGroundObservedExecutionTrace {
		return MirrorGroundRecord{}, errors.New("reviewed experience can only become observed chat or execution ground")
	}
	if createdAt.IsZero() || createdAt.Before(record.ReviewedAt) {
		return MirrorGroundRecord{}, errors.New("ground record creation must occur at or after the evidence review")
	}
	observedAt := record.ExperienceCapturedAt
	ground := MirrorGroundRecord{
		Schema:              MirrorGroundRecordSchema,
		ID:                  "reviewed-" + record.PromotionSHA256[:24],
		DataClass:           dataClass,
		SourceReceiptSHA256: record.PromotionSHA256,
		ObservedAt:          &observedAt,
		CreatedAt:           createdAt.UTC(),
		RootIDs:             append([]string(nil), rootIDs...),
		ChallengeKind:       challengeKind,
		EvidenceSignal:      record.OutcomeSignal,
		TrainingDisposition: MirrorGroundPositiveTarget,
		Messages: []MirrorGroundMessage{
			{Role: "user", Content: record.Prompt},
			{Role: "assistant", Content: record.TargetResponse},
		},
		Rationale:    "Evidence-reviewed experience promotion " + record.ReviewSHA256 + " backed by " + record.EvidenceClass + " evidence.",
		TrainingText: record.TrainingText,
		Authority:    Authority{},
	}
	if err := SealMirrorGroundRecord(&ground); err != nil {
		return MirrorGroundRecord{}, err
	}
	return ground, nil
}

func validMirrorExperienceEvidenceClass(class string) bool {
	switch class {
	case MirrorExperienceEvidenceUserConfirmed,
		MirrorExperienceEvidenceToolVerified,
		MirrorExperienceEvidenceExecutionVerified,
		MirrorExperienceEvidenceIndependentReview:
		return true
	default:
		return false
	}
}
