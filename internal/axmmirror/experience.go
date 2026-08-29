package axmmirror

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const (
	MirrorExperienceEventSchema          = "axm.waldo.mirror-experience-event/v0.38"
	MirrorExperienceOutcomeRequestSchema = "axm.waldo.mirror-experience-outcome-request/v0.38"
	MirrorExperienceLearningSchema       = "axm.waldo.mirror-experience-learning/v0.38"
	MirrorHermesMemorySchema             = "axm.waldo.hermes-memory-capsule/v0.38"

	MirrorExperienceSensed    = "SENSED"
	MirrorExperienceReacted   = "REACTED"
	MirrorExperienceObserved  = "OUTCOME_OBSERVED"
	MirrorExperienceReflected = "REFLECTED"

	MirrorExperienceHelpful      = "HELPFUL"
	MirrorExperienceCorrected    = "CORRECTED"
	MirrorExperienceHarmful      = "HARMFUL"
	MirrorExperienceInconclusive = "INCONCLUSIVE"

	MaxMirrorExperienceContextBytes = 256 * 1024
)

// MirrorExperiencePromptContext is retrieved, visible episode context. It can
// influence a neural candidate but carries no execution or mutation authority.
type MirrorExperiencePromptContext struct {
	Capsule    string
	SHA256     string
	EpisodeIDs []string
}

func (context MirrorExperiencePromptContext) Validate() error {
	if strings.TrimSpace(context.Capsule) == "" || len(context.Capsule) > MaxMirrorExperienceContextBytes {
		return fmt.Errorf("capsule is required and limited to %d bytes", MaxMirrorExperienceContextBytes)
	}
	if context.SHA256 != digestMirrorText(context.Capsule) {
		return errors.New("sha256 does not bind the visible experience capsule")
	}
	if len(context.EpisodeIDs) == 0 || len(context.EpisodeIDs) > 32 {
		return errors.New("experience context must name 1..32 episodes")
	}
	seen := map[string]bool{}
	for _, id := range context.EpisodeIDs {
		if err := validateMirrorEpisodeID(id); err != nil {
			return err
		}
		if seen[id] {
			return fmt.Errorf("duplicate experience episode %q", id)
		}
		seen[id] = true
	}
	return nil
}

type MirrorExperienceSense struct {
	RequestSHA256   string `json:"requestSha256"`
	Prompt          string `json:"prompt"`
	PromptSHA256    string `json:"promptSha256"`
	GroundingSHA256 string `json:"groundingSha256"`
	IdentitySHA256  string `json:"identitySha256,omitempty"`
}

type MirrorExperienceReaction struct {
	Status            string `json:"status"`
	SourceMode        string `json:"sourceMode"`
	Response          string `json:"response,omitempty"`
	ResponseSHA256    string `json:"responseSha256,omitempty"`
	Model             string `json:"model,omitempty"`
	Backend           string `json:"backend,omitempty"`
	NeuralCandidate   bool   `json:"neuralCandidate"`
	NeuralErrorSHA256 string `json:"neuralErrorSha256,omitempty"`
}

type MirrorExperienceOutcome struct {
	Signal                  string `json:"signal"`
	Feedback                string `json:"feedback"`
	FeedbackSHA256          string `json:"feedbackSha256"`
	CorrectedResponse       string `json:"correctedResponse,omitempty"`
	CorrectedResponseSHA256 string `json:"correctedResponseSha256,omitempty"`
}

type MirrorExperienceReflection struct {
	Lesson               string `json:"lesson"`
	LessonSHA256         string `json:"lessonSha256"`
	UseInFutureContext   bool   `json:"useInFutureContext"`
	TrainingReady        bool   `json:"trainingReady"`
	TrainingText         string `json:"text,omitempty"`
	TargetResponseSHA256 string `json:"targetResponseSha256,omitempty"`
}

// MirrorExperienceEvent is an append-only, per-episode hash chain. Raw chat
// content remains visible in the private ledger; the chain makes later edits
// or missing stages detectable.
type MirrorExperienceEvent struct {
	Schema              string                      `json:"schema"`
	EpisodeID           string                      `json:"episodeId"`
	Sequence            int                         `json:"sequence"`
	PreviousEventSHA256 string                      `json:"previousEventSha256,omitempty"`
	Type                string                      `json:"type"`
	Sense               *MirrorExperienceSense      `json:"sense,omitempty"`
	Reaction            *MirrorExperienceReaction   `json:"reaction,omitempty"`
	Outcome             *MirrorExperienceOutcome    `json:"outcome,omitempty"`
	Reflection          *MirrorExperienceReflection `json:"reflection,omitempty"`
	Authority           Authority                   `json:"authority"`
	OccurredAt          time.Time                   `json:"occurredAt"`
	EventSHA256         string                      `json:"eventSha256"`
}

