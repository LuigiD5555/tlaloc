package questionclass

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func fakeClassifierServer(t *testing.T, workerID string, output Output, confidence float64) *httptest.Server {
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
		if in.Question == "" {
			t.Error("expected a non-empty question")
		}

		outputRaw, _ := json.Marshal(output)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tlaloque.CapabilityResponse{WorkerID: workerID, Output: outputRaw, Confidence: confidence})
	}))
}

// Positive case: a well-formed response round-trips into the label and the
// model's confidence.
func TestClassify_PositiveCase(t *testing.T) {
	server := fakeClassifierServer(t, WorkerID, Output{Type: "COMPARISON"}, 0.97)
	defer server.Close()

	registry := NewRegistry(server.URL)
	out, confidence, err := Classify(context.Background(), registry, "compare A and B")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if out.Type != "COMPARISON" {
		t.Errorf("got type %q, want COMPARISON", out.Type)
	}
	if confidence != 0.97 {
		t.Errorf("got confidence %v, want 0.97", confidence)
	}
}

// Identity check: HTTPWorker (internal/tlaloque/http_worker.go, unmodified)
// rejects a response whose worker_id doesn't match the registered
// descriptor — this confirms that protection is wired up here.
func TestClassify_RejectsWorkerIdentityMismatch(t *testing.T) {
	server := fakeClassifierServer(t, "some-other-worker", Output{Type: "GENERAL"}, 1.0)
	defer server.Close()

	registry := NewRegistry(server.URL)
	if _, _, err := Classify(context.Background(), registry, "tell me about swarms"); err == nil {
		t.Fatal("expected an error for a mismatched worker_id, got nil")
	}
}

// Unreachable service: the caller gets an error (which swarmask turns into
// an honest fallback to the rule-based classifier).
func TestClassify_UnreachableService(t *testing.T) {
	registry := NewRegistry("http://127.0.0.1:0")
	if _, _, err := Classify(context.Background(), registry, "what is a swarm"); err == nil {
		t.Fatal("expected an error for an unreachable service, got nil")
	}
}
