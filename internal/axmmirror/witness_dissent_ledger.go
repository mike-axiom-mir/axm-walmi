package axmmirror

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

const (
	WitnessDissentLedgerSchema        = "axm.waldo-mirror.v0.14-witness-dissent-ledger/v0.1"
	WitnessDissentLedgerStatus        = "OBSERVED_APPEND_ONLY_DISSENT"
	WitnessDissentLedgerWitnessSchema = "axm.waldo-witness.v0.14-witness-dissent-ledger/v0.1"
	WitnessDissentV013Receipt         = "sha256:9daf12733393e19c6149215e6a8f4f2d774f870ff21ba22c04efd785d34c8303"
	WitnessDissentV013Witness         = "1113f359f149214e48c20902a73aaffca612b0a54d35c79e8499806b91845739"
	WitnessDissentPeerPack            = "sha256:5ebd8a83f405243f501dcd4930e5279b86c4e39dbc082f66ffc6ccab26aaaaf7"
)

type WitnessDissentPeerHold struct {
	AuditorStrategy       string   `json:"auditorStrategy"`
	TargetAuditorStrategy string   `json:"targetAuditorStrategy"`
	Verdict               string   `json:"verdict"`
	Findings              []string `json:"findings"`
	Authority             string   `json:"authority"`
}

type WitnessDissentSource struct {
	V013ReceiptDigest                    string                   `json:"v013ReceiptDigest"`
	V013WitnessDigest                    string                   `json:"v013WitnessDigest"`
	PeerAuditPackDigest                  string                   `json:"peerAuditPackDigest"`
	PeerPasses                           int                      `json:"peerPasses"`
	PeerHolds                            int                      `json:"peerHolds"`
	IndependentWitnessSuppliedAtPeerVote bool                     `json:"independentWitnessSuppliedAtPeerVote"`
	GuardedPeerHolds                     []WitnessDissentPeerHold `json:"guardedPeerHolds"`
}

type WitnessDissentWitnessA struct {
	ReceiptDigest  string `json:"receiptDigest"`
	Implementation string `json:"implementation"`
	VerdictAtT1    string `json:"verdictAtT1"`
}

type WitnessDissentWitnessB struct {
	HoldReceiptDigest string `json:"holdReceiptDigest"`
	PassReceiptDigest string `json:"passReceiptDigest"`
	Implementation    string `json:"implementation"`
	VerdictAtT1       string `json:"verdictAtT1"`
	VerdictAtT2       string `json:"verdictAtT2"`
}

type WitnessDissentWitnesses struct {
	WitnessA WitnessDissentWitnessA `json:"witnessA"`
	WitnessB WitnessDissentWitnessB `json:"witnessB"`
}

type WitnessDissentEntry struct {
	Sequence              int      `json:"sequence"`
	Phase                 string   `json:"phase"`
	WitnessID             string   `json:"witnessId"`
	Implementation        string   `json:"implementation"`
	WitnessReceiptDigest  string   `json:"witnessReceiptDigest"`
	EvidenceViewDigest    string   `json:"evidenceViewDigest"`
	Verdict               string   `json:"verdict"`
	Findings              []string `json:"findings"`
	Authority             string   `json:"authority"`
	PreviousEntryDigest   string   `json:"previousEntryDigest"`
	SupersedesEntryDigest string   `json:"supersedesEntryDigest"`
	EntryDigest           string   `json:"entryDigest"`
}

type WitnessDissentResolution struct {
	T1WitnessDisagreementObserved            bool   `json:"t1WitnessDisagreementObserved"`
	WitnessBHoldResolvedByAdditionalEvidence bool   `json:"witnessBHoldResolvedByAdditionalEvidence"`
	WitnessBHistoricalHoldPreserved          bool   `json:"witnessBHistoricalHoldPreserved"`
	PeerGuardedHoldsPreserved                bool   `json:"peerGuardedHoldsPreserved"`
	PeerHistoryRewritten                     bool   `json:"peerHistoryRewritten"`
	WitnessHistoryRewritten                  bool   `json:"witnessHistoryRewritten"`
	MajorityAuthorityGranted                 bool   `json:"majorityAuthorityGranted"`
	FinalJudgeAssigned                       bool   `json:"finalJudgeAssigned"`
	FinalVerdict                             string `json:"finalVerdict"`
	Authority                                string `json:"authority"`
}

