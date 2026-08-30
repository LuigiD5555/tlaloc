# Tlaloc current architecture

**TLALOC — Transformative Latent Adaptive Logic Orchestration Core** is a development kit for behavioral discovery, experimentation and distillation.

## Canonical lifecycle

```text
Intent
  -> BehaviorSpec + invariants
  -> bounded Tlaloque swarm
  -> execution / observations / traces
  -> reference behavior evidence
  -> distillation
  -> PromptIR / prompt candidates
  -> clean target-family execution
  -> behavioral-fidelity evaluation
  -> regression / candidate selection
  -> portable prompt artifact
```

The swarm is a **reference laboratory** used to discover a procedure that works. It is not the default production architecture.

## Prompt-first deployment

The portable deployment target is `L0_PROMPT_ONLY` whenever the required behavior can be preserved there.

```text
L0 PROMPT_ONLY
L1 PROMPT_PLUS_DECLARATIVE_CONTEXT_OR_IR
L2 PROMPT_PLUS_TOOLS
L3 PROMPT_PLUS_RUNTIME
L4 SPECIALIZED_MODEL_OR_TARGET_SPECIFIC_SYSTEM
```

Tlaloc minimizes deployment requirements subject to behavioral fidelity.

This means development may use a sandbox, tools, Go, many agents and large evaluators while an accepted L0 artifact must still work with only an LLM text interface.

## Authority hierarchy

1. `BehaviorSpec` + declared invariants define desired behavior.
2. Swarm execution and tests demonstrate a reference procedure that can satisfy that behavior.
3. Distillation extracts compact behavioral rules/instructions from successful traces.
4. Prompt/PromptIR artifacts are deployment candidates, not truth merely because they were generated.
5. Clean-target evaluation compares the candidate against the requested/reference behavior.
6. A target project remains authoritative over its own releases and contracts.

## Tlaloque

Tlaloque are deliberately small specialist workers under Tlaloc. They are useful because decomposition makes a complex requested behavior observable and testable as many simple actions.

A Tlaloque can perform a bounded operation, inspect one piece of state, test one predicate, compare one item, open one evidence source or propose one repair. Complex behavior comes from their composition, not from treating each Tlaloque as a general autonomous intelligence.

## Behavioral distillation

The distillation target is:

```text
Behavior(candidate artifact) ~= Behavior(reference swarm)
```

not textual reproduction of the trace.

A trace with dozens of agents and intermediate actions may compress to a much smaller prompt if the prompt captures the decision order, invariants, uncertainty behavior and verification rules that caused success.

## General target boundary

Tlaloc is not defined by Origami.

Possible targets include:

```text
Origami
calculator behavior
document workflows
classifiers
other prompted applications
other software/tool behavior
```

Target-specific adapters and experiments may be extensive, but they remain profiles on top of the general behavioral lifecycle.

## Origami target profile

For Origami, Tlaloc currently provides optional development machinery such as:

- Canonical Document IR / OCR and exact-memory experiments;
- Tlaloque proposal generation;
- cross-model perception campaigns;
- prompt/representation search;
- color/numeric/interference/depth/temporal candidate experiments;
- evidence-backed visual-profile tournaments.

Origami owns Origami semantics, its Master Prompt releases, visual grammar and canonical profile promotion. Tlaloc supplies experimental candidates and evidence.

## Tonal boundary

Tonal is not a required Tlaloc layer. Tonal may compose several independent development systems, for example Tlaloc plus Blueprint Framework, and pin exact revisions for reproducibility.

```text
Tonal
  -> Tlaloc
  -> Blueprint Framework
  -> other development tools
  -> target revisions
```

Tonal does not define Tlaloc's behavioral semantics and does not own releases of target projects.

## Clean-target rule

An L0 prompt candidate may not depend on:

- Tlaloc internal state;
- swarm traces;
- a hidden sandbox;
- undeclared tools;
- evaluator ground truth;
- private target answers.

Higher deployment levels may use additional dependencies only when those dependencies are explicitly declared.

## Existing richer runtimes

Existing Hybrid, tool-loop, PDF-memory, Origami visual-search and other runtime components remain valuable **development/evaluation machinery** and explicit higher-level deployment options. They do not change the prompt-first baseline.

## Operational agent guidance

`CLAUDE.md` and `.claude/skills/` are checked-in instructions for coding agents working on this repository. They are development assets, not Tlaloc's portable behavioral output.

## Current implementation

`behavior-lab/internal/distill/promptfirst.go` implements deterministic selection of the least demanding behaviorally valid artifact. The existing receiver distillation, PromptIR/compiler, evaluator, Tlaloque and target adapters remain available and are now interpreted within this general hierarchy.

See also:

- `docs/PROMPT_FIRST_R0.md`
- `behavior-lab/spec/PROMPT_FIRST_DISTILLATION_R0.json`
- `docs/NOMENCLATURE.md`
