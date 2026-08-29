package axmmirror

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func WitnessExecutionPermitV019Contract(data []byte) (ExecutionPermitV019Witness, error) {
	c, err := VerifyExecutionPermitV019Contract(data)
	if err != nil {
		return ExecutionPermitV019Witness{}, err
	}
	ids := make([]string, 0, len(c.RefusalCases))
	for _, rc := range c.RefusalCases {
		ids = append(ids, rc.CaseID)
	}
	w := ExecutionPermitV019Witness{
		Schema:                  "axm.waldo-witness.capability-execution-permit/v0.19",
		SourceReceipt:           c.SourceV018Receipt,
		SourceHead:              c.SourceV018Head,
		CandidateSetDigest:      c.CandidateSetDigest,
		SelectedCandidateID:     c.SelectedCandidateID,
		PermitID:                c.ExecutionPermit.PermitID,
		PermitValidation:        c.PermitValidation,
		RequestValidation:       c.RequestValidation,
		OverallDecision:         c.OverallDecision,
		OverallReason:           c.OverallReason,
		PreservedRefusalCaseIDs: ids,
	}
	raw, err := json.Marshal(w)
	if err != nil {
		return ExecutionPermitV019Witness{}, err
	}
	sum := sha256.Sum256(raw)
	w.WitnessDigest = hex.EncodeToString(sum[:])
	return w, nil
}
