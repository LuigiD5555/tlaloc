package tonalt1arms

import (
	"context"
	"fmt"
)

// ArmBExecutor runs the real Arm B (Parrot-centric DAG) live execution
// path: it walks the shared ShapeDAG node-by-node, routing every capability
// through Bindings (built once, immutably, from the frozen
// T1_D5_ARM_B_POLICY.json via BuildArmBBindings) -- LOCATE_REGION/
// CROP_REGION/VERIFY execute deterministically in-process; EXTRACT_NUMBER/
// NORMALIZE/COMPARE_NUMBERS/ARITHMETIC all route to Adapter.Call. It never
// shortcuts the DAG with one generic call, and it never reads any
// gold-bearing artifact.
type ArmBExecutor struct {
	Bindings map[string]Binding // from BuildArmBBindings -- immutable, Arm-B-only
	Policy   *ArmBPolicy        // for per-capability prompt_template/temperature/max_tokens
	Adapter  ParrotAdapter
	Manifest *ImageManifest // for per-node EXTRACT_NUMBER image verification
}

// callOutcome bundles one adapter call's NodeCallRecord with its parsed
// numeric value (when successful) and the Blackboard NodeStatus it should
// result in -- one consistent shape used by every call site in this file.
type callOutcome struct {
	record NodeCallRecord
	value  float64
	status NodeStatus
}

// ExecuteWorkflow walks wf's shape DAG node by node, given the exact
// startup-verified operand image bytes (keyed workflowID+"|"+role, per
// StartupSweepResult.OperandImages -- correction I: never re-materialized
// mid-run). Every node writes one Blackboard.Record and, when it made (or
// attempted) an adapter call, one NodeCallRecord. A failed or blocked
// upstream node propagates BLOCKED_BY_DEPENDENCY to its dependents
// automatically via Blackboard.Record; those dependents never call Adapter.
func (e *ArmBExecutor) ExecuteWorkflow(ctx context.Context, runID string, wf Workflow, operandImages map[string][]byte) (WorkflowRecord, []NodeCallRecord, *Blackboard, error) {
	if e.Adapter == nil {
		return WorkflowRecord{}, nil, nil, fmt.Errorf("tonalt1arms: ArmBExecutor: nil Adapter")
	}
	dag, err := BuildShapeDAG(wf.Shape)
	if err != nil {
		return WorkflowRecord{}, nil, nil, err
	}

	bb := NewBlackboard(wf.WorkflowID)
	var nodeRecords []NodeCallRecord
	requestIndex := 0

	for _, step := range dag.Steps {
		binding, hasBinding := e.Bindings[step.Capability]

		switch {
		case step.Capability == "LOCATE_REGION" || step.Capability == "CROP_REGION":
			_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, ExecutorID: bindingExecutorID(binding, hasBinding), DependsOn: step.DependsOn, Status: NodeStatusDone})
			continue

		case step.Operation == OpVerify:
			e.executeDeterministicVerify(bb, dag, step)
			continue
		}

		if !hasBinding || !binding.UsesParrot {
			// Frozen policy says this capability should route to Parrot but
			// no binding/adapter exists -- fail closed rather than silently
			// falling back to a deterministic computation Arm B is not
			// supposed to perform.
			_ = bb.Record(NodeRecord{
				NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation,
				DependsOn: step.DependsOn, Status: NodeStatusFailedContract,
				Error: fmt.Sprintf("Arm B: capability %q has no Parrot binding", step.Capability),
			})
			continue
		}

		var outcome callOutcome
		if step.Operation == OpRead {
			outcome = e.callExtractNumber(ctx, runID, wf, step, operandImages, requestIndex)
		} else {
			outcome = e.callGenerativeNode(ctx, runID, wf, step, bb, requestIndex)
		}
		requestIndex++
		nodeRecords = append(nodeRecords, outcome.record)

		nodeToRecord := NodeRecord{
			NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation,
			ExecutorID: armBParrotExecutorID, DependsOn: step.DependsOn, Status: outcome.status, ModelCallDelta: 1,
		}
		if outcome.status == NodeStatusDone {
			nodeToRecord.Outputs = map[string]float64{step.OutputKey: outcome.value}
		} else {
			nodeToRecord.Error = outcome.record.ContractStatus
			if outcome.status == NodeStatusBlockedByDependency {
				nodeToRecord.ModelCallDelta = 0
			}
		}
		_ = bb.Record(nodeToRecord)
	}

	return e.buildWorkflowRecord(runID, wf, dag, bb, "B"), nodeRecords, bb, nil
}

