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
	"strings"
)

const (
	CapabilityFabricProbeSchema        = "axm.capability-fabric-contract-probe/v0.15"
	CapabilityFabricProbeStatus        = "CONTRACT_PROBE_ONLY"
	CapabilityFabricProbeChallenge     = "ONE_CAPABILITY_MANY_BODIES"
	CapabilityFabricProbeWitnessSchema = "axm.waldo-witness.capability-fabric-contract-probe/v0.15"
	CapabilityFabricProbeWitnessReady  = "CAPABILITY_FABRIC_CONTRACT_WITNESSED"

	CapabilityFabricProbeReturnPacketSHA256 = "sha256:b82cf3a8d5d02e8a979c044b4bcd98c1516390950647949172e34651616b90d2"
	CapabilityFabricProbeLedgerSHA256       = "sha256:d8f5407b6fde3335385967c5d837d563d8474841e2a00038267e6010b06f70ec"
	CapabilityFabricProbeGapID              = "v014-material-evidence-view-incomplete"
	CapabilityFabricProbeCapabilityID       = "material-evidence-view-completion"
)

var capabilityFabricProbeKinds = []string{"CREATION_COMPOSITION", "HAND", "ORGAN", "SKILL"}

type CapabilityFabricProbeSource struct {
	WaldoReturnPacketDigest string `json:"waldoReturnPacketDigest"`
	V014LedgerReceipt       string `json:"v014LedgerReceipt"`
	AncestorGapID           string `json:"ancestorGapId"`
	SourceFinding           string `json:"sourceFinding"`
}

type CapabilityFabricRoutingAdjustment struct {
	PreviousReturnDestination string `json:"previousReturnDestination"`
	ProbeDestination          string `json:"probeDestination"`
	PreviousPacketRewritten   bool   `json:"previousPacketRewritten"`
	Reason                    string `json:"reason"`
}

type CapabilityFabricCapability struct {
	CapabilityID             string   `json:"capabilityId"`
	Purpose                  string   `json:"purpose"`
	AuthorityCeiling         []string `json:"authorityCeiling"`
	CandidateEmbodimentKinds []string `json:"candidateEmbodimentKinds"`
}

type CapabilityFabricProbeBoundary struct {
	LocalCapabilityFabricImplementationIncluded        bool `json:"localCapabilityFabricImplementationIncluded"`
	ActualCapabilityFabricExecutionObserved            bool `json:"actualCapabilityFabricExecutionObserved"`
	OutputIsContractProbe                              bool `json:"outputIsContractProbe"`
	RouteNamesAreContractLabelsNotCanonicalModuleNames bool `json:"routeNamesAreContractLabelsNotCanonicalModuleNames"`
	CandidateCodeGenerated                             bool `json:"candidateCodeGenerated"`
	CandidateCodeExecuted                              bool `json:"candidateCodeExecuted"`
	NetworkRequestedByProbe                            bool `json:"networkRequestedByProbe"`
	LiveAIProviderCalled                               bool `json:"liveAiProviderCalled"`
}

type CapabilityFabricLineage struct {
	SourceReturnPacketDigest  string `json:"sourceReturnPacketDigest"`
	AncestorGapID             string `json:"ancestorGapId"`
	SourceLedgerReceiptDigest string `json:"sourceLedgerReceiptDigest"`
}

type CapabilityFabricPassport struct {
	SourceLineageBound   bool     `json:"sourceLineageBound"`
	TestsDeclared        []string `json:"testsDeclared"`
	TestsExecutedByProbe []string `json:"testsExecutedByProbe"`
	AuthorityUsed        []string `json:"authorityUsed"`
	SideEffectsObserved  []string `json:"sideEffectsObserved"`
	ReplayStatus         string   `json:"replayStatus"`
	GateState            string   `json:"gateState"`
}

type CapabilityFabricEmbodiment struct {
	BodyID               string                   `json:"bodyId"`
	BodyKind             string                   `json:"bodyKind"`
	ProposedRoute        string                   `json:"proposedRoute"`
	ProposalState        string                   `json:"proposalState"`
	Purpose              string                   `json:"purpose"`
	RequiresAuthority    []string                 `json:"requiresAuthority"`
	AuthorityCeiling     []string                 `json:"authorityCeiling"`
	Provides             []string                 `json:"provides"`
	Consumes             []string                 `json:"consumes"`
	PotentialSideEffects []string                 `json:"potentialSideEffects"`
	Determinism          string                   `json:"determinism"`
	Components           []string                 `json:"components,omitempty"`
	Lineage              CapabilityFabricLineage  `json:"lineage"`
	VerificationPassport CapabilityFabricPassport `json:"verificationPassport"`
}

