package tlaloque

import (
	"sort"
	"testing"
)

type reverseIDSelectionStrategy struct{}

func (reverseIDSelectionStrategy) Select(candidates []SelectionCandidate, req SelectionRequest) Result[CapabilityWorker] {
	if len(candidates) == 0 {
		return noEligibleWorkerResult(req)
	}
	rows := append([]SelectionCandidate(nil), candidates...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Desc.ID > rows[j].Desc.ID })
	return Success(rows[0].Worker)
}

func selectionTestWorker(id string, parameters int64, deterministic bool) CapabilityWorker {
	engine := EngineModel
	if deterministic {
		engine = EngineDeterministic
	}
	return testWorker{desc: CapabilityDescriptor{
		ID:             id,
		Capability:     "CLASSIFY",
		Scope:          ScopeGeneral,
		Engine:         engine,
		InputSchema:    "text",
		OutputSchema:   "class",
		Deterministic:  deterministic,
		ParameterCount: parameters,
	}}
}

func workerIDs(workers []CapabilityWorker) []string {
	ids := make([]string, 0, len(workers))
	for _, worker := range workers {
		ids = append(ids, worker.Descriptor().ID)
	}
	return ids
}

func TestSelectManyUsesSameRankedOrderingAsSingleSelection(t *testing.T) {
	r := NewRegistry()
	for _, worker := range []CapabilityWorker{
		selectionTestWorker("model-large", 40_000_000, false),
		selectionTestWorker("model-small", 5_000_000, false),
		selectionTestWorker("deterministic", 0, true),
	} {
		if err := r.Register(worker); err != nil {
			t.Fatal(err)
		}
	}
	req := SelectionRequest{Capability: "CLASSIFY", PreferDeterministic: true}
	one, err := r.Select(req)
	if err != nil {
		t.Fatal(err)
	}
	many, err := r.SelectMany(req, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(many) != 2 {
		t.Fatalf("selected=%v", workerIDs(many))
	}
	if many[0].Descriptor().ID != one.Descriptor().ID {
		t.Fatalf("SelectMany first=%s Select=%s", many[0].Descriptor().ID, one.Descriptor().ID)
	}
	want := []string{"deterministic", "model-small"}
	got := workerIDs(many)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ordering=%v want=%v", got, want)
		}
	}
}

func TestSelectManyLimitIsMaximumNotRequiredQuorum(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(selectionTestWorker("only", 1_000_000, false)); err != nil {
		t.Fatal(err)
	}
	result := r.SelectManyResult(SelectionRequest{Capability: "CLASSIFY"}, 5)
	if !result.OK() || len(result.Value) != 1 {
		t.Fatalf("result=%+v ids=%v", result, workerIDs(result.Value))
	}
}

func TestSelectManyPinnedWorkerDoesNotAddPeers(t *testing.T) {
	r := NewRegistry()
	for _, worker := range []CapabilityWorker{
		selectionTestWorker("a", 1, false),
		selectionTestWorker("b", 2, false),
	} {
		if err := r.Register(worker); err != nil {
			t.Fatal(err)
		}
	}
	workers, err := r.SelectMany(SelectionRequest{Capability: "CLASSIFY", WorkerID: "b"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := workerIDs(workers); len(got) != 1 || got[0] != "b" {
		t.Fatalf("pinned selection=%v", got)
	}
}

func TestSelectManyFallsBackToRepeatedSingleSelectionStrategy(t *testing.T) {
	r := NewRegistry()
	for _, id := range []string{"a", "b", "c"} {
		if err := r.Register(selectionTestWorker(id, 1, false)); err != nil {
			t.Fatal(err)
		}
	}
	r.SetSelectionStrategy(reverseIDSelectionStrategy{})
	workers, err := r.SelectMany(SelectionRequest{Capability: "CLASSIFY"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	got := workerIDs(workers)
	want := []string{"c", "b"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("fallback ordering=%v want=%v", got, want)
		}
	}
}

func TestSelectManyRejectsInvalidLimitAndMissingCandidates(t *testing.T) {
	r := NewRegistry()
	invalid := r.SelectManyResult(SelectionRequest{Capability: "CLASSIFY"}, 0)
	if invalid.Code != ResultInvalidRequest {
		t.Fatalf("invalid limit result=%+v", invalid)
	}
	missing := r.SelectManyResult(SelectionRequest{Capability: "MISSING"}, 2)
	if missing.Code != ResultNoCandidate {
		t.Fatalf("missing result=%+v", missing)
	}
}
