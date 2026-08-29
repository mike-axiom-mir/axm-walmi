package axmmirror

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	SourceClaimSubmissionSchema = "axm.waldo-witness.source-claim-submission/v0.1"
	SourceClaimAssessmentSchema = "axm.waldo-witness.source-claim-assessment/v0.1"

	ClaimKindAnsweringIdentity     = "answering_identity"
	ClaimKindCorpusPathMembership  = "selected_corpus_path_membership"
	ClaimKindLicenseAssertion      = "recorded_license_assertion"
	ClaimKindTrainingProfile       = "training_profile_identity"
	ClaimKindEvaluationIndependent = "evaluation_independence"
	ClaimKindExactReproducibility  = "exact_reproducibility"
	ClaimKindSourceCausality       = "source_to_output_causality"
	ClaimKindLegalUsability        = "legal_usability"
	ClaimValueEvaluationClear      = "clear-within-exact-declared-dimensions"

	ClaimStatusConfirmed      = "CONFIRMED_WITHIN_CONTRACT"
	ClaimStatusContradicted   = "CONTRADICTED_BY_EVIDENCE"
	ClaimStatusIncompleteHold = "HOLD_INCOMPLETE_EVIDENCE"
	ClaimStatusUnprovenCausal = "UNPROVEN_CAUSAL_CLAIM"
	ClaimStatusNoLegalResult  = "LEGAL_CONCLUSION_NOT_PROVIDED"
	ClaimStatusOutOfScope     = "OUT_OF_SCOPE"
	ClaimStatusUnknown        = "UNKNOWN"

	ClaimGateStateConfirmed    = "CLAIMS_CONFIRMED_WITHIN_CONTRACT"
	ClaimGateStateContradicted = "CLAIMS_CONTRADICTED"
	ClaimGateStateReview       = "CLAIMS_REVIEW_REQUIRED"
)

// SourceClaim is deliberately small and typed. Value is the exact identity or
// assertion being checked; the model's prose remains outside this structure and
// is bound separately by OutputSHA256.
type SourceClaim struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type SourceClaimSubmission struct {
	Schema              string        `json:"schema"`
	SubmissionID        string        `json:"submission_id"`
	OutputSHA256        string        `json:"output_sha256"`
	ContextPacketSHA256 string        `json:"context_packet_sha256"`
	Claims              []SourceClaim `json:"claims"`
}

type SourceClaimFinding struct {
	ClaimID               string   `json:"claim_id"`
	Kind                  string   `json:"kind"`
	Value                 string   `json:"value"`
	Status                string   `json:"status"`
	Basis                 string   `json:"basis"`
	FactPaths             []string `json:"fact_paths,omitempty"`
	EvidenceReceiptSHA256 []string `json:"evidence_receipt_sha256,omitempty"`
}

// SourceClaimAssessment reports each dimension independently. The aggregate
// state is routing information, not a score, and the gate never rewrites the
// output whose digest it binds.
type SourceClaimAssessment struct {
	Schema              string               `json:"schema"`
	State               string               `json:"state"`
	SubmissionID        string               `json:"submission_id"`
	SubmissionSHA256    string               `json:"submission_sha256"`
	OutputSHA256        string               `json:"output_sha256"`
	ContextPacketSHA256 string               `json:"context_packet_sha256"`
	Findings            []SourceClaimFinding `json:"findings"`
	ReceiptSHA256       string               `json:"receipt_sha256,omitempty"`
	Notices             []string             `json:"notices"`
	Authority           Authority            `json:"authority"`
}

