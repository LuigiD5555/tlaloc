package tonalt1arms

import (
	"context"
	"testing"
)

func buildCrossArmRunner(t *testing.T, adapter ParrotAdapter) *CrossArmRunner {
	t.Helper()
	armAPolicy := loadRealArmAPolicy(t)
	armBExec := loadRealArmBExecutor(t, adapter)
	armCExec := loadRealArmCExecutor(t, adapter)
	return &CrossArmRunner{
		ArmA: &ArmAExecutor{Adapter: adapter, Policy: armAPolicy},
		ArmB: armBExec,
		ArmC: armCExec,
	}
}

// buildFakeImageBundle constructs deterministic fake composite/operand
// image bundles and a matching fake ImageManifest for the given workflows,
// and wires the manifest into the runner's Arm A/B/C executors so the
// hash-guard code paths are exercised (with fake, internally-consistent
// data -- not the real frozen manifest, which the slow
// TestStartupImageSweep_RealMaterialization_FullOffline test already
// covers).
func buildFakeImageBundle(workflows []Workflow) (compositeImages map[string][]byte, operandImagesByWorkflow map[string]map[string][]byte, manifest *ImageManifest) {
	compositeImages = make(map[string][]byte, len(workflows))
	operandImagesByWorkflow = make(map[string]map[string][]byte, len(workflows))
	manifest = &ImageManifest{}

	for _, wf := range workflows {
		compositeImages[wf.WorkflowID] = []byte("fake-composite-" + wf.WorkflowID)
		compositeHash := sha256Hex(compositeImages[wf.WorkflowID])
		manifest.Composites = append(manifest.Composites, CompositeImageRecord{WorkflowID: wf.WorkflowID, Run1CompositeSHA256: compositeHash, Run2CompositeSHA256: compositeHash, Equal: true})

		operandImages := make(map[string][]byte, len(wf.Operands))
		for _, op := range wf.Operands {
			data := []byte(op.CandidateID)
			operandImages[wf.WorkflowID+"|"+op.Role] = data
			hash := sha256Hex(data)
			manifest.Operands = append(manifest.Operands, OperandImageRecord{WorkflowID: wf.WorkflowID, Role: op.Role, CandidateID: op.CandidateID, Run1PreparedSHA256: hash, Run2PreparedSHA256: hash, Equal: true})
		}
		operandImagesByWorkflow[wf.WorkflowID] = operandImages
	}
	return compositeImages, operandImagesByWorkflow, manifest
}

