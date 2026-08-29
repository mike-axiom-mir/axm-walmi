package axmmirror_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openwaldo/waldo/internal/axmmirror"
	"github.com/openwaldo/waldo/internal/corpus"
	"github.com/openwaldo/waldo/internal/index"
	"github.com/openwaldo/waldo/internal/model"
	"github.com/openwaldo/waldo/internal/shard"
)

func TestCorpusLensDigestMatchesWALDOSchemaOneHash(t *testing.T) {
	data := readExample(t, "corpus-bom.json")
	var bom corpus.BOM
	if err := json.Unmarshal(data, &bom); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(bom)
	if err != nil {
		t.Fatal(err)
	}
	lens, err := axmmirror.LensCorpusBOM(data)
	if err != nil {
		t.Fatal(err)
	}
	if lens.BOMSHA256 != sha256Hex(canonical) {
		t.Fatalf("lens BOM digest = %s, WALDO schema-1 digest = %s", lens.BOMSHA256, sha256Hex(canonical))
	}
}

func TestCorpusLensDigestMatchesCurrentWALDOPrivacyBOM(t *testing.T) {
	measure := index.Measures{Shards: 1, Docs: 2, Tokens: 8, Bytes: 128}
	conversion := index.Conversion{Tool: "waldo", Version: "test", Profile: "text", Recipe: shard.TextWriterRecipe, Tokenizer: "byte"}
	assessment := &index.ContentAssessment{
		EmailAddresses:     &index.DetectionMeasure{Detector: shard.EmailDetector, Records: 0},
		RepetitiveContent:  &index.DetectionMeasure{Detector: shard.RepetitionDetector, Records: 1},
		BoilerplateContent: &index.DetectionMeasure{Detector: shard.BoilerplateDetector, Records: 0},
	}
	redaction := &index.ContentRedaction{Policy: shard.PrivacyRedactionPolicy, NamesRetained: true, EmailAddresses: 1, IPAddresses: 1}
	manifest := corpus.ManifestPin{
		Path: "core/current.json", SHA256: strings.Repeat("a", 64), Name: "current", Title: "Current corpus", Description: "Current privacy-aware fixture.",
		License: "CC0-1.0", Format: "parquet", RecordSchema: shard.TextRecordSchema, ConvertedBy: conversion,
		Sources:    []index.Source{{Name: "fixture", Source: "Fixture source", License: "CC0-1.0", URL: "https://example.invalid/source", SHA256: strings.Repeat("b", 64)}},
		Assessment: assessment, Redaction: redaction, Totals: measure, Licenses: map[string]index.Measures{"CC0-1.0": measure},
	}
	shardPin := corpus.ShardPin{
		Manifest: manifest.Path, URL: "https://example.invalid/shard.parquet", SHA256: strings.Repeat("c", 64), Format: "parquet",
		RecordSchema: shard.TextRecordSchema, License: "CC0-1.0", Sources: []string{"fixture"}, ConvertedBy: conversion,
		Docs: 2, Tokens: 8, Bytes: 128, Assessment: assessment, Redaction: redaction,
	}
	mainContent := true
	bom := corpus.BOM{
		Kind: "openwaldo-bom", Schema: corpus.BOMSchema, Subject: "corpus",
		Index: index.Identity{Remote: "https://example.invalid/index.git", Commit: strings.Repeat("d", 40)},
		Paths: []string{""}, RecordFilter: &corpus.RecordFilterPolicy{Schema: corpus.RecordFilterSchema, Global: &corpus.RecordFilter{MainContent: &mainContent}},
		Manifests: []corpus.ManifestPin{manifest}, Shards: []corpus.ShardPin{shardPin}, Totals: measure, Licenses: map[string]index.Measures{"CC0-1.0": measure},
	}
	if err := bom.Validate(); err != nil {
		t.Fatalf("current WALDO BOM fixture: %v", err)
	}
	data, err := json.MarshalIndent(bom, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(bom)
	if err != nil {
		t.Fatal(err)
	}
	lens, err := axmmirror.LensCorpusBOM(data)
	if err != nil {
		t.Fatal(err)
	}
	if lens.BOMSHA256 != sha256Hex(canonical) {
		t.Fatalf("current lens BOM digest = %s, WALDO digest = %s", lens.BOMSHA256, sha256Hex(canonical))
	}
	if lens.PrivacyRedaction == nil || lens.PrivacyRedaction.State != "RECORDED_COMPLETE" || lens.PrivacyRedaction.EmailAddresses != 1 || lens.Assessment == nil || lens.Assessment.State != "RECORDED_COMPLETE" || lens.RecordFilter == nil || lens.RecordFilter.State != "DECLARED" {
		t.Fatalf("current lens row evidence = filter %+v assessment %+v privacy %+v", lens.RecordFilter, lens.Assessment, lens.PrivacyRedaction)
	}
}

func TestRunWitnessDigestMatchesWALDOSchemaOneHash(t *testing.T) {
	runBOMData := readExample(t, "run-bom.json")
	runData := readExample(t, "run.json")
	var bom model.RunBOM
	if err := json.Unmarshal(runBOMData, &bom); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(bom)
	if err != nil {
		t.Fatal(err)
	}
	witness, err := axmmirror.WitnessTrainingRun(runBOMData, runData)
	if err != nil {
		t.Fatal(err)
	}
	if witness.RunBOMSHA256 != sha256Hex(canonical) {
		t.Fatalf("witness run BOM digest = %s, WALDO schema-1 digest = %s", witness.RunBOMSHA256, sha256Hex(canonical))
	}
}

func readExample(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "axm-mirror", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func sha256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
