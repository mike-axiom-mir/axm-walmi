package axmmirror

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	SituatedContextRequestSchema  = "axm.waldo-witness.situated-context-request/v0.1"
	SituatedContextEnvelopeSchema = "axm.waldo-witness.situated-context-envelope/v0.1"

	SituatedContextReady         = "SITUATED_CONTEXT_READY"
	SituatedContextJoinHold      = "HOLD_SITUATED_CONTEXT_JOIN"
	SituatedContextSensoryHold   = "HOLD_REQUIRED_SENSORY_EVIDENCE"
	SituatedContextSkillHold     = "HOLD_SKILL_CONTINUITY"
	SituatedContextDiscoveryHold = "HOLD_DISCOVERY_STANCE"

	MaxSituatedContextBytes = 512 * 1024

	VerificationSensoryEvidence = "waldo-sensory-evidence-set"
	VerificationSkillContinuity = "waldo-skill-backup-continuity"
	VerificationDiscoveryStance = "waldo-dual-discovery-stance"
	VerificationSituatedContext = "waldo-situated-context-envelope"
)

type SituatedContextRequest struct {
	Schema                        string                   `json:"schema"`
	EnvelopeID                    string                   `json:"envelope_id"`
	TargetAnsweringIdentitySHA256 string                   `json:"target_answering_identity_sha256"`
	Subject                       EvidenceSubject          `json:"subject"`
	ContextPacketSHA256           string                   `json:"context_packet_sha256"`
	RequiredSenses                []string                 `json:"required_senses"`
	SensoryEvidence               []SensoryEvidenceReceipt `json:"sensory_evidence"`
	SkillContinuity               SkillContinuityReceipt   `json:"skill_continuity"`
	DiscoveryStance               DiscoveryStancePacket    `json:"discovery_stance"`
}

// SituatedContextEnvelope binds bounded perception, capability continuity,
// AI-native seams, and human-native advice to one provenance context. It is a
// review surface around the learned clone, not a body or autonomous actor.
type SituatedContextEnvelope struct {
	Schema                        string                   `json:"schema"`
	State                         string                   `json:"state"`
	EnvelopeID                    string                   `json:"envelope_id"`
	RequestSHA256                 string                   `json:"request_sha256"`
	TargetAnsweringIdentitySHA256 string                   `json:"target_answering_identity_sha256"`
	Subject                       EvidenceSubject          `json:"subject"`
	ContextPacketSHA256           string                   `json:"context_packet_sha256"`
	RequiredSenses                []string                 `json:"required_senses"`
	SensoryEvidence               []SensoryEvidenceReceipt `json:"sensory_evidence"`
	SensorySetSHA256              string                   `json:"sensory_set_sha256"`
	SkillContinuity               SkillContinuityReceipt   `json:"skill_continuity"`
	DiscoveryStance               DiscoveryStancePacket    `json:"discovery_stance"`
	PacketBytes                   int                      `json:"packet_bytes"`
	EnvelopeSHA256                string                   `json:"envelope_sha256,omitempty"`
	Holds                         []string                 `json:"holds,omitempty"`
	Notices                       []string                 `json:"notices"`
	Authority                     Authority                `json:"authority"`
}

