package tonalt1arms

import "testing"

// TestDeriveArmBCallBudget_Real loads the actual frozen Arm-B policy and
// mechanically derives the total Parrot call count by walking all five
// frozen shape DAGs. This must equal 492 -- the frozen constant is not
// accepted on faith; it is recomputed from the DAG.
func TestDeriveArmBCallBudget_Real(t *testing.T) {
	policy, err := LoadArmBPolicy(RepoPathHelper("experiments/tonal-t1/d4/T1_D5_ARM_B_POLICY.json"))
	if err != nil {
		t.Fatalf("LoadArmBPolicy: %v", err)
	}
	if !policy.Frozen {
		t.Fatal("Arm-B policy is not marked frozen")
	}

	rows, total, err := DeriveArmBCallBudget(policy, 12)
	if err != nil {
		t.Fatalf("DeriveArmBCallBudget: %v", err)
	}

	want := map[string]int{
		"READ_AND_CHECK":         36,
		"COMPARE_TWO_VALUES":     60,
		"DIFFERENCE_THEN_VERIFY": 72,
		"RATIO_OF_DIFFERENCE":    120,
		"RECONCILIATION_CHAIN":   204,
	}
	for _, row := range rows {
		if row.ParrotCallsTotal != want[row.Family] {
			t.Errorf("%s: got %d Parrot calls, want %d", row.Family, row.ParrotCallsTotal, want[row.Family])
		}
	}

	if total != 492 {
		t.Fatalf("DERIVED_ARM_B_MODEL_CALLS = %d, want 492 (mechanical DAG derivation, not the accepted constant)", total)
	}
}

// TestArmACallBudget_Real derives Arm A's call count: exactly one
// monolithic Parrot invocation per workflow, per the frozen Arm-A policy's
// model_call_count_per_workflow field.
func TestArmACallBudget_Real(t *testing.T) {
	policy, err := LoadArmAPolicy(RepoPathHelper("experiments/tonal-t1/d4/T1_D5_ARM_A_POLICY.json"))
	if err != nil {
		t.Fatalf("LoadArmAPolicy: %v", err)
	}
	if policy.ModelCallCountPerWorkflow != 1 {
		t.Fatalf("Arm-A model_call_count_per_workflow = %d, want 1", policy.ModelCallCountPerWorkflow)
	}
	const workflows = 60
	got := workflows * policy.ModelCallCountPerWorkflow
	if got != 60 {
		t.Fatalf("DERIVED_ARM_A_MODEL_CALLS = %d, want 60", got)
	}
}

// TestArmCCallBudget_Real derives Arm C's call count: one EXTRACT_NUMBER
// Parrot call per operand-role assignment, per the frozen D4 workflow
// allocation (144 total).
func TestArmCCallBudget_Real(t *testing.T) {
	workflows, err := LoadWorkflows(RepoPathHelper("experiments/tonal-t1/d4/T1_D4_WORKFLOWS.json"))
	if err != nil {
		t.Fatalf("LoadWorkflows: %v", err)
	}
	total := 0
	for _, w := range workflows {
		total += len(w.Operands)
	}
	if total != 144 {
		t.Fatalf("DERIVED_ARM_C_MODEL_CALLS = %d, want 144", total)
	}
}

// TestPrimaryCallBudgetTotal_Real ties Arm A/B/C together and checks the
// frozen primary total 696.
func TestPrimaryCallBudgetTotal_Real(t *testing.T) {
	armAPolicy, err := LoadArmAPolicy(RepoPathHelper("experiments/tonal-t1/d4/T1_D5_ARM_A_POLICY.json"))
	if err != nil {
		t.Fatalf("LoadArmAPolicy: %v", err)
	}
	armBPolicy, err := LoadArmBPolicy(RepoPathHelper("experiments/tonal-t1/d4/T1_D5_ARM_B_POLICY.json"))
	if err != nil {
		t.Fatalf("LoadArmBPolicy: %v", err)
	}
	workflows, err := LoadWorkflows(RepoPathHelper("experiments/tonal-t1/d4/T1_D4_WORKFLOWS.json"))
	if err != nil {
		t.Fatalf("LoadWorkflows: %v", err)
	}

	armA := len(workflows) * armAPolicy.ModelCallCountPerWorkflow

	_, armB, err := DeriveArmBCallBudget(armBPolicy, 12)
	if err != nil {
		t.Fatalf("DeriveArmBCallBudget: %v", err)
	}

	armC := 0
	for _, w := range workflows {
		armC += len(w.Operands)
	}

	total := armA + armB + armC
	if armA != 60 || armB != 492 || armC != 144 || total != 696 {
		t.Fatalf("got A=%d B=%d C=%d total=%d, want A=60 B=492 C=144 total=696", armA, armB, armC, total)
	}
}
