package axmmirror

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPortableCapabilitySpineFixturesPinInitialReceipts(t *testing.T) {
	examples := filepath.Join("..", "..", "examples", "axm-mirror")

	var censusRequest SelfCapabilityCensusRequest
	readCapabilityFixture(t, filepath.Join(examples, "self-capability-census-request.json"), &censusRequest)
	self, err := CensusSelfCapabilities(censusRequest)
	if err != nil {
		t.Fatal(err)
	}
	if self.SnapshotSHA256 != "96679ab73fe4baeec56debcd792e450e5e5eedf1a527a82a4c8ff08eb6243fa4" {
		t.Fatalf("self capability snapshot digest = %s", self.SnapshotSHA256)
	}

	var externalSnapshot ExternalCapabilitySnapshot
	readCapabilityFixture(t, filepath.Join(examples, "external-capability-snapshot.json"), &externalSnapshot)
	external, err := IntakeExternalCapabilities(externalSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	if external.State != ExternalCapabilitySourceOnly || external.ReceiptSHA256 != "e35df8a87901637987278b382bedfb2f370bb40cf515fd63d7e8e1f79ab0a21b" {
		t.Fatalf("external capability receipt = %s %s", external.State, external.ReceiptSHA256)
	}

	var request CapabilityGapRequest
	readCapabilityFixture(t, filepath.Join(examples, "capability-gap-request.json"), &request)
	plan, err := PlanCapabilityHandoff(self, external, request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.State != HandoffStateSourceOnly || plan.PlanSHA256 != "9e910bfb60f25c60d4a50a7baa32fea336eab669366deb43a2fbe871f3f3b449" {
		t.Fatalf("capability handoff plan = %s %s", plan.State, plan.PlanSHA256)
	}

	var declaration CapabilityTranslationDeclaration
	readCapabilityFixture(t, filepath.Join(examples, "capability-translation-declaration.json"), &declaration)
	translation, err := SealCapabilityTranslation(plan, declaration)
	if err != nil {
		t.Fatal(err)
	}
	if translation.State != TranslationStateNotRequired || translation.ReceiptSHA256 != "8aa028c33b55126e410f30a70926df7aaf822685f3999af266f43e25b28f8903" {
		t.Fatalf("translation loss receipt = %s %s", translation.State, translation.ReceiptSHA256)
	}

	var draft CapabilityReturnDraft
	readCapabilityFixture(t, filepath.Join(examples, "capability-return-draft.json"), &draft)
	returned, err := VerifyCapabilityReturn(plan, translation, draft)
	if err != nil {
		t.Fatal(err)
	}
	if returned.State != CapabilityReturnPlanHold || returned.ReceiptSHA256 != "70ca49ca661b7399c60aa52538922378c742156f81f642baf572880252a3e303" {
		t.Fatalf("capability return receipt = %s %s", returned.State, returned.ReceiptSHA256)
	}
}

func readCapabilityFixture(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := DecodeStrictJSON(data, target); err != nil {
		t.Fatal(err)
	}
}
