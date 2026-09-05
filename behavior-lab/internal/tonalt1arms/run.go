package tonalt1arms

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CrossArmRunner wires the three real executors together to run the
// complete offline A+B+C sweep over the 60 frozen workflows.
type CrossArmRunner struct {
	ArmA *ArmAExecutor
	ArmB *ArmBExecutor
	ArmC *ArmCExecutor
}

// RunResult is the aggregated output of one complete cross-arm run:
// per-workflow and per-node raw records (task §17), call accounting (task
// §17/G), and runtime-derived call counts (never taken from the constants
// directly).
type RunResult struct {
	WorkflowRecords []WorkflowRecord
	NodeRecords     []NodeCallRecord
	Accounting      RunAccounting
	ArmACalls       int
	ArmBCalls       int
	ArmCCalls       int
	TotalCalls      int
}

// RunAll executes every workflow through all three arms. compositeImages
// and operandImages must be the exact startup-verified byte bundle (per
// StartupSweepResult / correction I) -- RunAll does not materialize or
// re-resolve images itself.
func (r *CrossArmRunner) RunAll(ctx context.Context, runID string, workflows []Workflow, compositeImages map[string][]byte, operandImagesByWorkflow map[string]map[string][]byte) (RunResult, error) {
	if r.ArmA == nil || r.ArmB == nil || r.ArmC == nil {
		return RunResult{}, fmt.Errorf("tonalt1arms: CrossArmRunner: all three arm executors are required")
	}

	var result RunResult
	for _, wf := range workflows {
		operandImages := operandImagesByWorkflow[wf.WorkflowID]

		armARec, armANode, err := r.ArmA.ExecuteWorkflow(ctx, runID, wf, compositeImages[wf.WorkflowID])
		if err != nil {
			return RunResult{}, fmt.Errorf("Arm A %s: %w", wf.WorkflowID, err)
		}
		result.WorkflowRecords = append(result.WorkflowRecords, armARec)
		result.NodeRecords = append(result.NodeRecords, armANode)

		armBRec, armBNodes, _, err := r.ArmB.ExecuteWorkflow(ctx, runID, wf, operandImages)
		if err != nil {
			return RunResult{}, fmt.Errorf("Arm B %s: %w", wf.WorkflowID, err)
		}
		result.WorkflowRecords = append(result.WorkflowRecords, armBRec)
		result.NodeRecords = append(result.NodeRecords, armBNodes...)

		armCRec, armCNodes, _, err := r.ArmC.ExecuteWorkflow(ctx, runID, wf, operandImages)
		if err != nil {
			return RunResult{}, fmt.Errorf("Arm C %s: %w", wf.WorkflowID, err)
		}
		result.WorkflowRecords = append(result.WorkflowRecords, armCRec)
		result.NodeRecords = append(result.NodeRecords, armCNodes...)
	}

	plannedSlots, err := totalPlannedSlots(workflows)
	if err != nil {
		return RunResult{}, err
	}
	result.Accounting = deriveAccounting(result.NodeRecords)
	result.Accounting.PlannedModelCallSlots = plannedSlots
	result.ArmACalls = countCallsForArm(result.NodeRecords, "A")
	result.ArmBCalls = countCallsForArm(result.NodeRecords, "B")
	result.ArmCCalls = countCallsForArm(result.NodeRecords, "C")
	result.TotalCalls = result.ArmACalls + result.ArmBCalls + result.ArmCCalls
	return result, nil
}

// totalPlannedSlots sums the static per-workflow generative-node-slot count
// (armA + armB + armC) across all given workflows -- this is
// PLANNED_MODEL_CALL_SLOTS, independent of what actually happened at
// runtime (task correction G).
func totalPlannedSlots(workflows []Workflow) (int, error) {
	total := 0
	for _, wf := range workflows {
		a, b, c, err := plannedSlotsForShape(wf.Shape)
		if err != nil {
			return 0, err
		}
		total += a + b + c
	}
	return total, nil
}

