package swarmbench

import (
	"context"
	"testing"
)

// The two axes are meant to compose without new plumbing: replicate the
// DECOMPOSED (8-node) swarm instead of the 5-node baseline. This is the
// combined condition the strategy table promised was possible but untested.
func TestReplicationComposesWithDecomposedPlan(t *testing.T) {
	registry, err := BuildDecomposedRegistry(12_000_000, 4_000_000, 4_000_000)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := Generate("replicated-decomposed", 606, 400)
	if err != nil {
		t.Fatal(err)
	}
	plan := BuildDecomposedPlan("replicated-decomposed", 4)

	for _, replicas := range []int{1, 4, 16, 64} {
		run, err := ExecuteReplicated(context.Background(), registry, plan, dataset, FanInTerminalNode, replicas)
		if err != nil {
			t.Fatalf("replicas=%d: %v", replicas, err)
		}
		if run.Score.ExactMatchRate != 1.0 {
			t.Fatalf("replicas=%d: exact_match_rate=%v, want 1.0", replicas, run.Score.ExactMatchRate)
		}
		// The replicated topology must still be the 8-node decomposed shape,
		// not silently fall back to the 5-node baseline's.
		if run.Topology.Nodes != 8 {
			t.Fatalf("replicas=%d: topology.nodes=%d, want 8 (decomposed)", replicas, run.Topology.Nodes)
		}
		if run.Topology.Depth != 4 {
			t.Fatalf("replicas=%d: topology.depth=%d, want 4", replicas, run.Topology.Depth)
		}
	}
}

// Combining both axes with the calibrated real-model failure must show the
// same ceiling replication alone showed: more replicas of a decomposed
// swarm cannot out-run a genuinely broken upstream classifier either,
// because replication does not touch per-item correctness at all.
func TestReplicationDoesNotRescueACollapsedStageEitherAtAnyReplicaCount(t *testing.T) {
	registry, err := BuildInProcessRegistryWithLogic(1_600_000_000, 18_000_000, IntentWorkerLogicLFM2VLProxy, nil)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := Generate("replicated-collapsed", 707, 400)
	if err != nil {
		t.Fatal(err)
	}
	plan := BuildFanInPlan("replicated-collapsed", 4)

	archiveShare := 0.0
	for _, item := range dataset.Items {
		if item.Expected.Route == RouteArchive {
			archiveShare++
		}
	}
	archiveShare /= float64(len(dataset.Items))

	for _, replicas := range []int{1, 8, 32} {
		run, err := ExecuteReplicated(context.Background(), registry, plan, dataset, FanInTerminalNode, replicas)
		if err != nil {
			t.Fatalf("replicas=%d: %v", replicas, err)
		}
		if diff := run.Score.RouteAccuracy - archiveShare; diff > 0.02 || diff < -0.02 {
			t.Fatalf("replicas=%d: route_accuracy=%v did not stay pinned to the collapsed-stage ceiling %v", replicas, run.Score.RouteAccuracy, archiveShare)
		}
	}
}
