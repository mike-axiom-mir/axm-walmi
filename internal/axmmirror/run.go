package axmmirror

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	TrainingRunWitnessSchema = "axm.waldo-witness.training-run-witness/v0.1"
	RunWitnessStateReady     = "READY"
	RunWitnessStateHold      = "HOLD"
)

type RunBackendEvidence struct {
	Name     string `json:"name"`
	Revision string `json:"revision"`
}

type RunHostEvidence struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
}

type RunAcceleratorEvidence struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	MemoryBytes  uint64 `json:"memory_bytes"`
}

type RunExecutionEvidence struct {
	Backend      RunBackendEvidence       `json:"backend"`
	Framework    string                   `json:"framework"`
	Runtime      string                   `json:"runtime"`
	Host         RunHostEvidence          `json:"host"`
	Accelerators []RunAcceleratorEvidence `json:"accelerators,omitempty"`
	Nodes        int                      `json:"nodes"`
	WorldSize    int                      `json:"world_size"`
}

type RunParametersEvidence struct {
	SHA256               string  `json:"sha256"`
	Profile              string  `json:"profile"`
	ProfileSchema        int     `json:"profile_schema"`
	Epochs               int64   `json:"epochs"`
	Steps                int64   `json:"steps"`
	BatchSize            int64   `json:"batch_size"`
	SequenceLength       int64   `json:"sequence_length"`
	LearningRate         float64 `json:"learning_rate"`
	Seed                 uint64  `json:"seed"`
	PlannedTokenCapacity int64   `json:"planned_token_capacity"`
}

type RunEvaluationSetEvidence struct {
	Selection    string `json:"selection"`
	Seed         uint64 `json:"seed"`
	Records      int64  `json:"records"`
	TokenTargets int64  `json:"token_targets"`
	TextBytes    int64  `json:"text_bytes"`
	SHA256       string `json:"sha256"`
}

type RunArtifactEvidence struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type RunInitializationEvidence struct {
	SourceType  string              `json:"source_type"`
	SourceID    string              `json:"source_id"`
	SourceRunID string              `json:"source_run_id,omitempty"`
	Artifact    RunArtifactEvidence `json:"artifact"`
}

// TrainingRunWitness binds an immutable run plan to its current durable run
// record. READY is deliberately narrow: only a complete, non-simulated run can
// back downstream witnessed behavior. Other valid lifecycle states still
// produce a HOLD receipt instead of disappearing.
type TrainingRunWitness struct {
	Schema                  string                     `json:"schema"`
	State                   string                     `json:"state"`
	RunID                   string                     `json:"run_id"`
	ModelID                 string                     `json:"model_id"`
	Stage                   string                     `json:"stage"`
	StageType               string                     `json:"stage_type"`
	Ordinal                 int                        `json:"ordinal"`
	Objective               string                     `json:"objective"`
	RunBOMSHA256            string                     `json:"run_bom_sha256"`
	RunBOMDocumentSHA256    string                     `json:"run_bom_document_sha256"`
	RunRecordSHA256         string                     `json:"run_record_sha256"`
	RunRecordDocumentSHA256 string                     `json:"run_record_document_sha256"`
	ReceiptSHA256           string                     `json:"receipt_sha256,omitempty"`
	ArchitectureSHA256      string                     `json:"architecture_sha256"`
	Corpus                  CorpusEvidenceLens         `json:"corpus"`
	Execution               RunExecutionEvidence       `json:"execution"`
	Parameters              RunParametersEvidence      `json:"parameters"`
	EvaluationSet           *RunEvaluationSetEvidence  `json:"evaluation_set,omitempty"`
	Initialization          *RunInitializationEvidence `json:"initialization,omitempty"`
	RunState                string                     `json:"run_state"`
	Simulated               bool                       `json:"simulated"`
	ObservationSHA256       string                     `json:"observation_sha256,omitempty"`
	ProgressSHA256          string                     `json:"progress_sha256,omitempty"`
	ObservedSteps           int64                      `json:"observed_steps,omitempty"`
	ConsumedTokens          int64                      `json:"consumed_tokens,omitempty"`
	CheckpointCount         int                        `json:"checkpoint_count"`
	EvaluationCount         int                        `json:"evaluation_count"`
	Artifacts               []RunArtifactEvidence      `json:"artifacts,omitempty"`
	Holds                   []string                   `json:"holds,omitempty"`
	Notices                 []string                   `json:"notices"`
	Authority               Authority                  `json:"authority"`
}

type waldoOptimizer struct {
	Name        string  `json:"name"`
	WeightDecay float64 `json:"weight_decay"`
	Beta1       float64 `json:"beta1"`
	Beta2       float64 `json:"beta2"`
	Epsilon     float64 `json:"epsilon"`
}

type waldoSchedule struct {
	Name             string  `json:"name"`
	WarmupSteps      int64   `json:"warmup_steps"`
	MinimumRateRatio float64 `json:"minimum_rate_ratio"`
}

type waldoDataPlan struct {
	Order                string            `json:"order"`
	ShuffleBufferRecords int               `json:"shuffle_buffer_records"`
	ShuffleBufferBytes   int64             `json:"shuffle_buffer_bytes"`
	Packing              string            `json:"packing"`
	CorpusWeights        map[string]uint64 `json:"corpus_weights,omitempty"`
}

type waldoEvaluationPolicy struct {
	Selection  string  `json:"selection"`
	Fraction   float64 `json:"fraction"`
	MaxRecords int     `json:"max_records"`
	MaxBytes   int64   `json:"max_bytes"`
}

