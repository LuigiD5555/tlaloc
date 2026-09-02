package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"tlaloc.local/behaviorlab/internal/tlaloque"
	"tlaloc.local/behaviorlab/internal/tlaloque/answerscore"
)

// Embedder is the minimal capability this server needs from a client
// talking to LM Studio's /embeddings endpoint. Defined here so scoreHandler
// can be tested against a fake without any network access.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float64, error)
}

// scoreHandler implements the tlaloque.CapabilityWorker HTTP_JSON contract
// (see internal/tlaloque/http_worker.go) for SCORE_ANSWER_RELEVANCE, backed
// by embedding cosine similarity instead of a chat-completion judge.
func scoreHandler(embedder Embedder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req tlaloque.CapabilityRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}

		var in answerscore.ScoreInput
		if err := json.Unmarshal(req.Input, &in); err != nil {
			http.Error(w, fmt.Sprintf("invalid input: %v", err), http.StatusBadRequest)
			return
		}

		passages := answerscore.SplitIntoPassages(in.PageContent)
		if len(passages) == 0 {
			passages = []string{in.PageContent}
		}

		vectors, err := embedder.Embed(r.Context(), append([]string{in.ModelAnswer}, passages...))
		if err != nil {
			http.Error(w, fmt.Sprintf("embedding failed: %v", err), http.StatusBadGateway)
			return
		}
		if len(vectors) != len(passages)+1 {
			http.Error(w, fmt.Sprintf("expected %d embedding vectors, got %d", len(passages)+1, len(vectors)), http.StatusBadGateway)
			return
		}

		out := answerscore.ScoreByBestPassageSimilarity(vectors[0], vectors[1:])
		outputRaw, err := json.Marshal(out)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := tlaloque.CapabilityResponse{
			WorkerID:   answerscore.EmbeddingScoreWorkerID,
			Output:     outputRaw,
			Confidence: out.Confidence,
			Notes:      out.Notes,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// healthHandler is not required by the HTTPWorker contract but makes the
// resident service self-describing for manual inspection.
func healthHandler(embeddingModel string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":     "ok",
			"descriptor": answerscore.EmbeddingScoreDescriptor(embeddingModel),
		})
	}
}
