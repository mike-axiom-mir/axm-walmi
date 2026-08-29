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
	OrganRuntimeReceiptSchema = "axm.waldo-mirror.organ-runtime-bridge-receipt/v0.1"
	OrganRuntimeReceiptStatus = "EXPERIMENTAL_OBSERVED"
	OrganRuntimeWitnessSchema = "axm.waldo-witness.organ-runtime-bridge/v0.1"
	OrganRuntimeWitnessReady  = "ORGAN_RUNTIME_WITNESSED"
	OrganRuntimeChallenge     = "ONE_CONCEPT_MANY_BODIES"
	OrganRuntimeConceptSHA256 = "sha256:9779b14236b2717ce1e8694adbacfd95c07d3afbacea10d3e4255cb52950856a"
	OrganRuntimePacketSHA256  = "sha256:8f4688db3abcba988011f2a2f36f172a09c3d848e57a2da54db90abd2669e634"
)

var organRuntimeStrategies = []string{"lean", "balanced", "guarded"}

type OrganRuntimeSourcePacket struct {
	SHA256           string `json:"sha256"`
	CompositionState string `json:"compositionState"`
}

type OrganRuntimeConceptRef struct {
	ID     string `json:"id"`
	Schema string `json:"schema"`
	SHA256 string `json:"sha256"`
}

type OrganRuntimeInfo struct {
	FactoryVersion string   `json:"factoryVersion"`
	Version        string   `json:"version"`
	Digest         string   `json:"digest"`
	Strategies     []string `json:"strategies"`
}

type OrganRuntimeFieldPack struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
}

type OrganRuntimeCandidate struct {
	Strategy         string `json:"strategy"`
	PackageDigest    string `json:"packageDigest"`
	DefinitionDigest string `json:"definitionDigest"`
	EvaluationDigest string `json:"evaluationDigest"`
	Score            int    `json:"score"`
	PackageVerified  bool   `json:"packageVerified"`
}

type OrganRuntimeBody struct {
	BodyID              string                  `json:"bodyId"`
	SourceKind          string                  `json:"sourceKind"`
	FieldPack           OrganRuntimeFieldPack   `json:"fieldPack"`
	ImportedStatus      string                  `json:"importedStatus"`
	RunStatus           string                  `json:"runStatus"`
	RunDigest           string                  `json:"runDigest"`
	CandidateCount      int                     `json:"candidateCount"`
	FailureCount        int                     `json:"failureCount"`
	Candidates          []OrganRuntimeCandidate `json:"candidates"`
	ComparisonDigest    string                  `json:"comparisonDigest"`
	RankingAdvisoryOnly bool                    `json:"rankingAdvisoryOnly"`
	SelectionPerformed  bool                    `json:"selectionPerformed"`
}

type OrganRuntimeTotals struct {
	ProposalCount  int `json:"proposalCount"`
	CandidateCount int `json:"candidateCount"`
	FailureCount   int `json:"failureCount"`
}

type OrganArchiveVerification struct {
	Schema       string   `json:"schema"`
	OK           bool     `json:"ok"`
	Errors       []string `json:"errors"`
	ObjectCount  int      `json:"objectCount"`
	FailureCount int      `json:"failureCount"`
	EventCount   int      `json:"eventCount"`
	IndexDigest  string   `json:"indexDigest"`
}

type OrganArchiveImport struct {
	Schema      string `json:"schema"`
	PackDigest  string `json:"packDigest"`
	ResultCount int    `json:"resultCount"`
	Registered  bool   `json:"registered"`
	Installed   bool   `json:"installed"`
}

type OrganRuntimeArchive struct {
	VerificationBeforeDedup      OrganArchiveVerification `json:"verificationBeforeDedup"`
	DuplicateReinsertDisposition string                   `json:"duplicateReinsertDisposition"`
	VerificationAfterDedup       OrganArchiveVerification `json:"verificationAfterDedup"`
	ExportedPackDigest           string                   `json:"exportedPackDigest"`
	ExportedObjectCount          int                      `json:"exportedObjectCount"`
	ImportResult                 OrganArchiveImport       `json:"importResult"`
	ImportedVerification         OrganArchiveVerification `json:"importedVerification"`
}

