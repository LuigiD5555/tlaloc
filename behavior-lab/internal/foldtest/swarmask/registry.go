package swarmask

import "tlaloc.local/behaviorlab/internal/tlaloque"

// NewRegistry builds a tlaloque.Registry with the scout and loro workers
// registered in-process — same convention as answerscore/questiongen and
// internal/lfm2boundary/campaign.go: constructed directly in Go, no
// manifest JSON.
func NewRegistry() *tlaloque.Registry {
	registry := tlaloque.NewRegistry()
	_ = registry.Register(PageScoutWorker{})
	_ = registry.Register(EntityScoutWorker{})
	_ = registry.Register(QuestionClassifierWorker{})
	_ = registry.Register(LoroAnswerWorker{})
	_ = registry.Register(ConsolidatorWorker{})
	return registry
}
