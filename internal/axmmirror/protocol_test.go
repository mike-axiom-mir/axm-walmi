package axmmirror

import (
	"encoding/json"
	"testing"
)

func TestEvaluationProtocolSealsReadyBeforeOutputs(t *testing.T) {
	draft := validProtocolDraft(t)
	seal, err := SealEvaluationProtocol(draft)
	if err != nil {
		t.Fatal(err)
	}
	if seal.State != ProtocolStateReady || len(seal.Holds) != 0 || seal.Target.ContextPacketSHA256 != draft.Context.PacketSHA256 {
		t.Fatalf("evaluation protocol seal = %+v", seal)
	}
	if err := seal.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestEvaluationProtocolCanonicalizesMetricAndDimensionOrder(t *testing.T) {
	left := validProtocolDraft(t)
	right := left
	right.AllowedMetrics = []string{"refusal-state", "exact-match"}
	right.ComparisonDimensions = []string{"verification", "output"}
	leftSeal, err := SealEvaluationProtocol(left)
	if err != nil {
		t.Fatal(err)
	}
	rightSeal, err := SealEvaluationProtocol(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftSeal.SealSHA256 != rightSeal.SealSHA256 || leftSeal.DraftSHA256 != rightSeal.DraftSHA256 {
		t.Fatalf("canonical protocol seals differ: %s != %s", leftSeal.SealSHA256, rightSeal.SealSHA256)
	}
}

func TestEvaluationProtocolHoldsContaminatedPack(t *testing.T) {
	draft := validProtocolDraft(t)
	comparison := validComparison()
	comparison.Training.RecordSHA256 = []string{repeatHex("3")}
	comparison.Authorship = AuthorshipDeclaration{State: "outside-authored", EvidenceSHA256: repeatHex("9")}
	report, err := CheckContamination(comparison)
	if err != nil {
		t.Fatal(err)
	}
	draft = protocolDraftWithContamination(t, draft, report)
	seal, err := SealEvaluationProtocol(draft)
	if err != nil {
		t.Fatal(err)
	}
	if seal.State != ProtocolStateContamination {
		t.Fatalf("evaluation protocol state = %q", seal.State)
	}
}

func TestEvaluationProtocolHoldsVisibleAnswerKey(t *testing.T) {
	draft := validProtocolDraft(t)
	draft.AnswerKeyVisibility = AnswerKeyVisible
	seal, err := SealEvaluationProtocol(draft)
	if err != nil {
		t.Fatal(err)
	}
	if seal.State != ProtocolStateAnswerKeyLeak {
		t.Fatalf("evaluation protocol state = %q", seal.State)
	}
}

func TestEvaluationProtocolRefusesAuthorityGrowth(t *testing.T) {
	draft := validProtocolDraft(t)
	draft.RequestedAuthority.Training = true
	seal, err := SealEvaluationProtocol(draft)
	if err != nil {
		t.Fatal(err)
	}
	if seal.State != ProtocolStateAuthorityRefused || !seal.Authority.closed() {
		t.Fatalf("evaluation protocol seal = %+v", seal)
	}
}

func TestEvaluationProtocolHoldsTargetMismatch(t *testing.T) {
	draft := validProtocolDraft(t)
	draft.TargetAnsweringIdentitySHA256 = repeatHex("f")
	seal, err := SealEvaluationProtocol(draft)
	if err != nil {
		t.Fatal(err)
	}
	if seal.State != ProtocolStateTargetMismatch {
		t.Fatalf("evaluation protocol state = %q", seal.State)
	}
}

func TestEvaluationProtocolDetectsSealTamper(t *testing.T) {
	seal, err := SealEvaluationProtocol(validProtocolDraft(t))
	if err != nil {
		t.Fatal(err)
	}
	seal.PermissionState = "denied"
	if err := seal.Validate(); err == nil {
		t.Fatal("tampered evaluation protocol seal validated")
	}
}

func TestEvaluationProtocolDraftStrictDecodeRejectsLaterOutput(t *testing.T) {
	data, err := json.Marshal(validProtocolDraft(t))
	if err != nil {
		t.Fatal(err)
	}
	data = append([]byte(`{"output_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",`), data[1:]...)
	var decoded EvaluationProtocolDraft
	if err := DecodeStrictJSON(data, &decoded); err == nil {
		t.Fatal("evaluation protocol draft accepted a later output field")
	}
}

func validProtocolDraft(t *testing.T) EvaluationProtocolDraft {
	t.Helper()
	contextRequest := validContextRequest(t)
	contextRequest.Fields = append(contextRequest.Fields, "evaluation.comparison_sha256")
	context, err := BuildProvenanceContext(contextRequest)
	if err != nil {
		t.Fatal(err)
	}
	return EvaluationProtocolDraft{
		Schema: EvaluationProtocolDraftSchema, ProtocolID: "evaluation-0001",
		Pack: EvaluationPackIdentity{
			SHA256: repeatHex("1"), DocumentSHA256: repeatHex("2"), CaseCount: 2,
			CaseOrderSHA256: repeatHex("3"), AuthorshipState: "outside-authored",
			AuthorshipEvidenceSHA256: repeatHex("9"), AnswerKeySHA256: repeatHex("4"),
		},
		TargetAnsweringIdentitySHA256: contextRequest.Anchor.AnsweringIdentitySHA256,
		Anchor:                        contextRequest.Anchor, Context: context, Contamination: *contextRequest.Contamination,
		RequestSetSHA256: repeatHex("5"), AllowedMetrics: []string{"exact-match", "refusal-state"},
		ComparisonDimensions: []string{"output", "verification"}, AnswerKeyVisibility: AnswerKeyWithheld,
		Limits:          EvaluationExecutionLimits{MaxCases: 2, MaxOutputBytes: 4096, TimeoutMillis: 30000},
		PermissionState: "allowed", RequestedAuthority: Authority{},
	}
}

func protocolDraftWithContamination(t *testing.T, draft EvaluationProtocolDraft, report ContaminationReport) EvaluationProtocolDraft {
	t.Helper()
	request := validContextRequest(t)
	request.Contamination = &report
	request.Fields = append(request.Fields, "evaluation.comparison_sha256")
	context, err := BuildProvenanceContext(request)
	if err != nil {
		t.Fatal(err)
	}
	draft.Anchor = request.Anchor
	draft.Context = context
	draft.Contamination = report
	draft.TargetAnsweringIdentitySHA256 = request.Anchor.AnsweringIdentitySHA256
	return draft
}
