package axmmirror

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"sort"
	"strconv"
	"strings"
)

type innerAssetPixel struct {
	ColourID string
	RGBA     color.NRGBA
}

type innerAssetRenderedFrame struct {
	ID     string
	Pixels []innerAssetPixel
}

type InnerAssetResolvedFrame struct {
	ID     string     `json:"id"`
	Pixels [][]string `json:"pixels"`
}

type InnerAssetResolvedGrid struct {
	Schema       string                    `json:"schema"`
	RecipeSHA256 string                    `json:"recipe_sha256"`
	Width        int                       `json:"width"`
	Height       int                       `json:"height"`
	Palette      []InnerAssetPaletteEntry  `json:"palette"`
	Frames       []InnerAssetResolvedFrame `json:"frames"`
}

type SpriteAtlasTag struct {
	Name      string `json:"name"`
	From      int    `json:"from"`
	To        int    `json:"to"`
	Direction string `json:"direction"`
}

type SpriteAtlasPivot struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type SpriteAtlasFrame struct {
	ID         string           `json:"id"`
	X          int              `json:"x"`
	Y          int              `json:"y"`
	Width      int              `json:"width"`
	Height     int              `json:"height"`
	DurationMS int              `json:"durationMs"`
	Pivot      SpriteAtlasPivot `json:"pivot"`
}

type SpriteAtlas struct {
	Schema      string             `json:"schema"`
	Name        string             `json:"name"`
	Image       string             `json:"image"`
	FrameWidth  int                `json:"frameWidth"`
	FrameHeight int                `json:"frameHeight"`
	FrameCount  int                `json:"frameCount"`
	FPS         float64            `json:"fps"`
	Loop        bool               `json:"loop"`
	Tags        []SpriteAtlasTag   `json:"tags"`
	Pivot       SpriteAtlasPivot   `json:"pivot"`
	Frames      []SpriteAtlasFrame `json:"frames"`
}

func parseInnerAssetRGBA(value string) (color.NRGBA, error) {
	if len(value) != 7 && len(value) != 9 || !strings.HasPrefix(value, "#") {
		return color.NRGBA{}, errors.New("rgba must be #RRGGBB or #RRGGBBAA")
	}
	decoded := make([]byte, (len(value)-1)/2)
	for i := range decoded {
		component, err := strconv.ParseUint(value[1+i*2:3+i*2], 16, 8)
		if err != nil {
			return color.NRGBA{}, errors.New("rgba must contain hexadecimal colour components")
		}
		decoded[i] = byte(component)
	}
	result := color.NRGBA{R: decoded[0], G: decoded[1], B: decoded[2], A: 0xff}
	if len(decoded) == 4 {
		result.A = decoded[3]
	}
	return result, nil
}

func formatInnerAssetRGBA(value color.NRGBA) string {
	return fmt.Sprintf("#%02x%02x%02x%02x", value.R, value.G, value.B, value.A)
}