type WitnessDissentTruth struct {
	WitnessImplementationsIndependent bool `json:"witnessImplementationsIndependent"`
	WitnessEvidenceViewsDifferentAtT1 bool `json:"witnessEvidenceViewsDifferentAtT1"`
	LiveAIProviderCalled              bool `json:"liveAIProviderCalled"`
	NetworkRequestedByHarness         bool `json:"networkRequestedByHarness"`
	TargetCodeMutated                 bool `json:"targetCodeMutated"`
	PeerReceiptMutated                bool `json:"peerReceiptMutated"`
	InstallationPerformed             bool `json:"installationPerformed"`
	RegistrationPerformed             bool `json:"registrationPerformed"`
	StagingPerformed                  bool `json:"stagingPerformed"`
	PromotionPerformed                bool `json:"promotionPerformed"`
	CanonChanged                      bool `json:"canonChanged"`
}

type WitnessDissentLedgerReceipt struct {
	Schema               string                   `json:"schema"`
	Status               string                   `json:"status"`
	Source               WitnessDissentSource     `json:"source"`
	IndependentWitnesses WitnessDissentWitnesses  `json:"independentWitnesses"`
	AppendOnlyLedger     []WitnessDissentEntry    `json:"appendOnlyLedger"`
	Resolution           WitnessDissentResolution `json:"resolution"`
	Truth                WitnessDissentTruth      `json:"truth"`
	Authority            string                   `json:"authority"`
	ReceiptDigest        string                   `json:"receiptDigest"`
}

type WitnessDissentLedgerWitness struct {
	Schema                    string `json:"schema"`
	State                     string `json:"state"`
	SourceReceiptDigest       string `json:"source_receipt_digest"`
	SourceWitnessDigest       string `json:"source_witness_digest"`
	PeerHistoricalHolds       int    `json:"peer_historical_holds"`
	LedgerEntries             int    `json:"ledger_entries"`
	T1IndependentDisagreement bool   `json:"t1_independent_disagreement"`
	HistoricalHoldPreserved   bool   `json:"historical_hold_preserved"`
	WitnessBResolvedAtT2      bool   `json:"witness_b_resolved_at_t2"`
	LedgerHeadDigest          string `json:"ledger_head_digest"`
	Authority                 string `json:"authority"`
	WitnessSHA256             string `json:"witness_sha256,omitempty"`
}

func VerifyWitnessDissentLedger(data []byte) (WitnessDissentLedgerReceipt, error) {
	var r WitnessDissentLedgerReceipt
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return r, fmt.Errorf("decode v0.14 dissent ledger: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return r, errors.New("v0.14 dissent ledger has trailing JSON")
	}
	if err := r.Validate(); err != nil {
		return WitnessDissentLedgerReceipt{}, err
	}
	var generic map[string]any
	dec2 := json.NewDecoder(bytes.NewReader(data))
	dec2.UseNumber()
	if err := dec2.Decode(&generic); err != nil {
		return r, err
	}
	expected, err := witnessDissentDigestMap(generic, "receiptDigest")
	if err != nil {
		return r, err
	}
	if expected != r.ReceiptDigest {
		return r, fmt.Errorf("v0.14 receipt digest mismatch: expected %s got %s", expected, r.ReceiptDigest)
	}
	return r, nil
}

