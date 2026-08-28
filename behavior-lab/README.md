# Tlaloc Behavior Lab

Behavior Lab is Tlaloc's R0 experiment for compiling a formal behavior contract into a prompt artifact, testing that artifact against a target model and using bounded **Tlaloque** to propose repairs from structured failures.

The compiler lifecycle is intended to become profile- and model-family-independent. In this R0 package, however, the bundled evaluator, reference semantics, default Tlaloque and curriculum are still specialized for the first profile: `origami.quantum-inspired.r0`.

## Terms

- `internal/reference` calculates deterministic expected states for the current profile.
- `internal/tlaloque` contains bounded specialist agents that diagnose findings and propose prompt patches.
- Tlaloc retains promotion authority; Tlaloque cannot promote their own proposals.

## Commands

```bash
go run ./cmd/behaviorlab compile
go run ./cmd/behaviorlab tlaloque
go run ./cmd/behaviorlab train -model <model-name>
```
