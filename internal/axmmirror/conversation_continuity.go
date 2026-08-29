package axmmirror

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/openwaldo/waldo/internal/record"
)

const (
	ConversationWitnessRequestSchema = "axm.waldo-witness.conversation-witness-request/v0.1"
	ConversationWitnessReceiptSchema = "axm.waldo-witness.conversation-witness-receipt/v0.1"
	ContinuityCapsuleDraftSchema     = "axm.waldo-witness.continuity-capsule-draft/v0.1"
	ContinuityCapsuleSchema          = "axm.waldo-witness.continuity-capsule/v0.1"

	ConversationWitnessReady               = "CONVERSATION_WITNESS_READY"
	ConversationWitnessTemplateHold        = "HOLD_INTERACTION_TEMPLATE_MISMATCH"
	ConversationWitnessAssistantTargetHold = "HOLD_ASSISTANT_TARGET_MISSING"
	ConversationWitnessAuthorityRefused    = "REFUSED_AUTHORITY_GROWTH"

	ContinuityCapsuleReady            = "CONTINUITY_CAPSULE_READY"
	ContinuityCapsuleBindingHold      = "HOLD_CONTINUITY_BINDING_MISMATCH"
	ContinuityCapsuleConversationHold = "HOLD_CONVERSATION_WITNESS"
	ContinuityCapsuleAuthorityRefused = "REFUSED_AUTHORITY_GROWTH"

	InteractionTemplateUserAssistantV1 = "user-assistant-v1"
	InteractionTemplateChatMLV1        = "chatml-v1"

	ConversationObjectiveCausal    = "causal-language-modeling"
	ConversationObjectiveAssistant = "assistant-response-modeling"
)

// ConversationInteractionBinding records the exact model-facing conversation
// contract without rendering the conversation or copying model-visible text.
// WALDO owns the template semantics; this AXM layer only witnesses the declared
// relationship between model identity and the training transformation.
type ConversationInteractionBinding struct {
	ModelTemplate    string   `json:"model_template"`
	TrainingTemplate string   `json:"training_template"`
	Objective        string   `json:"objective"`
	SupervisedRoles  []string `json:"supervised_roles,omitempty"`
}

// ConversationWitnessRequest deliberately carries only identifiers, digests,
// the interaction declaration, and requested authority. The canonical WALDO
// conversation is supplied separately to WitnessConversation so a receipt can
// be retained without retaining the raw dialogue in the receipt itself.
type ConversationWitnessRequest struct {
	Schema                  string                         `json:"schema"`
	WitnessID               string                         `json:"witness_id"`
	AnsweringIdentitySHA256 string                         `json:"answering_identity_sha256"`
	SourceRecordSHA256      string                         `json:"source_record_sha256"`
	Interaction             ConversationInteractionBinding `json:"interaction"`
	RequestedAuthority      Authority                      `json:"requested_authority"`
}

type ConversationRoleCounts struct {
	System    int `json:"system"`
	User      int `json:"user"`
	Assistant int `json:"assistant"`
	Tool      int `json:"tool"`
}

// ConversationTurnWitness preserves order and role while replacing content
// and optional context with digests. TurnSHA256 binds position, role, and both
// content digests so turn reordering remains detectable without raw text.
type ConversationTurnWitness struct {
	Position      int    `json:"position"`
	Role          string `json:"role"`
	ContentSHA256 string `json:"content_sha256"`
	ContextSHA256 string `json:"context_sha256,omitempty"`
	TurnSHA256    string `json:"turn_sha256"`
}

