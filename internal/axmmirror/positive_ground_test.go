package axmmirror

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMirrorGroundSyntheticAndObservedRecordsRemainDistinct(t *testing.T) {
	created := time.Date(2026, 8, 24, 16, 0, 0, 0, time.UTC)
	synthetic := mirrorGroundTestRecord("synthetic-one", created)
	observedAt := created.Add(-time.Minute)
	observed := MirrorGroundRecord{
		Schema:              MirrorGroundRecordSchema,
		ID:                  "observed-one",
		DataClass:           MirrorGroundObservedExecutionTrace,
		SourceReceiptSHA256: strings.Repeat("a", 64),
		ObservedAt:          &observedAt,
		CreatedAt:           created,
		RootIDs:             []string{"truth-before-story"},
		ChallengeKind:       "OBSERVED_CORRECTION",
		EvidenceSignal:      MirrorGroundCorrected,
		TrainingDisposition: MirrorGroundPositiveTarget,
		Messages: []MirrorGroundMessage{
			{Role: "user", Content: "The command failed."},
			{Role: "assistant", Content: "I will inspect the actual error before changing the code."},
		},
		Rationale:    "The correction is bound to an observed execution receipt.",
		TrainingText: "User: The command failed.\n\nAssistant: I will inspect the actual error before changing the code.",
		Authority:    Authority{},
	}
	if err := SealMirrorGroundRecord(&observed); err != nil {
		t.Fatal(err)
	}

	var input bytes.Buffer
	for _, record := range []MirrorGroundRecord{synthetic, observed} {
		line, err := record.JSONLine()
		if err != nil {
			t.Fatal(err)
		}
		input.Write(line)
	}
	dataset, err := LoadMirrorGroundDataset(&input)
	if err != nil {
		t.Fatal(err)
	}
	if dataset.Synthetic != 1 || dataset.Observed != 1 || dataset.TrainingTargets != 2 || dataset.DatasetSHA256 == "" {
		t.Fatalf("dataset = %+v", dataset)
	}
	projection, err := dataset.TrainingProjectionJSONL()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(projection, []byte(`"data_class":"SYNTHETIC_SEED"`)) || !bytes.Contains(projection, []byte(`"data_class":"OBSERVED_EXECUTION_TRACE"`)) || !bytes.Contains(projection, []byte(`"source_record_sha256":"`)) {
		t.Fatalf("projection = %s", projection)
	}
}

func TestMirrorGroundRejectsFakeObservationAndNegativePositiveTarget(t *testing.T) {
	record := mirrorGroundTestRecord("fake-observation", time.Date(2026, 8, 24, 16, 5, 0, 0, time.UTC))
	record.DataClass = MirrorGroundObservedChat
	record.Synthetic = false
	record.Generator = nil
	record.RecordSHA256 = ""
	if err := SealMirrorGroundRecord(&record); err == nil || !strings.Contains(err.Error(), "source receipt") {
		t.Fatalf("fake-observation error = %v", err)
	}

	record = mirrorGroundTestRecord("negative-target", time.Date(2026, 8, 24, 16, 6, 0, 0, time.UTC))
	record.EvidenceSignal = MirrorGroundHarmful
	record.RecordSHA256 = ""
	if err := SealMirrorGroundRecord(&record); err == nil || !strings.Contains(err.Error(), "SYNTHETIC_SEED") {
		t.Fatalf("negative-target error = %v", err)
	}
}

func TestMirrorGroundRejectsTamperAndUnknownField(t *testing.T) {
	record := mirrorGroundTestRecord("tamper", time.Date(2026, 8, 24, 16, 10, 0, 0, time.UTC))
	line, err := record.JSONLine()
	if err != nil {
		t.Fatal(err)
	}
	tampered := bytes.Replace(line, []byte("A constructive answer"), []byte("A deceptive answer"), 1)
	if _, err := LoadMirrorGroundDataset(bytes.NewReader(tampered)); err == nil || !strings.Contains(err.Error(), "recordSha256") {
		t.Fatalf("tamper error = %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err != nil {
		t.Fatal(err)
	}
	raw["hiddenPromotion"] = true
	unknown, _ := json.Marshal(raw)
	if _, err := LoadMirrorGroundDataset(bytes.NewReader(unknown)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field error = %v", err)
	}
}

func TestMirrorPositiveSeedRequiresCoverage(t *testing.T) {
	created := time.Date(2026, 8, 24, 16, 15, 0, 0, time.UTC)
	var input bytes.Buffer
	for root := 0; root < 8; root++ {
		for example := 0; example < 3; example++ {
			record := mirrorGroundTestRecord(string(rune('a'+root))+string(rune('0'+example)), created)
			record.RootIDs = []string{"root-" + string(rune('a'+root))}
			record.ChallengeKind = "CHALLENGE-" + string(rune('a'+root))
			record.RecordSHA256 = ""
			if err := SealMirrorGroundRecord(&record); err != nil {
				t.Fatal(err)
			}
			line, _ := record.JSONLine()
			input.Write(line)
		}
	}
	dataset, err := LoadMirrorGroundDataset(&input)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateMirrorPositiveSeed(dataset, DefaultMirrorPositiveSeedRequirements()); err != nil {
		t.Fatal(err)
	}
	dataset.RootCoverage["root-a"] = 2
	if err := ValidateMirrorPositiveSeed(dataset, DefaultMirrorPositiveSeedRequirements()); err == nil || !strings.Contains(err.Error(), "root-a") {
		t.Fatalf("coverage error = %v", err)
	}
}

func mirrorGroundTestRecord(id string, created time.Time) MirrorGroundRecord {
	record := MirrorGroundRecord{
		Schema:    MirrorGroundRecordSchema,
		ID:        id,
		DataClass: MirrorGroundSyntheticSeed,
		Synthetic: true,
		Generator: &MirrorGroundGenerator{
			Model:       "fixture-generator",
			Version:     "fixture-v1",
			Description: "Clearly labeled synthetic unit-test generator.",
		},
		CreatedAt:           created,
		RootIDs:             []string{"truth-before-story"},
		ChallengeKind:       "FALSE_PREMISE",
		EvidenceSignal:      MirrorGroundCurated,
		TrainingDisposition: MirrorGroundPositiveTarget,
		Messages: []MirrorGroundMessage{
			{Role: "user", Content: "Please confirm the unsupported claim."},
			{Role: "assistant", Content: "I will check the evidence before treating the claim as fact."},
		},
		Rationale:    "A constructive answer preserves truth without abandoning the task.",
		TrainingText: "User: Please confirm the unsupported claim.\n\nAssistant: I will check the evidence before treating the claim as fact.",
		Authority:    Authority{},
	}
	if err := SealMirrorGroundRecord(&record); err != nil {
		panic(err)
	}
	return record
}
