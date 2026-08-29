package axmmirror

import (
	"encoding/json"
	"os"
	"testing"
)

func loadExecutionRevocationV020Fixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/capability-execution-revocation-v0.20.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestExecutionRevocationV020ActualFixture(t *testing.T) {
	data := loadExecutionRevocationV020Fixture(t)
	c, err := VerifyExecutionRevocationV020Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	if c.ReceiptDigest != "sha256:edb16e8f804b6cace4e59eaf7f66344ef31648e52b028129eae373224e3645c2" {
		t.Fatalf("unexpected receipt %s", c.ReceiptDigest)
	}
	if c.CurrentState != "REVOKED_BEFORE_EXECUTION" || len(c.Ledger) != 2 || c.Ledger[0].Outcome != "ACCEPTED" || c.Ledger[1].Outcome != "REVOKED" {
		t.Fatalf("unexpected ledger/current state %+v / %s", c.Ledger, c.CurrentState)
	}
	w, err := WitnessExecutionRevocationV020Contract(data)
	if err != nil {
		t.Fatal(err)
	}
	if w.WitnessDigest != "8038f833ad35b2ea7714161c70ad473a2dd1e7feb441d6805d43415b97d81f73" {
		t.Fatalf("unexpected witness %s", w.WitnessDigest)
	}
	if w.PostRevocationDecision != "REFUSED" || w.PostRevocationReason != "PERMIT_REVOKED" {
		t.Fatalf("unexpected witness state %+v", w)
	}
}

func TestExecutionRevocationV020TemporalBoundary(t *testing.T) {
	data := loadExecutionRevocationV020Fixture(t)
	var c ExecutionRevocationV020Contract
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		seq      int
		body     bool
		decision string
		reason   string
	}{
		{1, true, "READY", "EXECUTION_AUTHORIZED_NOT_EXECUTED"},
		{1, false, "HOLD", "EXECUTABLE_BODY_ABSENT"},
		{2, true, "REFUSED", "PERMIT_REVOKED"},
		{3, true, "REFUSED", "PERMIT_REVOKED"},
		{3, false, "REFUSED", "PERMIT_REVOKED"},
	}
	for _, tc := range cases {
		gotDecision, gotReason := EvaluateExecutionRevocationV020(c.Revocation, tc.seq, tc.body)
		if gotDecision != tc.decision || gotReason != tc.reason {
			t.Fatalf("seq=%d body=%v got %s/%s", tc.seq, tc.body, gotDecision, gotReason)
		}
	}
}

func TestExecutionRevocationV020EvaluatorRefusals(t *testing.T) {
	data := loadExecutionRevocationV020Fixture(t)
	var c ExecutionRevocationV020Contract
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		mutate   func(*ExecutionRevocationV020Receipt)
		decision string
		reason   string
	}{
		{"source mismatch", func(r *ExecutionRevocationV020Receipt) {
			r.SourceV019Receipt = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}, "REFUSED", "REVOCATION_SOURCE_MISMATCH"},
		{"target mismatch", func(r *ExecutionRevocationV020Receipt) { r.TargetPermitID = "other" }, "REFUSED", "REVOCATION_TARGET_MISMATCH"},
		{"scope escalation", func(r *ExecutionRevocationV020Receipt) {
			r.Scope = []string{"REVOKE_EXECUTE_DISPOSABLE_PROBE", "PROMOTE"}
		}, "REFUSED", "REVOCATION_SCOPE_EXCEEDED"},
		{"waldo self revocation", func(r *ExecutionRevocationV020Receipt) { r.WaldoIssued = true }, "REFUSED", "WALDO_CANNOT_SELF_ISSUE_REVOCATION"},
		{"delete history", func(r *ExecutionRevocationV020Receipt) { r.DeletePriorPermit = true }, "REFUSED", "HISTORY_REWRITE_FORBIDDEN"},
		{"grant authority", func(r *ExecutionRevocationV020Receipt) { r.GrantsExecution = true }, "REFUSED", "REVOCATION_CANNOT_GRANT_AUTHORITY"},
		{"order unresolved", func(r *ExecutionRevocationV020Receipt) { r.EffectiveSequence = 1 }, "HOLD", "REVOCATION_ORDER_UNRESOLVED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := c.Revocation
			tt.mutate(&r)
			decision, reason := EvaluateExecutionRevocationV020(r, 3, true)
			if decision != tt.decision || reason != tt.reason {
				t.Fatalf("got %s/%s", decision, reason)
			}
		})
	}
}

func TestExecutionRevocationV020ResealedSemanticTamperRejected(t *testing.T) {
	data := loadExecutionRevocationV020Fixture(t)
	mutators := []func(*ExecutionRevocationV020Contract){
		func(c *ExecutionRevocationV020Contract) { c.ProbeBoundary.RealRevocationObserved = true },
		func(c *ExecutionRevocationV020Contract) { c.Ledger[0].Outcome = "DELETED" },
		func(c *ExecutionRevocationV020Contract) { c.Ledger[1].RewritesPriorEntry = true },
		func(c *ExecutionRevocationV020Contract) { c.Revocation.DeletePriorPermit = true },
		func(c *ExecutionRevocationV020Contract) { c.Revocation.WaldoIssued = true },
		func(c *ExecutionRevocationV020Contract) { c.Truth.PastAuthorizationRewritten = true },
		func(c *ExecutionRevocationV020Contract) { c.Truth.ExecutionObserved = true },
		func(c *ExecutionRevocationV020Contract) { c.Promoted = true },
		func(c *ExecutionRevocationV020Contract) {
			c.SourceV019Head = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		},
	}
	for i, mutate := range mutators {
		var c ExecutionRevocationV020Contract
		if err := json.Unmarshal(data, &c); err != nil {
			t.Fatal(err)
		}
		mutate(&c)
		resealed, err := SealExecutionRevocationV020Contract(c)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := VerifyExecutionRevocationV020Contract(resealed); err == nil {
			t.Fatalf("tamper %d unexpectedly accepted", i)
		}
	}
}

func TestExecutionRevocationV020StaleReceiptRejected(t *testing.T) {
	data := loadExecutionRevocationV020Fixture(t)
	var c ExecutionRevocationV020Contract
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	c.ReceiptDigest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	tampered, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyExecutionRevocationV020Contract(tampered); err == nil {
		t.Fatal("stale receipt unexpectedly accepted")
	}
}

func TestExecutionRevocationV020UnknownFieldRejected(t *testing.T) {
	data := loadExecutionRevocationV020Fixture(t)
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatal(err)
	}
	obj["silentReauthorization"] = true
	tampered, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyExecutionRevocationV020Contract(tampered); err == nil {
		t.Fatal("unknown field unexpectedly accepted")
	}
}
