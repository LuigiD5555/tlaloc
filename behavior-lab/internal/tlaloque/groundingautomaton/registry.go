package groundingautomaton

import "tlaloc.local/behaviorlab/internal/tlaloque"

func NewRegistry() *tlaloque.Registry {
	registry := tlaloque.NewRegistry()
	_ = registry.Register(Worker{})
	return registry
}
