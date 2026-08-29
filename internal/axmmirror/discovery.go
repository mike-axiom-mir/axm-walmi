package axmmirror

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	DiscoveryStanceRequestSchema = "axm.waldo-witness.discovery-stance-request/v0.1"
	DiscoveryStancePacketSchema  = "axm.waldo-witness.discovery-stance-packet/v0.1"

	DiscoveryStateReady          = "DISCOVERY_STANCE_READY"
	DiscoveryStateEvidenceHold   = "HOLD_AI_NATIVE_SEAM_EVIDENCE"
	DiscoveryStateClosureHold    = "HOLD_AI_NATIVE_SEAM_CLOSURE"
	DiscoveryStateInvocationHold = "HOLD_HUMAN_DISCOVERY_INVOCATION"

	HumanDiscoveryNotInvoked = "NOT_INVOKED"
	HumanDiscoveryAdvisory   = "AI_ADVISORY_REQUIRES_HUMAN_REVIEW"
)

var aiNativeDiscoveryStances = []string{
	"evidence", "contradiction", "boundary", "outcome",
	"calibration", "recovery", "lineage", "access",
}

var humanDiscoveryStages = map[string]struct{}{
	"knownSpace": {}, "rejectedDirections": {}, "blindSpots": {}, "seams": {},
	"patterns": {}, "realityChecks": {}, "soulChecks": {}, "minimalChecks": {},
}

var seamSeverityRank = map[string]int{"info": 0, "low": 1, "medium": 2, "high": 3, "critical": 4}

type SeamClosureEvidence struct {
	At           string   `json:"at"`
	Actor        string   `json:"actor"`
	TestStatus   string   `json:"test_status"`
	Statement    string   `json:"statement"`
	EvidenceRefs []string `json:"evidence_refs"`
}

// DiscoverySeam is the AI-native primary finding. It records a typed gap and
// the cheapest check that could disconfirm it; fluent prose cannot close it.
type DiscoverySeam struct {
	ID                 string                `json:"id"`
	Stance             string                `json:"stance"`
	Severity           string                `json:"severity"`
	Status             string                `json:"status"`
	Statement          string                `json:"statement"`
	EvidenceRefs       []string              `json:"evidence_refs"`
	DisconfirmingCheck string                `json:"disconfirming_check"`
	Blocks             []string              `json:"blocks,omitempty"`
	RepairHint         string                `json:"repair_hint,omitempty"`
	ClosureHistory     []SeamClosureEvidence `json:"closure_history,omitempty"`
}

type HumanDiscoveryEntry struct {
	ID         string `json:"id"`
	Stage      string `json:"stage"`
	Text       string `json:"text"`
	ClaimLabel string `json:"claim_label"`
	Source     string `json:"source,omitempty"`
	Rationale  string `json:"rationale,omitempty"`
}

type HumanDiscoveryInput struct {
	ExplicitInvocation bool                  `json:"explicit_invocation"`
	Entries            []HumanDiscoveryEntry `json:"entries"`
}

type DiscoveryStanceRequest struct {
	Schema    string               `json:"schema"`
	RequestID string               `json:"request_id"`
	Subject   EvidenceSubject      `json:"subject"`
	AISeams   []DiscoverySeam      `json:"ai_native_seams"`
	Human     *HumanDiscoveryInput `json:"human_native,omitempty"`
}

type DiscoverySeamSummary struct {
	Open            int    `json:"open"`
	Blocked         int    `json:"blocked"`
	Closed          int    `json:"closed"`
	HighestSeverity string `json:"highest_severity,omitempty"`
}

type AINativeDiscoverySurface struct {
	Identity string               `json:"identity"`
	Priority string               `json:"priority"`
	Stances  []string             `json:"stances"`
	Seams    []DiscoverySeam      `json:"seams"`
	Summary  DiscoverySeamSummary `json:"summary"`
	SHA256   string               `json:"sha256"`
	Boundary string               `json:"boundary"`
}

type HumanDiscoveryAdvisoryEntry struct {
	HumanDiscoveryEntry
	Status string `json:"status"`
}