// executeDeterministicVerify runs a VERIFY-operation node in-process: reads
// its single required upstream value from the Blackboard (fails closed via
// Require if missing/not DONE), records it, and -- for has_verify shapes --
// promotes it as the workflow's Fact.
func (e *ArmBExecutor) executeDeterministicVerify(bb *Blackboard, dag ShapeDAG, step dagStep) {
	if len(step.InputKeys) != 1 || len(step.DependsOn) == 0 {
		_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, Status: NodeStatusFailedContract, Error: "VERIFY node has unexpected shape"})
		return
	}
	upstreamRec, err := bb.Require(step.DependsOn[0])
	if err != nil {
		_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, Status: NodeStatusBlockedByDependency, Error: err.Error()})
		return
	}
	val, ok := upstreamRec.Outputs[step.InputKeys[0]]
	if !ok {
		_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, Status: NodeStatusFailedContract, Error: "VERIFY: upstream missing expected output key"})
		return
	}
	_ = bb.Record(NodeRecord{
		NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, ExecutorID: "arm-b-deterministic",
		DependsOn: step.DependsOn, Outputs: map[string]float64{step.OutputKey: val}, Status: NodeStatusDone,
	})
	if dag.HasVerify && step.LocalID == dag.TerminalNodeID {
		_ = bb.PromoteFinal(step.LocalID)
	}
}

// callExtractNumber verifies the target operand's image against the frozen
// manifest (fail closed before any call), then calls Adapter with the
// EXTRACT_NUMBER capability's frozen prompt/temperature/max_tokens.
func (e *ArmBExecutor) callExtractNumber(ctx context.Context, runID string, wf Workflow, step dagStep, operandImages map[string][]byte, requestIndex int) callOutcome {
	role := step.OutputKey
	key := wf.WorkflowID + "|" + role
	imageBytes := operandImages[key]

	rec := NodeCallRecord{RunID: runID, WorkflowID: wf.WorkflowID, Arm: "B", NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, RequestIndex: requestIndex, InputArtifact: key}

	if e.Manifest != nil {
		if err := VerifyOperandImage(e.Manifest, wf.WorkflowID, role, imageBytes); err != nil {
			rec.TransportStatus = "NOT_ATTEMPTED"
			rec.ContractStatus = "IMAGE_HASH_MISMATCH"
			return callOutcome{record: rec, status: NodeStatusFailedContract}
		}
	}

	_, promptTemplate, temperature, maxTokens := e.adapterSpecFor("EXTRACT_NUMBER")
	resp, err := e.Adapter.Call(ctx, ParrotRequest{Capability: "EXTRACT_NUMBER", Prompt: promptTemplate, Image: imageBytes, Temperature: temperature, MaxTokens: maxTokens})
	return finishGenerativeCall(rec, resp, err)
}

// callGenerativeNode calls Adapter for a non-visual generative capability
// (NORMALIZE/COMPARE_NUMBERS/ARITHMETIC). It resolves the node's numeric
// inputs from already-DONE upstream Blackboard nodes first -- a missing
// upstream fails this node closed as BLOCKED_BY_DEPENDENCY WITHOUT making
// any adapter call.
func (e *ArmBExecutor) callGenerativeNode(ctx context.Context, runID string, wf Workflow, step dagStep, bb *Blackboard, requestIndex int) callOutcome {
	rec := NodeCallRecord{RunID: runID, WorkflowID: wf.WorkflowID, Arm: "B", NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, RequestIndex: requestIndex}

	if _, err := resolveInputs(bb, step); err != nil {
		rec.TransportStatus = "NOT_ATTEMPTED"
		rec.ContractStatus = "BLOCKED_BY_DEPENDENCY"
		return callOutcome{record: rec, status: NodeStatusBlockedByDependency}
	}

	_, promptTemplate, temperature, maxTokens := e.adapterSpecFor(step.Capability)
	resp, callErr := e.Adapter.Call(ctx, ParrotRequest{Capability: step.Capability, Prompt: promptTemplate, Temperature: temperature, MaxTokens: maxTokens})
	return finishGenerativeCall(rec, resp, callErr)
}