// BuildSituatedContext performs exact joins against an already validated
// provenance packet. Required senses are explicit per task; supporting all
// thirteen Sensorium contracts does not mean every task silently invokes all.
func BuildSituatedContext(request SituatedContextRequest, context ProvenanceContextPacket) (SituatedContextEnvelope, error) {
	canonical, err := validateAndCanonicalizeSituatedRequest(request)
	if err != nil {
		return SituatedContextEnvelope{}, err
	}
	if err := context.Validate(); err != nil {
		return SituatedContextEnvelope{}, fmt.Errorf("situated provenance context: %w", err)
	}
	requestDigest, err := digestJSON(canonical, "situated context request")
	if err != nil {
		return SituatedContextEnvelope{}, err
	}
	sensoryDigest, err := digestJSON(canonical.SensoryEvidence, "situated sensory evidence set")
	if err != nil {
		return SituatedContextEnvelope{}, err
	}
	envelope := SituatedContextEnvelope{
		Schema: SituatedContextEnvelopeSchema, State: SituatedContextReady,
		EnvelopeID: canonical.EnvelopeID, RequestSHA256: requestDigest,
		TargetAnsweringIdentitySHA256: canonical.TargetAnsweringIdentitySHA256,
		Subject:                       canonical.Subject, ContextPacketSHA256: canonical.ContextPacketSHA256,
		RequiredSenses: canonical.RequiredSenses, SensoryEvidence: canonical.SensoryEvidence,
		SensorySetSHA256: sensoryDigest, SkillContinuity: canonical.SkillContinuity,
		DiscoveryStance: canonical.DiscoveryStance,
		Notices: []string{
			"the envelope carries typed receipts only and excludes raw sensory material, private memory, prompts, hidden reasoning, secrets, and executable skill bytes",
			"support for a sensory or skill contract does not imply adapter availability, invocation, permission, installation, restoration, compatibility, or successful behavior",
			"AI-native seams remain primary and human-native advice remains secondary; neither is collapsed into a quality or intelligence score",
			"the envelope grants no capture, tool, installation, restoration, training, promotion, CANON, permission, or world-action authority",
		},
		Authority: Authority{},
	}
	envelope.State, envelope.Holds = situatedContextState(canonical, context)
	envelope.PacketBytes, err = situatedContextPacketBytes(envelope)
	if err != nil {
		return SituatedContextEnvelope{}, err
	}
	if envelope.PacketBytes > MaxSituatedContextBytes {
		return SituatedContextEnvelope{}, fmt.Errorf("situated context requires %d bytes; maximum is %d", envelope.PacketBytes, MaxSituatedContextBytes)
	}
	envelope.EnvelopeSHA256, err = situatedContextEnvelopeDigest(envelope)
	if err != nil {
		return SituatedContextEnvelope{}, err
	}
	if err := envelope.Validate(); err != nil {
		return SituatedContextEnvelope{}, fmt.Errorf("generated situated context envelope: %w", err)
	}
	return envelope, nil
}

func validateAndCanonicalizeSituatedRequest(request SituatedContextRequest) (SituatedContextRequest, error) {
	if request.Schema != SituatedContextRequestSchema {
		return SituatedContextRequest{}, fmt.Errorf("situated context request schema must be %q", SituatedContextRequestSchema)
	}
	if strings.TrimSpace(request.EnvelopeID) == "" || request.EnvelopeID != strings.TrimSpace(request.EnvelopeID) {
		return SituatedContextRequest{}, errors.New("situated context envelope_id is required and must be trimmed")
	}
	if err := validateSHA256("situated target_answering_identity_sha256", request.TargetAnsweringIdentitySHA256); err != nil {
		return SituatedContextRequest{}, err
	}
	if err := request.Subject.validate("situated subject"); err != nil {
		return SituatedContextRequest{}, err
	}
	if err := validateSHA256("situated context_packet_sha256", request.ContextPacketSHA256); err != nil {
		return SituatedContextRequest{}, err
	}
	if len(request.RequiredSenses) == 0 {
		return SituatedContextRequest{}, errors.New("situated context requires at least one explicitly named sense")
	}
	canonical := request
	var err error
	canonical.RequiredSenses, err = canonicalNameSet("situated required_senses", request.RequiredSenses)
	if err != nil {
		return SituatedContextRequest{}, err
	}
	for _, id := range canonical.RequiredSenses {
		if _, ok := sensoryCapabilities[id]; !ok {
			return SituatedContextRequest{}, fmt.Errorf("situated context has unsupported required sense %q", id)
		}
	}
	canonical.SensoryEvidence = append([]SensoryEvidenceReceipt(nil), request.SensoryEvidence...)
	sort.Slice(canonical.SensoryEvidence, func(i, j int) bool {
		return canonical.SensoryEvidence[i].SenseID < canonical.SensoryEvidence[j].SenseID
	})
	for i, receipt := range canonical.SensoryEvidence {
		if err := receipt.Validate(); err != nil {
			return SituatedContextRequest{}, fmt.Errorf("situated sensory evidence %d: %w", i, err)
		}
		if i > 0 && canonical.SensoryEvidence[i-1].SenseID == receipt.SenseID {
			return SituatedContextRequest{}, fmt.Errorf("situated context has duplicate sensory receipt for %q", receipt.SenseID)
		}
	}
	if len(canonical.SensoryEvidence) != len(canonical.RequiredSenses) {
		return SituatedContextRequest{}, errors.New("situated sensory receipts must exactly match required_senses")
	}
	for i, id := range canonical.RequiredSenses {
		if canonical.SensoryEvidence[i].SenseID != id {
			return SituatedContextRequest{}, fmt.Errorf("situated sensory receipts do not contain exactly required sense %q", id)
		}
	}
	if err := canonical.SkillContinuity.Validate(); err != nil {
		return SituatedContextRequest{}, fmt.Errorf("situated skill continuity: %w", err)
	}
	if err := canonical.DiscoveryStance.Validate(); err != nil {
		return SituatedContextRequest{}, fmt.Errorf("situated discovery stance: %w", err)
	}
	return canonical, nil
}

