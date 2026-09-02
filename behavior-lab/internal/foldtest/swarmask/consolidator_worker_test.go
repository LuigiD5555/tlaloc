package swarmask

import (
	"context"
	"encoding/json"
	"testing"

	"tlaloc.local/behaviorlab/internal/blackboard"
)

// evaluateGrounding is the pure scoring step (no store, no network needed:
// answerscore.ScoreAnswer degrades to its deterministic KeywordOverlapWorker
// when there's no LM Studio to call, exactly the fallback already verified
// in the answerscore package).

func TestEvaluateGrounding_PositiveCase(t *testing.T) {
	out, err := evaluateGrounding(context.Background(), "fake-model", "http://127.0.0.1:1/unreachable",
		"What is a swarm?",
		"A swarm is a system of distributed coordinating agents.",
		"A swarm is a system composed of multiple distributed agents that coordinate their behavior.")
	if err != nil {
		t.Fatalf("evaluateGrounding: %v", err)
	}
	if out.Score < 0.5 {
		t.Errorf("expected a supported answer to score >= 0.5, got %f", out.Score)
	}
	if !out.Grounded {
		t.Error("expected Grounded=true for a well-supported answer")
	}
	if out.ScoredBy == "" {
		t.Error("expected ScoredBy to name which worker judged the answer")
	}
}

func TestEvaluateGrounding_ContradictionBoundary(t *testing.T) {
	out, err := evaluateGrounding(context.Background(), "fake-model", "http://127.0.0.1:1/unreachable",
		"What is a swarm?",
		"The capital of France is Paris.",
		"A swarm is a system composed of multiple distributed agents that coordinate their behavior.")
	if err != nil {
		t.Fatalf("evaluateGrounding: %v", err)
	}
	if out.Grounded {
		t.Errorf("expected an unrelated answer to not be grounded, got score %f", out.Score)
	}
}

// Non-applicable case: no suggested_page observation on the blackboard means
// there's nothing to verify against — the consolidator must not fabricate a
// verdict, and must not touch ExtractPageContent/answerscore at all.
func TestConsolidatorWorker_NonApplicableCase(t *testing.T) {
	req := mustAskRequest(t, AskInput{Question: "What is a swarm?", StoreDir: "/nonexistent"})
	req.Blackboard = &blackboard.Snapshot{Entries: []blackboard.Entry{}}
	req.Context = map[string]json.RawMessage{}

	resp, err := ConsolidatorWorker{}.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(resp.Observations) != 0 {
		t.Errorf("expected no observations when there is no page to verify against, got %d", len(resp.Observations))
	}
}

// A suggested page that can't actually be read (bad store) must degrade
// gracefully — no observation, no error — rather than failing the run.
func TestConsolidatorWorker_UnreadablePageDegradesGracefully(t *testing.T) {
	req := mustAskRequest(t, AskInput{Question: "What is a swarm?", StoreDir: "/nonexistent"})
	suggestion, _ := json.Marshal(suggestedPageObservation{Page: 1, Score: 0.5})
	req.Blackboard = &blackboard.Snapshot{Entries: []blackboard.Entry{
		{Key: suggestedPageKey, NodeID: "scout", Value: suggestion},
	}}
	answerOut, _ := json.Marshal(AnswerOutput{Answer: "some answer"})
	req.Context = map[string]json.RawMessage{"answer": answerOut}

	resp, err := ConsolidatorWorker{}.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(resp.Observations) != 0 {
		t.Errorf("expected no grounding observation when the page can't be read, got %d", len(resp.Observations))
	}
	var out ConsolidationOutput
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if out.Answer != "some answer" {
		t.Errorf("expected the answer text to still pass through, got %q", out.Answer)
	}
}
