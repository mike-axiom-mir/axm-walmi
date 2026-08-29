package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/openwaldo/waldo/internal/axmmirror"
)

const reviewedIntakeReceiptSchema = "axm.waldo.mirror-reviewed-intake-receipt/v0.41"

type stringListFlag []string

func (values *stringListFlag) String() string { return strings.Join(*values, ",") }
func (values *stringListFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("--root cannot be empty")
	}
	*values = append(*values, value)
	return nil
}

type reviewedIntakeReceipt struct {
	Schema                      string              `json:"schema"`
	Decision                    string              `json:"decision"`
	EpisodeID                   string              `json:"episodeId"`
	LearningRecordSHA256        string              `json:"learningRecordSha256"`
	ReviewSHA256                string              `json:"reviewSha256"`
	EvidenceClass               string              `json:"evidenceClass"`
	EvidenceReceiptSHA256       string              `json:"evidenceReceiptSha256"`
	ReviewedTrainingSHA256      string              `json:"reviewedTrainingSha256,omitempty"`
	GroundRecordSHA256          string              `json:"groundRecordSha256,omitempty"`
	ReviewedTrainingMutation    bool                `json:"reviewedTrainingMutation"`
	GroundRecordMutation        bool                `json:"groundRecordMutation"`
	MemoryDeletion              bool                `json:"memoryDeletion"`
	ModelWeightMutation         bool                `json:"modelWeightMutation"`
	HermesRuntimeMemoryMutation bool                `json:"hermesRuntimeMemoryMutation"`
	IdentityMutation            bool                `json:"identityMutation"`
	Authority                   axmmirror.Authority `json:"authority"`
	Timestamp                   time.Time           `json:"timestamp"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("waldo-mirror-review", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var roots stringListFlag
	var reviewedTo, groundTo, dataClass, challenge string
	var jsonOutput bool
	flags.Var(&roots, "root", "ground root identifier; repeat for multiple roots")
	flags.StringVar(&reviewedTo, "reviewed-to", "", "write the approved evidence-reviewed training record to a new private JSONL file")
	flags.StringVar(&groundTo, "ground-to", "", "also write a v0.39 observed-ground record to a new private JSONL file")
	flags.StringVar(&dataClass, "data-class", axmmirror.MirrorGroundObservedChat, "ground source class: OBSERVED_CHAT or OBSERVED_EXECUTION_TRACE")
	flags.StringVar(&challenge, "challenge", "", "challenge kind recorded when --ground-to is used")
	flags.BoolVar(&jsonOutput, "json", false, "emit structured JSON receipt")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 2 {
		return errors.New("usage: waldo-mirror-review [flags] <learning.json> <review.json>")
	}
	learningPath := flags.Arg(0)
	reviewPath := flags.Arg(1)
	if err := requireDistinctPaths(learningPath, reviewPath, reviewedTo, groundTo); err != nil {
		return err
	}

	learning, err := loadLearningRecord(learningPath)
	if err != nil {
		return err
	}
	review, err := loadTrainingReview(reviewPath)
	if err != nil {
		return err
	}
	promotion, err := axmmirror.PromoteMirrorExperienceLearning(learning, review)
	if err != nil {
		return fmt.Errorf("review experience learning candidate: %w", err)
	}

	receipt := reviewedIntakeReceipt{
		Schema:                reviewedIntakeReceiptSchema,
		Decision:              review.Decision,
		EpisodeID:             learning.EpisodeID,
		LearningRecordSHA256:  learning.LearningRecordSHA256,
		ReviewSHA256:          review.ReviewSHA256,
		EvidenceClass:         review.EvidenceClass,
		EvidenceReceiptSHA256: review.EvidenceReceiptSHA256,
		Authority:             axmmirror.Authority{},
		Timestamp:             time.Now().UTC(),
	}
	if promotion == nil {
		return writeReceipt(stdout, receipt, jsonOutput)
	}
	if strings.TrimSpace(reviewedTo) == "" {
		return errors.New("approved review requires --reviewed-to so the evidence-reviewed promotion remains auditable")
	}

	promotionLine, err := promotion.JSONLine()
	if err != nil {
		return err
	}
	var groundLine []byte
	var groundRecord axmmirror.MirrorGroundRecord
	if groundTo != "" {
		if len(roots) == 0 {
			return errors.New("--ground-to requires at least one --root")
		}
		if strings.TrimSpace(challenge) == "" {
			return errors.New("--ground-to requires --challenge")
		}
		normalizedRoots, err := normalizeRoots(roots)
		if err != nil {
			return err
		}
		groundRecord, err = promotion.GroundRecord(dataClass, normalizedRoots, strings.TrimSpace(challenge), time.Now().UTC())
		if err != nil {
			return fmt.Errorf("build reviewed observed-ground record: %w", err)
		}
		groundLine, err = groundRecord.JSONLine()
		if err != nil {
			return err
		}
	}

	if err := writePrivateFileExclusive(reviewedTo, promotionLine); err != nil {
		return fmt.Errorf("write reviewed training record %s: %w", reviewedTo, err)
	}
	receipt.ReviewedTrainingMutation = true
	receipt.ReviewedTrainingSHA256 = promotion.PromotionSHA256
	if groundTo != "" {
		if err := writePrivateFileExclusive(groundTo, groundLine); err != nil {
			_ = os.Remove(reviewedTo)
			return fmt.Errorf("write reviewed ground record %s: %w", groundTo, err)
		}
		receipt.GroundRecordMutation = true
		receipt.GroundRecordSHA256 = groundRecord.RecordSHA256
	}
	return writeReceipt(stdout, receipt, jsonOutput)
}

func loadLearningRecord(path string) (axmmirror.MirrorExperienceLearningRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return axmmirror.MirrorExperienceLearningRecord{}, fmt.Errorf("open experience learning candidate %s: %w", path, err)
	}
	defer file.Close()
	record, err := axmmirror.LoadMirrorExperienceLearningRecord(file)
	if err != nil {
		return record, fmt.Errorf("load experience learning candidate %s: %w", path, err)
	}
	return record, nil
}

func loadTrainingReview(path string) (axmmirror.MirrorExperienceTrainingReview, error) {
	file, err := os.Open(path)
	if err != nil {
		return axmmirror.MirrorExperienceTrainingReview{}, fmt.Errorf("open experience training review %s: %w", path, err)
	}
	defer file.Close()
	review, err := axmmirror.LoadMirrorExperienceTrainingReview(file)
	if err != nil {
		return review, fmt.Errorf("load experience training review %s: %w", path, err)
	}
	return review, nil
}

func normalizeRoots(values []string) ([]string, error) {
	if len(values) > 16 {
		return nil, errors.New("at most 16 --root values are allowed")
	}
	seen := map[string]bool{}
	roots := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || len(value) > 128 || strings.ContainsAny(value, "\r\n\t") {
			return nil, errors.New("root identifiers must be non-empty, single-line, and limited to 128 bytes")
		}
		if seen[value] {
			return nil, fmt.Errorf("duplicate root %q", value)
		}
		seen[value] = true
		roots = append(roots, value)
	}
	sort.Strings(roots)
	return roots, nil
}

func requireDistinctPaths(paths ...string) error {
	seen := map[string]string{}
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolve path %s: %w", path, err)
		}
		absolute = filepath.Clean(absolute)
		if prior, exists := seen[absolute]; exists {
			return fmt.Errorf("review intake paths must be distinct; %s aliases %s", path, prior)
		}
		seen[absolute] = path
	}
	return nil
}

func writePrivateFileExclusive(path string, data []byte) (returnErr error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	complete := false
	defer func() {
		if !complete {
			_ = file.Close()
			_ = os.Remove(path)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	complete = true
	return nil
}

func writeReceipt(output io.Writer, receipt reviewedIntakeReceipt, jsonOutput bool) error {
	if jsonOutput {
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		return encoder.Encode(receipt)
	}
	fmt.Fprintf(output, "Experience: %s\n", receipt.EpisodeID)
	fmt.Fprintf(output, "Decision: %s\n", receipt.Decision)
	fmt.Fprintf(output, "Evidence: %s %s\n", receipt.EvidenceClass, receipt.EvidenceReceiptSHA256)
	if receipt.ReviewedTrainingMutation {
		fmt.Fprintf(output, "Reviewed training promotion: %s\n", receipt.ReviewedTrainingSHA256)
	} else {
		fmt.Fprintln(output, "Reviewed training promotion: NONE; experience remains memory/context only")
	}
	if receipt.GroundRecordMutation {
		fmt.Fprintf(output, "Observed-ground record: %s\n", receipt.GroundRecordSHA256)
	}
	fmt.Fprintln(output, "Model weights: UNCHANGED")
	fmt.Fprintln(output, "Authority: NONE")
	return nil
}
