package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/openwaldo/waldo/internal/axmmirror"
)

func TestReadStrictJSONRejectsUnknownField(t *testing.T) {
	path := writeTemp(t, `{"known":"value","extra":true}`)
	var target struct {
		Known string `json:"known"`
	}
	if err := readStrictJSON(path, &target); err == nil {
		t.Fatal("readStrictJSON accepted unknown field")
	}
}

func TestReadStrictJSONRejectsTrailingValue(t *testing.T) {
	path := writeTemp(t, `{"known":"value"} {"second":true}`)
	var target struct {
		Known string `json:"known"`
	}
	if err := readStrictJSON(path, &target); err == nil {
		t.Fatal("readStrictJSON accepted trailing JSON value")
	}
}

func TestReadStrictJSONRejectsDuplicateField(t *testing.T) {
	path := writeTemp(t, `{"known":"first","known":"second"}`)
	var target struct {
		Known string `json:"known"`
	}
	if err := readStrictJSON(path, &target); err == nil {
		t.Fatal("readStrictJSON accepted duplicate field")
	}
}

func TestWriteJSONRefusesExistingOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipt.json")
	if err := os.WriteFile(path, []byte("original\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(path, map[string]string{"state": "replacement"}); err == nil {
		t.Fatal("writeJSON() overwrote an existing receipt")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original\n" {
		t.Fatalf("existing output changed to %q", data)
	}
}

func TestContaminationFileWritesBlockingReport(t *testing.T) {
	comparison := axmmirror.EvaluationComparison{
		Schema: axmmirror.EvaluationComparisonSchema,
		Training: axmmirror.EvidenceInventory{
			Complete: true, RecordSHA256: []string{repeat("a")},
		},
		Evaluation: axmmirror.EvidenceInventory{
			Complete: true, RecordSHA256: []string{repeat("a")},
		},
		Authorship: axmmirror.AuthorshipDeclaration{State: "unknown"},
	}
	data, err := json.Marshal(comparison)
	if err != nil {
		t.Fatal(err)
	}
	input := writeTemp(t, string(data))
	output := filepath.Join(t.TempDir(), "report.json")
	if err := contaminationFile(input, output); err == nil {
		t.Fatal("contaminationFile() did not block an overlap")
	}
	var report axmmirror.ContaminationReport
	if err := readStrictJSON(output, &report); err != nil {
		t.Fatalf("read report: %v", err)
	}
	if report.State != axmmirror.ContaminationStateContaminated {
		t.Fatalf("report state = %q", report.State)
	}
}

func TestLensCorpusFileWritesReadyReceipt(t *testing.T) {
	input := filepath.Join("..", "..", "examples", "axm-mirror", "corpus-bom.json")
	output := filepath.Join(t.TempDir(), "corpus-lens.json")
	if err := lensCorpusFile(input, output); err != nil {
		t.Fatalf("lensCorpusFile() error = %v", err)
	}
	var lens axmmirror.CorpusEvidenceLens
	if err := readStrictJSON(output, &lens); err != nil {
		t.Fatalf("read corpus lens: %v", err)
	}
	if lens.State != axmmirror.CorpusEvidenceStateReady {
		t.Fatalf("corpus lens state = %q", lens.State)
	}
}

