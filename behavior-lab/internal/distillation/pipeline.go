package distillation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

const ArtifactSchemaR0 = "tlaloc.specialist-artifact.r0"

type Config struct {
	SyntheticDatasetSize int     `json:"synthetic_dataset_size"`
	TrainingSeed         int64   `json:"training_seed"`
	EvaluationThreshold  float64 `json:"evaluation_threshold"`
	MaxRetries           int     `json:"max_retries"`
	ModelBackend         string  `json:"model_backend"`
}

func (config Config) normalize() (Config, error) {
	if config.SyntheticDatasetSize <= 0 {
		return Config{}, fmt.Errorf("synthetic_dataset_size must be positive")
	}
	if config.EvaluationThreshold < 0 || config.EvaluationThreshold > 1 {
		return Config{}, fmt.Errorf("evaluation_threshold must be between 0 and 1")
	}
	if config.MaxRetries < 0 {
		return Config{}, fmt.Errorf("max_retries cannot be negative")
	}
	config.ModelBackend = strings.TrimSpace(config.ModelBackend)
	if config.ModelBackend == "" {
		return Config{}, fmt.Errorf("model_backend is required")
	}
	return config, nil
}

type TaskSpec struct {
	Capability     string          `json:"capability"`
	PromptPattern  string          `json:"prompt_pattern,omitempty"`
	ReferenceCases json.RawMessage `json:"reference_cases,omitempty"`
}

func (task TaskSpec) normalize() (TaskSpec, error) {
	task.Capability = strings.ToUpper(strings.TrimSpace(task.Capability))
	task.PromptPattern = strings.TrimSpace(task.PromptPattern)
	if task.Capability == "" {
		return TaskSpec{}, fmt.Errorf("capability is required")
	}
	if len(task.ReferenceCases) > 0 && !json.Valid(task.ReferenceCases) {
		return TaskSpec{}, fmt.Errorf("reference_cases must be valid JSON")
	}
	return task, nil
}

type Example struct {
	Input    json.RawMessage `json:"input"`
	Expected json.RawMessage `json:"expected"`
}

type Dataset struct {
	Training []Example `json:"training"`
	Holdout  []Example `json:"holdout"`
}

type Metrics struct {
	Accuracy float64            `json:"accuracy"`
	Values   map[string]float64 `json:"values,omitempty"`
}

type Candidate struct {
	Worker      tlaloque.CapabilityWorker
	ArtifactURI string
}

type SpecialistTrainer interface {
	GenerateDataset(context.Context, TaskSpec, int, int64) (Dataset, error)
	Train(context.Context, Dataset, Config) (Candidate, error)
	Evaluate(context.Context, Candidate, Dataset) (Metrics, error)
}

