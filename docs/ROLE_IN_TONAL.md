# Tlaloc role in Tonal — Architecture R2

## Purpose

Tlaloc is the subsystem that develops and qualifies reusable machinery for Tonal.

It is not Tonal's runtime and does not own workflow scheduling, Blackboard state, final workflow authority, system-level routing or the external general-purpose model interface.

## Tlaloc owns

- Behavior Lab experiments;
- bounded Tlaloque construction;
- capability contracts;
- executor qualification;
- CapabilityProfile evidence;
- competence-envelope characterization;
- candidate comparison and ablation;
- reusable machinery packaging/versioning;
- promotion/deprecation of Tlaloc-managed Tlaloques/Machines;
- Episode analysis for future experience-to-structure compilation.

## Tlaloc does not own

- Tonal goal intake;
- Tonal workflow/DAG authority;
- Tonal scheduler;
- Tonal Blackboard;
- Tonal final answer;
- Origami semantics or visual profiles;
- Shponglese physical codec;
- Parrot or the external model provider behind it.

## Tlaloque boundary

A Tlaloque is reusable machinery whose bounded behavior is produced or qualified through Tlaloc. A Tlaloque may internally use deterministic code, algorithms, tools or a specialized model; that does not make every external model a Tlaloque.

Parrot is a separate Tonal component kind: `EXTERNAL_MODEL`.

## Output principle

Tlaloc should search for the smallest reliable reusable representation that preserves the required behavior and evidence threshold.

A successful behavior may become:

```text
deterministic operation
Machine/state machine
tool wrapper
Shponglese motif/program
prompt/policy
small specialized model
Tlaloque
hybrid machinery
```

Prompt-only portability remains useful when it is actually the best representation, but it is no longer the universal endpoint.

## Evidence before promotion

A candidate must not be promoted merely because it is repeated, compressed or proposed by a stronger model. Promotion should consider declared behavior/invariants, held-out reliability, root-cause failures, deterministic verification, competence envelope, cost/latency, complexity, reuse, ablation evidence and provenance.

## Interaction with Episodes

```text
Tonal execution
      ↓
verified Episode
      ↓
Tlaloc analysis
      ↓
recurring structure?
      ↓
candidate machinery
      ↓
test + verify + ablate + holdout
      ↓
promote or reject
```

Episodes are evidence records. They do not grant automatic authority to reflections, model explanations or inferred causal stories.

## Interaction with Parrot

Parrot is Tonal's external probabilistic cognition interface, not a Tlaloc artifact.

Tlaloc may:

- measure Parrot-assisted behavior;
- characterize where it succeeds/fails;
- analyze Episodes containing Parrot calls;
- propose bounded reusable machinery that replaces recurring Parrot work.

Tlaloc may not claim ownership of, promote, or silently redefine the external model itself.
