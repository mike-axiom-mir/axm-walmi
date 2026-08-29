package axmmirror

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"
)

// EncodeInnerAssetBundle produces one deterministic ZIP-compatible portable
// file. ZIP is only the transport container; candidate.json remains the typed
// authority and no entry is executable.
func EncodeInnerAssetBundle(build InnerAssetBuild) ([]byte, error) {
	if err := build.Candidate.Validate(); err != nil {
		return nil, fmt.Errorf("inner asset candidate: %w", err)
	}
	if err := build.Validation.Validate(); err != nil {
		return nil, fmt.Errorf("inner asset validation: %w", err)
	}
	if err := verifyInnerAssetBuildFiles(build.Candidate, build.Validation, build.Files); err != nil {
		return nil, err
	}
	if err := verifyInnerAssetDeterminism(build.Candidate, build.Validation, build.Files); err != nil {
		return nil, err
	}
	candidateBytes, err := encodeInnerAssetJSON(build.Candidate)
	if err != nil {
		return nil, fmt.Errorf("encode inner asset candidate: %w", err)
	}
	validationBytes, err := encodeInnerAssetJSON(build.Validation)
	if err != nil {
		return nil, fmt.Errorf("encode inner asset validation: %w", err)
	}
	entries := make(map[string][]byte, len(build.Files)+2)
	entries["candidate.json"] = candidateBytes
	entries["validation.json"] = validationBytes
	for name, data := range build.Files {
		entries[name] = data
	}
	names := make([]string, 0, len(entries))
	for name := range entries {
		if !safeInnerAssetFilename(name) {
			return nil, fmt.Errorf("inner asset bundle has unsafe filename %q", name)
		}
		names = append(names, name)
	}
	sort.Strings(names)

	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	fixedTime := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Store, Modified: fixedTime}
		header.SetMode(0o644)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("create inner asset bundle entry %s: %w", name, err)
		}
		if _, err := entry.Write(entries[name]); err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("write inner asset bundle entry %s: %w", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close inner asset bundle: %w", err)
	}
	if output.Len() > MaxInnerAssetBundleBytes {
		return nil, fmt.Errorf("inner asset bundle requires %d bytes; maximum is %d", output.Len(), MaxInnerAssetBundleBytes)
	}
	return output.Bytes(), nil
}

// VerifyInnerAssetBundle rejects unsafe/extra entries, re-hashes every file,
// and recompiles the normalized recipe to compare the exact deterministic
// candidate, validation receipt, and artifact bytes.
func VerifyInnerAssetBundle(data []byte) (InnerAssetCandidate, error) {
	if len(data) == 0 || len(data) > MaxInnerAssetBundleBytes {
		return InnerAssetCandidate{}, fmt.Errorf("inner asset bundle size %d is empty or exceeds %d", len(data), MaxInnerAssetBundleBytes)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return InnerAssetCandidate{}, fmt.Errorf("open inner asset bundle: %w", err)
	}
	entries := make(map[string][]byte, len(reader.File))
	totalBytes := 0
	for _, file := range reader.File {
		if file.FileInfo().IsDir() || !safeInnerAssetFilename(file.Name) || file.Method != zip.Store {
			return InnerAssetCandidate{}, fmt.Errorf("inner asset bundle entry %q is a directory, unsafe path, or unsupported compression method", file.Name)
		}
		if _, exists := entries[file.Name]; exists {
			return InnerAssetCandidate{}, fmt.Errorf("inner asset bundle has duplicate entry %q", file.Name)
		}
		if file.UncompressedSize64 > MaxInnerAssetBundleBytes || totalBytes+int(file.UncompressedSize64) > MaxInnerAssetBundleBytes {
			return InnerAssetCandidate{}, errors.New("inner asset bundle expands beyond its byte bound")
		}
		stream, err := file.Open()
		if err != nil {
			return InnerAssetCandidate{}, fmt.Errorf("open inner asset bundle entry %s: %w", file.Name, err)
		}
		content, readErr := io.ReadAll(io.LimitReader(stream, int64(MaxInnerAssetBundleBytes)+1))
		closeErr := stream.Close()
		if readErr != nil {
			return InnerAssetCandidate{}, fmt.Errorf("read inner asset bundle entry %s: %w", file.Name, readErr)
		}
		if closeErr != nil {
			return InnerAssetCandidate{}, fmt.Errorf("close inner asset bundle entry %s: %w", file.Name, closeErr)
		}
		if len(content) != int(file.UncompressedSize64) {
			return InnerAssetCandidate{}, fmt.Errorf("inner asset bundle entry %s size mismatch", file.Name)
		}
		totalBytes += len(content)
		entries[file.Name] = content
	}

	candidateBytes, ok := entries["candidate.json"]
	if !ok {
		return InnerAssetCandidate{}, errors.New("inner asset bundle lacks candidate.json")
	}
	validationBytes, ok := entries["validation.json"]
	if !ok {
		return InnerAssetCandidate{}, errors.New("inner asset bundle lacks validation.json")
	}
	var candidate InnerAssetCandidate
	if err := DecodeStrictJSON(candidateBytes, &candidate); err != nil {
		return InnerAssetCandidate{}, fmt.Errorf("decode inner asset candidate: %w", err)
	}
	if err := candidate.Validate(); err != nil {
		return InnerAssetCandidate{}, fmt.Errorf("validate inner asset candidate: %w", err)
	}
	var validation InnerAssetValidationReceipt
	if err := DecodeStrictJSON(validationBytes, &validation); err != nil {
		return InnerAssetCandidate{}, fmt.Errorf("decode inner asset validation: %w", err)
	}
	if err := validation.Validate(); err != nil {
		return InnerAssetCandidate{}, fmt.Errorf("validate inner asset validation: %w", err)
	}
	files := make(map[string][]byte, len(entries)-2)
	for name, content := range entries {
		if name != "candidate.json" && name != "validation.json" {
			files[name] = content
		}
	}
	if err := verifyInnerAssetBuildFiles(candidate, validation, files); err != nil {
		return InnerAssetCandidate{}, err
	}

	if err := verifyInnerAssetDeterminism(candidate, validation, files); err != nil {
		return InnerAssetCandidate{}, err
	}
	return candidate, nil
}

