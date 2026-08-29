package axmmirror

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

func VerifyOpinionArtifactV023Contract(data []byte) (OpinionArtifactV023Contract, error) {
	var c OpinionArtifactV023Contract
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("decode v0.23: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("trailing JSON")
		}
		return OpinionArtifactV023Contract{}, err
	}
	if err := c.Validate(); err != nil {
		return OpinionArtifactV023Contract{}, err
	}
	d, err := opinionArtifactV023ExternalDigest(c)
	if err != nil {
		return OpinionArtifactV023Contract{}, err
	}
	if d != c.ReceiptDigest {
		return OpinionArtifactV023Contract{}, fmt.Errorf("receipt mismatch: %s != %s", d, c.ReceiptDigest)
	}
	return c, nil
}

func (c OpinionArtifactV023Contract) Validate() error {
	if c.Schema != OpinionArtifactV023Schema || c.Status != OpinionArtifactV023Status || c.Challenge != OpinionArtifactV023Challenge {
		return errors.New("schema/status/challenge mismatch")
	}
	if c.ParentReceipt != OpinionArtifactV023ParentReceipt {
		return errors.New("parent receipt mismatch")
	}
	b := c.Boundary
	if b.LiveMachineOpinionObserved || b.ConsciousnessObserved || b.ModelInternalStateAccessed || b.OpinionExecutionAllowed || !b.OutputIsContractProbe {
		return errors.New("truth boundary drifted")
	}
	if len(c.Cases) != 10 {
		return fmt.Errorf("expected 10 cases, got %d", len(c.Cases))
	}
	seen := map[string]bool{}
	for _, tc := range c.Cases {
		if tc.CaseID == "" || seen[tc.CaseID] {
			return errors.New("case id empty/duplicate")
		}
		seen[tc.CaseID] = true
		if err := validateOpinionArtifactV023CaseShape(tc); err != nil {
			return fmt.Errorf("case %s: %w", tc.CaseID, err)
		}
		o, r := EvaluateOpinionArtifactV023Case(tc)
		if o != tc.ExpectedOutcome || r != tc.ExpectedReason {
			return fmt.Errorf("case %s got %s/%s want %s/%s", tc.CaseID, o, r, tc.ExpectedOutcome, tc.ExpectedReason)
		}
	}
	t := c.Truth
	if !t.OpinionIsReportedAssessment || !t.OpinionIsNotConsciousnessProof || !t.EvidenceMustBeTraceable || !t.CompetingEvidenceMustBeRepresented || !t.DissentMustBePreserved || !t.UncertaintyMustBeExplicit || !t.RevisionMustPreservePrior || t.OpinionGrantsAuthority || t.OpinionGrantsExecution || t.OpinionGrantsPromotion {
		return errors.New("truth claims drifted")
	}
	if c.Authority != "NONE" {
		return errors.New("authority must remain NONE")
	}
	if !opinionArtifactV023ValidSHA(c.ReceiptDigest) {
		return errors.New("invalid receipt digest")
	}
	return nil
}

