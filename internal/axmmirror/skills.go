package axmmirror

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	SkillContinuityRequestSchema = "axm.waldo-witness.skill-continuity-request/v0.1"
	SkillContinuityReceiptSchema = "axm.waldo-witness.skill-continuity-receipt/v0.1"

	SkillContinuityReady             = "SKILL_CONTINUITY_READY"
	SkillContinuityInventoryHold     = "HOLD_SKILL_INVENTORY_INCOMPLETE"
	SkillContinuityBackupHold        = "HOLD_SKILL_BACKUP_MISSING"
	SkillContinuityDriftHold         = "HOLD_SKILL_DRIFT"
	SkillContinuityCompatibilityHold = "HOLD_SKILL_INCOMPATIBLE"

	SkillUseKnowledge   = "KNOWLEDGE"
	SkillUseInstruction = "INSTRUCTION"
	SkillUseAdapter     = "ADAPTER"

	SkillFindingMatched       = "MATCHED"
	SkillFindingMissingNow    = "MISSING_CURRENT"
	SkillFindingMissingBackup = "MISSING_BACKUP"
	SkillFindingDrifted       = "DRIFTED"
	SkillFindingIncompatible  = "INCOMPATIBLE"
)

type SkillRequirement struct {
	ID       string `json:"id"`
	UseClass string `json:"use_class"`
}

// SkillInventoryItem distinguishes knowledge, instructions, and executable
// adapters. A source digest says which material was inventoried; it never
// installs the item or proves the behavior named by its metadata.
type SkillInventoryItem struct {
	ID                  string `json:"id"`
	Version             string `json:"version"`
	UseClass            string `json:"use_class"`
	SourceSHA256        string `json:"source_sha256"`
	InstructionSHA256   string `json:"instruction_sha256,omitempty"`
	Status              string `json:"status"`
	ExecutionStatus     string `json:"execution_status"`
	ProofStatus         string `json:"proof_status"`
	CompatibilityStatus string `json:"compatibility_status"`
	MemberCount         int    `json:"member_count,omitempty"`
}

type SkillInventorySet struct {
	SetID      string               `json:"set_id"`
	CapturedAt string               `json:"captured_at"`
	Complete   bool                 `json:"complete"`
	Items      []SkillInventoryItem `json:"items"`
}

type SkillContinuityRequest struct {
	Schema                        string             `json:"schema"`
	AssessmentID                  string             `json:"assessment_id"`
	TargetAnsweringIdentitySHA256 string             `json:"target_answering_identity_sha256"`
	Requirements                  []SkillRequirement `json:"requirements"`
	Current                       SkillInventorySet  `json:"current"`
	Backup                        SkillInventorySet  `json:"backup"`
}

type SkillContinuityFinding struct {
	SkillID           string `json:"skill_id"`
	UseClass          string `json:"use_class"`
	Status            string `json:"status"`
	Basis             string `json:"basis"`
	CurrentItemSHA256 string `json:"current_item_sha256,omitempty"`
	BackupItemSHA256  string `json:"backup_item_sha256,omitempty"`
}

// SkillRecoveryCandidate is a review plan, not an operation. It deliberately
// has no path, command, installer, permission, or auto-apply field.
type SkillRecoveryCandidate struct {
	SkillID          string `json:"skill_id"`
	BackupItemSHA256 string `json:"backup_item_sha256"`
	Reason           string `json:"reason"`
	ProposedAction   string `json:"proposed_action"`
	HumanReview      bool   `json:"human_review_required"`
	Automatic        bool   `json:"automatic"`
}

type SkillContinuityReceipt struct {
	Schema                        string                   `json:"schema"`
	State                         string                   `json:"state"`
	AssessmentID                  string                   `json:"assessment_id"`
	TargetAnsweringIdentitySHA256 string                   `json:"target_answering_identity_sha256"`
	RequestSHA256                 string                   `json:"request_sha256"`
	CurrentSetSHA256              string                   `json:"current_set_sha256"`
	BackupSetSHA256               string                   `json:"backup_set_sha256"`
	Requirements                  []SkillRequirement       `json:"requirements"`
	CurrentComplete               bool                     `json:"current_complete"`
	BackupComplete                bool                     `json:"backup_complete"`
	Findings                      []SkillContinuityFinding `json:"findings"`
	RecoveryCandidates            []SkillRecoveryCandidate `json:"recovery_candidates,omitempty"`
	ReceiptSHA256                 string                   `json:"receipt_sha256,omitempty"`
	Holds                         []string                 `json:"holds,omitempty"`
	Notices                       []string                 `json:"notices"`
	Authority                     Authority                `json:"authority"`
}

