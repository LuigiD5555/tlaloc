# Tlaloc current architecture

**TLALOC — Transformative Latent Adaptive Logic Orchestration Core** is the work system.

```text
Intent
  -> BehaviorSpec / invariants
  -> PromptIR / future SkillIR
  -> target-family compiler
  -> model execution
  -> evaluator / reference-semantics comparison
  -> bounded Tlaloque diagnosis + repair proposals
  -> regression + promotion decision
```

## Ownership boundary

Tlaloc owns orchestration, behavior compilation, model-facing adapters, Tlaloque coordination/training, evaluation campaigns, verification and promotion control.

Origami is an independent representation project. It can supply state schemas, semantics and a reference semantics engine to Tlaloc, but Tlaloc does not own or redefine Origami semantics. Tlaloc can also operate with representations other than Origami.

## Authority hierarchy

1. `BehaviorSpec` + declared invariants are the source of truth for desired behavior.
2. A representation provider (for example Origami) is authoritative for its own semantic contract.
3. Reference semantics calculate expected state behavior; Tlaloc evaluators compare model behavior against those expectations.
4. Prompts, skills and target-specific instruction packages are compiled artifacts, never the source of truth.
5. Tlaloque may propose bounded repairs; promotion authority remains centralized and test-gated in Tlaloc.

## Tlaloque

Tlaloque are deliberately small specialist agents under Tlaloc. They are not a second source of truth and they do not self-promote changes. The current R0 implementation uses rule-based Tlaloque guards; later versions may add small models where a deterministic worker is insufficient.

## Current implementation boundary (R0)

The lifecycle now selects an explicit registered profile. `origami.quantum-inspired.r0` retains its coherent-state evaluator; `origami.relational-core.r0` consumes versioned upstream fixtures with its own strict comparator, curriculum, and bounded Tlaloque. Unknown or incompatible profiles are `UNSUPPORTED`, with no fallback. Model-family-specific compilation remains unimplemented.

See `NOMENCLATURE.md` for naming rules.

## Operational agent guidance

`CLAUDE.md` and `.claude/skills/` are checked-in instructions for coding agents working on this repository. They summarize architecture and workflows but do not define model behavior. The behavioral source of truth remains `BehaviorSpec + invariants`, and future compiler-generated SkillIR remains a separate, not-yet-implemented layer.

The installer preserves these files inside the managed Tlaloc version but does not install or modify global `~/.claude` configuration.

## Origami perceptual-contract awareness

Tlaloc tracks Origami contracts independently of implementing them. Origami `6.0.0-alpha.4` publishes executable relational fixtures consumed here without copying its Reference Machine. The perceptual contract remains upstream-known but unsupported at runtime, preventing semantic drift without pretending that moire, depth, or temporal-latent-image operations have evaluators.
