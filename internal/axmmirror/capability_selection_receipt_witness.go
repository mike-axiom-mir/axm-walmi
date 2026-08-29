package axmmirror

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

func WitnessSelectionReceiptV018(data []byte) (SelectionReceiptV018Witness, error) {
	c, err := VerifySelectionReceiptV018(data)
	if err != nil {
		return SelectionReceiptV018Witness{}, err
	}
	ids := make([]string, 0, len(c.RefusalCases))
	for _, x := range c.RefusalCases {
		ids = append(ids, x.CaseID)
	}
	sort.Strings(ids)
	w := SelectionReceiptV018Witness{
		Schema:                 "axm.waldo-witness.capability-selection-receipt/v0.18",
		SourceReceipt:          c.SourceV017Receipt,
		SourceHead:             c.SourceV017Head,
		CandidateSetDigest:     c.CandidateSetDigest,
		SelectedCandidateID:    c.SelectionReceipt.SelectedCandidateID,
		SelectionValidation:    c.SelectionValidation,
		OverallDecision:        c.OverallDecision,
		OverallReason:          c.OverallReason,
		ExecutionAuthority:     c.ExecutionAuthority,
		PreservedRefusalCaseID: ids,
	}
	raw, err := json.Marshal(w)
	if err != nil {
		return SelectionReceiptV018Witness{}, err
	}
	sum := sha256.Sum256(raw)
	w.WitnessDigest = hex.EncodeToString(sum[:])
	return w, nil
}

func SealSelectionReceiptV018(c SelectionReceiptV018Contract) ([]byte, error) {
	digest, err := selectionReceiptV018ExternalDigest(c)
	if err != nil {
		return nil, err
	}
	c.ReceiptDigest = digest
	return json.MarshalIndent(c, "", "  ")
}

func selectionReceiptV018ExternalDigest(c SelectionReceiptV018Contract) (string, error) {
	c.ReceiptDigest = ""
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func selectionReceiptV018SliceExact(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func selectionReceiptV018ValidSHA(v string) bool {
	if !strings.HasPrefix(v, "sha256:") || len(v) != 71 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(v, "sha256:"))
	return err == nil
}
