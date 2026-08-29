package axmmirror

const (
	HermesRunCapsuleV022Schema        = "axm.hermes-run-capsule-evidence-contract/v0.22"
	HermesRunCapsuleV022Status        = "CONTRACT_PROBE_ONLY"
	HermesRunCapsuleV022Challenge     = "RUN_CAPSULE_EVIDENCE_IS_NOT_PROMOTION"
	HermesRunCapsuleV022WitnessSchema = "axm.waldo-witness.hermes-run-capsule-evidence/v0.22"
	HermesRunCapsuleV022PlatformRepo  = "mike-axiom-mir/axm-collaboration-platform"
	HermesRunCapsuleV022PlatformPR    = 48
	HermesRunCapsuleV022PlatformHead  = "7290dacf4fe2bb9a502f6a339fc53ecdbe0d420b"
)

type HermesRunCapsuleV022DonorFile struct {
	Path string `json:"path"`
	Blob string `json:"blob"`
}

type HermesRunCapsuleV022Source struct {
	PlatformRepo          string                          `json:"platformRepo"`
	PlatformPR            int                             `json:"platformPr"`
	PlatformHead          string                          `json:"platformHead"`
	RuntimeGateRunID      int64                           `json:"runtimeGateRunId"`
	RuntimeGateConclusion string                          `json:"runtimeGateConclusion"`
	DonorFiles            []HermesRunCapsuleV022DonorFile `json:"donorFiles"`
}

type HermesRunCapsuleV022Boundary struct {
	PlatformHermesRuntimeCopied          bool `json:"platformHermesRuntimeCopied"`
	PlatformHermesRuntimeExecutedByWaldo bool `json:"platformHermesRuntimeExecutedByWaldo"`
	LiveHermesRunObserved                bool `json:"liveHermesRunObserved"`
	LiveAIProviderCalled                 bool `json:"liveAiProviderCalled"`
	OutputIsContractProbe                bool `json:"outputIsContractProbe"`
}

type HermesRunCapsuleV022Case struct {
	CaseID                           string `json:"caseId"`
	AuthorizedToolCalls              int    `json:"authorizedToolCalls"`
	CompletionReceipts               int    `json:"completionReceipts"`
	ProviderPolicyMismatch           bool   `json:"providerPolicyMismatch"`
	PolicyChangedDuringRun           bool   `json:"policyChangedDuringRun"`
	ProfileChangedDuringRun          bool   `json:"profileChangedDuringRun"`
	SourceVerifiedBefore             bool   `json:"sourceVerifiedBefore"`
	SourceVerifiedAfter              bool   `json:"sourceVerifiedAfter"`
	ConsentRevokedAfterAuthorization bool   `json:"consentRevokedAfterAuthorization"`
	CompletionEvidenceRecorded       bool   `json:"completionEvidenceRecorded"`
	RawPromptStored                  bool   `json:"rawPromptStored"`
	RawResponseStored                bool   `json:"rawResponseStored"`
	RawToolArgumentsStored           bool   `json:"rawToolArgumentsStored"`
	RawToolResultsStored             bool   `json:"rawToolResultsStored"`
	Canon                            bool   `json:"canon"`
	Promotion                        string `json:"promotion"`
	ExpectedOutcome                  string `json:"expectedOutcome"`
	ExpectedReason                   string `json:"expectedReason"`
}

type HermesRunCapsuleV022Truth struct {
	ReceiptRecordingIndependentOfCurrentConsent bool `json:"receiptRecordingIndependentOfCurrentConsent"`
	RevocationDoesNotAuthorizeNewAction         bool `json:"revocationDoesNotAuthorizeNewAction"`
	EvidenceGapIsExplicit                       bool `json:"evidenceGapIsExplicit"`
	ProviderMismatchIsExplicit                  bool `json:"providerMismatchIsExplicit"`
	PolicyDriftIsExplicit                       bool `json:"policyDriftIsExplicit"`
	RawContentExcluded                          bool `json:"rawContentExcluded"`
	ReturnPacketCandidateOnly                   bool `json:"returnPacketCandidateOnly"`
	ReturnPacketGrantsPromotion                 bool `json:"returnPacketGrantsPromotion"`
	WaldoOwnsHermesRuntime                      bool `json:"waldoOwnsHermesRuntime"`
}

type HermesRunCapsuleV022Contract struct {
	Schema        string                       `json:"schema"`
	Status        string                       `json:"status"`
	Challenge     string                       `json:"challenge"`
	Source        HermesRunCapsuleV022Source   `json:"source"`
	Boundary      HermesRunCapsuleV022Boundary `json:"boundary"`
	Cases         []HermesRunCapsuleV022Case   `json:"cases"`
	Truth         HermesRunCapsuleV022Truth    `json:"truth"`
	Authority     string                       `json:"authority"`
	ReceiptDigest string                       `json:"receiptDigest"`
}

type HermesRunCapsuleV022Witness struct {
	Schema              string `json:"schema"`
	PlatformHead        string `json:"platformHead"`
	RuntimeGateRunID    int64  `json:"runtimeGateRunId"`
	DonorFileCount      int    `json:"donorFileCount"`
	CaseCount           int    `json:"caseCount"`
	SourceReceiptSHA256 string `json:"sourceReceiptSha256"`
	Authority           string `json:"authority"`
	WitnessDigest       string `json:"witnessDigest,omitempty"`
}
