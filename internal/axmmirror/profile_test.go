package axmmirror

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTrainingProfileContractWitnessesCurrentBehaviorName(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	bom.Parameters.Profile = profileShuffled
	repinRun(t, bom, &run)
	receipt := profileContractFixture(t, bom, run)
	if receipt.State != ProfileStateWitnessed || receipt.DeclaredProfile != profileShuffled || receipt.CanonicalProfile != profileShuffled || receipt.LegacyAlias {
		t.Fatalf("profile contract = %+v", receipt)
	}
}

func TestTrainingProfileContractPreservesLegacyAlias(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	receipt := profileContractFixture(t, bom, run)
	if receipt.State != ProfileStateLegacyWitnessed || receipt.CanonicalProfile != profileShuffled || !receipt.LegacyAlias {
		t.Fatalf("profile contract = %+v", receipt)
	}
}

func TestTrainingProfileContractWitnessesBalancedAndHistoricalWeightedProfiles(t *testing.T) {
	tests := []struct {
		name      string
		declared  string
		schema    int
		canonical string
		legacy    bool
		weighted  bool
	}{
		{name: "current balanced", declared: profileBalanced, schema: 1, canonical: profileBalanced},
		{name: "legacy balanced", declared: legacyProfileBalanced, schema: 2, canonical: profileBalanced, legacy: true},
		{name: "legacy weighted", declared: legacyProfileWeighted, schema: 3, canonical: profileWeighted, legacy: true, weighted: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bom, run := validRunFixture(t, "pytorch", false, "complete")
			configureSingleNamedCorpus(t, &bom, &run)
			bom.Parameters.Profile = test.declared
			bom.Parameters.ProfileSchema = test.schema
			bom.Parameters.Data.Order = "corpus-balanced-shuffle-v1"
			bom.Parameters.Evaluation.Selection = "stratified-lowest-sha256-v1"
			bom.EvaluationSet.Selection = "stratified-lowest-sha256-v1"
			if test.weighted {
				bom.Parameters.Data.Order = "corpus-weighted-shuffle-v1"
				bom.Parameters.Data.CorpusWeights = map[string]uint64{"core/example": 2}
			}
			repinRun(t, bom, &run)
			receipt := profileContractFixture(t, bom, run)
			wantState := ProfileStateWitnessed
			if test.legacy {
				wantState = ProfileStateLegacyWitnessed
			}
			if receipt.State != wantState || receipt.CanonicalProfile != test.canonical || receipt.LegacyAlias != test.legacy {
				t.Fatalf("profile contract = %+v", receipt)
			}
		})
	}
}

func TestTrainingProfileContractWitnessesWeightedBehavior(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	configureSingleNamedCorpus(t, &bom, &run)
	bom.Parameters.Profile = profileWeighted
	bom.Parameters.ProfileSchema = 1
	bom.Parameters.Data.Order = "corpus-weighted-shuffle-v1"
	bom.Parameters.Data.CorpusWeights = map[string]uint64{"core/example": 3}
	bom.Parameters.Evaluation.Selection = "stratified-lowest-sha256-v1"
	bom.EvaluationSet.Selection = "stratified-lowest-sha256-v1"
	repinRun(t, bom, &run)
	receipt := profileContractFixture(t, bom, run)
	if receipt.State != ProfileStateWitnessed || receipt.Data.WeightCount != 1 || len(receipt.Data.Weights) != 1 || receipt.Data.Weights[0].Weight != 3 {
		t.Fatalf("profile contract = %+v", receipt)
	}
}

func TestTrainingProfileContractHoldsUnknownProfile(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	bom.Parameters.Profile = "causal-pretrain-mystery"
	repinRun(t, bom, &run)
	receipt := profileContractFixture(t, bom, run)
	if receipt.State != ProfileStateUnknownHold || len(receipt.Holds) == 0 {
		t.Fatalf("profile contract = %+v", receipt)
	}
}

