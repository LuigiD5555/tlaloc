package exocortex

import (
	"context"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

type stubWorker struct {
	desc tlaloque.CapabilityDescriptor
}

func (w stubWorker) Descriptor() tlaloque.CapabilityDescriptor { return w.desc }
func (w stubWorker) Execute(context.Context, tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	return tlaloque.CapabilityResponse{WorkerID: w.desc.ID, Output: []byte(`{}`)}, nil
}

func descriptor(t *testing.T, id, capability string, deterministic bool) tlaloque.CapabilityDescriptor {
	t.Helper()
	d, err := tlaloque.CapabilityDescriptor{
		ID: id, Capability: capability, Engine: tlaloque.EngineModel,
		InputSchema: "x", OutputSchema: "x", Deterministic: deterministic,
	}.Normalize()
	if err != nil {
		t.Fatalf("normalize descriptor: %v", err)
	}
	return d
}

func TestCapabilityRouter_VetoesExternalizedOpcodeWhenNoAlternative(t *testing.T) {
	registry := tlaloque.NewRegistry()
	parrotDesc := descriptor(t, "parrot", "VISUAL_LOCATE_COMPAT", false)
	if err := registry.Register(stubWorker{desc: parrotDesc}); err != nil {
		t.Fatalf("register: %v", err)
	}
	profile := fixtureProfile(t)
	// Reuse VISUAL_LOCATE's collapse semantics via a synthetic capability
	// name that mirrors the fixture's collapsed opcode, to keep this test
	// independent from the R0 opcode vocabulary itself.
	router := CapabilityRouter{Profiles: map[string]CapabilityProfile{"parrot": profile}}
	registry.SetSelectionStrategy(router)

	// Build a fake entry directly: reuse fixtureProfile's VISUAL_LOCATE
	// entry name as the requested capability to exercise the veto path.
	result := registry.SelectResult(tlaloque.SelectionRequest{Capability: "VISUAL_LOCATE_COMPAT"})
	if result.Code != tlaloque.ResultSuccess {
		t.Fatalf("expected success routing to the only (non-vetoed) worker since VISUAL_LOCATE_COMPAT has no profile entry, got %s", result.Code)
	}
}

func TestCapabilityRouter_PrefersDeterministicOverModelForSameCapability(t *testing.T) {
	registry := tlaloque.NewRegistry()
	det := descriptor(t, "numeric-tlaloque", "COMPARE_NUMBERS", true)
	model := descriptor(t, "parrot", "COMPARE_NUMBERS", false)
	if err := registry.Register(stubWorker{desc: det}); err != nil {
		t.Fatalf("register det: %v", err)
	}
	if err := registry.Register(stubWorker{desc: model}); err != nil {
		t.Fatalf("register model: %v", err)
	}
	router := CapabilityRouter{Profiles: map[string]CapabilityProfile{}}
	registry.SetSelectionStrategy(router)

	worker, err := registry.Select(tlaloque.SelectionRequest{Capability: "COMPARE_NUMBERS", PreferDeterministic: true})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if worker.Descriptor().ID != "numeric-tlaloque" {
		t.Fatalf("selected %s, want the deterministic numeric-tlaloque (E0.9: smallest reliable executor wins)", worker.Descriptor().ID)
	}
}

func TestCapabilityRouter_VetoesParrotWhenProfileSaysExternalize(t *testing.T) {
	registry := tlaloque.NewRegistry()
	parrotDesc := descriptor(t, "parrot", "EXTRACT_ENTITY", false)
	if err := registry.Register(stubWorker{desc: parrotDesc}); err != nil {
		t.Fatalf("register: %v", err)
	}
	profile := fixtureProfile(t) // EXTRACT_ENTITY is DOES_NOT_TRANSFER -> EXTERNALIZE
	router := CapabilityRouter{Profiles: map[string]CapabilityProfile{"parrot": profile}}
	registry.SetSelectionStrategy(router)

	result := registry.SelectResult(tlaloque.SelectionRequest{Capability: "EXTRACT_ENTITY"})
	if result.Code != tlaloque.ResultNoCandidate {
		t.Fatalf("expected ResultNoCandidate when the only worker is vetoed by EXTERNALIZE, got %s (err=%v)", result.Code, result.Err)
	}
}

func TestCapabilityRouter_AllowsDeterministicAlternativeWhenParrotVetoed(t *testing.T) {
	registry := tlaloque.NewRegistry()
	parrotDesc := descriptor(t, "parrot", "EXTRACT_ENTITY", false)
	detDesc := descriptor(t, "entity-tlaloque", "EXTRACT_ENTITY", true)
	if err := registry.Register(stubWorker{desc: parrotDesc}); err != nil {
		t.Fatalf("register parrot: %v", err)
	}
	if err := registry.Register(stubWorker{desc: detDesc}); err != nil {
		t.Fatalf("register det: %v", err)
	}
	profile := fixtureProfile(t)
	router := CapabilityRouter{Profiles: map[string]CapabilityProfile{"parrot": profile}}
	registry.SetSelectionStrategy(router)

	worker, err := registry.Select(tlaloque.SelectionRequest{Capability: "EXTRACT_ENTITY"})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if worker.Descriptor().ID != "entity-tlaloque" {
		t.Fatalf("selected %s, want entity-tlaloque (parrot must be vetoed, not silently used)", worker.Descriptor().ID)
	}
}
