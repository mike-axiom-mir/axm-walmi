package axmmirror

const (
	SelectionReceiptV018Schema             = "axm.capability-selection-receipt-contract/v0.18"
	SelectionReceiptV018Status             = "CONTRACT_PROBE_ONLY"
	SelectionReceiptV018Challenge          = "SELECTION_DOES_NOT_GRANT_EXECUTION"
	SelectionReceiptV018SourceReceipt      = "sha256:62d12d9ef02d3527878ba5e1c9baee6e79dc973d3617caf03796170fc5787dbe"
	SelectionReceiptV018SourceHead         = "6b5071cf7a632291e5d44c6d80909d0ae73ee937"
	SelectionReceiptV018CandidateSetDigest = "sha256:d396b84ddaa15f230273e96cb7b6c35dfece8d4d81ffbb46a127cccb2e0b8aa0"
	SelectionReceiptV018DirectDigest       = "sha256:8ed1d23b0c7020fe1f3ee2427c2984c6132c46e49f86b0183828c4c1dc775241"
	SelectionReceiptV018ProjectedDigest    = "sha256:721ba695c6d8243ea8e24c5c5a52414c7ef6bf456ca5ea67975069a80b74fdf9"
)

type SelectionReceiptV018Candidate struct {
	ID     string `json:"id"`
	Digest string `json:"digest"`
}

type SelectionReceiptV018Receipt struct {
	ReceiptID                string   `json:"receiptId"`
	IssuerClass              string   `json:"issuerClass"`
	Authenticity             string   `json:"authenticity"`
	SourceCandidateSetDigest string   `json:"sourceCandidateSetDigest"`
	SelectedCandidateID      string   `json:"selectedCandidateId"`
	SelectedCandidateDigest  string   `json:"selectedCandidateDigest"`
	Scope                    []string `json:"scope"`
	GrantsExecution          bool     `json:"grantsExecution"`
	GrantsInstall            bool     `json:"grantsInstall"`
	GrantsPromotion          bool     `json:"grantsPromotion"`
	GrantsCanon              bool     `json:"grantsCanon"`
}

type SelectionReceiptV018ConflictCase struct {
	FirstReceiptID  string `json:"firstReceiptId"`
	SecondReceiptID string `json:"secondReceiptId"`
	ExpectedOutcome string `json:"expectedOutcome"`
	ExpectedReason  string `json:"expectedReason"`
	LastWriterWins  bool   `json:"lastWriterWins"`
}

type SelectionReceiptV018RefusalCase struct {
	CaseID          string `json:"caseId"`
	ExpectedOutcome string `json:"expectedOutcome"`
	ExpectedReason  string `json:"expectedReason"`
}

type SelectionReceiptV018Contract struct {
	Schema                         string                            `json:"schema"`
	Status                         string                            `json:"status"`
	Challenge                      string                            `json:"challenge"`
	SourceV017Receipt              string                            `json:"sourceV017Receipt"`
	SourceV017Head                 string                            `json:"sourceV017Head"`
	CandidateSetDigest             string                            `json:"candidateSetDigest"`
	FrozenCandidates               []SelectionReceiptV018Candidate   `json:"frozenCandidates"`
	SelectionReceipt               SelectionReceiptV018Receipt       `json:"selectionReceipt"`
	SelectionValidation            string                            `json:"selectionValidation"`
	SelectedProposalState          string                            `json:"selectedProposalState"`
	OverallDecision                string                            `json:"overallDecision"`
	OverallReason                  string                            `json:"overallReason"`
	ExecutionAuthority             string                            `json:"executionAuthority"`
	ConflictReceipt                SelectionReceiptV018Receipt       `json:"conflictReceipt"`
	ConflictCase                   SelectionReceiptV018ConflictCase  `json:"conflictCase"`
	RefusalCases                   []SelectionReceiptV018RefusalCase `json:"refusalCases"`
	RealGovernanceDecisionObserved bool                              `json:"realGovernanceDecisionObserved"`
	LiveAIProviderCalled           bool                              `json:"liveAiProviderCalled"`
	WaldoIssuedSelection           bool                              `json:"waldoIssuedSelection"`
	BuildStarted                   bool                              `json:"buildStarted"`
	Installed                      bool                              `json:"installed"`
	Promoted                       bool                              `json:"promoted"`
	CanonChanged                   bool                              `json:"canonChanged"`
	ReceiptDigest                  string                            `json:"receiptDigest"`
}

type SelectionReceiptV018Witness struct {
	Schema                 string   `json:"schema"`
	SourceReceipt          string   `json:"sourceReceipt"`
	SourceHead             string   `json:"sourceHead"`
	CandidateSetDigest     string   `json:"candidateSetDigest"`
	SelectedCandidateID    string   `json:"selectedCandidateId"`
	SelectionValidation    string   `json:"selectionValidation"`
	OverallDecision        string   `json:"overallDecision"`
	OverallReason          string   `json:"overallReason"`
	ExecutionAuthority     string   `json:"executionAuthority"`
	PreservedRefusalCaseID []string `json:"preservedRefusalCaseIds"`
	WitnessDigest          string   `json:"witnessDigest,omitempty"`
}
