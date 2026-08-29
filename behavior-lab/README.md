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
  -> deterministic receiver proposal
       UniversalPrompt
       + BOOT strategy
       + Rosetta constraints
       + MicroAgent rules
  -> fitness tournament
  -> Origami import / validation / storage
```

The point of distillation is not to preserve private chain-of-thought. It preserves externally relevant state transitions, **actions**, effects and provenance so a complex swarm can be replaced at receive time by simpler deterministic local behavior where evidence shows that is safe.

## Terms

- `internal/reference` calculates deterministic expected states for the original coherent-state profile.
- `internal/tlaloque` contains bounded specialist agents that diagnose findings and propose prompt patches.
- `internal/distill` contains the experimental swarm-trace -> deterministic receiver-candidate path and receiver tournament.
- `profiles/origami/hybrid-receiver-r0.json` defines the experimental receiver search objective and hard safety gates.
- `tlaloc.origami-hybrid-artifact-set.r0` is the importable proposal contract consumed by Origami.
- Origami remains authoritative for its receiver semantics, carrier-local physical bindings and promoted receiver storage.
- Tlaloc retains candidate-search/evaluation authority; Tlaloque cannot promote their own proposals.

## Two targets from one discovered behavior

Tlaloc deliberately separates:

```text
UniversalPrompt
  -> external receiver bootstrap

BOOT strategy + Rosetta constraints + MicroProgram
  -> carrier/runtime behavior proposal for Origami
```

The universal prompt should stay small and carrier-agnostic. Concrete glyph meanings belong inside each Origami carrier's own ROSETTA, not in Tlaloc's prompt.

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

Hybrid Receiver R0 adds:

```bash
go run ./cmd/behaviorlab receiver-distill \
  -trace testdata/receiver/swarm-trace-r0.json \
  -prompt testdata/receiver/universal-bootstrap-r0.md \
  -window 4000 \
  -out generated/origami-hybrid-receiver-r0.candidate.json \
  -hybrid-out generated/origami-hybrid-receiver-r0.artifact-set.json

go run ./cmd/behaviorlab receiver-rank \
  -in testdata/receiver/scored-candidates-r0.json \
  -window 4000 \
  -out generated/origami-hybrid-receiver-r0.ranking.json
```

`receiver-distill` now writes both the internal distilled candidate and the explicit Hybrid artifact set that Origami can import. `receiver-rank` applies the multi-objective fitness function and hard safety gates.

These commands produce Tlaloc **candidates/evidence**; they do not write directly into Origami's promoted receiver registry.

See `HYBRID_RECEIVER_PIPELINE_R0.md` for the complete swarm -> distillation -> Origami import loop.
