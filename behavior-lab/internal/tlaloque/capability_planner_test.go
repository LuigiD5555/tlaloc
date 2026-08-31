package tlaloque

import "testing"

func TestResolveGoalBuildsPinnedDependencyDAG(t *testing.T) {
	r := NewRegistry()
	workers := []CapabilityWorker{
		testWorker{desc:CapabilityDescriptor{ID:"intent", Capability:"DETECT_INTENT", Scope:ScopeGeneral, Engine:EngineModel, InputSchema:"text", OutputSchema:"intent", ParameterCount:10}},
		testWorker{desc:CapabilityDescriptor{ID:"entities", Capability:"EXTRACT_ENTITY", Scope:ScopeGeneral, Engine:EngineModel, InputSchema:"text", OutputSchema:"entities", ParameterCount:20}},
		testWorker{desc:CapabilityDescriptor{ID:"router", Capability:"ROUTE", Scope:ScopeGeneral, Engine:EngineDeterministic, InputSchema:"context", OutputSchema:"route", Deterministic:true, Dependencies:[]string{"DETECT_INTENT","EXTRACT_ENTITY"}}},
	}
	for _, w := range workers { if err := r.Register(w); err != nil { t.Fatal(err) } }
	resolved, err := r.ResolveGoal(CapabilityGoal{Capability:"ROUTE", PreferDeterministic:true}, "auto-route", 2)
	if err != nil { t.Fatal(err) }
	if len(resolved.Plan.Nodes) != 3 { t.Fatalf("nodes=%d", len(resolved.Plan.Nodes)) }
	var root *SwarmNode
	for i := range resolved.Plan.Nodes {
		if resolved.Plan.Nodes[i].WorkerID == "router" { root = &resolved.Plan.Nodes[i] }
	}
	if root == nil { t.Fatal("router node not selected") }
	if len(root.DependsOn) != 2 { t.Fatalf("deps=%v", root.DependsOn) }
	if resolved.Plan.MaxParallel != 2 { t.Fatalf("max_parallel=%d", resolved.Plan.MaxParallel) }
}
