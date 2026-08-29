package axmmirror

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/openwaldo/waldo/internal/record"
)

func testConversation(t *testing.T) record.Conversation {
	t.Helper()
	return record.Conversation{
		Messages: []record.Message{
			{Role: "system", Content: "Keep the task bounded."},
			{Role: "user", Content: "Private synthetic request 7f2."},
			{Role: "assistant", Content: "I will inspect the declared tool first."},
			{Role: "tool", Content: `{"status":"ok","value":3}`},
			{Role: "assistant", Content: "The synthetic result is 3.", Context: "verified fixture context"},
		},
		Tools: json.RawMessage(`[{"name":"fixture_lookup","description":"synthetic test tool"}]`),
	}
}

func testConversationRequest(t *testing.T, conversation record.Conversation) ConversationWitnessRequest {
	t.Helper()
	encoded, err := record.EncodeConversation(conversation)
	if err != nil {
		t.Fatalf("encode conversation: %v", err)
	}
	return ConversationWitnessRequest{
		Schema:                  ConversationWitnessRequestSchema,
		WitnessID:               "conversation-fixture-001",
		AnsweringIdentitySHA256: strings.Repeat("a", 64),
		SourceRecordSHA256:      record.TextHash(encoded),
		Interaction: ConversationInteractionBinding{
			ModelTemplate:    InteractionTemplateChatMLV1,
			TrainingTemplate: InteractionTemplateChatMLV1,
			Objective:        ConversationObjectiveAssistant,
			SupervisedRoles:  []string{"assistant"},
		},
		RequestedAuthority: Authority{},
	}
}

func TestWitnessConversationStructuredToolUse(t *testing.T) {
	conversation := testConversation(t)
	request := testConversationRequest(t, conversation)

	receipt, err := WitnessConversation(request, conversation)
	if err != nil {
		t.Fatalf("WitnessConversation: %v", err)
	}
	if receipt.State != ConversationWitnessReady {
		t.Fatalf("state = %s, want %s", receipt.State, ConversationWitnessReady)
	}
	if receipt.MessageCount != 5 || len(receipt.Turns) != 5 {
		t.Fatalf("unexpected turn count: message_count=%d turns=%d", receipt.MessageCount, len(receipt.Turns))
	}
	if receipt.RoleCounts.System != 1 || receipt.RoleCounts.User != 1 || receipt.RoleCounts.Assistant != 2 || receipt.RoleCounts.Tool != 1 {
		t.Fatalf("unexpected role counts: %+v", receipt.RoleCounts)
	}
	if !receipt.ToolsPresent || receipt.ToolsSHA256 == "" {
		t.Fatal("expected tool definitions to be witnessed by digest")
	}
	if receipt.LastTurnSHA256 != receipt.Turns[4].TurnSHA256 {
		t.Fatal("last turn binding mismatch")
	}
	if err := VerifyConversationWitness(receipt); err != nil {
		t.Fatalf("VerifyConversationWitness: %v", err)
	}

	payload, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("marshal receipt: %v", err)
	}
	for _, forbidden := range []string{"Private synthetic request 7f2.", "fixture_lookup", "synthetic result is 3"} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("receipt leaked raw conversation/tool content %q", forbidden)
		}
	}
}

func TestWitnessConversationTemplateMismatchHolds(t *testing.T) {
	conversation := testConversation(t)
	request := testConversationRequest(t, conversation)
	request.Interaction.TrainingTemplate = InteractionTemplateUserAssistantV1

	receipt, err := WitnessConversation(request, conversation)
	if err != nil {
		t.Fatalf("WitnessConversation: %v", err)
	}
	if receipt.State != ConversationWitnessTemplateHold {
		t.Fatalf("state = %s, want %s", receipt.State, ConversationWitnessTemplateHold)
	}
	if len(receipt.Holds) == 0 {
		t.Fatal("template mismatch should carry a hold explanation")
	}
	if err := VerifyConversationWitness(receipt); err != nil {
		t.Fatalf("VerifyConversationWitness held receipt: %v", err)
	}
}

func TestWitnessConversationAssistantTargetHold(t *testing.T) {
	conversation := testConversation(t)
	request := testConversationRequest(t, conversation)
	request.Interaction.SupervisedRoles = []string{"tool"}

	receipt, err := WitnessConversation(request, conversation)
	if err != nil {
		t.Fatalf("WitnessConversation: %v", err)
	}
	if receipt.State != ConversationWitnessAssistantTargetHold {
		t.Fatalf("state = %s, want %s", receipt.State, ConversationWitnessAssistantTargetHold)
	}
}

func TestWitnessConversationRefusesAuthorityGrowth(t *testing.T) {
	conversation := testConversation(t)
	request := testConversationRequest(t, conversation)
	request.RequestedAuthority = Authority{Training: true}

	receipt, err := WitnessConversation(request, conversation)
	if err != nil {
		t.Fatalf("WitnessConversation: %v", err)
	}
	if receipt.State != ConversationWitnessAuthorityRefused {
		t.Fatalf("state = %s, want %s", receipt.State, ConversationWitnessAuthorityRefused)
	}
	if !receipt.Authority.closed() {
		t.Fatal("receipt granted authority")
	}
}

