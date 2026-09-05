# Tlaloc 6.0.0-alpha.21

**TLALOC — Transformative Latent Adaptive Logic Orchestration Core**  
**Architecture R2 role: Tonal capability foundry + Behavior Lab**

Tlaloc develops, tests, qualifies, packages and studies bounded reusable capabilities for Tonal.

Its purpose is not to become the runtime that solves every task. Tonal owns runtime workflow execution, routing, scheduling, Blackboard state and final system results. Tlaloc owns capability development and evidence.

## Core loop

```text
behavioral intent / recurring need
        ↓
BehaviorSpec + invariants
        ↓
bounded candidate capability
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

A Tlaloque is one bounded, typed, measurable capability. It may be deterministic, symbolic, tool-backed, neural, model-backed or hybrid.

## Parrot

Parrot is one probabilistic Tlaloque.

It may be useful for ambiguous perception, extraction, interpretation or generation, but it has no system-level authority. It is not Tonal's router, verifier, memory, source of truth or default executor for every task.

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
probabilistic Tlaloque
hybrid capability
```

`BehaviorSpec + invariants` remain behavioral authority. Implementations, prompts, models and tools are operational artifacts.

## Episodes and Cognitive JIT

Tonal execution traces may be normalized into Episodes. Future Tlaloc research may use verified Episode corpora to discover recurring structure and propose cheaper reusable capabilities.

```text
Tonal execution
      ↓
verified Episode
      ↓
recurring structure?
      ↓
candidate capability
      ↓
test + holdout + ablation + verification
      ↓
promote or reject
```

Recurrence is not promotion. Model reflection is not truth. Evidence gates remain authoritative.

See `docs/COGNITIVE_JIT.md` and `docs/research/`.

## Origami and Shponglese

- **Shponglese** is semantic operational IR for primitive and composed behavior.
- **Origami** is an independently testable representation, carrier, addressing and virtual-memory substrate.

Tlaloc may experimentally develop or qualify capabilities that interact with either, but it does not own their representation semantics.

## Behavior Lab

Existing Behavior Lab machinery, learning memory, adaptive search, grounding/verifier work, real-model campaigns and closed-loop experiments remain valuable for the claims they actually establish.

Architecture R2 changes their place in the ecosystem: they are capability-development and evidence machinery, not proof that Tlaloc is the complete Tonal runtime.

## Research direction

Near-term R2 work focuses on:

1. stabilizing capability and Episode contracts;
2. supporting Tonal's generic capability-selection runtime;
3. Primitive Swarm / MICRO-ISA experiments;
4. later experience-to-structure / Cognitive JIT experiments.

See:

- `CLAUDE.md`
- `docs/CURRENT_STATE.md`
- `docs/ROLE_IN_TONAL.md`
- `docs/NOMENCLATURE.md`
- `docs/research/`

## Verification

For Behavior Lab changes, preserve the established promotion checks:

```bash
cd behavior-lab
go test ./...
go vet ./...
go test -race ./...
```

## Documentation authority

Current architecture documents override historical material. Older detailed alpha-era descriptions remain available through Git history and should progressively move under `docs/archive/` instead of occupying the repository entry point.

> Tlaloc develops and qualifies capabilities. Tonal decides when and how to use them.
