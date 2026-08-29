package axmmirror

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	IdentityToolExperienceSchema = "axm.waldo-witness.identity-tool-experience/v0.1"
	IdentityToolMemorySchema     = "axm.waldo-witness.identity-tool-memory-shard/v0.1"
	IdentityWisdomQuerySchema    = "axm.waldo-witness.identity-wisdom-query/v0.1"
	IdentityWisdomViewSchema     = "axm.waldo-witness.identity-wisdom-view/v0.1"

	ToolMemoryReady          = "TOOL_MEMORY_READY"
	ToolWisdomReady          = "TOOL_WISDOM_READY"
	ToolWisdomNoRelevantHold = "HOLD_NO_RELEVANT_TOOL_WISDOM"

	ToolOutcomeSuccess = "SUCCESS"
	ToolOutcomeFailure = "FAILURE"

	ToolVerificationSuccess = "INDEPENDENTLY_VERIFIED_SUCCESS"
	ToolVerificationFailure = "INDEPENDENTLY_VERIFIED_FAILURE"

	ToolWisdomReuse = "REUSE"
	ToolWisdomAvoid = "AVOID"

	ToolMemoryContentPublicSafe = "PUBLIC_SAFE_DISTILLATION"

	maxToolMemoryEntries = 64
	maxToolMemoryTags    = 16
	maxWisdomTextBytes   = 512
	maxToolMemoryTTL     = int64(365 * 24 * time.Hour / time.Millisecond)
)

// IdentityToolExperience is a compact evidence pointer and distilled lesson.
// It contains no raw prompt, input, output, hidden reasoning, or tool payload.
type IdentityToolExperience struct {
	Schema                        string    `json:"schema"`
	MemoryID                      string    `json:"memory_id"`
	ExperienceID                  string    `json:"experience_id"`
	TargetAnsweringIdentitySHA256 string    `json:"target_answering_identity_sha256"`
	ToolID                        string    `json:"tool_id"`
	ToolVersion                   string    `json:"tool_version"`
	TaskClass                     string    `json:"task_class"`
	UseTags                       []string  `json:"use_tags"`
	InputArtifactSHA256           string    `json:"input_artifact_sha256"`
	OutputArtifactSHA256          string    `json:"output_artifact_sha256"`
	ToolReceiptSHA256             string    `json:"tool_receipt_sha256"`
	IndependentVerificationSHA256 string    `json:"independent_verification_sha256"`
	Outcome                       string    `json:"outcome"`
	VerificationVerdict           string    `json:"verification_verdict"`
	WisdomClass                   string    `json:"wisdom_class"`
	Wisdom                        string    `json:"wisdom"`
	ObservedAt                    string    `json:"observed_at"`
	TTLMillis                     int64     `json:"ttl_millis"`
	ContentClass                  string    `json:"content_class"`
	RawContentIncluded            bool      `json:"raw_content_included"`
	HiddenReasoningIncluded       bool      `json:"hidden_reasoning_included"`
	SecretsIncluded               bool      `json:"secrets_included"`
	Authority                     Authority `json:"authority"`
	ExperienceSHA256              string    `json:"experience_sha256,omitempty"`
}

type IdentityToolMemoryShard struct {
	Schema                        string                   `json:"schema"`
	State                         string                   `json:"state"`
	MemoryID                      string                   `json:"memory_id"`
	TargetAnsweringIdentitySHA256 string                   `json:"target_answering_identity_sha256"`
	ToolID                        string                   `json:"tool_id"`
	ToolVersion                   string                   `json:"tool_version"`
	Generation                    int                      `json:"generation"`
	PreviousShardSHA256           string                   `json:"previous_shard_sha256,omitempty"`
	Entries                       []IdentityToolExperience `json:"entries"`
	Notices                       []string                 `json:"notices"`
	Authority                     Authority                `json:"authority"`
	ShardSHA256                   string                   `json:"shard_sha256,omitempty"`
}

