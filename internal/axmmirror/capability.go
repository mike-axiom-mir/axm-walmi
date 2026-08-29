package axmmirror

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	SelfCapabilityCensusRequestSchema = "axm.waldo-witness.self-capability-census-request/v0.1"
	SelfCapabilitySnapshotSchema      = "axm.waldo-witness.self-capability-snapshot/v0.1"
	ExternalCapabilitySnapshotSchema  = "axm.waldo-witness.external-capability-snapshot/v0.1"
	ExternalCapabilityReceiptSchema   = "axm.waldo-witness.external-capability-receipt/v0.1"
	CapabilityGapRequestSchema        = "axm.waldo-witness.capability-gap-request/v0.1"
	CapabilityHandoffPlanSchema       = "axm.waldo-witness.capability-handoff-plan/v0.1"

	SelfCapabilityStateDeclared = "SELF_CAPABILITIES_DECLARED"

	ExternalObservationLive       = "LIVE"
	ExternalObservationSourceOnly = "SOURCE_ONLY"
	ExternalObservationAbsent     = "ABSENT"
	ExternalObservationUnknown    = "UNKNOWN"

	ExternalCapabilityLive       = "EXTERNAL_CAPABILITIES_LIVE"
	ExternalCapabilitySourceOnly = "EXTERNAL_CAPABILITIES_SOURCE_ONLY"
	ExternalCapabilityStale      = "HOLD_EXTERNAL_CAPABILITIES_STALE"
	ExternalCapabilityFuture     = "HOLD_EXTERNAL_CAPABILITIES_FUTURE"
	ExternalCapabilityIncomplete = "HOLD_EXTERNAL_CAPABILITIES_INCOMPLETE"
	ExternalCapabilityAbsent     = "HOLD_EXTERNAL_CAPABILITIES_ABSENT"
	ExternalCapabilityUnknown    = "HOLD_EXTERNAL_CAPABILITIES_UNKNOWN"

	ExternalFreshnessFresh  = "FRESH"
	ExternalFreshnessStale  = "STALE"
	ExternalFreshnessFuture = "FUTURE"

	ExternalDeclarationDeclared    = "DECLARED"
	ExternalDeclarationAvailable   = "OBSERVED_AVAILABLE"
	ExternalDeclarationUnavailable = "OBSERVED_UNAVAILABLE"
	ExternalDeclarationUnknown     = "UNKNOWN"

	CapabilityDataPublic    = "PUBLIC"
	CapabilityDataInternal  = "INTERNAL"
	CapabilityDataSensitive = "SENSITIVE"

	HandoffStateReady             = "HANDOFF_CONTRACT_READY"
	HandoffStateLocal             = "NO_EXTERNAL_HANDOFF_REQUIRED"
	HandoffStateExternalHold      = "HOLD_EXTERNAL_CAPABILITY_SNAPSHOT"
	HandoffStateSourceOnly        = "HOLD_EXTERNAL_CAPABILITY_SOURCE_ONLY"
	HandoffStateUnavailable       = "HOLD_CAPABILITY_UNAVAILABLE"
	HandoffStateDataClass         = "HOLD_CAPABILITY_DATA_CLASS"
	HandoffStateSchemaTranslation = "HOLD_SCHEMA_TRANSLATION_REQUIRED"
	HandoffStateProviderAmbiguous = "HOLD_PROVIDER_AMBIGUITY"
	HandoffStatePermission        = "HOLD_PERMISSION_REVIEW"

	CapabilityCompatibilityExact       = "EXACT"
	CapabilityCompatibilityTranslation = "TRANSLATION_REQUIRED"
	CapabilityCompatibilityDataClass   = "DATA_CLASS_MISMATCH"
	CapabilityCompatibilityUnavailable = "UNAVAILABLE"

	MaxCapabilityReceiptTTLMillis  = int64(7 * 24 * time.Hour / time.Millisecond)
	MaxCapabilityArtifactBytes     = int64(64 * 1024 * 1024)
	MaxExternalCapabilityDocuments = 512
	MaxExternalCapabilities        = 4096
)

var gitObjectIDPattern = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

// CapabilityDeclaration is a compiled contract advertisement. It describes a
// deterministic surface in this binary; it is not a runtime health result.
type CapabilityDeclaration struct {
	ID            string   `json:"id"`
	Version       string   `json:"version"`
	Command       string   `json:"command"`
	InputSchemas  []string `json:"input_schemas"`
	OutputSchemas []string `json:"output_schemas"`
	EvidenceState string   `json:"evidence_state"`
}

type SelfCapabilityCensusRequest struct {
	Schema                        string    `json:"schema"`
	CensusID                      string    `json:"census_id"`
	TargetAnsweringIdentitySHA256 string    `json:"target_answering_identity_sha256"`
	ObservedBuildSHA256           string    `json:"observed_build_sha256"`
	ObservationReceiptSHA256      string    `json:"observation_receipt_sha256"`
	CapturedAt                    string    `json:"captured_at"`
	Authority                     Authority `json:"authority"`
}

// SelfCapabilitySnapshot is a Technical-Glasses-like, clone-local declaration
// census. The caller-supplied build hashes remain explicitly external evidence;
// the census does not inspect its own executable or claim runtime readiness.
type SelfCapabilitySnapshot struct {
	Schema                        string                  `json:"schema"`
	State                         string                  `json:"state"`
	CensusID                      string                  `json:"census_id"`
	TargetAnsweringIdentitySHA256 string                  `json:"target_answering_identity_sha256"`
	ObservedBuildSHA256           string                  `json:"observed_build_sha256"`
	ObservationReceiptSHA256      string                  `json:"observation_receipt_sha256"`
	CapturedAt                    string                  `json:"captured_at"`
	CatalogSHA256                 string                  `json:"catalog_sha256"`
	Capabilities                  []CapabilityDeclaration `json:"capabilities"`
	Notices                       []string                `json:"notices"`
	Authority                     Authority               `json:"authority"`
	SnapshotSHA256                string                  `json:"snapshot_sha256,omitempty"`
}

type CapabilitySourceDocument struct {
	Path       string `json:"path"`
	GitBlobSHA string `json:"git_blob_sha"`
	SHA256     string `json:"sha256"`
}

type CapabilitySnapshotSource struct {
	Repository string                     `json:"repository"`
	Branch     string                     `json:"branch"`
	Commit     string                     `json:"commit"`
	Documents  []CapabilitySourceDocument `json:"documents"`
}

// ExternalCapability preserves declarations, observed availability, schemas,
// data boundaries, and permission requirements without making a provider call.
type ExternalCapability struct {
	ID                  string   `json:"id"`
	ProviderID          string   `json:"provider_id"`
	Version             string   `json:"version"`
	ExecutionClass      string   `json:"execution_class"`
	DeclarationState    string   `json:"declaration_state"`
	InputSchemas        []string `json:"input_schemas"`
	OutputSchemas       []string `json:"output_schemas"`
	AcceptedDataClasses []string `json:"accepted_data_classes"`
	RequiredPermissions []string `json:"required_permissions,omitempty"`
	Refusals            []string `json:"refusals,omitempty"`
}

