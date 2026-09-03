package exocortex

import (
	"context"
	"encoding/json"
	"testing"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func obsEntry(t *testing.T, key, value, recordedAt string) blackboard.Entry {
	t.Helper()
	return obsEntryForRun(t, "r", key, value, recordedAt)
}

func obsEntryForRun(t *testing.T, runID, key, value, recordedAt string) blackboard.Entry {
	t.Helper()
	e := blackboard.Entry{Schema: blackboard.EntrySchema, Type: blackboard.EntryObservation, RunID: runID, TaskID: "t", NodeID: "n", WorkerID: "w", Key: key, Value: []byte(value), RecordedAt: recordedAt}
	id, err := blackboard.ContentID(e)
	if err != nil {
		t.Fatalf("ContentID: %v", err)
	}
	e.ID = id
	return e
}

func TestVerifyTlaloque_PromotesInRangeNumberToVerifiedFact(t *testing.T) {
	worker := VerifyTlaloque{}
	snapshot := blackboard.Snapshot{Schema: blackboard.SnapshotSchema, RunID: "r", Entries: []blackboard.Entry{
		obsEntry(t, "count", `"126"`, "2026-01-01T00:00:00Z"),
	}}
	min := 0.0
	max := 1000.0
	input, _ := json.Marshal(VerifyInput{TargetKey: "count", FactID: "count", ExpectedType: TargetTypeNumber, MinValue: &min, MaxValue: &max})
	resp, err := worker.Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "v1", Input: input, Blackboard: &snapshot})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(resp.Observations) != 1 || resp.Observations[0].Key != "fact.count" {
		t.Fatalf("expected one fact.count observation, got %+v", resp.Observations)
	}
	var fact blackboard.Fact
	if err := json.Unmarshal(resp.Observations[0].Value, &fact); err != nil {
		t.Fatalf("decode fact: %v", err)
	}
	if fact.Status != blackboard.FactVerified {
		t.Fatalf("status = %q, want VERIFIED", fact.Status)
	}
	if len(fact.DerivedFrom) != 1 {
		t.Fatalf("derived_from = %v, want exactly one source observation", fact.DerivedFrom)
	}
}

func TestVerifyTlaloque_OutOfRangeIsUnsupportedNotFabricated(t *testing.T) {
	worker := VerifyTlaloque{}
	snapshot := blackboard.Snapshot{Schema: blackboard.SnapshotSchema, RunID: "r", Entries: []blackboard.Entry{
		obsEntry(t, "count", `"99999"`, "2026-01-01T00:00:00Z"),
	}}
	max := 1000.0
	input, _ := json.Marshal(VerifyInput{TargetKey: "count", ExpectedType: TargetTypeNumber, MaxValue: &max})
	resp, err := worker.Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "v1", Input: input, Blackboard: &snapshot})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var fact blackboard.Fact
	json.Unmarshal(resp.Observations[0].Value, &fact)
	if fact.Status != blackboard.FactUnsupported {
		t.Fatalf("status = %q, want UNSUPPORTED for an out-of-range value", fact.Status)
	}
}

func TestVerifyTlaloque_UnparseableNumberIsUnsupported(t *testing.T) {
	worker := VerifyTlaloque{}
	snapshot := blackboard.Snapshot{Schema: blackboard.SnapshotSchema, RunID: "r", Entries: []blackboard.Entry{
		obsEntry(t, "count", `"garbled response"`, "2026-01-01T00:00:00Z"),
	}}
	input, _ := json.Marshal(VerifyInput{TargetKey: "count", ExpectedType: TargetTypeNumber})
	resp, err := worker.Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "v1", Input: input, Blackboard: &snapshot})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var fact blackboard.Fact
	json.Unmarshal(resp.Observations[0].Value, &fact)
	if fact.Status != blackboard.FactUnsupported {
		t.Fatalf("status = %q, want UNSUPPORTED for unparseable output", fact.Status)
	}
}

func TestVerifyTlaloque_ErrorsWhenNoObservationExists(t *testing.T) {
	worker := VerifyTlaloque{}
	snapshot := blackboard.Snapshot{Schema: blackboard.SnapshotSchema, RunID: "r"}
	input, _ := json.Marshal(VerifyInput{TargetKey: "missing", ExpectedType: TargetTypeNumber})
	if _, err := worker.Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "v1", Input: input, Blackboard: &snapshot}); err == nil {
		t.Fatalf("expected an error when no observation exists to verify")
	}
}

func TestVerifyTlaloque_ChoiceMustBeInAllowedSet(t *testing.T) {
	worker := VerifyTlaloque{}
	snapshot := blackboard.Snapshot{Schema: blackboard.SnapshotSchema, RunID: "r", Entries: []blackboard.Entry{
		obsEntry(t, "choice", `"maybe"`, "2026-01-01T00:00:00Z"),
	}}
	input, _ := json.Marshal(VerifyInput{TargetKey: "choice", ExpectedType: TargetTypeChoice, AllowedChoices: []string{"SAME", "DIFFERENT"}})
	resp, err := worker.Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "v1", Input: input, Blackboard: &snapshot})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var fact blackboard.Fact
	json.Unmarshal(resp.Observations[0].Value, &fact)
	if fact.Status != blackboard.FactUnsupported {
		t.Fatalf("status = %q, want UNSUPPORTED for a choice outside allowed set", fact.Status)
	}
}

func TestVerifyTlaloque_RecordNode_ReclassifiesFactObservationAsEntryFact(t *testing.T) {
	store := blackboard.New(t.TempDir())
	registry := tlaloque.NewRegistry()
	if err := registry.Register(VerifyTlaloque{}); err != nil {
		t.Fatalf("register: %v", err)
	}
	// Seed a prior observation; the verify step reads it back through the
	// SwarmRunner's own blackboard snapshot for this run.
	seed := obsEntryForRun(t, "run-verify", "count", `"42"`, "2026-01-01T00:00:00Z")
	if _, _, err := store.Append(seed); err != nil {
		t.Fatalf("seed append: %v", err)
	}
	runner := tlaloque.SwarmRunner{Registry: registry, Blackboard: &tlaloque.BlackboardRuntime{Store: store, RunID: "run-verify"}}
	plan, err := tlaloque.SwarmPlan{ID: "verify-plan", Nodes: []tlaloque.SwarmNode{{ID: "v1", Capability: OpVerify}}}.Normalize()
	if err != nil {
		t.Fatalf("normalize plan: %v", err)
	}
	input, _ := json.Marshal(VerifyInput{TargetKey: "count", ExpectedType: TargetTypeNumber})
	report, err := runner.Run(context.Background(), plan, "verify-task", input)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !report.Succeeded {
		t.Fatalf("report did not succeed: %+v", report)
	}
	snap, err := store.Snapshot("run-verify")
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	found := false
	for _, e := range snap.Entries {
		if e.Type == blackboard.EntryFact && e.Key == "fact.count" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a FACT entry keyed fact.count to be persisted, got %+v", snap.Entries)
	}
}