type waldoResolvedParameters struct {
	Profile              string                 `json:"profile"`
	ProfileSchema        int                    `json:"profile_schema"`
	Epochs               int64                  `json:"epochs,omitempty"`
	RequestedTokens      int64                  `json:"requested_tokens,omitempty"`
	Steps                int64                  `json:"steps"`
	BatchSize            int64                  `json:"batch_size"`
	SequenceLength       int64                  `json:"sequence_length"`
	LearningRate         float64                `json:"learning_rate"`
	Seed                 uint64                 `json:"seed"`
	Optimizer            waldoOptimizer         `json:"optimizer"`
	Schedule             waldoSchedule          `json:"schedule"`
	Data                 waldoDataPlan          `json:"data"`
	Evaluation           *waldoEvaluationPolicy `json:"evaluation,omitempty"`
	CheckpointEvery      int64                  `json:"checkpoint_every"`
	EvaluateEvery        int64                  `json:"evaluate_every"`
	PlannedTokenCapacity int64                  `json:"planned_token_capacity"`
}

// waldoConversationTransform mirrors the documented schema-1 model-side
// conversation view. Keeping it in the local wire projection preserves the
// exact WALDO run BOM identity without importing WALDO's internal packages.
type waldoConversationTransform struct {
	Template        string   `json:"template"`
	SupervisedRoles []string `json:"supervised_roles"`
}

func (transform waldoConversationTransform) isZero() bool {
	return transform.Template == "" && len(transform.SupervisedRoles) == 0
}

type waldoEvaluationSet struct {
	Selection    string `json:"selection"`
	Seed         uint64 `json:"seed"`
	Records      int64  `json:"records"`
	TokenTargets int64  `json:"token_targets"`
	TextBytes    int64  `json:"text_bytes"`
	SHA256       string `json:"sha256"`
}

type waldoRunArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type waldoInitialization struct {
	SourceType  string           `json:"source_type,omitempty"`
	SourceID    string           `json:"source_id,omitempty"`
	SourceRunID string           `json:"source_run_id,omitempty"`
	Artifact    waldoRunArtifact `json:"artifact"`
}

type waldoRunBOM struct {
	Kind               string                     `json:"kind"`
	Schema             int                        `json:"schema"`
	Subject            string                     `json:"subject"`
	ID                 string                     `json:"id"`
	ModelID            string                     `json:"model_id"`
	Stage              string                     `json:"stage"`
	StageType          string                     `json:"stage_type"`
	Ordinal            int                        `json:"ordinal"`
	Objective          string                     `json:"objective"`
	Conversation       waldoConversationTransform `json:"conversation,omitzero"`
	Execution          RunExecutionEvidence       `json:"execution"`
	ArchitectureSHA256 string                     `json:"architecture_sha256"`
	CorpusBOMSHA256    string                     `json:"corpus_bom_sha256"`
	CorpusBOM          waldoCorpusBOM             `json:"corpus_bom"`
	Parameters         waldoResolvedParameters    `json:"parameters"`
	EvaluationSet      *waldoEvaluationSet        `json:"evaluation_set,omitempty"`
	Initialization     *waldoInitialization       `json:"initialization,omitempty"`
}

type waldoRunCheckpoint struct {
	Step      int64              `json:"step"`
	Tokens    int64              `json:"tokens"`
	Artifacts []waldoRunArtifact `json:"artifacts"`
}

type waldoRunEvaluation struct {
	Step    int64              `json:"step"`
	Tokens  int64              `json:"tokens"`
	Metrics map[string]float64 `json:"metrics"`
}

type waldoCorpusConsumption struct {
	Corpus       string `json:"corpus"`
	TokenTargets int64  `json:"token_targets"`
}

type waldoRunObservation struct {
	Simulated      bool                     `json:"simulated"`
	Steps          int64                    `json:"steps"`
	ConsumedTokens int64                    `json:"consumed_tokens"`
	FinalLoss      *float64                 `json:"final_loss,omitempty"`
	Checkpoints    []waldoRunCheckpoint     `json:"checkpoints,omitempty"`
	Evaluations    []waldoRunEvaluation     `json:"evaluations,omitempty"`
	Artifacts      []waldoRunArtifact       `json:"artifacts"`
	Consumption    []waldoCorpusConsumption `json:"consumption,omitempty"`
}

type waldoRunProgress struct {
	Steps          int64                `json:"steps"`
	ConsumedTokens int64                `json:"consumed_tokens"`
	LastLoss       *float64             `json:"last_loss,omitempty"`
	Checkpoints    []waldoRunCheckpoint `json:"checkpoints,omitempty"`
	Evaluations    []waldoRunEvaluation `json:"evaluations,omitempty"`
}

type waldoRunAttempt struct {
	Ordinal    int    `json:"ordinal"`
	Started    string `json:"started"`
	Finished   string `json:"finished,omitempty"`
	State      string `json:"state"`
	Error      string `json:"error,omitempty"`
	ResumeStep int64  `json:"resume_step,omitempty"`
}

type waldoRunRecord struct {
	Kind        string               `json:"kind"`
	Schema      int                  `json:"schema"`
	ID          string               `json:"id"`
	State       string               `json:"state"`
	BOMSHA256   string               `json:"bom_sha256"`
	Planned     string               `json:"planned"`
	Started     string               `json:"started,omitempty"`
	Finished    string               `json:"finished,omitempty"`
	Observation *waldoRunObservation `json:"observation,omitempty"`
	Progress    *waldoRunProgress    `json:"progress,omitempty"`
	Attempts    []waldoRunAttempt    `json:"attempts,omitempty"`
	Error       string               `json:"error,omitempty"`
}

