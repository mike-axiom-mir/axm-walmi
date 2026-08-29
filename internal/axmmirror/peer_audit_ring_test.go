package axmmirror

import (
	"encoding/json"
	"os"
	"testing"
)

func TestPeerAuditRingActualReceipt(t *testing.T) {
	data, err := os.ReadFile("testdata/peer-audit-ring-v0.13.json")
	if err != nil {
		t.Fatal(err)
	}
	s, err := VerifyPeerAuditRingReceipt(data)
	if err != nil {
		t.Fatal(err)
	}
	if s.TargetEvidencePasses != 36 || s.PeerHolds != 2 {
		t.Fatal("unexpected peer-audit totals")
	}
	w, err := WitnessPeerAuditRingReceipt(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
	if w.WitnessSHA256 != "1113f359f149214e48c20902a73aaffca612b0a54d35c79e8499806b91845739" {
		t.Fatalf("witness digest = %s", w.WitnessSHA256)
	}
}

func TestPeerAuditRingRejectsTamperedAuthority(t *testing.T) {
	data, err := os.ReadFile("testdata/peer-audit-ring-v0.13.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	raw["authority"] = "CANON"
	tampered, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyPeerAuditRingReceipt(tampered); err == nil {
		t.Fatal("tampered authority unexpectedly accepted")
	}
}
