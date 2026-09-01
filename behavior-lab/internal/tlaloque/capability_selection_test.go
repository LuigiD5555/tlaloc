package tlaloque

import (
	"strings"
	"testing"
)

func modelWorker(id, capability string, parameters int64) testWorker {
	return testWorker{desc: CapabilityDescriptor{
		ID:             id,
		Capability:     capability,
		Scope:          ScopeGeneral,
		Engine:         EngineModel,
		InputSchema:    "text",
		OutputSchema:   "json",
		ParameterCount: parameters,
	}}
}

func TestDescriptorNormalizeRejectsIncompleteContracts(t *testing.T) {
	cases := []struct {
		name string
		desc CapabilityDescriptor
		want string
	}{
		{
			name: "missing id",
			desc: CapabilityDescriptor{Capability: "A", InputSchema: "in", OutputSchema: "out"},
			want: "are required",
		},
		{
			name: "missing capability",
			desc: CapabilityDescriptor{ID: "a", InputSchema: "in", OutputSchema: "out"},
			want: "are required",
		},
		{
			name: "missing input schema",
			desc: CapabilityDescriptor{ID: "a", Capability: "A", OutputSchema: "out"},
			want: "are required",
		},
		{
			name: "missing output schema",
			desc: CapabilityDescriptor{ID: "a", Capability: "A", InputSchema: "in"},
			want: "are required",
		},
		{
			name: "unsupported scope",
			desc: CapabilityDescriptor{ID: "a", Capability: "A", Scope: "REGIONAL", InputSchema: "in", OutputSchema: "out"},
			want: "unsupported scope",
		},
		{
			name: "specific without domain",
			desc: CapabilityDescriptor{ID: "a", Capability: "A", Scope: ScopeSpecific, InputSchema: "in", OutputSchema: "out"},
			want: "requires domain",
		},
		{
			name: "unexpected schema",
			desc: CapabilityDescriptor{Schema: "tlaloc.other.r1", ID: "a", Capability: "A", InputSchema: "in", OutputSchema: "out"},
			want: "unexpected capability schema",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.desc.Normalize()
			if err == nil {
				t.Fatalf("expected rejection for %s", testCase.name)
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error %q does not mention %q", err, testCase.want)
			}
		})
	}
}

func TestDescriptorNormalizeAppliesDefaults(t *testing.T) {
	desc, err := (CapabilityDescriptor{
		ID:           " date-resolver ",
		Capability:   " resolve_date ",
		InputSchema:  "text",
		OutputSchema: "date",
	}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if desc.Schema != CapabilitySchemaR0 {
		t.Fatalf("schema=%s", desc.Schema)
	}
	if desc.ID != "date-resolver" || desc.Capability != "RESOLVE_DATE" {
		t.Fatalf("identity not normalised: %+v", desc)
	}
	if desc.Scope != ScopeGeneral {
		t.Fatalf("scope default=%s, want GENERAL", desc.Scope)
	}
	if desc.Engine != EngineModel {
		t.Fatalf("engine default=%s, want MODEL", desc.Engine)
	}
	if desc.MaxConcurrency != 1 {
		t.Fatalf("max_concurrency default=%d, want 1", desc.MaxConcurrency)
	}
}

func TestRegistryRefusesDuplicateAndInvalidWorkers(t *testing.T) {
	registry := NewRegistry()
	worker := modelWorker("intent", "DETECT_INTENT", 12_000_000)
	if err := registry.Register(worker); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(worker); err == nil {
		t.Fatal("expected duplicate worker id to be refused")
	}
	if err := registry.Register(nil); err == nil {
		t.Fatal("expected nil worker to be refused")
	}
	invalid := testWorker{desc: CapabilityDescriptor{ID: "broken"}}
	if err := registry.Register(invalid); err == nil {
		t.Fatal("expected a worker with an incomplete contract to be refused")
	}
}

// The parameter budget is what makes "many tiny models" a testable claim
// rather than a slogan: no selected individual may exceed it.
func TestSelectEnforcesMaxParameters(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry,
		modelWorker("big", "DETECT_INTENT", 400_000_000),
		modelWorker("small", "DETECT_INTENT", 12_000_000),
	)
	worker, err := registry.Select(SelectionRequest{Capability: "DETECT_INTENT", MaxParameters: 20_000_000})
	if err != nil {
		t.Fatal(err)
	}
	if worker.Descriptor().ID != "small" {
		t.Fatalf("selected %s over the parameter budget", worker.Descriptor().ID)
	}
	if _, err := registry.Select(SelectionRequest{Capability: "DETECT_INTENT", MaxParameters: 1_000}); err == nil {
		t.Fatal("expected refusal when no worker fits the parameter budget")
	}
}

