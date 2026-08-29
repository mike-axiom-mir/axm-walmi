package axmmirror

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

func VerifyHermesRunCapsuleV022Contract(data []byte) (HermesRunCapsuleV022Contract, error) {
	var c HermesRunCapsuleV022Contract
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("decode v0.22: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("trailing JSON")
		}
		return HermesRunCapsuleV022Contract{}, err
	}
	if err := c.Validate(); err != nil {
		return HermesRunCapsuleV022Contract{}, err
	}
	d, err := hermesRunCapsuleV022ExternalDigest(c)
	if err != nil {
		return HermesRunCapsuleV022Contract{}, err
	}
	if d != c.ReceiptDigest {
		return HermesRunCapsuleV022Contract{}, fmt.Errorf("receipt mismatch: %s != %s", d, c.ReceiptDigest)
	}
	return c, nil
}

func (c HermesRunCapsuleV022Contract) Validate() error {
	if c.Schema != HermesRunCapsuleV022Schema || c.Status != HermesRunCapsuleV022Status || c.Challenge != HermesRunCapsuleV022Challenge {
		return errors.New("schema/status/challenge mismatch")
	}
	if c.Source.PlatformRepo != HermesRunCapsuleV022PlatformRepo || c.Source.PlatformPR != HermesRunCapsuleV022PlatformPR || c.Source.PlatformHead != HermesRunCapsuleV022PlatformHead {
		return errors.New("platform source mismatch")
	}
	if c.Source.RuntimeGateRunID != 32672295737 || c.Source.RuntimeGateConclusion != "success" {
		return errors.New("platform runtime gate evidence mismatch")
	}
	if err := validateHermesRunCapsuleV022DonorFiles(c.Source.DonorFiles); err != nil {
		return err
	}
	b := c.Boundary
	if b.PlatformHermesRuntimeCopied || b.PlatformHermesRuntimeExecutedByWaldo || b.LiveHermesRunObserved || b.LiveAIProviderCalled || !b.OutputIsContractProbe {
		return errors.New("truth boundary drifted")
	}
	if len(c.Cases) != 7 {
		return fmt.Errorf("expected 7 cases, got %d", len(c.Cases))
	}
	seen := map[string]bool{}
	for _, tc := range c.Cases {
		if tc.CaseID == "" || seen[tc.CaseID] {
			return errors.New("case id empty/duplicate")
		}
		seen[tc.CaseID] = true
		o, r := EvaluateHermesRunCapsuleV022Case(tc)
		if o != tc.ExpectedOutcome || r != tc.ExpectedReason {
			return fmt.Errorf("case %s got %s/%s want %s/%s", tc.CaseID, o, r, tc.ExpectedOutcome, tc.ExpectedReason)
		}
	}
	t := c.Truth
	if !t.ReceiptRecordingIndependentOfCurrentConsent || !t.RevocationDoesNotAuthorizeNewAction || !t.EvidenceGapIsExplicit || !t.ProviderMismatchIsExplicit || !t.PolicyDriftIsExplicit || !t.RawContentExcluded || !t.ReturnPacketCandidateOnly || t.ReturnPacketGrantsPromotion || t.WaldoOwnsHermesRuntime {
		return errors.New("truth claims drifted")
	}
	if c.Authority != "NONE" {
		return errors.New("authority must remain NONE")
	}
	if !hermesRunCapsuleV022ValidSHA(c.ReceiptDigest) {
		return errors.New("invalid receipt digest")
	}
	return nil
}

func EvaluateHermesRunCapsuleV022Case(tc HermesRunCapsuleV022Case) (string, string) {
	if tc.RawPromptStored || tc.RawResponseStored || tc.RawToolArgumentsStored || tc.RawToolResultsStored {
		return "REFUSED", "RAW_CONTENT_STORAGE_FORBIDDEN"
	}
	if tc.Canon || tc.Promotion != "candidate-only" {
		return "REFUSED", "PROMOTION_OR_CANON_FORBIDDEN"
	}
	if !tc.SourceVerifiedBefore {
		return "HOLD", "SOURCE_NOT_VERIFIED_BEFORE_RUN"
	}
	if !tc.SourceVerifiedAfter {
		return "HOLD", "SOURCE_NOT_VERIFIED_AFTER_RUN"
	}
	if tc.CompletionReceipts < tc.AuthorizedToolCalls {
		return "HOLD", "COMPLETION_RECEIPT_GAP"
	}
	if tc.ProviderPolicyMismatch {
		return "HOLD", "PROVIDER_POLICY_MISMATCH"
	}
	if tc.PolicyChangedDuringRun {
		return "HOLD", "POLICY_CHANGED_DURING_RUN"
	}
	if tc.ProfileChangedDuringRun {
		return "HOLD", "PROFILE_CHANGED_DURING_RUN"
	}
	if tc.ConsentRevokedAfterAuthorization {
		if tc.AuthorizedToolCalls != 1 {
			return "REFUSED", "REVOCATION_CANNOT_AUTHORIZE_ADDITIONAL_ACTION"
		}
		if !tc.CompletionEvidenceRecorded {
			return "HOLD", "IN_FLIGHT_COMPLETION_EVIDENCE_MISSING"
		}
		return "OBSERVED", "COMPLETION_EVIDENCE_RETAINED_AFTER_REVOCATION"
	}
	return "CANDIDATE", "EVIDENCE_COMPLETE_NOT_PROMOTED"
}

func SealHermesRunCapsuleV022Contract(c HermesRunCapsuleV022Contract) ([]byte, error) {
	d, err := hermesRunCapsuleV022ExternalDigest(c)
	if err != nil {
		return nil, err
	}
	c.ReceiptDigest = d
	return json.MarshalIndent(c, "", "  ")
}

func hermesRunCapsuleV022ExternalDigest(c HermesRunCapsuleV022Contract) (string, error) {
	c.ReceiptDigest = ""
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func validateHermesRunCapsuleV022DonorFiles(files []HermesRunCapsuleV022DonorFile) error {
	expected := map[string]string{
		"tools/hermes/axm/axm_receipt.py":          "774015cbac1c50f78aa270ad8ed04d90520a83e5",
		"tools/hermes/axm/axm_provider_receipt.py": "91c8101eae290f9fefc7ed02e9152daf93136302",
		"tools/hermes/axm/axm_session_event.py":    "8ed2b418cf99b3a8d52439dc4e738161268723e2",
		"tools/hermes/axm/run-ledger.js":           "c8a956976e1871263cd3639608808a1138c7711e",
		"tools/hermes/axm/axm_gate.py":             "abefcfc227353b02e615215326df099883d478c7",
		"tools/hermes/axm/runtime-policy.js":       "92cec223b7a0e7713c2771fa21e3fb9822d0794b",
		"tools/hermes/module.contract.json":        "138dcb038361daed93a5bd7abf1c06785572e641",
	}
	if len(files) != len(expected) {
		return fmt.Errorf("donor file count mismatch: %d", len(files))
	}
	seen := map[string]bool{}
	for _, f := range files {
		want, ok := expected[f.Path]
		if !ok || seen[f.Path] || f.Blob != want {
			return fmt.Errorf("donor file mismatch: %s", f.Path)
		}
		seen[f.Path] = true
	}
	return nil
}

func hermesRunCapsuleV022ValidSHA(v string) bool {
	if !strings.HasPrefix(v, "sha256:") || len(v) != 71 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(v, "sha256:"))
	return err == nil
}