// AssessSkillContinuity compares exact current and backup manifests. It can
// name a recovery candidate, but cannot read backup bytes, install, restore,
// enable, execute, grant permission, or mutate the target clone.
func AssessSkillContinuity(request SkillContinuityRequest) (SkillContinuityReceipt, error) {
	canonical, err := validateAndCanonicalizeSkillRequest(request)
	if err != nil {
		return SkillContinuityReceipt{}, err
	}
	requestDigest, err := digestJSON(canonical, "skill continuity request")
	if err != nil {
		return SkillContinuityReceipt{}, err
	}
	currentDigest, err := digestJSON(canonical.Current, "current skill inventory")
	if err != nil {
		return SkillContinuityReceipt{}, err
	}
	backupDigest, err := digestJSON(canonical.Backup, "backup skill inventory")
	if err != nil {
		return SkillContinuityReceipt{}, err
	}
	current := indexSkillItems(canonical.Current.Items)
	backup := indexSkillItems(canonical.Backup.Items)
	receipt := SkillContinuityReceipt{
		Schema: SkillContinuityReceiptSchema, State: SkillContinuityReady,
		AssessmentID: canonical.AssessmentID, TargetAnsweringIdentitySHA256: canonical.TargetAnsweringIdentitySHA256,
		RequestSHA256: requestDigest, CurrentSetSHA256: currentDigest, BackupSetSHA256: backupDigest,
		Requirements: canonical.Requirements, CurrentComplete: canonical.Current.Complete, BackupComplete: canonical.Backup.Complete,
		Findings: make([]SkillContinuityFinding, 0, len(canonical.Requirements)),
		Notices: []string{
			"knowledge, instruction, and executable-adapter inventory states remain distinct; archive membership never becomes runtime availability",
			"a matching backup proves only exact manifest continuity and does not prove compatibility, safety, or successful restoration",
			"recovery candidates require separate human review and host authorization; this receipt performs no install, restore, enable, execution, permission, promotion, or CANON action",
		},
		Authority: Authority{},
	}
	for _, requirement := range canonical.Requirements {
		finding, recovery, err := assessSkillRequirement(requirement, current[requirement.ID], backup[requirement.ID])
		if err != nil {
			return SkillContinuityReceipt{}, err
		}
		receipt.Findings = append(receipt.Findings, finding)
		if recovery != nil {
			receipt.RecoveryCandidates = append(receipt.RecoveryCandidates, *recovery)
		}
	}
	receipt.State, receipt.Holds = skillContinuityState(canonical, receipt.Findings)
	receipt.ReceiptSHA256, err = skillContinuityReceiptDigest(receipt)
	if err != nil {
		return SkillContinuityReceipt{}, err
	}
	if err := receipt.Validate(); err != nil {
		return SkillContinuityReceipt{}, fmt.Errorf("generated skill continuity receipt: %w", err)
	}
	return receipt, nil
}

