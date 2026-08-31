package closedloop

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"tlaloc.local/behaviorlab/internal/visualsearch"
)

func TestClosedLoopAdvancesIncumbentAndMovesFailureFrontier(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.png")
	candidate1Path := filepath.Join(dir, "candidate-t2.png")
	candidate2Path := filepath.Join(dir, "candidate-temporal.png")
	baseline := writeTestPNG(t, baselinePath, 0)
	candidate1 := writeTestPNG(t, candidate1Path, 127)
	candidate2 := writeTestPNG(t, candidate2Path, 255)
	_ = baseline
	c1b64 := base64.StdEncoding.EncodeToString(candidate1)
	c2b64 := base64.StdEncoding.EncodeToString(candidate2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode: %v", err)
			w.WriteHeader(400)
			return
		}
		body := string(req.Messages[len(req.Messages)-1].Content)
		system := string(req.Messages[0].Content)
		question := extractQuestion(body)
		diagnostic := strings.Contains(system, "DIAGNOSTIC MODE")
		isC1 := strings.Contains(body, c1b64)
		isC2 := strings.Contains(body, c2b64)
		answer := answerFor(question)

		if !isC1 && !isC2 && strings.Contains(question, "causes B") {
			answer = "I cannot locate the transition."
			if diagnostic {
				answer += `
ORIGAMI_DEBUG_R0={"schema":"tlaloc.origami-debug-trace.r0","status":"FAIL","last_completed_stage":"ROSETTA","selected_codec":"ST2","last_instruction":"READ_ROSETTA","next_instruction":"LOCATE_T2","failure_code":"T2_NOT_FOUND","evidence_refs":["T0","T1"],"confidence":0.9}`
			}
		}
		if !isC2 && strings.Contains(question, "after B") {
			answer = "I cannot determine the declared temporal consequence."
			if diagnostic {
				answer += `
ORIGAMI_DEBUG_R0={"schema":"tlaloc.origami-debug-trace.r0","status":"FAIL","last_completed_stage":"TEMPORAL_ROUTE","selected_codec":"ST4","last_instruction":"READ_TRANSITION","next_instruction":"SIMULATE_DECLARED_STEP","failure_code":"TEMPORAL_RULE_AMBIGUOUS","evidence_refs":["T2","RULES"],"confidence":0.88}`
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": answer}}},
		})
	}))
	defer server.Close()

	cfg := Config{
		Schema:                   ConfigSchema,
		RunID:                    "incumbent-chain",
		OutputDir:                filepath.Join(dir, "run"),
		MemoryRoot:               filepath.Join(dir, "memory"),
		TrialsPerModel:           1,
		CandidatesPerGeneration: 1,
		MaxGenerations:           3,
		MinIncumbentImprovement:  0.01,
		DiagnosticRetries:        true,
		Conditions:               []string{"NATIVE_PNG_ONLY"},
		OutcomeMetric:            OutcomeNative,
		Models: []ModelConfig{{
			Name:           "fake",
			Provider:       "OPENAI_COMPAT",
			BaseURL:        server.URL,
			Model:          "fake-vlm",
			TimeoutSeconds: 10,
		}},
		Baseline: SpecimenConfig{ID: "baseline", PNG: baselinePath},
		Candidates: []CandidateConfig{
			{
				ID:               "candidate-t2",
				PNG:              candidate1Path,
				BaseProfileID:    "profile-3",
				ParentSpecimenID: "baseline",
				Mutations: []visualsearch.Mutation{{
					Kind:         visualsearch.MutationLayout,
					Target:       "T1_TO_T2_ENTRY_ROUTE",
					Value:        "EXPLICIT_DIRECTIONAL_ANCHOR",
					Experimental: true,
				}},
			},
			{
				ID:               "candidate-temporal",
				PNG:              candidate2Path,
				BaseProfileID:    "profile-3",
				ParentSpecimenID: "candidate-t2",
				Mutations: []visualsearch.Mutation{{
					Kind:         visualsearch.MutationTemporalStructure,
					Target:       "TEMPORAL_GRAMMAR",
					Value:        "EXPLICIT_PHASE_EVENT_CHECKPOINT_STRUCTURE",
					Experimental: true,
				}},
			},
		},
	}

	report, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.ExecutionErrors) != 0 {
		t.Fatalf("execution errors: %+v", report.ExecutionErrors)
	}
	if len(report.Generations) != 2 {
		t.Fatalf("generations=%d want 2: %+v", len(report.Generations), report.Generations)
	}
	if !report.Generations[0].IncumbentAdvanced || report.Generations[0].IncumbentAfterID != "candidate-t2" {
		t.Fatalf("generation 1 did not advance T2 candidate: %+v", report.Generations[0])
	}
	if report.Generations[0].ActiveFailureCount < 2 {
		t.Fatalf("generation 1 should start with at least two active failures: %+v", report.Generations[0])
	}
	if !report.Generations[1].IncumbentAdvanced || report.Generations[1].IncumbentBeforeID != "candidate-t2" || report.Generations[1].IncumbentAfterID != "candidate-temporal" {
		t.Fatalf("generation 2 did not chain from experimental incumbent: %+v", report.Generations[1])
	}
	if report.Generations[1].ActiveFailureCount != 1 {
		t.Fatalf("generation 2 should expose only the next temporal failure, got %d", report.Generations[1].ActiveFailureCount)
	}
	if report.FinalIncumbentID != "candidate-temporal" {
		t.Fatalf("final incumbent=%q", report.FinalIncumbentID)
	}
	if report.StopReason != "INCUMBENT_NO_ACTIVE_FAILURES" {
		t.Fatalf("stop reason=%q", report.StopReason)
	}
}

func TestClosedLoopRejectsCandidateParentCycle(t *testing.T) {
	cfg := Config{
		Schema:    ConfigSchema,
		RunID:     "cycle",
		OutputDir: "out",
		Models:    []ModelConfig{{Name: "fake", Provider: "OPENAI_COMPAT", BaseURL: "http://127.0.0.1:1", Model: "fake"}},
		Baseline:  SpecimenConfig{ID: "baseline", PNG: "baseline.png"},
		Candidates: []CandidateConfig{
			{ID: "a", PNG: "a.png", BaseProfileID: "p", ParentSpecimenID: "b", Mutations: []visualsearch.Mutation{{Kind: visualsearch.MutationLayout, Target: "x", Value: "y", Experimental: true}}},
			{ID: "b", PNG: "b.png", BaseProfileID: "p", ParentSpecimenID: "a", Mutations: []visualsearch.Mutation{{Kind: visualsearch.MutationPrompt, Target: "x", Value: "y", Experimental: true}}},
		},
	}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected parent cycle validation error, got %v", err)
	}
}
