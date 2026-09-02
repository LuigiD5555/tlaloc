package swarmask

import (
	"sort"
	"testing"

	"tlaloc.local/behaviorlab/internal/intent"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// shapeByCapability maps each node's capability to the sorted set of
// capabilities it depends on — a plan's topology independent of node
// naming.
func shapeByCapability(plan tlaloque.SwarmPlan) map[string][]string {
	capByNode := map[string]string{}
	for _, node := range plan.Nodes {
		capByNode[node.ID] = node.Capability
	}
	shape := map[string][]string{}
	for _, node := range plan.Nodes {
		deps := make([]string, 0, len(node.DependsOn))
		for _, dep := range node.DependsOn {
			deps = append(deps, capByNode[dep])
		}
		sort.Strings(deps)
		shape[node.Capability] = deps
	}
	return shape
}

func expectedShape() map[string][]string {
	return map[string][]string{
		ScoutCapability:        {},
		EntityCapability:       {},
		ClassifierCapability:   {},
		AnswerCapability:       sortedCaps(ScoutCapability, EntityCapability, ClassifierCapability),
		ConsolidatorCapability: {AnswerCapability},
	}
}

func sortedCaps(caps ...string) []string {
	out := append([]string(nil), caps...)
	sort.Strings(out)
	return out
}

func equalShape(a, b map[string][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for capability, depsA := range a {
		depsB, ok := b[capability]
		if !ok || len(depsA) != len(depsB) {
			return false
		}
		for i := range depsA {
			if depsA[i] != depsB[i] {
				return false
			}
		}
	}
	return true
}

// NewPlan resolves the DAG from worker-descriptor Dependencies via
// ResolveGoal — it must reconstruct exactly the 5-node topology the plan
// used to hand-wire.
func TestNewPlan_ResolvesTheHandWiredTopology(t *testing.T) {
	registry, err := NewRegistry(RegistryConfig{})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	plan, err := NewPlan(registry)
	if err != nil {
		t.Fatalf("NewPlan: %v", err)
	}
	if len(plan.Nodes) != 5 {
		t.Fatalf("expected 5 nodes, got %d", len(plan.Nodes))
	}
	if got := shapeByCapability(plan); !equalShape(got, expectedShape()) {
		t.Errorf("resolved topology mismatch:\n got  %v\n want %v", got, expectedShape())
	}
}

// The full seam: IntentIR -> Compile -> PlanFor -> ResolveGoal produces the
// same swarm DAG that swarmask.NewPlan builds, against the real registry.
func TestIntentPlanFor_ProducesTheSwarmDAG(t *testing.T) {
	registry, err := NewRegistry(RegistryConfig{})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	compiled, err := intent.Compile(intent.IntentIR{
		Schema:          intent.Schema,
		Version:         "1",
		Goal:            "Answer a question about the book and verify the answer is grounded.",
		RequiredOutputs: []string{ConsolidatorCapability},
		Invariants:      []intent.Invariant{{ID: "grounded", Statement: "the answer must be supported by the cited page"}},
		Budget:          intent.Budget{MaxTokens: 4000},
		Risk:            intent.RiskProfile{Level: "medium"},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	planned, err := intent.PlanFor(registry, compiled, "intent-swarmask", planMaxParallel)
	if err != nil {
		t.Fatalf("PlanFor: %v", err)
	}
	if len(planned.Plan.Nodes) != 5 {
		t.Fatalf("expected a 5-node plan from the intent, got %d", len(planned.Plan.Nodes))
	}
	if got := shapeByCapability(planned.Plan); !equalShape(got, expectedShape()) {
		t.Errorf("intent-planned topology mismatch:\n got  %v\n want %v", got, expectedShape())
	}
}
