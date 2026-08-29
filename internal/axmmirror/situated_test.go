package axmmirror

import "testing"

func TestSituatedContextJoinsAllThirteenSensoriumContracts(t *testing.T) {
	fixture := validGatedFixture(t)
	envelope := validSituatedEnvelope(t, fixture)
	if envelope.State != SituatedContextReady || len(envelope.RequiredSenses) != 13 || len(envelope.SensoryEvidence) != 13 {
		t.Fatalf("situated envelope = %+v", envelope)
	}
	if err := envelope.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestSealSituatedGatedBindsSensorySkillsAndDiscovery(t *testing.T) {
	fixture := validGatedFixture(t)
	envelope := validSituatedEnvelope(t, fixture)
	sealed, err := SealSituatedGated(
		fixture.Draft, fixture.Anchor, fixture.Witness, fixture.Profile,
		fixture.Context, fixture.Claims, fixture.Protocol, envelope,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		VerificationSensoryEvidence, VerificationSkillContinuity,
		VerificationDiscoveryStance, VerificationSituatedContext,
	} {
		verification := findVerification(sealed.Record.Verification, name)
		if verification == nil || verification.State != "pass" || verification.EvidenceSHA256 == "" {
			t.Fatalf("situated verification %q = %+v", name, verification)
		}
	}
	if err := sealed.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestSealSituatedGatedRefusesPassingOutcomeWithStaleSense(t *testing.T) {
	fixture := validGatedFixture(t)
	subject := testEvidenceSubject()
	request := validSituatedRequest(t, fixture, subject)
	stale := validSensoryDraft(subject, request.RequiredSenses[0])
	stale.TTLMillis = 1
	stale.AssessedAt = "2026-08-15T10:00:01Z"
	receipt, err := IntakeSensoryEvidence(stale)
	if err != nil {
		t.Fatal(err)
	}
	request.SensoryEvidence[0] = receipt
	envelope, err := BuildSituatedContext(request, fixture.Context)
	if err != nil {
		t.Fatal(err)
	}
	if envelope.State != SituatedContextSensoryHold {
		t.Fatalf("situated state = %q", envelope.State)
	}
	if _, err := SealSituatedGated(fixture.Draft, fixture.Anchor, fixture.Witness, fixture.Profile, fixture.Context, fixture.Claims, fixture.Protocol, envelope); err == nil {
		t.Fatal("passing behavior accepted stale required sensory evidence")
	}
	fixture.Draft.OutcomeState = "hold"
	sealed, err := SealSituatedGated(fixture.Draft, fixture.Anchor, fixture.Witness, fixture.Profile, fixture.Context, fixture.Claims, fixture.Protocol, envelope)
	if err != nil {
		t.Fatal(err)
	}
	if verification := findVerification(sealed.Record.Verification, VerificationSensoryEvidence); verification == nil || verification.State != "hold" {
		t.Fatalf("sensory hold verification = %+v", verification)
	}
}

func TestSituatedContextPreservesIdentityJoinHoldAndDetectsTamper(t *testing.T) {
	fixture := validGatedFixture(t)
	request := validSituatedRequest(t, fixture, testEvidenceSubject())
	request.TargetAnsweringIdentitySHA256 = repeatHex("e")
	request.SkillContinuity = mustSkillReceipt(t, validSkillContinuityRequest(request.TargetAnsweringIdentitySHA256))
	envelope, err := BuildSituatedContext(request, fixture.Context)
	if err != nil {
		t.Fatal(err)
	}
	if envelope.State != SituatedContextJoinHold {
		t.Fatalf("situated state = %q", envelope.State)
	}
	envelope.Subject.ID = "tampered"
	if err := envelope.Validate(); err == nil {
		t.Fatal("tampered situated context validated")
	}
}

func validSituatedEnvelope(t *testing.T, fixture gatedFixture) SituatedContextEnvelope {
	t.Helper()
	request := validSituatedRequest(t, fixture, testEvidenceSubject())
	envelope, err := BuildSituatedContext(request, fixture.Context)
	if err != nil {
		t.Fatal(err)
	}
	return envelope
}

func validSituatedRequest(t *testing.T, fixture gatedFixture, subject EvidenceSubject) SituatedContextRequest {
	t.Helper()
	capabilities := SupportedSensoryCapabilities()
	required := make([]string, 0, len(capabilities))
	receipts := make([]SensoryEvidenceReceipt, 0, len(capabilities))
	for _, capability := range capabilities {
		required = append(required, capability.SenseID)
		receipt, err := IntakeSensoryEvidence(validSensoryDraft(subject, capability.SenseID))
		if err != nil {
			t.Fatal(err)
		}
		receipts = append(receipts, receipt)
	}
	skill := mustSkillReceipt(t, validSkillContinuityRequest(fixture.Anchor.AnsweringIdentitySHA256))
	discovery, err := BuildDiscoveryStance(validDiscoveryRequest(subject))
	if err != nil {
		t.Fatal(err)
	}
	return SituatedContextRequest{
		Schema: SituatedContextRequestSchema, EnvelopeID: "situated-0001",
		TargetAnsweringIdentitySHA256: fixture.Anchor.AnsweringIdentitySHA256,
		Subject:                       subject, ContextPacketSHA256: fixture.Context.PacketSHA256,
		RequiredSenses: required, SensoryEvidence: receipts,
		SkillContinuity: skill, DiscoveryStance: discovery,
	}
}

func mustSkillReceipt(t *testing.T, request SkillContinuityRequest) SkillContinuityReceipt {
	t.Helper()
	receipt, err := AssessSkillContinuity(request)
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}
