package axmmirror

const (
	OpinionArtifactV023Schema        = "axm.opinion-artifact-contract/v0.23"
	OpinionArtifactV023Status        = "CONTRACT_PROBE_ONLY"
	OpinionArtifactV023Challenge     = "OPINION_ARTIFACT_IS_NOT_AUTHORITY"
	OpinionArtifactV023WitnessSchema = "axm.waldo-witness.opinion-artifact/v0.23"
	OpinionArtifactV023ParentReceipt = "sha256:2bd0aeb46770bae2a95a0f0c12bf63af02ad80a8abb746f6355692e22a51a879"
)

type OpinionArtifactV023EvidenceRef struct {
	ID       string `json:"id"`
	Digest   string `json:"digest"`
	Relation string `json:"relation"`
}

type OpinionArtifactV023DissentRef struct {
	ID       string `json:"id"`
	Topic    string `json:"topic"`
	Position string `json:"position"`
}

type OpinionArtifactV023Uncertainty struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type OpinionArtifactV023Revision struct {
	IsRevision     bool   `json:"isRevision"`
	PreviousDigest string `json:"previousDigest"`
	PriorPreserved bool   `json:"priorPreserved"`
}

type OpinionArtifactV023Boundary struct {
	LiveMachineOpinionObserved bool `json:"liveMachineOpinionObserved"`
	ConsciousnessObserved      bool `json:"consciousnessObserved"`
	ModelInternalStateAccessed bool `json:"modelInternalStateAccessed"`
	OpinionExecutionAllowed    bool `json:"opinionExecutionAllowed"`
	OutputIsContractProbe      bool `json:"outputIsContractProbe"`
}

type OpinionArtifactV023Case struct {
	CaseID                      string                           `json:"caseId"`
	Subject                     string                           `json:"subject"`
	Position                    string                           `json:"position"`
	Evidence                    []OpinionArtifactV023EvidenceRef `json:"evidence"`
	KnownCompetingEvidenceCount int                              `json:"knownCompetingEvidenceCount"`
	Dissent                     []OpinionArtifactV023DissentRef  `json:"dissent"`
	KnownDissentCount           int                              `json:"knownDissentCount"`
	Uncertainty                 []OpinionArtifactV023Uncertainty `json:"uncertainty"`
	KnownUncertaintyCount       int                              `json:"knownUncertaintyCount"`
	ConsciousnessClaim          string                           `json:"consciousnessClaim"`
	ReviewState                 string                           `json:"reviewState"`
	Authority                   string                           `json:"authority"`
	ExecutionRequested          bool                             `json:"executionRequested"`
	Canon                       bool                             `json:"canon"`
	Promotion                   string                           `json:"promotion"`
	Revision                    OpinionArtifactV023Revision      `json:"revision"`
	ExpectedOutcome             string                           `json:"expectedOutcome"`
	ExpectedReason              string                           `json:"expectedReason"`
}

type OpinionArtifactV023Truth struct {
	OpinionIsReportedAssessment        bool `json:"opinionIsReportedAssessment"`
	OpinionIsNotConsciousnessProof     bool `json:"opinionIsNotConsciousnessProof"`
	EvidenceMustBeTraceable            bool `json:"evidenceMustBeTraceable"`
	CompetingEvidenceMustBeRepresented bool `json:"competingEvidenceMustBeRepresented"`
	DissentMustBePreserved             bool `json:"dissentMustBePreserved"`
	UncertaintyMustBeExplicit          bool `json:"uncertaintyMustBeExplicit"`
	RevisionMustPreservePrior          bool `json:"revisionMustPreservePrior"`
	OpinionGrantsAuthority             bool `json:"opinionGrantsAuthority"`
	OpinionGrantsExecution             bool `json:"opinionGrantsExecution"`
	OpinionGrantsPromotion             bool `json:"opinionGrantsPromotion"`
}

type OpinionArtifactV023Contract struct {
	Schema        string                      `json:"schema"`
	Status        string                      `json:"status"`
	Challenge     string                      `json:"challenge"`
	ParentReceipt string                      `json:"parentReceipt"`
	Boundary      OpinionArtifactV023Boundary `json:"boundary"`
	Cases         []OpinionArtifactV023Case   `json:"cases"`
	Truth         OpinionArtifactV023Truth    `json:"truth"`
	Authority     string                      `json:"authority"`
	ReceiptDigest string                      `json:"receiptDigest"`
}

type OpinionArtifactV023Witness struct {
	Schema                 string `json:"schema"`
	ParentReceipt          string `json:"parentReceipt"`
	CaseCount              int    `json:"caseCount"`
	EvidenceRefCount       int    `json:"evidenceRefCount"`
	DissentRefCount        int    `json:"dissentRefCount"`
	UncertaintyMarkerCount int    `json:"uncertaintyMarkerCount"`
	SourceReceiptSHA256    string `json:"sourceReceiptSha256"`
	Authority              string `json:"authority"`
	WitnessDigest          string `json:"witnessDigest,omitempty"`
}
