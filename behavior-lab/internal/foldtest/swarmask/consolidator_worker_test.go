package swarmask

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/tlaloque"
	"tlaloc.local/behaviorlab/internal/tlaloque/groundingscore"
)

// evaluateGrounding is the pure scoring step (no store, no network needed:
// answerscore.ScoreAnswer degrades to its deterministic KeywordOverlapWorker
// when there's no LM Studio to call, exactly the fallback already verified
// in the answerscore package).

func TestEvaluateGrounding_PositiveCase(t *testing.T) {
	out, err := ConsolidatorWorker{}.evaluateGrounding(context.Background(), "fake-model", "http://127.0.0.1:1/unreachable",
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
	out, err := ConsolidatorWorker{}.evaluateGrounding(context.Background(), "fake-model", "http://127.0.0.1:1/unreachable",
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

// When a distilled groundingscore service is wired, the consolidator uses
// it as the grounding judge and marks the verdict as independent of the
// parrot — no LM Studio call for scoring at all.
func TestEvaluateGrounding_PrefersDistilledJudge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out, _ := json.Marshal(groundingscore.Output{Score: 0.83, Confidence: 0.9})
		_ = json.NewEncoder(w).Encode(tlaloque.CapabilityResponse{
			WorkerID: groundingscore.WorkerID, Output: out, Confidence: 0.9,
		})
	}))
	defer server.Close()

	worker := ConsolidatorWorker{GroundingRegistry: groundingscore.NewRegistry(server.URL)}
	out, err := worker.evaluateGrounding(context.Background(), "parrot-model", "http://127.0.0.1:1/unreachable",
		"What is a swarm?", "A swarm is coordinating agents.", "A swarm is a system of distributed agents.")
	if err != nil {
		t.Fatalf("evaluateGrounding: %v", err)
	}
	if out.ScoredBy != groundingscore.WorkerID {
		t.Errorf("expected the distilled judge, got %q", out.ScoredBy)
	}
	if !out.JudgeIndependent {
		t.Error("distilled judge must be marked independent of the parrot")
	}
	if out.Score != 0.83 {
		t.Errorf("got score %v, want 0.83", out.Score)
	}
}

// An unreachable distilled service falls through to answerscore (which
// degrades to its deterministic keyword worker with no LM Studio).
func TestEvaluateGrounding_FallsPastUnreachableDistilledJudge(t *testing.T) {
	worker := ConsolidatorWorker{GroundingRegistry: groundingscore.NewRegistry("http://127.0.0.1:0")}
	out, err := worker.evaluateGrounding(context.Background(), "parrot-model", "http://127.0.0.1:1/unreachable",
		"What is a swarm?",
		"A swarm is a system of distributed coordinating agents.",
		"A swarm is a system composed of multiple distributed agents that coordinate their behavior.")
	if err != nil {
		t.Fatalf("evaluateGrounding: %v", err)
	}
	if out.ScoredBy == groundingscore.WorkerID {
		t.Error("expected fallback away from the unreachable distilled judge")
	}
	if out.ScoredBy == "" {
		t.Error("expected ScoredBy to name the fallback worker")
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
