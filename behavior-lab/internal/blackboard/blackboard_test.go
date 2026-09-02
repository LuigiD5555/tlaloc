package blackboard

import (
	"encoding/json"
	"sync"
	"testing"
)

func entry(run, node, worker, key, value string) Entry {
	return Entry{Type: EntryObservation, RunID: run, TaskID: "task", NodeID: node, WorkerID: worker, Key: key, Value: json.RawMessage(value), Confidence: 0.9}
}

func TestContentIDStableAndAppendIdempotent(t *testing.T) {
	s := New(t.TempDir())
	e := entry("run-1", "n1", "w1", "state", `{"value":"ACTIVE"}`)
	id1, err := ContentID(e)
	if err != nil {
		t.Fatal(err)
	}
	e.RecordedAt = "2099-01-01T00:00:00Z"
	id2, err := ContentID(e)
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Fatalf("unstable ids: %s != %s", id1, id2)
	}
	added, stored, err := s.Append(e)
	if err != nil || !added {
		t.Fatalf("added=%v err=%v", added, err)
	}
	added, again, err := s.Append(e)
	if err != nil || added {
		t.Fatalf("second added=%v err=%v", added, err)
	}
	if stored.ID != again.ID {
		t.Fatalf("ids differ: %s %s", stored.ID, again.ID)
	}
}

func TestAppendPublishesOneAtomicEntryUnderConcurrency(t *testing.T) {
	s := New(t.TempDir())
	e := entry("run-1", "n1", "w1", "state", `{"value":"ACTIVE"}`)
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _, err := s.Append(e); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	all, err := s.LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("entries=%d want 1", len(all))
	}
}

func TestSnapshotIsolatedByRunAndSortedByID(t *testing.T) {
	s := New(t.TempDir())
	for _, e := range []Entry{
		entry("run-a", "n2", "w", "b", `2`),
		entry("run-b", "n1", "w", "x", `9`),
		entry("run-a", "n1", "w", "a", `1`),
	} {
		if _, _, err := s.Append(e); err != nil {
			t.Fatal(err)
		}
	}
	snap, err := s.Snapshot("run-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Entries) != 2 {
		t.Fatalf("entries=%d", len(snap.Entries))
	}
	if snap.Entries[0].ID > snap.Entries[1].ID {
		t.Fatal("snapshot is not sorted by ID")
	}
	for _, e := range snap.Entries {
		if e.RunID != "run-a" {
			t.Fatalf("leaked run %s", e.RunID)
		}
	}
}

func TestConsolidateRequiresTwoThirdsAndPreservesConflicts(t *testing.T) {
	s := New(t.TempDir())
	for i, raw := range []string{`{"state":"ACTIVE"}`, `{"state":"ACTIVE"}`, `{"state":"IDLE"}`} {
		e := entry("run", string(rune('a'+i)), "w", "state", raw)
		if _, _, err := s.Append(e); err != nil {
			t.Fatal(err)
		}
	}
	snap, err := s.Snapshot("run")
	if err != nil {
		t.Fatal(err)
	}
	c, err := Consolidate(snap, "state", nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != ConsensusConfirmed || c.Votes != 2 || c.RequiredVotes != 2 {
		t.Fatalf("consensus=%+v", c)
	}
	if len(snap.Entries) != 3 {
		t.Fatalf("contradiction was lost: %d", len(snap.Entries))
	}
}

func TestConsolidateTieOrContractViolationIsUnknown(t *testing.T) {
	snap := Snapshot{Schema: SnapshotSchema, RunID: "run", Entries: []Entry{
		{Schema: EntrySchema, ID: "1", Type: EntryObservation, RunID: "run", TaskID: "t", NodeID: "a", WorkerID: "w", Key: "k", Value: json.RawMessage(`"A"`)},
		{Schema: EntrySchema, ID: "2", Type: EntryObservation, RunID: "run", TaskID: "t", NodeID: "b", WorkerID: "w", Key: "k", Value: json.RawMessage(`"B"`)},
	}}
	c, err := Consolidate(snap, "k", nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != ConsensusUnknown {
		t.Fatalf("tie=%+v", c)
	}
	c, err = Consolidate(snap, "k", func(v json.RawMessage) bool { return string(v) == `"A"` })
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != ConsensusUnknown {
		t.Fatalf("contract=%+v", c)
	}
}
