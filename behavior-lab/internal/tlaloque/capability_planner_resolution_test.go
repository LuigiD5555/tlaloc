package tlaloque

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func dependentWorker(id, capability string, parameters int64, dependencies ...string) testWorker {
	worker := modelWorker(id, capability, parameters)
	worker.desc.Dependencies = dependencies
	return worker
}

func TestResolveGoalRequiresCapability(t *testing.T) {
	registry := NewRegistry()
	if _, err := registry.ResolveGoal(CapabilityGoal{}, "", 1); err == nil {
		t.Fatal("expected a missing goal capability to be refused")
	}
}

func TestResolveGoalAppliesDefaults(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry, modelWorker("intent", "DETECT_INTENT", 12_000_000))
	resolved, err := registry.ResolveGoal(CapabilityGoal{Capability: " detect_intent "}, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Goal.Capability != "DETECT_INTENT" {
		t.Fatalf("goal not normalised: %+v", resolved.Goal)
	}
	if resolved.Plan.ID != "auto-detect-intent" {
		t.Fatalf("plan id=%q, want a derived default", resolved.Plan.ID)
	}
	if resolved.Plan.MaxParallel != 1 {
		t.Fatalf("max_parallel=%d, want 1", resolved.Plan.MaxParallel)
	}
}

// Planning twice from the same catalog must yield byte-identical DAGs, or the
// scaling measurements are comparing different systems.
func TestResolveGoalIsReproducible(t *testing.T) {
	build := func() *Registry {
		registry := NewRegistry()
		mustRegister(t, registry,
			modelWorker("intent-a", "DETECT_INTENT", 12_000_000),
			modelWorker("intent-b", "DETECT_INTENT", 12_000_000),
			modelWorker("entity-a", "EXTRACT_ENTITY", 18_000_000),
			modelWorker("entity-b", "EXTRACT_ENTITY", 18_000_000),
			dependentWorker("router", "ROUTE", 0, "DETECT_INTENT", "EXTRACT_ENTITY"),
		)
		return registry
	}
	var previous []byte
	for attempt := 0; attempt < 20; attempt++ {
		resolved, err := build().ResolveGoal(CapabilityGoal{Capability: "ROUTE"}, "route-plan", 4)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(resolved.Plan)
		if err != nil {
			t.Fatal(err)
		}
		if previous != nil && !reflect.DeepEqual(previous, encoded) {
			t.Fatalf("planner is not reproducible:\n%s\n%s", previous, encoded)
		}
		previous = encoded
	}
}

// Every node in a generated plan must pin an explicit worker id so a later run
// cannot silently drift onto a different individual.
func TestResolveGoalPinsEveryNode(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry,
		modelWorker("intent", "DETECT_INTENT", 12_000_000),
		modelWorker("entity", "EXTRACT_ENTITY", 18_000_000),
		dependentWorker("router", "ROUTE", 0, "DETECT_INTENT", "EXTRACT_ENTITY"),
	)
	resolved, err := registry.ResolveGoal(CapabilityGoal{Capability: "ROUTE"}, "pinned", 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Selected) != len(resolved.Plan.Nodes) {
		t.Fatalf("selected=%d nodes=%d", len(resolved.Selected), len(resolved.Plan.Nodes))
	}
	for _, node := range resolved.Plan.Nodes {
		if node.WorkerID == "" {
			t.Fatalf("node %s is not pinned to a worker", node.ID)
		}
		if node.ID != node.WorkerID {
			t.Fatalf("node id %s and worker id %s diverge", node.ID, node.WorkerID)
		}
	}
}

// The parameter budget must reach transitive dependencies, not just the goal.
func TestResolveGoalPropagatesParameterBudgetToDependencies(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry,
		modelWorker("intent-huge", "DETECT_INTENT", 400_000_000),
		modelWorker("intent-tiny", "DETECT_INTENT", 12_000_000),
		dependentWorker("router", "ROUTE", 0, "DETECT_INTENT"),
	)
	resolved, err := registry.ResolveGoal(CapabilityGoal{Capability: "ROUTE", MaxParameters: 20_000_000}, "budget", 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, descriptor := range resolved.Selected {
		if descriptor.ParameterCount > 20_000_000 {
			t.Fatalf("worker %s has %d parameters, over the declared budget", descriptor.ID, descriptor.ParameterCount)
		}
	}
	for _, node := range resolved.Plan.Nodes {
		if node.MaxParameters != 20_000_000 {
			t.Fatalf("node %s did not carry the budget forward: %d", node.ID, node.MaxParameters)
		}
	}
}

func TestResolveGoalFailsWhenDependencyIsUnsatisfiable(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry, dependentWorker("router", "ROUTE", 0, "DETECT_INTENT"))
	_, err := registry.ResolveGoal(CapabilityGoal{Capability: "ROUTE"}, "broken", 1)
	if err == nil {
		t.Fatal("expected an unsatisfiable dependency to fail planning")
	}
	if !strings.Contains(err.Error(), "DETECT_INTENT") {
		t.Fatalf("error does not name the missing capability: %v", err)
	}
}

