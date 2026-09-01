package tlaloque

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// buildFanInSwarm creates `width` independent atoms feeding one aggregator,
// the shape the 1 -> 2 -> 4 -> ... -> 128 measurements will use.
func buildFanInSwarm(t *testing.T, width int, maxParallel int, delay time.Duration, active, peak *int32) (*Registry, SwarmPlan) {
	t.Helper()
	registry := NewRegistry()
	nodes := make([]SwarmNode, 0, width+1)
	dependencies := make([]string, 0, width)
	for index := 0; index < width; index++ {
		id := fmt.Sprintf("atom-%03d", index)
		capability := fmt.Sprintf("ATOM_%03d", index)
		worker := testWorker{desc: generalDescriptor(id, capability), delay: delay, active: active, peak: peak}
		worker.fn = func(CapabilityRequest) json.RawMessage { return json.RawMessage(`{"v":1}`) }
		mustRegister(t, registry, worker)
		nodes = append(nodes, SwarmNode{ID: id, Capability: capability})
		dependencies = append(dependencies, id)
	}
	aggregator := testWorker{desc: generalDescriptor("aggregator", "AGGREGATE")}
	aggregator.fn = func(req CapabilityRequest) json.RawMessage {
		return json.RawMessage(fmt.Sprintf(`{"received":%d}`, len(req.Context)))
	}
	mustRegister(t, registry, aggregator)
	nodes = append(nodes, SwarmNode{ID: "aggregate", Capability: "AGGREGATE", DependsOn: dependencies})

	return registry, SwarmPlan{ID: fmt.Sprintf("fan-in-%d", width), MaxParallel: maxParallel, Nodes: nodes}
}

// The swarm must stay correct and bounded at every population the experiment
// will sweep, and the aggregator must actually see every individual's output.
func TestSwarmScalesAcrossPopulations(t *testing.T) {
	for _, width := range []int{1, 2, 4, 8, 16, 32, 64, 128} {
		t.Run(fmt.Sprintf("width-%d", width), func(t *testing.T) {
			var active, peak int32
			maxParallel := 8
			registry, plan := buildFanInSwarm(t, width, maxParallel, 0, &active, &peak)

			report, err := (SwarmRunner{Registry: registry}).Run(context.Background(), plan, "scale", json.RawMessage(`{}`))
			if err != nil {
				t.Fatal(err)
			}
			if !report.Succeeded {
				t.Fatalf("width %d did not succeed: %+v", width, report)
			}
			if report.ExecutedNodes != width+1 {
				t.Fatalf("executed=%d, want %d", report.ExecutedNodes, width+1)
			}
			if observed := atomic.LoadInt32(&peak); observed > int32(maxParallel) {
				t.Fatalf("width %d exceeded MaxParallel: %d", width, observed)
			}
			var aggregated struct {
				Received int `json:"received"`
			}
			if err := json.Unmarshal(report.TerminalOutputs["aggregate"], &aggregated); err != nil {
				t.Fatal(err)
			}
			if aggregated.Received != width {
				t.Fatalf("aggregator saw %d of %d individuals", aggregated.Received, width)
			}
			if report.RegisteredWorkers != width+1 {
				t.Fatalf("registered=%d, want %d", report.RegisteredWorkers, width+1)
			}
		})
	}
}

// Raising the parallel budget on the same population must actually raise the
// observed concurrency, otherwise the measured speedup would be an artefact.
func TestSwarmParallelBudgetChangesObservedConcurrency(t *testing.T) {
	const width = 32
	observedFor := func(maxParallel int) int32 {
		var active, peak int32
		registry, plan := buildFanInSwarm(t, width, maxParallel, 5*time.Millisecond, &active, &peak)
		report, err := (SwarmRunner{Registry: registry}).Run(context.Background(), plan, "budget", json.RawMessage(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		if !report.Succeeded {
			t.Fatalf("report=%+v", report)
		}
		return atomic.LoadInt32(&peak)
	}
	serial := observedFor(1)
	if serial != 1 {
		t.Fatalf("MaxParallel=1 observed %d concurrent workers", serial)
	}
	wide := observedFor(8)
	if wide <= serial {
		t.Fatalf("raising the budget did not increase concurrency: %d then %d", serial, wide)
	}
	if wide > 8 {
		t.Fatalf("observed %d concurrent workers over a budget of 8", wide)
	}
}

// A single failing individual in a large population must fail the run and be
// attributable, not be lost among 127 successes.
func TestSwarmAttributesFailureInLargePopulation(t *testing.T) {
	var active, peak int32
	registry, plan := buildFanInSwarm(t, 64, 8, 0, &active, &peak)
	mustRegister(t, registry, failingWorker{desc: generalDescriptor("saboteur", "SABOTAGE"), failWith: "weights missing"})
	plan.Nodes = append(plan.Nodes, SwarmNode{ID: "saboteur", Capability: "SABOTAGE"})

	report, err := (SwarmRunner{Registry: registry}).Run(context.Background(), plan, "sabotage", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected the run to fail")
	}
	if report.Succeeded {
		t.Fatalf("report=%+v", report)
	}
	found := false
	for _, node := range report.Nodes {
		if node.NodeID == "saboteur" && node.Error != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("the failing individual was not attributed in the report")
	}
}
