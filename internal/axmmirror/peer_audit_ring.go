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
	PeerAuditRingSchema        = "axm.waldo-mirror.v0.13-generated-peer-audit-ring/v0.1"
	PeerAuditRingStatus        = "OBSERVED_DISSENT_PRESERVED"
	PeerAuditRingWitnessSchema = "axm.waldo-witness.generated-peer-audit-ring/v0.1"
	PeerAuditRingV012Digest    = "sha256:5fcf0530cb97e6568793baaae9c0a9ef4cc4285c235a590665986b27460799a6"
	PeerAuditRingPacketSHA256  = "sha256:8f4688db3abcba988011f2a2f36f172a09c3d848e57a2da54db90abd2669e634"
	PeerAuditRingPackDigest    = "sha256:5ebd8a83f405243f501dcd4930e5279b86c4e39dbc082f66ffc6ccab26aaaaf7"
)

type PeerAuditRingSummary struct {
	ReceiptSHA256          string
	AuditorRunSHA256       string
	AuditorCount           int
	TargetAuditExecutions  int
	TargetEvidencePasses   int
	PeerAuditExecutions    int
	PeerEvidencePasses     int
	PeerHolds              int
	DissentCases           int
	IndependentWitnessUsed bool
}

type PeerAuditRingWitness struct {
	Schema                       string   `json:"schema"`
	State                        string   `json:"state"`
	SourceReceiptSHA256          string   `json:"source_receipt_sha256"`
	V012ReceiptSHA256            string   `json:"v012_receipt_sha256"`
	PeerAuditPackSHA256          string   `json:"peer_audit_pack_sha256"`
	AuditorRunSHA256             string   `json:"auditor_run_sha256"`
	AuditorCount                 int      `json:"auditor_count"`
	TargetAuditExecutions        int      `json:"target_audit_executions"`
	TargetEvidencePasses         int      `json:"target_evidence_passes"`
	PeerAuditExecutions          int      `json:"peer_audit_executions"`
	PeerEvidencePasses           int      `json:"peer_evidence_passes"`
	PeerHolds                    int      `json:"peer_holds"`
	DissentCases                 int      `json:"dissent_cases"`
	IndependentWitnessWasMissing bool     `json:"independent_witness_was_missing"`
	Notices                      []string `json:"notices"`
	Authority                    string   `json:"authority"`
	WitnessSHA256                string   `json:"witness_sha256,omitempty"`
}

