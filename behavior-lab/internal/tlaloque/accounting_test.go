package tlaloque

import (
	"math"
	"testing"
	"time"
)

func TestComputeAccounting_CriticalPathAndEfficiency(t *testing.T) {
	plan := SwarmPlan{
		ID: "acct-test",
		Nodes: []SwarmNode{
			{ID: "a", Capability: "A"},
			{ID: "b", Capability: "B"},
			{ID: "c", Capability: "C", DependsOn: []string{"a", "b"}},
		},
	}
	nodes := map[string]SwarmNode{}
	for _, node := range plan.Nodes {
		nodes[node.ID] = node
	}
	started := time.Now().UTC()
	executions := map[string]NodeExecution{
		"a": {NodeID: "a", StartedAt: started, DurationMS: 10, TokensIn: 5, TokensOut: 7},
		"b": {NodeID: "b", StartedAt: started, DurationMS: 30},
		"c": {NodeID: "c", StartedAt: started, DurationMS: 5, TokensIn: 100, TokensOut: 40, UpstreamCalls: 1},
	}

	// wall == critical path (perfect parallelism on the a/b fan-in).
	acc := computeAccounting(plan, nodes, executions, 35, 4096, 2)

	if acc.TotalWorkMS != 45 {
		t.Errorf("total work: got %d, want 45", acc.TotalWorkMS)
	}
	if acc.CriticalPathMS != 35 { // max(10,30) + 5
		t.Errorf("critical path: got %d, want 35", acc.CriticalPathMS)
	}
	if acc.CoordinationOverheadMS != 0 {
		t.Errorf("coordination overhead: got %d, want 0", acc.CoordinationOverheadMS)
	}
	if math.Abs(acc.ParallelEfficiency-45.0/35.0) > 1e-9 {
		t.Errorf("parallel efficiency: got %v, want %v", acc.ParallelEfficiency, 45.0/35.0)
	}
	if acc.TotalTokensIn != 105 || acc.TotalTokensOut != 47 || acc.TotalUpstreamCalls != 1 {
		t.Errorf("token totals wrong: %+v", acc)
	}
	if acc.NodesExecuted != 3 {
		t.Errorf("nodes executed: got %d, want 3", acc.NodesExecuted)
	}
	if acc.PeakRSSDeltaBytes != 4096 {
		t.Errorf("rss delta passthrough: got %d, want 4096", acc.PeakRSSDeltaBytes)
	}
}

func TestComputeAccounting_SequentialChainHasLowEfficiency(t *testing.T) {
	plan := SwarmPlan{
		ID: "chain",
		Nodes: []SwarmNode{
			{ID: "x", Capability: "X"},
			{ID: "y", Capability: "Y", DependsOn: []string{"x"}},
			{ID: "z", Capability: "Z", DependsOn: []string{"y"}},
		},
	}
	nodes := map[string]SwarmNode{}
	for _, node := range plan.Nodes {
		nodes[node.ID] = node
	}
	started := time.Now().UTC()
	executions := map[string]NodeExecution{
		"x": {NodeID: "x", StartedAt: started, DurationMS: 20},
		"y": {NodeID: "y", StartedAt: started, DurationMS: 20},
		"z": {NodeID: "z", StartedAt: started, DurationMS: 20},
	}
	acc := computeAccounting(plan, nodes, executions, 60, 0, 1)
	if acc.CriticalPathMS != 60 {
		t.Errorf("critical path of a pure chain must equal total work: got %d", acc.CriticalPathMS)
	}
	if math.Abs(acc.ParallelEfficiency-1.0) > 1e-9 {
		t.Errorf("a chain run at wall==work has efficiency 1.0, got %v", acc.ParallelEfficiency)
	}
}

func TestHashTopology_IgnoresBindingsButNotShape(t *testing.T) {
	shapeA := map[string]SwarmNode{
		"a": {ID: "a", Capability: "CAP_ONE", WorkerID: "worker-1"},
		"b": {ID: "b", Capability: "CAP_TWO", DependsOn: []string{"a"}},
	}
	shapeARebound := map[string]SwarmNode{
		"a": {ID: "a", Capability: "CAP_DIFFERENT", WorkerID: "worker-9"},
		"b": {ID: "b", Capability: "CAP_TWO", DependsOn: []string{"a"}},
	}
	shapeB := map[string]SwarmNode{
		"a": {ID: "a", Capability: "CAP_ONE"},
		"b": {ID: "b", Capability: "CAP_TWO"}, // no dependency — different DAG
	}

	if hashTopology(shapeA) != hashTopology(shapeARebound) {
		t.Error("topology hash must not change when only capability/worker bindings change")
	}
	if hashTopology(shapeA) == hashTopology(shapeB) {
		t.Error("topology hash must change when the dependency shape changes")
	}
}
