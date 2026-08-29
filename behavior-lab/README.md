# Tlaloc Behavior Lab

Behavior Lab is Tlaloc's R0 experiment for compiling a formal behavior contract into a prompt artifact, testing that artifact against a target model and using bounded **Tlaloque** to propose repairs from structured failures.

The compiler lifecycle uses an explicit profile registry. The coherent-state and relational-core profiles have separate comparators, curricula and bounded Tlaloque. The relational path consumes upstream Origami fixtures; it does not duplicate the Origami Reference Machine.

## Terms

- `internal/reference` calculates deterministic expected states for the current profile.
- `internal/tlaloque` contains bounded specialist agents that diagnose findings and propose prompt patches.
- Tlaloc retains promotion authority; Tlaloque cannot promote their own proposals.

## Commands

```bash
go run ./cmd/behaviorlab compile
go run ./cmd/behaviorlab compile -spec profiles/origami/relational-core-r0.json -out generated/origami-relational-core-r0.generic.prompt.md
go run ./cmd/behaviorlab tlaloque
go run ./cmd/behaviorlab train -model <model-name>
```
