package microisadecoder

import (
	"time"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// NewRegistry registers microisa-cnn-r0 as an HTTP_JSON tlaloque.HTTPWorker
// pointed at a running origami/tools/microisa_serve.py instance. HTTPWorker
// already implements the full CapabilityWorker contract (worker-identity
// check, JSON validity — internal/tlaloque/http_worker.go); there is no
// worker-specific Execute() to write here.
func NewRegistry(endpoint string) *tlaloque.Registry {
	registry := tlaloque.NewRegistry()
	_ = registry.Register(tlaloque.HTTPWorker{
		Desc:     MicroISADescriptor(),
		Endpoint: endpoint,
		Timeout:  10 * time.Second,
	})
	return registry
}
