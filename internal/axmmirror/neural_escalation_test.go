package axmmirror

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

type mirrorEscalatorStub struct {
	candidate MirrorNeuralCandidate
	err       error
	calls     int
	prompt    string
}

func (stub *mirrorEscalatorStub) Escalate(_ context.Context, prompt string) (MirrorNeuralCandidate, error) {
	stub.calls++
	stub.prompt = prompt
	return stub.candidate, stub.err
}

func TestMirrorNeuralEscalationKeepsDeterministicResultPrimary(t *testing.T) {
	request := mirrorEscalationRequest(GroundingStable, MirrorEscalationNone)
	request.DeterministicResponse = "deterministic answer"
	stub := &mirrorEscalatorStub{candidate: MirrorNeuralCandidate{Text: "must not run"}}
	receipt, err := RunMirrorNeuralEscalation(t.Context(), request, true, stub)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != MirrorStatusDeterministicResolved || receipt.DeterministicResponse != "deterministic answer" || receipt.NeuralCalled || stub.calls != 0 {
		t.Fatalf("receipt = %+v, calls = %d", receipt, stub.calls)
	}
}

func TestMirrorNeuralEscalationRequiresRuntimeOptIn(t *testing.T) {
	request := mirrorEscalationRequest(GroundingUncertain, MirrorEscalationNovelty)
	stub := &mirrorEscalatorStub{candidate: MirrorNeuralCandidate{Text: "candidate"}}
	receipt, err := RunMirrorNeuralEscalation(t.Context(), request, false, stub)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != MirrorStatusNeuralOptInRequired || receipt.NeuralCalled || stub.calls != 0 || receipt.LearningLedgerMutation || receipt.ModelMemoryMutation || receipt.TrainingMutation || receipt.IdentityMutation {
		t.Fatalf("receipt = %+v, calls = %d", receipt, stub.calls)
	}
}

func TestMirrorNeuralEscalationCallsLocalCandidateWithVisibleIdentity(t *testing.T) {
	request := mirrorEscalationRequest(GroundingUncertain, MirrorEscalationAmbiguity)
	identity, err := NewMirrorIdentityCapsule("axiom-mir", "roots-v1", []string{"truth before story", "keep dissent visible"})
	if err != nil {
		t.Fatal(err)
	}
	request.Identity = &identity
	stub := &mirrorEscalatorStub{candidate: MirrorNeuralCandidate{Text: "bounded possibility", Model: "local", Backend: "pytorch", Tokens: 3}}
	receipt, err := RunMirrorNeuralEscalation(t.Context(), request, true, stub)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != MirrorStatusNeuralCandidate || !receipt.NeuralCalled || stub.calls != 1 || receipt.NeuralCandidate == nil || receipt.NeuralCandidate.Text != "bounded possibility" {
		t.Fatalf("receipt = %+v, calls = %d", receipt, stub.calls)
	}
	for _, value := range []string{"identity_id=axiom-mir", "truth before story", "state=UNCERTAIN", "task:\nWhat changed?"} {
		if !strings.Contains(stub.prompt, value) {
			t.Fatalf("neural prompt is missing %q: %s", value, stub.prompt)
		}
	}
	if receipt.IdentitySHA256 != request.Identity.RootSHA256 || receipt.NeuralCandidateSHA256 == "" || receipt.NeuralPromptSHA256 == "" || !receipt.Authority.closed() {
		t.Fatalf("receipt binding = %+v", receipt)
	}
}

func TestMirrorNeuralEscalationFailureIsAVisibleHold(t *testing.T) {
	request := mirrorEscalationRequest(GroundingConflict, MirrorEscalationConflict)
	stub := &mirrorEscalatorStub{err: errors.New("backend stopped")}
	receipt, err := RunMirrorNeuralEscalation(t.Context(), request, true, stub)
	if err == nil || receipt.Status != MirrorStatusNeuralFailed || receipt.NeuralErrorSHA256 == "" || !receipt.NeuralCalled {
		t.Fatalf("receipt = %+v, err = %v", receipt, err)
	}
}

func TestMirrorChatLearningRecordIsExplicitAndNotYetTraining(t *testing.T) {
	request := mirrorEscalationRequest(GroundingUncertain, MirrorEscalationNovelty)
	stub := &mirrorEscalatorStub{candidate: MirrorNeuralCandidate{Text: "candidate", Model: "local", Backend: "pytorch"}}
	receipt, err := RunMirrorNeuralEscalation(t.Context(), request, true, stub)
	if err != nil {
		t.Fatal(err)
	}
	record, err := BuildMirrorChatLearningRecord(receipt, request.Prompt, MirrorLearningCaptureCandidate)
	if err != nil {
		t.Fatal(err)
	}
	if record.Consent != MirrorLearningConsentExplicit || record.ReviewState != MirrorLearningReviewRequired || record.TrainingReady || !record.NeuralCandidate || record.LearningRecordSHA256 == "" {
		t.Fatalf("learning record = %+v", record)
	}
	line, err := record.JSONLine()
	if err != nil || !bytes.HasSuffix(line, []byte{'\n'}) {
		t.Fatalf("line = %q, err = %v", line, err)
	}
	record.Response = "silent rewrite"
	if _, err := record.JSONLine(); err == nil {
		t.Fatal("tampered learning record was accepted")
	}
}

func TestMirrorChatLearningCanBeExplicitlyApprovedForTraining(t *testing.T) {
	request := mirrorEscalationRequest(GroundingStable, MirrorEscalationNone)
	request.DeterministicResponse = "accepted response"
	receipt, err := RunMirrorNeuralEscalation(t.Context(), request, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	record, err := BuildMirrorChatLearningRecord(receipt, request.Prompt, MirrorLearningApproveTraining)
	if err != nil {
		t.Fatal(err)
	}
	if record.ReviewState != MirrorLearningApproved || !record.TrainingReady || record.Consent != MirrorLearningConsentExplicit || !strings.Contains(record.TrainingText, "User: What changed?\n\nAssistant: accepted response") {
		t.Fatalf("approved learning record = %+v", record)
	}
}

func TestLoadMirrorNeuralEscalationRequestFailsClosed(t *testing.T) {
	request := mirrorEscalationRequest(GroundingUncertain, MirrorEscalationAmbiguity)
	payload := `{"schema":"` + request.Schema + `","prompt":"question","escalationReason":"AMBIGUITY","grounding":{"schema":"` + LiveNeuralGroundingSchema + `","state":"UNCERTAIN","consequence":"LOW","authority":{}},"hidden":true}`
	if _, err := LoadMirrorNeuralEscalationRequest(strings.NewReader(payload)); err == nil {
		t.Fatal("unknown request field was accepted")
	}
	request.DeterministicResponse = "false certainty"
	if err := request.Validate(); err == nil {
		t.Fatal("unresolved request claimed a deterministic response")
	}
}

func mirrorEscalationRequest(state, reason string) MirrorNeuralEscalationRequest {
	return MirrorNeuralEscalationRequest{
		Schema:           MirrorNeuralEscalationRequestSchema,
		Prompt:           "What changed?",
		EscalationReason: reason,
		Grounding: LiveNeuralGrounding{
			Schema:      LiveNeuralGroundingSchema,
			State:       state,
			Consequence: ConsequenceLow,
			Authority:   Authority{},
		},
	}
}
