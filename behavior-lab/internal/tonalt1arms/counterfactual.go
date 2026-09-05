package tonalt1arms

import (
	"fmt"
	"math"
)

// HISTORICAL/REFERENCE CODE (v1): ArmCState, NewArmCState, RunPoison,
// RunRemove below are preserved UNMODIFIED as a semantic reference/oracle
// (per task instruction) and remain covered by their existing tests. They
// call the historical v1 ComputeGold (A-B for COMPARE_TWO_VALUES) and are
// NOT used by the live T1_V2 runtime. The final scientific counterfactual
// implementation is RunPoisonOnBlackboard/RunRemoveOnBlackboard, further
// down this file, which operates on a completed v2 Arm-C *Blackboard* (not
// ArmCState) and replays via the shared ShapeDAG's Operation functions
// (v2semantics.go) -- never ComputeGold, never the avg/sub_B-style
// intermediate formulas, only T1_V2 semantics throughout.

// ArmCState is a completed Arm-C Blackboard snapshot for one workflow: the
// post-extraction role -> numeric-value map produced by EXTRACT_NUMBER
// calls, plus the deterministic terminal result computed from it. It is the
// "completed Arm-C Blackboard/state" the frozen counterfactual scope
// (ARM_C_ONLY) operates on: extraction has already happened; only the
// deterministic downstream recomputation is replayed by the runner below.
type ArmCState struct {
	WorkflowID    string
	Shape         string
	OperandValues map[string]float64 // role -> extracted numeric value (post-EXTRACT_NUMBER)
	FinalValue    float64
	FinalStatus   string
}

// NewArmCState builds a completed Arm-C state from a frozen Workflow's own
// operand values (i.e. as if EXTRACT_NUMBER had already run and recovered
// each operand's gold value exactly) and its deterministic terminal result.
// This is the offline/synthetic construction used for counterfactual
// testing without any model call; a live run would populate OperandValues
// from actual EXTRACT_NUMBER responses instead.
func NewArmCState(wf Workflow) (ArmCState, error) {
	values := make(map[string]float64, len(wf.Operands))
	for _, op := range wf.Operands {
		values[op.Role] = op.NumericValue
	}
	final, status, err := ComputeGold(wf)
	if err != nil {
		return ArmCState{}, fmt.Errorf("tonalt1arms: NewArmCState: %w", err)
	}
	return ArmCState{
		WorkflowID:    wf.WorkflowID,
		Shape:         wf.Shape,
		OperandValues: values,
		FinalValue:    final,
		FinalStatus:   status,
	}, nil
}

// clone deep-copies the state so a counterfactual mutation never touches
// the original.
func (s ArmCState) clone() ArmCState {
	values := make(map[string]float64, len(s.OperandValues))
	for k, v := range s.OperandValues {
		values[k] = v
	}
	return ArmCState{
		WorkflowID:    s.WorkflowID,
		Shape:         s.Shape,
		OperandValues: values,
		FinalValue:    s.FinalValue,
		FinalStatus:   s.FinalStatus,
	}
}

// CounterfactualOutcome is the full trace of one POISON or REMOVE trial:
// what changed, what was recomputed, and the resulting terminal state. No
// field here is ever populated by a model call.
type CounterfactualOutcome struct {
	Operation       string // "POISON" | "REMOVE"
	WorkflowID      string
	TargetRole      string
	OriginalValue   float64
	OriginalPresent bool
	MutatedValue    float64 // POISON only; zero value for REMOVE
	OriginalFinal   float64
	OriginalStatus  string
	ResultingFinal  float64
	ResultingStatus string
	TerminalChanged bool
	FailedClosed    bool   // true if REMOVE hit a missing required operand and safely failed
	FailureReason   string // set when FailedClosed
	ModelCallCount  int    // must always be 0
	AffectedNodes   []string
	DependentReplay bool
}

