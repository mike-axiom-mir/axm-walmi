package axmmirror

import (
	"bytes"
	"image/png"
	"testing"
)

func TestInnerAssetForgeBuildsDeterministicPortableCandidate(t *testing.T) {
	recipe := validInnerAssetRecipe()
	first, err := ForgeInnerAsset(recipe)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ForgeInnerAsset(recipe)
	if err != nil {
		t.Fatal(err)
	}
	if first.Candidate.State != InnerAssetStateReady || first.Candidate.TechnicalStatus != InnerAssetTechnicalPass || first.Candidate.VisualStatus != InnerAssetVisualPending {
		t.Fatalf("inner asset candidate = %+v", first.Candidate)
	}
	if !first.Candidate.CandidateOnly || first.Candidate.HumanApproved || first.Candidate.Canonical || first.Candidate.AutomaticPromotion || !first.Candidate.Authority.closed() {
		t.Fatalf("inner asset candidate gained authority or approval: %+v", first.Candidate)
	}
	if first.Candidate.CandidateSHA256 != second.Candidate.CandidateSHA256 || first.Validation.ReceiptSHA256 != second.Validation.ReceiptSHA256 {
		t.Fatal("same recipe produced different candidate or validation digests")
	}
	for name, firstData := range first.Files {
		if !bytes.Equal(firstData, second.Files[name]) {
			t.Fatalf("same recipe produced different %s bytes", name)
		}
	}

	configuration, err := png.DecodeConfig(bytes.NewReader(first.Files["asset.png"]))
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Width != 64 || configuration.Height != 32 {
		t.Fatalf("primary PNG dimensions = %dx%d, want 64x32", configuration.Width, configuration.Height)
	}
	var atlas SpriteAtlas
	if err := DecodeStrictJSON(first.Files["atlas.json"], &atlas); err != nil {
		t.Fatal(err)
	}
	if atlas.Schema != SpriteAtlasSchema || atlas.FrameCount != 2 || atlas.FrameWidth != 32 || atlas.FrameHeight != 32 || !atlas.Loop {
		t.Fatalf("sprite atlas = %+v", atlas)
	}

	firstBundle, err := EncodeInnerAssetBundle(first)
	if err != nil {
		t.Fatal(err)
	}
	secondBundle, err := EncodeInnerAssetBundle(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstBundle, secondBundle) {
		t.Fatal("same recipe produced different portable bundle bytes")
	}
	verified, err := VerifyInnerAssetBundle(firstBundle)
	if err != nil {
		t.Fatal(err)
	}
	if verified.CandidateSHA256 != first.Candidate.CandidateSHA256 {
		t.Fatalf("verified candidate = %s, want %s", verified.CandidateSHA256, first.Candidate.CandidateSHA256)
	}
}

func TestInnerAssetSeedChangesOnlyDeterministicRecipeBranch(t *testing.T) {
	first, err := ForgeInnerAsset(validInnerAssetRecipe())
	if err != nil {
		t.Fatal(err)
	}
	recipe := validInnerAssetRecipe()
	recipe.Seed = "witness-orb-branch-b"
	second, err := ForgeInnerAsset(recipe)
	if err != nil {
		t.Fatal(err)
	}
	if first.Candidate.CandidateSHA256 == second.Candidate.CandidateSHA256 {
		t.Fatal("different seed produced the same candidate digest")
	}
	if bytes.Equal(first.Files["asset.png"], second.Files["asset.png"]) {
		t.Fatal("different seed did not change the seeded presentation scatter")
	}
}

func TestInnerAssetProtectedSemanticPixelsRejectPresentationOverwrite(t *testing.T) {
	recipe := validInnerAssetRecipe()
	for frameIndex := range recipe.Frames {
		recipe.Frames[frameIndex].Layers[1].Operations = []InnerAssetOperation{{
			ID: "presentation-fill", Kind: "rect", Colour: "accent",
			X: 0, Y: 0, Width: recipe.Canvas.GridWidth, Height: recipe.Canvas.GridHeight,
		}}
	}
	build, err := ForgeInnerAsset(recipe)
	if err != nil {
		t.Fatal(err)
	}
	var grid InnerAssetResolvedGrid
	if err := DecodeStrictJSON(build.Files["resolved-grid.json"], &grid); err != nil {
		t.Fatal(err)
	}
	if grid.Frames[0].Pixels[8][5] != "highlight" {
		t.Fatalf("protected semantic pixel was overwritten with %q", grid.Frames[0].Pixels[8][5])
	}
	if grid.Frames[0].Pixels[0][0] != "accent" {
		t.Fatalf("unprotected presentation pixel = %q, want accent", grid.Frames[0].Pixels[0][0])
	}
}

