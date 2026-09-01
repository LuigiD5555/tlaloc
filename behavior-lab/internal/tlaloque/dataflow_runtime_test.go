package tlaloque

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type blockingTestWorker struct {
	desc    CapabilityDescriptor
	blockOn <-chan struct{}
	fn      func(CapabilityRequest) json.RawMessage
}

func (w blockingTestWorker) Descriptor() CapabilityDescriptor { return w.desc }

func (w blockingTestWorker) Execute(ctx context.Context, req CapabilityRequest) (CapabilityResponse, error) {
	if w.blockOn != nil {
		select {
		case <-w.blockOn:
		case <-ctx.Done():
			return CapabilityResponse{}, ctx.Err()
		}
	}
	out := json.RawMessage(`{"ok":true}`)
	if w.fn != nil {
		out = w.fn(req)
	}
	return CapabilityResponse{WorkerID: w.desc.ID, Output: out, Confidence: .9}, nil
}

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
	releaseSlow := make(chan struct{})
	r := NewRegistry()
	fast := testWorker{
		desc: CapabilityDescriptor{ID: "fast", Capability: "FAST", Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "json", OutputSchema: "json", Deterministic: true},
		fn:   func(CapabilityRequest) json.RawMessage { return json.RawMessage(`{"fast":true}`) },
	}
	slow := blockingTestWorker{
		desc:    CapabilityDescriptor{ID: "slow", Capability: "SLOW", Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "json", OutputSchema: "json", Deterministic: true},
		blockOn: releaseSlow,
		fn:      func(CapabilityRequest) json.RawMessage { return json.RawMessage(`{"slow":true}`) },
	}
	join := testWorker{
		desc: CapabilityDescriptor{ID: "join", Capability: "JOIN", Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "json", OutputSchema: "json", Deterministic: true},
		fn: func(req CapabilityRequest) json.RawMessage {
			if len(req.Context["fast"]) == 0 {
				panic("ANY join missing first successful dependency")
			}
			if len(req.Context["slow"]) != 0 {
				panic("ANY join waited for the blocked sibling")
			}
			close(releaseSlow)
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	report, err := (SwarmRunner{Registry: r}).Run(ctx, plan, "any-task", json.RawMessage(`{"x":1}`))
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
	releaseThird := make(chan struct{})
	r := NewRegistry()
	mk := func(id string) testWorker {
		return testWorker{
			desc: CapabilityDescriptor{ID: id, Capability: id, Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "json", OutputSchema: "json", Deterministic: true},
			fn:   func(CapabilityRequest) json.RawMessage { return json.RawMessage(`{"ok":true}`) },
		}
	}
	third := blockingTestWorker{
		desc:    CapabilityDescriptor{ID: "C", Capability: "C", Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "json", OutputSchema: "json", Deterministic: true},
		blockOn: releaseThird,
	}
	fuse := testWorker{
		desc: CapabilityDescriptor{ID: "FUSE", Capability: "FUSE", Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "json", OutputSchema: "json", Deterministic: true},
		fn: func(req CapabilityRequest) json.RawMessage {
			if len(req.Context) != 2 {
				panic("QUORUM join should receive exactly the two completed dependencies")
			}
			if len(req.Context["c"]) != 0 {
				panic("QUORUM join waited for the blocked third dependency")
			}
			close(releaseThird)
			return json.RawMessage(`{"fused":true}`)
		},
	}
	for _, worker := range []CapabilityWorker{mk("A"), mk("B"), third, fuse} {
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	report, err := (SwarmRunner{Registry: r}).Run(ctx, plan, "quorum-task", json.RawMessage(`{"x":1}`))
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
