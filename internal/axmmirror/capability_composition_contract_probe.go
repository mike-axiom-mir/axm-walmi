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

const (
	CapabilityCompositionSchema        = "axm.capability-composition-contract/v0.16"
	CapabilityCompositionStatus        = "CONTRACT_PROBE_ONLY"
	CapabilityCompositionChallenge     = "MULTI_CAPABILITY_COMPOSITION_CONTRACT"
	CapabilityCompositionWitnessSchema = "axm.waldo-witness.capability-composition-contract/v0.16"
	CapabilityCompositionWitnessReady  = "CAPABILITY_COMPOSITION_CONTRACT_WITNESSED"

	CapabilityCompositionV015Receipt = "sha256:75e6466701520c7aeab56e76c719ab275519e8ed2fffa5fceaf3310cce4ec0bf"
	CapabilityCompositionGapID       = "v014-material-evidence-view-incomplete"
	CapabilityCompositionNeedID      = "material-evidence-view-completion"
	CapabilityCompositionAuthority   = "READ_ONLY_PUBLIC_ARTIFACT_ACCESS"
)

type CapabilityCompositionSource struct {
	V015ContractReceipt string `json:"v015ContractReceipt"`
	AncestorGapID       string `json:"ancestorGapId"`
	CapabilityNeedID    string `json:"capabilityNeedId"`
}

type CapabilityCompositionBoundary struct {
	LocalCapabilityFabricImplementationIncluded bool `json:"localCapabilityFabricImplementationIncluded"`
	ActualCapabilityFabricExecutionObserved     bool `json:"actualCapabilityFabricExecutionObserved"`
	OutputIsContractProbe                       bool `json:"outputIsContractProbe"`
	CandidateCodeGenerated                      bool `json:"candidateCodeGenerated"`
	CandidateCodeExecuted                       bool `json:"candidateCodeExecuted"`
	NetworkRequestedByProbe                     bool `json:"networkRequestedByProbe"`
	LiveAIProviderCalled                        bool `json:"liveAiProviderCalled"`
}

type CapabilityCompositionLineage struct {
	V015ContractReceipt string `json:"v015ContractReceipt"`
	AncestorGapID       string `json:"ancestorGapId"`
	CapabilityNeedID    string `json:"capabilityNeedId"`
}

type CapabilityCompositionPassport struct {
	SourceLineageBound   bool     `json:"sourceLineageBound"`
	TestsDeclared        []string `json:"testsDeclared"`
	TestsExecutedByProbe []string `json:"testsExecutedByProbe"`
	AuthorityUsed        []string `json:"authorityUsed"`
	SideEffectsObserved  []string `json:"sideEffectsObserved"`
	ReplayStatus         string   `json:"replayStatus"`
	GateState            string   `json:"gateState"`
}

type CapabilityCompositionLink struct {
	LinkID               string                        `json:"linkId"`
	BodyKind             string                        `json:"bodyKind"`
	ProposedRoute        string                        `json:"proposedRoute"`
	ProposalState        string                        `json:"proposalState"`
	Purpose              string                        `json:"purpose"`
	Provides             []string                      `json:"provides"`
	Consumes             []string                      `json:"consumes"`
	RequiresAuthority    []string                      `json:"requiresAuthority"`
	AuthorityCeiling     []string                      `json:"authorityCeiling"`
	PotentialSideEffects []string                      `json:"potentialSideEffects"`
	Determinism          string                        `json:"determinism"`
	Lineage              CapabilityCompositionLineage  `json:"lineage"`
	VerificationPassport CapabilityCompositionPassport `json:"verificationPassport"`
}

type CapabilityCompositionChain struct {
	ChainID          string                      `json:"chainId"`
	ProposalState    string                      `json:"proposalState"`
	RootNeed         string                      `json:"rootNeed"`
	RequiredOutput   string                      `json:"requiredOutput"`
	AuthorityCeiling []string                    `json:"authorityCeiling"`
	Links            []CapabilityCompositionLink `json:"links"`
	Outcome          string                      `json:"outcome"`
	Reason           *string                     `json:"reason"`
}