// ConversationWitnessReceipt is a privacy-preserving witness of one canonical
// WALDO conversation and its declared model/training interaction contract. It
// contains no prompt, answer, tool definition, hidden reasoning, or memory
// payload.
type ConversationWitnessReceipt struct {
	Schema                  string                         `json:"schema"`
	State                   string                         `json:"state"`
	WitnessID               string                         `json:"witness_id"`
	AnsweringIdentitySHA256 string                         `json:"answering_identity_sha256"`
	ConversationSHA256      string                         `json:"conversation_sha256"`
	Interaction             ConversationInteractionBinding `json:"interaction"`
	MessageCount            int                            `json:"message_count"`
	RoleCounts              ConversationRoleCounts         `json:"role_counts"`
	Turns                   []ConversationTurnWitness      `json:"turns"`
	ToolsPresent            bool                           `json:"tools_present"`
	ToolsSHA256             string                         `json:"tools_sha256,omitempty"`
	LastTurnSHA256          string                         `json:"last_turn_sha256"`
	RequestSHA256           string                         `json:"request_sha256"`
	ReceiptSHA256           string                         `json:"receipt_sha256,omitempty"`
	Holds                   []string                       `json:"holds,omitempty"`
	Notices                 []string                       `json:"notices"`
	Authority               Authority                      `json:"authority"`
}

// WitnessConversation binds an already-normalized WALDO conversation to its
// model interaction declaration. It observes and hashes; it does not render a
// prompt, tokenize, train, execute a tool, alter memory, or activate a model.
func WitnessConversation(request ConversationWitnessRequest, conversation record.Conversation) (ConversationWitnessReceipt, error) {
	canonicalRequest, err := canonicalizeConversationWitnessRequest(request)
	if err != nil {
		return ConversationWitnessReceipt{}, err
	}
	if err := conversation.Validate(); err != nil {
		return ConversationWitnessReceipt{}, fmt.Errorf("validate WALDO conversation: %w", err)
	}
	encoded, err := record.EncodeConversation(conversation)
	if err != nil {
		return ConversationWitnessReceipt{}, fmt.Errorf("encode WALDO conversation: %w", err)
	}
	conversationDigest := record.TextHash(encoded)
	if conversationDigest != canonicalRequest.SourceRecordSHA256 {
		return ConversationWitnessReceipt{}, fmt.Errorf("source_record_sha256 does not match canonical WALDO conversation: expected %s, got %s", canonicalRequest.SourceRecordSHA256, conversationDigest)
	}
	requestDigest, err := digestJSON(canonicalRequest, "conversation witness request")
	if err != nil {
		return ConversationWitnessReceipt{}, err
	}

	receipt := ConversationWitnessReceipt{
		Schema: ConversationWitnessReceiptSchema, State: ConversationWitnessReady,
		WitnessID: canonicalRequest.WitnessID, AnsweringIdentitySHA256: canonicalRequest.AnsweringIdentitySHA256,
		ConversationSHA256: conversationDigest, Interaction: canonicalRequest.Interaction,
		MessageCount: len(conversation.Messages), Turns: make([]ConversationTurnWitness, 0, len(conversation.Messages)),
		ToolsPresent: len(conversation.Tools) > 0, RequestSHA256: requestDigest,
		Notices: []string{
			"turn order and roles are retained while message content, context, and tool definitions are represented only by digests in this receipt",
			"matching interaction templates prove declaration consistency only; they do not prove model behavior, training quality, tool correctness, or safe tool use",
			"this witness performs no prompt rendering, tokenization, training, inference, tool execution, memory write, promotion, or CANON action",
		},
		Authority: Authority{},
	}

	for position, message := range conversation.Messages {
		turn := ConversationTurnWitness{
			Position:      position + 1,
			Role:          message.Role,
			ContentSHA256: record.TextHash(message.Content),
		}
		if message.Context != "" {
			turn.ContextSHA256 = record.TextHash(message.Context)
		}
		turn.TurnSHA256, err = conversationTurnDigest(turn)
		if err != nil {
			return ConversationWitnessReceipt{}, err
		}
		receipt.Turns = append(receipt.Turns, turn)
		switch message.Role {
		case "system":
			receipt.RoleCounts.System++
		case "user":
			receipt.RoleCounts.User++
		case "assistant":
			receipt.RoleCounts.Assistant++
		case "tool":
			receipt.RoleCounts.Tool++
		}
	}
	receipt.LastTurnSHA256 = receipt.Turns[len(receipt.Turns)-1].TurnSHA256

	if receipt.ToolsPresent {
		toolDigest, err := canonicalToolDigest(conversation.Tools)
		if err != nil {
			return ConversationWitnessReceipt{}, err
		}
		receipt.ToolsSHA256 = toolDigest
	}

	switch {
	case !canonicalRequest.RequestedAuthority.closed():
		receipt.State = ConversationWitnessAuthorityRefused
		receipt.Holds = []string{"conversation witnessing requested authority beyond evidence-only observation; no authority was granted"}
	case canonicalRequest.Interaction.ModelTemplate != canonicalRequest.Interaction.TrainingTemplate:
		receipt.State = ConversationWitnessTemplateHold
		receipt.Holds = []string{"model interaction template and conversation training template differ; WALDO requires an explicit matching contract"}
	case canonicalRequest.Interaction.Objective == ConversationObjectiveAssistant && !containsString(canonicalRequest.Interaction.SupervisedRoles, "assistant"):
		receipt.State = ConversationWitnessAssistantTargetHold
		receipt.Holds = []string{"assistant-response-modeling does not declare assistant as a supervised role; AXM will not treat this as a Mirror answering-identity training witness"}
	}

	receipt.ReceiptSHA256, err = conversationWitnessReceiptDigest(receipt)
	if err != nil {
		return ConversationWitnessReceipt{}, err
	}
	if err := VerifyConversationWitness(receipt); err != nil {
		return ConversationWitnessReceipt{}, fmt.Errorf("verify generated conversation witness: %w", err)
	}
	return receipt, nil
}

