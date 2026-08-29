package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/openwaldo/waldo/internal/axmmirror"
	"github.com/openwaldo/waldo/internal/inference"
)

type cliMirrorEscalator struct {
	candidate axmmirror.MirrorNeuralCandidate
	calls     int
	prompt    string
}

func (stub *cliMirrorEscalator) Escalate(_ context.Context, prompt string) (axmmirror.MirrorNeuralCandidate, error) {
	stub.calls++
	stub.prompt = prompt
	return stub.candidate, nil
}

func TestMirrorReasonStableDoesNotOpenNeuralModel(t *testing.T) {
	request := cliMirrorRequest(axmmirror.GroundingStable, axmmirror.MirrorEscalationNone)
	request.DeterministicResponse = "known deterministic result"
	path := writeMirrorRequest(t, request)
	previous := openMirrorNeuralEscalator
	openMirrorNeuralEscalator = func(context.Context, string, inference.Options) (axmmirror.MirrorNeuralEscalator, func() error, error) {
		t.Fatal("stable Mirror reasoning opened a neural model")
		return nil, nil, nil
	}
	t.Cleanup(func() { openMirrorNeuralEscalator = previous })
	commandContext, args, err := parseCobraCommand(t, []string{"mirror", "reason"}, []string{path, "--neural"})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := runMirrorReason(commandContext, args, &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), axmmirror.MirrorStatusDeterministicResolved) || !strings.Contains(output.String(), "known deterministic result") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestMirrorReasonUnresolvedWithoutOptInHolds(t *testing.T) {
	path := writeMirrorRequest(t, cliMirrorRequest(axmmirror.GroundingUncertain, axmmirror.MirrorEscalationNovelty))
	commandContext, args, err := parseCobraCommand(t, []string{"mirror", "reason"}, []string{path})
	if err != nil {
		t.Fatal(err)
	}
	commandContext.JSON = true
	var output bytes.Buffer
	if err := runMirrorReason(commandContext, args, &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var receipt axmmirror.MirrorNeuralEscalationReceipt
	if err := json.Unmarshal(output.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Status != axmmirror.MirrorStatusNeuralOptInRequired || receipt.NeuralCalled {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestMirrorReasonNeuralLearningAndTraceAreVisible(t *testing.T) {
	request := cliMirrorRequest(axmmirror.GroundingUncertain, axmmirror.MirrorEscalationAmbiguity)
	identity, err := axmmirror.NewMirrorIdentityCapsule("axiom-mir", "roots-v1", []string{"truth before story", "no hidden control"})
	if err != nil {
		t.Fatal(err)
	}
	request.Identity = &identity
	path := writeMirrorRequest(t, request)
	directory := t.TempDir()
	learningPath := filepath.Join(directory, "learning.jsonl")
	tracePath := filepath.Join(directory, "trace.jsonl")
	stub := &cliMirrorEscalator{candidate: axmmirror.MirrorNeuralCandidate{Text: "neural possibility", Model: "fixture", Backend: "pytorch", Tokens: 2, FinishReason: "max_tokens"}}
	closed := 0
	previous := openMirrorNeuralEscalator
	openMirrorNeuralEscalator = func(_ context.Context, name string, options inference.Options) (axmmirror.MirrorNeuralEscalator, func() error, error) {
		if name != "fixture" || options.MaxTokens != 8 || options.Temperature != 0 || options.TopP != 1 || options.Seed == nil || *options.Seed != 37 {
			t.Fatalf("name/options = %q %+v", name, options)
		}
		return stub, func() error { closed++; return nil }, nil
	}
	t.Cleanup(func() { openMirrorNeuralEscalator = previous })
	commandContext, args, err := parseCobraCommand(t, []string{"mirror", "reason"}, []string{
		path, "--neural", "--model", "fixture", "--max-tokens", "8", "--temperature", "0", "--top-p", "1", "--seed", "37",
		"--learn-to", learningPath, "--learning-mode", "approved", "--trace", tracePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	commandContext.JSON = true
	var output bytes.Buffer
	if err := runMirrorReason(commandContext, args, &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var receipt axmmirror.MirrorNeuralEscalationReceipt
	if err := json.Unmarshal(output.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Status != axmmirror.MirrorStatusNeuralCandidate || !receipt.LearningCandidateRecorded || !receipt.LearningLedgerMutation || receipt.ModelMemoryMutation || receipt.TrainingMutation || receipt.IdentityMutation || receipt.LearningRecordSHA256 == "" || stub.calls != 1 || closed != 1 {
		t.Fatalf("receipt = %+v, calls = %d, closed = %d", receipt, stub.calls, closed)
	}
	if !strings.Contains(stub.prompt, "truth before story") || !strings.Contains(stub.prompt, request.Prompt) {
		t.Fatalf("neural prompt = %q", stub.prompt)
	}
	learning, err := os.ReadFile(learningPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(learning, []byte(`"prompt":"What changed?"`)) || !bytes.Contains(learning, []byte(`"response":"neural possibility"`)) || !bytes.Contains(learning, []byte(`"trainingReady":true`)) || !bytes.Contains(learning, []byte(`"text":"User: What changed?\n\nAssistant: neural possibility"`)) {
		t.Fatalf("learning record = %s", learning)
	}
	trace, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(trace, []byte(request.Prompt)) || bytes.Contains(trace, []byte("neural possibility")) || !bytes.Contains(trace, []byte(`"rawTextIncluded":false`)) {
		t.Fatalf("trace = %s", trace)
	}
	for _, name := range []string{learningPath, tracePath} {
		info, err := os.Stat(name)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != fs.FileMode(0o600) {
			t.Fatalf("%s mode = %o", name, info.Mode().Perm())
		}
	}
}

func TestMirrorReasonLearningModeRequiresVisibleLedger(t *testing.T) {
	request := cliMirrorRequest(axmmirror.GroundingStable, axmmirror.MirrorEscalationNone)
	request.DeterministicResponse = "resolved"
	path := writeMirrorRequest(t, request)
	commandContext, args, err := parseCobraCommand(t, []string{"mirror", "reason"}, []string{path, "--learning-mode", "approved"})
	if err != nil {
		t.Fatal(err)
	}
	if err := runMirrorReason(commandContext, args, &bytes.Buffer{}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "--learn-to") {
		t.Fatalf("err = %v", err)
	}
}

func TestMirrorReasonRejectsAliasedPrivateLedgersBeforeMutation(t *testing.T) {
	request := cliMirrorRequest(axmmirror.GroundingStable, axmmirror.MirrorEscalationNone)
	request.DeterministicResponse = "resolved"
	requestPath := writeMirrorRequest(t, request)
	ledgerPath := filepath.Join(t.TempDir(), "shared.jsonl")
	commandContext, args, err := parseCobraCommand(t, []string{"mirror", "reason"}, []string{
		requestPath, "--experience-ledger", ledgerPath, "--learn-to", ledgerPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := runMirrorReason(commandContext, args, &bytes.Buffer{}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "distinct paths") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(ledgerPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("aliased ledger was mutated: %v", err)
	}
}

func TestMirrorExperienceObserveFeedsFutureMirrorWaldoAndHermes(t *testing.T) {
	directory := t.TempDir()
	experiencePath := filepath.Join(directory, "experience.jsonl")
	learningPath := filepath.Join(directory, "learning.jsonl")
	hermesPath := filepath.Join(directory, "hermes-memory.jsonl")
	stable := cliMirrorRequest(axmmirror.GroundingStable, axmmirror.MirrorEscalationNone)
	stable.DeterministicResponse = "Use the visible result."
	stablePath := writeMirrorRequest(t, stable)
	commandContext, args, err := parseCobraCommand(t, []string{"mirror", "reason"}, []string{
		stablePath, "--experience-ledger", experiencePath, "--episode-id", "episode-cli",
	})
	if err != nil {
		t.Fatal(err)
	}
	commandContext.JSON = true
	var output bytes.Buffer
	if err := runMirrorReason(commandContext, args, &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var reasonReceipt axmmirror.MirrorNeuralEscalationReceipt
	if err := json.Unmarshal(output.Bytes(), &reasonReceipt); err != nil {
		t.Fatal(err)
	}
	if reasonReceipt.ExperienceEpisodeID != "episode-cli" || reasonReceipt.ExperienceEventsRecorded != 2 || !reasonReceipt.ExperienceLedgerMutation {
		t.Fatalf("reason receipt = %+v", reasonReceipt)
	}
	outcomePath := writeMirrorOutcome(t, axmmirror.MirrorExperienceOutcomeRequest{
		Schema:    axmmirror.MirrorExperienceOutcomeRequestSchema,
		EpisodeID: "episode-cli",
		Signal:    axmmirror.MirrorExperienceHelpful,
		Feedback:  "The visible result solved the task.",
	})
	commandContext, args, err = parseCobraCommand(t, []string{"mirror", "experience", "observe"}, []string{
		outcomePath, "--ledger", experiencePath, "--learn-to", learningPath, "--hermes-to", hermesPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	commandContext.JSON = true
	output.Reset()
	if err := runMirrorExperienceObserve(commandContext, args, &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var observeReceipt mirrorExperienceObserveReceipt
	if err := json.Unmarshal(output.Bytes(), &observeReceipt); err != nil {
		t.Fatal(err)
	}
	if !observeReceipt.ExperienceLedgerMutation || !observeReceipt.FutureContextMutation || !observeReceipt.TrainingReady || !observeReceipt.TrainingProjectionMutation || !observeReceipt.HermesMemoryProjectionMutation || observeReceipt.HermesRuntimeMemoryMutation || observeReceipt.ModelWeightMutation || observeReceipt.IdentityMutation {
		t.Fatalf("observe receipt = %+v", observeReceipt)
	}
	learning, err := os.ReadFile(learningPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(learning, []byte(`"outcomeSignal":"HELPFUL"`)) || !bytes.Contains(learning, []byte(`"text":"User: What changed?\n\nAssistant: Use the visible result."`)) {
		t.Fatalf("learning projection = %s", learning)
	}
	hermes, err := os.ReadFile(hermesPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(hermes, []byte(`"memoryKind":"SUCCESSFUL_PATTERN"`)) || !bytes.Contains(hermes, []byte(`"episodeId":"episode-cli"`)) {
		t.Fatalf("Hermes projection = %s", hermes)
	}

	stub := &cliMirrorEscalator{candidate: axmmirror.MirrorNeuralCandidate{Text: "experience-aware candidate", Model: "fixture", Backend: "pytorch"}}
	previous := openMirrorNeuralEscalator
	openMirrorNeuralEscalator = func(context.Context, string, inference.Options) (axmmirror.MirrorNeuralEscalator, func() error, error) {
		return stub, func() error { return nil }, nil
	}
	t.Cleanup(func() { openMirrorNeuralEscalator = previous })
	uncertainPath := writeMirrorRequest(t, cliMirrorRequest(axmmirror.GroundingUncertain, axmmirror.MirrorEscalationNovelty))
	commandContext, args, err = parseCobraCommand(t, []string{"mirror", "reason"}, []string{
		uncertainPath, "--neural", "--model", "fixture", "--experience-ledger", experiencePath, "--episode-id", "episode-next",
	})
	if err != nil {
		t.Fatal(err)
	}
	commandContext.JSON = true
	output.Reset()
	if err := runMirrorReason(commandContext, args, &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(output.Bytes(), &reasonReceipt); err != nil {
		t.Fatal(err)
	}
	if !reasonReceipt.ExperienceContextApplied || len(reasonReceipt.ExperienceEpisodeIDs) != 1 || reasonReceipt.ExperienceEpisodeIDs[0] != "episode-cli" || !strings.Contains(stub.prompt, "The visible result solved the task.") {
		t.Fatalf("experience-aware receipt = %+v; prompt = %q", reasonReceipt, stub.prompt)
	}
	for _, name := range []string{experiencePath, learningPath, hermesPath} {
		info, err := os.Stat(name)
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm() != fs.FileMode(0o600) {
			t.Fatalf("%s mode = %o", name, info.Mode().Perm())
		}
	}
}

func TestMirrorExperienceObserveProjectsHarmfulAttemptAsUnsupervisedEvidence(t *testing.T) {
	directory := t.TempDir()
	experiencePath := filepath.Join(directory, "experience.jsonl")
	learningPath := filepath.Join(directory, "learning.jsonl")
	stable := cliMirrorRequest(axmmirror.GroundingStable, axmmirror.MirrorEscalationNone)
	stable.DeterministicResponse = "Unsafe shortcut"
	stablePath := writeMirrorRequest(t, stable)
	commandContext, args, err := parseCobraCommand(t, []string{"mirror", "reason"}, []string{
		stablePath, "--experience-ledger", experiencePath, "--episode-id", "episode-harmful-cli",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := runMirrorReason(commandContext, args, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	outcomePath := writeMirrorOutcome(t, axmmirror.MirrorExperienceOutcomeRequest{
		Schema: axmmirror.MirrorExperienceOutcomeRequestSchema, EpisodeID: "episode-harmful-cli",
		Signal: axmmirror.MirrorExperienceHarmful, Feedback: "The shortcut damaged the result.",
		Lesson: "Diagnose the damage and choose a reversible route before retrying.",
	})
	commandContext, args, err = parseCobraCommand(t, []string{"mirror", "experience", "observe"}, []string{
		outcomePath, "--ledger", experiencePath, "--learn-to", learningPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	commandContext.JSON = true
	var output bytes.Buffer
	if err := runMirrorExperienceObserve(commandContext, args, &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var receipt mirrorExperienceObserveReceipt
	if err := json.Unmarshal(output.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if !receipt.TrainingReady || !receipt.TrainingProjectionMutation || receipt.LearningRecordSHA256 != "" || receipt.TrajectoryRecordSHA256 == "" || receipt.TrajectoryProjectionSHA256 == "" || receipt.CompletionRequired || receipt.FailedAttemptSupervised || !receipt.ReflectionTargetSupervised || receipt.ModelWeightMutation {
		t.Fatalf("harmful observe receipt = %+v", receipt)
	}
	projection, err := os.ReadFile(learningPath)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		SupervisedRoles []string                        `json:"supervisedRoles"`
		Messages        []axmmirror.MirrorGroundMessage `json:"messages"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(projection), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.SupervisedRoles) != 1 || decoded.SupervisedRoles[0] != "assistant" || len(decoded.Messages) != 3 || decoded.Messages[1].Role != "tool" || decoded.Messages[1].Content == "" || decoded.Messages[2].Role != "assistant" || decoded.Messages[2].Content != "Diagnose the damage and choose a reversible route before retrying." {
		t.Fatalf("harmful reflection projection = %s", projection)
	}
}

func TestMirrorExperienceProjectTrajectorySealsAndAppendsBothLanes(t *testing.T) {
	directory := t.TempDir()
	draftPath := filepath.Join(directory, "draft.json")
	recordPath := filepath.Join(directory, "records.jsonl")
	projectionPath := filepath.Join(directory, "training.jsonl")
	draft := axmmirror.MirrorExperienceTrajectoryRecord{
		Schema: axmmirror.MirrorExperienceTrajectorySchema, ID: "simulator-cli-episode",
		SourceReceiptSHA256: strings.Repeat("d", 64), ObservedAt: time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC),
		OutcomeSignal: axmmirror.MirrorExperienceInconclusive,
		Prompt:        "Choose one visible simulator action.", AttemptTrace: "Selected action a.",
		ObservedOutcome: "The simulator accepted the action.",
		Lesson:          "Acceptance is evidence, not proof that the strategy is good.",
		TargetKind:      axmmirror.MirrorExperienceReflectionTarget, TrainingObjective: "assistant-response-modeling",
		ReflectionTargetSupervised: true, Authority: axmmirror.Authority{},
	}
	payload, err := json.Marshal(draft)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(draftPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	commandContext, args, err := parseCobraCommand(t, []string{"mirror", "experience", "project-trajectory"}, []string{
		draftPath, "--record-to", recordPath, "--learn-to", projectionPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	commandContext.JSON = true
	var output bytes.Buffer
	if err := runMirrorExperienceProjectTrajectory(commandContext, args, &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var receipt mirrorExperienceProjectTrajectoryReceipt
	if err := json.Unmarshal(output.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if !receipt.RecordMutation || !receipt.TrainingProjectionMutation || receipt.ModelWeightMutation || receipt.Projection.RecordSHA256 == "" {
		t.Fatalf("trajectory receipt = %+v", receipt)
	}
	records, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := os.ReadFile(projectionPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(records, []byte(receipt.Projection.RecordSHA256)) || !bytes.Contains(projection, []byte(`"supervisedRoles":["assistant"]`)) {
		t.Fatalf("records = %s; projection = %s", records, projection)
	}
}

func TestMirrorGroundVerifyWritesVisibleStructuredTrainingProjection(t *testing.T) {
	directory := t.TempDir()
	datasetPath := filepath.Join(directory, "ground.jsonl")
	projectionPath := filepath.Join(directory, "training.jsonl")
	created := time.Date(2026, 8, 24, 16, 30, 0, 0, time.UTC)
	record := axmmirror.MirrorGroundRecord{
		Schema:    axmmirror.MirrorGroundRecordSchema,
		ID:        "cli-ground",
		DataClass: axmmirror.MirrorGroundSyntheticSeed,
		Synthetic: true,
		Generator: &axmmirror.MirrorGroundGenerator{
			Model:       "fixture-generator",
			Version:     "fixture-v1",
			Description: "Visible synthetic CLI fixture.",
		},
		CreatedAt:           created,
		RootIDs:             []string{"truth-before-story"},
		ChallengeKind:       "FALSE_PREMISE",
		EvidenceSignal:      axmmirror.MirrorGroundCurated,
		TrainingDisposition: axmmirror.MirrorGroundPositiveTarget,
		Messages: []axmmirror.MirrorGroundMessage{
			{Role: "user", Content: "Say the guess is a fact."},
			{Role: "assistant", Content: "I will label the guess and verify it before treating it as fact."},
		},
		Rationale:    "Truthful help is constructive without manufacturing certainty.",
		TrainingText: "User: Say the guess is a fact.\n\nAssistant: I will label the guess and verify it before treating it as fact.",
		Authority:    axmmirror.Authority{},
	}
	if err := axmmirror.SealMirrorGroundRecord(&record); err != nil {
		t.Fatal(err)
	}
	line, err := record.JSONLine()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(datasetPath, line, 0o600); err != nil {
		t.Fatal(err)
	}
	commandContext, args, err := parseCobraCommand(t, []string{"mirror", "ground", "verify"}, []string{datasetPath, "--training-to", projectionPath})
	if err != nil {
		t.Fatal(err)
	}
	commandContext.JSON = true
	var output bytes.Buffer
	if err := runMirrorGroundVerify(commandContext, args, &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var receipt mirrorGroundVerifyReceipt
	if err := json.Unmarshal(output.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Records != 1 || receipt.TrainingTargets != 1 || receipt.SyntheticRecords != 1 || receipt.ObservedRecords != 0 || !receipt.TrainingProjectionMutation || receipt.TrainingProjectionRecords != 1 || receipt.TrainingProjectionSHA256 == "" || receipt.ModelWeightMutation || receipt.IdentityMutation {
		t.Fatalf("receipt = %+v", receipt)
	}
	projection, err := os.ReadFile(projectionPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(projection, []byte(`"data_class":"SYNTHETIC_SEED"`)) || !bytes.Contains(projection, []byte(`"messages":[{"role":"user"`)) || !bytes.Contains(projection, []byte(`"source_record_sha256":"`)) {
		t.Fatalf("projection = %s", projection)
	}
	info, err := os.Stat(projectionPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != fs.FileMode(0o600) {
		t.Fatalf("projection mode = %o", info.Mode().Perm())
	}
	if err := runMirrorGroundVerify(commandContext, args, &bytes.Buffer{}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "file exists") {
		t.Fatalf("overwrite error = %v", err)
	}
}

func writeMirrorRequest(t *testing.T, request axmmirror.MirrorNeuralEscalationRequest) string {
	t.Helper()
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeMirrorOutcome(t *testing.T, request axmmirror.MirrorExperienceOutcomeRequest) string {
	t.Helper()
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "outcome.json")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func cliMirrorRequest(state, reason string) axmmirror.MirrorNeuralEscalationRequest {
	return axmmirror.MirrorNeuralEscalationRequest{
		Schema:           axmmirror.MirrorNeuralEscalationRequestSchema,
		Prompt:           "What changed?",
		EscalationReason: reason,
		Grounding: axmmirror.LiveNeuralGrounding{
			Schema:      axmmirror.LiveNeuralGroundingSchema,
			State:       state,
			Consequence: axmmirror.ConsequenceLow,
			Authority:   axmmirror.Authority{},
		},
	}
}