type MirrorExperienceOutcomeRequest struct {
	Schema            string `json:"schema"`
	EpisodeID         string `json:"episodeId"`
	Signal            string `json:"signal"`
	Feedback          string `json:"feedback"`
	CorrectedResponse string `json:"correctedResponse,omitempty"`
	Lesson            string `json:"lesson,omitempty"`
}

type MirrorExperienceLearningRecord struct {
	Schema                string    `json:"schema"`
	EpisodeID             string    `json:"episodeId"`
	OutcomeSignal         string    `json:"outcomeSignal"`
	TrainingReady         bool      `json:"trainingReady"`
	Prompt                string    `json:"prompt"`
	TargetResponse        string    `json:"targetResponse"`
	Feedback              string    `json:"feedback"`
	Lesson                string    `json:"lesson"`
	TrainingText          string    `json:"text"`
	ReflectionEventSHA256 string    `json:"reflectionEventSha256"`
	Authority             Authority `json:"authority"`
	CapturedAt            time.Time `json:"capturedAt"`
	LearningRecordSHA256  string    `json:"learningRecordSha256"`
}

// MirrorHermesMemoryRecord is a portable memory handoff for the separate AXM
// Hermes runtime lane. Writing it does not claim that Hermes is installed or
// that its runtime memory has already changed.
type MirrorHermesMemoryRecord struct {
	Schema             string    `json:"schema"`
	MemoryID           string    `json:"memoryId"`
	EpisodeID          string    `json:"episodeId"`
	MemoryKind         string    `json:"memoryKind"`
	OutcomeSignal      string    `json:"outcomeSignal"`
	Prompt             string    `json:"prompt"`
	Reaction           string    `json:"reaction,omitempty"`
	Feedback           string    `json:"feedback"`
	CorrectedResponse  string    `json:"correctedResponse,omitempty"`
	Lesson             string    `json:"lesson"`
	ContextText        string    `json:"contextText"`
	ReflectionSHA256   string    `json:"reflectionEventSha256"`
	Authority          Authority `json:"authority"`
	CapturedAt         time.Time `json:"capturedAt"`
	MemoryRecordSHA256 string    `json:"memoryRecordSha256"`
}

type MirrorExperienceClosure struct {
	OutcomeEvent     MirrorExperienceEvent
	ReflectionEvent  MirrorExperienceEvent
	LearningRecord   *MirrorExperienceLearningRecord
	TrajectoryRecord *MirrorExperienceTrajectoryRecord
	HermesMemory     MirrorHermesMemoryRecord
}

type MirrorExperienceLedger struct {
	Episodes map[string][]MirrorExperienceEvent
}

func BuildMirrorExperienceEpisode(receipt MirrorNeuralEscalationReceipt, prompt, episodeID string, observedAt time.Time) ([]MirrorExperienceEvent, error) {
	if receipt.Schema != MirrorNeuralEscalationReceiptSchema || !receipt.Authority.closed() {
		return nil, errors.New("experience capture requires a valid closed-authority Mirror receipt")
	}
	if digestMirrorText(prompt) != receipt.PromptSHA256 {
		return nil, errors.New("experience prompt does not match the Mirror receipt")
	}
	if observedAt.IsZero() {
		return nil, errors.New("experience capture time is required")
	}
	if episodeID == "" {
		episodeID = "experience-" + digestMirrorText(receipt.RequestSHA256 + "\n" + observedAt.UTC().Format(time.RFC3339Nano))[:20]
	}
	if err := validateMirrorEpisodeID(episodeID); err != nil {
		return nil, err
	}
	reaction := MirrorExperienceReaction{Status: receipt.Status, SourceMode: "HOLD"}
	switch receipt.Status {
	case MirrorStatusDeterministicResolved:
		reaction.SourceMode = "DETERMINISTIC"
		reaction.Response = receipt.DeterministicResponse
	case MirrorStatusNeuralCandidate:
		if receipt.NeuralCandidate == nil {
			return nil, errors.New("neural-candidate receipt is missing its candidate")
		}
		reaction.SourceMode = "NEURAL_CANDIDATE"
		reaction.Response = receipt.NeuralCandidate.Text
		reaction.Model = receipt.NeuralCandidate.Model
		reaction.Backend = receipt.NeuralCandidate.Backend
		reaction.NeuralCandidate = true
	case MirrorStatusNeuralOptInRequired:
	case MirrorStatusNeuralFailed:
		reaction.SourceMode = "FAILED"
		reaction.NeuralErrorSHA256 = receipt.NeuralErrorSHA256
	default:
		return nil, fmt.Errorf("unsupported Mirror reaction status %q", receipt.Status)
	}
	if reaction.Response != "" {
		reaction.ResponseSHA256 = digestMirrorText(reaction.Response)
	}
	sense := MirrorExperienceEvent{
		Schema:    MirrorExperienceEventSchema,
		EpisodeID: episodeID,
		Sequence:  1,
		Type:      MirrorExperienceSensed,
		Sense: &MirrorExperienceSense{
			RequestSHA256:   receipt.RequestSHA256,
			Prompt:          prompt,
			PromptSHA256:    receipt.PromptSHA256,
			GroundingSHA256: receipt.GroundingSHA256,
			IdentitySHA256:  receipt.IdentitySHA256,
		},
		Authority:  Authority{},
		OccurredAt: observedAt.UTC(),
	}
	if err := sense.seal(); err != nil {
		return nil, err
	}
	reacted := MirrorExperienceEvent{
		Schema:              MirrorExperienceEventSchema,
		EpisodeID:           episodeID,
		Sequence:            2,
		PreviousEventSHA256: sense.EventSHA256,
		Type:                MirrorExperienceReacted,
		Reaction:            &reaction,
		Authority:           Authority{},
		OccurredAt:          observedAt.UTC().Add(time.Nanosecond),
	}
	if err := reacted.seal(); err != nil {
		return nil, err
	}
	return []MirrorExperienceEvent{sense, reacted}, nil
}

