package tlaloque

import "testing"

func empiricalWorker(id string, parameters int64) CapabilityWorker {
	return testWorker{desc: CapabilityDescriptor{
		ID:             id,
		Capability:     "CLASSIFY",
		Scope:          ScopeGeneral,
		Engine:         EngineModel,
		InputSchema:    "text",
		OutputSchema:   "class",
		ParameterCount: parameters,
	}}
}

func TestEmpiricalStrategyPrefersComplementaryWeakerMember(t *testing.T) {
	r := NewRegistry()
	for _, worker := range []CapabilityWorker{
		empiricalWorker("A", 1_000_000),
		empiricalWorker("B", 2_000_000),
		empiricalWorker("C", 3_000_000),
	} {
		if err := r.Register(worker); err != nil {
			t.Fatal(err)
		}
	}
	source, err := NewStaticEmpiricalSelectionSource(
		[]WorkerEmpiricalMetric{
			{WorkerID: "A", Cases: 100, Accuracy: 0.90},
			{WorkerID: "B", Cases: 100, Accuracy: 0.85},
			{WorkerID: "C", Cases: 100, Accuracy: 0.70},
		},
		[]WorkerPairEmpiricalMetric{
			{WorkerA: "A", WorkerB: "B", SharedCases: 100, Complementarity: 0.10},
			{WorkerA: "A", WorkerB: "C", SharedCases: 100, Complementarity: 0.90},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	r.SetSelectionStrategy(EmpiricalEnsembleSelectionStrategy{
		Source:                source,
		QualityWeight:         1,
		ComplementarityWeight: 1,
	})
	workers, err := r.SelectMany(SelectionRequest{Capability: "CLASSIFY"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	got := workerIDs(workers)
	if len(got) != 2 || got[0] != "A" || got[1] != "C" {
		t.Fatalf("ensemble=%v want [A C]", got)
	}
}

func TestEmpiricalStrategySingleSelectionPreservesFallback(t *testing.T) {
	r := NewRegistry()
	for _, worker := range []CapabilityWorker{
		empiricalWorker("A", 1_000_000),
		empiricalWorker("B", 2_000_000),
	} {
		if err := r.Register(worker); err != nil {
			t.Fatal(err)
		}
	}
	source, err := NewStaticEmpiricalSelectionSource(
		[]WorkerEmpiricalMetric{
			{WorkerID: "A", Cases: 100, Accuracy: 0.10},
			{WorkerID: "B", Cases: 100, Accuracy: 0.99},
		}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	r.SetSelectionStrategy(EmpiricalEnsembleSelectionStrategy{Source: source})
	worker, err := r.Select(SelectionRequest{Capability: "CLASSIFY"})
	if err != nil {
		t.Fatal(err)
	}
	if worker.Descriptor().ID != "A" {
		t.Fatalf("single routing changed to %s", worker.Descriptor().ID)
	}
}

func TestEmpiricalStrategyRequiresEnoughSharedEvidence(t *testing.T) {
	r := NewRegistry()
	for _, worker := range []CapabilityWorker{
		empiricalWorker("A", 1_000_000),
		empiricalWorker("B", 2_000_000),
		empiricalWorker("C", 3_000_000),
	} {
		if err := r.Register(worker); err != nil {
			t.Fatal(err)
		}
	}
	source, err := NewStaticEmpiricalSelectionSource(
		[]WorkerEmpiricalMetric{
			{WorkerID: "A", Cases: 100, Accuracy: 0.90},
			{WorkerID: "B", Cases: 100, Accuracy: 0.85},
			{WorkerID: "C", Cases: 100, Accuracy: 0.70},
		},
		[]WorkerPairEmpiricalMetric{
			{WorkerA: "A", WorkerB: "C", SharedCases: 2, Complementarity: 1.0},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	r.SetSelectionStrategy(EmpiricalEnsembleSelectionStrategy{
		Source:                source,
		MinSharedCases:        10,
		QualityWeight:         1,
		ComplementarityWeight: 1,
	})
	workers, err := r.SelectMany(SelectionRequest{Capability: "CLASSIFY"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	got := workerIDs(workers)
	if len(got) != 2 || got[0] != "A" || got[1] != "B" {
		t.Fatalf("insufficient evidence should preserve quality ordering: %v", got)
	}
}

func TestEmpiricalStrategyWithoutSourceMatchesBaseline(t *testing.T) {
	r := NewRegistry()
	for _, worker := range []CapabilityWorker{
		empiricalWorker("A", 1_000_000),
		empiricalWorker("B", 2_000_000),
		empiricalWorker("C", 3_000_000),
	} {
		if err := r.Register(worker); err != nil {
			t.Fatal(err)
		}
	}
	r.SetSelectionStrategy(EmpiricalEnsembleSelectionStrategy{})
	workers, err := r.SelectMany(SelectionRequest{Capability: "CLASSIFY"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	got := workerIDs(workers)
	want := []string{"A", "B", "C"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got=%v want=%v", got, want)
		}
	}
}

func TestStaticEmpiricalSourceIsSymmetricAndValidated(t *testing.T) {
	source, err := NewStaticEmpiricalSelectionSource(
		[]WorkerEmpiricalMetric{{WorkerID: "A", Cases: 10, Accuracy: 0.8}},
		[]WorkerPairEmpiricalMetric{{WorkerA: "B", WorkerB: "A", SharedCases: 10, Complementarity: 0.6}},
	)
	if err != nil {
		t.Fatal(err)
	}
	forward, okForward := source.PairMetric("A", "B")
	reverse, okReverse := source.PairMetric("B", "A")
	if !okForward || !okReverse || forward.Complementarity != reverse.Complementarity {
		t.Fatalf("forward=%+v/%v reverse=%+v/%v", forward, okForward, reverse, okReverse)
	}
	if _, err := NewStaticEmpiricalSelectionSource(
		[]WorkerEmpiricalMetric{{WorkerID: "A", Cases: 1, Accuracy: 1.2}}, nil,
	); err == nil {
		t.Fatal("expected invalid accuracy")
	}
	if _, err := NewStaticEmpiricalSelectionSource(nil,
		[]WorkerPairEmpiricalMetric{{WorkerA: "A", WorkerB: "A", SharedCases: 1, Complementarity: 0.5}},
	); err == nil {
		t.Fatal("expected self pair rejection")
	}
}