// WitnessTrainingRun reads only the supplied durable records. It validates
// their identities and internal links but deliberately does not follow or
// execute artifact paths.
func WitnessTrainingRun(runBOMData, runRecordData []byte) (TrainingRunWitness, error) {
	var runBOM waldoRunBOM
	if err := decodeJSON(runBOMData, &runBOM, false); err != nil {
		return TrainingRunWitness{}, fmt.Errorf("decode training run BOM: %w", err)
	}
	if err := validateRunBOM(runBOM); err != nil {
		return TrainingRunWitness{}, err
	}
	canonicalRunBOM, err := json.Marshal(runBOM)
	if err != nil {
		return TrainingRunWitness{}, fmt.Errorf("canonicalize training run BOM: %w", err)
	}
	runBOMDigest := digestBytes(canonicalRunBOM)

	corpusData, err := json.Marshal(runBOM.CorpusBOM)
	if err != nil {
		return TrainingRunWitness{}, fmt.Errorf("encode embedded corpus BOM: %w", err)
	}
	corpusLens, err := LensCorpusBOM(corpusData)
	if err != nil {
		return TrainingRunWitness{}, fmt.Errorf("embedded corpus BOM: %w", err)
	}
	if corpusLens.BOMSHA256 != runBOM.CorpusBOMSHA256 {
		return TrainingRunWitness{}, fmt.Errorf("embedded corpus BOM digest mismatch: run pins %s, got %s", runBOM.CorpusBOMSHA256, corpusLens.BOMSHA256)
	}

	var run waldoRunRecord
	if err := decodeJSON(runRecordData, &run, false); err != nil {
		return TrainingRunWitness{}, fmt.Errorf("decode training run record: %w", err)
	}
	if err := validateRunRecord(runBOM, runBOMDigest, run); err != nil {
		return TrainingRunWitness{}, err
	}
	canonicalRun, err := json.Marshal(run)
	if err != nil {
		return TrainingRunWitness{}, fmt.Errorf("canonicalize training run record: %w", err)
	}
	parameterData, err := json.Marshal(runBOM.Parameters)
	if err != nil {
		return TrainingRunWitness{}, fmt.Errorf("canonicalize training parameters: %w", err)
	}

	witness := TrainingRunWitness{
		Schema: TrainingRunWitnessSchema, State: RunWitnessStateHold,
		RunID: runBOM.ID, ModelID: runBOM.ModelID, Stage: runBOM.Stage, StageType: runBOM.StageType,
		Ordinal: runBOM.Ordinal, Objective: runBOM.Objective,
		RunBOMSHA256: runBOMDigest, RunBOMDocumentSHA256: digestBytes(runBOMData),
		RunRecordSHA256: digestBytes(canonicalRun), RunRecordDocumentSHA256: digestBytes(runRecordData),
		ArchitectureSHA256: runBOM.ArchitectureSHA256, Corpus: corpusLens, Execution: runBOM.Execution,
		Parameters: RunParametersEvidence{
			SHA256: digestBytes(parameterData), Profile: runBOM.Parameters.Profile, ProfileSchema: runBOM.Parameters.ProfileSchema,
			Epochs: runBOM.Parameters.Epochs, Steps: runBOM.Parameters.Steps, BatchSize: runBOM.Parameters.BatchSize,
			SequenceLength: runBOM.Parameters.SequenceLength, LearningRate: runBOM.Parameters.LearningRate,
			Seed: runBOM.Parameters.Seed, PlannedTokenCapacity: runBOM.Parameters.PlannedTokenCapacity,
		},
		RunState: run.State, Simulated: runBOM.Execution.Backend.Name == "fake",
		Notices: []string{
			"the witness validates recorded identities and lifecycle structure but does not reread artifact bytes",
			"a complete run does not by itself prove model quality, safety, legal usability, or source-to-output causality",
		},
		Authority: Authority{},
	}
	if runBOM.EvaluationSet != nil {
		witness.EvaluationSet = &RunEvaluationSetEvidence{
			Selection: runBOM.EvaluationSet.Selection, Seed: runBOM.EvaluationSet.Seed, Records: runBOM.EvaluationSet.Records,
			TokenTargets: runBOM.EvaluationSet.TokenTargets, TextBytes: runBOM.EvaluationSet.TextBytes, SHA256: runBOM.EvaluationSet.SHA256,
		}
	}
	if runBOM.Initialization != nil {
		witness.Initialization = &RunInitializationEvidence{
			SourceType: runBOM.Initialization.SourceType, SourceID: runBOM.Initialization.SourceID, SourceRunID: runBOM.Initialization.SourceRunID,
			Artifact: projectRunArtifact(runBOM.Initialization.Artifact),
		}
	}
	if run.Observation != nil {
		observationData, err := json.Marshal(run.Observation)
		if err != nil {
			return TrainingRunWitness{}, fmt.Errorf("canonicalize training observation: %w", err)
		}
		witness.ObservationSHA256 = digestBytes(observationData)
		witness.ObservedSteps = run.Observation.Steps
		witness.ConsumedTokens = run.Observation.ConsumedTokens
		witness.CheckpointCount = len(run.Observation.Checkpoints)
		witness.EvaluationCount = len(run.Observation.Evaluations)
		witness.Artifacts = projectRunArtifacts(run.Observation.Artifacts)
		witness.Simulated = witness.Simulated || run.Observation.Simulated
	}
	if run.Progress != nil {
		progressData, err := json.Marshal(run.Progress)
		if err != nil {
			return TrainingRunWitness{}, fmt.Errorf("canonicalize training progress: %w", err)
		}
		witness.ProgressSHA256 = digestBytes(progressData)
		witness.ObservedSteps = run.Progress.Steps
		witness.ConsumedTokens = run.Progress.ConsumedTokens
		witness.CheckpointCount = len(run.Progress.Checkpoints)
		witness.EvaluationCount = len(run.Progress.Evaluations)
	}

	switch {
	case run.State != "complete":
		witness.Holds = append(witness.Holds, "training run state is "+run.State+", not complete")
	case witness.Simulated:
		witness.Holds = append(witness.Holds, "training run is explicitly simulated")
	default:
		witness.State = RunWitnessStateReady
	}
	witness.ReceiptSHA256, err = runWitnessReceiptDigest(witness)
	if err != nil {
		return TrainingRunWitness{}, err
	}
	if err := witness.Validate(); err != nil {
		return TrainingRunWitness{}, fmt.Errorf("generated training run witness: %w", err)
	}
	return witness, nil
}

