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

The lifecycle is general, but the bundled evaluator, reference engine, Tlaloque guards and curriculum are still specialized for the first consumer profile: `origami.quantum-inspired.r0`. Model-family-specific compilation (Claude/GPT/Qwen/LFM profiles and generated skills) is not implemented in this release.

See `NOMENCLATURE.md` for naming rules.

## Operational agent guidance

`CLAUDE.md` and `.claude/skills/` are checked-in instructions for coding agents working on this repository. They summarize architecture and workflows but do not define model behavior. The behavioral source of truth remains `BehaviorSpec + invariants`, and future compiler-generated SkillIR remains a separate, not-yet-implemented layer.

The installer preserves these files inside the managed Tlaloc version but does not install or modify global `~/.claude` configuration.

## Origami perceptual-contract awareness

Tlaloc can track Origami contracts independently of implementing them. Origami `6.0.0-alpha.3` preserves `origami.perceptual-channels.r0` (introduced in alpha.2) and clarifies that OHF is a nested Origami research track; Tlaloc records that contract and hierarchy as upstream-known while the executable behavior profile remains `origami.quantum-inspired.r0`. This prevents semantic drift without pretending that moire, stereoscopic/depth or temporal-latent-image operations already have reference evaluators.
