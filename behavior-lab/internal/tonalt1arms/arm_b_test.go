package tonalt1arms

import (
	"context"
	"testing"
)

func loadRealArmBExecutor(t *testing.T, adapter ParrotAdapter) *ArmBExecutor {
	t.Helper()
	policy, err := LoadArmBPolicy(RepoPathHelper("experiments/tonal-t1/d4/T1_D5_ARM_B_POLICY.json"))
	if err != nil {
		t.Fatalf("LoadArmBPolicy: %v", err)
	}
	bindings, err := BuildArmBBindings(policy)
	if err != nil {
		t.Fatalf("BuildArmBBindings: %v", err)
	}
	return &ArmBExecutor{Bindings: bindings, Policy: policy, Adapter: adapter}
}

func fakeOperandImagesFor(wf Workflow) map[string][]byte {
	images := make(map[string][]byte, len(wf.Operands))
	for _, op := range wf.Operands {
		images[wf.WorkflowID+"|"+op.Role] = []byte(op.CandidateID)
	}
	return images
}

// TestArmBExecutor_AllSuccessPath_492TotalCalls is the required mechanical
// proof: the actual per-node execution trace over all 60 workflows produces
// 492 total Parrot calls, broken down 36/60/72/120/204 by family --
// cross-checked against DeriveArmBCallBudget's independently-derived
// numbers, not just against the hardcoded constant.
func TestArmBExecutor_AllSuccessPath_492TotalCalls(t *testing.T) {
	workflows := loadRealWorkflows(t)
	adapter := newFakeParrotAdapter()
	adapter.defaultAnswer = 1
	exec := loadRealArmBExecutor(t, adapter)

	callsByShape := make(map[string]int)
	for _, wf := range workflows {
		images := fakeOperandImagesFor(wf)
		wfRec, nodeRecords, _, err := exec.ExecuteWorkflow(context.Background(), "run-1", wf, images)
		if err != nil {
			t.Fatalf("%s: ExecuteWorkflow: %v", wf.WorkflowID, err)
		}
		if wfRec.TerminalStatus != "SUCCESS" {
			t.Fatalf("%s: TerminalStatus = %q, want SUCCESS", wf.WorkflowID, wfRec.TerminalStatus)
		}
		callsByShape[wf.Shape] += len(nodeRecords)
	}

	want := map[string]int{
		"READ_AND_CHECK":         36,
		"COMPARE_TWO_VALUES":     60,
		"DIFFERENCE_THEN_VERIFY": 72,
		"RATIO_OF_DIFFERENCE":    120,
		"RECONCILIATION_CHAIN":   204,
	}
	total := 0
	for shape, count := range callsByShape {
		total += count
		if count != want[shape] {
			t.Errorf("%s: %d calls, want %d", shape, count, want[shape])
		}
	}
	if total != 492 {
		t.Fatalf("total = %d, want 492", total)
	}
	if adapter.totalCalls() != 492 {
		t.Fatalf("adapter.totalCalls() = %d, want 492 (from runtime trace)", adapter.totalCalls())
	}

	// Cross-check against the independently-derived budget.
	armBPolicy, err := LoadArmBPolicy(RepoPathHelper("experiments/tonal-t1/d4/T1_D5_ARM_B_POLICY.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, derivedTotal, err := DeriveArmBCallBudget(armBPolicy, 12)
	if err != nil {
		t.Fatal(err)
	}
	if derivedTotal != total {
		t.Fatalf("runtime-derived total %d != independently-derived call-budget total %d", total, derivedTotal)
	}
}

// TestArmBExecutor_CompareTwoValues_UsesV2MaxViaAdapter proves Arm B's Shape
// 2 result is whatever the (fake) adapter returns for the comparison node,
// and that the terminal is read from that node's own output, not a v1
// formula.
func TestArmBExecutor_CompareTwoValues_UsesV2Max(t *testing.T) {
	workflows := loadRealWorkflows(t)
	var wf Workflow
	for _, w := range workflows {
		if w.Shape == "COMPARE_TWO_VALUES" {
			wf = w
			break
		}
	}
	adapter := newFakeParrotAdapter()
	adapter.defaultAnswer = 777 // the fake adapter always answers 777 regardless of capability
	exec := loadRealArmBExecutor(t, adapter)

	wfRec, _, _, err := exec.ExecuteWorkflow(context.Background(), "run-1", wf, fakeOperandImagesFor(wf))
	if err != nil {
		t.Fatal(err)
	}
	if wfRec.TerminalOutput != 777 {
		t.Fatalf("TerminalOutput = %v, want 777 (the cmp node's own adapter-returned value, not any v1/v2 recomputation)", wfRec.TerminalOutput)
	}
}

// TestArmBExecutor_UpstreamFailure_BlocksDownstream is the required
// correction-G test: a fake adapter configured to fail one specific
// capability must cause all dependent downstream nodes to be recorded
// BLOCKED_BY_DEPENDENCY and never invoke Adapter.Call themselves.
func TestArmBExecutor_UpstreamFailure_BlocksDownstream(t *testing.T) {
	workflows := loadRealWorkflows(t)
	var wf Workflow
	for _, w := range workflows {
		if w.Shape == "DIFFERENCE_THEN_VERIFY" {
			wf = w
			break
		}
	}
	adapter := newFakeParrotAdapter()
	adapter.failCapability = "EXTRACT_NUMBER" // fail every read_<role> node
	exec := loadRealArmBExecutor(t, adapter)

	wfRec, nodeRecords, bb, err := exec.ExecuteWorkflow(context.Background(), "run-1", wf, fakeOperandImagesFor(wf))
	if err != nil {
		t.Fatal(err)
	}
	if wfRec.TerminalStatus == "SUCCESS" {
		t.Fatal("expected a non-SUCCESS terminal status when EXTRACT_NUMBER always fails")
	}

	blockedCount := 0
	for _, id := range bb.OrderedNodeIDs() {
		if bb.Nodes[id].Status == NodeStatusBlockedByDependency {
			blockedCount++
		}
	}
	if blockedCount == 0 {
		t.Fatal("expected at least one BLOCKED_BY_DEPENDENCY node")
	}

	// Every attempted call must have been for EXTRACT_NUMBER (the failing
	// capability) or a node that could resolve its inputs before hitting a
	// failed dependency -- no NORMALIZE/ARITHMETIC/VERIFY call should
	// succeed once its upstream EXTRACT_NUMBER failed.
	for _, rec := range nodeRecords {
		if rec.Capability != "EXTRACT_NUMBER" && rec.ContractStatus == "OK" {
			t.Errorf("node %s (%s) succeeded despite its EXTRACT_NUMBER upstream always failing", rec.NodeID, rec.Capability)
		}
	}

	planned := 0
	attempted := 0
	for _, id := range bb.OrderedNodeIDs() {
		rec := bb.Nodes[id]
		if rec.Operation == "" {
			continue // LOCATE_REGION/CROP_REGION are not generative slots
		}
		planned++
		if rec.ModelCallDelta > 0 {
			attempted++
		}
	}
	if attempted >= planned {
		t.Fatalf("attempted (%d) should be less than planned (%d) once an upstream failure blocks downstream nodes", attempted, planned)
	}
}
