package groundingscore

import (
	"time"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// NewRegistry registers groundingscore-distilled-r0 as an HTTP_JSON
// tlaloque.HTTPWorker pointed at a running tools/grounding_serve.py
// instance. The timeout covers the service's own calls to the LM Studio
// embedding endpoint, so it is more generous than questionclass's.
func NewRegistry(endpoint string) *tlaloque.Registry {
	registry := tlaloque.NewRegistry()
	_ = registry.Register(tlaloque.HTTPWorker{
		Desc:     Descriptor(),
		Endpoint: endpoint,
		Timeout:  20 * time.Second,
	})
	return registry
}
