package tlaloque

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSelectProducerUsesRegistryStrategy(t *testing.T) {
	r := NewRegistry()
	workers := []CapabilityWorker{
		testWorker{desc: CapabilityDescriptor{ID: "tiny", Capability: "DETECT_INTENT", Scope: ScopeGeneral, Engine: EngineModel, InputSchema: "text", OutputSchema: "intent", ParameterCount: 4, Produces: []string{"claim.intent"}}},
		testWorker{desc: CapabilityDescriptor{ID: "rules", Capability: "DETECT_INTENT", Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "text", OutputSchema: "intent", Deterministic: true, ParameterCount: 100, Produces: []string{"claim.intent"}}},
	}
	for _, worker := range workers {
		if err := r.Register(worker); err != nil {
			t.Fatal(err)
		}
	}
	worker, err := r.SelectProducer(ProductSelectionRequest{Product: "claim.intent", PreferDeterministic: true})
	if err != nil {
		t.Fatal(err)
	}
	if worker.Descriptor().ID != "rules" {
		t.Fatalf("selected=%s", worker.Descriptor().ID)
	}
}

func TestResolveGoalMaterializesTypedProductDAG(t *testing.T) {
	r := NewRegistry()
	workers := []CapabilityWorker{
		testWorker{desc: CapabilityDescriptor{
			ID: "intent", Capability: "DETECT_INTENT", Scope: ScopeGeneral, Engine: EngineModel,
			InputSchema: "text", OutputSchema: "intent", ParameterCount: 4,
			Requires: []string{"input.text"}, Produces: []string{"claim.intent"},
		}},
		testWorker{desc: CapabilityDescriptor{
			ID: "router", Capability: "ROUTE", Scope: ScopeGeneral, Engine: EngineDeterministic,
			InputSchema: "intent", OutputSchema: "route", Deterministic: true,
			Requires: []string{"claim.intent"}, Produces: []string{"claim.route"},
		}},
	}
	for _, worker := range workers {
		if err := r.Register(worker); err != nil {
			t.Fatal(err)
		}
	}
	resolved, err := r.ResolveGoal(CapabilityGoal{
		Capability:          "ROUTE",
		PreferDeterministic: true,
		AvailableProducts:   []string{"input.text"},
	}, "typed-route", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Plan.Nodes) != 2 {
		t.Fatalf("nodes=%d", len(resolved.Plan.Nodes))
	}
	var intent, router *SwarmNode
	for i := range resolved.Plan.Nodes {
		node := &resolved.Plan.Nodes[i]
		switch node.WorkerID {
		case "intent":
			intent = node
		case "router":
			router = node
		}
	}
	if intent == nil || router == nil {
		t.Fatalf("plan=%+v", resolved.Plan.Nodes)
	}
	if len(intent.DependsOn) != 0 {
		t.Fatalf("external input should not create dependency: %v", intent.DependsOn)
	}
	if len(router.DependsOn) != 1 || router.DependsOn[0] != "intent" {
		t.Fatalf("router deps=%v", router.DependsOn)
	}
	if router.InputBindings["claim.intent"] != "intent" {
		t.Fatalf("router bindings=%v", router.InputBindings)
	}
}

func TestResolveGoalFailsWhenRequiredProductHasNoProducer(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(testWorker{desc: CapabilityDescriptor{
		ID: "router", Capability: "ROUTE", Scope: ScopeGeneral, Engine: EngineDeterministic,
		InputSchema: "intent", OutputSchema: "route", Deterministic: true,
		Requires: []string{"claim.intent"},
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ResolveGoal(CapabilityGoal{Capability: "ROUTE"}, "missing-product", 1); err == nil {
		t.Fatal("expected missing product producer error")
	}
}

func TestRunnerExposesTypedProductInsteadOfProducerIdentity(t *testing.T) {
	r := NewRegistry()
	producer := testWorker{
		desc: CapabilityDescriptor{ID: "intent", Capability: "DETECT_INTENT", Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "text", OutputSchema: "intent", Deterministic: true},
		fn:   func(CapabilityRequest) json.RawMessage { return json.RawMessage(`{"label":"BUY"}`) },
	}
	consumer := testWorker{
		desc: CapabilityDescriptor{ID: "router", Capability: "ROUTE", Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "intent", OutputSchema: "route", Deterministic: true},
		fn: func(req CapabilityRequest) json.RawMessage {
			if len(req.Context["claim.intent"]) == 0 {
				panic("typed product missing from context")
			}
			if len(req.Context["intent"]) != 0 {
				panic("typed binding leaked producer identity into context")
			}
			return json.RawMessage(`{"route":"checkout"}`)
		},
	}
	for _, worker := range []CapabilityWorker{producer, consumer} {
		if err := r.Register(worker); err != nil {
			t.Fatal(err)
		}
	}
	plan := SwarmPlan{ID: "typed-context", MaxParallel: 2, Nodes: []SwarmNode{
		{ID: "intent", Capability: "DETECT_INTENT", WorkerID: "intent"},
		{ID: "router", Capability: "ROUTE", WorkerID: "router", DependsOn: []string{"intent"}, InputBindings: map[string]string{"claim.intent": "intent"}},
	}}
	report, err := (SwarmRunner{Registry: r}).Run(context.Background(), plan, "typed-context-task", json.RawMessage(`{"text":"buy now"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !report.Succeeded {
		t.Fatalf("report=%+v", report)
	}
}

func TestSwarmPlanRejectsBindingOutsideDependencies(t *testing.T) {
	_, err := (SwarmPlan{ID: "bad-binding", Nodes: []SwarmNode{
		{ID: "a", Capability: "A"},
		{ID: "b", Capability: "B", InputBindings: map[string]string{"claim.a": "a"}},
	}}).Normalize()
	if err == nil {
		t.Fatal("expected binding/dependency validation error")
	}
}
