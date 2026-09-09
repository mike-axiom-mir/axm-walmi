// Copyright (c) 2026 OpenWALDO Project contributors
// Copyright (c) 2026 CtrlIQ, Inc.
// Copyright (c) 2026 Gregory M. Kurtzer
// SPDX-License-Identifier: Apache-2.0

package training

import (
	"math"
	"strings"
	"testing"
)

func TestWorkerCompleteObservationValidation(t *testing.T) {
	loss := 1.25
	valid := Observation{
		Steps:          4,
		ConsumedTokens: 32,
		FinalLoss:      &loss,
		Checkpoints: []Checkpoint{{
			Step: 4, Tokens: 32,
			Artifacts: []Artifact{{Path: "artifacts/checkpoints/step-4.bin", SHA256: "checkpoint-sha", Bytes: 8}},
		}},
		Evaluations: []Evaluation{{Step: 4, Tokens: 32, Metrics: map[string]float64{"heldout_loss": 1.5}}},
		Artifacts:   []Artifact{{Path: "artifacts/model.bin", SHA256: "model-sha", Bytes: 16}},
		Consumption: []CorpusConsumption{{Corpus: "base", TokenTargets: 32}},
	}
	if err := (WorkerOutputFrame{Kind: "complete", Schema: WorkerProtocolSchema, Observation: &valid}).Validate(); err != nil {
		t.Fatalf("valid completion observation rejected: %v", err)
	}

	tests := []struct {
		name        string
		observation Observation
		want        string
	}{
		{name: "negative progress", observation: Observation{Steps: -1}, want: "negative progress"},
		{name: "invalid final loss", observation: func() Observation { value := math.NaN(); return Observation{FinalLoss: &value} }(), want: "invalid final loss"},
		{name: "negative checkpoint progress", observation: Observation{Checkpoints: []Checkpoint{{Step: -1}}}, want: "checkpoint 1 contains negative progress"},
		{name: "nonfinite evaluation", observation: Observation{Evaluations: []Evaluation{{Metrics: map[string]float64{"heldout_loss": math.Inf(1)}}}}, want: "metric \"heldout_loss\" is not finite"},
		{name: "unnamed evaluation metric", observation: Observation{Evaluations: []Evaluation{{Metrics: map[string]float64{"": 1}}}}, want: "unnamed metric"},
		{name: "missing artifact path", observation: Observation{Artifacts: []Artifact{{SHA256: "sha", Bytes: 1}}}, want: "artifact path is required"},
		{name: "missing artifact hash", observation: Observation{Artifacts: []Artifact{{Path: "artifacts/model.bin", Bytes: 1}}}, want: "SHA-256 is required"},
		{name: "negative artifact size", observation: Observation{Artifacts: []Artifact{{Path: "artifacts/model.bin", SHA256: "sha", Bytes: -1}}}, want: "size cannot be negative"},
		{name: "missing consumption corpus", observation: Observation{Consumption: []CorpusConsumption{{TokenTargets: 1}}}, want: "has no corpus"},
		{name: "negative consumption", observation: Observation{Consumption: []CorpusConsumption{{Corpus: "base", TokenTargets: -1}}}, want: "negative token targets"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			frame := WorkerOutputFrame{Kind: "complete", Schema: WorkerProtocolSchema, Observation: &test.observation}
			err := frame.Validate()
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Validate() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestReadWorkerOutputRejectsInvalidCompletionObservation(t *testing.T) {
	input := strings.NewReader("{\"kind\":\"complete\",\"schema\":1,\"observation\":{\"steps\":-1}}\n")
	called := false
	err := ReadWorkerOutput(input, func(WorkerOutputFrame) error {
		called = true
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "negative progress") {
		t.Fatalf("ReadWorkerOutput() error = %v, want negative progress", err)
	}
	if called {
		t.Fatal("invalid completion frame reached consumer")
	}
}
