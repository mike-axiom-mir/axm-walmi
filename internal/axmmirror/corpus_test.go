package axmmirror

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestCorpusEvidenceLensProjectsValidatedBOM(t *testing.T) {
	bom := validCorpusBOM()
	data, err := json.MarshalIndent(bom, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	lens, err := LensCorpusBOM(data)
	if err != nil {
		t.Fatalf("LensCorpusBOM() error = %v", err)
	}
	canonical, err := json.Marshal(bom)
	if err != nil {
		t.Fatal(err)
	}
	if lens.State != CorpusEvidenceStateReady || lens.BOMSHA256 != digestBytes(canonical) || lens.DocumentSHA256 != digestBytes(data) || lens.ReceiptSHA256 == "" {
		t.Fatalf("lens identity = %+v", lens)
	}
	if lens.Index.State != "CLEAN_PINNED" || len(lens.Paths) != 1 || lens.Paths[0] != "" {
		t.Fatalf("lens index/paths = %+v / %#v", lens.Index, lens.Paths)
	}
	if len(lens.Manifests) != 1 || len(lens.Manifests[0].Sources) != 1 || len(lens.Licenses) != 1 || lens.Licenses[0].State != "RECORDED_ASSERTION" {
		t.Fatalf("lens projection = %+v", lens)
	}
	if lens.Attestation.State != "NOT_RECORDED" || lens.Attestation.NotRecorded != 1 {
		t.Fatalf("lens attestation = %+v", lens.Attestation)
	}
	if lens.Schema != CorpusEvidenceLensSchema || lens.RecordFilter == nil || lens.RecordFilter.State != "NOT_DECLARED" || lens.Assessment == nil || lens.Assessment.State != "NOT_APPLICABLE_LEGACY" || lens.PrivacyRedaction == nil || lens.PrivacyRedaction.State != "NOT_RECORDED" {
		t.Fatalf("lens v0.2 row evidence = filter %+v assessment %+v privacy %+v", lens.RecordFilter, lens.Assessment, lens.PrivacyRedaction)
	}
	if err := lens.Validate(); err != nil {
		t.Fatalf("lens.Validate() error = %v", err)
	}
}

func TestCorpusEvidenceLensDetectsReceiptTamper(t *testing.T) {
	data, _ := json.Marshal(validCorpusBOM())
	lens, err := LensCorpusBOM(data)
	if err != nil {
		t.Fatal(err)
	}
	lens.Totals.Tokens++
	if err := lens.Validate(); err == nil || !strings.Contains(err.Error(), "receipt digest mismatch") {
		t.Fatalf("lens.Validate() tamper error = %v", err)
	}
}

func TestCorpusEvidenceLensAcceptsAdditiveUnknownField(t *testing.T) {
	bom := validCorpusBOM()
	canonical, err := json.Marshal(bom)
	if err != nil {
		t.Fatal(err)
	}
	data := append([]byte(nil), canonical[:len(canonical)-1]...)
	data = append(data, []byte(`,"future_addition":{"accepted":true}}`)...)
	lens, err := LensCorpusBOM(data)
	if err != nil {
		t.Fatalf("LensCorpusBOM() additive field error = %v", err)
	}
	if lens.BOMSHA256 != digestBytes(canonical) {
		t.Fatalf("typed BOM digest = %s, want %s", lens.BOMSHA256, digestBytes(canonical))
	}
}

func TestCorpusEvidenceLensRejectsDuplicateField(t *testing.T) {
	data, err := json.Marshal(validCorpusBOM())
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte(`"kind":"openwaldo-bom"`), []byte(`"kind":"openwaldo-bom","kind":"openwaldo-bom"`), 1)
	if _, err := LensCorpusBOM(data); err == nil || !strings.Contains(err.Error(), "duplicate JSON key") {
		t.Fatalf("LensCorpusBOM() duplicate error = %v", err)
	}
}

func TestCorpusEvidenceLensReportsDirtyAndUnpinnedIndex(t *testing.T) {
	bom := validCorpusBOM()
	bom.Index.Dirty = true
	data, _ := json.Marshal(bom)
	lens, err := LensCorpusBOM(data)
	if err != nil {
		t.Fatal(err)
	}
	if lens.Index.State != "DIRTY_MANIFEST_PINNED" {
		t.Fatalf("dirty state = %q", lens.Index.State)
	}
	bom.Index.Commit = ""
	data, _ = json.Marshal(bom)
	lens, err = LensCorpusBOM(data)
	if err != nil {
		t.Fatal(err)
	}
	if lens.Index.State != "MANIFEST_PINNED_NO_COMMIT" {
		t.Fatalf("unpinned state = %q", lens.Index.State)
	}
}

