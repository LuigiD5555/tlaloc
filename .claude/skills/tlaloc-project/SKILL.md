---
name: tlaloc-project
description: This skill should be used when reviewing, editing, explaining, or navigating the Tlaloc project architecture, naming, ownership boundaries, current-vs-historical documentation, or version split with Origami.
version: 0.1.0
---

# Tlaloc project architecture

Apply the current ownership model before making changes.

## Required distinctions

- Treat Tlaloc as the complete work system.
- Treat Tlaloque as bounded specialist agents coordinated by Tlaloc.
- Treat Origami as an independent representation/state-machine language.
- Treat reference semantics as deterministic verification machinery, not an autonomous agent.
- Keep `swarm` only as a generic distributed-computation description when useful; never use `Swarm Trainer` as the current component identity.
- Do not introduce `Oracle`, `Adivino`, or `Báculo` as current components.

## Authority

Use `BehaviorSpec + invariants` as the behavioral source of truth. Treat prompts, skills, examples and model-family instruction packages as derived or operational artifacts.

When a document conflicts with `docs/NOMENCLATURE.md`, `docs/ARCHITECTURE.md`, or `docs/CAPABILITY_STATUS.md`, flag the conflict rather than silently merging the two descriptions.

## Version ownership

Increment Tlaloc only for Tlaloc-owned changes. Increment Origami only for representation/semantic changes. Their shared historical branch point is `6.0.0-alpha.1`, but versions now advance independently.