func LoadMirrorExperienceOutcomeRequest(reader io.Reader) (MirrorExperienceOutcomeRequest, error) {
	var request MirrorExperienceOutcomeRequest
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, fmt.Errorf("decode Mirror experience outcome: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("trailing JSON")
		}
		return MirrorExperienceOutcomeRequest{}, fmt.Errorf("decode Mirror experience outcome: %w", err)
	}
	if err := request.Validate(); err != nil {
		return MirrorExperienceOutcomeRequest{}, err
	}
	return request, nil
}

func (request MirrorExperienceOutcomeRequest) Validate() error {
	if request.Schema != MirrorExperienceOutcomeRequestSchema {
		return fmt.Errorf("schema must be %q", MirrorExperienceOutcomeRequestSchema)
	}
	if err := validateMirrorEpisodeID(request.EpisodeID); err != nil {
		return err
	}
	if strings.TrimSpace(request.Feedback) == "" || len(request.Feedback) > MaxMirrorEscalationTextBytes {
		return fmt.Errorf("feedback is required and limited to %d bytes", MaxMirrorEscalationTextBytes)
	}
	if len(request.CorrectedResponse) > MaxMirrorEscalationTextBytes || len(request.Lesson) > MaxMirrorEscalationTextBytes {
		return fmt.Errorf("correctedResponse and lesson are limited to %d bytes", MaxMirrorEscalationTextBytes)
	}
	switch request.Signal {
	case MirrorExperienceHelpful, MirrorExperienceHarmful, MirrorExperienceInconclusive:
		if request.CorrectedResponse != "" {
			return fmt.Errorf("signal %s cannot include correctedResponse", request.Signal)
		}
	case MirrorExperienceCorrected:
		if strings.TrimSpace(request.CorrectedResponse) == "" {
			return errors.New("CORRECTED outcome requires correctedResponse")
		}
	default:
		return errors.New("signal must be HELPFUL, CORRECTED, HARMFUL, or INCONCLUSIVE")
	}
	return nil
}

func LoadMirrorExperienceLedger(reader io.Reader) (MirrorExperienceLedger, error) {
	ledger := MirrorExperienceLedger{Episodes: map[string][]MirrorExperienceEvent{}}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var event MirrorExperienceEvent
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&event); err != nil {
			return MirrorExperienceLedger{}, fmt.Errorf("decode experience ledger line %d: %w", lineNumber, err)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			return MirrorExperienceLedger{}, fmt.Errorf("decode experience ledger line %d: trailing JSON", lineNumber)
		}
		if err := event.Validate(); err != nil {
			return MirrorExperienceLedger{}, fmt.Errorf("validate experience ledger line %d: %w", lineNumber, err)
		}
		events := ledger.Episodes[event.EpisodeID]
		if event.Sequence != len(events)+1 {
			return MirrorExperienceLedger{}, fmt.Errorf("experience %s sequence gap: got %d, want %d", event.EpisodeID, event.Sequence, len(events)+1)
		}
		if len(events) == 0 {
			if event.PreviousEventSHA256 != "" {
				return MirrorExperienceLedger{}, fmt.Errorf("experience %s first event names a predecessor", event.EpisodeID)
			}
		} else if event.PreviousEventSHA256 != events[len(events)-1].EventSHA256 {
			return MirrorExperienceLedger{}, fmt.Errorf("experience %s hash chain is broken at sequence %d", event.EpisodeID, event.Sequence)
		}
		if event.Sequence == 4 {
			outcome := events[2].Outcome
			sense := events[0].Sense
			reaction := events[1].Reaction
			var target string
			switch outcome.Signal {
			case MirrorExperienceHelpful:
				target = reaction.Response
			case MirrorExperienceCorrected:
				target = outcome.CorrectedResponse
			}
			reflection := event.Reflection
			if target == "" {
				if reflection.TrainingReady || reflection.TrainingText != "" || reflection.TargetResponseSHA256 != "" {
					return MirrorExperienceLedger{}, fmt.Errorf("experience %s negative or inconclusive outcome became a positive training target", event.EpisodeID)
				}
			} else {
				expectedText := "User: " + sense.Prompt + "\n\nAssistant: " + target
				if !reflection.TrainingReady || reflection.TrainingText != expectedText || reflection.TargetResponseSHA256 != digestMirrorText(target) {
					return MirrorExperienceLedger{}, fmt.Errorf("experience %s training projection does not bind the observed target", event.EpisodeID)
				}
			}
		}
		ledger.Episodes[event.EpisodeID] = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return MirrorExperienceLedger{}, fmt.Errorf("read experience ledger: %w", err)
	}
	return ledger, nil
}