func validateAndCanonicalizeSkillRequest(request SkillContinuityRequest) (SkillContinuityRequest, error) {
	if request.Schema != SkillContinuityRequestSchema {
		return SkillContinuityRequest{}, fmt.Errorf("skill continuity request schema must be %q", SkillContinuityRequestSchema)
	}
	if strings.TrimSpace(request.AssessmentID) == "" || request.AssessmentID != strings.TrimSpace(request.AssessmentID) {
		return SkillContinuityRequest{}, errors.New("skill continuity assessment_id is required and must be trimmed")
	}
	if err := validateSHA256("skill continuity target_answering_identity_sha256", request.TargetAnsweringIdentitySHA256); err != nil {
		return SkillContinuityRequest{}, err
	}
	if len(request.Requirements) == 0 {
		return SkillContinuityRequest{}, errors.New("skill continuity requires at least one skill")
	}
	canonical := request
	canonical.Requirements = append([]SkillRequirement(nil), request.Requirements...)
	sort.Slice(canonical.Requirements, func(i, j int) bool { return canonical.Requirements[i].ID < canonical.Requirements[j].ID })
	for i, requirement := range canonical.Requirements {
		if strings.TrimSpace(requirement.ID) == "" || requirement.ID != strings.TrimSpace(requirement.ID) || !oneOf(requirement.UseClass, SkillUseKnowledge, SkillUseInstruction, SkillUseAdapter) {
			return SkillContinuityRequest{}, fmt.Errorf("skill requirement %d is incomplete or has unsupported use_class %q", i, requirement.UseClass)
		}
		if i > 0 && canonical.Requirements[i-1].ID == requirement.ID {
			return SkillContinuityRequest{}, fmt.Errorf("duplicate skill requirement %q", requirement.ID)
		}
	}
	var err error
	canonical.Current, err = canonicalizeSkillSet("current", request.Current)
	if err != nil {
		return SkillContinuityRequest{}, err
	}
	canonical.Backup, err = canonicalizeSkillSet("backup", request.Backup)
	if err != nil {
		return SkillContinuityRequest{}, err
	}
	return canonical, nil
}

func canonicalizeSkillSet(name string, set SkillInventorySet) (SkillInventorySet, error) {
	if strings.TrimSpace(set.SetID) == "" || set.SetID != strings.TrimSpace(set.SetID) {
		return SkillInventorySet{}, fmt.Errorf("%s skill set_id is required and must be trimmed", name)
	}
	if _, err := time.Parse(time.RFC3339Nano, set.CapturedAt); err != nil {
		return SkillInventorySet{}, fmt.Errorf("parse %s skill captured_at: %w", name, err)
	}
	canonical := set
	canonical.Items = append([]SkillInventoryItem(nil), set.Items...)
	sort.Slice(canonical.Items, func(i, j int) bool { return canonical.Items[i].ID < canonical.Items[j].ID })
	for i, item := range canonical.Items {
		if err := validateSkillItem(fmt.Sprintf("%s skill item %d", name, i), item); err != nil {
			return SkillInventorySet{}, err
		}
		if i > 0 && canonical.Items[i-1].ID == item.ID {
			return SkillInventorySet{}, fmt.Errorf("%s skill set has duplicate item %q", name, item.ID)
		}
	}
	return canonical, nil
}

func validateSkillItem(name string, item SkillInventoryItem) error {
	if strings.TrimSpace(item.ID) == "" || item.ID != strings.TrimSpace(item.ID) || strings.TrimSpace(item.Version) == "" || item.Version != strings.TrimSpace(item.Version) {
		return fmt.Errorf("%s requires trimmed id and version", name)
	}
	if !oneOf(item.UseClass, SkillUseKnowledge, SkillUseInstruction, SkillUseAdapter) {
		return fmt.Errorf("%s has unsupported use_class %q", name, item.UseClass)
	}
	if err := validateSHA256(name+".source_sha256", item.SourceSHA256); err != nil {
		return err
	}
	if item.InstructionSHA256 != "" {
		if err := validateSHA256(name+".instruction_sha256", item.InstructionSHA256); err != nil {
			return err
		}
	}
	if item.UseClass == SkillUseInstruction && item.InstructionSHA256 == "" {
		return fmt.Errorf("%s instruction item requires instruction_sha256", name)
	}
	if !oneOf(item.Status, "TEST", "EXPERIMENTAL", "WORKING", "UNKNOWN") {
		return fmt.Errorf("%s has unsupported status %q", name, item.Status)
	}
	if !oneOf(item.ExecutionStatus, "INERT", "HOST_MEDIATED", "EXECUTABLE", "UNAVAILABLE") {
		return fmt.Errorf("%s has unsupported execution_status %q", name, item.ExecutionStatus)
	}
	if !oneOf(item.ProofStatus, "RUNTIME_PASS", "CONTRACT_PASS", "UNTESTED", "UNKNOWN") {
		return fmt.Errorf("%s has unsupported proof_status %q", name, item.ProofStatus)
	}
	if !oneOf(item.CompatibilityStatus, "VERIFIED_WITHIN_CONTRACT", "UNVERIFIED", "INCOMPATIBLE", "UNKNOWN") {
		return fmt.Errorf("%s has unsupported compatibility_status %q", name, item.CompatibilityStatus)
	}
	if item.MemberCount < 0 {
		return fmt.Errorf("%s member_count must not be negative", name)
	}
	return nil
}

