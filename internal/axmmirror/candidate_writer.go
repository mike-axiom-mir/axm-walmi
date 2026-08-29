package axmmirror

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	CandidateWriteRequestSchema = "axm.waldo.candidate-write-request/v0.51"
	CandidateWriteReceiptSchema = "axm.waldo.candidate-write-receipt/v0.51"

	CandidateWriteMaxOperations = 64
	CandidateWriteMaxFileBytes  = 512 * 1024
	CandidateWriteMaxTotalBytes = 2 * 1024 * 1024
)

type CandidateWriteRequest struct {
	Schema       string                    `json:"schema"`
	CandidateID  string                    `json:"candidate_id"`
	Operations   []CandidateWriteOperation `json:"operations"`
	SourceSHA256 string                    `json:"source_sha256"`
}

type CandidateWriteOperation struct {
	Action         string `json:"action"`
	Path           string `json:"path"`
	Content        string `json:"content,omitempty"`
	ContentSHA256  string `json:"content_sha256,omitempty"`
	ExpectedSHA256 string `json:"expected_sha256,omitempty"`
}

type CandidateWriteReceipt struct {
	Schema             string                           `json:"schema"`
	State              string                           `json:"state"`
	CandidateID        string                           `json:"candidate_id"`
	SourceSHA256       string                           `json:"source_sha256"`
	RequestSHA256      string                           `json:"request_sha256"`
	Operations         []CandidateWriteOperationReceipt `json:"operations"`
	FilesChanged       int                              `json:"files_changed"`
	BytesWritten       int                              `json:"bytes_written"`
	WorkspaceMutation  bool                             `json:"workspace_mutation"`
	WriterAuthoredCode bool                             `json:"writer_authored_code"`
	NetworkUsed        bool                             `json:"network_used"`
	Installed          bool                             `json:"installed"`
	Promoted           bool                             `json:"promoted"`
	CanonChanged       bool                             `json:"canon_changed"`
	ReceiptSHA256      string                           `json:"receipt_sha256"`
}

type CandidateWriteOperationReceipt struct {
	Action       string `json:"action"`
	Path         string `json:"path"`
	BeforeSHA256 string `json:"before_sha256,omitempty"`
	AfterSHA256  string `json:"after_sha256,omitempty"`
	Bytes        int    `json:"bytes"`
}

type candidateWritePlan struct {
	operation CandidateWriteOperation
	target    string
	before    []byte
	mode      os.FileMode
}

func ApplyCandidateWrites(root string, request CandidateWriteRequest) (CandidateWriteReceipt, error) {
	canonical, requestDigest, plans, err := preflightCandidateWrites(root, request)
	if err != nil {
		return CandidateWriteReceipt{}, err
	}
	for index, plan := range plans {
		if err := applyCandidateWrite(plan); err != nil {
			rollbackCandidateWrites(plans[:index])
			return CandidateWriteReceipt{}, fmt.Errorf("apply candidate operation %d %s %s: %w", index+1, plan.operation.Action, plan.operation.Path, err)
		}
	}

	receipt := CandidateWriteReceipt{
		Schema: CandidateWriteReceiptSchema, State: "APPLIED", CandidateID: canonical.CandidateID,
		SourceSHA256: canonical.SourceSHA256, RequestSHA256: requestDigest,
		WorkspaceMutation: true, WriterAuthoredCode: false, NetworkUsed: false,
		Installed: false, Promoted: false, CanonChanged: false,
	}
	for _, plan := range plans {
		operation := CandidateWriteOperationReceipt{Action: plan.operation.Action, Path: plan.operation.Path}
		if plan.before != nil {
			operation.BeforeSHA256 = candidateBytesSHA256(plan.before)
		}
		if plan.operation.Action != "delete" {
			operation.AfterSHA256 = plan.operation.ContentSHA256
			operation.Bytes = len([]byte(plan.operation.Content))
			receipt.BytesWritten += operation.Bytes
		}
		receipt.Operations = append(receipt.Operations, operation)
	}
	receipt.FilesChanged = len(receipt.Operations)
	receipt.ReceiptSHA256, err = digestJSON(receipt, "candidate write receipt")
	if err != nil {
		rollbackCandidateWrites(plans)
		return CandidateWriteReceipt{}, err
	}
	return receipt, nil
}

