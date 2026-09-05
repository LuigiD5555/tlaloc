package tonalt1arms

import (
	"context"
	"fmt"
)

// ArmAExecutor runs the real Arm A (monolithic Parrot) live execution path:
// exactly one Parrot call per workflow, against the already-frozen,
// startup-verified Arm-A composite image for that workflow -- never a
// per-operand call, never a node-by-node DAG walk (both explicitly
// prohibited by the frozen Arm-A policy). It never reads any gold-bearing
// artifact (no-gold-leakage invariant): TerminalOutput comes only from the
// parsed model response.
type ArmAExecutor struct {
	Manifest *ImageManifest // the frozen TONAL_T1_IMAGE_MANIFEST_FINAL.json, for the pre-call re-hash (correction I)
	Adapter  ParrotAdapter
	Policy   *ArmAPolicy
}

// ExecuteWorkflow verifies compositeBytes against the frozen manifest
// (fail-closed BEFORE touching Adapter, defense-in-depth re-hash of the
// exact startup-verified bytes per correction I), makes exactly one
// Adapter.Call, parses the response with the corrected ParseArmAResponse,
// and returns one WorkflowRecord + one NodeCallRecord.
func (e *ArmAExecutor) ExecuteWorkflow(ctx context.Context, runID string, wf Workflow, compositeBytes []byte) (WorkflowRecord, NodeCallRecord, error) {
	if e.Adapter == nil {
		return WorkflowRecord{}, NodeCallRecord{}, fmt.Errorf("tonalt1arms: ArmAExecutor: nil Adapter")
	}

	// A nil Manifest skips the hash-guard check -- consistent with Arm B/C's
	// own per-node Manifest handling, and legitimate for offline
	// tests/doctor checks using fixture bytes with no real frozen manifest.
	// A live caller must always supply the real frozen manifest (this is
	// enforced by the live CLI's own preflight sequence, not by this
	// executor refusing to run without one).
	if e.Manifest != nil {
		if err := VerifyComposite(e.Manifest, wf.WorkflowID, compositeBytes); err != nil {
			return WorkflowRecord{
					RunID: runID, WorkflowID: wf.WorkflowID, Family: wf.Shape, Arm: "A",
					TerminalStatus: "IMAGE_HASH_MISMATCH", ContractStatus: "IMAGE_HASH_MISMATCH",
				}, NodeCallRecord{
					RunID: runID, WorkflowID: wf.WorkflowID, Arm: "A", NodeID: "arm-a-composite-call",
					Capability: "ARM_A_MONOLITHIC", TransportStatus: "NOT_ATTEMPTED", ContractStatus: "IMAGE_HASH_MISMATCH",
				}, err
		}
	}

	prompt := armAPromptForShape(e.Policy, wf.Shape)
	resp, err := e.Adapter.Call(ctx, ParrotRequest{
		Capability:  "ARM_A_MONOLITHIC",
		Prompt:      prompt,
		Image:       compositeBytes,
		Temperature: 0,
		MaxTokens:   32,
	})

	nodeRec := NodeCallRecord{
		RunID: runID, WorkflowID: wf.WorkflowID, Arm: "A", NodeID: "arm-a-composite-call",
		Capability: "ARM_A_MONOLITHIC", RequestIndex: 0,
	}
	wfRec := WorkflowRecord{RunID: runID, WorkflowID: wf.WorkflowID, Family: wf.Shape, Arm: "A", TotalParrotCalls: 1}

	if err != nil {
		nodeRec.TransportStatus = "FAILED"
		nodeRec.ContractStatus = "TRANSPORT_FAILURE"
		wfRec.TerminalStatus = "FAILED_TRANSPORT"
		wfRec.ContractStatus = "TRANSPORT_FAILURE"
		return wfRec, nodeRec, nil
	}
	nodeRec.TransportStatus = "OK"
	nodeRec.RawOutput = resp.RawOutput

	value, ok, failureCode := ParseArmAResponse(resp.RawOutput)
	nodeRec.ParsedOutput = fmt.Sprintf("%v", value)
	if !ok {
		nodeRec.SchemaStatus = "FAILED"
		nodeRec.ContractStatus = failureCode
		wfRec.TerminalStatus = "PARSE_FAILURE"
		wfRec.ContractStatus = failureCode
		return wfRec, nodeRec, nil
	}
	nodeRec.SchemaStatus = "OK"
	nodeRec.ContractStatus = "OK"

	wfRec.TerminalOutput = value
	wfRec.TerminalStatus = "SUCCESS"
	wfRec.ContractStatus = "OK"
	return wfRec, nodeRec, nil
}

// armAPromptForShape returns the frozen per-shape prompt text from
// T1_D5_ARM_A_POLICY.json's parrot_prompt block (task_1..task_5, in the
// order READ_AND_CHECK, COMPARE_TWO_VALUES, DIFFERENCE_THEN_VERIFY,
// RATIO_OF_DIFFERENCE, RECONCILIATION_CHAIN -- the frozen file's own
// ordering).
func armAPromptForShape(policy *ArmAPolicy, shape string) string {
	if policy == nil {
		return ""
	}
	shapeToTaskKey := map[string]string{
		"READ_AND_CHECK":         "task_1",
		"COMPARE_TWO_VALUES":     "task_2",
		"DIFFERENCE_THEN_VERIFY": "task_3",
		"RATIO_OF_DIFFERENCE":    "task_4",
		"RECONCILIATION_CHAIN":   "task_5",
	}
	key, ok := shapeToTaskKey[shape]
	if !ok {
		return ""
	}
	raw, ok := policy.ParrotPrompt[key]
	if !ok {
		return ""
	}
	s, _ := raw.(string)
	return s
}
