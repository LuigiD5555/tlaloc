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
			ExecutorID: "parrot", Model: "lfm2-vl-1.6b", RequestIndex: 1,
			InputArtifact: "img-2.png", InputHash: "hash-2", RawOutput: "raw-2", ParsedOutput: "2",
			TransportStatus: "OK", SchemaStatus: "OK", ContractStatus: "OK", LatencyMS: 80,
		},
		{
			RunID: "run-1", WorkflowID: "wf-001", Arm: "B",
			NodeID: "n1", Capability: "EXTRACT_NUMBER", Operation: "EXTRACT_NUMBER",
			ExecutorID: "parrot", Model: "lfm2-vl-1.6b", RequestIndex: 0,
			InputArtifact: "img-1.png", InputHash: "hash-1", RawOutput: "raw-1", ParsedOutput: "1",
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
	if got.RunID != "run-1" || got.Arm != "B" || got.Family != "DIFF_RATIO" {
		t.Errorf("provenance = run=%q arm=%q family=%q, want run-1/B/DIFF_RATIO", got.RunID, got.Arm, got.Family)
	}
	if got.TaskID != "wf-001/B" {
		t.Errorf("TaskID = %q, want %q", got.TaskID, "wf-001/B")
	}
	if !got.Success || !got.SemanticCorrect || !got.ExactCorrect {
		t.Errorf("correctness = success:%v semantic:%v exact:%v, want all true", got.Success, got.SemanticCorrect, got.ExactCorrect)
	}
	if got.TerminalStatus != "DONE" || got.ContractStatus != "OK" {
		t.Errorf("terminal status = %q/%q, want DONE/OK", got.TerminalStatus, got.ContractStatus)
	}
	if got.FailureRootCause != "" {
		t.Errorf("FailureRootCause = %q, want empty for successful workflow", got.FailureRootCause)
	}
	if len(got.Steps) != 2 {
		t.Fatalf("len(Steps) = %d, want 2 (only wf-001/B nodes)", len(got.Steps))
	}
	// Deterministic ordering by RequestIndex, not input slice order.
	if got.Steps[0].NodeID != "n1" || got.Steps[1].NodeID != "n2" {
		t.Errorf("Steps order = [%s, %s], want [n1, n2]", got.Steps[0].NodeID, got.Steps[1].NodeID)
	}
	if got.Steps[0].RawOutput != "raw-1" || got.Steps[0].ParsedOutput != "1" || got.Steps[0].InputHash != "hash-1" {
		t.Errorf("Step evidence was not preserved: %+v", got.Steps[0])
	}
	if got.Steps[0].Model != "lfm2-vl-1.6b" || got.Steps[0].RequestIndex != 0 {
		t.Errorf("Step model/request index = %q/%d, want lfm2-vl-1.6b/0", got.Steps[0].Model, got.Steps[0].RequestIndex)
	}
	if got.Cost.ModelCalls != 2 {
		t.Errorf("Cost.ModelCalls = %d, want 2", got.Cost.ModelCalls)
	}
	if got.Cost.HTTPRequestAttempts != 2 || got.Cost.ValidCompletions != 2 {
		t.Errorf("Cost HTTP attempts/valid = %d/%d, want 2/2", got.Cost.HTTPRequestAttempts, got.Cost.ValidCompletions)
	}
	if got.Cost.LatencyMS != 150 {
		t.Errorf("Cost.LatencyMS = %d, want 150", got.Cost.LatencyMS)
	}
}

func TestFromT1Workflow_DependencyBlockedDoesNotCountAsHTTPRequest(t *testing.T) {
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
		t.Errorf("Cost.ModelCalls = %d, want 0", got.Cost.ModelCalls)
	}
	if got.Cost.HTTPRequestAttempts != 0 {
		t.Errorf("Cost.HTTPRequestAttempts = %d, want 0 for dependency-blocked node", got.Cost.HTTPRequestAttempts)
	}
	if got.Cost.BlockedByDependency != 1 {
		t.Errorf("Cost.BlockedByDependency = %d, want 1", got.Cost.BlockedByDependency)
	}
	if got.Steps[0].Status != "BLOCKED_BY_DEPENDENCY" {
		t.Errorf("Steps[0].Status = %q, want %q", got.Steps[0].Status, "BLOCKED_BY_DEPENDENCY")
	}
	// Semantic incorrectness is the terminal task-level explanation; the
	// blocked node itself is a consequence and remains available in Cost.
	if got.FailureRootCause != "SEMANTIC_INCORRECT" {
		t.Errorf("FailureRootCause = %q, want SEMANTIC_INCORRECT", got.FailureRootCause)
	}
}

func TestFromT1Workflow_FailedTransportCountsAttempt(t *testing.T) {
	wf := tonalt1arms.WorkflowRecord{RunID: "run-2", WorkflowID: "wf-002", Arm: "A", SemanticCorrect: false}
	nodeRecords := []tonalt1arms.NodeCallRecord{
		{
			RunID: "run-2", WorkflowID: "wf-002", Arm: "A",
			NodeID: "n1", TransportStatus: "FAILED", ContractStatus: "",
		},
	}

	got := FromT1Workflow(wf, nodeRecords)

	if got.Cost.ModelCalls != 0 {
		t.Errorf("Cost.ModelCalls = %d, want 0 because no transport completed", got.Cost.ModelCalls)
	}
	if got.Cost.HTTPRequestAttempts != 1 || got.Cost.TransportFailures != 1 {
		t.Errorf("attempts/transport failures = %d/%d, want 1/1", got.Cost.HTTPRequestAttempts, got.Cost.TransportFailures)
	}
	if got.FailureRootCause != "TRANSPORT_FAILURE" {
		t.Errorf("FailureRootCause = %q, want TRANSPORT_FAILURE", got.FailureRootCause)
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
