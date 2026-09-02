package intent

import (
	"context"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

type stubWorker struct{ desc tlaloque.CapabilityDescriptor }

func (w stubWorker) Descriptor() tlaloque.CapabilityDescriptor { return w.desc }
func (w stubWorker) Execute(context.Context, tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	return tlaloque.CapabilityResponse{WorkerID: w.desc.ID, Output: []byte(`{}`)}, nil
}

func registryWithChain(t *testing.T) *tlaloque.Registry {
	t.Helper()
	registry := tlaloque.NewRegistry()
	workers := []tlaloque.CapabilityWorker{
		stubWorker{desc: tlaloque.CapabilityDescriptor{ID: "reader", Capability: "READ_DOCUMENT", Scope: tlaloque.ScopeGeneral, Engine: tlaloque.EngineDeterministic, InputSchema: "path", OutputSchema: "text", Deterministic: true}},
		stubWorker{desc: tlaloque.CapabilityDescriptor{ID: "analyzer", Capability: "ANALYZE_CONTENT", Scope: tlaloque.ScopeGeneral, Engine: tlaloque.EngineModel, InputSchema: "text", OutputSchema: "finding", ParameterCount: 30_000, Dependencies: []string{"READ_DOCUMENT"}}},
		stubWorker{desc: tlaloque.CapabilityDescriptor{ID: "verifier", Capability: "VERIFY_CLAIM", Scope: tlaloque.ScopeGeneral, Engine: tlaloque.EngineDeterministic, InputSchema: "finding", OutputSchema: "verdict", Deterministic: true, Dependencies: []string{"ANALYZE_CONTENT"}}},
		stubWorker{desc: tlaloque.CapabilityDescriptor{ID: "giant", Capability: "ANALYZE_CONTENT", Scope: tlaloque.ScopeGeneral, Engine: tlaloque.EngineModel, InputSchema: "text", OutputSchema: "finding", ParameterCount: 7_000_000_000, Dependencies: []string{"READ_DOCUMENT"}}},
	}
	for _, worker := range workers {
		if err := registry.Register(worker); err != nil {
			t.Fatalf("register %s: %v", worker.Descriptor().ID, err)
		}
	}
	return registry
}

func TestCompile_RejectsIncompleteIntent(t *testing.T) {
	if _, err := Compile(IntentIR{RequiredOutputs: []string{"VERIFY_CLAIM"}}); err == nil {
		t.Error("expected an error for a missing version")
	}
	if _, err := Compile(IntentIR{Version: "1"}); err == nil {
		t.Error("expected an error for no required outputs")
	}
	if _, err := Compile(IntentIR{Version: "1", RequiredOutputs: []string{"X"}, Risk: RiskProfile{Level: "extreme"}}); err == nil {
		t.Error("expected an error for an invalid risk level")
	}
	if _, err := Compile(IntentIR{Version: "1", RequiredOutputs: []string{"X"}, EvidenceRequirements: []EvidenceRequirement{{ForOutput: "X", MinLevel: "Z"}}}); err == nil {
		t.Error("expected an error for an invalid evidence level")
	}
}

func TestCompile_HighRiskForcesDeterministicPreference(t *testing.T) {
	compiled, err := Compile(IntentIR{
		Version:         "1",
		RequiredOutputs: []string{"verify_claim"},
		Risk:            RiskProfile{Level: "high"},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !compiled.Goals[0].PreferDeterministic {
		t.Error("a high-risk intent must set PreferDeterministic on its goals")
	}
	if compiled.Goals[0].Capability != "VERIFY_CLAIM" {
		t.Errorf("capability not normalized: %q", compiled.Goals[0].Capability)
	}
}

// The core path the user asked for: intent -> requirements -> DAG.
func TestPlanFor_IntentToDAG(t *testing.T) {
	registry := registryWithChain(t)

	ir := IntentIR{
		Schema:          Schema,
		Version:         "1",
		Goal:            "Read the document, analyze what it says, and verify it.",
		Inputs:          []TypedInput{{Name: "doc", Kind: "path"}},
		RequiredOutputs: []string{"VERIFY_CLAIM"},
		Invariants:      []Invariant{{ID: "no-fabrication", Statement: "never state a fact not grounded in the document"}},
		Constraints:     []Constraint{{Kind: "max_parameters", Value: "1000000"}, {Kind: "prefer_deterministic", Value: "true"}},
		Budget:          Budget{MaxTokens: 4000},
		Risk:            RiskProfile{Level: "medium"},
	}

	compiled, err := Compile(ir)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if compiled.Goals[0].MaxParameters != 1_000_000 {
		t.Errorf("max_parameters constraint not threaded: %d", compiled.Goals[0].MaxParameters)
	}

	planned, err := PlanFor(registry, compiled, "doc-verify", 2)
	if err != nil {
		t.Fatalf("PlanFor: %v", err)
	}

	// VERIFY_CLAIM -> ANALYZE_CONTENT -> READ_DOCUMENT, and the small
	// analyzer must win over the 7B "giant" because max_parameters=1e6.
	if len(planned.Plan.Nodes) != 3 {
		t.Fatalf("expected a 3-node DAG, got %d: %+v", len(planned.Plan.Nodes), planned.Plan.Nodes)
	}
	byWorker := map[string]tlaloque.SwarmNode{}
	for _, node := range planned.Plan.Nodes {
		byWorker[node.WorkerID] = node
	}
	if _, ok := byWorker["giant"]; ok {
		t.Error("the 7B worker must not be selected under a max_parameters=1e6 constraint")
	}
	if _, ok := byWorker["analyzer"]; !ok {
		t.Fatal("the small analyzer should have been selected")
	}
	verifier, ok := byWorker["verifier"]
	if !ok || len(verifier.DependsOn) != 1 {
		t.Fatalf("verifier node missing or wrong deps: %+v", verifier)
	}
}

func TestPlanFor_AbstentionRequiredWarnsOnProbabilisticWorker(t *testing.T) {
	registry := registryWithChain(t)
	compiled, err := Compile(IntentIR{
		Version:         "1",
		RequiredOutputs: []string{"ANALYZE_CONTENT"},
		Constraints:     []Constraint{{Kind: "max_parameters", Value: "1000000"}},
		Risk:            RiskProfile{Level: "medium", AbstentionRequired: true},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	planned, err := PlanFor(registry, compiled, "analyze", 2)
	if err != nil {
		t.Fatalf("PlanFor: %v", err)
	}
	found := false
	for _, warning := range planned.Warnings {
		if warning == `risk.abstention_required: worker "analyzer" is a probabilistic model; its CalibrationProfile must be checked before execution` {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an abstention advisory for the probabilistic analyzer, got: %v", planned.Warnings)
	}
}