func indexSkillItems(items []SkillInventoryItem) map[string]*SkillInventoryItem {
	result := make(map[string]*SkillInventoryItem, len(items))
	for i := range items {
		result[items[i].ID] = &items[i]
	}
	return result
}

func assessSkillRequirement(requirement SkillRequirement, current, backup *SkillInventoryItem) (SkillContinuityFinding, *SkillRecoveryCandidate, error) {
	finding := SkillContinuityFinding{SkillID: requirement.ID, UseClass: requirement.UseClass}
	var currentDigest, backupDigest string
	var err error
	if current != nil {
		currentDigest, err = digestJSON(*current, "current skill item "+requirement.ID)
		if err != nil {
			return SkillContinuityFinding{}, nil, err
		}
		finding.CurrentItemSHA256 = currentDigest
	}
	if backup != nil {
		backupDigest, err = digestJSON(*backup, "backup skill item "+requirement.ID)
		if err != nil {
			return SkillContinuityFinding{}, nil, err
		}
		finding.BackupItemSHA256 = backupDigest
	}
	var recovery *SkillRecoveryCandidate
	makeRecovery := func(reason string) *SkillRecoveryCandidate {
		if backupDigest == "" {
			return nil
		}
		return &SkillRecoveryCandidate{
			SkillID: requirement.ID, BackupItemSHA256: backupDigest, Reason: reason,
			ProposedAction: "REVIEW_RESTORE_CANDIDATE", HumanReview: true, Automatic: false,
		}
	}
	switch {
	case current == nil:
		finding.Status = SkillFindingMissingNow
		finding.Basis = "the required skill is absent from the complete current inventory"
		recovery = makeRecovery("current item is missing")
	case current.UseClass != requirement.UseClass || !skillItemUsableForRequirement(*current, requirement):
		finding.Status = SkillFindingIncompatible
		finding.Basis = "the current item does not satisfy the requested knowledge, instruction, or adapter role within its declared contract"
		recovery = makeRecovery("current item does not satisfy the declared role")
	case backup == nil:
		finding.Status = SkillFindingMissingBackup
		finding.Basis = "the current item has no exact backup manifest entry"
	case currentDigest != backupDigest:
		finding.Status = SkillFindingDrifted
		finding.Basis = "the current and backup item manifests differ; the receipt does not choose which version is preferable"
		recovery = makeRecovery("current and backup manifests differ")
	default:
		finding.Status = SkillFindingMatched
		finding.Basis = "the required current item exactly matches the backup manifest within the requested role"
	}
	return finding, recovery, nil
}

func skillItemUsableForRequirement(item SkillInventoryItem, requirement SkillRequirement) bool {
	switch requirement.UseClass {
	case SkillUseKnowledge:
		return item.UseClass == SkillUseKnowledge && item.ExecutionStatus == "INERT"
	case SkillUseInstruction:
		return item.UseClass == SkillUseInstruction && item.InstructionSHA256 != "" && item.ExecutionStatus != "UNAVAILABLE"
	case SkillUseAdapter:
		return item.UseClass == SkillUseAdapter && oneOf(item.ExecutionStatus, "EXECUTABLE", "HOST_MEDIATED") && item.CompatibilityStatus == "VERIFIED_WITHIN_CONTRACT"
	default:
		return false
	}
}

func skillContinuityState(request SkillContinuityRequest, findings []SkillContinuityFinding) (string, []string) {
	if !request.Current.Complete {
		return SkillContinuityInventoryHold, []string{"the current skill inventory is declared incomplete"}
	}
	if !request.Backup.Complete {
		return SkillContinuityBackupHold, []string{"the backup skill inventory is declared incomplete"}
	}
	hasMissing, hasDrift, hasIncompatible := false, false, false
	for _, finding := range findings {
		switch finding.Status {
		case SkillFindingMissingNow, SkillFindingMissingBackup:
			hasMissing = true
		case SkillFindingDrifted:
			hasDrift = true
		case SkillFindingIncompatible:
			hasIncompatible = true
		}
	}
	switch {
	case hasMissing:
		return SkillContinuityBackupHold, []string{"one or more required skills are missing from the current or backup inventory"}
	case hasIncompatible:
		return SkillContinuityCompatibilityHold, []string{"one or more current skills do not satisfy the requested role within their declared contract"}
	case hasDrift:
		return SkillContinuityDriftHold, []string{"one or more current skill manifests differ from their backup entries"}
	default:
		return SkillContinuityReady, nil
	}
}