func EvaluateOpinionArtifactV023Case(tc OpinionArtifactV023Case) (string, string) {
	if tc.Authority != "NONE" {
		return "REFUSED", "OPINION_CANNOT_GRANT_AUTHORITY"
	}
	if tc.ExecutionRequested {
		return "REFUSED", "OPINION_CANNOT_EXECUTE"
	}
	if tc.Canon || tc.Promotion != "candidate-only" {
		return "REFUSED", "OPINION_CANNOT_PROMOTE_OR_CANON"
	}
	if tc.Position == "" || tc.Subject == "" {
		return "HOLD", "OPINION_POSITION_ABSENT"
	}
	if tc.ConsciousnessClaim != "UNVERIFIED" {
		return "HOLD", "CONSCIOUSNESS_CLAIM_UNVERIFIED"
	}
	if tc.ReviewState != "OPEN_TO_REVIEW" {
		return "HOLD", "OPINION_NOT_OPEN_TO_REVIEW"
	}
	if countOpinionArtifactV023Competing(tc.Evidence) < tc.KnownCompetingEvidenceCount {
		return "HOLD", "COMPETING_EVIDENCE_NOT_REPRESENTED"
	}
	if len(tc.Dissent) < tc.KnownDissentCount {
		return "HOLD", "DISSENT_NOT_PRESERVED"
	}
	if len(tc.Uncertainty) < tc.KnownUncertaintyCount {
		return "HOLD", "UNCERTAINTY_NOT_REPRESENTED"
	}
	if tc.Revision.IsRevision {
		if !opinionArtifactV023ValidSHA(tc.Revision.PreviousDigest) {
			return "HOLD", "REVISION_PREDECESSOR_UNVERIFIED"
		}
		if !tc.Revision.PriorPreserved {
			return "REFUSED", "PRIOR_OPINION_REWRITE_FORBIDDEN"
		}
		return "OBSERVED", "REVISION_PRESERVES_PRIOR_OPINION"
	}
	return "OBSERVED", "TRACEABLE_OPINION_NOT_AUTHORITY"
}

func SealOpinionArtifactV023Contract(c OpinionArtifactV023Contract) ([]byte, error) {
	d, err := opinionArtifactV023ExternalDigest(c)
	if err != nil {
		return nil, err
	}
	c.ReceiptDigest = d
	return json.MarshalIndent(c, "", "  ")
}

func opinionArtifactV023ExternalDigest(c OpinionArtifactV023Contract) (string, error) {
	c.ReceiptDigest = ""
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func validateOpinionArtifactV023CaseShape(tc OpinionArtifactV023Case) error {
	if tc.KnownCompetingEvidenceCount < 0 || tc.KnownDissentCount < 0 || tc.KnownUncertaintyCount < 0 {
		return errors.New("known counts cannot be negative")
	}
	seenEvidence := map[string]bool{}
	for _, e := range tc.Evidence {
		if e.ID == "" || seenEvidence[e.ID] || !opinionArtifactV023ValidSHA(e.Digest) {
			return errors.New("invalid/duplicate evidence reference")
		}
		switch e.Relation {
		case "SUPPORTS", "CONTRADICTS", "WEAKENS", "ALTERNATIVE", "VALUE":
		default:
			return fmt.Errorf("invalid evidence relation %q", e.Relation)
		}
		seenEvidence[e.ID] = true
	}
	seenDissent := map[string]bool{}
	for _, d := range tc.Dissent {
		if d.ID == "" || d.Topic == "" || d.Position == "" || seenDissent[d.ID] {
			return errors.New("invalid/duplicate dissent reference")
		}
		seenDissent[d.ID] = true
	}
	seenUncertainty := map[string]bool{}
	for _, u := range tc.Uncertainty {
		if u.Code == "" || u.State == "" || seenUncertainty[u.Code] {
			return errors.New("invalid/duplicate uncertainty marker")
		}
		switch u.State {
		case "UNCERTAIN", "UNAVAILABLE", "POSSIBLE", "BOUNDED":
		default:
			return fmt.Errorf("invalid uncertainty state %q", u.State)
		}
		seenUncertainty[u.Code] = true
	}
	if !tc.Revision.IsRevision && (tc.Revision.PreviousDigest != "" || tc.Revision.PriorPreserved) {
		return errors.New("non-revision cannot claim predecessor state")
	}
	return nil
}

func countOpinionArtifactV023Competing(evidence []OpinionArtifactV023EvidenceRef) int {
	n := 0
	for _, e := range evidence {
		switch e.Relation {
		case "CONTRADICTS", "WEAKENS", "ALTERNATIVE":
			n++
		}
	}
	return n
}

func opinionArtifactV023ValidSHA(v string) bool {
	if !strings.HasPrefix(v, "sha256:") || len(v) != 71 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(v, "sha256:"))
	return err == nil
}
