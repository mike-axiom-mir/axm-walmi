package axmmirror

const (
	ExecutionRevocationV020Schema        = "axm.capability-execution-revocation-contract/v0.20"
	ExecutionRevocationV020Status        = "CONTRACT_PROBE_ONLY"
	ExecutionRevocationV020Challenge     = "REVOCATION_STOPS_FUTURE_ACTION_WITHOUT_REWRITING_HISTORY"
	ExecutionRevocationV020SourceReceipt = "sha256:88885cdb000a567746030914f170c161444db87ec6116ba8ce4b6a58da6257dc"
	ExecutionRevocationV020SourceHead    = "ecf1be23166e8445677aaede78cfb20956ad7a7e"
	ExecutionRevocationV020SourceWitness = "960c7eab77597a87256da843663e88fe22d1290fcc1e42dbff1b1da14bfee7d3"
	ExecutionRevocationV020PermitID      = "external-disposable-execution-permit-direct-read"
	ExecutionRevocationV020CandidateID   = "direct-read-chain"
	ExecutionRevocationV020CandidateSHA  = "sha256:8ed1d23b0c7020fe1f3ee2427c2984c6132c46e49f86b0183828c4c1dc775241"
)

type ExecutionRevocationV020Boundary struct {
	LocalCapabilityFabricImplementationIncluded bool `json:"localCapabilityFabricImplementationIncluded"`
	ExecutableCandidateBodyIncluded             bool `json:"executableCandidateBodyIncluded"`
	ProbeExecutesCandidate                      bool `json:"probeExecutesCandidate"`
	LiveAIProviderCalled                        bool `json:"liveAiProviderCalled"`
	RealRevocationObserved                      bool `json:"realRevocationObserved"`
}

type ExecutionRevocationV020LedgerEntry struct {
	Sequence           int    `json:"sequence"`
	EventID            string `json:"eventId"`
	EventType          string `json:"eventType"`
	PermitID           string `json:"permitId"`
	Outcome            string `json:"outcome"`
	Reason             string `json:"reason"`
	ReferencesEventID  string `json:"referencesEventId,omitempty"`
	RewritesPriorEntry bool   `json:"rewritesPriorEntry"`
}

type ExecutionRevocationV020Receipt struct {
	ReceiptID          string   `json:"receiptId"`
	IssuerClass        string   `json:"issuerClass"`
	Authenticity       string   `json:"authenticity"`
	SourceV019Receipt  string   `json:"sourceV019Receipt"`
	TargetPermitID     string   `json:"targetPermitId"`
	TargetCandidateID  string   `json:"targetCandidateId"`
	TargetCandidateSHA string   `json:"targetCandidateDigest"`
	Scope              []string `json:"scope"`
	EffectiveSequence  int      `json:"effectiveSequence"`
	WaldoIssued        bool     `json:"waldoIssued"`
	DeletePriorPermit  bool     `json:"deletePriorPermit"`
	GrantsExecution    bool     `json:"grantsExecution"`
	GrantsInstall      bool     `json:"grantsInstall"`
	GrantsPromotion    bool     `json:"grantsPromotion"`
	GrantsCanon        bool     `json:"grantsCanon"`
}

type ExecutionRevocationV020Scenario struct {
	ScenarioID       string `json:"scenarioId"`
	RequestSequence  int    `json:"requestSequence"`
	BodyPresent      bool   `json:"bodyPresent"`
	ExpectedDecision string `json:"expectedDecision"`
	ExpectedReason   string `json:"expectedReason"`
}

type ExecutionRevocationV020RefusalCase struct {
	CaseID          string `json:"caseId"`
	ExpectedOutcome string `json:"expectedOutcome"`
	ExpectedReason  string `json:"expectedReason"`
}

type ExecutionRevocationV020DuplicateCase struct {
	DuplicateReceiptID string `json:"duplicateReceiptId"`
	ExpectedOutcome    string `json:"expectedOutcome"`
	ExtraAuthority     bool   `json:"extraAuthority"`
	HistoryRewritten   bool   `json:"historyRewritten"`
}

type ExecutionRevocationV020Truth struct {
	PriorPermitAcceptancePreserved bool `json:"priorPermitAcceptancePreserved"`
	RevocationAppendOnly           bool `json:"revocationAppendOnly"`
	PriorPermitDeleted             bool `json:"priorPermitDeleted"`
	PastAuthorizationRewritten     bool `json:"pastAuthorizationRewritten"`
	FutureActionBlocked            bool `json:"futureActionBlocked"`
	ExecutionObserved              bool `json:"executionObserved"`
	InstallAuthorityGranted        bool `json:"installAuthorityGranted"`
	PromotionAuthorityGranted      bool `json:"promotionAuthorityGranted"`
	CanonAuthorityGranted          bool `json:"canonAuthorityGranted"`
	WaldoOwnsRevocation            bool `json:"waldoOwnsRevocation"`
}

type ExecutionRevocationV020Contract struct {
	Schema                  string                               `json:"schema"`
	Status                  string                               `json:"status"`
	Challenge               string                               `json:"challenge"`
	SourceV019Receipt       string                               `json:"sourceV019Receipt"`
	SourceV019Head          string                               `json:"sourceV019Head"`
	SourceV019Witness       string                               `json:"sourceV019Witness"`
	PermitID                string                               `json:"permitId"`
	SelectedCandidateID     string                               `json:"selectedCandidateId"`
	SelectedCandidateDigest string                               `json:"selectedCandidateDigest"`
	ProbeBoundary           ExecutionRevocationV020Boundary      `json:"probeBoundary"`
	Ledger                  []ExecutionRevocationV020LedgerEntry `json:"ledger"`
	Revocation              ExecutionRevocationV020Receipt       `json:"revocation"`
	Scenarios               []ExecutionRevocationV020Scenario    `json:"scenarios"`
	RefusalCases            []ExecutionRevocationV020RefusalCase `json:"refusalCases"`
	DuplicateCase           ExecutionRevocationV020DuplicateCase `json:"duplicateCase"`
	CurrentState            string                               `json:"currentState"`
	Truth                   ExecutionRevocationV020Truth         `json:"truth"`
	Installed               bool                                 `json:"installed"`
	Promoted                bool                                 `json:"promoted"`
	CanonChanged            bool                                 `json:"canonChanged"`
	ReceiptDigest           string                               `json:"receiptDigest"`
}

type ExecutionRevocationV020Witness struct {
	Schema                  string   `json:"schema"`
	SourceReceipt           string   `json:"sourceReceipt"`
	SourceHead              string   `json:"sourceHead"`
	PermitID                string   `json:"permitId"`
	RevocationReceiptID     string   `json:"revocationReceiptId"`
	LedgerEventIDs          []string `json:"ledgerEventIds"`
	CurrentState            string   `json:"currentState"`
	PostRevocationDecision  string   `json:"postRevocationDecision"`
	PostRevocationReason    string   `json:"postRevocationReason"`
	PreservedRefusalCaseIDs []string `json:"preservedRefusalCaseIds"`
	WitnessDigest           string   `json:"witnessDigest,omitempty"`
}