func canonicalizeConversationWitnessRequest(request ConversationWitnessRequest) (ConversationWitnessRequest, error) {
	if request.Schema != ConversationWitnessRequestSchema {
		return ConversationWitnessRequest{}, fmt.Errorf("conversation witness request schema must be %q", ConversationWitnessRequestSchema)
	}
	if strings.TrimSpace(request.WitnessID) == "" || request.WitnessID != strings.TrimSpace(request.WitnessID) {
		return ConversationWitnessRequest{}, errors.New("conversation witness_id is required and must be trimmed")
	}
	if err := validateSHA256("conversation answering_identity_sha256", request.AnsweringIdentitySHA256); err != nil {
		return ConversationWitnessRequest{}, err
	}
	if err := validateSHA256("conversation source_record_sha256", request.SourceRecordSHA256); err != nil {
		return ConversationWitnessRequest{}, err
	}
	binding := request.Interaction
	if !supportedInteractionTemplate(binding.ModelTemplate) {
		return ConversationWitnessRequest{}, fmt.Errorf("unsupported model interaction template %q", binding.ModelTemplate)
	}
	if !supportedInteractionTemplate(binding.TrainingTemplate) {
		return ConversationWitnessRequest{}, fmt.Errorf("unsupported conversation training template %q", binding.TrainingTemplate)
	}
	if !oneOf(binding.Objective, ConversationObjectiveCausal, ConversationObjectiveAssistant) {
		return ConversationWitnessRequest{}, fmt.Errorf("unsupported conversation objective %q", binding.Objective)
	}
	binding.SupervisedRoles = append([]string(nil), binding.SupervisedRoles...)
	sort.Strings(binding.SupervisedRoles)
	for index, role := range binding.SupervisedRoles {
		if !oneOf(role, "system", "user", "assistant", "tool") {
			return ConversationWitnessRequest{}, fmt.Errorf("unsupported supervised role %q", role)
		}
		if index > 0 && binding.SupervisedRoles[index-1] == role {
			return ConversationWitnessRequest{}, fmt.Errorf("duplicate supervised role %q", role)
		}
	}
	if binding.Objective == ConversationObjectiveAssistant && len(binding.SupervisedRoles) == 0 {
		return ConversationWitnessRequest{}, errors.New("assistant-response-modeling requires at least one supervised role")
	}
	canonical := request
	canonical.Interaction = binding
	return canonical, nil
}

