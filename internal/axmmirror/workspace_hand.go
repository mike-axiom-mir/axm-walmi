package axmmirror

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	WorkspaceHandSchema      = "axm.waldo.workspace-hand/v0.2"
	WorkspaceHandMaxEntries  = 4096
	WorkspaceHandMaxReadByte = 512 * 1024
)

type WorkspaceHand struct {
	root string
}

type WorkspaceEntry struct {
	Path       string `json:"path"`
	Kind       string `json:"kind"`
	Bytes      int64  `json:"bytes,omitempty"`
	Mode       string `json:"mode"`
	SHA256     string `json:"sha256,omitempty"`
	Executable bool   `json:"executable"`
}

type WorkspaceObservation struct {
	Schema            string           `json:"schema"`
	Tool              string           `json:"tool"`
	Path              string           `json:"path,omitempty"`
	Entries           []WorkspaceEntry `json:"entries,omitempty"`
	Content           string           `json:"content,omitempty"`
	Entry             *WorkspaceEntry  `json:"entry,omitempty"`
	WriteReceipt      *CandidateWriteReceipt `json:"write_receipt,omitempty"`
	Truncated         bool             `json:"truncated"`
	WorkspaceMutation bool             `json:"workspace_mutation"`
	NetworkUsed       bool             `json:"network_used"`
	Authority         string           `json:"authority"`
}

func NewWorkspaceHand(root string) (*WorkspaceHand, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect workspace root %s: %w", root, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, errors.New("workspace root must be an existing non-symlink directory")
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root %s: %w", root, err)
	}
	realRoot, err = filepath.Abs(realRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute workspace root: %w", err)
	}
	return &WorkspaceHand{root: realRoot}, nil
}

func (hand *WorkspaceHand) Root() string { return hand.root }

