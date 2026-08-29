package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/openwaldo/waldo/internal/axmmirror"
	"github.com/openwaldo/waldo/internal/inference"
	"github.com/openwaldo/waldo/internal/model"
)

const mirrorLearningModeCandidate = "candidate"
const mirrorLearningModeApproved = "approved"

var mirrorReasonInput io.Reader = os.Stdin
var mirrorExperienceOutcomeInput io.Reader = os.Stdin

var openMirrorNeuralEscalator = func(ctx context.Context, name string, options inference.Options) (axmmirror.MirrorNeuralEscalator, func() error, error) {
	root, err := configuredModelRoot()
	if err != nil {
		return nil, nil, err
	}
	inspection, err := model.Inspect(root, name)
	if err != nil {
		return nil, nil, err
	}
	if len(inspection.Model.Runs) == 0 && inspection.Origin == nil {
		return nil, nil, fmt.Errorf("model %q is untrained", name)
	}
	// Reverse Mirror escalation intentionally opens the existing raw local
	// session. The Mirror layer owns candidate status and authority here; routing
	// the request back through the v0.35 response wrapper would create a loop.
	opened, err := inference.Open(ctx, inspection)
	if err != nil {
		return nil, nil, err
	}
	escalator := &localMirrorNeuralEscalator{
		session:     opened.Session,
		description: opened.Description,
		interaction: inspection.Model.Interaction,
		options:     options,
	}
	return escalator, opened.Session.Close, nil
}

type localMirrorNeuralEscalator struct {
	session     inference.Session
	description inference.Description
	interaction model.Interaction
	options     inference.Options
}

func (escalator *localMirrorNeuralEscalator) Escalate(ctx context.Context, prompt string) (axmmirror.MirrorNeuralCandidate, error) {
	options := escalator.options
	options.Stop = escalator.interaction.Stops()
	renderedPrompt := escalator.interaction.Prompt("", prompt)
	result, err := escalator.session.Generate(ctx, renderedPrompt, options, nil)
	if err != nil {
		return axmmirror.MirrorNeuralCandidate{}, err
	}
	return axmmirror.MirrorNeuralCandidate{
		Text:         escalator.interaction.TrimResponse(result.Text),
		Model:        escalator.description.Model,
		Backend:      escalator.description.Backend,
		SourceType:   escalator.description.SourceType,
		SourceID:     escalator.description.SourceID,
		RunID:        escalator.description.RunID,
		Tokens:       result.Tokens,
		FinishReason: result.FinishReason,
		DurationMS:   result.DurationMS,
	}, nil
}

