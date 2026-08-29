package axmmirror

import "testing"

func TestSupportedSensoryCapabilitiesMatchesWorkshopThirteen(t *testing.T) {
	capabilities := SupportedSensoryCapabilities()
	if len(capabilities) != 13 {
		t.Fatalf("supported sensory capabilities = %d, want 13", len(capabilities))
	}
	for i, capability := range capabilities {
		if capability.SenseID == "" || capability.Capability == "" {
			t.Fatalf("capability %d = %+v", i, capability)
		}
		if i > 0 && capabilities[i-1].SenseID >= capability.SenseID {
			t.Fatal("supported sensory capabilities are not sorted")
		}
	}
}

func TestSensoryEvidenceIntakeAcceptsFreshPassAndFail(t *testing.T) {
	for _, verdict := range []string{"PASS", "FAIL"} {
		draft := validSensoryDraft(testEvidenceSubject(), "eye-static-image-inspector")
		draft.Verdict = verdict
		receipt, err := IntakeSensoryEvidence(draft)
		if err != nil {
			t.Fatal(err)
		}
		if receipt.State != SensoryStateReady || receipt.FreshnessStatus != SensoryFreshnessLive || receipt.Verdict != verdict {
			t.Fatalf("sensory receipt = %+v", receipt)
		}
		if err := receipt.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSensoryEvidenceIntakeHoldsStaleEvidence(t *testing.T) {
	draft := validSensoryDraft(testEvidenceSubject(), "ears-stream-listener")
	draft.TTLMillis = 1000
	draft.AssessedAt = "2026-08-15T10:00:02Z"
	receipt, err := IntakeSensoryEvidence(draft)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != SensoryStateStale || receipt.FreshnessStatus != SensoryFreshnessStale || len(receipt.Holds) == 0 {
		t.Fatalf("sensory receipt = %+v", receipt)
	}
}

func TestSensoryEvidenceIntakeHoldsRawRetentionAndRefusesAuthorityInheritance(t *testing.T) {
	retained := validSensoryDraft(testEvidenceSubject(), "touch-environment-probe")
	retained.RawRetainedItemsAfterSeal = 1
	receipt, err := IntakeSensoryEvidence(retained)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != SensoryStateRetention {
		t.Fatalf("retention state = %q", receipt.State)
	}

	inherited := validSensoryDraft(testEvidenceSubject(), "time-sense-ttl-verifier")
	inherited.AuthorityLeaseID = "source-lease"
	inherited.AuthorityInherited = true
	receipt, err = IntakeSensoryEvidence(inherited)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != SensoryStateAuthorityRefused || receipt.AuthorityInherited {
		t.Fatalf("authority receipt = %+v", receipt)
	}
}

func TestSensoryEvidenceReceiptDetectsTamper(t *testing.T) {
	receipt, err := IntakeSensoryEvidence(validSensoryDraft(testEvidenceSubject(), "eye-change-differ"))
	if err != nil {
		t.Fatal(err)
	}
	receipt.Verdict = "FAIL"
	receipt.AgeMillis++
	receipt.ReceiptSHA256, err = sensoryEvidenceReceiptDigest(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := receipt.Validate(); err == nil {
		t.Fatal("internally inconsistent but self-digested sensory receipt validated")
	}
}

func validSensoryDraft(subject EvidenceSubject, senseID string) SensoryEvidenceDraft {
	return SensoryEvidenceDraft{
		Schema: SensoryEvidenceDraftSchema, ObservationID: "observation-" + senseID,
		ClaimID: "claim-situated-0001", SenseID: senseID, Capability: sensoryCapabilities[senseID],
		Subject: subject, SeatID: "workshop-seat", BackendID: "fixture",
		ObservedAt: "2026-08-15T10:00:00Z", SealedAt: "2026-08-15T10:00:00.100Z",
		AssessedAt: "2026-08-15T10:00:00.200Z", TTLMillis: 60_000,
		ObservationState: SensoryObservationObserved, Verdict: "PASS",
		NamedSeams: []string{"bounded-fixture"}, TypedObservationSHA256: repeatHex("a"),
		SpecificReceiptSchema: "axm.sensorium-fixture/v1", SpecificReceiptSHA256: repeatHex("b"),
		CleanupComplete: true,
	}
}

func testEvidenceSubject() EvidenceSubject {
	return EvidenceSubject{ID: "subject-0001", Kind: "evaluation-case", SHA256: repeatHex("c")}
}
