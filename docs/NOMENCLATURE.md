# Tlaloc nomenclature — current

The project uses a deliberately small set of conceptual names.

## TLALOC

**TLALOC — Transformative Latent Adaptive Logic Orchestration Core**

Tlaloc is a **development kit for behavioral discovery and distillation**.

Its canonical job is:

```text
intent
 -> decompose into bounded actions
 -> run a Tlaloque swarm as a reference experiment
 -> verify the requested behavior
 -> distill the demonstrated behavior
 -> prefer a portable prompt artifact
 -> test that prompt on clean target models
```

Tlaloc can help develop Origami, a calculator behavior, a document workflow, a classifier or another target. Origami is not the definition of Tlaloc.

## Tlaloque

**Tlaloque** are Tlaloc's bounded specialist workers.

They are deliberately small. A Tlaloque may inspect one item, evaluate one condition, compare one pair, follow one relation, open one evidence source, mark UNKNOWN or perform another explicitly bounded local operation.

A distributed set of Tlaloque may be described generically as a **swarm**.

The swarm is a reference/playground mechanism for discovering a working behavior. It is not the preferred final deployment when the behavior can be distilled into a portable prompt.

A Tlaloque does not own the BehaviorSpec and cannot promote its own result.

## Prompt artifact

The default portable output of Tlaloc is a prompt or PromptIR-derived prompt that reproduces the required behavior without requiring Tlaloc, a sandbox or undeclared tools.

This deployment baseline is called:

```text
L0_PROMPT_ONLY
```

Richer levels remain explicit fallbacks:

```text
L1 prompt + declarative context/IR
L2 prompt + tools
L3 prompt + runtime
L4 specialized target/model system
```

## BehaviorSpec

BehaviorSpec + declared invariants describe the behavior being developed. They remain the desired-behavior authority during swarm experimentation and prompt distillation.

## Reference behavior

A successful swarm execution plus its test/evidence record is the **reference behavior** or **reference procedure** for a distillation experiment.

It demonstrates that some procedure can satisfy the intent. It does not prove that a candidate prompt can.

## Origami

Origami is an independent representation/state-machine/visual language and one possible Tlaloc development target.

Tlaloc may experiment with Origami prompts, representation strategies and perceptual channels, but Origami owns the decision about what becomes a canonical Origami version/profile.

## Tonal

Tonal is an optional multi-tool composition/reproducibility layer. It may combine Tlaloc, Blueprint Framework and future development tools with exact target revisions.

Tonal is not a required stage in Tlaloc behavior distillation and does not own target-project releases.

## Reference semantics

A deterministic mechanism used to calculate expected state behavior is called a **reference semantics engine** or simply **reference** in code. It is not an agent and is not a named mythological component.

## Retired current-architecture terms

The following names are not used for current components:

- `Oracle`
- `Adivino`
- `Báculo`

Historical documents may contain older vocabulary when recording what a previous version actually called a component.

## Short rule

> **Tlaloc discovers and distills behavior. Tlaloque perform bounded experimental steps. Prompts are the portable default. Origami is one possible target. Tonal optionally composes development tools.**
