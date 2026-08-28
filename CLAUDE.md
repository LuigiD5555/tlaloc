# Tlaloc project instructions

Tlaloc is the work system. Tlaloque are bounded specialist agents. Origami is an independent representation provider. Reference semantics are deterministic verification machinery.

Use `BehaviorSpec + invariants` as the behavioral source of truth. Treat prompts and skills as derived/operational artifacts. Do not reintroduce retired component names (`Oracle`, `Adivino`, `Báculo`, `Swarm Trainer`).

When changing Tlaloc:

- keep Tlaloque bounded and unable to self-promote;
- keep representation ownership outside Tlaloc;
- keep current capability claims aligned with code;
- preserve deterministic prompt generation;
- preserve installer/uninstaller safety and retrocompatibility;
- run `go test ./...`, `go vet ./...`, and `go test -race ./...` in `behavior-lab` before promotion.

See `.claude/skills/` for task-specific workflows and `docs/` for current architecture. Historical material is under `docs/history/`.
