package axmmirror

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const (
	MirrorNeuralEscalationRequestSchema = "axm.waldo.mirror-neural-escalation-request/v0.37"
	MirrorNeuralEscalationReceiptSchema = "axm.waldo.mirror-neural-escalation-receipt/v0.38"
	MirrorChatLearningRecordSchema      = "axm.waldo.mirror-chat-learning-candidate/v0.37"

	MirrorEscalationNone      = "NONE"
	MirrorEscalationAmbiguity = "AMBIGUITY"
	MirrorEscalationNovelty   = "NOVELTY"
	MirrorEscalationConflict  = "CONFLICT"

	MirrorStatusDeterministicResolved = "DETERMINISTIC_RESOLVED"
	MirrorStatusNeuralOptInRequired   = "HOLD_NEURAL_OPT_IN_REQUIRED"
	MirrorStatusNeuralCandidate       = "NEURAL_CANDIDATE"
	MirrorStatusNeuralFailed          = "HOLD_NEURAL_ESCALATION_FAILED"

	MirrorLearningReviewRequired   = "REVIEW_REQUIRED"
	MirrorLearningApproved         = "APPROVED_FOR_TRAINING"
	MirrorLearningConsentExplicit  = "EXPLICIT_COMMAND_OPT_IN"
	MirrorLearningCaptureCandidate = "CAPTURE_CANDIDATE"
	MirrorLearningApproveTraining  = "APPROVE_FOR_TRAINING"

	MaxMirrorEscalationTextBytes = 64 * 1024
)

// MirrorNeuralEscalationRequest keeps deterministic Mirror state primary. A
// local model is eligible only when state is unresolved and the caller also
// supplies a separate runtime opt-in.
type MirrorNeuralEscalationRequest struct {
	Schema                string                 `json:"schema"`
	Prompt                string                 `json:"prompt"`
	DeterministicResponse string                 `json:"deterministicResponse,omitempty"`
	EscalationReason      string                 `json:"escalationReason"`
	Grounding             LiveNeuralGrounding    `json:"grounding"`
	Identity              *MirrorIdentityCapsule `json:"identity,omitempty"`
}

// MirrorIdentityCapsule makes identity input visible and hash-bound. It is
// prompt context only; using it never mutates the identity source.
type MirrorIdentityCapsule struct {
	ID           string   `json:"id"`
	Revision     string   `json:"revision"`
	RootSHA256   string   `json:"rootSha256"`
	Instructions []string `json:"instructions"`
}

func NewMirrorIdentityCapsule(id, revision string, instructions []string) (MirrorIdentityCapsule, error) {
	identity := MirrorIdentityCapsule{ID: id, Revision: revision, Instructions: append([]string(nil), instructions...)}
	digest, err := identity.digest()
	if err != nil {
		return MirrorIdentityCapsule{}, err
	}
	identity.RootSHA256 = digest
	if err := identity.Validate(); err != nil {
		return MirrorIdentityCapsule{}, err
	}
	return identity, nil
}

type MirrorNeuralCandidate struct {
	Text         string `json:"text"`
	Model        string `json:"model"`
	Backend      string `json:"backend"`
	SourceType   string `json:"sourceType,omitempty"`
	SourceID     string `json:"sourceId,omitempty"`
	RunID        string `json:"runId,omitempty"`
	Tokens       int    `json:"tokens,omitempty"`
	FinishReason string `json:"finishReason,omitempty"`
	DurationMS   int64  `json:"durationMs,omitempty"`
}

type MirrorNeuralEscalator interface {
	Escalate(context.Context, string) (MirrorNeuralCandidate, error)
}