type IdentityWisdomQuery struct {
	Schema                        string    `json:"schema"`
	QueryID                       string    `json:"query_id"`
	TargetAnsweringIdentitySHA256 string    `json:"target_answering_identity_sha256"`
	ToolID                        string    `json:"tool_id"`
	ToolVersion                   string    `json:"tool_version"`
	TaskClass                     string    `json:"task_class"`
	UseTags                       []string  `json:"use_tags"`
	AsOf                          string    `json:"as_of"`
	MaxEntries                    int       `json:"max_entries"`
	Authority                     Authority `json:"authority"`
}

type IdentityWisdomEntry struct {
	ExperienceID                  string   `json:"experience_id"`
	ExperienceSHA256              string   `json:"experience_sha256"`
	TaskClass                     string   `json:"task_class"`
	MatchedUseTags                []string `json:"matched_use_tags"`
	Outcome                       string   `json:"outcome"`
	VerificationVerdict           string   `json:"verification_verdict"`
	WisdomClass                   string   `json:"wisdom_class"`
	Wisdom                        string   `json:"wisdom"`
	ToolReceiptSHA256             string   `json:"tool_receipt_sha256"`
	IndependentVerificationSHA256 string   `json:"independent_verification_sha256"`
	ObservedAt                    string   `json:"observed_at"`
	AgeMillis                     int64    `json:"age_millis"`
	TTLMillis                     int64    `json:"ttl_millis"`
	Freshness                     string   `json:"freshness"`
}

// IdentityWisdomView is the only object intended for model context. It
// contains the bounded matching slice, never the complete cross-tool memory.
type IdentityWisdomView struct {
	Schema                        string                `json:"schema"`
	State                         string                `json:"state"`
	Query                         IdentityWisdomQuery   `json:"query"`
	QuerySHA256                   string                `json:"query_sha256"`
	SourceShardSHA256             string                `json:"source_shard_sha256"`
	TargetAnsweringIdentitySHA256 string                `json:"target_answering_identity_sha256"`
	ToolID                        string                `json:"tool_id"`
	ToolVersion                   string                `json:"tool_version"`
	Selected                      []IdentityWisdomEntry `json:"selected,omitempty"`
	OmittedRelevantCount          int                   `json:"omitted_relevant_count"`
	Notices                       []string              `json:"notices"`
	Authority                     Authority             `json:"authority"`
	ViewSHA256                    string                `json:"view_sha256,omitempty"`
}

func SealIdentityToolExperience(experience IdentityToolExperience) (IdentityToolExperience, error) {
	canonical, err := canonicalizeIdentityToolExperience(experience, false)
	if err != nil {
		return IdentityToolExperience{}, err
	}
	canonical.ExperienceSHA256, err = identityToolExperienceDigest(canonical)
	if err != nil {
		return IdentityToolExperience{}, err
	}
	return canonical, nil
}

func StartIdentityToolMemory(experience IdentityToolExperience) (IdentityToolMemoryShard, error) {
	canonical, err := canonicalizeIdentityToolExperience(experience, true)
	if err != nil {
		return IdentityToolMemoryShard{}, err
	}
	shard := IdentityToolMemoryShard{
		Schema: IdentityToolMemorySchema, State: ToolMemoryReady,
		MemoryID: canonical.MemoryID, TargetAnsweringIdentitySHA256: canonical.TargetAnsweringIdentitySHA256,
		ToolID: canonical.ToolID, ToolVersion: canonical.ToolVersion, Generation: 1,
		Entries: []IdentityToolExperience{canonical},
		Notices: []string{
			"this shard contains one exact identity-and-tool scope; other tools and identities require separate shards",
			"entries contain public-safe distillations and evidence digests only, not raw prompts, payloads, outputs, hidden reasoning, or secrets",
			"independent verification supports a bounded lesson but does not prove general competence, safety, promotion, or CANON status",
		},
		Authority: Authority{},
	}
	shard.ShardSHA256, err = identityToolMemoryShardDigest(shard)
	if err != nil {
		return IdentityToolMemoryShard{}, err
	}
	if err := shard.Validate(); err != nil {
		return IdentityToolMemoryShard{}, fmt.Errorf("generated identity tool memory shard: %w", err)
	}
	return shard, nil
}