// requiredRoleModel is a stand-in for "compute deterministic descendants"
// affecting only the roles Workflow.ComputeGold actually reads for each
// shape. Every shape's ComputeGold reads all of its declared operand roles
// to produce FinalValue, so for T1's frozen shapes the dependency closure of
// the single terminal node is always "all operand roles of this workflow" --
// there is no shape where a role is acquired but not causally required by
// the terminal value. This is recorded explicitly rather than assumed.
func requiredRoles(shape string) ([]string, error) {
	switch shape {
	case "READ_AND_CHECK":
		return []string{"A"}, nil
	case "COMPARE_TWO_VALUES", "DIFFERENCE_THEN_VERIFY":
		return []string{"A", "B"}, nil
	case "RATIO_OF_DIFFERENCE":
		return []string{"A", "B", "C"}, nil
	case "RECONCILIATION_CHAIN":
		return []string{"A", "a", "B", "b"}, nil
	default:
		return nil, fmt.Errorf("tonalt1arms: requiredRoles: unknown shape %q", shape)
	}
}

// RunPoison replaces the completed state's stored observation for
// targetRole with poisonValue, then recomputes the deterministic terminal
// result from the mutated operand map. It never invokes EXTRACT_NUMBER or
// any model transport -- ModelCallCount is always 0. The original state is
// never mutated; RunPoison operates on a clone.
func RunPoison(completed ArmCState, targetRole string, poisonValue float64) (CounterfactualOutcome, error) {
	roles, err := requiredRoles(completed.Shape)
	if err != nil {
		return CounterfactualOutcome{}, err
	}
	required := false
	for _, r := range roles {
		if r == targetRole {
			required = true
			break
		}
	}
	if !required {
		return CounterfactualOutcome{}, fmt.Errorf("tonalt1arms: RunPoison: role %q is not a required operand of shape %s", targetRole, completed.Shape)
	}

	mutated := completed.clone()
	original, present := mutated.OperandValues[targetRole]
	mutated.OperandValues[targetRole] = poisonValue

	wf, err := stateToWorkflow(mutated)
	if err != nil {
		return CounterfactualOutcome{}, err
	}
	newFinal, newStatus, err := ComputeGold(wf)
	if err != nil {
		return CounterfactualOutcome{}, err
	}

	return CounterfactualOutcome{
		Operation:       "POISON",
		WorkflowID:      completed.WorkflowID,
		TargetRole:      targetRole,
		OriginalValue:   original,
		OriginalPresent: present,
		MutatedValue:    poisonValue,
		OriginalFinal:   completed.FinalValue,
		OriginalStatus:  completed.FinalStatus,
		ResultingFinal:  newFinal,
		ResultingStatus: newStatus,
		TerminalChanged: !valuesEqual(newFinal, completed.FinalValue) || newStatus != completed.FinalStatus,
		FailedClosed:    false,
		ModelCallCount:  0,
		AffectedNodes:   roles,
		DependentReplay: true,
	}, nil
}