func TestCorpusEvidenceLensRejectsAggregateDrift(t *testing.T) {
	bom := validCorpusBOM()
	bom.Totals.Tokens++
	data, _ := json.Marshal(bom)
	if _, err := LensCorpusBOM(data); err == nil || !strings.Contains(err.Error(), "totals") {
		t.Fatalf("LensCorpusBOM() aggregate error = %v", err)
	}
}

func TestCorpusEvidenceLensRejectsUnknownShardSource(t *testing.T) {
	bom := validCorpusBOM()
	bom.Shards[0].Sources = []string{"not-declared"}
	data, _ := json.Marshal(bom)
	if _, err := LensCorpusBOM(data); err == nil || !strings.Contains(err.Error(), "unknown or duplicate source") {
		t.Fatalf("LensCorpusBOM() source error = %v", err)
	}
}

func TestCorpusEvidenceLensRejectsPolicyViolation(t *testing.T) {
	bom := validCorpusBOM()
	bom.Policy.Exclude = []string{"CC0-*"}
	data, _ := json.Marshal(bom)
	if _, err := LensCorpusBOM(data); err == nil || !strings.Contains(err.Error(), "violates the BOM policy") {
		t.Fatalf("LensCorpusBOM() policy error = %v", err)
	}
}

func TestCorpusEvidenceLensSummarizesEmbeddedAttestation(t *testing.T) {
	bom := validCorpusBOM()
	shardBOM := &waldoShardBOM{
		Kind: "openwaldo-bom", Schema: 1, Subject: "shard", PlanSHA256: repeatHex("f"),
		RecordSchema: 1, WriterRecipe: waldoFormerTextBOMRecipe, Tokenizer: "byte",
		Records: 2, Tokens: 8, ContentBytes: 64, Licenses: []string{"CC0-1.0"},
		Validation: waldoShardValidation{CanonicalRecords: true, ContentHashes: true, TokenCounts: true, ExactLicenseDedup: true},
	}
	encoded, err := json.Marshal(shardBOM)
	if err != nil {
		t.Fatal(err)
	}
	bom.Shards[0].Attestation = &waldoShardAttestation{Status: "embedded", WriterRecipe: waldoFormerTextBOMRecipe, BOMSHA256: digestBytes(encoded), BOM: shardBOM}
	data, _ := json.Marshal(bom)
	lens, err := LensCorpusBOM(data)
	if err != nil {
		t.Fatalf("LensCorpusBOM() error = %v", err)
	}
	if lens.Attestation.State != "RECORDED_COMPLETE" || lens.Attestation.Embedded != 1 || lens.Attestation.NotRecorded != 0 {
		t.Fatalf("embedded attestation = %+v", lens.Attestation)
	}
}

