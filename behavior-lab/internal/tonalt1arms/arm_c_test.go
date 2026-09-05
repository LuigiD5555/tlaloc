package tonalt1arms

import (
	"context"
	"testing"
)

func loadRealArmCExecutor(t *testing.T, adapter ParrotAdapter) *ArmCExecutor {
	t.Helper()
	policy, err := LoadArmCPolicy(RepoPathHelper("experiments/tonal-t1/d4/T1_D5_ARM_C_POLICY.json"))
	if err != nil {
		t.Fatalf("LoadArmCPolicy: %v", err)
	}
	bindings, err := BuildArmCBindings(policy)
	if err != nil {
		t.Fatalf("BuildArmCBindings: %v", err)
	}
	return &ArmCExecutor{Bindings: bindings, Adapter: adapter}
}

// TestArmCExecutor_AllSuccessPath_144TotalCalls is the required mechanical
// proof: 60 workflows -> 144 Parrot calls (== sum(len(wf.Operands)),
// matching TestArmCCallBudget_Real's independent count), and zero calls for
// any non-EXTRACT_NUMBER node.
func TestArmCExecutor_AllSuccessPath_144TotalCalls(t *testing.T) {
	workflows := loadRealWorkflows(t)
	adapter := newFakeParrotAdapter()
	adapter.defaultAnswer = 1
	exec := loadRealArmCExecutor(t, adapter)

	successCount := 0
	for _, wf := range workflows {
		wfRec, nodeRecords, _, err := exec.ExecuteWorkflow(context.Background(), "run-1", wf, fakeOperandImagesFor(wf))
		if err != nil {
			t.Fatalf("%s: ExecuteWorkflow: %v", wf.WorkflowID, err)
		}
		if wfRec.TerminalStatus == "SUCCESS" {
			successCount++
		}
		for _, rec := range nodeRecords {
			if rec.Capability != "EXTRACT_NUMBER" {
				t.Errorf("%s: unexpected adapter call for capability %q -- Arm C must only call Parrot for EXTRACT_NUMBER", wf.WorkflowID, rec.Capability)
			}
		}
	}
	if successCount != 60 {
		t.Fatalf("successCount = %d, want 60", successCount)
	}
	if adapter.totalCalls() != 144 {
		t.Fatalf("adapter.totalCalls() = %d, want 144 (from runtime trace)", adapter.totalCalls())
	}
	if adapter.callsFor("EXTRACT_NUMBER") != 144 {
		t.Fatalf("EXTRACT_NUMBER calls = %d, want 144", adapter.callsFor("EXTRACT_NUMBER"))
	}
	for _, cap := range []string{"NORMALIZE", "COMPARE_NUMBERS", "ARITHMETIC"} {
		if adapter.callsFor(cap) != 0 {
			t.Errorf("Arm C made %d calls for %s, want 0 (deterministic)", adapter.callsFor(cap), cap)
		}
	}
}

// TestArmCExecutor_CompareTwoValues_UsesV2Max_NeverComputeGold proves Arm
// C's Shape 2 terminal is exactly max(A,B) computed via opMax -- never the
// historical v1 ComputeGold formula (A-B) and never a value read from any
// gold artifact.
func TestArmCExecutor_CompareTwoValues_UsesV2Max_NeverComputeGold(t *testing.T) {
	workflows := loadRealWorkflows(t)
	var wf Workflow
	for _, w := range workflows {
		if w.Shape == "COMPARE_TWO_VALUES" {
			wf = w
			break
		}
	}
	roleMap := makeRoleMap(wf.Operands)

	adapter := newFakeParrotAdapter()
	// Configure EXTRACT_NUMBER to return each operand's real gold value,
	// keyed by the fake image bytes (== candidate ID) used in this test.
	for _, op := range wf.Operands {
		key := "EXTRACT_NUMBER|" + op.CandidateID
		adapter.answerForImage[key] = op.NumericValue
	}
	exec := loadRealArmCExecutor(t, adapter)

	wfRec, _, _, err := exec.ExecuteWorkflow(context.Background(), "run-1", wf, fakeOperandImagesFor(wf))
	if err != nil {
		t.Fatal(err)
	}

	wantMax := roleMap["A"]
	if roleMap["B"] > wantMax {
		wantMax = roleMap["B"]
	}
	if wfRec.TerminalOutput != wantMax {
		t.Fatalf("TerminalOutput = %v, want max(A,B)=%v", wfRec.TerminalOutput, wantMax)
	}
	v1Result := roleMap["A"] - roleMap["B"]
	if wantMax != v1Result && wfRec.TerminalOutput == v1Result {
		t.Fatalf("TerminalOutput matches the historical v1 A-B formula (%v), not max(A,B)", v1Result)
	}
}

// TestArmCExecutor_DivisionByZero_ReportsContractFailure exercises the
// frozen INVALID_INPUT_DENOMINATOR_ZERO path through the real deterministic
// executor (RATIO_OF_DIFFERENCE with C=0).
func TestArmCExecutor_DivisionByZero_ReportsContractFailure(t *testing.T) {
	wf := Workflow{
		WorkflowID: "wf-zero-denom", Shape: "RATIO_OF_DIFFERENCE",
		Operands: []Operand{
			{Role: "A", NumericValue: 5, CandidateID: "cand-a"},
			{Role: "B", NumericValue: 5, CandidateID: "cand-b"},
			{Role: "C", NumericValue: 0, CandidateID: "cand-c"},
		},
	}
	adapter := newFakeParrotAdapter()
	for _, op := range wf.Operands {
		adapter.answerForImage["EXTRACT_NUMBER|"+op.CandidateID] = op.NumericValue
	}
	exec := loadRealArmCExecutor(t, adapter)

	wfRec, _, _, err := exec.ExecuteWorkflow(context.Background(), "run-1", wf, fakeOperandImagesFor(wf))
	if err != nil {
		t.Fatal(err)
	}
	if wfRec.TerminalStatus == "SUCCESS" {
		t.Fatalf("expected a contract failure for zero-denominator, got SUCCESS with TerminalOutput=%v", wfRec.TerminalOutput)
	}
}
