package axmmirror

import (
	"encoding/json"
	"testing"
)

func TestSourceClaimGateConfirmsOnlyWithinContract(t *testing.T) {
	packet := readyContextPacket(t)
	identity := contextStringFact(t, packet, "answering.identity_sha256")
	profile := contextStringFact(t, packet, "training.profile.canonical")
	receipt, err := AssessSourceClaims(SourceClaimSubmission{
		Schema: SourceClaimSubmissionSchema, SubmissionID: "claims-0001",
		OutputSHA256: repeatHex("a"), ContextPacketSHA256: packet.PacketSHA256,
		Claims: []SourceClaim{
			{ID: "identity", Kind: ClaimKindAnsweringIdentity, Value: identity},
			{ID: "license", Kind: ClaimKindLicenseAssertion, Value: "CC0-1.0"},
			{ID: "path", Kind: ClaimKindCorpusPathMembership, Value: "core/example"},
			{ID: "profile", Kind: ClaimKindTrainingProfile, Value: profile},
			{ID: "evaluation", Kind: ClaimKindEvaluationIndependent, Value: ClaimValueEvaluationClear},
		},
	}, packet)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != ClaimGateStateConfirmed {
		t.Fatalf("claim gate state = %q, findings = %+v", receipt.State, receipt.Findings)
	}
	for _, finding := range receipt.Findings {
		if finding.Status != ClaimStatusConfirmed || len(finding.EvidenceReceiptSHA256) == 0 {
			t.Fatalf("finding = %+v", finding)
		}
	}
}

func TestSourceClaimGatePreservesCausalLegalAndReproductionCeilings(t *testing.T) {
	packet := readyContextPacket(t)
	receipt, err := AssessSourceClaims(SourceClaimSubmission{
		Schema: SourceClaimSubmissionSchema, SubmissionID: "claims-ceilings",
		OutputSHA256: repeatHex("a"), ContextPacketSHA256: packet.PacketSHA256,
		Claims: []SourceClaim{
			{ID: "causal", Kind: ClaimKindSourceCausality, Value: "core/example"},
			{ID: "legal", Kind: ClaimKindLegalUsability, Value: "CC0-1.0"},
			{ID: "reproduction", Kind: ClaimKindExactReproducibility, Value: "exact"},
		},
	}, packet)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != ClaimGateStateReview {
		t.Fatalf("claim gate state = %q", receipt.State)
	}
	want := map[string]string{
		"causal": ClaimStatusUnprovenCausal, "legal": ClaimStatusNoLegalResult, "reproduction": ClaimStatusIncompleteHold,
	}
	for _, finding := range receipt.Findings {
		if finding.Status != want[finding.ClaimID] {
			t.Fatalf("finding = %+v", finding)
		}
	}
}

func TestSourceClaimGateContradictsWrongIdentity(t *testing.T) {
	packet := readyContextPacket(t)
	receipt, err := AssessSourceClaims(SourceClaimSubmission{
		Schema: SourceClaimSubmissionSchema, SubmissionID: "claims-wrong-identity",
		OutputSHA256: repeatHex("a"), ContextPacketSHA256: packet.PacketSHA256,
		Claims: []SourceClaim{{ID: "identity", Kind: ClaimKindAnsweringIdentity, Value: repeatHex("f")}},
	}, packet)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != ClaimGateStateContradicted || receipt.Findings[0].Status != ClaimStatusContradicted {
		t.Fatalf("claim assessment = %+v", receipt)
	}
}

func TestSourceClaimGateRejectsDifferentContextPacket(t *testing.T) {
	packet := readyContextPacket(t)
	_, err := AssessSourceClaims(SourceClaimSubmission{
		Schema: SourceClaimSubmissionSchema, SubmissionID: "claims-context-mismatch",
		OutputSHA256: repeatHex("a"), ContextPacketSHA256: repeatHex("b"),
		Claims: []SourceClaim{{ID: "identity", Kind: ClaimKindAnsweringIdentity, Value: repeatHex("f")}},
	}, packet)
	if err == nil {
		t.Fatal("different context packet was accepted")
	}
}

func TestSourceClaimAssessmentDetectsOutputDigestTamper(t *testing.T) {
	packet := readyContextPacket(t)
	receipt, err := AssessSourceClaims(SourceClaimSubmission{
		Schema: SourceClaimSubmissionSchema, SubmissionID: "claims-tamper",
		OutputSHA256: repeatHex("a"), ContextPacketSHA256: packet.PacketSHA256,
		Claims: []SourceClaim{{ID: "identity", Kind: ClaimKindAnsweringIdentity, Value: contextStringFact(t, packet, "answering.identity_sha256")}},
	}, packet)
	if err != nil {
		t.Fatal(err)
	}
	receipt.OutputSHA256 = repeatHex("b")
	if err := receipt.Validate(); err == nil {
		t.Fatal("tampered output binding validated")
	}
}

func TestSourceClaimGateRejectsDuplicateClaimID(t *testing.T) {
	packet := readyContextPacket(t)
	_, err := AssessSourceClaims(SourceClaimSubmission{
		Schema: SourceClaimSubmissionSchema, SubmissionID: "claims-duplicate",
		OutputSHA256: repeatHex("a"), ContextPacketSHA256: packet.PacketSHA256,
		Claims: []SourceClaim{
			{ID: "same", Kind: ClaimKindSourceCausality, Value: "one"},
			{ID: "same", Kind: ClaimKindSourceCausality, Value: "two"},
		},
	}, packet)
	if err == nil {
		t.Fatal("duplicate claim id was accepted")
	}
}

func readyContextPacket(t *testing.T) ProvenanceContextPacket {
	t.Helper()
	packet, err := BuildProvenanceContext(validContextRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	if packet.State != ContextStateReady {
		t.Fatalf("context packet state = %q", packet.State)
	}
	return packet
}

func contextStringFact(t *testing.T, packet ProvenanceContextPacket, path string) string {
	t.Helper()
	fact, ok := findProvenanceFact(packet, path)
	if !ok {
		t.Fatalf("missing context fact %q", path)
	}
	var value string
	if err := json.Unmarshal(fact.Value, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
