package swarmask

import (
	"tlaloc.local/behaviorlab/internal/tlaloque"
	"tlaloc.local/behaviorlab/internal/tlaloque/questionclass"
)

// NewRegistry builds a tlaloque.Registry with the scout and loro workers
// registered in-process — same convention as answerscore/questiongen and
// internal/lfm2boundary/campaign.go: constructed directly in Go, no
// manifest JSON. classifierEndpoint, when non-empty, backs the question
// classifier node with the trained questionclass-charcnn-r0 HTTP service
// (rule-based classifyQuestion stays as its honest fallback).
func NewRegistry(classifierEndpoint string) *tlaloque.Registry {
	registry := tlaloque.NewRegistry()
	_ = registry.Register(PageScoutWorker{})
	_ = registry.Register(EntityScoutWorker{})

	classifier := QuestionClassifierWorker{}
	if classifierEndpoint != "" {
		classifier.ModelRegistry = questionclass.NewRegistry(classifierEndpoint)
	}
	_ = registry.Register(classifier)

	_ = registry.Register(LoroAnswerWorker{})
	_ = registry.Register(ConsolidatorWorker{})
	return registry
}