func supportedInteractionTemplate(value string) bool {
	return oneOf(value, InteractionTemplateUserAssistantV1, InteractionTemplateChatMLV1)
}

func canonicalToolDigest(raw json.RawMessage) (string, error) {
	var value any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return "", fmt.Errorf("decode canonical tool JSON: %w", err)
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal canonical tool JSON: %w", err)
	}
	return record.TextHash(string(payload)), nil
}

func conversationTurnDigest(turn ConversationTurnWitness) (string, error) {
	basis := turn
	basis.TurnSHA256 = ""
	return digestJSON(basis, "conversation turn witness")
}

func conversationWitnessReceiptDigest(receipt ConversationWitnessReceipt) (string, error) {
	basis := receipt
	basis.ReceiptSHA256 = ""
	return digestJSON(basis, "conversation witness receipt")
}

// VerifyConversationWitness checks receipt structure and self-digests. Because
// the raw dialogue is intentionally absent, this verifies integrity of the
// witness, not semantic truth of the original conversation.
func VerifyConversationWitness(receipt ConversationWitnessReceipt) error {
	if receipt.Schema != ConversationWitnessReceiptSchema {
		return fmt.Errorf("conversation witness receipt schema must be %q", ConversationWitnessReceiptSchema)
	}
	if !oneOf(receipt.State, ConversationWitnessReady, ConversationWitnessTemplateHold, ConversationWitnessAssistantTargetHold, ConversationWitnessAuthorityRefused) {
		return fmt.Errorf("unsupported conversation witness state %q", receipt.State)
	}
	if strings.TrimSpace(receipt.WitnessID) == "" || receipt.WitnessID != strings.TrimSpace(receipt.WitnessID) {
		return errors.New("conversation witness_id is required and must be trimmed")
	}
	for name, digest := range map[string]string{
		"answering_identity_sha256": receipt.AnsweringIdentitySHA256,
		"conversation_sha256":       receipt.ConversationSHA256,
		"last_turn_sha256":          receipt.LastTurnSHA256,
		"request_sha256":            receipt.RequestSHA256,
		"receipt_sha256":            receipt.ReceiptSHA256,
	} {
		if err := validateSHA256("conversation witness "+name, digest); err != nil {
			return err
		}
	}
	if receipt.ToolsPresent {
		if err := validateSHA256("conversation witness tools_sha256", receipt.ToolsSHA256); err != nil {
			return err
		}
	} else if receipt.ToolsSHA256 != "" {
		return errors.New("conversation witness tools_sha256 is set while tools_present is false")
	}
	if !supportedInteractionTemplate(receipt.Interaction.ModelTemplate) || !supportedInteractionTemplate(receipt.Interaction.TrainingTemplate) {
		return errors.New("conversation witness contains unsupported interaction template")
	}
	if !oneOf(receipt.Interaction.Objective, ConversationObjectiveCausal, ConversationObjectiveAssistant) {
		return fmt.Errorf("unsupported conversation witness objective %q", receipt.Interaction.Objective)
	}
	canonicalRoles := append([]string(nil), receipt.Interaction.SupervisedRoles...)
	sort.Strings(canonicalRoles)
	for index, role := range canonicalRoles {
		if !oneOf(role, "system", "user", "assistant", "tool") {
			return fmt.Errorf("conversation witness has unsupported supervised role %q", role)
		}
		if index > 0 && canonicalRoles[index-1] == role {
			return fmt.Errorf("conversation witness has duplicate supervised role %q", role)
		}
	}
	if !stringSlicesEqual(canonicalRoles, receipt.Interaction.SupervisedRoles) {
		return errors.New("conversation witness supervised_roles are not canonical")
	}
	if receipt.MessageCount <= 0 || len(receipt.Turns) != receipt.MessageCount {
		return errors.New("conversation witness message_count does not match turns")
	}
	counts := ConversationRoleCounts{}
	for index, turn := range receipt.Turns {
		if turn.Position != index+1 {
			return fmt.Errorf("conversation witness turn %d has non-contiguous position %d", index, turn.Position)
		}
		if !oneOf(turn.Role, "system", "user", "assistant", "tool") {
			return fmt.Errorf("conversation witness turn %d has unsupported role %q", index, turn.Role)
		}
		if err := validateSHA256(fmt.Sprintf("conversation witness turn %d content_sha256", index), turn.ContentSHA256); err != nil {
			return err
		}
		if turn.ContextSHA256 != "" {
			if err := validateSHA256(fmt.Sprintf("conversation witness turn %d context_sha256", index), turn.ContextSHA256); err != nil {
				return err
			}
		}
		if err := validateSHA256(fmt.Sprintf("conversation witness turn %d turn_sha256", index), turn.TurnSHA256); err != nil {
			return err
		}
		expected, err := conversationTurnDigest(turn)
		if err != nil {
			return err
		}
		if expected != turn.TurnSHA256 {
			return fmt.Errorf("conversation witness turn %d digest mismatch", index)
		}
		switch turn.Role {
		case "system":
			counts.System++
		case "user":
			counts.User++
		case "assistant":
			counts.Assistant++
		case "tool":
			counts.Tool++
		}
	}
	if counts != receipt.RoleCounts {
		return errors.New("conversation witness role_counts do not match turns")
	}
	if receipt.LastTurnSHA256 != receipt.Turns[len(receipt.Turns)-1].TurnSHA256 {
		return errors.New("conversation witness last_turn_sha256 does not match final turn")
	}
	if !receipt.Authority.closed() {
		return errors.New("conversation witness must carry closed authority")
	}
	expected, err := conversationWitnessReceiptDigest(receipt)
	if err != nil {
		return err
	}
	if expected != receipt.ReceiptSHA256 {
		return errors.New("conversation witness receipt digest mismatch")
	}
	return nil
}

