package axmmirror

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	VerifierRegistrySchema      = "axm.waldo-witness.verifier-registry/v0.1"
	VerifierChangeRequestSchema = "axm.waldo-witness.verifier-change-request/v0.1"
	VerifierChangeReceiptSchema = "axm.waldo-witness.verifier-change-receipt/v0.1"

	VerifierKernelID      = "axm.waldo-witness.verifier-invariant-kernel"
	VerifierKernelVersion = "v0.1"

	VerifierClassSchemaInvariant   = "SCHEMA_INVARIANT"
	VerifierClassProvenanceBinding = "PROVENANCE_BINDING"
	VerifierClassBehaviorEval      = "BEHAVIOR_EVALUATION"
	VerifierClassPrivacyBoundary   = "PRIVACY_BOUNDARY"
	VerifierClassContinuity        = "CONTINUITY_FRESHNESS"
	VerifierClassAuthority         = "AUTHORITY_PERMISSION"
	VerifierClassArtifactRuntime   = "ARTIFACT_RUNTIME"

	VerifierPeerPass = "PASS"
	VerifierPeerFail = "FAIL"
	VerifierPeerHold = "HOLD"

	VerifierCheckPass = "PASS"
	VerifierCheckFail = "FAIL"
	VerifierCheckHold = "HOLD"

	VerifierChangeReady              = "VERIFIER_CHANGE_READY"
	VerifierChangeHoldQuorum         = "HOLD_INDEPENDENT_QUORUM"
	VerifierChangeHoldEvidence       = "HOLD_PEER_EVIDENCE"
	VerifierChangeHoldMergeGate      = "HOLD_PROTECTED_MERGE_GATE"
	VerifierChangeRollbackBinding    = "ROLLBACK_BINDING_MISMATCH"
	VerifierChangeRollbackSelf       = "ROLLBACK_SELF_APPROVAL"
	VerifierChangeRollbackDrift      = "ROLLBACK_REVIEWER_DRIFT"
	VerifierChangeRollbackShadow     = "ROLLBACK_SHADOW_MISMATCH"
	VerifierChangeRollbackPeerReject = "ROLLBACK_PEER_REJECTED"

	maxVerifierDefinitions = 64
	maxVerifierPeerReviews = 16
)

// VerifierInvariantKernel is deliberately outside the evolvable verifier
// collection. Its compiled v0.1 values define the rules that a candidate
// generation cannot weaken: no self-approval, two independent peers, frozen
// replay evidence, closed authority, and retained rollback generations.
type VerifierInvariantKernel struct {
	ID                               string `json:"id"`
	Version                          string `json:"version"`
	MinPeerQuorum                    int    `json:"min_peer_quorum"`
	MinIndependentGroups             int    `json:"min_independent_groups"`
	MaxRetainedGenerations           int    `json:"max_retained_generations"`
	ProtectedChangesRequireMergeGate bool   `json:"protected_changes_require_merge_gate"`
	ContractSHA256                   string `json:"contract_sha256"`
}

// VerifierDefinition is a digest-bound declaration, not executable code. A
// change can name new implementation bytes, rules, and tests, but cannot load
// or execute them through this package.
type VerifierDefinition struct {
	ID                   string `json:"id"`
	Version              string `json:"version"`
	Class                string `json:"class"`
	IndependenceGroup    string `json:"independence_group"`
	DefinitionSHA256     string `json:"definition_sha256"`
	TestPackSHA256       string `json:"test_pack_sha256"`
	ImplementationSHA256 string `json:"implementation_sha256"`
	Protected            bool   `json:"protected"`
}

type VerifierRegistry struct {
	Schema                 string                  `json:"schema"`
	RegistryID             string                  `json:"registry_id"`
	Generation             int                     `json:"generation"`
	PreviousRegistrySHA256 string                  `json:"previous_registry_sha256,omitempty"`
	RetainedRegistrySHA256 []string                `json:"retained_registry_sha256,omitempty"`
	Kernel                 VerifierInvariantKernel `json:"kernel"`
	Verifiers              []VerifierDefinition    `json:"verifiers"`
	Authority              Authority               `json:"authority"`
	RegistrySHA256         string                  `json:"registry_sha256,omitempty"`
}

type VerifierShadowProtocol struct {
	ProtocolID        string `json:"protocol_id"`
	FixtureSetSHA256  string `json:"fixture_set_sha256"`
	CaseCount         int    `json:"case_count"`
	MaxRegressions    int    `json:"max_regressions"`
	MaxUnknownResults int    `json:"max_unknown_results"`
}

