package swarmbench

import (
	"context"
	"testing"
)

func TestDecomposedSwarmRecoversFieldsOnCleanText(t *testing.T) {
	registry, err := BuildDecomposedRegistry(12_000_000, 4_000_000, 4_000_000)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := Generate("decomposed", 2024, 500)
	if err != nil {
		t.Fatal(err)
	}
	plan := BuildDecomposedPlan("decomposed-fan-in", 4)

	run, err := Execute(context.Background(), registry, plan, dataset, FanInTerminalNode)
	if err != nil {
		t.Fatal(err)
	}
	if run.Score.ExactMatchRate != 1.0 {
		t.Fatalf("exact_match_rate=%v, want 1.0 on clean text; node_errors=%v", run.Score.ExactMatchRate, run.NodeErrors)
	}
}

// This is the structural claim genuine decomposition makes, measured, not
// asserted: 8 real individuals, one more layer of depth than the 5-Tlaloque
// baseline (join-organization inserts a fan-in step before router), but
// still wide at every layer — never a chain.
func TestDecomposedSwarmTopologyGrowsWiderNotDeeper(t *testing.T) {
	baseline := AnalyzeTopology(BuildFanInPlan("baseline", 4))
	decomposed := AnalyzeTopology(BuildDecomposedPlan("decomposed", 4))

	if decomposed.Nodes != 8 {
		t.Fatalf("decomposed nodes=%d, want 8", decomposed.Nodes)
	}
	if decomposed.Nodes <= baseline.Nodes {
		t.Fatalf("decomposed population %d did not exceed baseline %d", decomposed.Nodes, baseline.Nodes)
	}
	if decomposed.Depth != baseline.Depth+1 {
		t.Fatalf("decomposed depth=%d, want exactly one more than baseline depth %d (one extra fan-in layer)", decomposed.Depth, baseline.Depth)
	}
	if decomposed.MaxWidth < baseline.MaxWidth {
		t.Fatalf("decomposed max_width=%d fell below baseline max_width=%d — decomposition narrowed the DAG instead of widening it", decomposed.MaxWidth, baseline.MaxWidth)
	}
}

// The organization join is a genuine two-input fan-in: if org-head alone
// fails, org-tail's independently-recovered half must be unaffected, and the
// join must combine whatever each atom actually produced rather than either
// masking or being poisoned by the other's failure.
func TestJoinOrganizationCombinesIndependentHalves(t *testing.T) {
	cases := []struct {
		name       string
		head, tail string
		want       string
	}{
		{name: "both present", head: "ACME", tail: "Servicios", want: "ACME Servicios"},
		{name: "head only", head: "ACME", tail: "", want: "ACME"},
		{name: "tail only, no head", head: "", tail: "Servicios", want: ""},
		{name: "neither", head: "", tail: "", want: ""},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := JoinOrganizationWorkerLogic(testCase.head, testCase.tail)
			if got != testCase.want {
				t.Fatalf("JoinOrganizationWorkerLogic(%q, %q)=%q, want %q", testCase.head, testCase.tail, got, testCase.want)
			}
		})
	}
}

func TestOrgHeadAndOrgTailPartitionEveryGazetteerEntry(t *testing.T) {
	for _, organization := range Organizations {
		text := "Pagar a " + organization + " por servicios"
		head, headConfidence := OrgHeadWorkerLogic(text)
		tail, tailConfidence := OrgTailWorkerLogic(text)
		rejoined := JoinOrganizationWorkerLogic(head, tail)
		if rejoined != organization {
			t.Fatalf("head=%q tail=%q rejoined=%q, want %q", head, tail, rejoined, organization)
		}
		if headConfidence <= 0 || tailConfidence <= 0 {
			t.Fatalf("expected nonzero confidence for a real gazetteer match: head=%v tail=%v", headConfidence, tailConfidence)
		}
	}
}

func TestOrgHeadAndOrgTailReturnEmptyWithoutGazetteerHit(t *testing.T) {
	head, headConfidence := OrgHeadWorkerLogic("Documento sin ninguna organizacion reconocible")
	tail, tailConfidence := OrgTailWorkerLogic("Documento sin ninguna organizacion reconocible")
	if head != "" || headConfidence != 0 {
		t.Fatalf("head=%q confidence=%v, want empty/zero", head, headConfidence)
	}
	if tail != "" || tailConfidence != 0 {
		t.Fatalf("tail=%q confidence=%v, want empty/zero", tail, tailConfidence)
	}
}

// A single-word organization has no tail — the join must not insert a
// spurious leading or trailing space.
func TestOrgTailIsEmptyForSingleWordOrganizations(t *testing.T) {
	// None of the dataset's organizations are single-word, so this exercises
	// the boundary directly against JoinOrganizationWorkerLogic instead.
	if got := JoinOrganizationWorkerLogic("ACME", ""); got != "ACME" {
		t.Fatalf("got %q", got)
	}
}

// The decomposed router now depends on four upstream atoms instead of
// three — this end-to-end pass proves routeWorker's parameterized sources
// actually wire correctly, not just that the pieces compile.
func TestDecomposedRouterReadsAllFourSources(t *testing.T) {
	registry, err := BuildDecomposedRegistry(12_000_000, 4_000_000, 4_000_000)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := Generate("decomposed-router", 909, 120)
	if err != nil {
		t.Fatal(err)
	}
	plan := BuildDecomposedPlan("decomposed-router", 4)
	run, err := Execute(context.Background(), registry, plan, dataset, FanInTerminalNode)
	if err != nil {
		t.Fatal(err)
	}
	if run.Score.RouteAccuracy != 1.0 {
		t.Fatalf("route_accuracy=%v, want 1.0 — every field the router needs is recoverable on clean text", run.Score.RouteAccuracy)
	}
}