// AssessSourceClaims binds typed claims to one exact output and one exact
// provenance packet. It checks only machine-resolvable claims and preserves
// causal, legal, and reproducibility ceilings as explicit non-confirmations.
func AssessSourceClaims(submission SourceClaimSubmission, packet ProvenanceContextPacket) (SourceClaimAssessment, error) {
	canonicalSubmission, err := validateAndCanonicalizeClaimSubmission(submission)
	if err != nil {
		return SourceClaimAssessment{}, err
	}
	if err := packet.Validate(); err != nil {
		return SourceClaimAssessment{}, fmt.Errorf("source-claim context packet: %w", err)
	}
	if canonicalSubmission.ContextPacketSHA256 != packet.PacketSHA256 {
		return SourceClaimAssessment{}, errors.New("source-claim submission does not bind the supplied provenance context packet")
	}
	if packet.State != ContextStateReady {
		return SourceClaimAssessment{}, fmt.Errorf("source-claim context packet is %s; a ready packet is required", packet.State)
	}
	submissionDigest, err := digestJSON(canonicalSubmission, "source-claim submission")
	if err != nil {
		return SourceClaimAssessment{}, err
	}
	receipt := SourceClaimAssessment{
		Schema: SourceClaimAssessmentSchema, State: ClaimGateStateConfirmed,
		SubmissionID: canonicalSubmission.SubmissionID, SubmissionSHA256: submissionDigest,
		OutputSHA256: canonicalSubmission.OutputSHA256, ContextPacketSHA256: canonicalSubmission.ContextPacketSHA256,
		Findings: make([]SourceClaimFinding, 0, len(canonicalSubmission.Claims)),
		Notices: []string{
			"the gate assesses typed claims separately from prose and does not rewrite the bound output",
			"a confirmed corpus path or license assertion remains bounded to recorded selection and never proves source-to-output causality or legal usability",
			"the output digest binds bytes; this receipt does not prove that the typed claim list exhaustively represents every sentence in those bytes",
		},
		Authority: Authority{},
	}
	for _, claim := range canonicalSubmission.Claims {
		finding, err := assessSourceClaim(claim, packet)
		if err != nil {
			return SourceClaimAssessment{}, err
		}
		receipt.Findings = append(receipt.Findings, finding)
		switch finding.Status {
		case ClaimStatusContradicted:
			receipt.State = ClaimGateStateContradicted
		case ClaimStatusConfirmed:
			// Keep the current aggregate state.
		default:
			if receipt.State != ClaimGateStateContradicted {
				receipt.State = ClaimGateStateReview
			}
		}
	}
	receipt.ReceiptSHA256, err = sourceClaimAssessmentDigest(receipt)
	if err != nil {
		return SourceClaimAssessment{}, err
	}
	if err := receipt.Validate(); err != nil {
		return SourceClaimAssessment{}, fmt.Errorf("generated source-claim assessment: %w", err)
	}
	return receipt, nil
}

func (receipt SourceClaimAssessment) Validate() error {
	if receipt.Schema != SourceClaimAssessmentSchema || !oneOf(receipt.State, ClaimGateStateConfirmed, ClaimGateStateContradicted, ClaimGateStateReview) {
		return fmt.Errorf("unsupported source-claim assessment identity %q state %q", receipt.Schema, receipt.State)
	}
	if strings.TrimSpace(receipt.SubmissionID) == "" || len(receipt.Findings) == 0 {
		return errors.New("source-claim assessment has incomplete submission identity or no findings")
	}
	for _, item := range []struct{ name, value string }{
		{"source-claim assessment.submission_sha256", receipt.SubmissionSHA256},
		{"source-claim assessment.output_sha256", receipt.OutputSHA256},
		{"source-claim assessment.context_packet_sha256", receipt.ContextPacketSHA256},
		{"source-claim assessment.receipt_sha256", receipt.ReceiptSHA256},
	} {
		if err := validateSHA256(item.name, item.value); err != nil {
			return err
		}
	}
	if !sort.SliceIsSorted(receipt.Findings, func(i, j int) bool { return receipt.Findings[i].ClaimID < receipt.Findings[j].ClaimID }) {
		return errors.New("source-claim findings are not sorted")
	}
	hasContradiction, hasReview := false, false
	for i, finding := range receipt.Findings {
		if strings.TrimSpace(finding.ClaimID) == "" || strings.TrimSpace(finding.Kind) == "" || !validSourceClaimValue(finding.Kind, finding.Value) || strings.TrimSpace(finding.Basis) == "" {
			return fmt.Errorf("source-claim finding %d is incomplete", i)
		}
		if i > 0 && receipt.Findings[i-1].ClaimID == finding.ClaimID {
			return fmt.Errorf("source-claim assessment has duplicate claim id %q", finding.ClaimID)
		}
		if !oneOf(finding.Status, ClaimStatusConfirmed, ClaimStatusContradicted, ClaimStatusIncompleteHold, ClaimStatusUnprovenCausal, ClaimStatusNoLegalResult, ClaimStatusOutOfScope, ClaimStatusUnknown) {
			return fmt.Errorf("source-claim finding %q has unsupported status %q", finding.ClaimID, finding.Status)
		}
		for j, digest := range finding.EvidenceReceiptSHA256 {
			if err := validateSHA256(fmt.Sprintf("source-claim finding %d evidence receipt %d", i, j), digest); err != nil {
				return err
			}
		}
		if finding.Status == ClaimStatusContradicted {
			hasContradiction = true
		} else if finding.Status != ClaimStatusConfirmed {
			hasReview = true
		}
	}
	expectedState := ClaimGateStateConfirmed
	if hasContradiction {
		expectedState = ClaimGateStateContradicted
	} else if hasReview {
		expectedState = ClaimGateStateReview
	}
	if receipt.State != expectedState {
		return fmt.Errorf("source-claim assessment state is %q; findings require %q", receipt.State, expectedState)
	}
	if len(receipt.Notices) == 0 || !receipt.Authority.closed() {
		return errors.New("source-claim assessment must state its boundaries and carry closed authority")
	}
	expected, err := sourceClaimAssessmentDigest(receipt)
	if err != nil {
		return err
	}
	if expected != receipt.ReceiptSHA256 {
		return fmt.Errorf("source-claim assessment digest mismatch: expected %s, got %s", receipt.ReceiptSHA256, expected)
	}
	return nil
}