func GrowIdentityToolMemory(current IdentityToolMemoryShard, experience IdentityToolExperience) (IdentityToolMemoryShard, error) {
	if err := current.Validate(); err != nil {
		return IdentityToolMemoryShard{}, fmt.Errorf("current identity tool memory shard: %w", err)
	}
	canonical, err := canonicalizeIdentityToolExperience(experience, true)
	if err != nil {
		return IdentityToolMemoryShard{}, err
	}
	if canonical.MemoryID != current.MemoryID || canonical.TargetAnsweringIdentitySHA256 != current.TargetAnsweringIdentitySHA256 || canonical.ToolID != current.ToolID || canonical.ToolVersion != current.ToolVersion {
		return IdentityToolMemoryShard{}, errors.New("tool experience does not match the exact memory, answering identity, tool, and version scope")
	}
	if len(current.Entries) >= maxToolMemoryEntries {
		return IdentityToolMemoryShard{}, fmt.Errorf("identity tool memory shard reached its %d-entry bound; seal a reviewed compacted successor", maxToolMemoryEntries)
	}
	for _, entry := range current.Entries {
		if entry.ExperienceID == canonical.ExperienceID || entry.ExperienceSHA256 == canonical.ExperienceSHA256 {
			return IdentityToolMemoryShard{}, errors.New("identity tool memory refuses a duplicate experience")
		}
	}
	lastObserved, _ := time.Parse(time.RFC3339Nano, current.Entries[len(current.Entries)-1].ObservedAt)
	observed, _ := time.Parse(time.RFC3339Nano, canonical.ObservedAt)
	if observed.Before(lastObserved) {
		return IdentityToolMemoryShard{}, errors.New("identity tool experience predates the latest append-only entry")
	}
	next := current
	next.Generation++
	next.PreviousShardSHA256 = current.ShardSHA256
	next.Entries = append(append([]IdentityToolExperience(nil), current.Entries...), canonical)
	next.ShardSHA256 = ""
	next.ShardSHA256, err = identityToolMemoryShardDigest(next)
	if err != nil {
		return IdentityToolMemoryShard{}, err
	}
	if err := next.Validate(); err != nil {
		return IdentityToolMemoryShard{}, fmt.Errorf("grown identity tool memory shard: %w", err)
	}
	return next, nil
}

