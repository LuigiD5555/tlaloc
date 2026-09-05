# Tlaloc nomenclature — Architecture R2

The Tonal ecosystem uses a deliberately small set of current conceptual names.

## TONAL

Tonal is the complete heterogeneous runtime and research system.

Tonal owns goal intake, workflow/DAG execution, capability selection, routing, scheduling, Blackboard state, resource accounting, verification coordination, tracing and final workflow results.

The historical composition/pinning/provenance layer remains Tonal infrastructure, but it no longer defines Tonal as a whole.

## TLALOC

Tlaloc is Tonal's capability foundry, Behavior Lab and capability-lifecycle subsystem.

Its canonical job is:

```text
behavioral intent / observed need
 -> define bounded behavior + invariants
 -> discover or construct candidate capability
 -> run reference experiments
 -> verify and characterize competence
 -> compare/ablate
 -> package candidate
 -> promote or reject with evidence
```

Tlaloc may study repeated verified Episodes and propose reusable structure. It does not own Tonal's runtime or Blackboard.

## TLALOQUE

A Tlaloque is one bounded, typed and measurable capability.

A Tlaloque may be deterministic, symbolic, tool-backed, neural, model-backed or hybrid. It performs an explicitly bounded operation under a declared contract and competence profile.

A Tlaloque does not own the BehaviorSpec and cannot promote itself.

## PARROT

Parrot is one probabilistic Tlaloque.

It may be useful for ambiguous perception, extraction, interpretation or generation. It has no system-level authority and is not Tonal's brain, router, verifier, memory or source of truth.

Parrot should be selected only when its capability is appropriate under the same routing/evidence rules as other executors.

## CAPABILITY ARTIFACT

Tlaloc no longer assumes that every successful behavior should ultimately become a portable prompt.

The preferred artifact is the smallest reliable reusable representation justified by evidence. Depending on the behavior, it may be:

```text
deterministic operation
Machine/state machine
tool wrapper
Shponglese motif/program
prompt/policy artifact
small specialized model
probabilistic Tlaloque
hybrid capability
```

Prompt-only portability remains useful when it is genuinely the best representation.

## BEHAVIORSPEC

`BehaviorSpec + declared invariants` describe the behavior being developed and remain behavioral authority during experiments and distillation.

Implementations, prompts, tools and models are derived operational artifacts.

## REFERENCE BEHAVIOR

A successful bounded execution plus its evidence record is reference behavior for an experiment. It demonstrates that a procedure can satisfy the intent under measured conditions; it does not automatically prove that another representation or distilled candidate can.

## EPISODE

An Episode is a normalized evidence record of an execution trajectory, including selected capabilities, operations, outcomes, cost and failures where available.

Episodes may support later pattern discovery and Cognitive JIT research, but model reflections or inferred stories inside an Episode are not semantic truth merely because they were recorded.

## SHPONGLESE

Shponglese is semantic operational IR for primitive and composed behavior.

It begins as an IR, not as a claim of emergent language. Its meaning should remain invariant across physical codecs. Future language-like claims require compositionality and held-out evidence.

## ORIGAMI

Origami is an independent representation, transport, addressing and virtual-memory substrate.

Origami may carry Shponglese or other semantic structures, but Origami owns its own representation semantics and evidence gates, not Shponglese meaning or Tonal runtime authority.

## REFERENCE SEMANTICS

A deterministic mechanism used to calculate expected behavior is a reference semantics engine or simply a reference in code. It is not an agent and is not a mythological component.

## RETIRED CURRENT-ARCHITECTURE TERMS

The following names are not used for current components:

- `Oracle`
- `Adivino`
- `Báculo`
- `Swarm Trainer`

Historical documents may preserve old vocabulary when recording previous architecture.

## Short rule

> Tonal runs and coordinates the system. Tlaloc develops and qualifies capabilities. Tlaloques perform bounded work. Parrot is one probabilistic Tlaloque. Shponglese expresses operational semantics. Origami may carry or address those semantics without owning them.
