package tlaloque

import "testing"

type lastCandidateStrategy struct{}

func (lastCandidateStrategy) Select(candidates []SelectionCandidate, _ SelectionRequest) Result[CapabilityWorker] {
	if len(candidates) == 0 {
		return DomainResult[CapabilityWorker](ResultNoCandidate, nil)
	}
	return Success(candidates[len(candidates)-1].Worker)
}

func TestRegistrySelectionStrategyCanBeReplaced(t *testing.T) {
	r := NewRegistry()
	for _, id := range []string{"a", "b"} {
		if err := r.Register(testWorker{desc: CapabilityDescriptor{ID: id, Capability: "CLASSIFY", Scope: ScopeGeneral, Engine: EngineModel, InputSchema: "text", OutputSchema: "label"}}); err != nil {
			t.Fatal(err)
		}
	}
	r.SetSelectionStrategy(lastCandidateStrategy{})
	worker, err := r.Select(SelectionRequest{Capability: "CLASSIFY"})
	if err != nil {
		t.Fatal(err)
	}
	if worker.Descriptor().ID != "b" {
		t.Fatalf("got %s", worker.Descriptor().ID)
	}
}

func TestSelectResultRepresentsNoCandidateAsDomainOutcome(t *testing.T) {
	r := NewRegistry()
	result := r.SelectResult(SelectionRequest{Capability: "MISSING"})
	if result.Err != nil {
		t.Fatalf("unexpected infrastructure error: %v", result.Err)
	}
	if result.Code != ResultNoCandidate {
		t.Fatalf("code=%s", result.Code)
	}
}

func TestCapabilityDescriptorNormalizesDataContracts(t *testing.T) {
	d, err := (CapabilityDescriptor{
		ID:           "intent",
		Capability:   "detect_intent",
		Scope:        ScopeGeneral,
		Engine:       EngineModel,
		InputSchema:  "text",
		OutputSchema: "intent",
		Requires:     []string{" input.text ", "input.text"},
		Produces:     []string{" claim.intent ", "claim.intent"},
	}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Requires) != 1 || d.Requires[0] != "input.text" {
		t.Fatalf("requires=%v", d.Requires)
	}
	if len(d.Produces) != 1 || d.Produces[0] != "claim.intent" {
		t.Fatalf("produces=%v", d.Produces)
	}
}
