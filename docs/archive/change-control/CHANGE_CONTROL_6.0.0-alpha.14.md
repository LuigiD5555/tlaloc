# Change Control — Tlaloc 6.0.0-alpha.14

## Change

Make prompt-first behavioral distillation the canonical Tlaloc organization.

## Motivation

Tlaloc may use a rich swarm, tools, sandboxes and runtime helpers during development, but the deployed target model may have only a text interface. The previous architecture described many useful Origami-specific runtimes but did not state strongly enough that those development dependencies should be removed whenever a prompt can preserve the demonstrated behavior.

## New canonical rule

```text
intent
 -> bounded Tlaloque swarm
 -> verified reference behavior
 -> distillation
 -> prompt candidate
 -> clean target evaluation
 -> least demanding valid deployment artifact
```

Default portable target:

```text
L0_PROMPT_ONLY
```

Fallback levels are explicit and ordered from declarative context through tools/runtime to specialized targets.

## Preserved work

- BehaviorSpec and PromptIR;
- existing Tlaloque/reference engines;
- Origami Canonical Memory R2;
- Perception Promotion Campaign R1;
- Origami Visual Evolution R0;
- Hybrid/tool-loop runtimes;
- installers and project-local skills.

Those become development/evaluation machinery or explicit higher deployment levels rather than universal deployment requirements.

## Authority correction

Origami is one Tlaloc target and owns Origami releases. Tonal is an optional multi-tool composition layer. Neither relationship defines Tlaloc's core behavioral lifecycle.

## Hard invariants

```text
PROMPT_FIRST_FOR_PORTABILITY
SWARM_IS_REFERENCE_LAB_NOT_DEFAULT_DEPLOYMENT
L0_REQUIRES_NO_SANDBOX
L0_REQUIRES_NO_TOOLS
L0_REQUIRES_NO_TLALOC_RUNTIME
DEVELOPMENT_TOOLS_NE_DEPLOYMENT_REQUIREMENTS
BEHAVIORAL_FIDELITY_NE_TRACE_TEXT_SIMILARITY
CLEAN_TARGET_EVALUATION_REQUIRED
ORIGAMI_IS_A_TARGET_NOT_TLALOC_IDENTITY
```
