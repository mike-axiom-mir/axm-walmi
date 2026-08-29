package axmmirror

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	EvaluationComparisonSchema = "axm.waldo-witness.evaluation-comparison/v0.1"
	ContaminationReportSchema  = "axm.waldo-witness.contamination-report/v0.1"

	ContaminationStateClear        = "CLEAR"
	ContaminationStateHold         = "HOLD"
	ContaminationStateContaminated = "CONTAMINATED"
)

// EvidenceInventory is a declared identity surface, not raw training or
// evaluation content. Complete means complete for every populated dimension.
type EvidenceInventory struct {
	Complete     bool     `json:"complete"`
	RecordSHA256 []string `json:"record_sha256,omitempty"`
	TextSHA256   []string `json:"text_sha256,omitempty"`
	SourceGroups []string `json:"source_groups,omitempty"`
	CorpusPaths  []string `json:"corpus_paths,omitempty"`
}

// AuthorshipDeclaration remains a declaration even when an evidence document
// is attached. This organ does not turn it into independently verified truth.
type AuthorshipDeclaration struct {
	State          string `json:"state"`
	EvidenceSHA256 string `json:"evidence_sha256,omitempty"`
}

// EvaluationComparison declares the exact surfaces available for a bounded
// overlap comparison.
type EvaluationComparison struct {
	Schema     string                `json:"schema"`
	Training   EvidenceInventory     `json:"training"`
	Evaluation EvidenceInventory     `json:"evaluation"`
	Authorship AuthorshipDeclaration `json:"authorship"`
}

type ContaminationOverlap struct {
	Dimension string   `json:"dimension"`
	Values    []string `json:"values"`
}

type AuthorshipFinding struct {
	State          string `json:"state"`
	Basis          string `json:"basis"`
	EvidenceSHA256 string `json:"evidence_sha256,omitempty"`
}

// ContaminationReport keeps overlap and authorship claims separate. State
// answers only the held-out overlap question within the declared inventories.
type ContaminationReport struct {
	Schema             string                 `json:"schema"`
	State              string                 `json:"state"`
	ComparisonSHA256   string                 `json:"comparison_sha256"`
	ComparedDimensions []string               `json:"compared_dimensions,omitempty"`
	Overlaps           []ContaminationOverlap `json:"overlaps,omitempty"`
	Holds              []string               `json:"holds,omitempty"`
	Authorship         AuthorshipFinding      `json:"authorship"`
	Authority          Authority              `json:"authority"`
}

func (comparison EvaluationComparison) Validate() error {
	if comparison.Schema != EvaluationComparisonSchema {
		return fmt.Errorf("comparison schema must be %q", EvaluationComparisonSchema)
	}
	if err := validateInventory("training", comparison.Training); err != nil {
		return err
	}
	if err := validateInventory("evaluation", comparison.Evaluation); err != nil {
		return err
	}
	if !oneOf(comparison.Authorship.State, "outside-authored", "same-author", "unknown") {
		return fmt.Errorf("unsupported authorship state %q", comparison.Authorship.State)
	}
	if comparison.Authorship.EvidenceSHA256 != "" {
		if err := validateSHA256("authorship.evidence_sha256", comparison.Authorship.EvidenceSHA256); err != nil {
			return err
		}
	}
	return nil
}

// CheckContamination performs exact identity intersections only. Similarity,
// semantic leakage, and memorization require separate declared instruments.
func CheckContamination(comparison EvaluationComparison) (ContaminationReport, error) {
	if err := comparison.Validate(); err != nil {
		return ContaminationReport{}, err
	}
	comparisonDigest, err := canonicalComparisonDigest(comparison)
	if err != nil {
		return ContaminationReport{}, err
	}
	report := ContaminationReport{
		Schema:           ContaminationReportSchema,
		State:            ContaminationStateClear,
		ComparisonSHA256: comparisonDigest,
		Authority:        Authority{},
	}
	for _, dimension := range []struct {
		name       string
		training   []string
		evaluation []string
	}{
		{"record_sha256", comparison.Training.RecordSHA256, comparison.Evaluation.RecordSHA256},
		{"text_sha256", comparison.Training.TextSHA256, comparison.Evaluation.TextSHA256},
		{"source_groups", comparison.Training.SourceGroups, comparison.Evaluation.SourceGroups},
		{"corpus_paths", comparison.Training.CorpusPaths, comparison.Evaluation.CorpusPaths},
	} {
		if len(dimension.training) == 0 && len(dimension.evaluation) == 0 {
			continue
		}
		if len(dimension.training) == 0 || len(dimension.evaluation) == 0 {
			report.Holds = append(report.Holds, fmt.Sprintf("%s is not declared in both inventories", dimension.name))
			continue
		}
		report.ComparedDimensions = append(report.ComparedDimensions, dimension.name)
		values := intersection(dimension.training, dimension.evaluation)
		if len(values) > 0 {
			report.Overlaps = append(report.Overlaps, ContaminationOverlap{Dimension: dimension.name, Values: values})
		}
	}
	if len(report.Overlaps) > 0 {
		report.State = ContaminationStateContaminated
	} else {
		if !comparison.Training.Complete {
			report.Holds = append(report.Holds, "training identity inventory is incomplete")
		}
		if !comparison.Evaluation.Complete {
			report.Holds = append(report.Holds, "evaluation identity inventory is incomplete")
		}
		if len(report.ComparedDimensions) == 0 {
			report.Holds = append(report.Holds, "training and evaluation inventories have no comparable identity dimension")
		}
		if len(report.Holds) > 0 {
			report.State = ContaminationStateHold
		}
	}

	report.Authorship = authorshipFinding(comparison.Authorship)
	return report, nil
}

