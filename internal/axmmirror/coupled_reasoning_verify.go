package axmmirror

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

func VerifyCoupledReasoningV024Contract(data []byte) (CoupledReasoningV024Contract, error) {
	var c CoupledReasoningV024Contract
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("decode v0.24: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("trailing JSON")
		}
		return CoupledReasoningV024Contract{}, err
	}
	if err := c.Validate(); err != nil {
		return CoupledReasoningV024Contract{}, err
	}
	d, err := coupledReasoningV024ExternalDigest(c)
	if err != nil {
		return CoupledReasoningV024Contract{}, err
	}
	if d != c.ReceiptDigest {
		return CoupledReasoningV024Contract{}, fmt.Errorf("receipt mismatch: %s != %s", d, c.ReceiptDigest)
	}
	return c, nil
}

func (c CoupledReasoningV024Contract) Validate() error {
	if c.Schema != CoupledReasoningV024Schema || c.Status != CoupledReasoningV024Status || c.Challenge != CoupledReasoningV024Challenge {
		return errors.New("schema/status/challenge mismatch")
	}
	if c.ParentReceipt != CoupledReasoningV024ParentReceipt {
		return errors.New("parent receipt mismatch")
	}
	b := c.Boundary
	if b.LiveCoupledRuntimeObserved || b.LiveEnvironmentExecutionSeen || b.PhysicalLatencyMeasured || b.QuantumEntanglementClaimed || !b.OutputIsContractProbe {
		return errors.New("truth boundary drifted")
	}
	if len(c.Cases) != 11 {
		return fmt.Errorf("expected 11 cases, got %d", len(c.Cases))
	}
	seen := map[string]bool{}
	for _, tc := range c.Cases {
		if tc.CaseID == "" || seen[tc.CaseID] {
			return errors.New("case id empty/duplicate")
		}
		seen[tc.CaseID] = true
		if err := validateCoupledReasoningV024CaseShape(tc); err != nil {
			return fmt.Errorf("case %s: %w", tc.CaseID, err)
		}
		o, r := EvaluateCoupledReasoningV024Case(tc)
		if o != tc.ExpectedOutcome || r != tc.ExpectedReason {
			return fmt.Errorf("case %s got %s/%s want %s/%s", tc.CaseID, o, r, tc.ExpectedOutcome, tc.ExpectedReason)
		}
	}
	t := c.Truth
	if !t.CouplingIsEngineeringMetaphor || !t.ReasoningMayInterleave || !t.ReasoningCanPreemptPendingPlan || t.CouplingGrantsExecutionAuthority || !t.ExecutionControlIsEnvironmentScoped || !t.PermitDoesNotGeneralize || !t.ActuatorEdgeStillRequired || !t.ZeroLatencyNotClaimed || !t.PhysicalAndToolLatencyRemain || !t.ReducedOrchestrationLatencyIsHypothesis || !t.SharedStateDoesNotEraseDissent {
		return errors.New("truth claims drifted")
	}
	if c.Authority != "NONE" {
		return errors.New("authority must remain NONE")
	}
	if !coupledReasoningV024ValidSHA(c.ReceiptDigest) {
		return errors.New("invalid receipt digest")
	}
	return nil
}

func EvaluateCoupledReasoningV024Case(tc CoupledReasoningV024Case) (string, string) {
	if tc.Canon || tc.Promotion != "candidate-only" {
		return "REFUSED", "COUPLED_REASONING_CANNOT_PROMOTE_OR_CANON"
	}
	if tc.ReviewState != "OPEN_TO_REVIEW" {
		return "HOLD", "COUPLED_REASONING_NOT_OPEN_TO_REVIEW"
	}
	if tc.ZeroLatencyClaim {
		return "HOLD", "ZERO_LATENCY_EXECUTION_UNPROVEN"
	}
	if tc.PermitInheritedThroughCoupling {
		return "REFUSED", "COUPLING_CANNOT_SHARE_EXECUTION_AUTHORITY"
	}
	if tc.UnresolvedConflict {
		return "HOLD", "UNRESOLVED_REASONING_CONFLICT"
	}
	if tc.ExecutionRequested {
		if !tc.ActuatorEdgePresent {
			return "HOLD", "ACTUATOR_EDGE_ABSENT"
		}
		if tc.PermitRevokedBeforeAction {
			return "REFUSED", "EXECUTION_REVOKED"
		}
		if !tc.Permit.Present {
			return "HOLD", "EXECUTION_AUTHORITY_ABSENT"
		}
		if tc.Permit.AuthorityScope != "EXECUTE_IN_ENVIRONMENT" || !tc.Permit.Revocable {
			return "REFUSED", "EXECUTION_SCOPE_INVALID"
		}
		if tc.Permit.Environment != tc.RequestedEnvironment {
			return "REFUSED", "ENVIRONMENT_SCOPE_MISMATCH"
		}
		if tc.Permit.Capability != tc.RequestedCapability {
			return "REFUSED", "CAPABILITY_SCOPE_MISMATCH"
		}
		return "READY", "ENVIRONMENT_SCOPED_EXECUTION_AUTHORIZED_NOT_EXECUTED"
	}
	if coupledReasoningV024HasKind(tc.Events, "CONTRADICT") && coupledReasoningV024HasKind(tc.Events, "REPLAN") {
		return "OBSERVED", "COUPLED_REPLAN_BEFORE_EXECUTION"
	}
	return "OBSERVED", "COUPLED_REASONING_ONLY"
}

