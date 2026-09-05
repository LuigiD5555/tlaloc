package tonalt1arms

import (
	"context"
	"fmt"
)

// ArmCExecutor runs the real Arm C (heterogeneous Tonal) live execution
// path: it walks the SAME shared ShapeDAG as Arm B, but routes through
// Bindings built from the frozen T1_D5_ARM_C_POLICY.json via
// BuildArmCBindings -- only EXTRACT_NUMBER calls Adapter; every other
// generative capability (NORMALIZE/COMPARE_NUMBERS/ARITHMETIC) executes
// in-process via the node's Operation function from v2semantics.go. Arm C's
// arithmetic is NEVER implemented by calling or mirroring the historical v1
// ComputeGold -- each node's Operation function is the same one Arm B's
// Parrot adapter would otherwise be asked to compute, executed
// deterministically node-by-node here instead (T1_V2 semantics throughout).
type ArmCExecutor struct {
	Bindings map[string]Binding // from BuildArmCBindings -- immutable, Arm-C-only
	Adapter  ParrotAdapter
	Manifest *ImageManifest
}

// ExecuteWorkflow mirrors ArmBExecutor.ExecuteWorkflow's shape/record
// schema, but every non-EXTRACT_NUMBER generative node computes its own
// value via the shared v2 Operation function rather than an adapter call.
func (e *ArmCExecutor) ExecuteWorkflow(ctx context.Context, runID string, wf Workflow, operandImages map[string][]byte) (WorkflowRecord, []NodeCallRecord, *Blackboard, error) {
	if e.Adapter == nil {
		return WorkflowRecord{}, nil, nil, fmt.Errorf("tonalt1arms: ArmCExecutor: nil Adapter")
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

		case step.Operation == OpThresholdCheck:
			// Same side-observation semantics as Arm B's threshold check,
			// but Arm C's policy routes COMPARE_NUMBERS to
			// deterministic_preferred -- executed in-process, no call.
			e.executeDeterministicThresholdCheck(bb, step)
			continue
		}

		if hasBinding && binding.UsesParrot && step.Operation == OpRead {
			outcome := e.callExtractNumber(ctx, runID, wf, step, operandImages, requestIndex)
			requestIndex++
			nodeRecords = append(nodeRecords, outcome.record)
			e.recordCallOutcome(bb, step, outcome)
			continue
		}

		// Every other generative node (NORMALIZE/COMPARE_NUMBERS/ARITHMETIC,
		// per the frozen Arm-C policy: deterministic_preferred/required) is
		// computed in-process via the shared v2 Operation function -- never
		// an adapter call, never ComputeGold.
		e.executeDeterministicOperation(bb, step)
	}

	return e.buildWorkflowRecord(runID, wf, dag, bb, "C"), nodeRecords, bb, nil
}

// executeDeterministicVerify mirrors ArmBExecutor's VERIFY handling exactly
// (frozen policy: VERIFY is deterministic_required in both arms).
func (e *ArmCExecutor) executeDeterministicVerify(bb *Blackboard, dag ShapeDAG, step dagStep) {
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
		NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, ExecutorID: armCDeterministicExecutor,
		DependsOn: step.DependsOn, Outputs: map[string]float64{step.OutputKey: val}, Status: NodeStatusDone,
	})
	if dag.HasVerify && step.LocalID == dag.TerminalNodeID {
		_ = bb.PromoteFinal(step.LocalID)
	}
}

func (e *ArmCExecutor) executeDeterministicThresholdCheck(bb *Blackboard, step dagStep) {
	if len(step.DependsOn) == 0 {
		_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, Status: NodeStatusFailedContract, Error: "THRESHOLD_CHECK node has no dependency"})
		return
	}
	if _, err := bb.Require(step.DependsOn[0]); err != nil {
		_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, Status: NodeStatusBlockedByDependency, Error: err.Error()})
		return
	}
	_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, ExecutorID: armCDeterministicExecutor, DependsOn: step.DependsOn, Status: NodeStatusDone})
}