func verifyInnerAssetDeterminism(candidate InnerAssetCandidate, validation InnerAssetValidationReceipt, files map[string][]byte) error {
	recipeArtifact := findInnerAssetArtifact(candidate.Artifacts, "normalized-recipe")
	if recipeArtifact == nil {
		return errors.New("inner asset bundle lacks its normalized recipe artifact")
	}
	var recipe InnerAssetRecipe
	if err := DecodeStrictJSON(files[recipeArtifact.Filename], &recipe); err != nil {
		return fmt.Errorf("decode normalized inner asset recipe: %w", err)
	}
	expected, err := ForgeInnerAsset(recipe)
	if err != nil {
		return fmt.Errorf("recompile normalized inner asset recipe: %w", err)
	}
	if expected.Candidate.CandidateSHA256 != candidate.CandidateSHA256 || expected.Validation.ReceiptSHA256 != validation.ReceiptSHA256 {
		return errors.New("inner asset bundle candidate or validation differs from deterministic recompilation")
	}
	if len(expected.Files) != len(files) {
		return errors.New("inner asset bundle file set differs from deterministic recompilation")
	}
	for name, expectedData := range expected.Files {
		actual, exists := files[name]
		if !exists || !bytes.Equal(expectedData, actual) {
			return fmt.Errorf("inner asset artifact %s differs from deterministic recompilation", name)
		}
	}
	return nil
}

func verifyInnerAssetBuildFiles(candidate InnerAssetCandidate, validation InnerAssetValidationReceipt, files map[string][]byte) error {
	validationBytes, err := encodeInnerAssetJSON(validation)
	if err != nil {
		return fmt.Errorf("encode inner asset validation: %w", err)
	}
	if candidate.ValidationReceiptSHA256 != validation.ReceiptSHA256 || candidate.ValidationFileSHA256 != innerAssetBytesSHA256(validationBytes) || candidate.RecipeSHA256 != validation.RecipeSHA256 || candidate.TargetCanvasSHA256 != validation.TargetCanvasSHA256 || candidate.TechnicalStatus != validation.State {
		return errors.New("inner asset candidate and validation receipt bindings do not match")
	}
	if len(files) != len(candidate.Artifacts) {
		return errors.New("inner asset bundle artifact count does not match candidate")
	}
	validationByID := make(map[string]InnerAssetArtifactDigest, len(validation.ArtifactDigests))
	for _, digest := range validation.ArtifactDigests {
		validationByID[digest.ID] = digest
	}
	for _, artifact := range candidate.Artifacts {
		data, exists := files[artifact.Filename]
		if !exists {
			return fmt.Errorf("inner asset bundle lacks artifact %s (%s)", artifact.ID, artifact.Filename)
		}
		if len(data) != artifact.SizeBytes || innerAssetBytesSHA256(data) != artifact.SHA256 {
			return fmt.Errorf("inner asset artifact %s size or digest mismatch", artifact.ID)
		}
		digest, exists := validationByID[artifact.ID]
		if !exists || digest.MIME != artifact.MIME || digest.SHA256 != artifact.SHA256 {
			return fmt.Errorf("inner asset validation does not bind artifact %s", artifact.ID)
		}
		delete(validationByID, artifact.ID)
	}
	if len(validationByID) != 0 {
		return errors.New("inner asset validation binds unknown artifacts")
	}
	return nil
}

func findInnerAssetArtifact(artifacts []InnerAssetArtifact, id string) *InnerAssetArtifact {
	for index := range artifacts {
		if artifacts[index].ID == id {
			return &artifacts[index]
		}
	}
	return nil
}
