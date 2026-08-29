package axmmirror

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	TrainingProfileContractSchema = "axm.waldo-witness.training-profile-contract/v0.1"

	ProfileStateWitnessed         = "PROFILE_CONTRACT_WITNESSED"
	ProfileStateLegacyWitnessed   = "LEGACY_ALIAS_WITNESSED"
	ProfileStateUnknownHold       = "HOLD_UNKNOWN_PROFILE_CONTRACT"
	ProfileStateSchemaHold        = "HOLD_PROFILE_SCHEMA_MISMATCH"
	ProfileStateBehaviorHold      = "HOLD_PROFILE_BEHAVIOR_MISMATCH"
	ProfileStateWitnessMismatch   = "HOLD_PARAMETER_WITNESS_MISMATCH"
	maxProjectedProfileWeightRows = 128
)

const (
	profileShuffled       = "causal-pretrain-shuffled"
	profileBalanced       = "causal-pretrain-balanced"
	profileWeighted       = "causal-pretrain-weighted"
	legacyProfileShuffled = "causal-pretrain-v1"
	legacyProfileBalanced = "causal-pretrain-v2"
	legacyProfileWeighted = "causal-pretrain-v3"
)

// CorpusWeightEvidence is a deterministic, bounded projection of one resolved
// corpus weight. The full map remains bound by WeightSetSHA256 even when the
// projection is omitted because it exceeds the public packet limit.
type CorpusWeightEvidence struct {
	Path   string `json:"path"`
	Weight uint64 `json:"weight"`
}

type TrainingDataContract struct {
	Order                    string                 `json:"order"`
	ShuffleBufferRecords     int                    `json:"shuffle_buffer_records"`
	ShuffleBufferBytes       int64                  `json:"shuffle_buffer_bytes"`
	Packing                  string                 `json:"packing"`
	WeightCount              int                    `json:"weight_count"`
	WeightSetSHA256          string                 `json:"weight_set_sha256"`
	WeightProjectionComplete bool                   `json:"weight_projection_complete"`
	Weights                  []CorpusWeightEvidence `json:"weights,omitempty"`
}

type TrainingEvaluationContract struct {
	Selection  string  `json:"selection"`
	Fraction   float64 `json:"fraction"`
	MaxRecords int     `json:"max_records"`
	MaxBytes   int64   `json:"max_bytes"`
}

// TrainingProfileContract exposes the behavior hidden inside a resolved WALDO
// parameter object. It does not infer that a selected or weighted source caused
// any later model output.
type TrainingProfileContract struct {
	Schema                      string                      `json:"schema"`
	State                       string                      `json:"state"`
	RunID                       string                      `json:"run_id"`
	ModelID                     string                      `json:"model_id"`
	RunWitnessState             string                      `json:"run_witness_state"`
	RunBOMSHA256                string                      `json:"run_bom_sha256"`
	RunBOMDocumentSHA256        string                      `json:"run_bom_document_sha256"`
	WitnessRunBOMSHA256         string                      `json:"witness_run_bom_sha256"`
	WitnessRunBOMDocumentSHA256 string                      `json:"witness_run_bom_document_sha256"`
	RunWitnessReceiptSHA256     string                      `json:"run_witness_receipt_sha256"`
	ParameterSHA256             string                      `json:"parameter_sha256"`
	WitnessParameterSHA256      string                      `json:"witness_parameter_sha256"`
	DeclaredProfile             string                      `json:"declared_profile"`
	CanonicalProfile            string                      `json:"canonical_profile,omitempty"`
	LegacyAlias                 bool                        `json:"legacy_alias"`
	ProfileSchema               int                         `json:"profile_schema"`
	Epochs                      int64                       `json:"epochs"`
	RequestedTokens             int64                       `json:"requested_tokens,omitempty"`
	Steps                       int64                       `json:"steps"`
	PlannedTokenCapacity        int64                       `json:"planned_token_capacity"`
	Seed                        uint64                      `json:"seed"`
	Data                        TrainingDataContract        `json:"data"`
	Evaluation                  *TrainingEvaluationContract `json:"evaluation,omitempty"`
	ReceiptSHA256               string                      `json:"receipt_sha256,omitempty"`
	Holds                       []string                    `json:"holds,omitempty"`
	Notices                     []string                    `json:"notices"`
	ClaimCeilings               []string                    `json:"claim_ceilings"`
	Authority                   Authority                   `json:"authority"`
}