func RecallIdentityToolWisdom(shard IdentityToolMemoryShard, query IdentityWisdomQuery) (IdentityWisdomView, error) {
	if err := shard.Validate(); err != nil {
		return IdentityWisdomView{}, fmt.Errorf("identity tool memory shard: %w", err)
	}
	canonical, err := canonicalizeIdentityWisdomQuery(query)
	if err != nil {
		return IdentityWisdomView{}, err
	}
	if canonical.TargetAnsweringIdentitySHA256 != shard.TargetAnsweringIdentitySHA256 || canonical.ToolID != shard.ToolID || canonical.ToolVersion != shard.ToolVersion {
		return IdentityWisdomView{}, errors.New("wisdom query does not match the exact answering identity, tool, and version shard")
	}
	queryDigest, err := digestJSON(canonical, "identity wisdom query")
	if err != nil {
		return IdentityWisdomView{}, err
	}
	asOf, _ := time.Parse(time.RFC3339Nano, canonical.AsOf)
	relevant := make([]IdentityWisdomEntry, 0)
	for _, entry := range shard.Entries {
		if entry.TaskClass != canonical.TaskClass {
			continue
		}
		matched := intersectSortedStrings(entry.UseTags, canonical.UseTags)
		if len(matched) == 0 {
			continue
		}
		observed, _ := time.Parse(time.RFC3339Nano, entry.ObservedAt)
		age := asOf.Sub(observed)
		if age < 0 || age.Milliseconds() > entry.TTLMillis {
			continue
		}
		relevant = append(relevant, IdentityWisdomEntry{
			ExperienceID: entry.ExperienceID, ExperienceSHA256: entry.ExperienceSHA256,
			TaskClass: entry.TaskClass, MatchedUseTags: matched, Outcome: entry.Outcome,
			VerificationVerdict: entry.VerificationVerdict, WisdomClass: entry.WisdomClass, Wisdom: entry.Wisdom,
			ToolReceiptSHA256:             entry.ToolReceiptSHA256,
			IndependentVerificationSHA256: entry.IndependentVerificationSHA256,
			ObservedAt:                    entry.ObservedAt, AgeMillis: age.Milliseconds(), TTLMillis: entry.TTLMillis, Freshness: "FRESH",
		})
	}
	sort.Slice(relevant, func(i, j int) bool {
		if relevant[i].ObservedAt != relevant[j].ObservedAt {
			return relevant[i].ObservedAt > relevant[j].ObservedAt
		}
		return relevant[i].ExperienceID < relevant[j].ExperienceID
	})
	view := IdentityWisdomView{
		Schema: IdentityWisdomViewSchema, State: ToolWisdomReady,
		Query: canonical, QuerySHA256: queryDigest, SourceShardSHA256: shard.ShardSHA256,
		TargetAnsweringIdentitySHA256: shard.TargetAnsweringIdentitySHA256,
		ToolID:                        shard.ToolID, ToolVersion: shard.ToolVersion,
		Notices: []string{
			"the view contains only fresh entries matching the exact answering identity, tool version, task class, and at least one declared use tag",
			"the model receives this bounded selected slice rather than the complete memory shard or a cross-tool memory blob",
			"wisdom entries are evidence-bound guidance, not hidden model-weight updates, general competence, permission, promotion, or CANON",
		},
		Authority: Authority{},
	}
	if len(relevant) == 0 {
		view.State = ToolWisdomNoRelevantHold
	} else {
		limit := canonical.MaxEntries
		if len(relevant) > limit {
			view.OmittedRelevantCount = len(relevant) - limit
			relevant = relevant[:limit]
		}
		view.Selected = relevant
	}
	view.ViewSHA256, err = identityWisdomViewDigest(view)
	if err != nil {
		return IdentityWisdomView{}, err
	}
	if err := view.Validate(); err != nil {
		return IdentityWisdomView{}, fmt.Errorf("generated identity wisdom view: %w", err)
	}
	return view, nil
}

func (shard IdentityToolMemoryShard) Validate() error {
	if shard.Schema != IdentityToolMemorySchema || shard.State != ToolMemoryReady {
		return errors.New("identity tool memory shard has unsupported schema or state")
	}
	for name, value := range map[string]string{"memory_id": shard.MemoryID, "tool_id": shard.ToolID, "tool_version": shard.ToolVersion} {
		if err := requireTrimmed("identity tool memory "+name, value); err != nil {
			return err
		}
	}
	if err := validateSHA256("identity tool memory target_answering_identity_sha256", shard.TargetAnsweringIdentitySHA256); err != nil {
		return err
	}
	if shard.Generation < 1 || len(shard.Entries) < 1 || len(shard.Entries) > maxToolMemoryEntries {
		return errors.New("identity tool memory generation and entry count are outside bounds")
	}
	if shard.Generation != len(shard.Entries) {
		return errors.New("identity tool memory v0.1 generation must equal its append-only entry count")
	}
	if shard.Generation == 1 && shard.PreviousShardSHA256 != "" {
		return errors.New("initial identity tool memory shard must not name a previous shard")
	}
	if shard.Generation > 1 {
		if err := validateSHA256("identity tool memory previous_shard_sha256", shard.PreviousShardSHA256); err != nil {
			return err
		}
	}
	seen := map[string]struct{}{}
	var previous time.Time
	for i, entry := range shard.Entries {
		canonical, err := canonicalizeIdentityToolExperience(entry, true)
		if err != nil {
			return fmt.Errorf("identity tool memory entry %d: %w", i, err)
		}
		if canonical.MemoryID != shard.MemoryID || canonical.TargetAnsweringIdentitySHA256 != shard.TargetAnsweringIdentitySHA256 || canonical.ToolID != shard.ToolID || canonical.ToolVersion != shard.ToolVersion {
			return fmt.Errorf("identity tool memory entry %d is outside shard scope", i)
		}
		if _, exists := seen[canonical.ExperienceID]; exists {
			return fmt.Errorf("duplicate identity tool experience %q", canonical.ExperienceID)
		}
		seen[canonical.ExperienceID] = struct{}{}
		observed, _ := time.Parse(time.RFC3339Nano, canonical.ObservedAt)
		if i > 0 && observed.Before(previous) {
			return errors.New("identity tool memory entries are not append-only by observation time")
		}
		previous = observed
	}
	if len(shard.Notices) < 3 || !shard.Authority.closed() {
		return errors.New("identity tool memory requires boundary notices and closed authority")
	}
	if err := validateSHA256("identity tool memory shard_sha256", shard.ShardSHA256); err != nil {
		return err
	}
	expected, err := identityToolMemoryShardDigest(shard)
	if err != nil {
		return err
	}
	if expected != shard.ShardSHA256 {
		return errors.New("identity tool memory shard digest mismatch")
	}
	return nil
}

