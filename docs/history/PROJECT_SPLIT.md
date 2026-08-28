> **HISTORICAL RECORD — SUPERSEDED.** Preserved for traceability; do not use this file as the current architecture contract. See `../ARCHITECTURE.md` and `../CAPABILITY_STATUS.md`.

# Project split at 6.0.0-alpha.1

The former unified Origami working tree mixed two concepts under the Origami name. The formal split occurred at `6.0.0-alpha.1` without resetting historical numbering.

- **Tlaloc** owns the work system: orchestration, behavior compilation, swarm training, verification, promotion control, and model-facing execution.
- **Origami** owns representation: visual/state language, machine semantics, dynamics, folding/unfolding, interference, coupling, and related representations.

`6.0.0-alpha.1` is therefore the common branch point. After that release each project increments only when its own public behavior, packaging, or contracts change.

Historical alpha.2 wording called Origami a representation "dependency/profile package". **That dependency wording is superseded:** Origami is an independent representation provider that may be bundled with Tlaloc but is not mandatory.
