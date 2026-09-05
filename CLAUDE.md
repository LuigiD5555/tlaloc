# Tlaloc project instructions — Architecture R2

## Role

Tlaloc is Tonal's capability foundry, Behavior Lab and capability-lifecycle subsystem.

Tlaloc discovers, constructs, tests, qualifies, packages, promotes, deprecates and studies bounded reusable capabilities. It does not own Tonal's runtime, scheduler or Blackboard.

## Shared North Star

The Tonal ecosystem investigates whether complex, reliable behavior can emerge from the composition of small, bounded, verifiable and reusable capabilities instead of requiring one increasingly complex general-purpose model.

Tlaloc's contribution is to find the smallest reliable reusable structure that removes recurring uncertainty or repeated work.

## Canonical terms

- **Tonal**: complete heterogeneous runtime and research system.
- **Tlaloc**: capability-development, experimentation and qualification subsystem.
- **Tlaloque**: one bounded, typed, measurable capability. It may be deterministic or probabilistic.
- **Parrot**: one probabilistic Tlaloque. It has no system-level authority.
- **Shponglese**: semantic operational IR for primitive and composed behavior.
- **Origami**: independent representation, transport and virtual-memory substrate.

## Capability lifecycle

Tlaloc should not assume that successful behavior must ultimately become a prompt.

The preferred output is the smallest reliable reusable representation justified by evidence. Depending on the behavior, that may be:

- a deterministic operation;
- a Machine or state machine;
- a tool wrapper;
- a Shponglese motif/program;
- a prompt/policy artifact;
- a small specialized model;
- a probabilistic Tlaloque;
- a hybrid capability.

Use `BehaviorSpec + invariants` as the behavioral source of truth. Treat prompts, models, tools and skills as derived/operational artifacts rather than behavioral authority.

## Invariants

When changing Tlaloc:

- keep Tlaloques bounded and unable to self-promote;
- keep representation ownership outside Tlaloc;
- keep Tonal runtime authority outside Tlaloc;
- keep current capability claims aligned with code and evidence;
- preserve deterministic authority where deterministic verification is possible;
- preserve installer/uninstaller safety and retrocompatibility;
- prefer explicit CapabilityProfile evidence over model prestige or size;
- never give Parrot privileged authority because it is a language model;
- preserve experiment freeze boundaries.

Run `go test ./...`, `go vet ./...`, and `go test -race ./...` in `behavior-lab` before promotion.

## Episodes and Cognitive JIT direction

Tonal execution traces may be adapted into Episodes. Episodes are evidence records, not automatic training truth.

Future Tlaloc work may use verified Episode corpora to discover recurring trajectories, candidate abstractions, smaller specialists or deterministic Machines. Candidate recurrence alone is not enough for promotion: holdout reliability, failure analysis, verification and cost/complexity tradeoffs are required.

## Document authority

Read in this order:

1. `CLAUDE.md`
2. `README.md`
3. `docs/CURRENT_STATE.md`
4. `docs/ARCHITECTURE.md`
5. `docs/ROLE_IN_TONAL.md`
6. active Behavior Lab / experiment specification

Historical material belongs under `docs/archive/` or existing history areas and is not current authority.

If archived documentation conflicts with current Architecture R2 documentation, current documentation wins. Frozen experiment specifications remain authoritative for their experiments.

See `.claude/skills/` for task-specific workflows.
