> **HISTORICAL RECORD — SUPERSEDED.** Preserved for traceability; do not use this file as the current architecture contract. See `../ARCHITECTURE.md` and `../CAPABILITY_STATUS.md`.

# Change control — Tlaloc 6.0.0-alpha.1

Date: 2026-08-28

## Change

Formal separation of Tlaloc from Origami while preserving the 6.0.0-alpha.1 historical branch number.

## Before

Behavior Compiler and Swarm Trainer were packaged under the Origami project identity.

## After

They are owned by Tlaloc. Origami remains a representation provider and the first behavior profile exercised by Tlaloc Behavior Lab.

## Compatibility

No behavior-lab algorithm was intentionally changed by the namespace split. Go import/module names and user-visible ownership labels changed.

## Promotion gate

- `go test ./...`
- `go vet ./...`
- `go test -race ./...`
- deterministic prompt rebuild comparison
