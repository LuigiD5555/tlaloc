---
name: tlaloc-behavior
description: This skill should be used when changing BehaviorSpec, PromptIR, prompt compilation, target execution, evaluation, behavior training loops, or generated behavioral artifacts in Tlaloc Behavior Lab.
version: 0.1.0
---

# Tlaloc behavior compilation workflow

Preserve the separation between specification and generated artifacts.

## Source-of-truth chain

1. Read the `BehaviorSpec` and declared invariants.
2. Identify any representation-provider contract required by the profile.
3. Compile deterministically into `PromptIR` and target artifacts.
4. Execute the frozen target model.
5. Compare structured results with reference semantics/evaluator expectations.
6. Pass findings to bounded Tlaloque for repair proposals.
7. Run regression and centralized promotion gates.

Never edit a generated prompt as the authoritative fix when the source behavior can be expressed in `BehaviorSpec`, compiler logic, or a target profile.

## Current implementation boundary

Do not claim native Anthropic API support or model-family-specific prompt optimization unless the corresponding adapter/backend exists. Project-local Claude Code skills are onboarding/workflow aids; they are not generated SkillIR outputs.

## Verification

Run `go test ./...`, `go vet ./...`, `go test -race ./...`, deterministic artifact rebuild, and the smallest relevant behavior curriculum before promotion.
