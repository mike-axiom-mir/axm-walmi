package axmmirror

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

const (
	WorkspaceLoopRequestSchema = "axm.waldo.workspace-loop-request/v0.2"
	WorkspaceLoopReceiptSchema = "axm.waldo.workspace-loop-receipt/v0.2"
	WorkspaceLoopMaxTurns      = 32
	WorkspaceVerifyMaxOutput   = 1024 * 1024
)

type WorkspaceNeuralModel interface {
	Generate(context.Context, string) (string, error)
}

type WorkspaceLoopRequest struct {
	Schema         string     `json:"schema"`
	Prompt         string     `json:"prompt"`
	MaxTurns       int        `json:"max_turns"`
	VerifyCommands [][]string `json:"verify_commands,omitempty"`
}

type WorkspaceToolCall struct {
	Tool         string                 `json:"tool,omitempty"`
	Path         string                 `json:"path,omitempty"`
	MaxBytes     int                    `json:"max_bytes,omitempty"`
	CommandIndex int                    `json:"command_index,omitempty"`
	WriteRequest *CandidateWriteRequest `json:"write_request,omitempty"`
	Final        string                 `json:"final,omitempty"`
}

type WorkspaceVerification struct {
	CommandIndex int      `json:"command_index"`
	Command      []string `json:"command"`
	ExitCode     int      `json:"exit_code"`
	Output       string   `json:"output"`
	Truncated    bool     `json:"truncated"`
	DurationMS   int64    `json:"duration_ms"`
	Passed       bool     `json:"passed"`
}

type WorkspaceLoopTurn struct {
	Index         int                    `json:"index"`
	ModelResponse string                 `json:"model_response"`
	Call          WorkspaceToolCall      `json:"call"`
	Observation   *WorkspaceObservation  `json:"observation,omitempty"`
	Verification  *WorkspaceVerification `json:"verification,omitempty"`
	Error         string                 `json:"error,omitempty"`
}

type WorkspaceLoopReceipt struct {
	Schema               string              `json:"schema"`
	State                string              `json:"state"`
	RootSHA256           string              `json:"root_sha256"`
	RequestSHA256        string              `json:"request_sha256"`
	Turns                []WorkspaceLoopTurn `json:"turns"`
	Final                string              `json:"final,omitempty"`
	WorkspaceMutation    bool                `json:"workspace_mutation"`
	VerificationRun      bool                `json:"verification_run"`
	VerificationPassed   bool                `json:"verification_passed"`
	NetworkUsedByHand    bool                `json:"network_used_by_hand"`
	InstallAuthority     bool                `json:"install_authority"`
	IntegrationAuthority bool                `json:"integration_authority"`
	PromotionAuthority   bool                `json:"promotion_authority"`
	LearningAuthority    bool                `json:"learning_authority"`
	CanonAuthority       bool                `json:"canon_authority"`
	Authority            string              `json:"authority"`
	ReceiptSHA256        string              `json:"receipt_sha256"`
}

func LoadWorkspaceLoopRequest(data []byte) (WorkspaceLoopRequest, error) {
	var request WorkspaceLoopRequest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return WorkspaceLoopRequest{}, fmt.Errorf("decode workspace loop request: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return WorkspaceLoopRequest{}, errors.New("workspace loop request contains more than one JSON value")
		}
		return WorkspaceLoopRequest{}, fmt.Errorf("decode trailing workspace loop request data: %w", err)
	}
	if err := request.Validate(); err != nil {
		return WorkspaceLoopRequest{}, err
	}
	return request, nil
}

func (request WorkspaceLoopRequest) Validate() error {
	if request.Schema != WorkspaceLoopRequestSchema {
		return fmt.Errorf("workspace loop request schema must be %q", WorkspaceLoopRequestSchema)
	}
	if strings.TrimSpace(request.Prompt) == "" || len(request.Prompt) > 64*1024 {
		return errors.New("workspace loop prompt must contain 1-65536 bytes")
	}
	if request.MaxTurns < 1 || request.MaxTurns > WorkspaceLoopMaxTurns {
		return fmt.Errorf("workspace loop max_turns must be in 1..%d", WorkspaceLoopMaxTurns)
	}
	if len(request.VerifyCommands) > 8 {
		return errors.New("workspace loop accepts at most 8 explicit verification commands")
	}
	for index, command := range request.VerifyCommands {
		if len(command) == 0 || len(command) > 32 {
			return fmt.Errorf("verify command %d must contain 1-32 arguments", index)
		}
		for _, argument := range command {
			if argument == "" || strings.ContainsRune(argument, 0) {
				return fmt.Errorf("verify command %d contains an invalid argument", index)
			}
		}
	}
	return nil
}