func (view IdentityWisdomView) Validate() error {
	if view.Schema != IdentityWisdomViewSchema || !oneOf(view.State, ToolWisdomReady, ToolWisdomNoRelevantHold) {
		return errors.New("identity wisdom view has unsupported schema or state")
	}
	query, err := canonicalizeIdentityWisdomQuery(view.Query)
	if err != nil {
		return err
	}
	queryDigest, err := digestJSON(query, "identity wisdom query")
	if err != nil {
		return err
	}
	if queryDigest != view.QuerySHA256 || view.TargetAnsweringIdentitySHA256 != query.TargetAnsweringIdentitySHA256 || view.ToolID != query.ToolID || view.ToolVersion != query.ToolVersion {
		return errors.New("identity wisdom view query binding mismatch")
	}
	for name, digest := range map[string]string{"source_shard_sha256": view.SourceShardSHA256, "view_sha256": view.ViewSHA256} {
		if err := validateSHA256("identity wisdom view "+name, digest); err != nil {
			return err
		}
	}
	if len(view.Selected) > query.MaxEntries || view.OmittedRelevantCount < 0 {
		return errors.New("identity wisdom view selection is outside query bounds")
	}
	if view.State == ToolWisdomReady && len(view.Selected) == 0 {
		return errors.New("ready identity wisdom view requires a selected entry")
	}
	if view.State == ToolWisdomNoRelevantHold && (len(view.Selected) != 0 || view.OmittedRelevantCount != 0) {
		return errors.New("held identity wisdom view must not select or omit entries")
	}
	for i, entry := range view.Selected {
		if err := validateSHA256(fmt.Sprintf("identity wisdom entry %d experience_sha256", i), entry.ExperienceSHA256); err != nil {
			return err
		}
		if err := validateSHA256(fmt.Sprintf("identity wisdom entry %d tool_receipt_sha256", i), entry.ToolReceiptSHA256); err != nil {
			return err
		}
		if err := validateSHA256(fmt.Sprintf("identity wisdom entry %d independent_verification_sha256", i), entry.IndependentVerificationSHA256); err != nil {
			return err
		}
		matched, err := canonicalToolMemoryTags(entry.MatchedUseTags)
		if err != nil || !stringSlicesEqual(matched, entry.MatchedUseTags) {
			return fmt.Errorf("identity wisdom entry %d has invalid matched use tags", i)
		}
		for _, tag := range matched {
			if !containsString(query.UseTags, tag) {
				return fmt.Errorf("identity wisdom entry %d contains a tag outside the query", i)
			}
		}
		observed, err := time.Parse(time.RFC3339Nano, entry.ObservedAt)
		if err != nil {
			return fmt.Errorf("parse identity wisdom entry %d observed_at: %w", i, err)
		}
		asOf, _ := time.Parse(time.RFC3339Nano, query.AsOf)
		if entry.TaskClass != query.TaskClass || entry.AgeMillis != asOf.Sub(observed).Milliseconds() || entry.AgeMillis < 0 || entry.TTLMillis < 1 || entry.TTLMillis > maxToolMemoryTTL || entry.AgeMillis > entry.TTLMillis || entry.Freshness != "FRESH" || strings.TrimSpace(entry.Wisdom) == "" {
			return fmt.Errorf("identity wisdom entry %d is incomplete", i)
		}
		if entry.Outcome == ToolOutcomeSuccess {
			if entry.VerificationVerdict != ToolVerificationSuccess || entry.WisdomClass != ToolWisdomReuse {
				return fmt.Errorf("identity wisdom entry %d has inconsistent success evidence", i)
			}
		} else if entry.Outcome == ToolOutcomeFailure {
			if entry.VerificationVerdict != ToolVerificationFailure || entry.WisdomClass != ToolWisdomAvoid {
				return fmt.Errorf("identity wisdom entry %d has inconsistent failure evidence", i)
			}
		} else {
			return fmt.Errorf("identity wisdom entry %d has unsupported outcome", i)
		}
	}
	if len(view.Notices) < 3 || !view.Authority.closed() {
		return errors.New("identity wisdom view requires boundary notices and closed authority")
	}
	expected, err := identityWisdomViewDigest(view)
	if err != nil {
		return err
	}
	if expected != view.ViewSHA256 {
		return errors.New("identity wisdom view digest mismatch")
	}
	return nil
}

