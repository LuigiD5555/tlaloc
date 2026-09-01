package swarmbench

import (
	"context"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// This is the actual experiment from Phase 2: five real Tlaloques — two
// heuristic stand-ins for small models (intent, entity), three deterministic,
// zero-parameter individuals (date-number, router, verifier) — wired into
// the wide, shallow topology the design rule recommends, run end-to-end
// against a generated dataset with ground truth.
func TestFiveTlaloqueSwarmRecoversFieldsOnCleanText(t *testing.T) {
	registry, err := BuildInProcessRegistry(12_000_000, 18_000_000)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := Generate("phase2", 2024, 500)
	if err != nil {
		t.Fatal(err)
	}
	plan := BuildFanInPlan("phase2-fan-in", 4)

	run, err := Execute(context.Background(), registry, plan, dataset, FanInTerminalNode)
	if err != nil {
		t.Fatal(err)
	}
	if run.Score.ItemCount != 500 {
		t.Fatalf("item_count=%d", run.Score.ItemCount)
	}
	// The gazetteer and lexicon are exhaustive over the dataset's own
	// vocabulary, and date/amount/route are exact — a clean-text sweep should
	// recover every field on every item.
	if run.Score.ExactMatchRate != 1.0 {
		t.Fatalf("exact_match_rate=%v, want 1.0 on clean generated text; node_errors=%v", run.Score.ExactMatchRate, run.NodeErrors)
	}
	if run.Score.RouteAccuracy != 1.0 {
		t.Fatalf("route_accuracy=%v, want 1.0", run.Score.RouteAccuracy)
	}
	if len(run.NodeErrors) != 0 {
		t.Fatalf("node_errors=%v, want none on clean text", run.NodeErrors)
	}
}

// The topology actually executed must match the wide, shallow shape the
// design rule recommends: depth 3 regardless of how many atoms sit at layer
// 1, never a chain that would compound error with dataset size.
func TestFiveTlaloqueSwarmTopologyIsWideNotDeep(t *testing.T) {
	plan := BuildFanInPlan("phase2-fan-in", 4)
	topology := AnalyzeTopology(plan)
	if topology.Nodes != 5 {
		t.Fatalf("nodes=%d", topology.Nodes)
	}
	if topology.Depth != 3 {
		t.Fatalf("depth=%d, want 3 (intent/entity/date-number -> route -> verify)", topology.Depth)
	}
	if topology.MaxWidth != 3 {
		t.Fatalf("max_width=%d, want 3 (the three independent extractors)", topology.MaxWidth)
	}
	if topology.Edges != 4 {
		t.Fatalf("edges=%d, want 4", topology.Edges)
	}
}

// This is the composition payoff made concrete: the verifier must recover a
// document the router got wrong purely because entity extraction missed,
// without EntityWorkerLogic itself being fixed. Route only depends on
// intent/amount/date, not on organization — so a missed entity should never
// have broken the route in the first place, and the swarm's exact-match on
// route must survive even though organization alone is now wrong.
func TestSwarmDegradesGracefullyWhenEntityIsUnrecoverable(t *testing.T) {
	registry, err := BuildInProcessRegistry(12_000_000, 18_000_000)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := Generate("phase2-degraded", 77, 200)
	if err != nil {
		t.Fatal(err)
	}
	// Replace every organization with one absent from the gazetteer so the
	// entity Tlaloque cannot recover it, while intent/date/amount remain
	// intact — this isolates one stage's failure from the rest of the swarm.
	for index := range dataset.Items {
		dataset.Items[index].Text = replaceOrganization(t, dataset.Items[index].Text, dataset.Items[index].Expected.Organization, "Proveedor Sin Registrar")
		dataset.Items[index].Expected.Organization = "Proveedor Sin Registrar"
	}
	plan := BuildFanInPlan("phase2-degraded", 4)

	run, err := Execute(context.Background(), registry, plan, dataset, FanInTerminalNode)
	if err != nil {
		t.Fatal(err)
	}
	// Organization must fail on every item (this is the injected failure).
	for _, field := range run.Score.FieldAccuracies {
		if field.Field == "organization" && field.Accuracy != 0 {
			t.Fatalf("organization_accuracy=%v, want 0 under the injected failure", field.Accuracy)
		}
		// Route must be entirely unaffected: it never depends on organization.
		if field.Field == "route" && field.Accuracy != 1.0 {
			t.Fatalf("route_accuracy=%v, want 1.0 — route does not depend on organization", field.Accuracy)
		}
	}
	if run.Score.ExactMatchRate != 0 {
		t.Fatalf("exact_match_rate=%v, want 0 (organization is part of an exact match)", run.Score.ExactMatchRate)
	}
}

func replaceOrganization(t *testing.T, text, oldOrganization, newOrganization string) string {
	t.Helper()
	index := indexOf(text, oldOrganization)
	if index < 0 {
		t.Fatalf("organization %q not found in %q", oldOrganization, text)
	}
	return text[:index] + newOrganization + text[index+len(oldOrganization):]
}

func indexOf(text, substr string) int {
	for index := 0; index+len(substr) <= len(text); index++ {
		if text[index:index+len(substr)] == substr {
			return index
		}
	}
	return -1
}

// A swarm that cannot reach its declared terminal node must fail loudly, not
// silently return an all-zero Fields.
func TestExecuteRejectsUnknownTerminalNode(t *testing.T) {
	registry, err := BuildInProcessRegistry(12_000_000, 18_000_000)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := Generate("phase2-bad-terminal", 3, 5)
	if err != nil {
		t.Fatal(err)
	}
	plan := BuildFanInPlan("phase2-bad-terminal", 2)
	run, err := Execute(context.Background(), registry, plan, dataset, "not-a-real-node")
	if err != nil {
		t.Fatal(err)
	}
	if run.Score.ExactMatchRate != 0 {
		t.Fatalf("exact_match_rate=%v, want 0 when the terminal node never resolves", run.Score.ExactMatchRate)
	}
	if len(run.NodeErrors) == 0 {
		t.Fatal("expected node_errors to record the missing terminal node")
	}
}

func TestExecuteRejectsInvalidPlan(t *testing.T) {
	registry, err := BuildInProcessRegistry(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := Generate("phase2-invalid-plan", 5, 5)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Execute(context.Background(), registry, tlaloque.SwarmPlan{}, dataset, "verify"); err == nil {
		t.Fatal("expected an invalid plan to be rejected")
	}
}
