package exocortex

import (
	"encoding/json"
	"testing"

	"tlaloc.local/behaviorlab/internal/blackboard"
)

func TestWorkingSetBuilder_ResolvesOnlyDeclaredInputs(t *testing.T) {
	snapshot := &blackboard.Snapshot{
		Schema: blackboard.SnapshotSchema,
		RunID:  "run1",
		Entries: []blackboard.Entry{
			{Type: blackboard.EntryObservation, RunID: "run1", TaskID: "t", NodeID: "n1", WorkerID: "w1", Key: "region_r3", Value: []byte(`"crop-path"`), RecordedAt: "2026-01-01T00:00:00Z"},
			{Type: blackboard.EntryObservation, RunID: "run1", TaskID: "t", NodeID: "n2", WorkerID: "w2", Key: "unrelated_history", Value: []byte(`"noise"`), RecordedAt: "2026-01-01T00:00:01Z"},
			{Type: blackboard.EntryFact, RunID: "run1", TaskID: "t", NodeID: "n3", WorkerID: "verify", Key: "fact.some_other_fact", Value: []byte(`{"fact_id":"some_other_fact","status":"VERIFIED","value":1,"derived_from":["x"]}`), RecordedAt: "2026-01-01T00:00:02Z"},
		},
	}
	step := Step{ID: "s2", Opcode: OpExtractNumber, Inputs: []Address{"region_r3"}, Output: "observation://count"}
	built, err := WorkingSetBuilder{}.Build(snapshot, step, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(built) != 1 {
		t.Fatalf("working set has %d entries, want exactly 1 (no workflow history, no unrelated facts): %v", len(built), built)
	}
	var value string
	if err := json.Unmarshal(built["region_r3"], &value); err != nil || value != "crop-path" {
		t.Fatalf("region_r3 = %s, want \"crop-path\"", built["region_r3"])
	}
}

func TestWorkingSetBuilder_PrefersFactOverRawObservation(t *testing.T) {
	snapshot := &blackboard.Snapshot{
		Schema: blackboard.SnapshotSchema,
		RunID:  "run1",
		Entries: []blackboard.Entry{
			{Type: blackboard.EntryObservation, RunID: "run1", TaskID: "t", NodeID: "n1", WorkerID: "parrot", Key: "count", Value: []byte(`"126 "`), RecordedAt: "2026-01-01T00:00:00Z"},
			{Type: blackboard.EntryFact, RunID: "run1", TaskID: "t", NodeID: "n2", WorkerID: "verify", Key: "fact.count", Value: []byte(`{"fact_id":"count","status":"VERIFIED","value":126,"derived_from":["x"]}`), RecordedAt: "2026-01-01T00:00:01Z"},
		},
	}
	step := Step{ID: "s3", Opcode: OpCompareNumbers, Inputs: []Address{"count"}, Output: "observation://cmp"}
	built, err := WorkingSetBuilder{}.Build(snapshot, step, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var fact blackboard.Fact
	if err := json.Unmarshal(built["count"], &fact); err != nil {
		t.Fatalf("expected the verified Fact, got raw observation %s: %v", built["count"], err)
	}
	if fact.Status != blackboard.FactVerified {
		t.Fatalf("status = %q, want VERIFIED", fact.Status)
	}
}

func TestWorkingSetBuilder_UsesResolvedOverrideForOutOfBandOperands(t *testing.T) {
	step := Step{ID: "s1", Opcode: OpLocateRegion, Inputs: []Address{"page176"}, Output: "region://page176/r3"}
	resolved := Resolved{"page176": json.RawMessage(`"/tmp/page176.png"`)}
	built, err := WorkingSetBuilder{}.Build(&blackboard.Snapshot{Schema: blackboard.SnapshotSchema, RunID: "run1"}, step, resolved)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if string(built["page176"]) != `"/tmp/page176.png"` {
		t.Fatalf("got %s", built["page176"])
	}
}

func TestWorkingSetBuilder_ErrorsWhenInputUnresolvable(t *testing.T) {
	step := Step{ID: "s1", Opcode: OpExtractNumber, Inputs: []Address{"missing"}, Output: "observation://x"}
	_, err := WorkingSetBuilder{}.Build(&blackboard.Snapshot{Schema: blackboard.SnapshotSchema, RunID: "run1"}, step, nil)
	if err == nil {
		t.Fatalf("expected an error for an unresolvable input")
	}
}
