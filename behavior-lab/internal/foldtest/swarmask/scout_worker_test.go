package swarmask

import (
	"context"
	"encoding/json"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func mustAskRequest(t *testing.T, in AskInput) tlaloque.CapabilityRequest {
	t.Helper()
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal AskInput: %v", err)
	}
	return tlaloque.CapabilityRequest{Input: raw}
}

const testCover = `--- Page Index ---
  page 1: gato, perro, jardin
  page 2: enjambre, agentes, coordinacion
  page 3: enjambre, robotica, coordinacion
`

// Positive case: a question whose words overlap one page's terms gets that
// page suggested, with a real observation attached.
func TestPageScoutWorker_PositiveCase(t *testing.T) {
	resp, err := PageScoutWorker{}.Execute(context.Background(), mustAskRequest(t, AskInput{
		Question: "¿Cómo coordinan los agentes del enjambre?",
		Cover:    testCover,
	}))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var out ScoutOutput
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if out.SuggestedPage != 2 {
		t.Errorf("expected page 2 (first page matching 'enjambre'/'agentes'/'coordinacion'), got %d", out.SuggestedPage)
	}
	if len(resp.Observations) != 1 {
		t.Fatalf("expected exactly 1 observation, got %d", len(resp.Observations))
	}
	if resp.Observations[0].Key != suggestedPageKey {
		t.Errorf("expected observation key %q, got %q", suggestedPageKey, resp.Observations[0].Key)
	}
}

// Non-applicable case: a question with zero overlap must not fabricate a
// suggestion.
func TestPageScoutWorker_NonApplicableCase(t *testing.T) {
	resp, err := PageScoutWorker{}.Execute(context.Background(), mustAskRequest(t, AskInput{
		Question: "¿Cuál es la capital de Francia?",
		Cover:    testCover,
	}))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var out ScoutOutput
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if out.SuggestedPage != 0 {
		t.Errorf("expected no suggestion (page 0), got %d", out.SuggestedPage)
	}
	if len(resp.Observations) != 0 {
		t.Errorf("expected no observations when there is no overlap, got %d", len(resp.Observations))
	}
}

// Determinism boundary: pages 2 and 3 both share "enjambre" and
// "coordinacion" with the question — the lowest-numbered page must win,
// consistently across repeated calls with the same input.
func TestPageScoutWorker_TieBreaksDeterministically(t *testing.T) {
	req := mustAskRequest(t, AskInput{
		Question: "¿Qué relación hay entre enjambre y coordinacion?",
		Cover:    testCover,
	})

	for i := 0; i < 5; i++ {
		resp, err := PageScoutWorker{}.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		var out ScoutOutput
		if err := json.Unmarshal(resp.Output, &out); err != nil {
			t.Fatalf("unmarshal output: %v", err)
		}
		if out.SuggestedPage != 2 {
			t.Fatalf("run %d: expected deterministic tie-break to page 2, got %d", i, out.SuggestedPage)
		}
	}
}
