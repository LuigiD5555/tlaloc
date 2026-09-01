package tlaloque

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// failingWorker lets a test assert how the swarm isolates a worker error.
type failingWorker struct {
	desc     CapabilityDescriptor
	failWith string
	calls    *int32
}

func (w failingWorker) Descriptor() CapabilityDescriptor { return w.desc }
func (w failingWorker) Execute(context.Context, CapabilityRequest) (CapabilityResponse, error) {
	if w.calls != nil {
		atomic.AddInt32(w.calls, 1)
	}
	return CapabilityResponse{}, fmt.Errorf("%s", w.failWith)
}

func generalDescriptor(id, capability string) CapabilityDescriptor {
	return CapabilityDescriptor{
		ID:            id,
		Capability:    capability,
		Scope:         ScopeGeneral,
		Engine:        EngineDeterministic,
		InputSchema:   "json",
		OutputSchema:  "json",
		Deterministic: true,
	}
}

func mustRegister(t *testing.T, registry *Registry, workers ...CapabilityWorker) {
	t.Helper()
	for _, worker := range workers {
		if err := registry.Register(worker); err != nil {
			t.Fatalf("register %s: %v", worker.Descriptor().ID, err)
		}
	}
}

// MaxParallel is the ceiling the scaling experiment relies on: an unbounded
// swarm would make 1 -> 2 -> ... -> 128 measurements meaningless.
func TestSwarmNeverExceedsMaxParallel(t *testing.T) {
	var active, peak int32
	registry := NewRegistry()
	nodes := make([]SwarmNode, 0, 8)
	for index := 0; index < 8; index++ {
		id := fmt.Sprintf("leaf-%d", index)
		worker := testWorker{
			desc:   generalDescriptor(id, "LEAF_"+id),
			delay:  25 * time.Millisecond,
			active: &active,
			peak:   &peak,
		}
		mustRegister(t, registry, worker)
		nodes = append(nodes, SwarmNode{ID: id, Capability: "LEAF_" + id})
	}

	plan := SwarmPlan{ID: "parallel-ceiling", MaxParallel: 3, Nodes: nodes}
	report, err := (SwarmRunner{Registry: registry}).Run(context.Background(), plan, "task", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if !report.Succeeded || report.ExecutedNodes != 8 {
		t.Fatalf("report=%+v", report)
	}
	if observed := atomic.LoadInt32(&peak); observed > 3 {
		t.Fatalf("observed %d concurrent workers, MaxParallel=3", observed)
	}
	if report.PeakParallel > plan.MaxParallel {
		t.Fatalf("reported peak %d exceeds MaxParallel %d", report.PeakParallel, plan.MaxParallel)
	}
	if report.PeakParallel < 2 {
		t.Fatalf("expected real parallelism, peak=%d", report.PeakParallel)
	}
}

// A resident model worker declaring MAX_CONCURRENCY=1 must never be entered
// twice at once, even when the global budget would allow it.
func TestSwarmHonoursPerWorkerMaxConcurrency(t *testing.T) {
	var active, peak int32
	registry := NewRegistry()
	shared := generalDescriptor("serial-model", "SCORE")
	shared.MaxConcurrency = 1
	mustRegister(t, registry, testWorker{desc: shared, delay: 20 * time.Millisecond, active: &active, peak: &peak})

	nodes := []SwarmNode{
		{ID: "score-a", Capability: "SCORE", WorkerID: "serial-model"},
		{ID: "score-b", Capability: "SCORE", WorkerID: "serial-model"},
		{ID: "score-c", Capability: "SCORE", WorkerID: "serial-model"},
	}
	plan := SwarmPlan{ID: "serial-worker", MaxParallel: 8, Nodes: nodes}
	report, err := (SwarmRunner{Registry: registry}).Run(context.Background(), plan, "task", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if !report.Succeeded {
		t.Fatalf("report=%+v", report)
	}
	if observed := atomic.LoadInt32(&peak); observed != 1 {
		t.Fatalf("worker with MaxConcurrency=1 entered %d times concurrently", observed)
	}
}

// Independent workers with room in the global budget must overlap even when
// each one individually is serial.
func TestSwarmParallelisesDistinctSerialWorkers(t *testing.T) {
	var active, peak int32
	registry := NewRegistry()
	for _, id := range []string{"intent", "entity"} {
		desc := generalDescriptor(id, strings.ToUpper(id))
		desc.MaxConcurrency = 1
		mustRegister(t, registry, testWorker{desc: desc, delay: 25 * time.Millisecond, active: &active, peak: &peak})
	}
	plan := SwarmPlan{ID: "two-residents", MaxParallel: 2, Nodes: []SwarmNode{
		{ID: "intent", Capability: "INTENT"},
		{ID: "entity", Capability: "ENTITY"},
	}}
	report, err := (SwarmRunner{Registry: registry}).Run(context.Background(), plan, "task", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if report.PeakParallel != 2 || atomic.LoadInt32(&peak) != 2 {
		t.Fatalf("expected 2 distinct workers in parallel, report=%d observed=%d", report.PeakParallel, peak)
	}
}

// A failed node must not release its dependents: a swarm that routes on a
// missing intent would silently produce a wrong answer instead of an error.
func TestSwarmFailureBlocksDependentsAndIsReported(t *testing.T) {
	var routerCalls int32
	registry := NewRegistry()
	mustRegister(t, registry,
		failingWorker{desc: generalDescriptor("intent", "DETECT_INTENT"), failWith: "model unavailable"},
		failingWorker{desc: generalDescriptor("router", "ROUTE"), failWith: "unreachable", calls: &routerCalls},
	)
	plan := SwarmPlan{ID: "blocked", MaxParallel: 4, Nodes: []SwarmNode{
		{ID: "intent", Capability: "DETECT_INTENT"},
		{ID: "route", Capability: "ROUTE", DependsOn: []string{"intent"}},
	}}
	report, err := (SwarmRunner{Registry: registry}).Run(context.Background(), plan, "task", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected the swarm to surface the node failure")
	}
	if report.Succeeded {
		t.Fatalf("report claims success despite failure: %+v", report)
	}
	if calls := atomic.LoadInt32(&routerCalls); calls != 0 {
		t.Fatalf("dependent ran %d times after its dependency failed", calls)
	}
	if report.ExecutedNodes != 1 {
		t.Fatalf("executed=%d, want only the failed node recorded", report.ExecutedNodes)
	}
	if report.Nodes[0].Error == "" || !strings.Contains(report.Nodes[0].Error, "model unavailable") {
		t.Fatalf("failure not attributed in report: %+v", report.Nodes[0])
	}
	if len(report.TerminalOutputs) != 0 {
		t.Fatalf("terminal outputs leaked from a failed swarm: %v", report.TerminalOutputs)
	}
}

// Selection failure is a planning error, not a worker error, and must be
// attributed to the node that could not be satisfied.
func TestSwarmReportsUnsatisfiableCapability(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry, testWorker{desc: generalDescriptor("intent", "DETECT_INTENT")})
	plan := SwarmPlan{ID: "missing", Nodes: []SwarmNode{{ID: "route", Capability: "ROUTE"}}}
	report, err := (SwarmRunner{Registry: registry}).Run(context.Background(), plan, "task", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected an error for an unsatisfiable capability")
	}
	if !strings.Contains(err.Error(), "route") {
		t.Fatalf("error does not name the node: %v", err)
	}
	if report.Succeeded {
		t.Fatalf("report=%+v", report)
	}
}

// Every node receives the untouched task payload plus its dependency outputs;
// nothing else may cross the boundary.
func TestSwarmPassesOriginalInputToEveryNode(t *testing.T) {
	registry := NewRegistry()
	input := `{"text":"find Pemex"}`
	seen := map[string]string{}
	var mu sync.Mutex

	record := func(id string) testWorker {
		worker := testWorker{desc: generalDescriptor(id, strings.ToUpper(id))}
		worker.fn = func(req CapabilityRequest) json.RawMessage {
			mu.Lock()
			seen[req.NodeID] = string(req.Input)
			mu.Unlock()
			return json.RawMessage(`{"id":"` + id + `"}`)
		}
		return worker
	}
	first := record("first")
	second := record("second")
	mustRegister(t, registry, first, second)

	plan := SwarmPlan{ID: "payload", MaxParallel: 2, Nodes: []SwarmNode{
		{ID: "first", Capability: "FIRST"},
		{ID: "second", Capability: "SECOND", DependsOn: []string{"first"}},
	}}
	if _, err := (SwarmRunner{Registry: registry}).Run(context.Background(), plan, "task", json.RawMessage(input)); err != nil {
		t.Fatal(err)
	}
	for _, node := range []string{"first", "second"} {
		if seen[node] != input {
			t.Fatalf("node %s received %q, want the original task input", node, seen[node])
		}
	}
}

// Cancellation mid-flight must not be reported as a successful run.
func TestSwarmCancellationIsNotReportedAsSuccess(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry,
		testWorker{desc: generalDescriptor("slow-a", "SLOW_A"), delay: 500 * time.Millisecond},
		testWorker{desc: generalDescriptor("slow-b", "SLOW_B"), delay: 500 * time.Millisecond},
	)
	plan := SwarmPlan{ID: "cancelled", MaxParallel: 1, Nodes: []SwarmNode{
		{ID: "slow-a", Capability: "SLOW_A"},
		{ID: "slow-b", Capability: "SLOW_B"},
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	report, _ := (SwarmRunner{Registry: registry}).Run(ctx, plan, "task", json.RawMessage(`{}`))
	if report.Succeeded {
		t.Fatalf("cancelled swarm reported success: %+v", report)
	}
	if report.ExecutedNodes >= len(plan.Nodes) {
		t.Fatalf("cancelled swarm executed every node: %+v", report)
	}
}

// A cancelled swarm must still explain itself. Without this the 128-Tlaloque
// runs can produce reports that fail with no attributable cause.
func TestSwarmCancellationLeavesAnAttributableCause(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry,
		testWorker{desc: generalDescriptor("slow-a", "SLOW_A"), delay: 500 * time.Millisecond},
		testWorker{desc: generalDescriptor("slow-b", "SLOW_B"), delay: 500 * time.Millisecond},
	)
	plan := SwarmPlan{ID: "cancelled-cause", MaxParallel: 1, Nodes: []SwarmNode{
		{ID: "slow-a", Capability: "SLOW_A"},
		{ID: "slow-b", Capability: "SLOW_B"},
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	report, err := (SwarmRunner{Registry: registry}).Run(ctx, plan, "task", json.RawMessage(`{}`))

	missingNodes := len(plan.Nodes) - report.ExecutedNodes
	if missingNodes == 0 {
		t.Skip("swarm completed before cancellation; nothing to attribute")
	}
	var recorded []string
	for _, node := range report.Nodes {
		if node.Error != "" {
			recorded = append(recorded, node.NodeID+": "+node.Error)
		}
	}
	if err == nil && len(recorded) == 0 {
		t.Fatalf("swarm dropped %d node(s) with no error and no per-node cause; report=%+v", missingNodes, report)
	}
}

// This is the direct regression test for the abandoned-node gap: every root
// node is launched (none depend on anything), but MaxParallel:1 means only
// one can ever acquire the global semaphore before the short-lived context
// expires. Every launched node must take exactly one of two paths — execute,
// or be recorded as skipped — never vanish.
func TestSwarmRecordsSkippedNodesOnCancellation(t *testing.T) {
	const populationSize = 6
	registry := NewRegistry()
	nodes := make([]SwarmNode, 0, populationSize)
	for index := 0; index < populationSize; index++ {
		id := fmt.Sprintf("slow-%d", index)
		mustRegister(t, registry, testWorker{desc: generalDescriptor(id, "SLOW_"+id), delay: 300 * time.Millisecond})
		nodes = append(nodes, SwarmNode{ID: id, Capability: "SLOW_" + id})
	}
	plan := SwarmPlan{ID: "skipped-on-cancel", MaxParallel: 1, Nodes: nodes}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	report, err := (SwarmRunner{Registry: registry}).Run(ctx, plan, "task", json.RawMessage(`{}`))

	if err == nil {
		t.Fatal("expected the cancelled run to surface an error")
	}
	if report.SkippedNodes == 0 {
		t.Fatalf("expected at least one skipped node, report=%+v", report)
	}
	if report.ExecutedNodes+report.SkippedNodes != populationSize {
		t.Fatalf("executed(%d) + skipped(%d) = %d, want every launched root accounted for (%d)",
			report.ExecutedNodes, report.SkippedNodes, report.ExecutedNodes+report.SkippedNodes, populationSize)
	}
	skippedCount := 0
	for _, node := range report.Nodes {
		if node.Skipped {
			skippedCount++
			if node.Error == "" {
				t.Fatalf("skipped node %s has no error", node.NodeID)
			}
			if node.WorkerID == "" {
				t.Fatalf("skipped node %s has no worker id — not attributable", node.NodeID)
			}
		}
	}
	if skippedCount != report.SkippedNodes {
		t.Fatalf("found %d Skipped:true entries, report.SkippedNodes=%d", skippedCount, report.SkippedNodes)
	}
}

// The population test above only exercises the global semaphore. This
// exercises the second, independent escape point: several nodes pinned to
// the same resident worker (MaxConcurrency:1), with the global budget wide
// open so they can only ever block on the worker's own semaphore.
func TestSwarmRecordsSkippedNodesOnWorkerSemaphoreCancellation(t *testing.T) {
	registry := NewRegistry()
	sharedDescriptor := generalDescriptor("shared-resident", "SCORE")
	sharedDescriptor.MaxConcurrency = 1
	mustRegister(t, registry, testWorker{desc: sharedDescriptor, delay: 300 * time.Millisecond})

	nodes := []SwarmNode{
		{ID: "score-a", Capability: "SCORE", WorkerID: "shared-resident"},
		{ID: "score-b", Capability: "SCORE", WorkerID: "shared-resident"},
		{ID: "score-c", Capability: "SCORE", WorkerID: "shared-resident"},
	}
	plan := SwarmPlan{ID: "skipped-on-worker-cancel", MaxParallel: 8, Nodes: nodes}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	report, err := (SwarmRunner{Registry: registry}).Run(ctx, plan, "task", json.RawMessage(`{}`))

	if err == nil {
		t.Fatal("expected the cancelled run to surface an error")
	}
	if report.SkippedNodes == 0 {
		t.Fatalf("expected at least one node skipped waiting on the worker semaphore, report=%+v", report)
	}
	if report.ExecutedNodes+report.SkippedNodes != len(nodes) {
		t.Fatalf("executed(%d) + skipped(%d) != %d launched nodes", report.ExecutedNodes, report.SkippedNodes, len(nodes))
	}
	for _, node := range report.Nodes {
		if node.Skipped && node.WorkerID != "shared-resident" {
			t.Fatalf("skipped node %s has worker_id=%q, want the pinned worker attributed", node.NodeID, node.WorkerID)
		}
	}
}

func TestSwarmPlanNormalizeRejectsMalformedPlans(t *testing.T) {
	cases := []struct {
		name string
		plan SwarmPlan
		want string
	}{
		{
			name: "missing id",
			plan: SwarmPlan{Nodes: []SwarmNode{{ID: "a", Capability: "A"}}},
			want: "swarm id is required",
		},
		{
			name: "no nodes",
			plan: SwarmPlan{ID: "empty"},
			want: "at least one node",
		},
		{
			name: "duplicate node",
			plan: SwarmPlan{ID: "dup", Nodes: []SwarmNode{{ID: "a", Capability: "A"}, {ID: "a", Capability: "B"}}},
			want: "duplicate node",
		},
		{
			name: "unknown dependency",
			plan: SwarmPlan{ID: "ghost", Nodes: []SwarmNode{{ID: "a", Capability: "A", DependsOn: []string{"nowhere"}}}},
			want: "unknown node",
		},
		{
			name: "self dependency",
			plan: SwarmPlan{ID: "self", Nodes: []SwarmNode{{ID: "a", Capability: "A", DependsOn: []string{"a"}}}},
			want: "depends on itself",
		},
		{
			name: "node without capability",
			plan: SwarmPlan{ID: "bare", Nodes: []SwarmNode{{ID: "a"}}},
			want: "capability are required",
		},
		{
			name: "unexpected schema",
			plan: SwarmPlan{Schema: "tlaloc.something-else.r9", ID: "x", Nodes: []SwarmNode{{ID: "a", Capability: "A"}}},
			want: "unexpected swarm schema",
		},
		{
			name: "three node cycle",
			plan: SwarmPlan{ID: "ring", Nodes: []SwarmNode{
				{ID: "a", Capability: "A", DependsOn: []string{"c"}},
				{ID: "b", Capability: "B", DependsOn: []string{"a"}},
				{ID: "c", Capability: "C", DependsOn: []string{"b"}},
			}},
			want: "cycle",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.plan.Normalize()
			if err == nil {
				t.Fatalf("expected rejection for %s", testCase.name)
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error %q does not mention %q", err, testCase.want)
			}
		})
	}
}

func TestSwarmPlanNormalizeAppliesDefaults(t *testing.T) {
	plan, err := (SwarmPlan{ID: "defaults", Nodes: []SwarmNode{
		{ID: "a", Capability: " detect_intent ", ScopeHint: "general", DomainHint: "cfdi"},
	}}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if plan.Schema != SwarmSchemaR0 {
		t.Fatalf("schema=%s", plan.Schema)
	}
	if plan.MaxParallel != 1 {
		t.Fatalf("MaxParallel default=%d, want 1", plan.MaxParallel)
	}
	node := plan.Nodes[0]
	if node.Capability != "DETECT_INTENT" || node.ScopeHint != ScopeGeneral || node.DomainHint != "CFDI" {
		t.Fatalf("node not normalised: %+v", node)
	}
}

func TestSwarmRejectsInvalidTaskInput(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry, testWorker{desc: generalDescriptor("a", "A")})
	plan := SwarmPlan{ID: "input", Nodes: []SwarmNode{{ID: "a", Capability: "A"}}}
	runner := SwarmRunner{Registry: registry}
	for _, input := range []json.RawMessage{nil, json.RawMessage(`not json`)} {
		if _, err := runner.Run(context.Background(), plan, "task", input); err == nil {
			t.Fatalf("expected rejection for input %q", input)
		}
	}
}

func TestSwarmRequiresRegistry(t *testing.T) {
	plan := SwarmPlan{ID: "no-registry", Nodes: []SwarmNode{{ID: "a", Capability: "A"}}}
	if _, err := (SwarmRunner{}).Run(context.Background(), plan, "task", json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected an error when no registry is wired")
	}
}

// The report is the measurement instrument for the scaling experiment, so its
// counters must describe the run faithfully.
func TestSwarmReportDescribesTheRun(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry,
		testWorker{desc: generalDescriptor("intent", "DETECT_INTENT"), delay: 10 * time.Millisecond},
		testWorker{desc: generalDescriptor("entity", "EXTRACT_ENTITY"), delay: 10 * time.Millisecond},
		testWorker{desc: generalDescriptor("router", "ROUTE")},
		testWorker{desc: generalDescriptor("unused", "UNUSED")},
	)
	plan := SwarmPlan{ID: "metrics", MaxParallel: 2, Nodes: []SwarmNode{
		{ID: "intent", Capability: "DETECT_INTENT"},
		{ID: "entity", Capability: "EXTRACT_ENTITY"},
		{ID: "route", Capability: "ROUTE", DependsOn: []string{"intent", "entity"}},
	}}
	report, err := (SwarmRunner{Registry: registry}).Run(context.Background(), plan, "measured", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if report.Schema != SwarmSchemaR0+".report" {
		t.Fatalf("schema=%s", report.Schema)
	}
	if report.PlanID != "metrics" || report.TaskID != "measured" {
		t.Fatalf("identity=%s/%s", report.PlanID, report.TaskID)
	}
	if report.RegisteredWorkers != 4 {
		t.Fatalf("registered=%d, want every registered worker counted", report.RegisteredWorkers)
	}
	if report.ExecutedNodes != 3 {
		t.Fatalf("executed=%d", report.ExecutedNodes)
	}
	if report.MaxParallel != 2 {
		t.Fatalf("max_parallel=%d", report.MaxParallel)
	}
	if len(report.TerminalOutputs) != 1 {
		t.Fatalf("terminal outputs=%v, want only the sink node", report.TerminalOutputs)
	}
	if _, ok := report.TerminalOutputs["route"]; !ok {
		t.Fatalf("terminal outputs=%v, want route", report.TerminalOutputs)
	}
	for _, node := range report.Nodes {
		if node.WorkerID == "" {
			t.Fatalf("node %s has no attributed worker", node.NodeID)
		}
		if node.Confidence <= 0 {
			t.Fatalf("node %s has no confidence", node.NodeID)
		}
		if node.StartedAt.IsZero() {
			t.Fatalf("node %s has no start timestamp", node.NodeID)
		}
	}
	// Node order must be stable so repeated runs stay comparable.
	for index := 1; index < len(report.Nodes); index++ {
		if report.Nodes[index-1].NodeID > report.Nodes[index].NodeID {
			t.Fatalf("report nodes are not deterministically ordered: %v", report.Nodes)
		}
	}
}

func TestSwarmDefaultsTaskIDFromPlan(t *testing.T) {
	registry := NewRegistry()
	seen := ""
	worker := testWorker{desc: generalDescriptor("a", "A")}
	worker.fn = func(req CapabilityRequest) json.RawMessage {
		seen = req.TaskID
		return json.RawMessage(`{}`)
	}
	mustRegister(t, registry, worker)
	plan := SwarmPlan{ID: "plan-x", Nodes: []SwarmNode{{ID: "a", Capability: "A"}}}
	if _, err := (SwarmRunner{Registry: registry}).Run(context.Background(), plan, "  ", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	if seen != "plan-x-task" {
		t.Fatalf("task id=%q, want plan-derived default", seen)
	}
}
