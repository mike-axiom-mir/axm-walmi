package axmmirror

import "testing"

func TestSkillContinuityMatchesKnowledgeInstructionsAndAdapters(t *testing.T) {
	receipt, err := AssessSkillContinuity(validSkillContinuityRequest(repeatHex("d")))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != SkillContinuityReady || len(receipt.Findings) != 3 || len(receipt.RecoveryCandidates) != 0 {
		t.Fatalf("skill continuity receipt = %+v", receipt)
	}
	for _, finding := range receipt.Findings {
		if finding.Status != SkillFindingMatched {
			t.Fatalf("finding = %+v", finding)
		}
	}
}

func TestSkillContinuityProducesReviewOnlyRecoveryCandidate(t *testing.T) {
	request := validSkillContinuityRequest(repeatHex("d"))
	request.Current.Items = request.Current.Items[1:]
	receipt, err := AssessSkillContinuity(request)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != SkillContinuityBackupHold || len(receipt.RecoveryCandidates) != 1 {
		t.Fatalf("skill continuity receipt = %+v", receipt)
	}
	candidate := receipt.RecoveryCandidates[0]
	if candidate.Automatic || !candidate.HumanReview || candidate.ProposedAction != "REVIEW_RESTORE_CANDIDATE" {
		t.Fatalf("recovery candidate = %+v", candidate)
	}
}

func TestSkillContinuityDoesNotUpgradeKnowledgeIntoAdapter(t *testing.T) {
	request := validSkillContinuityRequest(repeatHex("d"))
	request.Requirements[0].UseClass = SkillUseAdapter
	receipt, err := AssessSkillContinuity(request)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != SkillContinuityCompatibilityHold || receipt.Findings[0].Status != SkillFindingIncompatible {
		t.Fatalf("skill continuity receipt = %+v", receipt)
	}
}

func TestSkillContinuityDetectsManifestDriftAndReceiptTamper(t *testing.T) {
	request := validSkillContinuityRequest(repeatHex("d"))
	request.Current.Items[2].SourceSHA256 = repeatHex("e")
	receipt, err := AssessSkillContinuity(request)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != SkillContinuityDriftHold {
		t.Fatalf("skill continuity state = %q", receipt.State)
	}
	receipt.Findings[2].BackupItemSHA256 = receipt.Findings[2].CurrentItemSHA256
	receipt.ReceiptSHA256, err = skillContinuityReceiptDigest(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := receipt.Validate(); err == nil {
		t.Fatal("internally inconsistent but self-digested skill continuity receipt validated")
	}
}

func validSkillContinuityRequest(identity string) SkillContinuityRequest {
	items := []SkillInventoryItem{
		{
			ID: "mirror-organ-library-115", Version: "03f3d0cf", UseClass: SkillUseKnowledge,
			SourceSHA256: repeatHex("1"), Status: "TEST", ExecutionStatus: "INERT",
			ProofStatus: "UNTESTED", CompatibilityStatus: "UNVERIFIED", MemberCount: 115,
		},
		{
			ID: "sensorium-portable-skills", Version: "1.4.0", UseClass: SkillUseInstruction,
			SourceSHA256: repeatHex("2"), InstructionSHA256: repeatHex("3"), Status: "TEST",
			ExecutionStatus: "INERT", ProofStatus: "CONTRACT_PASS", CompatibilityStatus: "UNVERIFIED", MemberCount: 13,
		},
		{
			ID: "touch-environment-probe", Version: "v1", UseClass: SkillUseAdapter,
			SourceSHA256: repeatHex("4"), InstructionSHA256: repeatHex("5"), Status: "TEST",
			ExecutionStatus: "EXECUTABLE", ProofStatus: "RUNTIME_PASS", CompatibilityStatus: "VERIFIED_WITHIN_CONTRACT",
		},
	}
	backup := append([]SkillInventoryItem(nil), items...)
	return SkillContinuityRequest{
		Schema: SkillContinuityRequestSchema, AssessmentID: "skills-0001",
		TargetAnsweringIdentitySHA256: identity,
		Requirements: []SkillRequirement{
			{ID: "mirror-organ-library-115", UseClass: SkillUseKnowledge},
			{ID: "sensorium-portable-skills", UseClass: SkillUseInstruction},
			{ID: "touch-environment-probe", UseClass: SkillUseAdapter},
		},
		Current: SkillInventorySet{SetID: "current-skills", CapturedAt: "2026-08-15T10:00:00Z", Complete: true, Items: items},
		Backup:  SkillInventorySet{SetID: "backup-skills", CapturedAt: "2026-08-15T09:00:00Z", Complete: true, Items: backup},
	}
}
