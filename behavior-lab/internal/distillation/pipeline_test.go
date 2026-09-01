package distillation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

type candidateWorker struct {
	id string
}

func (worker candidateWorker) Descriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{ID: worker.id, Capability: "INTENT", InputSchema: "input", OutputSchema: "output"}
}

func (worker candidateWorker) Execute(context.Context, tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	return tlaloque.CapabilityResponse{WorkerID: worker.id, Output: json.RawMessage(`{}`)}, nil
}

type scriptedTrainer struct {
	accuracies []float64
	attempt    int
}

func (trainer *scriptedTrainer) GenerateDataset(context.Context, TaskSpec, int, int64) (Dataset, error) {
	return Dataset{
		Training: []Example{{Input: json.RawMessage(`{}`), Expected: json.RawMessage(`{}`)}},
		Holdout:  []Example{{Input: json.RawMessage(`{}`), Expected: json.RawMessage(`{}`)}},
	}, nil
}

func (trainer *scriptedTrainer) Train(context.Context, Dataset, Config) (Candidate, error) {
	trainer.attempt++
	return Candidate{Worker: candidateWorker{id: fmt.Sprintf("intent-v%d", trainer.attempt)}}, nil
}

func (trainer *scriptedTrainer) Evaluate(context.Context, Candidate, Dataset) (Metrics, error) {
	return Metrics{Accuracy: trainer.accuracies[trainer.attempt-1]}, nil
}

type memoryStore struct {
	artifacts []Artifact
	err       error
}

func (store *memoryStore) Save(_ context.Context, artifact Artifact) error {
	if store.err != nil {
		return store.err
	}
	store.artifacts = append(store.artifacts, artifact)
	return nil
}

func TestPipelineRetriesAndPromotesPassingCandidate(t *testing.T) {
	registry := tlaloque.NewRegistry()
	trainer := &scriptedTrainer{accuracies: []float64{0.6, 0.9}}
	store := &memoryStore{}
	pipeline := Pipeline{
		Config:  Config{SyntheticDatasetSize: 10, TrainingSeed: 41, EvaluationThreshold: 0.8, MaxRetries: 1, ModelBackend: "test"},
		Trainer: trainer, Registry: registry, Store: store,
		Now: func() time.Time { return time.Unix(100, 0) },
	}
	result, err := pipeline.Distill(context.Background(), TaskSpec{Capability: "intent"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Promoted || result.Attempts != 2 || result.Artifact.WorkerID != "intent-v2" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if activeID, ok := registry.ActiveWorkerID("INTENT"); !ok || activeID != "intent-v2" {
		t.Fatalf("active worker=(%q,%t), want intent-v2", activeID, ok)
	}
	if len(store.artifacts) != 1 || store.artifacts[0].TrainingSeed != 42 {
		t.Fatalf("unexpected artifacts: %+v", store.artifacts)
	}
}

func TestPipelineDoesNotPromoteWhenArtifactPersistenceFails(t *testing.T) {
	registry := tlaloque.NewRegistry()
	pipeline := Pipeline{
		Config:  Config{SyntheticDatasetSize: 1, EvaluationThreshold: 0.5, ModelBackend: "test"},
		Trainer: &scriptedTrainer{accuracies: []float64{1}}, Registry: registry,
		Store: &memoryStore{err: errors.New("disk full")},
	}
	if _, err := pipeline.Distill(context.Background(), TaskSpec{Capability: "INTENT"}); err == nil {
		t.Fatal("persistence failure was accepted")
	}
	if _, exists := registry.Get("intent-v1"); exists {
		t.Fatal("unpersisted candidate remained registered")
	}
}

type requestCollector struct {
	requests []TrainingRequest
}

func (collector *requestCollector) Schedule(_ context.Context, request TrainingRequest) error {
	collector.requests = append(collector.requests, request)
	return nil
}

func TestMonitorCoversAllTriggerSignals(t *testing.T) {
	collector := &requestCollector{}
	monitor := &Monitor{Config: MonitorConfig{RepeatThreshold: 2, AccuracyThreshold: 0.8, CostBudget: 3}, Scheduler: collector}
	metric := tlaloque.ExecutionMetric{Capability: "intent", Succeeded: true}
	monitor.ObserveExecution(context.Background(), metric)
	monitor.ObserveExecution(context.Background(), metric)
	if err := monitor.RecordAccuracy(context.Background(), TaskSpec{Capability: "entity"}, 0.7); err != nil {
		t.Fatal(err)
	}
	if err := monitor.RecordCost(context.Background(), TaskSpec{Capability: "route"}, 3); err != nil {
		t.Fatal(err)
	}
	if err := monitor.Demand(context.Background(), TaskSpec{Capability: "date"}); err != nil {
		t.Fatal(err)
	}
	want := []TriggerReason{TriggerRepeatedTask, TriggerAccuracyDrop, TriggerCostBudget, TriggerExplicitDemand}
	if len(collector.requests) != len(want) {
		t.Fatalf("got %d requests, want %d", len(collector.requests), len(want))
	}
	for index, reason := range want {
		if collector.requests[index].Reason != reason {
			t.Fatalf("request %d reason=%s, want %s", index, collector.requests[index].Reason, reason)
		}
	}
}