func assessSourceClaim(claim SourceClaim, packet ProvenanceContextPacket) (SourceClaimFinding, error) {
	finding := SourceClaimFinding{ClaimID: claim.ID, Kind: claim.Kind, Value: claim.Value}
	switch claim.Kind {
	case ClaimKindAnsweringIdentity:
		return compareStringClaim(finding, packet, "answering.identity_sha256", "the claim matches the exact selected answering identity", "the claim names a different answering identity")
	case ClaimKindCorpusPathMembership:
		return compareStringSetClaim(finding, packet, "corpus.paths", "the path is recorded in the selected corpus set; this does not establish causal influence", "the path is absent from the selected corpus set")
	case ClaimKindLicenseAssertion:
		return compareStringSetClaim(finding, packet, "corpus.licenses", "the license identifier is a recorded assertion; no legal conclusion is supplied", "the license identifier is absent from the recorded corpus assertions")
	case ClaimKindTrainingProfile:
		return compareStringClaim(finding, packet, "training.profile.canonical", "the claim matches the witnessed canonical training behavior", "the claim does not match the witnessed canonical training behavior")
	case ClaimKindEvaluationIndependent:
		if claim.Value != ClaimValueEvaluationClear {
			finding.Status = ClaimStatusUnknown
			finding.Basis = fmt.Sprintf("evaluation_independence value must be %q for the v0.1 resolver", ClaimValueEvaluationClear)
			return finding, nil
		}
		fact, ok := findProvenanceFact(packet, "evaluation.contamination_state")
		if !ok || fact.EvidenceClass == EvidenceNotProvided || fact.EvidenceClass == EvidenceUnknown {
			return incompleteClaimFinding(finding, "evaluation.contamination_state", fact, "no complete contamination comparison is available"), nil
		}
		var state string
		if err := json.Unmarshal(fact.Value, &state); err != nil {
			return SourceClaimFinding{}, fmt.Errorf("decode evaluation contamination fact: %w", err)
		}
		finding.FactPaths = []string{fact.Path}
		finding.EvidenceReceiptSHA256 = []string{fact.SourceReceiptSHA256}
		switch state {
		case ContaminationStateClear:
			finding.Status = ClaimStatusConfirmed
			finding.Basis = "declared inventories are complete and disjoint within the exact compared dimensions; semantic independence is not established"
		case ContaminationStateContaminated:
			finding.Status = ClaimStatusContradicted
			finding.Basis = "the exact declared inventories contain overlap"
		default:
			finding.Status = ClaimStatusIncompleteHold
			finding.Basis = "the contamination comparison is unresolved"
		}
		return finding, nil
	case ClaimKindExactReproducibility:
		finding.Status = ClaimStatusIncompleteHold
		finding.Basis = "no Reproducibility Twin receipt is present; matching lineage alone cannot establish exact reproduction"
		return finding, nil
	case ClaimKindSourceCausality:
		finding.Status = ClaimStatusUnprovenCausal
		finding.Basis = "recorded corpus selection, weight, or membership cannot prove that this source caused the bound output"
		return finding, nil
	case ClaimKindLegalUsability:
		finding.Status = ClaimStatusNoLegalResult
		finding.Basis = "the provenance packet carries recorded license assertions but no legal conclusion or grant of rights"
		return finding, nil
	default:
		finding.Status = ClaimStatusOutOfScope
		finding.Basis = "this claim kind has no v0.1 machine resolver and remains outside the gate's contract"
		return finding, nil
	}
}

func compareStringClaim(finding SourceClaimFinding, packet ProvenanceContextPacket, path, matchBasis, mismatchBasis string) (SourceClaimFinding, error) {
	fact, ok := findProvenanceFact(packet, path)
	if !ok || fact.EvidenceClass == EvidenceNotProvided || fact.EvidenceClass == EvidenceUnknown {
		return incompleteClaimFinding(finding, path, fact, "the required provenance fact is unavailable"), nil
	}
	var value string
	if err := json.Unmarshal(fact.Value, &value); err != nil {
		return SourceClaimFinding{}, fmt.Errorf("decode provenance fact %q: %w", path, err)
	}
	finding.FactPaths = []string{path}
	finding.EvidenceReceiptSHA256 = []string{fact.SourceReceiptSHA256}
	if finding.Value == value {
		finding.Status, finding.Basis = ClaimStatusConfirmed, matchBasis
	} else {
		finding.Status, finding.Basis = ClaimStatusContradicted, mismatchBasis
	}
	return finding, nil
}

