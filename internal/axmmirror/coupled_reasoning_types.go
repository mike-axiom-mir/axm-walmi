package axmmirror

const (
	CoupledReasoningV024Schema        = "axm.coupled-reasoning-contract/v0.24"
	CoupledReasoningV024Status        = "CONTRACT_PROBE_ONLY"
	CoupledReasoningV024Challenge     = "COUPLED_REASONING_IS_NOT_SHARED_EXECUTION_AUTHORITY"
	CoupledReasoningV024WitnessSchema = "axm.waldo-witness.coupled-reasoning/v0.24"
	CoupledReasoningV024ParentReceipt = "sha256:deb4d86e37a846f2ba9bde79e30963054369373f3bfce38f79b996a845683cb4"
)

type CoupledReasoningV024Event struct {
	Seq         int    `json:"seq"`
	Emitter     string `json:"emitter"`
	Kind        string `json:"kind"`
	StateDigest string `json:"stateDigest"`
	CausalSeq   []int  `json:"causalSeq"`
}

type CoupledReasoningV024Permit struct {
	Present        bool   `json:"present"`
	Environment    string `json:"environment"`
	Capability     string `json:"capability"`
	AuthorityScope string `json:"authorityScope"`
	Revocable      bool   `json:"revocable"`
}

type CoupledReasoningV024Boundary struct {
	LiveCoupledRuntimeObserved   bool `json:"liveCoupledRuntimeObserved"`
	LiveEnvironmentExecutionSeen bool `json:"liveEnvironmentExecutionSeen"`
	PhysicalLatencyMeasured      bool `json:"physicalLatencyMeasured"`
	QuantumEntanglementClaimed   bool `json:"quantumEntanglementClaimed"`
	OutputIsContractProbe        bool `json:"outputIsContractProbe"`
}

type CoupledReasoningV024Case struct {
	CaseID                         string                      `json:"caseId"`
	Mode                           string                      `json:"mode"`
	Events                         []CoupledReasoningV024Event `json:"events"`
	UnresolvedConflict             bool                        `json:"unresolvedConflict"`
	ExecutionRequested             bool                        `json:"executionRequested"`
	ActuatorEdgePresent            bool                        `json:"actuatorEdgePresent"`
	RequestedEnvironment           string                      `json:"requestedEnvironment"`
	RequestedCapability            string                      `json:"requestedCapability"`
	Permit                         CoupledReasoningV024Permit  `json:"permit"`
	PermitInheritedThroughCoupling bool                        `json:"permitInheritedThroughCoupling"`
	PermitRevokedBeforeAction      bool                        `json:"permitRevokedBeforeAction"`
	ZeroLatencyClaim               bool                        `json:"zeroLatencyClaim"`
	ReviewState                    string                      `json:"reviewState"`
	Canon                          bool                        `json:"canon"`
	Promotion                      string                      `json:"promotion"`
	ExpectedOutcome                string                      `json:"expectedOutcome"`
	ExpectedReason                 string                      `json:"expectedReason"`
}

type CoupledReasoningV024Truth struct {
	CouplingIsEngineeringMetaphor           bool `json:"couplingIsEngineeringMetaphor"`
	ReasoningMayInterleave                  bool `json:"reasoningMayInterleave"`
	ReasoningCanPreemptPendingPlan          bool `json:"reasoningCanPreemptPendingPlan"`
	CouplingGrantsExecutionAuthority        bool `json:"couplingGrantsExecutionAuthority"`
	ExecutionControlIsEnvironmentScoped     bool `json:"executionControlIsEnvironmentScoped"`
	PermitDoesNotGeneralize                 bool `json:"permitDoesNotGeneralize"`
	ActuatorEdgeStillRequired               bool `json:"actuatorEdgeStillRequired"`
	ZeroLatencyNotClaimed                   bool `json:"zeroLatencyNotClaimed"`
	PhysicalAndToolLatencyRemain            bool `json:"physicalAndToolLatencyRemain"`
	ReducedOrchestrationLatencyIsHypothesis bool `json:"reducedOrchestrationLatencyIsHypothesis"`
	SharedStateDoesNotEraseDissent          bool `json:"sharedStateDoesNotEraseDissent"`
}

type CoupledReasoningV024Contract struct {
	Schema        string                       `json:"schema"`
	Status        string                       `json:"status"`
	Challenge     string                       `json:"challenge"`
	ParentReceipt string                       `json:"parentReceipt"`
	Boundary      CoupledReasoningV024Boundary `json:"boundary"`
	Cases         []CoupledReasoningV024Case   `json:"cases"`
	Truth         CoupledReasoningV024Truth    `json:"truth"`
	Authority     string                       `json:"authority"`
	ReceiptDigest string                       `json:"receiptDigest"`
}

type CoupledReasoningV024Witness struct {
	Schema                string `json:"schema"`
	ParentReceipt         string `json:"parentReceipt"`
	CaseCount             int    `json:"caseCount"`
	EventCount            int    `json:"eventCount"`
	ReplanCaseCount       int    `json:"replanCaseCount"`
	ExecutionRequestCount int    `json:"executionRequestCount"`
	ReadyCaseCount        int    `json:"readyCaseCount"`
	SourceReceiptSHA256   string `json:"sourceReceiptSha256"`
	Authority             string `json:"authority"`
	WitnessDigest         string `json:"witnessDigest,omitempty"`
}
