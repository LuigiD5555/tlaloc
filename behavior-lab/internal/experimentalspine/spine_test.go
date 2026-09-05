package experimentalspine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tlaloc.local/behaviorlab/internal/episode"
	"tlaloc.local/behaviorlab/internal/tonalt1arms"
)

func TestSummarizeFindsFailureFrontier(t *testing.T) {
	manifest := RunManifest{
		Schema: ManifestSchema, RunID: "run-1", SourceExperiment: "PROTO_X",
		Prototype: Prototype{ID: "PROTO_X", Version: "0.2", ParentVersion: "0.1"},
	}
	episodes := []episode.Episode{
		{
			Schema: episode.Schema, EpisodeID: "ep-1", SourceExperiment: "PROTO_X", RunID: "run-1",
			TaskID: "task-1", Arm: "A", Family: "F1", Success: true, SemanticCorrect: true, ExactCorrect: true,
			Cost: episode.Cost{ModelCalls: 1, HTTPRequestAttempts: 1, ValidCompletions: 1, LatencyMS: 10},
			Steps: []episode.Step{{SelectedCapability: "EXTRACT", TransportStatus: "OK", SchemaStatus: "OK", ContractStatus: "OK", LatencyMS: 10}},
		},
		{
			Schema: episode.Schema, EpisodeID: "ep-2", SourceExperiment: "PROTO_X", RunID: "run-1",
			TaskID: "task-2", Arm: "B", Family: "F1", Success: false, SemanticCorrect: false, ExactCorrect: false,
			FailureRootCause: "MODEL_CONTRACT_FAILURE",
			Cost: episode.Cost{ModelCalls: 1, HTTPRequestAttempts: 1, ModelContractFailures: 1, BlockedByDependency: 1, LatencyMS: 30},
			Steps: []episode.Step{
				{SelectedCapability: "COMPARE", TransportStatus: "OK", SchemaStatus: "OK", ContractStatus: "BAD_OUTPUT", LatencyMS: 30},
				{SelectedCapability: "ARITHMETIC", TransportStatus: "NOT_ATTEMPTED", ContractStatus: "BLOCKED_BY_DEPENDENCY", LatencyMS: 0},
			},
		},
	}

	got := Summarize(manifest, episodes)
	if got.Episodes != 2 || got.Successful != 1 || got.Failed != 1 {
		t.Fatalf("episode totals = %+v", got)
	}
	if got.SemanticAccuracy != 0.5 || got.ExactAccuracy != 0.5 {
		t.Errorf("accuracy semantic/exact = %v/%v, want 0.5/0.5", got.SemanticAccuracy, got.ExactAccuracy)
	}
	if got.Cost.HTTPRequestAttempts != 2 || got.Cost.ValidCompletions != 1 || got.Cost.ModelContractFailures != 1 || got.Cost.BlockedByDependency != 1 {
		t.Errorf("cost = %+v", got.Cost)
	}
	if got.MostFailedCapability != "COMPARE" || got.NextDebugTarget != "capability:COMPARE" {
		t.Errorf("debug target = %q/%q, want COMPARE/capability:COMPARE", got.MostFailedCapability, got.NextDebugTarget)
	}
	for _, capability := range got.ByCapability {
		if capability.Capability == "ARITHMETIC" {
			if capability.FailedSteps != 0 || capability.BlockedSteps != 1 {
				t.Errorf("blocked ARITHMETIC = %+v, want failed=0 blocked=1", capability)
			}
		}
	}
	if got.Latency.P50MS != 10 || got.Latency.P95MS != 30 || got.Latency.MaxMS != 30 {
		t.Errorf("latency = %+v, want p50=10 p95=30 max=30", got.Latency)
	}
}