type HumanNativeDiscoverySurface struct {
	Identity            string                        `json:"identity"`
	Priority            string                        `json:"priority"`
	InvocationState     string                        `json:"invocation_state"`
	ExplicitInvocation  bool                          `json:"explicit_invocation"`
	Entries             []HumanDiscoveryAdvisoryEntry `json:"entries"`
	MayCloseNativeSeams bool                          `json:"may_close_native_seams"`
	MayPromoteLearning  bool                          `json:"may_promote_learning"`
	MayGrantPermissions bool                          `json:"may_grant_permissions"`
	SHA256              string                        `json:"sha256"`
	Boundary            string                        `json:"boundary"`
}

// DiscoveryStancePacket keeps machine-native seam judgement and human-native
// meaning review separate. Neither surface is averaged into a score.
type DiscoveryStancePacket struct {
	Schema        string                      `json:"schema"`
	State         string                      `json:"state"`
	RequestID     string                      `json:"request_id"`
	RequestSHA256 string                      `json:"request_sha256"`
	Subject       EvidenceSubject             `json:"subject"`
	AINative      AINativeDiscoverySurface    `json:"ai_native"`
	HumanNative   HumanNativeDiscoverySurface `json:"human_native"`
	ReceiptSHA256 string                      `json:"receipt_sha256,omitempty"`
	Holds         []string                    `json:"holds,omitempty"`
	Notices       []string                    `json:"notices"`
	Authority     Authority                   `json:"authority"`
}

// BuildDiscoveryStance assembles independently supplied findings. It does not
// invent semantic seams, run probes, close findings, or implement advice.
func BuildDiscoveryStance(request DiscoveryStanceRequest) (DiscoveryStancePacket, error) {
	canonical, err := validateAndCanonicalizeDiscoveryRequest(request)
	if err != nil {
		return DiscoveryStancePacket{}, err
	}
	requestDigest, err := digestJSON(canonical, "discovery stance request")
	if err != nil {
		return DiscoveryStancePacket{}, err
	}
	aiSurface := AINativeDiscoverySurface{
		Identity: "waldo-witness/ai-native-seam-cell", Priority: "PRIMARY",
		Stances: append([]string(nil), aiNativeDiscoveryStances...), Seams: canonical.AISeams,
		Summary:  summarizeDiscoverySeams(canonical.AISeams),
		Boundary: "A seam is a typed gap or tension, not proof of failure, novelty, danger, intelligence, or resolution. Closure requires a passing check and evidence references.",
	}
	aiSurface.SHA256, err = digestJSON(struct {
		Subject EvidenceSubject          `json:"subject"`
		Surface AINativeDiscoverySurface `json:"surface"`
	}{canonical.Subject, aiSurface}, "AI-native discovery surface")
	if err != nil {
		return DiscoveryStancePacket{}, err
	}
	humanSurface := makeHumanDiscoverySurface(canonical.Human)
	humanSurface.SHA256, err = digestJSON(struct {
		Subject EvidenceSubject             `json:"subject"`
		Surface HumanNativeDiscoverySurface `json:"surface"`
	}{canonical.Subject, humanSurface}, "human-native discovery surface")
	if err != nil {
		return DiscoveryStancePacket{}, err
	}
	packet := DiscoveryStancePacket{
		Schema: DiscoveryStancePacketSchema, State: DiscoveryStateReady,
		RequestID: canonical.RequestID, RequestSHA256: requestDigest, Subject: canonical.Subject,
		AINative: aiSurface, HumanNative: humanSurface,
		Notices: []string{
			"AI-native evidence, contradiction, boundary, outcome, calibration, recovery, lineage, and access seams remain the primary machine judgement surface",
			"human-native clarity, usefulness, beginner friction, craft, accessibility, and product-soul observations remain explicit secondary advice",
			"the two surfaces are not averaged, ranked, or allowed to impersonate one another; open seams and human meaning remain separately reviewable",
			"this packet performs no probe, tool call, implementation, permission change, learning promotion, seam closure, or CANON action",
		},
		Authority: Authority{},
	}
	packet.State, packet.Holds = discoveryPacketState(canonical)
	packet.ReceiptSHA256, err = discoveryStancePacketDigest(packet)
	if err != nil {
		return DiscoveryStancePacket{}, err
	}
	if err := packet.Validate(); err != nil {
		return DiscoveryStancePacket{}, fmt.Errorf("generated discovery stance packet: %w", err)
	}
	return packet, nil
}

