package target

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Embeddings calls an OpenAI-compatible /embeddings endpoint, such as the
// one LM Studio exposes for resident embedding models (e.g. MiniLM, BGE,
// Nomic) alongside its chat-completion models.
type Embeddings struct {
	BaseURL        string
	Model          string
	Client         *http.Client
	RequestTimeout time.Duration
}

type embeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingsResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

// Embed returns one embedding vector per input text, in the same order.
func (e Embeddings) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	base := strings.TrimRight(e.BaseURL, "/")
	if base == "" {
		base = "http://127.0.0.1:1234/v1"
	}
	if e.Model == "" {
		return nil, fmt.Errorf("embedding model is required")
	}
	client := httpClientForTimeout(ctx, e.Client, e.RequestTimeout)
	body, _ := json.Marshal(embeddingsRequest{Model: e.Model, Input: texts})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("embeddings target status %s: %s", resp.Status, string(b))
	}
	var out embeddingsResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	if len(out.Data) != len(texts) {
		return nil, fmt.Errorf("embeddings target returned %d vectors for %d inputs", len(out.Data), len(texts))
	}
	vectors := make([][]float64, len(out.Data))
	for _, d := range out.Data {
		if d.Index < 0 || d.Index >= len(vectors) {
			return nil, fmt.Errorf("embeddings target returned out-of-range index %d", d.Index)
		}
		vectors[d.Index] = d.Embedding
	}
	return vectors, nil
}