// ContinuityCapsuleDraft names only already-sealed state. The capsule does not
// contain the underlying task, memory, dissent, skill, verifier, or dialogue
// payloads; those remain separately governed artifacts.
type ContinuityCapsuleDraft struct {
	Schema                       string    `json:"schema"`
	CapsuleID                    string    `json:"capsule_id"`
	CapturedAt                   string    `json:"captured_at"`
	ExpiresAt                    string    `json:"expires_at"`
	AnsweringIdentitySHA256      string    `json:"answering_identity_sha256"`
	ConversationWitnessSHA256    string    `json:"conversation_witness_sha256"`
	TaskStateSHA256              string    `json:"task_state_sha256"`
	ActiveVerifierRegistrySHA256 string    `json:"active_verifier_registry_sha256"`
	SkillContinuityReceiptSHA256 string    `json:"skill_continuity_receipt_sha256,omitempty"`
	MemoryShardSHA256            []string  `json:"memory_shard_sha256,omitempty"`
	OpenDissentSHA256            []string  `json:"open_dissent_sha256,omitempty"`
	EvidenceSHA256               []string  `json:"evidence_sha256,omitempty"`
	RequestedAuthority           Authority `json:"requested_authority"`
}

// ContinuityCapsule is a portable, digest-only resumption spine. It carries
// enough binding information to ask "is this the same answering identity,
// conversation lineage, verifier generation, task state, memory catalog, and
// dissent set?" without importing the underlying private payloads.
type ContinuityCapsule struct {
	Schema                       string    `json:"schema"`
	State                        string    `json:"state"`
	CapsuleID                    string    `json:"capsule_id"`
	CapturedAt                   string    `json:"captured_at"`
	ExpiresAt                    string    `json:"expires_at"`
	AnsweringIdentitySHA256      string    `json:"answering_identity_sha256"`
	ConversationWitnessSHA256    string    `json:"conversation_witness_sha256"`
	ConversationWitnessState     string    `json:"conversation_witness_state"`
	ConversationSHA256           string    `json:"conversation_sha256"`
	InteractionTemplate          string    `json:"interaction_template"`
	LastTurnSHA256               string    `json:"last_turn_sha256"`
	TurnCount                    int       `json:"turn_count"`
	TaskStateSHA256              string    `json:"task_state_sha256"`
	ActiveVerifierRegistrySHA256 string    `json:"active_verifier_registry_sha256"`
	SkillContinuityReceiptSHA256 string    `json:"skill_continuity_receipt_sha256,omitempty"`
	MemoryShardSHA256            []string  `json:"memory_shard_sha256,omitempty"`
	OpenDissentSHA256            []string  `json:"open_dissent_sha256,omitempty"`
	EvidenceSHA256               []string  `json:"evidence_sha256,omitempty"`
	DraftSHA256                  string    `json:"draft_sha256"`
	CapsuleSHA256                string    `json:"capsule_sha256,omitempty"`
	Holds                        []string  `json:"holds,omitempty"`
	Notices                      []string  `json:"notices"`
	Authority                    Authority `json:"authority"`
}

