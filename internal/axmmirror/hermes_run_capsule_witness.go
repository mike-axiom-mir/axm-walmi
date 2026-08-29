package axmmirror

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func WitnessHermesRunCapsuleV022Contract(data []byte) (HermesRunCapsuleV022Witness, error) {
	c, err := VerifyHermesRunCapsuleV022Contract(data)
	if err != nil {
		return HermesRunCapsuleV022Witness{}, err
	}
	w := HermesRunCapsuleV022Witness{
		Schema:              HermesRunCapsuleV022WitnessSchema,
		PlatformHead:        c.Source.PlatformHead,
		RuntimeGateRunID:    c.Source.RuntimeGateRunID,
		DonorFileCount:      len(c.Source.DonorFiles),
		CaseCount:           len(c.Cases),
		SourceReceiptSHA256: c.ReceiptDigest,
		Authority:           "NONE",
	}
	raw, err := json.Marshal(w)
	if err != nil {
		return HermesRunCapsuleV022Witness{}, err
	}
	sum := sha256.Sum256(raw)
	w.WitnessDigest = hex.EncodeToString(sum[:])
	return w, nil
}
