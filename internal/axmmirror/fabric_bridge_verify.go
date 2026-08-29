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

func VerifyPlatformFabricReceipt(data []byte) (PlatformFabricReceipt, error) {
	var receipt PlatformFabricReceipt
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return PlatformFabricReceipt{}, fmt.Errorf("decode platform Fabric receipt: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("platform Fabric receipt has trailing JSON")
		}
		return PlatformFabricReceipt{}, err
	}
	if err := receipt.Validate(); err != nil {
		return PlatformFabricReceipt{}, err
	}
	expected, err := platformFabricExternalReceiptDigest(data)
	if err != nil {
		return PlatformFabricReceipt{}, err
	}
	if expected != receipt.ReceiptSHA256 {
		return PlatformFabricReceipt{}, fmt.Errorf("platform Fabric receipt digest mismatch: expected %s, got %s", expected, receipt.ReceiptSHA256)
	}
	return receipt, nil
}

func (receipt PlatformFabricReceipt) Validate() error {
	if receipt.Schema != PlatformFabricReceiptSchema || receipt.Status != "EXPERIMENTAL" || receipt.Challenge != PlatformFabricChallenge {
		return errors.New("platform Fabric receipt has unsupported schema, status, or challenge")
	}
	if receipt.Donor.Repository != PlatformFabricRepository || receipt.Donor.Commit != PlatformFabricCommit {
		return errors.New("platform Fabric donor repository or commit mismatch")
	}
	if !stringMapEqual(receipt.Donor.Blobs, platformFabricDonorBlobs) {
		return errors.New("platform Fabric donor blob set mismatch")
	}
	if err := validatePlatformFabricRef("concept", receipt.ConceptRef); err != nil {
		return err
	}
	if receipt.ConceptRef.ID != "one-concept-many-bodies" || receipt.ConceptRef.Schema != "axm.waldo-mirror.fabric-concept/v0.1" {
		return errors.New("platform Fabric concept identity mismatch")
	}
	if !receipt.DeterministicDefault {
		return errors.New("platform Fabric deterministic default must remain true")
	}
	if receipt.AIOptInUsed != (receipt.AIProposalRef != nil) {
		return errors.New("platform Fabric AI opt-in and proposal reference disagree")
	}
	if receipt.AIProposalRef != nil {
		if err := validatePlatformFabricRef("AI proposal", *receipt.AIProposalRef); err != nil {
			return err
		}
		if receipt.AIProposalRef.Schema != "axm.waldo-mirror.fabric-ai-proposal/v0.1" {
			return errors.New("platform Fabric AI proposal schema mismatch")
		}
	}
	expectedBodies := 3
	if receipt.AIOptInUsed {
		expectedBodies = 4
	}
	if len(receipt.Bodies) != expectedBodies {
		return fmt.Errorf("platform Fabric receipt requires %d bodies, got %d", expectedBodies, len(receipt.Bodies))
	}
	seenIDs := map[string]bool{}
	seenBlueprints := map[string]bool{}
	aiBodies := 0
	for index, body := range receipt.Bodies {
		if strings.TrimSpace(body.BodyID) == "" || body.BodyID != strings.TrimSpace(body.BodyID) {
			return fmt.Errorf("platform Fabric body %d has invalid id", index)
		}
		if seenIDs[body.BodyID] {
			return fmt.Errorf("platform Fabric duplicate body %q", body.BodyID)
		}
		seenIDs[body.BodyID] = true
		if body.SourceMode != PlatformFabricDeterministicSource && body.SourceMode != PlatformFabricAIOptInSource {
			return fmt.Errorf("platform Fabric body %q has unsupported source mode %q", body.BodyID, body.SourceMode)
		}
		if body.SourceMode == PlatformFabricAIOptInSource {
			aiBodies++
		}
		for label, ref := range map[string]PlatformFabricRef{"source": body.SourceRef, "concept": body.ConceptRef, "intent": body.IntentRef, "blueprint": body.BlueprintRef} {
			if err := validatePlatformFabricRef("body "+body.BodyID+" "+label, ref); err != nil {
				return err
			}
		}
		if strings.TrimSpace(body.AcceptanceMatrixRef.Schema) == "" {
			return fmt.Errorf("platform Fabric body %q acceptance matrix schema is empty", body.BodyID)
		}
		if err := validatePrefixedSHA256("platform Fabric body "+body.BodyID+" acceptance matrix sha256", body.AcceptanceMatrixRef.SHA256); err != nil {
			return err
		}
		if body.ConceptRef != receipt.ConceptRef {
			return fmt.Errorf("platform Fabric body %q lost shared concept lineage", body.BodyID)
		}
		if body.IntentRef.ID != body.BodyID || body.IntentRef.Schema != "axm.fabric-declarative-blueprint-intent/v1" {
			return fmt.Errorf("platform Fabric body %q intent binding mismatch", body.BodyID)
		}
		if body.BlueprintRef.ID != body.BodyID || body.BlueprintRef.Schema != "axm.fabric-declarative-capability-blueprint/v1" {
			return fmt.Errorf("platform Fabric body %q blueprint binding mismatch", body.BodyID)
		}
		if body.AcceptanceMatrixRef.Schema != "axm.fabric-blueprint-acceptance-matrix/v1" {
			return fmt.Errorf("platform Fabric body %q acceptance matrix schema mismatch", body.BodyID)
		}
		if seenBlueprints[body.BlueprintRef.SHA256] {
			return errors.New("different platform Fabric bodies collapsed to one blueprint digest")
		}
		seenBlueprints[body.BlueprintRef.SHA256] = true
		if len(body.InterfaceSchemaRefs) != 2 {
			return fmt.Errorf("platform Fabric body %q requires exactly two interface schema refs", body.BodyID)
		}
		directions := map[string]bool{}
		for _, ref := range body.InterfaceSchemaRefs {
			if ref.Direction != "input" && ref.Direction != "output" {
				return fmt.Errorf("platform Fabric body %q has unsupported interface direction %q", body.BodyID, ref.Direction)
			}
			if directions[ref.Direction] {
				return fmt.Errorf("platform Fabric body %q duplicates interface direction %q", body.BodyID, ref.Direction)
			}
			directions[ref.Direction] = true
			if strings.TrimSpace(ref.Schema) == "" {
				return fmt.Errorf("platform Fabric body %q interface schema is empty", body.BodyID)
			}
			if err := validatePrefixedSHA256("platform Fabric interface sha256", ref.SHA256); err != nil {
				return err
			}
		}
		if !body.Truth.DeterministicReplayMatched || !body.Truth.DesiredOutcomesRemainUnproven || body.Truth.AcceptanceCasesExecuted || body.Truth.PermissionsGranted || body.Truth.GeneratedCodeExecuted || body.Truth.Installed || body.Truth.Promoted || body.Truth.CanonChanged {
			return fmt.Errorf("platform Fabric body %q truth boundary drifted", body.BodyID)
		}
		if body.SourceMode == PlatformFabricDeterministicSource {
			if body.SourceRef.ID != body.BodyID || body.SourceRef.Schema != "axm.waldo-mirror.fabric-deterministic-body/v0.1" {
				return fmt.Errorf("platform Fabric deterministic body %q source binding mismatch", body.BodyID)
			}
		} else if receipt.AIProposalRef == nil || body.SourceRef != *receipt.AIProposalRef {
			return fmt.Errorf("platform Fabric AI body %q is not bound to the explicit proposal", body.BodyID)
		}
	}
	if receipt.AIOptInUsed && aiBodies != 1 {
		return errors.New("platform Fabric AI opt-in receipt requires exactly one AI-proposed body")
	}
	if !receipt.AIOptInUsed && aiBodies != 0 {
		return errors.New("platform Fabric deterministic receipt contains an AI-proposed body")
	}
	if receipt.PrivateOrganFactory.Included || receipt.PrivateOrganFactory.Availability != "NOT_PUBLIC_IN_DONOR" || receipt.PrivateOrganFactory.EffectOnThisResult != "NONE" {
		return errors.New("platform Fabric private Organ Factory boundary drifted")
	}
	if receipt.Truth.DonorFabricModified || receipt.Truth.LiveAIInvocationPerformedByHarness || receipt.Truth.NetworkRequestedByHarness || receipt.Truth.GeneratedExecutableCodeProduced || receipt.Truth.CandidateCodeExecuted || receipt.Truth.RuntimeBehaviorProven || receipt.Truth.HumanQualityApproved || receipt.Truth.InstallationPerformed || receipt.Truth.PromotionPerformed || receipt.Truth.CanonChanged {
		return errors.New("platform Fabric top-level truth boundary drifted")
	}
	return validatePrefixedSHA256("platform Fabric receiptSha256", receipt.ReceiptSHA256)
}

func validatePlatformFabricRef(label string, ref PlatformFabricRef) error {
	if strings.TrimSpace(ref.ID) == "" || strings.TrimSpace(ref.Schema) == "" {
		return fmt.Errorf("%s reference id and schema are required", label)
	}
	return validatePrefixedSHA256(label+" sha256", ref.SHA256)
}

func validatePrefixedSHA256(name, value string) error {
	if !strings.HasPrefix(value, "sha256:") {
		return fmt.Errorf("%s must use sha256: prefix", name)
	}
	hexPart := strings.TrimPrefix(value, "sha256:")
	if len(hexPart) != 64 || hexPart != strings.ToLower(hexPart) {
		return fmt.Errorf("%s must contain 64 lowercase hex characters", name)
	}
	decoded, err := hex.DecodeString(hexPart)
	if err != nil || len(decoded) != sha256.Size {
		return fmt.Errorf("%s must contain a valid SHA-256 digest", name)
	}
	return nil
}

func stringMapEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