func situatedContextState(request SituatedContextRequest, context ProvenanceContextPacket) (string, []string) {
	var joins []string
	if context.State != ContextStateReady || request.ContextPacketSHA256 != context.PacketSHA256 {
		joins = append(joins, "the supplied provenance context is not READY or does not match context_packet_sha256")
	}
	identityFact, ok := findProvenanceFact(context, "answering.identity_sha256")
	var identity string
	if !ok || decodeFactValue(identityFact, &identity) != nil || identity != request.TargetAnsweringIdentitySHA256 {
		joins = append(joins, "the provenance context does not bind target_answering_identity_sha256")
	}
	if request.SkillContinuity.TargetAnsweringIdentitySHA256 != request.TargetAnsweringIdentitySHA256 {
		joins = append(joins, "skill continuity targets a different answering identity")
	}
	if request.DiscoveryStance.Subject != request.Subject {
		joins = append(joins, "discovery stance targets a different subject")
	}
	for _, receipt := range request.SensoryEvidence {
		if receipt.Subject != request.Subject {
			joins = append(joins, fmt.Sprintf("sensory receipt %q targets a different subject", receipt.SenseID))
		}
	}
	if len(joins) != 0 {
		return SituatedContextJoinHold, joins
	}
	for _, receipt := range request.SensoryEvidence {
		if receipt.State != SensoryStateReady {
			return SituatedContextSensoryHold, []string{fmt.Sprintf("required sense %q is %s", receipt.SenseID, receipt.State)}
		}
	}
	if request.SkillContinuity.State != SkillContinuityReady {
		return SituatedContextSkillHold, []string{fmt.Sprintf("skill continuity is %s", request.SkillContinuity.State)}
	}
	if request.DiscoveryStance.State != DiscoveryStateReady {
		return SituatedContextDiscoveryHold, []string{fmt.Sprintf("discovery stance is %s", request.DiscoveryStance.State)}
	}
	return SituatedContextReady, nil
}

