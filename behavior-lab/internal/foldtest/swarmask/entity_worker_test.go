package swarmask

import (
	"context"
	"encoding/json"
	"testing"
)

// Positive case: a question with a year and an acronym produces exactly
// that observation.
func TestEntityScoutWorker_PositiveCase(t *testing.T) {
	resp, err := EntityScoutWorker{}.Execute(context.Background(), mustAskRequest(t, AskInput{
		Question: "What happened with NATO in 2019?",
	}))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var out EntityOutput
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(out.Years) != 1 || out.Years[0] != 2019 {
		t.Errorf("expected years=[2019], got %v", out.Years)
	}
	if len(out.Acronyms) != 1 || out.Acronyms[0] != "NATO" {
		t.Errorf("expected acronyms=[NATO], got %v", out.Acronyms)
	}
	if len(resp.Observations) != 1 {
		t.Fatalf("expected exactly 1 observation, got %d", len(resp.Observations))
	}
	if resp.Observations[0].Key != questionEntitiesKey {
		t.Errorf("expected observation key %q, got %q", questionEntitiesKey, resp.Observations[0].Key)
	}
	if resp.Observations[0].Confidence != 1.0 {
		t.Errorf("expected confidence 1.0 for literal extraction, got %f", resp.Observations[0].Confidence)
	}
}

// Non-applicable case: a question with no years, numbers, or acronyms must
// not fabricate an observation.
func TestEntityScoutWorker_NonApplicableCase(t *testing.T) {
	resp, err := EntityScoutWorker{}.Execute(context.Background(), mustAskRequest(t, AskInput{
		Question: "How do swarms coordinate their agents?",
	}))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var out EntityOutput
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(out.Years) != 0 || len(out.Numbers) != 0 || len(out.Acronyms) != 0 {
		t.Errorf("expected no entities found, got %+v", out)
	}
	if len(resp.Observations) != 0 {
		t.Errorf("expected no observations when nothing was found, got %d", len(resp.Observations))
	}
}

// Boundary: a 4-digit number outside the plausible year range must be
// classified as a number, not a year.
func TestEntityScoutWorker_OutOfRangeYearBoundary(t *testing.T) {
	out := extractQuestionEntities("What is special about the number 3500 or 0999?")
	if len(out.Years) != 0 {
		t.Errorf("expected no years extracted from out-of-range 4-digit numbers, got %v", out.Years)
	}
	foundNumbers := map[float64]bool{}
	for _, n := range out.Numbers {
		foundNumbers[n] = true
	}
	if !foundNumbers[3500] || !foundNumbers[999] {
		t.Errorf("expected 3500 and 999 to be classified as plain numbers, got %v", out.Numbers)
	}
}