func SealCoupledReasoningV024Contract(c CoupledReasoningV024Contract) ([]byte, error) {
	d, err := coupledReasoningV024ExternalDigest(c)
	if err != nil {
		return nil, err
	}
	c.ReceiptDigest = d
	return json.MarshalIndent(c, "", "  ")
}

func coupledReasoningV024ExternalDigest(c CoupledReasoningV024Contract) (string, error) {
	c.ReceiptDigest = ""
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func validateCoupledReasoningV024CaseShape(tc CoupledReasoningV024Case) error {
	if tc.Mode != "COUPLED_SHARED_STATE" && tc.Mode != "INTERLEAVED" && tc.Mode != "SEQUENTIAL_BASELINE" {
		return fmt.Errorf("invalid mode %q", tc.Mode)
	}
	if len(tc.Events) == 0 {
		return errors.New("events absent")
	}
	seenSeq := map[int]bool{}
	last := 0
	for _, e := range tc.Events {
		if e.Seq <= last || seenSeq[e.Seq] || e.Emitter == "" || !coupledReasoningV024ValidSHA(e.StateDigest) {
			return errors.New("invalid event sequence")
		}
		if !coupledReasoningV024ValidEmitter(e.Emitter) || !coupledReasoningV024ValidKind(e.Kind) {
			return errors.New("invalid event emitter/kind")
		}
		for _, cause := range e.CausalSeq {
			if cause <= 0 || cause >= e.Seq || !seenSeq[cause] {
				return errors.New("causal event must reference earlier observed event")
			}
		}
		seenSeq[e.Seq] = true
		last = e.Seq
	}
	if tc.ExecutionRequested {
		if tc.RequestedEnvironment == "" || tc.RequestedCapability == "" {
			return errors.New("execution request missing environment/capability")
		}
	} else if tc.RequestedEnvironment != "" || tc.RequestedCapability != "" || tc.ActuatorEdgePresent || tc.Permit.Present || tc.PermitRevokedBeforeAction || tc.PermitInheritedThroughCoupling {
		return errors.New("non-execution case carries execution state")
	}
	if tc.Permit.Present {
		if tc.Permit.Environment == "" || tc.Permit.Capability == "" || tc.Permit.AuthorityScope == "" {
			return errors.New("present permit missing scope")
		}
	} else if tc.Permit.Environment != "" || tc.Permit.Capability != "" || tc.Permit.AuthorityScope != "" || tc.Permit.Revocable {
		return errors.New("absent permit carries scope")
	}
	return nil
}

func coupledReasoningV024ValidEmitter(v string) bool {
	switch v {
	case "PERCEPTION", "BUILDER", "WITNESS", "GAP", "PREDICTION", "REPAIR", "OPINION", "EXECUTION_CONTROL":
		return true
	default:
		return false
	}
}

func coupledReasoningV024ValidKind(v string) bool {
	switch v {
	case "OBSERVE", "PROPOSE", "CONTRADICT", "OPEN_GAP", "PREDICT", "REPLAN", "HOLD", "REQUEST_EXECUTION", "REVOKE_EXECUTION":
		return true
	default:
		return false
	}
}

func coupledReasoningV024HasKind(events []CoupledReasoningV024Event, kind string) bool {
	for _, e := range events {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

func coupledReasoningV024ValidSHA(v string) bool {
	if !strings.HasPrefix(v, "sha256:") || len(v) != 71 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(v, "sha256:"))
	return err == nil
}