func (witness TrainingRunWitness) Validate() error {
	if witness.Schema != TrainingRunWitnessSchema || !oneOf(witness.State, RunWitnessStateReady, RunWitnessStateHold) {
		return fmt.Errorf("unsupported training run witness identity %q state %q", witness.Schema, witness.State)
	}
	if strings.TrimSpace(witness.RunID) == "" || strings.TrimSpace(witness.ModelID) == "" || strings.TrimSpace(witness.Stage) == "" || strings.TrimSpace(witness.StageType) == "" || strings.TrimSpace(witness.Objective) == "" || witness.Ordinal < 1 {
		return errors.New("training run witness has incomplete run identity")
	}
	for _, item := range []struct{ name, value string }{
		{"run witness.run_bom_sha256", witness.RunBOMSHA256},
		{"run witness.run_bom_document_sha256", witness.RunBOMDocumentSHA256},
		{"run witness.run_record_sha256", witness.RunRecordSHA256},
		{"run witness.run_record_document_sha256", witness.RunRecordDocumentSHA256},
		{"run witness.receipt_sha256", witness.ReceiptSHA256},
		{"run witness.architecture_sha256", witness.ArchitectureSHA256},
		{"run witness.parameters.sha256", witness.Parameters.SHA256},
	} {
		if err := validateSHA256(item.name, item.value); err != nil {
			return err
		}
	}
	for _, item := range []struct{ name, value string }{
		{"run witness.observation_sha256", witness.ObservationSHA256},
		{"run witness.progress_sha256", witness.ProgressSHA256},
	} {
		if item.value != "" {
			if err := validateSHA256(item.name, item.value); err != nil {
				return err
			}
		}
	}
	if err := witness.Corpus.Validate(); err != nil {
		return fmt.Errorf("run witness corpus: %w", err)
	}
	if err := validateExecution(witness.Execution); err != nil {
		return err
	}
	if strings.TrimSpace(witness.Parameters.Profile) == "" || witness.Parameters.ProfileSchema < 1 || witness.Parameters.Epochs < 1 || witness.Parameters.Steps < 1 || witness.Parameters.BatchSize < 1 || witness.Parameters.SequenceLength < 1 || !finitePositive(witness.Parameters.LearningRate) {
		return errors.New("training run witness has invalid parameter summary")
	}
	capacity, overflow := multiplyPositive(witness.Parameters.Steps, witness.Parameters.BatchSize, witness.Parameters.SequenceLength)
	if overflow || capacity != witness.Parameters.PlannedTokenCapacity {
		return errors.New("training run witness planned token capacity does not match its parameter summary")
	}
	if witness.EvaluationSet != nil {
		if strings.TrimSpace(witness.EvaluationSet.Selection) == "" || witness.EvaluationSet.Records < 0 || witness.EvaluationSet.Records > witness.Corpus.Totals.Docs || witness.EvaluationSet.TokenTargets < 0 || witness.EvaluationSet.TextBytes < 0 {
			return errors.New("training run witness has invalid evaluation-set evidence")
		}
		if err := validateSHA256("run witness evaluation_set.sha256", witness.EvaluationSet.SHA256); err != nil {
			return err
		}
	}
	if witness.Initialization != nil {
		if !oneOf(witness.Initialization.SourceType, "run", "origin") || strings.TrimSpace(witness.Initialization.SourceID) == "" {
			return errors.New("training run witness has invalid initialization identity")
		}
		if witness.Initialization.SourceType == "run" && (witness.Initialization.SourceRunID == "" || witness.Initialization.SourceRunID != witness.Initialization.SourceID) {
			return errors.New("training run witness run initialization has inconsistent source_run_id")
		}
		if witness.Initialization.SourceType == "origin" && witness.Initialization.SourceRunID != "" {
			return errors.New("training run witness origin initialization must not carry source_run_id")
		}
		if err := validateRunArtifactEvidence("run witness initialization artifact", witness.Initialization.Artifact); err != nil {
			return err
		}
	}
	if !oneOf(witness.RunState, "planned", "running", "complete", "failed", "interrupted") {
		return fmt.Errorf("training run witness has unsupported run_state %q", witness.RunState)
	}
	if witness.ObservedSteps < 0 || witness.ObservedSteps > witness.Parameters.Steps || witness.ConsumedTokens < 0 || witness.ConsumedTokens > witness.Parameters.PlannedTokenCapacity || witness.CheckpointCount < 0 || witness.EvaluationCount < 0 {
		return errors.New("training run witness observations are outside the planned range")
	}
	seenArtifacts := map[string]bool{}
	for i, artifact := range witness.Artifacts {
		if err := validateRunArtifactEvidence(fmt.Sprintf("run witness artifact %d", i), artifact); err != nil {
			return err
		}
		if seenArtifacts[artifact.Path] {
			return fmt.Errorf("training run witness has duplicate artifact path %q", artifact.Path)
		}
		seenArtifacts[artifact.Path] = true
	}
	if witness.Simulated != (witness.Execution.Backend.Name == "fake") {
		return errors.New("training run witness simulation state does not match its backend")
	}
	switch witness.RunState {
	case "planned":
		if witness.ObservationSHA256 != "" || witness.ProgressSHA256 != "" || len(witness.Artifacts) != 0 {
			return errors.New("planned training run witness contains observations or progress")
		}
	case "running", "failed", "interrupted":
		if witness.ObservationSHA256 != "" || len(witness.Artifacts) != 0 {
			return fmt.Errorf("%s training run witness contains a complete observation", witness.RunState)
		}
	case "complete":
		if witness.ObservationSHA256 == "" || witness.ProgressSHA256 != "" || len(witness.Artifacts) == 0 {
			return errors.New("complete training run witness has incomplete observation evidence")
		}
	}
	if witness.State == RunWitnessStateReady {
		if witness.RunState != "complete" || witness.Simulated || witness.ObservationSHA256 == "" || len(witness.Artifacts) == 0 || len(witness.Holds) != 0 {
			return errors.New("READY training run witness is not a complete real observation")
		}
	} else if len(witness.Holds) == 0 {
		return errors.New("HOLD training run witness requires at least one reason")
	}
	if len(witness.Notices) == 0 {
		return errors.New("training run witness must state its evidence boundaries")
	}
	if !witness.Authority.closed() {
		return errors.New("training run witness must carry closed authority")
	}
	actual, err := runWitnessReceiptDigest(witness)
	if err != nil {
		return err
	}
	if actual != witness.ReceiptSHA256 {
		return fmt.Errorf("training run witness receipt digest mismatch: expected %s, got %s", witness.ReceiptSHA256, actual)
	}
	return nil
}

