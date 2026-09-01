package swarmbench

import "tlaloc.local/behaviorlab/internal/tlaloque"

// Topology is the structural description of one SwarmPlan, read directly from
// its edges rather than estimated. It is the input to the cost model
//
//	T(N) = t_setup + ceil(N/P) * mean_node_latency + E*c_edge + t_aggregate
//
// discussed for the decomposition-vs-replication experiment: overhead is
// governed by edge count, not by population, and accuracy is expected to
// decay exponentially with critical-path depth rather than with node count.
type Topology struct {
	Nodes       int `json:"nodes"`        // N: population size
	Edges       int `json:"edges"`        // E: dependency edges in the DAG
	Depth       int `json:"depth"`        // D: longest dependency chain (critical path)
	MaxWidth    int `json:"max_width"`    // W: largest independent layer
	MaxParallel int `json:"max_parallel"` // P: declared concurrency ceiling
}

// AnalyzeTopology computes Topology from an already-normalized SwarmPlan.
func AnalyzeTopology(plan tlaloque.SwarmPlan) Topology {
	edges := 0
	indegree := map[string]int{}
	children := map[string][]string{}
	for _, node := range plan.Nodes {
		indegree[node.ID] = len(node.DependsOn)
		edges += len(node.DependsOn)
		for _, dependency := range node.DependsOn {
			children[dependency] = append(children[dependency], node.ID)
		}
	}

	depth := map[string]int{}
	layer := []string{}
	remaining := map[string]int{}
	for id, count := range indegree {
		remaining[id] = count
		if count == 0 {
			layer = append(layer, id)
			depth[id] = 1
		}
	}

	maxWidth := 0
	longestChain := 0
	for len(layer) > 0 {
		if len(layer) > maxWidth {
			maxWidth = len(layer)
		}
		next := []string{}
		for _, id := range layer {
			if depth[id] > longestChain {
				longestChain = depth[id]
			}
			for _, child := range children[id] {
				remaining[child]--
				if depth[child] < depth[id]+1 {
					depth[child] = depth[id] + 1
				}
				if remaining[child] == 0 {
					next = append(next, child)
				}
			}
		}
		layer = next
	}

	return Topology{
		Nodes:       len(plan.Nodes),
		Edges:       edges,
		Depth:       longestChain,
		MaxWidth:    maxWidth,
		MaxParallel: plan.MaxParallel,
	}
}
