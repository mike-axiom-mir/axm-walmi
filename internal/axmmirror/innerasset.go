package axmmirror

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	InnerAssetRecipeSchema     = "axm.waldo-witness.inner-asset-recipe/v0.1"
	InnerAssetGridSchema       = "axm.waldo-witness.inner-asset-grid/v0.1"
	InnerAssetCandidateSchema  = "axm.waldo-witness.inner-asset-candidate/v0.1"
	InnerAssetValidationSchema = "axm.waldo-witness.inner-asset-validation/v0.1"
	SpriteAtlasSchema          = "axm.sprite-atlas/v1"

	InnerAssetStateReady        = "INNER_ASSET_CANDIDATE_READY"
	InnerAssetStateBudgetHold   = "HOLD_INNER_ASSET_BUDGET"
	InnerAssetStateAlphaHold    = "HOLD_INNER_ASSET_ALPHA"
	InnerAssetStateEmptyHold    = "HOLD_INNER_ASSET_EMPTY"
	InnerAssetStateSemanticHold = "HOLD_INNER_ASSET_SEMANTIC_CORE"

	InnerAssetTechnicalPass = "PASS"
	InnerAssetTechnicalHold = "HOLD"
	InnerAssetVisualPending = "UNREVIEWED"

	MaxInnerAssetBundleBytes = 16 * 1024 * 1024
	maxInnerAssetPixels      = 1024 * 1024
	maxInnerAssetOperations  = 4096
)

var innerAssetIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*$`)

// InnerAssetCanvas is the small pixel substrate implemented by the WALDO
// specialist. It is not the Workshop target-canvas runtime and cannot route to
// external hands, renderers, generators, or devices.
type InnerAssetCanvas struct {
	GridWidth             int    `json:"grid_width"`
	GridHeight            int    `json:"grid_height"`
	OutputScale           int    `json:"output_scale"`
	PreviewScale          int    `json:"preview_scale"`
	ColourSpace           string `json:"colour_space"`
	Transparency          string `json:"transparency"`
	FramesPerSecond       int    `json:"frames_per_second"`
	Loop                  bool   `json:"loop"`
	MaxFileBytes          int    `json:"max_file_bytes"`
	MaxTextureMemoryBytes int    `json:"max_texture_memory_bytes"`
}

type InnerAssetPaletteEntry struct {
	ID   string `json:"id"`
	Role string `json:"role"`
	RGBA string `json:"rgba"`
}

// InnerAssetOperation is deliberately bounded. It names pixel primitives, not
// source code, scripts, paths, URLs, prompts, or an external generation call.
type InnerAssetOperation struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Colour string `json:"colour,omitempty"`
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
	X2     int    `json:"x2,omitempty"`
	Y2     int    `json:"y2,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Count  int    `json:"count,omitempty"`
	Filled bool   `json:"filled,omitempty"`
}

// InnerAssetLayer preserves the important visual-grammar distinction between
// semantic shape and presentation. A protected semantic layer is the stable
// core that later style variation must not silently rewrite.
type InnerAssetLayer struct {
	ID         string                `json:"id"`
	Role       string                `json:"role"`
	Plane      string                `json:"plane"`
	Protected  bool                  `json:"protected"`
	Editable   bool                  `json:"editable"`
	Operations []InnerAssetOperation `json:"operations"`
}

type InnerAssetFrame struct {
	ID     string            `json:"id"`
	Layers []InnerAssetLayer `json:"layers"`
}

// InnerAssetRecipe is intended to be authored by the future specialist clone
// or by a human. The deterministic compiler, not the prose author, owns the
// resulting artifact bytes and technical receipt.
type InnerAssetRecipe struct {
	Schema      string                   `json:"schema"`
	RecipeID    string                   `json:"recipe_id"`
	Title       string                   `json:"title"`
	Kind        string                   `json:"kind"`
	IntendedUse string                   `json:"intended_use"`
	Profile     string                   `json:"profile"`
	Seed        string                   `json:"seed"`
	CreatedAt   string                   `json:"created_at"`
	Canvas      InnerAssetCanvas         `json:"canvas"`
	Palette     []InnerAssetPaletteEntry `json:"palette"`
	Frames      []InnerAssetFrame        `json:"frames"`
	Authority   Authority                `json:"authority"`
}