func TestInnerAssetCLIFlow(t *testing.T) {
	input := filepath.Join("..", "..", "examples", "axm-mirror", "inner-asset-recipe.json")
	output := filepath.Join(t.TempDir(), "witness-orb.axmasset")
	if err := forgeInnerAssetFile(input, output); err != nil {
		t.Fatalf("forgeInnerAssetFile() error = %v", err)
	}
	if err := verifyInnerAssetFile(output); err != nil {
		t.Fatalf("verifyInnerAssetFile() error = %v", err)
	}
	if err := forgeInnerAssetFile(input, output); err == nil {
		t.Fatal("forgeInnerAssetFile() overwrote an existing candidate")
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := axmmirror.VerifyInnerAssetBundle(data)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.State != axmmirror.InnerAssetStateReady || !candidate.CandidateOnly || candidate.VisualStatus != axmmirror.InnerAssetVisualPending {
		t.Fatalf("inner asset candidate = %+v", candidate)
	}
}

func TestPortableCapabilitySpineCLIFlow(t *testing.T) {
	directory := t.TempDir()
	censusRequest := axmmirror.SelfCapabilityCensusRequest{
		Schema: axmmirror.SelfCapabilityCensusRequestSchema, CensusID: "cli-self-census-0001",
		TargetAnsweringIdentitySHA256: repeat("1"), ObservedBuildSHA256: repeat("2"),
		ObservationReceiptSHA256: repeat("3"), CapturedAt: "2026-08-15T10:00:00Z", Authority: axmmirror.Authority{},
	}
	selfPath := filepath.Join(directory, "self-capabilities.json")
	if err := censusCapabilitiesFile(writeValueTemp(t, censusRequest), selfPath); err != nil {
		t.Fatalf("censusCapabilitiesFile() error = %v", err)
	}
	var self axmmirror.SelfCapabilitySnapshot
	if err := readStrictJSON(selfPath, &self); err != nil {
		t.Fatal(err)
	}

	externalSnapshot := axmmirror.ExternalCapabilitySnapshot{
		Schema: axmmirror.ExternalCapabilitySnapshotSchema, SnapshotID: "cli-external-live-0001",
		Source: axmmirror.CapabilitySnapshotSource{
			Repository: "example/platform", Branch: "main", Commit: repeat40("a"),
			Documents: []axmmirror.CapabilitySourceDocument{{Path: "tools/render-hand/module.contract.json", GitBlobSHA: repeat40("b"), SHA256: repeat("c")}},
		},
		ObservationState: axmmirror.ExternalObservationLive, ObservedAt: "2026-08-15T10:00:00Z", AssessedAt: "2026-08-15T10:00:00Z",
		TTLMillis: 60_000, InventoryComplete: true, RuntimeEvidenceSHA256: repeat("d"),
		Capabilities: []axmmirror.ExternalCapability{{
			ID: "asset.external.render", ProviderID: "render-hand", Version: "v1", ExecutionClass: "EXTERNAL_HAND",
			DeclarationState: axmmirror.ExternalDeclarationAvailable, InputSchemas: []string{"axm.asset-brief/v1"},
			OutputSchemas: []string{"image/png"}, AcceptedDataClasses: []string{axmmirror.CapabilityDataPublic},
		}},
		Authority: axmmirror.Authority{},
	}
	externalPath := filepath.Join(directory, "external-capabilities.json")
	if err := intakeCapabilitiesFile(writeValueTemp(t, externalSnapshot), externalPath); err != nil {
		t.Fatalf("intakeCapabilitiesFile() error = %v", err)
	}

	request := axmmirror.CapabilityGapRequest{
		Schema: axmmirror.CapabilityGapRequestSchema, RequestID: "cli-gap-0001",
		TargetAnsweringIdentitySHA256: self.TargetAnsweringIdentitySHA256, CapabilityID: "asset.external.render",
		Input:                axmmirror.CapabilityArtifactRef{Schema: "axm.asset-brief/v1", SHA256: repeat("4"), Bytes: 512},
		RequiredOutputSchema: "image/png", DataClass: axmmirror.CapabilityDataPublic, MaxOutputBytes: 4096,
		CreatedAt: "2026-08-15T10:00:01Z", ExpiresAt: "2026-08-15T10:10:00Z", Authority: axmmirror.Authority{},
	}
	planPath := filepath.Join(directory, "handoff-plan.json")
	if err := planHandoffFile(selfPath, externalPath, writeValueTemp(t, request), planPath); err != nil {
		t.Fatalf("planHandoffFile() error = %v", err)
	}
	var plan axmmirror.CapabilityHandoffPlan
	if err := readStrictJSON(planPath, &plan); err != nil {
		t.Fatal(err)
	}
	if plan.State != axmmirror.HandoffStateReady || plan.ProviderSelected || plan.ExternalExecutionAllowed {
		t.Fatalf("handoff plan = %+v", plan)
	}

	translationDeclaration := axmmirror.CapabilityTranslationDeclaration{
		Schema: axmmirror.CapabilityTranslationDeclarationSchema, TranslationID: "cli-translation-0001",
		PlanSHA256: plan.PlanSHA256, ProviderID: plan.UniqueCandidateProviderID,
		InputFromSchema: request.Input.Schema, InputToSchema: request.Input.Schema,
		OutputFromSchema: request.RequiredOutputSchema, OutputToSchema: request.RequiredOutputSchema,
		DeclaredAt: "2026-08-15T10:00:01Z", Authority: axmmirror.Authority{},
	}
	translationPath := filepath.Join(directory, "translation-receipt.json")
	if err := sealTranslationFile(planPath, writeValueTemp(t, translationDeclaration), translationPath); err != nil {
		t.Fatalf("sealTranslationFile() error = %v", err)
	}
	var translation axmmirror.TranslationLossReceipt
	if err := readStrictJSON(translationPath, &translation); err != nil {
		t.Fatal(err)
	}

	returnDraft := axmmirror.CapabilityReturnDraft{
		Schema: axmmirror.CapabilityReturnDraftSchema, ReturnID: "cli-return-0001", PlanSHA256: plan.PlanSHA256,
		TranslationReceiptSHA256: translation.ReceiptSHA256, ProviderID: plan.UniqueCandidateProviderID, Input: request.Input,
		Outputs: []axmmirror.CapabilityOutputArtifact{{
			ID: "primary", CapabilityArtifactRef: axmmirror.CapabilityArtifactRef{Schema: "image/png", SHA256: repeat("5"), Bytes: 1024},
		}},
		ProviderReceiptSHA256: repeat("6"), StartedAt: "2026-08-15T10:00:02Z", CompletedAt: "2026-08-15T10:00:03Z",
		AssessedAt: "2026-08-15T10:00:04Z", RuntimeState: axmmirror.CapabilityRuntimeComplete,
		PermissionState: axmmirror.CapabilityPermissionConfirmed, CleanupState: axmmirror.CapabilityCleanupConfirmed,
		Authority: axmmirror.Authority{},
	}
	returnPath := filepath.Join(directory, "return-receipt.json")
	if err := verifyHandoffReturnFile(planPath, translationPath, writeValueTemp(t, returnDraft), returnPath); err != nil {
		t.Fatalf("verifyHandoffReturnFile() error = %v", err)
	}
	var returned axmmirror.CapabilityReturnReceipt
	if err := readStrictJSON(returnPath, &returned); err != nil {
		t.Fatal(err)
	}
	if returned.State != axmmirror.CapabilityReturnVerified {
		t.Fatalf("return receipt state = %q", returned.State)
	}
}

func TestVerifierEvolutionRepairAndToolMemoryCLIFlows(t *testing.T) {
	directory := t.TempDir()
	registry, err := axmmirror.NewVerifierRegistry("cli-waldo-verifiers", []axmmirror.VerifierDefinition{
		{ID: "schema-guard", Version: "v0.1", Class: axmmirror.VerifierClassSchemaInvariant, IndependenceGroup: "kernel", DefinitionSHA256: repeat("1"), TestPackSHA256: repeat("2"), ImplementationSHA256: repeat("3"), Protected: true},
		{ID: "provenance-peer", Version: "v0.1", Class: axmmirror.VerifierClassProvenanceBinding, IndependenceGroup: "provenance", DefinitionSHA256: repeat("4"), TestPackSHA256: repeat("5"), ImplementationSHA256: repeat("6")},
		{ID: "authority-peer", Version: "v0.1", Class: axmmirror.VerifierClassAuthority, IndependenceGroup: "authority", DefinitionSHA256: repeat("7"), TestPackSHA256: repeat("8"), ImplementationSHA256: repeat("9")},
		{ID: "privacy-boundary", Version: "v0.1", Class: axmmirror.VerifierClassPrivacyBoundary, IndependenceGroup: "privacy", DefinitionSHA256: repeat("a"), TestPackSHA256: repeat("b"), ImplementationSHA256: repeat("c")},
	})
	if err != nil {
		t.Fatal(err)
	}
	registryPath := writeValueTemp(t, registry)
	request := cliVerifierChangeRequest(t, registry)
	requestPath := writeValueTemp(t, request)
	readyPath := filepath.Join(directory, "verifier-ready.json")
	if err := assessVerifierChangeFile(registryPath, requestPath, readyPath); err != nil {
		t.Fatalf("assessVerifierChangeFile() error = %v", err)
	}
	nextPath := filepath.Join(directory, "verifier-next.json")
	if err := materializeVerifierChangeFile(registryPath, readyPath, nextPath); err != nil {
		t.Fatalf("materializeVerifierChangeFile() error = %v", err)
	}
	var next axmmirror.VerifierRegistry
	if err := readStrictJSON(nextPath, &next); err != nil {
		t.Fatal(err)
	}
	if next.Generation != 2 || next.PreviousRegistrySHA256 != registry.RegistrySHA256 {
		t.Fatalf("next registry = %+v", next)
	}

	failedRequest := request
	failedRequest.PeerReviews = append([]axmmirror.VerifierPeerReview(nil), request.PeerReviews...)
	failedRequest.PeerReviews[0].State = axmmirror.VerifierPeerFail
	failedRequest.PeerReviews[0].RegressionCount = 1
	failedRequest.PeerReviews[0].Finding = "regression found"
	failedPath := filepath.Join(directory, "verifier-failed.json")
	if err := assessVerifierChangeFile(registryPath, writeValueTemp(t, failedRequest), failedPath); err == nil {
		t.Fatal("failed verifier change returned success")
	}
	var failed axmmirror.VerifierChangeReceipt
	if err := readStrictJSON(failedPath, &failed); err != nil {
		t.Fatal(err)
	}
	repairRequest := axmmirror.RepairBuddyRequest{
		Schema: axmmirror.RepairBuddyRequestSchema, RepairID: "cli-repair-0001",
		FailedChangeReceiptSHA256: failed.ReceiptSHA256, IncidentEvidenceSHA256: repeat("d"),
		ObservedAt: "2026-08-16T07:00:00Z", MaxAttempts: 2, Authority: axmmirror.Authority{},
	}
	repairPath := filepath.Join(directory, "repair-plan.json")
	if err := planVerifierRepairFile(registryPath, failedPath, writeValueTemp(t, repairRequest), repairPath); err != nil {
		t.Fatalf("planVerifierRepairFile() error = %v", err)
	}
	var repair axmmirror.RepairBuddyPlan
	if err := readStrictJSON(repairPath, &repair); err != nil {
		t.Fatal(err)
	}
	if repair.State != axmmirror.RepairBuddyCandidateReady || repair.AutomaticRepair {
		t.Fatalf("repair plan = %+v", repair)
	}

	experienceDraft := axmmirror.IdentityToolExperience{
		Schema: axmmirror.IdentityToolExperienceSchema, MemoryID: "cli-render-memory", ExperienceID: "cli-experience-0001",
		TargetAnsweringIdentitySHA256: repeat("e"), ToolID: "image-render-hand", ToolVersion: "v1",
		TaskClass: "asset-render", UseTags: []string{"transparent-background", "pixel-art"},
		InputArtifactSHA256: repeat("1"), OutputArtifactSHA256: repeat("2"), ToolReceiptSHA256: repeat("3"),
		IndependentVerificationSHA256: repeat("4"), Outcome: axmmirror.ToolOutcomeSuccess,
		VerificationVerdict: axmmirror.ToolVerificationSuccess, WisdomClass: axmmirror.ToolWisdomReuse,
		Wisdom:     "Reuse the bounded transparent-background recipe when the exact pixel-art constraints match.",
		ObservedAt: "2026-08-16T07:00:00Z", TTLMillis: 86_400_000,
		ContentClass: axmmirror.ToolMemoryContentPublicSafe, Authority: axmmirror.Authority{},
	}
	sealedExperiencePath := filepath.Join(directory, "experience.json")
	if err := sealToolExperienceFile(writeValueTemp(t, experienceDraft), sealedExperiencePath); err != nil {
		t.Fatalf("sealToolExperienceFile() error = %v", err)
	}
	shardPath := filepath.Join(directory, "tool-memory.json")
	if err := startToolMemoryFile(sealedExperiencePath, shardPath); err != nil {
		t.Fatalf("startToolMemoryFile() error = %v", err)
	}
	query := axmmirror.IdentityWisdomQuery{
		Schema: axmmirror.IdentityWisdomQuerySchema, QueryID: "cli-query-0001",
		TargetAnsweringIdentitySHA256: repeat("e"), ToolID: "image-render-hand", ToolVersion: "v1",
		TaskClass: "asset-render", UseTags: []string{"pixel-art"}, AsOf: "2026-08-16T07:01:00Z",
		MaxEntries: 2, Authority: axmmirror.Authority{},
	}
	viewPath := filepath.Join(directory, "wisdom-view.json")
	if err := recallToolWisdomFile(shardPath, writeValueTemp(t, query), viewPath); err != nil {
		t.Fatalf("recallToolWisdomFile() error = %v", err)
	}
	var view axmmirror.IdentityWisdomView
	if err := readStrictJSON(viewPath, &view); err != nil {
		t.Fatal(err)
	}
	if view.State != axmmirror.ToolWisdomReady || len(view.Selected) != 1 {
		t.Fatalf("wisdom view = %+v", view)
	}
}

func cliVerifierChangeRequest(t *testing.T, registry axmmirror.VerifierRegistry) axmmirror.VerifierChangeRequest {
	t.Helper()
	definitions := map[string]axmmirror.VerifierDefinition{}
	for _, definition := range registry.Verifiers {
		definitions[definition.ID] = definition
	}
	target := definitions["privacy-boundary"]
	candidate := target
	candidate.Version = "v0.2"
	candidate.DefinitionSHA256 = repeat("d")
	intent, err := axmmirror.SealVerifierChangeIntent(axmmirror.VerifierChangeIntent{
		IntentID: "cli-intent-0001", TargetVerifierID: target.ID,
		BaselineDefinitionSHA256: cliJSONDigest(t, target), Candidate: candidate,
		Reason: "tighten the privacy verifier with frozen replay evidence", SourceEvidenceSHA256: repeat("e"),
		ShadowProtocol: axmmirror.VerifierShadowProtocol{ProtocolID: "cli-shadow-v1", FixtureSetSHA256: repeat("f"), CaseCount: 16},
		Authority:      axmmirror.Authority{},
	})
	if err != nil {
		t.Fatal(err)
	}
	reviews := make([]axmmirror.VerifierPeerReview, 0, 2)
	for i, id := range []string{"authority-peer", "provenance-peer"} {
		reviewer := definitions[id]
		reviews = append(reviews, axmmirror.VerifierPeerReview{
			ReviewerID: reviewer.ID, ReviewerVersion: reviewer.Version, ReviewerDefinitionSHA256: cliJSONDigest(t, reviewer),
			ReviewerIndependenceGroup: reviewer.IndependenceGroup, IntentSHA256: intent.IntentSHA256,
			ProtocolID: intent.ShadowProtocol.ProtocolID, FixtureSetSHA256: intent.ShadowProtocol.FixtureSetSHA256,
			BaselineResultSetSHA256: repeat("1"), CandidateResultSetSHA256: repeat("2"),
			EvidenceSHA256: repeat(fmt.Sprintf("%d", i+3)), State: axmmirror.VerifierPeerPass,
			Finding: "exact frozen replay passed", Authority: axmmirror.Authority{},
		})
	}
	return axmmirror.VerifierChangeRequest{
		Schema: axmmirror.VerifierChangeRequestSchema, ChangeID: "cli-change-0001",
		CurrentRegistrySHA256: registry.RegistrySHA256, Intent: intent, PeerReviews: reviews, Authority: axmmirror.Authority{},
	}
}

func cliJSONDigest(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	return fmt.Sprintf("%x", digest)
}

func TestWitnessRunFileWritesHeldLifecycleReceipt(t *testing.T) {
	runBOM := filepath.Join("..", "..", "examples", "axm-mirror", "run-bom.json")
	run := writeTemp(t, `{
  "kind":"waldo-training-run",
  "schema":1,
  "id":"example-real-run-0001",
  "state":"planned",
  "bom_sha256":"176558b03e78383529a40fbdee15849d53a96c6272171aeb344df06926aaf0b6",
  "planned":"2026-08-15T03:00:00Z"
}`)
	output := filepath.Join(t.TempDir(), "run-witness.json")
	if err := witnessRunFile(runBOM, run, output); err == nil {
		t.Fatal("witnessRunFile() did not return a HOLD error")
	}
	var witness axmmirror.TrainingRunWitness
	if err := readStrictJSON(output, &witness); err != nil {
		t.Fatalf("read run witness: %v", err)
	}
	if witness.State != axmmirror.RunWitnessStateHold || witness.RunState != "planned" || len(witness.Holds) == 0 {
		t.Fatalf("run witness = %+v", witness)
	}
}

func TestGatedCloneFoundationCLIFlow(t *testing.T) {
	examples := filepath.Join("..", "..", "examples", "axm-mirror")
	directory := t.TempDir()
	runBOM := filepath.Join(examples, "run-bom.json")
	run := filepath.Join(examples, "run.json")
	witnessPath := filepath.Join(directory, "run-witness.json")
	if err := witnessRunFile(runBOM, run, witnessPath); err != nil {
		t.Fatalf("witnessRunFile() error = %v", err)
	}
	profilePath := filepath.Join(directory, "profile-contract.json")
	if err := profileContractFile(runBOM, witnessPath, profilePath); err != nil {
		t.Fatalf("profileContractFile() error = %v", err)
	}
	anchorPath := filepath.Join(directory, "anchor.json")
	if err := anchorFile(filepath.Join(examples, "model-release-bom.json"), anchorPath); err != nil {
		t.Fatalf("anchorFile() error = %v", err)
	}
	contaminationPath := filepath.Join(directory, "contamination.json")
	if err := contaminationFile(filepath.Join(examples, "evaluation-comparison.json"), contaminationPath); err != nil {
		t.Fatalf("contaminationFile() error = %v", err)
	}

	var witness axmmirror.TrainingRunWitness
	if err := readStrictJSON(witnessPath, &witness); err != nil {
		t.Fatal(err)
	}
	var profile axmmirror.TrainingProfileContract
	if err := readStrictJSON(profilePath, &profile); err != nil {
		t.Fatal(err)
	}
	var anchor axmmirror.OriginAnchor
	if err := readStrictJSON(anchorPath, &anchor); err != nil {
		t.Fatal(err)
	}
	var contamination axmmirror.ContaminationReport
	if err := readStrictJSON(contaminationPath, &contamination); err != nil {
		t.Fatal(err)
	}
	lock, err := axmmirror.CompareIdentity(anchor, anchor)
	if err != nil {
		t.Fatal(err)
	}
	request := axmmirror.ProvenanceContextRequest{
		Schema: axmmirror.ProvenanceContextRequestSchema, RequestID: "cli-context-0001",
		MaxFactBytes: axmmirror.MaxProvenanceFactBytes,
		Fields: []string{
			"answering.identity_sha256", "corpus.paths", "corpus.licenses", "training.profile.canonical",
			"evaluation.contamination_state", "evaluation.comparison_sha256",
		},
		Anchor: anchor, Corpus: witness.Corpus, RunWitness: &witness, ProfileContract: &profile,
		IdentityLock: &lock, Contamination: &contamination,
	}
	requestPath := writeValueTemp(t, request)
	contextPath := filepath.Join(directory, "context.json")
	if err := contextFile(requestPath, contextPath); err != nil {
		t.Fatalf("contextFile() error = %v", err)
	}
	var context axmmirror.ProvenanceContextPacket
	if err := readStrictJSON(contextPath, &context); err != nil {
		t.Fatal(err)
	}

	submission := axmmirror.SourceClaimSubmission{
		Schema: axmmirror.SourceClaimSubmissionSchema, SubmissionID: "cli-claims-0001",
		OutputSHA256: repeat("a"), ContextPacketSHA256: context.PacketSHA256,
		Claims: []axmmirror.SourceClaim{
			{ID: "identity", Kind: axmmirror.ClaimKindAnsweringIdentity, Value: anchor.AnsweringIdentitySHA256},
			{ID: "license", Kind: axmmirror.ClaimKindLicenseAssertion, Value: "CC0-1.0"},
			{ID: "path", Kind: axmmirror.ClaimKindCorpusPathMembership, Value: "research/example-corpus"},
			{ID: "profile", Kind: axmmirror.ClaimKindTrainingProfile, Value: "causal-pretrain-shuffled"},
			{ID: "evaluation", Kind: axmmirror.ClaimKindEvaluationIndependent, Value: axmmirror.ClaimValueEvaluationClear},
		},
	}
	submissionPath := writeValueTemp(t, submission)
	assessmentPath := filepath.Join(directory, "assessment.json")
	if err := gateClaimsFile(contextPath, submissionPath, assessmentPath); err != nil {
		t.Fatalf("gateClaimsFile() error = %v", err)
	}
	var assessment axmmirror.SourceClaimAssessment
	if err := readStrictJSON(assessmentPath, &assessment); err != nil {
		t.Fatal(err)
	}
	if assessment.State != axmmirror.ClaimGateStateConfirmed {
		t.Fatalf("assessment state = %q", assessment.State)
	}

	draft := axmmirror.EvaluationProtocolDraft{
		Schema: axmmirror.EvaluationProtocolDraftSchema, ProtocolID: "cli-protocol-0001",
		Pack: axmmirror.EvaluationPackIdentity{
			SHA256: repeat("1"), DocumentSHA256: repeat("2"), CaseCount: 2,
			CaseOrderSHA256: repeat("3"), AuthorshipState: "outside-authored",
			AuthorshipEvidenceSHA256: repeat("6"), AnswerKeySHA256: repeat("4"),
		},
		TargetAnsweringIdentitySHA256: anchor.AnsweringIdentitySHA256,
		Anchor:                        anchor, Context: context, Contamination: contamination,
		RequestSetSHA256: repeat("5"), AllowedMetrics: []string{"exact-match", "refusal-state"},
		ComparisonDimensions: []string{"output", "verification"}, AnswerKeyVisibility: axmmirror.AnswerKeyWithheld,
		Limits:          axmmirror.EvaluationExecutionLimits{MaxCases: 2, MaxOutputBytes: 4096, TimeoutMillis: 30000},
		PermissionState: "allowed", RequestedAuthority: axmmirror.Authority{},
	}
	draftPath := writeValueTemp(t, draft)
	sealPath := filepath.Join(directory, "protocol-seal.json")
	if err := sealEvaluationFile(draftPath, sealPath); err != nil {
		t.Fatalf("sealEvaluationFile() error = %v", err)
	}
	var seal axmmirror.EvaluationProtocolSeal
	if err := readStrictJSON(sealPath, &seal); err != nil {
		t.Fatal(err)
	}
	if seal.State != axmmirror.ProtocolStateReady {
		t.Fatalf("protocol state = %q", seal.State)
	}
	behaviorDraft := axmmirror.BehaviorEvidenceDraft{
		Schema: axmmirror.BehaviorEvidenceSchema, ExperimentID: seal.ProtocolID,
		WALDO: axmmirror.WALDOLineage{
			CorpusBOMSHA256:  witness.Corpus.BOMSHA256,
			RunBOMSHA256:     witness.RunBOMSHA256,
			ReleaseBOMSHA256: anchor.BOMSHA256,
		},
		ContextSHA256: context.PacketSHA256, RequestSHA256: seal.RequestSetSHA256,
		OutputSHA256: assessment.OutputSHA256, PermissionState: "allowed", OutcomeState: "pass",
		Authority: axmmirror.Authority{},
	}
	behaviorDraftPath := writeValueTemp(t, behaviorDraft)
	sealedPath := filepath.Join(directory, "gated-behavior.json")
	if err := sealGatedFile(anchorPath, witnessPath, profilePath, contextPath, assessmentPath, sealPath, behaviorDraftPath, sealedPath); err != nil {
		t.Fatalf("sealGatedFile() error = %v", err)
	}
	var sealed axmmirror.SealedBehaviorEvidence
	if err := readStrictJSON(sealedPath, &sealed); err != nil {
		t.Fatal(err)
	}
	if err := sealed.Verify(); err != nil {
		t.Fatalf("gated behavior verification: %v", err)
	}

	// Add the situated layer without changing the earlier provenance packet or
	// gated behavior schemas.
	subject := axmmirror.EvidenceSubject{ID: "cli-case-0001", Kind: "evaluation-case", SHA256: repeat("c")}
	sensoryDraft := axmmirror.SensoryEvidenceDraft{
		Schema: axmmirror.SensoryEvidenceDraftSchema, ObservationID: "cli-touch-0001", ClaimID: "cli-claim-0001",
		SenseID: "touch-environment-probe", Capability: "sense.environment.touch/v1", Subject: subject,
		SeatID: "workshop-seat", BackendID: "cli-fixture", ObservedAt: "2026-08-15T10:00:00Z",
		SealedAt: "2026-08-15T10:00:00.100Z", AssessedAt: "2026-08-15T10:00:00.200Z", TTLMillis: 60_000,
		ObservationState: axmmirror.SensoryObservationObserved, Verdict: "PASS",
		TypedObservationSHA256: repeat("d"), SpecificReceiptSchema: "axm.touch-fixture/v1",
		SpecificReceiptSHA256: repeat("e"), CleanupComplete: true,
	}
	sensoryDraftPath := writeValueTemp(t, sensoryDraft)
	sensoryPath := filepath.Join(directory, "sensory.json")
	if err := intakeSensoryFile(sensoryDraftPath, sensoryPath); err != nil {
		t.Fatalf("intakeSensoryFile() error = %v", err)
	}
	var sensory axmmirror.SensoryEvidenceReceipt
	if err := readStrictJSON(sensoryPath, &sensory); err != nil {
		t.Fatal(err)
	}

	skillItems := []axmmirror.SkillInventoryItem{
		{ID: "mirror-organ-library-115", Version: "03f3d0cf", UseClass: axmmirror.SkillUseKnowledge, SourceSHA256: repeat("1"), Status: "TEST", ExecutionStatus: "INERT", ProofStatus: "UNTESTED", CompatibilityStatus: "UNVERIFIED", MemberCount: 115},
		{ID: "sensorium-portable-skills", Version: "1.4.0", UseClass: axmmirror.SkillUseInstruction, SourceSHA256: repeat("2"), InstructionSHA256: repeat("3"), Status: "TEST", ExecutionStatus: "INERT", ProofStatus: "CONTRACT_PASS", CompatibilityStatus: "UNVERIFIED", MemberCount: 13},
	}
	skillRequest := axmmirror.SkillContinuityRequest{
		Schema: axmmirror.SkillContinuityRequestSchema, AssessmentID: "cli-skills-0001",
		TargetAnsweringIdentitySHA256: anchor.AnsweringIdentitySHA256,
		Requirements: []axmmirror.SkillRequirement{
			{ID: "mirror-organ-library-115", UseClass: axmmirror.SkillUseKnowledge},
			{ID: "sensorium-portable-skills", UseClass: axmmirror.SkillUseInstruction},
		},
		Current: axmmirror.SkillInventorySet{SetID: "cli-current", CapturedAt: "2026-08-15T10:00:00Z", Complete: true, Items: skillItems},
		Backup:  axmmirror.SkillInventorySet{SetID: "cli-backup", CapturedAt: "2026-08-15T09:00:00Z", Complete: true, Items: append([]axmmirror.SkillInventoryItem(nil), skillItems...)},
	}
	skillRequestPath := writeValueTemp(t, skillRequest)
	skillPath := filepath.Join(directory, "skills.json")
	if err := assessSkillsFile(skillRequestPath, skillPath); err != nil {
		t.Fatalf("assessSkillsFile() error = %v", err)
	}
	var skills axmmirror.SkillContinuityReceipt
	if err := readStrictJSON(skillPath, &skills); err != nil {
		t.Fatal(err)
	}

	discoveryRequest := axmmirror.DiscoveryStanceRequest{
		Schema: axmmirror.DiscoveryStanceRequestSchema, RequestID: "cli-discovery-0001", Subject: subject,
		AISeams: []axmmirror.DiscoverySeam{{
			ID: "freshness-visible", Stance: "evidence", Severity: "medium", Status: "OPEN",
			Statement: "The task depends on a bounded sensory receipt.", EvidenceRefs: []string{sensory.ReceiptSHA256},
			DisconfirmingCheck: "Re-run the exact observation after its TTL and compare typed receipts.", Blocks: []string{"unbounded-current-state-claim"},
		}},
		Human: &axmmirror.HumanDiscoveryInput{ExplicitInvocation: true, Entries: []axmmirror.HumanDiscoveryEntry{{
			ID: "human-use-0001", Stage: "blindSpots", Text: "Explain stale evidence in plain language.", ClaimLabel: "VALUE",
		}}},
	}
	discoveryRequestPath := writeValueTemp(t, discoveryRequest)
	discoveryPath := filepath.Join(directory, "discovery.json")
	if err := discoveryFile(discoveryRequestPath, discoveryPath); err != nil {
		t.Fatalf("discoveryFile() error = %v", err)
	}
	var discovery axmmirror.DiscoveryStancePacket
	if err := readStrictJSON(discoveryPath, &discovery); err != nil {
		t.Fatal(err)
	}

	situatedRequest := axmmirror.SituatedContextRequest{
		Schema: axmmirror.SituatedContextRequestSchema, EnvelopeID: "cli-situated-0001",
		TargetAnsweringIdentitySHA256: anchor.AnsweringIdentitySHA256, Subject: subject,
		ContextPacketSHA256: context.PacketSHA256, RequiredSenses: []string{sensory.SenseID},
		SensoryEvidence: []axmmirror.SensoryEvidenceReceipt{sensory}, SkillContinuity: skills, DiscoveryStance: discovery,
	}
	situatedRequestPath := writeValueTemp(t, situatedRequest)
	situatedPath := filepath.Join(directory, "situated.json")
	if err := situatedContextFile(contextPath, situatedRequestPath, situatedPath); err != nil {
		t.Fatalf("situatedContextFile() error = %v", err)
	}
	var situated axmmirror.SituatedContextEnvelope
	if err := readStrictJSON(situatedPath, &situated); err != nil {
		t.Fatal(err)
	}
	if situated.State != axmmirror.SituatedContextReady {
		t.Fatalf("situated state = %q", situated.State)
	}

	situatedSealedPath := filepath.Join(directory, "situated-behavior.json")
	if err := sealSituatedFile(anchorPath, witnessPath, profilePath, contextPath, assessmentPath, sealPath, situatedPath, behaviorDraftPath, situatedSealedPath); err != nil {
		t.Fatalf("sealSituatedFile() error = %v", err)
	}
	var situatedSealed axmmirror.SealedBehaviorEvidence
	if err := readStrictJSON(situatedSealedPath, &situatedSealed); err != nil {
		t.Fatal(err)
	}
	if err := situatedSealed.Verify(); err != nil {
		t.Fatalf("situated behavior verification: %v", err)
	}
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeValueTemp(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return writeTemp(t, string(data))
}

func repeat(ch string) string {
	value := ""
	for i := 0; i < 64; i++ {
		value += ch
	}
	return value
}

func repeat40(ch string) string {
	value := ""
	for i := 0; i < 40; i++ {
		value += ch
	}
	return value
}
