package tonalt1arms

import (
	"math"
	"testing"
)

// buildCompletedV2Blackboard hand-constructs a completed Arm-C-shaped
// Blackboard for the given shape/operand values, as if Arm C's executor had
// already run every node to DONE. This is offline/synthetic (no model
// call), matching NewArmCState's role in the historical v1 tests, but
// produces a real *Blackboard* with per-node records rather than a flat
// role->value map, and drives its arithmetic exclusively through
// ComputeGoldV2 -- never gold.go's ComputeGold.
func buildCompletedV2Blackboard(t *testing.T, shape string, operandValues map[string]float64) (*Blackboard, ShapeDAG) {
	t.Helper()
	dag, err := BuildShapeDAG(shape)
	if err != nil {
		t.Fatal(err)
	}
	bb := NewBlackboard("wf-test-" + shape)

	values := make(map[string]float64)
	for _, step := range dag.Steps {
		switch step.Operation {
		case "":
			_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, DependsOn: step.DependsOn, Status: NodeStatusDone})
		case OpRead:
			role := step.OutputKey
			v, ok := operandValues[role]
			if !ok {
				t.Fatalf("missing operand value for role %q", role)
			}
			values[role] = v
			_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, Outputs: map[string]float64{role: v}, Status: NodeStatusDone})
		case OpNormalize:
			in := values[step.InputKeys[0]]
			out := opNormalize(in)
			values[step.OutputKey] = out
			_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, Outputs: map[string]float64{step.OutputKey: out}, Status: NodeStatusDone})
		case OpMax:
			a, b := values[step.InputKeys[0]], values[step.InputKeys[1]]
			out := opMax(a, b)
			values[step.OutputKey] = out
			_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, Outputs: map[string]float64{step.OutputKey: out}, Status: NodeStatusDone})
		case OpSubtract:
			a, b := values[step.InputKeys[0]], values[step.InputKeys[1]]
			out := opSubtract(a, b)
			values[step.OutputKey] = out
			_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, Outputs: map[string]float64{step.OutputKey: out}, Status: NodeStatusDone})
		case OpDivide:
			a, b := values[step.InputKeys[0]], values[step.InputKeys[1]]
			out, err := opDivide(a, b)
			if err != nil {
				out = math.NaN()
			}
			values[step.OutputKey] = out
			_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, Outputs: map[string]float64{step.OutputKey: out}, Status: NodeStatusDone})
		case OpPercentDifference:
			a, b := values[step.InputKeys[0]], values[step.InputKeys[1]]
			out := opPercentDifference(a, b)
			values[step.OutputKey] = out
			_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, Outputs: map[string]float64{step.OutputKey: out}, Status: NodeStatusDone})
		case OpPercentToFraction:
			in := values[step.InputKeys[0]]
			out := opPercentToFraction(in)
			values[step.OutputKey] = out
			_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, Outputs: map[string]float64{step.OutputKey: out}, Status: NodeStatusDone})
		case OpSubtractTolerance:
			in := values[step.InputKeys[0]]
			out := opSubtractTolerance(in)
			values[step.OutputKey] = out
			_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, Outputs: map[string]float64{step.OutputKey: out}, Status: NodeStatusDone})
		case OpCompareZero:
			in := values[step.InputKeys[0]]
			verdict := opCompareZero(in)
			_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, OutputVerdict: verdict, Status: NodeStatusDone})
		case OpThresholdCheck:
			_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, Status: NodeStatusDone})
		case OpVerify:
			in := values[step.InputKeys[0]]
			values[step.OutputKey] = in
			_ = bb.Record(NodeRecord{NodeID: step.LocalID, Capability: step.Capability, Operation: step.Operation, DependsOn: step.DependsOn, Outputs: map[string]float64{step.OutputKey: in}, Status: NodeStatusDone})
		}
	}
	if dag.HasVerify {
		if err := bb.PromoteFinal(dag.TerminalNodeID); err != nil {
			t.Fatalf("PromoteFinal: %v", err)
		}
	}
	return bb, dag
}

var shapeFixtures = []struct {
	shape    string
	operands map[string]float64
}{
	{"READ_AND_CHECK", map[string]float64{"A": 95}},
	{"COMPARE_TWO_VALUES", map[string]float64{"A": 60, "B": 420}},
	{"DIFFERENCE_THEN_VERIFY", map[string]float64{"A": 989, "B": 50}},
	{"RATIO_OF_DIFFERENCE", map[string]float64{"A": 0.5, "B": 420, "C": 0.85}},
	{"RECONCILIATION_CHAIN", map[string]float64{"A": 4.8, "a": 1, "B": 420, "b": 45}},
}