// executeDeterministicOperation resolves step's inputs from already-DONE
// upstream Blackboard nodes and computes its output via the shared
// v2semantics.go Operation function -- the exact same function Arm B would
// otherwise ask Parrot to approximate, run here deterministically instead.
func (e *ArmCExecutor) executeDeterministicOperation(bb *Blackboard, step dagStep) {
	values, err := resolveInputs(bb, step)
	if err != nil {
		_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, Status: NodeStatusBlockedByDependency, Error: err.Error()})
		return
	}

	rec := NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, ExecutorID: armCDeterministicExecutor, DependsOn: step.DependsOn}

	switch step.Operation {
	case OpNormalize:
		rec.Outputs = map[string]float64{step.OutputKey: opNormalize(values[step.InputKeys[0]])}
	case OpMax:
		a, b := values[step.InputKeys[0]], values[step.InputKeys[1]]
		rec.Outputs = map[string]float64{step.OutputKey: opMax(a, b)}
		rec.OutputVerdict = opCompareNumbers(a, b)
	case OpSubtract:
		rec.Outputs = map[string]float64{step.OutputKey: opSubtract(values[step.InputKeys[0]], values[step.InputKeys[1]])}
	case OpDivide:
		result, divErr := opDivide(values[step.InputKeys[0]], values[step.InputKeys[1]])
		if divErr != nil {
			rec.Status = NodeStatusFailedContract
			rec.Error = "INVALID_INPUT_DENOMINATOR_ZERO"
			_ = bb.Record(rec)
			return
		}
		rec.Outputs = map[string]float64{step.OutputKey: result}
	case OpPercentDifference:
		rec.Outputs = map[string]float64{step.OutputKey: opPercentDifference(values[step.InputKeys[0]], values[step.InputKeys[1]])}
	case OpPercentToFraction:
		rec.Outputs = map[string]float64{step.OutputKey: opPercentToFraction(values[step.InputKeys[0]])}
	case OpSubtractTolerance:
		rec.Outputs = map[string]float64{step.OutputKey: opSubtractTolerance(values[step.InputKeys[0]])}
	case OpCompareZero:
		rec.OutputVerdict = opCompareZero(values[step.InputKeys[0]])
	default:
		rec.Status = NodeStatusFailedContract
		rec.Error = fmt.Sprintf("Arm C: unrecognized deterministic Operation %q", step.Operation)
		_ = bb.Record(rec)
		return
	}
	rec.Status = NodeStatusDone
	_ = bb.Record(rec)
}

// callExtractNumber is identical in shape to ArmBExecutor's, since
// EXTRACT_NUMBER is parrot_required in both arms' frozen policies.
func (e *ArmCExecutor) callExtractNumber(ctx context.Context, runID string, wf Workflow, step dagStep, operandImages map[string][]byte, requestIndex int) callOutcome {
	role := step.OutputKey
	key := wf.WorkflowID + "|" + role
	imageBytes := operandImages[key]

	rec := NodeCallRecord{RunID: runID, WorkflowID: wf.WorkflowID, Arm: "C", NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, RequestIndex: requestIndex, InputArtifact: key}

	if e.Manifest != nil {
		if err := VerifyOperandImage(e.Manifest, wf.WorkflowID, role, imageBytes); err != nil {
			rec.TransportStatus = "NOT_ATTEMPTED"
			rec.ContractStatus = "IMAGE_HASH_MISMATCH"
			return callOutcome{record: rec, status: NodeStatusFailedContract}
		}
	}

	resp, err := e.Adapter.Call(ctx, ParrotRequest{Capability: "EXTRACT_NUMBER", Image: imageBytes})
	return finishGenerativeCall(rec, resp, err)
}

func (e *ArmCExecutor) recordCallOutcome(bb *Blackboard, step dagStep, outcome callOutcome) {
	nodeToRecord := NodeRecord{
		NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation,
		ExecutorID: armCParrotExecutorID, DependsOn: step.DependsOn, Status: outcome.status, ModelCallDelta: 1,
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

// buildWorkflowRecord is identical in shape to ArmBExecutor's.
func (e *ArmCExecutor) buildWorkflowRecord(runID string, wf Workflow, dag ShapeDAG, bb *Blackboard, arm string) WorkflowRecord {
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
