package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/openwaldo/waldo/internal/axmmirror"
	"github.com/openwaldo/waldo/internal/inference"
)

type localWorkspaceNeuralModel struct {
	escalator axmmirror.MirrorNeuralEscalator
}

func (model localWorkspaceNeuralModel) Generate(ctx context.Context, prompt string) (string, error) {
	candidate, err := model.escalator.Escalate(ctx, prompt)
	if err != nil {
		return "", err
	}
	return candidate.Text, nil
}

type workspaceExperienceEnvelope struct {
	Schema           string                         `json:"schema"`
	State            string                         `json:"state"`
	RecordedAt       time.Time                      `json:"recorded_at"`
	Receipt          axmmirror.WorkspaceLoopReceipt `json:"receipt"`
	Authority        string                         `json:"authority"`
	LearningMutation bool                           `json:"learning_mutation"`
	WeightMutation   bool                           `json:"weight_mutation"`
}

func runMirrorWorkspace(commandContext Context, args []string, stdout, _ io.Writer) error {
	root := args[0]
	requestData, err := os.ReadFile(args[1])
	if err != nil {
		return fmt.Errorf("read workspace request %s: %w", args[1], err)
	}
	request, err := axmmirror.LoadWorkspaceLoopRequest(requestData)
	if err != nil {
		return err
	}
	hand, err := axmmirror.NewWorkspaceHand(root)
	if err != nil {
		return err
	}
	options := inference.Options{MaxTokens: intOption(commandContext, "max-tokens"), Temperature: float64Option(commandContext, "temperature"), TopP: float64Option(commandContext, "top-p")}
	if optionChanged(commandContext, "seed") {
		seed := uint64Option(commandContext, "seed")
		options.Seed = &seed
	}
	if err := options.Validate(); err != nil {
		return err
	}
	timeoutSeconds := intOption(commandContext, "timeout-seconds")
	if timeoutSeconds < 1 || timeoutSeconds > 3600 {
		return errors.New("--timeout-seconds must be in 1..3600")
	}
	modelName := strings.TrimSpace(stringOption(commandContext, "model"))
	escalator, closeEscalator, err := openMirrorNeuralEscalator(commandContext.Execution, modelName, options)
	if err != nil {
		return fmt.Errorf("open local WALDO model %q for workspace loop: %w", modelName, err)
	}
	ctx, cancel := context.WithTimeout(commandContext.Execution, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	receipt, runErr := axmmirror.RunWorkspaceLoop(ctx, hand, localWorkspaceNeuralModel{escalator: escalator}, request, boolOption(commandContext, "allow-write"))
	closeErr := closeEscalator()
	experiencePath := strings.TrimSpace(stringOption(commandContext, "experience-to"))
	if experiencePath != "" && receipt.Schema != "" {
		envelope := workspaceExperienceEnvelope{Schema: "axm.waldo.workspace-experience-candidate/v0.2", State: "CANDIDATE_REVIEW_REQUIRED", RecordedAt: time.Now().UTC(), Receipt: receipt, Authority: "NONE"}
		line, marshalErr := json.Marshal(envelope)
		if marshalErr != nil {
			return marshalErr
		}
		if appendErr := appendPrivateLine(experiencePath, line); appendErr != nil {
			return fmt.Errorf("append workspace experience candidate: %w", appendErr)
		}
	}
	if receipt.Schema != "" {
		if commandContext.JSON {
			if err := writeJSON(stdout, receipt); err != nil {
				return err
			}
		} else if _, err := fmt.Fprintf(stdout, "%s\nstate: %s\nturns: %d\nworkspace mutation: %t\nverification: run=%t passed=%t\nauthority: %s\n", receipt.Final, receipt.State, len(receipt.Turns), receipt.WorkspaceMutation, receipt.VerificationRun, receipt.VerificationPassed, receipt.Authority); err != nil {
			return err
		}
	}
	return errors.Join(runErr, closeErr)
}
