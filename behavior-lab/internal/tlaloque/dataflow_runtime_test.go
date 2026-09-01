package tlaloque

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"
)

func TestNodeStateMachineRejectsIllegalTransition(t *testing.T) {
	state, err := transitionNode(NodePending, NodeDependenciesSatisfied)
	if err != nil || state != NodeReady {
		t.Fatalf("pending -> ready: state=%s err=%v", state, err)
	}
	if _, err := transitionNode(NodeCompleted, NodeDispatched); err == nil {
		t.Fatal("expected terminal state transition to fail")
	}
}

func TestSwarmAnyJoinStartsOnFirstSuccessfulDependency(t *testing.T) {
	var slowDone atomic.Bool
	r := NewRegistry()
	fast := testWorker{
		desc: CapabilityDescriptor{ID: "fast", Capability: "FAST", Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "json", OutputSchema: "json", Deterministic: true},
		delay: 10 * time.Millisecond,
		fn: func(CapabilityRequest) json.RawMessage { return json.RawMessage(`{"fast":true}`) },
	}
	slow := testWorker{
		desc: CapabilityDescriptor{ID: "slow", Capability: "SLOW", Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "json", OutputSchema: "json", Deterministic: true},
		delay: 120 * time.Millisecond,
		fn: func(CapabilityRequest) json.RawMessage {
			slowDone.Store(true)
			return json.RawMessage(`{"slow":true}`)
		},
	}
	join := testWorker{
		desc: CapabilityDescriptor{ID: "join", Capability: "JOIN", Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "json", OutputSchema: "json", Deterministic: true},
		fn: func(req CapabilityRequest) json.RawMessage {
			if slowDone.Load() {
				panic("ANY join waited for the slow sibling")
			}
			if len(req.Context["fast"]) == 0 {
				panic("ANY join missing first successful dependency")
			}
			return json.RawMessage(`{"joined":true}`)
		},
	}
	for _, worker := range []CapabilityWorker{fast, slow, join} {
		if err := r.Register(worker); err != nil {
			t.Fatal(err)
		}
	}
	plan := SwarmPlan{ID: "any-join", MaxParallel: 3, Nodes: []SwarmNode{
		{ID: "fast", Capability: "FAST"},
		{ID: "slow", Capability: "SLOW"},
		{ID: "join", Capability: "JOIN", DependsOn: []string{"fast", "slow"}, JoinMode: string(JoinAny)},
	}}
	report, err := (SwarmRunner{Registry: r}).Run(context.Background(), plan, "any-task", json.RawMessage(`{"x":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if !report.Succeeded {
		t.Fatalf("report=%+v", report)
	}
	for _, node := range report.Nodes {
		if node.State != NodeCompleted {
			t.Fatalf("node %s state=%s", node.NodeID, node.State)
		}
	}
}

func TestSwarmQuorumJoinStartsAfterThreshold(t *testing.T) {
	var slowDone atomic.Bool
	r := NewRegistry()
	mk := func(id string, delay time.Duration, done *atomic.Bool) testWorker {
		return testWorker{
			desc: CapabilityDescriptor{ID: id, Capability: id, Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "json", OutputSchema: "json", Deterministic: true},
			delay: delay,
			fn: func(CapabilityRequest) json.RawMessage {
				if done != nil {
					done.Store(true)
				}
				return json.RawMessage(`{"ok":true}`)
			},
		}
	}
	for _, worker := range []CapabilityWorker{
		mk("A", 10*time.Millisecond, nil),
		mk("B", 20*time.Millisecond, nil),
		mk("C", 150*time.Millisecond, &slowDone),
		testWorker{
			desc: CapabilityDescriptor{ID: "FUSE", Capability: "FUSE", Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "json", OutputSchema: "json", Deterministic: true},
			fn: func(req CapabilityRequest) json.RawMessage {
				if slowDone.Load() {
					panic("QUORUM join waited for all dependencies")
				}
				if len(req.Context) != 2 {
					panic("QUORUM join should receive exactly the two completed dependencies")
				}
				return json.RawMessage(`{"fused":true}`)
			},
		},
	} {
		if err := r.Register(worker); err != nil {
			t.Fatal(err)
		}
	}
	plan := SwarmPlan{ID: "quorum-join", MaxParallel: 4, Nodes: []SwarmNode{
		{ID: "a", Capability: "A"},
		{ID: "b", Capability: "B"},
		{ID: "c", Capability: "C"},
		{ID: "fuse", Capability: "FUSE", DependsOn: []string{"a", "b", "c"}, JoinMode: string(JoinQuorum), MinDependencies: 2},
	}}
	report, err := (SwarmRunner{Registry: r}).Run(context.Background(), plan, "quorum-task", json.RawMessage(`{"x":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if !report.Succeeded {
		t.Fatalf("report=%+v", report)
	}
}

func TestSwarmPlanRejectsInvalidQuorum(t *testing.T) {
	_, err := (SwarmPlan{ID: "bad-quorum", Nodes: []SwarmNode{
		{ID: "a", Capability: "A"},
		{ID: "b", Capability: "B", DependsOn: []string{"a"}, JoinMode: string(JoinQuorum), MinDependencies: 2},
	}}).Normalize()
	if err == nil {
		t.Fatal("expected invalid quorum")
	}
}