func TestCorpusEvidenceLensWitnessesCurrentAssessmentFilterAndPrivacyContract(t *testing.T) {
	bom := validCurrentCorpusBOM(t)
	data, err := json.MarshalIndent(bom, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	lens, err := LensCorpusBOM(data)
	if err != nil {
		t.Fatalf("LensCorpusBOM() current contract error = %v", err)
	}
	canonical, err := json.Marshal(bom)
	if err != nil {
		t.Fatal(err)
	}
	if lens.BOMSHA256 != digestBytes(canonical) || lens.RecordFilter == nil || lens.RecordFilter.State != "DECLARED" || lens.RecordFilter.Schema != 1 || !lens.RecordFilter.GlobalDeclared {
		t.Fatalf("current filter evidence = %+v", lens.RecordFilter)
	}
	if lens.Assessment == nil || lens.Assessment.State != "RECORDED_COMPLETE" || lens.Assessment.AssessedShards != 1 || lens.Assessment.RepetitiveContentRecords != 1 {
		t.Fatalf("current assessment evidence = %+v", lens.Assessment)
	}
	if lens.PrivacyRedaction == nil || lens.PrivacyRedaction.State != "RECORDED_COMPLETE" || lens.PrivacyRedaction.Policy != waldoPrivacyRedactionPolicy || !lens.PrivacyRedaction.NamesRetained || lens.PrivacyRedaction.EmailAddresses != 1 || lens.PrivacyRedaction.IPAddresses != 1 {
		t.Fatalf("current privacy evidence = %+v", lens.PrivacyRedaction)
	}
	if lens.Attestation.State != "RECORDED_COMPLETE" || lens.Attestation.Embedded != 1 {
		t.Fatalf("current attestation evidence = %+v", lens.Attestation)
	}
}

func TestCorpusEvidenceLensAcceptsCompatibleSchemaTwoWriters(t *testing.T) {
	for _, recipe := range []string{waldoFormerMainContentRecipe, waldoFormerAssessmentRecipe} {
		t.Run(recipe, func(t *testing.T) {
			bom := validCurrentCorpusBOM(t)
			bom.Manifests[0].ConvertedBy.Recipe = recipe
			bom.Manifests[0].Redaction = nil
			bom.Shards[0].ConvertedBy.Recipe = recipe
			bom.Shards[0].Redaction = nil
			bom.Shards[0].Attestation.WriterRecipe = recipe
			bom.Shards[0].Attestation.BOM.WriterRecipe = recipe
			bom.Shards[0].Attestation.BOM.Redaction = waldoContentRedaction{}
			encoded, err := json.Marshal(bom.Shards[0].Attestation.BOM)
			if err != nil {
				t.Fatal(err)
			}
			bom.Shards[0].Attestation.BOMSHA256 = digestBytes(encoded)
			data, err := json.Marshal(bom)
			if err != nil {
				t.Fatal(err)
			}
			lens, err := LensCorpusBOM(data)
			if err != nil {
				t.Fatalf("LensCorpusBOM() compatible writer error = %v", err)
			}
			if lens.Assessment == nil || lens.Assessment.State != "RECORDED_COMPLETE" || lens.PrivacyRedaction == nil || lens.PrivacyRedaction.State != "NOT_RECORDED" || lens.PrivacyRedaction.UnredactedCompatibleShards != 1 {
				t.Fatalf("compatible writer evidence = assessment %+v privacy %+v", lens.Assessment, lens.PrivacyRedaction)
			}
		})
	}
}

func TestCorpusEvidenceLensRejectsMissingCurrentPrivacyEvidence(t *testing.T) {
	bom := validCurrentCorpusBOM(t)
	bom.Shards[0].Redaction = nil
	data, _ := json.Marshal(bom)
	if _, err := LensCorpusBOM(data); err == nil || !strings.Contains(err.Error(), "redaction") {
		t.Fatalf("LensCorpusBOM() missing privacy error = %v", err)
	}
}

func TestCorpusEvidenceLensRetainsStrictLegacyReceiptValidation(t *testing.T) {
	data, err := json.Marshal(validCorpusBOM())
	if err != nil {
		t.Fatal(err)
	}
	lens, err := LensCorpusBOM(data)
	if err != nil {
		t.Fatal(err)
	}
	lens.Schema = CorpusEvidenceLensSchemaV1
	lens.RecordFilter = nil
	lens.Assessment = nil
	lens.PrivacyRedaction = nil
	lens.ReceiptSHA256, err = corpusLensReceiptDigest(lens)
	if err != nil {
		t.Fatal(err)
	}
	if err := lens.Validate(); err != nil {
		t.Fatalf("legacy corpus evidence lens should remain valid: %v", err)
	}
	lens.RecordFilter = &CorpusRecordFilterEvidence{State: "NOT_DECLARED", PolicySHA256: repeatHex("a")}
	if err := lens.Validate(); err == nil || !strings.Contains(err.Error(), "legacy corpus evidence lens") {
		t.Fatalf("legacy receipt with v0.2 fields error = %v", err)
	}
}

func TestCorpusEvidenceLensRejectsWrongImplicitRecipe(t *testing.T) {
	bom := validCorpusBOM()
	bom.Shards[0].Attestation = &waldoShardAttestation{Status: "implicit-v4", WriterRecipe: "not-the-v4-recipe"}
	data, _ := json.Marshal(bom)
	if _, err := LensCorpusBOM(data); err == nil || !strings.Contains(err.Error(), "invalid implicit-v4") {
		t.Fatalf("LensCorpusBOM() implicit recipe error = %v", err)
	}
}

func validCorpusBOM() waldoCorpusBOM {
	measure := EvidenceMeasures{Shards: 1, Docs: 2, Tokens: 8, Bytes: 128}
	conversion := waldoConversion{Tool: "waldo", Version: "test", Profile: "text", Recipe: "fixture/v1"}
	manifest := waldoCorpusManifest{
		Path: "core/example.json", SHA256: repeatHex("a"), Name: "example", Title: "Example corpus",
		Description: "A bounded test corpus.", License: "CC0-1.0", Format: "parquet", RecordSchema: 1,
		ConvertedBy: conversion,
		Sources:     []waldoSource{{Name: "fixture", Source: "Fixture source", Version: "1", License: "CC0-1.0", URL: "https://example.invalid/source", SHA256: repeatHex("b")}},
		Totals:      measure, Licenses: map[string]EvidenceMeasures{"CC0-1.0": measure},
	}
	shard := waldoCorpusShard{
		Manifest: manifest.Path, URL: "https://example.invalid/shard.parquet", SHA256: repeatHex("c"), Format: "parquet",
		RecordSchema: 1, License: "CC0-1.0", Sources: []string{"fixture"}, ConvertedBy: conversion,
		Docs: 2, Tokens: 8, Bytes: 128,
	}
	return waldoCorpusBOM{
		Kind: "openwaldo-bom", Schema: 1, Subject: "corpus",
		Index: waldoCorpusIndex{Remote: "https://example.invalid/index.git", Commit: strings.Repeat("d", 40)},
		Paths: []string{""}, Manifests: []waldoCorpusManifest{manifest}, Shards: []waldoCorpusShard{shard},
		Totals: measure, Licenses: map[string]EvidenceMeasures{"CC0-1.0": measure},
	}
}

func validCurrentCorpusBOM(t *testing.T) waldoCorpusBOM {
	t.Helper()
	bom := validCorpusBOM()
	conversion := waldoConversion{Tool: "waldo", Version: "test", Profile: "text", Recipe: waldoTextWriterRecipe, Tokenizer: "byte"}
	assessment := &waldoContentAssessment{
		EmailAddresses:     &waldoDetectionMeasure{Detector: "waldo/email-address-v1", Records: 0},
		RepetitiveContent:  &waldoDetectionMeasure{Detector: "waldo/gopher-ngram-repetition-v1", Records: 1},
		BoilerplateContent: &waldoDetectionMeasure{Detector: "waldo/gopher-structural-duplication-v1", Records: 0},
	}
	redaction := &waldoContentRedaction{Policy: waldoPrivacyRedactionPolicy, NamesRetained: true, EmailAddresses: 1, IPAddresses: 1}
	bom.RecordFilter = &waldoRecordFilterPolicy{Schema: 1, Global: &waldoRecordFilter{MainContent: boolPointer(true)}}
	bom.Manifests[0].RecordSchema = waldoTextRecordSchema
	bom.Manifests[0].ConvertedBy = conversion
	bom.Manifests[0].Assessment = assessment
	bom.Manifests[0].Redaction = redaction
	bom.Shards[0].RecordSchema = waldoTextRecordSchema
	bom.Shards[0].ConvertedBy = conversion
	bom.Shards[0].Assessment = assessment
	bom.Shards[0].Redaction = redaction
	shardBOM := &waldoShardBOM{
		Kind: "openwaldo-bom", Schema: 1, Subject: "shard", PlanSHA256: repeatHex("f"),
		RecordSchema: waldoTextRecordSchema, WriterRecipe: waldoTextWriterRecipe, Tokenizer: "byte",
		Records: 2, Tokens: 8, ContentBytes: 64, RepetitiveContentRecords: 1,
		Redaction: *redaction, Licenses: []string{"CC0-1.0"},
		Validation: waldoShardValidation{CanonicalRecords: true, ContentHashes: true, TokenCounts: true, ExactLicenseDedup: true},
	}
	encoded, err := json.Marshal(shardBOM)
	if err != nil {
		t.Fatal(err)
	}
	bom.Shards[0].Attestation = &waldoShardAttestation{Status: "embedded", WriterRecipe: waldoTextWriterRecipe, BOMSHA256: digestBytes(encoded), BOM: shardBOM}
	return bom
}

func boolPointer(value bool) *bool {
	return &value
}