func TestTrainingProfileContractHoldsSchemaMismatch(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	configureSingleNamedCorpus(t, &bom, &run)
	bom.Parameters.Profile = profileBalanced
	bom.Parameters.ProfileSchema = 2
	bom.Parameters.Data.Order = "corpus-balanced-shuffle-v1"
	bom.Parameters.Evaluation.Selection = "stratified-lowest-sha256-v1"
	bom.EvaluationSet.Selection = "stratified-lowest-sha256-v1"
	repinRun(t, bom, &run)
	receipt := profileContractFixture(t, bom, run)
	if receipt.State != ProfileStateSchemaHold {
		t.Fatalf("profile contract state = %q", receipt.State)
	}
}

func TestTrainingProfileContractHoldsBehaviorMismatch(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	bom.Parameters.Profile = profileBalanced
	bom.Parameters.ProfileSchema = 1
	repinRun(t, bom, &run)
	receipt := profileContractFixture(t, bom, run)
	if receipt.State != ProfileStateBehaviorHold || !strings.Contains(strings.Join(receipt.Holds, " "), "data order") {
		t.Fatalf("profile contract = %+v", receipt)
	}
}

func TestTrainingProfileContractHoldsDifferentValidWitness(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	bom.Parameters.Profile = profileShuffled
	repinRun(t, bom, &run)
	otherBOM, otherRun := validRunFixture(t, "pytorch", false, "complete")
	otherBOM.ID = "run-0002"
	otherRun.ID = otherBOM.ID
	repinRun(t, otherBOM, &otherRun)
	witness := witnessFixture(t, otherBOM, otherRun)
	bomData, err := json.MarshalIndent(bom, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := LensTrainingProfile(bomData, witness)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != ProfileStateWitnessMismatch {
		t.Fatalf("profile contract state = %q", receipt.State)
	}
}

func TestTrainingProfileContractDetectsReceiptTamper(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	receipt := profileContractFixture(t, bom, run)
	receipt.Data.Order = "tampered"
	if err := receipt.Validate(); err == nil {
		t.Fatal("tampered profile contract validated")
	}
}

func TestTrainingProfileContractBindsLateV01BudgetAndWitnessFields(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	bom.Parameters.RequestedTokens = 7
	repinRun(t, bom, &run)
	bomData, err := json.MarshalIndent(bom, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	witness := witnessFixture(t, bom, run)
	receipt, err := WitnessTrainingProfileContract(bomData, witness)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ModelID != bom.ModelID || receipt.RunWitnessState != witness.State || receipt.RequestedTokens != 7 || receipt.Epochs != 1 || receipt.Steps != 2 || receipt.PlannedTokenCapacity != 8 {
		t.Fatalf("profile contract budget/witness projection = %+v", receipt)
	}
	if receipt.WitnessRunBOMSHA256 != receipt.RunBOMSHA256 || receipt.WitnessParameterSHA256 != receipt.ParameterSHA256 || len(receipt.ClaimCeilings) == 0 {
		t.Fatalf("profile contract witness bindings = %+v", receipt)
	}
}

func profileContractFixture(t *testing.T, bom waldoRunBOM, run waldoRunRecord) TrainingProfileContract {
	t.Helper()
	bomData, err := json.MarshalIndent(bom, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	runData, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	witness, err := WitnessTrainingRun(bomData, runData)
	if err != nil {
		t.Fatalf("WitnessTrainingRun() error = %v", err)
	}
	receipt, err := LensTrainingProfile(bomData, witness)
	if err != nil {
		t.Fatalf("LensTrainingProfile() error = %v", err)
	}
	return receipt
}

func configureSingleNamedCorpus(t *testing.T, bom *waldoRunBOM, run *waldoRunRecord) {
	t.Helper()
	bom.CorpusBOM.Paths = []string{"core/example"}
	corpusData, err := json.Marshal(bom.CorpusBOM)
	if err != nil {
		t.Fatal(err)
	}
	bom.CorpusBOMSHA256 = digestBytes(corpusData)
	if run.Observation != nil {
		run.Observation.Consumption = []waldoCorpusConsumption{{Corpus: "core/example", TokenTargets: run.Observation.ConsumedTokens}}
	}
}