type InnerAssetArtifact struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Filename  string `json:"filename"`
	MIME      string `json:"mime"`
	Format    string `json:"format"`
	Editable  bool   `json:"editable"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	SizeBytes int    `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

type InnerAssetArtifactDigest struct {
	ID     string `json:"id"`
	MIME   string `json:"mime"`
	SHA256 string `json:"sha256"`
}

type InnerAssetCheck struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

// InnerAssetValidationReceipt proves bounded technical checks over exact
// artifact digests. It makes no claim about appearance, meaning, usefulness,
// originality, accessibility, game fitness, safety, or human approval.
type InnerAssetValidationReceipt struct {
	Schema             string                     `json:"schema"`
	State              string                     `json:"state"`
	ValidatorID        string                     `json:"validator_id"`
	ValidatorVersion   string                     `json:"validator_version"`
	Capability         string                     `json:"capability"`
	RecipeSHA256       string                     `json:"recipe_sha256"`
	TargetCanvasSHA256 string                     `json:"target_canvas_sha256"`
	ArtifactDigests    []InnerAssetArtifactDigest `json:"artifact_digests"`
	Checks             []InnerAssetCheck          `json:"checks"`
	CreatedAt          string                     `json:"created_at"`
	Holds              []string                   `json:"holds,omitempty"`
	Notices            []string                   `json:"notices"`
	Authority          Authority                  `json:"authority"`
	ReceiptSHA256      string                     `json:"receipt_sha256,omitempty"`
}

type InnerAssetMeasures struct {
	GridWidth               int      `json:"grid_width"`
	GridHeight              int      `json:"grid_height"`
	FrameCount              int      `json:"frame_count"`
	PaletteEntries          int      `json:"palette_entries"`
	ColoursUsed             int      `json:"colours_used"`
	NonTransparentPixels    int      `json:"non_transparent_pixels"`
	TransparentPixels       int      `json:"transparent_pixels"`
	ProtectedSemanticLayers []string `json:"protected_semantic_layers"`
	PrimaryTextureBytes     int      `json:"primary_texture_bytes"`
	PreviewTextureBytes     int      `json:"preview_texture_bytes"`
}

// InnerAssetCandidate is deliberately candidate-only. The artifact can be
// downloaded and inspected, but the receipt cannot approve or promote it.
type InnerAssetCandidate struct {
	Schema                  string               `json:"schema"`
	State                   string               `json:"state"`
	CandidateID             string               `json:"candidate_id"`
	RecipeID                string               `json:"recipe_id"`
	RecipeSHA256            string               `json:"recipe_sha256"`
	TargetCanvasSHA256      string               `json:"target_canvas_sha256"`
	ArtifactSetSHA256       string               `json:"artifact_set_sha256"`
	ValidationFilename      string               `json:"validation_filename"`
	ValidationFileSHA256    string               `json:"validation_file_sha256"`
	ValidationReceiptSHA256 string               `json:"validation_receipt_sha256"`
	PrimaryArtifactID       string               `json:"primary_artifact_id"`
	PreviewArtifactID       string               `json:"preview_artifact_id"`
	Artifacts               []InnerAssetArtifact `json:"artifacts"`
	Measures                InnerAssetMeasures   `json:"measures"`
	TechnicalStatus         string               `json:"technical_status"`
	VisualStatus            string               `json:"visual_status"`
	CandidateOnly           bool                 `json:"candidate_only"`
	HumanApproved           bool                 `json:"human_approved"`
	Canonical               bool                 `json:"canonical"`
	AutomaticPromotion      bool                 `json:"automatic_promotion"`
	CreatedAt               string               `json:"created_at"`
	Holds                   []string             `json:"holds,omitempty"`
	Notices                 []string             `json:"notices"`
	Authority               Authority            `json:"authority"`
	CandidateSHA256         string               `json:"candidate_sha256,omitempty"`
}

// InnerAssetBuild keeps raw artifact bytes ephemeral to the compile call. Only
// the portable bundle encoder persists them, alongside their typed receipts.
type InnerAssetBuild struct {
	Candidate  InnerAssetCandidate
	Validation InnerAssetValidationReceipt
	Files      map[string][]byte
}

