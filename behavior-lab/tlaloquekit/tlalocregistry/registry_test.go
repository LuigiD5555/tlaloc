package tlalocregistry

import (
	"context"
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	kit "tlaloc.local/behaviorlab/tlaloquekit"
)

func mustRegistry(t *testing.T) kit.QualifiedRegistry {
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
		if d.Engine != kit.EngineDeterministic || !d.Deterministic {
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
	candidates := mustRegistry(t).Candidates("ARITHMETIC", kit.Goal{})
	if len(candidates) != 1 || !candidates[0].Selected {
		t.Fatalf("expected one selected ARITHMETIC candidate, got %+v", candidates)
	}
	if candidates[0].Descriptor.ID == "" || candidates[0].Reason == "" {
		t.Fatalf("candidate is missing id/reason: %+v", candidates[0])
	}
}

func TestResolve_PinsADeterministicDAG(t *testing.T) {
	resolution, err := mustRegistry(t).Resolve(kit.Goal{Capability: "ARITHMETIC"}, "t1-plan", 1)
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
	if _, err := mustRegistry(t).Resolve(kit.Goal{Capability: "SUMMARIZE_DOCUMENT"}, "p", 1); err == nil {
		t.Fatalf("expected an error resolving a capability with no qualified executor")
	}
}

func TestExecute_RunsArithmeticEndToEnd(t *testing.T) {
	registry := mustRegistry(t)
	resolution, err := registry.Resolve(kit.Goal{Capability: "ARITHMETIC"}, "p", 1)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	input, _ := json.Marshal(map[string]string{"operation": "PERCENT_DIFFERENCE", "a": "120", "b": "100"})
	result, err := registry.Execute(context.Background(), kit.ExecutionRequest{
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

	prior := []kit.Observation{{
		Producer: "region-or-parrot", Capability: "EXTRACT_NUMBER", Key: "extract_a",
		Value: json.RawMessage(`"42"`), Kind: "OBSERVATION",
	}}
	input, _ := json.Marshal(map[string]any{"target_key": "extract_a", "fact_id": "a_value", "expected_type": "number"})
	result, err := registry.Execute(context.Background(), kit.ExecutionRequest{
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

func TestParrotRegistration_IsR1AwareAndDoesNotExpandCapabilities(t *testing.T) {
	registry, err := BuildQualifiedRegistry(Config{Parrot: &ParrotConfig{
		ProfilePath:         "../../profiles/parrot-lfm2-vl-1.6b-r1.json",
		ExpectedProfileHash: "8acc959b",
		Endpoint:            kit.ParrotEndpoint{Model: "lfm2-vl-1.6b"},
	}})
	if err != nil {
		t.Fatalf("BuildQualifiedRegistry with Parrot: %v", err)
	}
	if registry.ParrotProfileID() != "parrot-lfm2-vl-1.6b@r1.0.0" ||
		registry.ParrotProfileHash() != "8acc959ba72334e64704c9f5b114bfb5230ca1f58375421c17a956e26b9ba729" {
		t.Fatalf("parrot profile identity not exposed: id=%q hash=%q", registry.ParrotProfileID(), registry.ParrotProfileHash())
	}

	extract := registry.Candidates("EXTRACT_NUMBER", kit.Goal{})
	if len(extract) != 1 || !extract[0].Selected || extract[0].Descriptor.Engine != kit.EngineGenerative {
		t.Fatalf("EXTRACT_NUMBER should resolve to the one generative Parrot candidate, got %+v", extract)
	}
	if extract[0].Descriptor.ProfileRef != "parrot-lfm2-vl-1.6b@r1.0.0" {
		t.Fatalf("EXTRACT_NUMBER candidate missing profile ref: %+v", extract[0].Descriptor)
	}

	// R1 covers only EXTRACT_NUMBER for T1; deterministic capabilities must
	// NOT gain a generative candidate.
	for _, capability := range []string{"NORMALIZE", "COMPARE_NUMBERS", "ARITHMETIC", "VERIFY"} {
		for _, candidate := range registry.Candidates(capability, kit.Goal{}) {
			if candidate.Descriptor.Engine == kit.EngineGenerative {
				t.Fatalf("%s unexpectedly gained a generative candidate: %+v", capability, candidate)
			}
		}
	}

	// no implicit capability -> Parrot fallback
	if _, err := registry.Resolve(kit.Goal{Capability: "READ_ASSOCIATED_NUMBER"}, "p", 1); err == nil {
		t.Fatal("READ_ASSOCIATED_NUMBER is not required by T1 and must not resolve")
	}
}

func TestNoInternalImportsInPublicContractPackage(t *testing.T) {
	fileSet := token.NewFileSet()
	packages, err := parser.ParseDir(fileSet, "..", func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse tlaloquekit: %v", err)
	}
	contract, ok := packages["tlaloquekit"]
	if !ok {
		t.Fatal("tlaloquekit package not found")
	}
	for name, file := range contract.Files {
		for _, imp := range file.Imports {
			value := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(value, "tlaloc.local/behaviorlab/internal/") {
				t.Errorf("%s: public contract package imports internal %q", name, value)
			}
		}
	}
}