type CapabilityCompositionMissingLinkCase struct {
	CaseID              string `json:"caseId"`
	RemovedLinkID       string `json:"removedLinkId"`
	ExpectedOutcome     string `json:"expectedOutcome"`
	ExpectedReason      string `json:"expectedReason"`
	InventedReplacement bool   `json:"inventedReplacement"`
}

type CapabilityCompositionRefusal struct {
	CaseID             string   `json:"caseId"`
	ChildLinkID        string   `json:"childLinkId"`
	RequestedAuthority []string `json:"requestedAuthority"`
	Decision           string   `json:"decision"`
	Reason             string   `json:"reason"`
	CodeGenerated      bool     `json:"codeGenerated"`
}

type CapabilityCompositionDecision struct {
	SelectionPerformed   bool    `json:"selectionPerformed"`
	PreferredComposition *string `json:"preferredComposition"`
	BuildStarted         bool    `json:"buildStarted"`
	Installed            bool    `json:"installed"`
	Registered           bool    `json:"registered"`
	Promoted             bool    `json:"promoted"`
	CanonChanged         bool    `json:"canonChanged"`
}

type CapabilityCompositionTruth struct {
	SharedAncestorNeedAcrossLinks     bool `json:"sharedAncestorNeedAcrossLinks"`
	SharedAuthorityCeilingAcrossLinks bool `json:"sharedAuthorityCeilingAcrossLinks"`
	TypedProvidesConsumesClosed       bool `json:"typedProvidesConsumesClosed"`
	MissingLinkReturnsHold            bool `json:"missingLinkReturnsHold"`
	MissingCapabilityInvented         bool `json:"missingCapabilityInvented"`
	ChildAuthorityEscalationRefused   bool `json:"childAuthorityEscalationRefused"`
	VerificationPassportsPresent      bool `json:"verificationPassportsPresent"`
	WaldoOwnsCapabilityFabric         bool `json:"waldoOwnsCapabilityFabric"`
	WaldoOwnsComposition              bool `json:"waldoOwnsComposition"`
	HistoricalV015ReceiptRewritten    bool `json:"historicalV015ReceiptRewritten"`
}

type CapabilityCompositionContract struct {
	Schema           string                                 `json:"schema"`
	Status           string                                 `json:"status"`
	Challenge        string                                 `json:"challenge"`
	Source           CapabilityCompositionSource            `json:"source"`
	ProbeBoundary    CapabilityCompositionBoundary          `json:"probeBoundary"`
	Chain            CapabilityCompositionChain             `json:"chain"`
	MissingLinkCases []CapabilityCompositionMissingLinkCase `json:"missingLinkCases"`
	RefusalCases     []CapabilityCompositionRefusal         `json:"refusalCases"`
	Decision         CapabilityCompositionDecision          `json:"decision"`
	Truth            CapabilityCompositionTruth             `json:"truth"`
	Authority        string                                 `json:"authority"`
	ReceiptDigest    string                                 `json:"receiptDigest"`
}

type CapabilityCompositionWitness struct {
	Schema              string   `json:"schema"`
	State               string   `json:"state"`
	SourceV015Receipt   string   `json:"source_v015_receipt"`
	SourceReceiptSHA256 string   `json:"source_receipt_sha256"`
	AncestorGapID       string   `json:"ancestor_gap_id"`
	CapabilityNeedID    string   `json:"capability_need_id"`
	ChainID             string   `json:"chain_id"`
	LinkIDs             []string `json:"link_ids"`
	MissingLinkHolds    int      `json:"missing_link_holds"`
	AuthorityRefusals   int      `json:"authority_refusals"`
	Authority           string   `json:"authority"`
	WitnessSHA256       string   `json:"witness_sha256,omitempty"`
}