func canonicalizeInnerAssetRecipe(recipe InnerAssetRecipe) (InnerAssetRecipe, error) {
	if recipe.Schema != InnerAssetRecipeSchema {
		return InnerAssetRecipe{}, fmt.Errorf("inner asset recipe schema must be %q", InnerAssetRecipeSchema)
	}
	for name, value := range map[string]string{
		"recipe_id": recipe.RecipeID, "title": recipe.Title, "intended_use": recipe.IntendedUse,
		"seed": recipe.Seed,
	} {
		if strings.TrimSpace(value) == "" || value != strings.TrimSpace(value) {
			return InnerAssetRecipe{}, fmt.Errorf("inner asset %s is required and must be trimmed", name)
		}
	}
	if !validInnerAssetID(recipe.RecipeID) {
		return InnerAssetRecipe{}, errors.New("inner asset recipe_id must use lowercase letters, numbers, dots, or hyphens")
	}
	if len(recipe.Title) > 160 || len(recipe.IntendedUse) > 240 || len(recipe.Seed) > 160 {
		return InnerAssetRecipe{}, errors.New("inner asset title, intended_use, or seed exceeds its bound")
	}
	if !oneOf(recipe.Kind, "icon", "sprite", "tile", "effect", "ui") {
		return InnerAssetRecipe{}, fmt.Errorf("unsupported inner asset kind %q", recipe.Kind)
	}
	if !oneOf(recipe.Profile, "pixel-8bit", "pixel-16bit", "pixel-custom") {
		return InnerAssetRecipe{}, fmt.Errorf("unsupported inner asset profile %q", recipe.Profile)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, recipe.CreatedAt)
	if err != nil {
		return InnerAssetRecipe{}, fmt.Errorf("parse inner asset created_at: %w", err)
	}
	if !recipe.Authority.closed() {
		return InnerAssetRecipe{}, errors.New("inner asset recipe must carry closed authority")
	}
	if err := validateInnerAssetCanvas(recipe.Profile, recipe.Canvas, len(recipe.Frames)); err != nil {
		return InnerAssetRecipe{}, err
	}
	if len(recipe.Palette) == 0 {
		return InnerAssetRecipe{}, errors.New("inner asset recipe requires a palette")
	}
	paletteLimit := 256
	if recipe.Profile == "pixel-8bit" {
		paletteLimit = 16
	} else if recipe.Profile == "pixel-16bit" {
		paletteLimit = 64
	}
	if len(recipe.Palette) > paletteLimit {
		return InnerAssetRecipe{}, fmt.Errorf("inner asset profile %s permits at most %d palette entries", recipe.Profile, paletteLimit)
	}

	canonical := recipe
	canonical.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
	canonical.Palette = append([]InnerAssetPaletteEntry(nil), recipe.Palette...)
	for i := range canonical.Palette {
		entry := &canonical.Palette[i]
		if !validInnerAssetID(entry.ID) || strings.TrimSpace(entry.Role) == "" || entry.Role != strings.TrimSpace(entry.Role) {
			return InnerAssetRecipe{}, fmt.Errorf("inner asset palette entry %d has invalid id or role", i)
		}
		rgba, err := parseInnerAssetRGBA(entry.RGBA)
		if err != nil {
			return InnerAssetRecipe{}, fmt.Errorf("inner asset palette entry %q: %w", entry.ID, err)
		}
		entry.RGBA = formatInnerAssetRGBA(rgba)
	}
	sort.Slice(canonical.Palette, func(i, j int) bool { return canonical.Palette[i].ID < canonical.Palette[j].ID })
	paletteIDs := make(map[string]struct{}, len(canonical.Palette))
	for i, entry := range canonical.Palette {
		if i > 0 && canonical.Palette[i-1].ID == entry.ID {
			return InnerAssetRecipe{}, fmt.Errorf("duplicate inner asset palette id %q", entry.ID)
		}
		paletteIDs[entry.ID] = struct{}{}
	}

	canonical.Frames = make([]InnerAssetFrame, len(recipe.Frames))
	frameIDs := make(map[string]struct{}, len(recipe.Frames))
	totalOperations := 0
	for frameIndex, frame := range recipe.Frames {
		if !validInnerAssetID(frame.ID) {
			return InnerAssetRecipe{}, fmt.Errorf("inner asset frame %d has invalid id", frameIndex)
		}
		if _, exists := frameIDs[frame.ID]; exists {
			return InnerAssetRecipe{}, fmt.Errorf("duplicate inner asset frame id %q", frame.ID)
		}
		frameIDs[frame.ID] = struct{}{}
		if len(frame.Layers) == 0 {
			return InnerAssetRecipe{}, fmt.Errorf("inner asset frame %q requires at least one layer", frame.ID)
		}
		canonicalFrame := InnerAssetFrame{ID: frame.ID, Layers: make([]InnerAssetLayer, len(frame.Layers))}
		layerIDs := make(map[string]struct{}, len(frame.Layers))
		for layerIndex, layer := range frame.Layers {
			if !validInnerAssetID(layer.ID) || strings.TrimSpace(layer.Role) == "" || layer.Role != strings.TrimSpace(layer.Role) || !oneOf(layer.Plane, "semantic", "presentation") {
				return InnerAssetRecipe{}, fmt.Errorf("inner asset frame %q layer %d has invalid identity, role, or plane", frame.ID, layerIndex)
			}
			if layer.Protected && layer.Editable {
				return InnerAssetRecipe{}, fmt.Errorf("inner asset layer %q cannot be protected and editable", layer.ID)
			}
			if _, exists := layerIDs[layer.ID]; exists {
				return InnerAssetRecipe{}, fmt.Errorf("duplicate inner asset layer id %q in frame %q", layer.ID, frame.ID)
			}
			layerIDs[layer.ID] = struct{}{}
			if len(layer.Operations) == 0 {
				return InnerAssetRecipe{}, fmt.Errorf("inner asset layer %q requires at least one operation", layer.ID)
			}
			canonicalLayer := layer
			canonicalLayer.Operations = append([]InnerAssetOperation(nil), layer.Operations...)
			operationIDs := make(map[string]struct{}, len(layer.Operations))
			for operationIndex, operation := range canonicalLayer.Operations {
				if err := validateInnerAssetOperation(recipe.Canvas, operation, paletteIDs); err != nil {
					return InnerAssetRecipe{}, fmt.Errorf("inner asset frame %q layer %q operation %d: %w", frame.ID, layer.ID, operationIndex, err)
				}
				if _, exists := operationIDs[operation.ID]; exists {
					return InnerAssetRecipe{}, fmt.Errorf("duplicate inner asset operation id %q in layer %q", operation.ID, layer.ID)
				}
				operationIDs[operation.ID] = struct{}{}
				totalOperations++
			}
			canonicalFrame.Layers[layerIndex] = canonicalLayer
		}
		canonical.Frames[frameIndex] = canonicalFrame
	}
	if totalOperations > maxInnerAssetOperations {
		return InnerAssetRecipe{}, fmt.Errorf("inner asset recipe has %d operations; maximum is %d", totalOperations, maxInnerAssetOperations)
	}
	return canonical, nil
}