// RunRemove removes the completed state's stored observation for
// targetRole entirely, then attempts to recompute the deterministic
// terminal result. Because every frozen T1 shape's terminal value causally
// requires all of its declared operand roles (requiredRoles), removing any
// one of them makes deterministic recomputation impossible -- RunRemove
// fails closed (FailedClosed=true) rather than substituting a default or
// silently proceeding, exactly the frozen "missing-state/safe-failure"
// behavior. It never invokes EXTRACT_NUMBER or any model transport --
// ModelCallCount is always 0. The original state is never mutated.
func RunRemove(completed ArmCState, targetRole string) (CounterfactualOutcome, error) {
	roles, err := requiredRoles(completed.Shape)
	if err != nil {
		return CounterfactualOutcome{}, err
	}
	required := false
	for _, r := range roles {
		if r == targetRole {
			required = true
			break
		}
	}
	if !required {
		return CounterfactualOutcome{}, fmt.Errorf("tonalt1arms: RunRemove: role %q is not a required operand of shape %s", targetRole, completed.Shape)
	}

	mutated := completed.clone()
	original, present := mutated.OperandValues[targetRole]
	delete(mutated.OperandValues, targetRole)

	wf, err := stateToWorkflow(mutated)
	if err != nil {
		return CounterfactualOutcome{}, err
	}
	// ComputeGold requires every declared operand to have a role present;
	// stateToWorkflow will omit the removed role's Operand entirely, which
	// ComputeGold's own "missing role X" checks reject. This is the fail-
	// closed path: no fallback value is substituted, no Parrot call is made
	// to re-derive it, and the failure is recorded rather than silently
	// swallowed.
	_, _, err = ComputeGold(wf)
	if err == nil {
		return CounterfactualOutcome{}, fmt.Errorf("tonalt1arms: RunRemove: expected fail-closed behavior removing required role %q, but ComputeGold succeeded -- this shape does not actually require this role, which contradicts requiredRoles", targetRole)
	}

	return CounterfactualOutcome{
		Operation:       "REMOVE",
		WorkflowID:      completed.WorkflowID,
		TargetRole:      targetRole,
		OriginalValue:   original,
		OriginalPresent: present,
		OriginalFinal:   completed.FinalValue,
		OriginalStatus:  completed.FinalStatus,
		ResultingStatus: "UNKNOWN",
		TerminalChanged: true,
		FailedClosed:    true,
		FailureReason:   err.Error(),
		ModelCallCount:  0,
		AffectedNodes:   roles,
		DependentReplay: true,
	}, nil
}

// stateToWorkflow reconstructs a minimal Workflow from a (possibly mutated)
// ArmCState so the existing frozen ComputeGold can be reused verbatim as
// the deterministic-descendant replay engine, rather than re-implementing
// per-shape arithmetic a second time.
func stateToWorkflow(s ArmCState) (Workflow, error) {
	wf := Workflow{WorkflowID: s.WorkflowID, Shape: s.Shape}
	for role, value := range s.OperandValues {
		wf.Operands = append(wf.Operands, Operand{Role: role, NumericValue: value})
	}
	return wf, nil
}

func valuesEqual(a, b float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.IsNaN(a) && math.IsNaN(b)
	}
	return a == b
}

// --- V2 Blackboard counterfactual implementation (the real, scientific
// implementation; see the historical-code note at the top of this file) ---

// BlackboardCounterfactualOutcome is RunPoisonOnBlackboard/
// RunRemoveOnBlackboard's result: identical in spirit to
// CounterfactualOutcome above, but computed entirely from a v2 Blackboard
// and the shared ShapeDAG's Operation functions -- never ComputeGold.
type BlackboardCounterfactualOutcome struct {
	Operation                     string // "POISON" | "REMOVE"
	WorkflowID                    string
	TargetNodeID                  string
	OriginalValue                 float64
	MutatedValue                  float64 // POISON only
	OriginalFinal                 float64
	ResultingFinal                float64
	TerminalChanged               bool
	FailedClosed                  bool
	FailureReason                 string
	PrimaryObservationUnavailable bool // true iff the target's primary EXTRACT_NUMBER observation was never DONE in the input Blackboard
	ModelCallCount                int  // must always be 0
	ReplayedNodeIDs               []string
	SemanticsVersion              string
}

// requirePrimaryObservation checks that targetNodeID (an EXTRACT_NUMBER/READ
// node) exists and is DONE in bb. Per task correction D, if it is not, the
// runner does not manufacture a value from gold -- it reports
// PRIMARY_OBSERVATION_UNAVAILABLE and makes zero model calls to fetch one.
func requirePrimaryObservation(bb *Blackboard, targetNodeID string) (*NodeRecord, bool) {
	rec, ok := bb.Nodes[targetNodeID]
	if !ok || rec.Status != NodeStatusDone || rec.Operation != OpRead {
		return nil, false
	}
	return rec, true
}

