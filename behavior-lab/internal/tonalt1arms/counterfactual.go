package tonalt1arms

import (
	"fmt"
	"math"
)

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