func (hand *WorkspaceHand) List(path string) (WorkspaceObservation, error) {
	target, relative, err := hand.resolve(path, true)
	if err != nil {
		return WorkspaceObservation{}, err
	}
	info, err := os.Lstat(target)
	if err != nil {
		return WorkspaceObservation{}, fmt.Errorf("inspect workspace path %q: %w", relative, err)
	}
	if !info.IsDir() {
		return WorkspaceObservation{}, fmt.Errorf("workspace path %q is not a directory", relative)
	}
	entries := make([]WorkspaceEntry, 0)
	err = filepath.WalkDir(target, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == target {
			return nil
		}
		relFromRoot, err := filepath.Rel(hand.root, current)
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(relFromRoot)
		if entry.Name() == ".git" && entry.IsDir() {
			return filepath.SkipDir
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("workspace path %q contains a refused symlink", slash)
		}
		if len(entries) >= WorkspaceHandMaxEntries {
			return errWorkspaceEntryLimit
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		entries = append(entries, workspaceEntry(slash, fileInfo, ""))
		return nil
	})
	truncated := errors.Is(err, errWorkspaceEntryLimit)
	if err != nil && !truncated {
		return WorkspaceObservation{}, fmt.Errorf("list workspace path %q: %w", relative, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return workspaceObservation("LIST_FILES", relative, entries, truncated), nil
}

func (hand *WorkspaceHand) Read(path string, maxBytes int) (WorkspaceObservation, error) {
	target, relative, err := hand.resolve(path, false)
	if err != nil {
		return WorkspaceObservation{}, err
	}
	if maxBytes <= 0 || maxBytes > WorkspaceHandMaxReadByte {
		maxBytes = WorkspaceHandMaxReadByte
	}
	info, err := os.Lstat(target)
	if err != nil {
		return WorkspaceObservation{}, fmt.Errorf("inspect workspace file %q: %w", relative, err)
	}
	if !info.Mode().IsRegular() {
		return WorkspaceObservation{}, fmt.Errorf("workspace path %q is not a regular file", relative)
	}
	file, err := os.Open(target)
	if err != nil {
		return WorkspaceObservation{}, fmt.Errorf("open workspace file %q: %w", relative, err)
	}
	defer file.Close()
	buffer := make([]byte, maxBytes+1)
	count, err := file.Read(buffer)
	if err != nil && count == 0 {
		return WorkspaceObservation{}, fmt.Errorf("read workspace file %q: %w", relative, err)
	}
	truncated := count > maxBytes
	if truncated {
		count = maxBytes
	}
	return WorkspaceObservation{Schema: WorkspaceHandSchema, Tool: "READ_FILE", Path: relative, Content: string(buffer[:count]), Truncated: truncated, Authority: "NONE"}, nil
}

func (hand *WorkspaceHand) Stat(path string, includeHash bool) (WorkspaceObservation, error) {
	target, relative, err := hand.resolve(path, true)
	if err != nil {
		return WorkspaceObservation{}, err
	}
	info, err := os.Lstat(target)
	if err != nil {
		return WorkspaceObservation{}, fmt.Errorf("inspect workspace path %q: %w", relative, err)
	}
	digest := ""
	if includeHash {
		if !info.Mode().IsRegular() {
			return WorkspaceObservation{}, fmt.Errorf("workspace path %q cannot be hashed because it is not a regular file", relative)
		}
		data, err := os.ReadFile(target)
		if err != nil {
			return WorkspaceObservation{}, fmt.Errorf("hash workspace file %q: %w", relative, err)
		}
		sum := sha256.Sum256(data)
		digest = hex.EncodeToString(sum[:])
	}
	entry := workspaceEntry(relative, info, digest)
	tool := "STAT_FILE"
	if includeHash {
		tool = "HASH_FILE"
	}
	return WorkspaceObservation{Schema: WorkspaceHandSchema, Tool: tool, Path: relative, Entry: &entry, Authority: "NONE"}, nil
}

func (hand *WorkspaceHand) Apply(request CandidateWriteRequest, allowed bool) (WorkspaceObservation, error) {
	if !allowed {
		return WorkspaceObservation{}, errors.New("workspace write requires explicit --allow-write opt-in")
	}
	receipt, err := ApplyCandidateWrites(hand.root, request)
	if err != nil {
		return WorkspaceObservation{}, err
	}
	return WorkspaceObservation{Schema: WorkspaceHandSchema, Tool: "APPLY_WRITES", WriteReceipt: &receipt, WorkspaceMutation: true, Authority: "NONE"}, nil
}

func (hand *WorkspaceHand) resolve(value string, allowRoot bool) (string, string, error) {
	if value == "" || value == "." {
		if allowRoot {
			return hand.root, ".", nil
		}
		return "", "", errors.New("workspace file path is required")
	}
	if strings.TrimSpace(value) != value || strings.ContainsRune(value, 0) || strings.Contains(value, "\\") || strings.Contains(value, ":") || filepath.IsAbs(value) {
		return "", "", fmt.Errorf("workspace path %q is invalid", value)
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if clean != value || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", "", fmt.Errorf("workspace path %q must be a clean relative slash path", value)
	}
	for _, segment := range strings.Split(clean, "/") {
		if segment == "" || segment == "." || segment == ".." || segment == ".git" {
			return "", "", fmt.Errorf("workspace path %q contains a refused segment", value)
		}
	}
	target := filepath.Join(hand.root, filepath.FromSlash(clean))
	if !candidatePathInside(hand.root, target) {
		return "", "", fmt.Errorf("workspace path %q escapes root", value)
	}
	if err := rejectCandidateSymlink(hand.root, target); err != nil {
		return "", "", fmt.Errorf("workspace path %q: %w", value, err)
	}
	return target, clean, nil
}

var errWorkspaceEntryLimit = errors.New("workspace entry limit reached")

func workspaceObservation(tool, path string, entries []WorkspaceEntry, truncated bool) WorkspaceObservation {
	return WorkspaceObservation{Schema: WorkspaceHandSchema, Tool: tool, Path: path, Entries: entries, Truncated: truncated, Authority: "NONE"}
}

func workspaceEntry(path string, info os.FileInfo, digest string) WorkspaceEntry {
	kind := "file"
	if info.IsDir() {
		kind = "directory"
	}
	return WorkspaceEntry{Path: path, Kind: kind, Bytes: info.Size(), Mode: info.Mode().String(), SHA256: digest, Executable: info.Mode().Perm()&0o111 != 0}
}

