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
	MirrorExperienceTrajectorySchema           = "axm.waldo.experience-trajectory/v0.51"
	MirrorExperienceTrajectoryProjectionSchema = "axm.waldo.experience-trajectory-training/v0.51"
	MirrorExperienceTrajectoryReceiptSchema    = "axm.waldo.experience-trajectory-projection-receipt/v0.51"
	MirrorExperienceReflectionTarget           = "OUTCOME_CONDITIONED_REFLECTION"
)

// MirrorExperienceTrajectoryRecord keeps an observed attempt and outcome in
// the training path without treating a failed reaction as an answer to copy.
// The only supervised assistant turn is the visible lesson after the observed
// trace.
type MirrorExperienceTrajectoryRecord struct {
	Schema                     string    `json:"schema"`
	ID                         string    `json:"id"`
	SourceReceiptSHA256        string    `json:"sourceReceiptSha256"`
	ObservedAt                 time.Time `json:"observedAt"`
	OutcomeSignal              string    `json:"outcomeSignal"`
	Prompt                     string    `json:"prompt"`
	AttemptTrace               string    `json:"attemptTrace"`
	ObservedOutcome            string    `json:"observedOutcome"`
	Lesson                     string    `json:"lesson"`
	TargetKind                 string    `json:"targetKind"`
	TrainingObjective          string    `json:"trainingObjective"`
	CompletionRequired         bool      `json:"completionRequired"`
	FailedAttemptSupervised    bool      `json:"failedAttemptSupervised"`
	ReflectionTargetSupervised bool      `json:"reflectionTargetSupervised"`
	Authority                  Authority `json:"authority"`
	RecordSHA256               string    `json:"recordSha256"`
}

type MirrorExperienceTrajectoryProjectionReceipt struct {
	Schema                     string    `json:"schema"`
	State                      string    `json:"state"`
	RecordSHA256               string    `json:"recordSha256"`
	SourceReceiptSHA256        string    `json:"sourceReceiptSha256"`
	ProjectionSHA256           string    `json:"projectionSha256"`
	OutcomeSignal              string    `json:"outcomeSignal"`
	TargetKind                 string    `json:"targetKind"`
	TrainingObjective          string    `json:"trainingObjective"`
	ObservedExperience         bool      `json:"observedExperience"`
	CompletionRequired         bool      `json:"completionRequired"`
	FailedAttemptSupervised    bool      `json:"failedAttemptSupervised"`
	ReflectionTargetSupervised bool      `json:"reflectionTargetSupervised"`
	TrainingInvoked            bool      `json:"trainingInvoked"`
	WeightsChanged             bool      `json:"weightsChanged"`
	Authority                  Authority `json:"authority"`
	ReceiptSHA256              string    `json:"receiptSha256"`
}

type mirrorExperienceTrajectoryProjection struct {
	Schema              string                `json:"schema"`
	ID                  string                `json:"id"`
	DataClass           string                `json:"dataClass"`
	SourceReceiptSHA256 string                `json:"sourceReceiptSha256"`
	ObservedAt          time.Time             `json:"observedAt"`
	OutcomeSignal       string                `json:"outcomeSignal"`
	TargetKind          string                `json:"targetKind"`
	TrainingObjective   string                `json:"trainingObjective"`
	SupervisedRoles     []string              `json:"supervisedRoles"`
	Messages            []MirrorGroundMessage `json:"messages"`
}

// LoadMirrorExperienceTrajectoryDraft decodes one strict, unsealed trajectory.
// Sealing remains a separate step so callers cannot accidentally bless a
// caller-supplied digest or ignore trailing JSON.
func LoadMirrorExperienceTrajectoryDraft(reader io.Reader) (MirrorExperienceTrajectoryRecord, error) {
	if reader == nil {
		return MirrorExperienceTrajectoryRecord{}, errors.New("experience trajectory input is required")
	}
	decoder := json.NewDecoder(io.LimitReader(reader, MaxMirrorEscalationTextBytes*8))
	decoder.DisallowUnknownFields()
	var record MirrorExperienceTrajectoryRecord
	if err := decoder.Decode(&record); err != nil {
		return record, fmt.Errorf("decode experience trajectory draft: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return record, errors.New("experience trajectory draft must contain exactly one JSON object")
		}
		return record, fmt.Errorf("decode trailing experience trajectory data: %w", err)
	}
	if strings.TrimSpace(record.RecordSHA256) != "" {
		return record, errors.New("experience trajectory draft must not supply recordSha256")
	}
	return record, nil
}

func SealMirrorExperienceTrajectory(record *MirrorExperienceTrajectoryRecord) error {
	if record == nil {
		return errors.New("experience trajectory record is required")
	}
	record.RecordSHA256 = ""
	digest, err := digestJSON(*record, "experience trajectory record")
	if err != nil {
		return err
	}
	record.RecordSHA256 = digest
	return record.Validate()
}

