package axmmirror

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

func TestCapabilityFabricContractProbeActualFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/capability-fabric-contract-probe-v0.15.json")
	if err != nil {
		t.Fatal(err)
	}
	probe, err := VerifyCapabilityFabricContractProbe(data)
	if err != nil {
		t.Fatal(err)
	}
	if probe.ReceiptDigest != "sha256:75e6466701520c7aeab56e76c719ab275519e8ed2fffa5fceaf3310cce4ec0bf" {
		t.Fatalf("unexpected receipt digest %s", probe.ReceiptDigest)
	}
	w, err := WitnessCapabilityFabricContractProbe(data)
	if err != nil {
		t.Fatal(err)
	}
	if w.ProposalCount != 4 || w.RefusalCount != 1 {
		t.Fatalf("unexpected witness topology: %+v", w)
	}
}

func TestCapabilityFabricContractProbeRejectsSemanticTampering(t *testing.T) {
	data := capabilityFabricProbeFixture(t)

	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{
			name: "claims actual local fabric execution",
			mutate: func(root map[string]any) {
				root["probeBoundary"].(map[string]any)["actualCapabilityFabricExecutionObserved"] = true
			},
		},
		{
			name: "hand asks for write authority",
			mutate: func(root map[string]any) {
				rows := root["embodiments"].([]any)
				for _, raw := range rows {
					e := raw.(map[string]any)
					if e["bodyKind"] == "HAND" {
						e["requiresAuthority"] = []any{"READ_ONLY_PUBLIC_ARTIFACT_ACCESS", "WRITE_WORKSPACE"}
						return
					}
				}
				t.Fatal("HAND embodiment not found")
			},
		},
		{
			name: "selects an embodiment",
			mutate: func(root map[string]any) {
				root["decision"].(map[string]any)["selectionPerformed"] = true
				root["decision"].(map[string]any)["preferredEmbodiment"] = "HAND"
			},
		},
		{
			name: "makes Waldo own capability fabric",
			mutate: func(root map[string]any) {
				root["truth"].(map[string]any)["waldoOwnsCapabilityFabric"] = true
			},
		},
		{
			name: "rewrites prior routing history",
			mutate: func(root map[string]any) {
				root["routingAdjustment"].(map[string]any)["previousPacketRewritten"] = true
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var root map[string]any
			if err := json.Unmarshal(data, &root); err != nil {
				t.Fatal(err)
			}
			tc.mutate(root)
			resealCapabilityFabricProbe(t, root)
			changed, err := json.Marshal(root)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := VerifyCapabilityFabricContractProbe(changed); err == nil {
				t.Fatal("tampered contract probe unexpectedly verified")
			}
		})
	}
}

func capabilityFabricProbeFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/capability-fabric-contract-probe-v0.15.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func resealCapabilityFabricProbe(t *testing.T, root map[string]any) {
	t.Helper()
	delete(root, "receiptDigest")
	canonical, err := encodePeerCanonical(root)
	if err != nil {
		t.Fatal(err)
	}
	sum := capProbeSHA256Bytes(canonical)
	root["receiptDigest"] = "sha256:" + sum
}

func capProbeSHA256Bytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