type MirrorNeuralEscalationReceipt struct {
	Schema                    string                 `json:"schema"`
	Status                    string                 `json:"status"`
	DeterministicPrimary      bool                   `json:"deterministicPrimary"`
	NeuralOptIn               bool                   `json:"neuralOptIn"`
	NeuralCalled              bool                   `json:"neuralCalled"`
	EscalationReason          string                 `json:"escalationReason"`
	Consequence               string                 `json:"consequence"`
	RequestSHA256             string                 `json:"requestSha256"`
	PromptSHA256              string                 `json:"promptSha256"`
	NeuralPromptSHA256        string                 `json:"neuralPromptSha256,omitempty"`
	GroundingSHA256           string                 `json:"groundingSha256"`
	IdentitySHA256            string                 `json:"identitySha256,omitempty"`
	DeterministicResponse     string                 `json:"deterministicResponse,omitempty"`
	NeuralCandidate           *MirrorNeuralCandidate `json:"neuralCandidate,omitempty"`
	NeuralCandidateSHA256     string                 `json:"neuralCandidateSha256,omitempty"`
	NeuralErrorSHA256         string                 `json:"neuralErrorSha256,omitempty"`
	LearningCandidateRecorded bool                   `json:"learningCandidateRecorded"`
	LearningRecordSHA256      string                 `json:"learningRecordSha256,omitempty"`
	LearningLedgerMutation    bool                   `json:"learningLedgerMutation"`
	ExperienceContextApplied  bool                   `json:"experienceContextApplied"`
	ExperienceContextSHA256   string                 `json:"experienceContextSha256,omitempty"`
	ExperienceEpisodeIDs      []string               `json:"experienceEpisodeIds,omitempty"`
	ExperienceEpisodeID       string                 `json:"experienceEpisodeId,omitempty"`
	ExperienceEventsRecorded  int                    `json:"experienceEventsRecorded"`
	ExperienceLedgerMutation  bool                   `json:"experienceLedgerMutation"`
	ModelMemoryMutation       bool                   `json:"modelMemoryMutation"`
	TrainingMutation          bool                   `json:"trainingMutation"`
	IdentityMutation          bool                   `json:"identityMutation"`
	Authority                 Authority              `json:"authority"`
	Timestamp                 time.Time              `json:"timestamp"`
}

type MirrorChatLearningRecord struct {
	Schema               string    `json:"schema"`
	ReviewState          string    `json:"reviewState"`
	Consent              string    `json:"consent"`
	TrainingReady        bool      `json:"trainingReady"`
	IdentityMutation     bool      `json:"identityMutation"`
	SourceMode           string    `json:"sourceMode"`
	RequestSHA256        string    `json:"requestSha256"`
	Prompt               string    `json:"prompt"`
	Response             string    `json:"response"`
	TrainingText         string    `json:"text,omitempty"`
	ResponseSHA256       string    `json:"responseSha256"`
	GroundingSHA256      string    `json:"groundingSha256"`
	IdentitySHA256       string    `json:"identitySha256,omitempty"`
	Model                string    `json:"model,omitempty"`
	Backend              string    `json:"backend,omitempty"`
	NeuralCandidate      bool      `json:"neuralCandidate"`
	Authority            Authority `json:"authority"`
	CapturedAt           time.Time `json:"capturedAt"`
	LearningRecordSHA256 string    `json:"learningRecordSha256,omitempty"`
}

func LoadMirrorNeuralEscalationRequest(reader io.Reader) (MirrorNeuralEscalationRequest, error) {
	var request MirrorNeuralEscalationRequest
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, fmt.Errorf("decode Mirror neural escalation request: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("trailing JSON")
		}
		return MirrorNeuralEscalationRequest{}, fmt.Errorf("decode Mirror neural escalation request: %w", err)
	}
	if err := request.Validate(); err != nil {
		return MirrorNeuralEscalationRequest{}, err
	}
	return request, nil
}