type VerifierChangeIntent struct {
	IntentID                 string                 `json:"intent_id"`
	TargetVerifierID         string                 `json:"target_verifier_id"`
	BaselineDefinitionSHA256 string                 `json:"baseline_definition_sha256"`
	Candidate                VerifierDefinition     `json:"candidate"`
	Reason                   string                 `json:"reason"`
	SourceEvidenceSHA256     string                 `json:"source_evidence_sha256"`
	ShadowProtocol           VerifierShadowProtocol `json:"shadow_protocol"`
	Authority                Authority              `json:"authority"`
	IntentSHA256             string                 `json:"intent_sha256,omitempty"`
}

// VerifierPeerReview is evidence authored by a currently registered verifier.
// AssessVerifierChange verifies the reviewer identity and definition against
// the current registry. It never invents or executes the peer review.
type VerifierPeerReview struct {
	ReviewerID                string    `json:"reviewer_id"`
	ReviewerVersion           string    `json:"reviewer_version"`
	ReviewerDefinitionSHA256  string    `json:"reviewer_definition_sha256"`
	ReviewerIndependenceGroup string    `json:"reviewer_independence_group"`
	IntentSHA256              string    `json:"intent_sha256"`
	ProtocolID                string    `json:"protocol_id"`
	FixtureSetSHA256          string    `json:"fixture_set_sha256"`
	BaselineResultSetSHA256   string    `json:"baseline_result_set_sha256"`
	CandidateResultSetSHA256  string    `json:"candidate_result_set_sha256"`
	EvidenceSHA256            string    `json:"evidence_sha256"`
	State                     string    `json:"state"`
	RegressionCount           int       `json:"regression_count"`
	UnknownResultCount        int       `json:"unknown_result_count"`
	Finding                   string    `json:"finding"`
	Authority                 Authority `json:"authority"`
}

type VerifierChangeRequest struct {
	Schema                string               `json:"schema"`
	ChangeID              string               `json:"change_id"`
	CurrentRegistrySHA256 string               `json:"current_registry_sha256"`
	Intent                VerifierChangeIntent `json:"intent"`
	PeerReviews           []VerifierPeerReview `json:"peer_reviews"`
	Authority             Authority            `json:"authority"`
}

type VerifierChangeCheck struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Detail string `json:"detail"`
}

// VerifierChangeReceipt is both the assessment and rollback receipt. READY
// embeds a next-generation registry candidate. Every other state leaves the
// current generation active and pins it as the exact rollback target.
type VerifierChangeReceipt struct {
	Schema                 string                `json:"schema"`
	State                  string                `json:"state"`
	Request                VerifierChangeRequest `json:"request"`
	RequestSHA256          string                `json:"request_sha256"`
	ActiveRegistrySHA256   string                `json:"active_registry_sha256"`
	RollbackTargetSHA256   string                `json:"rollback_target_sha256"`
	RollbackRequired       bool                  `json:"rollback_required"`
	MaterializationAllowed bool                  `json:"materialization_allowed"`
	HumanReviewRequired    bool                  `json:"human_review_required"`
	Checks                 []VerifierChangeCheck `json:"checks"`
	CandidateRegistry      *VerifierRegistry     `json:"candidate_registry,omitempty"`
	Notices                []string              `json:"notices"`
	Authority              Authority             `json:"authority"`
	ReceiptSHA256          string                `json:"receipt_sha256,omitempty"`
}

func DefaultVerifierInvariantKernel() VerifierInvariantKernel {
	kernel := VerifierInvariantKernel{
		ID: VerifierKernelID, Version: VerifierKernelVersion,
		MinPeerQuorum: 2, MinIndependentGroups: 2, MaxRetainedGenerations: 3,
		ProtectedChangesRequireMergeGate: true,
	}
	digest, err := verifierKernelDigest(kernel)
	if err != nil {
		panic(err)
	}
	kernel.ContractSHA256 = digest
	return kernel
}

func NewVerifierRegistry(registryID string, verifiers []VerifierDefinition) (VerifierRegistry, error) {
	if err := requireTrimmed("verifier registry_id", registryID); err != nil {
		return VerifierRegistry{}, err
	}
	canonical, err := canonicalizeVerifierDefinitions(verifiers)
	if err != nil {
		return VerifierRegistry{}, err
	}
	registry := VerifierRegistry{
		Schema: VerifierRegistrySchema, RegistryID: registryID, Generation: 1,
		Kernel: DefaultVerifierInvariantKernel(), Verifiers: canonical, Authority: Authority{},
	}
	registry.RegistrySHA256, err = verifierRegistryDigest(registry)
	if err != nil {
		return VerifierRegistry{}, err
	}
	if err := registry.Validate(); err != nil {
		return VerifierRegistry{}, fmt.Errorf("generated verifier registry: %w", err)
	}
	return registry, nil
}