// replayDescendants recomputes every descendant of targetNodeID (per the
// shared ShapeDAG's real DependsOn-edge graph walk, ShapeDAG.Descendants --
// not a hardcoded per-shape table) using each node's Operation function from
// v2semantics.go. It never touches EXTRACT_NUMBER/LOCATE_REGION/CROP_REGION
// nodes (those are never descendants of a value mutation in these frozen
// DAGs, but this is asserted explicitly rather than assumed) and never
// invokes any model transport.
func replayDescendants(bb *Blackboard, dag ShapeDAG, targetNodeID string) ([]string, error) {
	descendants := dag.Descendants(targetNodeID)
	values := make(map[string]float64)
	verdicts := make(map[string]string)

	// Seed `values`/`verdicts` from every non-descendant DONE node's
	// Outputs/OutputVerdict, so a descendant that depends on an untouched
	// sibling (e.g. RECONCILIATION_CHAIN's sub_A depends on norm_A AND
	// norm_a, but only norm_A might be downstream of the mutated node) can
	// still resolve its inputs.
	descendantSet := make(map[string]bool, len(descendants))
	for _, id := range descendants {
		descendantSet[id] = true
	}
	for id, rec := range bb.Nodes {
		if descendantSet[id] {
			continue
		}
		if rec.Status != NodeStatusDone {
			continue
		}
		for k, v := range rec.Outputs {
			values[k] = v
		}
		if rec.OutputVerdict != "" {
			step, ok := dag.StepByID(id)
			if ok {
				verdicts[step.OutputKey] = rec.OutputVerdict
			}
		}
	}
	// Seed the mutated target node's own (already-updated) output too.
	if targetRec, ok := bb.Nodes[targetNodeID]; ok {
		for k, v := range targetRec.Outputs {
			values[k] = v
		}
	}

	for _, nodeID := range descendants {
		step, ok := dag.StepByID(nodeID)
		if !ok {
			return nil, fmt.Errorf("tonalt1arms: replayDescendants: node %q not found in shape %s's DAG", nodeID, dag.Shape)
		}
		if step.Capability == "EXTRACT_NUMBER" || step.Operation == OpRead {
			return nil, fmt.Errorf("tonalt1arms: replayDescendants: refusing to replay EXTRACT_NUMBER node %q -- descendant replay must never re-invoke a model call", nodeID)
		}

		switch step.Operation {
		case "":
			continue
		case OpNormalize:
			in, ok := values[step.InputKeys[0]]
			if !ok {
				return nil, fmt.Errorf("tonalt1arms: replayDescendants: node %q missing upstream %q", nodeID, step.InputKeys[0])
			}
			values[step.OutputKey] = opNormalize(in)
		case OpMax:
			a, b, err := twoInputs(step, values)
			if err != nil {
				return nil, err
			}
			values[step.OutputKey] = opMax(a, b)
			verdicts["comparison_verdict"] = opCompareNumbers(a, b)
		case OpSubtract:
			a, b, err := twoInputs(step, values)
			if err != nil {
				return nil, err
			}
			values[step.OutputKey] = opSubtract(a, b)
		case OpDivide:
			a, b, err := twoInputs(step, values)
			if err != nil {
				return nil, err
			}
			result, divErr := opDivide(a, b)
			if divErr != nil {
				values[step.OutputKey] = math.NaN()
				continue
			}
			values[step.OutputKey] = result
		case OpPercentDifference:
			a, b, err := twoInputs(step, values)
			if err != nil {
				return nil, err
			}
			values[step.OutputKey] = opPercentDifference(a, b)
		case OpPercentToFraction:
			in, ok := values[step.InputKeys[0]]
			if !ok {
				return nil, fmt.Errorf("tonalt1arms: replayDescendants: node %q missing upstream %q", nodeID, step.InputKeys[0])
			}
			values[step.OutputKey] = opPercentToFraction(in)
		case OpSubtractTolerance:
			in, ok := values[step.InputKeys[0]]
			if !ok {
				return nil, fmt.Errorf("tonalt1arms: replayDescendants: node %q missing upstream %q", nodeID, step.InputKeys[0])
			}
			values[step.OutputKey] = opSubtractTolerance(in)
		case OpCompareZero:
			in, ok := values[step.InputKeys[0]]
			if !ok {
				return nil, fmt.Errorf("tonalt1arms: replayDescendants: node %q missing upstream %q", nodeID, step.InputKeys[0])
			}
			verdicts[step.OutputKey] = opCompareZero(in)
		case OpThresholdCheck:
			// side-observation, nothing to propagate
		case OpVerify:
			in, ok := values[step.InputKeys[0]]
			if !ok {
				return nil, fmt.Errorf("tonalt1arms: replayDescendants: node %q missing upstream %q", nodeID, step.InputKeys[0])
			}
			values[step.OutputKey] = in
		default:
			return nil, fmt.Errorf("tonalt1arms: replayDescendants: unknown Operation %q on node %q", step.Operation, nodeID)
		}

		rec := NodeRecord{
			NodeID:     nodeID,
			Capability: step.Capability,
			Operation:  step.Operation,
			ExecutorID: "counterfactual-replay",
			DependsOn:  step.DependsOn,
			Outputs:    map[string]float64{step.OutputKey: values[step.OutputKey]},
			Status:     NodeStatusDone,
		}
		if v, ok := verdicts[step.OutputKey]; ok {
			rec.OutputVerdict = v
			rec.Outputs = nil
		}
		bb.Nodes[nodeID] = &rec
	}

	terminalStep, ok := dag.StepByID(dag.TerminalNodeID)
	if !ok {
		return descendants, fmt.Errorf("tonalt1arms: replayDescendants: terminal node %q not found", dag.TerminalNodeID)
	}
	if _, ok := values[terminalStep.OutputKey]; !ok {
		return descendants, fmt.Errorf("tonalt1arms: replayDescendants: terminal value %q was not recomputed", terminalStep.OutputKey)
	}
	return descendants, nil
}

