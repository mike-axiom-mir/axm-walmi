package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openwaldo/waldo/internal/axmmirror"
)

func TestBuildReviewHTMLUsesVerifiedCandidateBytes(t *testing.T) {
	bundle := testInnerAssetBundle(t)
	page, candidate, err := buildReviewHTML(bundle)
	if err != nil {
		t.Fatal(err)
	}
	text := string(page)
	for _, want := range []string{
		"WALMI · Inner Asset Review Desk",
		"VERIFIED BUNDLE",
		"DISPLAY ≠ APPROVAL",
		"NO AUTO-MATERIALIZE",
		candidate.CandidateSHA256,
		candidate.RecipeSHA256,
		candidate.ValidationReceiptSHA256,
		"data:image/png;base64,",
		"Visual review",
		axmmirror.InnerAssetVisualPending,
		"License fitness",
		"does not prove appearance quality",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("review page missing %q", want)
		}
	}
	if strings.Contains(text, "http://") || strings.Contains(text, "https://") {
		t.Fatal("review page must not introduce network URLs")
	}
	if candidate.HumanApproved || candidate.Canonical || candidate.AutomaticPromotion {
		t.Fatal("verified review fixture unexpectedly carries approval or promotion authority")
	}
}

func TestBuildReviewHTMLRejectsTamperedBundle(t *testing.T) {
	bundle := append([]byte(nil), testInnerAssetBundle(t)...)
	if len(bundle) < 64 {
		t.Fatal("fixture bundle unexpectedly small")
	}
	bundle[len(bundle)/2] ^= 0x01
	if _, _, err := buildReviewHTML(bundle); err == nil {
		t.Fatal("tampered candidate unexpectedly produced a review")
	}
}

func TestWriteReviewFileRefusesExistingOutput(t *testing.T) {
	bundle := testInnerAssetBundle(t)
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "candidate.axmasset")
	outputPath := filepath.Join(dir, "review.html")
	if err := os.WriteFile(bundlePath, bundle, 0o600); err != nil {
		t.Fatal(err)
	}
	candidate, err := writeReviewFile(bundlePath, outputPath)
	if err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.CandidateSHA256 == "" || !bytes.Contains(first, []byte(candidate.CandidateSHA256)) {
		t.Fatal("written review does not expose exact candidate identity")
	}
	if _, err := writeReviewFile(bundlePath, outputPath); err == nil {
		t.Fatal("existing review output was silently replaced")
	}
	second, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("failed no-replace attempt changed the existing review")
	}
}

func testInnerAssetBundle(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "examples", "axm-mirror", "inner-asset-recipe.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read synthetic recipe: %v", err)
	}
	var recipe axmmirror.InnerAssetRecipe
	if err := axmmirror.DecodeStrictJSON(data, &recipe); err != nil {
		t.Fatalf("decode synthetic recipe: %v", err)
	}
	build, err := axmmirror.ForgeInnerAsset(recipe)
	if err != nil {
		t.Fatalf("forge synthetic asset: %v", err)
	}
	if build.Candidate.State != axmmirror.InnerAssetStateReady {
		t.Fatalf("synthetic review fixture state = %s, want %s", build.Candidate.State, axmmirror.InnerAssetStateReady)
	}
	bundle, err := axmmirror.EncodeInnerAssetBundle(build)
	if err != nil {
		t.Fatalf("encode synthetic asset: %v", err)
	}
	return bundle
}