func preflightCandidateWrites(root string, request CandidateWriteRequest) (CandidateWriteRequest, string, []candidateWritePlan, error) {
	if request.Schema != CandidateWriteRequestSchema {
		return CandidateWriteRequest{}, "", nil, fmt.Errorf("candidate write request schema must be %q", CandidateWriteRequestSchema)
	}
	if err := validateCandidateID(request.CandidateID); err != nil {
		return CandidateWriteRequest{}, "", nil, err
	}
	if err := validateSHA256("source_sha256", request.SourceSHA256); err != nil {
		return CandidateWriteRequest{}, "", nil, err
	}
	if len(request.Operations) == 0 || len(request.Operations) > CandidateWriteMaxOperations {
		return CandidateWriteRequest{}, "", nil, fmt.Errorf("candidate write operations must contain 1-%d entries", CandidateWriteMaxOperations)
	}

	rootInfo, err := os.Lstat(root)
	if err != nil {
		return CandidateWriteRequest{}, "", nil, fmt.Errorf("inspect candidate root %s: %w", root, err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return CandidateWriteRequest{}, "", nil, errors.New("candidate root must be an existing non-symlink directory")
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return CandidateWriteRequest{}, "", nil, fmt.Errorf("resolve candidate root %s: %w", root, err)
	}
	realRoot, err = filepath.Abs(realRoot)
	if err != nil {
		return CandidateWriteRequest{}, "", nil, fmt.Errorf("resolve absolute candidate root: %w", err)
	}

	canonical := request
	canonical.Operations = append([]CandidateWriteOperation(nil), request.Operations...)
	seen := make(map[string]struct{}, len(canonical.Operations))
	plans := make([]candidateWritePlan, 0, len(canonical.Operations))
	totalBytes := 0
	for index := range canonical.Operations {
		op := &canonical.Operations[index]
		op.Action = strings.ToLower(strings.TrimSpace(op.Action))
		normalized, err := normalizeCandidatePath(op.Path)
		if err != nil {
			return CandidateWriteRequest{}, "", nil, fmt.Errorf("operation %d: %w", index+1, err)
		}
		op.Path = normalized
		if _, exists := seen[op.Path]; exists {
			return CandidateWriteRequest{}, "", nil, fmt.Errorf("duplicate candidate path %q", op.Path)
		}
		seen[op.Path] = struct{}{}
		target := filepath.Join(realRoot, filepath.FromSlash(op.Path))
		if !candidatePathInside(realRoot, target) {
			return CandidateWriteRequest{}, "", nil, fmt.Errorf("candidate path %q escapes root", op.Path)
		}
		if err := rejectCandidateSymlink(realRoot, target); err != nil {
			return CandidateWriteRequest{}, "", nil, fmt.Errorf("candidate path %q: %w", op.Path, err)
		}
		plan := candidateWritePlan{operation: *op, target: target, mode: 0o644}
		info, statErr := os.Lstat(target)
		exists := statErr == nil
		if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return CandidateWriteRequest{}, "", nil, fmt.Errorf("inspect candidate path %q: %w", op.Path, statErr)
		}
		if exists {
			if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				return CandidateWriteRequest{}, "", nil, fmt.Errorf("candidate path %q must address a regular non-symlink file", op.Path)
			}
			plan.before, err = os.ReadFile(target)
			if err != nil {
				return CandidateWriteRequest{}, "", nil, fmt.Errorf("read candidate path %q: %w", op.Path, err)
			}
			plan.mode = info.Mode().Perm()
		}
		switch op.Action {
		case "create":
			if exists {
				return CandidateWriteRequest{}, "", nil, fmt.Errorf("create candidate path %q already exists", op.Path)
			}
			if op.ExpectedSHA256 != "" {
				return CandidateWriteRequest{}, "", nil, fmt.Errorf("create candidate path %q must not declare expected_sha256", op.Path)
			}
		case "update", "delete":
			if !exists {
				return CandidateWriteRequest{}, "", nil, fmt.Errorf("%s candidate path %q does not exist", op.Action, op.Path)
			}
			if err := validateSHA256("expected_sha256", op.ExpectedSHA256); err != nil {
				return CandidateWriteRequest{}, "", nil, fmt.Errorf("candidate path %q: %w", op.Path, err)
			}
			if candidateBytesSHA256(plan.before) != op.ExpectedSHA256 {
				return CandidateWriteRequest{}, "", nil, fmt.Errorf("candidate path %q stale expected_sha256", op.Path)
			}
		default:
			return CandidateWriteRequest{}, "", nil, fmt.Errorf("candidate path %q has unsupported action %q", op.Path, op.Action)
		}
		if op.Action == "delete" {
			if op.Content != "" || op.ContentSHA256 != "" {
				return CandidateWriteRequest{}, "", nil, fmt.Errorf("delete candidate path %q must not carry content", op.Path)
			}
		} else {
			contentBytes := []byte(op.Content)
			if len(contentBytes) > CandidateWriteMaxFileBytes {
				return CandidateWriteRequest{}, "", nil, fmt.Errorf("candidate path %q exceeds %d byte file limit", op.Path, CandidateWriteMaxFileBytes)
			}
			if err := validateSHA256("content_sha256", op.ContentSHA256); err != nil {
				return CandidateWriteRequest{}, "", nil, fmt.Errorf("candidate path %q: %w", op.Path, err)
			}
			if candidateBytesSHA256(contentBytes) != op.ContentSHA256 {
				return CandidateWriteRequest{}, "", nil, fmt.Errorf("candidate path %q content_sha256 mismatch", op.Path)
			}
			totalBytes += len(contentBytes)
			if totalBytes > CandidateWriteMaxTotalBytes {
				return CandidateWriteRequest{}, "", nil, fmt.Errorf("candidate transaction exceeds %d byte limit", CandidateWriteMaxTotalBytes)
			}
		}
		plan.operation = *op
		plans = append(plans, plan)
	}
	requestDigest, err := digestJSON(canonical, "candidate write request")
	if err != nil {
		return CandidateWriteRequest{}, "", nil, err
	}
	return canonical, requestDigest, plans, nil
}

