package axmmirror

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifierEvolutionAndRepairFixturesPinInitialReceipts(t *testing.T) {
	examples := filepath.Join("..", "..", "examples", "axm-mirror")
	var registry VerifierRegistry
	readV07Fixture(t, filepath.Join(examples, "verifier-registry.json"), &registry)
	if err := registry.Validate(); err != nil {
		t.Fatal(err)
	}
	if registry.RegistrySHA256 != "6e689080591bdb6f07d2b644bffff27b7002fb33d73ee33cba0a041d2fdf446d" {
		t.Fatalf("verifier registry digest = %s", registry.RegistrySHA256)
	}

	var readyRequest VerifierChangeRequest
	readV07Fixture(t, filepath.Join(examples, "verifier-change-request.json"), &readyRequest)
	ready, err := AssessVerifierChange(registry, readyRequest)
	if err != nil {
		t.Fatal(err)
	}
	if ready.State != VerifierChangeReady || ready.ReceiptSHA256 != "0d267b72b2b78dbcb8879e95499ad88df6d26a206c188c6f69292786f5a9fc0e" || ready.CandidateRegistry == nil || ready.CandidateRegistry.RegistrySHA256 != "5e9d4bf45c8442932ee66af1b55439709679c948ee6d9f70d84acaf834113c44" {
		t.Fatalf("ready verifier change = %s %s %+v", ready.State, ready.ReceiptSHA256, ready.CandidateRegistry)
	}

	var regressionRequest VerifierChangeRequest
	readV07Fixture(t, filepath.Join(examples, "verifier-change-regression-request.json"), &regressionRequest)
	failed, err := AssessVerifierChange(registry, regressionRequest)
	if err != nil {
		t.Fatal(err)
	}
	if failed.State != VerifierChangeRollbackPeerReject || failed.ReceiptSHA256 != "f77e838904f61e50f2342391f77a320fa1ce95cea78dc7f77c8f47c75872e45f" || failed.CandidateRegistry != nil {
		t.Fatalf("failed verifier change = %s %s", failed.State, failed.ReceiptSHA256)
	}

	var repairRequest RepairBuddyRequest
	readV07Fixture(t, filepath.Join(examples, "repair-buddy-request.json"), &repairRequest)
	plan, err := PlanVerifierRepair(registry, failed, repairRequest)
	if err != nil {
		t.Fatal(err)
	}
	if plan.State != RepairBuddyCandidateReady || plan.PlanSHA256 != "d460b045dd56724147475428ebefdb3cd6f6bdcd53d82c0b23c055168dc3ffba" {
		t.Fatalf("Repair Buddy plan = %s %s", plan.State, plan.PlanSHA256)
	}
}

func TestIdentityToolMemoryFixturesPinSelectiveWisdom(t *testing.T) {
	examples := filepath.Join("..", "..", "examples", "axm-mirror")
	var draft IdentityToolExperience
	readV07Fixture(t, filepath.Join(examples, "identity-tool-experience-draft.json"), &draft)
	sealed, err := SealIdentityToolExperience(draft)
	if err != nil {
		t.Fatal(err)
	}
	if sealed.ExperienceSHA256 != "e25494f9e625a8e34dee0508f6fbb8b30e8846df8372db185d1aab4310bc0e6f" {
		t.Fatalf("identity tool experience digest = %s", sealed.ExperienceSHA256)
	}
	var checkedExperience IdentityToolExperience
	readV07Fixture(t, filepath.Join(examples, "identity-tool-experience.json"), &checkedExperience)
	if checkedExperience.ExperienceSHA256 != sealed.ExperienceSHA256 {
		t.Fatal("checked identity tool experience differs from sealed draft")
	}
	shard, err := StartIdentityToolMemory(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if shard.ShardSHA256 != "3b01992e91bb77c3491a634ed4376be29066169623ba24f18f073c1f471b4ff1" {
		t.Fatalf("identity tool memory shard digest = %s", shard.ShardSHA256)
	}
	var checkedShard IdentityToolMemoryShard
	readV07Fixture(t, filepath.Join(examples, "identity-tool-memory-shard.json"), &checkedShard)
	if err := checkedShard.Validate(); err != nil || checkedShard.ShardSHA256 != shard.ShardSHA256 {
		t.Fatalf("checked identity tool memory shard = %s, error = %v", checkedShard.ShardSHA256, err)
	}
	var query IdentityWisdomQuery
	readV07Fixture(t, filepath.Join(examples, "identity-wisdom-query.json"), &query)
	view, err := RecallIdentityToolWisdom(shard, query)
	if err != nil {
		t.Fatal(err)
	}
	if view.State != ToolWisdomReady || view.ViewSHA256 != "86cc6bdd9ee0a9938978c450c484ca6e3c345b5c624381cbdc977e5b6d2130c8" || len(view.Selected) != 1 {
		t.Fatalf("identity wisdom view = %s %s %+v", view.State, view.ViewSHA256, view.Selected)
	}
	var checkedView IdentityWisdomView
	readV07Fixture(t, filepath.Join(examples, "identity-wisdom-view.json"), &checkedView)
	if err := checkedView.Validate(); err != nil || checkedView.ViewSHA256 != view.ViewSHA256 {
		t.Fatalf("checked identity wisdom view = %s, error = %v", checkedView.ViewSHA256, err)
	}
}

func readV07Fixture(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := DecodeStrictJSON(data, target); err != nil {
		t.Fatal(err)
	}
}