type ExternalCapabilitySnapshot struct {
	Schema                string                   `json:"schema"`
	SnapshotID            string                   `json:"snapshot_id"`
	Source                CapabilitySnapshotSource `json:"source"`
	ObservationState      string                   `json:"observation_state"`
	ObservedAt            string                   `json:"observed_at"`
	AssessedAt            string                   `json:"assessed_at"`
	TTLMillis             int64                    `json:"ttl_millis"`
	InventoryComplete     bool                     `json:"inventory_complete"`
	RuntimeEvidenceSHA256 string                   `json:"runtime_evidence_sha256,omitempty"`
	Capabilities          []ExternalCapability     `json:"capabilities"`
	Authority             Authority                `json:"authority"`
}

// ExternalCapabilityReceipt is an intake verdict over one exact snapshot. A
// SOURCE_ONLY receipt is valid knowledge, but never becomes live readiness.
type ExternalCapabilityReceipt struct {
	Schema         string                     `json:"schema"`
	State          string                     `json:"state"`
	Freshness      string                     `json:"freshness"`
	AgeMillis      int64                      `json:"age_millis"`
	Snapshot       ExternalCapabilitySnapshot `json:"snapshot"`
	SnapshotSHA256 string                     `json:"snapshot_sha256"`
	Holds          []string                   `json:"holds,omitempty"`
	Notices        []string                   `json:"notices"`
	Authority      Authority                  `json:"authority"`
	ReceiptSHA256  string                     `json:"receipt_sha256,omitempty"`
}