func CloseMirrorExperienceEpisode(ledger MirrorExperienceLedger, request MirrorExperienceOutcomeRequest, observedAt time.Time) (MirrorExperienceClosure, error) {
	if err := request.Validate(); err != nil {
		return MirrorExperienceClosure{}, err
	}
	if observedAt.IsZero() {
		return MirrorExperienceClosure{}, errors.New("outcome observation time is required")
	}
	events := ledger.Episodes[request.EpisodeID]
	if len(events) == 0 {
		return MirrorExperienceClosure{}, fmt.Errorf("experience episode %q was not found", request.EpisodeID)
	}
	if len(events) != 2 || events[0].Type != MirrorExperienceSensed || events[1].Type != MirrorExperienceReacted {
		return MirrorExperienceClosure{}, fmt.Errorf("experience episode %q is already closed or incomplete", request.EpisodeID)
	}
	sense := events[0].Sense
	reaction := events[1].Reaction
	if request.Signal == MirrorExperienceHelpful && strings.TrimSpace(reaction.Response) == "" {
		return MirrorExperienceClosure{}, errors.New("HELPFUL outcome requires an observed response")
	}
	outcome := MirrorExperienceOutcome{
		Signal:            request.Signal,
		Feedback:          request.Feedback,
		FeedbackSHA256:    digestMirrorText(request.Feedback),
		CorrectedResponse: request.CorrectedResponse,
	}
	if request.CorrectedResponse != "" {
		outcome.CorrectedResponseSHA256 = digestMirrorText(request.CorrectedResponse)
	}
	outcomeEvent := MirrorExperienceEvent{
		Schema:              MirrorExperienceEventSchema,
		EpisodeID:           request.EpisodeID,
		Sequence:            3,
		PreviousEventSHA256: events[1].EventSHA256,
		Type:                MirrorExperienceObserved,
		Outcome:             &outcome,
		Authority:           Authority{},
		OccurredAt:          observedAt.UTC(),
	}
	if err := outcomeEvent.seal(); err != nil {
		return MirrorExperienceClosure{}, err
	}
	lesson := strings.TrimSpace(request.Lesson)
	if lesson == "" {
		lesson = defaultMirrorExperienceLesson(request.Signal, request.Feedback)
	}
	reflection := MirrorExperienceReflection{
		Lesson:             lesson,
		LessonSHA256:       digestMirrorText(lesson),
		UseInFutureContext: true,
	}
	var target string
	switch request.Signal {
	case MirrorExperienceHelpful:
		target = reaction.Response
	case MirrorExperienceCorrected:
		target = request.CorrectedResponse
	}
	if target != "" {
		reflection.TrainingReady = true
		reflection.TrainingText = "User: " + sense.Prompt + "\n\nAssistant: " + target
		reflection.TargetResponseSHA256 = digestMirrorText(target)
	}
	reflectionEvent := MirrorExperienceEvent{
		Schema:              MirrorExperienceEventSchema,
		EpisodeID:           request.EpisodeID,
		Sequence:            4,
		PreviousEventSHA256: outcomeEvent.EventSHA256,
		Type:                MirrorExperienceReflected,
		Reflection:          &reflection,
		Authority:           Authority{},
		OccurredAt:          observedAt.UTC().Add(time.Nanosecond),
	}
	if err := reflectionEvent.seal(); err != nil {
		return MirrorExperienceClosure{}, err
	}
	closure := MirrorExperienceClosure{OutcomeEvent: outcomeEvent, ReflectionEvent: reflectionEvent}
	if reflection.TrainingReady {
		record := MirrorExperienceLearningRecord{
			Schema:                MirrorExperienceLearningSchema,
			EpisodeID:             request.EpisodeID,
			OutcomeSignal:         request.Signal,
			TrainingReady:         true,
			Prompt:                sense.Prompt,
			TargetResponse:        target,
			Feedback:              request.Feedback,
			Lesson:                lesson,
			TrainingText:          reflection.TrainingText,
			ReflectionEventSHA256: reflectionEvent.EventSHA256,
			Authority:             Authority{},
			CapturedAt:            observedAt.UTC(),
		}
		if err := record.seal(); err != nil {
			return MirrorExperienceClosure{}, err
		}
		closure.LearningRecord = &record
	} else {
		attemptTrace := strings.TrimSpace(reaction.Response)
		if attemptTrace == "" {
			attemptTrace = fmt.Sprintf("Mirror reaction status: %s\nSource mode: %s", reaction.Status, reaction.SourceMode)
			if reaction.NeuralErrorSHA256 != "" {
				attemptTrace += "\nNeural error receipt: " + reaction.NeuralErrorSHA256
			}
		}
		trajectory := MirrorExperienceTrajectoryRecord{
			Schema:                     MirrorExperienceTrajectorySchema,
			ID:                         request.EpisodeID,
			SourceReceiptSHA256:        reflectionEvent.EventSHA256,
			ObservedAt:                 observedAt.UTC(),
			OutcomeSignal:              request.Signal,
			Prompt:                     sense.Prompt,
			AttemptTrace:               attemptTrace,
			ObservedOutcome:            request.Feedback,
			Lesson:                     lesson,
			TargetKind:                 MirrorExperienceReflectionTarget,
			TrainingObjective:          "assistant-response-modeling",
			CompletionRequired:         false,
			FailedAttemptSupervised:    false,
			ReflectionTargetSupervised: true,
			Authority:                  Authority{},
		}
		if err := SealMirrorExperienceTrajectory(&trajectory); err != nil {
			return MirrorExperienceClosure{}, err
		}
		closure.TrajectoryRecord = &trajectory
	}
	memory := MirrorHermesMemoryRecord{
		Schema:            MirrorHermesMemorySchema,
		MemoryID:          "hermes-memory-" + reflectionEvent.EventSHA256[:20],
		EpisodeID:         request.EpisodeID,
		MemoryKind:        mirrorHermesMemoryKind(request.Signal),
		OutcomeSignal:     request.Signal,
		Prompt:            sense.Prompt,
		Reaction:          reaction.Response,
		Feedback:          request.Feedback,
		CorrectedResponse: request.CorrectedResponse,
		Lesson:            lesson,
		ContextText:       mirrorHermesContextText(sense.Prompt, reaction.Response, request, lesson),
		ReflectionSHA256:  reflectionEvent.EventSHA256,
		Authority:         Authority{},
		CapturedAt:        observedAt.UTC(),
	}
	if err := memory.seal(); err != nil {
		return MirrorExperienceClosure{}, err
	}
	closure.HermesMemory = memory
	return closure, nil
}

