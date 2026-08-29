# Tlaloc Behavior Compilation — R0

## Thesis

A prompt is not the source specification. Prompts, skills and receiver micro-programs are compiled/distilled behavioral artifacts.

Base path:

```text
Intent -> BehaviorSpec -> PromptIR -> compiled artifact -> Evaluation -> Tlaloque repair proposals -> Promotion
```

Origami Hybrid Receiver extension:

```text
Origami receiver contract
        ↓
rich swarm/Tlaloque execution
        ↓
successful behavioral trace
        ↓
receiver distillation
   ├── Master Prompt candidate
   ├── BOOT/Rosetta strategy
   └── deterministic MicroAgent IR
        ↓
receiver tournament
        ↓
Origami validation/promotion
```

The target model stays frozen during prompt-level training. The evolving object is the behavioral artifact and, for receiver work, the candidate bundle that attempts to reproduce the successful swarm behavior with much simpler local machinery.

## What R0 implements

The existing Behavior Lab deterministically compiles a `BehaviorSpec` into `PromptIR` and a generic prompt artifact. The first bundled profile is `origami.quantum-inspired.r0`; its evaluator and reference-state curriculum are specialized for that profile.

The Hybrid Receiver feature adds an experimental distillation layer under `internal/distill`:

- `SwarmStep` records externally relevant semantic transitions;
- `Distill` collapses repeated deterministic transitions into simple `MicroRule` candidates;
- conflicting transitions are rejected rather than hidden behind rule order/model judgment;
- candidates retain SHA-256 provenance to the source swarm trace;
- the tournament uses correctness, bootstrap, Rosetta, navigation, evidence and UNKNOWN metrics;
- contamination, false exactness or active-window violations are hard rejection gates.

This is a reference distiller, not a claim that arbitrary swarm cognition has already been losslessly compiled. The experimental question is how much useful complex behavior can be preserved by progressively simpler deterministic rules.

## Why micro-agents

The target is intentionally asymmetric:

```text
TRAIN / SEARCH TIME
rich model agents + swarm coordination + diagnosis

              ↓ distill

RECEIVE / RUN TIME
small deterministic local rules + carrier state + bounded model guidance
```

Each micro-agent should be locally simple. Complexity can arise from composition, propagation, inhibition, temporal trajectories and interaction with carrier state. The model should not need to recreate every mechanical step in natural language when the distilled machine can execute it deterministically.

## Prompt role in the Hybrid Receiver

The receiver Master Prompt should converge toward a universal bootstrap discipline rather than a giant hard-coded Origami manual. Carrier-specific symbol meanings belong to the carrier's own `BOOT/ROSETTA` structures.

Tlaloc can therefore optimize:

- wording that helps diverse models find BOOT;
- how to make models respect local Rosetta mappings;
- when to delegate mechanical work to Origami tools;
- how much semantic state to retain between bounded accesses;
- when to verify or return UNKNOWN.

It must not optimize by copying the hidden source, answer key or carrier-local Rosetta table into the external prompt.

## Tlaloque design

Tlaloque are deliberately bounded specialists and receive structured findings. They do not own Origami's source semantics and do not promote their own patches. Tlaloc owns candidate generation/evaluation; Origami owns semantic acceptance and canonical storage of promoted receiver artifacts.

Existing Origami-profile Tlaloque remain useful for state-semantic failures. Receiver-specific future Tlaloque can specialize in bootstrap discovery, Rosetta discipline, prompt reduction, navigation, UNKNOWN calibration, evidence verification and micro-rule simplification.

## Reference semantics

The current profile calculates deterministic expected states through `internal/reference`. For Hybrid Receiver campaigns, Origami's own receiver/reference fixtures remain the semantic reference authority. Tlaloc evaluates candidates against that reference evidence rather than reimplementing or redefining it.

## Fitness

Receiver fitness is multi-objective. Correctness alone is insufficient. Current R0 tournament fields include:

- bootstrap success;
- Rosetta success;
- navigation;
- correctness;
- evidence;
- UNKNOWN accuracy;
- peak active token-equivalent;
- tool cost.

Hard rejection applies to contamination, `FALSE_EXACT != 0`, or configured interface-window violations.

## Curriculum

The earlier bundled curriculum covers Origami coherent-state transitions. Hybrid Receiver curriculum adds:

1. BOOT discovery;
2. Rosetta resolution;
3. symbol permutation across equivalent carriers;
4. micro-program initialization;
5. selective address/navigation;
6. context recycling;
7. UNKNOWN on missing semantics;
8. verification on demand;
9. Native vs Computational vs Hybrid failure isolation;
10. cross-model replication.

The goal is not to prove one fixed prompt. It is to discover the smallest reliable receiver behavior that can bootstrap a self-describing Origami carrier and cooperate with deterministic micro-agents.
