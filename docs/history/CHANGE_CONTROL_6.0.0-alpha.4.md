# Change control — Tlaloc 6.0.0-alpha.4

## Change

Normalize current architecture names after the Tlaloc/Origami split and remove the misleading `Oracle` component identity.

## Before

- bounded agents lived under `internal/swarm` and were called workers;
- deterministic expected-state code lived under `internal/oracle`;
- active documentation used `oracle` as a Tlaloc verification component;
- `swarm` was treated as a first-class component name.

## After

- bounded specialist agents live under `internal/tlaloque` and implement the `Tlaloque` interface;
- deterministic expected-state code lives under `internal/reference` and is described as reference semantics;
- current active documentation contains no `Oracle`, `Adivino` or `Báculo` component;
- `swarm` is retained only as a generic implementation descriptor where useful, not as the architecture name;
- `tlaloc tlaloque` exposes the current built-in specialists;
- Origami remains `6.0.0-alpha.1`; its state semantics are unchanged.

## Invariants

- Tlaloc remains the complete work system.
- Tlaloque are bounded and cannot promote their own changes.
- Origami remains an independent representation/state-machine language.
- Reference semantics are deterministic verification machinery, not an autonomous agent.
- BPFW/PipeCraft and user `.me/origami` workspaces remain outside managed cleanup.

## Impact

Behavior Lab package paths changed, so this is a Tlaloc alpha increment. The behavior contract and generated prompt semantics do not intentionally change.

## Promotion evidence

Required before packaging:

- `go test ./...`
- `go vet ./...`
- `go test -race ./...`
- deterministic prompt rebuild
- terminology scan on active docs/code
- installer lifecycle in isolated HOME
- independent Tlaloc/Origami uninstall
- ZIP integrity/hash verification
