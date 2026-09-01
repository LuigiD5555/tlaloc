package lfm2boundary

import (
	"context"
	"fmt"
	"time"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// PooledProcessWorker represents one isolated specialist slot. The swarm pins
// nodes round-robin across these slots; each invocation chooses the role from
// the question and then delegates to the existing PROCESS transport.
type PooledProcessWorker struct {
	ID      string
	Binary  string
	Timeout time.Duration
}

func (w PooledProcessWorker) Descriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{ID:w.ID, Capability:VisualCapability, Scope:tlaloque.ScopeGeneral, Engine:tlaloque.EngineProcess, InputSchema:"tlaloc.lfm2-boundary.r0.visual-task", OutputSchema:"tlaloc.lfm2-boundary.r0.visual-output", Deterministic:false, ParameterCount:1_600_000_000, MaxConcurrency:1, Tags:[]string{"lfm2-vl","vision","isolated-process"}}
}

func (w PooledProcessWorker) Execute(ctx context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	qid := QuestionIDFromNode(req.NodeID)
	role, ok := ResponsibilityForQuestion(qid)
	if !ok { return tlaloque.CapabilityResponse{}, fmt.Errorf("cannot map node %q to a specialist responsibility", req.NodeID) }
	return (tlaloque.ProcessWorker{Desc:w.Descriptor(), Command:[]string{w.Binary, role}, Timeout:w.Timeout}).Execute(ctx, req)
}

func ConsolidatorWorker(binary string, timeout time.Duration) tlaloque.ProcessWorker {
	return tlaloque.ProcessWorker{Desc:tlaloque.CapabilityDescriptor{ID:"lfm2-boundary-consolidator", Capability:ConsolidateCapability, Scope:tlaloque.ScopeGeneral, Engine:tlaloque.EngineDeterministic, InputSchema:"tlaloc.lfm2-boundary.r0.visual-task", OutputSchema:"tlaloc.lfm2-boundary.r0.consolidated", Deterministic:true, MaxConcurrency:1, Tags:[]string{"blackboard-decision","deterministic"}}, Command:[]string{binary,"consolidate"}, Timeout:timeout}
}
