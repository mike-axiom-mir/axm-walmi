package axmmirror

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSituatedInputFixturesPinInitialReceiptDigests(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		expected string
		build    func([]byte) (string, error)
	}{
		{
			name: "sensory evidence", file: "sensory-evidence-draft.json",
			expected: "5dfb7c0f27716b91d78d1ff998e0a18d94cc5265e8a107daa76d2bfe2f91541b",
			build: func(data []byte) (string, error) {
				var draft SensoryEvidenceDraft
				if err := DecodeStrictJSON(data, &draft); err != nil {
					return "", err
				}
				receipt, err := IntakeSensoryEvidence(draft)
				return receipt.ReceiptSHA256, err
			},
		},
		{
			name: "skill continuity", file: "skill-continuity-request.json",
			expected: "42b89ed2893035c6ed81e6547b9c329d2e450a8a6026afb69ab0c4e69c4feb4a",
			build: func(data []byte) (string, error) {
				var request SkillContinuityRequest
				if err := DecodeStrictJSON(data, &request); err != nil {
					return "", err
				}
				receipt, err := AssessSkillContinuity(request)
				return receipt.ReceiptSHA256, err
			},
		},
		{
			name: "dual discovery", file: "discovery-stance-request.json",
			expected: "e8fd044e11a9eb1f5684e9c8941a12f959e61c9b62daa1bdad61a4287e71af4a",
			build: func(data []byte) (string, error) {
				var request DiscoveryStanceRequest
				if err := DecodeStrictJSON(data, &request); err != nil {
					return "", err
				}
				packet, err := BuildDiscoveryStance(request)
				return packet.ReceiptSHA256, err
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "examples", "axm-mirror", test.file))
			if err != nil {
				t.Fatal(err)
			}
			digest, err := test.build(data)
			if err != nil {
				t.Fatal(err)
			}
			if digest != test.expected {
				t.Fatalf("receipt digest = %s, want %s", digest, test.expected)
			}
		})
	}
}
