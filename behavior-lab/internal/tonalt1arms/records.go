package tonalt1arms

// WorkflowRecord is the frozen workflow-level raw execution record (task
// §17): one per (workflow, arm) -- 180 total across a complete A+B+C run
// over the 60 frozen workflows.
type WorkflowRecord struct {
	RunID            string
	WorkflowID       string
	Family           string // Shape
	Arm              string // "A" | "B" | "C"
	TerminalStatus   string
	TerminalOutput   float64
	SemanticCorrect  bool
	ExactCorrect     bool
	ContractStatus   string
	TotalParrotCalls int
	LatencyMS        int64
}

// NodeCallRecord is the frozen node/model-call-level raw execution record
// (task §17): one per DAG node execution, whether or not it actually
// reached the network (a BLOCKED_BY_DEPENDENCY node still gets a record).
type NodeCallRecord struct {
	RunID           string
	WorkflowID      string
	Arm             string
	NodeID          string
	Capability      string
	Operation       string
	ExecutorID      string
	Model           string
	RequestIndex    int
	InputArtifact   string
	InputHash       string
	RawOutput       string
	ParsedOutput    string
	TransportStatus string
	SchemaStatus    string
	ContractStatus  string
	LatencyMS       int64
}

// RunAccounting is the frozen call-accounting record (task §17/G): kept
// separately from any single ambiguous "model_calls" counter.
// PlannedModelCallSlots is the STATIC count of generative node slots in the
// full A+B+C sweep (696 for the frozen 60-workflow primary run) --
// independent of what actually happened at runtime. HTTPRequestAttempts is
// the DYNAMIC count of slots that actually attempted a transport call; on
// an all-success run these are equal, but fail-closed dependency
// propagation can legitimately make HTTPRequestAttempts smaller (task
// correction G: never require them to be equal unconditionally).
type RunAccounting struct {
	PlannedModelCallSlots int
	HTTPRequestAttempts   int
	ValidCompletions      int
	TransportFailures     int
	SchemaFailures        int
	ModelContractFailures int
	BlockedByDependency   int
}
