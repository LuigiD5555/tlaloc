package swarmbench

import (
	"context"
	"testing"
)

// The control condition's whole point: accuracy must not move with replica
// count, because every replica runs the identical DAG over the identical
// logic. Only throughput may change.
func TestReplicationAccuracyIsFlatAcrossReplicaCount(t *testing.T) {
	registry, err := BuildInProcessRegistry(12_000_000, 18_000_000)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := Generate("replication-flat", 55, 200)
	if err != nil {
		t.Fatal(err)
	}
	plan := BuildFanInPlan("replication-flat", 4)

	for _, replicas := range []int{1, 2, 4, 8, 16} {
		run, err := ExecuteReplicated(context.Background(), registry, plan, dataset, FanInTerminalNode, replicas)
		if err != nil {
			t.Fatalf("replicas=%d: %v", replicas, err)
		}
		if run.Score.ExactMatchRate != 1.0 {
			t.Fatalf("replicas=%d: exact_match_rate=%v, want 1.0 (clean text, exhaustive lexicon)", replicas, run.Score.ExactMatchRate)
		}
		if run.ItemCount != 200 {
			t.Fatalf("replicas=%d: item_count=%d", replicas, run.ItemCount)
		}
		if run.ReplicaCount != replicas {
			t.Fatalf("replica_count=%d, want %d", run.ReplicaCount, replicas)
		}
	}
}

// Replication must not change the DAG's own shape — the whole point of the
// control is that E/D/W stay fixed while only concurrent load changes.
func TestReplicationTopologyMatchesSingleExecute(t *testing.T) {
	registry, err := BuildInProcessRegistry(12_000_000, 18_000_000)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := Generate("replication-topology", 7, 40)
	if err != nil {
		t.Fatal(err)
	}
	plan := BuildFanInPlan("replication-topology", 4)

	single, err := Execute(context.Background(), registry, plan, dataset, FanInTerminalNode)
	if err != nil {
		t.Fatal(err)
	}
	replicated, err := ExecuteReplicated(context.Background(), registry, plan, dataset, FanInTerminalNode, 8)
	if err != nil {
		t.Fatal(err)
	}
	if replicated.Topology != single.Topology {
		t.Fatalf("replicated topology=%+v, want identical to single-execute topology=%+v", replicated.Topology, single.Topology)
	}
}

// Every item must be accounted for exactly once regardless of how unevenly
// replicaCount divides the dataset — no item silently dropped or doubled.
func TestReplicationCoversEveryItemExactlyOnceWithUnevenSharding(t *testing.T) {
	registry, err := BuildInProcessRegistry(12_000_000, 18_000_000)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := Generate("replication-uneven", 13, 97) // prime, does not divide evenly
	if err != nil {
		t.Fatal(err)
	}
	plan := BuildFanInPlan("replication-uneven", 4)
	run, err := ExecuteReplicated(context.Background(), registry, plan, dataset, FanInTerminalNode, 12)
	if err != nil {
		t.Fatal(err)
	}
	if run.ItemCount != 97 || run.Score.ItemCount != 97 {
		t.Fatalf("item_count=%d score.item_count=%d, want 97 for both", run.ItemCount, run.Score.ItemCount)
	}
}

func TestReplicationDefaultsToOneReplica(t *testing.T) {
	registry, err := BuildInProcessRegistry(12_000_000, 18_000_000)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := Generate("replication-default", 3, 10)
	if err != nil {
		t.Fatal(err)
	}
	plan := BuildFanInPlan("replication-default", 4)
	run, err := ExecuteReplicated(context.Background(), registry, plan, dataset, FanInTerminalNode, 0)
	if err != nil {
		t.Fatal(err)
	}
	if run.ReplicaCount != 1 {
		t.Fatalf("replica_count=%d, want the zero value defaulted to 1", run.ReplicaCount)
	}
}

// This is the population sweep replication actually supports: 1 -> 128
// concurrent replicas of the same fixed-shape swarm, correctness preserved
// throughout.
func TestReplicationScalesTo128ReplicasWithoutLosingAccuracy(t *testing.T) {
	registry, err := BuildInProcessRegistry(12_000_000, 18_000_000)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := Generate("replication-128", 128128, 512)
	if err != nil {
		t.Fatal(err)
	}
	plan := BuildFanInPlan("replication-128", 8)
	run, err := ExecuteReplicated(context.Background(), registry, plan, dataset, FanInTerminalNode, 128)
	if err != nil {
		t.Fatal(err)
	}
	if run.Score.ExactMatchRate != 1.0 {
		t.Fatalf("exact_match_rate=%v at 128 replicas, want 1.0", run.Score.ExactMatchRate)
	}
	if len(run.NodeErrors) != 0 {
		t.Fatalf("node_errors=%v at 128 replicas, want none", run.NodeErrors)
	}
}