func VerifyPeerAuditRingReceipt(data []byte) (PeerAuditRingSummary, error) {
	root, err := peerDecodeObject(data)
	if err != nil {
		return PeerAuditRingSummary{}, err
	}
	if err := peerExactKeys(root, "schema", "status", "source", "peerAuditPack", "auditorGeneration", "targetAudit", "dissent", "peerRing", "interpretation", "truth", "authority", "receiptDigest"); err != nil {
		return PeerAuditRingSummary{}, err
	}
	if peerString(root, "schema") != PeerAuditRingSchema || peerString(root, "status") != PeerAuditRingStatus || peerString(root, "authority") != "NONE" {
		return PeerAuditRingSummary{}, errors.New("peer audit ring schema/status/authority mismatch")
	}

	source := peerObject(root, "source")
	if peerString(source, "v012PatchedReceiptDigest") != PeerAuditRingV012Digest || peerString(source, "sourcePacketSha256") != PeerAuditRingPacketSHA256 || peerString(source, "patchedFactoryVersion") != "1.0.1-v012-parity" {
		return PeerAuditRingSummary{}, errors.New("peer audit source binding mismatch")
	}
	if err := validatePeerSHA("runtime", peerString(source, "patchedRuntimeDigest")); err != nil {
		return PeerAuditRingSummary{}, err
	}
	pack := peerObject(root, "peerAuditPack")
	if peerString(pack, "id") != "peer-evidence-audit" || peerString(pack, "version") != "0.1.0" || peerString(pack, "digest") != PeerAuditRingPackDigest {
		return PeerAuditRingSummary{}, errors.New("peer audit pack binding mismatch")
	}

	gen := peerObject(root, "auditorGeneration")
	auditors := peerArray(gen, "auditors")
	if peerInt(gen, "candidateCount") != 3 || peerInt(gen, "failureCount") != 0 || len(auditors) != 3 {
		return PeerAuditRingSummary{}, errors.New("expected three auditors and zero generation failures")
	}
	runDigest := peerString(gen, "runDigest")
	if err := validatePeerSHA("auditor run", runDigest); err != nil {
		return PeerAuditRingSummary{}, err
	}
	strategies := map[string]bool{}
	packages := map[string]bool{}
	for _, raw := range auditors {
		a := peerAsObject(raw)
		s := peerString(a, "strategy")
		if !peerStrategy(s) || strategies[s] {
			return PeerAuditRingSummary{}, errors.New("auditor strategy set invalid")
		}
		strategies[s] = true
		pkg := peerString(a, "packageDigest")
		if packages[pkg] {
			return PeerAuditRingSummary{}, errors.New("duplicate auditor package")
		}
		packages[pkg] = true
		for n, d := range map[string]string{"package": pkg, "definition": peerString(a, "definitionDigest"), "evaluation": peerString(a, "evaluationDigest")} {
			if err := validatePeerSHA(n, d); err != nil {
				return PeerAuditRingSummary{}, err
			}
		}
		e := peerObject(a, "evidence")
		if !peerBool(e, "packageVerified") || !peerBool(e, "selftestPassed") || !peerBool(e, "positiveParity") || !peerBool(e, "closedInputParity") || !peerBool(e, "authorityClosed") || !peerBool(e, "lineageBound") || peerBool(e, "independentWitness") || peerString(e, "targetKind") != "auditor-candidate" {
			return PeerAuditRingSummary{}, fmt.Errorf("auditor %s evidence boundary drifted", s)
		}
	}

	target := peerObject(root, "targetAudit")
	targetAudits := peerArray(target, "audits")
	if peerInt(target, "targetCandidates") != 12 || peerInt(target, "auditExecutions") != 36 || peerInt(target, "evidencePasses") != 36 || peerInt(target, "holds") != 0 || len(targetAudits) != 36 {
		return PeerAuditRingSummary{}, errors.New("target audit totals mismatch")
	}
	perAuditor := map[string]int{}
	seenTarget := map[string]bool{}
	for _, raw := range targetAudits {
		a := peerAsObject(raw)
		s, targetID := peerString(a, "auditorStrategy"), peerString(a, "targetId")
		if !peerStrategy(s) || strings.TrimSpace(targetID) == "" || !peerStrategy(peerString(a, "targetStrategy")) || peerString(a, "verdict") != "EVIDENCE_PASS" || len(peerStrings(a, "findings")) != 0 || peerString(a, "authority") != "NONE" {
			return PeerAuditRingSummary{}, errors.New("known-good target audit boundary drifted")
		}
		key := s + "|" + targetID
		if seenTarget[key] {
			return PeerAuditRingSummary{}, errors.New("duplicate target audit")
		}
		seenTarget[key], perAuditor[s] = true, perAuditor[s]+1
	}
	for _, s := range []string{"lean", "balanced", "guarded"} {
		if perAuditor[s] != 12 {
			return PeerAuditRingSummary{}, errors.New("each auditor must review all 12 targets")
		}
	}

	dissent := peerArray(peerObject(root, "dissent"), "cases")
	want := map[string][3]string{
		"closed-input-parity-missing": {"EVIDENCE_PASS", "HOLD", "HOLD"},
		"authority-open":              {"EVIDENCE_PASS", "EVIDENCE_PASS", "HOLD"},
		"lineage-unbound":             {"EVIDENCE_PASS", "EVIDENCE_PASS", "HOLD"},
		"independent-witness-missing": {"EVIDENCE_PASS", "EVIDENCE_PASS", "HOLD"},
	}
	if len(dissent) != len(want) {
		return PeerAuditRingSummary{}, errors.New("dissent case count mismatch")
	}
	seenDissent := map[string]bool{}
	for _, raw := range dissent {
		c := peerAsObject(raw)
		id := peerString(c, "caseId")
		expected, ok := want[id]
		if !ok || seenDissent[id] {
			return PeerAuditRingSummary{}, errors.New("dissent identity mismatch")
		}
		seenDissent[id] = true
		verdicts := peerObject(c, "verdicts")
		for i, s := range []string{"lean", "balanced", "guarded"} {
			v := peerObject(verdicts, s)
			if peerString(v, "verdict") != expected[i] || peerString(v, "authority") != "NONE" || len(peerStrings(v, "findings")) == 0 {
				return PeerAuditRingSummary{}, fmt.Errorf("dissent verdict drift in %s", id)
			}
		}
	}

	ring := peerObject(root, "peerRing")
	peerAudits := peerArray(ring, "audits")
	if peerInt(ring, "auditExecutions") != 6 || peerInt(ring, "evidencePasses") != 4 || peerInt(ring, "holds") != 2 || peerBool(ring, "independentWitnessSupplied") || len(peerAudits) != 6 {
		return PeerAuditRingSummary{}, errors.New("peer ring totals mismatch")
	}
	seenPeer := map[string]bool{}
	guardedHolds := 0
	for _, raw := range peerAudits {
		a := peerAsObject(raw)
		s, t := peerString(a, "auditorStrategy"), peerString(a, "targetAuditorStrategy")
		if !peerStrategy(s) || !peerStrategy(t) || s == t || peerString(a, "authority") != "NONE" || !peerContains(peerStrings(a, "findings"), "INDEPENDENT_WITNESS_MISSING") {
			return PeerAuditRingSummary{}, errors.New("peer audit boundary drifted")
		}
		key := s + "|" + t
		if seenPeer[key] {
			return PeerAuditRingSummary{}, errors.New("duplicate peer audit")
		}
		seenPeer[key] = true
		if s == "guarded" {
			if peerString(a, "verdict") != "HOLD" {
				return PeerAuditRingSummary{}, errors.New("guarded peer audit must hold without independent witness")
			}
			guardedHolds++
		} else if peerString(a, "verdict") != "EVIDENCE_PASS" {
			return PeerAuditRingSummary{}, errors.New("lean/balanced peer verdict drifted")
		}
	}
	if guardedHolds != 2 {
		return PeerAuditRingSummary{}, errors.New("expected two guarded peer holds")
	}

	truth := peerObject(root, "truth")
	if !peerBool(truth, "generatedAuditorCodeExecuted") || peerBool(truth, "targetCodeMutated") || peerBool(truth, "targetPackagesSelected") || peerBool(truth, "liveAIProviderCalled") || peerBool(truth, "networkRequestedByHarness") || !peerBool(truth, "executionRootsDisposable") || peerBool(truth, "installationPerformed") || peerBool(truth, "registrationPerformed") || peerBool(truth, "stagingPerformed") || peerBool(truth, "promotionPerformed") || peerBool(truth, "canonChanged") || peerBool(truth, "quorumAuthority") {
		return PeerAuditRingSummary{}, errors.New("peer audit truth boundary drifted")
	}

	receiptDigest := peerString(root, "receiptDigest")
	if err := validatePeerSHA("receipt", receiptDigest); err != nil {
		return PeerAuditRingSummary{}, err
	}
	expectedDigest, err := peerAuditExternalDigest(root)
	if err != nil {
		return PeerAuditRingSummary{}, err
	}
	if expectedDigest != receiptDigest {
		return PeerAuditRingSummary{}, errors.New("peer audit receipt digest mismatch")
	}
	return PeerAuditRingSummary{ReceiptSHA256: receiptDigest, AuditorRunSHA256: runDigest, AuditorCount: 3, TargetAuditExecutions: 36, TargetEvidencePasses: 36, PeerAuditExecutions: 6, PeerEvidencePasses: 4, PeerHolds: 2, DissentCases: 4, IndependentWitnessUsed: false}, nil
}

