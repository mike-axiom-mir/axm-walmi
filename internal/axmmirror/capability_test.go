package axmmirror

import "testing"

func TestSelfCapabilityCensusIsDeterministicAndClosed(t *testing.T) {
	request := testSelfCapabilityCensusRequest()
	first, err := CensusSelfCapabilities(request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CensusSelfCapabilities(request)
	if err != nil {
		t.Fatal(err)
	}
	if first.SnapshotSHA256 != second.SnapshotSHA256 || first.CatalogSHA256 != second.CatalogSHA256 {
		t.Fatalf("self census is not deterministic: first=%s second=%s", first.SnapshotSHA256, second.SnapshotSHA256)
	}
	if first.State != SelfCapabilityStateDeclared || len(first.Capabilities) < 19 || !first.Authority.closed() {
		t.Fatalf("self census = %+v", first)
	}
	request.Authority.ToolExecution = true
	if _, err := CensusSelfCapabilities(request); err == nil {
		t.Fatal("self census accepted tool authority")
	}
}

func TestExternalCapabilityIntakePreservesLiveSourceOnlyAndStale(t *testing.T) {
	live := testExternalCapabilitySnapshot(testExternalCapability())
	receipt, err := IntakeExternalCapabilities(live)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != ExternalCapabilityLive || receipt.Freshness != ExternalFreshnessFresh {
		t.Fatalf("live receipt = %+v", receipt)
	}

	sourceOnly := testExternalCapabilitySnapshot(testExternalCapability())
	sourceOnly.SnapshotID = "source-only"
	sourceOnly.ObservationState = ExternalObservationSourceOnly
	sourceOnly.RuntimeEvidenceSHA256 = ""
	sourceOnly.Capabilities[0].DeclarationState = ExternalDeclarationDeclared
	sourceReceipt, err := IntakeExternalCapabilities(sourceOnly)
	if err != nil {
		t.Fatal(err)
	}
	if sourceReceipt.State != ExternalCapabilitySourceOnly {
		t.Fatalf("source-only receipt state = %q", sourceReceipt.State)
	}

	stale := testExternalCapabilitySnapshot(testExternalCapability())
	stale.SnapshotID = "stale"
	stale.AssessedAt = "2026-08-15T10:02:00Z"
	staleReceipt, err := IntakeExternalCapabilities(stale)
	if err != nil {
		t.Fatal(err)
	}
	if staleReceipt.State != ExternalCapabilityStale || staleReceipt.AgeMillis != 120_000 {
		t.Fatalf("stale receipt = %+v", staleReceipt)
	}

	absent := testExternalCapabilitySnapshot()
	absent.SnapshotID = "absent"
	absent.ObservationState = ExternalObservationAbsent
	absentReceipt, err := IntakeExternalCapabilities(absent)
	if err != nil {
		t.Fatal(err)
	}
	if absentReceipt.State != ExternalCapabilityAbsent {
		t.Fatalf("absent receipt state = %q", absentReceipt.State)
	}

	unknown := testExternalCapabilitySnapshot()
	unknown.SnapshotID = "unknown"
	unknown.ObservationState = ExternalObservationUnknown
	unknown.InventoryComplete = false
	unknown.RuntimeEvidenceSHA256 = ""
	unknownReceipt, err := IntakeExternalCapabilities(unknown)
	if err != nil {
		t.Fatal(err)
	}
	if unknownReceipt.State != ExternalCapabilityUnknown {
		t.Fatalf("unknown receipt state = %q", unknownReceipt.State)
	}

	live.Authority.WorldAction = true
	if _, err := IntakeExternalCapabilities(live); err == nil {
		t.Fatal("external intake accepted world-action authority")
	}
}

func TestCapabilityHandoffPlansExactLocalAmbiguousPermissionAndTranslationStates(t *testing.T) {
	self := testSelfCapabilitySnapshot(t)
	request := testCapabilityGapRequest()
	external := testExternalCapabilityReceipt(t, testExternalCapability())

	ready, err := PlanCapabilityHandoff(self, external, request)
	if err != nil {
		t.Fatal(err)
	}
	if ready.State != HandoffStateReady || ready.UniqueCandidateProviderID != "pixel-hand" || ready.ProviderSelected || ready.ExternalExecutionAllowed {
		t.Fatalf("ready plan = %+v", ready)
	}

	localRequest := request
	localRequest.RequestID = "local-gap"
	localRequest.CapabilityID = "capability.self.census"
	localRequest.Input.Schema = SelfCapabilityCensusRequestSchema
	localRequest.RequiredOutputSchema = SelfCapabilitySnapshotSchema
	local, err := PlanCapabilityHandoff(self, external, localRequest)
	if err != nil {
		t.Fatal(err)
	}
	if local.State != HandoffStateLocal {
		t.Fatalf("local plan state = %q", local.State)
	}

	second := testExternalCapability()
	second.ProviderID = "second-pixel-hand"
	ambiguousExternal := testExternalCapabilityReceipt(t, testExternalCapability(), second)
	ambiguous, err := PlanCapabilityHandoff(self, ambiguousExternal, request)
	if err != nil {
		t.Fatal(err)
	}
	if ambiguous.State != HandoffStateProviderAmbiguous || len(ambiguous.Candidates) != 2 {
		t.Fatalf("ambiguous plan = %+v", ambiguous)
	}

	permissionCapability := testExternalCapability()
	permissionCapability.RequiredPermissions = []string{"network"}
	permissionExternal := testExternalCapabilityReceipt(t, permissionCapability)
	permission, err := PlanCapabilityHandoff(self, permissionExternal, request)
	if err != nil {
		t.Fatal(err)
	}
	if permission.State != HandoffStatePermission || permission.ExternalExecutionAllowed {
		t.Fatalf("permission plan = %+v", permission)
	}

	translationCapability := testExternalCapability()
	translationCapability.InputSchemas = []string{"axm.asset-brief/v2"}
	translationExternal := testExternalCapabilityReceipt(t, translationCapability)
	translation, err := PlanCapabilityHandoff(self, translationExternal, request)
	if err != nil {
		t.Fatal(err)
	}
	if translation.State != HandoffStateSchemaTranslation || translation.Candidates[0].Compatibility != CapabilityCompatibilityTranslation {
		t.Fatalf("translation plan = %+v", translation)
	}

	staleRequest := request
	staleRequest.RequestID = "stale-at-planning"
	staleRequest.CreatedAt = "2026-08-15T10:02:00Z"
	staleRequest.ExpiresAt = "2026-08-15T10:12:00Z"
	stalePlan, err := PlanCapabilityHandoff(self, external, staleRequest)
	if err != nil {
		t.Fatal(err)
	}
	if stalePlan.State != HandoffStateExternalHold {
		t.Fatalf("stale-at-planning state = %q", stalePlan.State)
	}

	sourceSnapshot := testExternalCapabilitySnapshot(testExternalCapability())
	sourceSnapshot.ObservationState = ExternalObservationSourceOnly
	sourceSnapshot.RuntimeEvidenceSHA256 = ""
	sourceSnapshot.Capabilities[0].DeclarationState = ExternalDeclarationDeclared
	sourceExternal, err := IntakeExternalCapabilities(sourceSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	sourcePlan, err := PlanCapabilityHandoff(self, sourceExternal, request)
	if err != nil {
		t.Fatal(err)
	}
	if sourcePlan.State != HandoffStateSourceOnly {
		t.Fatalf("source-only plan state = %q", sourcePlan.State)
	}
}

func TestTranslationLossReceiptNeverHidesDeclaredLoss(t *testing.T) {
	ready := testReadyCapabilityPlan(t)
	none := CapabilityTranslationDeclaration{
		Schema: CapabilityTranslationDeclarationSchema, TranslationID: "translation-none",
		PlanSHA256: ready.PlanSHA256, ProviderID: ready.UniqueCandidateProviderID,
		InputFromSchema: ready.Request.Input.Schema, InputToSchema: ready.Request.Input.Schema,
		OutputFromSchema: ready.Request.RequiredOutputSchema, OutputToSchema: ready.Request.RequiredOutputSchema,
		DeclaredAt: "2026-08-15T10:00:01Z", Authority: Authority{},
	}
	receipt, err := SealCapabilityTranslation(ready, none)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != TranslationStateNotRequired || receipt.HumanReviewRequired || receipt.AutomaticAdapterGenerated {
		t.Fatalf("no-translation receipt = %+v", receipt)
	}
	expiredDeclaration := none
	expiredDeclaration.TranslationID = "translation-after-expiry"
	expiredDeclaration.DeclaredAt = "2026-08-15T10:11:00Z"
	if _, err := SealCapabilityTranslation(ready, expiredDeclaration); err == nil {
		t.Fatal("translation declaration outside the plan lifetime was accepted")
	}

	self := testSelfCapabilitySnapshot(t)
	request := testCapabilityGapRequest()
	capability := testExternalCapability()
	capability.InputSchemas = []string{"axm.asset-brief/v2"}
	plan, err := PlanCapabilityHandoff(self, testExternalCapabilityReceipt(t, capability), request)
	if err != nil {
		t.Fatal(err)
	}
	rename := CapabilityTranslationDeclaration{
		Schema: CapabilityTranslationDeclarationSchema, TranslationID: "translation-rename",
		PlanSHA256: plan.PlanSHA256, ProviderID: "pixel-hand",
		InputFromSchema: "axm.asset-brief/v1", InputToSchema: "axm.asset-brief/v2",
		OutputFromSchema: "image/png", OutputToSchema: "image/png", DeclaredAt: "2026-08-15T10:00:01Z",
		Mappings: []CapabilityFieldTranslation{{
			Direction: TranslationDirectionInput, SourceField: "intent", TargetField: "brief.intent",
			Method: TranslationMethodRename, Loss: TranslationLossNone, Note: "field moved without a declared value change",
		}},
		Authority: Authority{},
	}
	missingMapping := rename
	missingMapping.TranslationID = "translation-missing-mapping"
	missingMapping.Mappings = nil
	if _, err := SealCapabilityTranslation(plan, missingMapping); err == nil {
		t.Fatal("different schemas without an explicit mapping were accepted")
	}
	renameReceipt, err := SealCapabilityTranslation(plan, rename)
	if err != nil {
		t.Fatal(err)
	}
	if renameReceipt.State != TranslationStateNoDeclaredLoss {
		t.Fatalf("rename translation state = %q", renameReceipt.State)
	}

	approximation := rename
	approximation.TranslationID = "translation-approximation"
	approximation.Mappings[0].Method = TranslationMethodApproximation
	approximation.Mappings[0].Loss = TranslationLossDeclared
	approximation.Mappings[0].Note = "provider accepts a shorter intent field"
	approximationReceipt, err := SealCapabilityTranslation(plan, approximation)
	if err != nil {
		t.Fatal(err)
	}
	if approximationReceipt.State != TranslationStateReviewLoss || !approximationReceipt.HumanReviewRequired || len(approximationReceipt.DeclaredLosses) != 1 {
		t.Fatalf("approximation receipt = %+v", approximationReceipt)
	}

	approximation.Authority.Promotion = true
	if _, err := SealCapabilityTranslation(plan, approximation); err == nil {
		t.Fatal("translation declaration accepted promotion authority")
	}
}

func TestCapabilityReturnVerifiesBindingsAndPreservesHolds(t *testing.T) {
	plan := testReadyCapabilityPlan(t)
	translation := testNoTranslationReceipt(t, plan)
	draft := testCapabilityReturnDraft(plan, translation)

	verified, err := VerifyCapabilityReturn(plan, translation, draft)
	if err != nil {
		t.Fatal(err)
	}
	if verified.State != CapabilityReturnVerified || len(verified.Holds) != 0 {
		t.Fatalf("verified return = %+v", verified)
	}

	tampered := draft
	tampered.Input.SHA256 = repeatCapabilityHex("8")
	binding, err := VerifyCapabilityReturn(plan, translation, tampered)
	if err != nil {
		t.Fatal(err)
	}
	if binding.State != CapabilityReturnBindingHold {
		t.Fatalf("tampered return state = %q", binding.State)
	}

	expired := draft
	expired.CompletedAt = "2026-08-15T10:11:00Z"
	expired.AssessedAt = "2026-08-15T10:11:01Z"
	expiry, err := VerifyCapabilityReturn(plan, translation, expired)
	if err != nil {
		t.Fatal(err)
	}
	if expiry.State != CapabilityReturnExpiredHold {
		t.Fatalf("expired return state = %q", expiry.State)
	}

	retained := draft
	retained.CleanupState = CapabilityCleanupUnconfirmed
	retained.RetainedInputBytes = 32
	cleanup, err := VerifyCapabilityReturn(plan, translation, retained)
	if err != nil {
		t.Fatal(err)
	}
	if cleanup.State != CapabilityReturnCleanupHold {
		t.Fatalf("retained return state = %q", cleanup.State)
	}

	wrongOutput := draft
	wrongOutput.Outputs = append([]CapabilityOutputArtifact(nil), draft.Outputs...)
	wrongOutput.Outputs[0].Schema = "image/jpeg"
	output, err := VerifyCapabilityReturn(plan, translation, wrongOutput)
	if err != nil {
		t.Fatal(err)
	}
	if output.State != CapabilityReturnOutputHold {
		t.Fatalf("wrong-schema return state = %q", output.State)
	}

	draft.Authority.Canon = true
	if _, err := VerifyCapabilityReturn(plan, translation, draft); err == nil {
		t.Fatal("capability return accepted CANON authority")
	}
}

func testSelfCapabilityCensusRequest() SelfCapabilityCensusRequest {
	return SelfCapabilityCensusRequest{
		Schema: SelfCapabilityCensusRequestSchema, CensusID: "self-census-0001",
		TargetAnsweringIdentitySHA256: repeatCapabilityHex("1"), ObservedBuildSHA256: repeatCapabilityHex("2"),
		ObservationReceiptSHA256: repeatCapabilityHex("3"), CapturedAt: "2026-08-15T10:00:00Z", Authority: Authority{},
	}
}

func testSelfCapabilitySnapshot(t *testing.T) SelfCapabilitySnapshot {
	t.Helper()
	snapshot, err := CensusSelfCapabilities(testSelfCapabilityCensusRequest())
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func testExternalCapability() ExternalCapability {
	return ExternalCapability{
		ID: "asset.external.render", ProviderID: "pixel-hand", Version: "v1",
		ExecutionClass: "EXTERNAL_HAND", DeclarationState: ExternalDeclarationAvailable,
		InputSchemas: []string{"axm.asset-brief/v1"}, OutputSchemas: []string{"image/png"},
		AcceptedDataClasses: []string{CapabilityDataPublic}, Refusals: []string{"automatic-promotion"},
	}
}

func testExternalCapabilitySnapshot(capabilities ...ExternalCapability) ExternalCapabilitySnapshot {
	return ExternalCapabilitySnapshot{
		Schema: ExternalCapabilitySnapshotSchema, SnapshotID: "external-live-0001",
		Source: CapabilitySnapshotSource{
			Repository: "example/platform", Branch: "main", Commit: repeatCapabilityHex40("a"),
			Documents: []CapabilitySourceDocument{{Path: "tools/pixel-hand/module.contract.json", GitBlobSHA: repeatCapabilityHex40("b"), SHA256: repeatCapabilityHex("c")}},
		},
		ObservationState: ExternalObservationLive, ObservedAt: "2026-08-15T10:00:00Z", AssessedAt: "2026-08-15T10:00:00Z",
		TTLMillis: 60_000, InventoryComplete: true, RuntimeEvidenceSHA256: repeatCapabilityHex("d"), Capabilities: capabilities,
		Authority: Authority{},
	}
}

func testExternalCapabilityReceipt(t *testing.T, capabilities ...ExternalCapability) ExternalCapabilityReceipt {
	t.Helper()
	receipt, err := IntakeExternalCapabilities(testExternalCapabilitySnapshot(capabilities...))
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func testCapabilityGapRequest() CapabilityGapRequest {
	return CapabilityGapRequest{
		Schema: CapabilityGapRequestSchema, RequestID: "capability-gap-0001",
		TargetAnsweringIdentitySHA256: repeatCapabilityHex("1"), CapabilityID: "asset.external.render",
		Input:                CapabilityArtifactRef{Schema: "axm.asset-brief/v1", SHA256: repeatCapabilityHex("4"), Bytes: 512},
		RequiredOutputSchema: "image/png", DataClass: CapabilityDataPublic, MaxOutputBytes: 4096,
		CreatedAt: "2026-08-15T10:00:01Z", ExpiresAt: "2026-08-15T10:10:00Z", Authority: Authority{},
	}
}

func testReadyCapabilityPlan(t *testing.T) CapabilityHandoffPlan {
	t.Helper()
	plan, err := PlanCapabilityHandoff(testSelfCapabilitySnapshot(t), testExternalCapabilityReceipt(t, testExternalCapability()), testCapabilityGapRequest())
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func testNoTranslationReceipt(t *testing.T, plan CapabilityHandoffPlan) TranslationLossReceipt {
	t.Helper()
	receipt, err := SealCapabilityTranslation(plan, CapabilityTranslationDeclaration{
		Schema: CapabilityTranslationDeclarationSchema, TranslationID: "translation-none",
		PlanSHA256: plan.PlanSHA256, ProviderID: plan.UniqueCandidateProviderID,
		InputFromSchema: plan.Request.Input.Schema, InputToSchema: plan.Request.Input.Schema,
		OutputFromSchema: plan.Request.RequiredOutputSchema, OutputToSchema: plan.Request.RequiredOutputSchema,
		DeclaredAt: "2026-08-15T10:00:01Z", Authority: Authority{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func testCapabilityReturnDraft(plan CapabilityHandoffPlan, translation TranslationLossReceipt) CapabilityReturnDraft {
	return CapabilityReturnDraft{
		Schema: CapabilityReturnDraftSchema, ReturnID: "return-0001", PlanSHA256: plan.PlanSHA256,
		TranslationReceiptSHA256: translation.ReceiptSHA256, ProviderID: plan.UniqueCandidateProviderID,
		Input:                 plan.Request.Input,
		Outputs:               []CapabilityOutputArtifact{{ID: "primary", CapabilityArtifactRef: CapabilityArtifactRef{Schema: plan.Request.RequiredOutputSchema, SHA256: repeatCapabilityHex("5"), Bytes: 1024}}},
		ProviderReceiptSHA256: repeatCapabilityHex("6"), StartedAt: "2026-08-15T10:00:02Z", CompletedAt: "2026-08-15T10:00:03Z",
		AssessedAt: "2026-08-15T10:00:04Z", RuntimeState: CapabilityRuntimeComplete,
		PermissionState: CapabilityPermissionConfirmed, CleanupState: CapabilityCleanupConfirmed, Authority: Authority{},
	}
}

func repeatCapabilityHex(character string) string {
	value := ""
	for i := 0; i < 64; i++ {
		value += character
	}
	return value
}

func repeatCapabilityHex40(character string) string {
	value := ""
	for i := 0; i < 40; i++ {
		value += character
	}
	return value
}