// SealWitnessed requires both answering identity and real-run lineage before
// accepting a behavior record. Origin-backed models can still use
// SealAnchored; this stronger operation is intentionally run-only.
func SealWitnessed(draft BehaviorEvidenceDraft, anchor OriginAnchor, witness TrainingRunWitness) (SealedBehaviorEvidence, error) {
	if err := anchor.Validate(); err != nil {
		return SealedBehaviorEvidence{}, fmt.Errorf("origin anchor: %w", err)
	}
	if err := witness.Validate(); err != nil {
		return SealedBehaviorEvidence{}, fmt.Errorf("training run witness: %w", err)
	}
	if anchor.State != AnchorStateAnchored {
		return SealedBehaviorEvidence{}, errors.New("origin anchor is HOLD; a complete answering identity is required")
	}
	if anchor.SourceType != "run" || anchor.RunID == "" {
		return SealedBehaviorEvidence{}, errors.New("seal-witnessed requires a run-backed origin anchor")
	}
	if witness.State != RunWitnessStateReady {
		return SealedBehaviorEvidence{}, errors.New("training run witness is HOLD; a complete non-simulated run is required")
	}
	if anchor.ModelID != witness.ModelID {
		return SealedBehaviorEvidence{}, errors.New("training run witness model_id does not match origin anchor")
	}
	if anchor.RunID != witness.RunID {
		return SealedBehaviorEvidence{}, errors.New("training run witness run_id does not match origin anchor")
	}
	if draft.WALDO.RunBOMSHA256 != witness.RunBOMSHA256 {
		return SealedBehaviorEvidence{}, errors.New("draft run_bom_sha256 does not match training run witness")
	}
	if draft.WALDO.CorpusBOMSHA256 != witness.Corpus.BOMSHA256 {
		return SealedBehaviorEvidence{}, errors.New("draft corpus_bom_sha256 does not match training run witness")
	}
	const verificationName = "waldo-training-run-witness"
	found := false
	for _, verification := range draft.Verification {
		if verification.Name != verificationName {
			continue
		}
		found = true
		if verification.State != "pass" || verification.EvidenceSHA256 != witness.ReceiptSHA256 {
			return SealedBehaviorEvidence{}, errors.New("draft training-run witness verification does not match the supplied receipt")
		}
	}
	if !found {
		draft.Verification = append(draft.Verification, Verification{Name: verificationName, State: "pass", EvidenceSHA256: witness.ReceiptSHA256})
	}
	return SealAnchored(draft, anchor)
}

func runWitnessReceiptDigest(witness TrainingRunWitness) (string, error) {
	witness.ReceiptSHA256 = ""
	data, err := json.Marshal(witness)
	if err != nil {
		return "", fmt.Errorf("encode training run witness receipt: %w", err)
	}
	return digestBytes(data), nil
}

func validateRunBOM(bom waldoRunBOM) error {
	if bom.Kind != "openwaldo-bom" || bom.Schema != 1 || bom.Subject != "training-run" {
		return fmt.Errorf("unsupported training run BOM identity %q schema %d subject %q", bom.Kind, bom.Schema, bom.Subject)
	}
	if strings.TrimSpace(bom.ID) == "" || strings.TrimSpace(bom.ModelID) == "" || strings.TrimSpace(bom.Stage) == "" || strings.TrimSpace(bom.StageType) == "" || strings.TrimSpace(bom.Objective) == "" || bom.Ordinal < 1 {
		return errors.New("training run BOM has incomplete run identity")
	}
	if err := validateExecution(bom.Execution); err != nil {
		return err
	}
	if err := validateSHA256("training run BOM architecture_sha256", bom.ArchitectureSHA256); err != nil {
		return err
	}
	if err := validateSHA256("training run BOM corpus_bom_sha256", bom.CorpusBOMSHA256); err != nil {
		return err
	}
	if err := validateResolvedParameters(bom.Parameters); err != nil {
		return err
	}
	if err := validateConversationTransform(bom.Objective, bom.Conversation); err != nil {
		return err
	}
	if err := validateEvaluationSet(bom); err != nil {
		return err
	}
	if bom.Initialization != nil {
		initialization := bom.Initialization
		if !oneOf(initialization.SourceType, "run", "origin") || strings.TrimSpace(initialization.SourceID) == "" {
			return errors.New("training run initialization requires run or origin source identity")
		}
		if initialization.SourceType == "run" && (initialization.SourceRunID == "" || initialization.SourceRunID != initialization.SourceID) {
			return errors.New("run initialization requires source_run_id equal to source_id")
		}
		if initialization.SourceType == "origin" && initialization.SourceRunID != "" {
			return errors.New("origin initialization must not carry source_run_id")
		}
		if err := validateRunArtifact("training run initialization artifact", initialization.Artifact); err != nil {
			return err
		}
	}
	return nil
}

