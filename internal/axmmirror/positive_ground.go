package axmmirror

import (
	"bufio"
	"bytes"
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
	MirrorGroundRecordSchema     = "axm.waldo.mirror-ground-record/v0.39"
	MirrorGroundProjectionSchema = "axm.waldo.mirror-ground-training-projection/v0.39"

	MirrorGroundSyntheticSeed          = "SYNTHETIC_SEED"
	MirrorGroundObservedChat           = "OBSERVED_CHAT"
	MirrorGroundObservedExecutionTrace = "OBSERVED_EXECUTION_TRACE"

	MirrorGroundCurated      = "CURATED_SYNTHETIC"
	MirrorGroundHelpful      = MirrorExperienceHelpful
	MirrorGroundCorrected    = MirrorExperienceCorrected
	MirrorGroundHarmful      = MirrorExperienceHarmful
	MirrorGroundInconclusive = MirrorExperienceInconclusive
	MirrorGroundUnreviewed   = "UNREVIEWED"

	MirrorGroundPositiveTarget = "POSITIVE_TARGET"
	MirrorGroundMemoryOnly     = "MEMORY_ONLY"
	MirrorGroundReviewRequired = "REVIEW_REQUIRED"

	MaxMirrorGroundRecords = 100_000
)

// MirrorGroundGenerator makes the origin of synthetic examples explicit.
// Synthetic curriculum is useful training input, but it is not an observed
// interaction and must not be allowed to acquire that label later.
type MirrorGroundGenerator struct {
	Model       string `json:"model"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type MirrorGroundMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MirrorGroundRecord is the review boundary between retained experience and a
// model-training target. The source class, evidence signal, and disposition
// remain separate so a raw log never becomes a positive example merely because
// it was recorded.
type MirrorGroundRecord struct {
	Schema              string                 `json:"schema"`
	ID                  string                 `json:"id"`
	DataClass           string                 `json:"dataClass"`
	Synthetic           bool                   `json:"synthetic"`
	Generator           *MirrorGroundGenerator `json:"generator,omitempty"`
	SourceReceiptSHA256 string                 `json:"sourceReceiptSha256,omitempty"`
	ObservedAt          *time.Time             `json:"observedAt,omitempty"`
	CreatedAt           time.Time              `json:"createdAt"`
	RootIDs             []string               `json:"rootIds"`
	ChallengeKind       string                 `json:"challengeKind"`
	EvidenceSignal      string                 `json:"evidenceSignal"`
	TrainingDisposition string                 `json:"trainingDisposition"`
	Messages            []MirrorGroundMessage  `json:"messages"`
	Rationale           string                 `json:"rationale"`
	TrainingText        string                 `json:"text,omitempty"`
	Authority           Authority              `json:"authority"`
	RecordSHA256        string                 `json:"recordSha256"`
}

type MirrorGroundDataset struct {
	Records           []MirrorGroundRecord
	DatasetSHA256     string
	TrainingTargets   int
	MemoryOnly        int
	ReviewRequired    int
	Synthetic         int
	Observed          int
	DataClasses       map[string]int
	RootCoverage      map[string]int
	ChallengeCoverage map[string]int
}

// MirrorPositiveSeedRequirements distinguish a valid file from a curriculum
// with enough deliberate coverage to be called the positive starting seed.
type MirrorPositiveSeedRequirements struct {
	MinimumRecords        int
	MinimumRoots          int
	MinimumPerRoot        int
	MinimumChallengeKinds int
}

func DefaultMirrorPositiveSeedRequirements() MirrorPositiveSeedRequirements {
	return MirrorPositiveSeedRequirements{
		MinimumRecords:        24,
		MinimumRoots:          8,
		MinimumPerRoot:        3,
		MinimumChallengeKinds: 8,
	}
}

func SealMirrorGroundRecord(record *MirrorGroundRecord) error {
	if record == nil {
		return errors.New("Mirror ground record is required")
	}
	record.RecordSHA256 = ""
	digest, err := record.digest()
	if err != nil {
		return err
	}
	record.RecordSHA256 = digest
	return record.Validate()
}

func (record MirrorGroundRecord) Validate() error {
	if record.Schema != MirrorGroundRecordSchema {
		return fmt.Errorf("schema must be %q", MirrorGroundRecordSchema)
	}
	if strings.TrimSpace(record.ID) == "" || len(record.ID) > 128 || strings.ContainsAny(record.ID, "\r\n\t") {
		return errors.New("id is required, single-line, and limited to 128 bytes")
	}
	if record.CreatedAt.IsZero() {
		return errors.New("createdAt is required")
	}
	if !record.Authority.closed() {
		return errors.New("ground records carry no execution, training, promotion, canon, or world-action authority")
	}
	if len(record.RootIDs) == 0 || len(record.RootIDs) > 16 || !sort.StringsAreSorted(record.RootIDs) {
		return errors.New("rootIds must contain 1..16 sorted identifiers")
	}
	previous := ""
	for _, root := range record.RootIDs {
		if strings.TrimSpace(root) == "" || len(root) > 128 || strings.ContainsAny(root, "\r\n\t") || root == previous {
			return errors.New("rootIds must be non-empty, unique, single-line, and limited to 128 bytes")
		}
		previous = root
	}
	if strings.TrimSpace(record.ChallengeKind) == "" || len(record.ChallengeKind) > 128 || strings.ContainsAny(record.ChallengeKind, "\r\n\t") {
		return errors.New("challengeKind is required, single-line, and limited to 128 bytes")
	}
	if strings.TrimSpace(record.Rationale) == "" || len(record.Rationale) > MaxMirrorEscalationTextBytes {
		return fmt.Errorf("rationale is required and limited to %d bytes", MaxMirrorEscalationTextBytes)
	}
	if len(record.Messages) != 2 || record.Messages[0].Role != "user" || record.Messages[1].Role != "assistant" {
		return errors.New("messages must contain exactly one user turn followed by one assistant turn")
	}
	for position, message := range record.Messages {
		if strings.TrimSpace(message.Content) == "" || len(message.Content) > MaxMirrorEscalationTextBytes {
			return fmt.Errorf("messages[%d].content is required and limited to %d bytes", position, MaxMirrorEscalationTextBytes)
		}
	}

	switch record.DataClass {
	case MirrorGroundSyntheticSeed:
		if !record.Synthetic || record.Generator == nil || record.ObservedAt != nil || record.SourceReceiptSHA256 != "" {
			return errors.New("SYNTHETIC_SEED requires visible generator provenance and cannot claim observation evidence")
		}
		if strings.TrimSpace(record.Generator.Model) == "" || strings.TrimSpace(record.Generator.Version) == "" || strings.TrimSpace(record.Generator.Description) == "" {
			return errors.New("synthetic generator model, version, and description are required")
		}
		if record.EvidenceSignal != MirrorGroundCurated || record.TrainingDisposition != MirrorGroundPositiveTarget {
			return errors.New("SYNTHETIC_SEED must be explicitly curated as a positive target")
		}
	case MirrorGroundObservedChat, MirrorGroundObservedExecutionTrace:
		if record.Synthetic || record.Generator != nil || record.ObservedAt == nil || record.ObservedAt.IsZero() || !validMirrorSHA(record.SourceReceiptSHA256) {
			return errors.New("observed ground data requires a timestamp and source receipt and cannot claim synthetic provenance")
		}
		if record.ObservedAt.After(record.CreatedAt) {
			return errors.New("observedAt cannot be after createdAt")
		}
	default:
		return fmt.Errorf("unsupported dataClass %q", record.DataClass)
	}

	switch record.TrainingDisposition {
	case MirrorGroundPositiveTarget:
		if record.EvidenceSignal != MirrorGroundCurated && record.EvidenceSignal != MirrorGroundHelpful && record.EvidenceSignal != MirrorGroundCorrected {
			return errors.New("positive targets require curated, helpful, or corrected evidence")
		}
		expected := "User: " + record.Messages[0].Content + "\n\nAssistant: " + record.Messages[1].Content
		if record.TrainingText != expected {
			return errors.New("positive target text does not bind the structured user and assistant turns")
		}
	case MirrorGroundMemoryOnly:
		if record.EvidenceSignal != MirrorGroundHarmful && record.EvidenceSignal != MirrorGroundInconclusive {
			return errors.New("memory-only records require harmful or inconclusive evidence")
		}
		if record.TrainingText != "" {
			return errors.New("memory-only records cannot contain a positive training projection")
		}
	case MirrorGroundReviewRequired:
		if record.EvidenceSignal != MirrorGroundUnreviewed {
			return errors.New("review-required records require unreviewed evidence")
		}
		if record.TrainingText != "" {
			return errors.New("review-required records cannot contain a training projection")
		}
	default:
		return fmt.Errorf("unsupported trainingDisposition %q", record.TrainingDisposition)
	}

	digest, err := record.digest()
	if err != nil {
		return err
	}
	if record.RecordSHA256 != digest {
		return errors.New("recordSha256 does not bind the complete ground record")
	}
	return nil
}

func (record MirrorGroundRecord) digest() (string, error) {
	copy := record
	copy.RecordSHA256 = ""
	payload, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	return digestMirrorText(string(payload)), nil
}

func (record MirrorGroundRecord) JSONLine() ([]byte, error) {
	if err := record.Validate(); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}

func LoadMirrorGroundDataset(reader io.Reader) (MirrorGroundDataset, error) {
	if reader == nil {
		return MirrorGroundDataset{}, errors.New("Mirror ground reader is required")
	}
	dataset := MirrorGroundDataset{
		DataClasses:       map[string]int{},
		RootCoverage:      map[string]int{},
		ChallengeCoverage: map[string]int{},
	}
	seen := map[string]bool{}
	hash := sha256.New()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if len(dataset.Records) >= MaxMirrorGroundRecords {
			return MirrorGroundDataset{}, fmt.Errorf("Mirror ground dataset exceeds %d records", MaxMirrorGroundRecords)
		}
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		var record MirrorGroundRecord
		if err := decoder.Decode(&record); err != nil {
			return MirrorGroundDataset{}, fmt.Errorf("decode Mirror ground line %d: %w", lineNumber, err)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			return MirrorGroundDataset{}, fmt.Errorf("decode Mirror ground line %d: trailing JSON", lineNumber)
		}
		if err := record.Validate(); err != nil {
			return MirrorGroundDataset{}, fmt.Errorf("validate Mirror ground line %d: %w", lineNumber, err)
		}
		if seen[record.ID] {
			return MirrorGroundDataset{}, fmt.Errorf("duplicate Mirror ground id %q", record.ID)
		}
		seen[record.ID] = true
		canonical, err := record.JSONLine()
		if err != nil {
			return MirrorGroundDataset{}, err
		}
		_, _ = hash.Write(canonical)
		dataset.Records = append(dataset.Records, record)
		dataset.DataClasses[record.DataClass]++
		dataset.ChallengeCoverage[record.ChallengeKind]++
		for _, root := range record.RootIDs {
			dataset.RootCoverage[root]++
		}
		if record.Synthetic {
			dataset.Synthetic++
		} else {
			dataset.Observed++
		}
		switch record.TrainingDisposition {
		case MirrorGroundPositiveTarget:
			dataset.TrainingTargets++
		case MirrorGroundMemoryOnly:
			dataset.MemoryOnly++
		case MirrorGroundReviewRequired:
			dataset.ReviewRequired++
		}
	}
	if err := scanner.Err(); err != nil {
		return MirrorGroundDataset{}, fmt.Errorf("read Mirror ground dataset: %w", err)
	}
	if len(dataset.Records) == 0 {
		return MirrorGroundDataset{}, errors.New("Mirror ground dataset contains no records")
	}
	dataset.DatasetSHA256 = hex.EncodeToString(hash.Sum(nil))
	return dataset, nil
}

func ValidateMirrorPositiveSeed(dataset MirrorGroundDataset, requirements MirrorPositiveSeedRequirements) error {
	if requirements.MinimumRecords < 1 || requirements.MinimumRoots < 1 || requirements.MinimumPerRoot < 1 || requirements.MinimumChallengeKinds < 1 {
		return errors.New("positive-seed requirements must all be positive")
	}
	if len(dataset.Records) < requirements.MinimumRecords {
		return fmt.Errorf("positive seed has %d records; require at least %d", len(dataset.Records), requirements.MinimumRecords)
	}
	if dataset.TrainingTargets != len(dataset.Records) || dataset.Synthetic != len(dataset.Records) || dataset.Observed != 0 || len(dataset.DataClasses) != 1 || dataset.DataClasses[MirrorGroundSyntheticSeed] != len(dataset.Records) {
		return errors.New("positive seed must contain only synthetic, explicitly curated positive targets")
	}
	if len(dataset.RootCoverage) < requirements.MinimumRoots {
		return fmt.Errorf("positive seed covers %d roots; require at least %d", len(dataset.RootCoverage), requirements.MinimumRoots)
	}
	for root, count := range dataset.RootCoverage {
		if count < requirements.MinimumPerRoot {
			return fmt.Errorf("positive seed root %q has %d examples; require at least %d", root, count, requirements.MinimumPerRoot)
		}
	}
	if len(dataset.ChallengeCoverage) < requirements.MinimumChallengeKinds {
		return fmt.Errorf("positive seed covers %d challenge kinds; require at least %d", len(dataset.ChallengeCoverage), requirements.MinimumChallengeKinds)
	}
	return nil
}

type mirrorGroundTrainingProjection struct {
	Schema             string                `json:"schema"`
	ID                 string                `json:"id"`
	DataClass          string                `json:"data_class"`
	Synthetic          bool                  `json:"synthetic"`
	RootIDs            []string              `json:"root_ids"`
	ChallengeKind      string                `json:"challenge_kind"`
	EvidenceSignal     string                `json:"evidence_signal"`
	SourceRecordSHA256 string                `json:"source_record_sha256"`
	Messages           []MirrorGroundMessage `json:"messages"`
	Text               string                `json:"text"`
}

func (dataset MirrorGroundDataset) TrainingProjectionJSONL() ([]byte, error) {
	var output bytes.Buffer
	for _, record := range dataset.Records {
		if record.TrainingDisposition != MirrorGroundPositiveTarget {
			continue
		}
		projection := mirrorGroundTrainingProjection{
			Schema:             MirrorGroundProjectionSchema,
			ID:                 record.ID,
			DataClass:          record.DataClass,
			Synthetic:          record.Synthetic,
			RootIDs:            append([]string(nil), record.RootIDs...),
			ChallengeKind:      record.ChallengeKind,
			EvidenceSignal:     record.EvidenceSignal,
			SourceRecordSHA256: record.RecordSHA256,
			Messages:           append([]MirrorGroundMessage(nil), record.Messages...),
			Text:               record.TrainingText,
		}
		payload, err := json.Marshal(projection)
		if err != nil {
			return nil, err
		}
		output.Write(payload)
		output.WriteByte('\n')
	}
	if output.Len() == 0 {
		return nil, errors.New("Mirror ground dataset has no positive training targets")
	}
	return output.Bytes(), nil
}
