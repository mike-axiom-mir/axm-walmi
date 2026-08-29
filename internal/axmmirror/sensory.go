package axmmirror

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	SensoryEvidenceDraftSchema   = "axm.waldo-witness.sensory-evidence-draft/v0.1"
	SensoryEvidenceReceiptSchema = "axm.waldo-witness.sensory-evidence-receipt/v0.1"

	SensoryStateReady            = "SENSORY_EVIDENCE_READY"
	SensoryStateStale            = "HOLD_SENSORY_STALE"
	SensoryStateUntimed          = "HOLD_SENSORY_UNTIMED"
	SensoryStateTimestamp        = "HOLD_SENSORY_TIMESTAMP"
	SensoryStateIncomplete       = "HOLD_SENSORY_INCOMPLETE"
	SensoryStateUnavailable      = "HOLD_SENSORY_UNAVAILABLE"
	SensoryStateSimulated        = "HOLD_SENSORY_SIMULATED"
	SensoryStateRetention        = "HOLD_SENSORY_RETENTION"
	SensoryStateAuthorityRefused = "REFUSED_SENSORY_AUTHORITY_INHERITANCE"

	SensoryObservationObserved    = "OBSERVED"
	SensoryObservationIncomplete  = "INCOMPLETE"
	SensoryObservationUnavailable = "UNAVAILABLE"
	SensoryObservationSimulated   = "SIMULATED"

	SensoryFreshnessLive    = "LIVE"
	SensoryFreshnessStale   = "STALE"
	SensoryFreshnessUntimed = "UNTIMED"
	SensoryFreshnessFuture  = "FUTURE"
)

// EvidenceSubject is the common immutable target used by situated evidence.
// A name or path alone is insufficient because it can be reused after change.
type EvidenceSubject struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
}

func (subject EvidenceSubject) validate(name string) error {
	if strings.TrimSpace(subject.ID) == "" || strings.TrimSpace(subject.Kind) == "" {
		return fmt.Errorf("%s requires id and kind", name)
	}
	return validateSHA256(name+".sha256", subject.SHA256)
}

// SensoryCapability describes the Workshop Sensorium surface understood by
// this intake. It is an interoperability catalog, not an installed adapter.
type SensoryCapability struct {
	SenseID    string `json:"sense_id"`
	Capability string `json:"capability"`
}

var sensoryCapabilities = map[string]string{
	"compaction-steward":           "sense.provenance.compaction/v1",
	"corroboration-triangulator":   "sense.consensus.claim-corroboration/v1",
	"drift-detector-ambient":       "sense.behavior.drift/v1",
	"ears-stream-listener":         "sense.hearing.bounded-stream/v1",
	"eye-accessibility-inspector":  "visual.inspect.accessibility/v1",
	"eye-change-differ":            "visual.diff.between-stills/v1",
	"eye-live-visual-verifier":     "visual.capture.ephemeral-rolling-buffer/v1",
	"eye-static-image-inspector":   "visual.inspect.static/v1",
	"handoff-continuity-steward":   "sense.continuity.handoff/v1",
	"interoception-capacity-gauge": "sense.interoception.own-capacity/v1",
	"taint-sniffer":                "sense.integrity.taint/v1",
	"time-sense-ttl-verifier":      "sense.time.ttl/v1",
	"touch-environment-probe":      "sense.environment.touch/v1",
}

// SupportedSensoryCapabilities returns a stable copy of the thirteen
// Sensorium contracts. It does not report host availability or permission.
func SupportedSensoryCapabilities() []SensoryCapability {
	result := make([]SensoryCapability, 0, len(sensoryCapabilities))
	for id, capability := range sensoryCapabilities {
		result = append(result, SensoryCapability{SenseID: id, Capability: capability})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].SenseID < result[j].SenseID })
	return result
}