func normalizeCandidatePath(value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value || strings.ContainsRune(value, 0) || strings.Contains(value, "\\") || strings.Contains(value, ":") {
		return "", errors.New("candidate path is invalid")
	}
	if filepath.IsAbs(value) || strings.HasPrefix(value, "/") || filepath.ToSlash(filepath.Clean(filepath.FromSlash(value))) != value {
		return "", fmt.Errorf("candidate path %q must be a clean relative slash path", value)
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." || segment == ".git" {
			return "", fmt.Errorf("candidate path %q contains a refused segment", value)
		}
	}
	return value, nil
}

func validateCandidateID(value string) error {
	if value == "" || len(value) > 128 {
		return errors.New("candidate_id is invalid")
	}
	for index, character := range value {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '.' || character == '_' || character == '-' {
			continue
		}
		return fmt.Errorf("candidate_id contains invalid character at position %d", index)
	}
	return nil
}

func rejectCandidateSymlink(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
		return errors.New("candidate target is outside root")
	}
	current := root
	for _, segment := range strings.Split(filepath.ToSlash(relative), "/") {
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("symlink traversal is refused")
		}
	}
	return nil
}

func candidatePathInside(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func applyCandidateWrite(plan candidateWritePlan) error {
	current, err := os.ReadFile(plan.target)
	if plan.operation.Action == "create" {
		if err == nil || !errors.Is(err, os.ErrNotExist) {
			if err == nil {
				return errors.New("create target appeared after preflight")
			}
			return err
		}
	} else {
		if err != nil {
			return err
		}
		if candidateBytesSHA256(current) != plan.operation.ExpectedSHA256 {
			return errors.New("target changed after preflight")
		}
	}
	if plan.operation.Action == "delete" {
		return os.Remove(plan.target)
	}
	if err := os.MkdirAll(filepath.Dir(plan.target), 0o755); err != nil {
		return err
	}
	return writeCandidateAtomic(plan.target, []byte(plan.operation.Content), plan.mode)
}

func writeCandidateAtomic(target string, content []byte, mode os.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(target), ".waldo-candidate-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	if _, err := temporary.Write(content); err != nil {
		return err
	}
	if err := temporary.Chmod(mode.Perm()); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, target)
}

func rollbackCandidateWrites(plans []candidateWritePlan) {
	for index := len(plans) - 1; index >= 0; index-- {
		plan := plans[index]
		if plan.before == nil {
			_ = os.Remove(plan.target)
			continue
		}
		_ = os.MkdirAll(filepath.Dir(plan.target), 0o755)
		_ = writeCandidateAtomic(plan.target, plan.before, plan.mode)
	}
}

func candidateBytesSHA256(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
