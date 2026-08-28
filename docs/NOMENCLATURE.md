# Tlaloc nomenclature — current

The project uses a deliberately small set of conceptual names.

## TLALOC

**TLALOC — Transformative Latent Adaptive Logic Orchestration Core**

Tlaloc is the complete work system. It owns orchestration, behavior compilation, target-model execution, evaluation campaigns, coordination of bounded agents, regression and promotion control.

## Tlaloque

**Tlaloque** are Tlaloc's bounded specialist agents.

A Tlaloque may inspect a structured finding, orient a local decision, test a hypothesis, propose a repair or execute another explicitly bounded local task. A Tlaloque does **not** own the source specification and cannot promote its own changes.

The implementation package is `internal/tlaloque`. The distributed mechanism may still be described generically as a swarm, but **Swarm is not the component name**.

Current Behavior Lab Tlaloque:

- `collapse_guard`
- `branch_guard`
- `coupling_guard`
- `cancellation_guard`
- `output_guard`

## Origami

Origami is the independent representation/state-machine language. It defines how supported classes of states are represented and transformed. Tlaloc may use Origami, another representation provider, or no Origami representation at all.

## Reference semantics

The deterministic mechanism used to calculate expected states is called the **reference semantics engine** or simply **reference** in code. It is not an agent and is not a named mythological component.

## Retired current-architecture terms

The following names are not used for current components:

- `Oracle`
- `Adivino`
- `Báculo`

Historical documents may contain older vocabulary when recording what a previous version actually called a component. Those historical names must not be interpreted as current architecture.

## Short rule

> **Tlaloc coordinates. Tlaloque perform bounded work. Origami represents. Reference semantics verify expected state behavior.**