// SensoryEvidenceDraft contains only a bounded, already sealed observation.
// It never contains frames, recordings, transcripts, logs, directory trees,
// or other raw sensory buffers.
type SensoryEvidenceDraft struct {
	Schema                    string          `json:"schema"`
	ObservationID             string          `json:"observation_id"`
	ClaimID                   string          `json:"claim_id"`
	SenseID                   string          `json:"sense_id"`
	Capability                string          `json:"capability"`
	Subject                   EvidenceSubject `json:"subject"`
	SeatID                    string          `json:"seat_id"`
	BackendID                 string          `json:"backend_id,omitempty"`
	ObservedAt                string          `json:"observed_at"`
	SealedAt                  string          `json:"sealed_at"`
	AssessedAt                string          `json:"assessed_at"`
	TTLMillis                 int64           `json:"ttl_millis"`
	ObservationState          string          `json:"observation_state"`
	Verdict                   string          `json:"verdict"`
	NamedSeams                []string        `json:"named_seams,omitempty"`
	TypedObservationSHA256    string          `json:"typed_observation_sha256"`
	SpecificReceiptSchema     string          `json:"specific_receipt_schema"`
	SpecificReceiptSHA256     string          `json:"specific_receipt_sha256"`
	AuthorityLeaseID          string          `json:"authority_lease_id,omitempty"`
	AuthorityInherited        bool            `json:"authority_inherited"`
	RawRetainedBytesAfterSeal int64           `json:"raw_retained_bytes_after_seal"`
	RawRetainedItemsAfterSeal int             `json:"raw_retained_items_after_seal"`
	CleanupComplete           bool            `json:"cleanup_complete"`
}

// SensoryEvidenceReceipt is a WALDO-native witness over a supplied Sensorium
// receipt. READY means the typed evidence is fresh and retained no raw input;
// it does not mean the underlying claim passed, nor that WALDO captured it.
type SensoryEvidenceReceipt struct {
	Schema                        string          `json:"schema"`
	State                         string          `json:"state"`
	ObservationID                 string          `json:"observation_id"`
	ClaimID                       string          `json:"claim_id"`
	SenseID                       string          `json:"sense_id"`
	Capability                    string          `json:"capability"`
	Subject                       EvidenceSubject `json:"subject"`
	SeatID                        string          `json:"seat_id"`
	BackendID                     string          `json:"backend_id,omitempty"`
	ObservedAt                    string          `json:"observed_at"`
	SealedAt                      string          `json:"sealed_at"`
	AssessedAt                    string          `json:"assessed_at"`
	TTLMillis                     int64           `json:"ttl_millis"`
	AgeMillis                     int64           `json:"age_millis"`
	FreshnessStatus               string          `json:"freshness_status"`
	ObservationState              string          `json:"observation_state"`
	Verdict                       string          `json:"verdict"`
	NamedSeams                    []string        `json:"named_seams,omitempty"`
	TypedObservationSHA256        string          `json:"typed_observation_sha256"`
	SpecificReceiptSchema         string          `json:"specific_receipt_schema"`
	SpecificReceiptSHA256         string          `json:"specific_receipt_sha256"`
	DraftSHA256                   string          `json:"draft_sha256"`
	AuthorityLeaseRecorded        bool            `json:"authority_lease_recorded"`
	AuthorityInheritanceRequested bool            `json:"authority_inheritance_requested"`
	AuthorityInherited            bool            `json:"authority_inherited"`
	RawRetainedBytesAfterSeal     int64           `json:"raw_retained_bytes_after_seal"`
	RawRetainedItemsAfterSeal     int             `json:"raw_retained_items_after_seal"`
	CleanupComplete               bool            `json:"cleanup_complete"`
	ReceiptSHA256                 string          `json:"receipt_sha256,omitempty"`
	Holds                         []string        `json:"holds,omitempty"`
	Notices                       []string        `json:"notices"`
	Authority                     Authority       `json:"authority"`
}

