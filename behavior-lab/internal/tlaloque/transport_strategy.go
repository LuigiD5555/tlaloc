package tlaloque

import (
	"fmt"
	"strings"
	"time"
)

type WorkerTransportStrategy interface {
	Validate(WorkerSpec, CapabilityDescriptor) error
	Build(WorkerSpec, time.Duration) CapabilityWorker
}

type workerTransportStrategy struct {
	validate func(WorkerSpec, CapabilityDescriptor) error
	build    func(WorkerSpec, time.Duration) CapabilityWorker
}

func (s workerTransportStrategy) Validate(spec WorkerSpec, desc CapabilityDescriptor) error {
	return s.validate(spec, desc)
}

func (s workerTransportStrategy) Build(spec WorkerSpec, timeout time.Duration) CapabilityWorker {
	return s.build(spec, timeout)
}

var workerTransportStrategies = map[string]WorkerTransportStrategy{
	TransportProcess: workerTransportStrategy{
		validate: func(spec WorkerSpec, desc CapabilityDescriptor) error {
			if len(spec.Command) == 0 {
				return fmt.Errorf("worker %q command is required for PROCESS transport", desc.ID)
			}
			return nil
		},
		build: func(spec WorkerSpec, timeout time.Duration) CapabilityWorker {
			return ProcessWorker{Desc: spec.Descriptor, Command: spec.Command, Timeout: timeout}
		},
	},
	TransportHTTPJSON: workerTransportStrategy{
		validate: func(spec WorkerSpec, desc CapabilityDescriptor) error {
			if strings.TrimSpace(spec.Endpoint) == "" {
				return fmt.Errorf("worker %q endpoint is required for HTTP_JSON transport", desc.ID)
			}
			return nil
		},
		build: func(spec WorkerSpec, timeout time.Duration) CapabilityWorker {
			return HTTPWorker{Desc: spec.Descriptor, Endpoint: spec.Endpoint, Timeout: timeout}
		},
	},
}

func normalizedTransport(spec WorkerSpec) string {
	name := strings.ToUpper(strings.TrimSpace(spec.Transport))
	if name != "" {
		return name
	}
	if strings.TrimSpace(spec.Endpoint) != "" {
		return TransportHTTPJSON
	}
	return TransportProcess
}

func resolveTransportStrategy(name string) (WorkerTransportStrategy, error) {
	strategy, ok := workerTransportStrategies[name]
	if !ok {
		return nil, fmt.Errorf("unsupported transport %q", name)
	}
	return strategy, nil
}
