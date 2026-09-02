package questiongen

import (
	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// NewRegistry builds a tlaloque.Registry with both GENERATE_PAGE_QUESTIONS
// workers registered: the semantic model generator (model/baseURL-backed)
// and its deterministic template fallback. Both run in-process, so no
// PROCESS/HTTP_JSON transport or manifest file is needed.
func NewRegistry(model, baseURL string) *tlaloque.Registry {
	registry := tlaloque.NewRegistry()
	_ = registry.Register(SemanticModelWorker{Client: target.OpenAICompat{Model: model, BaseURL: baseURL}})
	_ = registry.Register(TemplateWorker{})
	return registry
}