// ForgeInnerAsset compiles one canonical recipe with no network, external
// generator, filesystem lookup, runtime adapter, or hidden model call.
func ForgeInnerAsset(recipe InnerAssetRecipe) (InnerAssetBuild, error) {
	canonical, err := canonicalizeInnerAssetRecipe(recipe)
	if err != nil {
		return InnerAssetBuild{}, err
	}
	recipeBytes, err := encodeInnerAssetJSON(canonical)
	if err != nil {
		return InnerAssetBuild{}, fmt.Errorf("encode normalized inner asset recipe: %w", err)
	}
	recipeDigest := innerAssetBytesSHA256(recipeBytes)
	targetCanvasDigest, err := digestJSON(canonical.Canvas, "inner asset target canvas")
	if err != nil {
		return InnerAssetBuild{}, err
	}
	rendered, protectedLayers, err := renderInnerAssetFrames(canonical)
	if err != nil {
		return InnerAssetBuild{}, err
	}
	grid := resolveInnerAssetGrid(canonical, recipeDigest, rendered)
	gridBytes, err := encodeInnerAssetJSON(grid)
	if err != nil {
		return InnerAssetBuild{}, fmt.Errorf("encode resolved inner asset grid: %w", err)
	}
	primaryBytes, primaryWidth, primaryHeight, err := encodeInnerAssetPNG(canonical.Canvas, rendered, canonical.Canvas.OutputScale)
	if err != nil {
		return InnerAssetBuild{}, fmt.Errorf("encode inner asset primary PNG: %w", err)
	}
	previewBytes, previewWidth, previewHeight, err := encodeInnerAssetPNG(canonical.Canvas, rendered, canonical.Canvas.PreviewScale)
	if err != nil {
		return InnerAssetBuild{}, fmt.Errorf("encode inner asset preview PNG: %w", err)
	}
	atlas := makeInnerAssetAtlas(canonical, primaryWidth/len(rendered), primaryHeight)
	atlasBytes, err := encodeInnerAssetJSON(atlas)
	if err != nil {
		return InnerAssetBuild{}, fmt.Errorf("encode inner asset sprite atlas: %w", err)
	}

	files := map[string][]byte{
		"normalized-recipe.json": recipeBytes,
		"resolved-grid.json":     gridBytes,
		"asset.png":              primaryBytes,
		"preview.png":            previewBytes,
		"atlas.json":             atlasBytes,
	}
	artifacts := []InnerAssetArtifact{
		makeInnerAssetArtifact("normalized-recipe", "editable-recipe", "normalized-recipe.json", "application/json", "JSON", true, 0, 0, recipeBytes),
		makeInnerAssetArtifact("resolved-grid", "editable-pixel-grid", "resolved-grid.json", "application/json", "JSON", true, canonical.Canvas.GridWidth, canonical.Canvas.GridHeight, gridBytes),
		makeInnerAssetArtifact("primary-png", "primary-deliverable", "asset.png", "image/png", "PNG", false, primaryWidth, primaryHeight, primaryBytes),
		makeInnerAssetArtifact("preview-png", "preview", "preview.png", "image/png", "PNG", false, previewWidth, previewHeight, previewBytes),
		makeInnerAssetArtifact("sprite-atlas", "runtime-metadata", "atlas.json", "application/json", "JSON", true, 0, 0, atlasBytes),
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].ID < artifacts[j].ID })
	artifactSetDigest, err := digestJSON(artifacts, "inner asset artifact set")
	if err != nil {
		return InnerAssetBuild{}, err
	}
	measures := measureInnerAsset(canonical, rendered, protectedLayers, primaryWidth, primaryHeight, previewWidth, previewHeight)
	checks, state, holds := assessInnerAsset(canonical, measures, len(primaryBytes), len(previewBytes), protectedLayers)
	technicalState := InnerAssetTechnicalPass
	if state != InnerAssetStateReady {
		technicalState = InnerAssetTechnicalHold
	}
	artifactDigests := make([]InnerAssetArtifactDigest, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifactDigests = append(artifactDigests, InnerAssetArtifactDigest{ID: artifact.ID, MIME: artifact.MIME, SHA256: artifact.SHA256})
	}
	validation := InnerAssetValidationReceipt{
		Schema: InnerAssetValidationSchema, State: technicalState,
		ValidatorID: "waldo-inner-pixel-foundry", ValidatorVersion: "0.1.0",
		Capability:   "asset.pixel.compile-deterministic/v0.1",
		RecipeSHA256: recipeDigest, TargetCanvasSHA256: targetCanvasDigest,
		ArtifactDigests: artifactDigests, Checks: checks, CreatedAt: canonical.CreatedAt,
		Holds: holds,
		Notices: []string{
			"technical PASS proves only deterministic bounded compilation and declared constraint checks over exact artifact bytes",
			"the validator does not inspect appearance, meaning, originality, accessibility, usefulness, game fitness, safety, or human preference",
			"the compiler performs no external generation, source retrieval, tool routing, installation, publication, approval, promotion, CANON, or world action",
		},
		Authority: Authority{},
	}
	validation.ReceiptSHA256, err = innerAssetValidationDigest(validation)
	if err != nil {
		return InnerAssetBuild{}, err
	}
	if err := validation.Validate(); err != nil {
		return InnerAssetBuild{}, fmt.Errorf("generated inner asset validation: %w", err)
	}
	validationBytes, err := encodeInnerAssetJSON(validation)
	if err != nil {
		return InnerAssetBuild{}, fmt.Errorf("encode inner asset validation: %w", err)
	}

	candidate := InnerAssetCandidate{
		Schema: InnerAssetCandidateSchema, State: state,
		CandidateID: "inner-asset-" + recipeDigest[:24], RecipeID: canonical.RecipeID,
		RecipeSHA256: recipeDigest, TargetCanvasSHA256: targetCanvasDigest,
		ArtifactSetSHA256:  artifactSetDigest,
		ValidationFilename: "validation.json", ValidationFileSHA256: innerAssetBytesSHA256(validationBytes),
		ValidationReceiptSHA256: validation.ReceiptSHA256,
		PrimaryArtifactID:       "primary-png", PreviewArtifactID: "preview-png",
		Artifacts: artifacts, Measures: measures, TechnicalStatus: technicalState,
		VisualStatus: InnerAssetVisualPending, CandidateOnly: true,
		HumanApproved: false, Canonical: false, AutomaticPromotion: false,
		CreatedAt: canonical.CreatedAt, Holds: holds,
		Notices: []string{
			"this is a portable unreviewed candidate; technical validity is not visual approval or shared-vocabulary admission",
			"the normalized recipe and resolved grid remain editable, while protected semantic layers identify shape that later style changes must not silently rewrite",
			"primary delivery and preview are separate artifacts; neither is automatically installed into a game, model, Workshop, Asset Vault, release, or CANON surface",
			"candidate generation grants no model, tool, filesystem, network, execution, training, promotion, CANON, or world-action authority",
		},
		Authority: Authority{},
	}
	candidate.CandidateSHA256, err = innerAssetCandidateDigest(candidate)
	if err != nil {
		return InnerAssetBuild{}, err
	}
	if err := candidate.Validate(); err != nil {
		return InnerAssetBuild{}, fmt.Errorf("generated inner asset candidate: %w", err)
	}
	return InnerAssetBuild{Candidate: candidate, Validation: validation, Files: files}, nil
}