func validateAndCanonicalizeDiscoveryRequest(request DiscoveryStanceRequest) (DiscoveryStanceRequest, error) {
	if request.Schema != DiscoveryStanceRequestSchema {
		return DiscoveryStanceRequest{}, fmt.Errorf("discovery request schema must be %q", DiscoveryStanceRequestSchema)
	}
	if strings.TrimSpace(request.RequestID) == "" || request.RequestID != strings.TrimSpace(request.RequestID) {
		return DiscoveryStanceRequest{}, errors.New("discovery request_id is required and must be trimmed")
	}
	if err := request.Subject.validate("discovery subject"); err != nil {
		return DiscoveryStanceRequest{}, err
	}
	canonical := request
	canonical.AISeams = append([]DiscoverySeam(nil), request.AISeams...)
	seenSeams := make(map[string]struct{}, len(canonical.AISeams))
	for i := range canonical.AISeams {
		seam, err := canonicalizeDiscoverySeam(canonical.AISeams[i])
		if err != nil {
			return DiscoveryStanceRequest{}, fmt.Errorf("AI-native seam %d: %w", i, err)
		}
		if _, exists := seenSeams[seam.ID]; exists {
			return DiscoveryStanceRequest{}, fmt.Errorf("duplicate AI-native seam %q", seam.ID)
		}
		seenSeams[seam.ID] = struct{}{}
		canonical.AISeams[i] = seam
	}
	sort.Slice(canonical.AISeams, func(i, j int) bool {
		left, right := seamSeverityRank[canonical.AISeams[i].Severity], seamSeverityRank[canonical.AISeams[j].Severity]
		return left > right || left == right && canonical.AISeams[i].ID < canonical.AISeams[j].ID
	})
	if request.Human != nil {
		human := *request.Human
		human.Entries = append([]HumanDiscoveryEntry(nil), request.Human.Entries...)
		sort.Slice(human.Entries, func(i, j int) bool { return human.Entries[i].ID < human.Entries[j].ID })
		for i, entry := range human.Entries {
			if err := validateHumanDiscoveryEntry(entry); err != nil {
				return DiscoveryStanceRequest{}, fmt.Errorf("human discovery entry %d: %w", i, err)
			}
			if i > 0 && human.Entries[i-1].ID == entry.ID {
				return DiscoveryStanceRequest{}, fmt.Errorf("duplicate human discovery entry %q", entry.ID)
			}
		}
		if !human.ExplicitInvocation && len(human.Entries) == 0 {
			canonical.Human = nil
		} else {
			canonical.Human = &human
		}
	}
	return canonical, nil
}

func canonicalizeDiscoverySeam(seam DiscoverySeam) (DiscoverySeam, error) {
	if strings.TrimSpace(seam.ID) == "" || seam.ID != strings.TrimSpace(seam.ID) || strings.TrimSpace(seam.Statement) == "" || seam.Statement != strings.TrimSpace(seam.Statement) {
		return DiscoverySeam{}, errors.New("id and statement are required and must be trimmed")
	}
	if strings.TrimSpace(seam.DisconfirmingCheck) != seam.DisconfirmingCheck || strings.TrimSpace(seam.RepairHint) != seam.RepairHint {
		return DiscoverySeam{}, errors.New("disconfirming_check and repair_hint must be trimmed")
	}
	if !oneOf(seam.Stance, aiNativeDiscoveryStances...) {
		return DiscoverySeam{}, fmt.Errorf("unsupported stance %q", seam.Stance)
	}
	if _, ok := seamSeverityRank[seam.Severity]; !ok {
		return DiscoverySeam{}, fmt.Errorf("unsupported severity %q", seam.Severity)
	}
	if !oneOf(seam.Status, "OPEN", "BLOCKED", "CLOSED") {
		return DiscoverySeam{}, fmt.Errorf("unsupported status %q", seam.Status)
	}
	canonical := seam
	var err error
	canonical.EvidenceRefs, err = canonicalNameSet("seam evidence_refs", seam.EvidenceRefs)
	if err != nil {
		return DiscoverySeam{}, err
	}
	canonical.Blocks, err = canonicalNameSet("seam blocks", seam.Blocks)
	if err != nil {
		return DiscoverySeam{}, err
	}
	canonical.ClosureHistory = append([]SeamClosureEvidence(nil), seam.ClosureHistory...)
	var previous time.Time
	for i := range canonical.ClosureHistory {
		closure := &canonical.ClosureHistory[i]
		if strings.TrimSpace(closure.Actor) == "" || closure.Actor != strings.TrimSpace(closure.Actor) || strings.TrimSpace(closure.Statement) == "" || closure.Statement != strings.TrimSpace(closure.Statement) || !oneOf(closure.TestStatus, "PASS", "FAIL", "HOLD", "UNKNOWN") {
			return DiscoverySeam{}, fmt.Errorf("closure history %d is incomplete", i)
		}
		at, err := time.Parse(time.RFC3339Nano, closure.At)
		if err != nil {
			return DiscoverySeam{}, fmt.Errorf("parse closure history %d at: %w", i, err)
		}
		if !previous.IsZero() && at.Before(previous) {
			return DiscoverySeam{}, errors.New("closure history is not chronological")
		}
		previous = at
		closure.EvidenceRefs, err = canonicalNameSet("closure evidence_refs", closure.EvidenceRefs)
		if err != nil {
			return DiscoverySeam{}, err
		}
	}
	return canonical, nil
}

