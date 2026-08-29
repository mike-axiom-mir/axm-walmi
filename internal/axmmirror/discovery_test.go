package axmmirror

import "testing"

func TestDiscoveryStancePreservesPrimaryAIAndSecondaryHumanSurfaces(t *testing.T) {
	packet, err := BuildDiscoveryStance(validDiscoveryRequest(testEvidenceSubject()))
	if err != nil {
		t.Fatal(err)
	}
	if packet.State != DiscoveryStateReady || packet.AINative.Priority != "PRIMARY" || packet.HumanNative.Priority != "SECONDARY" {
		t.Fatalf("discovery packet = %+v", packet)
	}
	if packet.HumanNative.MayCloseNativeSeams || packet.HumanNative.MayPromoteLearning || packet.HumanNative.MayGrantPermissions {
		t.Fatalf("human advisory gained authority: %+v", packet.HumanNative)
	}
	if packet.AINative.Summary.Open != 1 || packet.HumanNative.InvocationState != HumanDiscoveryAdvisory {
		t.Fatalf("discovery surfaces = %+v / %+v", packet.AINative, packet.HumanNative)
	}
}

func TestDiscoveryStanceRequiresExplicitHumanInvocation(t *testing.T) {
	request := validDiscoveryRequest(testEvidenceSubject())
	request.Human.ExplicitInvocation = false
	packet, err := BuildDiscoveryStance(request)
	if err != nil {
		t.Fatal(err)
	}
	if packet.State != DiscoveryStateInvocationHold || len(packet.Holds) == 0 {
		t.Fatalf("discovery packet = %+v", packet)
	}
}

func TestDiscoveryStanceRefusesUnevidencedClosure(t *testing.T) {
	request := validDiscoveryRequest(testEvidenceSubject())
	request.AISeams[0].Status = "CLOSED"
	packet, err := BuildDiscoveryStance(request)
	if err != nil {
		t.Fatal(err)
	}
	if packet.State != DiscoveryStateClosureHold {
		t.Fatalf("discovery state = %q", packet.State)
	}

	request.AISeams[0].ClosureHistory = []SeamClosureEvidence{{
		At: "2026-08-15T10:01:00Z", Actor: "independent-reviewer", TestStatus: "PASS",
		Statement: "The exact countercheck passed.", EvidenceRefs: []string{"receipt-countercheck"},
	}}
	packet, err = BuildDiscoveryStance(request)
	if err != nil {
		t.Fatal(err)
	}
	if packet.State != DiscoveryStateReady || packet.AINative.Summary.Closed != 1 {
		t.Fatalf("closed discovery packet = %+v", packet)
	}
}

func TestDiscoveryStanceDetectsTamper(t *testing.T) {
	packet, err := BuildDiscoveryStance(validDiscoveryRequest(testEvidenceSubject()))
	if err != nil {
		t.Fatal(err)
	}
	packet.HumanNative.Entries[0].Text = "tampered"
	if err := packet.Validate(); err == nil {
		t.Fatal("tampered discovery packet validated")
	}
}

func TestDiscoveryStanceRejectsDuplicateSeamIDsAcrossSeverities(t *testing.T) {
	request := validDiscoveryRequest(testEvidenceSubject())
	duplicate := request.AISeams[0]
	duplicate.Severity = "low"
	request.AISeams = append(request.AISeams, duplicate)
	if _, err := BuildDiscoveryStance(request); err == nil {
		t.Fatal("duplicate AI-native seam ID across severities was accepted")
	}
}

func TestDiscoveryStanceRejectsInvalidHumanClaimLabel(t *testing.T) {
	request := validDiscoveryRequest(testEvidenceSubject())
	request.Human.Entries[0].ClaimLabel = "CERTAIN"
	if _, err := BuildDiscoveryStance(request); err == nil {
		t.Fatal("invalid human-native claim label was accepted")
	}
}

func validDiscoveryRequest(subject EvidenceSubject) DiscoveryStanceRequest {
	return DiscoveryStanceRequest{
		Schema: DiscoveryStanceRequestSchema, RequestID: "discovery-0001", Subject: subject,
		AISeams: []DiscoverySeam{{
			ID: "freshness-boundary", Stance: "evidence", Severity: "high", Status: "OPEN",
			Statement:          "The result depends on a sensory observation whose current freshness must remain visible.",
			EvidenceRefs:       []string{"claim-situated-0001"},
			DisconfirmingCheck: "Reassess the same typed observation inside its declared TTL.",
			Blocks:             []string{"fresh-world-state-claim"}, RepairHint: "Request a new bounded receipt; do not reuse stale raw material.",
		}},
		Human: &HumanDiscoveryInput{
			ExplicitInvocation: true,
			Entries: []HumanDiscoveryEntry{{
				ID: "human-meaning-0001", Stage: "soulChecks",
				Text:       "The explanation should preserve why the distinction matters to a person, not only the machine state.",
				ClaimLabel: "VALUE", Source: "explicit human-facing review", Rationale: "Machine correctness and human meaning are different review dimensions.",
			}},
		},
	}
}