func TestRunPoisonOnBlackboard_AllShapes_ChangesTerminal(t *testing.T) {
	for _, fx := range shapeFixtures {
		t.Run(fx.shape, func(t *testing.T) {
			bb, dag := buildCompletedV2Blackboard(t, fx.shape, fx.operands)
			outcome, _, err := RunPoisonOnBlackboard(bb, dag, "read_A", 999999)
			if err != nil {
				t.Fatal(err)
			}
			if outcome.PrimaryObservationUnavailable {
				t.Fatal("unexpected PrimaryObservationUnavailable")
			}
			if outcome.ModelCallCount != 0 {
				t.Errorf("ModelCallCount = %d, want 0", outcome.ModelCallCount)
			}
			if !outcome.TerminalChanged {
				t.Errorf("expected terminal to change after poisoning role A to 999999")
			}
			if outcome.SemanticsVersion != "T1_V2" {
				t.Errorf("SemanticsVersion = %q, want T1_V2", outcome.SemanticsVersion)
			}
		})
	}
}

// TestRunPoisonOnBlackboard_CompareTwoValues_UsesV2Max is the required
// cross-check: Shape 2's poison replay must show max(A,B) behavior verified
// against ComputeGoldV2, never against the historical ComputeGold.
func TestRunPoisonOnBlackboard_CompareTwoValues_UsesV2Max(t *testing.T) {
	bb, dag := buildCompletedV2Blackboard(t, "COMPARE_TWO_VALUES", map[string]float64{"A": 60, "B": 420})
	outcome, clone, err := RunPoisonOnBlackboard(bb, dag, "read_A", 1000)
	if err != nil {
		t.Fatal(err)
	}
	wantFinal, _, _, _, err := ComputeGoldV2(dag, map[string]float64{"A": 1000, "B": 420})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.ResultingFinal != wantFinal {
		t.Errorf("ResultingFinal = %v, want %v (ComputeGoldV2/max)", outcome.ResultingFinal, wantFinal)
	}
	if outcome.ResultingFinal != 1000 {
		t.Errorf("ResultingFinal = %v, want 1000 (max(1000,420))", outcome.ResultingFinal)
	}
	// Sanity: the old v1 A-B formula would give 1000-420=580, must not appear.
	if outcome.ResultingFinal == 580 {
		t.Errorf("ResultingFinal matches the historical v1 A-B formula")
	}
	_ = clone
}

func TestRunRemoveOnBlackboard_AllShapes_FailClosed(t *testing.T) {
	for _, fx := range shapeFixtures {
		t.Run(fx.shape, func(t *testing.T) {
			bb, dag := buildCompletedV2Blackboard(t, fx.shape, fx.operands)
			outcome, _, err := RunRemoveOnBlackboard(bb, dag, "read_A")
			if err != nil {
				t.Fatal(err)
			}
			if !outcome.FailedClosed {
				t.Error("expected FailedClosed=true")
			}
			if outcome.ModelCallCount != 0 {
				t.Errorf("ModelCallCount = %d, want 0", outcome.ModelCallCount)
			}
		})
	}
}

func TestRunPoisonOnBlackboard_OnlyDescendantsReplay(t *testing.T) {
	bb, dag := buildCompletedV2Blackboard(t, "RECONCILIATION_CHAIN", map[string]float64{"A": 4.8, "a": 1, "B": 420, "b": 45})
	before := map[string]NodeRecord{}
	for id, rec := range bb.Nodes {
		before[id] = *rec
	}

	_, clone, err := RunPoisonOnBlackboard(bb, dag, "read_A", 999)
	if err != nil {
		t.Fatal(err)
	}

	descendants := dag.Descendants("read_A")
	descendantSet := make(map[string]bool, len(descendants))
	descendantSet["read_A"] = true
	for _, id := range descendants {
		descendantSet[id] = true
	}

	// Every non-descendant node (e.g. the entire "a"/"B"/"b" acquire chains)
	// must be byte-identical (deep-equal) in the clone vs the original.
	for id, orig := range before {
		if descendantSet[id] {
			continue
		}
		cloned, ok := clone.Nodes[id]
		if !ok {
			t.Fatalf("non-descendant node %q missing from clone", id)
		}
		if cloned.Status != orig.Status {
			t.Errorf("non-descendant node %q Status changed: %v -> %v", id, orig.Status, cloned.Status)
		}
		for k, v := range orig.Outputs {
			if cloned.Outputs[k] != v {
				t.Errorf("non-descendant node %q Outputs[%q] changed: %v -> %v", id, k, v, cloned.Outputs[k])
			}
		}
	}

	// The ORIGINAL Blackboard must be completely untouched.
	for id, orig := range before {
		origNow := *bb.Nodes[id]
		if origNow.Status != orig.Status {
			t.Errorf("original Blackboard node %q Status mutated: %v -> %v", id, orig.Status, origNow.Status)
		}
		for k, v := range orig.Outputs {
			if origNow.Outputs[k] != v {
				t.Errorf("original Blackboard node %q Outputs[%q] mutated: %v -> %v", id, k, v, origNow.Outputs[k])
			}
		}
	}
}