func validateHumanDiscoveryEntry(entry HumanDiscoveryEntry) error {
	if strings.TrimSpace(entry.ID) == "" || entry.ID != strings.TrimSpace(entry.ID) || strings.TrimSpace(entry.Text) == "" || entry.Text != strings.TrimSpace(entry.Text) {
		return errors.New("id and text are required and must be trimmed")
	}
	if strings.TrimSpace(entry.Source) != entry.Source || strings.TrimSpace(entry.Rationale) != entry.Rationale {
		return errors.New("source and rationale must be trimmed")
	}
	if _, ok := humanDiscoveryStages[entry.Stage]; !ok {
		return fmt.Errorf("unsupported stage %q", entry.Stage)
	}
	if !oneOf(entry.ClaimLabel, "OBSERVATION", "HYPOTHESIS", "QUESTION", "VALUE", "UNKNOWN", "CONTRADICTION") {
		return fmt.Errorf("unsupported claim_label %q", entry.ClaimLabel)
	}
	return nil
}

func makeHumanDiscoverySurface(input *HumanDiscoveryInput) HumanNativeDiscoverySurface {
	surface := HumanNativeDiscoverySurface{
		Identity: "waldo-witness/human-native-discovery", Priority: "SECONDARY",
		InvocationState: HumanDiscoveryNotInvoked, MayCloseNativeSeams: false,
		MayPromoteLearning: false, MayGrantPermissions: false,
		Boundary: "Human-facing usefulness and meaning review is advisory. It cannot outrank AI-native seam evidence or convert model opinion into proof.",
	}
	if input == nil {
		return surface
	}
	surface.ExplicitInvocation = input.ExplicitInvocation
	if input.ExplicitInvocation {
		surface.InvocationState = HumanDiscoveryAdvisory
	}
	surface.Entries = make([]HumanDiscoveryAdvisoryEntry, 0, len(input.Entries))
	for _, entry := range input.Entries {
		surface.Entries = append(surface.Entries, HumanDiscoveryAdvisoryEntry{HumanDiscoveryEntry: entry, Status: HumanDiscoveryAdvisory})
	}
	return surface
}

func summarizeDiscoverySeams(seams []DiscoverySeam) DiscoverySeamSummary {
	result := DiscoverySeamSummary{}
	for _, seam := range seams {
		switch seam.Status {
		case "OPEN":
			result.Open++
		case "BLOCKED":
			result.Blocked++
		case "CLOSED":
			result.Closed++
		}
		if result.HighestSeverity == "" || seamSeverityRank[seam.Severity] > seamSeverityRank[result.HighestSeverity] {
			result.HighestSeverity = seam.Severity
		}
	}
	return result
}