type CapabilityFabricRefusal struct {
	ProposalID         string   `json:"proposalId"`
	BodyKind           string   `json:"bodyKind"`
	RequestedAuthority []string `json:"requestedAuthority"`
	Decision           string   `json:"decision"`
	Reason             string   `json:"reason"`
	CodeGenerated      bool     `json:"codeGenerated"`
}

type CapabilityFabricDecision struct {
	SelectionPerformed  bool    `json:"selectionPerformed"`
	PreferredEmbodiment *string `json:"preferredEmbodiment"`
	BuildStarted        bool    `json:"buildStarted"`
	Installed           bool    `json:"installed"`
	Registered          bool    `json:"registered"`
	Promoted            bool    `json:"promoted"`
	CanonChanged        bool    `json:"canonChanged"`
}

type CapabilityFabricTruth struct {
	SameAncestorGapAcrossBodies      bool `json:"sameAncestorGapAcrossBodies"`
	SameAuthorityCeilingAcrossBodies bool `json:"sameAuthorityCeilingAcrossBodies"`
	ProvenancePreserved              bool `json:"provenancePreserved"`
	VerificationPassportsPresent     bool `json:"verificationPassportsPresent"`
	AuthorityEscalationRefused       bool `json:"authorityEscalationRefused"`
	WaldoOwnsCapabilityFabric        bool `json:"waldoOwnsCapabilityFabric"`
	CreationFabricReplaced           bool `json:"creationFabricReplaced"`
	OrganFabricReplaced              bool `json:"organFabricReplaced"`
	HistoricalHoldRewritten          bool `json:"historicalHoldRewritten"`
}

type CapabilityFabricContractProbe struct {
	Schema            string                            `json:"schema"`
	Status            string                            `json:"status"`
	Challenge         string                            `json:"challenge"`
	Source            CapabilityFabricProbeSource       `json:"source"`
	RoutingAdjustment CapabilityFabricRoutingAdjustment `json:"routingAdjustment"`
	Capability        CapabilityFabricCapability        `json:"capability"`
	ProbeBoundary     CapabilityFabricProbeBoundary     `json:"probeBoundary"`
	Embodiments       []CapabilityFabricEmbodiment      `json:"embodiments"`
	RefusalCases      []CapabilityFabricRefusal         `json:"refusalCases"`
	Decision          CapabilityFabricDecision          `json:"decision"`
	Truth             CapabilityFabricTruth             `json:"truth"`
	Authority         string                            `json:"authority"`
	ReceiptDigest     string                            `json:"receiptDigest"`
}

type CapabilityFabricProbeWitness struct {
	Schema                   string    `json:"schema"`
	State                    string    `json:"state"`
	SourceReceiptSHA256      string    `json:"source_receipt_sha256"`
	SourceReturnPacketSHA256 string    `json:"source_return_packet_sha256"`
	AncestorGapID            string    `json:"ancestor_gap_id"`
	CapabilityID             string    `json:"capability_id"`
	EmbodimentKinds          []string  `json:"embodiment_kinds"`
	ProposalCount            int       `json:"proposal_count"`
	RefusalCount             int       `json:"refusal_count"`
	AuthorityCeiling         []string  `json:"authority_ceiling"`
	Notices                  []string  `json:"notices"`
	Authority                Authority `json:"authority"`
	WitnessSHA256            string    `json:"witness_sha256,omitempty"`
}

func VerifyCapabilityFabricContractProbe(data []byte) (CapabilityFabricContractProbe, error) {
	var probe CapabilityFabricContractProbe
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&probe); err != nil {
		return probe, fmt.Errorf("decode capability-fabric contract probe: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("capability-fabric contract probe has trailing JSON")
		}
		return CapabilityFabricContractProbe{}, err
	}
	if err := probe.Validate(); err != nil {
		return CapabilityFabricContractProbe{}, err
	}
	expected, err := capabilityFabricProbeExternalDigest(data)
	if err != nil {
		return CapabilityFabricContractProbe{}, err
	}
	if expected != probe.ReceiptDigest {
		return CapabilityFabricContractProbe{}, fmt.Errorf("capability-fabric receipt digest mismatch: expected %s, got %s", expected, probe.ReceiptDigest)
	}
	return probe, nil
}

