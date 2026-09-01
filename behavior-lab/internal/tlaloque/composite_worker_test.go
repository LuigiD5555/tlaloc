package tlaloque

import (
	"context"
	"encoding/json"
	"testing"
)

func constantWorker(id, capability, output string) testWorker {
	worker := testWorker{desc: generalDescriptor(id, capability)}
	worker.fn = func(CapabilityRequest) json.RawMessage { return json.RawMessage(output) }
	return worker
}

// Hierarchical composition is the core claim: a micro-swarm of atoms must be
// registrable as one ordinary Tlaloque inside a larger swarm, recursively.
func TestCompositeWorkerNestsInsideAnotherSwarm(t *testing.T) {
	atoms := NewRegistry()
	mustRegister(t, atoms,
		constantWorker("intent", "DETECT_INTENT", `{"intent":"SEARCH"}`),
		constantWorker("entity", "EXTRACT_ENTITY", `{"entity":"PEMEX"}`),
		constantWorker("date", "RESOLVE_DATE", `{"date":"2026-08-31"}`),
	)

	documentRouter := CompositeWorker{
		Desc: CapabilityDescriptor{
			ID: "document-router", Capability: "DOCUMENT_ROUTER", Scope: ScopeGeneral,
			Engine: "COMPOSITE", InputSchema: "text", OutputSchema: "route", Deterministic: true,
		},
		Plan: SwarmPlan{ID: "router-internals", MaxParallel: 3, Nodes: []SwarmNode{
			{ID: "intent", Capability: "DETECT_INTENT"},
			{ID: "entity", Capability: "EXTRACT_ENTITY"},
			{ID: "date", Capability: "RESOLVE_DATE"},
		}},
		Registry: atoms,
	}

	// The composite is registered as one individual in an outer swarm.
	outer := NewRegistry()
	mustRegister(t, outer, documentRouter, constantWorker("verifier", "VERIFY", `{"verified":true}`))

	plan := SwarmPlan{ID: "outer", MaxParallel: 2, Nodes: []SwarmNode{
		{ID: "router", Capability: "DOCUMENT_ROUTER"},
		{ID: "verify", Capability: "VERIFY", DependsOn: []string{"router"}},
	}}
	report, err := (SwarmRunner{Registry: outer}).Run(context.Background(), plan, "task", json.RawMessage(`{"text":"find Pemex"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !report.Succeeded {
		t.Fatalf("report=%+v", report)
	}
	// The outer swarm sees two individuals, not five.
	if report.ExecutedNodes != 2 {
		t.Fatalf("outer executed=%d, want the composite to count as one node", report.ExecutedNodes)
	}
	if report.RegisteredWorkers != 2 {
		t.Fatalf("outer registered=%d, want the sub-swarm hidden", report.RegisteredWorkers)
	}

	var routerNode NodeExecution
	for _, node := range report.Nodes {
		if node.NodeID == "router" {
			routerNode = node
		}
	}
	if routerNode.WorkerID != "document-router" {
		t.Fatalf("composite not attributed: %+v", routerNode)
	}
	// The composite's output carries the whole sub-swarm's terminal results.
	var terminals map[string]json.RawMessage
	if err := json.Unmarshal(routerNode.Output, &terminals); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"intent", "entity", "date"} {
		if _, ok := terminals[expected]; !ok {
			t.Fatalf("composite output missing %s: %v", expected, terminals)
		}
	}
}

// Three levels deep: atoms -> composite -> composite. This is the recursion
// the architecture depends on.
func TestCompositeWorkerComposesRecursively(t *testing.T) {
	atoms := NewRegistry()
	mustRegister(t, atoms, constantWorker("atom", "ATOM", `{"value":1}`))

	inner := CompositeWorker{
		Desc:     generalDescriptor("inner", "INNER"),
		Plan:     SwarmPlan{ID: "inner-plan", Nodes: []SwarmNode{{ID: "atom", Capability: "ATOM"}}},
		Registry: atoms,
	}
	middleRegistry := NewRegistry()
	mustRegister(t, middleRegistry, inner)

	outer := CompositeWorker{
		Desc:     generalDescriptor("outer", "OUTER"),
		Plan:     SwarmPlan{ID: "outer-plan", Nodes: []SwarmNode{{ID: "inner", Capability: "INNER"}}},
		Registry: middleRegistry,
	}
	response, err := outer.Execute(context.Background(), CapabilityRequest{TaskID: "t", NodeID: "n", Input: json.RawMessage(`{"x":1}`)})
	if err != nil {
		t.Fatal(err)
	}
	if response.WorkerID != "outer" {
		t.Fatalf("worker id=%s", response.WorkerID)
	}
	if !json.Valid(response.Output) {
		t.Fatalf("output=%s", response.Output)
	}
}

// A composite receiving dependency context must forward both the task input
// and that context to its sub-swarm.
func TestCompositeWorkerForwardsDependencyContext(t *testing.T) {
	atoms := NewRegistry()
	var seen json.RawMessage
	inspector := testWorker{desc: generalDescriptor("inspector", "INSPECT")}
	inspector.fn = func(req CapabilityRequest) json.RawMessage {
		seen = req.Input
		return json.RawMessage(`{"ok":true}`)
	}
	mustRegister(t, atoms, inspector)

	composite := CompositeWorker{
		Desc:     generalDescriptor("wrapper", "WRAP"),
		Plan:     SwarmPlan{ID: "wrap-plan", Nodes: []SwarmNode{{ID: "inspector", Capability: "INSPECT"}}},
		Registry: atoms,
	}
	_, err := composite.Execute(context.Background(), CapabilityRequest{
		TaskID:  "task",
		NodeID:  "node",
		Input:   json.RawMessage(`{"text":"hello"}`),
		Context: map[string]json.RawMessage{"intent": json.RawMessage(`{"intent":"SEARCH"}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	var wrapped struct {
		Input   json.RawMessage            `json:"input"`
		Context map[string]json.RawMessage `json:"context"`
	}
	if err := json.Unmarshal(seen, &wrapped); err != nil {
		t.Fatalf("sub-swarm input is not the wrapped envelope: %s", seen)
	}
	if string(wrapped.Input) != `{"text":"hello"}` {
		t.Fatalf("original input lost: %s", wrapped.Input)
	}
	if len(wrapped.Context) != 1 {
		t.Fatalf("dependency context lost: %v", wrapped.Context)
	}
}

// Without dependencies the sub-swarm receives the bare task input, unwrapped.
func TestCompositeWorkerPassesBareInputWithoutDependencies(t *testing.T) {
	atoms := NewRegistry()
	var seen json.RawMessage
	inspector := testWorker{desc: generalDescriptor("inspector", "INSPECT")}
	inspector.fn = func(req CapabilityRequest) json.RawMessage {
		seen = req.Input
		return json.RawMessage(`{"ok":true}`)
	}
	mustRegister(t, atoms, inspector)

	composite := CompositeWorker{
		Desc:     generalDescriptor("wrapper", "WRAP"),
		Plan:     SwarmPlan{ID: "wrap-plan", Nodes: []SwarmNode{{ID: "inspector", Capability: "INSPECT"}}},
		Registry: atoms,
	}
	if _, err := composite.Execute(context.Background(), CapabilityRequest{TaskID: "t", NodeID: "n", Input: json.RawMessage(`{"text":"hello"}`)}); err != nil {
		t.Fatal(err)
	}
	if string(seen) != `{"text":"hello"}` {
		t.Fatalf("sub-swarm input=%s, want the bare task input", seen)
	}
}

// A failure inside the sub-swarm must surface as a failure of the composite,
// not as a low-confidence success.
func TestCompositeWorkerPropagatesSubSwarmFailure(t *testing.T) {
	atoms := NewRegistry()
	mustRegister(t, atoms, failingWorker{desc: generalDescriptor("broken", "BROKEN"), failWith: "weights missing"})
	composite := CompositeWorker{
		Desc:     generalDescriptor("wrapper", "WRAP"),
		Plan:     SwarmPlan{ID: "wrap-plan", Nodes: []SwarmNode{{ID: "broken", Capability: "BROKEN"}}},
		Registry: atoms,
	}
	if _, err := composite.Execute(context.Background(), CapabilityRequest{TaskID: "t", NodeID: "n", Input: json.RawMessage(`{}`)}); err == nil {
		t.Fatal("expected the sub-swarm failure to surface")
	}
}

func TestCompositeWorkerRequiresRegistry(t *testing.T) {
	composite := CompositeWorker{
		Desc: generalDescriptor("wrapper", "WRAP"),
		Plan: SwarmPlan{ID: "wrap-plan", Nodes: []SwarmNode{{ID: "a", Capability: "A"}}},
	}
	if _, err := composite.Execute(context.Background(), CapabilityRequest{Input: json.RawMessage(`{}`)}); err == nil {
		t.Fatal("expected a composite with no registry to fail")
	}
}

func TestAggregateConfidenceIgnoresFailedNodes(t *testing.T) {
	cases := []struct {
		name  string
		nodes []NodeExecution
		want  float64
	}{
		{name: "no nodes", nodes: nil, want: 0},
		{
			name:  "averages successes",
			nodes: []NodeExecution{{Confidence: 0.8}, {Confidence: 0.6}},
			want:  0.7,
		},
		{
			name:  "skips errored nodes",
			nodes: []NodeExecution{{Confidence: 0.9}, {Confidence: 0.1, Error: "boom"}},
			want:  0.9,
		},
		{
			name:  "skips unscored nodes",
			nodes: []NodeExecution{{Confidence: 0.5}, {Confidence: 0}},
			want:  0.5,
		},
		{
			name:  "all failed",
			nodes: []NodeExecution{{Confidence: 0.9, Error: "boom"}},
			want:  0,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := aggregateConfidence(testCase.nodes)
			if diff := got - testCase.want; diff > 1e-9 || diff < -1e-9 {
				t.Fatalf("confidence=%v, want %v", got, testCase.want)
			}
		})
	}
}