func (envelope SituatedContextEnvelope) Validate() error {
	if envelope.Schema != SituatedContextEnvelopeSchema || !oneOf(envelope.State, SituatedContextReady, SituatedContextJoinHold, SituatedContextSensoryHold, SituatedContextSkillHold, SituatedContextDiscoveryHold) {
		return fmt.Errorf("unsupported situated context identity %q state %q", envelope.Schema, envelope.State)
	}
	if strings.TrimSpace(envelope.EnvelopeID) == "" || len(envelope.RequiredSenses) == 0 || len(envelope.SensoryEvidence) != len(envelope.RequiredSenses) {
		return errors.New("situated context envelope has incomplete identity or sensory set")
	}
	if err := envelope.Subject.validate("situated envelope subject"); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"request_sha256":                   envelope.RequestSHA256,
		"target_answering_identity_sha256": envelope.TargetAnsweringIdentitySHA256,
		"context_packet_sha256":            envelope.ContextPacketSHA256,
		"sensory_set_sha256":               envelope.SensorySetSHA256,
		"envelope_sha256":                  envelope.EnvelopeSHA256,
	} {
		if err := validateSHA256("situated context "+name, value); err != nil {
			return err
		}
	}
	canonicalSenses, err := canonicalNameSet("situated envelope required_senses", envelope.RequiredSenses)
	if err != nil || !stringSlicesEqual(canonicalSenses, envelope.RequiredSenses) {
		return errors.New("situated envelope required_senses are invalid or not canonical")
	}
	for i, receipt := range envelope.SensoryEvidence {
		if err := receipt.Validate(); err != nil {
			return fmt.Errorf("situated envelope sensory receipt %d: %w", i, err)
		}
		if receipt.SenseID != envelope.RequiredSenses[i] || receipt.Subject != envelope.Subject {
			return errors.New("situated envelope sensory receipts do not match required senses and subject")
		}
	}
	sensoryDigest, err := digestJSON(envelope.SensoryEvidence, "situated sensory evidence set")
	if err != nil {
		return err
	}
	if sensoryDigest != envelope.SensorySetSHA256 {
		return errors.New("situated envelope sensory set digest does not match")
	}
	if err := envelope.SkillContinuity.Validate(); err != nil {
		return fmt.Errorf("situated envelope skill continuity: %w", err)
	}
	if err := envelope.DiscoveryStance.Validate(); err != nil {
		return fmt.Errorf("situated envelope discovery stance: %w", err)
	}
	if envelope.SkillContinuity.TargetAnsweringIdentitySHA256 != envelope.TargetAnsweringIdentitySHA256 || envelope.DiscoveryStance.Subject != envelope.Subject {
		return errors.New("situated envelope contains a mismatched skill or discovery target")
	}
	request := SituatedContextRequest{
		Schema: SituatedContextRequestSchema, EnvelopeID: envelope.EnvelopeID,
		TargetAnsweringIdentitySHA256: envelope.TargetAnsweringIdentitySHA256,
		Subject:                       envelope.Subject, ContextPacketSHA256: envelope.ContextPacketSHA256,
		RequiredSenses: envelope.RequiredSenses, SensoryEvidence: envelope.SensoryEvidence,
		SkillContinuity: envelope.SkillContinuity, DiscoveryStance: envelope.DiscoveryStance,
	}
	requestDigest, err := digestJSON(request, "situated context request")
	if err != nil {
		return err
	}
	if requestDigest != envelope.RequestSHA256 {
		return errors.New("situated context request digest does not match its embedded inputs")
	}
	expectedState := SituatedContextReady
	for _, receipt := range envelope.SensoryEvidence {
		if receipt.State != SensoryStateReady {
			expectedState = SituatedContextSensoryHold
			break
		}
	}
	if expectedState == SituatedContextReady && envelope.SkillContinuity.State != SkillContinuityReady {
		expectedState = SituatedContextSkillHold
	}
	if expectedState == SituatedContextReady && envelope.DiscoveryStance.State != DiscoveryStateReady {
		expectedState = SituatedContextDiscoveryHold
	}
	// Join holds require the original context for replay, so a structurally
	// valid envelope may preserve that stronger state even when its embedded
	// component receipts are otherwise ready.
	if envelope.State != expectedState && envelope.State != SituatedContextJoinHold {
		return fmt.Errorf("situated context state is %q; embedded receipts require %q", envelope.State, expectedState)
	}
	if envelope.State == SituatedContextReady {
		if len(envelope.Holds) != 0 {
			return errors.New("ready situated context carries holds")
		}
	} else if len(envelope.Holds) == 0 {
		return errors.New("held situated context requires a reason")
	}
	if len(envelope.Notices) == 0 || !envelope.Authority.closed() {
		return errors.New("situated context must state its boundaries and carry closed authority")
	}
	packetBytes, err := situatedContextPacketBytes(envelope)
	if err != nil {
		return err
	}
	if packetBytes != envelope.PacketBytes || packetBytes > MaxSituatedContextBytes {
		return errors.New("situated context packet byte count is wrong or exceeds its bound")
	}
	expected, err := situatedContextEnvelopeDigest(envelope)
	if err != nil {
		return err
	}
	if expected != envelope.EnvelopeSHA256 {
		return fmt.Errorf("situated context digest mismatch: expected %s, got %s", envelope.EnvelopeSHA256, expected)
	}
	return nil
}