// TestRunPoisonOnBlackboard_PrimaryObservationUnavailable is the required
// test for correction D: if the target's primary EXTRACT_NUMBER observation
// was never completed, the runner must report
// PRIMARY_OBSERVATION_UNAVAILABLE and make zero model calls -- never
// backfill from gold.
func TestRunPoisonOnBlackboard_PrimaryObservationUnavailable(t *testing.T) {
	dag, err := BuildShapeDAG("READ_AND_CHECK")
	if err != nil {
		t.Fatal(err)
	}
	bb := NewBlackboard("wf-incomplete")
	// Deliberately do NOT record read_A as DONE.
	outcome, returned, err := RunPoisonOnBlackboard(bb, dag, "read_A", 42)
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.PrimaryObservationUnavailable {
		t.Error("expected PrimaryObservationUnavailable = true")
	}
	if outcome.ModelCallCount != 0 {
		t.Errorf("ModelCallCount = %d, want 0", outcome.ModelCallCount)
	}
	if returned != bb {
		t.Error("expected the same (unmutated) Blackboard to be returned when observation unavailable")
	}
}

func TestRunRemoveOnBlackboard_PrimaryObservationUnavailable(t *testing.T) {
	dag, err := BuildShapeDAG("READ_AND_CHECK")
	if err != nil {
		t.Fatal(err)
	}
	bb := NewBlackboard("wf-incomplete")
	outcome, _, err := RunRemoveOnBlackboard(bb, dag, "read_A")
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.PrimaryObservationUnavailable {
		t.Error("expected PrimaryObservationUnavailable = true")
	}
	if outcome.ModelCallCount != 0 {
		t.Errorf("ModelCallCount = %d, want 0", outcome.ModelCallCount)
	}
}

// TestBlackboardCounterfactual_Full288TrialSweep_ZeroModelCalls exercises
// POISON+REMOVE across all 5 shapes x every acquire-chain role (matching the
// frozen 144+144=288 counterfactual trial count in spirit, offline), and
// asserts an aggregate zero EXTRACT_NUMBER/model-transport call count.
func TestBlackboardCounterfactual_Full288TrialSweep_ZeroModelCalls(t *testing.T) {
	totalModelCalls := 0
	trials := 0
	for _, fx := range shapeFixtures {
		for role := range fx.operands {
			bb, dag := buildCompletedV2Blackboard(t, fx.shape, fx.operands)
			nodeID := "read_" + role

			poisonOutcome, _, err := RunPoisonOnBlackboard(bb, dag, nodeID, 12345)
			if err != nil {
				t.Fatalf("%s/%s POISON: %v", fx.shape, role, err)
			}
			totalModelCalls += poisonOutcome.ModelCallCount
			trials++

			removeOutcome, _, err := RunRemoveOnBlackboard(bb, dag, nodeID)
			if err != nil {
				t.Fatalf("%s/%s REMOVE: %v", fx.shape, role, err)
			}
			totalModelCalls += removeOutcome.ModelCallCount
			trials++
		}
	}
	if totalModelCalls != 0 {
		t.Fatalf("aggregate ModelCallCount across %d trials = %d, want 0", trials, totalModelCalls)
	}
	if trials != 2*(1+2+2+3+4) { // roles per shape: 1+2+2+3+4 = 12, x2 (poison+remove)
		t.Fatalf("ran %d trials, want %d", trials, 2*(1+2+2+3+4))
	}
}