// A catalog whose declared dependencies loop back must be rejected at plan
// time rather than deadlocking at run time.
func TestResolveGoalRejectsCapabilityCycle(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry,
		dependentWorker("alpha", "ALPHA", 0, "BETA"),
		dependentWorker("beta", "BETA", 0, "ALPHA"),
	)
	_, err := registry.ResolveGoal(CapabilityGoal{Capability: "ALPHA"}, "cycle", 2)
	if err == nil {
		t.Fatal("expected a capability dependency cycle to be refused")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("error does not report a cycle: %v", err)
	}
}

// A worker reused by two parents must appear once, so a diamond stays a DAG
// and the shared individual is not executed twice.
func TestResolveGoalDeduplicatesSharedDependency(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry,
		modelWorker("tokenizer", "TOKENIZE", 0),
		dependentWorker("intent", "DETECT_INTENT", 12_000_000, "TOKENIZE"),
		dependentWorker("entity", "EXTRACT_ENTITY", 18_000_000, "TOKENIZE"),
		dependentWorker("router", "ROUTE", 0, "DETECT_INTENT", "EXTRACT_ENTITY"),
	)
	resolved, err := registry.ResolveGoal(CapabilityGoal{Capability: "ROUTE"}, "diamond", 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Plan.Nodes) != 4 {
		t.Fatalf("nodes=%d, want the shared dependency collapsed into one node", len(resolved.Plan.Nodes))
	}
	occurrences := 0
	for _, node := range resolved.Plan.Nodes {
		if node.WorkerID == "tokenizer" {
			occurrences++
		}
	}
	if occurrences != 1 {
		t.Fatalf("shared dependency appears %d times", occurrences)
	}
}

// Planning and execution must agree: a generated plan has to run as-is.
func TestResolvedPlanExecutesEndToEnd(t *testing.T) {
	registry := NewRegistry()
	intent := modelWorker("intent", "DETECT_INTENT", 12_000_000)
	intent.fn = func(CapabilityRequest) json.RawMessage { return json.RawMessage(`{"intent":"SEARCH"}`) }
	entity := modelWorker("entity", "EXTRACT_ENTITY", 18_000_000)
	entity.fn = func(CapabilityRequest) json.RawMessage { return json.RawMessage(`{"entity":"PEMEX"}`) }
	router := dependentWorker("router", "ROUTE", 0, "DETECT_INTENT", "EXTRACT_ENTITY")
	router.fn = func(req CapabilityRequest) json.RawMessage {
		if len(req.Context) != 2 {
			t.Errorf("router received %d dependency outputs, want 2", len(req.Context))
		}
		return json.RawMessage(`{"route":"documents"}`)
	}
	mustRegister(t, registry, intent, entity, router)

	resolved, err := registry.ResolveGoal(CapabilityGoal{Capability: "ROUTE", MaxParameters: 20_000_000}, "end-to-end", 2)
	if err != nil {
		t.Fatal(err)
	}
	report, err := (SwarmRunner{Registry: registry}).Run(context.Background(), resolved.Plan, "task", json.RawMessage(`{"text":"find Pemex"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !report.Succeeded || report.ExecutedNodes != 3 {
		t.Fatalf("report=%+v", report)
	}
	if string(report.TerminalOutputs["router"]) != `{"route":"documents"}` {
		t.Fatalf("terminal=%v", report.TerminalOutputs)
	}
}

// Domain evidence must reach dependencies too: a CFDI goal should not pull a
// generic extractor when a CFDI specialist exists.
func TestResolveGoalPropagatesDomainToDependencies(t *testing.T) {
	registry := NewRegistry()
	specialist := testWorker{desc: CapabilityDescriptor{
		ID: "cfdi-entity", Capability: "EXTRACT_ENTITY", Scope: ScopeSpecific, Domain: "CFDI",
		Engine: EngineModel, InputSchema: "text", OutputSchema: "entities", ParameterCount: 4_000_000,
	}}
	mustRegister(t, registry,
		modelWorker("generic-entity", "EXTRACT_ENTITY", 18_000_000),
		specialist,
		dependentWorker("router", "ROUTE", 0, "EXTRACT_ENTITY"),
	)
	resolved, err := registry.ResolveGoal(CapabilityGoal{Capability: "ROUTE", DomainHint: "CFDI"}, "domain", 2)
	if err != nil {
		t.Fatal(err)
	}
	chosen := map[string]bool{}
	for _, descriptor := range resolved.Selected {
		chosen[descriptor.ID] = true
	}
	if !chosen["cfdi-entity"] {
		t.Fatalf("domain evidence did not reach the dependency: %v", chosen)
	}
}