type OrganRuntimeTruth struct {
	DonorPacketChecksumVerified          bool `json:"donorPacketChecksumVerified"`
	DonorSelftestsExecutedInThisRun      bool `json:"donorSelftestsExecutedInThisRun"`
	DonorSelftestChecks                  int  `json:"donorSelftestChecks"`
	PacketRuntimeExecuted                bool `json:"packetRuntimeExecuted"`
	LiveAIProviderCalled                 bool `json:"liveAiProviderCalled"`
	AIProposalSourcePresent              bool `json:"aiProposalSourcePresent"`
	GeneratedCandidatesObserved          int  `json:"generatedCandidatesObserved"`
	ArchiveWritesDisposableOnly          bool `json:"archiveWritesDisposableOnly"`
	DuplicateArchiveWriteAddedEvent      bool `json:"duplicateArchiveWriteAddedEvent"`
	ArchiveExportImportRoundTripVerified bool `json:"archiveExportImportRoundTripVerified"`
	SelectionPerformed                   bool `json:"selectionPerformed"`
	Installed                            bool `json:"installed"`
	Registered                           bool `json:"registered"`
	Staged                               bool `json:"staged"`
	Promoted                             bool `json:"promoted"`
	CanonChanged                         bool `json:"canonChanged"`
}

type OrganRuntimeReceipt struct {
	Schema        string                   `json:"schema"`
	Status        string                   `json:"status"`
	Challenge     string                   `json:"challenge"`
	SourcePacket  OrganRuntimeSourcePacket `json:"sourcePacket"`
	ConceptRef    OrganRuntimeConceptRef   `json:"conceptRef"`
	Runtime       OrganRuntimeInfo         `json:"runtime"`
	Bodies        []OrganRuntimeBody       `json:"bodies"`
	Totals        OrganRuntimeTotals       `json:"totals"`
	Archive       OrganRuntimeArchive      `json:"archive"`
	Truth         OrganRuntimeTruth        `json:"truth"`
	Authority     string                   `json:"authority"`
	ReceiptDigest string                   `json:"receiptDigest"`
}

type OrganRuntimeBodyWitness struct {
	BodyID                  string   `json:"body_id"`
	SourceKind              string   `json:"source_kind"`
	FieldPackID             string   `json:"field_pack_id"`
	RunDigest               string   `json:"run_digest"`
	CandidatePackageDigests []string `json:"candidate_package_digests"`
}

type OrganRuntimeWitness struct {
	Schema              string                    `json:"schema"`
	State               string                    `json:"state"`
	SourceReceiptSHA256 string                    `json:"source_receipt_sha256"`
	SourcePacketSHA256  string                    `json:"source_packet_sha256"`
	ConceptSHA256       string                    `json:"concept_sha256"`
	RuntimeDigest       string                    `json:"runtime_digest"`
	ProposalCount       int                       `json:"proposal_count"`
	CandidateCount      int                       `json:"candidate_count"`
	FailureCount        int                       `json:"failure_count"`
	AIProposalObserved  bool                      `json:"ai_proposal_observed"`
	Bodies              []OrganRuntimeBodyWitness `json:"bodies"`
	ArchiveIndexSHA256  string                    `json:"archive_index_sha256"`
	ArchivePackSHA256   string                    `json:"archive_pack_sha256"`
	Notices             []string                  `json:"notices"`
	Authority           Authority                 `json:"authority"`
	WitnessSHA256       string                    `json:"witness_sha256,omitempty"`
}

func VerifyOrganRuntimeReceipt(data []byte) (OrganRuntimeReceipt, error) {
	var receipt OrganRuntimeReceipt
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&receipt); err != nil {
		return receipt, fmt.Errorf("decode Organ runtime receipt: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("Organ runtime receipt has trailing JSON")
		}
		return OrganRuntimeReceipt{}, err
	}
	if err := receipt.Validate(); err != nil {
		return OrganRuntimeReceipt{}, err
	}
	expected, err := organRuntimeExternalReceiptDigest(data)
	if err != nil {
		return OrganRuntimeReceipt{}, err
	}
	if expected != receipt.ReceiptDigest {
		return OrganRuntimeReceipt{}, fmt.Errorf("Organ runtime receipt digest mismatch: expected %s, got %s", expected, receipt.ReceiptDigest)
	}
	return receipt, nil
}