// finishGenerativeCall classifies an adapter call's result into the
// transport/schema/contract statuses the record schema requires and the
// Blackboard NodeStatus it maps to.
func finishGenerativeCall(rec NodeCallRecord, resp ParrotResponse, err error) callOutcome {
	if err != nil {
		rec.TransportStatus = "FAILED"
		rec.ContractStatus = "TRANSPORT_FAILURE"
		return callOutcome{record: rec, status: NodeStatusFailedTransport}
	}
	rec.TransportStatus = "OK"
	rec.RawOutput = resp.RawOutput
	if !resp.ParsedOK {
		rec.SchemaStatus = "FAILED"
		rec.ContractStatus = resp.FailureCode
		return callOutcome{record: rec, status: NodeStatusFailedSchema}
	}
	rec.SchemaStatus = "OK"
	rec.ContractStatus = "OK"
	rec.ParsedOutput = fmt.Sprintf("%v", resp.ParsedValue)
	return callOutcome{record: rec, value: resp.ParsedValue, status: NodeStatusDone}
}

// adapterSpecFor looks up a capability's frozen AdapterSpec from
// T1_D5_ARM_B_POLICY.json, returning its prompt template, temperature, and
// max_tokens.
func (e *ArmBExecutor) adapterSpecFor(capability string) (AdapterSpec, string, float64, int) {
	if e.Policy == nil {
		return AdapterSpec{}, "", 0, 0
	}
	spec, ok := e.Policy.ParrotAdapters[capability]
	if !ok {
		return AdapterSpec{}, "", 0, 0
	}
	return spec, spec.PromptTemplate, float64(spec.Temperature), spec.MaxTokens
}

// resolveInputs fetches every one of step's InputKeys from already-DONE
// upstream Blackboard nodes, failing closed if any is missing.
func resolveInputs(bb *Blackboard, step dagStep) (map[string]float64, error) {
	inputs := make(map[string]float64, len(step.InputKeys))
	for _, dep := range step.DependsOn {
		depRec, err := bb.Require(dep)
		if err != nil {
			return nil, err
		}
		for k, v := range depRec.Outputs {
			inputs[k] = v
		}
	}
	for _, key := range step.InputKeys {
		if _, ok := inputs[key]; !ok {
			return nil, fmt.Errorf("tonalt1arms: resolveInputs: node %q missing input %q", step.LocalID, key)
		}
	}
	return inputs, nil
}

func bindingExecutorID(b Binding, has bool) string {
	if !has {
		return ""
	}
	return b.ExecutorID
}

// buildWorkflowRecord derives the workflow-level record from the completed
// Blackboard: TerminalOutput/TerminalStatus from the terminal node's own
// output (has_verify=false) or the promoted Fact (has_verify=true, per
// ShapeDAG.HasVerify), TotalParrotCalls summed from every node's
// ModelCallDelta.
func (e *ArmBExecutor) buildWorkflowRecord(runID string, wf Workflow, dag ShapeDAG, bb *Blackboard, arm string) WorkflowRecord {
	rec := WorkflowRecord{RunID: runID, WorkflowID: wf.WorkflowID, Family: wf.Shape, Arm: arm}
	for _, id := range bb.OrderedNodeIDs() {
		rec.TotalParrotCalls += bb.Nodes[id].ModelCallDelta
	}

	if dag.HasVerify {
		val, _, ok := bb.Promoted()
		if !ok {
			rec.TerminalStatus = "VERIFY_NOT_PROMOTED"
			rec.ContractStatus = "CONTRACT_FAILURE"
			return rec
		}
		rec.TerminalOutput = val
		rec.TerminalStatus = "SUCCESS"
		rec.ContractStatus = "OK"
		return rec
	}

	val := readTerminal(bb, dag)
	if isNaNFloat(val) {
		rec.TerminalStatus = "TERMINAL_NOT_COMPUTED"
		rec.ContractStatus = "CONTRACT_FAILURE"
		return rec
	}
	rec.TerminalOutput = val
	rec.TerminalStatus = "SUCCESS"
	rec.ContractStatus = "OK"
	return rec
}

func isNaNFloat(v float64) bool { return v != v }
