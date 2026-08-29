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
	LiveNeuralGroundingSchema    = "axm.waldo.live-neural-grounding/v0.35"
	LiveNeuralGroundingChallenge = "NEURAL_OUTPUT_IS_NOT_GROUNDING_TRUTH"
)

const (
	GroundingStable    = "STABLE"
	GroundingUncertain = "UNCERTAIN"
	GroundingConflict  = "CONFLICT"
)

const (
	ConsequenceLow  = "LOW"
	ConsequenceHigh = "HIGH"
)

const (
	ReferenceVerified  = "VERIFIED_REFERENCE"
	ReferenceObserved  = "OBSERVED_REFERENCE"
	ReferenceCandidate = "CANDIDATE"
	ReferenceUncertain = "UNCERTAIN"
)

const (
	FreshnessCurrent = "CURRENT"
	FreshnessStale   = "STALE"
)

type LiveGroundingReference struct {
	ID             string `json:"id"`
	Text           string `json:"text"`
	Class          string `json:"class"`
	Freshness      string `json:"freshness"`
	ProvenanceRoot string `json:"provenanceRoot"`
}

type LiveNeuralGrounding struct {
	Schema      string                   `json:"schema"`
	State       string                   `json:"state"`
	Consequence string                   `json:"consequence"`
	References  []LiveGroundingReference `json:"references,omitempty"`
	Authority   Authority                `json:"authority"`
}

func DefaultLiveNeuralGrounding() LiveNeuralGrounding {
	return LiveNeuralGrounding{
		Schema:      LiveNeuralGroundingSchema,
		State:       GroundingStable,
		Consequence: ConsequenceLow,
		Authority:   Authority{},
	}
}

func LoadLiveNeuralGrounding(reader io.Reader) (LiveNeuralGrounding, error) {
	var grounding LiveNeuralGrounding
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&grounding); err != nil {
		return LiveNeuralGrounding{}, fmt.Errorf("decode live neural grounding: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("trailing JSON")
		}
		return LiveNeuralGrounding{}, fmt.Errorf("decode live neural grounding: %w", err)
	}
	if err := grounding.Validate(); err != nil {
		return LiveNeuralGrounding{}, err
	}
	return grounding, nil
}

func (grounding LiveNeuralGrounding) Validate() error {
	if grounding.Schema != LiveNeuralGroundingSchema {
		return fmt.Errorf("schema must be %q", LiveNeuralGroundingSchema)
	}
	if !oneOf(grounding.State, GroundingStable, GroundingUncertain, GroundingConflict) {
		return fmt.Errorf("unsupported grounding state %q", grounding.State)
	}
	if !oneOf(grounding.Consequence, ConsequenceLow, ConsequenceHigh) {
		return fmt.Errorf("unsupported consequence %q", grounding.Consequence)
	}
	if !grounding.Authority.closed() {
		return errors.New("live neural grounding must carry closed authority")
	}
	if len(grounding.References) > 64 {
		return fmt.Errorf("at most 64 grounding references are supported")
	}
	seen := map[string]struct{}{}
	for i, reference := range grounding.References {
		id := strings.TrimSpace(reference.ID)
		if id == "" || len(id) > 128 {
			return fmt.Errorf("references[%d].id is required and limited to 128 bytes", i)
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("duplicate grounding reference id %q", id)
		}
		seen[id] = struct{}{}
		text := strings.TrimSpace(reference.Text)
		if text == "" || len(text) > 2048 {
			return fmt.Errorf("references[%d].text is required and limited to 2048 bytes", i)
		}
		if !oneOf(reference.Class, ReferenceVerified, ReferenceObserved, ReferenceCandidate, ReferenceUncertain) {
			return fmt.Errorf("references[%d] has unsupported class %q", i, reference.Class)
		}
		if !oneOf(reference.Freshness, FreshnessCurrent, FreshnessStale) {
			return fmt.Errorf("references[%d] has unsupported freshness %q", i, reference.Freshness)
		}
		root := strings.TrimSpace(reference.ProvenanceRoot)
		if root == "" || len(root) > 256 {
			return fmt.Errorf("references[%d].provenanceRoot is required and limited to 256 bytes", i)
		}
	}
	return nil
}

func (grounding LiveNeuralGrounding) ShouldDeterministicallyHold() bool {
	return grounding.Consequence == ConsequenceHigh && (grounding.State == GroundingUncertain || grounding.State == GroundingConflict)
}

func (grounding LiveNeuralGrounding) Capsule() (string, string, error) {
	if err := grounding.Validate(); err != nil {
		return "", "", err
	}
	normalized := grounding
	normalized.References = append([]LiveGroundingReference(nil), grounding.References...)
	sort.Slice(normalized.References, func(i, j int) bool {
		return normalized.References[i].ID < normalized.References[j].ID
	})
	payload, err := json.Marshal(normalized)
	if err != nil {
		return "", "", fmt.Errorf("marshal live neural grounding: %w", err)
	}
	sum := sha256.Sum256(payload)
	digest := hex.EncodeToString(sum[:])

	var buffer bytes.Buffer
	fmt.Fprintln(&buffer, "[AXM_GROUNDING_CAPSULE v0.35]")
	fmt.Fprintln(&buffer, "authority=NONE")
	fmt.Fprintf(&buffer, "state=%s\n", normalized.State)
	fmt.Fprintf(&buffer, "consequence=%s\n", normalized.Consequence)
	fmt.Fprintf(&buffer, "grounding_sha256=%s\n", digest)
	fmt.Fprintln(&buffer, "rules:")
	fmt.Fprintln(&buffer, "- neural output is a candidate, not evidence, permission, execution authority, promotion, or CANON")
	fmt.Fprintln(&buffer, "- verified/reference means a bounded reference, not immutable truth")
	fmt.Fprintln(&buffer, "- stale evidence remains historical evidence and must not be silently presented as current")
	fmt.Fprintln(&buffer, "- candidate and uncertain evidence must remain distinguishable from verified/reference evidence")
	fmt.Fprintln(&buffer, "- missing evidence is not contradictory evidence")
	fmt.Fprintln(&buffer, "references:")
	if len(normalized.References) == 0 {
		fmt.Fprintln(&buffer, "- none supplied")
	} else {
		for _, reference := range normalized.References {
			fmt.Fprintf(&buffer, "- id=%s class=%s freshness=%s root=%s text=%s\n",
				oneLine(reference.ID), reference.Class, reference.Freshness, oneLine(reference.ProvenanceRoot), oneLine(reference.Text))
		}
	}
	fmt.Fprintln(&buffer, "[/AXM_GROUNDING_CAPSULE]")
	return buffer.String(), digest, nil
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
