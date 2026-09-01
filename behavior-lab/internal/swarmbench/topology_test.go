package swarmbench

import (
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func TestAnalyzeTopologyOnFanIn(t *testing.T) {
	// Wide, shallow: this is the topology the design rule recommends.
	// atom-1..atom-8 -> aggregate. E = 8, D = 2, W = 8.
	nodes := []tlaloque.SwarmNode{{ID: "aggregate", Capability: "AGGREGATE"}}
	dependencies := make([]string, 0, 8)
	for index := 0; index < 8; index++ {
		id := string(rune('a' + index))
		nodes = append(nodes, tlaloque.SwarmNode{ID: id, Capability: "ATOM"})
		dependencies = append(dependencies, id)
	}
	nodes[0].DependsOn = dependencies
	plan := tlaloque.SwarmPlan{ID: "fan-in", MaxParallel: 4, Nodes: nodes}

	topology := AnalyzeTopology(plan)
	if topology.Nodes != 9 {
		t.Fatalf("nodes=%d", topology.Nodes)
	}
	if topology.Edges != 8 {
		t.Fatalf("edges=%d, want 8", topology.Edges)
	}
	if topology.Depth != 2 {
		t.Fatalf("depth=%d, want 2", topology.Depth)
	}
	if topology.MaxWidth != 8 {
		t.Fatalf("max_width=%d, want 8", topology.MaxWidth)
	}
	if topology.MaxParallel != 4 {
		t.Fatalf("max_parallel=%d", topology.MaxParallel)
	}
}

// Narrow, deep: this is the topology the design rule warns against. Same
// node count as a chain, but depth equals population and width never exceeds 1.
func TestAnalyzeTopologyOnChain(t *testing.T) {
	nodes := []tlaloque.SwarmNode{{ID: "n0", Capability: "STAGE_0"}}
	for index := 1; index < 8; index++ {
		id := string(rune('a' + index))
		nodes = append(nodes, tlaloque.SwarmNode{ID: id, Capability: "STAGE", DependsOn: []string{nodes[len(nodes)-1].ID}})
	}
	plan := tlaloque.SwarmPlan{ID: "chain", MaxParallel: 8, Nodes: nodes}

	topology := AnalyzeTopology(plan)
	if topology.Nodes != 8 {
		t.Fatalf("nodes=%d", topology.Nodes)
	}
	if topology.Edges != 7 {
		t.Fatalf("edges=%d, want 7", topology.Edges)
	}
	if topology.Depth != 8 {
		t.Fatalf("depth=%d, want 8 (every node on the critical path)", topology.Depth)
	}
	if topology.MaxWidth != 1 {
		t.Fatalf("max_width=%d, want 1 (nothing runs concurrently)", topology.MaxWidth)
	}
}

// A diamond: two independent branches re-converge. Depth counts the longest
// path (3), not the node count (4), and width peaks at the branching layer.
func TestAnalyzeTopologyOnDiamond(t *testing.T) {
	plan := tlaloque.SwarmPlan{ID: "diamond", MaxParallel: 2, Nodes: []tlaloque.SwarmNode{
		{ID: "source", Capability: "SOURCE"},
		{ID: "left", Capability: "LEFT", DependsOn: []string{"source"}},
		{ID: "right", Capability: "RIGHT", DependsOn: []string{"source"}},
		{ID: "sink", Capability: "SINK", DependsOn: []string{"left", "right"}},
	}}
	topology := AnalyzeTopology(plan)
	if topology.Nodes != 4 {
		t.Fatalf("nodes=%d", topology.Nodes)
	}
	if topology.Edges != 4 {
		t.Fatalf("edges=%d, want 4", topology.Edges)
	}
	if topology.Depth != 3 {
		t.Fatalf("depth=%d, want 3 (source -> branch -> sink)", topology.Depth)
	}
	if topology.MaxWidth != 2 {
		t.Fatalf("max_width=%d, want 2 (left and right in the same layer)", topology.MaxWidth)
	}
}

// A single node with no dependencies: the population-1 baseline every sweep
// starts from.
func TestAnalyzeTopologyOnSingleNode(t *testing.T) {
	plan := tlaloque.SwarmPlan{ID: "solo", MaxParallel: 1, Nodes: []tlaloque.SwarmNode{{ID: "only", Capability: "EVERYTHING"}}}
	topology := AnalyzeTopology(plan)
	if topology.Nodes != 1 || topology.Edges != 0 || topology.Depth != 1 || topology.MaxWidth != 1 {
		t.Fatalf("topology=%+v, want the trivial single-node baseline", topology)
	}
}

// Edge count must scale with the number of dependency declarations, not with
// population — this is what the O(E) overhead claim depends on being true of
// the actual structure being measured, independent of node count N.
func TestAnalyzeTopologyEdgeCountTracksDependenciesNotPopulation(t *testing.T) {
	wideIndependent := tlaloque.SwarmPlan{ID: "wide", Nodes: make([]tlaloque.SwarmNode, 0, 32)}
	for index := 0; index < 32; index++ {
		wideIndependent.Nodes = append(wideIndependent.Nodes, tlaloque.SwarmNode{ID: string(rune('a'+index%26)) + string(rune('0'+index/26)), Capability: "ATOM"})
	}
	topology := AnalyzeTopology(wideIndependent)
	if topology.Edges != 0 {
		t.Fatalf("edges=%d, want 0 for 32 fully independent nodes", topology.Edges)
	}
	if topology.Depth != 1 {
		t.Fatalf("depth=%d, want 1 for fully independent nodes", topology.Depth)
	}
	if topology.MaxWidth != 32 {
		t.Fatalf("max_width=%d, want 32", topology.MaxWidth)
	}
}