// IntakeSensoryEvidence validates one caller-supplied typed observation. Time
// is supplied in the draft so replay remains deterministic and testable.
func IntakeSensoryEvidence(draft SensoryEvidenceDraft) (SensoryEvidenceReceipt, error) {
	canonical, observedAt, sealedAt, assessedAt, err := validateAndCanonicalizeSensoryDraft(draft)
	if err != nil {
		return SensoryEvidenceReceipt{}, err
	}
	draftDigest, err := digestJSON(canonical, "sensory evidence draft")
	if err != nil {
		return SensoryEvidenceReceipt{}, err
	}
	ageMillis := assessedAt.Sub(observedAt).Milliseconds()
	freshness := SensoryFreshnessLive
	if canonical.TTLMillis == 0 {
		freshness = SensoryFreshnessUntimed
	} else if ageMillis < 0 {
		freshness = SensoryFreshnessFuture
	} else if ageMillis > canonical.TTLMillis {
		freshness = SensoryFreshnessStale
	}
	receipt := SensoryEvidenceReceipt{
		Schema: SensoryEvidenceReceiptSchema, State: SensoryStateReady,
		ObservationID: canonical.ObservationID, ClaimID: canonical.ClaimID,
		SenseID: canonical.SenseID, Capability: canonical.Capability, Subject: canonical.Subject,
		SeatID: canonical.SeatID, BackendID: canonical.BackendID,
		ObservedAt: canonical.ObservedAt, SealedAt: canonical.SealedAt, AssessedAt: canonical.AssessedAt,
		TTLMillis: canonical.TTLMillis, AgeMillis: ageMillis, FreshnessStatus: freshness,
		ObservationState: canonical.ObservationState, Verdict: canonical.Verdict,
		NamedSeams: canonical.NamedSeams, TypedObservationSHA256: canonical.TypedObservationSHA256,
		SpecificReceiptSchema: canonical.SpecificReceiptSchema, SpecificReceiptSHA256: canonical.SpecificReceiptSHA256,
		DraftSHA256: draftDigest, AuthorityLeaseRecorded: canonical.AuthorityLeaseID != "",
		AuthorityInheritanceRequested: canonical.AuthorityInherited,
		AuthorityInherited:            false, RawRetainedBytesAfterSeal: canonical.RawRetainedBytesAfterSeal,
		RawRetainedItemsAfterSeal: canonical.RawRetainedItemsAfterSeal, CleanupComplete: canonical.CleanupComplete,
		Notices: []string{
			"the receipt witnesses a caller-supplied typed observation and never claims that WALDO captured raw sensory material",
			"PASS and FAIL are both observations; READY describes intake integrity and freshness, not success of the observed claim",
			"perception is not permission: a recorded source lease is not inherited and grants no tool, capture, execution, or world-action authority",
		},
		Authority: Authority{},
	}

	switch {
	case canonical.AuthorityInherited:
		receipt.State = SensoryStateAuthorityRefused
		receipt.Holds = []string{"the draft attempted to inherit source authority; no authority was transferred"}
	case canonical.RawRetainedBytesAfterSeal != 0 || canonical.RawRetainedItemsAfterSeal != 0 || !canonical.CleanupComplete:
		receipt.State = SensoryStateRetention
		receipt.Holds = []string{"raw sensory retention counters must be zero and cleanup_complete must be true"}
	case canonical.ObservationState == SensoryObservationSimulated:
		receipt.State = SensoryStateSimulated
		receipt.Holds = []string{"the supplied observation is simulated and cannot stand in for observed sensory evidence"}
	case canonical.ObservationState == SensoryObservationUnavailable:
		receipt.State = SensoryStateUnavailable
		receipt.Holds = []string{"the requested sensory observation was unavailable"}
	case canonical.ObservationState == SensoryObservationIncomplete || canonical.Verdict == "UNKNOWN":
		receipt.State = SensoryStateIncomplete
		receipt.Holds = []string{"the typed observation is incomplete or has UNKNOWN verdict"}
	case sealedAt.Before(observedAt) || assessedAt.Before(sealedAt):
		receipt.State = SensoryStateTimestamp
		receipt.Holds = []string{"observed_at, sealed_at, and assessed_at are not in causal order"}
	case freshness == SensoryFreshnessUntimed:
		receipt.State = SensoryStateUntimed
		receipt.Holds = []string{"ttl_millis is zero, so the observation cannot be treated as current"}
	case freshness == SensoryFreshnessFuture:
		receipt.State = SensoryStateTimestamp
		receipt.Holds = []string{"observed_at is later than assessed_at"}
	case freshness == SensoryFreshnessStale:
		receipt.State = SensoryStateStale
		receipt.Holds = []string{fmt.Sprintf("the observation age %d ms exceeds ttl_millis %d", ageMillis, canonical.TTLMillis)}
	}
	receipt.ReceiptSHA256, err = sensoryEvidenceReceiptDigest(receipt)
	if err != nil {
		return SensoryEvidenceReceipt{}, err
	}
	if err := receipt.Validate(); err != nil {
		return SensoryEvidenceReceipt{}, fmt.Errorf("generated sensory evidence receipt: %w", err)
	}
	return receipt, nil
}

