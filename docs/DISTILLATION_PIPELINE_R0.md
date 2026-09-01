# Distillation Pipeline R0

## Status

V1 foundation implemented. The model-specific training backend remains an
explicit adapter and is not silently simulated by the runtime.

## Active path

1. `tlaloque.SwarmRunner` selects an active worker from the shared registry.
2. When selection fails, an optional `WorkerProvisioner` may create and promote
   a specialist, after which selection is retried once.
3. Every executed node emits an `ExecutionMetric` to an optional observer.
4. `distillation.Monitor` converts repetition, accuracy, explicit-demand and
   cost signals into `TrainingRequest` values.
5. `distillation.Pipeline` generates a dataset, trains, evaluates and retries a
   candidate. It registers and activates only candidates meeting the configured
   threshold and whose metadata was persisted successfully.
6. `tlaloque.Registry` retains older versions for pinned execution or rollback,
   while exposing only one active default version per capability.

## Authority boundaries

- Trainers create candidates and metrics; they cannot promote themselves.
- The pipeline owns evaluation-gated promotion.
- The registry owns active-version state and rollback.
- The DAG runtime owns execution only and depends on interfaces for observation
  and provisioning.
- A scheduler owns synchronous, subprocess or background execution policy.

## Backend adapter contract

Implement `distillation.SpecialistTrainer` for the selected backend. LM Studio
probe generation and `swarmbench.ScoreDataset` remain reusable evidence sources,
but are not coupled into the generic pipeline because their current dataset is
specific to the V0 banking benchmark.

The trainer must return a `tlaloque.CapabilityWorker` with a unique versioned ID
such as `intent-v2`, an artifact URI, and holdout accuracy. The pipeline writes
reproducibility metadata through `ModelStore`; `FileStore` provides the local
filesystem implementation.