func (receipt SkillContinuityReceipt) Validate() error {
	if receipt.Schema != SkillContinuityReceiptSchema || !oneOf(receipt.State, SkillContinuityReady, SkillContinuityInventoryHold, SkillContinuityBackupHold, SkillContinuityDriftHold, SkillContinuityCompatibilityHold) {
		return fmt.Errorf("unsupported skill continuity identity %q state %q", receipt.Schema, receipt.State)
	}
	if strings.TrimSpace(receipt.AssessmentID) == "" || len(receipt.Requirements) == 0 || len(receipt.Findings) != len(receipt.Requirements) {
		return errors.New("skill continuity receipt has incomplete assessment identity or findings")
	}
	for name, value := range map[string]string{
		"target_answering_identity_sha256": receipt.TargetAnsweringIdentitySHA256,
		"request_sha256":                   receipt.RequestSHA256,
		"current_set_sha256":               receipt.CurrentSetSHA256,
		"backup_set_sha256":                receipt.BackupSetSHA256,
		"receipt_sha256":                   receipt.ReceiptSHA256,
	} {
		if err := validateSHA256("skill continuity "+name, value); err != nil {
			return err
		}
	}
	if !sort.SliceIsSorted(receipt.Requirements, func(i, j int) bool { return receipt.Requirements[i].ID < receipt.Requirements[j].ID }) || !sort.SliceIsSorted(receipt.Findings, func(i, j int) bool { return receipt.Findings[i].SkillID < receipt.Findings[j].SkillID }) {
		return errors.New("skill continuity requirements and findings must be sorted")
	}
	seen := map[string]struct{}{}
	recoveryBySkill := make(map[string]SkillRecoveryCandidate, len(receipt.RecoveryCandidates))
	for i, candidate := range receipt.RecoveryCandidates {
		if strings.TrimSpace(candidate.SkillID) == "" || strings.TrimSpace(candidate.Reason) == "" || candidate.ProposedAction != "REVIEW_RESTORE_CANDIDATE" || !candidate.HumanReview || candidate.Automatic {
			return fmt.Errorf("skill recovery candidate %d grants action or is incomplete", i)
		}
		if err := validateSHA256(fmt.Sprintf("skill recovery candidate %d backup_item_sha256", i), candidate.BackupItemSHA256); err != nil {
			return err
		}
		if _, exists := recoveryBySkill[candidate.SkillID]; exists {
			return fmt.Errorf("duplicate skill recovery candidate %q", candidate.SkillID)
		}
		if i > 0 && receipt.RecoveryCandidates[i-1].SkillID >= candidate.SkillID {
			return errors.New("skill recovery candidates are unsorted")
		}
		recoveryBySkill[candidate.SkillID] = candidate
	}
	expectedRecovery := 0
	for i, requirement := range receipt.Requirements {
		if strings.TrimSpace(requirement.ID) == "" || !oneOf(requirement.UseClass, SkillUseKnowledge, SkillUseInstruction, SkillUseAdapter) {
			return fmt.Errorf("skill continuity requirement %d is invalid", i)
		}
		if _, exists := seen[requirement.ID]; exists {
			return fmt.Errorf("skill continuity has duplicate requirement %q", requirement.ID)
		}
		seen[requirement.ID] = struct{}{}
		finding := receipt.Findings[i]
		if finding.SkillID != requirement.ID || finding.UseClass != requirement.UseClass || strings.TrimSpace(finding.Basis) == "" || !oneOf(finding.Status, SkillFindingMatched, SkillFindingMissingNow, SkillFindingMissingBackup, SkillFindingDrifted, SkillFindingIncompatible) {
			return fmt.Errorf("skill continuity finding %d does not match its requirement", i)
		}
		for name, digest := range map[string]string{"current_item_sha256": finding.CurrentItemSHA256, "backup_item_sha256": finding.BackupItemSHA256} {
			if digest != "" {
				if err := validateSHA256("skill continuity finding "+name, digest); err != nil {
					return err
				}
			}
		}
		requiresRecovery := false
		switch finding.Status {
		case SkillFindingMatched:
			if finding.CurrentItemSHA256 == "" || finding.CurrentItemSHA256 != finding.BackupItemSHA256 {
				return fmt.Errorf("matched skill finding %q lacks equal current and backup digests", finding.SkillID)
			}
		case SkillFindingMissingNow:
			if finding.CurrentItemSHA256 != "" {
				return fmt.Errorf("missing-current skill finding %q carries a current digest", finding.SkillID)
			}
			requiresRecovery = finding.BackupItemSHA256 != ""
		case SkillFindingMissingBackup:
			if finding.CurrentItemSHA256 == "" || finding.BackupItemSHA256 != "" {
				return fmt.Errorf("missing-backup skill finding %q has inconsistent digests", finding.SkillID)
			}
		case SkillFindingDrifted:
			if finding.CurrentItemSHA256 == "" || finding.BackupItemSHA256 == "" || finding.CurrentItemSHA256 == finding.BackupItemSHA256 {
				return fmt.Errorf("drifted skill finding %q lacks distinct current and backup digests", finding.SkillID)
			}
			requiresRecovery = true
		case SkillFindingIncompatible:
			if finding.CurrentItemSHA256 == "" {
				return fmt.Errorf("incompatible skill finding %q lacks a current digest", finding.SkillID)
			}
			requiresRecovery = finding.BackupItemSHA256 != ""
		}
		candidate, hasCandidate := recoveryBySkill[finding.SkillID]
		if requiresRecovery {
			expectedRecovery++
			if !hasCandidate || candidate.BackupItemSHA256 != finding.BackupItemSHA256 {
				return fmt.Errorf("skill finding %q lacks its exact review-only recovery candidate", finding.SkillID)
			}
		} else if hasCandidate {
			return fmt.Errorf("skill finding %q has an inapplicable recovery candidate", finding.SkillID)
		}
	}
	if expectedRecovery != len(receipt.RecoveryCandidates) {
		return errors.New("skill recovery candidates do not match findings")
	}
	expectedState := SkillContinuityReady
	if !receipt.CurrentComplete {
		expectedState = SkillContinuityInventoryHold
	} else if !receipt.BackupComplete {
		expectedState = SkillContinuityBackupHold
	} else {
		hasMissing, hasDrift, hasIncompatible := false, false, false
		for _, finding := range receipt.Findings {
			switch finding.Status {
			case SkillFindingMissingNow, SkillFindingMissingBackup:
				hasMissing = true
			case SkillFindingDrifted:
				hasDrift = true
			case SkillFindingIncompatible:
				hasIncompatible = true
			}
		}
		switch {
		case hasMissing:
			expectedState = SkillContinuityBackupHold
		case hasIncompatible:
			expectedState = SkillContinuityCompatibilityHold
		case hasDrift:
			expectedState = SkillContinuityDriftHold
		}
	}
	if receipt.State != expectedState {
		return fmt.Errorf("skill continuity state is %q; findings require %q", receipt.State, expectedState)
	}
	if receipt.State == SkillContinuityReady {
		if len(receipt.Holds) != 0 || len(receipt.RecoveryCandidates) != 0 {
			return errors.New("ready skill continuity receipt carries holds or recovery candidates")
		}
	} else if len(receipt.Holds) == 0 {
		return errors.New("held skill continuity receipt requires a reason")
	}
	if len(receipt.Notices) == 0 || !receipt.Authority.closed() {
		return errors.New("skill continuity receipt must state its boundaries and carry closed authority")
	}
	expected, err := skillContinuityReceiptDigest(receipt)
	if err != nil {
		return err
	}
	if expected != receipt.ReceiptSHA256 {
		return fmt.Errorf("skill continuity receipt digest mismatch: expected %s, got %s", receipt.ReceiptSHA256, expected)
	}
	return nil
}

func skillContinuityReceiptDigest(receipt SkillContinuityReceipt) (string, error) {
	receipt.ReceiptSHA256 = ""
	return digestJSON(receipt, "skill continuity receipt")
}
