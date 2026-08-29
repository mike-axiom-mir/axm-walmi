package axmmirror

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func WitnessOpinionArtifactV023Contract(data []byte) (OpinionArtifactV023Witness, error) {
	c, err := VerifyOpinionArtifactV023Contract(data)
	if err != nil {
		return OpinionArtifactV023Witness{}, err
	}
	w := OpinionArtifactV023Witness{
		Schema:              OpinionArtifactV023WitnessSchema,
		ParentReceipt:       c.ParentReceipt,
		CaseCount:           len(c.Cases),
		SourceReceiptSHA256: c.ReceiptDigest,
		Authority:           "NONE",
	}
	for _, tc := range c.Cases {
		w.EvidenceRefCount += len(tc.Evidence)
		w.DissentRefCount += len(tc.Dissent)
		w.UncertaintyMarkerCount += len(tc.Uncertainty)
	}
	raw, err := json.Marshal(w)
	if err != nil {
		return OpinionArtifactV023Witness{}, err
	}
	sum := sha256.Sum256(raw)
	w.WitnessDigest = hex.EncodeToString(sum[:])
	return w, nil
}
