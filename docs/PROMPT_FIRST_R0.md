# Prompt-First Distillation R0

Tlaloc is a development kit for discovering how to perform an intention through a swarm of deliberately small actions, validating that behavior, and then compressing the demonstrated behavior into the most portable deployable instruction artifact.

The default deployment target is a **prompt**.

```text
INTENT
  -> SWARM OF SMALL STEPS
  -> EXECUTION + TESTS
  -> SUCCESSFUL REFERENCE BEHAVIOR
  -> DISTILLATION
  -> PROMPT CANDIDATES
  -> SAME TESTS ON CLEAN TARGET MODELS
  -> BEHAVIORAL FIDELITY
  -> PROMPT ARTIFACT
```

The swarm is the experimental reference procedure, not the default production runtime.

## Why prompt-first

A prompt is the lowest common deployment interface for an LLM. It can remain useful when the target model has no sandbox, no Go/Python runtime, no tools, no file access and no Tlaloc installation.

Tlaloc may use richer infrastructure while developing and testing behavior. That development complexity must not silently become a deployment dependency.

## Deployment ladder

Tlaloc prefers the least demanding artifact that preserves the required behavior:

```text
L0  PROMPT_ONLY
L1  PROMPT_PLUS_DECLARATIVE_CONTEXT_OR_IR
L2  PROMPT_PLUS_TOOLS
L3  PROMPT_PLUS_RUNTIME
L4  SPECIALIZED_MODEL_OR_TARGET_SPECIFIC_SYSTEM
```

Selection rule:

```text
minimize DeploymentRequirements(artifact)
subject to BehavioralFidelity(artifact, reference_swarm) >= required_threshold
```

L1-L4 are explicit fallbacks or optimizations when L0 cannot preserve the requested behavior. They do not redefine the portable baseline.

## Behavioral equivalence

Distillation does not copy the swarm transcript. It attempts to preserve the behavior that made the swarm succeed:

```text
Behavior(prompt) ~= Behavior(reference swarm)
```

not:

```text
Text(prompt) ~= Trace(reference swarm)
```

The reference trace may contain many Tlaloque, loops and intermediate checks. The final prompt should encode the smallest useful rules, ordering, uncertainty discipline and verification behavior that reproduce the outcome.

## Tlaloque

Tlaloque are intentionally bounded workers. A useful swarm may contain many individually simple steps such as:

```text
extract one claim
classify one relation
check one condition
compare one pair
open one evidence address
mark UNKNOWN
validate one invariant
```

Complex behavior emerges from composition, state, order, branching and repetition.

## Generality

Origami is one development target, not Tlaloc's definition.

The same loop may be used to develop:

- an Origami Master Prompt or representation rule;
- a calculator behavior;
- a document-analysis procedure;
- a classifier;
- a workflow;
- another software/tool behavior.

Tlaloc may discover code-like or runtime-assisted procedures during experimentation, but prompt-only deployment remains the compatibility target whenever behavioral fidelity permits it.

## Clean-target evaluation

The final prompt must be tested against clean target models that do not receive the hidden swarm trace, development sandbox, evaluator internals or tool state unless the selected deployment class explicitly permits those dependencies.

A successful swarm is evidence that a procedure can solve the task. It is not evidence that the distilled prompt can solve it.

## Hard rules

```text
SWARM_IS_REFERENCE_LAB_NOT_DEFAULT_DEPLOYMENT
PROMPT_FIRST_FOR_PORTABILITY
DEVELOPMENT_TOOLS_NE_DEPLOYMENT_REQUIREMENTS
BEHAVIORAL_FIDELITY_NE_TRACE_TEXT_SIMILARITY
CLEAN_TARGET_EVALUATION_REQUIRED
NO_TOOL_ASSUMPTION_FOR_L0
NO_SANDBOX_ASSUMPTION_FOR_L0
FALLBACK_LEVEL_MUST_BE_EXPLICIT
```