func TestSelectPrefersDeterministicWhenAsked(t *testing.T) {
	registry := NewRegistry()
	deterministic := generalDescriptor("rules", "RESOLVE_DATE")
	model := modelWorker("date-model", "RESOLVE_DATE", 3_000_000)
	mustRegister(t, registry, testWorker{desc: deterministic}, model)

	worker, err := registry.Select(SelectionRequest{Capability: "RESOLVE_DATE", PreferDeterministic: true})
	if err != nil {
		t.Fatal(err)
	}
	if worker.Descriptor().ID != "rules" {
		t.Fatalf("selected %s, want the deterministic worker", worker.Descriptor().ID)
	}
}

// Equal-scoring candidates must resolve identically on every run, otherwise a
// measured swarm is not reproducible.
func TestSelectIsDeterministicAcrossRepeatedCalls(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry,
		modelWorker("charlie", "CLASSIFY", 8_000_000),
		modelWorker("alpha", "CLASSIFY", 8_000_000),
		modelWorker("bravo", "CLASSIFY", 8_000_000),
	)
	first, err := registry.Select(SelectionRequest{Capability: "CLASSIFY"})
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 25; attempt++ {
		next, err := registry.Select(SelectionRequest{Capability: "CLASSIFY"})
		if err != nil {
			t.Fatal(err)
		}
		if next.Descriptor().ID != first.Descriptor().ID {
			t.Fatalf("selection is not reproducible: %s then %s", first.Descriptor().ID, next.Descriptor().ID)
		}
	}
	if first.Descriptor().ID != "alpha" {
		t.Fatalf("tie broken by %s, want lowest id for reproducibility", first.Descriptor().ID)
	}
}

func TestSelectPrefersSmallerModelAmongEquals(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry,
		modelWorker("large", "CLASSIFY", 60_000_000),
		modelWorker("tiny", "CLASSIFY", 3_000_000),
	)
	worker, err := registry.Select(SelectionRequest{Capability: "CLASSIFY"})
	if err != nil {
		t.Fatal(err)
	}
	if worker.Descriptor().ID != "tiny" {
		t.Fatalf("selected %s, want the smaller model", worker.Descriptor().ID)
	}
}

// Pinning a worker is how a plan freezes an experiment; a pin that disagrees
// with the requested capability must fail loudly rather than silently run.
func TestSelectPinnedWorkerMustMatchCapability(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry, modelWorker("intent", "DETECT_INTENT", 12_000_000))

	worker, err := registry.Select(SelectionRequest{Capability: "DETECT_INTENT", WorkerID: "intent"})
	if err != nil {
		t.Fatal(err)
	}
	if worker.Descriptor().ID != "intent" {
		t.Fatalf("pin ignored, got %s", worker.Descriptor().ID)
	}
	if _, err := registry.Select(SelectionRequest{Capability: "ROUTE", WorkerID: "intent"}); err == nil {
		t.Fatal("expected a capability mismatch on the pinned worker")
	}
	if _, err := registry.Select(SelectionRequest{Capability: "DETECT_INTENT", WorkerID: "ghost"}); err == nil {
		t.Fatal("expected an error for an unregistered pinned worker")
	}
}

// A pinned specialist bypasses domain inference deliberately: the plan author
// has already supplied the evidence.
func TestSelectPinnedSpecialistBypassesDomainInference(t *testing.T) {
	registry := NewRegistry()
	specialist := CapabilityDescriptor{
		ID: "cfdi-classifier", Capability: "CLASSIFY_DOCUMENT", Scope: ScopeSpecific,
		Domain: "CFDI", Engine: EngineModel, InputSchema: "text", OutputSchema: "class", ParameterCount: 3_000_000,
	}
	mustRegister(t, registry, testWorker{desc: specialist})
	worker, err := registry.Select(SelectionRequest{Capability: "CLASSIFY_DOCUMENT", WorkerID: "cfdi-classifier"})
	if err != nil {
		t.Fatal(err)
	}
	if worker.Descriptor().ID != "cfdi-classifier" {
		t.Fatalf("got %s", worker.Descriptor().ID)
	}
}

