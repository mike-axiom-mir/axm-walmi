package axmmirror

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

var hermesExposureV021WebhookSafeTools = []string{"clarify", "vision_analyze", "web_extract", "web_search"}
var hermesExposureV021DesktopOnlyTools = []string{"focus_pane", "open_preview", "read_terminal"}

func VerifyHermesExposureV021Contract(data []byte) (HermesExposureV021Contract, error) {
	var c HermesExposureV021Contract
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("decode hermes exposure contract: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("hermes exposure contract has trailing JSON")
		}
		return HermesExposureV021Contract{}, err
	}
	if err := c.Validate(); err != nil {
		return HermesExposureV021Contract{}, err
	}
	expected, err := hermesExposureV021ExternalDigest(c)
	if err != nil {
		return HermesExposureV021Contract{}, err
	}
	if expected != c.ReceiptDigest {
		return HermesExposureV021Contract{}, fmt.Errorf("hermes exposure receipt digest mismatch: expected %s, got %s", expected, c.ReceiptDigest)
	}
	return c, nil
}

func (c HermesExposureV021Contract) Validate() error {
	if c.Schema != HermesExposureV021Schema || c.Status != HermesExposureV021Status || c.Challenge != HermesExposureV021Challenge {
		return errors.New("hermes exposure schema/status/challenge mismatch")
	}
	if c.Source.SourceV020Receipt != HermesExposureV021SourceV020Receipt || c.Source.SourceV020Head != HermesExposureV021SourceV020Head || c.Source.HermesRepo != HermesExposureV021HermesRepo || c.Source.LocalDonorCommit != HermesExposureV021LocalDonorCommit || c.Source.ReviewedCommit != HermesExposureV021ReviewedCommit {
		return errors.New("hermes exposure source identity mismatch")
	}
	if err := validateHermesExposureV021DonorFiles(c.Source.DonorFiles); err != nil {
		return err
	}
	b := c.Boundary
	if b.HermesRuntimeImported || b.HermesRuntimeExecuted || b.LocalCapabilityFabricExecuted || b.LiveAIProviderCalled || !b.DonorBlobContinuityVerified || !b.ContextCostsAreFixtureEstimates || !b.OutputIsContractProbe {
		return errors.New("hermes exposure truth boundary drifted")
	}
	profiles, err := validateHermesExposureV021Profiles(c.Profiles)
	if err != nil {
		return err
	}
	if len(c.Cases) != 5 {
		return fmt.Errorf("hermes exposure contract must retain exactly 5 evaluation cases, got %d", len(c.Cases))
	}
	seenCases := map[string]bool{}
	for _, tc := range c.Cases {
		if strings.TrimSpace(tc.CaseID) == "" || seenCases[tc.CaseID] {
			return errors.New("hermes exposure case id empty or duplicated")
		}
		seenCases[tc.CaseID] = true
		profile, ok := profiles[tc.SurfaceProfileID]
		if !ok {
			return fmt.Errorf("hermes exposure case %s references unknown profile %s", tc.CaseID, tc.SurfaceProfileID)
		}
		outcome, reason := EvaluateHermesExposureV021Case(profile, tc)
		if outcome != tc.ExpectedOutcome || reason != tc.ExpectedReason {
			return fmt.Errorf("hermes exposure case %s mismatch: got %s/%s want %s/%s", tc.CaseID, outcome, reason, tc.ExpectedOutcome, tc.ExpectedReason)
		}
	}
	if err := validateHermesExposureV021Refusals(c.RefusalCases); err != nil {
		return err
	}
	t := c.Truth
	if !t.InstalledDoesNotImplyExposed || !t.ExposedDoesNotImplyAuthorized || !t.AuthorizedDoesNotImplyExecuted || !t.WebhookSafeSetPreserved || !t.DesktopOnlyToolsRemainSurfaceBound || !t.ContextBudgetOverflowReturnsHold || t.ContextSilentlyPruned || t.SurfaceExposureGrantsAuthority || t.WaldoOwnsSurfaceExposure || t.HistoricalV020ReceiptRewritten {
		return errors.New("hermes exposure truth claims drifted")
	}
	if c.Authority != "NONE" {
		return errors.New("hermes exposure contract authority must remain NONE")
	}
	if !hermesExposureV021ValidSHA(c.ReceiptDigest) {
		return errors.New("hermes exposure receipt digest must be sha256:<64 hex>")
	}
	return nil
}