func SealVerifierChangeIntent(intent VerifierChangeIntent) (VerifierChangeIntent, error) {
	canonical, err := canonicalizeVerifierChangeIntent(intent, false)
	if err != nil {
		return VerifierChangeIntent{}, err
	}
	canonical.IntentSHA256, err = verifierChangeIntentDigest(canonical)
	if err != nil {
		return VerifierChangeIntent{}, err
	}
	return canonical, nil
}

func AssessVerifierChange(current VerifierRegistry, request VerifierChangeRequest) (VerifierChangeReceipt, error) {
	if err := current.Validate(); err != nil {
		return VerifierChangeReceipt{}, fmt.Errorf("current verifier registry: %w", err)
	}
	canonical, err := canonicalizeVerifierChangeRequest(request)
	if err != nil {
		return VerifierChangeReceipt{}, err
	}
	requestDigest, err := digestJSON(canonical, "verifier change request")
	if err != nil {
		return VerifierChangeReceipt{}, err
	}
	receipt := VerifierChangeReceipt{
		Schema: VerifierChangeReceiptSchema, State: VerifierChangeReady,
		Request: canonical, RequestSHA256: requestDigest,
		ActiveRegistrySHA256: current.RegistrySHA256, RollbackTargetSHA256: current.RegistrySHA256,
		MaterializationAllowed: true, HumanReviewRequired: true,
		Notices: []string{
			"peer reviews are externally supplied evidence bound to the current registry and frozen shadow protocol; this assessment does not execute a verifier",
			"READY permits only materialization of a sealed candidate generation; it grants no runtime activation, tool execution, promotion, CANON, or world-action authority",
			"on HOLD or ROLLBACK the current registry remains active and is the exact retained rollback target",
		},
		Authority: Authority{},
	}

	target, targetFound := findVerifier(current.Verifiers, canonical.Intent.TargetVerifierID)
	baselineDigest := ""
	if targetFound {
		baselineDigest, err = verifierDefinitionDigest(target)
		if err != nil {
			return VerifierChangeReceipt{}, err
		}
	}
	addVerifierCheck(&receipt, "baseline-binding", targetFound && canonical.CurrentRegistrySHA256 == current.RegistrySHA256 && canonical.Intent.BaselineDefinitionSHA256 == baselineDigest, false,
		"the request must bind the exact current registry and target definition")
	unchangedBoundary := targetFound && canonical.Intent.Candidate.ID == target.ID && canonical.Intent.Candidate.Class == target.Class && canonical.Intent.Candidate.IndependenceGroup == target.IndependenceGroup && canonical.Intent.Candidate.Protected == target.Protected
	candidateChanged := targetFound && canonical.Intent.Candidate.Version != target.Version && (canonical.Intent.Candidate.DefinitionSHA256 != target.DefinitionSHA256 || canonical.Intent.Candidate.TestPackSHA256 != target.TestPackSHA256 || canonical.Intent.Candidate.ImplementationSHA256 != target.ImplementationSHA256)
	addVerifierCheck(&receipt, "candidate-boundary", unchangedBoundary && candidateChanged, false,
		"v0.1 may revise versioned rules, tests, or implementation digests but not identity, class, independence group, or protection")

	registryByID := make(map[string]VerifierDefinition, len(current.Verifiers))
	for _, definition := range current.Verifiers {
		registryByID[definition.ID] = definition
	}
	selfApproval := false
	reviewerDrift := false
	shadowMismatch := false
	peerRejected := false
	peerHeld := false
	passCount := 0
	groups := map[string]struct{}{}
	baselineResults := ""
	candidateResults := ""
	for _, review := range canonical.PeerReviews {
		if review.ReviewerID == canonical.Intent.TargetVerifierID {
			selfApproval = true
		}
		registered, ok := registryByID[review.ReviewerID]
		registeredDigest := ""
		if ok {
			registeredDigest, err = verifierDefinitionDigest(registered)
			if err != nil {
				return VerifierChangeReceipt{}, err
			}
		}
		if !ok || review.ReviewerVersion != registered.Version || review.ReviewerDefinitionSHA256 != registeredDigest || review.ReviewerIndependenceGroup != registered.IndependenceGroup {
			reviewerDrift = true
		}
		if review.IntentSHA256 != canonical.Intent.IntentSHA256 || review.ProtocolID != canonical.Intent.ShadowProtocol.ProtocolID || review.FixtureSetSHA256 != canonical.Intent.ShadowProtocol.FixtureSetSHA256 {
			shadowMismatch = true
		}
		if baselineResults == "" {
			baselineResults = review.BaselineResultSetSHA256
			candidateResults = review.CandidateResultSetSHA256
		} else if review.BaselineResultSetSHA256 != baselineResults || review.CandidateResultSetSHA256 != candidateResults {
			shadowMismatch = true
		}
		if review.State == VerifierPeerFail || review.RegressionCount > canonical.Intent.ShadowProtocol.MaxRegressions {
			peerRejected = true
		}
		if review.State == VerifierPeerHold || review.UnknownResultCount > canonical.Intent.ShadowProtocol.MaxUnknownResults {
			peerHeld = true
		}
		if review.State == VerifierPeerPass && review.RegressionCount <= canonical.Intent.ShadowProtocol.MaxRegressions && review.UnknownResultCount <= canonical.Intent.ShadowProtocol.MaxUnknownResults {
			passCount++
			groups[review.ReviewerIndependenceGroup] = struct{}{}
		}
	}
	addVerifierCheck(&receipt, "no-self-approval", !selfApproval, false, "the changed verifier cannot review its own candidate")
	addVerifierCheck(&receipt, "reviewer-current-binding", !reviewerDrift, false, "every reviewer must match an active verifier definition in the current registry")
	addVerifierCheck(&receipt, "frozen-shadow-consensus", !shadowMismatch, false, "all peers must bind the same intent, protocol, fixtures, baseline results, and candidate results")
	addVerifierCheck(&receipt, "peer-regression", !peerRejected, false, "any peer FAIL or excess regression rejects the candidate")
	addVerifierCheck(&receipt, "peer-evidence-complete", !peerHeld, peerHeld, "peer HOLD or excess unknown results leaves the change unverified")
	quorum := passCount >= current.Kernel.MinPeerQuorum && len(groups) >= current.Kernel.MinIndependentGroups
	addVerifierCheck(&receipt, "independent-peer-quorum", quorum, !quorum, fmt.Sprintf("requires at least %d passing peers from %d independent groups", current.Kernel.MinPeerQuorum, current.Kernel.MinIndependentGroups))
	protected := targetFound && target.Protected && current.Kernel.ProtectedChangesRequireMergeGate
	addVerifierCheck(&receipt, "protected-merge-gate", !protected, protected, "protected verifier changes remain at the explicit human Merge Gate")

	switch {
	case !targetFound || canonical.CurrentRegistrySHA256 != current.RegistrySHA256 || canonical.Intent.BaselineDefinitionSHA256 != baselineDigest || !unchangedBoundary || !candidateChanged:
		setVerifierRollback(&receipt, VerifierChangeRollbackBinding)
	case selfApproval:
		setVerifierRollback(&receipt, VerifierChangeRollbackSelf)
	case reviewerDrift:
		setVerifierRollback(&receipt, VerifierChangeRollbackDrift)
	case shadowMismatch:
		setVerifierRollback(&receipt, VerifierChangeRollbackShadow)
	case peerRejected:
		setVerifierRollback(&receipt, VerifierChangeRollbackPeerReject)
	case peerHeld:
		setVerifierHold(&receipt, VerifierChangeHoldEvidence)
	case !quorum:
		setVerifierHold(&receipt, VerifierChangeHoldQuorum)
	case protected:
		setVerifierHold(&receipt, VerifierChangeHoldMergeGate)
	default:
		candidate, err := materializeVerifierCandidate(current, canonical.Intent.Candidate)
		if err != nil {
			return VerifierChangeReceipt{}, err
		}
		receipt.CandidateRegistry = &candidate
	}
	sort.Slice(receipt.Checks, func(i, j int) bool { return receipt.Checks[i].ID < receipt.Checks[j].ID })
	receipt.ReceiptSHA256, err = verifierChangeReceiptDigest(receipt)
	if err != nil {
		return VerifierChangeReceipt{}, err
	}
	if err := receipt.Validate(); err != nil {
		return VerifierChangeReceipt{}, fmt.Errorf("generated verifier change receipt: %w", err)
	}
	return receipt, nil
}