func (r OrganRuntimeReceipt) Validate() error {
	if r.Schema != OrganRuntimeReceiptSchema || r.Status != OrganRuntimeReceiptStatus || r.Challenge != OrganRuntimeChallenge {
		return errors.New("Organ runtime receipt has unsupported schema, status, or challenge")
	}
	if r.SourcePacket.SHA256 != OrganRuntimePacketSHA256 || r.SourcePacket.CompositionState != "SEPARATE_UNMERGED_SOURCE_SETS" {
		return errors.New("Organ runtime donor packet binding mismatch")
	}
	if r.ConceptRef.ID != "one-concept-many-bodies" || r.ConceptRef.Schema != "axm.waldo-mirror.fabric-concept/v0.1" || r.ConceptRef.SHA256 != OrganRuntimeConceptSHA256 {
		return errors.New("Organ runtime concept binding mismatch")
	}
	if r.Runtime.Version != "1.0.0" || r.Runtime.FactoryVersion != "1.0.0" || !organRuntimeStrategiesMatch(r.Runtime.Strategies) {
		return errors.New("Organ runtime version or strategy set mismatch")
	}
	for name, d := range map[string]string{"runtime": r.Runtime.Digest, "receipt": r.ReceiptDigest} {
		if err := validatePrefixedSHA256("Organ runtime "+name, d); err != nil {
			return err
		}
	}
	if len(r.Bodies) != 4 || r.Totals.ProposalCount != 4 || r.Totals.CandidateCount != 12 || r.Totals.FailureCount != 0 {
		return errors.New("Organ runtime expected 4 proposals, 12 candidates, and zero failures")
	}
	seenBodies := map[string]bool{}
	seenPackages := map[string]bool{}
	aiBodies := 0
	total := 0
	for i, b := range r.Bodies {
		if strings.TrimSpace(b.BodyID) == "" || seenBodies[b.BodyID] {
			return fmt.Errorf("Organ runtime body %d has empty or duplicate id", i)
		}
		seenBodies[b.BodyID] = true
		if b.SourceKind != "EXTERNAL" && b.SourceKind != "AI" {
			return fmt.Errorf("Organ runtime body %q has unsupported source kind", b.BodyID)
		}
		if b.SourceKind == "AI" {
			aiBodies++
		}
		if b.ImportedStatus != "NORMALIZED_UNTRUSTED_PROPOSAL" || b.RunStatus != "COMPLETE" || b.CandidateCount != 3 || b.FailureCount != 0 || len(b.Candidates) != 3 || !b.RankingAdvisoryOnly || b.SelectionPerformed {
			return fmt.Errorf("Organ runtime body %q boundary or run status drifted", b.BodyID)
		}
		if strings.TrimSpace(b.FieldPack.ID) == "" || b.FieldPack.Version != "1.0.0" {
			return fmt.Errorf("Organ runtime body %q field pack binding invalid", b.BodyID)
		}
		for name, d := range map[string]string{"field pack": b.FieldPack.Digest, "run": b.RunDigest, "comparison": b.ComparisonDigest} {
			if err := validatePrefixedSHA256("Organ runtime body "+b.BodyID+" "+name, d); err != nil {
				return err
			}
		}
		strategies := map[string]bool{}
		for _, c := range b.Candidates {
			if !organRuntimeHasStrategy(c.Strategy) || strategies[c.Strategy] {
				return fmt.Errorf("Organ runtime body %q candidate strategy set invalid", b.BodyID)
			}
			strategies[c.Strategy] = true
			if !c.PackageVerified {
				return fmt.Errorf("Organ runtime body %q has unverified candidate", b.BodyID)
			}
			if c.Score < 0 || c.Score > 100 {
				return fmt.Errorf("Organ runtime body %q candidate score out of bounds", b.BodyID)
			}
			for name, d := range map[string]string{"package": c.PackageDigest, "definition": c.DefinitionDigest, "evaluation": c.EvaluationDigest} {
				if err := validatePrefixedSHA256("Organ runtime candidate "+name, d); err != nil {
					return err
				}
			}
			if seenPackages[c.PackageDigest] {
				return errors.New("Organ runtime candidate package digest duplicated across bodies")
			}
			seenPackages[c.PackageDigest] = true
			total++
		}
	}
	if aiBodies != 1 || total != r.Totals.CandidateCount {
		return errors.New("Organ runtime AI body or candidate total mismatch")
	}
	before, after, imported := r.Archive.VerificationBeforeDedup, r.Archive.VerificationAfterDedup, r.Archive.ImportedVerification
	for label, v := range map[string]OrganArchiveVerification{"before": before, "after": after, "imported": imported} {
		if v.Schema != "axm.organ-archive-verification/v1" || !v.OK || len(v.Errors) != 0 || v.ObjectCount != 12 || v.FailureCount != 0 || v.EventCount != 12 {
			return fmt.Errorf("Organ archive %s verification boundary drifted", label)
		}
		if err := validatePrefixedSHA256("Organ archive "+label+" index digest", v.IndexDigest); err != nil {
			return err
		}
	}
	if before.IndexDigest != after.IndexDigest || before.IndexDigest != imported.IndexDigest || r.Archive.DuplicateReinsertDisposition != "DEDUPLICATED" {
		return errors.New("Organ archive dedup or round-trip identity mismatch")
	}
	if r.Archive.ExportedObjectCount != 12 || r.Archive.ImportResult.Schema != "axm.organ-archive-import/v1" || r.Archive.ImportResult.ResultCount != 12 || r.Archive.ImportResult.Registered || r.Archive.ImportResult.Installed || r.Archive.ImportResult.PackDigest != r.Archive.ExportedPackDigest {
		return errors.New("Organ archive export/import boundary drifted")
	}
	if err := validatePrefixedSHA256("Organ archive exported pack digest", r.Archive.ExportedPackDigest); err != nil {
		return err
	}
	t := r.Truth
	if !t.DonorPacketChecksumVerified || !t.DonorSelftestsExecutedInThisRun || t.DonorSelftestChecks != 151 || !t.PacketRuntimeExecuted || t.LiveAIProviderCalled || !t.AIProposalSourcePresent || t.GeneratedCandidatesObserved != 12 || !t.ArchiveWritesDisposableOnly || t.DuplicateArchiveWriteAddedEvent || !t.ArchiveExportImportRoundTripVerified || t.SelectionPerformed || t.Installed || t.Registered || t.Staged || t.Promoted || t.CanonChanged {
		return errors.New("Organ runtime truth boundary drifted")
	}
	if r.Authority != "NONE" {
		return errors.New("Organ runtime receipt authority must remain NONE")
	}
	return nil
}