func EvaluateHermesExposureV021Case(profile HermesExposureV021SurfaceProfile, tc HermesExposureV021Case) (string, string) {
	if profile.SchemaCostTokens > profile.ContextBudgetTokens {
		return "HOLD", "CONTEXT_BUDGET_EXCEEDED"
	}
	if !tc.CapabilityDeclared {
		return "HOLD", "CAPABILITY_NOT_DECLARED"
	}
	if !tc.CapabilityInstalled {
		return "HOLD", "CAPABILITY_NOT_INSTALLED"
	}
	actualExposed := hermesExposureV021Contains(profile.ExposedTools, tc.ToolName)
	if tc.CapabilityExposed != actualExposed {
		return "REFUSED", "EXPOSURE_EVIDENCE_MISMATCH"
	}
	if profile.Surface == "WEBHOOK_UNTRUSTED" && !hermesExposureV021Contains(hermesExposureV021WebhookSafeTools, tc.ToolName) {
		return "REFUSED", "SURFACE_EXPOSURE_FORBIDDEN"
	}
	if profile.Surface != "DESKTOP" && hermesExposureV021Contains(hermesExposureV021DesktopOnlyTools, tc.ToolName) {
		return "HOLD", "CAPABILITY_NOT_EXPOSED_ON_SURFACE"
	}
	if !actualExposed {
		return "HOLD", "CAPABILITY_NOT_EXPOSED_ON_SURFACE"
	}
	if tc.ActionRequested && !tc.AuthorityPresent {
		return "HOLD", "AUTHORITY_ABSENT"
	}
	if tc.ActionRequested && tc.AuthorityPresent {
		return "READY", "AUTHORIZED_NOT_EXECUTED"
	}
	return "OBSERVED", "EXPOSED_NOT_ACTIONED"
}

func SealHermesExposureV021Contract(c HermesExposureV021Contract) ([]byte, error) {
	digest, err := hermesExposureV021ExternalDigest(c)
	if err != nil {
		return nil, err
	}
	c.ReceiptDigest = digest
	return json.MarshalIndent(c, "", "  ")
}

func hermesExposureV021ExternalDigest(c HermesExposureV021Contract) (string, error) {
	c.ReceiptDigest = ""
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func validateHermesExposureV021DonorFiles(files []HermesExposureV021DonorFile) error {
	expected := map[string]string{
		"toolsets.py":                "23eacf088afcf72ca0bcd12e28ffdc8313c33d25",
		"agent/context_breakdown.py": "4527c5dca220ac2e9e51ece5b766b203c70b579b",
		"agent/shell_hooks.py":       "8751aeb6fd95421fa7293be0b68b75aebfbe3627",
	}
	if len(files) != len(expected) {
		return fmt.Errorf("hermes exposure donor file count mismatch: got %d", len(files))
	}
	seen := map[string]bool{}
	for _, f := range files {
		want, ok := expected[f.Path]
		if !ok || seen[f.Path] {
			return fmt.Errorf("hermes exposure donor file path unexpected or duplicated: %s", f.Path)
		}
		seen[f.Path] = true
		if f.BlobAtLocalPin != want || f.BlobAtReviewedHead != want || !f.Unchanged {
			return fmt.Errorf("hermes exposure donor file %s continuity mismatch", f.Path)
		}
	}
	return nil
}

func validateHermesExposureV021Profiles(profiles []HermesExposureV021SurfaceProfile) (map[string]HermesExposureV021SurfaceProfile, error) {
	if len(profiles) != 4 {
		return nil, fmt.Errorf("hermes exposure profile count mismatch: got %d", len(profiles))
	}
	out := map[string]HermesExposureV021SurfaceProfile{}
	for _, p := range profiles {
		if p.ProfileID == "" || out[p.ProfileID].ProfileID != "" {
			return nil, errors.New("hermes exposure profile id empty or duplicated")
		}
		if p.CostEvidenceClass != "FIXTURE_ESTIMATE_NOT_HERMES_MEASUREMENT" || p.ContextBudgetTokens <= 0 || p.SchemaCostTokens < 0 {
			return nil, fmt.Errorf("hermes exposure profile %s cost declaration drifted", p.ProfileID)
		}
		if !sort.StringsAreSorted(p.ExposedTools) || !sort.StringsAreSorted(p.ExposedToolsets) {
			return nil, fmt.Errorf("hermes exposure profile %s exposure lists must be canonical sorted", p.ProfileID)
		}
		out[p.ProfileID] = p
	}
	webhook, ok := out["webhook-untrusted-safe"]
	if !ok || webhook.Surface != "WEBHOOK_UNTRUSTED" || webhook.SourceTrust != "UNTRUSTED_THIRD_PARTY_CONTENT" || !hermesExposureV021SliceExact(webhook.ExposedTools, hermesExposureV021WebhookSafeTools) {
		return nil, errors.New("hermes webhook-safe surface drifted")
	}
	desktop, ok := out["desktop-interactive"]
	if !ok || desktop.Surface != "DESKTOP" || desktop.SourceTrust != "TRUSTED_INTERACTIVE" {
		return nil, errors.New("hermes desktop surface drifted")
	}
	for _, tool := range hermesExposureV021DesktopOnlyTools {
		if !hermesExposureV021Contains(desktop.ExposedTools, tool) {
			return nil, fmt.Errorf("desktop surface missing desktop-only tool %s", tool)
		}
	}
	cli, ok := out["cli-interactive"]
	if !ok || cli.Surface != "CLI" || cli.SourceTrust != "TRUSTED_INTERACTIVE" {
		return nil, errors.New("hermes CLI surface drifted")
	}
	over, ok := out["cli-overbudget"]
	if !ok || over.SchemaCostTokens <= over.ContextBudgetTokens {
		return nil, errors.New("hermes overbudget profile no longer exceeds its declared context budget")
	}
	return out, nil
}

func hermesExposureV021Contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func hermesExposureV021SliceExact(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func hermesExposureV021ValidSHA(v string) bool {
	if !strings.HasPrefix(v, "sha256:") || len(v) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(v, "sha256:"))
	return err == nil
}