// MaterializeVerifierChange recomputes the complete assessment before
// returning the embedded next generation. The caller can write that candidate
// to a new path; this function does not replace an active registry or pointer.
func MaterializeVerifierChange(current VerifierRegistry, receipt VerifierChangeReceipt) (VerifierRegistry, error) {
	if err := current.Validate(); err != nil {
		return VerifierRegistry{}, fmt.Errorf("current verifier registry: %w", err)
	}
	if err := receipt.Validate(); err != nil {
		return VerifierRegistry{}, fmt.Errorf("verifier change receipt: %w", err)
	}
	recomputed, err := AssessVerifierChange(current, receipt.Request)
	if err != nil {
		return VerifierRegistry{}, err
	}
	if recomputed.ReceiptSHA256 != receipt.ReceiptSHA256 {
		return VerifierRegistry{}, errors.New("verifier change receipt does not match a fresh assessment")
	}
	if receipt.State != VerifierChangeReady || !receipt.MaterializationAllowed || receipt.CandidateRegistry == nil {
		return VerifierRegistry{}, fmt.Errorf("verifier change is %s; current registry %s remains active", receipt.State, current.RegistrySHA256)
	}
	return *receipt.CandidateRegistry, nil
}

func (registry VerifierRegistry) Validate() error {
	if registry.Schema != VerifierRegistrySchema {
		return fmt.Errorf("verifier registry schema must be %q", VerifierRegistrySchema)
	}
	if err := requireTrimmed("verifier registry_id", registry.RegistryID); err != nil {
		return err
	}
	if registry.Generation < 1 {
		return errors.New("verifier registry generation must be positive")
	}
	if registry.Generation == 1 && registry.PreviousRegistrySHA256 != "" {
		return errors.New("initial verifier registry must not name a previous generation")
	}
	if registry.Generation > 1 {
		if err := validateSHA256("verifier registry previous_registry_sha256", registry.PreviousRegistrySHA256); err != nil {
			return err
		}
		if len(registry.RetainedRegistrySHA256) == 0 || registry.RetainedRegistrySHA256[0] != registry.PreviousRegistrySHA256 {
			return errors.New("verifier registry must retain its immediate previous generation first")
		}
	}
	if err := validateVerifierKernel(registry.Kernel); err != nil {
		return err
	}
	if len(registry.RetainedRegistrySHA256) > registry.Kernel.MaxRetainedGenerations {
		return errors.New("verifier registry retains more generations than the invariant kernel permits")
	}
	seenRetained := map[string]struct{}{}
	for i, digest := range registry.RetainedRegistrySHA256 {
		if err := validateSHA256(fmt.Sprintf("verifier registry retained_registry_sha256[%d]", i), digest); err != nil {
			return err
		}
		if _, exists := seenRetained[digest]; exists {
			return fmt.Errorf("duplicate retained verifier registry digest %q", digest)
		}
		seenRetained[digest] = struct{}{}
	}
	canonical, err := canonicalizeVerifierDefinitions(registry.Verifiers)
	if err != nil {
		return err
	}
	if !verifierDefinitionsEqual(canonical, registry.Verifiers) {
		return errors.New("verifier registry definitions are not canonical")
	}
	if !registry.Authority.closed() {
		return errors.New("verifier registry must carry closed authority")
	}
	if err := validateSHA256("verifier registry registry_sha256", registry.RegistrySHA256); err != nil {
		return err
	}
	expected, err := verifierRegistryDigest(registry)
	if err != nil {
		return err
	}
	if expected != registry.RegistrySHA256 {
		return errors.New("verifier registry digest mismatch")
	}
	return nil
}