func WitnessPeerAuditRingReceipt(data []byte) (PeerAuditRingWitness, error) {
	s, err := VerifyPeerAuditRingReceipt(data)
	if err != nil {
		return PeerAuditRingWitness{}, err
	}
	w := PeerAuditRingWitness{
		Schema: PeerAuditRingWitnessSchema, State: "PEER_AUDIT_RING_WITNESSED", SourceReceiptSHA256: s.ReceiptSHA256,
		V012ReceiptSHA256: PeerAuditRingV012Digest, PeerAuditPackSHA256: PeerAuditRingPackDigest, AuditorRunSHA256: s.AuditorRunSHA256,
		AuditorCount: 3, TargetAuditExecutions: 36, TargetEvidencePasses: 36, PeerAuditExecutions: 6, PeerEvidencePasses: 4, PeerHolds: 2, DissentCases: 4,
		IndependentWitnessWasMissing: true,
		Notices: []string{
			"Generated auditors only consume typed evidence; they do not gain target mutation authority.",
			"The guarded auditor preserved a HOLD when peer evidence lacked an independent witness.",
			"No majority or quorum result authorizes selection, installation, promotion, merge, or CANON.",
		}, Authority: "NONE",
	}
	w.WitnessSHA256, err = peerWitnessDigest(w)
	return w, err
}