func runMirrorReason(commandContext Context, args []string, stdout, _ io.Writer) error {
	request, err := loadMirrorReasonRequest(args[0])
	if err != nil {
		return err
	}
	options := inference.Options{
		MaxTokens:   intOption(commandContext, "max-tokens"),
		Temperature: float64Option(commandContext, "temperature"),
		TopP:        float64Option(commandContext, "top-p"),
	}
	if optionChanged(commandContext, "seed") {
		seed := uint64Option(commandContext, "seed")
		options.Seed = &seed
	}
	if err := options.Validate(); err != nil {
		return err
	}
	timeoutSeconds := intOption(commandContext, "timeout-seconds")
	if timeoutSeconds < 1 || timeoutSeconds > 900 {
		return errors.New("--timeout-seconds must be in 1..900")
	}
	neuralOptIn := boolOption(commandContext, "neural")
	modelName := strings.TrimSpace(stringOption(commandContext, "model"))
	learnPath := strings.TrimSpace(stringOption(commandContext, "learn-to"))
	learningMode := strings.ToLower(strings.TrimSpace(stringOption(commandContext, "learning-mode")))
	if learningMode == "" {
		learningMode = mirrorLearningModeCandidate
	}
	if learnPath == "" && optionChanged(commandContext, "learning-mode") {
		return errors.New("--learning-mode requires --learn-to")
	}
	var learningDisposition string
	if learnPath != "" {
		learningDisposition, err = mirrorLearningDisposition(learningMode)
		if err != nil {
			return err
		}
	}
	tracePath := strings.TrimSpace(stringOption(commandContext, "trace"))
	experiencePath := strings.TrimSpace(stringOption(commandContext, "experience-ledger"))
	episodeID := strings.TrimSpace(stringOption(commandContext, "episode-id"))
	experienceLimit := intOption(commandContext, "experience-limit")
	if experiencePath == "" && (optionChanged(commandContext, "episode-id") || optionChanged(commandContext, "experience-limit")) {
		return errors.New("--episode-id and --experience-limit require --experience-ledger")
	}
	if experienceLimit < 1 || experienceLimit > 32 {
		return errors.New("--experience-limit must be in 1..32")
	}
	if err := requireDistinctMirrorPaths(experiencePath, learnPath, tracePath); err != nil {
		return err
	}
	var experienceContext *axmmirror.MirrorExperiencePromptContext
	if experiencePath != "" {
		ledger, ledgerErr := loadMirrorExperienceLedgerFile(experiencePath, true)
		if ledgerErr != nil {
			return ledgerErr
		}
		experienceContext, ledgerErr = axmmirror.SelectMirrorExperienceContext(ledger, experienceLimit)
		if ledgerErr != nil {
			return fmt.Errorf("select visible Mirror experience context: %w", ledgerErr)
		}
		if episodeID != "" && len(ledger.Episodes[episodeID]) != 0 {
			return fmt.Errorf("experience episode %q already exists; choose a new --episode-id", episodeID)
		}
	}
	var escalator axmmirror.MirrorNeuralEscalator
	closeEscalator := func() error { return nil }
	if request.NeedsNeuralEscalation() && neuralOptIn {
		if modelName == "" {
			return errors.New("--model is required when unresolved Mirror reasoning opts into --neural")
		}
		escalator, closeEscalator, err = openMirrorNeuralEscalator(commandContext.Execution, modelName, options)
		if err != nil {
			return fmt.Errorf("open local WALDO model %q for Mirror escalation: %w", modelName, err)
		}
	}
	ctx, cancel := context.WithTimeout(commandContext.Execution, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	receipt, runErr := axmmirror.RunMirrorNeuralEscalationWithExperience(ctx, request, neuralOptIn, escalator, experienceContext)
	closeErr := closeEscalator()
	if experiencePath != "" && receipt.Schema != "" {
		events, experienceErr := axmmirror.BuildMirrorExperienceEpisode(receipt, request.Prompt, episodeID, time.Now().UTC())
		if experienceErr != nil {
			return experienceErr
		}
		if experienceErr := appendMirrorExperienceEvents(experiencePath, events); experienceErr != nil {
			return fmt.Errorf("append visible Mirror experience episode: %w", experienceErr)
		}
		receipt.ExperienceEpisodeID = events[0].EpisodeID
		receipt.ExperienceEventsRecorded = len(events)
		receipt.ExperienceLedgerMutation = true
	}

	if learnPath != "" && runErr == nil {
		record, recordErr := axmmirror.BuildMirrorChatLearningRecord(receipt, request.Prompt, learningDisposition)
		if recordErr != nil {
			return recordErr
		}
		line, recordErr := record.JSONLine()
		if recordErr != nil {
			return recordErr
		}
		if recordErr := appendPrivateLine(learnPath, line); recordErr != nil {
			return fmt.Errorf("append visible Mirror chat learning candidate: %w", recordErr)
		}
		receipt.LearningCandidateRecorded = true
		receipt.LearningRecordSHA256 = record.LearningRecordSHA256
		receipt.LearningLedgerMutation = true
	}

	if tracePath != "" && receipt.Schema != "" {
		line, traceErr := mirrorTraceLine(receipt)
		if traceErr != nil {
			return traceErr
		}
		if traceErr := appendPrivateLine(tracePath, line); traceErr != nil {
			return fmt.Errorf("append Mirror escalation trace: %w", traceErr)
		}
	}
	if receipt.Schema != "" {
		if err := writeMirrorReceipt(commandContext, stdout, receipt); err != nil {
			return err
		}
	}
	return errors.Join(runErr, closeErr)
}

func loadMirrorReasonRequest(path string) (axmmirror.MirrorNeuralEscalationRequest, error) {
	if path == "-" {
		return axmmirror.LoadMirrorNeuralEscalationRequest(mirrorReasonInput)
	}
	file, err := os.Open(path)
	if err != nil {
		return axmmirror.MirrorNeuralEscalationRequest{}, fmt.Errorf("open Mirror reasoning request %s: %w", path, err)
	}
	defer file.Close()
	request, err := axmmirror.LoadMirrorNeuralEscalationRequest(file)
	if err != nil {
		return request, fmt.Errorf("load Mirror reasoning request %s: %w", path, err)
	}
	return request, nil
}

func mirrorLearningDisposition(mode string) (string, error) {
	switch mode {
	case mirrorLearningModeCandidate:
		return axmmirror.MirrorLearningCaptureCandidate, nil
	case mirrorLearningModeApproved:
		return axmmirror.MirrorLearningApproveTraining, nil
	default:
		return "", errors.New("--learning-mode must be candidate or approved")
	}
}

func writeMirrorReceipt(commandContext Context, output io.Writer, receipt axmmirror.MirrorNeuralEscalationReceipt) error {
	if commandContext.JSON {
		return writeJSON(output, receipt)
	}
	fmt.Fprintf(output, "Mirror: %s\n", receipt.Status)
	fmt.Fprintln(output, "Authority: NONE")
	switch receipt.Status {
	case axmmirror.MirrorStatusDeterministicResolved:
		fmt.Fprintln(output, "Source: deterministic Mirror")
		fmt.Fprintln(output)
		fmt.Fprintln(output, receipt.DeterministicResponse)
	case axmmirror.MirrorStatusNeuralOptInRequired:
		fmt.Fprintln(output, "Neural escalation was not opted in; unresolved reasoning remains on HOLD.")
	case axmmirror.MirrorStatusNeuralCandidate:
		fmt.Fprintf(output, "Source: local WALDO neural candidate (%s via %s)\n", receipt.NeuralCandidate.Model, receipt.NeuralCandidate.Backend)
		fmt.Fprintln(output, "Candidate only: no permission, execution, identity mutation, training, promotion, or CANON.")
		fmt.Fprintln(output)
		fmt.Fprintln(output, strings.ToValidUTF8(receipt.NeuralCandidate.Text, "�"))
	case axmmirror.MirrorStatusNeuralFailed:
		fmt.Fprintln(output, "Local neural escalation failed closed; unresolved reasoning remains on HOLD.")
	}
	if receipt.LearningCandidateRecorded {
		fmt.Fprintf(output, "Learning record: %s\n", receipt.LearningRecordSHA256)
	}
	if receipt.ExperienceContextApplied {
		fmt.Fprintf(output, "Experience context: %s (%d episodes)\n", receipt.ExperienceContextSHA256, len(receipt.ExperienceEpisodeIDs))
	}
	if receipt.ExperienceLedgerMutation {
		fmt.Fprintf(output, "Experience episode: %s (%d events)\n", receipt.ExperienceEpisodeID, receipt.ExperienceEventsRecorded)
	}
	return nil
}

type mirrorEscalationTrace struct {
	Schema                    string    `json:"schema"`
	Status                    string    `json:"status"`
	DeterministicPrimary      bool      `json:"deterministicPrimary"`
	NeuralOptIn               bool      `json:"neuralOptIn"`
	NeuralCalled              bool      `json:"neuralCalled"`
	EscalationReason          string    `json:"escalationReason"`
	Consequence               string    `json:"consequence"`
	RequestSHA256             string    `json:"requestSha256"`
	PromptSHA256              string    `json:"promptSha256"`
	NeuralPromptSHA256        string    `json:"neuralPromptSha256,omitempty"`
	GroundingSHA256           string    `json:"groundingSha256"`
	IdentitySHA256            string    `json:"identitySha256,omitempty"`
	NeuralCandidateSHA256     string    `json:"neuralCandidateSha256,omitempty"`
	NeuralErrorSHA256         string    `json:"neuralErrorSha256,omitempty"`
	LearningCandidateRecorded bool      `json:"learningCandidateRecorded"`
	LearningRecordSHA256      string    `json:"learningRecordSha256,omitempty"`
	LearningLedgerMutation    bool      `json:"learningLedgerMutation"`
	ExperienceContextApplied  bool      `json:"experienceContextApplied"`
	ExperienceContextSHA256   string    `json:"experienceContextSha256,omitempty"`
	ExperienceEpisodeIDs      []string  `json:"experienceEpisodeIds,omitempty"`
	ExperienceEpisodeID       string    `json:"experienceEpisodeId,omitempty"`
	ExperienceEventsRecorded  int       `json:"experienceEventsRecorded"`
	ExperienceLedgerMutation  bool      `json:"experienceLedgerMutation"`
	ModelMemoryMutation       bool      `json:"modelMemoryMutation"`
	TrainingMutation          bool      `json:"trainingMutation"`
	IdentityMutation          bool      `json:"identityMutation"`
	RawTextIncluded           bool      `json:"rawTextIncluded"`
	Authority                 string    `json:"authority"`
	Timestamp                 time.Time `json:"timestamp"`
}

func mirrorTraceLine(receipt axmmirror.MirrorNeuralEscalationReceipt) ([]byte, error) {
	trace := mirrorEscalationTrace{
		Schema:                    "axm.waldo.mirror-neural-escalation-trace/v0.38",
		Status:                    receipt.Status,
		DeterministicPrimary:      receipt.DeterministicPrimary,
		NeuralOptIn:               receipt.NeuralOptIn,
		NeuralCalled:              receipt.NeuralCalled,
		EscalationReason:          receipt.EscalationReason,
		Consequence:               receipt.Consequence,
		RequestSHA256:             receipt.RequestSHA256,
		PromptSHA256:              receipt.PromptSHA256,
		NeuralPromptSHA256:        receipt.NeuralPromptSHA256,
		GroundingSHA256:           receipt.GroundingSHA256,
		IdentitySHA256:            receipt.IdentitySHA256,
		NeuralCandidateSHA256:     receipt.NeuralCandidateSHA256,
		NeuralErrorSHA256:         receipt.NeuralErrorSHA256,
		LearningCandidateRecorded: receipt.LearningCandidateRecorded,
		LearningRecordSHA256:      receipt.LearningRecordSHA256,
		LearningLedgerMutation:    receipt.LearningLedgerMutation,
		ExperienceContextApplied:  receipt.ExperienceContextApplied,
		ExperienceContextSHA256:   receipt.ExperienceContextSHA256,
		ExperienceEpisodeIDs:      append([]string(nil), receipt.ExperienceEpisodeIDs...),
		ExperienceEpisodeID:       receipt.ExperienceEpisodeID,
		ExperienceEventsRecorded:  receipt.ExperienceEventsRecorded,
		ExperienceLedgerMutation:  receipt.ExperienceLedgerMutation,
		ModelMemoryMutation:       receipt.ModelMemoryMutation,
		TrainingMutation:          receipt.TrainingMutation,
		IdentityMutation:          receipt.IdentityMutation,
		RawTextIncluded:           false,
		Authority:                 "NONE",
		Timestamp:                 receipt.Timestamp,
	}
	payload, err := json.Marshal(trace)
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}

type mirrorExperienceObserveReceipt struct {
	Schema                         string              `json:"schema"`
	EpisodeID                      string              `json:"episodeId"`
	OutcomeSignal                  string              `json:"outcomeSignal"`
	OutcomeEventSHA256             string              `json:"outcomeEventSha256"`
	ReflectionEventSHA256          string              `json:"reflectionEventSha256"`
	ExperienceLedgerMutation       bool                `json:"experienceLedgerMutation"`
	FutureContextMutation          bool                `json:"futureContextMutation"`
	TrainingReady                  bool                `json:"trainingReady"`
	TrainingProjectionMutation     bool                `json:"trainingProjectionMutation"`
	LearningRecordSHA256           string              `json:"learningRecordSha256,omitempty"`
	TrajectoryRecordSHA256         string              `json:"trajectoryRecordSha256,omitempty"`
	TrajectoryProjectionSHA256     string              `json:"trajectoryProjectionSha256,omitempty"`
	CompletionRequired             bool                `json:"completionRequired"`
	FailedAttemptSupervised        bool                `json:"failedAttemptSupervised"`
	ReflectionTargetSupervised     bool                `json:"reflectionTargetSupervised"`
	HermesMemoryProjectionMutation bool                `json:"hermesMemoryProjectionMutation"`
	HermesMemoryRecordSHA256       string              `json:"hermesMemoryRecordSha256,omitempty"`
	HermesRuntimeMemoryMutation    bool                `json:"hermesRuntimeMemoryMutation"`
	ModelWeightMutation            bool                `json:"modelWeightMutation"`
	IdentityMutation               bool                `json:"identityMutation"`
	Authority                      axmmirror.Authority `json:"authority"`
	Timestamp                      time.Time           `json:"timestamp"`
}

func runMirrorExperienceObserve(commandContext Context, args []string, stdout, _ io.Writer) error {
	request, err := loadMirrorExperienceOutcomeRequest(args[0])
	if err != nil {
		return err
	}
	ledgerPath := strings.TrimSpace(stringOption(commandContext, "ledger"))
	if ledgerPath == "" {
		return errors.New("--ledger is required")
	}
	ledger, err := loadMirrorExperienceLedgerFile(ledgerPath, false)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	closure, err := axmmirror.CloseMirrorExperienceEpisode(ledger, request, now)
	if err != nil {
		return err
	}
	if err := appendMirrorExperienceEvents(ledgerPath, []axmmirror.MirrorExperienceEvent{closure.OutcomeEvent, closure.ReflectionEvent}); err != nil {
		return fmt.Errorf("append visible Mirror experience outcome: %w", err)
	}
	receipt := mirrorExperienceObserveReceipt{
		Schema:                   "axm.waldo.mirror-experience-observe-receipt/v0.38",
		EpisodeID:                request.EpisodeID,
		OutcomeSignal:            request.Signal,
		OutcomeEventSHA256:       closure.OutcomeEvent.EventSHA256,
		ReflectionEventSHA256:    closure.ReflectionEvent.EventSHA256,
		ExperienceLedgerMutation: true,
		FutureContextMutation:    true,
		TrainingReady:            closure.LearningRecord != nil || closure.TrajectoryRecord != nil,
		Authority:                axmmirror.Authority{},
		Timestamp:                now,
	}
	learnPath := strings.TrimSpace(stringOption(commandContext, "learn-to"))
	hermesPath := strings.TrimSpace(stringOption(commandContext, "hermes-to"))
	if err := requireDistinctMirrorPaths(ledgerPath, learnPath, hermesPath); err != nil {
		return err
	}
	if learnPath != "" && closure.LearningRecord != nil {
		line, lineErr := closure.LearningRecord.JSONLine()
		if lineErr != nil {
			return lineErr
		}
		if lineErr := appendPrivateLine(learnPath, line); lineErr != nil {
			return fmt.Errorf("append experience-derived WALDO learning projection: %w", lineErr)
		}
		receipt.TrainingProjectionMutation = true
		receipt.LearningRecordSHA256 = closure.LearningRecord.LearningRecordSHA256
	} else if learnPath != "" && closure.TrajectoryRecord != nil {
		line, projectionReceipt, lineErr := axmmirror.ProjectMirrorExperienceTrajectory(*closure.TrajectoryRecord)
		if lineErr != nil {
			return lineErr
		}
		if lineErr := appendPrivateLine(learnPath, line); lineErr != nil {
			return fmt.Errorf("append experience-derived WALDO reflection projection: %w", lineErr)
		}
		receipt.TrainingProjectionMutation = true
		receipt.TrajectoryRecordSHA256 = projectionReceipt.RecordSHA256
		receipt.TrajectoryProjectionSHA256 = projectionReceipt.ProjectionSHA256
		receipt.ReflectionTargetSupervised = projectionReceipt.ReflectionTargetSupervised
	}
	if hermesPath != "" {
		line, lineErr := closure.HermesMemory.JSONLine()
		if lineErr != nil {
			return lineErr
		}
		if lineErr := appendPrivateLine(hermesPath, line); lineErr != nil {
			return fmt.Errorf("append Hermes memory projection: %w", lineErr)
		}
		receipt.HermesMemoryProjectionMutation = true
		receipt.HermesMemoryRecordSHA256 = closure.HermesMemory.MemoryRecordSHA256
	}
	if commandContext.JSON {
		return writeJSON(stdout, receipt)
	}
	fmt.Fprintf(stdout, "Experience: %s\n", receipt.EpisodeID)
	fmt.Fprintf(stdout, "Outcome: %s\n", receipt.OutcomeSignal)
	fmt.Fprintf(stdout, "Reflection: %s\n", receipt.ReflectionEventSHA256)
	fmt.Fprintln(stdout, "Future Mirror context: UPDATED")
	if receipt.TrainingProjectionMutation {
		if receipt.LearningRecordSHA256 != "" {
			fmt.Fprintf(stdout, "WALDO response-target projection: %s\n", receipt.LearningRecordSHA256)
		} else {
			fmt.Fprintf(stdout, "WALDO outcome-reflection projection: %s\n", receipt.TrajectoryRecordSHA256)
		}
	} else if receipt.TrainingReady {
		fmt.Fprintln(stdout, "WALDO learning projection: READY (no --learn-to path supplied)")
	} else {
		fmt.Fprintln(stdout, "WALDO learning projection: NOT TRAINING-READY")
	}
	if receipt.HermesMemoryProjectionMutation {
		fmt.Fprintf(stdout, "Hermes memory projection: %s\n", receipt.HermesMemoryRecordSHA256)
	}
	return nil
}

type mirrorExperienceProjectTrajectoryReceipt struct {
	Schema                     string                                                `json:"schema"`
	State                      string                                                `json:"state"`
	RecordMutation             bool                                                  `json:"recordMutation"`
	TrainingProjectionMutation bool                                                  `json:"trainingProjectionMutation"`
	ModelWeightMutation        bool                                                  `json:"modelWeightMutation"`
	Projection                 axmmirror.MirrorExperienceTrajectoryProjectionReceipt `json:"projection"`
	Authority                  axmmirror.Authority                                   `json:"authority"`
	Timestamp                  time.Time                                             `json:"timestamp"`
}

func runMirrorExperienceProjectTrajectory(commandContext Context, args []string, stdout, _ io.Writer) error {
	draftPath := args[0]
	file, err := os.Open(draftPath)
	if err != nil {
		return fmt.Errorf("open Mirror experience trajectory draft %s: %w", draftPath, err)
	}
	record, loadErr := axmmirror.LoadMirrorExperienceTrajectoryDraft(file)
	closeErr := file.Close()
	if loadErr != nil {
		return fmt.Errorf("load Mirror experience trajectory draft %s: %w", draftPath, loadErr)
	}
	if closeErr != nil {
		return closeErr
	}
	if err := axmmirror.SealMirrorExperienceTrajectory(&record); err != nil {
		return fmt.Errorf("seal Mirror experience trajectory: %w", err)
	}
	recordPath := strings.TrimSpace(stringOption(commandContext, "record-to"))
	learnPath := strings.TrimSpace(stringOption(commandContext, "learn-to"))
	if recordPath == "" || learnPath == "" {
		return errors.New("--record-to and --learn-to are required")
	}
	if err := requireDistinctMirrorPaths(draftPath, recordPath, learnPath); err != nil {
		return err
	}
	recordLine, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode sealed Mirror experience trajectory: %w", err)
	}
	recordLine = append(recordLine, '\n')
	projectionLine, projectionReceipt, err := axmmirror.ProjectMirrorExperienceTrajectory(record)
	if err != nil {
		return err
	}
	if err := appendPrivateLine(recordPath, recordLine); err != nil {
		return fmt.Errorf("append observed Mirror experience trajectory: %w", err)
	}
	if err := appendPrivateLine(learnPath, projectionLine); err != nil {
		return fmt.Errorf("append experience-derived WALDO reflection projection: %w", err)
	}
	receipt := mirrorExperienceProjectTrajectoryReceipt{
		Schema:                     "axm.waldo.mirror-experience-project-trajectory-receipt/v1",
		State:                      "OBSERVED_TRAJECTORY_PROJECTED",
		RecordMutation:             true,
		TrainingProjectionMutation: true,
		Projection:                 projectionReceipt,
		Authority:                  axmmirror.Authority{},
		Timestamp:                  time.Now().UTC(),
	}
	if commandContext.JSON {
		return writeJSON(stdout, receipt)
	}
	fmt.Fprintf(stdout, "Observed trajectory: %s\n", projectionReceipt.RecordSHA256)
	fmt.Fprintf(stdout, "Reflection projection: %s\n", projectionReceipt.ProjectionSHA256)
	fmt.Fprintln(stdout, "Model weights: UNCHANGED")
	return nil
}

