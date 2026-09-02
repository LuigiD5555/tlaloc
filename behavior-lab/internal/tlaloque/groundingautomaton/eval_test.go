package groundingautomaton

import "testing"

func TestEvaluateTracksFalseSupportedContradictions(t *testing.T) {
	cases := []EvalCase{{
		ID:       "numeric",
		Answer:   "The model has 28 million parameters.",
		Evidence: "The model has 27 million parameters.",
		Expected: VerdictContradicted,
	}}
	_, metrics := Evaluate(cases)
	if metrics.FalseSupportedContradiction != 0 {
		t.Fatalf("expected zero false-supported contradictions, got %+v", metrics)
	}
	if metrics.ContradictionTruePositive != 1 {
		t.Fatalf("expected contradiction TP=1, got %+v", metrics)
	}
}