func (receipt VerifierChangeReceipt) Validate() error {
	if receipt.Schema != VerifierChangeReceiptSchema {
		return fmt.Errorf("verifier change receipt schema must be %q", VerifierChangeReceiptSchema)
	}
	canonicalRequest, err := canonicalizeVerifierChangeRequest(receipt.Request)
	if err != nil {
		return err
	}
	requestDigest, err := digestJSON(canonicalRequest, "verifier change request")
	if err != nil {
		return err
	}
	if requestDigest != receipt.RequestSHA256 {
		return errors.New("verifier change request digest mismatch")
	}
	for name, digest := range map[string]string{
		"active_registry_sha256": receipt.ActiveRegistrySHA256,
		"rollback_target_sha256": receipt.RollbackTargetSHA256,
		"receipt_sha256":         receipt.ReceiptSHA256,
	} {
		if err := validateSHA256("verifier change receipt "+name, digest); err != nil {
			return err
		}
	}
	if receipt.ActiveRegistrySHA256 != receipt.Request.CurrentRegistrySHA256 || receipt.RollbackTargetSHA256 != receipt.ActiveRegistrySHA256 {
		return errors.New("verifier change receipt must retain the current registry as its rollback target")
	}
	if !oneOf(receipt.State, VerifierChangeReady, VerifierChangeHoldQuorum, VerifierChangeHoldEvidence, VerifierChangeHoldMergeGate, VerifierChangeRollbackBinding, VerifierChangeRollbackSelf, VerifierChangeRollbackDrift, VerifierChangeRollbackShadow, VerifierChangeRollbackPeerReject) {
		return fmt.Errorf("unsupported verifier change state %q", receipt.State)
	}
	ready := receipt.State == VerifierChangeReady
	rollback := strings.HasPrefix(receipt.State, "ROLLBACK_")
	if receipt.RollbackRequired != rollback {
		return errors.New("verifier change rollback_required does not match state")
	}
	if receipt.MaterializationAllowed != ready || ready != (receipt.CandidateRegistry != nil) {
		return errors.New("only a READY verifier change may carry and materialize a candidate registry")
	}
	if ready {
		if err := receipt.CandidateRegistry.Validate(); err != nil {
			return fmt.Errorf("candidate verifier registry: %w", err)
		}
		if receipt.CandidateRegistry.PreviousRegistrySHA256 != receipt.ActiveRegistrySHA256 {
			return errors.New("candidate verifier registry does not descend from the active registry")
		}
	}
	if !receipt.HumanReviewRequired {
		return errors.New("verifier changes always require human review")
	}
	if len(receipt.Checks) == 0 || len(receipt.Notices) < 3 {
		return errors.New("verifier change receipt requires checks and boundary notices")
	}
	for i, check := range receipt.Checks {
		if err := requireTrimmed(fmt.Sprintf("verifier change check %d id", i), check.ID); err != nil {
			return err
		}
		if !oneOf(check.State, VerifierCheckPass, VerifierCheckFail, VerifierCheckHold) || strings.TrimSpace(check.Detail) == "" {
			return fmt.Errorf("verifier change check %d is incomplete", i)
		}
		if i > 0 && receipt.Checks[i-1].ID >= check.ID {
			return errors.New("verifier change checks must be unique and sorted")
		}
	}
	if !receipt.Authority.closed() {
		return errors.New("verifier change receipt must carry closed authority")
	}
	expected, err := verifierChangeReceiptDigest(receipt)
	if err != nil {
		return err
	}
	if expected != receipt.ReceiptSHA256 {
		return errors.New("verifier change receipt digest mismatch")
	}
	return nil
}