func (request MirrorNeuralEscalationRequest) Validate() error {
	if request.Schema != MirrorNeuralEscalationRequestSchema {
		return fmt.Errorf("schema must be %q", MirrorNeuralEscalationRequestSchema)
	}
	if prompt := strings.TrimSpace(request.Prompt); prompt == "" || len(request.Prompt) > MaxMirrorEscalationTextBytes {
		return fmt.Errorf("prompt is required and limited to %d bytes", MaxMirrorEscalationTextBytes)
	}
	if len(request.DeterministicResponse) > MaxMirrorEscalationTextBytes {
		return fmt.Errorf("deterministicResponse is limited to %d bytes", MaxMirrorEscalationTextBytes)
	}
	if err := request.Grounding.Validate(); err != nil {
		return fmt.Errorf("grounding: %w", err)
	}
	switch request.Grounding.State {
	case GroundingStable:
		if strings.TrimSpace(request.DeterministicResponse) == "" {
			return errors.New("STABLE grounding requires deterministicResponse")
		}
		if request.EscalationReason != MirrorEscalationNone {
			return errors.New("STABLE grounding requires escalationReason NONE")
		}
	case GroundingUncertain:
		if request.DeterministicResponse != "" {
			return errors.New("UNCERTAIN grounding cannot claim a deterministicResponse")
		}
		if request.EscalationReason != MirrorEscalationAmbiguity && request.EscalationReason != MirrorEscalationNovelty {
			return errors.New("UNCERTAIN grounding requires escalationReason AMBIGUITY or NOVELTY")
		}
	case GroundingConflict:
		if request.DeterministicResponse != "" {
			return errors.New("CONFLICT grounding cannot claim a deterministicResponse")
		}
		if request.EscalationReason != MirrorEscalationConflict {
			return errors.New("CONFLICT grounding requires escalationReason CONFLICT")
		}
	}
	if request.Identity != nil {
		if err := request.Identity.Validate(); err != nil {
			return fmt.Errorf("identity: %w", err)
		}
	}
	return nil
}

func (identity MirrorIdentityCapsule) Validate() error {
	if strings.TrimSpace(identity.ID) == "" || len(identity.ID) > 128 {
		return errors.New("id is required and limited to 128 bytes")
	}
	if strings.TrimSpace(identity.Revision) == "" || len(identity.Revision) > 128 {
		return errors.New("revision is required and limited to 128 bytes")
	}
	if len(identity.Instructions) == 0 || len(identity.Instructions) > 64 {
		return errors.New("instructions must contain 1..64 visible roots")
	}
	for i, instruction := range identity.Instructions {
		if strings.TrimSpace(instruction) == "" || len(instruction) > 2048 {
			return fmt.Errorf("instructions[%d] is required and limited to 2048 bytes", i)
		}
	}
	digest, err := identity.digest()
	if err != nil {
		return err
	}
	if identity.RootSHA256 != digest {
		return errors.New("rootSha256 does not bind the visible identity roots")
	}
	return nil
}

func (identity MirrorIdentityCapsule) digest() (string, error) {
	binding := struct {
		ID           string   `json:"id"`
		Revision     string   `json:"revision"`
		Instructions []string `json:"instructions"`
	}{identity.ID, identity.Revision, identity.Instructions}
	payload, err := json.Marshal(binding)
	if err != nil {
		return "", err
	}
	return digestMirrorText(string(payload)), nil
}

func (request MirrorNeuralEscalationRequest) NeedsNeuralEscalation() bool {
	return request.Grounding.State == GroundingUncertain || request.Grounding.State == GroundingConflict
}

func (request MirrorNeuralEscalationRequest) NeuralPrompt() (string, string, string, error) {
	return request.neuralPrompt(nil)
}

