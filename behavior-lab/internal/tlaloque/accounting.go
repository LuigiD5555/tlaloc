package tlaloque

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// WorkerUsage is what a worker optionally reports about the cost it
// incurred. A deterministic script leaves it nil; a worker that called an
// LLM (directly or through a service) fills in what the provider told it.
// It rides on CapabilityResponse and is aggregated into SwarmAccounting.
type WorkerUsage struct {
	TokensIn      int   `json:"tokens_in,omitempty"`
	TokensOut     int   `json:"tokens_out,omitempty"`
	UpstreamCalls int   `json:"upstream_calls,omitempty"`
	ModelLoadMS   int64 `json:"model_load_ms,omitempty"`
}

// SwarmAccounting is the per-run cost picture needed to answer the central
// question: was fanning out to several small specialists actually cheaper
// and/or faster than one large call?
//
// Honesty note: TotalWorkMS, CriticalPathMS and the token totals are
// genuinely per-node and additive. PeakRSSDeltaBytes is process-global —
// it is the whole run's peak RSS growth, NOT attributable to any single
// node, because parallel nodes share one process (and share one LM Studio
// instance whose memory this cannot see at all). Treat it as a run-level
// ceiling, not a per-node measurement.
type SwarmAccounting struct {
	Schema string `json:"schema"`

	WallMS                 int64   `json:"wall_ms"`
	TotalWorkMS            int64   `json:"total_work_ms"`
	CriticalPathMS         int64   `json:"critical_path_ms"`
	CoordinationOverheadMS int64   `json:"coordination_overhead_ms"`
	ParallelEfficiency     float64 `json:"parallel_efficiency"`

	TotalTokensIn      int `json:"total_tokens_in,omitempty"`
	TotalTokensOut     int `json:"total_tokens_out,omitempty"`
	TotalUpstreamCalls int `json:"total_upstream_calls,omitempty"`

	TotalBytesIn  int64 `json:"total_bytes_in,omitempty"`
	TotalBytesOut int64 `json:"total_bytes_out,omitempty"`
	TotalQueueMS  int64 `json:"total_queue_ms,omitempty"`
	TotalRetries  int   `json:"total_retries,omitempty"`

	PeakRSSDeltaBytes int64 `json:"peak_rss_delta_bytes,omitempty"`

	NodesExecuted int    `json:"nodes_executed"`
	PeakParallel  int    `json:"peak_parallel"`
	PlanHash      string `json:"plan_hash"`
	TopologyHash  string `json:"topology_hash"`
}

const swarmAccountingSchema = "tlaloc.tlaloque-swarm.r0.accounting"

// computeAccounting derives the accounting block from the finished plan and
// the per-node executions. wallMS and peakRSSDelta come from the runner
// (they are measured, not derived); everything else is computed here so it
// can be unit-tested without a running swarm.
func computeAccounting(plan SwarmPlan, nodes map[string]SwarmNode, executions map[string]NodeExecution, wallMS, peakRSSDelta int64, peakParallel int) SwarmAccounting {
	acc := SwarmAccounting{
		Schema:            swarmAccountingSchema,
		WallMS:            wallMS,
		PeakRSSDeltaBytes: peakRSSDelta,
		PeakParallel:      peakParallel,
		PlanHash:          hashPlan(plan),
		TopologyHash:      hashTopology(nodes),
	}

	for _, ex := range executions {
		if ex.StartedAt.IsZero() {
			continue
		}
		acc.NodesExecuted++
		acc.TotalWorkMS += ex.DurationMS
		acc.TotalTokensIn += ex.TokensIn
		acc.TotalTokensOut += ex.TokensOut
		acc.TotalUpstreamCalls += ex.UpstreamCalls
		acc.TotalBytesIn += ex.BytesIn
		acc.TotalBytesOut += ex.BytesOut
		acc.TotalQueueMS += ex.QueueMS
		acc.TotalRetries += ex.Retries
	}

	acc.CriticalPathMS = criticalPathMS(nodes, executions)
	acc.CoordinationOverheadMS = wallMS - acc.CriticalPathMS
	if acc.CoordinationOverheadMS < 0 {
		acc.CoordinationOverheadMS = 0
	}
	if wallMS > 0 {
		acc.ParallelEfficiency = float64(acc.TotalWorkMS) / float64(wallMS)
	}
	return acc
}

// criticalPathMS is the longest dependency chain measured in node wall
// time — the shortest the run could possibly have been with unlimited
// parallelism. longest[n] = duration[n] + max(longest[dep]).
func criticalPathMS(nodes map[string]SwarmNode, executions map[string]NodeExecution) int64 {
	longest := map[string]int64{}
	var resolve func(id string) int64
	visiting := map[string]bool{}
	resolve = func(id string) int64 {
		if value, done := longest[id]; done {
			return value
		}
		if visiting[id] {
			return 0 // a cycle should be impossible here; fail safe
		}
		visiting[id] = true
		var deepest int64
		for _, dep := range nodes[id].DependsOn {
			if chain := resolve(dep); chain > deepest {
				deepest = chain
			}
		}
		visiting[id] = false
		total := deepest + executions[id].DurationMS
		longest[id] = total
		return total
	}

	var max int64
	for id := range nodes {
		if chain := resolve(id); chain > max {
			max = chain
		}
	}
	return max
}

func hashPlan(plan SwarmPlan) string {
	body, err := json.Marshal(plan)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// hashTopology hashes only the shape (node id -> sorted deps), independent
// of capability bindings or hints — two plans with the same DAG shape hash
// the same here even if they use different workers.
func hashTopology(nodes map[string]SwarmNode) string {
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	lines := make([]string, 0, len(ids))
	for _, id := range ids {
		deps := append([]string(nil), nodes[id].DependsOn...)
		sort.Strings(deps)
		line := id + ":"
		for index, dep := range deps {
			if index > 0 {
				line += ","
			}
			line += dep
		}
		lines = append(lines, line)
	}
	joined := ""
	for index, line := range lines {
		if index > 0 {
			joined += "\n"
		}
		joined += line
	}
	sum := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(sum[:])
}