func validateExecution(execution RunExecutionEvidence) error {
	if strings.TrimSpace(execution.Backend.Name) == "" || strings.TrimSpace(execution.Backend.Revision) == "" || strings.TrimSpace(execution.Framework) == "" || strings.TrimSpace(execution.Runtime) == "" || strings.TrimSpace(execution.Host.OS) == "" || strings.TrimSpace(execution.Host.Architecture) == "" || execution.Nodes < 1 || execution.WorldSize < 1 {
		return errors.New("training execution has incomplete backend, framework, runtime, host, or topology identity")
	}
	for i, accelerator := range execution.Accelerators {
		if strings.TrimSpace(accelerator.Manufacturer) == "" || strings.TrimSpace(accelerator.Model) == "" || accelerator.MemoryBytes == 0 {
			return fmt.Errorf("training execution accelerator %d has incomplete identity", i)
		}
	}
	return nil
}

func validateResolvedParameters(parameters waldoResolvedParameters) error {
	if strings.TrimSpace(parameters.Profile) == "" || parameters.ProfileSchema < 1 || parameters.Epochs < 1 || parameters.Steps < 1 || parameters.BatchSize < 1 || parameters.SequenceLength < 1 || !finitePositive(parameters.LearningRate) {
		return errors.New("training run parameters have invalid profile or core numeric fields")
	}
	capacity, overflow := multiplyPositive(parameters.Steps, parameters.BatchSize, parameters.SequenceLength)
	if overflow || capacity != parameters.PlannedTokenCapacity {
		return errors.New("training run planned token capacity does not match steps, batch size, and sequence length")
	}
	if parameters.RequestedTokens < 0 || parameters.RequestedTokens > parameters.PlannedTokenCapacity {
		return errors.New("training run requested token budget is outside the planned capacity")
	}
	if parameters.RequestedTokens > 0 {
		stepCapacity, stepOverflow := multiplyPositive(parameters.BatchSize, parameters.SequenceLength)
		if stepOverflow || parameters.PlannedTokenCapacity-parameters.RequestedTokens >= stepCapacity {
			return errors.New("training run requested token budget is not rounded to the smallest complete optimizer step")
		}
	}
	if strings.TrimSpace(parameters.Optimizer.Name) == "" || parameters.Optimizer.WeightDecay < 0 || parameters.Optimizer.WeightDecay > 1 || !finite(parameters.Optimizer.WeightDecay) || !finite(parameters.Optimizer.Beta1) || !finite(parameters.Optimizer.Beta2) || !finitePositive(parameters.Optimizer.Epsilon) {
		return errors.New("training run optimizer is incomplete or non-finite")
	}
	if strings.TrimSpace(parameters.Schedule.Name) == "" || parameters.Schedule.WarmupSteps < 0 || parameters.Schedule.WarmupSteps > parameters.Steps || !finite(parameters.Schedule.MinimumRateRatio) || parameters.Schedule.MinimumRateRatio < 0 {
		return errors.New("training run schedule is incomplete or invalid")
	}
	if strings.TrimSpace(parameters.Data.Order) == "" || strings.TrimSpace(parameters.Data.Packing) == "" || parameters.Data.ShuffleBufferRecords < 1 || parameters.Data.ShuffleBufferBytes < 1 {
		return errors.New("training run data plan is incomplete")
	}
	for corpus, weight := range parameters.Data.CorpusWeights {
		if strings.TrimSpace(corpus) == "" || weight == 0 {
			return errors.New("training run data plan has invalid corpus weight")
		}
	}
	if parameters.CheckpointEvery < 0 || parameters.CheckpointEvery > parameters.Steps || parameters.EvaluateEvery < 0 || parameters.EvaluateEvery > parameters.Steps {
		return errors.New("training run checkpoint or evaluation interval is outside the planned step range")
	}
	if policy := parameters.Evaluation; policy != nil {
		if strings.TrimSpace(policy.Selection) == "" || !finite(policy.Fraction) || policy.Fraction < 0 || policy.Fraction >= 1 || policy.MaxRecords < 0 || policy.MaxBytes < 0 {
			return errors.New("training run evaluation policy is invalid")
		}
	}
	return nil
}

func validateConversationTransform(objective string, transform waldoConversationTransform) error {
	if transform.isZero() {
		if objective == "assistant-response-modeling" {
			return errors.New("assistant-response-modeling requires a conversation transform")
		}
		return nil
	}
	if !oneOf(transform.Template, "user-assistant-v1", "chatml-v1") || len(transform.SupervisedRoles) == 0 {
		return errors.New("training run conversation transform has an unsupported template or no supervised roles")
	}
	seen := map[string]bool{}
	for _, role := range transform.SupervisedRoles {
		if !oneOf(role, "system", "user", "assistant", "tool") || seen[role] {
			return fmt.Errorf("training run conversation transform has invalid or duplicate supervised role %q", role)
		}
		seen[role] = true
	}
	return nil
}

