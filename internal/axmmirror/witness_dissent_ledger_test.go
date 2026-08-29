package axmmirror

import (
	"encoding/json"
	"os"
	"testing"
)

func loadWitnessDissentFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/witness-dissent-ledger-v0.14.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestWitnessDissentLedgerObservedReceipt(t *testing.T) {
	data := loadWitnessDissentFixture(t)
	r, err := VerifyWitnessDissentLedger(data)
	if err != nil {
		t.Fatal(err)
	}
	if r.Source.PeerHolds != 2 || len(r.AppendOnlyLedger) != 3 {
		t.Fatalf("unexpected topology: holds=%d entries=%d", r.Source.PeerHolds, len(r.AppendOnlyLedger))
	}
	w, err := WitnessWitnessDissentLedger(data)
	if err != nil {
		t.Fatal(err)
	}
	if w.State != "DISSENT_HISTORY_WITNESSED" || w.PeerHistoricalHolds != 2 || w.LedgerEntries != 3 || w.Authority != "NONE" {
		t.Fatalf("unexpected witness: %#v", w)
	}
}

func TestWitnessDissentLedgerRejectsHistoricalRewrite(t *testing.T) {
	data := loadWitnessDissentFixture(t)
	var r WitnessDissentLedgerReceipt
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	r.Resolution.WitnessBHistoricalHoldPreserved = false
	tampered, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyWitnessDissentLedger(tampered); err == nil {
		t.Fatal("expected historical rewrite rejection")
	}
}

func TestWitnessDissentLedgerRejectsFinalJudge(t *testing.T) {
	data := loadWitnessDissentFixture(t)
	var r WitnessDissentLedgerReceipt
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	r.Resolution.FinalJudgeAssigned = true
	tampered, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyWitnessDissentLedger(tampered); err == nil {
		t.Fatal("expected final judge rejection")
	}
}
