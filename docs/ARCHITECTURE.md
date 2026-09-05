# Tlaloc Architecture R2

Tlaloc is Tonal's capability foundry, Behavior Lab and reusable-machinery lifecycle subsystem.

## Canonical lifecycle

```text
Intent / recurring need
  -> BehaviorSpec + invariants
  -> bounded candidate machinery
  -> experiment / execution / observations
  -> deterministic evaluation where possible
  -> competence envelope + failure analysis
  -> candidate representation
  -> holdout / ablation / regression checks
  -> promote or reject
  -> publish to Tonal Capability Registry
```

The reference swarm remains a laboratory technique for discovering procedures. It is not the definition of a Tlaloque and is not Tonal's production runtime.

## Tlaloque

A Tlaloque is deliberately bounded reusable machinery produced or qualified through Tlaloc.

It may perform one operation, inspect one piece of state, test one predicate, compare one item, open one evidence source, execute a bounded specialist or propose one repair. Complex behavior comes from composition, state, verification and reuse rather than treating each Tlaloque as a general autonomous intelligence.

A Tlaloque may be deterministic, algorithmic, symbolic, tool-backed, specialized-model-backed or hybrid. The bounded mechanism—not the mere presence of a model—is what Tlaloc qualifies.

## Parrot boundary

Parrot is not a Tlaloque. It is Tonal's singular external probabilistic cognition interface.

Tlaloc may characterize Parrot-assisted behavior and analyze verified Episodes containing Parrot calls, but it does not own, manufacture or promote the external model.

A central R2 research loop is:

```text
external cognition resolves novelty
        ↓
verified Episode
        ↓
repeated bounded pattern?
        ↓
Tlaloc candidate machinery
        ↓
qualification
        ↓
future execution needs less external cognition
```

This loop is a research target, not yet an autonomous promotion claim.

## Distillation target

The target is:

```text
Behavior(candidate artifact) ~= required verified behavior
```

not textual reproduction of a trace.

Architecture R2 broadens the possible output beyond prompt-first deployment. The smallest reliable reusable representation may be:

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

Existing prompt-first work remains valid as one deployment strategy and historical experiment line; it is not the universal architectural endpoint.

## Authority hierarchy

1. `BehaviorSpec` + declared invariants define desired behavior.
2. Experiments demonstrate a candidate procedure.
3. Verification and evidence establish its competence envelope.
4. Distillation/compilation proposes a reusable representation.
5. Holdout, ablation, regression and cost evidence decide promotion.
6. Tonal decides at runtime whether a published capability should be selected.

Tlaloc does not self-promote individual workers into Tonal execution.

## Tonal boundary

Tonal owns goal intake, workflow/DAG authority, scheduling, Blackboard state, runtime selection, execution coordination, verification coordination, accounting and final workflow results.

Tlaloc publishes qualified machinery and evidence into a contract Tonal can consume. Tonal's generic Registry must also be able to contain non-Tlaloc components such as Parrot (`EXTERNAL_MODEL`) and ordinary tools/Machines.

## Origami and Shponglese

Tlaloc is not defined by Origami.

- Shponglese owns semantic operational IR.
- Origami owns representation/carrier/addressing/selective-unfolding mechanisms.
- Tlaloc may develop/test machinery that interacts with either without owning their semantics.

## Existing implementation

Existing receiver distillation, PromptIR/compiler, evaluators, learning memory, adaptive search, Tlaloque implementations and target adapters remain available for the claims they actually establish.

Frozen T1/R1 code may retain historical `Parrot Tlaloque` terminology because the frozen public contract used it. Current R2 architecture and new interfaces use the corrected external-model boundary.

See also `CLAUDE.md`, `docs/ROLE_IN_TONAL.md`, `docs/NOMENCLATURE.md` and `docs/research/`.
