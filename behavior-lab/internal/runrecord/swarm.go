package runrecord

// SwarmSchema is the persistable swarm-run record schema. It is distinct
// from the component-level Record (Schema "tlaloc.run-record.v1"): a swarm
// run is a DAG of workers, not one model call, so its cost picture is
// per-node plus a set of run-level derived metrics.
const SwarmSchema = "tlaloc.swarm-run-record.v2"

// SwarmRecord is the durable form of a finished swarm run: identity, host,
// the plan/topology hashes that pin what actually ran, and the accounting
// needed to compare a small-specialist swarm against a single larger call.
type SwarmRecord struct {
	Schema     string          `json:"schema"`
	RunID      string          `json:"run_id"`
	PlanID     string          `json:"plan_id"`
	TaskID     string          `json:"task_id"`
	Component  Component       `json:"component"`
	Host       Host            `json:"host"`
	StartedAt  string          `json:"started_at"`
	Succeeded  bool            `json:"succeeded"`
	Nodes      []SwarmNode     `json:"nodes"`
	Accounting SwarmAccounting `json:"accounting"`
}

// SwarmNode is one worker execution inside a swarm run, cost included.
type SwarmNode struct {
	NodeID        string `json:"node_id"`
	Capability    string `json:"capability"`
	WorkerID      string `json:"worker_id"`
	State         string `json:"state,omitempty"`
	DurationMS    int64  `json:"duration_ms"`
	QueueMS       int64  `json:"queue_ms,omitempty"`
	BytesIn       int64  `json:"bytes_in,omitempty"`
	BytesOut      int64  `json:"bytes_out,omitempty"`
	TokensIn      int    `json:"tokens_in,omitempty"`
	TokensOut     int    `json:"tokens_out,omitempty"`
	UpstreamCalls int    `json:"upstream_calls,omitempty"`
	ModelLoadMS   int64  `json:"model_load_ms,omitempty"`
	Retries       int    `json:"retries,omitempty"`
	Error         string `json:"error,omitempty"`
}

// SwarmAccounting mirrors internal/tlaloque.SwarmAccounting as a plain
// persistable copy (this package must not import the runtime). See that
// type for the honesty note on PeakRSSDeltaBytes being process-global.
type SwarmAccounting struct {
	WallMS                 int64   `json:"wall_ms"`
	TotalWorkMS            int64   `json:"total_work_ms"`
	CriticalPathMS         int64   `json:"critical_path_ms"`
	CoordinationOverheadMS int64   `json:"coordination_overhead_ms"`
	ParallelEfficiency     float64 `json:"parallel_efficiency"`
	TotalTokensIn          int     `json:"total_tokens_in,omitempty"`
	TotalTokensOut         int     `json:"total_tokens_out,omitempty"`
	TotalUpstreamCalls     int     `json:"total_upstream_calls,omitempty"`
	TotalBytesIn           int64   `json:"total_bytes_in,omitempty"`
	TotalBytesOut          int64   `json:"total_bytes_out,omitempty"`
	TotalQueueMS           int64   `json:"total_queue_ms,omitempty"`
	TotalRetries           int     `json:"total_retries,omitempty"`
	PeakRSSDeltaBytes      int64   `json:"peak_rss_delta_bytes,omitempty"`
	NodesExecuted          int     `json:"nodes_executed"`
	PeakParallel           int     `json:"peak_parallel"`
	PlanHash               string  `json:"plan_hash"`
	TopologyHash           string  `json:"topology_hash"`
}

// BaselineComparison answers the central swarm hypothesis question against
// a single-call baseline: is the swarm at least as good, cheaper, and
// faster? Quality is supplied by the caller (the swarm has no opinion on
// its own correctness); cost here is total generated+consumed tokens.
type BaselineComparison struct {
	SwarmQuality    float64 `json:"swarm_quality"`
	BaselineQuality float64 `json:"baseline_quality"`
	SwarmTokens     int     `json:"swarm_tokens"`
	BaselineTokens  int     `json:"baseline_tokens"`
	SwarmWallMS     int64   `json:"swarm_wall_ms"`
	BaselineWallMS  int64   `json:"baseline_wall_ms"`

	QualityAtLeastBaseline bool `json:"quality_at_least_baseline"`
	CheaperThanBaseline    bool `json:"cheaper_than_baseline"`
	FasterThanBaseline     bool `json:"faster_than_baseline"`
	SwarmWins              bool `json:"swarm_wins"`
}

// CompareToBaseline evaluates the three conditions the swarm must all clear
// to be the right tool for a task.
func CompareToBaseline(swarmQuality, baselineQuality float64, swarm SwarmAccounting, baselineTokens int, baselineWallMS int64) BaselineComparison {
	swarmTokens := swarm.TotalTokensIn + swarm.TotalTokensOut
	comparison := BaselineComparison{
		SwarmQuality:    swarmQuality,
		BaselineQuality: baselineQuality,
		SwarmTokens:     swarmTokens,
		BaselineTokens:  baselineTokens,
		SwarmWallMS:     swarm.WallMS,
		BaselineWallMS:  baselineWallMS,
	}
	comparison.QualityAtLeastBaseline = swarmQuality >= baselineQuality
	comparison.CheaperThanBaseline = swarmTokens < baselineTokens
	comparison.FasterThanBaseline = swarm.WallMS < baselineWallMS
	comparison.SwarmWins = comparison.QualityAtLeastBaseline && comparison.CheaperThanBaseline && comparison.FasterThanBaseline
	return comparison
}