func canonicalizeVerifierChangeRequest(request VerifierChangeRequest) (VerifierChangeRequest, error) {
	if request.Schema != VerifierChangeRequestSchema {
		return VerifierChangeRequest{}, fmt.Errorf("verifier change request schema must be %q", VerifierChangeRequestSchema)
	}
	if err := requireTrimmed("verifier change_id", request.ChangeID); err != nil {
		return VerifierChangeRequest{}, err
	}
	if err := validateSHA256("verifier change current_registry_sha256", request.CurrentRegistrySHA256); err != nil {
		return VerifierChangeRequest{}, err
	}
	intent, err := canonicalizeVerifierChangeIntent(request.Intent, true)
	if err != nil {
		return VerifierChangeRequest{}, err
	}
	if len(request.PeerReviews) > maxVerifierPeerReviews {
		return VerifierChangeRequest{}, fmt.Errorf("verifier change has more than %d peer reviews", maxVerifierPeerReviews)
	}
	canonical := request
	canonical.Intent = intent
	canonical.PeerReviews = append([]VerifierPeerReview(nil), request.PeerReviews...)
	sort.Slice(canonical.PeerReviews, func(i, j int) bool { return canonical.PeerReviews[i].ReviewerID < canonical.PeerReviews[j].ReviewerID })
	for i, review := range canonical.PeerReviews {
		if err := validateVerifierPeerReview(i, review); err != nil {
			return VerifierChangeRequest{}, err
		}
		if i > 0 && canonical.PeerReviews[i-1].ReviewerID == review.ReviewerID {
			return VerifierChangeRequest{}, fmt.Errorf("duplicate verifier peer review from %q", review.ReviewerID)
		}
	}
	if !request.Authority.closed() {
		return VerifierChangeRequest{}, errors.New("verifier change request must carry closed authority")
	}
	return canonical, nil
}

func canonicalizeVerifierChangeIntent(intent VerifierChangeIntent, requireDigest bool) (VerifierChangeIntent, error) {
	for name, value := range map[string]string{
		"intent_id": intent.IntentID, "target_verifier_id": intent.TargetVerifierID, "reason": intent.Reason,
	} {
		if err := requireTrimmed("verifier change intent "+name, value); err != nil {
			return VerifierChangeIntent{}, err
		}
	}
	for name, digest := range map[string]string{
		"baseline_definition_sha256": intent.BaselineDefinitionSHA256,
		"source_evidence_sha256":     intent.SourceEvidenceSHA256,
	} {
		if err := validateSHA256("verifier change intent "+name, digest); err != nil {
			return VerifierChangeIntent{}, err
		}
	}
	if err := validateVerifierDefinition(intent.Candidate); err != nil {
		return VerifierChangeIntent{}, fmt.Errorf("candidate verifier: %w", err)
	}
	if intent.Candidate.ID != intent.TargetVerifierID {
		return VerifierChangeIntent{}, errors.New("candidate verifier id must match target_verifier_id")
	}
	if err := validateVerifierShadowProtocol(intent.ShadowProtocol); err != nil {
		return VerifierChangeIntent{}, err
	}
	if !intent.Authority.closed() {
		return VerifierChangeIntent{}, errors.New("verifier change intent must carry closed authority")
	}
	if requireDigest {
		if err := validateSHA256("verifier change intent intent_sha256", intent.IntentSHA256); err != nil {
			return VerifierChangeIntent{}, err
		}
		expected, err := verifierChangeIntentDigest(intent)
		if err != nil {
			return VerifierChangeIntent{}, err
		}
		if expected != intent.IntentSHA256 {
			return VerifierChangeIntent{}, errors.New("verifier change intent digest mismatch")
		}
	} else {
		intent.IntentSHA256 = ""
	}
	return intent, nil
}

