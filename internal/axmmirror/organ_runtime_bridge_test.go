package axmmirror

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOrganRuntimeBridgeActualV011Receipt(t *testing.T) {
	data, err := os.ReadFile("testdata/organ-runtime-bridge-v0.11.json")
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := VerifyOrganRuntimeReceipt(data)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Totals.ProposalCount != 4 || receipt.Totals.CandidateCount != 12 || receipt.Totals.FailureCount != 0 {
		t.Fatal("observed Organ runtime totals drifted")
	}
	witness, err := WitnessOrganRuntimeReceipt(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := witness.Validate(); err != nil {
		t.Fatal(err)
	}
	if witness.WitnessSHA256 != "2235bcfbdc1beefd4b3ec3835e1ad96eaf50433f58a393e3c1d102ad0c644db9" {
		t.Fatalf("unexpected Organ runtime witness digest %s", witness.WitnessSHA256)
	}
}

func TestOrganRuntimeBridgeRejectsTamper(t *testing.T) {
	data, err := os.ReadFile("testdata/organ-runtime-bridge-v0.11.json")
	if err != nil {
		t.Fatal(err)
	}
	var receipt OrganRuntimeReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Totals.CandidateCount != 12 {
		t.Fatalf("unexpected fixture candidate count %d", receipt.Totals.CandidateCount)
	}
	receipt.Totals.CandidateCount = 11
	tampered, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyOrganRuntimeReceipt(tampered); err == nil {
		t.Fatal("tampered Organ runtime receipt passed")
	}
}