func testContinuityDraft(witness ConversationWitnessReceipt) ContinuityCapsuleDraft {
	return ContinuityCapsuleDraft{
		Schema:                       ContinuityCapsuleDraftSchema,
		CapsuleID:                    "continuity-fixture-001",
		CapturedAt:                   "2026-08-23T05:00:00Z",
		ExpiresAt:                    "2026-08-30T05:00:00Z",
		AnsweringIdentitySHA256:      witness.AnsweringIdentitySHA256,
		ConversationWitnessSHA256:    witness.ReceiptSHA256,
		TaskStateSHA256:              strings.Repeat("b", 64),
		ActiveVerifierRegistrySHA256: strings.Repeat("c", 64),
		SkillContinuityReceiptSHA256: strings.Repeat("d", 64),
		MemoryShardSHA256:            []string{strings.Repeat("f", 64), strings.Repeat("e", 64)},
		OpenDissentSHA256:            []string{strings.Repeat("1", 64)},
		EvidenceSHA256:               []string{strings.Repeat("3", 64), strings.Repeat("2", 64)},
		RequestedAuthority:           Authority{},
	}
}

func TestContinuityCapsuleSealsDigestOnly(t *testing.T) {
	conversation := testConversation(t)
	request := testConversationRequest(t, conversation)
	witness, err := WitnessConversation(request, conversation)
	if err != nil {
		t.Fatalf("WitnessConversation: %v", err)
	}

	capsule, err := SealContinuityCapsule(testContinuityDraft(witness), witness)
	if err != nil {
		t.Fatalf("SealContinuityCapsule: %v", err)
	}
	if capsule.State != ContinuityCapsuleReady {
		t.Fatalf("state = %s, want %s", capsule.State, ContinuityCapsuleReady)
	}
	if capsule.InteractionTemplate != InteractionTemplateChatMLV1 || capsule.TurnCount != 5 {
		t.Fatalf("unexpected conversation binding: template=%s turns=%d", capsule.InteractionTemplate, capsule.TurnCount)
	}
	if len(capsule.MemoryShardSHA256) != 2 || capsule.MemoryShardSHA256[0] != strings.Repeat("e", 64) {
		t.Fatalf("memory catalog was not canonicalized: %v", capsule.MemoryShardSHA256)
	}
	if err := VerifyContinuityCapsule(capsule); err != nil {
		t.Fatalf("VerifyContinuityCapsule: %v", err)
	}

	payload, err := json.Marshal(capsule)
	if err != nil {
		t.Fatalf("marshal capsule: %v", err)
	}
	for _, forbidden := range []string{"Private synthetic request 7f2.", "fixture_lookup", "Keep the task bounded."} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("capsule leaked raw payload %q", forbidden)
		}
	}

	fresh, err := capsule.FreshAt(time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("FreshAt: %v", err)
	}
	if !fresh {
		t.Fatal("capsule should be fresh inside its declared interval")
	}
	fresh, err = capsule.FreshAt(time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("FreshAt stale: %v", err)
	}
	if fresh {
		t.Fatal("capsule should be stale after expires_at")
	}
}

func TestContinuityCapsuleBindingMismatchHolds(t *testing.T) {
	conversation := testConversation(t)
	request := testConversationRequest(t, conversation)
	witness, err := WitnessConversation(request, conversation)
	if err != nil {
		t.Fatalf("WitnessConversation: %v", err)
	}
	draft := testContinuityDraft(witness)
	draft.ConversationWitnessSHA256 = strings.Repeat("9", 64)

	capsule, err := SealContinuityCapsule(draft, witness)
	if err != nil {
		t.Fatalf("SealContinuityCapsule: %v", err)
	}
	if capsule.State != ContinuityCapsuleBindingHold {
		t.Fatalf("state = %s, want %s", capsule.State, ContinuityCapsuleBindingHold)
	}
	if len(capsule.Holds) == 0 {
		t.Fatal("binding mismatch should carry a hold explanation")
	}
}

func TestContinuityCapsulePropagatesConversationHold(t *testing.T) {
	conversation := testConversation(t)
	request := testConversationRequest(t, conversation)
	request.Interaction.TrainingTemplate = InteractionTemplateUserAssistantV1
	witness, err := WitnessConversation(request, conversation)
	if err != nil {
		t.Fatalf("WitnessConversation: %v", err)
	}
	draft := testContinuityDraft(witness)

	capsule, err := SealContinuityCapsule(draft, witness)
	if err != nil {
		t.Fatalf("SealContinuityCapsule: %v", err)
	}
	if capsule.State != ContinuityCapsuleConversationHold {
		t.Fatalf("state = %s, want %s", capsule.State, ContinuityCapsuleConversationHold)
	}
}

func TestContinuityCapsuleRefusesDuplicateDigestSet(t *testing.T) {
	conversation := testConversation(t)
	request := testConversationRequest(t, conversation)
	witness, err := WitnessConversation(request, conversation)
	if err != nil {
		t.Fatalf("WitnessConversation: %v", err)
	}
	draft := testContinuityDraft(witness)
	draft.MemoryShardSHA256 = []string{strings.Repeat("e", 64), strings.Repeat("e", 64)}

	if _, err := SealContinuityCapsule(draft, witness); err == nil {
		t.Fatal("duplicate memory digest should be rejected")
	}
}