type profileBehavior struct {
	Canonical       string
	Legacy          bool
	ExpectedSchema  int
	Order           string
	Selection       string
	RequiresWeights bool
}

// LensTrainingProfile joins the exact RUN-BOM bytes to an already verified run
// witness. Valid but incompatible inputs produce a typed HOLD receipt; malformed
// receipts or malformed WALDO documents are rejected.
func LensTrainingProfile(runBOMData []byte, witness TrainingRunWitness) (TrainingProfileContract, error) {
	if err := witness.Validate(); err != nil {
		return TrainingProfileContract{}, fmt.Errorf("training run witness: %w", err)
	}
	var runBOM waldoRunBOM
	if err := decodeJSON(runBOMData, &runBOM, false); err != nil {
		return TrainingProfileContract{}, fmt.Errorf("decode training run BOM: %w", err)
	}
	if err := validateRunBOM(runBOM); err != nil {
		return TrainingProfileContract{}, err
	}
	canonicalRunBOM, err := json.Marshal(runBOM)
	if err != nil {
		return TrainingProfileContract{}, fmt.Errorf("canonicalize training run BOM: %w", err)
	}
	parameterData, err := json.Marshal(runBOM.Parameters)
	if err != nil {
		return TrainingProfileContract{}, fmt.Errorf("canonicalize training parameters: %w", err)
	}
	weights := sortedCorpusWeights(runBOM.Parameters.Data.CorpusWeights)
	weightData, err := json.Marshal(weights)
	if err != nil {
		return TrainingProfileContract{}, fmt.Errorf("canonicalize training corpus weights: %w", err)
	}
	data := TrainingDataContract{
		Order:                    runBOM.Parameters.Data.Order,
		ShuffleBufferRecords:     runBOM.Parameters.Data.ShuffleBufferRecords,
		ShuffleBufferBytes:       runBOM.Parameters.Data.ShuffleBufferBytes,
		Packing:                  runBOM.Parameters.Data.Packing,
		WeightCount:              len(weights),
		WeightSetSHA256:          digestBytes(weightData),
		WeightProjectionComplete: len(weights) <= maxProjectedProfileWeightRows,
	}
	if data.WeightProjectionComplete {
		data.Weights = weights
	}
	receipt := TrainingProfileContract{
		Schema: TrainingProfileContractSchema, State: ProfileStateWitnessed,
		RunID: runBOM.ID, ModelID: runBOM.ModelID, RunWitnessState: witness.State,
		RunBOMSHA256: digestBytes(canonicalRunBOM), RunBOMDocumentSHA256: digestBytes(runBOMData),
		WitnessRunBOMSHA256: witness.RunBOMSHA256, WitnessRunBOMDocumentSHA256: witness.RunBOMDocumentSHA256,
		RunWitnessReceiptSHA256: witness.ReceiptSHA256, ParameterSHA256: digestBytes(parameterData), WitnessParameterSHA256: witness.Parameters.SHA256,
		DeclaredProfile: runBOM.Parameters.Profile, ProfileSchema: runBOM.Parameters.ProfileSchema,
		Epochs: runBOM.Parameters.Epochs, RequestedTokens: runBOM.Parameters.RequestedTokens,
		Steps: runBOM.Parameters.Steps, PlannedTokenCapacity: runBOM.Parameters.PlannedTokenCapacity,
		Seed: runBOM.Parameters.Seed, Data: data,
		Notices: []string{
			"the contract exposes resolved data-selection behavior; it does not establish source-to-output causality",
			"legacy aliases preserve historical identity while canonical_profile names the equivalent current behavior",
			"receipt digests detect mutation but do not authenticate who produced the receipt",
		},
		ClaimCeilings: []string{
			"the receipt witnesses a declared training-selection contract, not the quality or safety of the resulting model",
			"corpus selection or weight does not prove that a source caused a particular output",
			"recorded license assertions are not legal conclusions",
		},
		Authority: Authority{},
	}
	if policy := runBOM.Parameters.Evaluation; policy != nil {
		receipt.Evaluation = &TrainingEvaluationContract{
			Selection: policy.Selection, Fraction: policy.Fraction, MaxRecords: policy.MaxRecords, MaxBytes: policy.MaxBytes,
		}
	}

	behavior, known := resolveProfileBehavior(runBOM.Parameters.Profile)
	if known {
		receipt.CanonicalProfile = behavior.Canonical
		receipt.LegacyAlias = behavior.Legacy
	}

	witnessMatches := witness.RunID == runBOM.ID && witness.ModelID == runBOM.ModelID &&
		witness.State == receipt.RunWitnessState &&
		witness.RunBOMSHA256 == receipt.RunBOMSHA256 &&
		witness.RunBOMDocumentSHA256 == receipt.RunBOMDocumentSHA256 &&
		witness.Parameters.SHA256 == receipt.ParameterSHA256 &&
		witness.Parameters.Profile == runBOM.Parameters.Profile &&
		witness.Parameters.ProfileSchema == runBOM.Parameters.ProfileSchema &&
		witness.Parameters.Epochs == runBOM.Parameters.Epochs &&
		witness.Parameters.Steps == runBOM.Parameters.Steps &&
		witness.Parameters.PlannedTokenCapacity == runBOM.Parameters.PlannedTokenCapacity &&
		witness.Parameters.Seed == runBOM.Parameters.Seed
	if !witnessMatches {
		receipt.State = ProfileStateWitnessMismatch
		receipt.Holds = append(receipt.Holds, "the supplied run witness does not bind the exact run BOM and resolved parameters")
	} else if !known {
		receipt.State = ProfileStateUnknownHold
		receipt.Holds = append(receipt.Holds, fmt.Sprintf("profile %q has no witnessed WALDO behavior contract", runBOM.Parameters.Profile))
	} else if runBOM.Parameters.ProfileSchema != behavior.ExpectedSchema {
		receipt.State = ProfileStateSchemaHold
		receipt.Holds = append(receipt.Holds, fmt.Sprintf("profile %q schema is %d; expected %d", runBOM.Parameters.Profile, runBOM.Parameters.ProfileSchema, behavior.ExpectedSchema))
	} else {
		receipt.Holds = profileBehaviorHolds(runBOM, behavior)
		if len(receipt.Holds) != 0 {
			receipt.State = ProfileStateBehaviorHold
		} else if behavior.Legacy {
			receipt.State = ProfileStateLegacyWitnessed
		}
	}

	receipt.ReceiptSHA256, err = trainingProfileReceiptDigest(receipt)
	if err != nil {
		return TrainingProfileContract{}, err
	}
	if err := receipt.Validate(); err != nil {
		return TrainingProfileContract{}, fmt.Errorf("generated training profile contract: %w", err)
	}
	return receipt, nil
}