func validateVerifierPeerReview(index int, review VerifierPeerReview) error {
	for name, value := range map[string]string{
		"reviewer_id": review.ReviewerID, "reviewer_version": review.ReviewerVersion,
		"reviewer_independence_group": review.ReviewerIndependenceGroup, "protocol_id": review.ProtocolID,
		"finding": review.Finding,
	} {
		if err := requireTrimmed(fmt.Sprintf("verifier peer review %d %s", index, name), value); err != nil {
			return err
		}
	}
	for name, digest := range map[string]string{
		"reviewer_definition_sha256": review.ReviewerDefinitionSHA256, "intent_sha256": review.IntentSHA256,
		"fixture_set_sha256": review.FixtureSetSHA256, "baseline_result_set_sha256": review.BaselineResultSetSHA256,
		"candidate_result_set_sha256": review.CandidateResultSetSHA256, "evidence_sha256": review.EvidenceSHA256,
	} {
		if err := validateSHA256(fmt.Sprintf("verifier peer review %d %s", index, name), digest); err != nil {
			return err
		}
	}
	if !oneOf(review.State, VerifierPeerPass, VerifierPeerFail, VerifierPeerHold) {
		return fmt.Errorf("verifier peer review %d has unsupported state %q", index, review.State)
	}
	if review.RegressionCount < 0 || review.UnknownResultCount < 0 {
		return fmt.Errorf("verifier peer review %d counts must not be negative", index)
	}
	if !review.Authority.closed() {
		return fmt.Errorf("verifier peer review %d must carry closed authority", index)
	}
	return nil
}

func validateVerifierShadowProtocol(protocol VerifierShadowProtocol) error {
	if err := requireTrimmed("verifier shadow protocol_id", protocol.ProtocolID); err != nil {
		return err
	}
	if err := validateSHA256("verifier shadow fixture_set_sha256", protocol.FixtureSetSHA256); err != nil {
		return err
	}
	if protocol.CaseCount < 1 || protocol.CaseCount > 100000 {
		return errors.New("verifier shadow case_count must be between 1 and 100000")
	}
	if protocol.MaxRegressions != 0 || protocol.MaxUnknownResults != 0 {
		return errors.New("verifier invariant kernel v0.1 requires zero regressions and zero unknown results")
	}
	return nil
}

func validateVerifierKernel(kernel VerifierInvariantKernel) error {
	compiled := DefaultVerifierInvariantKernel()
	if kernel.ID != compiled.ID || kernel.Version != compiled.Version || kernel.MinPeerQuorum != compiled.MinPeerQuorum || kernel.MinIndependentGroups != compiled.MinIndependentGroups || kernel.MaxRetainedGenerations != compiled.MaxRetainedGenerations || kernel.ProtectedChangesRequireMergeGate != compiled.ProtectedChangesRequireMergeGate {
		return errors.New("verifier invariant kernel differs from the compiled v0.1 contract")
	}
	if kernel.ContractSHA256 != compiled.ContractSHA256 {
		return errors.New("verifier invariant kernel digest mismatch")
	}
	return nil
}

func canonicalizeVerifierDefinitions(input []VerifierDefinition) ([]VerifierDefinition, error) {
	if len(input) < 3 || len(input) > maxVerifierDefinitions {
		return nil, fmt.Errorf("verifier registry requires between 3 and %d definitions", maxVerifierDefinitions)
	}
	output := append([]VerifierDefinition(nil), input...)
	sort.Slice(output, func(i, j int) bool { return output[i].ID < output[j].ID })
	groups := map[string]struct{}{}
	for i, definition := range output {
		if err := validateVerifierDefinition(definition); err != nil {
			return nil, fmt.Errorf("verifier definition %d: %w", i, err)
		}
		groups[definition.IndependenceGroup] = struct{}{}
		if i > 0 && output[i-1].ID == definition.ID {
			return nil, fmt.Errorf("duplicate verifier definition %q", definition.ID)
		}
	}
	if len(groups) < 2 {
		return nil, errors.New("verifier registry requires at least two independence groups")
	}
	return output, nil
}

