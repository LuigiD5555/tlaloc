# Tlaloc Behavior Compilation — Prompt-First R0

## Thesis

A prompt is not the source specification. `BehaviorSpec + invariants` describe the requested behavior. The swarm is a development-time reference procedure used to discover and verify how that behavior can be achieved. The preferred deployable result is a prompt that reproduces the demonstrated behavior with fewer requirements.

```text
Intent
  -> BehaviorSpec / invariants
  -> bounded Tlaloque decomposition
  -> swarm execution
  -> successful reference behavior
  -> behavioral distillation
  -> PromptIR / prompt candidates
  -> clean target-model evaluation
  -> behavioral-fidelity comparison
  -> prompt artifact
```

The target model may remain completely frozen.

## Why the swarm exists

A difficult intention is easier to debug when decomposed into many simple operations. Tlaloc can observe which steps, conditions, ordering rules, branches, loops and verification checks actually caused success.

The swarm therefore serves as an executable reference implementation of behavior.

It is **not** assumed to be the cheapest or most portable way to deploy that behavior.

## Distillation target

The goal is:

```text
Behavior(candidate) ~= Behavior(reference swarm)
```

not:

```text
candidate text ~= swarm transcript
```

Tlaloc should remove agent identities, incidental wording, redundant steps and hidden development scaffolding when those details are not necessary to reproduce behavior.

## Prompt-first deployment hierarchy

```text
L0  PROMPT_ONLY
L1  PROMPT_PLUS_DECLARATIVE_CONTEXT_OR_IR
L2  PROMPT_PLUS_TOOLS
L3  PROMPT_PLUS_RUNTIME
L4  SPECIALIZED_MODEL_OR_TARGET_SPECIFIC_SYSTEM
```

Tlaloc selects the least demanding level that reaches the BehaviorSpec's required fidelity.

A target that has only a text interface must still be able to consume an L0 artifact. L0 therefore forbids hidden dependencies on a sandbox, tools, Go/Python execution, Tlaloc runtime state or private swarm traces.

## R0 implementation

The repository contains three complementary implementation families:

1. generic `BehaviorSpec -> PromptIR -> prompt` compilation;
2. swarm/reference trace distillation under `internal/distill`;
3. `promptfirst.go`, which deterministically selects the lowest deployment class that satisfies behavioral evidence.

Prompt-first selection currently considers behavioral fidelity, pass rate, regression rate and clean-target trial count. Within the same valid deployment class it prefers the smaller prompt before adding unnecessary dependencies.

## Clean target

A distilled prompt must be evaluated on a clean target environment. For L0 this means the model receives the prompt and task inputs allowed by the BehaviorSpec, but not:

- hidden swarm state;
- the development sandbox;
- evaluator answers;
- undeclared tools;
- Tlaloc internal memory.

Success of the reference swarm and success of the prompt are two different claims and require separate evidence.

## Generality

The behavior compiler is target-neutral.

Examples:

```text
Origami receiver/writer behavior
calculator behavior
document-analysis behavior
classification behavior
workflow behavior
other software/tool behavior
```

Tlaloc can use target-specific evaluators during development without turning those target dependencies into the definition of Tlaloc itself.

## Origami extension

Existing Origami receiver work remains an important target-specific case:

```text
Origami receiver contract
        ↓
Tlaloque/reference swarm
        ↓
successful traces
        ↓
receiver distillation
        ↓
Master Prompt candidate + optional higher-level support
        ↓
clean model/perception/tool experiments
        ↓
recommendation evidence
        ↓
Origami decides canonical adoption
```

The preferred compatibility outcome is still an Origami Master Prompt usable by a model without Tlaloc. Tools and deterministic runtime paths are optional higher deployment levels, not the universal baseline.

## Tlaloque design

Tlaloque remain deliberately bounded. A worker should solve a small local question whenever possible. Their value comes from making complex behavior decomposable, observable and falsifiable.

The final prompt need not mention the Tlaloque. It should preserve only the behavioral structure that matters.

## Fitness

Behavioral fitness is project-specific but should distinguish at least:

```text
task success
behavioral fidelity to reference behavior
generalization
regression rate
UNKNOWN/uncertainty discipline when applicable
prompt size
deployment level
tool/runtime requirements
cross-model transfer
```

The development system should not optimize prompt size by sacrificing hard invariants.

## Core rule

> Use complexity at development time to discover simplicity at deployment time.