// WitnessTrainingProfileContract preserves the late v0.1 PR entry point while
// routing it through the later gated profile lens used by the integrated stack.
func WitnessTrainingProfileContract(runBOMData []byte, witness TrainingRunWitness) (TrainingProfileContract, error) {
	return LensTrainingProfile(runBOMData, witness)
}

func (receipt TrainingProfileContract) Validate() error {
	if receipt.Schema != TrainingProfileContractSchema || !oneOf(receipt.State,
		ProfileStateWitnessed, ProfileStateLegacyWitnessed, ProfileStateUnknownHold,
		ProfileStateSchemaHold, ProfileStateBehaviorHold, ProfileStateWitnessMismatch) {
		return fmt.Errorf("unsupported training profile contract identity %q state %q", receipt.Schema, receipt.State)
	}
	if strings.TrimSpace(receipt.RunID) == "" || strings.TrimSpace(receipt.ModelID) == "" || strings.TrimSpace(receipt.DeclaredProfile) == "" || receipt.ProfileSchema < 1 {
		return errors.New("training profile contract has incomplete run or profile identity")
	}
	if !oneOf(receipt.RunWitnessState, RunWitnessStateReady, RunWitnessStateHold) {
		return fmt.Errorf("training profile contract has unsupported run witness state %q", receipt.RunWitnessState)
	}
	for _, item := range []struct{ name, value string }{
		{"profile contract.run_bom_sha256", receipt.RunBOMSHA256},
		{"profile contract.run_bom_document_sha256", receipt.RunBOMDocumentSHA256},
		{"profile contract.witness_run_bom_sha256", receipt.WitnessRunBOMSHA256},
		{"profile contract.witness_run_bom_document_sha256", receipt.WitnessRunBOMDocumentSHA256},
		{"profile contract.run_witness_receipt_sha256", receipt.RunWitnessReceiptSHA256},
		{"profile contract.parameter_sha256", receipt.ParameterSHA256},
		{"profile contract.witness_parameter_sha256", receipt.WitnessParameterSHA256},
		{"profile contract.data.weight_set_sha256", receipt.Data.WeightSetSHA256},
		{"profile contract.receipt_sha256", receipt.ReceiptSHA256},
	} {
		if err := validateSHA256(item.name, item.value); err != nil {
			return err
		}
	}
	if receipt.State != ProfileStateWitnessMismatch && (receipt.RunBOMSHA256 != receipt.WitnessRunBOMSHA256 || receipt.RunBOMDocumentSHA256 != receipt.WitnessRunBOMDocumentSHA256 || receipt.ParameterSHA256 != receipt.WitnessParameterSHA256) {
		return errors.New("training profile contract witness digests do not match the projected run BOM")
	}
	if receipt.Epochs < 1 || receipt.Steps < 1 || receipt.PlannedTokenCapacity < 1 || receipt.RequestedTokens < 0 || receipt.RequestedTokens > receipt.PlannedTokenCapacity {
		return errors.New("training profile contract has invalid budget evidence")
	}
	if strings.TrimSpace(receipt.Data.Order) == "" || strings.TrimSpace(receipt.Data.Packing) == "" || receipt.Data.ShuffleBufferRecords < 1 || receipt.Data.ShuffleBufferBytes < 1 || receipt.Data.WeightCount < 0 {
		return errors.New("training profile contract has incomplete data behavior")
	}
	if receipt.Data.WeightProjectionComplete != (receipt.Data.WeightCount <= maxProjectedProfileWeightRows) {
		return errors.New("training profile contract weight projection completeness is inconsistent")
	}
	if receipt.Data.WeightProjectionComplete && len(receipt.Data.Weights) != receipt.Data.WeightCount {
		return errors.New("training profile contract complete weight projection has the wrong count")
	}
	if !receipt.Data.WeightProjectionComplete && len(receipt.Data.Weights) != 0 {
		return errors.New("training profile contract oversized weight projection must be omitted")
	}
	if !sort.SliceIsSorted(receipt.Data.Weights, func(i, j int) bool { return receipt.Data.Weights[i].Path < receipt.Data.Weights[j].Path }) {
		return errors.New("training profile contract weights are not sorted")
	}
	for i, weight := range receipt.Data.Weights {
		if strings.TrimSpace(weight.Path) == "" || weight.Weight == 0 {
			return fmt.Errorf("training profile contract weight %d is invalid", i)
		}
		if i > 0 && receipt.Data.Weights[i-1].Path == weight.Path {
			return fmt.Errorf("training profile contract has duplicate weight path %q", weight.Path)
		}
	}
	if receipt.Data.WeightProjectionComplete {
		weights := receipt.Data.Weights
		if weights == nil && receipt.Data.WeightCount == 0 {
			weights = []CorpusWeightEvidence{}
		}
		weightData, err := json.Marshal(weights)
		if err != nil {
			return fmt.Errorf("encode training profile contract weights: %w", err)
		}
		if digestBytes(weightData) != receipt.Data.WeightSetSHA256 {
			return errors.New("training profile contract weight projection does not match its set digest")
		}
	}
	if receipt.Evaluation != nil {
		if strings.TrimSpace(receipt.Evaluation.Selection) == "" || !finite(receipt.Evaluation.Fraction) || receipt.Evaluation.Fraction < 0 || receipt.Evaluation.Fraction >= 1 || receipt.Evaluation.MaxRecords < 0 || receipt.Evaluation.MaxBytes < 0 {
			return errors.New("training profile contract has invalid evaluation behavior")
		}
	}
	behavior, known := resolveProfileBehavior(receipt.DeclaredProfile)
	if known {
		if receipt.CanonicalProfile != behavior.Canonical || receipt.LegacyAlias != behavior.Legacy {
			return errors.New("training profile contract canonical profile projection is inconsistent")
		}
	} else if receipt.CanonicalProfile != "" || receipt.LegacyAlias {
		return errors.New("unknown training profile contract must not invent a canonical profile")
	}
	isHold := !oneOf(receipt.State, ProfileStateWitnessed, ProfileStateLegacyWitnessed)
	if isHold != (len(receipt.Holds) > 0) {
		return errors.New("training profile contract HOLD state and reasons are inconsistent")
	}
	if len(receipt.Notices) == 0 || len(receipt.ClaimCeilings) == 0 || !receipt.Authority.closed() {
		return errors.New("training profile contract must state its boundaries and carry closed authority")
	}
	actual, err := trainingProfileReceiptDigest(receipt)
	if err != nil {
		return err
	}
	if actual != receipt.ReceiptSHA256 {
		return fmt.Errorf("training profile contract receipt digest mismatch: expected %s, got %s", receipt.ReceiptSHA256, actual)
	}
	return nil
}