type mirrorGroundVerifyReceipt struct {
	Schema                     string              `json:"schema"`
	DatasetSHA256              string              `json:"datasetSha256"`
	Records                    int                 `json:"records"`
	TrainingTargets            int                 `json:"trainingTargets"`
	MemoryOnly                 int                 `json:"memoryOnly"`
	ReviewRequired             int                 `json:"reviewRequired"`
	SyntheticRecords           int                 `json:"syntheticRecords"`
	ObservedRecords            int                 `json:"observedRecords"`
	DataClasses                map[string]int      `json:"dataClasses"`
	RootCoverage               map[string]int      `json:"rootCoverage"`
	ChallengeCoverage          map[string]int      `json:"challengeCoverage"`
	PositiveSeedValidated      bool                `json:"positiveSeedValidated"`
	TrainingProjectionMutation bool                `json:"trainingProjectionMutation"`
	TrainingProjectionRecords  int                 `json:"trainingProjectionRecords"`
	TrainingProjectionSHA256   string              `json:"trainingProjectionSha256,omitempty"`
	ModelWeightMutation        bool                `json:"modelWeightMutation"`
	IdentityMutation           bool                `json:"identityMutation"`
	Authority                  axmmirror.Authority `json:"authority"`
	Timestamp                  time.Time           `json:"timestamp"`
}

