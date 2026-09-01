package tlaloque

import (
	"context"
	"encoding/json"
	"testing"
)

type registeringProvisioner struct {
	registry *Registry
	calls    int
}

func (provisioner *registeringProvisioner) Provision(_ context.Context, request SelectionRequest) error {
	provisioner.calls++
	return provisioner.registry.Register(testWorker{desc: generalDescriptor("intent-v1", request.Capability)})
}

type metricCollector struct {
	metrics []ExecutionMetric
}

func (collector *metricCollector) ObserveExecution(_ context.Context, metric ExecutionMetric) {
	collector.metrics = append(collector.metrics, metric)
}

func TestSwarmProvisionsMissingWorkerAndEmitsMetric(t *testing.T) {
	registry := NewRegistry()
	provisioner := &registeringProvisioner{registry: registry}
	observer := &metricCollector{}
	plan := SwarmPlan{ID: "on-demand", Nodes: []SwarmNode{{ID: "intent", Capability: "INTENT"}}}
	report, err := (SwarmRunner{Registry: registry, Provisioner: provisioner, Observer: observer}).Run(
		context.Background(), plan, "task", json.RawMessage(`{}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Succeeded || provisioner.calls != 1 {
		t.Fatalf("report=%+v provision calls=%d", report, provisioner.calls)
	}
	if len(observer.metrics) != 1 || !observer.metrics[0].Succeeded || observer.metrics[0].WorkerID != "intent-v1" {
		t.Fatalf("unexpected metrics: %+v", observer.metrics)
	}
}
