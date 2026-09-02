package questionclass

import (
	"time"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// NewRegistry registers questionclass-charcnn-r0 as an HTTP_JSON
// tlaloque.HTTPWorker pointed at a running tools/questionclass_serve.py
// instance. HTTPWorker already implements the full CapabilityWorker
// contract (worker-identity check, JSON validity — see
// internal/tlaloque/http_worker.go); there is no worker-specific Execute()
// to write here. Same convention as internal/tlaloque/microisadecoder.
func NewRegistry(endpoint string) *tlaloque.Registry {
	registry := tlaloque.NewRegistry()
	_ = registry.Register(tlaloque.HTTPWorker{
		Desc:     Descriptor(),
		Endpoint: endpoint,
		Timeout:  5 * time.Second,
	})
	return registry
}