func VerifyCapabilityCompositionContract(data []byte) (CapabilityCompositionContract, error) {
	var c CapabilityCompositionContract
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("decode capability composition contract: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("capability composition contract has trailing JSON")
		}
		return CapabilityCompositionContract{}, err
	}
	if err := c.Validate(); err != nil {
		return CapabilityCompositionContract{}, err
	}
	expected, err := capabilityCompositionExternalDigest(c)
	if err != nil {
		return CapabilityCompositionContract{}, err
	}
	if expected != c.ReceiptDigest {
		return CapabilityCompositionContract{}, fmt.Errorf("capability composition receipt digest mismatch: expected %s, got %s", expected, c.ReceiptDigest)
	}
	return c, nil
}

func (c CapabilityCompositionContract) Validate() error {
	if c.Schema != CapabilityCompositionSchema || c.Status != CapabilityCompositionStatus || c.Challenge != CapabilityCompositionChallenge {
		return errors.New("capability composition schema/status/challenge mismatch")
	}
	if c.Source.V015ContractReceipt != CapabilityCompositionV015Receipt || c.Source.AncestorGapID != CapabilityCompositionGapID || c.Source.CapabilityNeedID != CapabilityCompositionNeedID {
		return errors.New("capability composition source binding mismatch")
	}
	b := c.ProbeBoundary
	if b.LocalCapabilityFabricImplementationIncluded || b.ActualCapabilityFabricExecutionObserved || !b.OutputIsContractProbe || b.CandidateCodeGenerated || b.CandidateCodeExecuted || b.NetworkRequestedByProbe || b.LiveAIProviderCalled {
		return errors.New("capability composition truth boundary drifted")
	}
	ch := c.Chain
	if ch.ChainID != "material-evidence-view-readonly-chain" || ch.ProposalState != "PROPOSAL_ONLY" || ch.RootNeed != CapabilityCompositionNeedID || ch.RequiredOutput != "material-evidence-view-verified" || !stringSliceExact(ch.AuthorityCeiling, []string{CapabilityCompositionAuthority}) || ch.Outcome != "PROPOSAL_COMPLETE" || ch.Reason != nil {
		return errors.New("capability composition chain declaration mismatch")
	}
	if len(ch.Links) != 3 {
		return errors.New("capability composition chain must contain exactly three declared links")
	}
	if err := validateCompositionLinks(ch.Links); err != nil {
		return err
	}
	outcome, reason := evaluateComposition(ch.Links)
	if outcome != "PROPOSAL_COMPLETE" || reason != "" {
		return fmt.Errorf("capability composition chain does not close: %s/%s", outcome, reason)
	}
	if len(c.MissingLinkCases) != 3 {
		return errors.New("capability composition must retain one missing-link HOLD case per declared link")
	}
	seenCases := map[string]bool{}
	for _, mc := range c.MissingLinkCases {
		if strings.TrimSpace(mc.CaseID) == "" || seenCases[mc.CaseID] {
			return errors.New("capability composition missing-link case id empty or duplicated")
		}
		seenCases[mc.CaseID] = true
		if mc.ExpectedOutcome != "HOLD" || mc.ExpectedReason != "MISSING_LINK" || mc.InventedReplacement {
			return fmt.Errorf("missing-link case %s must HOLD without invention", mc.CaseID)
		}
		if !containsLinkID(ch.Links, mc.RemovedLinkID) {
			return fmt.Errorf("missing-link case %s references unknown link %s", mc.CaseID, mc.RemovedLinkID)
		}
		reduced := removeLink(ch.Links, mc.RemovedLinkID)
		gotOutcome, gotReason := evaluateComposition(reduced)
		if gotOutcome != mc.ExpectedOutcome || gotReason != mc.ExpectedReason {
			return fmt.Errorf("missing-link case %s evaluation mismatch: got %s/%s", mc.CaseID, gotOutcome, gotReason)
		}
	}
	if len(c.RefusalCases) != 1 {
		return errors.New("capability composition must retain one child-authority escalation refusal")
	}
	r := c.RefusalCases[0]
	if r.CaseID != "hand-authority-escalation" || r.ChildLinkID != "public-artifact-reader-hand" || r.Decision != "REFUSED" || r.Reason != "CHILD_AUTHORITY_EXCEEDS_ANCESTOR_GAP" || r.CodeGenerated {
		return errors.New("capability composition authority refusal drifted")
	}
	for _, must := range []string{"WRITE_WORKSPACE", "INSTALL_CAPABILITY", "PROMOTE", "CANON"} {
		if !capCompositionContainsString(r.RequestedAuthority, must) {
			return fmt.Errorf("authority refusal missing forbidden authority %s", must)
		}
	}
	d := c.Decision
	if d.SelectionPerformed || d.PreferredComposition != nil || d.BuildStarted || d.Installed || d.Registered || d.Promoted || d.CanonChanged {
		return errors.New("capability composition contract unexpectedly selected or acted")
	}
	t := c.Truth
	if !t.SharedAncestorNeedAcrossLinks || !t.SharedAuthorityCeilingAcrossLinks || !t.TypedProvidesConsumesClosed || !t.MissingLinkReturnsHold || t.MissingCapabilityInvented || !t.ChildAuthorityEscalationRefused || !t.VerificationPassportsPresent || t.WaldoOwnsCapabilityFabric || t.WaldoOwnsComposition || t.HistoricalV015ReceiptRewritten {
		return errors.New("capability composition truth claims drifted")
	}
	if c.Authority != "NONE" {
		return errors.New("capability composition authority must remain NONE")
	}
	if !validSHA(c.ReceiptDigest) {
		return errors.New("capability composition receipt digest must be sha256:<64 hex>")
	}
	return nil
}