func validateInnerAssetCanvas(profile string, canvas InnerAssetCanvas, frameCount int) error {
	if canvas.GridWidth < 1 || canvas.GridHeight < 1 || canvas.GridWidth > 64 || canvas.GridHeight > 64 {
		return errors.New("inner asset grid dimensions must be between 1 and 64 pixels")
	}
	if profile == "pixel-8bit" && (canvas.GridWidth > 32 || canvas.GridHeight > 32) {
		return errors.New("pixel-8bit profile is bounded to a 32x32 grid")
	}
	if canvas.OutputScale < 1 || canvas.OutputScale > 16 || canvas.PreviewScale < canvas.OutputScale || canvas.PreviewScale > 16 {
		return errors.New("inner asset scales must be 1..16 and preview_scale must be at least output_scale")
	}
	if canvas.ColourSpace != "srgb" || !oneOf(canvas.Transparency, "required", "allowed", "opaque") {
		return errors.New("inner asset canvas supports only sRGB and required/allowed/opaque transparency")
	}
	if frameCount < 1 || frameCount > 8 {
		return errors.New("inner asset recipe requires 1..8 frames")
	}
	if canvas.FramesPerSecond < 1 || canvas.FramesPerSecond > 60 {
		return errors.New("inner asset frames_per_second must be between 1 and 60")
	}
	if canvas.Loop && frameCount == 1 {
		return errors.New("inner asset loop requires more than one frame")
	}
	if canvas.MaxFileBytes < 1 || canvas.MaxFileBytes > 8*1024*1024 || canvas.MaxTextureMemoryBytes < 1 || canvas.MaxTextureMemoryBytes > 16*1024*1024 {
		return errors.New("inner asset file and texture budgets are missing or exceed the bounded compiler limits")
	}
	primaryPixels := canvas.GridWidth * canvas.OutputScale * frameCount * canvas.GridHeight * canvas.OutputScale
	previewPixels := canvas.GridWidth * canvas.PreviewScale * frameCount * canvas.GridHeight * canvas.PreviewScale
	if primaryPixels > maxInnerAssetPixels || previewPixels > maxInnerAssetPixels {
		return fmt.Errorf("inner asset primary or preview sheet exceeds %d pixels", maxInnerAssetPixels)
	}
	return nil
}