func resolveProfileBehavior(profile string) (profileBehavior, bool) {
	switch profile {
	case profileShuffled:
		return profileBehavior{Canonical: profileShuffled, ExpectedSchema: 1, Order: "bounded-shuffle-v1", Selection: "lowest-sha256-v1"}, true
	case profileBalanced:
		return profileBehavior{Canonical: profileBalanced, ExpectedSchema: 1, Order: "corpus-balanced-shuffle-v1", Selection: "stratified-lowest-sha256-v1"}, true
	case profileWeighted:
		return profileBehavior{Canonical: profileWeighted, ExpectedSchema: 1, Order: "corpus-weighted-shuffle-v1", Selection: "stratified-lowest-sha256-v1", RequiresWeights: true}, true
	case legacyProfileShuffled:
		return profileBehavior{Canonical: profileShuffled, Legacy: true, ExpectedSchema: 1, Order: "bounded-shuffle-v1", Selection: "lowest-sha256-v1"}, true
	case legacyProfileBalanced:
		return profileBehavior{Canonical: profileBalanced, Legacy: true, ExpectedSchema: 2, Order: "corpus-balanced-shuffle-v1", Selection: "stratified-lowest-sha256-v1"}, true
	case legacyProfileWeighted:
		return profileBehavior{Canonical: profileWeighted, Legacy: true, ExpectedSchema: 3, Order: "corpus-weighted-shuffle-v1", Selection: "stratified-lowest-sha256-v1", RequiresWeights: true}, true
	default:
		return profileBehavior{}, false
	}
}