func validateCompositionLinks(links []CapabilityCompositionLink) error {
	expected := []struct {
		id, kind, route, purpose, consume, provide, determinism string
		authority, effects                                      []string
	}{
		{"material-view-request-skill", "SKILL", "MODULAR_SKILL_BUILDER", "Normalize the inherited evidence gap into a bounded material-view request contract.", CapabilityCompositionNeedID, "bounded-material-view-request", "DETERMINISTIC", nil, nil},
		{"public-artifact-reader-hand", "HAND", "MODULAR_HAND_BUILDER", "Represent read-only acquisition of the inherited public material artifact view.", "bounded-material-view-request", "public-artifact-material-view", "DETERMINISTIC_REQUEST_BOUNDED_EXTERNAL_AVAILABILITY", []string{CapabilityCompositionAuthority}, []string{"READ_PUBLIC_ARTIFACT"}},
		{"material-view-verifier-organ", "ORGAN", "ORGAN_FABRIC", "Verify that the supplied material view satisfies the inherited evidence requirement without widening authority.", "public-artifact-material-view", "material-evidence-view-verified", "DETERMINISTIC", nil, nil},
	}
	ids := map[string]bool{}
	for i, l := range links {
		e := expected[i]
		if l.LinkID != e.id || ids[l.LinkID] || l.BodyKind != e.kind || l.ProposedRoute != e.route || l.ProposalState != "PROPOSAL_ONLY" || l.Purpose != e.purpose || !stringSliceExact(l.Consumes, []string{e.consume}) || !stringSliceExact(l.Provides, []string{e.provide}) || !stringSliceExact(l.RequiresAuthority, e.authority) || !stringSliceExact(l.AuthorityCeiling, []string{CapabilityCompositionAuthority}) || !stringSliceExact(l.PotentialSideEffects, e.effects) || l.Determinism != e.determinism {
			return fmt.Errorf("capability composition link %d (%s) shape drifted", i, l.LinkID)
		}
		ids[l.LinkID] = true
		if l.Lineage.V015ContractReceipt != CapabilityCompositionV015Receipt || l.Lineage.AncestorGapID != CapabilityCompositionGapID || l.Lineage.CapabilityNeedID != CapabilityCompositionNeedID {
			return fmt.Errorf("capability composition link %s lineage drifted", l.LinkID)
		}
		p := l.VerificationPassport
		if !p.SourceLineageBound || len(p.TestsDeclared) < 3 || len(p.TestsExecutedByProbe) != 0 || len(p.AuthorityUsed) != 0 || len(p.SideEffectsObserved) != 0 || p.ReplayStatus != "PROPOSAL_REPLAY_MATCHED" || p.GateState != "HOLD_FOR_REAL_CAPABILITY_FABRIC" {
			return fmt.Errorf("capability composition link %s passport drifted", l.LinkID)
		}
		for _, a := range l.RequiresAuthority {
			if a != CapabilityCompositionAuthority {
				return fmt.Errorf("capability composition link %s requests authority above ancestor ceiling", l.LinkID)
			}
		}
	}
	return nil
}

