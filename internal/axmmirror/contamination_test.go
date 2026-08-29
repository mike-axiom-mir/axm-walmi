package axmmirror

import "testing"

func TestContaminationGuardFindsExactOverlap(t *testing.T) {
	comparison := validComparison()
	comparison.Evaluation.RecordSHA256 = []string{repeatHex("1")}
	report, err := CheckContamination(comparison)
	if err != nil {
		t.Fatalf("CheckContamination() error = %v", err)
	}
	if report.State != ContaminationStateContaminated || len(report.Overlaps) != 1 {
		t.Fatalf("report = %+v", report)
	}
	if report.Overlaps[0].Dimension != "record_sha256" || report.Overlaps[0].Values[0] != repeatHex("1") {
		t.Fatalf("overlap = %+v", report.Overlaps[0])
	}
}

func TestContaminationGuardHoldsIncompleteInventories(t *testing.T) {
	comparison := validComparison()
	comparison.Training.Complete = false
	report, err := CheckContamination(comparison)
	if err != nil {
		t.Fatalf("CheckContamination() error = %v", err)
	}
	if report.State != ContaminationStateHold || len(report.Holds) != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestContaminationGuardClearsCompleteDisjointInventories(t *testing.T) {
	report, err := CheckContamination(validComparison())
	if err != nil {
		t.Fatalf("CheckContamination() error = %v", err)
	}
	if report.State != ContaminationStateClear || report.ComparisonSHA256 == "" || len(report.ComparedDimensions) != 4 || len(report.Overlaps) != 0 || len(report.Holds) != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestContaminationGuardHoldsNonComparableInventories(t *testing.T) {
	comparison := EvaluationComparison{
		Schema: EvaluationComparisonSchema,
		Training: EvidenceInventory{
			Complete: true, RecordSHA256: []string{repeatHex("1")},
		},
		Evaluation: EvidenceInventory{
			Complete: true, TextSHA256: []string{repeatHex("2")},
		},
		Authorship: AuthorshipDeclaration{State: "unknown"},
	}
	report, err := CheckContamination(comparison)
	if err != nil {
		t.Fatalf("CheckContamination() error = %v", err)
	}
	if report.State != ContaminationStateHold || len(report.ComparedDimensions) != 0 || len(report.Holds) < 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestContaminationComparisonDigestIgnoresSetOrder(t *testing.T) {
	first := validComparison()
	first.Training.RecordSHA256 = append(first.Training.RecordSHA256, repeatHex("7"))
	second := first
	second.Training.RecordSHA256 = []string{repeatHex("7"), repeatHex("1")}
	firstReport, err := CheckContamination(first)
	if err != nil {
		t.Fatal(err)
	}
	secondReport, err := CheckContamination(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstReport.ComparisonSHA256 != secondReport.ComparisonSHA256 {
		t.Fatalf("comparison digests differ: %s != %s", firstReport.ComparisonSHA256, secondReport.ComparisonSHA256)
	}
}

func TestContaminationGuardKeepsAuthorshipAsDeclaration(t *testing.T) {
	comparison := validComparison()
	comparison.Authorship = AuthorshipDeclaration{State: "outside-authored", EvidenceSHA256: repeatHex("9")}
	report, err := CheckContamination(comparison)
	if err != nil {
		t.Fatalf("CheckContamination() error = %v", err)
	}
	if report.Authorship.State != "RECORDED_ASSERTION" || report.Authorship.EvidenceSHA256 != repeatHex("9") {
		t.Fatalf("authorship = %+v", report.Authorship)
	}
}

func TestContaminationGuardRejectsDuplicateIdentity(t *testing.T) {
	comparison := validComparison()
	comparison.Training.RecordSHA256 = []string{repeatHex("1"), repeatHex("1")}
	if _, err := CheckContamination(comparison); err == nil {
		t.Fatal("CheckContamination() accepted duplicate identity")
	}
}

func TestContaminationGuardRejectsInvalidDigest(t *testing.T) {
	comparison := validComparison()
	comparison.Evaluation.TextSHA256 = []string{"not-a-digest"}
	if _, err := CheckContamination(comparison); err == nil {
		t.Fatal("CheckContamination() accepted invalid digest")
	}
}

func TestContaminationGuardRejectsNonCanonicalCorpusPath(t *testing.T) {
	comparison := validComparison()
	comparison.Training.CorpusPaths = []string{"training/../evaluation/questions.jsonl"}
	if _, err := CheckContamination(comparison); err == nil {
		t.Fatal("CheckContamination() accepted a non-canonical corpus path")
	}
}

func validComparison() EvaluationComparison {
	return EvaluationComparison{
		Schema: EvaluationComparisonSchema,
		Training: EvidenceInventory{
			Complete: true, RecordSHA256: []string{repeatHex("1")},
			TextSHA256: []string{repeatHex("2")}, SourceGroups: []string{"training-public-v1"},
			CorpusPaths: []string{"training/shard-0001.jsonl"},
		},
		Evaluation: EvidenceInventory{
			Complete: true, RecordSHA256: []string{repeatHex("3")},
			TextSHA256: []string{repeatHex("4")}, SourceGroups: []string{"heldout-private-v1"},
			CorpusPaths: []string{"evaluation/questions.jsonl"},
		},
		Authorship: AuthorshipDeclaration{State: "unknown"},
	}
}