func canonicalizeIdentityToolExperience(experience IdentityToolExperience, requireDigest bool) (IdentityToolExperience, error) {
	for name, value := range map[string]string{
		"memory_id": experience.MemoryID, "experience_id": experience.ExperienceID, "tool_id": experience.ToolID,
		"tool_version": experience.ToolVersion, "task_class": experience.TaskClass,
	} {
		if err := requireTrimmed("identity tool experience "+name, value); err != nil {
			return IdentityToolExperience{}, err
		}
	}
	for name, digest := range map[string]string{
		"target_answering_identity_sha256": experience.TargetAnsweringIdentitySHA256,
		"input_artifact_sha256":            experience.InputArtifactSHA256,
		"output_artifact_sha256":           experience.OutputArtifactSHA256,
		"tool_receipt_sha256":              experience.ToolReceiptSHA256,
		"independent_verification_sha256":  experience.IndependentVerificationSHA256,
	} {
		if err := validateSHA256("identity tool experience "+name, digest); err != nil {
			return IdentityToolExperience{}, err
		}
	}
	tags, err := canonicalToolMemoryTags(experience.UseTags)
	if err != nil {
		return IdentityToolExperience{}, err
	}
	experience.UseTags = tags
	if len(experience.Wisdom) < 1 || len([]byte(experience.Wisdom)) > maxWisdomTextBytes || experience.Wisdom != strings.TrimSpace(experience.Wisdom) || strings.ContainsAny(experience.Wisdom, "\r\n\x00") {
		return IdentityToolExperience{}, fmt.Errorf("identity tool wisdom must be trimmed and between 1 and %d bytes", maxWisdomTextBytes)
	}
	if experience.Outcome == ToolOutcomeSuccess {
		if experience.VerificationVerdict != ToolVerificationSuccess || experience.WisdomClass != ToolWisdomReuse {
			return IdentityToolExperience{}, errors.New("successful tool experience requires independently verified success and REUSE wisdom")
		}
	} else if experience.Outcome == ToolOutcomeFailure {
		if experience.VerificationVerdict != ToolVerificationFailure || experience.WisdomClass != ToolWisdomAvoid {
			return IdentityToolExperience{}, errors.New("failed tool experience requires independently verified failure and AVOID wisdom")
		}
	} else {
		return IdentityToolExperience{}, fmt.Errorf("unsupported identity tool outcome %q", experience.Outcome)
	}
	if _, err := time.Parse(time.RFC3339Nano, experience.ObservedAt); err != nil {
		return IdentityToolExperience{}, fmt.Errorf("parse identity tool experience observed_at: %w", err)
	}
	if experience.TTLMillis < 1 || experience.TTLMillis > maxToolMemoryTTL {
		return IdentityToolExperience{}, errors.New("identity tool experience ttl_millis is outside bounds")
	}
	if experience.ContentClass != ToolMemoryContentPublicSafe || experience.RawContentIncluded || experience.HiddenReasoningIncluded || experience.SecretsIncluded {
		return IdentityToolExperience{}, errors.New("identity tool memory accepts only public-safe distillation without raw content, hidden reasoning, or secrets")
	}
	if !experience.Authority.closed() {
		return IdentityToolExperience{}, errors.New("identity tool experience must carry closed authority")
	}
	if requireDigest {
		if err := validateSHA256("identity tool experience experience_sha256", experience.ExperienceSHA256); err != nil {
			return IdentityToolExperience{}, err
		}
		expected, err := identityToolExperienceDigest(experience)
		if err != nil {
			return IdentityToolExperience{}, err
		}
		if expected != experience.ExperienceSHA256 {
			return IdentityToolExperience{}, errors.New("identity tool experience digest mismatch")
		}
	} else {
		experience.ExperienceSHA256 = ""
	}
	return experience, nil
}