type Artifact struct {
	Schema       string    `json:"schema"`
	Capability   string    `json:"capability"`
	WorkerID     string    `json:"worker_id"`
	Backend      string    `json:"backend"`
	DatasetSize  int       `json:"dataset_size"`
	TrainingSeed int64     `json:"training_seed"`
	Attempt      int       `json:"attempt"`
	Metrics      Metrics   `json:"metrics"`
	ArtifactURI  string    `json:"artifact_uri,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type ModelStore interface {
	Save(context.Context, Artifact) error
}

type Result struct {
	Promoted bool     `json:"promoted"`
	Attempts int      `json:"attempts"`
	Artifact Artifact `json:"artifact,omitempty"`
	Metrics  Metrics  `json:"metrics"`
}

// Pipeline owns the generation, training, evaluation and promotion sequence.
// Trainers produce candidates but never receive registry promotion authority.
type Pipeline struct {
	Config   Config
	Trainer  SpecialistTrainer
	Registry *tlaloque.Registry
	Store    ModelStore
	Now      func() time.Time
}

func (pipeline Pipeline) Distill(ctx context.Context, task TaskSpec) (Result, error) {
	config, err := pipeline.Config.normalize()
	if err != nil {
		return Result{}, err
	}
	task, err = task.normalize()
	if err != nil {
		return Result{}, err
	}
	if pipeline.Trainer == nil || pipeline.Registry == nil || pipeline.Store == nil {
		return Result{}, fmt.Errorf("trainer, registry and store are required")
	}
	now := pipeline.Now
	if now == nil {
		now = time.Now
	}

	result := Result{}
	for attempt := 1; attempt <= config.MaxRetries+1; attempt++ {
		result.Attempts = attempt
		dataset, generateErr := pipeline.Trainer.GenerateDataset(ctx, task, config.SyntheticDatasetSize, config.TrainingSeed+int64(attempt-1))
		if generateErr != nil {
			if attempt <= config.MaxRetries {
				continue
			}
			return result, fmt.Errorf("generate dataset: %w", generateErr)
		}
		candidate, trainErr := pipeline.Trainer.Train(ctx, dataset, config)
		if trainErr != nil {
			if attempt <= config.MaxRetries {
				continue
			}
			return result, fmt.Errorf("train specialist: %w", trainErr)
		}
		if candidate.Worker == nil {
			return result, fmt.Errorf("trainer returned a nil worker")
		}
		descriptor, descriptorErr := candidate.Worker.Descriptor().Normalize()
		if descriptorErr != nil {
			return result, fmt.Errorf("candidate descriptor: %w", descriptorErr)
		}
		if descriptor.Capability != task.Capability {
			return result, fmt.Errorf("candidate capability=%s, want %s", descriptor.Capability, task.Capability)
		}
		metrics, evaluateErr := pipeline.Trainer.Evaluate(ctx, candidate, dataset)
		result.Metrics = metrics
		if evaluateErr != nil {
			if attempt <= config.MaxRetries {
				continue
			}
			return result, fmt.Errorf("evaluate specialist: %w", evaluateErr)
		}
		if metrics.Accuracy < config.EvaluationThreshold {
			continue
		}
		artifact := Artifact{
			Schema: ArtifactSchemaR0, Capability: task.Capability, WorkerID: descriptor.ID,
			Backend: config.ModelBackend, DatasetSize: len(dataset.Training) + len(dataset.Holdout),
			TrainingSeed: config.TrainingSeed + int64(attempt-1), Attempt: attempt,
			Metrics: metrics, ArtifactURI: candidate.ArtifactURI, CreatedAt: now().UTC(),
		}
		if err := pipeline.Registry.Register(candidate.Worker); err != nil {
			return result, fmt.Errorf("register candidate: %w", err)
		}
		if err := pipeline.Store.Save(ctx, artifact); err != nil {
			_ = pipeline.Registry.Unregister(descriptor.ID)
			return result, fmt.Errorf("save artifact: %w", err)
		}
		if err := pipeline.Registry.Activate(descriptor.ID); err != nil {
			_ = pipeline.Registry.Unregister(descriptor.ID)
			return result, fmt.Errorf("activate candidate: %w", err)
		}
		result.Promoted = true
		result.Artifact = artifact
		return result, nil
	}
	return result, nil
}

// Provision lets SwarmRunner request a missing capability without depending
// on the concrete distillation package. A pinned worker ID is passed as the
// prompt pattern so a trainer may honor plans that require a named version.
func (pipeline Pipeline) Provision(ctx context.Context, request tlaloque.SelectionRequest) error {
	return pipeline.distillAndRequirePromotion(ctx, TaskSpec{Capability: request.Capability, PromptPattern: request.WorkerID})
}

// Schedule makes Pipeline the synchronous scheduler for explicit CLI flows.
// A background queue can implement TrainingScheduler instead without changing
// monitoring or runtime contracts.
func (pipeline Pipeline) Schedule(ctx context.Context, request TrainingRequest) error {
	return pipeline.distillAndRequirePromotion(ctx, request.Task)
}

func (pipeline Pipeline) distillAndRequirePromotion(ctx context.Context, task TaskSpec) error {
	result, err := pipeline.Distill(ctx, task)
	if err != nil {
		return err
	}
	if !result.Promoted {
		return fmt.Errorf("candidate did not reach evaluation threshold")
	}
	return nil
}