func makeInnerAssetArtifact(id, role, filename, mime, format string, editable bool, width, height int, data []byte) InnerAssetArtifact {
	return InnerAssetArtifact{
		ID: id, Role: role, Filename: filename, MIME: mime, Format: format,
		Editable: editable, Width: width, Height: height,
		SizeBytes: len(data), SHA256: innerAssetBytesSHA256(data),
	}
}

func renderInnerAssetFrames(recipe InnerAssetRecipe) ([]innerAssetRenderedFrame, []string, error) {
	palette := make(map[string]color.NRGBA, len(recipe.Palette))
	for _, entry := range recipe.Palette {
		rgba, err := parseInnerAssetRGBA(entry.RGBA)
		if err != nil {
			return nil, nil, err
		}
		palette[entry.ID] = rgba
	}
	protected := []string{}
	rendered := make([]innerAssetRenderedFrame, 0, len(recipe.Frames))
	for _, frame := range recipe.Frames {
		pixels := make([]innerAssetPixel, recipe.Canvas.GridWidth*recipe.Canvas.GridHeight)
		locked := make([]bool, len(pixels))
		for _, layer := range frame.Layers {
			layerPixels := make([]innerAssetPixel, len(pixels))
			touched := make([]bool, len(pixels))
			if layer.Plane == "semantic" && layer.Protected {
				protected = append(protected, frame.ID+"/"+layer.ID)
			}
			for _, operation := range layer.Operations {
				if err := drawInnerAssetOperation(recipe, frame.ID, layer.ID, operation, palette[operation.Colour], layerPixels, touched); err != nil {
					return nil, nil, err
				}
			}
			for index, changed := range touched {
				if changed && !locked[index] {
					pixels[index] = layerPixels[index]
				}
			}
			if layer.Plane == "semantic" && layer.Protected {
				for index, changed := range touched {
					if changed {
						locked[index] = true
					}
				}
			}
		}
		rendered = append(rendered, innerAssetRenderedFrame{ID: frame.ID, Pixels: pixels})
	}
	return rendered, protected, nil
}

func drawInnerAssetOperation(recipe InnerAssetRecipe, frameID, layerID string, operation InnerAssetOperation, rgba color.NRGBA, pixels []innerAssetPixel, touched []bool) error {
	width := recipe.Canvas.GridWidth
	set := func(x, y int) {
		index := y*width + x
		pixels[index] = innerAssetPixel{ColourID: operation.Colour, RGBA: rgba}
		touched[index] = true
	}
	switch operation.Kind {
	case "pixel":
		set(operation.X, operation.Y)
	case "rect":
		for y := operation.Y; y < operation.Y+operation.Height; y++ {
			for x := operation.X; x < operation.X+operation.Width; x++ {
				set(x, y)
			}
		}
	case "line":
		drawInnerAssetLine(operation.X, operation.Y, operation.X2, operation.Y2, set)
	case "ellipse":
		drawInnerAssetEllipse(operation, set)
	case "scatter":
		generator := newInnerAssetPRNG(recipe.Seed + "\x00" + frameID + "\x00" + layerID + "\x00" + operation.ID)
		area := uint64(operation.Width * operation.Height)
		for index := 0; index < operation.Count; index++ {
			position := generator.next() % area
			set(operation.X+int(position%uint64(operation.Width)), operation.Y+int(position/uint64(operation.Width)))
		}
	default:
		return fmt.Errorf("unsupported inner asset operation %q", operation.Kind)
	}
	return nil
}