func TestWriteBundleIsImmutable(t *testing.T) {
	out := t.TempDir()
	observedAt := time.Date(2026, 9, 4, 20, 0, 0, 0, time.UTC)
	manifest := RunManifest{
		Schema: ManifestSchema, RunID: "run-immutable", SourceExperiment: "PROTO_X",
		Prototype: Prototype{ID: "PROTO_X", Version: "0.1"},
	}
	episodes := []episode.Episode{{
		Schema: episode.Schema, EpisodeID: "ep-immutable", SourceExperiment: "PROTO_X", RunID: "run-immutable", TaskID: "task-1",
	}}

	paths, err := WriteBundle(out, manifest, episodes, observedAt)
	if err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}
	for _, path := range []string{paths.Manifest, paths.Summary} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(paths.EpisodesRoot, "2026-09", "ep-immutable.json")); err != nil {
		t.Errorf("expected stored episode: %v", err)
	}
	if _, err := WriteBundle(out, manifest, episodes, observedAt); err == nil {
		t.Fatal("second WriteBundle unexpectedly overwrote immutable experience bundle")
	}
}

func TestWriteT1BundleMatchesRawAccounting(t *testing.T) {
	out := t.TempDir()
	observedAt := time.Date(2026, 9, 4, 20, 0, 0, 0, time.UTC)
	manifest := RunManifest{
		Schema: ManifestSchema, RunID: "t1-run", SourceExperiment: episode.SourceT1,
		Prototype: Prototype{ID: "TONAL_T1", Version: "T1"},
	}
	result := tonalt1arms.RunResult{
		WorkflowRecords: []tonalt1arms.WorkflowRecord{{
			RunID: "t1-run", WorkflowID: "wf-001", Family: "SINGLE", Arm: "A",
			SemanticCorrect: true, ExactCorrect: true, ContractStatus: "OK",
		}},
		NodeRecords: []tonalt1arms.NodeCallRecord{{
			RunID: "t1-run", WorkflowID: "wf-001", Arm: "A", NodeID: "n1",
			Capability: "EXTRACT_NUMBER", Operation: "EXTRACT_NUMBER", RequestIndex: 0,
			TransportStatus: "OK", SchemaStatus: "OK", ContractStatus: "OK", LatencyMS: 12,
		}},
		Accounting: tonalt1arms.RunAccounting{
			PlannedModelCallSlots: 1,
			HTTPRequestAttempts:   1,
			ValidCompletions:      1,
		},
	}

	paths, err := WriteT1Bundle(out, manifest, result, observedAt)
	if err != nil {
		t.Fatalf("WriteT1Bundle: %v", err)
	}
	body, err := os.ReadFile(paths.Summary)
	if err != nil {
		t.Fatal(err)
	}
	var summary Summary
	if err := json.Unmarshal(body, &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Cost.PlannedModelCallSlots != 1 || summary.Cost.HTTPRequestAttempts != 1 || summary.Cost.ValidCompletions != 1 {
		t.Errorf("T1 summary cost = %+v", summary.Cost)
	}
}

func TestWriteT1BundleRejectsAccountingDrift(t *testing.T) {
	manifest := RunManifest{
		Schema: ManifestSchema, RunID: "t1-run", SourceExperiment: episode.SourceT1,
		Prototype: Prototype{ID: "TONAL_T1"},
	}
	result := tonalt1arms.RunResult{
		WorkflowRecords: []tonalt1arms.WorkflowRecord{{RunID: "t1-run", WorkflowID: "wf-001", Arm: "A"}},
		NodeRecords: []tonalt1arms.NodeCallRecord{{
			RunID: "t1-run", WorkflowID: "wf-001", Arm: "A", NodeID: "n1",
			TransportStatus: "FAILED",
		}},
		Accounting: tonalt1arms.RunAccounting{HTTPRequestAttempts: 0}, // deliberately wrong: raw node says one attempt.
	}

	_, err := WriteT1Bundle(t.TempDir(), manifest, result, time.Now())
	if err == nil {
		t.Fatal("WriteT1Bundle accepted accounting drift")
	}
}
