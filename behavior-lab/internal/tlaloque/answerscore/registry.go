package answerscore

import (
	"time"

	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// NewRegistry builds a tlaloque.Registry with the SCORE_ANSWER_RELEVANCE
// workers registered: the in-process semantic model judge
// (model/baseURL-backed), its deterministic keyword-overlap fallback, and,
// when embeddingServiceURL is non-empty, the resident embedding-similarity
// worker served by cmd/tlaloc-embedding-scorer over HTTP_JSON. An empty
// embeddingServiceURL keeps the previous two-worker behavior unchanged.
func NewRegistry(model, baseURL, embeddingServiceURL string) *tlaloque.Registry {
	registry := tlaloque.NewRegistry()
	if embeddingServiceURL != "" {
		_ = registry.Register(tlaloque.HTTPWorker{
			Desc:     EmbeddingScoreDescriptor(""),
			Endpoint: embeddingServiceURL,
			Timeout:  10 * time.Second,
		})
	}
	_ = registry.Register(SemanticModelWorker{Client: target.OpenAICompat{Model: model, BaseURL: baseURL}})
	_ = registry.Register(KeywordOverlapWorker{})
	return registry
}
