package axmmirror

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	CapabilityTranslationDeclarationSchema = "axm.waldo-witness.capability-translation-declaration/v0.1"
	TranslationLossReceiptSchema           = "axm.waldo-witness.translation-loss-receipt/v0.1"

	TranslationStateNotRequired    = "TRANSLATION_NOT_REQUIRED"
	TranslationStateNoDeclaredLoss = "TRANSLATION_NO_DECLARED_LOSS"
	TranslationStateReviewLoss     = "REVIEW_DECLARED_TRANSLATION_LOSS"
	TranslationStateBlocking       = "HOLD_BLOCKING_TRANSLATION_LOSS"

	TranslationDirectionInput  = "INPUT"
	TranslationDirectionOutput = "OUTPUT"

	TranslationMethodIdentity       = "IDENTITY"
	TranslationMethodRename         = "RENAME"
	TranslationMethodUnitConversion = "UNIT_CONVERSION"
	TranslationMethodApproximation  = "APPROXIMATION"
	TranslationMethodDrop           = "DROP"
	TranslationMethodSynthesize     = "SYNTHESIZE"

	TranslationLossNone     = "NONE"
	TranslationLossDeclared = "DECLARED"
	TranslationLossBlocking = "BLOCKING"

	MaxCapabilityTranslationMappings = 4096
)

type CapabilityFieldTranslation struct {
	Direction   string `json:"direction"`
	SourceField string `json:"source_field,omitempty"`
	TargetField string `json:"target_field,omitempty"`
	Method      string `json:"method"`
	Loss        string `json:"loss"`
	Note        string `json:"note,omitempty"`
}

type CapabilityTranslationDeclaration struct {
	Schema           string                       `json:"schema"`
	TranslationID    string                       `json:"translation_id"`
	PlanSHA256       string                       `json:"plan_sha256"`
	ProviderID       string                       `json:"provider_id"`
	InputFromSchema  string                       `json:"input_from_schema"`
	InputToSchema    string                       `json:"input_to_schema"`
	OutputFromSchema string                       `json:"output_from_schema"`
	OutputToSchema   string                       `json:"output_to_schema"`
	Mappings         []CapabilityFieldTranslation `json:"mappings"`
	DeclaredAt       string                       `json:"declared_at"`
	Authority        Authority                    `json:"authority"`
}

type TranslationLossFinding struct {
	Direction   string `json:"direction"`
	SourceField string `json:"source_field,omitempty"`
	TargetField string `json:"target_field,omitempty"`
	Method      string `json:"method"`
	Loss        string `json:"loss"`
	Note        string `json:"note"`
}

// TranslationLossReceipt makes caller-declared mapping loss explicit. It does
// not generate an adapter, execute a conversion, or prove semantic fidelity.
type TranslationLossReceipt struct {
	Schema                    string                           `json:"schema"`
	State                     string                           `json:"state"`
	Declaration               CapabilityTranslationDeclaration `json:"declaration"`
	DeclarationSHA256         string                           `json:"declaration_sha256"`
	PlanState                 string                           `json:"plan_state"`
	DeclaredLosses            []TranslationLossFinding         `json:"declared_losses,omitempty"`
	HumanReviewRequired       bool                             `json:"human_review_required"`
	AutomaticAdapterGenerated bool                             `json:"automatic_adapter_generated"`
	Notices                   []string                         `json:"notices"`
	Authority                 Authority                        `json:"authority"`
	ReceiptSHA256             string                           `json:"receipt_sha256,omitempty"`
}