func validateVerifierDefinition(definition VerifierDefinition) error {
	for name, value := range map[string]string{
		"id": definition.ID, "version": definition.Version, "independence_group": definition.IndependenceGroup,
	} {
		if err := requireTrimmed("verifier definition "+name, value); err != nil {
			return err
		}
	}
	if !oneOf(definition.Class, VerifierClassSchemaInvariant, VerifierClassProvenanceBinding, VerifierClassBehaviorEval, VerifierClassPrivacyBoundary, VerifierClassContinuity, VerifierClassAuthority, VerifierClassArtifactRuntime) {
		return fmt.Errorf("unsupported verifier class %q", definition.Class)
	}
	for name, digest := range map[string]string{
		"definition_sha256":     definition.DefinitionSHA256,
		"test_pack_sha256":      definition.TestPackSHA256,
		"implementation_sha256": definition.ImplementationSHA256,
	} {
		if err := validateSHA256("verifier definition "+name, digest); err != nil {
			return err
		}
	}
	return nil
}

func materializeVerifierCandidate(current VerifierRegistry, replacement VerifierDefinition) (VerifierRegistry, error) {
	next := current
	next.Generation++
	next.PreviousRegistrySHA256 = current.RegistrySHA256
	next.RetainedRegistrySHA256 = append([]string{current.RegistrySHA256}, current.RetainedRegistrySHA256...)
	if len(next.RetainedRegistrySHA256) > current.Kernel.MaxRetainedGenerations {
		next.RetainedRegistrySHA256 = next.RetainedRegistrySHA256[:current.Kernel.MaxRetainedGenerations]
	}
	next.Verifiers = append([]VerifierDefinition(nil), current.Verifiers...)
	replaced := false
	for i := range next.Verifiers {
		if next.Verifiers[i].ID == replacement.ID {
			next.Verifiers[i] = replacement
			replaced = true
			break
		}
	}
	if !replaced {
		return VerifierRegistry{}, fmt.Errorf("target verifier %q is not active", replacement.ID)
	}
	next.RegistrySHA256 = ""
	var err error
	next.RegistrySHA256, err = verifierRegistryDigest(next)
	if err != nil {
		return VerifierRegistry{}, err
	}
	if err := next.Validate(); err != nil {
		return VerifierRegistry{}, fmt.Errorf("materialized verifier candidate: %w", err)
	}
	return next, nil
}

func setVerifierRollback(receipt *VerifierChangeReceipt, state string) {
	receipt.State = state
	receipt.RollbackRequired = true
	receipt.MaterializationAllowed = false
	receipt.CandidateRegistry = nil
}

func setVerifierHold(receipt *VerifierChangeReceipt, state string) {
	receipt.State = state
	receipt.RollbackRequired = false
	receipt.MaterializationAllowed = false
	receipt.CandidateRegistry = nil
}

func addVerifierCheck(receipt *VerifierChangeReceipt, id string, pass bool, hold bool, detail string) {
	state := VerifierCheckFail
	if pass {
		state = VerifierCheckPass
	} else if hold {
		state = VerifierCheckHold
	}
	receipt.Checks = append(receipt.Checks, VerifierChangeCheck{ID: id, State: state, Detail: detail})
}

func findVerifier(definitions []VerifierDefinition, id string) (VerifierDefinition, bool) {
	index := sort.Search(len(definitions), func(i int) bool { return definitions[i].ID >= id })
	if index < len(definitions) && definitions[index].ID == id {
		return definitions[index], true
	}
	return VerifierDefinition{}, false
}

func verifierDefinitionDigest(definition VerifierDefinition) (string, error) {
	return digestJSON(definition, "verifier definition")
}

func verifierKernelDigest(kernel VerifierInvariantKernel) (string, error) {
	kernel.ContractSHA256 = ""
	return digestJSON(kernel, "verifier invariant kernel")
}

func verifierRegistryDigest(registry VerifierRegistry) (string, error) {
	registry.RegistrySHA256 = ""
	return digestJSON(registry, "verifier registry")
}

func verifierChangeIntentDigest(intent VerifierChangeIntent) (string, error) {
	intent.IntentSHA256 = ""
	return digestJSON(intent, "verifier change intent")
}

func verifierChangeReceiptDigest(receipt VerifierChangeReceipt) (string, error) {
	receipt.ReceiptSHA256 = ""
	return digestJSON(receipt, "verifier change receipt")
}

func verifierDefinitionsEqual(left, right []VerifierDefinition) bool {
	leftDigest, err := digestJSON(left, "left verifier definitions")
	if err != nil {
		return false
	}
	rightDigest, err := digestJSON(right, "right verifier definitions")
	return err == nil && leftDigest == rightDigest
}