func runMirrorGroundVerify(commandContext Context, args []string, stdout, _ io.Writer) error {
	path := args[0]
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open Mirror ground dataset %s: %w", path, err)
	}
	dataset, loadErr := axmmirror.LoadMirrorGroundDataset(file)
	closeErr := file.Close()
	if loadErr != nil {
		return fmt.Errorf("load Mirror ground dataset %s: %w", path, loadErr)
	}
	if closeErr != nil {
		return closeErr
	}
	positiveSeed := boolOption(commandContext, "positive-seed")
	if positiveSeed {
		if err := axmmirror.ValidateMirrorPositiveSeed(dataset, axmmirror.DefaultMirrorPositiveSeedRequirements()); err != nil {
			return fmt.Errorf("validate positive starting seed %s: %w", path, err)
		}
	}
	receipt := mirrorGroundVerifyReceipt{
		Schema:                "axm.waldo.mirror-ground-verify-receipt/v0.39",
		DatasetSHA256:         dataset.DatasetSHA256,
		Records:               len(dataset.Records),
		TrainingTargets:       dataset.TrainingTargets,
		MemoryOnly:            dataset.MemoryOnly,
		ReviewRequired:        dataset.ReviewRequired,
		SyntheticRecords:      dataset.Synthetic,
		ObservedRecords:       dataset.Observed,
		DataClasses:           dataset.DataClasses,
		RootCoverage:          dataset.RootCoverage,
		ChallengeCoverage:     dataset.ChallengeCoverage,
		PositiveSeedValidated: positiveSeed,
		Authority:             axmmirror.Authority{},
		Timestamp:             time.Now().UTC(),
	}
	projectionPath := strings.TrimSpace(stringOption(commandContext, "training-to"))
	if projectionPath != "" {
		if err := requireDistinctMirrorPaths(path, projectionPath); err != nil {
			return err
		}
		projection, err := dataset.TrainingProjectionJSONL()
		if err != nil {
			return err
		}
		if err := writePrivateFileExclusive(projectionPath, projection); err != nil {
			return fmt.Errorf("write Mirror ground training projection %s: %w", projectionPath, err)
		}
		digest := sha256.Sum256(projection)
		receipt.TrainingProjectionMutation = true
		receipt.TrainingProjectionRecords = dataset.TrainingTargets
		receipt.TrainingProjectionSHA256 = fmt.Sprintf("%x", digest[:])
	}
	if commandContext.JSON {
		return writeJSON(stdout, receipt)
	}
	fmt.Fprintf(stdout, "Ground dataset: %s\n", receipt.DatasetSHA256)
	fmt.Fprintf(stdout, "Records: %d (%d synthetic, %d observed)\n", receipt.Records, receipt.SyntheticRecords, receipt.ObservedRecords)
	fmt.Fprintf(stdout, "Dispositions: %d positive targets, %d memory-only, %d review-required\n", receipt.TrainingTargets, receipt.MemoryOnly, receipt.ReviewRequired)
	fmt.Fprintf(stdout, "Coverage: %d roots, %d challenge kinds\n", len(receipt.RootCoverage), len(receipt.ChallengeCoverage))
	if receipt.PositiveSeedValidated {
		fmt.Fprintln(stdout, "Positive starting seed: VALIDATED")
	}
	if receipt.TrainingProjectionMutation {
		fmt.Fprintf(stdout, "Structured WALDO training projection: %d records, %s\n", receipt.TrainingProjectionRecords, receipt.TrainingProjectionSHA256)
	}
	return nil
}