func (w PeerAuditRingWitness) Validate() error {
	if w.Schema != PeerAuditRingWitnessSchema || w.State != "PEER_AUDIT_RING_WITNESSED" || w.Authority != "NONE" || w.AuditorCount != 3 || w.TargetAuditExecutions != 36 || w.TargetEvidencePasses != 36 || w.PeerAuditExecutions != 6 || w.PeerEvidencePasses != 4 || w.PeerHolds != 2 || w.DissentCases != 4 || !w.IndependentWitnessWasMissing || len(w.Notices) < 3 {
		return errors.New("peer audit witness boundary drifted")
	}
	for name, digest := range map[string]string{"source": w.SourceReceiptSHA256, "v012": w.V012ReceiptSHA256, "pack": w.PeerAuditPackSHA256, "run": w.AuditorRunSHA256} {
		if err := validatePeerSHA(name, digest); err != nil {
			return err
		}
	}
	if len(w.WitnessSHA256) != 64 {
		return errors.New("peer audit witness digest length invalid")
	}
	expected, err := peerWitnessDigest(w)
	if err != nil {
		return err
	}
	if expected != w.WitnessSHA256 {
		return errors.New("peer audit witness digest mismatch")
	}
	return nil
}

func peerDecodeObject(data []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, errors.New("peer audit receipt has trailing JSON")
	}
	return peerAsObject(v), nil
}
func peerExactKeys(m map[string]any, keys ...string) error {
	if len(m) != len(keys) {
		return errors.New("peer audit receipt top-level shape drifted")
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return fmt.Errorf("peer audit receipt missing %s", k)
		}
	}
	return nil
}
func peerAsObject(v any) map[string]any                    { m, _ := v.(map[string]any); return m }
func peerObject(m map[string]any, k string) map[string]any { return peerAsObject(m[k]) }
func peerArray(m map[string]any, k string) []any           { v, _ := m[k].([]any); return v }
func peerString(m map[string]any, k string) string         { v, _ := m[k].(string); return v }
func peerBool(m map[string]any, k string) bool             { v, _ := m[k].(bool); return v }
func peerInt(m map[string]any, k string) int {
	n, _ := m[k].(json.Number)
	v, _ := strconv.Atoi(n.String())
	return v
}
func peerStrings(m map[string]any, k string) []string {
	raw := peerArray(m, k)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
func peerContains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
func peerStrategy(s string) bool { return s == "lean" || s == "balanced" || s == "guarded" }
func validatePeerSHA(name, s string) error {
	if len(s) != 71 || !strings.HasPrefix(s, "sha256:") {
		return fmt.Errorf("%s must be sha256-prefixed", name)
	}
	_, err := hex.DecodeString(s[7:])
	return err
}

func peerAuditExternalDigest(root map[string]any) (string, error) {
	copyRoot := make(map[string]any, len(root))
	for k, v := range root {
		if k != "receiptDigest" {
			copyRoot[k] = v
		}
	}
	canonical, err := encodePeerCanonical(copyRoot)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
func encodePeerCanonical(v any) ([]byte, error) {
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
			return nil, errors.New("canonical receipt accepts integer numbers only")
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
			z, err := encodePeerCanonical(e)
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
			z, err := encodePeerCanonical(x[k])
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
func peerWitnessDigest(w PeerAuditRingWitness) (string, error) {
	w.WitnessSHA256 = ""
	b, err := json.Marshal(w)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
