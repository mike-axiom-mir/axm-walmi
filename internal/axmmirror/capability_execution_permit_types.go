package axmmirror

const (
	ExecutionPermitV019Schema             = "axm.capability-execution-permit-contract/v0.19"
	ExecutionPermitV019Status             = "CONTRACT_PROBE_ONLY"
	ExecutionPermitV019Challenge          = "PERMISSION_IS_NOT_ACTION_OR_CAPABILITY"
	ExecutionPermitV019SourceReceipt      = "sha256:768d32a9bc59008065f09b31e8b09be6b1fd20ce77dd9d578c1b76af2474b06f"
	ExecutionPermitV019SourceHead         = "730ef20b2f6c95519a09756f18c8a7fca83fe4f2"
	ExecutionPermitV019CandidateSetDigest = "sha256:d396b84ddaa15f230273e96cb7b6c35dfece8d4d81ffbb46a127cccb2e0b8aa0"
	ExecutionPermitV019CandidateID        = "direct-read-chain"
	ExecutionPermitV019CandidateDigest    = "sha256:8ed1d23b0c7020fe1f3ee2427c2984c6132c46e49f86b0183828c4c1dc775241"
)

type ExecutionPermitV019Boundary struct {
	LocalCapabilityFabricImplementationIncluded bool `json:"localCapabilityFabricImplementationIncluded"`
	ExecutableCandidateBodyIncluded             bool `json:"executableCandidateBodyIncluded"`
	ProbeExecutesCandidate                      bool `json:"probeExecutesCandidate"`
	LiveAIProviderCalled                        bool `json:"liveAiProviderCalled"`
	RealExecutionApprovalObserved               bool `json:"realExecutionApprovalObserved"`
}

type ExecutionPermitV019Permit struct {
	PermitID                       string   `json:"permitId"`
	IssuerClass                    string   `json:"issuerClass"`
	Authenticity                   string   `json:"authenticity"`
	SourceSelectionContractReceipt string   `json:"sourceSelectionContractReceipt"`
	SourceCandidateSetDigest       string   `json:"sourceCandidateSetDigest"`
	SelectedCandidateID            string   `json:"selectedCandidateId"`
	SelectedCandidateDigest        string   `json:"selectedCandidateDigest"`
	Scope                          []string `json:"scope"`
	GrantsInstall                  bool     `json:"grantsInstall"`
	GrantsPromotion                bool     `json:"grantsPromotion"`
	GrantsCanon                    bool     `json:"grantsCanon"`
	WaldoIssued                    bool     `json:"waldoIssued"`
}

type ExecutionPermitV019Request struct {
	RequestID               string   `json:"requestId"`
	RequestedByClass        string   `json:"requestedByClass"`
	PermitID                string   `json:"permitId"`
	SelectedCandidateID     string   `json:"selectedCandidateId"`
	SelectedCandidateDigest string   `json:"selectedCandidateDigest"`
	Action                  string   `json:"action"`
	ExecutableBodyPresent   bool     `json:"executableBodyPresent"`
	AutoBuildMissingBody    bool     `json:"autoBuildMissingBody"`
	InventedBody            bool     `json:"inventedBody"`
	RequestedSideEffects    []string `json:"requestedSideEffects"`
}

type ExecutionPermitV019RefusalCase struct {
	CaseID          string `json:"caseId"`
	ExpectedOutcome string `json:"expectedOutcome"`
	ExpectedReason  string `json:"expectedReason"`
}

type ExecutionPermitV019DuplicateCase struct {
	DuplicatePermitID string `json:"duplicatePermitId"`
	ExpectedOutcome   string `json:"expectedOutcome"`
	ExtraAuthority    bool   `json:"extraAuthority"`
	ExtraAction       bool   `json:"extraAction"`
}

type ExecutionPermitV019Truth struct {
	PermissionAccepted             bool `json:"permissionAccepted"`
	PermissionTreatedAsInstruction bool `json:"permissionTreatedAsInstruction"`
	MissingBodyReturnsHold         bool `json:"missingBodyReturnsHold"`
	MissingBodyAutoBuilt           bool `json:"missingBodyAutoBuilt"`
	MissingBodyInvented            bool `json:"missingBodyInvented"`
	ExecutionObserved              bool `json:"executionObserved"`
	InstallAuthorityGranted        bool `json:"installAuthorityGranted"`
	PromotionAuthorityGranted      bool `json:"promotionAuthorityGranted"`
	CanonAuthorityGranted          bool `json:"canonAuthorityGranted"`
	WaldoOwnsExecution             bool `json:"waldoOwnsExecution"`
}

type ExecutionPermitV019Contract struct {
	Schema                  string                           `json:"schema"`
	Status                  string                           `json:"status"`
	Challenge               string                           `json:"challenge"`
	SourceV018Receipt       string                           `json:"sourceV018Receipt"`
	SourceV018Head          string                           `json:"sourceV018Head"`
	CandidateSetDigest      string                           `json:"candidateSetDigest"`
	SelectedCandidateID     string                           `json:"selectedCandidateId"`
	SelectedCandidateDigest string                           `json:"selectedCandidateDigest"`
	ProbeBoundary           ExecutionPermitV019Boundary      `json:"probeBoundary"`
	ExecutionPermit         ExecutionPermitV019Permit        `json:"executionPermit"`
	ExecutionRequest        ExecutionPermitV019Request       `json:"executionRequest"`
	PermitValidation        string                           `json:"permitValidation"`
	RequestValidation       string                           `json:"requestValidation"`
	OverallDecision         string                           `json:"overallDecision"`
	OverallReason           string                           `json:"overallReason"`
	RefusalCases            []ExecutionPermitV019RefusalCase `json:"refusalCases"`
	DuplicateCase           ExecutionPermitV019DuplicateCase `json:"duplicateCase"`
	Truth                   ExecutionPermitV019Truth         `json:"truth"`
	BuildStarted            bool                             `json:"buildStarted"`
	Installed               bool                             `json:"installed"`
	Promoted                bool                             `json:"promoted"`
	CanonChanged            bool                             `json:"canonChanged"`
	ReceiptDigest           string                           `json:"receiptDigest"`
}

type ExecutionPermitV019Witness struct {
	Schema                  string   `json:"schema"`
	SourceReceipt           string   `json:"sourceReceipt"`
	SourceHead              string   `json:"sourceHead"`
	CandidateSetDigest      string   `json:"candidateSetDigest"`
	SelectedCandidateID     string   `json:"selectedCandidateId"`
	PermitID                string   `json:"permitId"`
	PermitValidation        string   `json:"permitValidation"`
	RequestValidation       string   `json:"requestValidation"`
	OverallDecision         string   `json:"overallDecision"`
	OverallReason           string   `json:"overallReason"`
	PreservedRefusalCaseIDs []string `json:"preservedRefusalCaseIds"`
	WitnessDigest           string   `json:"witnessDigest,omitempty"`
}