func loadMirrorExperienceOutcomeRequest(path string) (axmmirror.MirrorExperienceOutcomeRequest, error) {
	if path == "-" {
		return axmmirror.LoadMirrorExperienceOutcomeRequest(mirrorExperienceOutcomeInput)
	}
	file, err := os.Open(path)
	if err != nil {
		return axmmirror.MirrorExperienceOutcomeRequest{}, fmt.Errorf("open Mirror experience outcome %s: %w", path, err)
	}
	defer file.Close()
	request, err := axmmirror.LoadMirrorExperienceOutcomeRequest(file)
	if err != nil {
		return request, fmt.Errorf("load Mirror experience outcome %s: %w", path, err)
	}
	return request, nil
}

func loadMirrorExperienceLedgerFile(path string, allowMissing bool) (axmmirror.MirrorExperienceLedger, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) && allowMissing {
		return axmmirror.MirrorExperienceLedger{Episodes: map[string][]axmmirror.MirrorExperienceEvent{}}, nil
	}
	if err != nil {
		return axmmirror.MirrorExperienceLedger{}, fmt.Errorf("open Mirror experience ledger %s: %w", path, err)
	}
	defer file.Close()
	ledger, err := axmmirror.LoadMirrorExperienceLedger(file)
	if err != nil {
		return axmmirror.MirrorExperienceLedger{}, fmt.Errorf("load Mirror experience ledger %s: %w", path, err)
	}
	return ledger, nil
}

func appendMirrorExperienceEvents(path string, events []axmmirror.MirrorExperienceEvent) error {
	var buffer bytes.Buffer
	for _, event := range events {
		line, err := event.JSONLine()
		if err != nil {
			return err
		}
		buffer.Write(line)
	}
	return appendPrivateLine(path, buffer.Bytes())
}

func requireDistinctMirrorPaths(paths ...string) error {
	seen := map[string]string{}
	for _, path := range paths {
		if path == "" {
			continue
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolve private Mirror path %s: %w", path, err)
		}
		absolute = filepath.Clean(absolute)
		if previous, exists := seen[absolute]; exists {
			return fmt.Errorf("private Mirror ledgers must use distinct paths; %s aliases %s", path, previous)
		}
		seen[absolute] = path
	}
	return nil
}

func appendPrivateLine(path string, line []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(line); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func writePrivateFileExclusive(path string, data []byte) (returnErr error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	complete := false
	defer func() {
		if !complete {
			_ = file.Close()
			_ = os.Remove(path)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	complete = true
	return nil
}
