package swarmask

import (
	"context"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque/groundingautomaton"
)

func TestEvaluateGroundingUsesAutomatonForPolarityContradiction(t *testing.T) {
	out, err := ConsolidatorWorker{}.evaluateGrounding(
		context.Background(),
		"fake-model",
		"http://127.0.0.1:1/unreachable",
		"Does the system distribute the load?",
		"The system does not distribute the load between three agents.",
		"The system distributes the load between three agents.",
	)
	if err != nil {
		t.Fatalf("evaluateGrounding: %v", err)
	}
	if out.Grounded || out.Score != 0 {
		t.Fatalf("expected deterministic rejection, got %+v", out)
	}
	if out.ScoredBy != groundingautomaton.WorkerID {
		t.Fatalf("expected %s, got %s", groundingautomaton.WorkerID, out.ScoredBy)
	}
}

func TestEvaluateGroundingUsesAutomatonForNumericContradiction(t *testing.T) {
	out, err := ConsolidatorWorker{}.evaluateGrounding(
		context.Background(),
		"fake-model",
		"http://127.0.0.1:1/unreachable",
		"How many parameters does the model have?",
		"The model has 28 million parameters.",
		"The model has 27 million parameters.",
	)
	if err != nil {
		t.Fatalf("evaluateGrounding: %v", err)
	}
	if out.Grounded || out.ScoredBy != groundingautomaton.WorkerID {
		t.Fatalf("expected automaton contradiction, got %+v", out)
	}
}

func TestEvaluateGroundingDefersWhenAutomatonAbstains(t *testing.T) {
	out, err := ConsolidatorWorker{}.evaluateGrounding(
		context.Background(),
		"fake-model",
		"http://127.0.0.1:1/unreachable",
		"What changes?",
		"The organization continually reshapes itself according to environmental conditions.",
		"The swarm adapts its topology dynamically.",
	)
	if err != nil {
		t.Fatalf("evaluateGrounding: %v", err)
	}
	if out.ScoredBy == groundingautomaton.WorkerID {
		t.Fatalf("automaton must abstain on low lexical alignment, got %+v", out)
	}
}