func (p CapabilityFabricContractProbe) Validate() error {
	if p.Schema != CapabilityFabricProbeSchema || p.Status != CapabilityFabricProbeStatus || p.Challenge != CapabilityFabricProbeChallenge {
		return errors.New("capability-fabric probe schema/status/challenge mismatch")
	}
	if p.Source.WaldoReturnPacketDigest != CapabilityFabricProbeReturnPacketSHA256 ||
		p.Source.V014LedgerReceipt != CapabilityFabricProbeLedgerSHA256 ||
		p.Source.AncestorGapID != CapabilityFabricProbeGapID ||
		p.Source.SourceFinding != "MATERIAL_EVIDENCE_VIEW_INCOMPLETE" {
		return errors.New("capability-fabric source binding mismatch")
	}
	if p.RoutingAdjustment.PreviousReturnDestination != "Creation Fabric evidence intake" ||
		p.RoutingAdjustment.ProbeDestination != "Capability Fabric intake" ||
		p.RoutingAdjustment.PreviousPacketRewritten ||
		p.RoutingAdjustment.Reason != "CAPABILITY_FABRIC_NOW_SITS_ABOVE_CREATION_AND_ORGAN_EMBODIMENT_LANES" {
		return errors.New("capability-fabric routing adjustment drifted")
	}
	if p.Capability.CapabilityID != CapabilityFabricProbeCapabilityID ||
		strings.TrimSpace(p.Capability.Purpose) == "" ||
		!capProbeStringSliceExact(p.Capability.AuthorityCeiling, []string{"READ_ONLY_PUBLIC_ARTIFACT_ACCESS"}) ||
		!capProbeStringSliceExact(p.Capability.CandidateEmbodimentKinds, []string{"ORGAN", "HAND", "SKILL", "CREATION_COMPOSITION"}) {
		return errors.New("capability-fabric capability declaration mismatch")
	}
	b := p.ProbeBoundary
	if b.LocalCapabilityFabricImplementationIncluded || b.ActualCapabilityFabricExecutionObserved ||
		!b.OutputIsContractProbe || !b.RouteNamesAreContractLabelsNotCanonicalModuleNames ||
		b.CandidateCodeGenerated || b.CandidateCodeExecuted || b.NetworkRequestedByProbe || b.LiveAIProviderCalled {
		return errors.New("capability-fabric probe truth boundary drifted")
	}
	if len(p.Embodiments) != 4 {
		return errors.New("capability-fabric probe must preserve exactly four embodiment proposals")
	}
	seenIDs := map[string]bool{}
	seenKinds := map[string]bool{}
	for _, e := range p.Embodiments {
		if strings.TrimSpace(e.BodyID) == "" || seenIDs[e.BodyID] {
			return errors.New("capability-fabric embodiment id empty or duplicated")
		}
		seenIDs[e.BodyID] = true
		if !capabilityFabricProbeKind(e.BodyKind) || seenKinds[e.BodyKind] {
			return fmt.Errorf("capability-fabric embodiment kind invalid or duplicated: %s", e.BodyKind)
		}
		seenKinds[e.BodyKind] = true
		if e.ProposalState != "PROPOSAL_ONLY" || strings.TrimSpace(e.Purpose) == "" {
			return fmt.Errorf("capability-fabric embodiment %s is not proposal-only", e.BodyID)
		}
		if !capProbeStringSliceExact(e.AuthorityCeiling, []string{"READ_ONLY_PUBLIC_ARTIFACT_ACCESS"}) {
			return fmt.Errorf("capability-fabric embodiment %s authority ceiling drifted", e.BodyID)
		}
		for _, a := range e.RequiresAuthority {
			if a != "READ_ONLY_PUBLIC_ARTIFACT_ACCESS" {
				return fmt.Errorf("capability-fabric embodiment %s requests authority above the gap ceiling", e.BodyID)
			}
		}
		if e.Lineage.SourceReturnPacketDigest != CapabilityFabricProbeReturnPacketSHA256 ||
			e.Lineage.AncestorGapID != CapabilityFabricProbeGapID ||
			e.Lineage.SourceLedgerReceiptDigest != CapabilityFabricProbeLedgerSHA256 {
			return fmt.Errorf("capability-fabric embodiment %s lineage mismatch", e.BodyID)
		}
		v := e.VerificationPassport
		if !v.SourceLineageBound || len(v.TestsDeclared) < 3 || len(v.TestsExecutedByProbe) != 0 ||
			len(v.AuthorityUsed) != 0 || len(v.SideEffectsObserved) != 0 ||
			v.ReplayStatus != "PROPOSAL_REPLAY_MATCHED" || v.GateState != "HOLD_FOR_REAL_CAPABILITY_FABRIC" {
			return fmt.Errorf("capability-fabric embodiment %s verification passport drifted", e.BodyID)
		}
		if err := validateCapabilityFabricEmbodimentShape(e); err != nil {
			return err
		}
	}
	if len(p.RefusalCases) != 1 {
		return errors.New("capability-fabric probe must contain one authority-escalation refusal")
	}
	r := p.RefusalCases[0]
	if r.ProposalID != "auto-promoting-repair-hand" || r.BodyKind != "HAND" ||
		r.Decision != "REFUSED" || r.Reason != "AUTHORITY_EXCEEDS_ANCESTOR_GAP" || r.CodeGenerated {
		return errors.New("capability-fabric refusal case drifted")
	}
	for _, must := range []string{"WRITE_WORKSPACE", "INSTALL_CAPABILITY", "PROMOTE", "CANON"} {
		if !capProbeContainsString(r.RequestedAuthority, must) {
			return fmt.Errorf("capability-fabric refusal case missing forbidden authority %s", must)
		}
	}
	d := p.Decision
	if d.SelectionPerformed || d.PreferredEmbodiment != nil || d.BuildStarted || d.Installed || d.Registered || d.Promoted || d.CanonChanged {
		return errors.New("capability-fabric probe unexpectedly selected or acted")
	}
	t := p.Truth
	if !t.SameAncestorGapAcrossBodies || !t.SameAuthorityCeilingAcrossBodies || !t.ProvenancePreserved ||
		!t.VerificationPassportsPresent || !t.AuthorityEscalationRefused ||
		t.WaldoOwnsCapabilityFabric || t.CreationFabricReplaced || t.OrganFabricReplaced || t.HistoricalHoldRewritten {
		return errors.New("capability-fabric truth boundary drifted")
	}
	if p.Authority != "NONE" {
		return errors.New("capability-fabric probe authority must remain NONE")
	}
	if err := validatePeerSHA("capability-fabric receipt digest", p.ReceiptDigest); err != nil {
		return err
	}
	return nil
}

