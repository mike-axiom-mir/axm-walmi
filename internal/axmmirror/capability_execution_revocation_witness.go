package axmmirror

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func WitnessExecutionRevocationV020Contract(data []byte) (ExecutionRevocationV020Witness, error) {
	c, err := VerifyExecutionRevocationV020Contract(data)
	if err != nil {
		return ExecutionRevocationV020Witness{}, err
	}
	eventIDs := make([]string, 0, len(c.Ledger))
	for _, e := range c.Ledger {
		eventIDs = append(eventIDs, e.EventID)
	}
	refusalIDs := make([]string, 0, len(c.RefusalCases))
	for _, r := range c.RefusalCases {
		refusalIDs = append(refusalIDs, r.CaseID)
	}
	post := c.Scenarios[2]
	w := ExecutionRevocationV020Witness{
		Schema:                  "axm.waldo-witness.capability-execution-revocation/v0.20",
		SourceReceipt:           c.SourceV019Receipt,
		SourceHead:              c.SourceV019Head,
		PermitID:                c.PermitID,
		RevocationReceiptID:     c.Revocation.ReceiptID,
		LedgerEventIDs:          eventIDs,
		CurrentState:            c.CurrentState,
		PostRevocationDecision:  post.ExpectedDecision,
		PostRevocationReason:    post.ExpectedReason,
		PreservedRefusalCaseIDs: refusalIDs,
	}
	raw, err := json.Marshal(w)
	if err != nil {
		return ExecutionRevocationV020Witness{}, err
	}
	sum := sha256.Sum256(raw)
	w.WitnessDigest = hex.EncodeToString(sum[:])
	return w, nil
}
