package tlaloquekit

import (
	"context"
	"encoding/json"
	"testing"
)

func mustRegistry(t *testing.T) QualifiedRegistry {
	t.Helper()
	registry, err := BuildQualifiedRegistry(Config{})
	if err != nil {
		t.Fatalf("BuildQualifiedRegistry: %v", err)
	}
	return registry
}

func TestCapabilities_PublishesTheDeterministicSetWithoutInternalLeak(t *testing.T) {
	got := mustRegistry(t).Capabilities()
	want := map[string]bool{
		"LOCATE_REGION": true, "CROP_REGION": true, "NORMALIZE": true,
		"COMPARE_NUMBERS": true, "ARITHMETIC": true, "VERIFY": true,
	}
	seen := map[string]bool{}
	for _, d := range got {
		seen[d.Capability] = true
		if d.Engine != EngineDeterministic || !d.Deterministic {
			t.Fatalf("%s: expected a deterministic engine, got %q/%v", d.ID, d.Engine, d.Deterministic)
		}
	}
	for capability := range want {
		if !seen[capability] {
			t.Fatalf("capability %s was not published; got %+v", capability, got)
		}
	}
}

func TestCandidates_MarksTheSelectedExecutor(t *testing.T) {
	candidates := mustRegistry(t).Candidates("ARITHMETIC", Goal{})
	if len(candidates) != 1 || !candidates[0].Selected {
		t.Fatalf("expected one selected ARITHMETIC candidate, got %+v", candidates)
	}
	if candidates[0].Descriptor.ID == "" || candidates[0].Reason == "" {
		t.Fatalf("candidate is missing id/reason: %+v", candidates[0])
	}
}

func TestResolve_PinsADeterministicDAG(t *testing.T) {
	resolution, err := mustRegistry(t).Resolve(Goal{Capability: "ARITHMETIC"}, "t1-plan", 1)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resolution.Nodes) != 1 || resolution.Nodes[0].Capability != "ARITHMETIC" {
		t.Fatalf("unexpected DAG: %+v", resolution.Nodes)
	}
	if resolution.Nodes[0].WorkerID == "" {
		t.Fatalf("resolved node must pin a worker id: %+v", resolution.Nodes[0])
	}
	if _, ok := resolution.Candidates["ARITHMETIC"]; !ok {
		t.Fatalf("resolution must carry candidate analysis for the routing trace")
	}
}

func TestResolve_UnknownCapabilityIsAnError(t *testing.T) {
	if _, err := mustRegistry(t).Resolve(Goal{Capability: "SUMMARIZE_DOCUMENT"}, "p", 1); err == nil {
		t.Fatalf("expected an error resolving a capability with no qualified executor")
	}
}

func TestExecute_RunsArithmeticEndToEnd(t *testing.T) {
	registry := mustRegistry(t)
	resolution, err := registry.Resolve(Goal{Capability: "ARITHMETIC"}, "p", 1)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	input, _ := json.Marshal(map[string]string{"operation": "PERCENT_DIFFERENCE", "a": "120", "b": "100"})
	result, err := registry.Execute(context.Background(), ExecutionRequest{
		TaskID: "p", NodeID: resolution.Nodes[0].ID, Capability: "ARITHMETIC",
		WorkerID: resolution.Nodes[0].WorkerID, Input: input,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out struct {
		Result    float64 `json:"result"`
		HasResult bool    `json:"has_result"`
	}
	if err := json.Unmarshal(result.Output, &out); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if !out.HasResult || out.Result != 20 {
		t.Fatalf("percent difference 120 vs 100 = %v (has_result=%v), want 20", out.Result, out.HasResult)
	}
	if len(result.Observations) != 1 || result.Observations[0].Producer == "" {
		t.Fatalf("expected one observation with a producer, got %+v", result.Observations)
	}
}

func TestExecute_VerifyReadsForwardedPriorObservations(t *testing.T) {
	registry := mustRegistry(t)
	verifyID := ""
	for _, d := range registry.Capabilities() {
		if d.Capability == "VERIFY" {
			verifyID = d.ID
		}
	}
	if verifyID == "" {
		t.Fatal("no VERIFY tlaloque registered")
	}

	prior := []Observation{{
		Producer: "region-or-parrot", Capability: "EXTRACT_NUMBER", Key: "extract_a",
		Value: json.RawMessage(`"42"`), Kind: "OBSERVATION",
	}}
	input, _ := json.Marshal(map[string]any{"target_key": "extract_a", "fact_id": "a_value", "expected_type": "number"})
	result, err := registry.Execute(context.Background(), ExecutionRequest{
		TaskID: "p", NodeID: "verify_a", Capability: "VERIFY", WorkerID: verifyID,
		Input: input, PriorObservations: prior,
	})
	if err != nil {
		t.Fatalf("Execute VERIFY: %v", err)
	}
	if len(result.Observations) != 1 {
		t.Fatalf("expected one fact observation, got %+v", result.Observations)
	}
	got := result.Observations[0]
	if got.Kind != "FACT" || got.Status != "VERIFIED" {
		t.Fatalf("expected a VERIFIED fact, got kind=%q status=%q", got.Kind, got.Status)
	}
}
