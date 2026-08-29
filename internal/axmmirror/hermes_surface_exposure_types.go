package axmmirror

const (
	HermesExposureV021Schema        = "axm.hermes-surface-capability-exposure-contract/v0.21"
	HermesExposureV021Status        = "CONTRACT_PROBE_ONLY"
	HermesExposureV021Challenge     = "SURFACE_EXPOSURE_IS_NOT_AUTHORITY"
	HermesExposureV021WitnessSchema = "axm.waldo-witness.hermes-surface-capability-exposure/v0.21"

	HermesExposureV021SourceV020Receipt = "sha256:edb16e8f804b6cace4e59eaf7f66344ef31648e52b028129eae373224e3645c2"
	HermesExposureV021SourceV020Head    = "b3d23a8262c96a3ba8a6e32b80b3e0b25cb366b0"
	HermesExposureV021HermesRepo        = "NousResearch/hermes-agent"
	HermesExposureV021LocalDonorCommit  = "739bc555b1932e66c169b20edec3a48368e2dd3f"
	HermesExposureV021ReviewedCommit    = "2ebb1cb41400660ccc3712157c18b3d39f278ab6"
)

type HermesExposureV021DonorFile struct {
	Path               string `json:"path"`
	BlobAtLocalPin     string `json:"blobAtLocalPin"`
	BlobAtReviewedHead string `json:"blobAtReviewedHead"`
	Unchanged          bool   `json:"unchanged"`
}

type HermesExposureV021Source struct {
	SourceV020Receipt string                        `json:"sourceV020Receipt"`
	SourceV020Head    string                        `json:"sourceV020Head"`
	HermesRepo        string                        `json:"hermesRepo"`
	LocalDonorCommit  string                        `json:"localDonorCommit"`
	ReviewedCommit    string                        `json:"reviewedCommit"`
	DonorFiles        []HermesExposureV021DonorFile `json:"donorFiles"`
}

type HermesExposureV021Boundary struct {
	HermesRuntimeImported           bool `json:"hermesRuntimeImported"`
	HermesRuntimeExecuted           bool `json:"hermesRuntimeExecuted"`
	LocalCapabilityFabricExecuted   bool `json:"localCapabilityFabricExecuted"`
	LiveAIProviderCalled            bool `json:"liveAiProviderCalled"`
	DonorBlobContinuityVerified     bool `json:"donorBlobContinuityVerified"`
	ContextCostsAreFixtureEstimates bool `json:"contextCostsAreFixtureEstimates"`
	OutputIsContractProbe           bool `json:"outputIsContractProbe"`
}

type HermesExposureV021SurfaceProfile struct {
	ProfileID           string   `json:"profileId"`
	Surface             string   `json:"surface"`
	SourceTrust         string   `json:"sourceTrust"`
	ExposedToolsets     []string `json:"exposedToolsets"`
	ExposedTools        []string `json:"exposedTools"`
	SchemaCostTokens    int      `json:"schemaCostTokens"`
	ContextBudgetTokens int      `json:"contextBudgetTokens"`
	CostEvidenceClass   string   `json:"costEvidenceClass"`
}

type HermesExposureV021Case struct {
	CaseID              string `json:"caseId"`
	SurfaceProfileID    string `json:"surfaceProfileId"`
	ToolName            string `json:"toolName"`
	CapabilityDeclared  bool   `json:"capabilityDeclared"`
	CapabilityInstalled bool   `json:"capabilityInstalled"`
	CapabilityExposed   bool   `json:"capabilityExposed"`
	AuthorityPresent    bool   `json:"authorityPresent"`
	ActionRequested     bool   `json:"actionRequested"`
	ExpectedOutcome     string `json:"expectedOutcome"`
	ExpectedReason      string `json:"expectedReason"`
}

type HermesExposureV021RefusalCase struct {
	CaseID          string `json:"caseId"`
	ExpectedOutcome string `json:"expectedOutcome"`
	ExpectedReason  string `json:"expectedReason"`
}

type HermesExposureV021Truth struct {
	InstalledDoesNotImplyExposed       bool `json:"installedDoesNotImplyExposed"`
	ExposedDoesNotImplyAuthorized      bool `json:"exposedDoesNotImplyAuthorized"`
	AuthorizedDoesNotImplyExecuted     bool `json:"authorizedDoesNotImplyExecuted"`
	WebhookSafeSetPreserved            bool `json:"webhookSafeSetPreserved"`
	DesktopOnlyToolsRemainSurfaceBound bool `json:"desktopOnlyToolsRemainSurfaceBound"`
	ContextBudgetOverflowReturnsHold   bool `json:"contextBudgetOverflowReturnsHold"`
	ContextSilentlyPruned              bool `json:"contextSilentlyPruned"`
	SurfaceExposureGrantsAuthority     bool `json:"surfaceExposureGrantsAuthority"`
	WaldoOwnsSurfaceExposure           bool `json:"waldoOwnsSurfaceExposure"`
	HistoricalV020ReceiptRewritten     bool `json:"historicalV020ReceiptRewritten"`
}

type HermesExposureV021Contract struct {
	Schema        string                             `json:"schema"`
	Status        string                             `json:"status"`
	Challenge     string                             `json:"challenge"`
	Source        HermesExposureV021Source           `json:"source"`
	Boundary      HermesExposureV021Boundary         `json:"boundary"`
	Profiles      []HermesExposureV021SurfaceProfile `json:"profiles"`
	Cases         []HermesExposureV021Case           `json:"cases"`
	RefusalCases  []HermesExposureV021RefusalCase    `json:"refusalCases"`
	Truth         HermesExposureV021Truth            `json:"truth"`
	Authority     string                             `json:"authority"`
	ReceiptDigest string                             `json:"receiptDigest"`
}

type HermesExposureV021Witness struct {
	Schema                 string   `json:"schema"`
	SourceV020Receipt      string   `json:"sourceV020Receipt"`
	SourceV020Head         string   `json:"sourceV020Head"`
	HermesLocalDonorCommit string   `json:"hermesLocalDonorCommit"`
	HermesReviewedCommit   string   `json:"hermesReviewedCommit"`
	DonorFileCount         int      `json:"donorFileCount"`
	SurfaceProfiles        []string `json:"surfaceProfiles"`
	EvaluatedCases         int      `json:"evaluatedCases"`
	RefusalCases           int      `json:"refusalCases"`
	SourceReceiptSHA256    string   `json:"sourceReceiptSha256"`
	Authority              string   `json:"authority"`
	WitnessDigest          string   `json:"witnessDigest,omitempty"`
}