func discoveryPacketState(request DiscoveryStanceRequest) (string, []string) {
	for _, seam := range request.AISeams {
		if len(seam.EvidenceRefs) == 0 || strings.TrimSpace(seam.DisconfirmingCheck) == "" {
			return DiscoveryStateEvidenceHold, []string{"one or more AI-native seams lack evidence references or a cheapest disconfirming check"}
		}
		if seam.Status == "CLOSED" {
			if len(seam.ClosureHistory) == 0 {
				return DiscoveryStateClosureHold, []string{"one or more CLOSED AI-native seams lack closure history"}
			}
			closure := seam.ClosureHistory[len(seam.ClosureHistory)-1]
			if closure.TestStatus != "PASS" || len(closure.EvidenceRefs) == 0 {
				return DiscoveryStateClosureHold, []string{"one or more CLOSED AI-native seams lack a latest passing check with evidence references"}
			}
		}
	}
	if request.Human != nil && !request.Human.ExplicitInvocation && len(request.Human.Entries) != 0 {
		return DiscoveryStateInvocationHold, []string{"human-native Discovery/Stance entries were supplied without explicit invocation"}
	}
	return DiscoveryStateReady, nil
}

func (packet DiscoveryStancePacket) Validate() error {
	if packet.Schema != DiscoveryStancePacketSchema || !oneOf(packet.State, DiscoveryStateReady, DiscoveryStateEvidenceHold, DiscoveryStateClosureHold, DiscoveryStateInvocationHold) {
		return fmt.Errorf("unsupported discovery packet identity %q state %q", packet.Schema, packet.State)
	}
	if strings.TrimSpace(packet.RequestID) == "" {
		return errors.New("discovery packet request_id is required")
	}
	if err := packet.Subject.validate("discovery packet subject"); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"request_sha256":      packet.RequestSHA256,
		"ai_native.sha256":    packet.AINative.SHA256,
		"human_native.sha256": packet.HumanNative.SHA256,
		"receipt_sha256":      packet.ReceiptSHA256,
	} {
		if err := validateSHA256("discovery packet "+name, value); err != nil {
			return err
		}
	}
	if packet.AINative.Identity != "waldo-witness/ai-native-seam-cell" || packet.AINative.Priority != "PRIMARY" || !stringSlicesEqual(packet.AINative.Stances, aiNativeDiscoveryStances) || strings.TrimSpace(packet.AINative.Boundary) == "" {
		return errors.New("discovery packet AI-native primary surface is incomplete")
	}
	for i, seam := range packet.AINative.Seams {
		canonical, err := canonicalizeDiscoverySeam(seam)
		if err != nil {
			return fmt.Errorf("discovery packet seam %d: %w", i, err)
		}
		if !discoverySeamsEqual(canonical, seam) {
			return fmt.Errorf("discovery packet seam %q is not canonical", seam.ID)
		}
		if i > 0 {
			previous := packet.AINative.Seams[i-1]
			if seamSeverityRank[previous.Severity] < seamSeverityRank[seam.Severity] || seamSeverityRank[previous.Severity] == seamSeverityRank[seam.Severity] && previous.ID >= seam.ID {
				return errors.New("discovery packet AI-native seams are unsorted or duplicated")
			}
		}
	}
	if packet.AINative.Summary != summarizeDiscoverySeams(packet.AINative.Seams) {
		return errors.New("discovery packet AI-native summary does not match its seams")
	}
	if packet.HumanNative.Identity != "waldo-witness/human-native-discovery" || packet.HumanNative.Priority != "SECONDARY" || packet.HumanNative.MayCloseNativeSeams || packet.HumanNative.MayPromoteLearning || packet.HumanNative.MayGrantPermissions || strings.TrimSpace(packet.HumanNative.Boundary) == "" {
		return errors.New("discovery packet human-native secondary boundary is incomplete")
	}
	if packet.HumanNative.ExplicitInvocation && packet.HumanNative.InvocationState != HumanDiscoveryAdvisory || !packet.HumanNative.ExplicitInvocation && packet.HumanNative.InvocationState != HumanDiscoveryNotInvoked {
		return errors.New("discovery packet human invocation state is inconsistent")
	}
	for i, entry := range packet.HumanNative.Entries {
		if entry.Status != HumanDiscoveryAdvisory {
			return fmt.Errorf("discovery packet human entry %d is incomplete or not advisory", i)
		}
		if err := validateHumanDiscoveryEntry(entry.HumanDiscoveryEntry); err != nil {
			return fmt.Errorf("discovery packet human entry %d: %w", i, err)
		}
		if i > 0 && packet.HumanNative.Entries[i-1].ID >= entry.ID {
			return errors.New("discovery packet human entries are unsorted or duplicated")
		}
	}
	reconstructed := DiscoveryStanceRequest{
		Schema: DiscoveryStanceRequestSchema, RequestID: packet.RequestID,
		Subject: packet.Subject, AISeams: packet.AINative.Seams,
	}
	if packet.HumanNative.ExplicitInvocation || len(packet.HumanNative.Entries) != 0 {
		entries := make([]HumanDiscoveryEntry, 0, len(packet.HumanNative.Entries))
		for _, entry := range packet.HumanNative.Entries {
			entries = append(entries, entry.HumanDiscoveryEntry)
		}
		reconstructed.Human = &HumanDiscoveryInput{
			ExplicitInvocation: packet.HumanNative.ExplicitInvocation,
			Entries:            entries,
		}
	}
	requestDigest, err := digestJSON(reconstructed, "discovery stance request")
	if err != nil {
		return err
	}
	if requestDigest != packet.RequestSHA256 {
		return errors.New("discovery packet request digest does not match its surfaces")
	}
	aiDigestSurface := packet.AINative
	aiDigestSurface.SHA256 = ""
	aiDigest, err := digestJSON(struct {
		Subject EvidenceSubject          `json:"subject"`
		Surface AINativeDiscoverySurface `json:"surface"`
	}{packet.Subject, aiDigestSurface}, "AI-native discovery surface")
	if err != nil {
		return err
	}
	if aiDigest != packet.AINative.SHA256 {
		return errors.New("discovery packet AI-native digest does not match")
	}
	humanDigestSurface := packet.HumanNative
	humanDigestSurface.SHA256 = ""
	humanDigest, err := digestJSON(struct {
		Subject EvidenceSubject             `json:"subject"`
		Surface HumanNativeDiscoverySurface `json:"surface"`
	}{packet.Subject, humanDigestSurface}, "human-native discovery surface")
	if err != nil {
		return err
	}
	if humanDigest != packet.HumanNative.SHA256 {
		return errors.New("discovery packet human-native digest does not match")
	}
	expectedState, _ := discoveryStateFromPacket(packet)
	if packet.State != expectedState {
		return fmt.Errorf("discovery packet state is %q; surfaces require %q", packet.State, expectedState)
	}
	if packet.State == DiscoveryStateReady {
		if len(packet.Holds) != 0 {
			return errors.New("ready discovery packet carries holds")
		}
	} else if len(packet.Holds) == 0 {
		return errors.New("held discovery packet requires a reason")
	}
	if len(packet.Notices) == 0 || !packet.Authority.closed() {
		return errors.New("discovery packet must state its boundaries and carry closed authority")
	}
	expected, err := discoveryStancePacketDigest(packet)
	if err != nil {
		return err
	}
	if expected != packet.ReceiptSHA256 {
		return fmt.Errorf("discovery packet digest mismatch: expected %s, got %s", packet.ReceiptSHA256, expected)
	}
	return nil
}

func discoveryStateFromPacket(packet DiscoveryStancePacket) (string, []string) {
	request := DiscoveryStanceRequest{AISeams: packet.AINative.Seams}
	if len(packet.HumanNative.Entries) != 0 || packet.HumanNative.ExplicitInvocation {
		entries := make([]HumanDiscoveryEntry, 0, len(packet.HumanNative.Entries))
		for _, entry := range packet.HumanNative.Entries {
			entries = append(entries, entry.HumanDiscoveryEntry)
		}
		request.Human = &HumanDiscoveryInput{ExplicitInvocation: packet.HumanNative.ExplicitInvocation, Entries: entries}
	}
	return discoveryPacketState(request)
}

func discoverySeamsEqual(left, right DiscoverySeam) bool {
	leftDigest, leftErr := digestJSON(left, "canonical discovery seam")
	rightDigest, rightErr := digestJSON(right, "discovery seam")
	return leftErr == nil && rightErr == nil && leftDigest == rightDigest
}

func discoveryStancePacketDigest(packet DiscoveryStancePacket) (string, error) {
	packet.ReceiptSHA256 = ""
	return digestJSON(packet, "discovery stance packet")
}