func canonicalizeIdentityWisdomQuery(query IdentityWisdomQuery) (IdentityWisdomQuery, error) {
	if query.Schema != IdentityWisdomQuerySchema {
		return IdentityWisdomQuery{}, fmt.Errorf("identity wisdom query schema must be %q", IdentityWisdomQuerySchema)
	}
	for name, value := range map[string]string{
		"query_id": query.QueryID, "tool_id": query.ToolID, "tool_version": query.ToolVersion, "task_class": query.TaskClass,
	} {
		if err := requireTrimmed("identity wisdom query "+name, value); err != nil {
			return IdentityWisdomQuery{}, err
		}
	}
	if err := validateSHA256("identity wisdom query target_answering_identity_sha256", query.TargetAnsweringIdentitySHA256); err != nil {
		return IdentityWisdomQuery{}, err
	}
	tags, err := canonicalToolMemoryTags(query.UseTags)
	if err != nil {
		return IdentityWisdomQuery{}, err
	}
	query.UseTags = tags
	if _, err := time.Parse(time.RFC3339Nano, query.AsOf); err != nil {
		return IdentityWisdomQuery{}, fmt.Errorf("parse identity wisdom query as_of: %w", err)
	}
	if query.MaxEntries < 1 || query.MaxEntries > 8 {
		return IdentityWisdomQuery{}, errors.New("identity wisdom query max_entries must be between 1 and 8")
	}
	if !query.Authority.closed() {
		return IdentityWisdomQuery{}, errors.New("identity wisdom query must carry closed authority")
	}
	return query, nil
}

func canonicalToolMemoryTags(tags []string) ([]string, error) {
	if len(tags) < 1 || len(tags) > maxToolMemoryTags {
		return nil, fmt.Errorf("identity tool memory requires between 1 and %d use tags", maxToolMemoryTags)
	}
	canonical, err := canonicalNameSet("identity tool memory use_tags", tags)
	if err != nil {
		return nil, err
	}
	return canonical, nil
}

func intersectSortedStrings(left, right []string) []string {
	result := make([]string, 0)
	i, j := 0, 0
	for i < len(left) && j < len(right) {
		switch {
		case left[i] == right[j]:
			result = append(result, left[i])
			i++
			j++
		case left[i] < right[j]:
			i++
		default:
			j++
		}
	}
	return result
}

func identityToolExperienceDigest(experience IdentityToolExperience) (string, error) {
	experience.ExperienceSHA256 = ""
	return digestJSON(experience, "identity tool experience")
}

func identityToolMemoryShardDigest(shard IdentityToolMemoryShard) (string, error) {
	shard.ShardSHA256 = ""
	return digestJSON(shard, "identity tool memory shard")
}

func identityWisdomViewDigest(view IdentityWisdomView) (string, error) {
	view.ViewSHA256 = ""
	return digestJSON(view, "identity wisdom view")
}