func validateAndCanonicalizeSensoryDraft(draft SensoryEvidenceDraft) (SensoryEvidenceDraft, time.Time, time.Time, time.Time, error) {
	if draft.Schema != SensoryEvidenceDraftSchema {
		return SensoryEvidenceDraft{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("sensory draft schema must be %q", SensoryEvidenceDraftSchema)
	}
	for name, value := range map[string]string{
		"observation_id": draft.ObservationID, "claim_id": draft.ClaimID,
		"sense_id": draft.SenseID, "seat_id": draft.SeatID,
		"specific_receipt_schema": draft.SpecificReceiptSchema,
	} {
		if strings.TrimSpace(value) == "" || value != strings.TrimSpace(value) {
			return SensoryEvidenceDraft{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("sensory %s is required and must be trimmed", name)
		}
	}
	capability, ok := sensoryCapabilities[draft.SenseID]
	if !ok {
		return SensoryEvidenceDraft{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("unsupported sensory sense_id %q", draft.SenseID)
	}
	if draft.Capability != capability {
		return SensoryEvidenceDraft{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("sense_id %q requires capability %q", draft.SenseID, capability)
	}
	if err := draft.Subject.validate("sensory subject"); err != nil {
		return SensoryEvidenceDraft{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	for name, value := range map[string]string{
		"typed_observation_sha256": draft.TypedObservationSHA256,
		"specific_receipt_sha256":  draft.SpecificReceiptSHA256,
	} {
		if err := validateSHA256("sensory "+name, value); err != nil {
			return SensoryEvidenceDraft{}, time.Time{}, time.Time{}, time.Time{}, err
		}
	}
	if !oneOf(draft.ObservationState, SensoryObservationObserved, SensoryObservationIncomplete, SensoryObservationUnavailable, SensoryObservationSimulated) {
		return SensoryEvidenceDraft{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("unsupported sensory observation_state %q", draft.ObservationState)
	}
	if !oneOf(draft.Verdict, "PASS", "FAIL", "UNKNOWN") {
		return SensoryEvidenceDraft{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("unsupported sensory verdict %q", draft.Verdict)
	}
	if draft.TTLMillis < 0 || draft.RawRetainedBytesAfterSeal < 0 || draft.RawRetainedItemsAfterSeal < 0 {
		return SensoryEvidenceDraft{}, time.Time{}, time.Time{}, time.Time{}, errors.New("sensory ttl and retention counters must not be negative")
	}
	observedAt, err := time.Parse(time.RFC3339Nano, draft.ObservedAt)
	if err != nil {
		return SensoryEvidenceDraft{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("parse sensory observed_at: %w", err)
	}
	sealedAt, err := time.Parse(time.RFC3339Nano, draft.SealedAt)
	if err != nil {
		return SensoryEvidenceDraft{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("parse sensory sealed_at: %w", err)
	}
	assessedAt, err := time.Parse(time.RFC3339Nano, draft.AssessedAt)
	if err != nil {
		return SensoryEvidenceDraft{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("parse sensory assessed_at: %w", err)
	}
	canonical := draft
	canonical.NamedSeams, err = canonicalNameSet("sensory named_seams", draft.NamedSeams)
	if err != nil {
		return SensoryEvidenceDraft{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	return canonical, observedAt, sealedAt, assessedAt, nil
}

func (receipt SensoryEvidenceReceipt) Validate() error {
	if receipt.Schema != SensoryEvidenceReceiptSchema || !oneOf(receipt.State,
		SensoryStateReady, SensoryStateStale, SensoryStateUntimed, SensoryStateTimestamp,
		SensoryStateIncomplete, SensoryStateUnavailable, SensoryStateSimulated,
		SensoryStateRetention, SensoryStateAuthorityRefused) {
		return fmt.Errorf("unsupported sensory receipt identity %q state %q", receipt.Schema, receipt.State)
	}
	if strings.TrimSpace(receipt.ObservationID) == "" || strings.TrimSpace(receipt.ClaimID) == "" || strings.TrimSpace(receipt.SeatID) == "" {
		return errors.New("sensory receipt has incomplete observation identity")
	}
	if capability, ok := sensoryCapabilities[receipt.SenseID]; !ok || capability != receipt.Capability {
		return errors.New("sensory receipt has an unsupported sense/capability binding")
	}
	if err := receipt.Subject.validate("sensory receipt subject"); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"typed_observation_sha256": receipt.TypedObservationSHA256,
		"specific_receipt_sha256":  receipt.SpecificReceiptSHA256,
		"draft_sha256":             receipt.DraftSHA256,
		"receipt_sha256":           receipt.ReceiptSHA256,
	} {
		if err := validateSHA256("sensory receipt "+name, value); err != nil {
			return err
		}
	}
	if strings.TrimSpace(receipt.SpecificReceiptSchema) == "" || !oneOf(receipt.ObservationState, SensoryObservationObserved, SensoryObservationIncomplete, SensoryObservationUnavailable, SensoryObservationSimulated) || !oneOf(receipt.Verdict, "PASS", "FAIL", "UNKNOWN") {
		return errors.New("sensory receipt has invalid observation state, verdict, or source schema")
	}
	if !oneOf(receipt.FreshnessStatus, SensoryFreshnessLive, SensoryFreshnessStale, SensoryFreshnessUntimed, SensoryFreshnessFuture) {
		return fmt.Errorf("unsupported sensory freshness_status %q", receipt.FreshnessStatus)
	}
	if receipt.TTLMillis < 0 || receipt.RawRetainedBytesAfterSeal < 0 || receipt.RawRetainedItemsAfterSeal < 0 {
		return errors.New("sensory receipt has negative ttl or retention counters")
	}
	expectedState, expectedFreshness, expectedAge, err := expectedSensoryReceiptState(receipt)
	if err != nil {
		return err
	}
	if receipt.State != expectedState || receipt.FreshnessStatus != expectedFreshness || receipt.AgeMillis != expectedAge {
		return fmt.Errorf("sensory receipt state/freshness/age are inconsistent: got %s/%s/%d, want %s/%s/%d", receipt.State, receipt.FreshnessStatus, receipt.AgeMillis, expectedState, expectedFreshness, expectedAge)
	}
	canonicalSeams, err := canonicalNameSet("sensory receipt named_seams", receipt.NamedSeams)
	if err != nil || !stringSlicesEqual(canonicalSeams, receipt.NamedSeams) {
		return errors.New("sensory receipt named_seams are invalid or not canonical")
	}
	if receipt.AuthorityInherited || !receipt.Authority.closed() {
		return errors.New("sensory receipt must not inherit or grant authority")
	}
	if receipt.State == SensoryStateReady {
		if len(receipt.Holds) != 0 || receipt.FreshnessStatus != SensoryFreshnessLive || receipt.ObservationState != SensoryObservationObserved || receipt.Verdict == "UNKNOWN" || receipt.RawRetainedBytesAfterSeal != 0 || receipt.RawRetainedItemsAfterSeal != 0 || !receipt.CleanupComplete {
			return errors.New("ready sensory receipt is stale, incomplete, retained, or contradictory")
		}
	} else if len(receipt.Holds) == 0 {
		return errors.New("held sensory receipt requires a reason")
	}
	if len(receipt.Notices) == 0 {
		return errors.New("sensory receipt must state its boundaries")
	}
	expected, err := sensoryEvidenceReceiptDigest(receipt)
	if err != nil {
		return err
	}
	if expected != receipt.ReceiptSHA256 {
		return fmt.Errorf("sensory receipt digest mismatch: expected %s, got %s", receipt.ReceiptSHA256, expected)
	}
	return nil
}

func expectedSensoryReceiptState(receipt SensoryEvidenceReceipt) (string, string, int64, error) {
	observedAt, err := time.Parse(time.RFC3339Nano, receipt.ObservedAt)
	if err != nil {
		return "", "", 0, fmt.Errorf("parse sensory receipt observed_at: %w", err)
	}
	sealedAt, err := time.Parse(time.RFC3339Nano, receipt.SealedAt)
	if err != nil {
		return "", "", 0, fmt.Errorf("parse sensory receipt sealed_at: %w", err)
	}
	assessedAt, err := time.Parse(time.RFC3339Nano, receipt.AssessedAt)
	if err != nil {
		return "", "", 0, fmt.Errorf("parse sensory receipt assessed_at: %w", err)
	}
	ageMillis := assessedAt.Sub(observedAt).Milliseconds()
	freshness := SensoryFreshnessLive
	if receipt.TTLMillis == 0 {
		freshness = SensoryFreshnessUntimed
	} else if ageMillis < 0 {
		freshness = SensoryFreshnessFuture
	} else if ageMillis > receipt.TTLMillis {
		freshness = SensoryFreshnessStale
	}
	state := SensoryStateReady
	switch {
	case receipt.AuthorityInheritanceRequested:
		state = SensoryStateAuthorityRefused
	case receipt.RawRetainedBytesAfterSeal != 0 || receipt.RawRetainedItemsAfterSeal != 0 || !receipt.CleanupComplete:
		state = SensoryStateRetention
	case receipt.ObservationState == SensoryObservationSimulated:
		state = SensoryStateSimulated
	case receipt.ObservationState == SensoryObservationUnavailable:
		state = SensoryStateUnavailable
	case receipt.ObservationState == SensoryObservationIncomplete || receipt.Verdict == "UNKNOWN":
		state = SensoryStateIncomplete
	case sealedAt.Before(observedAt) || assessedAt.Before(sealedAt):
		state = SensoryStateTimestamp
	case freshness == SensoryFreshnessUntimed:
		state = SensoryStateUntimed
	case freshness == SensoryFreshnessFuture:
		state = SensoryStateTimestamp
	case freshness == SensoryFreshnessStale:
		state = SensoryStateStale
	}
	return state, freshness, ageMillis, nil
}

func sensoryEvidenceReceiptDigest(receipt SensoryEvidenceReceipt) (string, error) {
	receipt.ReceiptSHA256 = ""
	return digestJSON(receipt, "sensory evidence receipt")
}

func canonicalNameSet(name string, values []string) ([]string, error) {
	result := append([]string(nil), values...)
	for i, value := range result {
		if strings.TrimSpace(value) == "" || value != strings.TrimSpace(value) {
			return nil, fmt.Errorf("%s[%d] is empty or not trimmed", name, i)
		}
	}
	sort.Strings(result)
	for i := 1; i < len(result); i++ {
		if result[i] == result[i-1] {
			return nil, fmt.Errorf("%s has duplicate %q", name, result[i])
		}
	}
	return result, nil
}