func drawInnerAssetLine(x0, y0, x1, y1 int, set func(int, int)) {
	dx := absInnerAsset(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -absInnerAsset(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		set(x0, y0)
		if x0 == x1 && y0 == y1 {
			return
		}
		twice := 2 * err
		if twice >= dy {
			err += dy
			x0 += sx
		}
		if twice <= dx {
			err += dx
			y0 += sy
		}
	}
}

func drawInnerAssetEllipse(operation InnerAssetOperation, set func(int, int)) {
	w, h := int64(operation.Width), int64(operation.Height)
	inside := func(localX, localY int) bool {
		if localX < 0 || localY < 0 || localX >= operation.Width || localY >= operation.Height {
			return false
		}
		x := int64(2*localX+1) - w
		y := int64(2*localY+1) - h
		return x*x*h*h+y*y*w*w <= w*w*h*h
	}
	for y := 0; y < operation.Height; y++ {
		for x := 0; x < operation.Width; x++ {
			if !inside(x, y) {
				continue
			}
			if operation.Filled || !inside(x-1, y) || !inside(x+1, y) || !inside(x, y-1) || !inside(x, y+1) {
				set(operation.X+x, operation.Y+y)
			}
		}
	}
}

type innerAssetPRNG struct{ state uint64 }

func newInnerAssetPRNG(seed string) *innerAssetPRNG {
	digest := sha256.Sum256([]byte(seed))
	state := binary.LittleEndian.Uint64(digest[:8])
	if state == 0 {
		state = 0x9e3779b97f4a7c15
	}
	return &innerAssetPRNG{state: state}
}

func (generator *innerAssetPRNG) next() uint64 {
	state := generator.state
	state ^= state >> 12
	state ^= state << 25
	state ^= state >> 27
	generator.state = state
	return state * 2685821657736338717
}

func absInnerAsset(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func resolveInnerAssetGrid(recipe InnerAssetRecipe, recipeDigest string, rendered []innerAssetRenderedFrame) InnerAssetResolvedGrid {
	grid := InnerAssetResolvedGrid{
		Schema: InnerAssetGridSchema, RecipeSHA256: recipeDigest,
		Width: recipe.Canvas.GridWidth, Height: recipe.Canvas.GridHeight,
		Palette: append([]InnerAssetPaletteEntry(nil), recipe.Palette...),
		Frames:  make([]InnerAssetResolvedFrame, 0, len(rendered)),
	}
	for _, frame := range rendered {
		rows := make([][]string, recipe.Canvas.GridHeight)
		for y := 0; y < recipe.Canvas.GridHeight; y++ {
			rows[y] = make([]string, recipe.Canvas.GridWidth)
			for x := 0; x < recipe.Canvas.GridWidth; x++ {
				rows[y][x] = frame.Pixels[y*recipe.Canvas.GridWidth+x].ColourID
			}
		}
		grid.Frames = append(grid.Frames, InnerAssetResolvedFrame{ID: frame.ID, Pixels: rows})
	}
	return grid
}

func encodeInnerAssetPNG(canvas InnerAssetCanvas, rendered []innerAssetRenderedFrame, scale int) ([]byte, int, int, error) {
	frameWidth := canvas.GridWidth * scale
	frameHeight := canvas.GridHeight * scale
	width := frameWidth * len(rendered)
	height := frameHeight
	imageData := image.NewNRGBA(image.Rect(0, 0, width, height))
	for frameIndex, frame := range rendered {
		for y := 0; y < canvas.GridHeight; y++ {
			for x := 0; x < canvas.GridWidth; x++ {
				pixel := frame.Pixels[y*canvas.GridWidth+x].RGBA
				for scaledY := 0; scaledY < scale; scaledY++ {
					for scaledX := 0; scaledX < scale; scaledX++ {
						imageData.SetNRGBA(frameIndex*frameWidth+x*scale+scaledX, y*scale+scaledY, pixel)
					}
				}
			}
		}
	}
	var output bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(&output, imageData); err != nil {
		return nil, 0, 0, err
	}
	return output.Bytes(), width, height, nil
}

func makeInnerAssetAtlas(recipe InnerAssetRecipe, frameWidth, frameHeight int) SpriteAtlas {
	pivot := SpriteAtlasPivot{X: 0.5, Y: 1}
	duration := max(1, 1000/recipe.Canvas.FramesPerSecond)
	atlas := SpriteAtlas{
		Schema: SpriteAtlasSchema, Name: recipe.Title, Image: "asset.png",
		FrameWidth: frameWidth, FrameHeight: frameHeight, FrameCount: len(recipe.Frames),
		FPS: float64(recipe.Canvas.FramesPerSecond), Loop: recipe.Canvas.Loop,
		Tags:  []SpriteAtlasTag{{Name: "default", From: 0, To: len(recipe.Frames) - 1, Direction: "forward"}},
		Pivot: pivot, Frames: make([]SpriteAtlasFrame, 0, len(recipe.Frames)),
	}
	for index, frame := range recipe.Frames {
		atlas.Frames = append(atlas.Frames, SpriteAtlasFrame{
			ID: frame.ID, X: index * frameWidth, Y: 0, Width: frameWidth, Height: frameHeight,
			DurationMS: duration, Pivot: pivot,
		})
	}
	return atlas
}

func measureInnerAsset(recipe InnerAssetRecipe, rendered []innerAssetRenderedFrame, protected []string, primaryWidth, primaryHeight, previewWidth, previewHeight int) InnerAssetMeasures {
	colours := map[string]struct{}{}
	nonTransparent, transparent := 0, 0
	for _, frame := range rendered {
		for _, pixel := range frame.Pixels {
			if pixel.ColourID != "" {
				colours[formatInnerAssetRGBA(pixel.RGBA)] = struct{}{}
			}
			if pixel.RGBA.A == 0 {
				transparent++
			} else {
				nonTransparent++
			}
		}
	}
	return InnerAssetMeasures{
		GridWidth: recipe.Canvas.GridWidth, GridHeight: recipe.Canvas.GridHeight,
		FrameCount: len(rendered), PaletteEntries: len(recipe.Palette), ColoursUsed: len(colours),
		NonTransparentPixels: nonTransparent, TransparentPixels: transparent,
		ProtectedSemanticLayers: append([]string(nil), protected...),
		PrimaryTextureBytes:     primaryWidth * primaryHeight * 4,
		PreviewTextureBytes:     previewWidth * previewHeight * 4,
	}
}

func assessInnerAsset(recipe InnerAssetRecipe, measures InnerAssetMeasures, primaryFileBytes, previewFileBytes int, protected []string) ([]InnerAssetCheck, string, []string) {
	protectedFrames := map[string]struct{}{}
	for _, value := range protected {
		frame, _, _ := strings.Cut(value, "/")
		protectedFrames[frame] = struct{}{}
	}
	semanticPass := len(protectedFrames) == len(recipe.Frames)
	alphaPass := true
	switch recipe.Canvas.Transparency {
	case "required":
		alphaPass = measures.TransparentPixels > 0
	case "opaque":
		alphaPass = measures.TransparentPixels == 0
	}
	fileBudgetPass := primaryFileBytes <= recipe.Canvas.MaxFileBytes && previewFileBytes <= recipe.Canvas.MaxFileBytes
	textureBudgetPass := measures.PrimaryTextureBytes <= recipe.Canvas.MaxTextureMemoryBytes
	nonEmptyPass := measures.NonTransparentPixels > 0
	checks := []InnerAssetCheck{
		{Name: "alpha-contract", Pass: alphaPass, Detail: fmt.Sprintf("transparency=%s transparent_pixels=%d", recipe.Canvas.Transparency, measures.TransparentPixels)},
		{Name: "file-budget", Pass: fileBudgetPass, Detail: fmt.Sprintf("primary=%d preview=%d max_each=%d", primaryFileBytes, previewFileBytes, recipe.Canvas.MaxFileBytes)},
		{Name: "non-empty-render", Pass: nonEmptyPass, Detail: fmt.Sprintf("non_transparent_pixels=%d", measures.NonTransparentPixels)},
		{Name: "protected-semantic-core", Pass: semanticPass, Detail: fmt.Sprintf("protected_frames=%d total_frames=%d", len(protectedFrames), len(recipe.Frames))},
		{Name: "texture-budget", Pass: textureBudgetPass, Detail: fmt.Sprintf("primary_texture_bytes=%d max=%d", measures.PrimaryTextureBytes, recipe.Canvas.MaxTextureMemoryBytes)},
	}
	sort.Slice(checks, func(i, j int) bool { return checks[i].Name < checks[j].Name })
	var holds []string
	for _, check := range checks {
		if !check.Pass {
			holds = append(holds, check.Name+": "+check.Detail)
		}
	}
	state := InnerAssetStateReady
	switch {
	case !nonEmptyPass:
		state = InnerAssetStateEmptyHold
	case !semanticPass:
		state = InnerAssetStateSemanticHold
	case !alphaPass:
		state = InnerAssetStateAlphaHold
	case !fileBudgetPass || !textureBudgetPass:
		state = InnerAssetStateBudgetHold
	}
	return checks, state, holds
}