func SealCapabilityTranslation(plan CapabilityHandoffPlan, declaration CapabilityTranslationDeclaration) (TranslationLossReceipt, error) {
	if err := plan.Validate(); err != nil {
		return TranslationLossReceipt{}, fmt.Errorf("capability handoff plan: %w", err)
	}
	canonical, err := canonicalizeTranslationDeclaration(declaration)
	if err != nil {
		return TranslationLossReceipt{}, err
	}
	if canonical.PlanSHA256 != plan.PlanSHA256 {
		return TranslationLossReceipt{}, errors.New("translation declaration does not bind the supplied capability handoff plan")
	}
	declaredAt, _ := parseCapabilityTime("capability translation declared_at", canonical.DeclaredAt)
	planCreatedAt, _ := parseCapabilityTime("capability gap created_at", plan.Request.CreatedAt)
	planExpiresAt, _ := parseCapabilityTime("capability gap expires_at", plan.Request.ExpiresAt)
	if declaredAt.Before(planCreatedAt) || declaredAt.After(planExpiresAt) {
		return TranslationLossReceipt{}, errors.New("translation declared_at must fall within the capability handoff plan lifetime")
	}
	candidate, ok := findCapabilityCandidate(plan.Candidates, canonical.ProviderID)
	if !ok {
		return TranslationLossReceipt{}, fmt.Errorf("translation provider %q is not a candidate in the supplied plan", canonical.ProviderID)
	}
	if canonical.InputFromSchema != plan.Request.Input.Schema || canonical.OutputToSchema != plan.Request.RequiredOutputSchema {
		return TranslationLossReceipt{}, errors.New("translation outer schemas do not match the capability gap request")
	}
	if !containsString(candidate.InputSchemas, canonical.InputToSchema) || !containsString(candidate.OutputSchemas, canonical.OutputFromSchema) {
		return TranslationLossReceipt{}, errors.New("translation provider schemas are not declared by the named plan candidate")
	}
	if canonical.InputFromSchema != canonical.InputToSchema && !hasTranslationDirection(canonical.Mappings, TranslationDirectionInput) {
		return TranslationLossReceipt{}, errors.New("different input schemas require at least one explicit INPUT mapping")
	}
	if canonical.OutputFromSchema != canonical.OutputToSchema && !hasTranslationDirection(canonical.Mappings, TranslationDirectionOutput) {
		return TranslationLossReceipt{}, errors.New("different output schemas require at least one explicit OUTPUT mapping")
	}
	declarationDigest, err := digestJSON(canonical, "capability translation declaration")
	if err != nil {
		return TranslationLossReceipt{}, err
	}
	state, losses := translationState(canonical)
	receipt := TranslationLossReceipt{
		Schema: TranslationLossReceiptSchema, State: state, Declaration: canonical,
		DeclarationSHA256: declarationDigest, PlanState: plan.State, DeclaredLosses: losses,
		HumanReviewRequired:       state == TranslationStateReviewLoss || state == TranslationStateBlocking,
		AutomaticAdapterGenerated: false,
		Notices: []string{
			"NO_DECLARED_LOSS means only that this declaration names no loss; it is not proof of semantic equivalence, conformance, fidelity, or successful conversion",
			"the receipt records mappings and loss but does not generate, install, select, invoke, or verify an adapter",
			"translation cannot transfer permissions or upgrade a held handoff plan; execution, promotion, CANON, and world-action authority remain closed",
		},
		Authority: Authority{},
	}
	receipt.ReceiptSHA256, err = translationLossReceiptDigest(receipt)
	if err != nil {
		return TranslationLossReceipt{}, err
	}
	if err := receipt.Validate(); err != nil {
		return TranslationLossReceipt{}, fmt.Errorf("generated translation loss receipt: %w", err)
	}
	return receipt, nil
}

func canonicalizeTranslationDeclaration(declaration CapabilityTranslationDeclaration) (CapabilityTranslationDeclaration, error) {
	if declaration.Schema != CapabilityTranslationDeclarationSchema {
		return CapabilityTranslationDeclaration{}, fmt.Errorf("capability translation declaration schema must be %q", CapabilityTranslationDeclarationSchema)
	}
	for name, value := range map[string]string{
		"translation_id": declaration.TranslationID, "provider_id": declaration.ProviderID,
		"input_from_schema": declaration.InputFromSchema, "input_to_schema": declaration.InputToSchema,
		"output_from_schema": declaration.OutputFromSchema, "output_to_schema": declaration.OutputToSchema,
	} {
		if err := requireTrimmed("capability translation "+name, value); err != nil {
			return CapabilityTranslationDeclaration{}, err
		}
	}
	if err := validateSHA256("capability translation plan_sha256", declaration.PlanSHA256); err != nil {
		return CapabilityTranslationDeclaration{}, err
	}
	declaredAt, err := parseCapabilityTime("capability translation declared_at", declaration.DeclaredAt)
	if err != nil {
		return CapabilityTranslationDeclaration{}, err
	}
	if !declaration.Authority.closed() {
		return CapabilityTranslationDeclaration{}, errors.New("capability translation declaration must carry closed authority")
	}
	if len(declaration.Mappings) > MaxCapabilityTranslationMappings {
		return CapabilityTranslationDeclaration{}, fmt.Errorf("capability translation has %d mappings; maximum is %d", len(declaration.Mappings), MaxCapabilityTranslationMappings)
	}
	canonical := declaration
	canonical.DeclaredAt = declaredAt.UTC().Format(time.RFC3339Nano)
	canonical.Mappings = append([]CapabilityFieldTranslation(nil), declaration.Mappings...)
	for i, mapping := range canonical.Mappings {
		if err := validateCapabilityFieldTranslation(i, mapping); err != nil {
			return CapabilityTranslationDeclaration{}, err
		}
	}
	sort.Slice(canonical.Mappings, func(i, j int) bool {
		left, right := canonical.Mappings[i], canonical.Mappings[j]
		if left.Direction != right.Direction {
			return left.Direction < right.Direction
		}
		if left.SourceField != right.SourceField {
			return left.SourceField < right.SourceField
		}
		if left.TargetField != right.TargetField {
			return left.TargetField < right.TargetField
		}
		return left.Method < right.Method
	})
	for i := 1; i < len(canonical.Mappings); i++ {
		left, right := canonical.Mappings[i-1], canonical.Mappings[i]
		if left.Direction == right.Direction && left.SourceField == right.SourceField && left.TargetField == right.TargetField {
			return CapabilityTranslationDeclaration{}, fmt.Errorf("duplicate %s translation mapping %q to %q", right.Direction, right.SourceField, right.TargetField)
		}
	}
	return canonical, nil
}