func validateInnerAssetOperation(canvas InnerAssetCanvas, operation InnerAssetOperation, palette map[string]struct{}) error {
	if !validInnerAssetID(operation.ID) || !oneOf(operation.Kind, "pixel", "rect", "line", "ellipse", "scatter") {
		return fmt.Errorf("operation has invalid id or unsupported kind %q", operation.Kind)
	}
	if _, ok := palette[operation.Colour]; !ok {
		return fmt.Errorf("operation %q references unknown colour %q", operation.ID, operation.Colour)
	}
	pointInBounds := func(x, y int) bool { return x >= 0 && y >= 0 && x < canvas.GridWidth && y < canvas.GridHeight }
	boxInBounds := func(x, y, width, height int) bool {
		return width > 0 && height > 0 && x >= 0 && y >= 0 && x+width <= canvas.GridWidth && y+height <= canvas.GridHeight
	}
	switch operation.Kind {
	case "pixel":
		if !pointInBounds(operation.X, operation.Y) || operation.X2 != 0 || operation.Y2 != 0 || operation.Width != 0 || operation.Height != 0 || operation.Count != 0 || operation.Filled {
			return fmt.Errorf("pixel operation %q has out-of-bounds or inapplicable fields", operation.ID)
		}
	case "rect":
		if !boxInBounds(operation.X, operation.Y, operation.Width, operation.Height) || operation.X2 != 0 || operation.Y2 != 0 || operation.Count != 0 || operation.Filled {
			return fmt.Errorf("rect operation %q has out-of-bounds or inapplicable fields", operation.ID)
		}
	case "line":
		if !pointInBounds(operation.X, operation.Y) || !pointInBounds(operation.X2, operation.Y2) || operation.Width != 0 || operation.Height != 0 || operation.Count != 0 || operation.Filled {
			return fmt.Errorf("line operation %q has out-of-bounds or inapplicable fields", operation.ID)
		}
	case "ellipse":
		if !boxInBounds(operation.X, operation.Y, operation.Width, operation.Height) || operation.X2 != 0 || operation.Y2 != 0 || operation.Count != 0 {
			return fmt.Errorf("ellipse operation %q has out-of-bounds or inapplicable fields", operation.ID)
		}
	case "scatter":
		if !boxInBounds(operation.X, operation.Y, operation.Width, operation.Height) || operation.X2 != 0 || operation.Y2 != 0 || operation.Count < 1 || operation.Count > 4096 || operation.Filled {
			return fmt.Errorf("scatter operation %q has out-of-bounds or inapplicable fields", operation.ID)
		}
	}
	return nil
}

func validInnerAssetID(value string) bool {
	return len(value) <= 120 && value == strings.TrimSpace(value) && innerAssetIDPattern.MatchString(value)
}

func safeInnerAssetFilename(value string) bool {
	return value != "" && value == filepath.Base(value) && value != "." && value != ".." && !strings.ContainsAny(value, `/\\`)
}

func encodeInnerAssetJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func innerAssetBytesSHA256(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func innerAssetValidationDigest(receipt InnerAssetValidationReceipt) (string, error) {
	receipt.ReceiptSHA256 = ""
	return digestJSON(receipt, "inner asset validation receipt")
}

func innerAssetCandidateDigest(candidate InnerAssetCandidate) (string, error) {
	candidate.CandidateSHA256 = ""
	return digestJSON(candidate, "inner asset candidate")
}

func (receipt InnerAssetValidationReceipt) Validate() error {
	if receipt.Schema != InnerAssetValidationSchema || !oneOf(receipt.State, InnerAssetTechnicalPass, InnerAssetTechnicalHold) {
		return fmt.Errorf("unsupported inner asset validation identity %q state %q", receipt.Schema, receipt.State)
	}
	if receipt.ValidatorID != "waldo-inner-pixel-foundry" || receipt.ValidatorVersion != "0.1.0" || receipt.Capability != "asset.pixel.compile-deterministic/v0.1" {
		return errors.New("inner asset validation has an unsupported validator identity")
	}
	for name, value := range map[string]string{
		"recipe_sha256": receipt.RecipeSHA256, "target_canvas_sha256": receipt.TargetCanvasSHA256,
		"receipt_sha256": receipt.ReceiptSHA256,
	} {
		if err := validateSHA256("inner asset validation "+name, value); err != nil {
			return err
		}
	}
	if _, err := time.Parse(time.RFC3339Nano, receipt.CreatedAt); err != nil {
		return fmt.Errorf("parse inner asset validation created_at: %w", err)
	}
	if len(receipt.ArtifactDigests) == 0 || !sort.SliceIsSorted(receipt.ArtifactDigests, func(i, j int) bool { return receipt.ArtifactDigests[i].ID < receipt.ArtifactDigests[j].ID }) {
		return errors.New("inner asset validation artifact digests are missing or unsorted")
	}
	seenArtifacts := map[string]struct{}{}
	for i, artifact := range receipt.ArtifactDigests {
		if !validInnerAssetID(artifact.ID) || strings.TrimSpace(artifact.MIME) == "" {
			return fmt.Errorf("inner asset validation artifact %d has invalid identity", i)
		}
		if _, exists := seenArtifacts[artifact.ID]; exists {
			return fmt.Errorf("duplicate inner asset validation artifact %q", artifact.ID)
		}
		seenArtifacts[artifact.ID] = struct{}{}
		if err := validateSHA256(fmt.Sprintf("inner asset validation artifact %d sha256", i), artifact.SHA256); err != nil {
			return err
		}
	}
	if len(receipt.Checks) == 0 || !sort.SliceIsSorted(receipt.Checks, func(i, j int) bool { return receipt.Checks[i].Name < receipt.Checks[j].Name }) {
		return errors.New("inner asset validation checks are missing or unsorted")
	}
	failed := 0
	for i, check := range receipt.Checks {
		if strings.TrimSpace(check.Name) == "" || strings.TrimSpace(check.Detail) == "" {
			return fmt.Errorf("inner asset validation check %d is incomplete", i)
		}
		if i > 0 && receipt.Checks[i-1].Name == check.Name {
			return fmt.Errorf("duplicate inner asset validation check %q", check.Name)
		}
		if !check.Pass {
			failed++
		}
	}
	if receipt.State == InnerAssetTechnicalPass {
		if failed != 0 || len(receipt.Holds) != 0 {
			return errors.New("passing inner asset validation carries failed checks or holds")
		}
	} else if failed == 0 || len(receipt.Holds) == 0 {
		return errors.New("held inner asset validation requires failed checks and reasons")
	}
	if len(receipt.Notices) == 0 || !receipt.Authority.closed() {
		return errors.New("inner asset validation must state its boundaries and carry closed authority")
	}
	expected, err := innerAssetValidationDigest(receipt)
	if err != nil {
		return err
	}
	if expected != receipt.ReceiptSHA256 {
		return fmt.Errorf("inner asset validation digest mismatch: expected %s, got %s", receipt.ReceiptSHA256, expected)
	}
	return nil
}

func (candidate InnerAssetCandidate) Validate() error {
	if candidate.Schema != InnerAssetCandidateSchema || !oneOf(candidate.State, InnerAssetStateReady, InnerAssetStateBudgetHold, InnerAssetStateAlphaHold, InnerAssetStateEmptyHold, InnerAssetStateSemanticHold) {
		return fmt.Errorf("unsupported inner asset candidate identity %q state %q", candidate.Schema, candidate.State)
	}
	if !strings.HasPrefix(candidate.CandidateID, "inner-asset-") || !validInnerAssetID(candidate.RecipeID) || candidate.PrimaryArtifactID != "primary-png" || candidate.PreviewArtifactID != "preview-png" || candidate.ValidationFilename != "validation.json" {
		return errors.New("inner asset candidate has incomplete or unsupported identity bindings")
	}
	for name, value := range map[string]string{
		"recipe_sha256": candidate.RecipeSHA256, "target_canvas_sha256": candidate.TargetCanvasSHA256,
		"artifact_set_sha256": candidate.ArtifactSetSHA256, "validation_file_sha256": candidate.ValidationFileSHA256,
		"validation_receipt_sha256": candidate.ValidationReceiptSHA256, "candidate_sha256": candidate.CandidateSHA256,
	} {
		if err := validateSHA256("inner asset candidate "+name, value); err != nil {
			return err
		}
	}
	if _, err := time.Parse(time.RFC3339Nano, candidate.CreatedAt); err != nil {
		return fmt.Errorf("parse inner asset candidate created_at: %w", err)
	}
	if len(candidate.Artifacts) != 5 || !sort.SliceIsSorted(candidate.Artifacts, func(i, j int) bool { return candidate.Artifacts[i].ID < candidate.Artifacts[j].ID }) {
		return errors.New("inner asset candidate must carry the five sorted recipe/grid/primary/preview/atlas artifacts")
	}
	required := map[string]bool{"normalized-recipe": false, "resolved-grid": false, "primary-png": false, "preview-png": false, "sprite-atlas": false}
	seenFiles := map[string]struct{}{}
	for i, artifact := range candidate.Artifacts {
		if _, ok := required[artifact.ID]; !ok || !validInnerAssetID(artifact.ID) || strings.TrimSpace(artifact.Role) == "" || strings.TrimSpace(artifact.MIME) == "" || strings.TrimSpace(artifact.Format) == "" || !safeInnerAssetFilename(artifact.Filename) || artifact.SizeBytes < 1 || artifact.Width < 0 || artifact.Height < 0 {
			return fmt.Errorf("inner asset artifact %d has invalid metadata", i)
		}
		if _, exists := seenFiles[artifact.Filename]; exists {
			return fmt.Errorf("duplicate inner asset artifact filename %q", artifact.Filename)
		}
		seenFiles[artifact.Filename] = struct{}{}
		required[artifact.ID] = true
		if err := validateSHA256(fmt.Sprintf("inner asset artifact %d sha256", i), artifact.SHA256); err != nil {
			return err
		}
	}
	for id, present := range required {
		if !present {
			return fmt.Errorf("inner asset candidate lacks artifact %q", id)
		}
	}
	artifactSetDigest, err := digestJSON(candidate.Artifacts, "inner asset artifact set")
	if err != nil {
		return err
	}
	if artifactSetDigest != candidate.ArtifactSetSHA256 {
		return errors.New("inner asset candidate artifact set digest does not match")
	}
	if candidate.Measures.GridWidth < 1 || candidate.Measures.GridHeight < 1 || candidate.Measures.FrameCount < 1 || candidate.Measures.PaletteEntries < 1 || candidate.Measures.PrimaryTextureBytes < 1 || candidate.Measures.PreviewTextureBytes < 1 {
		return errors.New("inner asset candidate measures are incomplete")
	}
	if !candidate.CandidateOnly || candidate.HumanApproved || candidate.Canonical || candidate.AutomaticPromotion || candidate.VisualStatus != InnerAssetVisualPending || !candidate.Authority.closed() {
		return errors.New("inner asset candidate gained approval, promotion, canon, or authority")
	}
	if candidate.State == InnerAssetStateReady {
		if candidate.TechnicalStatus != InnerAssetTechnicalPass || len(candidate.Holds) != 0 {
			return errors.New("ready inner asset candidate carries a technical hold")
		}
	} else if candidate.TechnicalStatus != InnerAssetTechnicalHold || len(candidate.Holds) == 0 {
		return errors.New("held inner asset candidate lacks technical hold state or reasons")
	}
	if len(candidate.Notices) == 0 {
		return errors.New("inner asset candidate must state its boundaries")
	}
	expected, err := innerAssetCandidateDigest(candidate)
	if err != nil {
		return err
	}
	if expected != candidate.CandidateSHA256 {
		return fmt.Errorf("inner asset candidate digest mismatch: expected %s, got %s", candidate.CandidateSHA256, expected)
	}
	return nil
}