func validateCapabilityFabricEmbodimentShape(e CapabilityFabricEmbodiment) error {
	switch e.BodyKind {
	case "ORGAN":
		if e.ProposedRoute != "ORGAN_FABRIC" || len(e.RequiresAuthority) != 0 ||
			len(e.PotentialSideEffects) != 0 || e.Determinism != "DETERMINISTIC" {
			return errors.New("ORGAN embodiment shape drifted")
		}
	case "HAND":
		if e.ProposedRoute != "MODULAR_HAND_BUILDER" ||
			!capProbeStringSliceExact(e.RequiresAuthority, []string{"READ_ONLY_PUBLIC_ARTIFACT_ACCESS"}) ||
			!capProbeStringSliceExact(e.PotentialSideEffects, []string{"READ_PUBLIC_ARTIFACT"}) ||
			e.Determinism != "DETERMINISTIC_REQUEST_BOUNDED_EXTERNAL_AVAILABILITY" {
			return errors.New("HAND embodiment shape drifted")
		}
	case "SKILL":
		if e.ProposedRoute != "MODULAR_SKILL_BUILDER" || len(e.RequiresAuthority) != 0 ||
			len(e.PotentialSideEffects) != 0 || e.Determinism != "DETERMINISTIC_PROCEDURE" {
			return errors.New("SKILL embodiment shape drifted")
		}
	case "CREATION_COMPOSITION":
		want := []string{
			"material-evidence-completion-skill",
			"material-evidence-view-guard-organ",
			"public-artifact-read-hand",
			"material-evidence-view-guard-organ",
			"existing-witness-b-python-material",
		}
		if e.ProposedRoute != "CREATION_FABRIC" ||
			!capProbeStringSliceExact(e.RequiresAuthority, []string{"READ_ONLY_PUBLIC_ARTIFACT_ACCESS"}) ||
			!capProbeStringSliceExact(e.PotentialSideEffects, []string{"READ_PUBLIC_ARTIFACT"}) ||
			e.Determinism != "PROPOSAL_DETERMINISTIC_EXECUTION_NOT_RUN" ||
			!capProbeStringSliceExact(e.Components, want) {
			return errors.New("CREATION_COMPOSITION embodiment shape drifted")
		}
	}
	return nil
}

