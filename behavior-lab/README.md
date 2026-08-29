# Tlaloc Behavior Lab

Behavior Lab is Tlaloc's R0 experiment for compiling formal behavior contracts into model-facing artifacts, testing them against target models and using bounded **Tlaloque** to propose repairs from structured failures.

The original compiler path is:

```text
BehaviorSpec -> PromptIR -> prompt -> target model -> findings -> Tlaloque repairs
```

The experimental Origami Hybrid Receiver path adds a second stage:

```text
Origami receiver contract
  -> rich swarm/Tlaloque receiver behavior
  -> successful semantic trace
  -> internal/distill
  -> small receiver candidate
       prompt + BOOT/Rosetta strategy + MicroAgent rules
  -> fitness tournament
  -> Origami validation/promotion
```

The point of distillation is not to preserve private chain-of-thought. It preserves externally relevant state transitions, actions and effects so a complex swarm can be replaced at receive time by simpler deterministic local behavior where evidence shows that is safe.

## Terms

- `internal/reference` calculates deterministic expected states for the original coherent-state profile.
- `internal/tlaloque` contains bounded specialist agents that diagnose findings and propose prompt patches.
- `internal/distill` contains the experimental swarm-trace -> deterministic receiver-candidate path and receiver tournament.
- `profiles/origami/hybrid-receiver-r0.json` defines the experimental receiver search objective and hard safety gates.
- Origami remains authoritative for its receiver semantics and stores any promoted receiver artifact.
- Tlaloc retains candidate-search/evaluation authority; Tlaloque cannot promote their own proposals.

## Current receiver hard gates

A candidate is ineligible when any of these hold:

- contaminated trial;
- false exactness;
- active interface budget violation;
- conflicting/non-deterministic distilled transition.

Correctness alone cannot override these gates.

## Commands

Existing Behavior Lab commands remain:

```bash
go run ./cmd/behaviorlab compile
go run ./cmd/behaviorlab tlaloque
go run ./cmd/behaviorlab train -model <model-name>
```

A dedicated receiver-tournament CLI is not yet promoted. The R0 receiver distiller/tournament currently exists as a Go package and testable contract before command-surface expansion.