func RunWorkspaceLoop(ctx context.Context, hand *WorkspaceHand, model WorkspaceNeuralModel, request WorkspaceLoopRequest, allowWrite bool) (WorkspaceLoopReceipt, error) {
	if hand == nil || model == nil {
		return WorkspaceLoopReceipt{}, errors.New("workspace hand and neural model are required")
	}
	if err := request.Validate(); err != nil {
		return WorkspaceLoopReceipt{}, err
	}
	requestDigest, err := digestJSON(request, "workspace loop request")
	if err != nil {
		return WorkspaceLoopReceipt{}, err
	}
	rootDigest := candidateBytesSHA256([]byte(hand.Root()))
	receipt := WorkspaceLoopReceipt{
		Schema: WorkspaceLoopReceiptSchema, State: "RUNNING", RootSHA256: rootDigest, RequestSHA256: requestDigest,
		VerificationPassed: true, Authority: "NONE",
	}
	transcript := workspaceSystemPrompt(request, allowWrite)
	for turnIndex := 1; turnIndex <= request.MaxTurns; turnIndex++ {
		response, generateErr := model.Generate(ctx, transcript)
		turn := WorkspaceLoopTurn{Index: turnIndex, ModelResponse: response}
		if generateErr != nil {
			turn.Error = generateErr.Error()
			receipt.Turns = append(receipt.Turns, turn)
			receipt.State = "MODEL_ERROR"
			return sealWorkspaceLoopReceipt(receipt, generateErr)
		}
		call, parseErr := parseWorkspaceToolCall(response)
		turn.Call = call
		if parseErr != nil {
			turn.Error = parseErr.Error()
			receipt.Turns = append(receipt.Turns, turn)
			transcript = appendWorkspaceTurn(transcript, turn)
			continue
		}
		if call.Final != "" {
			receipt.Final = call.Final
			receipt.State = "COMPLETED"
			receipt.Turns = append(receipt.Turns, turn)
			return sealWorkspaceLoopReceipt(receipt, nil)
		}
		observation, verification, toolErr := executeWorkspaceTool(ctx, hand, request, call, allowWrite)
		if observation.Schema != "" {
			turn.Observation = &observation
			if observation.WorkspaceMutation {
				receipt.WorkspaceMutation = true
			}
		}
		if verification != nil {
			turn.Verification = verification
			receipt.VerificationRun = true
			receipt.VerificationPassed = receipt.VerificationPassed && verification.Passed
		}
		if toolErr != nil {
			turn.Error = toolErr.Error()
		}
		receipt.Turns = append(receipt.Turns, turn)
		transcript = appendWorkspaceTurn(transcript, turn)
	}
	receipt.State = "TURN_LIMIT"
	return sealWorkspaceLoopReceipt(receipt, errors.New("workspace loop reached its turn limit without a final response"))
}

func parseWorkspaceToolCall(response string) (WorkspaceToolCall, error) {
	var call WorkspaceToolCall
	decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(response)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&call); err != nil {
		return WorkspaceToolCall{}, fmt.Errorf("model must return one strict JSON tool call: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return WorkspaceToolCall{}, errors.New("model returned more than one JSON value")
	} else if !errors.Is(err, io.EOF) {
		return WorkspaceToolCall{}, fmt.Errorf("model returned invalid trailing data: %w", err)
	}
	if call.Final != "" {
		if call.Tool != "" || call.Path != "" || call.WriteRequest != nil {
			return WorkspaceToolCall{}, errors.New("final response must not include a tool call")
		}
		return call, nil
	}
	call.Tool = strings.ToUpper(strings.TrimSpace(call.Tool))
	switch call.Tool {
	case "LIST_FILES", "READ_FILE", "STAT_FILE", "HASH_FILE", "APPLY_WRITES", "RUN_VERIFY":
		return call, nil
	default:
		return WorkspaceToolCall{}, fmt.Errorf("unsupported workspace tool %q", call.Tool)
	}
}