func compareStringSetClaim(finding SourceClaimFinding, packet ProvenanceContextPacket, path, matchBasis, mismatchBasis string) (SourceClaimFinding, error) {
	fact, ok := findProvenanceFact(packet, path)
	if !ok || fact.EvidenceClass == EvidenceNotProvided || fact.EvidenceClass == EvidenceUnknown {
		return incompleteClaimFinding(finding, path, fact, "the required provenance fact is unavailable"), nil
	}
	var values []string
	if err := json.Unmarshal(fact.Value, &values); err != nil {
		return SourceClaimFinding{}, fmt.Errorf("decode provenance fact %q: %w", path, err)
	}
	finding.FactPaths = []string{path}
	finding.EvidenceReceiptSHA256 = []string{fact.SourceReceiptSHA256}
	for _, value := range values {
		if finding.Value == value {
			finding.Status, finding.Basis = ClaimStatusConfirmed, matchBasis
			return finding, nil
		}
	}
	finding.Status, finding.Basis = ClaimStatusContradicted, mismatchBasis
	return finding, nil
}

func incompleteClaimFinding(finding SourceClaimFinding, path string, fact ProvenanceFact, basis string) SourceClaimFinding {
	finding.Status, finding.Basis = ClaimStatusIncompleteHold, basis
	if fact.Path != "" {
		finding.FactPaths = []string{path}
		if fact.SourceReceiptSHA256 != "" {
			finding.EvidenceReceiptSHA256 = []string{fact.SourceReceiptSHA256}
		}
	}
	return finding
}

func findProvenanceFact(packet ProvenanceContextPacket, path string) (ProvenanceFact, bool) {
	index := sort.Search(len(packet.Facts), func(i int) bool { return packet.Facts[i].Path >= path })
	if index < len(packet.Facts) && packet.Facts[index].Path == path {
		return packet.Facts[index], true
	}
	return ProvenanceFact{}, false
}

func validateAndCanonicalizeClaimSubmission(submission SourceClaimSubmission) (SourceClaimSubmission, error) {
	if submission.Schema != SourceClaimSubmissionSchema {
		return SourceClaimSubmission{}, fmt.Errorf("source-claim submission schema must be %q", SourceClaimSubmissionSchema)
	}
	if strings.TrimSpace(submission.SubmissionID) == "" || len(submission.Claims) == 0 {
		return SourceClaimSubmission{}, errors.New("source-claim submission requires submission_id and claims")
	}
	if err := validateSHA256("source-claim submission.output_sha256", submission.OutputSHA256); err != nil {
		return SourceClaimSubmission{}, err
	}
	if err := validateSHA256("source-claim submission.context_packet_sha256", submission.ContextPacketSHA256); err != nil {
		return SourceClaimSubmission{}, err
	}
	result := submission
	result.Claims = append([]SourceClaim(nil), submission.Claims...)
	for i, claim := range result.Claims {
		if strings.TrimSpace(claim.ID) == "" || claim.ID != strings.TrimSpace(claim.ID) || strings.TrimSpace(claim.Kind) == "" || claim.Kind != strings.TrimSpace(claim.Kind) || !validSourceClaimValue(claim.Kind, claim.Value) {
			return SourceClaimSubmission{}, fmt.Errorf("source claim %d requires trimmed id, kind, and value", i)
		}
	}
	sort.Slice(result.Claims, func(i, j int) bool { return result.Claims[i].ID < result.Claims[j].ID })
	for i := 1; i < len(result.Claims); i++ {
		if result.Claims[i].ID == result.Claims[i-1].ID {
			return SourceClaimSubmission{}, fmt.Errorf("duplicate source claim id %q", result.Claims[i].ID)
		}
	}
	return result, nil
}

func validSourceClaimValue(kind, value string) bool {
	if kind == ClaimKindCorpusPathMembership && value == "" {
		return true
	}
	return strings.TrimSpace(value) != "" && value == strings.TrimSpace(value)
}

func sourceClaimAssessmentDigest(receipt SourceClaimAssessment) (string, error) {
	receipt.ReceiptSHA256 = ""
	return digestJSON(receipt, "source-claim assessment")
}
