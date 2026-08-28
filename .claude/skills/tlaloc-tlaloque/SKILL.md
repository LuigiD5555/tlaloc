---
name: tlaloc-tlaloque
description: This skill should be used when creating, reviewing, renaming, or modifying Tlaloque specialist agents, their findings, repair proposals, boundaries, coordination, or promotion permissions.
version: 0.1.0
---

# Tlaloque design contract

Design every Tlaloque as a bounded specialist under Tlaloc.

## Invariants

- Give each Tlaloque one narrow responsibility.
- Prefer deterministic code when it is sufficient; introduce a small model only when deterministic logic is inadequate and evaluation justifies it.
- Accept structured inputs/findings where possible.
- Emit structured diagnosis, evidence, or repair proposals.
- Never let a Tlaloque modify the source specification without an explicit Tlaloc-controlled patch path.
- Never let a Tlaloque promote, approve, or certify its own proposal.
- Keep centralized regression and promotion authority in Tlaloc.

When adding a Tlaloque, add tests for its positive case, non-applicable case, false-positive boundary, and interaction with promotion gates.
