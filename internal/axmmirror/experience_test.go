package axmmirror

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMirrorExperienceClosesCorrectedEpisodeIntoContextTrainingAndHermesMemory(t *testing.T) {
	now := time.Date(2026, 8, 24, 15, 0, 0, 0, time.UTC)
	request := experienceTestRequest(GroundingStable, MirrorEscalationNone)
	request.DeterministicResponse = "The first reaction"
	receipt, err := RunMirrorNeuralEscalation(context.Background(), request, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	events, err := BuildMirrorExperienceEpisode(receipt, request.Prompt, "episode-corrected", now)
	if err != nil {
		t.Fatal(err)
	}
	var ledgerBytes bytes.Buffer
	for _, event := range events {
		line, lineErr := event.JSONLine()
		if lineErr != nil {
			t.Fatal(lineErr)
		}
		ledgerBytes.Write(line)
	}
	ledger, err := LoadMirrorExperienceLedger(bytes.NewReader(ledgerBytes.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	closure, err := CloseMirrorExperienceEpisode(ledger, MirrorExperienceOutcomeRequest{
		Schema:            MirrorExperienceOutcomeRequestSchema,
		EpisodeID:         "episode-corrected",
		Signal:            MirrorExperienceCorrected,
		Feedback:          "The response missed the visible outcome.",
		CorrectedResponse: "Use the observed outcome instead.",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if closure.LearningRecord == nil || closure.LearningRecord.TargetResponse != "Use the observed outcome instead." || !strings.Contains(closure.LearningRecord.TrainingText, "Assistant: Use the observed outcome instead.") {
		t.Fatalf("learning record = %+v", closure.LearningRecord)
	}
	if closure.HermesMemory.MemoryKind != "CORRECTION" || !strings.Contains(closure.HermesMemory.ContextText, "Correction: Use the observed outcome instead.") {
		t.Fatalf("Hermes memory = %+v", closure.HermesMemory)
	}
	for _, event := range []MirrorExperienceEvent{closure.OutcomeEvent, closure.ReflectionEvent} {
		line, lineErr := event.JSONLine()
		if lineErr != nil {
			t.Fatal(lineErr)
		}
		ledgerBytes.Write(line)
	}
	closedLedger, err := LoadMirrorExperienceLedger(bytes.NewReader(ledgerBytes.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	experienceContext, err := SelectMirrorExperienceContext(closedLedger, 8)
	if err != nil {
		t.Fatal(err)
	}
	if experienceContext == nil || !strings.Contains(experienceContext.Capsule, "episode-corrected") || !strings.Contains(experienceContext.Capsule, "observed_correction") {
		t.Fatalf("experience context = %+v", experienceContext)
	}
	if err := experienceContext.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := closure.LearningRecord.JSONLine(); err != nil {
		t.Fatal(err)
	}
	if _, err := closure.HermesMemory.JSONLine(); err != nil {
		t.Fatal(err)
	}
}

func TestMirrorExperienceHarmfulOutcomeTeachesContextWithoutPositiveTrainingTarget(t *testing.T) {
	now := time.Date(2026, 8, 24, 15, 5, 0, 0, time.UTC)
	request := experienceTestRequest(GroundingStable, MirrorEscalationNone)
	request.DeterministicResponse = "Unsafe shortcut"
	receipt, err := RunMirrorNeuralEscalation(context.Background(), request, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	events, err := BuildMirrorExperienceEpisode(receipt, request.Prompt, "episode-negative", now)
	if err != nil {
		t.Fatal(err)
	}
	ledger := MirrorExperienceLedger{Episodes: map[string][]MirrorExperienceEvent{"episode-negative": events}}
	closure, err := CloseMirrorExperienceEpisode(ledger, MirrorExperienceOutcomeRequest{
		Schema:    MirrorExperienceOutcomeRequestSchema,
		EpisodeID: "episode-negative",
		Signal:    MirrorExperienceHarmful,
		Feedback:  "The shortcut damaged the result.",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if closure.LearningRecord != nil || closure.ReflectionEvent.Reflection.TrainingReady {
		t.Fatalf("negative outcome produced positive training target: %+v", closure)
	}
	if closure.TrajectoryRecord == nil || closure.TrajectoryRecord.CompletionRequired || closure.TrajectoryRecord.FailedAttemptSupervised || !closure.TrajectoryRecord.ReflectionTargetSupervised || closure.TrajectoryRecord.Lesson != closure.ReflectionEvent.Reflection.Lesson {
		t.Fatalf("negative outcome reflection trajectory = %+v", closure.TrajectoryRecord)
	}
	projection, projectionReceipt, err := ProjectMirrorExperienceTrajectory(*closure.TrajectoryRecord)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(projection, []byte(`"role":"tool"`)) || !bytes.Contains(projection, []byte(`"role":"assistant"`)) || projectionReceipt.FailedAttemptSupervised || !projectionReceipt.ReflectionTargetSupervised {
		t.Fatalf("negative outcome projection = %s; receipt = %+v", projection, projectionReceipt)
	}
	if closure.HermesMemory.MemoryKind != "NEGATIVE_EXPERIENCE" || !strings.Contains(closure.HermesMemory.Lesson, "Do not repeat") {
		t.Fatalf("Hermes negative memory = %+v", closure.HermesMemory)
	}
	closure.ReflectionEvent.Reflection.TrainingReady = true
	closure.ReflectionEvent.Reflection.TrainingText = "User: manufactured\n\nAssistant: unsafe"
	closure.ReflectionEvent.Reflection.TargetResponseSHA256 = digestMirrorText("unsafe")
	if err := closure.ReflectionEvent.seal(); err != nil {
		t.Fatal(err)
	}
	var tampered bytes.Buffer
	for _, event := range append(events, closure.OutcomeEvent, closure.ReflectionEvent) {
		line, lineErr := event.JSONLine()
		if lineErr != nil {
			t.Fatal(lineErr)
		}
		tampered.Write(line)
	}
	if _, err := LoadMirrorExperienceLedger(bytes.NewReader(tampered.Bytes())); err == nil || !strings.Contains(err.Error(), "positive training target") {
		t.Fatalf("negative-training error = %v", err)
	}
}

func TestMirrorExperienceContextReachesNeuralReactionVisibly(t *testing.T) {
	contextCapsule := "[AXM_MIRROR_EXPERIENCE_CONTEXT v0.38]\nretained_lesson=repair the prior answer\n[/AXM_MIRROR_EXPERIENCE_CONTEXT]\n"
	experienceContext := &MirrorExperiencePromptContext{
		Capsule:    contextCapsule,
		SHA256:     digestMirrorText(contextCapsule),
		EpisodeIDs: []string{"episode-prior"},
	}
	request := experienceTestRequest(GroundingUncertain, MirrorEscalationNovelty)
	escalator := &experiencePromptEscalator{}
	receipt, err := RunMirrorNeuralEscalationWithExperience(context.Background(), request, true, escalator, experienceContext)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != MirrorStatusNeuralCandidate || !receipt.ExperienceContextApplied || receipt.ExperienceContextSHA256 != experienceContext.SHA256 || len(receipt.ExperienceEpisodeIDs) != 1 {
		t.Fatalf("receipt = %+v", receipt)
	}
	if !strings.Contains(escalator.prompt, "repair the prior answer") || !strings.Contains(escalator.prompt, request.Prompt) {
		t.Fatalf("neural prompt = %q", escalator.prompt)
	}
}

func TestMirrorExperienceLedgerRejectsTamperAndUnknownFields(t *testing.T) {
	now := time.Date(2026, 8, 24, 15, 10, 0, 0, time.UTC)
	request := experienceTestRequest(GroundingStable, MirrorEscalationNone)
	request.DeterministicResponse = "response"
	receipt, err := RunMirrorNeuralEscalation(context.Background(), request, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	events, err := BuildMirrorExperienceEpisode(receipt, request.Prompt, "episode-tamper", now)
	if err != nil {
		t.Fatal(err)
	}
	first, _ := events[0].JSONLine()
	secondPayload, _ := json.Marshal(events[1])
	secondPayload = bytes.Replace(secondPayload, []byte(`"previousEventSha256":"`), []byte(`"unknown":true,"previousEventSha256":"`), 1)
	ledger := append(first, append(secondPayload, '\n')...)
	if _, err := LoadMirrorExperienceLedger(bytes.NewReader(ledger)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field error = %v", err)
	}
	events[1].PreviousEventSHA256 = strings.Repeat("0", 64)
	second, _ := json.Marshal(events[1])
	ledger = append(first, append(second, '\n')...)
	if _, err := LoadMirrorExperienceLedger(bytes.NewReader(ledger)); err == nil {
		t.Fatal("broken experience chain was accepted")
	}
}

type experiencePromptEscalator struct {
	prompt string
}

func (escalator *experiencePromptEscalator) Escalate(_ context.Context, prompt string) (MirrorNeuralCandidate, error) {
	escalator.prompt = prompt
	return MirrorNeuralCandidate{Text: "revised candidate", Model: "fixture", Backend: "pytorch"}, nil
}

func experienceTestRequest(state, reason string) MirrorNeuralEscalationRequest {
	return MirrorNeuralEscalationRequest{
		Schema:           MirrorNeuralEscalationRequestSchema,
		Prompt:           "What should change next?",
		EscalationReason: reason,
		Grounding: LiveNeuralGrounding{
			Schema:      LiveNeuralGroundingSchema,
			State:       state,
			Consequence: ConsequenceLow,
			Authority:   Authority{},
		},
	}
}