func WitnessOrganRuntimeReceipt(data []byte) (OrganRuntimeWitness, error) {
	r, err := VerifyOrganRuntimeReceipt(data)
	if err != nil {
		return OrganRuntimeWitness{}, err
	}
	bodies := make([]OrganRuntimeBodyWitness, 0, len(r.Bodies))
	for _, b := range r.Bodies {
		pkgs := make([]string, 0, len(b.Candidates))
		for _, c := range b.Candidates {
			pkgs = append(pkgs, c.PackageDigest)
		}
		sort.Strings(pkgs)
		bodies = append(bodies, OrganRuntimeBodyWitness{BodyID: b.BodyID, SourceKind: b.SourceKind, FieldPackID: b.FieldPack.ID, RunDigest: b.RunDigest, CandidatePackageDigests: pkgs})
	}
	sort.Slice(bodies, func(i, j int) bool { return bodies[i].BodyID < bodies[j].BodyID })
	w := OrganRuntimeWitness{Schema: OrganRuntimeWitnessSchema, State: OrganRuntimeWitnessReady, SourceReceiptSHA256: r.ReceiptDigest, SourcePacketSHA256: r.SourcePacket.SHA256, ConceptSHA256: r.ConceptRef.SHA256, RuntimeDigest: r.Runtime.Digest, ProposalCount: r.Totals.ProposalCount, CandidateCount: r.Totals.CandidateCount, FailureCount: r.Totals.FailureCount, AIProposalObserved: true, Bodies: bodies, ArchiveIndexSHA256: r.Archive.VerificationAfterDedup.IndexDigest, ArchivePackSHA256: r.Archive.ExportedPackDigest, Notices: []string{"the donor packet was executed only after later explicit human authorization; the original packet evidence remains historically unchanged", "all twelve generated candidates were verified and deterministically replayed, but candidate scores remain advisory and no winner was selected", "archive writes occurred only in disposable test roots; installation, registration, staging, promotion, persistent learning, and CANON remained closed"}, Authority: Authority{}}
	w.WitnessSHA256, err = organRuntimeWitnessDigest(w)
	if err != nil {
		return OrganRuntimeWitness{}, err
	}
	if err := w.Validate(); err != nil {
		return OrganRuntimeWitness{}, err
	}
	return w, nil
}