func validateEvaluationSet(bom waldoRunBOM) error {
	policy := bom.Parameters.Evaluation
	if policy == nil {
		if bom.EvaluationSet != nil {
			return errors.New("training run has unexpected held-out evidence without an evaluation policy")
		}
		return nil
	}
	set := bom.EvaluationSet
	if set == nil || set.Selection != policy.Selection || set.Seed != bom.Parameters.Seed || set.Records < 0 || set.Records > bom.CorpusBOM.Totals.Docs || set.TokenTargets < 0 || set.TextBytes < 0 {
		return errors.New("training run evaluation set does not match its resolved policy or corpus")
	}
	if err := validateSHA256("training run evaluation set.sha256", set.SHA256); err != nil {
		return err
	}
	if policy.Fraction == 0 || policy.MaxRecords == 0 || policy.MaxBytes == 0 {
		if set.Records != 0 || set.TokenTargets != 0 || set.TextBytes != 0 {
			return errors.New("disabled training evaluation policy selected records")
		}
	} else if bom.CorpusBOM.Totals.Docs > 1 && set.Records == 0 {
		return errors.New("enabled training evaluation policy selected no records")
	}
	return nil
}

func validateRunRecord(bom waldoRunBOM, bomDigest string, run waldoRunRecord) error {
	if run.Kind != "waldo-training-run" || run.Schema != 1 || run.ID != bom.ID || strings.TrimSpace(run.Planned) == "" {
		return errors.New("training run record identity does not match its run BOM")
	}
	if run.BOMSHA256 != bomDigest {
		return fmt.Errorf("training run record BOM digest mismatch: record pins %s, got %s", run.BOMSHA256, bomDigest)
	}
	if err := validateAttempts(run); err != nil {
		return err
	}
	if run.Progress != nil {
		if run.Observation != nil {
			return errors.New("training run record contains both progress and a complete observation")
		}
		if err := validateRunProgress(*run.Progress, bom.Parameters); err != nil {
			return err
		}
	}
	switch run.State {
	case "planned":
		if run.Started != "" || run.Finished != "" || run.Observation != nil || run.Progress != nil || run.Error != "" {
			return errors.New("planned training run contains observations or terminal state")
		}
	case "running":
		if run.Started == "" || run.Finished != "" || run.Observation != nil || run.Error != "" {
			return errors.New("running training run has inconsistent state")
		}
	case "complete":
		if run.Started == "" || run.Finished == "" || run.Observation == nil || run.Progress != nil || run.Error != "" || len(run.Observation.Artifacts) == 0 {
			return errors.New("complete training run has incomplete observations")
		}
		if err := validateRunObservation(*run.Observation, bom); err != nil {
			return err
		}
	case "failed", "interrupted":
		if run.Started == "" || run.Finished == "" || run.Observation != nil || strings.TrimSpace(run.Error) == "" {
			return errors.New("terminal training run has inconsistent failure state")
		}
	default:
		return fmt.Errorf("training run record has unsupported state %q", run.State)
	}
	return nil
}

func validateAttempts(run waldoRunRecord) error {
	for i, attempt := range run.Attempts {
		if attempt.Ordinal != i+1 || attempt.Started == "" || attempt.ResumeStep < 0 {
			return fmt.Errorf("training run attempt %d has invalid identity", i+1)
		}
		switch attempt.State {
		case "running":
			if i != len(run.Attempts)-1 || run.State != "running" || attempt.Finished != "" || attempt.Error != "" {
				return fmt.Errorf("training run attempt %d has inconsistent running state", i+1)
			}
		case "complete":
			if i != len(run.Attempts)-1 || run.State != "complete" || attempt.Finished == "" || attempt.Error != "" {
				return fmt.Errorf("training run attempt %d has inconsistent completion", i+1)
			}
		case "failed", "interrupted":
			if attempt.Finished == "" || strings.TrimSpace(attempt.Error) == "" {
				return fmt.Errorf("training run attempt %d has incomplete terminal state", i+1)
			}
		default:
			return fmt.Errorf("training run attempt %d has unsupported state %q", i+1, attempt.State)
		}
	}
	return nil
}