// countCallsForArm counts NodeCallRecords for the given arm whose
// TransportStatus is "OK" (an actual completed call) -- this is the
// runtime-derived count the task requires, not a static constant.
func countCallsForArm(records []NodeCallRecord, arm string) int {
	count := 0
	for _, rec := range records {
		if rec.Arm == arm && rec.TransportStatus == "OK" {
			count++
		}
	}
	return count
}

// plannedSlotsForShape is the static per-workflow generative-node-slot count
// for one shape, derived directly from the shared DAG's own capability
// classification (EXTRACT_NUMBER/NORMALIZE/COMPARE_NUMBERS/ARITHMETIC are
// the Parrot-eligible slots in Arm B; only EXTRACT_NUMBER in Arm C; exactly
// 1 in Arm A) -- not a flat average, and not read from any policy file
// (this mirrors, and is cross-checked against, DeriveArmBCallBudget's own
// classification in callbudget.go).
func plannedSlotsForShape(shape string) (armA, armB, armC int, err error) {
	dag, err := BuildShapeDAG(shape)
	if err != nil {
		return 0, 0, 0, err
	}
	armA = 1
	for _, step := range dag.Steps {
		if step.Capability == "EXTRACT_NUMBER" {
			armC++
		}
		switch step.Capability {
		case "EXTRACT_NUMBER", "NORMALIZE", "COMPARE_NUMBERS", "ARITHMETIC":
			armB++
		}
	}
	return armA, armB, armC, nil
}

// deriveAccounting computes the DYNAMIC fields of RunAccounting from the
// actual node-record trace. PlannedModelCallSlots (STATIC, independent of
// what happened) is set separately by the caller.
func deriveAccounting(records []NodeCallRecord) RunAccounting {
	var acc RunAccounting
	for _, rec := range records {
		switch {
		case rec.ContractStatus == "BLOCKED_BY_DEPENDENCY":
			acc.BlockedByDependency++
		case rec.TransportStatus == "NOT_ATTEMPTED":
			// image-hash mismatch or other pre-call refusal: not a planned
			// HTTP attempt, not a blocked-by-dependency either.
		case rec.TransportStatus == "FAILED":
			acc.HTTPRequestAttempts++
			acc.TransportFailures++
		case rec.TransportStatus == "OK" && rec.SchemaStatus == "FAILED":
			acc.HTTPRequestAttempts++
			acc.SchemaFailures++
		case rec.TransportStatus == "OK" && rec.ContractStatus != "OK":
			acc.HTTPRequestAttempts++
			acc.ModelContractFailures++
		case rec.TransportStatus == "OK" && rec.ContractStatus == "OK":
			acc.HTTPRequestAttempts++
			acc.ValidCompletions++
		}
	}
	return acc
}

// Freeze writes the primary raw records to outDir, refusing to write a
// partial/incomplete run: exactly 180 WorkflowRecords (60 workflows x 3
// arms) and 696 accountable planned slots are required for the frozen
// primary campaign (task §18/§11). A run with a different workflow count
// (e.g. an offline test using a handful of fixture workflows) legitimately
// will not satisfy this and should not call Freeze -- it is reserved for
// the actual 60-workflow primary campaign.
func (r RunResult) Freeze(outDir string) error {
	if len(r.WorkflowRecords) != 180 {
		return fmt.Errorf("tonalt1arms: RunResult.Freeze: refusing to freeze %d workflow records, want exactly 180 (60 workflows x 3 arms)", len(r.WorkflowRecords))
	}
	if r.Accounting.PlannedModelCallSlots != 696 {
		return fmt.Errorf("tonalt1arms: RunResult.Freeze: refusing to freeze a run whose PlannedModelCallSlots is %d, want exactly 696", r.Accounting.PlannedModelCallSlots)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("tonalt1arms: RunResult.Freeze: %w", err)
	}
	if err := writeJSONFile(filepath.Join(outDir, "workflow_records.json"), r.WorkflowRecords); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(outDir, "node_call_records.json"), r.NodeRecords); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(outDir, "run_accounting.json"), r.Accounting); err != nil {
		return err
	}
	return nil
}

func writeJSONFile(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("tonalt1arms: writeJSONFile(%s): %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("tonalt1arms: writeJSONFile(%s): %w", path, err)
	}
	return nil
}