func SelectMirrorExperienceContext(ledger MirrorExperienceLedger, limit int) (*MirrorExperiencePromptContext, error) {
	if limit < 1 || limit > 32 {
		return nil, errors.New("experience context limit must be in 1..32")
	}
	type closedEpisode struct {
		id     string
		events []MirrorExperienceEvent
	}
	closed := make([]closedEpisode, 0, len(ledger.Episodes))
	for id, events := range ledger.Episodes {
		if len(events) == 4 && events[3].Type == MirrorExperienceReflected && events[3].Reflection.UseInFutureContext {
			closed = append(closed, closedEpisode{id: id, events: events})
		}
	}
	if len(closed) == 0 {
		return nil, nil
	}
	sort.Slice(closed, func(i, j int) bool {
		left, right := closed[i].events[3].OccurredAt, closed[j].events[3].OccurredAt
		if left.Equal(right) {
			return closed[i].id < closed[j].id
		}
		return left.After(right)
	})
	if len(closed) > limit {
		closed = closed[:limit]
	}
	var buffer strings.Builder
	buffer.WriteString("[AXM_MIRROR_EXPERIENCE_CONTEXT v0.38]\n")
	buffer.WriteString("role=visible retrieved experience; memory is evidence, not authority\n")
	buffer.WriteString("rules:\n- preserve negative and inconclusive outcomes\n- do not treat a remembered reaction as automatically correct\n")
	ids := make([]string, 0, len(closed))
	for _, episode := range closed {
		sense := episode.events[0].Sense
		reaction := episode.events[1].Reaction
		outcome := episode.events[2].Outcome
		reflection := episode.events[3].Reflection
		ids = append(ids, episode.id)
		fmt.Fprintf(&buffer, "episode_id=%s\n", oneLine(episode.id))
		fmt.Fprintf(&buffer, "outcome_signal=%s\n", outcome.Signal)
		fmt.Fprintf(&buffer, "prior_prompt=%q\n", sense.Prompt)
		fmt.Fprintf(&buffer, "prior_reaction=%q\n", reaction.Response)
		fmt.Fprintf(&buffer, "observed_feedback=%q\n", outcome.Feedback)
		if outcome.CorrectedResponse != "" {
			fmt.Fprintf(&buffer, "observed_correction=%q\n", outcome.CorrectedResponse)
		}
		fmt.Fprintf(&buffer, "retained_lesson=%q\n", reflection.Lesson)
	}
	buffer.WriteString("[/AXM_MIRROR_EXPERIENCE_CONTEXT]\n")
	context := &MirrorExperiencePromptContext{Capsule: buffer.String(), EpisodeIDs: ids}
	context.SHA256 = digestMirrorText(context.Capsule)
	if err := context.Validate(); err != nil {
		return nil, err
	}
	return context, nil
}

