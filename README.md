# Tlaloc 6.0.0-alpha.21

**TLALOC — Transformative Latent Adaptive Logic Orchestration Core**  
**Architecture R2 role: Tonal capability foundry + Behavior Lab**

Tlaloc develops, tests, qualifies, packages and studies bounded reusable machinery for Tonal.

Its purpose is not to become the runtime that solves every task. Tonal owns runtime workflow execution, routing, scheduling, Blackboard state and final system results. Tlaloc owns machinery development and evidence.

## Core loop

```text
behavioral intent / recurring need
        ↓
BehaviorSpec + invariants
        ↓
bounded candidate machinery
        ↓
Behavior Lab experiment
        ↓
verification + competence envelope
        ↓
compare / ablate / holdout
        ↓
promote or reject
        ↓
Tonal Capability Registry
```

A **Tlaloque** is bounded, typed, measurable reusable machinery produced or qualified through Tlaloc. It may be deterministic, symbolic, tool-backed, specialized-model-backed or hybrid.

## Parrot is not a Tlaloque

Parrot is Tonal's singular external probabilistic cognition interface. It is not manufactured, owned or promoted by Tlaloc.

Tlaloc may measure Parrot-assisted behavior and analyze Episodes that contain Parrot calls. When recurring successful behavior can be made bounded and reusable, Tlaloc may propose a Tlaloque or Machine that replaces that repeated external cognition.

```text
Parrot-assisted success
        ↓
verified Episodes
        ↓
recurring bounded structure?
        ↓
candidate Tlaloque / Machine
        ↓
holdout + ablation + verification
        ↓
promote or reject
```

## Distillation is broader than prompts

Earlier Tlaloc work emphasized prompt-first portability. Architecture R2 retains prompt artifacts as one useful target but no longer treats them as the universal preferred endpoint.

The preferred output is the **smallest reliable reusable representation justified by evidence**. Depending on the behavior, that may be:

```text
deterministic operation
Machine / state machine
tool wrapper
Shponglese motif/program
prompt / policy
small specialized model
Tlaloque
hybrid machinery
```

`BehaviorSpec + invariants` remain behavioral authority. Implementations, prompts, models and tools are operational artifacts.

## Episodes and Cognitive JIT

Tonal execution traces may be normalized into Episodes. Future Tlaloc research may use verified Episode corpora to discover recurring structure and propose cheaper reusable machinery. Recurrence is not promotion; model reflection is not truth; evidence gates remain authoritative.

See `docs/COGNITIVE_JIT.md` and `docs/research/`.

## Origami and Shponglese

- **Shponglese** is semantic operational IR for primitive and composed behavior.
- **Origami** is an independently testable representation, carrier, addressing and virtual-memory substrate.

Tlaloc may experimentally develop or qualify machinery that interacts with either, but it does not own their representation semantics.

## Behavior Lab

Existing Behavior Lab machinery, learning memory, adaptive search, grounding/verifier work, real-model campaigns and closed-loop experiments remain valuable for the claims they actually establish.

Architecture R2 changes their place in the ecosystem: they are machinery-development and evidence systems, not proof that Tlaloc is the complete Tonal runtime.

## T1 compatibility note

Frozen T1/R1 code historically published Parrot through `tlaloquekit` as a generative Tlaloque. Preserve that terminology where needed to reproduce the frozen contract. New R2 APIs and current architecture classify Parrot as external cognition instead.

## Research direction

Near-term R2 work focuses on stabilizing capability/Episode contracts, supporting Tonal's generic registry, Primitive Swarm/MICRO-ISA experiments and later experience-to-structure/Cognitive JIT experiments.

## Verification

For Behavior Lab changes:

```bash
cd behavior-lab
go test ./...
go vet ./...
go test -race ./...
```

## Documentation authority

Start with `CLAUDE.md`, this README, `docs/CURRENT_STATE.md`, `docs/ARCHITECTURE.md`, `docs/ROLE_IN_TONAL.md`, then the active experiment specification. Historical files under `docs/archive/` are not current authority.
