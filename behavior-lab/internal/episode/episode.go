// Package episode defines the minimal Prototype Lab v0.1 record: one
// complete real-task execution, reusable as experience by later prototype
// iterations. This is deliberately the smallest schema that can hold a T1
// arm/workflow execution -- it does not yet carry controller/goal fields
// that only apply once a T2 controller exists (see FromT1Workflow's doc
// comment for what is intentionally left null for T1).
package episode

// Schema is the persisted schema identifier for Episode.
const Schema = "tlaloc.episode.v1"

// Episode is one complete execution: a task attempted, the steps taken to
// attempt it, and whether it succeeded. SourceExperiment marks provenance
// so imported data (e.g. T1) is never confused with native prototype-loop
// data.
type Episode struct {
	Schema           string `json:"schema"`
	EpisodeID        string `json:"episode_id"`
	SourceExperiment string `json:"source_experiment"`
	PrototypeVersion string `json:"prototype_version,omitempty"`
	TaskID           string `json:"task_id"`
	Goal             string `json:"goal,omitempty"`

	Steps []Step `json:"steps"`

	Success         bool   `json:"success"`
	FailureRootCause string `json:"failure_root_cause,omitempty"`

	Cost Cost `json:"cost"`
}

// Step is one recorded action inside an episode. For a fixed-DAG source
// (T1's arms A/B/C) there is no controller decision to record -- the DAG
// itself is the decision -- so ControllerDecision stays empty and
// SelectedCapability/SelectedOperation carry the equivalent information.
type Step struct {
	NodeID               string             `json:"node_id"`
	SelectedCapability   string             `json:"selected_capability"`
	SelectedOperation    string             `json:"selected_operation,omitempty"`
	ExecutorID           string             `json:"executor_id,omitempty"`
	Inputs               map[string]float64 `json:"inputs,omitempty"`
	Outputs              map[string]float64 `json:"outputs,omitempty"`
	Status               string             `json:"status"`
	ModelCalls           int                `json:"model_calls"`
	LatencyMS            int64              `json:"latency_ms"`
	Error                string             `json:"error,omitempty"`
}

// Cost is the episode-level rollup of its steps' cost fields.
type Cost struct {
	ModelCalls int   `json:"model_calls"`
	LatencyMS  int64 `json:"latency_ms"`
}
