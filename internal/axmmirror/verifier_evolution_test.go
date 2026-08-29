package axmmirror

import "testing"

func TestVerifierChangeRequiresIndependentMutualReview(t *testing.T) {
	registry := verifierTestRegistry(t)
	request := verifierTestChangeRequest(t, registry, "privacy-boundary", VerifierPeerPass)

	receipt, err := AssessVerifierChange(registry, request)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != VerifierChangeReady || !receipt.MaterializationAllowed || receipt.CandidateRegistry == nil {
		t.Fatalf("receipt = %+v", receipt)
	}
	next, err := MaterializeVerifierChange(registry, receipt)
	if err != nil {
		t.Fatal(err)
	}
	if next.Generation != registry.Generation+1 || next.PreviousRegistrySHA256 != registry.RegistrySHA256 || next.RetainedRegistrySHA256[0] != registry.RegistrySHA256 {
		t.Fatalf("next registry = %+v", next)
	}
	changed, ok := findVerifier(next.Verifiers, "privacy-boundary")
	if !ok || changed.Version != "v0.2" {
		t.Fatalf("changed verifier = %+v, found = %v", changed, ok)
	}
}

func TestVerifierCannotApproveOwnChange(t *testing.T) {
	registry := verifierTestRegistry(t)
	request := verifierTestChangeRequest(t, registry, "privacy-boundary", VerifierPeerPass)
	target, _ := findVerifier(registry.Verifiers, "privacy-boundary")
	targetDigest, _ := verifierDefinitionDigest(target)
	request.PeerReviews[0].ReviewerID = target.ID
	request.PeerReviews[0].ReviewerVersion = target.Version
	request.PeerReviews[0].ReviewerDefinitionSHA256 = targetDigest
	request.PeerReviews[0].ReviewerIndependenceGroup = target.IndependenceGroup

	receipt, err := AssessVerifierChange(registry, request)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != VerifierChangeRollbackSelf || !receipt.RollbackRequired || receipt.CandidateRegistry != nil || receipt.RollbackTargetSHA256 != registry.RegistrySHA256 {
		t.Fatalf("receipt = %+v", receipt)
	}
	if _, err := MaterializeVerifierChange(registry, receipt); err == nil {
		t.Fatal("self-approved change materialized")
	}
}

