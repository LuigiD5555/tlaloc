# Tlaloc Behavior Compilation — R0

## Thesis

A prompt is not the source specification. Prompts and future skills are compiled behavioral artifacts.

```text
Intent -> BehaviorSpec -> PromptIR -> compiled artifact -> Evaluation -> Tlaloque repair proposals -> Promotion
```

The target model stays frozen during prompt-level training. The evolving object is the compiled behavioral artifact.

## What R0 actually implements

R0 deterministically compiles a `BehaviorSpec` into `PromptIR` and a generic prompt artifact. The first bundled profile is `origami.quantum-inspired.r0`. The evaluation path, reference-state type, curriculum and default Tlaloque are currently specialized for that Origami profile.

The `target` value in R0 identifies the execution target but **does not yet select a Claude/GPT/Qwen/LFM-specific compiler backend**. Native model-family profiles and SkillIR are subsequent work.

## Tlaloque design

Tlaloque are deliberately small bounded specialists and receive structured findings. They do not own the source specification and they do not promote their own patches. Tlaloc owns the candidate artifact and the promotion gate remains centralized and test-gated.

Current Origami-profile Tlaloque:

- collapse guard
- branch guard
- coupling guard
- cancellation guard
- output-contract guard

Planned generic Tlaloque roles include intent keeper, prompt reducer, paraphrase/adversarial generator and target-family optimizer.

## Reference semantics

The current profile calculates deterministic expected states through `internal/reference`. This code is a reference semantics engine, not an agent. Evaluators compare target-model structured output against these expected states and produce findings for the Tlaloque.

## Fitness

R0 uses explicit pass/fail invariants plus aggregate score. Future fitness may include correctness, invariant preservation, repeatability, robustness, unseen-case generalization, prompt cost and latency.

## Curriculum

The bundled curriculum belongs to the Origami profile: deterministic state transitions, superposition, transform without collapse, interference, constraints, coupling, observation and temporal evolution. It is not the universal Tlaloc curriculum.
