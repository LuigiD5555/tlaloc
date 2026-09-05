package episode

import (
	"testing"

	"tlaloc.local/behaviorlab/internal/tonalt1arms"
)

func TestFromT1Workflow_MapsMatchingNodesOnly(t *testing.T) {
	wf := tonalt1arms.WorkflowRecord{
		RunID:            "run-1",
		WorkflowID:       "wf-001",
		Family:           "DIFF_RATIO",
		Arm:              "B",
		TerminalStatus:   "DONE",
		SemanticCorrect:  true,
		ExactCorrect:     true,
		ContractStatus:   "OK",
		TotalParrotCalls: 2,
		LatencyMS:        150,
	}

	nodeRecords := []tonalt1arms.NodeCallRecord{
		{
			RunID: "run-1", WorkflowID: "wf-001", Arm: "B",
			NodeID: "n2", Capability: "EXTRACT_NUMBER", Operation: "EXTRACT_NUMBER",
			ExecutorID: "parrot", RequestIndex: 1,
			TransportStatus: "OK", SchemaStatus: "OK", ContractStatus: "OK", LatencyMS: 80,
		},
		{
			RunID: "run-1", WorkflowID: "wf-001", Arm: "B",
			NodeID: "n1", Capability: "EXTRACT_NUMBER", Operation: "EXTRACT_NUMBER",
			ExecutorID: "parrot", RequestIndex: 0,
			TransportStatus: "OK", SchemaStatus: "OK", ContractStatus: "OK", LatencyMS: 70,
		},
		{
			// different arm for the same workflow: must be excluded.
			RunID: "run-1", WorkflowID: "wf-001", Arm: "C",
			NodeID: "n1", Capability: "EXTRACT_NUMBER", Operation: "EXTRACT_NUMBER",
			ExecutorID: "parrot", RequestIndex: 0,
			TransportStatus: "OK", SchemaStatus: "OK", ContractStatus: "OK", LatencyMS: 999,
		},
		{
			// different workflow, same arm: must be excluded.
			RunID: "run-1", WorkflowID: "wf-002", Arm: "B",
			NodeID: "n1", Capability: "EXTRACT_NUMBER", Operation: "EXTRACT_NUMBER",
			ExecutorID: "parrot", RequestIndex: 0,
			TransportStatus: "OK", SchemaStatus: "OK", ContractStatus: "OK", LatencyMS: 999,
		},
	}

	got := FromT1Workflow(wf, nodeRecords)

	if got.Schema != Schema {
		t.Errorf("Schema = %q, want %q", got.Schema, Schema)
	}
	if got.SourceExperiment != SourceT1 {
		t.Errorf("SourceExperiment = %q, want %q", got.SourceExperiment, SourceT1)
	}
	if got.EpisodeID != "t1-run-1-wf-001-B" {
		t.Errorf("EpisodeID = %q, want %q", got.EpisodeID, "t1-run-1-wf-001-B")
	}
	if got.TaskID != "wf-001/B" {
		t.Errorf("TaskID = %q, want %q", got.TaskID, "wf-001/B")
	}
	if !got.Success {
		t.Errorf("Success = false, want true (SemanticCorrect was true)")
	}
	if len(got.Steps) != 2 {
		t.Fatalf("len(Steps) = %d, want 2 (only wf-001/B nodes)", len(got.Steps))
	}
	// Deterministic ordering by RequestIndex, not input slice order.
	if got.Steps[0].NodeID != "n1" || got.Steps[1].NodeID != "n2" {
		t.Errorf("Steps order = [%s, %s], want [n1, n2]", got.Steps[0].NodeID, got.Steps[1].NodeID)
	}
	if got.Cost.ModelCalls != 2 {
		t.Errorf("Cost.ModelCalls = %d, want 2", got.Cost.ModelCalls)
	}
	if got.Cost.LatencyMS != 150 {
		t.Errorf("Cost.LatencyMS = %d, want 150", got.Cost.LatencyMS)
	}
}

func TestFromT1Workflow_FailedTransportNotCountedAsModelCall(t *testing.T) {
	wf := tonalt1arms.WorkflowRecord{RunID: "run-1", WorkflowID: "wf-001", Arm: "A", SemanticCorrect: false}
	nodeRecords := []tonalt1arms.NodeCallRecord{
		{
			RunID: "run-1", WorkflowID: "wf-001", Arm: "A",
			NodeID: "n1", TransportStatus: "FAILED", ContractStatus: "BLOCKED_BY_DEPENDENCY",
		},
	}

	got := FromT1Workflow(wf, nodeRecords)

	if got.Success {
		t.Errorf("Success = true, want false (SemanticCorrect was false)")
	}
	if got.Cost.ModelCalls != 0 {
		t.Errorf("Cost.ModelCalls = %d, want 0 (TransportStatus was FAILED, not OK)", got.Cost.ModelCalls)
	}
	if got.Steps[0].Status != "BLOCKED_BY_DEPENDENCY" {
		t.Errorf("Steps[0].Status = %q, want %q", got.Steps[0].Status, "BLOCKED_BY_DEPENDENCY")
	}
}

func TestFromT1RunResult_OneEpisodePerWorkflowRecord(t *testing.T) {
	result := tonalt1arms.RunResult{
		WorkflowRecords: []tonalt1arms.WorkflowRecord{
			{RunID: "run-1", WorkflowID: "wf-001", Arm: "A"},
			{RunID: "run-1", WorkflowID: "wf-001", Arm: "B"},
			{RunID: "run-1", WorkflowID: "wf-002", Arm: "A"},
		},
	}

	episodes := FromT1RunResult(result)

	if len(episodes) != 3 {
		t.Fatalf("len(episodes) = %d, want 3", len(episodes))
	}
	wantIDs := []string{"t1-run-1-wf-001-A", "t1-run-1-wf-001-B", "t1-run-1-wf-002-A"}
	for index, wantID := range wantIDs {
		if episodes[index].EpisodeID != wantID {
			t.Errorf("episodes[%d].EpisodeID = %q, want %q", index, episodes[index].EpisodeID, wantID)
		}
	}
}