func validateCapabilityFieldTranslation(index int, mapping CapabilityFieldTranslation) error {
	if !oneOf(mapping.Direction, TranslationDirectionInput, TranslationDirectionOutput) {
		return fmt.Errorf("translation mapping %d has unsupported direction %q", index, mapping.Direction)
	}
	if !oneOf(mapping.Method, TranslationMethodIdentity, TranslationMethodRename, TranslationMethodUnitConversion,
		TranslationMethodApproximation, TranslationMethodDrop, TranslationMethodSynthesize) {
		return fmt.Errorf("translation mapping %d has unsupported method %q", index, mapping.Method)
	}
	if !oneOf(mapping.Loss, TranslationLossNone, TranslationLossDeclared, TranslationLossBlocking) {
		return fmt.Errorf("translation mapping %d has unsupported loss %q", index, mapping.Loss)
	}
	if mapping.Method == TranslationMethodSynthesize {
		if mapping.SourceField != "" || strings.TrimSpace(mapping.TargetField) == "" {
			return fmt.Errorf("translation mapping %d SYNTHESIZE requires only target_field", index)
		}
	} else if mapping.Method == TranslationMethodDrop {
		if strings.TrimSpace(mapping.SourceField) == "" || mapping.TargetField != "" {
			return fmt.Errorf("translation mapping %d DROP requires only source_field", index)
		}
	} else if strings.TrimSpace(mapping.SourceField) == "" || strings.TrimSpace(mapping.TargetField) == "" {
		return fmt.Errorf("translation mapping %d requires source_field and target_field", index)
	}
	if mapping.SourceField != strings.TrimSpace(mapping.SourceField) || mapping.TargetField != strings.TrimSpace(mapping.TargetField) {
		return fmt.Errorf("translation mapping %d fields must be trimmed", index)
	}
	if (mapping.Method == TranslationMethodIdentity || mapping.Method == TranslationMethodRename) && mapping.Loss != TranslationLossNone {
		return fmt.Errorf("translation mapping %d %s must declare loss NONE", index, mapping.Method)
	}
	if (mapping.Method == TranslationMethodUnitConversion || mapping.Method == TranslationMethodApproximation ||
		mapping.Method == TranslationMethodDrop || mapping.Method == TranslationMethodSynthesize) && mapping.Loss == TranslationLossNone {
		return fmt.Errorf("translation mapping %d %s must declare loss", index, mapping.Method)
	}
	if mapping.Method != TranslationMethodIdentity && (strings.TrimSpace(mapping.Note) == "" || mapping.Note != strings.TrimSpace(mapping.Note)) {
		return fmt.Errorf("translation mapping %d %s requires a trimmed note", index, mapping.Method)
	}
	if mapping.Method == TranslationMethodIdentity && mapping.Note != strings.TrimSpace(mapping.Note) {
		return fmt.Errorf("translation mapping %d note must be trimmed", index)
	}
	return nil
}