func situatedContextPacketBytes(envelope SituatedContextEnvelope) (int, error) {
	copy := envelope
	// A real digest and the placeholder have the same fixed 64-byte JSON
	// length. Iterate the decimal PacketBytes field to its stable encoded size.
	copy.EnvelopeSHA256 = strings.Repeat("0", 64)
	size := copy.PacketBytes
	for attempts := 0; attempts < 8; attempts++ {
		copy.PacketBytes = size
		data, err := json.Marshal(copy)
		if err != nil {
			return 0, fmt.Errorf("encode situated context for byte count: %w", err)
		}
		next := len(data)
		if next == size {
			return next, nil
		}
		size = next
	}
	return 0, errors.New("situated context packet byte count did not converge")
}

func situatedContextEnvelopeDigest(envelope SituatedContextEnvelope) (string, error) {
	envelope.EnvelopeSHA256 = ""
	return digestJSON(envelope, "situated context envelope")
}

// SealSituatedGated extends the existing deterministic clone boundary without
// modifying its schemas. Held situated evidence may be preserved only with a
// non-passing behavior outcome.
func SealSituatedGated(
	draft BehaviorEvidenceDraft,
	anchor OriginAnchor,
	witness TrainingRunWitness,
	profile TrainingProfileContract,
	context ProvenanceContextPacket,
	claims SourceClaimAssessment,
	protocol EvaluationProtocolSeal,
	situated SituatedContextEnvelope,
) (SealedBehaviorEvidence, error) {
	if err := situated.Validate(); err != nil {
		return SealedBehaviorEvidence{}, fmt.Errorf("situated context envelope: %w", err)
	}
	if situated.ContextPacketSHA256 != context.PacketSHA256 || situated.TargetAnsweringIdentitySHA256 != anchor.AnsweringIdentitySHA256 {
		return SealedBehaviorEvidence{}, errors.New("situated context does not bind the supplied provenance context and answering identity")
	}
	if situated.State != SituatedContextReady && draft.OutcomeState == "pass" {
		return SealedBehaviorEvidence{}, errors.New("held situated context cannot be sealed as a passing behavior outcome")
	}
	sensoryState := "pass"
	for _, receipt := range situated.SensoryEvidence {
		if receipt.State != SensoryStateReady {
			sensoryState = "hold"
			break
		}
	}
	skillState := "pass"
	if situated.SkillContinuity.State != SkillContinuityReady {
		skillState = "hold"
	}
	discoveryState := "pass"
	if situated.DiscoveryStance.State != DiscoveryStateReady {
		discoveryState = "hold"
	}
	envelopeState := "pass"
	if situated.State != SituatedContextReady {
		envelopeState = "hold"
	}
	var err error
	for _, binding := range []Verification{
		{Name: VerificationSensoryEvidence, State: sensoryState, EvidenceSHA256: situated.SensorySetSHA256},
		{Name: VerificationSkillContinuity, State: skillState, EvidenceSHA256: situated.SkillContinuity.ReceiptSHA256},
		{Name: VerificationDiscoveryStance, State: discoveryState, EvidenceSHA256: situated.DiscoveryStance.ReceiptSHA256},
		{Name: VerificationSituatedContext, State: envelopeState, EvidenceSHA256: situated.EnvelopeSHA256},
	} {
		draft, err = bindBehaviorVerification(draft, binding)
		if err != nil {
			return SealedBehaviorEvidence{}, err
		}
	}
	return SealGated(draft, anchor, witness, profile, context, claims, protocol)
}
