package axmmirror

import (
	"strings"
	"testing"
)

func TestLiveNeuralGroundingDefaultIsClosedStable(t *testing.T) {
	grounding := DefaultLiveNeuralGrounding()
	if err := grounding.Validate(); err != nil {
		t.Fatal(err)
	}
	if grounding.ShouldDeterministicallyHold() {
		t.Fatal("default low-consequence stable grounding must not hold")
	}
	capsule, digest, err := grounding.Capsule()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(capsule, "neural output is a candidate") || len(digest) != 64 {
		t.Fatalf("unexpected capsule/digest: %q %q", capsule, digest)
	}
}

func TestLiveNeuralGroundingHighUncertaintyHolds(t *testing.T) {
	grounding := DefaultLiveNeuralGrounding()
	grounding.State = GroundingUncertain
	grounding.Consequence = ConsequenceHigh
	if !grounding.ShouldDeterministicallyHold() {
		t.Fatal("high-consequence unresolved grounding must hold")
	}
}

func TestLiveNeuralGroundingRejectsAuthorityWidening(t *testing.T) {
	grounding := DefaultLiveNeuralGrounding()
	grounding.Authority.ToolExecution = true
	if err := grounding.Validate(); err == nil {
		t.Fatal("expected authority widening refusal")
	}
}

func TestLiveNeuralGroundingCapsuleIsOrderStable(t *testing.T) {
	first := DefaultLiveNeuralGrounding()
	first.References = []LiveGroundingReference{
		{ID: "b", Text: "second", Class: ReferenceCandidate, Freshness: FreshnessCurrent, ProvenanceRoot: "root-b"},
		{ID: "a", Text: "first", Class: ReferenceVerified, Freshness: FreshnessCurrent, ProvenanceRoot: "root-a"},
	}
	second := first
	second.References = []LiveGroundingReference{first.References[1], first.References[0]}
	capsuleA, digestA, err := first.Capsule()
	if err != nil {
		t.Fatal(err)
	}
	capsuleB, digestB, err := second.Capsule()
	if err != nil {
		t.Fatal(err)
	}
	if capsuleA != capsuleB || digestA != digestB {
		t.Fatal("grounding capsule must be deterministic across input ordering")
	}
}

func TestLoadLiveNeuralGroundingFailsClosedOnUnknownField(t *testing.T) {
	_, err := LoadLiveNeuralGrounding(strings.NewReader(`{"schema":"axm.waldo.live-neural-grounding/v0.35","state":"STABLE","consequence":"LOW","references":[],"authority":{},"surprise":true}`))
	if err == nil {
		t.Fatal("expected unknown-field refusal")
	}
}