func (record MirrorExperienceTrajectoryRecord) Validate() error {
	if record.Schema != MirrorExperienceTrajectorySchema {
		return fmt.Errorf("experience trajectory schema must be %q", MirrorExperienceTrajectorySchema)
	}
	if err := validateMirrorEpisodeID(record.ID); err != nil {
		return fmt.Errorf("experience trajectory id: %w", err)
	}
	if !validMirrorSHA(record.SourceReceiptSHA256) || record.ObservedAt.IsZero() {
		return errors.New("experience trajectory requires an observed timestamp and source receipt")
	}
	switch record.OutcomeSignal {
	case MirrorExperienceHelpful, MirrorExperienceCorrected, MirrorExperienceHarmful, MirrorExperienceInconclusive:
	default:
		return errors.New("experience trajectory outcome must be HELPFUL, CORRECTED, HARMFUL, or INCONCLUSIVE")
	}
	for name, value := range map[string]string{
		"prompt": record.Prompt, "attemptTrace": record.AttemptTrace,
		"observedOutcome": record.ObservedOutcome, "lesson": record.Lesson,
	} {
		if strings.TrimSpace(value) == "" || len(value) > MaxMirrorEscalationTextBytes {
			return fmt.Errorf("experience trajectory %s is required and limited to %d bytes", name, MaxMirrorEscalationTextBytes)
		}
	}
	if record.TargetKind != MirrorExperienceReflectionTarget || record.TrainingObjective != "assistant-response-modeling" {
		return errors.New("experience trajectory must train an outcome-conditioned reflection with assistant-response-modeling")
	}
	if record.CompletionRequired || record.FailedAttemptSupervised || !record.ReflectionTargetSupervised {
		return errors.New("experience trajectory must learn without requiring completion or supervising the failed attempt")
	}
	if !record.Authority.closed() {
		return errors.New("experience trajectory carries no execution, training, promotion, canon, or world-action authority")
	}
	copy := record
	copy.RecordSHA256 = ""
	digest, err := digestJSON(copy, "experience trajectory record")
	if err != nil {
		return err
	}
	if record.RecordSHA256 != digest {
		return errors.New("experience trajectory record digest mismatch")
	}
	return nil
}

func ProjectMirrorExperienceTrajectory(record MirrorExperienceTrajectoryRecord) ([]byte, MirrorExperienceTrajectoryProjectionReceipt, error) {
	if err := record.Validate(); err != nil {
		return nil, MirrorExperienceTrajectoryProjectionReceipt{}, err
	}
	trace := "Observed attempt:\n" + record.AttemptTrace + "\n\nObserved outcome:\n" + record.ObservedOutcome
	projection := mirrorExperienceTrajectoryProjection{
		Schema: MirrorExperienceTrajectoryProjectionSchema, ID: record.ID,
		DataClass: MirrorGroundObservedExecutionTrace, SourceReceiptSHA256: record.SourceReceiptSHA256,
		ObservedAt: record.ObservedAt, OutcomeSignal: record.OutcomeSignal,
		TargetKind: record.TargetKind, TrainingObjective: record.TrainingObjective,
		SupervisedRoles: []string{"assistant"},
		Messages: []MirrorGroundMessage{
			{Role: "user", Content: record.Prompt},
			{Role: "tool", Content: trace},
			{Role: "assistant", Content: record.Lesson},
		},
	}
	payload, err := json.Marshal(projection)
	if err != nil {
		return nil, MirrorExperienceTrajectoryProjectionReceipt{}, err
	}
	payload = append(payload, '\n')
	receipt := MirrorExperienceTrajectoryProjectionReceipt{
		Schema: MirrorExperienceTrajectoryReceiptSchema, State: "EXPERIENCE_REFLECTION_PROJECTED",
		RecordSHA256: record.RecordSHA256, SourceReceiptSHA256: record.SourceReceiptSHA256,
		ProjectionSHA256: candidateBytesSHA256(payload), OutcomeSignal: record.OutcomeSignal,
		TargetKind: record.TargetKind, TrainingObjective: record.TrainingObjective,
		ObservedExperience: true, CompletionRequired: false, FailedAttemptSupervised: false,
		ReflectionTargetSupervised: true, TrainingInvoked: false, WeightsChanged: false,
		Authority: Authority{},
	}
	receipt.ReceiptSHA256, err = digestJSON(receipt, "experience trajectory projection receipt")
	if err != nil {
		return nil, MirrorExperienceTrajectoryProjectionReceipt{}, err
	}
	return payload, receipt, nil
}
