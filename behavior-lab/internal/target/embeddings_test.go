package target

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbeddings_Embed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req embeddingsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.Input) != 2 {
			t.Fatalf("expected 2 inputs, got %d", len(req.Input))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"embedding": []float64{0.1, 0.2, 0.3}, "index": 1},
				{"embedding": []float64{0.4, 0.5, 0.6}, "index": 0},
			},
		})
	}))
	defer server.Close()

	client := Embeddings{BaseURL: server.URL, Model: "test-embedding"}
	vectors, err := client.Embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vectors) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vectors))
	}
	if vectors[0][0] != 0.4 {
		t.Errorf("expected vectors reordered by index; vectors[0]=%v", vectors[0])
	}
	if vectors[1][0] != 0.1 {
		t.Errorf("expected vectors reordered by index; vectors[1]=%v", vectors[1])
	}
}

func TestEmbeddings_Embed_RequiresModel(t *testing.T) {
	client := Embeddings{BaseURL: "http://127.0.0.1:1"}
	if _, err := client.Embed(context.Background(), []string{"x"}); err == nil {
		t.Error("expected an error when Model is empty")
	}
}
