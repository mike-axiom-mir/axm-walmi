package axmmirror

import "testing"

func TestSealAndVerifyDeterministic(t *testing.T) {
	draft := validDraft()
	first, err := Seal(draft)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	second, err := Seal(draft)
	if err != nil {
		t.Fatalf("Seal() second error = %v", err)
	}
	if first.SHA256 != second.SHA256 {
		t.Fatalf("Seal() not deterministic: %s != %s", first.SHA256, second.SHA256)
	}
	if err := first.Verify(); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestVerifyDetectsTamper(t *testing.T) {
	sealed, err := Seal(validDraft())
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	sealed.Record.OutcomeState = "fail"
	if err := sealed.Verify(); err == nil {
		t.Fatal("Verify() accepted mutated record")
	}
}

func TestRejectsAuthorityEscalation(t *testing.T) {
	draft := validDraft()
	draft.Authority.ToolExecution = true
	if _, err := Seal(draft); err == nil {
		t.Fatal("Seal() accepted tool execution authority")
	}
}

func TestRejectsMissingWALDOModelLineage(t *testing.T) {
	draft := validDraft()
	draft.WALDO.ModelBOMSHA256 = ""
	if _, err := Seal(draft); err == nil {
		t.Fatal("Seal() accepted missing model/release BOM lineage")
	}
}

func TestRejectsDuplicateVerificationName(t *testing.T) {
	draft := validDraft()
	draft.Verification = append(draft.Verification, draft.Verification[0])
	if _, err := Seal(draft); err == nil {
		t.Fatal("Seal() accepted duplicate verification name")
	}
}

func validDraft() BehaviorEvidenceDraft {
	return BehaviorEvidenceDraft{
		Schema:       BehaviorEvidenceSchema,
		ExperimentID: "mirror-waldo-smoke-001",
		WALDO: WALDOLineage{
			ModelBOMSHA256: repeatHex("a"),
		},
		ContextSHA256:   repeatHex("b"),
		RequestSHA256:   repeatHex("c"),
		OutputSHA256:    repeatHex("d"),
		PermissionState: "observed",
		OutcomeState:    "hold",
		Verification: []Verification{
			{Name: "format", State: "pass", EvidenceSHA256: repeatHex("e")},
			{Name: "grounding", State: "hold"},
		},
		Dissent: []Dissent{
			{Source: "reviewer-a", State: "open", Note: "Grounding has not been independently reproduced."},
		},
		Authority: Authority{},
	}
}

func repeatHex(ch string) string {
	out := ""
	for i := 0; i < 64; i++ {
		out += ch
	}
	return out
}