// SealContinuityCapsule binds a continuity draft to one verified conversation
// witness. It does not fetch the referenced objects or make them available.
func SealContinuityCapsule(draft ContinuityCapsuleDraft, witness ConversationWitnessReceipt) (ContinuityCapsule, error) {
	canonical, err := canonicalizeContinuityDraft(draft)
	if err != nil {
		return ContinuityCapsule{}, err
	}
	if err := VerifyConversationWitness(witness); err != nil {
		return ContinuityCapsule{}, fmt.Errorf("verify conversation witness for continuity capsule: %w", err)
	}
	draftDigest, err := digestJSON(canonical, "continuity capsule draft")
	if err != nil {
		return ContinuityCapsule{}, err
	}

	capsule := ContinuityCapsule{
		Schema: ContinuityCapsuleSchema, State: ContinuityCapsuleReady,
		CapsuleID: canonical.CapsuleID, CapturedAt: canonical.CapturedAt, ExpiresAt: canonical.ExpiresAt,
		AnsweringIdentitySHA256:   canonical.AnsweringIdentitySHA256,
		ConversationWitnessSHA256: canonical.ConversationWitnessSHA256, ConversationWitnessState: witness.State,
		ConversationSHA256: witness.ConversationSHA256, InteractionTemplate: witness.Interaction.ModelTemplate,
		LastTurnSHA256: witness.LastTurnSHA256, TurnCount: witness.MessageCount,
		TaskStateSHA256: canonical.TaskStateSHA256, ActiveVerifierRegistrySHA256: canonical.ActiveVerifierRegistrySHA256,
		SkillContinuityReceiptSHA256: canonical.SkillContinuityReceiptSHA256,
		MemoryShardSHA256:            canonical.MemoryShardSHA256, OpenDissentSHA256: canonical.OpenDissentSHA256,
		EvidenceSHA256: canonical.EvidenceSHA256, DraftSHA256: draftDigest,
		Notices: []string{
			"the capsule stores references and conversation turn lineage only; raw dialogue, task payloads, memories, dissent text, evidence, verifier definitions, and skills remain outside the capsule",
			"a matching digest proves exact referenced bytes only when those bytes are separately available and verified; it does not prove freshness, compatibility, safety, or runtime availability",
			"resumption from this capsule requires separate retrieval, permission, compatibility checks, and host authorization; the capsule grants no restoration, execution, training, promotion, or CANON authority",
		},
		Authority: Authority{},
	}

	switch {
	case !canonical.RequestedAuthority.closed():
		capsule.State = ContinuityCapsuleAuthorityRefused
		capsule.Holds = []string{"continuity sealing requested authority beyond evidence-only packaging; no authority was granted"}
	case canonical.ConversationWitnessSHA256 != witness.ReceiptSHA256 || canonical.AnsweringIdentitySHA256 != witness.AnsweringIdentitySHA256:
		capsule.State = ContinuityCapsuleBindingHold
		capsule.Holds = []string{"continuity draft does not bind to the supplied conversation witness and answering identity"}
	case witness.State != ConversationWitnessReady:
		capsule.State = ContinuityCapsuleConversationHold
		capsule.Holds = []string{fmt.Sprintf("conversation witness state is %s, not %s", witness.State, ConversationWitnessReady)}
	}

	capsule.CapsuleSHA256, err = continuityCapsuleDigest(capsule)
	if err != nil {
		return ContinuityCapsule{}, err
	}
	if err := VerifyContinuityCapsule(capsule); err != nil {
		return ContinuityCapsule{}, fmt.Errorf("verify generated continuity capsule: %w", err)
	}
	return capsule, nil
}

