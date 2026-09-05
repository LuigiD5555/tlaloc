package tonalt1arms

import "testing"

// TestCounterfactual_Full288StructuralSweep_ZeroModelCalls is the required
// full structural enumeration (task mandate §24): for every one of the 60
// real frozen D4 workflows and every one of its real operand-role
// assignments (144 total, matching T1_D4_COUNTERFACTUAL_POISON.json/
// T1_D4_COUNTERFACTUAL_REMOVE.json's own frozen trial counts), run one
// POISON and one REMOVE trial against a synthetic/fake completed Arm-C V2
// Blackboard (built offline from the workflow's own operand values, never
// from a real model call) -- 144 POISON + 144 REMOVE = 288 trials total.
//
// This does not read or require any real T1 model output: every completed
// Blackboard is constructed offline via buildCompletedV2Blackboard, which
// runs ComputeGoldV2's node-by-node Operation functions against the
// workflow's own frozen operand values (not against any gold artifact --
// the terminal/intermediate values it produces are computed, not read).
func TestCounterfactual_Full288StructuralSweep_ZeroModelCalls(t *testing.T) {
	workflows := loadRealWorkflows(t)
	if len(workflows) != 60 {
		t.Fatalf("expected 60 frozen workflows, got %d", len(workflows))
	}

	var (
		poisonTrials              int
		removeTrials              int
		totalModelCalls           int
		terminalChanged           int
		failedClosedCount         int
		primaryObservationMissing int
	)

	for _, wf := range workflows {
		operandValues := makeRoleMap(wf.Operands)
		if len(operandValues) != len(wf.Operands) {
			t.Fatalf("%s: duplicate role in operand map", wf.WorkflowID)
		}

		for _, op := range wf.Operands {
			bb, dag := buildCompletedV2Blackboard(t, wf.Shape, operandValues)
			nodeID := "read_" + op.Role

			poisonOutcome, _, err := RunPoisonOnBlackboard(bb, dag, nodeID, op.NumericValue+123456)
			if err != nil {
				t.Fatalf("%s/%s POISON: %v", wf.WorkflowID, op.Role, err)
			}
			poisonTrials++
			totalModelCalls += poisonOutcome.ModelCallCount
			if poisonOutcome.PrimaryObservationUnavailable {
				primaryObservationMissing++
			} else if poisonOutcome.TerminalChanged {
				terminalChanged++
			}

			removeOutcome, _, err := RunRemoveOnBlackboard(bb, dag, nodeID)
			if err != nil {
				t.Fatalf("%s/%s REMOVE: %v", wf.WorkflowID, op.Role, err)
			}
			removeTrials++
			totalModelCalls += removeOutcome.ModelCallCount
			if removeOutcome.PrimaryObservationUnavailable {
				primaryObservationMissing++
			} else if removeOutcome.FailedClosed {
				failedClosedCount++
			}
		}
	}

	if poisonTrials != 144 {
		t.Fatalf("poisonTrials = %d, want 144 (one per real frozen operand-role assignment)", poisonTrials)
	}
	if removeTrials != 144 {
		t.Fatalf("removeTrials = %d, want 144", removeTrials)
	}
	if poisonTrials+removeTrials != 288 {
		t.Fatalf("total trials = %d, want 288", poisonTrials+removeTrials)
	}
	if totalModelCalls != 0 {
		t.Fatalf("totalModelCalls across all 288 trials = %d, want 0", totalModelCalls)
	}
	if primaryObservationMissing != 0 {
		t.Fatalf("primaryObservationMissing = %d, want 0 (every trial's target was built as a completed observation)", primaryObservationMissing)
	}
	// Every REMOVE trial removes a required acquire-chain role, so every one
	// must fail closed (no frozen T1 shape's terminal value survives losing
	// any of its declared operand roles).
	if failedClosedCount != 144 {
		t.Fatalf("failedClosedCount = %d, want 144 (every REMOVE trial must fail closed)", failedClosedCount)
	}
	t.Logf("288-trial structural sweep: %d POISON terminal-changed, %d REMOVE failed-closed, %d total model calls", terminalChanged, failedClosedCount, totalModelCalls)
}