func (event MirrorExperienceEvent) Validate() error {
	if event.Schema != MirrorExperienceEventSchema {
		return fmt.Errorf("schema must be %q", MirrorExperienceEventSchema)
	}
	if err := validateMirrorEpisodeID(event.EpisodeID); err != nil {
		return err
	}
	if event.Sequence < 1 || event.Sequence > 4 {
		return errors.New("sequence must be in 1..4")
	}
	if !event.Authority.closed() || event.OccurredAt.IsZero() {
		return errors.New("experience event requires closed authority and a timestamp")
	}
	payloads := 0
	if event.Sense != nil {
		payloads++
	}
	if event.Reaction != nil {
		payloads++
	}
	if event.Outcome != nil {
		payloads++
	}
	if event.Reflection != nil {
		payloads++
	}
	if payloads != 1 {
		return errors.New("experience event must contain exactly one typed payload")
	}
	switch event.Type {
	case MirrorExperienceSensed:
		if event.Sequence != 1 || event.Sense == nil || event.PreviousEventSHA256 != "" {
			return errors.New("SENSED must be sequence 1 without a predecessor")
		}
		if strings.TrimSpace(event.Sense.Prompt) == "" || len(event.Sense.Prompt) > MaxMirrorEscalationTextBytes || event.Sense.PromptSHA256 != digestMirrorText(event.Sense.Prompt) || !validMirrorSHA(event.Sense.RequestSHA256) || !validMirrorSHA(event.Sense.GroundingSHA256) || (event.Sense.IdentitySHA256 != "" && !validMirrorSHA(event.Sense.IdentitySHA256)) {
			return errors.New("SENSED payload identity mismatch")
		}
	case MirrorExperienceReacted:
		if event.Sequence != 2 || event.Reaction == nil || !validMirrorSHA(event.PreviousEventSHA256) {
			return errors.New("REACTED must be sequence 2 with a predecessor")
		}
		if event.Reaction.Response != "" && event.Reaction.ResponseSHA256 != digestMirrorText(event.Reaction.Response) {
			return errors.New("REACTED response digest mismatch")
		}
		if len(event.Reaction.Response) > MaxMirrorEscalationTextBytes {
			return fmt.Errorf("REACTED response is limited to %d bytes", MaxMirrorEscalationTextBytes)
		}
		switch event.Reaction.Status {
		case MirrorStatusDeterministicResolved:
			if event.Reaction.SourceMode != "DETERMINISTIC" || strings.TrimSpace(event.Reaction.Response) == "" || event.Reaction.NeuralCandidate {
				return errors.New("DETERMINISTIC_RESOLVED reaction payload drifted")
			}
		case MirrorStatusNeuralCandidate:
			if event.Reaction.SourceMode != "NEURAL_CANDIDATE" || strings.TrimSpace(event.Reaction.Response) == "" || !event.Reaction.NeuralCandidate || strings.TrimSpace(event.Reaction.Model) == "" || strings.TrimSpace(event.Reaction.Backend) == "" {
				return errors.New("NEURAL_CANDIDATE reaction payload drifted")
			}
		case MirrorStatusNeuralOptInRequired:
			if event.Reaction.SourceMode != "HOLD" || event.Reaction.Response != "" || event.Reaction.NeuralCandidate {
				return errors.New("HOLD reaction payload drifted")
			}
		case MirrorStatusNeuralFailed:
			if event.Reaction.SourceMode != "FAILED" || event.Reaction.Response != "" || event.Reaction.NeuralCandidate || !validMirrorSHA(event.Reaction.NeuralErrorSHA256) {
				return errors.New("FAILED reaction payload drifted")
			}
		default:
			return fmt.Errorf("unsupported REACTED status %q", event.Reaction.Status)
		}
	case MirrorExperienceObserved:
		if event.Sequence != 3 || event.Outcome == nil || !validMirrorSHA(event.PreviousEventSHA256) || event.Outcome.FeedbackSHA256 != digestMirrorText(event.Outcome.Feedback) {
			return errors.New("OUTCOME_OBSERVED payload mismatch")
		}
		if strings.TrimSpace(event.Outcome.Feedback) == "" || len(event.Outcome.Feedback) > MaxMirrorEscalationTextBytes || len(event.Outcome.CorrectedResponse) > MaxMirrorEscalationTextBytes {
			return errors.New("OUTCOME_OBSERVED feedback or correction bounds drifted")
		}
		if event.Outcome.CorrectedResponse != "" && event.Outcome.CorrectedResponseSHA256 != digestMirrorText(event.Outcome.CorrectedResponse) {
			return errors.New("OUTCOME_OBSERVED correction digest mismatch")
		}
		switch event.Outcome.Signal {
		case MirrorExperienceCorrected:
			if strings.TrimSpace(event.Outcome.CorrectedResponse) == "" {
				return errors.New("CORRECTED outcome requires a correction")
			}
		case MirrorExperienceHelpful, MirrorExperienceHarmful, MirrorExperienceInconclusive:
			if event.Outcome.CorrectedResponse != "" {
				return fmt.Errorf("%s outcome cannot carry a correction", event.Outcome.Signal)
			}
		default:
			return fmt.Errorf("unsupported outcome signal %q", event.Outcome.Signal)
		}
	case MirrorExperienceReflected:
		if event.Sequence != 4 || event.Reflection == nil || !validMirrorSHA(event.PreviousEventSHA256) || event.Reflection.LessonSHA256 != digestMirrorText(event.Reflection.Lesson) {
			return errors.New("REFLECTED payload mismatch")
		}
		if event.Reflection.TrainingReady != (event.Reflection.TrainingText != "") {
			return errors.New("REFLECTED training readiness does not match its training projection")
		}
		if strings.TrimSpace(event.Reflection.Lesson) == "" || len(event.Reflection.Lesson) > MaxMirrorEscalationTextBytes || !event.Reflection.UseInFutureContext {
			return errors.New("REFLECTED lesson must remain visible future context")
		}
		if event.Reflection.TrainingReady && !validMirrorSHA(event.Reflection.TargetResponseSHA256) {
			return errors.New("REFLECTED training target digest is invalid")
		}
	default:
		return fmt.Errorf("unsupported experience event type %q", event.Type)
	}
	digest, err := event.digest()
	if err != nil {
		return err
	}
	if event.EventSHA256 != digest {
		return errors.New("experience event digest mismatch")
	}
	return nil
}