func (request MirrorNeuralEscalationRequest) neuralPrompt(experience *MirrorExperiencePromptContext) (string, string, string, error) {
	if err := request.Validate(); err != nil {
		return "", "", "", err
	}
	if experience != nil {
		if err := experience.Validate(); err != nil {
			return "", "", "", fmt.Errorf("experience context: %w", err)
		}
	}
	groundingCapsule, groundingDigest, err := request.Grounding.Capsule()
	if err != nil {
		return "", "", "", err
	}
	requestDigest, err := request.digest()
	if err != nil {
		return "", "", "", err
	}
	var buffer bytes.Buffer
	fmt.Fprintln(&buffer, "[AXM_MIRROR_NEURAL_ESCALATION v0.37]")
	fmt.Fprintln(&buffer, "role=non-authoritative neural reasoning candidate")
	fmt.Fprintln(&buffer, "deterministic_primary=true")
	fmt.Fprintf(&buffer, "escalation_reason=%s\n", request.EscalationReason)
	fmt.Fprintf(&buffer, "request_sha256=%s\n", requestDigest)
	fmt.Fprintln(&buffer, "rules:")
	fmt.Fprintln(&buffer, "- answer the task, but remain a candidate for review")
	fmt.Fprintln(&buffer, "- do not claim permission, execution, promotion, identity mutation, training, or CANON")
	fmt.Fprintln(&buffer, "- preserve uncertainty and disagreements visible in the supplied grounding")
	if request.Identity != nil {
		fmt.Fprintf(&buffer, "identity_id=%s\n", oneLine(request.Identity.ID))
		fmt.Fprintf(&buffer, "identity_revision=%s\n", oneLine(request.Identity.Revision))
		fmt.Fprintf(&buffer, "identity_sha256=%s\n", request.Identity.RootSHA256)
		fmt.Fprintln(&buffer, "visible_identity_roots:")
		for _, instruction := range request.Identity.Instructions {
			fmt.Fprintf(&buffer, "- %s\n", oneLine(instruction))
		}
	}
	buffer.WriteString(groundingCapsule)
	if experience != nil {
		buffer.WriteString(experience.Capsule)
	}
	fmt.Fprintln(&buffer, "task:")
	fmt.Fprintln(&buffer, request.Prompt)
	fmt.Fprintln(&buffer, "[/AXM_MIRROR_NEURAL_ESCALATION]")
	return buffer.String(), requestDigest, groundingDigest, nil
}

func RunMirrorNeuralEscalation(ctx context.Context, request MirrorNeuralEscalationRequest, neuralOptIn bool, escalator MirrorNeuralEscalator) (MirrorNeuralEscalationReceipt, error) {
	return RunMirrorNeuralEscalationWithExperience(ctx, request, neuralOptIn, escalator, nil)
}

func RunMirrorNeuralEscalationWithExperience(ctx context.Context, request MirrorNeuralEscalationRequest, neuralOptIn bool, escalator MirrorNeuralEscalator, experience *MirrorExperiencePromptContext) (MirrorNeuralEscalationReceipt, error) {
	prompt, requestDigest, groundingDigest, err := request.neuralPrompt(experience)
	if err != nil {
		return MirrorNeuralEscalationReceipt{}, err
	}
	receipt := MirrorNeuralEscalationReceipt{
		Schema:               MirrorNeuralEscalationReceiptSchema,
		DeterministicPrimary: true,
		NeuralOptIn:          neuralOptIn,
		EscalationReason:     request.EscalationReason,
		Consequence:          request.Grounding.Consequence,
		RequestSHA256:        requestDigest,
		PromptSHA256:         digestMirrorText(request.Prompt),
		GroundingSHA256:      groundingDigest,
		Authority:            Authority{},
		Timestamp:            time.Now().UTC(),
	}
	if request.Identity != nil {
		receipt.IdentitySHA256 = request.Identity.RootSHA256
	}
	if experience != nil && request.NeedsNeuralEscalation() {
		receipt.ExperienceContextApplied = true
		receipt.ExperienceContextSHA256 = experience.SHA256
		receipt.ExperienceEpisodeIDs = append([]string(nil), experience.EpisodeIDs...)
	}
	if !request.NeedsNeuralEscalation() {
		receipt.Status = MirrorStatusDeterministicResolved
		receipt.DeterministicResponse = request.DeterministicResponse
		return receipt, nil
	}
	receipt.NeuralPromptSHA256 = digestMirrorText(prompt)
	if !neuralOptIn {
		receipt.Status = MirrorStatusNeuralOptInRequired
		return receipt, nil
	}
	if escalator == nil {
		return receipt, errors.New("Mirror neural escalation was opted in but no local neural escalator is available")
	}
	receipt.NeuralCalled = true
	candidate, err := escalator.Escalate(ctx, prompt)
	if err != nil {
		receipt.Status = MirrorStatusNeuralFailed
		receipt.NeuralErrorSHA256 = digestMirrorText(err.Error())
		return receipt, fmt.Errorf("local WALDO neural escalation: %w", err)
	}
	if strings.TrimSpace(candidate.Text) == "" {
		receipt.Status = MirrorStatusNeuralFailed
		receipt.NeuralErrorSHA256 = digestMirrorText("empty neural candidate")
		return receipt, errors.New("local WALDO neural escalation returned an empty candidate")
	}
	receipt.Status = MirrorStatusNeuralCandidate
	receipt.NeuralCandidate = &candidate
	receipt.NeuralCandidateSHA256 = digestMirrorText(candidate.Text)
	return receipt, nil
}

