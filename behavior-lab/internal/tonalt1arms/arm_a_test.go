package tonalt1arms

import (
	"context"
	"testing"
)

func loadRealWorkflows(t *testing.T) []Workflow {
	t.Helper()
	workflows, err := LoadWorkflows(RepoPathHelper("experiments/tonal-t1/d4/T1_D4_WORKFLOWS.json"))
	if err != nil {
		t.Fatalf("LoadWorkflows: %v", err)
	}
	return workflows
}

func loadRealArmAPolicy(t *testing.T) *ArmAPolicy {
	t.Helper()
	policy, err := LoadArmAPolicy(RepoPathHelper("experiments/tonal-t1/d4/T1_D5_ARM_A_POLICY.json"))
	if err != nil {
		t.Fatalf("LoadArmAPolicy: %v", err)
	}
	return policy
}

// fakeCompositeManifest builds a minimal ImageManifest whose composite
// records match exactly the given fake composite bytes' SHA-256, so
// ArmAExecutor's pre-call VerifyComposite passes without needing the real
// 60-composite frozen manifest or real materialization.
func fakeCompositeManifest(workflows []Workflow, compositeBytes map[string][]byte) *ImageManifest {
	m := &ImageManifest{}
	for _, wf := range workflows {
		data := compositeBytes[wf.WorkflowID]
		hash := sha256Hex(data)
		m.Composites = append(m.Composites, CompositeImageRecord{
			WorkflowID: wf.WorkflowID, Run1CompositeSHA256: hash, Run2CompositeSHA256: hash, Equal: true,
		})
	}
	return m
}

// TestArmAExecutor_60WorkflowsMake60Calls is the required mechanical proof:
// 60 workflows -> 60 calls, from the runtime trace, not asserted by
// inspection.
func TestArmAExecutor_60WorkflowsMake60Calls(t *testing.T) {
	workflows := loadRealWorkflows(t)
	if len(workflows) != 60 {
		t.Fatalf("expected 60 workflows, got %d", len(workflows))
	}

	compositeBytes := make(map[string][]byte, len(workflows))
	for _, wf := range workflows {
		compositeBytes[wf.WorkflowID] = []byte("fake-composite-" + wf.WorkflowID)
	}
	manifest := fakeCompositeManifest(workflows, compositeBytes)
	adapter := newFakeParrotAdapter()
	adapter.defaultAnswer = 42

	exec := &ArmAExecutor{Manifest: manifest, Adapter: adapter, Policy: loadRealArmAPolicy(t)}

	successCount := 0
	for _, wf := range workflows {
		wfRec, _, err := exec.ExecuteWorkflow(context.Background(), "run-1", wf, compositeBytes[wf.WorkflowID])
		if err != nil {
			t.Fatalf("%s: ExecuteWorkflow: %v", wf.WorkflowID, err)
		}
		if wfRec.TerminalStatus == "SUCCESS" {
			successCount++
		}
	}
	if successCount != 60 {
		t.Fatalf("successCount = %d, want 60", successCount)
	}
	if adapter.totalCalls() != 60 {
		t.Fatalf("adapter.totalCalls() = %d, want 60 (mechanical, from runtime trace)", adapter.totalCalls())
	}
}

// TestArmAExecutor_ImageHashMismatchFailsClosedBeforeCall proves the
// composite hash is checked BEFORE any adapter call.
func TestArmAExecutor_ImageHashMismatchFailsClosedBeforeCall(t *testing.T) {
	workflows := loadRealWorkflows(t)
	wf := workflows[0]
	manifest := fakeCompositeManifest([]Workflow{wf}, map[string][]byte{wf.WorkflowID: []byte("expected bytes")})
	adapter := newFakeParrotAdapter()
	exec := &ArmAExecutor{Manifest: manifest, Adapter: adapter, Policy: loadRealArmAPolicy(t)}

	wfRec, _, err := exec.ExecuteWorkflow(context.Background(), "run-1", wf, []byte("WRONG bytes"))
	if err == nil {
		t.Fatal("expected an image-hash-mismatch error")
	}
	if wfRec.TerminalStatus != "IMAGE_HASH_MISMATCH" {
		t.Fatalf("TerminalStatus = %q, want IMAGE_HASH_MISMATCH", wfRec.TerminalStatus)
	}
	if adapter.totalCalls() != 0 {
		t.Fatalf("adapter.totalCalls() = %d, want 0 (must fail BEFORE any call)", adapter.totalCalls())
	}
}