func (event *MirrorExperienceEvent) seal() error {
	digest, err := event.digest()
	if err != nil {
		return err
	}
	event.EventSHA256 = digest
	return event.Validate()
}

func (event MirrorExperienceEvent) digest() (string, error) {
	copy := event
	copy.EventSHA256 = ""
	payload, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	return digestMirrorText(string(payload)), nil
}

func (event MirrorExperienceEvent) JSONLine() ([]byte, error) {
	if err := event.Validate(); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}

func (record *MirrorExperienceLearningRecord) seal() error {
	copy := *record
	copy.LearningRecordSHA256 = ""
	payload, err := json.Marshal(copy)
	if err != nil {
		return err
	}
	record.LearningRecordSHA256 = digestMirrorText(string(payload))
	return record.Validate()
}

func (record MirrorExperienceLearningRecord) JSONLine() ([]byte, error) {
	if err := record.Validate(); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}

func (record MirrorExperienceLearningRecord) Validate() error {
	if record.Schema != MirrorExperienceLearningSchema || !record.Authority.closed() || record.CapturedAt.IsZero() {
		return errors.New("experience learning record identity, authority, or timestamp is invalid")
	}
	if err := validateMirrorEpisodeID(record.EpisodeID); err != nil {
		return err
	}
	if record.OutcomeSignal != MirrorExperienceHelpful && record.OutcomeSignal != MirrorExperienceCorrected {
		return errors.New("experience learning record requires a helpful or corrected outcome")
	}
	if !record.TrainingReady || strings.TrimSpace(record.Prompt) == "" || strings.TrimSpace(record.TargetResponse) == "" || strings.TrimSpace(record.Feedback) == "" || strings.TrimSpace(record.Lesson) == "" {
		return errors.New("experience learning record is not training-ready")
	}
	if len(record.Prompt) > MaxMirrorEscalationTextBytes || len(record.TargetResponse) > MaxMirrorEscalationTextBytes || len(record.Feedback) > MaxMirrorEscalationTextBytes || len(record.Lesson) > MaxMirrorEscalationTextBytes {
		return fmt.Errorf("experience learning fields are limited to %d bytes", MaxMirrorEscalationTextBytes)
	}
	if record.TrainingText != "User: "+record.Prompt+"\n\nAssistant: "+record.TargetResponse || !validMirrorSHA(record.ReflectionEventSHA256) {
		return errors.New("experience learning projection does not bind its prompt, target, or reflection")
	}
	copy := record
	copy.LearningRecordSHA256 = ""
	payload, err := json.Marshal(copy)
	if err != nil {
		return err
	}
	if record.LearningRecordSHA256 != digestMirrorText(string(payload)) {
		return errors.New("experience learning record digest mismatch")
	}
	return nil
}

