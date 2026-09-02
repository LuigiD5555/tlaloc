package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
	"tlaloc.local/behaviorlab/internal/tlaloque/answerscore"
)

type fakeEmbedder struct {
	embed func(texts []string) ([][]float64, error)
	err   error
}

func (f fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float64, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.embed(texts)
}

// identicalVectorsEmbedder returns the same fixed vector for every input
// text, regardless of how many passages the handler splits the page into.
func identicalVectorsEmbedder() fakeEmbedder {
	return fakeEmbedder{embed: func(texts []string) ([][]float64, error) {
		vectors := make([][]float64, len(texts))
		for i := range texts {
			vectors[i] = []float64{1, 0, 0}
		}
		return vectors, nil
	}}
}

func postScore(t *testing.T, handler http.HandlerFunc, in answerscore.ScoreInput) *httptest.ResponseRecorder {
	t.Helper()
	inputRaw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	reqBody, err := json.Marshal(tlaloque.CapabilityRequest{Input: inputRaw})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/execute", strings.NewReader(string(reqBody)))
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

// Positive case: a healthy embedder produces a well-formed CapabilityResponse
// carrying the embedding worker's identity.
func TestScoreHandler_PositiveCase(t *testing.T) {
	handler := scoreHandler(identicalVectorsEmbedder())
	rec := postScore(t, handler, answerscore.ScoreInput{
		ModelAnswer: "respuesta",
		PageContent: "Esta es la primera oración relevante del contenido. Esta es la segunda oración relevante.",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp tlaloque.CapabilityResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.WorkerID != answerscore.EmbeddingScoreWorkerID {
		t.Errorf("expected worker id %q, got %q", answerscore.EmbeddingScoreWorkerID, resp.WorkerID)
	}
	var out answerscore.ScoreOutput
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if out.Score != 1.0 {
		t.Errorf("expected score 1.0 for identical vectors, got %f", out.Score)
	}
}

// Malformed request body must be rejected with a client error, not a panic
// or a silently wrong score.
func TestScoreHandler_InvalidRequestBody(t *testing.T) {
	handler := scoreHandler(identicalVectorsEmbedder())
	req := httptest.NewRequest(http.MethodPost, "/execute", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON body, got %d", rec.Code)
	}
}

// An embedder failure (LM Studio unreachable) must surface as a clean error
// response rather than a fabricated score.
func TestScoreHandler_EmbedderError(t *testing.T) {
	handler := scoreHandler(fakeEmbedder{err: errors.New("connection refused")})
	rec := postScore(t, handler, answerscore.ScoreInput{ModelAnswer: "respuesta", PageContent: "contenido"})

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502 when the embedder fails, got %d", rec.Code)
	}
}