func (r WitnessDissentLedgerReceipt) Validate() error {
	if r.Schema != WitnessDissentLedgerSchema || r.Status != WitnessDissentLedgerStatus || r.Authority != "NONE" {
		return errors.New("v0.14 schema/status/authority mismatch")
	}
	s := r.Source
	if s.V013ReceiptDigest != WitnessDissentV013Receipt || s.V013WitnessDigest != WitnessDissentV013Witness || s.PeerAuditPackDigest != WitnessDissentPeerPack || s.PeerPasses != 4 || s.PeerHolds != 2 || s.IndependentWitnessSuppliedAtPeerVote {
		return errors.New("v0.14 source binding or peer counts drifted")
	}
	if len(s.GuardedPeerHolds) != 2 {
		return errors.New("v0.14 must preserve exactly two original guarded holds")
	}
	seen := map[string]bool{}
	for _, h := range s.GuardedPeerHolds {
		if h.AuditorStrategy != "guarded" || h.Verdict != "HOLD" || h.Authority != "NONE" || len(h.Findings) != 1 || h.Findings[0] != "INDEPENDENT_WITNESS_MISSING" {
			return errors.New("v0.14 historical peer hold changed")
		}
		if h.TargetAuditorStrategy != "lean" && h.TargetAuditorStrategy != "balanced" {
			return errors.New("v0.14 unexpected historical hold target")
		}
		seen[h.TargetAuditorStrategy] = true
	}
	if !seen["lean"] || !seen["balanced"] {
		return errors.New("v0.14 did not preserve both original hold edges")
	}
	wa, wb := r.IndependentWitnesses.WitnessA, r.IndependentWitnesses.WitnessB
	if wa.Implementation != "go-structural-v1" || wa.VerdictAtT1 != "EVIDENCE_PASS" || wb.Implementation != "python-material-v1" || wb.VerdictAtT1 != "HOLD" || wb.VerdictAtT2 != "EVIDENCE_PASS" {
		return errors.New("v0.14 independent witness transition mismatch")
	}
	for _, d := range []string{wa.ReceiptDigest, wb.HoldReceiptDigest, wb.PassReceiptDigest, r.ReceiptDigest} {
		if err := witnessDissentSHA(d); err != nil {
			return err
		}
	}
	if len(r.AppendOnlyLedger) != 3 {
		return errors.New("v0.14 ledger must retain all three observations")
	}
	wantID := []string{"witness-a-go-structural", "witness-b-python-material", "witness-b-python-material"}
	wantPhase := []string{"T1", "T1", "T2"}
	wantVerdict := []string{"EVIDENCE_PASS", "HOLD", "EVIDENCE_PASS"}
	prev := "GENESIS"
	for i, e := range r.AppendOnlyLedger {
		if e.Sequence != i+1 || e.WitnessID != wantID[i] || e.Phase != wantPhase[i] || e.Verdict != wantVerdict[i] || e.Authority != "NONE" || e.PreviousEntryDigest != prev {
			return fmt.Errorf("v0.14 ledger entry %d topology drifted", i+1)
		}
		if err := witnessDissentSHA(e.WitnessReceiptDigest); err != nil {
			return err
		}
		if err := witnessDissentSHA(e.EvidenceViewDigest); err != nil {
			return err
		}
		if i == 1 {
			if len(e.Findings) != 1 || e.Findings[0] != "MATERIAL_EVIDENCE_VIEW_INCOMPLETE" {
				return errors.New("v0.14 T1 material HOLD finding missing")
			}
		} else if len(e.Findings) != 0 {
			return errors.New("v0.14 unexpected witness findings")
		}
		if i < 2 && e.SupersedesEntryDigest != "" {
			return errors.New("v0.14 early observation cannot supersede history")
		}
		if i == 2 && e.SupersedesEntryDigest != r.AppendOnlyLedger[1].EntryDigest {
			return errors.New("v0.14 T2 resolution must point at retained T1 HOLD")
		}
		expected, err := witnessDissentEntryDigest(e)
		if err != nil {
			return err
		}
		if expected != e.EntryDigest {
			return fmt.Errorf("v0.14 ledger entry %d digest mismatch", i+1)
		}
		prev = e.EntryDigest
	}
	if r.AppendOnlyLedger[0].WitnessReceiptDigest != wa.ReceiptDigest || r.AppendOnlyLedger[1].WitnessReceiptDigest != wb.HoldReceiptDigest || r.AppendOnlyLedger[2].WitnessReceiptDigest != wb.PassReceiptDigest {
		return errors.New("v0.14 ledger/witness receipt binding mismatch")
	}
	z := r.Resolution
	if !z.T1WitnessDisagreementObserved || !z.WitnessBHoldResolvedByAdditionalEvidence || !z.WitnessBHistoricalHoldPreserved || !z.PeerGuardedHoldsPreserved || z.PeerHistoryRewritten || z.WitnessHistoryRewritten || z.MajorityAuthorityGranted || z.FinalJudgeAssigned || z.FinalVerdict != "NONE" || z.Authority != "NONE" {
		return errors.New("v0.14 resolution rewrote dissent or granted authority")
	}
	t := r.Truth
	if !t.WitnessImplementationsIndependent || !t.WitnessEvidenceViewsDifferentAtT1 || t.LiveAIProviderCalled || t.NetworkRequestedByHarness || t.TargetCodeMutated || t.PeerReceiptMutated || t.InstallationPerformed || t.RegistrationPerformed || t.StagingPerformed || t.PromotionPerformed || t.CanonChanged {
		return errors.New("v0.14 truth boundary drifted")
	}
	return nil
}