func translationState(declaration CapabilityTranslationDeclaration) (string, []TranslationLossFinding) {
	var losses []TranslationLossFinding
	blocking := false
	for _, mapping := range declaration.Mappings {
		if mapping.Loss == TranslationLossNone {
			continue
		}
		losses = append(losses, TranslationLossFinding{
			Direction: mapping.Direction, SourceField: mapping.SourceField, TargetField: mapping.TargetField,
			Method: mapping.Method, Loss: mapping.Loss, Note: mapping.Note,
		})
		if mapping.Loss == TranslationLossBlocking {
			blocking = true
		}
	}
	if blocking {
		return TranslationStateBlocking, losses
	}
	if len(losses) != 0 {
		return TranslationStateReviewLoss, losses
	}
	if declaration.InputFromSchema == declaration.InputToSchema && declaration.OutputFromSchema == declaration.OutputToSchema && len(declaration.Mappings) == 0 {
		return TranslationStateNotRequired, nil
	}
	return TranslationStateNoDeclaredLoss, nil
}

func (receipt TranslationLossReceipt) Validate() error {
	if receipt.Schema != TranslationLossReceiptSchema || !oneOf(receipt.State,
		TranslationStateNotRequired, TranslationStateNoDeclaredLoss, TranslationStateReviewLoss, TranslationStateBlocking) {
		return fmt.Errorf("unsupported translation loss receipt identity %q state %q", receipt.Schema, receipt.State)
	}
	canonical, err := canonicalizeTranslationDeclaration(receipt.Declaration)
	if err != nil {
		return err
	}
	if !translationDeclarationsEqual(canonical, receipt.Declaration) {
		return errors.New("translation loss declaration is not canonical")
	}
	for name, value := range map[string]string{"declaration_sha256": receipt.DeclarationSHA256, "receipt_sha256": receipt.ReceiptSHA256} {
		if err := validateSHA256("translation loss receipt "+name, value); err != nil {
			return err
		}
	}
	declarationDigest, err := digestJSON(receipt.Declaration, "capability translation declaration")
	if err != nil {
		return err
	}
	if declarationDigest != receipt.DeclarationSHA256 {
		return errors.New("translation declaration digest does not match")
	}
	if !validHandoffState(receipt.PlanState) {
		return fmt.Errorf("translation loss receipt has unsupported plan_state %q", receipt.PlanState)
	}
	expectedState, expectedLosses := translationState(receipt.Declaration)
	if receipt.State != expectedState || !translationLossesEqual(expectedLosses, receipt.DeclaredLosses) {
		return errors.New("translation loss receipt state or findings do not match the declaration")
	}
	expectedReview := receipt.State == TranslationStateReviewLoss || receipt.State == TranslationStateBlocking
	if receipt.HumanReviewRequired != expectedReview || receipt.AutomaticAdapterGenerated {
		return errors.New("translation loss receipt has invalid review or automatic-adapter state")
	}
	if len(receipt.Notices) == 0 || !receipt.Authority.closed() {
		return errors.New("translation loss receipt requires notices and closed authority")
	}
	digest, err := translationLossReceiptDigest(receipt)
	if err != nil {
		return err
	}
	if digest != receipt.ReceiptSHA256 {
		return errors.New("translation loss receipt digest does not match")
	}
	return nil
}

func findCapabilityCandidate(candidates []CapabilityRouteCandidate, providerID string) (CapabilityRouteCandidate, bool) {
	for _, candidate := range candidates {
		if candidate.ProviderID == providerID {
			return candidate, true
		}
	}
	return CapabilityRouteCandidate{}, false
}

func hasTranslationDirection(mappings []CapabilityFieldTranslation, direction string) bool {
	for _, mapping := range mappings {
		if mapping.Direction == direction {
			return true
		}
	}
	return false
}

func validHandoffState(state string) bool {
	return oneOf(state, HandoffStateReady, HandoffStateLocal, HandoffStateExternalHold, HandoffStateSourceOnly,
		HandoffStateUnavailable, HandoffStateDataClass, HandoffStateSchemaTranslation,
		HandoffStateProviderAmbiguous, HandoffStatePermission)
}

func translationLossReceiptDigest(receipt TranslationLossReceipt) (string, error) {
	receipt.ReceiptSHA256 = ""
	return digestJSON(receipt, "translation loss receipt")
}

func translationDeclarationsEqual(left, right CapabilityTranslationDeclaration) bool {
	leftDigest, err := digestJSON(left, "left translation declaration")
	if err != nil {
		return false
	}
	rightDigest, err := digestJSON(right, "right translation declaration")
	return err == nil && leftDigest == rightDigest
}

func translationLossesEqual(left, right []TranslationLossFinding) bool {
	leftDigest, err := digestJSON(left, "left translation losses")
	if err != nil {
		return false
	}
	rightDigest, err := digestJSON(right, "right translation losses")
	return err == nil && leftDigest == rightDigest
}