func TestSelectRejectsMalformedScopeHints(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry, modelWorker("entity", "EXTRACT_ENTITY", 18_000_000))

	if _, err := registry.Select(SelectionRequest{Capability: "EXTRACT_ENTITY", ScopeHint: "REGIONAL"}); err == nil {
		t.Fatal("expected an unsupported scope hint to be refused")
	}
	if _, err := registry.Select(SelectionRequest{Capability: "EXTRACT_ENTITY", ScopeHint: ScopeSpecific}); err == nil {
		t.Fatal("expected SPECIFIC without a domain to be refused")
	}
}

// With domain evidence a small specialist may win; without it the generalist
// must be chosen even though the specialist is cheaper.
func TestSelectUsesDomainEvidenceToPreferSpecialist(t *testing.T) {
	registry := NewRegistry()
	general := modelWorker("generic-entity", "EXTRACT_ENTITY", 18_000_000)
	specialist := testWorker{desc: CapabilityDescriptor{
		ID: "cfdi-entity", Capability: "EXTRACT_ENTITY", Scope: ScopeSpecific, Domain: "CFDI",
		Engine: EngineModel, InputSchema: "text", OutputSchema: "entities", ParameterCount: 4_000_000,
	}}
	unrelated := testWorker{desc: CapabilityDescriptor{
		ID: "medical-entity", Capability: "EXTRACT_ENTITY", Scope: ScopeSpecific, Domain: "MEDICAL",
		Engine: EngineModel, InputSchema: "text", OutputSchema: "entities", ParameterCount: 1_000_000,
	}}
	mustRegister(t, registry, general, specialist, unrelated)

	worker, err := registry.Select(SelectionRequest{Capability: "EXTRACT_ENTITY", DomainHint: "CFDI"})
	if err != nil {
		t.Fatal(err)
	}
	if worker.Descriptor().ID != "cfdi-entity" {
		t.Fatalf("selected %s, want the CFDI specialist", worker.Descriptor().ID)
	}

	// The cheapest worker overall is the MEDICAL one; it must never be reached
	// from CFDI evidence.
	worker, err = registry.Select(SelectionRequest{Capability: "EXTRACT_ENTITY"})
	if err != nil {
		t.Fatal(err)
	}
	if worker.Descriptor().ID != "generic-entity" {
		t.Fatalf("selected %s without domain evidence, want the generalist", worker.Descriptor().ID)
	}
}

func TestSelectGeneralScopeHintExcludesSpecialists(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry,
		modelWorker("generic", "CLASSIFY_DOCUMENT", 30_000_000),
		testWorker{desc: CapabilityDescriptor{
			ID: "cfdi", Capability: "CLASSIFY_DOCUMENT", Scope: ScopeSpecific, Domain: "CFDI",
			Engine: EngineModel, InputSchema: "text", OutputSchema: "class", ParameterCount: 3_000_000,
		}},
	)
	worker, err := registry.Select(SelectionRequest{Capability: "CLASSIFY_DOCUMENT", ScopeHint: ScopeGeneral, DomainHint: "CFDI"})
	if err != nil {
		t.Fatal(err)
	}
	if worker.Descriptor().ID != "generic" {
		t.Fatalf("GENERAL hint selected %s", worker.Descriptor().ID)
	}
}

func TestRegistryDescriptorsAreSortedSnapshot(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry,
		modelWorker("zulu", "A", 1),
		modelWorker("alpha", "B", 2),
		modelWorker("mike", "C", 3),
	)
	descriptors := registry.Descriptors()
	if len(descriptors) != 3 {
		t.Fatalf("descriptors=%d", len(descriptors))
	}
	for index := 1; index < len(descriptors); index++ {
		if descriptors[index-1].ID > descriptors[index].ID {
			t.Fatalf("descriptors are not sorted: %v", descriptors)
		}
	}
	if _, ok := registry.Get("alpha"); !ok {
		t.Fatal("registered worker not retrievable by id")
	}
	if _, ok := registry.Get("absent"); ok {
		t.Fatal("unregistered id resolved")
	}
}

func TestSelectMatchesCapabilityCaseInsensitively(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry, modelWorker("intent", "DETECT_INTENT", 12_000_000))
	worker, err := registry.Select(SelectionRequest{Capability: " detect_intent "})
	if err != nil {
		t.Fatal(err)
	}
	if worker.Descriptor().ID != "intent" {
		t.Fatalf("got %s", worker.Descriptor().ID)
	}
}