// RunPoisonOnBlackboard is the final scientific POISON implementation: it
// clones a completed v2 Arm-C Blackboard, replaces exactly the target node's
// observed value, invalidates and replays only its causal descendants via
// the shared DAG's Operation functions, and never invokes any model
// transport (ModelCallCount is always 0). If the target's primary
// EXTRACT_NUMBER observation was never completed in the input Blackboard,
// this reports PRIMARY_OBSERVATION_UNAVAILABLE and returns without
// mutating/replaying anything or fabricating a value from gold.
func RunPoisonOnBlackboard(bb *Blackboard, dag ShapeDAG, targetNodeID string, poisonValue float64) (BlackboardCounterfactualOutcome, *Blackboard, error) {
	targetRec, available := requirePrimaryObservation(bb, targetNodeID)
	if !available {
		return BlackboardCounterfactualOutcome{
			Operation: "POISON", WorkflowID: bb.WorkflowID, TargetNodeID: targetNodeID,
			PrimaryObservationUnavailable: true, ModelCallCount: 0, SemanticsVersion: CounterfactualSemanticsVersion,
		}, bb, nil
	}
	targetStep, ok := dag.StepByID(targetNodeID)
	if !ok {
		return BlackboardCounterfactualOutcome{}, bb, fmt.Errorf("tonalt1arms: RunPoisonOnBlackboard: node %q not found in shape %s's DAG", targetNodeID, dag.Shape)
	}

	role := targetStep.OutputKey
	original := targetRec.Outputs[role]
	originalFinalVal, _, originalPromoted := bb.Promoted()
	if !originalPromoted {
		if terminalStep, ok := dag.StepByID(dag.TerminalNodeID); ok {
			if rec, ok := bb.Nodes[dag.TerminalNodeID]; ok && rec.Status == NodeStatusDone {
				originalFinalVal = rec.Outputs[terminalStep.OutputKey]
			}
		}
	}

	clone := bb.Clone()
	mutated := *clone.Nodes[targetNodeID]
	mutated.Outputs = map[string]float64{role: poisonValue}
	clone.Nodes[targetNodeID] = &mutated

	replayed, err := replayDescendants(clone, dag, targetNodeID)
	if err != nil {
		return BlackboardCounterfactualOutcome{}, bb, err
	}

	newFinalVal := readTerminal(clone, dag)
	if dag.HasVerify {
		if terminalRec, ok := clone.Nodes[dag.TerminalNodeID]; ok && terminalRec.Status == NodeStatusDone {
			_ = clone.PromoteFinal(dag.TerminalNodeID)
		}
	}

	return BlackboardCounterfactualOutcome{
		Operation: "POISON", WorkflowID: bb.WorkflowID, TargetNodeID: targetNodeID,
		OriginalValue: original, MutatedValue: poisonValue,
		OriginalFinal: originalFinalVal, ResultingFinal: newFinalVal,
		TerminalChanged:  !valuesEqual(originalFinalVal, newFinalVal),
		ModelCallCount:   0,
		ReplayedNodeIDs:  replayed,
		SemanticsVersion: CounterfactualSemanticsVersion,
	}, clone, nil
}