func (w OrganRuntimeWitness) Validate() error {
	if w.Schema != OrganRuntimeWitnessSchema || w.State != OrganRuntimeWitnessReady {
		return errors.New("Organ runtime witness schema/state mismatch")
	}
	for name, d := range map[string]string{"source receipt": w.SourceReceiptSHA256, "source packet": w.SourcePacketSHA256, "concept": w.ConceptSHA256, "runtime": w.RuntimeDigest, "archive index": w.ArchiveIndexSHA256, "archive pack": w.ArchivePackSHA256} {
		if err := validatePrefixedSHA256("Organ runtime witness "+name, d); err != nil {
			return err
		}
	}
	if w.SourcePacketSHA256 != OrganRuntimePacketSHA256 || w.ConceptSHA256 != OrganRuntimeConceptSHA256 || w.ProposalCount != 4 || w.CandidateCount != 12 || w.FailureCount != 0 || !w.AIProposalObserved || len(w.Bodies) != 4 {
		return errors.New("Organ runtime witness observed counts or bindings drifted")
	}
	prevBody := ""
	seen := map[string]bool{}
	for _, b := range w.Bodies {
		if b.BodyID == "" || b.BodyID <= prevBody {
			return errors.New("Organ runtime witness bodies must be non-empty and strictly sorted")
		}
		prevBody = b.BodyID
		if b.SourceKind != "EXTERNAL" && b.SourceKind != "AI" {
			return errors.New("Organ runtime witness source kind unsupported")
		}
		if b.FieldPackID == "" {
			return errors.New("Organ runtime witness field pack id missing")
		}
		if err := validatePrefixedSHA256("Organ runtime witness run digest", b.RunDigest); err != nil {
			return err
		}
		if len(b.CandidatePackageDigests) != 3 {
			return errors.New("Organ runtime witness body requires three candidate packages")
		}
		prev := ""
		for _, d := range b.CandidatePackageDigests {
			if err := validatePrefixedSHA256("Organ runtime witness candidate package", d); err != nil {
				return err
			}
			if d <= prev || seen[d] {
				return errors.New("Organ runtime witness candidate package digests must be unique and sorted")
			}
			prev = d
			seen[d] = true
		}
	}
	if len(seen) != 12 {
		return errors.New("Organ runtime witness requires twelve unique candidate packages")
	}
	if len(w.Notices) < 3 || !w.Authority.closed() {
		return errors.New("Organ runtime witness requires notices and closed authority")
	}
	if err := validateSHA256("Organ runtime witness_sha256", w.WitnessSHA256); err != nil {
		return err
	}
	expected, err := organRuntimeWitnessDigest(w)
	if err != nil {
		return err
	}
	if expected != w.WitnessSHA256 {
		return errors.New("Organ runtime witness digest mismatch")
	}
	return nil
}

func organRuntimeExternalReceiptDigest(data []byte) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return "", err
	}
	root, ok := v.(map[string]any)
	if !ok {
		return "", errors.New("Organ runtime receipt root must be object")
	}
	delete(root, "receiptDigest")
	canonical, err := encodeOrganCanonicalJSON(root)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func encodeOrganCanonicalJSON(v any) ([]byte, error) {
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
			return nil, errors.New("Organ runtime bridge receipt accepts integer JSON numbers only")
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
			z, err := encodeOrganCanonicalJSON(e)
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
			z, err := encodeOrganCanonicalJSON(x[k])
			if err != nil {
				return nil, err
			}
			b.Write(z)
		}
		b.WriteByte('}')
		return b.Bytes(), nil
	default:
		return nil, fmt.Errorf("unsupported Organ canonical JSON type %T", v)
	}
}

func organRuntimeWitnessDigest(w OrganRuntimeWitness) (string, error) {
	w.WitnessSHA256 = ""
	payload, err := json.Marshal(w)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func organRuntimeStrategiesMatch(values []string) bool {
	if len(values) != len(organRuntimeStrategies) {
		return false
	}
	for i := range values {
		if values[i] != organRuntimeStrategies[i] {
			return false
		}
	}
	return true
}

func organRuntimeHasStrategy(v string) bool { return v == "lean" || v == "balanced" || v == "guarded" }