func WitnessWitnessDissentLedger(data []byte) (WitnessDissentLedgerWitness, error) {
	r, err := VerifyWitnessDissentLedger(data)
	if err != nil {
		return WitnessDissentLedgerWitness{}, err
	}
	w := WitnessDissentLedgerWitness{Schema: WitnessDissentLedgerWitnessSchema, State: "DISSENT_HISTORY_WITNESSED", SourceReceiptDigest: r.Source.V013ReceiptDigest, SourceWitnessDigest: r.Source.V013WitnessDigest, PeerHistoricalHolds: r.Source.PeerHolds, LedgerEntries: len(r.AppendOnlyLedger), T1IndependentDisagreement: true, HistoricalHoldPreserved: true, WitnessBResolvedAtT2: true, LedgerHeadDigest: r.AppendOnlyLedger[len(r.AppendOnlyLedger)-1].EntryDigest, Authority: "NONE"}
	w.WitnessSHA256 = ""
	b, err := json.Marshal(w)
	if err != nil {
		return w, err
	}
	sum := sha256.Sum256(b)
	w.WitnessSHA256 = hex.EncodeToString(sum[:])
	return w, nil
}

func witnessDissentEntryDigest(e WitnessDissentEntry) (string, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	var m map[string]any
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if err = d.Decode(&m); err != nil {
		return "", err
	}
	return witnessDissentDigestMap(m, "entryDigest")
}
func witnessDissentSHA(s string) error {
	if len(s) != 71 || !strings.HasPrefix(s, "sha256:") {
		return fmt.Errorf("expected sha256-prefixed digest, got %q", s)
	}
	_, err := hex.DecodeString(s[7:])
	return err
}
func witnessDissentDigestMap(m map[string]any, omit string) (string, error) {
	c := make(map[string]any, len(m))
	for k, v := range m {
		if k != omit {
			c[k] = v
		}
	}
	b, err := witnessDissentCanonical(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
func witnessDissentCanonical(v any) ([]byte, error) {
	switch x := v.(type) {
	case nil:
		return []byte("null"), nil
	case bool:
		if x {
			return []byte("true"), nil
		}
		return []byte("false"), nil
	case string:
		return json.Marshal(x)
	case json.Number:
		s := x.String()
		if strings.ContainsAny(s, ".eE") {
			return nil, errors.New("canonical v0.14 JSON accepts integer numbers only")
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, err
		}
		return []byte(strconv.FormatInt(n, 10)), nil
	case []any:
		var b bytes.Buffer
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			z, err := witnessDissentCanonical(e)
			if err != nil {
				return nil, err
			}
			b.Write(z)
		}
		b.WriteByte(']')
		return b.Bytes(), nil
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b bytes.Buffer
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			q, _ := json.Marshal(k)
			b.Write(q)
			b.WriteByte(':')
			z, err := witnessDissentCanonical(x[k])
			if err != nil {
				return nil, err
			}
			b.Write(z)
		}
		b.WriteByte('}')
		return b.Bytes(), nil
	default:
		return nil, fmt.Errorf("unsupported canonical JSON type %T", v)
	}
}