func (record *MirrorHermesMemoryRecord) seal() error {
	copy := *record
	copy.MemoryRecordSHA256 = ""
	payload, err := json.Marshal(copy)
	if err != nil {
		return err
	}
	record.MemoryRecordSHA256 = digestMirrorText(string(payload))
	return record.Validate()
}

func (record MirrorHermesMemoryRecord) JSONLine() ([]byte, error) {
	if err := record.Validate(); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}

func (record MirrorHermesMemoryRecord) Validate() error {
	if record.Schema != MirrorHermesMemorySchema || !record.Authority.closed() || record.CapturedAt.IsZero() {
		return errors.New("Hermes memory record identity, authority, or timestamp is invalid")
	}
	if err := validateMirrorEpisodeID(record.EpisodeID); err != nil {
		return err
	}
	if !validMirrorSHA(record.ReflectionSHA256) || record.MemoryID != "hermes-memory-"+record.ReflectionSHA256[:20] {
		return errors.New("Hermes memory identity does not bind its reflection")
	}
	if record.MemoryKind != mirrorHermesMemoryKind(record.OutcomeSignal) {
		return errors.New("Hermes memory kind does not match the observed outcome")
	}
	request := MirrorExperienceOutcomeRequest{
		Schema:            MirrorExperienceOutcomeRequestSchema,
		EpisodeID:         record.EpisodeID,
		Signal:            record.OutcomeSignal,
		Feedback:          record.Feedback,
		CorrectedResponse: record.CorrectedResponse,
		Lesson:            record.Lesson,
	}
	if err := request.Validate(); err != nil {
		return fmt.Errorf("Hermes memory outcome: %w", err)
	}
	if strings.TrimSpace(record.Prompt) == "" || strings.TrimSpace(record.Lesson) == "" || len(record.Prompt) > MaxMirrorEscalationTextBytes || len(record.Reaction) > MaxMirrorEscalationTextBytes || len(record.Lesson) > MaxMirrorEscalationTextBytes {
		return errors.New("Hermes memory visible context is missing or over its bound")
	}
	if record.ContextText != mirrorHermesContextText(record.Prompt, record.Reaction, request, record.Lesson) {
		return errors.New("Hermes memory context text does not bind the visible episode")
	}
	copy := record
	copy.MemoryRecordSHA256 = ""
	payload, err := json.Marshal(copy)
	if err != nil {
		return err
	}
	if record.MemoryRecordSHA256 != digestMirrorText(string(payload)) {
		return errors.New("Hermes memory record digest mismatch")
	}
	return nil
}

func validateMirrorEpisodeID(id string) error {
	if strings.TrimSpace(id) == "" || len(id) > 128 || strings.ContainsAny(id, "\r\n\t") {
		return errors.New("episodeId is required, single-line, and limited to 128 bytes")
	}
	return nil
}

func validMirrorSHA(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func defaultMirrorExperienceLesson(signal, feedback string) string {
	switch signal {
	case MirrorExperienceHelpful:
		return "Retain the successful response pattern when the grounding is comparable. Observed feedback: " + feedback
	case MirrorExperienceCorrected:
		return "Prefer the observed correction over the prior reaction when the grounding is comparable. Observed feedback: " + feedback
	case MirrorExperienceHarmful:
		return "Do not repeat the prior reaction without repair when the grounding is comparable. Observed feedback: " + feedback
	default:
		return "Preserve uncertainty and seek more evidence before repeating the prior reaction. Observed feedback: " + feedback
	}
}

func mirrorHermesMemoryKind(signal string) string {
	switch signal {
	case MirrorExperienceHelpful:
		return "SUCCESSFUL_PATTERN"
	case MirrorExperienceCorrected:
		return "CORRECTION"
	case MirrorExperienceHarmful:
		return "NEGATIVE_EXPERIENCE"
	default:
		return "UNRESOLVED_EXPERIENCE"
	}
}

func mirrorHermesContextText(prompt, reaction string, request MirrorExperienceOutcomeRequest, lesson string) string {
	var buffer strings.Builder
	fmt.Fprintf(&buffer, "Prompt: %s\nReaction: %s\nOutcome: %s\nFeedback: %s\n", prompt, reaction, request.Signal, request.Feedback)
	if request.CorrectedResponse != "" {
		fmt.Fprintf(&buffer, "Correction: %s\n", request.CorrectedResponse)
	}
	fmt.Fprintf(&buffer, "Lesson: %s", lesson)
	return buffer.String()
}