func WitnessCapabilityFabricContractProbe(data []byte) (CapabilityFabricProbeWitness, error) {
	p, err := VerifyCapabilityFabricContractProbe(data)
	if err != nil {
		return CapabilityFabricProbeWitness{}, err
	}
	kinds := make([]string, 0, len(p.Embodiments))
	for _, e := range p.Embodiments {
		kinds = append(kinds, e.BodyKind)
	}
	sort.Strings(kinds)
	w := CapabilityFabricProbeWitness{
		Schema:                   CapabilityFabricProbeWitnessSchema,
		State:                    CapabilityFabricProbeWitnessReady,
		SourceReceiptSHA256:      p.ReceiptDigest,
		SourceReturnPacketSHA256: p.Source.WaldoReturnPacketDigest,
		AncestorGapID:            p.Source.AncestorGapID,
		CapabilityID:             p.Capability.CapabilityID,
		EmbodimentKinds:          kinds,
		ProposalCount:            len(p.Embodiments),
		RefusalCount:             len(p.RefusalCases),
		AuthorityCeiling:         append([]string(nil), p.Capability.AuthorityCeiling...),
		Notices: []string{
			"contract probe only; local Capability Fabric implementation was not included or executed",
			"Waldo supplies evidence and gap lineage; it does not own the Capability Fabric",
			"no embodiment was selected, built, installed, promoted, or made CANON",
		},
		Authority: Authority{},
	}
	digest, err := capabilityFabricProbeWitnessDigest(w)
	if err != nil {
		return CapabilityFabricProbeWitness{}, err
	}
	w.WitnessSHA256 = digest
	return w, w.Validate()
}

func (w CapabilityFabricProbeWitness) Validate() error {
	if w.Schema != CapabilityFabricProbeWitnessSchema || w.State != CapabilityFabricProbeWitnessReady {
		return errors.New("capability-fabric witness schema/state mismatch")
	}
	if w.SourceReturnPacketSHA256 != CapabilityFabricProbeReturnPacketSHA256 ||
		w.AncestorGapID != CapabilityFabricProbeGapID || w.CapabilityID != CapabilityFabricProbeCapabilityID {
		return errors.New("capability-fabric witness source binding mismatch")
	}
	if err := validatePeerSHA("capability-fabric witness source receipt", w.SourceReceiptSHA256); err != nil {
		return err
	}
	if !capProbeStringSliceExact(w.EmbodimentKinds, capabilityFabricProbeKinds) || w.ProposalCount != 4 || w.RefusalCount != 1 {
		return errors.New("capability-fabric witness topology mismatch")
	}
	if !capProbeStringSliceExact(w.AuthorityCeiling, []string{"READ_ONLY_PUBLIC_ARTIFACT_ACCESS"}) || len(w.Notices) != 3 || !w.Authority.closed() {
		return errors.New("capability-fabric witness boundary drifted")
	}
	expected, err := capabilityFabricProbeWitnessDigest(w)
	if err != nil {
		return err
	}
	if expected != w.WitnessSHA256 {
		return errors.New("capability-fabric witness digest mismatch")
	}
	return nil
}

func capabilityFabricProbeExternalDigest(data []byte) (string, error) {
	var root map[string]any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return "", err
	}
	delete(root, "receiptDigest")
	canonical, err := encodePeerCanonical(root)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func capabilityFabricProbeWitnessDigest(w CapabilityFabricProbeWitness) (string, error) {
	w.WitnessSHA256 = ""
	b, err := json.Marshal(w)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func capabilityFabricProbeKind(s string) bool {
	for _, k := range capabilityFabricProbeKinds {
		if s == k {
			return true
		}
	}
	return false
}

func capProbeStringSliceExact(got, want []string) bool {
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

func capProbeContainsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
