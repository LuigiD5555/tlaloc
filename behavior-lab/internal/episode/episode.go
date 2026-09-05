// Package episode defines the minimal Prototype Lab record: one complete
// real-task execution, reusable as experience by later prototype iterations.
//
// Episode is deliberately a common experimental projection rather than a
// replacement for experiment-specific raw records. T1 keeps WorkflowRecord,
// NodeCallRecord and RunAccounting as its scientific source of truth; Episode
// preserves the fields needed for cross-prototype comparison and learning.
package episode

// Schema is the persisted schema identifier for Episode. The v1 schema is
// additive: the original fields remain valid while newer producers can record
// richer provenance, raw model I/O and explicit transport accounting.
const Schema = "tlaloc.episode.v1"

// Episode is one complete execution: a task attempted, the steps taken to
// attempt it, and whether it succeeded. SourceExperiment marks provenance so
// imported data (for example T1) is never confused with native prototype-loop
// data.
type Episode struct {
	Schema           string `json:"schema"`
	EpisodeID        string `json:"episode_id"`
	SourceExperiment string `json:"source_experiment"`
	RunID            string `json:"run_id,omitempty"`
	PrototypeVersion string `json:"prototype_version,omitempty"`
	TaskID           string `json:"task_id"`
	Goal             string `json:"goal,omitempty"`
	Arm              string `json:"arm,omitempty"`
	Family           string `json:"family,omitempty"`

	Steps []Step `json:"steps"`

	Success          bool   `json:"success"`
	SemanticCorrect  bool   `json:"semantic_correct"`
	ExactCorrect     bool   `json:"exact_correct"`
	TerminalStatus   string `json:"terminal_status,omitempty"`
	ContractStatus   string `json:"contract_status,omitempty"`
	FailureRootCause string `json:"failure_root_cause,omitempty"`

	Cost Cost `json:"cost"`
}

// Step is one recorded action inside an episode. It intentionally preserves
// experiment-observable execution facts, including raw/parsed model output and
// transport/schema/contract status. It does not request or store private model
// reasoning.
type Step struct {
	RequestIndex         int                `json:"request_index"`
	NodeID               string             `json:"node_id"`
	SelectedCapability   string             `json:"selected_capability"`
	SelectedOperation    string             `json:"selected_operation,omitempty"`
	ExecutorID           string             `json:"executor_id,omitempty"`
	Model                string             `json:"model,omitempty"`
	Inputs               map[string]float64 `json:"inputs,omitempty"`
	Outputs              map[string]float64 `json:"outputs,omitempty"`
	InputArtifact        string             `json:"input_artifact,omitempty"`
	InputHash            string             `json:"input_hash,omitempty"`
	RawOutput            string             `json:"raw_output,omitempty"`
	ParsedOutput         string             `json:"parsed_output,omitempty"`
	TransportStatus      string             `json:"transport_status,omitempty"`
	SchemaStatus         string             `json:"schema_status,omitempty"`
	ContractStatus       string             `json:"contract_status,omitempty"`
	Status               string             `json:"status"`
	ModelCalls           int                `json:"model_calls"`
	LatencyMS            int64              `json:"latency_ms"`
	Error                string             `json:"error,omitempty"`
}

// Cost is the episode-level execution-accounting rollup.
//
// ModelCalls is the number of model calls reported by the source experiment;
// its exact transport semantics remain source-specific. T1's historical
// adapter counts completed transports here, while a TONAL runtime adapter can
// report its native generative-call count. Cross-prototype analysis that needs
// transport semantics must use the explicit counters below, and producers must
// leave counters they cannot observe at zero rather than inventing them.
// HTTPRequestAttempts counts both successful and failed transport attempts
// only when the source experiment actually observes that boundary.
type Cost struct {
	ModelCalls            int   `json:"model_calls"`
	HTTPRequestAttempts   int   `json:"http_request_attempts"`
	ValidCompletions      int   `json:"valid_completions"`
	TransportFailures     int   `json:"transport_failures"`
	SchemaFailures        int   `json:"schema_failures"`
	ModelContractFailures int   `json:"model_contract_failures"`
	BlockedByDependency   int   `json:"blocked_by_dependency"`
	LatencyMS             int64 `json:"latency_ms"`
}
