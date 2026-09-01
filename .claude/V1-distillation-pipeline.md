# Tlaloc V1: Distillation Pipeline & Auto-trained Specialists

**Status**: V1 foundation integrated. Backend-specific synthetic generation and model training remain adapter work; see `docs/DISTILLATION_PIPELINE_R0.md` for the active contract.

## Problem Statement

V0 proves that decomposition + tiny-models beats SLM-only. But the swarm's workers are **static**: manually trained once, then frozen.

V1 closes the loop: **Tlaloc generates and evolves its own specialists** as the system detects repeated tasks or accuracy pressure. This is the "systematize origami creation" thesis — the system doesn't just execute behavior, it synthesizes new behavior.

## Core Insight: Distillation as Bootstrap

```
SLM (LFM2/Gemma/Claude Sonnet)
  ↓ prompt engineer for micro-task
  ↓ generate N synthetic examples with varied prompting
Dataset (synthetic, task-specific)
  ↓
Tlaloc training job:
  - take dataset
  - train tiny model (BERT/DistilBERT/linear classifier)
  - eval against reference cases
  - if accuracy ≥ threshold → register as permanent worker
  - if not → discard or retry with more data
  ↓
Tiny model enters registry as [capability]-v{N}
```

This is **knowledge distillation**: compress the SLM's capability for one micro-task into a tiny model that runs locally, needs no API calls, and is deterministic given seed.

## Architecture Changes

### 1. Registry Lifecycle (was static, becomes dynamic)

**Today** (V0):
```go
registry, err := swarmbench.BuildInProcessRegistryWithLogic(...)
// workers are fixed; run the DAG; done
```

**Tomorrow** (V1):
```go
registry := tlaloc.NewDynamicRegistry(masterModel, trainingConfig)
// workers can be added at runtime
// if swarm detects repeated task → auto-train specialist
// if accuracy drops → trigger retraining

plan := tlaloc.BuildPlanWithAutoSpecialists(businessRules)
// plan can reference workers that don't exist yet
// runtime creates them on demand
```

### 2. Training Pipeline (new component)

**Responsibilities**:
- Detect when a task is repeated enough to justify distillation
- Generate synthetic dataset via SLM prompting (deterministic seed)
- Train candidate model (CPU-only, in-process or subprocess)
- Evaluate against hold-out set
- Version and promote to registry if threshold met
- Fallback to SLM if training fails

**Interfaces** (sketch):
```go
type DistillationConfig struct {
    SynthDatasetSize int          // how many examples to generate
    TrainingSeed     int64        // deterministic
    EvalThreshold    float64      // e.g., 0.75 accuracy to promote
    MaxRetries       int
    ModelBackend     string       // "sklearn", "transformers", "onnx"
}

type SpecialistTrainer interface {
    // Generate synthetic examples for one capability
    GenerateDataset(ctx context.Context, task TaskSpec, size int) (Dataset, error)

    // Train tiny model on dataset
    Train(ctx context.Context, dataset Dataset, config DistillationConfig) (Model, error)

    // Eval model on test set, return accuracy/metrics
    Evaluate(ctx context.Context, model Model, testSet Dataset) (Metrics, error)
}

// Registry can accept runtime-added workers
type DynamicRegistry struct {
    permanent map[string]Worker
    temp      map[string]Worker // auto-trained, TTL-based invalidation
    mu        sync.RWMutex
}

func (r *DynamicRegistry) RegisterWorker(id string, w Worker) error
func (r *DynamicRegistry) UnregisterWorker(id string) error
```

### 3. Training Trigger Points

**When to train a new specialist**:

1. **Repeated task detection**: if the same (capability, prompt pattern) appears N times in a window → maybe worth distilling
2. **Accuracy monitoring**: if a capability's average accuracy drops below threshold → retrain with newer data
3. **Explicit demand**: user/system marks a task as "high-volume, distill this"
4. **Cost signal**: if API calls for one worker exceed budget → force distillation

**Where the trigger lives**:
- SwarmRunner observes execution → feeds metrics to a monitoring layer
- Monitoring layer decides "train now?"
- If yes, spawn training job async (background or explicit CLI command)
- Once trained model passes eval → promote to registry

### 4. Model Versioning & Promotion

Each specialist gets a **capability-version** identity:
```
intent-v1 (manually trained in V0)
intent-v2 (auto-trained after 1000 calls)
intent-v3 (auto-trained after drift detected)
```

Rules:
- Only one version is "active" at a time (e.g., intent → intent-v3)
- Older versions kept for rollback / A/B testing
- Training artifacts versioned (dataset seed, hyperparams, eval metrics) for reproducibility
- If new version underperforms, auto-rollback

### 5. Integration Points with Existing Code

**`internal/swarmbench/`** (Phase 5 prove-out):
- Reuse `GenerateLMStudioProbe()` logic as the SynthDataset generator
- Reuse `ScoreDataset()` as the evaluator
- Keep lfm2vl_proxy.go as the reference distilled model

**`internal/tlaloque/`** (DAG runtime):
- SwarmRunner emits per-node execution metrics → feeds monitoring
- Registry lookup gains a fallback: if worker not found, can trigger on-demand training

**New component** `internal/distillation/`:
- `SpecialistTrainer` implementation
- `TrainingJob` (run as subprocess or in-process)
- `ModelStore` (filesystem or model hub versioning)

## V0 → V1 Transition (Post-V0)

### V0 Deliverable
- Swarm of 5 tiny-models (intent, entity, amount, date, route) **manually trained once**
- Proves decomposition + tiny-models < SLM-only cost
- Validates end-to-end (swarmbench compare experiment)
- All workers are deterministic, versioned reference proxies

### V1 Deliverable (builds on V0)
- Training pipeline auto-generates new specialists on demand
- Monitoring detects when a task repeats → distills specialist
- Registry supports dynamic worker lifecycle
- Same 5 workers now updatable; new specialists can be added (e.g., if a new field type emerges)

### Why Split This Way

- **V0 is tight, falsifiable**: "tiny-models beat SLM" is proven by manual distillation + controlled experiment. No meta-learning complexity.
- **V1 adds the loop**: the system learns to generate its own specialists. This is where "systematize origami creation" fully materializes, but it's a separate validation gate.
- **Risk isolation**: if auto-training fails, V0 still works (fall back to manually trained models). If V1's monitoring is wrong, it won't hurt V0's baseline.

## Open Questions (for V1 planning)

1. **Synthetic data quality**: does SLM-generated data actually generalize to real data, or does it need regular real examples for calibration?
2. **Training latency**: how long does it take to distill a tiny-model on a laptop? (probably minutes, not hours, given dataset size)
3. **Model format**: ONNX? sklearn? Embedded transformers (e.g., sentence-transformers)? TinyBERT? Decision tree?
4. **Rollback semantics**: if a new specialist causes accuracy to drop, how fast do we revert? (automated, or manual gate?)
5. **Origami mapping**: how does a distilled model's behavior map to Origami's coherent-state / perceptual-channel abstraction? (separate concern, but V1 should answer it)

## References

- **Distillation paper**: Hinton et al., "Distilling the Knowledge in a Neural Network"
- **Existing evidence**: lfm2vl_proxy.go shows hand-calibrated distillation works; V1 automates the calibration
- **Phase 5 experiment**: compare-001.json validates the decomposed swarm beats single-shot; V1 keeps that win and adds adaptive specialist generation

---

**Next milestone**: After V0 is shipped, open this plan with user to lock down training backend + monitoring trigger strategy.