func validateRunObservation(observation waldoRunObservation, bom waldoRunBOM) error {
	if observation.Steps < 0 || observation.Steps > bom.Parameters.Steps || observation.ConsumedTokens < 0 || observation.ConsumedTokens > bom.Parameters.PlannedTokenCapacity {
		return errors.New("training run observation is outside the planned step or token range")
	}
	if observation.FinalLoss != nil && (!finite(*observation.FinalLoss) || *observation.FinalLoss < 0) {
		return errors.New("training run observation final loss is invalid")
	}
	seenArtifacts := map[string]bool{}
	for i, artifact := range observation.Artifacts {
		if err := validateRunArtifact(fmt.Sprintf("training run observation artifact %d", i), artifact); err != nil {
			return err
		}
		if seenArtifacts[artifact.Path] {
			return fmt.Errorf("training run observation has duplicate artifact path %q", artifact.Path)
		}
		seenArtifacts[artifact.Path] = true
	}
	if err := validateRunEvents(observation.Checkpoints, observation.Evaluations, observation.Steps, observation.ConsumedTokens, seenArtifacts); err != nil {
		return err
	}
	if set := bom.EvaluationSet; set != nil && set.Records > 0 && bom.Parameters.EvaluateEvery > 0 {
		if len(observation.Evaluations) == 0 {
			return errors.New("training run configured held-out evaluation but recorded no metrics")
		}
		for i, evaluation := range observation.Evaluations {
			if _, exists := evaluation.Metrics["heldout_loss"]; !exists {
				return fmt.Errorf("training evaluation %d does not report heldout_loss", i+1)
			}
		}
		if bom.Execution.Backend.Name == "pytorch" {
			finalMetrics := observation.Evaluations[len(observation.Evaluations)-1].Metrics
			if _, exists := finalMetrics["artifact_heldout_loss"]; !exists {
				return errors.New("final PyTorch evaluation does not verify the persisted model artifact")
			}
		}
	}
	if observation.Simulated != (bom.Execution.Backend.Name == "fake") {
		return errors.New("training run observation simulation state does not match its backend")
	}
	if bom.Parameters.Data.Order == "corpus-balanced-shuffle-v1" || bom.Parameters.Data.Order == "corpus-weighted-shuffle-v1" {
		selected := make(map[string]bool, len(bom.CorpusBOM.Paths))
		for _, path := range bom.CorpusBOM.Paths {
			selected[path] = true
		}
		seen := map[string]bool{}
		var total int64
		for _, item := range observation.Consumption {
			// The empty string is WALDO's documented whole-index selection, so
			// membership in the pinned corpus paths is the authoritative check.
			if !selected[item.Corpus] || item.TokenTargets <= 0 || seen[item.Corpus] {
				return errors.New("training run observation has invalid corpus consumption evidence")
			}
			seen[item.Corpus] = true
			total += item.TokenTargets
		}
		if len(seen) != len(bom.CorpusBOM.Paths) || total != observation.ConsumedTokens {
			return errors.New("training run observation corpus consumption does not account for selected paths and consumed tokens")
		}
	}
	return nil
}

func validateRunProgress(progress waldoRunProgress, parameters waldoResolvedParameters) error {
	if progress.Steps < 0 || progress.Steps > parameters.Steps || progress.ConsumedTokens < 0 || progress.ConsumedTokens > parameters.PlannedTokenCapacity || progress.LastLoss != nil && (!finite(*progress.LastLoss) || *progress.LastLoss < 0) {
		return errors.New("training run progress is outside the planned range")
	}
	return validateRunEvents(progress.Checkpoints, progress.Evaluations, progress.Steps, progress.ConsumedTokens, map[string]bool{})
}

func validateRunEvents(checkpoints []waldoRunCheckpoint, evaluations []waldoRunEvaluation, maxSteps, maxTokens int64, seenArtifacts map[string]bool) error {
	previous := int64(0)
	for i, checkpoint := range checkpoints {
		if checkpoint.Step <= previous || checkpoint.Step > maxSteps || checkpoint.Tokens < 0 || checkpoint.Tokens > maxTokens || len(checkpoint.Artifacts) == 0 {
			return fmt.Errorf("training checkpoint %d has invalid step, token count, or artifacts", i+1)
		}
		previous = checkpoint.Step
		for j, artifact := range checkpoint.Artifacts {
			if err := validateRunArtifact(fmt.Sprintf("training checkpoint %d artifact %d", i+1, j), artifact); err != nil {
				return err
			}
			if seenArtifacts[artifact.Path] {
				return fmt.Errorf("training run has duplicate artifact path %q", artifact.Path)
			}
			seenArtifacts[artifact.Path] = true
		}
	}
	previous = 0
	for i, evaluation := range evaluations {
		if evaluation.Step <= previous || evaluation.Step > maxSteps || evaluation.Tokens < 0 || evaluation.Tokens > maxTokens || len(evaluation.Metrics) == 0 {
			return fmt.Errorf("training evaluation %d has invalid step, token count, or metrics", i+1)
		}
		previous = evaluation.Step
		for name, value := range evaluation.Metrics {
			if strings.TrimSpace(name) == "" || !finite(value) {
				return fmt.Errorf("training evaluation %d has invalid metric %q", i+1, name)
			}
		}
	}
	return nil
}

func validateRunArtifact(name string, artifact waldoRunArtifact) error {
	if err := validateRelativePath(name+".path", artifact.Path); err != nil {
		return err
	}
	if artifact.Path == "artifacts" || !strings.HasPrefix(artifact.Path, "artifacts/") {
		return fmt.Errorf("%s.path must be beneath artifacts/", name)
	}
	if err := validateSHA256(name+".sha256", artifact.SHA256); err != nil {
		return err
	}
	if artifact.Bytes < 0 {
		return fmt.Errorf("%s.bytes must not be negative", name)
	}
	return nil
}

func validateRunArtifactEvidence(name string, artifact RunArtifactEvidence) error {
	return validateRunArtifact(name, waldoRunArtifact{Path: artifact.Path, SHA256: artifact.SHA256, Bytes: artifact.Bytes})
}

func projectRunArtifact(input waldoRunArtifact) RunArtifactEvidence {
	return RunArtifactEvidence{Path: input.Path, SHA256: input.SHA256, Bytes: input.Bytes}
}

func projectRunArtifacts(input []waldoRunArtifact) []RunArtifactEvidence {
	output := make([]RunArtifactEvidence, 0, len(input))
	for _, artifact := range input {
		output = append(output, projectRunArtifact(artifact))
	}
	sort.Slice(output, func(i, j int) bool { return output[i].Path < output[j].Path })
	return output
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func finitePositive(value float64) bool {
	return finite(value) && value > 0
}

func multiplyPositive(values ...int64) (int64, bool) {
	result := int64(1)
	for _, value := range values {
		if value <= 0 || result > math.MaxInt64/value {
			return 0, true
		}
		result *= value
	}
	return result, false
}