type CapabilityArtifactRef struct {
	Schema string `json:"schema"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type CapabilityGapRequest struct {
	Schema                        string                `json:"schema"`
	RequestID                     string                `json:"request_id"`
	TargetAnsweringIdentitySHA256 string                `json:"target_answering_identity_sha256"`
	CapabilityID                  string                `json:"capability_id"`
	Input                         CapabilityArtifactRef `json:"input"`
	RequiredOutputSchema          string                `json:"required_output_schema"`
	DataClass                     string                `json:"data_class"`
	MaxOutputBytes                int64                 `json:"max_output_bytes"`
	CreatedAt                     string                `json:"created_at"`
	ExpiresAt                     string                `json:"expires_at"`
	Authority                     Authority             `json:"authority"`
}

type CapabilityRouteCandidate struct {
	ProviderID          string   `json:"provider_id"`
	CapabilityID        string   `json:"capability_id"`
	Version             string   `json:"version"`
	ExecutionClass      string   `json:"execution_class"`
	DeclarationState    string   `json:"declaration_state"`
	InputSchemas        []string `json:"input_schemas"`
	OutputSchemas       []string `json:"output_schemas"`
	RequiredPermissions []string `json:"required_permissions,omitempty"`
	Compatibility       string   `json:"compatibility"`
}

// CapabilityHandoffPlan is a deterministic contract proposal. Even READY has
// no selected provider and carries no permission or execution authorization.
type CapabilityHandoffPlan struct {
	Schema                    string                     `json:"schema"`
	State                     string                     `json:"state"`
	Request                   CapabilityGapRequest       `json:"request"`
	RequestSHA256             string                     `json:"request_sha256"`
	SelfSnapshotSHA256        string                     `json:"self_snapshot_sha256"`
	ExternalReceiptSHA256     string                     `json:"external_receipt_sha256"`
	ExternalSnapshotSHA256    string                     `json:"external_snapshot_sha256"`
	Candidates                []CapabilityRouteCandidate `json:"candidates,omitempty"`
	UniqueCandidateProviderID string                     `json:"unique_candidate_provider_id,omitempty"`
	ProviderSelected          bool                       `json:"provider_selected"`
	HumanReviewRequired       bool                       `json:"human_review_required"`
	ExternalExecutionAllowed  bool                       `json:"external_execution_allowed"`
	Holds                     []string                   `json:"holds,omitempty"`
	Notices                   []string                   `json:"notices"`
	Authority                 Authority                  `json:"authority"`
	PlanSHA256                string                     `json:"plan_sha256,omitempty"`
}

func CensusSelfCapabilities(request SelfCapabilityCensusRequest) (SelfCapabilitySnapshot, error) {
	canonical, err := canonicalizeSelfCapabilityRequest(request)
	if err != nil {
		return SelfCapabilitySnapshot{}, err
	}
	catalog := selfCapabilityCatalog()
	catalogDigest, err := digestJSON(catalog, "self capability catalog")
	if err != nil {
		return SelfCapabilitySnapshot{}, err
	}
	snapshot := SelfCapabilitySnapshot{
		Schema: SelfCapabilitySnapshotSchema, State: SelfCapabilityStateDeclared,
		CensusID: canonical.CensusID, TargetAnsweringIdentitySHA256: canonical.TargetAnsweringIdentitySHA256,
		ObservedBuildSHA256: canonical.ObservedBuildSHA256, ObservationReceiptSHA256: canonical.ObservationReceiptSHA256,
		CapturedAt: canonical.CapturedAt, CatalogSHA256: catalogDigest, Capabilities: catalog,
		Notices: []string{
			"the catalog records compiled deterministic contract surfaces, not runtime health, host compatibility, successful behavior, or provider availability",
			"observed_build_sha256 and observation_receipt_sha256 are caller-supplied external evidence bindings; the clone does not author or verify its own build provenance",
			"the snapshot grants no tool, provider, installation, training, promotion, CANON, permission, or world-action authority",
		},
		Authority: Authority{},
	}
	snapshot.SnapshotSHA256, err = selfCapabilitySnapshotDigest(snapshot)
	if err != nil {
		return SelfCapabilitySnapshot{}, err
	}
	if err := snapshot.Validate(); err != nil {
		return SelfCapabilitySnapshot{}, fmt.Errorf("generated self capability snapshot: %w", err)
	}
	return snapshot, nil
}

func canonicalizeSelfCapabilityRequest(request SelfCapabilityCensusRequest) (SelfCapabilityCensusRequest, error) {
	if request.Schema != SelfCapabilityCensusRequestSchema {
		return SelfCapabilityCensusRequest{}, fmt.Errorf("self capability census schema must be %q", SelfCapabilityCensusRequestSchema)
	}
	if err := requireTrimmed("self capability census_id", request.CensusID); err != nil {
		return SelfCapabilityCensusRequest{}, err
	}
	for name, value := range map[string]string{
		"target_answering_identity_sha256": request.TargetAnsweringIdentitySHA256,
		"observed_build_sha256":            request.ObservedBuildSHA256,
		"observation_receipt_sha256":       request.ObservationReceiptSHA256,
	} {
		if err := validateSHA256("self capability "+name, value); err != nil {
			return SelfCapabilityCensusRequest{}, err
		}
	}
	capturedAt, err := parseCapabilityTime("self capability captured_at", request.CapturedAt)
	if err != nil {
		return SelfCapabilityCensusRequest{}, err
	}
	if !request.Authority.closed() {
		return SelfCapabilityCensusRequest{}, errors.New("self capability census must carry closed authority")
	}
	request.CapturedAt = capturedAt.UTC().Format(time.RFC3339Nano)
	return request, nil
}

func (snapshot SelfCapabilitySnapshot) Validate() error {
	if snapshot.Schema != SelfCapabilitySnapshotSchema || snapshot.State != SelfCapabilityStateDeclared {
		return fmt.Errorf("unsupported self capability snapshot identity %q state %q", snapshot.Schema, snapshot.State)
	}
	if err := requireTrimmed("self capability census_id", snapshot.CensusID); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"target_answering_identity_sha256": snapshot.TargetAnsweringIdentitySHA256,
		"observed_build_sha256":            snapshot.ObservedBuildSHA256,
		"observation_receipt_sha256":       snapshot.ObservationReceiptSHA256,
		"catalog_sha256":                   snapshot.CatalogSHA256,
		"snapshot_sha256":                  snapshot.SnapshotSHA256,
	} {
		if err := validateSHA256("self capability "+name, value); err != nil {
			return err
		}
	}
	if _, err := parseCapabilityTime("self capability captured_at", snapshot.CapturedAt); err != nil {
		return err
	}
	if len(snapshot.Capabilities) == 0 || len(snapshot.Notices) == 0 {
		return errors.New("self capability snapshot requires capabilities and notices")
	}
	canonical, err := canonicalizeCapabilityDeclarations(snapshot.Capabilities)
	if err != nil || !capabilityDeclarationsEqual(canonical, snapshot.Capabilities) {
		return errors.New("self capability catalog is invalid or not canonical")
	}
	catalogDigest, err := digestJSON(snapshot.Capabilities, "self capability catalog")
	if err != nil {
		return err
	}
	if catalogDigest != snapshot.CatalogSHA256 {
		return errors.New("self capability catalog digest does not match")
	}
	if !snapshot.Authority.closed() {
		return errors.New("self capability snapshot must carry closed authority")
	}
	digest, err := selfCapabilitySnapshotDigest(snapshot)
	if err != nil {
		return err
	}
	if digest != snapshot.SnapshotSHA256 {
		return errors.New("self capability snapshot digest does not match")
	}
	return nil
}

func IntakeExternalCapabilities(snapshot ExternalCapabilitySnapshot) (ExternalCapabilityReceipt, error) {
	canonical, observedAt, assessedAt, err := canonicalizeExternalCapabilitySnapshot(snapshot)
	if err != nil {
		return ExternalCapabilityReceipt{}, err
	}
	snapshotDigest, err := digestJSON(canonical, "external capability snapshot")
	if err != nil {
		return ExternalCapabilityReceipt{}, err
	}
	state, freshness, ageMillis, holds := externalCapabilityState(canonical, observedAt, assessedAt)
	receipt := ExternalCapabilityReceipt{
		Schema: ExternalCapabilityReceiptSchema, State: state, Freshness: freshness, AgeMillis: ageMillis,
		Snapshot: canonical, SnapshotSHA256: snapshotDigest, Holds: holds,
		Notices: []string{
			"LIVE means a caller supplied a fresh runtime evidence digest; this intake does not rerun the provider or independently prove availability",
			"SOURCE_ONLY is useful contract knowledge but cannot be routed as current runtime readiness; ABSENT, UNKNOWN, stale, future, and incomplete remain explicit",
			"capability declarations and handoff metadata grant no permission and do not authorize provider invocation, installation, selection, repair, promotion, CANON, or world action",
		},
		Authority: Authority{},
	}
	receipt.ReceiptSHA256, err = externalCapabilityReceiptDigest(receipt)
	if err != nil {
		return ExternalCapabilityReceipt{}, err
	}
	if err := receipt.Validate(); err != nil {
		return ExternalCapabilityReceipt{}, fmt.Errorf("generated external capability receipt: %w", err)
	}
	return receipt, nil
}

func canonicalizeExternalCapabilitySnapshot(snapshot ExternalCapabilitySnapshot) (ExternalCapabilitySnapshot, time.Time, time.Time, error) {
	if snapshot.Schema != ExternalCapabilitySnapshotSchema {
		return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, fmt.Errorf("external capability snapshot schema must be %q", ExternalCapabilitySnapshotSchema)
	}
	if err := requireTrimmed("external capability snapshot_id", snapshot.SnapshotID); err != nil {
		return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, err
	}
	if !oneOf(snapshot.ObservationState, ExternalObservationLive, ExternalObservationSourceOnly, ExternalObservationAbsent, ExternalObservationUnknown) {
		return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, fmt.Errorf("unsupported external capability observation_state %q", snapshot.ObservationState)
	}
	observedAt, err := parseCapabilityTime("external capability observed_at", snapshot.ObservedAt)
	if err != nil {
		return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, err
	}
	assessedAt, err := parseCapabilityTime("external capability assessed_at", snapshot.AssessedAt)
	if err != nil {
		return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, err
	}
	if snapshot.TTLMillis < 1 || snapshot.TTLMillis > MaxCapabilityReceiptTTLMillis {
		return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, fmt.Errorf("external capability ttl_millis must be between 1 and %d", MaxCapabilityReceiptTTLMillis)
	}
	if !snapshot.Authority.closed() {
		return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, errors.New("external capability snapshot must carry closed authority")
	}
	canonical := snapshot
	canonical.ObservedAt = observedAt.UTC().Format(time.RFC3339Nano)
	canonical.AssessedAt = assessedAt.UTC().Format(time.RFC3339Nano)
	canonical.Source, err = canonicalizeCapabilitySource(snapshot.Source)
	if err != nil {
		return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, err
	}
	if len(snapshot.Capabilities) > MaxExternalCapabilities {
		return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, fmt.Errorf("external capability snapshot has %d capabilities; maximum is %d", len(snapshot.Capabilities), MaxExternalCapabilities)
	}
	canonical.Capabilities = append([]ExternalCapability(nil), snapshot.Capabilities...)
	for i := range canonical.Capabilities {
		canonical.Capabilities[i], err = canonicalizeExternalCapability(canonical.Capabilities[i])
		if err != nil {
			return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, fmt.Errorf("external capability %d: %w", i, err)
		}
	}
	sort.Slice(canonical.Capabilities, func(i, j int) bool {
		if canonical.Capabilities[i].ID != canonical.Capabilities[j].ID {
			return canonical.Capabilities[i].ID < canonical.Capabilities[j].ID
		}
		return canonical.Capabilities[i].ProviderID < canonical.Capabilities[j].ProviderID
	})
	for i := 1; i < len(canonical.Capabilities); i++ {
		left, right := canonical.Capabilities[i-1], canonical.Capabilities[i]
		if left.ID == right.ID && left.ProviderID == right.ProviderID {
			return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, fmt.Errorf("duplicate external capability %q from provider %q", right.ID, right.ProviderID)
		}
	}
	if canonical.RuntimeEvidenceSHA256 != "" {
		if err := validateSHA256("external capability runtime_evidence_sha256", canonical.RuntimeEvidenceSHA256); err != nil {
			return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, err
		}
	}
	switch canonical.ObservationState {
	case ExternalObservationLive:
		if canonical.RuntimeEvidenceSHA256 == "" {
			return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, errors.New("LIVE external capability snapshot requires runtime_evidence_sha256")
		}
		for _, capability := range canonical.Capabilities {
			if capability.DeclarationState == ExternalDeclarationDeclared {
				return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, errors.New("LIVE external snapshot cannot label a capability only DECLARED")
			}
		}
	case ExternalObservationSourceOnly:
		if canonical.RuntimeEvidenceSHA256 != "" {
			return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, errors.New("SOURCE_ONLY external snapshot cannot carry runtime_evidence_sha256")
		}
		for _, capability := range canonical.Capabilities {
			if capability.DeclarationState == ExternalDeclarationAvailable || capability.DeclarationState == ExternalDeclarationUnavailable {
				return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, errors.New("SOURCE_ONLY external snapshot cannot claim observed availability")
			}
		}
	case ExternalObservationAbsent:
		if len(canonical.Capabilities) != 0 || !canonical.InventoryComplete || canonical.RuntimeEvidenceSHA256 == "" {
			return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, errors.New("ABSENT external snapshot requires a complete empty inventory and runtime evidence")
		}
	case ExternalObservationUnknown:
		if len(canonical.Capabilities) != 0 || canonical.InventoryComplete || canonical.RuntimeEvidenceSHA256 != "" {
			return ExternalCapabilitySnapshot{}, time.Time{}, time.Time{}, errors.New("UNKNOWN external snapshot requires an incomplete empty inventory without runtime evidence")
		}
	}
	return canonical, observedAt, assessedAt, nil
}

func canonicalizeCapabilitySource(source CapabilitySnapshotSource) (CapabilitySnapshotSource, error) {
	for name, value := range map[string]string{"repository": source.Repository, "branch": source.Branch} {
		if err := requireTrimmed("external capability source "+name, value); err != nil {
			return CapabilitySnapshotSource{}, err
		}
	}
	if !gitObjectIDPattern.MatchString(source.Commit) {
		return CapabilitySnapshotSource{}, errors.New("external capability source commit must be a lowercase 40- or 64-character Git object ID")
	}
	if len(source.Documents) == 0 || len(source.Documents) > MaxExternalCapabilityDocuments {
		return CapabilitySnapshotSource{}, fmt.Errorf("external capability source requires between 1 and %d documents", MaxExternalCapabilityDocuments)
	}
	canonical := source
	canonical.Documents = append([]CapabilitySourceDocument(nil), source.Documents...)
	sort.Slice(canonical.Documents, func(i, j int) bool { return canonical.Documents[i].Path < canonical.Documents[j].Path })
	for i, document := range canonical.Documents {
		if err := requireTrimmed(fmt.Sprintf("external capability source document %d path", i), document.Path); err != nil {
			return CapabilitySnapshotSource{}, err
		}
		if strings.HasPrefix(document.Path, "/") || strings.Contains(document.Path, "..") || strings.Contains(document.Path, "\\") {
			return CapabilitySnapshotSource{}, fmt.Errorf("external capability source document %q must be a safe repository-relative path", document.Path)
		}
		if !gitObjectIDPattern.MatchString(document.GitBlobSHA) {
			return CapabilitySnapshotSource{}, fmt.Errorf("external capability source document %q has invalid git_blob_sha", document.Path)
		}
		if err := validateSHA256("external capability source document "+document.Path+" sha256", document.SHA256); err != nil {
			return CapabilitySnapshotSource{}, err
		}
		if i > 0 && canonical.Documents[i-1].Path == document.Path {
			return CapabilitySnapshotSource{}, fmt.Errorf("duplicate external capability source document %q", document.Path)
		}
	}
	return canonical, nil
}

func canonicalizeExternalCapability(capability ExternalCapability) (ExternalCapability, error) {
	for name, value := range map[string]string{
		"id": capability.ID, "provider_id": capability.ProviderID, "version": capability.Version,
	} {
		if err := requireTrimmed("external capability "+name, value); err != nil {
			return ExternalCapability{}, err
		}
	}
	if !oneOf(capability.ExecutionClass, "EXTERNAL_HAND", "HOST_SERVICE", "INERT_CONTRACT") {
		return ExternalCapability{}, fmt.Errorf("external capability %q has unsupported execution_class %q", capability.ID, capability.ExecutionClass)
	}
	if !oneOf(capability.DeclarationState, ExternalDeclarationDeclared, ExternalDeclarationAvailable, ExternalDeclarationUnavailable, ExternalDeclarationUnknown) {
		return ExternalCapability{}, fmt.Errorf("external capability %q has unsupported declaration_state %q", capability.ID, capability.DeclarationState)
	}
	canonical := capability
	var err error
	canonical.InputSchemas, err = canonicalNonEmptyNames("external capability input_schemas", capability.InputSchemas)
	if err != nil {
		return ExternalCapability{}, err
	}
	canonical.OutputSchemas, err = canonicalNonEmptyNames("external capability output_schemas", capability.OutputSchemas)
	if err != nil {
		return ExternalCapability{}, err
	}
	canonical.AcceptedDataClasses, err = canonicalNonEmptyNames("external capability accepted_data_classes", capability.AcceptedDataClasses)
	if err != nil {
		return ExternalCapability{}, err
	}
	for _, value := range canonical.AcceptedDataClasses {
		if !oneOf(value, CapabilityDataPublic, CapabilityDataInternal, CapabilityDataSensitive) {
			return ExternalCapability{}, fmt.Errorf("external capability %q has unsupported data class %q", capability.ID, value)
		}
	}
	canonical.RequiredPermissions, err = canonicalNameSet("external capability required_permissions", capability.RequiredPermissions)
	if err != nil {
		return ExternalCapability{}, err
	}
	canonical.Refusals, err = canonicalNameSet("external capability refusals", capability.Refusals)
	if err != nil {
		return ExternalCapability{}, err
	}
	return canonical, nil
}

func (receipt ExternalCapabilityReceipt) Validate() error {
	if receipt.Schema != ExternalCapabilityReceiptSchema || !oneOf(receipt.State,
		ExternalCapabilityLive, ExternalCapabilitySourceOnly, ExternalCapabilityStale, ExternalCapabilityFuture,
		ExternalCapabilityIncomplete, ExternalCapabilityAbsent, ExternalCapabilityUnknown) {
		return fmt.Errorf("unsupported external capability receipt identity %q state %q", receipt.Schema, receipt.State)
	}
	canonical, observedAt, assessedAt, err := canonicalizeExternalCapabilitySnapshot(receipt.Snapshot)
	if err != nil {
		return err
	}
	if !externalSnapshotsEqual(canonical, receipt.Snapshot) {
		return errors.New("external capability receipt snapshot is not canonical")
	}
	for name, value := range map[string]string{"snapshot_sha256": receipt.SnapshotSHA256, "receipt_sha256": receipt.ReceiptSHA256} {
		if err := validateSHA256("external capability receipt "+name, value); err != nil {
			return err
		}
	}
	snapshotDigest, err := digestJSON(receipt.Snapshot, "external capability snapshot")
	if err != nil {
		return err
	}
	if snapshotDigest != receipt.SnapshotSHA256 {
		return errors.New("external capability snapshot digest does not match")
	}
	expectedState, expectedFreshness, expectedAge, expectedHolds := externalCapabilityState(receipt.Snapshot, observedAt, assessedAt)
	if receipt.State != expectedState || receipt.Freshness != expectedFreshness || receipt.AgeMillis != expectedAge || !stringSlicesEqual(receipt.Holds, expectedHolds) {
		return errors.New("external capability receipt state, freshness, age, or holds do not match the snapshot")
	}
	if len(receipt.Notices) == 0 || !receipt.Authority.closed() {
		return errors.New("external capability receipt requires notices and closed authority")
	}
	digest, err := externalCapabilityReceiptDigest(receipt)
	if err != nil {
		return err
	}
	if digest != receipt.ReceiptSHA256 {
		return errors.New("external capability receipt digest does not match")
	}
	return nil
}

func externalCapabilityState(snapshot ExternalCapabilitySnapshot, observedAt, assessedAt time.Time) (string, string, int64, []string) {
	age := assessedAt.Sub(observedAt).Milliseconds()
	freshness := ExternalFreshnessFresh
	if age < 0 {
		freshness = ExternalFreshnessFuture
		return ExternalCapabilityFuture, freshness, age, []string{"external capability observation occurs after assessed_at"}
	}
	if age > snapshot.TTLMillis {
		freshness = ExternalFreshnessStale
		return ExternalCapabilityStale, freshness, age, []string{fmt.Sprintf("external capability evidence age %dms exceeds ttl %dms", age, snapshot.TTLMillis)}
	}
	if !snapshot.InventoryComplete && snapshot.ObservationState != ExternalObservationUnknown {
		return ExternalCapabilityIncomplete, freshness, age, []string{"external capability inventory is incomplete"}
	}
	switch snapshot.ObservationState {
	case ExternalObservationLive:
		return ExternalCapabilityLive, freshness, age, nil
	case ExternalObservationSourceOnly:
		return ExternalCapabilitySourceOnly, freshness, age, nil
	case ExternalObservationAbsent:
		return ExternalCapabilityAbsent, freshness, age, []string{"external capability inventory records the provider surface as absent"}
	default:
		return ExternalCapabilityUnknown, freshness, age, []string{"external capability availability is unknown"}
	}
}

func PlanCapabilityHandoff(self SelfCapabilitySnapshot, external ExternalCapabilityReceipt, request CapabilityGapRequest) (CapabilityHandoffPlan, error) {
	if err := self.Validate(); err != nil {
		return CapabilityHandoffPlan{}, fmt.Errorf("self capability snapshot: %w", err)
	}
	if err := external.Validate(); err != nil {
		return CapabilityHandoffPlan{}, fmt.Errorf("external capability receipt: %w", err)
	}
	canonical, err := canonicalizeCapabilityGapRequest(request)
	if err != nil {
		return CapabilityHandoffPlan{}, err
	}
	if canonical.TargetAnsweringIdentitySHA256 != self.TargetAnsweringIdentitySHA256 {
		return CapabilityHandoffPlan{}, errors.New("capability gap request targets a different answering identity than the self census")
	}
	requestDigest, err := digestJSON(canonical, "capability gap request")
	if err != nil {
		return CapabilityHandoffPlan{}, err
	}
	plan := CapabilityHandoffPlan{
		Schema: CapabilityHandoffPlanSchema, Request: canonical, RequestSHA256: requestDigest,
		SelfSnapshotSHA256: self.SnapshotSHA256, ExternalReceiptSHA256: external.ReceiptSHA256,
		ExternalSnapshotSHA256: external.SnapshotSHA256, ProviderSelected: false,
		HumanReviewRequired: true, ExternalExecutionAllowed: false,
		Notices: []string{
			"the plan uses exact capability, schema, and data-class matches only; it performs no fuzzy or semantic compatibility inference",
			"a unique candidate is not a selected or invoked provider; permission, resource, transport, and execution decisions remain outside this receipt",
			"READY means only that one current declared contract matches without declared permission requirements; it does not prove runtime success, output quality, cleanup, promotion, CANON, or world action",
		},
		Authority: Authority{},
	}
	plan.Candidates = capabilityCandidates(external.Snapshot.Capabilities, canonical)
	plan.State, plan.UniqueCandidateProviderID, plan.Holds = capabilityPlanState(self, external, canonical, plan.Candidates)
	plan.PlanSHA256, err = capabilityHandoffPlanDigest(plan)
	if err != nil {
		return CapabilityHandoffPlan{}, err
	}
	if err := plan.Validate(); err != nil {
		return CapabilityHandoffPlan{}, fmt.Errorf("generated capability handoff plan: %w", err)
	}
	return plan, nil
}

func canonicalizeCapabilityGapRequest(request CapabilityGapRequest) (CapabilityGapRequest, error) {
	if request.Schema != CapabilityGapRequestSchema {
		return CapabilityGapRequest{}, fmt.Errorf("capability gap request schema must be %q", CapabilityGapRequestSchema)
	}
	for name, value := range map[string]string{
		"request_id": request.RequestID, "capability_id": request.CapabilityID,
		"required_output_schema": request.RequiredOutputSchema,
	} {
		if err := requireTrimmed("capability gap "+name, value); err != nil {
			return CapabilityGapRequest{}, err
		}
	}
	if err := validateSHA256("capability gap target_answering_identity_sha256", request.TargetAnsweringIdentitySHA256); err != nil {
		return CapabilityGapRequest{}, err
	}
	if err := validateCapabilityArtifact("capability gap input", request.Input, MaxCapabilityArtifactBytes); err != nil {
		return CapabilityGapRequest{}, err
	}
	if !oneOf(request.DataClass, CapabilityDataPublic, CapabilityDataInternal, CapabilityDataSensitive) {
		return CapabilityGapRequest{}, fmt.Errorf("unsupported capability gap data_class %q", request.DataClass)
	}
	if request.MaxOutputBytes < 1 || request.MaxOutputBytes > MaxCapabilityArtifactBytes {
		return CapabilityGapRequest{}, fmt.Errorf("capability gap max_output_bytes must be between 1 and %d", MaxCapabilityArtifactBytes)
	}
	createdAt, err := parseCapabilityTime("capability gap created_at", request.CreatedAt)
	if err != nil {
		return CapabilityGapRequest{}, err
	}
	expiresAt, err := parseCapabilityTime("capability gap expires_at", request.ExpiresAt)
	if err != nil {
		return CapabilityGapRequest{}, err
	}
	if !expiresAt.After(createdAt) || expiresAt.Sub(createdAt) > 7*24*time.Hour {
		return CapabilityGapRequest{}, errors.New("capability gap expires_at must be after created_at and within seven days")
	}
	if !request.Authority.closed() {
		return CapabilityGapRequest{}, errors.New("capability gap request must carry closed authority")
	}
	request.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
	request.ExpiresAt = expiresAt.UTC().Format(time.RFC3339Nano)
	return request, nil
}

func capabilityCandidates(capabilities []ExternalCapability, request CapabilityGapRequest) []CapabilityRouteCandidate {
	var candidates []CapabilityRouteCandidate
	for _, capability := range capabilities {
		if capability.ID != request.CapabilityID {
			continue
		}
		compatibility := CapabilityCompatibilityTranslation
		switch {
		case capability.DeclarationState == ExternalDeclarationUnavailable || capability.DeclarationState == ExternalDeclarationUnknown:
			compatibility = CapabilityCompatibilityUnavailable
		case !containsString(capability.AcceptedDataClasses, request.DataClass):
			compatibility = CapabilityCompatibilityDataClass
		case containsString(capability.InputSchemas, request.Input.Schema) && containsString(capability.OutputSchemas, request.RequiredOutputSchema):
			compatibility = CapabilityCompatibilityExact
		}
		candidates = append(candidates, CapabilityRouteCandidate{
			ProviderID: capability.ProviderID, CapabilityID: capability.ID, Version: capability.Version,
			ExecutionClass: capability.ExecutionClass, DeclarationState: capability.DeclarationState,
			InputSchemas: append([]string(nil), capability.InputSchemas...), OutputSchemas: append([]string(nil), capability.OutputSchemas...),
			RequiredPermissions: append([]string(nil), capability.RequiredPermissions...), Compatibility: compatibility,
		})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ProviderID < candidates[j].ProviderID })
	return candidates
}

func capabilityPlanState(self SelfCapabilitySnapshot, external ExternalCapabilityReceipt, request CapabilityGapRequest, candidates []CapabilityRouteCandidate) (string, string, []string) {
	for _, capability := range self.Capabilities {
		if capability.ID == request.CapabilityID && containsString(capability.InputSchemas, request.Input.Schema) && containsString(capability.OutputSchemas, request.RequiredOutputSchema) {
			return HandoffStateLocal, "", nil
		}
	}
	planningAt, _ := parseCapabilityTime("capability gap created_at", request.CreatedAt)
	if usable, hold := externalCapabilityUsableAt(external, planningAt); !usable {
		return HandoffStateExternalHold, "", []string{hold}
	}
	if external.State == ExternalCapabilitySourceOnly {
		return HandoffStateSourceOnly, "", []string{"external capability receipt is SOURCE_ONLY and cannot establish current provider readiness"}
	}
	if external.State != ExternalCapabilityLive {
		return HandoffStateExternalHold, "", []string{fmt.Sprintf("external capability receipt is %s, not %s", external.State, ExternalCapabilityLive)}
	}
	if len(candidates) == 0 {
		return HandoffStateUnavailable, "", []string{fmt.Sprintf("no external provider declares capability %q", request.CapabilityID)}
	}
	var available, dataCompatible, exact []CapabilityRouteCandidate
	for _, candidate := range candidates {
		if candidate.DeclarationState != ExternalDeclarationAvailable {
			continue
		}
		available = append(available, candidate)
		if candidate.Compatibility == CapabilityCompatibilityDataClass {
			continue
		}
		dataCompatible = append(dataCompatible, candidate)
		if candidate.Compatibility == CapabilityCompatibilityExact {
			exact = append(exact, candidate)
		}
	}
	if len(available) == 0 {
		return HandoffStateUnavailable, "", []string{"no matching provider is OBSERVED_AVAILABLE in the supplied live snapshot"}
	}
	if len(dataCompatible) == 0 {
		return HandoffStateDataClass, "", []string{fmt.Sprintf("no available provider accepts data class %q", request.DataClass)}
	}
	if len(exact) == 0 {
		return HandoffStateSchemaTranslation, "", []string{"no available provider exactly accepts the input schema and emits the required output schema"}
	}
	if len(exact) > 1 {
		return HandoffStateProviderAmbiguous, "", []string{fmt.Sprintf("%d exact providers require explicit human selection", len(exact))}
	}
	if len(exact[0].RequiredPermissions) != 0 {
		return HandoffStatePermission, exact[0].ProviderID, []string{"the unique exact provider declares permissions that require an external permission decision"}
	}
	return HandoffStateReady, exact[0].ProviderID, nil
}

func externalCapabilityUsableAt(external ExternalCapabilityReceipt, planningAt time.Time) (bool, string) {
	observedAt, _ := parseCapabilityTime("external capability observed_at", external.Snapshot.ObservedAt)
	assessedAt, _ := parseCapabilityTime("external capability assessed_at", external.Snapshot.AssessedAt)
	if planningAt.Before(observedAt) || planningAt.Before(assessedAt) {
		return false, "external capability evidence occurs after capability gap created_at"
	}
	ageMillis := planningAt.Sub(observedAt).Milliseconds()
	if ageMillis > external.Snapshot.TTLMillis {
		return false, fmt.Sprintf("external capability evidence age %dms at plan creation exceeds ttl %dms", ageMillis, external.Snapshot.TTLMillis)
	}
	return true, ""
}

func (plan CapabilityHandoffPlan) Validate() error {
	if plan.Schema != CapabilityHandoffPlanSchema || !oneOf(plan.State,
		HandoffStateReady, HandoffStateLocal, HandoffStateExternalHold, HandoffStateSourceOnly,
		HandoffStateUnavailable, HandoffStateDataClass, HandoffStateSchemaTranslation,
		HandoffStateProviderAmbiguous, HandoffStatePermission) {
		return fmt.Errorf("unsupported capability handoff plan identity %q state %q", plan.Schema, plan.State)
	}
	canonicalRequest, err := canonicalizeCapabilityGapRequest(plan.Request)
	if err != nil {
		return err
	}
	if !capabilityGapRequestsEqual(canonicalRequest, plan.Request) {
		return errors.New("capability handoff request is not canonical")
	}
	for name, value := range map[string]string{
		"request_sha256": plan.RequestSHA256, "self_snapshot_sha256": plan.SelfSnapshotSHA256,
		"external_receipt_sha256": plan.ExternalReceiptSHA256, "external_snapshot_sha256": plan.ExternalSnapshotSHA256,
		"plan_sha256": plan.PlanSHA256,
	} {
		if err := validateSHA256("capability handoff "+name, value); err != nil {
			return err
		}
	}
	requestDigest, err := digestJSON(plan.Request, "capability gap request")
	if err != nil {
		return err
	}
	if requestDigest != plan.RequestSHA256 {
		return errors.New("capability handoff request digest does not match")
	}
	if !sort.SliceIsSorted(plan.Candidates, func(i, j int) bool { return plan.Candidates[i].ProviderID < plan.Candidates[j].ProviderID }) {
		return errors.New("capability handoff candidates are not canonical")
	}
	seen := map[string]struct{}{}
	var available, exact, translationRequired, dataMismatch []CapabilityRouteCandidate
	for i, candidate := range plan.Candidates {
		if err := validateCapabilityCandidate(i, candidate, plan.Request.CapabilityID); err != nil {
			return err
		}
		if _, ok := seen[candidate.ProviderID]; ok {
			return fmt.Errorf("duplicate capability candidate provider %q", candidate.ProviderID)
		}
		seen[candidate.ProviderID] = struct{}{}
		if candidate.DeclarationState == ExternalDeclarationAvailable {
			available = append(available, candidate)
			switch candidate.Compatibility {
			case CapabilityCompatibilityExact:
				exact = append(exact, candidate)
			case CapabilityCompatibilityTranslation:
				translationRequired = append(translationRequired, candidate)
			case CapabilityCompatibilityDataClass:
				dataMismatch = append(dataMismatch, candidate)
			}
		}
	}
	switch plan.State {
	case HandoffStateReady:
		if len(exact) != 1 || plan.UniqueCandidateProviderID != exact[0].ProviderID || len(exact[0].RequiredPermissions) != 0 || len(plan.Holds) != 0 {
			return errors.New("READY capability handoff plan requires a unique candidate and no holds")
		}
	case HandoffStatePermission:
		if len(exact) != 1 || plan.UniqueCandidateProviderID != exact[0].ProviderID || len(exact[0].RequiredPermissions) == 0 {
			return errors.New("permission-held capability plan requires one exact candidate with declared permissions")
		}
	case HandoffStateProviderAmbiguous:
		if len(exact) < 2 {
			return errors.New("ambiguous capability plan requires at least two exact available candidates")
		}
	case HandoffStateSchemaTranslation:
		if len(exact) != 0 || len(translationRequired) == 0 {
			return errors.New("translation-held capability plan requires an available translation candidate and no exact candidate")
		}
	case HandoffStateDataClass:
		if len(available) == 0 || len(dataMismatch) != len(available) {
			return errors.New("data-class-held capability plan requires available candidates that all reject the data class")
		}
	case HandoffStateUnavailable:
		if len(available) != 0 {
			return errors.New("unavailable capability plan cannot contain an observed available candidate")
		}
	}
	if plan.State != HandoffStateReady && plan.State != HandoffStateLocal && len(plan.Holds) == 0 {
		return errors.New("held capability handoff plan requires a hold reason")
	}
	if plan.State != HandoffStateReady && plan.State != HandoffStatePermission && plan.UniqueCandidateProviderID != "" {
		return errors.New("capability handoff plan exposes a unique candidate only for READY or permission-review states")
	}
	if plan.ProviderSelected || plan.ExternalExecutionAllowed || !plan.HumanReviewRequired {
		return errors.New("capability handoff plan cannot select or authorize a provider and must require human review")
	}
	if len(plan.Notices) == 0 || !plan.Authority.closed() {
		return errors.New("capability handoff plan requires notices and closed authority")
	}
	digest, err := capabilityHandoffPlanDigest(plan)
	if err != nil {
		return err
	}
	if digest != plan.PlanSHA256 {
		return errors.New("capability handoff plan digest does not match")
	}
	return nil
}

func validateCapabilityCandidate(index int, candidate CapabilityRouteCandidate, capabilityID string) error {
	for name, value := range map[string]string{"provider_id": candidate.ProviderID, "version": candidate.Version} {
		if err := requireTrimmed(fmt.Sprintf("capability candidate %d %s", index, name), value); err != nil {
			return err
		}
	}
	if candidate.CapabilityID != capabilityID {
		return fmt.Errorf("capability candidate %d has capability %q, expected %q", index, candidate.CapabilityID, capabilityID)
	}
	if !oneOf(candidate.ExecutionClass, "EXTERNAL_HAND", "HOST_SERVICE", "INERT_CONTRACT") ||
		!oneOf(candidate.DeclarationState, ExternalDeclarationDeclared, ExternalDeclarationAvailable, ExternalDeclarationUnavailable, ExternalDeclarationUnknown) ||
		!oneOf(candidate.Compatibility, CapabilityCompatibilityExact, CapabilityCompatibilityTranslation, CapabilityCompatibilityDataClass, CapabilityCompatibilityUnavailable) {
		return fmt.Errorf("capability candidate %d has unsupported class, declaration, or compatibility", index)
	}
	for name, values := range map[string][]string{
		"input_schemas": candidate.InputSchemas, "output_schemas": candidate.OutputSchemas,
		"required_permissions": candidate.RequiredPermissions,
	} {
		canonical, err := canonicalNameSet("capability candidate "+name, values)
		if err != nil || !stringSlicesEqual(canonical, values) {
			return fmt.Errorf("capability candidate %d %s is invalid or not canonical", index, name)
		}
	}
	if len(candidate.InputSchemas) == 0 || len(candidate.OutputSchemas) == 0 {
		return fmt.Errorf("capability candidate %d requires input and output schemas", index)
	}
	return nil
}

func selfCapabilityCatalog() []CapabilityDeclaration {
	catalog := []CapabilityDeclaration{
		{ID: "asset.inner.forge", Version: "v0.1", Command: "forge-asset", InputSchemas: []string{InnerAssetRecipeSchema}, OutputSchemas: []string{InnerAssetCandidateSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "asset.inner.verify", Version: "v0.1", Command: "verify-asset", InputSchemas: []string{InnerAssetCandidateSchema}, OutputSchemas: []string{InnerAssetValidationSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "behavior.evidence.seal", Version: "v0.1", Command: "seal", InputSchemas: []string{BehaviorEvidenceSchema}, OutputSchemas: []string{BehaviorEvidenceSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "capability.external.intake", Version: "v0.1", Command: "intake-capabilities", InputSchemas: []string{ExternalCapabilitySnapshotSchema}, OutputSchemas: []string{ExternalCapabilityReceiptSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "capability.handoff.plan", Version: "v0.1", Command: "plan-handoff", InputSchemas: []string{CapabilityGapRequestSchema}, OutputSchemas: []string{CapabilityHandoffPlanSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "capability.return.verify", Version: "v0.1", Command: "verify-handoff-return", InputSchemas: []string{CapabilityReturnDraftSchema}, OutputSchemas: []string{CapabilityReturnReceiptSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "capability.self.census", Version: "v0.1", Command: "census-capabilities", InputSchemas: []string{SelfCapabilityCensusRequestSchema}, OutputSchemas: []string{SelfCapabilitySnapshotSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "capability.translation.seal", Version: "v0.1", Command: "seal-translation", InputSchemas: []string{CapabilityTranslationDeclarationSchema}, OutputSchemas: []string{TranslationLossReceiptSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "corpus.evidence.lens", Version: "v0.2", Command: "lens-corpus", InputSchemas: []string{"openwaldo-bom/corpus/v1"}, OutputSchemas: []string{CorpusEvidenceLensSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "discovery.dual.stance", Version: "v0.1", Command: "discover", InputSchemas: []string{DiscoveryStanceRequestSchema}, OutputSchemas: []string{DiscoveryStancePacketSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "evaluation.contamination.check", Version: "v0.1", Command: "contamination", InputSchemas: []string{EvaluationComparisonSchema}, OutputSchemas: []string{ContaminationReportSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "evaluation.protocol.seal", Version: "v0.1", Command: "seal-evaluation", InputSchemas: []string{EvaluationProtocolDraftSchema}, OutputSchemas: []string{EvaluationProtocolSealSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "identity.tool.experience.seal", Version: "v0.1", Command: "seal-tool-experience", InputSchemas: []string{IdentityToolExperienceSchema}, OutputSchemas: []string{IdentityToolExperienceSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "identity.tool.memory.grow", Version: "v0.1", Command: "grow-tool-memory", InputSchemas: []string{IdentityToolMemorySchema, IdentityToolExperienceSchema}, OutputSchemas: []string{IdentityToolMemorySchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "identity.tool.memory.start", Version: "v0.1", Command: "start-tool-memory", InputSchemas: []string{IdentityToolExperienceSchema}, OutputSchemas: []string{IdentityToolMemorySchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "identity.tool.wisdom.recall", Version: "v0.1", Command: "recall-tool-wisdom", InputSchemas: []string{IdentityToolMemorySchema, IdentityWisdomQuerySchema}, OutputSchemas: []string{IdentityWisdomViewSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "origin.anchor", Version: "v0.1", Command: "anchor", InputSchemas: []string{"openwaldo-bom/model/v1", "openwaldo-bom/model-release/v1"}, OutputSchemas: []string{OriginAnchorSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "profile.contract.lens", Version: "v0.1", Command: "profile-contract", InputSchemas: []string{"openwaldo-bom/training-run/v1"}, OutputSchemas: []string{TrainingProfileContractSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "provenance.context.build", Version: "v0.1", Command: "context", InputSchemas: []string{ProvenanceContextRequestSchema}, OutputSchemas: []string{ProvenanceContextSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "release.identity.lock", Version: "v0.1", Command: "lock", InputSchemas: []string{OriginAnchorSchema}, OutputSchemas: []string{IdentityLockSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "sensory.evidence.intake", Version: "v0.1", Command: "intake-sensory", InputSchemas: []string{SensoryEvidenceDraftSchema}, OutputSchemas: []string{SensoryEvidenceReceiptSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "situated.context.build", Version: "v0.1", Command: "situated-context", InputSchemas: []string{SituatedContextRequestSchema}, OutputSchemas: []string{SituatedContextEnvelopeSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "skills.continuity.assess", Version: "v0.1", Command: "assess-skills", InputSchemas: []string{SkillContinuityRequestSchema}, OutputSchemas: []string{SkillContinuityReceiptSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "source.claim.gate", Version: "v0.1", Command: "gate-claims", InputSchemas: []string{SourceClaimSubmissionSchema}, OutputSchemas: []string{SourceClaimAssessmentSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "training.run.witness", Version: "v0.1", Command: "witness-run", InputSchemas: []string{"openwaldo-bom/training-run/v1"}, OutputSchemas: []string{TrainingRunWitnessSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "verifier.change.assess", Version: "v0.1", Command: "assess-verifier-change", InputSchemas: []string{VerifierRegistrySchema, VerifierChangeRequestSchema}, OutputSchemas: []string{VerifierChangeReceiptSchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "verifier.change.materialize", Version: "v0.1", Command: "materialize-verifier-change", InputSchemas: []string{VerifierRegistrySchema, VerifierChangeReceiptSchema}, OutputSchemas: []string{VerifierRegistrySchema}, EvidenceState: "COMPILED_DECLARATION"},
		{ID: "verifier.repair.plan", Version: "v0.1", Command: "plan-verifier-repair", InputSchemas: []string{VerifierRegistrySchema, VerifierChangeReceiptSchema, RepairBuddyRequestSchema}, OutputSchemas: []string{RepairBuddyPlanSchema}, EvidenceState: "COMPILED_DECLARATION"},
	}
	canonical, err := canonicalizeCapabilityDeclarations(catalog)
	if err != nil {
		panic(err)
	}
	return canonical
}

func canonicalizeCapabilityDeclarations(input []CapabilityDeclaration) ([]CapabilityDeclaration, error) {
	result := append([]CapabilityDeclaration(nil), input...)
	for i := range result {
		capability := &result[i]
		for name, value := range map[string]string{"id": capability.ID, "version": capability.Version, "command": capability.Command} {
			if err := requireTrimmed("self capability "+name, value); err != nil {
				return nil, err
			}
		}
		if capability.EvidenceState != "COMPILED_DECLARATION" {
			return nil, fmt.Errorf("self capability %q has unsupported evidence_state %q", capability.ID, capability.EvidenceState)
		}
		var err error
		capability.InputSchemas, err = canonicalNonEmptyNames("self capability input_schemas", capability.InputSchemas)
		if err != nil {
			return nil, err
		}
		capability.OutputSchemas, err = canonicalNonEmptyNames("self capability output_schemas", capability.OutputSchemas)
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	for i := 1; i < len(result); i++ {
		if result[i-1].ID == result[i].ID {
			return nil, fmt.Errorf("duplicate self capability %q", result[i].ID)
		}
	}
	return result, nil
}

func canonicalNonEmptyNames(name string, values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("%s requires at least one value", name)
	}
	return canonicalNameSet(name, values)
}

func requireTrimmed(name, value string) error {
	if strings.TrimSpace(value) == "" || value != strings.TrimSpace(value) {
		return fmt.Errorf("%s is required and must be trimmed", name)
	}
	return nil
}

func parseCapabilityTime(name, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}

func validateCapabilityArtifact(name string, artifact CapabilityArtifactRef, maximum int64) error {
	if err := requireTrimmed(name+" schema", artifact.Schema); err != nil {
		return err
	}
	if err := validateSHA256(name+" sha256", artifact.SHA256); err != nil {
		return err
	}
	if artifact.Bytes < 1 || artifact.Bytes > maximum {
		return fmt.Errorf("%s bytes must be between 1 and %d", name, maximum)
	}
	return nil
}

func containsString(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
}

func selfCapabilitySnapshotDigest(snapshot SelfCapabilitySnapshot) (string, error) {
	snapshot.SnapshotSHA256 = ""
	return digestJSON(snapshot, "self capability snapshot")
}

func externalCapabilityReceiptDigest(receipt ExternalCapabilityReceipt) (string, error) {
	receipt.ReceiptSHA256 = ""
	return digestJSON(receipt, "external capability receipt")
}

func capabilityHandoffPlanDigest(plan CapabilityHandoffPlan) (string, error) {
	plan.PlanSHA256 = ""
	return digestJSON(plan, "capability handoff plan")
}

func capabilityDeclarationsEqual(left, right []CapabilityDeclaration) bool {
	leftDigest, err := digestJSON(left, "left capability declarations")
	if err != nil {
		return false
	}
	rightDigest, err := digestJSON(right, "right capability declarations")
	return err == nil && leftDigest == rightDigest
}

func externalSnapshotsEqual(left, right ExternalCapabilitySnapshot) bool {
	leftDigest, err := digestJSON(left, "left external capability snapshot")
	if err != nil {
		return false
	}
	rightDigest, err := digestJSON(right, "right external capability snapshot")
	return err == nil && leftDigest == rightDigest
}

func capabilityGapRequestsEqual(left, right CapabilityGapRequest) bool {
	leftDigest, err := digestJSON(left, "left capability gap request")
	if err != nil {
		return false
	}
	rightDigest, err := digestJSON(right, "right capability gap request")
	return err == nil && leftDigest == rightDigest
}