func canonicalizeContinuityDraft(draft ContinuityCapsuleDraft) (ContinuityCapsuleDraft, error) {
	if draft.Schema != ContinuityCapsuleDraftSchema {
		return ContinuityCapsuleDraft{}, fmt.Errorf("continuity capsule draft schema must be %q", ContinuityCapsuleDraftSchema)
	}
	if strings.TrimSpace(draft.CapsuleID) == "" || draft.CapsuleID != strings.TrimSpace(draft.CapsuleID) {
		return ContinuityCapsuleDraft{}, errors.New("continuity capsule_id is required and must be trimmed")
	}
	captured, err := time.Parse(time.RFC3339Nano, draft.CapturedAt)
	if err != nil {
		return ContinuityCapsuleDraft{}, fmt.Errorf("parse continuity captured_at: %w", err)
	}
	expires, err := time.Parse(time.RFC3339Nano, draft.ExpiresAt)
	if err != nil {
		return ContinuityCapsuleDraft{}, fmt.Errorf("parse continuity expires_at: %w", err)
	}
	if !captured.Before(expires) {
		return ContinuityCapsuleDraft{}, errors.New("continuity expires_at must be after captured_at")
	}
	for name, digest := range map[string]string{
		"answering_identity_sha256":       draft.AnsweringIdentitySHA256,
		"conversation_witness_sha256":     draft.ConversationWitnessSHA256,
		"task_state_sha256":               draft.TaskStateSHA256,
		"active_verifier_registry_sha256": draft.ActiveVerifierRegistrySHA256,
	} {
		if err := validateSHA256("continuity "+name, digest); err != nil {
			return ContinuityCapsuleDraft{}, err
		}
	}
	if draft.SkillContinuityReceiptSHA256 != "" {
		if err := validateSHA256("continuity skill_continuity_receipt_sha256", draft.SkillContinuityReceiptSHA256); err != nil {
			return ContinuityCapsuleDraft{}, err
		}
	}
	canonical := draft
	canonical.MemoryShardSHA256, err = canonicalDigestSet("continuity memory_shard_sha256", draft.MemoryShardSHA256)
	if err != nil {
		return ContinuityCapsuleDraft{}, err
	}
	canonical.OpenDissentSHA256, err = canonicalDigestSet("continuity open_dissent_sha256", draft.OpenDissentSHA256)
	if err != nil {
		return ContinuityCapsuleDraft{}, err
	}
	canonical.EvidenceSHA256, err = canonicalDigestSet("continuity evidence_sha256", draft.EvidenceSHA256)
	if err != nil {
		return ContinuityCapsuleDraft{}, err
	}
	return canonical, nil
}

func canonicalDigestSet(name string, values []string) ([]string, error) {
	canonical := append([]string(nil), values...)
	sort.Strings(canonical)
	for index, digest := range canonical {
		if err := validateSHA256(fmt.Sprintf("%s[%d]", name, index), digest); err != nil {
			return nil, err
		}
		if index > 0 && canonical[index-1] == digest {
			return nil, fmt.Errorf("%s contains duplicate digest %s", name, digest)
		}
	}
	return canonical, nil
}

func continuityCapsuleDigest(capsule ContinuityCapsule) (string, error) {
	basis := capsule
	basis.CapsuleSHA256 = ""
	return digestJSON(basis, "continuity capsule")
}