func canonicalComparisonDigest(comparison EvaluationComparison) (string, error) {
	canonical := comparison
	canonical.Training = cloneSortedInventory(comparison.Training)
	canonical.Evaluation = cloneSortedInventory(comparison.Evaluation)
	data, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("encode evaluation comparison: %w", err)
	}
	return digestBytes(data), nil
}

func cloneSortedInventory(inventory EvidenceInventory) EvidenceInventory {
	result := inventory
	result.RecordSHA256 = append([]string(nil), inventory.RecordSHA256...)
	result.TextSHA256 = append([]string(nil), inventory.TextSHA256...)
	result.SourceGroups = append([]string(nil), inventory.SourceGroups...)
	result.CorpusPaths = append([]string(nil), inventory.CorpusPaths...)
	sort.Strings(result.RecordSHA256)
	sort.Strings(result.TextSHA256)
	sort.Strings(result.SourceGroups)
	sort.Strings(result.CorpusPaths)
	return result
}

func validateInventory(name string, inventory EvidenceInventory) error {
	if len(inventory.RecordSHA256)+len(inventory.TextSHA256)+len(inventory.SourceGroups)+len(inventory.CorpusPaths) == 0 {
		return fmt.Errorf("%s inventory requires at least one declared identity", name)
	}
	for _, dimension := range []struct {
		name   string
		values []string
	}{
		{"record_sha256", inventory.RecordSHA256},
		{"text_sha256", inventory.TextSHA256},
	} {
		seen := map[string]struct{}{}
		for i, value := range dimension.values {
			if _, exists := seen[value]; exists {
				return fmt.Errorf("%s.%s has duplicate value %q", name, dimension.name, value)
			}
			seen[value] = struct{}{}
			if err := validateSHA256(fmt.Sprintf("%s.%s[%d]", name, dimension.name, i), value); err != nil {
				return err
			}
		}
	}
	seenGroups := map[string]struct{}{}
	for _, value := range inventory.SourceGroups {
		if strings.TrimSpace(value) == "" || value != strings.TrimSpace(value) {
			return fmt.Errorf("%s.source_groups values must be non-empty and trimmed", name)
		}
		if _, exists := seenGroups[value]; exists {
			return fmt.Errorf("%s.source_groups has duplicate value %q", name, value)
		}
		seenGroups[value] = struct{}{}
	}
	seenPaths := map[string]struct{}{}
	for i, value := range inventory.CorpusPaths {
		if err := validateRelativePath(fmt.Sprintf("%s.corpus_paths[%d]", name, i), value); err != nil {
			return err
		}
		if _, exists := seenPaths[value]; exists {
			return fmt.Errorf("%s.corpus_paths has duplicate value %q", name, value)
		}
		seenPaths[value] = struct{}{}
	}
	return nil
}

func authorshipFinding(declaration AuthorshipDeclaration) AuthorshipFinding {
	result := AuthorshipFinding{EvidenceSHA256: declaration.EvidenceSHA256}
	switch declaration.State {
	case "outside-authored":
		result.State = "RECORDED_ASSERTION"
		result.Basis = "evaluation is declared outside-authored; this organ has not independently verified authorship"
	case "same-author":
		result.State = "REFUTED_BY_DECLARATION"
		result.Basis = "evaluation is declared as having the same author"
	case "unknown":
		result.State = "UNKNOWN"
		result.Basis = "no authorship conclusion was declared"
	}
	if declaration.EvidenceSHA256 != "" {
		result.Basis += "; the declaration is linked to an evidence document by digest"
	}
	return result
}

func intersection(left, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	leftSet := make(map[string]struct{}, len(left))
	for _, value := range left {
		leftSet[value] = struct{}{}
	}
	var output []string
	for _, value := range right {
		if _, exists := leftSet[value]; exists {
			output = append(output, value)
		}
	}
	sort.Strings(output)
	return output
}
