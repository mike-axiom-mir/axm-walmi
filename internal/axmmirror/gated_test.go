package axmmirror

import "testing"

func TestSealGatedBindsCloneBoundaryIntoBehaviorEvidence(t *testing.T) {
	fixture := validGatedFixture(t)
	sealed, err := SealGated(fixture.Draft, fixture.Anchor, fixture.Witness, fixture.Profile, fixture.Context, fixture.Claims, fixture.Protocol)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{VerificationTrainingProfile, VerificationContextPacket, VerificationSourceClaims, VerificationProtocolSeal, "waldo-training-run-witness"} {
		if !hasVerification(sealed.Record.Verification, name) {
			t.Fatalf("sealed behavior is missing verification %q: %+v", name, sealed.Record.Verification)
		}
	}
	if err := sealed.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestSealGatedRefusesUnreadyProtocol(t *testing.T) {
	fixture := validGatedFixture(t)
	draft := validProtocolDraft(t)
	draft.AnswerKeyVisibility = AnswerKeyVisible
	protocol, err := SealEvaluationProtocol(draft)
	if err != nil {
		t.Fatal(err)
	}
	_, err = SealGated(fixture.Draft, fixture.Anchor, fixture.Witness, fixture.Profile, fixture.Context, fixture.Claims, protocol)
	if err == nil {
		t.Fatal("unready evaluation protocol was accepted")
	}
}

func TestSealGatedRefusesPassingOutcomeWithUnresolvedClaims(t *testing.T) {
	fixture := validGatedFixture(t)
	claims, err := AssessSourceClaims(SourceClaimSubmission{
		Schema: SourceClaimSubmissionSchema, SubmissionID: "gated-review",
		OutputSHA256: fixture.Draft.OutputSHA256, ContextPacketSHA256: fixture.Context.PacketSHA256,
		Claims: []SourceClaim{{ID: "causal", Kind: ClaimKindSourceCausality, Value: "core/example"}},
	}, fixture.Context)
	if err != nil {
		t.Fatal(err)
	}
	_, err = SealGated(fixture.Draft, fixture.Anchor, fixture.Witness, fixture.Profile, fixture.Context, claims, fixture.Protocol)
	if err == nil {
		t.Fatal("passing outcome with unresolved claims was accepted")
	}
}

func TestSealGatedPreservesHeldClaimEvidence(t *testing.T) {
	fixture := validGatedFixture(t)
	claims, err := AssessSourceClaims(SourceClaimSubmission{
		Schema: SourceClaimSubmissionSchema, SubmissionID: "gated-held",
		OutputSHA256: fixture.Draft.OutputSHA256, ContextPacketSHA256: fixture.Context.PacketSHA256,
		Claims: []SourceClaim{{ID: "causal", Kind: ClaimKindSourceCausality, Value: "core/example"}},
	}, fixture.Context)
	if err != nil {
		t.Fatal(err)
	}
	fixture.Draft.OutcomeState = "hold"
	sealed, err := SealGated(fixture.Draft, fixture.Anchor, fixture.Witness, fixture.Profile, fixture.Context, claims, fixture.Protocol)
	if err != nil {
		t.Fatal(err)
	}
	verification := findVerification(sealed.Record.Verification, VerificationSourceClaims)
	if verification == nil || verification.State != "hold" || verification.EvidenceSHA256 != claims.ReceiptSHA256 {
		t.Fatalf("source-claim verification = %+v", verification)
	}
}

func TestSealGatedRejectsDifferentOutputBinding(t *testing.T) {
	fixture := validGatedFixture(t)
	fixture.Draft.OutputSHA256 = repeatHex("b")
	_, err := SealGated(fixture.Draft, fixture.Anchor, fixture.Witness, fixture.Profile, fixture.Context, fixture.Claims, fixture.Protocol)
	if err == nil {
		t.Fatal("different behavior output binding was accepted")
	}
}

func TestSealGatedRejectsDifferentAnchorReceiptWithSameAnsweringIdentity(t *testing.T) {
	fixture := validGatedFixture(t)
	bom := validModelBOM()
	bom.ModelID = fixture.Anchor.ModelID
	bom.ArchitectureSHA256 = fixture.Anchor.ArchitectureSHA256
	additional := bom.Runs[0]
	additional.ID = "run-0002"
	additional.Ordinal = 2
	additional.RunBOM = "runs/0002-pretrain-run-0002/RUN-BOM.json"
	additional.BOMSHA256 = repeatHex("7")
	additional.State = "planned"
	additional.Artifacts = nil
	bom.Runs = append(bom.Runs, additional)
	bom.Generated = "2026-08-15T04:00:00Z"
	alternate := anchorModelForTest(t, bom)
	if alternate.AnsweringIdentitySHA256 != fixture.Anchor.AnsweringIdentitySHA256 || alternate.BOMSHA256 == fixture.Anchor.BOMSHA256 {
		t.Fatalf("alternate anchor did not isolate provenance-only drift: original=%+v alternate=%+v", fixture.Anchor, alternate)
	}
	_, err := SealGated(fixture.Draft, alternate, fixture.Witness, fixture.Profile, fixture.Context, fixture.Claims, fixture.Protocol)
	if err == nil {
		t.Fatal("different anchor receipt with the same answering identity was accepted")
	}
}

type gatedFixture struct {
	Draft    BehaviorEvidenceDraft
	Anchor   OriginAnchor
	Witness  TrainingRunWitness
	Profile  TrainingProfileContract
	Context  ProvenanceContextPacket
	Claims   SourceClaimAssessment
	Protocol EvaluationProtocolSeal
}

func validGatedFixture(t *testing.T) gatedFixture {
	t.Helper()
	request := validContextRequest(t)
	request.Fields = append(request.Fields, "evaluation.comparison_sha256")
	context, err := BuildProvenanceContext(request)
	if err != nil {
		t.Fatal(err)
	}
	protocolDraft := EvaluationProtocolDraft{
		Schema: EvaluationProtocolDraftSchema, ProtocolID: "gated-evaluation-0001",
		Pack: EvaluationPackIdentity{
			SHA256: repeatHex("1"), DocumentSHA256: repeatHex("2"), CaseCount: 2,
			CaseOrderSHA256: repeatHex("3"), AuthorshipState: "outside-authored",
			AuthorshipEvidenceSHA256: repeatHex("9"), AnswerKeySHA256: repeatHex("4"),
		},
		TargetAnsweringIdentitySHA256: request.Anchor.AnsweringIdentitySHA256,
		Anchor:                        request.Anchor, Context: context, Contamination: *request.Contamination,
		RequestSetSHA256: repeatHex("5"), AllowedMetrics: []string{"exact-match"},
		ComparisonDimensions: []string{"output"}, AnswerKeyVisibility: AnswerKeyWithheld,
		Limits:          EvaluationExecutionLimits{MaxCases: 2, MaxOutputBytes: 4096, TimeoutMillis: 30000},
		PermissionState: "allowed", RequestedAuthority: Authority{},
	}
	protocol, err := SealEvaluationProtocol(protocolDraft)
	if err != nil {
		t.Fatal(err)
	}
	outputDigest := repeatHex("a")
	claims, err := AssessSourceClaims(SourceClaimSubmission{
		Schema: SourceClaimSubmissionSchema, SubmissionID: "gated-claims-0001",
		OutputSHA256: outputDigest, ContextPacketSHA256: context.PacketSHA256,
		Claims: []SourceClaim{{ID: "identity", Kind: ClaimKindAnsweringIdentity, Value: request.Anchor.AnsweringIdentitySHA256}},
	}, context)
	if err != nil {
		t.Fatal(err)
	}
	draft := BehaviorEvidenceDraft{
		Schema: BehaviorEvidenceSchema, ExperimentID: protocol.ProtocolID,
		WALDO: WALDOLineage{
			CorpusBOMSHA256: request.RunWitness.Corpus.BOMSHA256,
			RunBOMSHA256:    request.RunWitness.RunBOMSHA256,
			ModelBOMSHA256:  request.Anchor.BOMSHA256,
		},
		ContextSHA256: context.PacketSHA256, RequestSHA256: protocol.RequestSetSHA256, OutputSHA256: outputDigest,
		PermissionState: "allowed", OutcomeState: "pass", Authority: Authority{},
	}
	return gatedFixture{
		Draft: draft, Anchor: request.Anchor, Witness: *request.RunWitness, Profile: *request.ProfileContract,
		Context: context, Claims: claims, Protocol: protocol,
	}
}

func hasVerification(values []Verification, name string) bool {
	return findVerification(values, name) != nil
}

func findVerification(values []Verification, name string) *Verification {
	for i := range values {
		if values[i].Name == name {
			return &values[i]
		}
	}
	return nil
}