func TestVerifierPeerRegressionRollsBackToCurrentGeneration(t *testing.T) {
	registry := verifierTestRegistry(t)
	request := verifierTestChangeRequest(t, registry, "privacy-boundary", VerifierPeerPass)
	request.PeerReviews[0].State = VerifierPeerFail
	request.PeerReviews[0].RegressionCount = 1
	request.PeerReviews[0].Finding = "candidate changed a previously accepted privacy decision"

	receipt, err := AssessVerifierChange(registry, request)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != VerifierChangeRollbackPeerReject || !receipt.RollbackRequired || receipt.ActiveRegistrySHA256 != registry.RegistrySHA256 || receipt.CandidateRegistry != nil {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestVerifierChangeHoldsWithoutTwoIndependentPeers(t *testing.T) {
	registry := verifierTestRegistry(t)
	request := verifierTestChangeRequest(t, registry, "privacy-boundary", VerifierPeerPass)
	request.PeerReviews = request.PeerReviews[:1]

	receipt, err := AssessVerifierChange(registry, request)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != VerifierChangeHoldQuorum || receipt.RollbackRequired || receipt.MaterializationAllowed {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestProtectedVerifierStaysAtMergeGateAfterPeerPass(t *testing.T) {
	registry := verifierTestRegistry(t)
	request := verifierTestChangeRequest(t, registry, "schema-guard", VerifierPeerPass)

	receipt, err := AssessVerifierChange(registry, request)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != VerifierChangeHoldMergeGate || receipt.MaterializationAllowed || receipt.CandidateRegistry != nil {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestRepairBuddyContainsFailureAndRequiresFullReentry(t *testing.T) {
	registry := verifierTestRegistry(t)
	request := verifierTestChangeRequest(t, registry, "privacy-boundary", VerifierPeerPass)
	request.PeerReviews[0].State = VerifierPeerFail
	request.PeerReviews[0].RegressionCount = 1
	request.PeerReviews[0].Finding = "candidate regression"
	failed, err := AssessVerifierChange(registry, request)
	if err != nil {
		t.Fatal(err)
	}
	repairRequest := RepairBuddyRequest{
		Schema: RepairBuddyRequestSchema, RepairID: "repair-privacy-0001",
		FailedChangeReceiptSHA256: failed.ReceiptSHA256, IncidentEvidenceSHA256: verifierRepeat("e"),
		ObservedAt: "2026-08-16T07:00:00Z", MaxAttempts: 2, Authority: Authority{},
	}
	plan, err := PlanVerifierRepair(registry, failed, repairRequest)
	if err != nil {
		t.Fatal(err)
	}
	if plan.State != RepairBuddyCandidateReady || plan.Disposition != "REVISE_OR_WITHDRAW_CANDIDATE" || !plan.BaselineRetained || !plan.CandidateOnly || plan.AutomaticRepair || !plan.HumanReviewRequired {
		t.Fatalf("plan = %+v", plan)
	}
	if len(plan.Stages) != 10 || plan.Stages[7].Stage != "REPAIR" || plan.Stages[7].State != RepairStageBlocked {
		t.Fatalf("repair stages = %+v", plan.Stages)
	}
}

func TestRepairBuddyDoesNotRepairReadyChange(t *testing.T) {
	registry := verifierTestRegistry(t)
	ready, err := AssessVerifierChange(registry, verifierTestChangeRequest(t, registry, "privacy-boundary", VerifierPeerPass))
	if err != nil {
		t.Fatal(err)
	}
	request := RepairBuddyRequest{
		Schema: RepairBuddyRequestSchema, RepairID: "repair-not-needed-0001",
		FailedChangeReceiptSHA256: ready.ReceiptSHA256, IncidentEvidenceSHA256: verifierRepeat("e"),
		ObservedAt: "2026-08-16T07:00:00Z", MaxAttempts: 1, Authority: Authority{},
	}
	plan, err := PlanVerifierRepair(registry, ready, request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.State != RepairBuddyNotNeededHold || plan.Disposition != "NO_REPAIR_CANDIDATE" {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestVerifierRegistryAndReceiptDetectMutation(t *testing.T) {
	registry := verifierTestRegistry(t)
	mutatedRegistry := registry
	mutatedRegistry.Verifiers = append([]VerifierDefinition(nil), registry.Verifiers...)
	mutatedRegistry.Verifiers[0].Version = "v9"
	if err := mutatedRegistry.Validate(); err == nil {
		t.Fatal("verifier registry accepted a mutated definition")
	}
	receipt, err := AssessVerifierChange(registry, verifierTestChangeRequest(t, registry, "privacy-boundary", VerifierPeerPass))
	if err != nil {
		t.Fatal(err)
	}
	receipt.Checks[0].Detail = "mutated"
	if err := receipt.Validate(); err == nil {
		t.Fatal("verifier change receipt accepted a mutated check")
	}
}

func verifierTestRegistry(t *testing.T) VerifierRegistry {
	t.Helper()
	registry, err := NewVerifierRegistry("waldo-clone-verifiers", []VerifierDefinition{
		{ID: "schema-guard", Version: "v0.1", Class: VerifierClassSchemaInvariant, IndependenceGroup: "kernel", DefinitionSHA256: verifierRepeat("1"), TestPackSHA256: verifierRepeat("2"), ImplementationSHA256: verifierRepeat("3"), Protected: true},
		{ID: "provenance-peer", Version: "v0.1", Class: VerifierClassProvenanceBinding, IndependenceGroup: "provenance", DefinitionSHA256: verifierRepeat("4"), TestPackSHA256: verifierRepeat("5"), ImplementationSHA256: verifierRepeat("6")},
		{ID: "authority-peer", Version: "v0.1", Class: VerifierClassAuthority, IndependenceGroup: "authority", DefinitionSHA256: verifierRepeat("7"), TestPackSHA256: verifierRepeat("8"), ImplementationSHA256: verifierRepeat("9")},
		{ID: "privacy-boundary", Version: "v0.1", Class: VerifierClassPrivacyBoundary, IndependenceGroup: "privacy", DefinitionSHA256: verifierRepeat("a"), TestPackSHA256: verifierRepeat("b"), ImplementationSHA256: verifierRepeat("c")},
	})
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func verifierTestChangeRequest(t *testing.T, registry VerifierRegistry, targetID, peerState string) VerifierChangeRequest {
	t.Helper()
	target, ok := findVerifier(registry.Verifiers, targetID)
	if !ok {
		t.Fatalf("missing target verifier %q", targetID)
	}
	baselineDigest, _ := verifierDefinitionDigest(target)
	candidate := target
	candidate.Version = "v0.2"
	candidate.DefinitionSHA256 = verifierRepeat("d")
	intent, err := SealVerifierChangeIntent(VerifierChangeIntent{
		IntentID: "change-" + targetID + "-0001", TargetVerifierID: targetID,
		BaselineDefinitionSHA256: baselineDigest, Candidate: candidate,
		Reason: "tighten the verifier using independently replayable evidence", SourceEvidenceSHA256: verifierRepeat("e"),
		ShadowProtocol: VerifierShadowProtocol{ProtocolID: "shadow-replay-v1", FixtureSetSHA256: verifierRepeat("f"), CaseCount: 32},
		Authority:      Authority{},
	})
	if err != nil {
		t.Fatal(err)
	}
	reviewerIDs := []string{"authority-peer", "provenance-peer"}
	if targetID == "schema-guard" {
		reviewerIDs = []string{"authority-peer", "privacy-boundary"}
	}
	reviews := make([]VerifierPeerReview, 0, 2)
	for i, reviewerID := range reviewerIDs {
		reviewer, _ := findVerifier(registry.Verifiers, reviewerID)
		reviewerDigest, _ := verifierDefinitionDigest(reviewer)
		reviews = append(reviews, VerifierPeerReview{
			ReviewerID: reviewer.ID, ReviewerVersion: reviewer.Version, ReviewerDefinitionSHA256: reviewerDigest,
			ReviewerIndependenceGroup: reviewer.IndependenceGroup, IntentSHA256: intent.IntentSHA256,
			ProtocolID: intent.ShadowProtocol.ProtocolID, FixtureSetSHA256: intent.ShadowProtocol.FixtureSetSHA256,
			BaselineResultSetSHA256: verifierRepeat("1"), CandidateResultSetSHA256: verifierRepeat("2"),
			EvidenceSHA256: verifierRepeat(string(rune('3' + i))), State: peerState,
			Finding: "frozen replay completed against the exact candidate", Authority: Authority{},
		})
	}
	return VerifierChangeRequest{
		Schema: VerifierChangeRequestSchema, ChangeID: "request-" + targetID + "-0001",
		CurrentRegistrySHA256: registry.RegistrySHA256, Intent: intent, PeerReviews: reviews, Authority: Authority{},
	}
}

func verifierRepeat(value string) string {
	result := ""
	for len(result) < 64 {
		result += value
	}
	return result[:64]
}