func BuildMirrorChatLearningRecord(receipt MirrorNeuralEscalationReceipt, prompt, disposition string) (MirrorChatLearningRecord, error) {
	if receipt.Schema != MirrorNeuralEscalationReceiptSchema || !receipt.Authority.closed() {
		return MirrorChatLearningRecord{}, errors.New("learning capture requires a valid closed-authority escalation receipt")
	}
	if digestMirrorText(prompt) != receipt.PromptSHA256 {
		return MirrorChatLearningRecord{}, errors.New("learning prompt does not match the escalation receipt")
	}
	record := MirrorChatLearningRecord{
		Schema:           MirrorChatLearningRecordSchema,
		ReviewState:      MirrorLearningReviewRequired,
		Consent:          MirrorLearningConsentExplicit,
		TrainingReady:    false,
		IdentityMutation: false,
		RequestSHA256:    receipt.RequestSHA256,
		Prompt:           prompt,
		GroundingSHA256:  receipt.GroundingSHA256,
		IdentitySHA256:   receipt.IdentitySHA256,
		Authority:        Authority{},
		CapturedAt:       time.Now().UTC(),
	}
	switch disposition {
	case MirrorLearningCaptureCandidate:
	case MirrorLearningApproveTraining:
		record.ReviewState = MirrorLearningApproved
		record.TrainingReady = true
	default:
		return MirrorChatLearningRecord{}, fmt.Errorf("unsupported learning disposition %q", disposition)
	}
	switch receipt.Status {
	case MirrorStatusDeterministicResolved:
		record.SourceMode = "DETERMINISTIC"
		record.Response = receipt.DeterministicResponse
	case MirrorStatusNeuralCandidate:
		if receipt.NeuralCandidate == nil {
			return MirrorChatLearningRecord{}, errors.New("neural-candidate receipt is missing its candidate")
		}
		record.SourceMode = "NEURAL_CANDIDATE"
		record.Response = receipt.NeuralCandidate.Text
		record.Model = receipt.NeuralCandidate.Model
		record.Backend = receipt.NeuralCandidate.Backend
		record.NeuralCandidate = true
	default:
		return MirrorChatLearningRecord{}, fmt.Errorf("receipt status %s has no response to capture", receipt.Status)
	}
	record.ResponseSHA256 = digestMirrorText(record.Response)
	if record.TrainingReady {
		record.TrainingText = "User: " + record.Prompt + "\n\nAssistant: " + record.Response
	}
	digest, err := record.digest()
	if err != nil {
		return MirrorChatLearningRecord{}, err
	}
	record.LearningRecordSHA256 = digest
	return record, nil
}

func (record MirrorChatLearningRecord) JSONLine() ([]byte, error) {
	digest, err := record.digest()
	if err != nil {
		return nil, err
	}
	if record.LearningRecordSHA256 != digest {
		return nil, errors.New("learning record digest mismatch")
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}

func (request MirrorNeuralEscalationRequest) digest() (string, error) {
	normalized := request
	normalized.Grounding.References = append([]LiveGroundingReference(nil), request.Grounding.References...)
	sort.Slice(normalized.Grounding.References, func(i, j int) bool {
		return normalized.Grounding.References[i].ID < normalized.Grounding.References[j].ID
	})
	payload, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return digestMirrorText(string(payload)), nil
}

func (record MirrorChatLearningRecord) digest() (string, error) {
	copy := record
	copy.LearningRecordSHA256 = ""
	payload, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	return digestMirrorText(string(payload)), nil
}

func digestMirrorText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