// TestCrossArmRunner_AllSuccess_696TotalCalls is the required complete
// offline cross-arm run: A=60, B=492, C=144, TOTAL=696, from runtime traces,
// cross-checked against callbudget.go's independent derivation.
func TestCrossArmRunner_AllSuccess_696TotalCalls(t *testing.T) {
	workflows := loadRealWorkflows(t)
	adapter := newFakeParrotAdapter()
	adapter.defaultAnswer = 1

	compositeImages, operandImagesByWorkflow, manifest := buildFakeImageBundle(workflows)
	runner := buildCrossArmRunner(t, adapter)
	runner.ArmA.Manifest = manifest
	runner.ArmB.Manifest = manifest
	runner.ArmC.Manifest = manifest

	result, err := runner.RunAll(context.Background(), "run-1", workflows, compositeImages, operandImagesByWorkflow)
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}

	if result.ArmACalls != 60 {
		t.Errorf("ArmACalls = %d, want 60", result.ArmACalls)
	}
	if result.ArmBCalls != 492 {
		t.Errorf("ArmBCalls = %d, want 492", result.ArmBCalls)
	}
	if result.ArmCCalls != 144 {
		t.Errorf("ArmCCalls = %d, want 144", result.ArmCCalls)
	}
	if result.TotalCalls != 696 {
		t.Errorf("TotalCalls = %d, want 696", result.TotalCalls)
	}
	if result.Accounting.PlannedModelCallSlots != 696 {
		t.Errorf("PlannedModelCallSlots = %d, want 696", result.Accounting.PlannedModelCallSlots)
	}
	if result.Accounting.HTTPRequestAttempts != 696 {
		t.Errorf("HTTPRequestAttempts = %d, want 696 (all-success run)", result.Accounting.HTTPRequestAttempts)
	}
	if len(result.WorkflowRecords) != 180 {
		t.Errorf("len(WorkflowRecords) = %d, want 180", len(result.WorkflowRecords))
	}

	// Cross-check against callbudget.go's independent derivation.
	armBPolicy, err := LoadArmBPolicy(RepoPathHelper("experiments/tonal-t1/d4/T1_D5_ARM_B_POLICY.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, derivedArmB, err := DeriveArmBCallBudget(armBPolicy, 12)
	if err != nil {
		t.Fatal(err)
	}
	if derivedArmB != result.ArmBCalls {
		t.Errorf("runtime-derived ArmBCalls %d != independently-derived %d", result.ArmBCalls, derivedArmB)
	}
}

// TestCrossArmRunner_UpstreamFailure_PartialAccounting is the required
// one-upstream-failure test: HTTPRequestAttempts < PlannedModelCallSlots ==
// 696, and every planned slot still has a terminal NodeCallRecord.
func TestCrossArmRunner_UpstreamFailure_PartialAccounting(t *testing.T) {
	workflows := loadRealWorkflows(t)
	adapter := newFakeParrotAdapter()
	adapter.defaultAnswer = 1
	adapter.failCapability = "NORMALIZE" // fails every NORMALIZE call in Arm B only (Arm C doesn't call NORMALIZE at all)

	compositeImages, operandImagesByWorkflow, manifest := buildFakeImageBundle(workflows)
	runner := buildCrossArmRunner(t, adapter)
	runner.ArmA.Manifest = manifest
	runner.ArmB.Manifest = manifest
	runner.ArmC.Manifest = manifest

	result, err := runner.RunAll(context.Background(), "run-1", workflows, compositeImages, operandImagesByWorkflow)
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}

	if result.Accounting.PlannedModelCallSlots != 696 {
		t.Fatalf("PlannedModelCallSlots = %d, want 696 (static, independent of the failure)", result.Accounting.PlannedModelCallSlots)
	}
	if result.Accounting.HTTPRequestAttempts >= result.Accounting.PlannedModelCallSlots {
		t.Fatalf("HTTPRequestAttempts (%d) should be less than PlannedModelCallSlots (%d) once NORMALIZE always fails and blocks its dependents", result.Accounting.HTTPRequestAttempts, result.Accounting.PlannedModelCallSlots)
	}
	if result.Accounting.BlockedByDependency == 0 {
		t.Fatal("expected at least one BLOCKED_BY_DEPENDENCY node in the accounting")
	}
	if result.Accounting.TransportFailures == 0 {
		t.Fatal("expected at least one TransportFailures entry for the always-failing NORMALIZE calls")
	}

	// Freeze must still succeed (180 records is about workflow count, not
	// about every node having succeeded).
	if err := result.Freeze(t.TempDir()); err != nil {
		t.Fatalf("Freeze should succeed on a partial-failure run with 180 workflow records: %v", err)
	}
}

// TestCrossArmRunner_Freeze_RefusesIncompleteRun proves Freeze fails loudly
// rather than writing a partial/incomplete run.
func TestCrossArmRunner_Freeze_RefusesIncompleteRun(t *testing.T) {
	incomplete := RunResult{
		WorkflowRecords: []WorkflowRecord{{WorkflowID: "only-one"}},
		Accounting:      RunAccounting{PlannedModelCallSlots: 696},
	}
	if err := incomplete.Freeze(t.TempDir()); err == nil {
		t.Fatal("expected Freeze to refuse a run with only 1 workflow record")
	}
}
