package semanticverify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
	"tlaloc.local/behaviorlab/internal/tlaloque/answerscore"
)

// A fake SCORE_ANSWER_RELEVANCE HTTP judge so the adapter can be tested
// without LM Studio. answerscore.NewRegistry wires an embedding-service
// worker at this URL.
func fakeScorer(t *testing.T, score, confidence float64) *tlaloque.Registry {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req tlaloque.CapabilityRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		out, _ := json.Marshal(answerscore.ScoreOutput{Score: score, Confidence: confidence})
		_ = json.NewEncoder(w).Encode(tlaloque.CapabilityResponse{WorkerID: answerscore.EmbeddingScoreWorkerID, Output: out})
	}))
	t.Cleanup(server.Close)
	return answerscore.NewRegistry("", "", server.URL)
}

func TestAgreesWith(t *testing.T) {
	ctx := context.Background()

	high := Answerscore{Registry: fakeScorer(t, 0.88, 0.9), AgreeThreshold: 0.6}
	agree, confidence, err := high.AgreesWith(ctx, "swarms self-organize", "the page describes decentralized self-organization")
	if err != nil {
		t.Fatalf("AgreesWith: %v", err)
	}
	if !agree || confidence != 0.9 {
		t.Errorf("expected agreement at high score: agree=%v conf=%v", agree, confidence)
	}

	low := Answerscore{Registry: fakeScorer(t, 0.31, 0.8)}
	agree, _, err = low.AgreesWith(ctx, "claim", "unrelated evidence")
	if err != nil {
		t.Fatalf("AgreesWith: %v", err)
	}
	if agree {
		t.Error("a below-threshold score must not count as agreement")
	}
}