func TestInnerAssetHonestTechnicalHolds(t *testing.T) {
	tests := []struct {
		name string
		want string
		edit func(*InnerAssetRecipe)
	}{
		{
			name: "semantic core", want: InnerAssetStateSemanticHold,
			edit: func(recipe *InnerAssetRecipe) {
				for frameIndex := range recipe.Frames {
					recipe.Frames[frameIndex].Layers[0].Protected = false
					recipe.Frames[frameIndex].Layers[0].Editable = true
				}
			},
		},
		{
			name: "opaque alpha", want: InnerAssetStateAlphaHold,
			edit: func(recipe *InnerAssetRecipe) { recipe.Canvas.Transparency = "opaque" },
		},
		{
			name: "file budget", want: InnerAssetStateBudgetHold,
			edit: func(recipe *InnerAssetRecipe) { recipe.Canvas.MaxFileBytes = 1 },
		},
		{
			name: "empty render", want: InnerAssetStateEmptyHold,
			edit: func(recipe *InnerAssetRecipe) {
				for frameIndex := range recipe.Frames {
					for layerIndex := range recipe.Frames[frameIndex].Layers {
						for operationIndex := range recipe.Frames[frameIndex].Layers[layerIndex].Operations {
							recipe.Frames[frameIndex].Layers[layerIndex].Operations[operationIndex].Colour = "transparent"
						}
					}
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recipe := validInnerAssetRecipe()
			test.edit(&recipe)
			build, err := ForgeInnerAsset(recipe)
			if err != nil {
				t.Fatal(err)
			}
			if build.Candidate.State != test.want || build.Candidate.TechnicalStatus != InnerAssetTechnicalHold || len(build.Candidate.Holds) == 0 {
				t.Fatalf("held inner asset = %+v", build.Candidate)
			}
			bundle, err := EncodeInnerAssetBundle(build)
			if err != nil {
				t.Fatal(err)
			}
			verified, err := VerifyInnerAssetBundle(bundle)
			if err != nil {
				t.Fatal(err)
			}
			if verified.State != test.want {
				t.Fatalf("verified state = %q, want %q", verified.State, test.want)
			}
		})
	}
}

func TestInnerAssetRefusesInvalidOperationAndAuthority(t *testing.T) {
	recipe := validInnerAssetRecipe()
	recipe.Frames[0].Layers[0].Operations[0].Width = 100
	if _, err := ForgeInnerAsset(recipe); err == nil {
		t.Fatal("out-of-bounds operation was accepted")
	}
	recipe = validInnerAssetRecipe()
	recipe.Authority.ToolExecution = true
	if _, err := ForgeInnerAsset(recipe); err == nil {
		t.Fatal("asset recipe with tool authority was accepted")
	}
}

func TestInnerAssetBundleDetectsArtifactAndSelfDigestedCandidateTamper(t *testing.T) {
	build, err := ForgeInnerAsset(validInnerAssetRecipe())
	if err != nil {
		t.Fatal(err)
	}
	tamperedFile := append([]byte(nil), build.Files["asset.png"]...)
	tamperedFile[len(tamperedFile)-1] ^= 0xff
	build.Files["asset.png"] = tamperedFile
	if _, err := EncodeInnerAssetBundle(build); err == nil {
		t.Fatal("tampered artifact was encoded")
	}

	build, err = ForgeInnerAsset(validInnerAssetRecipe())
	if err != nil {
		t.Fatal(err)
	}
	build.Candidate.Notices[0] = "self-digested rewrite"
	build.Candidate.CandidateSHA256, err = innerAssetCandidateDigest(build.Candidate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EncodeInnerAssetBundle(build); err == nil {
		t.Fatal("self-digested candidate rewrite survived pre-encoding deterministic recompilation")
	}
}

func validInnerAssetRecipe() InnerAssetRecipe {
	semanticFrame := func(id string, shift int) InnerAssetFrame {
		return InnerAssetFrame{
			ID: id,
			Layers: []InnerAssetLayer{
				{
					ID: "semantic-core", Role: "witness-orb-silhouette", Plane: "semantic", Protected: true,
					Operations: []InnerAssetOperation{
						{ID: "orb-outline", Kind: "ellipse", Colour: "outline", X: 2, Y: 2 + shift, Width: 12, Height: 12, Filled: true},
						{ID: "orb-body", Kind: "ellipse", Colour: "primary", X: 3, Y: 3 + shift, Width: 10, Height: 10, Filled: true},
						{ID: "witness-line", Kind: "line", Colour: "highlight", X: 5, Y: 8 + shift, X2: 10, Y2: 8 + shift},
					},
				},
				{
					ID: "presentation-signal", Role: "signal-sparks", Plane: "presentation", Editable: true,
					Operations: []InnerAssetOperation{
						{ID: "signal-scatter", Kind: "scatter", Colour: "accent", X: 0, Y: shift, Width: 16, Height: 14, Count: 8},
					},
				},
			},
		}
	}
	return InnerAssetRecipe{
		Schema: InnerAssetRecipeSchema, RecipeID: "witness-orb-v1", Title: "WALDO Witness Orb",
		Kind: "sprite", IntendedUse: "portable specialist-clone status sprite", Profile: "pixel-16bit",
		Seed: "witness-orb-branch-a", CreatedAt: "2026-08-15T10:00:00Z",
		Canvas: InnerAssetCanvas{
			GridWidth: 16, GridHeight: 16, OutputScale: 2, PreviewScale: 8,
			ColourSpace: "srgb", Transparency: "required", FramesPerSecond: 6, Loop: true,
			MaxFileBytes: 2 * 1024 * 1024, MaxTextureMemoryBytes: 2 * 1024 * 1024,
		},
		Palette: []InnerAssetPaletteEntry{
			{ID: "transparent", Role: "transparent", RGBA: "#00000000"},
			{ID: "outline", Role: "outline", RGBA: "#11182fff"},
			{ID: "primary", Role: "primary", RGBA: "#2e7f95ff"},
			{ID: "highlight", Role: "highlight", RGBA: "#d9fff5ff"},
			{ID: "accent", Role: "accent", RGBA: "#f0b35aff"},
		},
		Frames:    []InnerAssetFrame{semanticFrame("pulse-a", 0), semanticFrame("pulse-b", 1)},
		Authority: Authority{},
	}
}
