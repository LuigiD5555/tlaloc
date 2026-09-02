package groundingscore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func fakeScorerServer(t *testing.T, workerID string, output Output) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req tlaloque.CapabilityRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		var in Input
		if err := json.Unmarshal(req.Input, &in); err != nil {
			t.Fatalf("decode input: %v", err)
		}
		if in.ModelAnswer == "" || in.PageContent == "" {
			t.Error("expected non-empty model_answer and page_content")
		}
		outputRaw, _ := json.Marshal(output)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tlaloque.CapabilityResponse{
			WorkerID: workerID, Output: outputRaw, Confidence: output.Confidence,
		})
	}))
}

// Positive case: a well-formed response round-trips into the score and the
// model's self-estimated confidence.
func TestScore_PositiveCase(t *testing.T) {
	server := fakeScorerServer(t, WorkerID, Output{Score: 0.82, Confidence: 0.9, Notes: "distilled"})
	defer server.Close()

	registry := NewRegistry(server.URL)
	out, confidence, err := Score(context.Background(), registry, Input{
		Question: "what is a swarm", ModelAnswer: "a group of agents", PageContent: "a swarm is a group of agents",
	})
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	if out.Score != 0.82 {
		t.Errorf("got score %v, want 0.82", out.Score)
	}
	if confidence != 0.9 {
		t.Errorf("got confidence %v, want 0.9", confidence)
	}
}

// Identity check: HTTPWorker (unmodified) rejects a response whose
// worker_id doesn't match the registered descriptor.
func TestScore_RejectsWorkerIdentityMismatch(t *testing.T) {
	server := fakeScorerServer(t, "some-other-worker", Output{Score: 0.5, Confidence: 1.0})
	defer server.Close()

	registry := NewRegistry(server.URL)
	if _, _, err := Score(context.Background(), registry, Input{
		ModelAnswer: "x", PageContent: "y",
	}); err == nil {
		t.Fatal("expected an error for a mismatched worker_id, got nil")
	}
}

// Unreachable service: the caller gets an error, which the consolidator
// turns into an honest fallback to answerscore.
func TestScore_UnreachableService(t *testing.T) {
	registry := NewRegistry("http://127.0.0.1:0")
	if _, _, err := Score(context.Background(), registry, Input{
		ModelAnswer: "x", PageContent: "y",
	}); err == nil {
		t.Fatal("expected an error for an unreachable service, got nil")
	}
}

// Confidence falls back to the output body when the envelope omits it.
func TestScore_ConfidenceFromBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		outputRaw, _ := json.Marshal(Output{Score: 0.4, Confidence: 0.55})
		_ = json.NewEncoder(w).Encode(tlaloque.CapabilityResponse{WorkerID: WorkerID, Output: outputRaw})
	}))
	defer server.Close()

	_, confidence, err := Score(context.Background(), NewRegistry(server.URL), Input{
		ModelAnswer: "x", PageContent: "y",
	})
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	if confidence != 0.55 {
		t.Errorf("got confidence %v, want 0.55 (from body)", confidence)
	}
}
