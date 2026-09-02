package microisadecoder

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func fakeMicroISAServer(t *testing.T, workerID string, output GlyphOutput) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req tlaloque.CapabilityRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		var in GlyphInput
		if err := json.Unmarshal(req.Input, &in); err != nil {
			t.Fatalf("decode input: %v", err)
		}
		if in.CarrierPNGBase64 == "" {
			t.Error("expected a non-empty base64-encoded carrier image")
		}

		outputRaw, _ := json.Marshal(output)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tlaloque.CapabilityResponse{WorkerID: workerID, Output: outputRaw})
	}))
}

// Positive case: a well-formed response round-trips into the correct
// GlyphOutput fields.
func TestDecodeCarrier_PositiveCase(t *testing.T) {
	server := fakeMicroISAServer(t, WorkerID, GlyphOutput{Shape: 1, Holes: 1, Direction: 0, Frames: 0})
	defer server.Close()

	registry := NewRegistry(server.URL)
	out, err := DecodeCarrier(context.Background(), registry, []byte("fake-png-bytes"))
	if err != nil {
		t.Fatalf("DecodeCarrier: %v", err)
	}
	if out != (GlyphOutput{Shape: 1, Holes: 1, Direction: 0, Frames: 0}) {
		t.Errorf("unexpected output: %+v", out)
	}
}

// Identity check: HTTPWorker (internal/tlaloque/http_worker.go, unmodified)
// already rejects a response whose worker_id doesn't match the registered
// descriptor — this confirms that protection is actually wired up here,
// not just present in the library.
func TestDecodeCarrier_RejectsWorkerIdentityMismatch(t *testing.T) {
	server := fakeMicroISAServer(t, "some-other-worker", GlyphOutput{Shape: 1})
	defer server.Close()

	registry := NewRegistry(server.URL)
	_, err := DecodeCarrier(context.Background(), registry, []byte("fake-png-bytes"))
	if err == nil {
		t.Fatal("expected an error for a mismatched worker_id, got nil")
	}
}