// VerifyContinuityCapsule verifies the digest-only package itself. It cannot
// prove that referenced artifacts are present, current, authorized, or safe.
func VerifyContinuityCapsule(capsule ContinuityCapsule) error {
	if capsule.Schema != ContinuityCapsuleSchema {
		return fmt.Errorf("continuity capsule schema must be %q", ContinuityCapsuleSchema)
	}
	if !oneOf(capsule.State, ContinuityCapsuleReady, ContinuityCapsuleBindingHold, ContinuityCapsuleConversationHold, ContinuityCapsuleAuthorityRefused) {
		return fmt.Errorf("unsupported continuity capsule state %q", capsule.State)
	}
	if strings.TrimSpace(capsule.CapsuleID) == "" || capsule.CapsuleID != strings.TrimSpace(capsule.CapsuleID) {
		return errors.New("continuity capsule_id is required and must be trimmed")
	}
	captured, err := time.Parse(time.RFC3339Nano, capsule.CapturedAt)
	if err != nil {
		return fmt.Errorf("parse continuity capsule captured_at: %w", err)
	}
	expires, err := time.Parse(time.RFC3339Nano, capsule.ExpiresAt)
	if err != nil {
		return fmt.Errorf("parse continuity capsule expires_at: %w", err)
	}
	if !captured.Before(expires) {
		return errors.New("continuity capsule expires_at must be after captured_at")
	}
	for name, digest := range map[string]string{
		"answering_identity_sha256":       capsule.AnsweringIdentitySHA256,
		"conversation_witness_sha256":     capsule.ConversationWitnessSHA256,
		"conversation_sha256":             capsule.ConversationSHA256,
		"last_turn_sha256":                capsule.LastTurnSHA256,
		"task_state_sha256":               capsule.TaskStateSHA256,
		"active_verifier_registry_sha256": capsule.ActiveVerifierRegistrySHA256,
		"draft_sha256":                    capsule.DraftSHA256,
		"capsule_sha256":                  capsule.CapsuleSHA256,
	} {
		if err := validateSHA256("continuity capsule "+name, digest); err != nil {
			return err
		}
	}
	if capsule.SkillContinuityReceiptSHA256 != "" {
		if err := validateSHA256("continuity capsule skill_continuity_receipt_sha256", capsule.SkillContinuityReceiptSHA256); err != nil {
			return err
		}
	}
	if !supportedInteractionTemplate(capsule.InteractionTemplate) {
		return fmt.Errorf("unsupported continuity interaction_template %q", capsule.InteractionTemplate)
	}
	if capsule.TurnCount <= 0 {
		return errors.New("continuity capsule turn_count must be positive")
	}
	for name, values := range map[string][]string{
		"memory_shard_sha256": valuesOrEmpty(capsule.MemoryShardSHA256),
		"open_dissent_sha256": valuesOrEmpty(capsule.OpenDissentSHA256),
		"evidence_sha256":     valuesOrEmpty(capsule.EvidenceSHA256),
	} {
		canonical, err := canonicalDigestSet("continuity capsule "+name, values)
		if err != nil {
			return err
		}
		if !stringSlicesEqual(canonical, values) {
			return fmt.Errorf("continuity capsule %s is not canonical", name)
		}
	}
	if !capsule.Authority.closed() {
		return errors.New("continuity capsule must carry closed authority")
	}
	expected, err := continuityCapsuleDigest(capsule)
	if err != nil {
		return err
	}
	if expected != capsule.CapsuleSHA256 {
		return errors.New("continuity capsule digest mismatch")
	}
	return nil
}

func valuesOrEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// FreshAt evaluates only the capsule's declared freshness interval. It does not
// claim that referenced artifacts are still available or semantically current.
func (capsule ContinuityCapsule) FreshAt(at time.Time) (bool, error) {
	if err := VerifyContinuityCapsule(capsule); err != nil {
		return false, err
	}
	captured, _ := time.Parse(time.RFC3339Nano, capsule.CapturedAt)
	expires, _ := time.Parse(time.RFC3339Nano, capsule.ExpiresAt)
	return !at.Before(captured) && at.Before(expires), nil
}