// RunRemoveOnBlackboard is the final scientific REMOVE implementation:
// identical setup to RunPoisonOnBlackboard, but deletes the target node's
// record entirely (rather than mutating its value) and requires the
// replay to fail closed, since every frozen T1 shape's terminal value
// causally requires every one of its acquire-chain roles.
func RunRemoveOnBlackboard(bb *Blackboard, dag ShapeDAG, targetNodeID string) (BlackboardCounterfactualOutcome, *Blackboard, error) {
	targetRec, available := requirePrimaryObservation(bb, targetNodeID)
	if !available {
		return BlackboardCounterfactualOutcome{
			Operation: "REMOVE", WorkflowID: bb.WorkflowID, TargetNodeID: targetNodeID,
			PrimaryObservationUnavailable: true, ModelCallCount: 0, SemanticsVersion: CounterfactualSemanticsVersion,
		}, bb, nil
	}
	targetStep, ok := dag.StepByID(targetNodeID)
	if !ok {
		return BlackboardCounterfactualOutcome{}, bb, fmt.Errorf("tonalt1arms: RunRemoveOnBlackboard: node %q not found in shape %s's DAG", targetNodeID, dag.Shape)
	}
	role := targetStep.OutputKey
	original := targetRec.Outputs[role]
	originalFinalVal := readTerminal(bb, dag)

	clone := bb.Clone()
	delete(clone.Nodes, targetNodeID)

	replayed, err := replayDescendants(clone, dag, targetNodeID)
	if err == nil {
		return BlackboardCounterfactualOutcome{}, bb, fmt.Errorf("tonalt1arms: RunRemoveOnBlackboard: expected fail-closed behavior removing required node %q, but replay succeeded -- this shape does not actually require this node", targetNodeID)
	}

	return BlackboardCounterfactualOutcome{
		Operation: "REMOVE", WorkflowID: bb.WorkflowID, TargetNodeID: targetNodeID,
		OriginalValue: original, OriginalFinal: originalFinalVal,
		TerminalChanged:  true,
		FailedClosed:     true,
		FailureReason:    err.Error(),
		ModelCallCount:   0,
		ReplayedNodeIDs:  replayed,
		SemanticsVersion: CounterfactualSemanticsVersion,
	}, clone, nil
}

// readTerminal reads the shape's terminal value straight off its terminal
// node's own Outputs (for has_verify=false shapes) -- callers needing the
// VERIFY-promoted value specifically should use Blackboard.Promoted after
// promotion instead.
func readTerminal(bb *Blackboard, dag ShapeDAG) float64 {
	terminalStep, ok := dag.StepByID(dag.TerminalNodeID)
	if !ok {
		return math.NaN()
	}
	rec, ok := bb.Nodes[dag.TerminalNodeID]
	if !ok || rec.Status != NodeStatusDone {
		return math.NaN()
	}
	return rec.Outputs[terminalStep.OutputKey]
}
