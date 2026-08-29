package axmmirror

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func WitnessCoupledReasoningV024Contract(data []byte) (CoupledReasoningV024Witness, error) {
	c, err := VerifyCoupledReasoningV024Contract(data)
	if err != nil {
		return CoupledReasoningV024Witness{}, err
	}
	w := CoupledReasoningV024Witness{
		Schema:              CoupledReasoningV024WitnessSchema,
		ParentReceipt:       c.ParentReceipt,
		CaseCount:           len(c.Cases),
		SourceReceiptSHA256: c.ReceiptDigest,
		Authority:           "NONE",
	}
	for _, tc := range c.Cases {
		w.EventCount += len(tc.Events)
		if coupledReasoningV024HasKind(tc.Events, "REPLAN") {
			w.ReplanCaseCount++
		}
		if tc.ExecutionRequested {
			w.ExecutionRequestCount++
		}
		o, _ := EvaluateCoupledReasoningV024Case(tc)
		if o == "READY" {
			w.ReadyCaseCount++
		}
	}
	raw, err := json.Marshal(w)
	if err != nil {
		return CoupledReasoningV024Witness{}, err
	}
	sum := sha256.Sum256(raw)
	w.WitnessDigest = hex.EncodeToString(sum[:])
	return w, nil
}