func evaluateComposition(links []CapabilityCompositionLink) (string, string) {
	if len(links) != 3 {
		return "HOLD", "MISSING_LINK"
	}
	expectedIDs := []string{"material-view-request-skill", "public-artifact-reader-hand", "material-view-verifier-organ"}
	for i, id := range expectedIDs {
		if links[i].LinkID != id {
			return "HOLD", "MISSING_LINK"
		}
	}
	if !stringSliceExact(links[0].Consumes, []string{CapabilityCompositionNeedID}) || len(links[0].Provides) != 1 || !stringSliceExact(links[1].Consumes, links[0].Provides) || len(links[1].Provides) != 1 || !stringSliceExact(links[2].Consumes, links[1].Provides) || !stringSliceExact(links[2].Provides, []string{"material-evidence-view-verified"}) {
		return "HOLD", "MISSING_LINK"
	}
	return "PROPOSAL_COMPLETE", ""
}

func WitnessCapabilityCompositionContract(data []byte) (CapabilityCompositionWitness, error) {
	c, err := VerifyCapabilityCompositionContract(data)
	if err != nil {
		return CapabilityCompositionWitness{}, err
	}
	linkIDs := make([]string, 0, len(c.Chain.Links))
	for _, l := range c.Chain.Links {
		linkIDs = append(linkIDs, l.LinkID)
	}
	w := CapabilityCompositionWitness{
		Schema:              CapabilityCompositionWitnessSchema,
		State:               CapabilityCompositionWitnessReady,
		SourceV015Receipt:   CapabilityCompositionV015Receipt,
		SourceReceiptSHA256: c.ReceiptDigest,
		AncestorGapID:       CapabilityCompositionGapID,
		CapabilityNeedID:    CapabilityCompositionNeedID,
		ChainID:             c.Chain.ChainID,
		LinkIDs:             linkIDs,
		MissingLinkHolds:    len(c.MissingLinkCases),
		AuthorityRefusals:   len(c.RefusalCases),
		Authority:           "NONE",
	}
	raw, err := json.Marshal(w)
	if err != nil {
		return CapabilityCompositionWitness{}, err
	}
	sum := sha256.Sum256(raw)
	w.WitnessSHA256 = hex.EncodeToString(sum[:])
	return w, nil
}

func SealCapabilityCompositionContract(c CapabilityCompositionContract) ([]byte, error) {
	digest, err := capabilityCompositionExternalDigest(c)
	if err != nil {
		return nil, err
	}
	c.ReceiptDigest = digest
	return json.MarshalIndent(c, "", "  ")
}

func capabilityCompositionExternalDigest(c CapabilityCompositionContract) (string, error) {
	c.ReceiptDigest = ""
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func removeLink(in []CapabilityCompositionLink, id string) []CapabilityCompositionLink {
	out := make([]CapabilityCompositionLink, 0, len(in))
	for _, l := range in {
		if l.LinkID != id {
			out = append(out, l)
		}
	}
	return out
}

func containsLinkID(links []CapabilityCompositionLink, id string) bool {
	for _, l := range links {
		if l.LinkID == id {
			return true
		}
	}
	return false
}

func stringSliceExact(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func capCompositionContainsString(items []string, want string) bool {
	for _, s := range items {
		if s == want {
			return true
		}
	}
	return false
}

func validSHA(v string) bool {
	if !strings.HasPrefix(v, "sha256:") || len(v) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(v, "sha256:"))
	return err == nil
}