func profileBehaviorHolds(runBOM waldoRunBOM, behavior profileBehavior) []string {
	var holds []string
	parameters := runBOM.Parameters
	if parameters.Data.Order != behavior.Order {
		holds = append(holds, fmt.Sprintf("data order is %q; profile requires %q", parameters.Data.Order, behavior.Order))
	}
	if parameters.Data.Packing != "continuous-eos-v1" {
		holds = append(holds, fmt.Sprintf("packing is %q; profile requires %q", parameters.Data.Packing, "continuous-eos-v1"))
	}
	if parameters.Evaluation == nil {
		holds = append(holds, "resolved evaluation policy is missing")
	} else if parameters.Evaluation.Selection != behavior.Selection {
		holds = append(holds, fmt.Sprintf("evaluation selection is %q; profile requires %q", parameters.Evaluation.Selection, behavior.Selection))
	}
	weights := parameters.Data.CorpusWeights
	if behavior.RequiresWeights {
		selected := make(map[string]struct{}, len(runBOM.CorpusBOM.Paths))
		for _, path := range runBOM.CorpusBOM.Paths {
			selected[path] = struct{}{}
			if weights[path] == 0 {
				holds = append(holds, fmt.Sprintf("weighted profile has no positive weight for selected corpus %q", path))
			}
		}
		for path := range weights {
			if _, ok := selected[path]; !ok {
				holds = append(holds, fmt.Sprintf("weighted profile declares unselected corpus %q", path))
			}
		}
	} else if len(weights) != 0 {
		holds = append(holds, "non-weighted profile carries corpus weights")
	}
	return holds
}

func sortedCorpusWeights(input map[string]uint64) []CorpusWeightEvidence {
	result := make([]CorpusWeightEvidence, 0, len(input))
	for path, weight := range input {
		result = append(result, CorpusWeightEvidence{Path: path, Weight: weight})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func trainingProfileReceiptDigest(receipt TrainingProfileContract) (string, error) {
	receipt.ReceiptSHA256 = ""
	data, err := json.Marshal(receipt)
	if err != nil {
		return "", fmt.Errorf("encode training profile contract receipt: %w", err)
	}
	return digestBytes(data), nil
}
