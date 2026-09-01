package tlaloque

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type controlledEnsembleWorker struct {
	desc CapabilityDescriptor
	run  func(context.Context, CapabilityRequest) (CapabilityResponse, error)
}

func (w controlledEnsembleWorker) Descriptor() CapabilityDescriptor { return w.desc }
func (w controlledEnsembleWorker) Execute(ctx context.Context, req CapabilityRequest) (CapabilityResponse, error) {
	if w.run != nil {
		return w.run(ctx, req)
	}
	return CapabilityResponse{WorkerID: w.desc.ID, Output: json.RawMessage(`{"ok":true}`), Confidence: 0.9}, nil
}

func ensembleDescriptor(id, capability string) CapabilityDescriptor {
	return CapabilityDescriptor{
		ID:             id,
		Capability:     capability,
		Scope:          ScopeGeneral,
		Engine:         EngineModel,
		InputSchema:    "json",
		OutputSchema:   "json",
		ParameterCount: 1_000_000,
	}
}

func TestResolveEnsembleBuildsPinnedQuorumPlan(t *testing.T) {
	r := NewRegistry()
	for _, id := range []string{"m1", "m2", "m3"} {
		if err := r.Register(controlledEnsembleWorker{desc: ensembleDescriptor(id, "CLASSIFY")}); err != nil {
			t.Fatal(err)
		}
	}
	fuseDesc := ensembleDescriptor("fuse", "FUSE")
	fuseDesc.Deterministic = true
	fuseDesc.Engine = EngineDeterministic
	if err := r.Register(controlledEnsembleWorker{desc: fuseDesc}); err != nil {
		t.Fatal(err)
	}

	planned, err := r.ResolveEnsemble(EnsembleGoal{
		ID:               "classification-ensemble",
		Capability:       "CLASSIFY",
		Members:          3,
		FusionCapability: "FUSE",
		JoinMode:         string(JoinQuorum),
		MinMembers:       2,
	}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(planned.Members) != 3 || len(planned.Plan.Nodes) != 4 {
		t.Fatalf("members=%d nodes=%d", len(planned.Members), len(planned.Plan.Nodes))
	}
	var fusion *SwarmNode
	memberCount := 0
	for i := range planned.Plan.Nodes {
		node := &planned.Plan.Nodes[i]
		if node.ID == "ensemble-fusion" {
			fusion = node
			continue
		}
		memberCount++
		if node.FailurePolicy != string(FailureTolerated) {
			t.Fatalf("member %s failure_policy=%q", node.ID, node.FailurePolicy)
		}
	}
	if memberCount != 3 || fusion == nil {
		t.Fatalf("memberCount=%d fusion=%v", memberCount, fusion)
	}
	if fusion.JoinMode != string(JoinQuorum) || fusion.MinDependencies != 2 {
		t.Fatalf("fusion join=%s min=%d", fusion.JoinMode, fusion.MinDependencies)
	}
	if len(fusion.DependsOn) != 3 || len(fusion.InputBindings) != 3 {
		t.Fatalf("fusion deps=%v bindings=%v", fusion.DependsOn, fusion.InputBindings)
	}
}

func TestQuorumEnsembleSucceedsWhenOptionalMemberFails(t *testing.T) {
	r := NewRegistry()
	releaseSlow := make(chan struct{})
	var releaseOnce sync.Once

	for _, id := range []string{"m1", "m2"} {
		id := id
		worker := controlledEnsembleWorker{
			desc: ensembleDescriptor(id, "CLASSIFY"),
			run: func(context.Context, CapabilityRequest) (CapabilityResponse, error) {
				return CapabilityResponse{WorkerID: id, Output: json.RawMessage(`{"class":"A"}`), Confidence: 0.9}, nil
			},
		}
		if err := r.Register(worker); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Register(controlledEnsembleWorker{
		desc: ensembleDescriptor("m3", "CLASSIFY"),
		run: func(ctx context.Context, _ CapabilityRequest) (CapabilityResponse, error) {
			select {
			case <-releaseSlow:
				return CapabilityResponse{}, fmt.Errorf("synthetic member failure")
			case <-ctx.Done():
				return CapabilityResponse{}, ctx.Err()
			}
		},
	}); err != nil {
		t.Fatal(err)
	}

	fuseDesc := ensembleDescriptor("fuse", "FUSE")
	fuseDesc.Deterministic = true
	fuseDesc.Engine = EngineDeterministic
	if err := r.Register(controlledEnsembleWorker{
		desc: fuseDesc,
		run: func(_ context.Context, req CapabilityRequest) (CapabilityResponse, error) {
			members := EnsembleMemberInputs(req.Context, "")
			if len(members) != 2 {
				return CapabilityResponse{}, fmt.Errorf("fusion saw %d successful members, want 2", len(members))
			}
			releaseOnce.Do(func() { close(releaseSlow) })
			return CapabilityResponse{WorkerID: "fuse", Output: json.RawMessage(`{"class":"A","votes":2}`), Confidence: 1}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	planned, err := r.ResolveEnsemble(EnsembleGoal{
		ID:               "quorum-runtime",
		Capability:       "CLASSIFY",
		Members:          3,
		FusionCapability: "FUSE",
		JoinMode:         string(JoinQuorum),
		MinMembers:       2,
	}, 3)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	report, runErr := (SwarmRunner{Registry: r}).Run(ctx, planned.Plan, "ensemble-task", json.RawMessage(`{"text":"x"}`))
	if runErr != nil {
		t.Fatal(runErr)
	}
	if !report.Succeeded {
		t.Fatalf("report should succeed with quorum: %+v", report)
	}
	if len(report.TerminalOutputs["ensemble-fusion"]) == 0 {
		t.Fatalf("missing fusion output: %+v", report.TerminalOutputs)
	}
	failedTolerated := false
	for _, node := range report.Nodes {
		if node.WorkerID == "m3" && node.State == NodeFailed && node.FailureTolerated {
			failedTolerated = true
		}
	}
	if !failedTolerated {
		t.Fatalf("failed member was not reported as tolerated: %+v", report.Nodes)
	}
}

func TestAllEnsembleFailsAtFusionBoundaryWhenMemberFails(t *testing.T) {
	r := NewRegistry()
	for _, id := range []string{"m1", "m2"} {
		id := id
		if err := r.Register(controlledEnsembleWorker{
			desc: ensembleDescriptor(id, "CLASSIFY"),
			run: func(context.Context, CapabilityRequest) (CapabilityResponse, error) {
				return CapabilityResponse{WorkerID: id, Output: json.RawMessage(`{"ok":true}`)}, nil
			},
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Register(controlledEnsembleWorker{
		desc: ensembleDescriptor("m3", "CLASSIFY"),
		run: func(context.Context, CapabilityRequest) (CapabilityResponse, error) {
			return CapabilityResponse{}, fmt.Errorf("synthetic member failure")
		},
	}); err != nil {
		t.Fatal(err)
	}
	var fusionCalled atomic.Bool
	if err := r.Register(controlledEnsembleWorker{
		desc: ensembleDescriptor("fuse", "FUSE"),
		run: func(context.Context, CapabilityRequest) (CapabilityResponse, error) {
			fusionCalled.Store(true)
			return CapabilityResponse{WorkerID: "fuse", Output: json.RawMessage(`{"ok":true}`)}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	planned, err := r.ResolveEnsemble(EnsembleGoal{
		ID:               "all-runtime",
		Capability:       "CLASSIFY",
		Members:          3,
		FusionCapability: "FUSE",
		JoinMode:         string(JoinAll),
	}, 3)
	if err != nil {
		t.Fatal(err)
	}
	report, runErr := (SwarmRunner{Registry: r}).Run(context.Background(), planned.Plan, "ensemble-task", json.RawMessage(`{"text":"x"}`))
	if runErr == nil {
		t.Fatal("ALL ensemble should fail when one member fails")
	}
	if report.Succeeded || fusionCalled.Load() {
		t.Fatalf("report=%+v fusionCalled=%v", report, fusionCalled.Load())
	}
}

func TestFailurePolicyValidationAndRunSatisfaction(t *testing.T) {
	if _, err := (SwarmPlan{ID: "bad-failure-policy", Nodes: []SwarmNode{{
		ID:            "a",
		Capability:    "A",
		FailurePolicy: "MAYBE",
	}}}).Normalize(); err == nil {
		t.Fatal("expected invalid failure policy")
	}
	node := SwarmNode{FailurePolicy: string(FailureTolerated)}
	if !nodeStateSatisfiesRun(node, NodeFailed) || !nodeStateSatisfiesRun(node, NodeBlocked) {
		t.Fatal("tolerated failed/blocked nodes should satisfy run-level completion")
	}
	if nodeStateSatisfiesRun(SwarmNode{}, NodeFailed) {
		t.Fatal("strict failed node must not satisfy run-level completion")
	}
}

func TestResolveEnsembleRequiresRequestedMemberCount(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(controlledEnsembleWorker{desc: ensembleDescriptor("only", "CLASSIFY")}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(controlledEnsembleWorker{desc: ensembleDescriptor("fuse", "FUSE")}); err != nil {
		t.Fatal(err)
	}
	_, err := r.ResolveEnsemble(EnsembleGoal{Capability: "CLASSIFY", Members: 2, FusionCapability: "FUSE"}, 2)
	if err == nil {
		t.Fatal("expected insufficient member count to fail planning")
	}
}
