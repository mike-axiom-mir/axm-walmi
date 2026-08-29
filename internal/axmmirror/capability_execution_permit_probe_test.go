package axmmirror

import (
	"encoding/json"
	"os"
	"testing"
)

func loadExecutionPermitV019Fixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/capability-execution-permit-v0.19.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestExecutionPermitV019ActualFixture(t *testing.T) {
	data := loadExecutionPermitV019Fixture(t)
	c, err := VerifyExecutionPermitV019Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	if c.ReceiptDigest != "sha256:88885cdb000a567746030914f170c161444db87ec6116ba8ce4b6a58da6257dc" {
		t.Fatalf("unexpected receipt %s", c.ReceiptDigest)
	}
	if c.OverallDecision != "HOLD" || c.OverallReason != "EXECUTABLE_BODY_ABSENT" {
		t.Fatalf("unexpected decision %s/%s", c.OverallDecision, c.OverallReason)
	}
	w, err := WitnessExecutionPermitV019Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	if w.WitnessDigest != "960c7eab77597a87256da843663e88fe22d1290fcc1e42dbff1b1da14bfee7d3" {
		t.Fatalf("unexpected witness digest %s", w.WitnessDigest)
	}
	if w.OverallDecision != "HOLD" || len(w.PreservedRefusalCaseIDs) != 8 {
		t.Fatalf("unexpected witness %+v", w)
	}
}

func TestExecutionPermitV019PermissionAndBodyOnlyReachReadyNotExecution(t *testing.T) {
	data := loadExecutionPermitV019Fixture(t)
	var c ExecutionPermitV019Contract
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	r := c.ExecutionRequest
	r.ExecutableBodyPresent = true
	decision, reason := EvaluateExecutionPermitV019Request(c.ExecutionPermit, r)
	if decision != "READY" || reason != "EXECUTION_AUTHORIZED_NOT_EXECUTED" {
		t.Fatalf("got %s/%s", decision, reason)
	}
}

func TestExecutionPermitV019EvaluatorRefusals(t *testing.T) {
	data := loadExecutionPermitV019Fixture(t)
	var c ExecutionPermitV019Contract
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*ExecutionPermitV019Permit, *ExecutionPermitV019Request)
		reason string
	}{
		{"selection source mismatch", func(p *ExecutionPermitV019Permit, r *ExecutionPermitV019Request) {
			p.SourceSelectionContractReceipt = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}, "SELECTION_SOURCE_MISMATCH"},
		{"selection mismatch", func(p *ExecutionPermitV019Permit, r *ExecutionPermitV019Request) {
			p.SelectedCandidateID = "projected-read-chain"
		}, "EXECUTION_SELECTION_MISMATCH"},
		{"scope escalation", func(p *ExecutionPermitV019Permit, r *ExecutionPermitV019Request) { p.GrantsInstall = true }, "EXECUTION_SCOPE_EXCEEDED"},
		{"waldo self issue", func(p *ExecutionPermitV019Permit, r *ExecutionPermitV019Request) { p.WaldoIssued = true }, "WALDO_CANNOT_SELF_ISSUE_EXECUTION"},
		{"permit mismatch", func(p *ExecutionPermitV019Permit, r *ExecutionPermitV019Request) { r.PermitID = "other" }, "EXECUTION_REQUEST_PERMIT_MISMATCH"},
		{"auto build", func(p *ExecutionPermitV019Permit, r *ExecutionPermitV019Request) { r.AutoBuildMissingBody = true }, "AUTO_BUILD_FORBIDDEN"},
		{"invent", func(p *ExecutionPermitV019Permit, r *ExecutionPermitV019Request) { r.InventedBody = true }, "MISSING_BODY_INVENTION_FORBIDDEN"},
		{"side effect", func(p *ExecutionPermitV019Permit, r *ExecutionPermitV019Request) {
			r.RequestedSideEffects = []string{"WRITE_WORKSPACE"}
		}, "EXECUTION_REQUEST_SCOPE_EXCEEDED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, r := c.ExecutionPermit, c.ExecutionRequest
			tt.mutate(&p, &r)
			decision, reason := EvaluateExecutionPermitV019Request(p, r)
			if decision != "REFUSED" || reason != tt.reason {
				t.Fatalf("got %s/%s", decision, reason)
			}
		})
	}
}

func TestExecutionPermitV019ResealedSemanticTamperRejected(t *testing.T) {
	data := loadExecutionPermitV019Fixture(t)
	mutators := []func(*ExecutionPermitV019Contract){
		func(c *ExecutionPermitV019Contract) { c.ProbeBoundary.ExecutableCandidateBodyIncluded = true },
		func(c *ExecutionPermitV019Contract) { c.ExecutionPermit.GrantsPromotion = true },
		func(c *ExecutionPermitV019Contract) { c.ExecutionPermit.WaldoIssued = true },
		func(c *ExecutionPermitV019Contract) { c.ExecutionRequest.AutoBuildMissingBody = true },
		func(c *ExecutionPermitV019Contract) { c.ExecutionRequest.InventedBody = true },
		func(c *ExecutionPermitV019Contract) { c.Truth.ExecutionObserved = true },
		func(c *ExecutionPermitV019Contract) { c.BuildStarted = true },
		func(c *ExecutionPermitV019Contract) {
			c.SourceV018Receipt = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		},
	}
	for i, mutate := range mutators {
		var c ExecutionPermitV019Contract
		if err := json.Unmarshal(data, &c); err != nil {
			t.Fatal(err)
		}
		mutate(&c)
		resealed, err := SealExecutionPermitV019Contract(c)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := VerifyExecutionPermitV019Contract(resealed); err == nil {
			t.Fatalf("tamper %d unexpectedly accepted", i)
		}
	}
}

func TestExecutionPermitV019StaleReceiptRejected(t *testing.T) {
	data := loadExecutionPermitV019Fixture(t)
	var c ExecutionPermitV019Contract
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	c.ReceiptDigest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	tampered, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyExecutionPermitV019Contract(tampered); err == nil {
		t.Fatal("stale receipt unexpectedly accepted")
	}
}

func TestExecutionPermitV019UnknownFieldRejected(t *testing.T) {
	data := loadExecutionPermitV019Fixture(t)
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatal(err)
	}
	obj["hiddenAuthority"] = true
	tampered, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyExecutionPermitV019Contract(tampered); err == nil {
		t.Fatal("unknown field unexpectedly accepted")
	}
}
