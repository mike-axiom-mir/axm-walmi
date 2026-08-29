package axmmirror

import (
	"encoding/json"
	"strings"
	"testing"
)

func testPlatformFabricReceipt(t *testing.T, ai bool) []byte {
	t.Helper()
	concept := PlatformFabricRef{ID: "one-concept-many-bodies", Schema: "axm.waldo-mirror.fabric-concept/v0.1", SHA256: "sha256:" + strings.Repeat("1", 64)}
	bodies := []PlatformFabricBodyReceipt{}
	ids := []string{"evidence-trail-mini-game", "continuity-diagnostic-tool", "lineage-how-to-guide"}
	if ai {
		ids = append(ids, "tool-episode-quarantine-body")
	}
	var proposal *PlatformFabricRef
	if ai {
		value := PlatformFabricRef{ID: "ai-tool-episode-quarantine", Schema: "axm.waldo-mirror.fabric-ai-proposal/v0.1", SHA256: "sha256:" + strings.Repeat("a", 64)}
		proposal = &value
	}
	for index, id := range ids {
		source := PlatformFabricRef{ID: id, Schema: "axm.waldo-mirror.fabric-deterministic-body/v0.1", SHA256: "sha256:" + strings.Repeat(string(rune('2'+index)), 64)}
		mode := PlatformFabricDeterministicSource
		if ai && index == len(ids)-1 {
			source = *proposal
			mode = PlatformFabricAIOptInSource
		}
		bodies = append(bodies, PlatformFabricBodyReceipt{
			BodyID: id, SourceMode: mode, SourceRef: source, ConceptRef: concept,
			IntentRef:           PlatformFabricRef{ID: id, Schema: "axm.fabric-declarative-blueprint-intent/v1", SHA256: "sha256:" + strings.Repeat(string(rune('6'+index)), 64)},
			BlueprintRef:        PlatformFabricRef{ID: id, Schema: "axm.fabric-declarative-capability-blueprint/v1", SHA256: "sha256:" + strings.Repeat(string(rune('b'+index)), 64)},
			InterfaceSchemaRefs: []PlatformFabricInterfaceRef{{Direction: "input", Schema: "urn:test:input", SHA256: "sha256:" + strings.Repeat("c", 64)}, {Direction: "output", Schema: "urn:test:output", SHA256: "sha256:" + strings.Repeat("d", 64)}},
			AcceptanceMatrixRef: PlatformFabricSchemaDigestRef{Schema: "axm.fabric-blueprint-acceptance-matrix/v1", SHA256: "sha256:" + strings.Repeat([]string{"0", "1", "2", "3"}[index], 64)},
			Truth:               PlatformFabricBodyTruth{DeterministicReplayMatched: true, DesiredOutcomesRemainUnproven: true},
		})
	}
	receipt := PlatformFabricReceipt{
		Schema: PlatformFabricReceiptSchema, Status: "EXPERIMENTAL", Challenge: PlatformFabricChallenge,
		Donor:      PlatformFabricDonor{Repository: PlatformFabricRepository, Commit: PlatformFabricCommit, Blobs: platformFabricDonorBlobs},
		ConceptRef: concept, DeterministicDefault: true, AIOptInUsed: ai, AIProposalRef: proposal, Bodies: bodies,
		PrivateOrganFactory: PlatformFabricPrivateOrganFactory{Availability: "NOT_PUBLIC_IN_DONOR", EffectOnThisResult: "NONE"},
	}
	core, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := platformFabricExternalReceiptDigest(core)
	if err != nil {
		t.Fatal(err)
	}
	receipt.ReceiptSHA256 = digest
	data, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestPlatformFabricBridgeDeterministicAndAI(t *testing.T) {
	for _, ai := range []bool{false, true} {
		data := testPlatformFabricReceipt(t, ai)
		receipt, err := VerifyPlatformFabricReceipt(data)
		if err != nil {
			t.Fatalf("verify ai=%v: %v", ai, err)
		}
		witness, err := WitnessPlatformFabricReceipt(data)
		if err != nil {
			t.Fatalf("witness ai=%v: %v", ai, err)
		}
		if receipt.AIOptInUsed != witness.AIOptInUsed {
			t.Fatal("AI opt-in flag drifted")
		}
		if err := witness.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPlatformFabricBridgeTamperFails(t *testing.T) {
	data := testPlatformFabricReceipt(t, false)
	data = []byte(strings.Replace(string(data), "ONE_CONCEPT_MANY_BODIES", "ONE_CONCEPT_MANY_BODYX", 1))
	if _, err := VerifyPlatformFabricReceipt(data); err == nil {
		t.Fatal("tampered receipt passed")
	}
}