func executeWorkspaceTool(ctx context.Context, hand *WorkspaceHand, request WorkspaceLoopRequest, call WorkspaceToolCall, allowWrite bool) (WorkspaceObservation, *WorkspaceVerification, error) {
	switch call.Tool {
	case "LIST_FILES":
		observation, err := hand.List(call.Path)
		return observation, nil, err
	case "READ_FILE":
		maxBytes := call.MaxBytes
		if maxBytes <= 0 || maxBytes > 64*1024 {
			maxBytes = 64 * 1024
		}
		observation, err := hand.Read(call.Path, maxBytes)
		return observation, nil, err
	case "STAT_FILE":
		observation, err := hand.Stat(call.Path, false)
		return observation, nil, err
	case "HASH_FILE":
		observation, err := hand.Stat(call.Path, true)
		return observation, nil, err
	case "APPLY_WRITES":
		if call.WriteRequest == nil {
			return WorkspaceObservation{}, nil, errors.New("APPLY_WRITES requires write_request")
		}
		observation, err := hand.Apply(*call.WriteRequest, allowWrite)
		return observation, nil, err
	case "RUN_VERIFY":
		verification, err := runWorkspaceVerification(ctx, hand.Root(), request.VerifyCommands, call.CommandIndex)
		return WorkspaceObservation{}, &verification, err
	default:
		return WorkspaceObservation{}, nil, fmt.Errorf("unsupported workspace tool %q", call.Tool)
	}
}

func runWorkspaceVerification(ctx context.Context, root string, allowed [][]string, index int) (WorkspaceVerification, error) {
	if index < 0 || index >= len(allowed) {
		return WorkspaceVerification{}, fmt.Errorf("verification command index %d is not explicitly allowed", index)
	}
	command := append([]string(nil), allowed[index]...)
	started := time.Now()
	process := exec.CommandContext(ctx, command[0], command[1:]...)
	process.Dir = root
	var output boundedWorkspaceBuffer
	process.Stdout = &output
	process.Stderr = &output
	err := process.Run()
	receipt := WorkspaceVerification{CommandIndex: index, Command: command, ExitCode: 0, Output: output.String(), Truncated: output.truncated, DurationMS: time.Since(started).Milliseconds(), Passed: err == nil}
	if err != nil {
		receipt.ExitCode = -1
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			receipt.ExitCode = exitError.ExitCode()
		}
		return receipt, fmt.Errorf("verification command %d failed with exit code %d", index, receipt.ExitCode)
	}
	return receipt, nil
}

type boundedWorkspaceBuffer struct {
	buffer    bytes.Buffer
	truncated bool
}

func (buffer *boundedWorkspaceBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := WorkspaceVerifyMaxOutput - buffer.buffer.Len()
	if remaining <= 0 {
		buffer.truncated = true
		return original, nil
	}
	if len(data) > remaining {
		data = data[:remaining]
		buffer.truncated = true
	}
	_, _ = buffer.buffer.Write(data)
	return original, nil
}

func (buffer *boundedWorkspaceBuffer) String() string { return buffer.buffer.String() }

func workspaceSystemPrompt(request WorkspaceLoopRequest, allowWrite bool) string {
	writeRule := "APPLY_WRITES is disabled; return candidate edits in final text."
	if allowWrite {
		writeRule = "APPLY_WRITES is enabled only through the bounded transactional writer and requires exact hashes."
	}
	commands, _ := json.Marshal(request.VerifyCommands)
	return "You are WALMI's bounded project workspace loop. Work only inside the selected root. Return exactly one JSON object and no markdown each turn. Available forms: {\"tool\":\"LIST_FILES\",\"path\":\".\"}, {\"tool\":\"READ_FILE\",\"path\":\"relative/path\",\"max_bytes\":65536}, {\"tool\":\"STAT_FILE\",\"path\":\"relative/path\"}, {\"tool\":\"HASH_FILE\",\"path\":\"relative/path\"}, {\"tool\":\"APPLY_WRITES\",\"write_request\":{...}}, {\"tool\":\"RUN_VERIFY\",\"command_index\":0}, or {\"final\":\"visible answer\"}. Never invent file content: list and read first. " + writeRule + " Explicit verification commands by index: " + string(commands) + "\nUSER REQUEST:\n" + request.Prompt
}

func appendWorkspaceTurn(transcript string, turn WorkspaceLoopTurn) string {
	encoded, _ := json.Marshal(turn)
	if len(encoded) > 128*1024 {
		encoded = append(encoded[:128*1024], []byte("...[truncated]")...)
	}
	return transcript + "\nOBSERVATION:\n" + string(encoded)
}

func sealWorkspaceLoopReceipt(receipt WorkspaceLoopReceipt, runErr error) (WorkspaceLoopReceipt, error) {
	if receipt.WorkspaceMutation && !receipt.VerificationRun && receipt.State == "COMPLETED" {
		receipt.State = "COMPLETED_UNVERIFIED"
		receipt.VerificationPassed = false
	}
	receipt.ReceiptSHA256 = ""
	digest, digestErr := digestJSON(receipt, "workspace loop receipt")
	if digestErr != nil {
		return WorkspaceLoopReceipt{}, errors.Join(runErr, digestErr)
	}
	receipt.ReceiptSHA256 = digest
	return receipt, runErr
}
