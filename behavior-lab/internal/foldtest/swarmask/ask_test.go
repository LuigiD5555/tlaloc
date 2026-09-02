package swarmask

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

// fakeLMStudio simulates the OpenAI-compatible /chat/completions endpoint
// with a single fixed answer, so this test never touches real LM Studio.
func fakeLMStudio(t *testing.T, answer string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": answer}},
			},
		})
	}))
}

// Ask must run scout before answer, persist the scout's observation on the
// shared blackboard Store, and extract the parrot's answer from the
// consolidate node's terminal output. The test manifest has no pages, so
// the consolidator's ExtractPageContent lookup fails and it degrades
// gracefully (no grounding verdict) instead of failing the whole run —
// this exercises that degradation path for free.
func TestAsk_SharesScoutObservationWithParrot(t *testing.T) {
	server := fakeLMStudio(t, "The parrot's answer, informed by the blackboard.")
	defer server.Close()

	store := blackboard.New(filepath.Join(t.TempDir(), "blackboard"))

	out, report, err := Ask(context.Background(), store, "test-run", AskInput{
		Question: "¿Cómo coordinan los agentes del enjambre?",
		Cover:    testCover,
		WorkDir:  t.TempDir(),
		Manifest: pdfmemory.Manifest{},
		Model:    "fake-model",
		BaseURL:  server.URL,
		MaxTurns: 3,
	})
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !report.Succeeded {
		t.Fatalf("expected the swarm run to succeed, report: %+v", report)
	}
	if out.Answer != "The parrot's answer, informed by the blackboard." {
		t.Errorf("unexpected answer: %q", out.Answer)
	}

	entries, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Key == suggestedPageKey && e.NodeID == ScoutWorkerID {
			found = true
		}
	}
	if !found {
		t.Error("expected the scout's suggested_page observation to be persisted in the shared blackboard store")
	}
}
