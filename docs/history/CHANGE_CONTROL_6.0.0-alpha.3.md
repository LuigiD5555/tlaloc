# Change control — Tlaloc 6.0.0-alpha.3

## Component
Concept/documentation boundary + installation ownership cleanup.

## Before
Several files still reflected the pre-split vocabulary or overstated R0 capabilities: a stale Origami-owned generated prompt, a mandatory-dependency implication, ambiguous oracle ownership, `Master Prompt` terminology, target-specific compilation claims without model-family backends, and Tlaloc ownership markers inside Origami installation roots.

## After
- current architecture and capability-status documents are authoritative;
- historical split/change records are archived under `docs/history/`;
- Origami is explicitly independent and optional to Tlaloc;
- reference semantics (Origami) and verification/oracle role (Tlaloc) are separated;
- compiled prompts/skills are artifacts, never the source specification;
- native Claude/skills support is explicitly marked not implemented;
- component-specific installation ownership markers are used;
- `tlaloc-uninstall` and `origami-uninstall` are component-scoped by default; bundle removal is explicit;
- alpha.2 marker formats remain accepted for safe upgrade/uninstall;
- the bundled Origami BehaviorSpec is stored as a profile, not a generic example.

## Promotion gate
- documentation consistency scan;
- `go test ./...`;
- `go vet ./...`;
- `go test -race ./...`;
- deterministic prompt rebuild;
- fake-HOME install/uninstall lifecycle;
- legacy alpha.2 marker compatibility;
- BPFW/PipeCraft and `.me/origami` preservation.

## Evidence
- `go test ./...`: PASS
- `go vet ./...`: PASS
- `go test -race ./...`: PASS
- deterministic PromptIR 0.2 rebuild: PASS (byte-identical)
- fake-HOME install + legacy cleanup + component-scoped uninstall: PASS
- alpha.2 Origami marker uninstall compatibility: PASS
- bundle uninstall smoke: PASS
- current-document forbidden/overclaim scan: PASS
- BPFW/PipeCraft and `.me/origami` preservation regression: PASS

## Decision
PROMOTED_FOR_ALPHA — documentation and ownership cleanup are internally consistent for the alpha line.
