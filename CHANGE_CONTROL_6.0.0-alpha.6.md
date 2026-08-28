# Change control — Tlaloc 6.0.0-alpha.6

Date: 2026-08-28

## Change

Make the GitHub Tlaloc repository operationally independent from the Origami repository while preserving retrocompatible cleanup for historical unified installations.

## Before

- alpha.5 source tree was packaged inside a combined Tlaloc + Origami installer bundle;
- `doctor.sh` treated an installed Origami component as mandatory;
- direct execution of the shared `uninstall.sh` could default to the old bundle scope.

## After

- repository-local `install.sh` installs only Tlaloc;
- Origami is optional and independently versioned;
- `doctor.sh` reports Origami presence informationally but does not fail when absent;
- direct `uninstall.sh` removes Tlaloc by default;
- explicit legacy/bundle modes remain available solely for safe migration from historical installations;
- legacy scanning still protects BPFW/PipeCraft and `.me/origami` workspaces.

## Impact

BehaviorSpec, PromptIR, Tlaloque behavior, reference semantics and the generated Origami profile prompt are unchanged.

## Promotion evidence

Required gates:

- `go test ./...`
- `go vet ./...`
- `go test -race ./...`
- Bash syntax checks
- isolated-HOME install without Origami
- `tlaloc doctor` without Origami
- Tlaloc-only uninstall
- legacy scanner protection checks

## Decision

Promote as `6.0.0-alpha.6` if all gates pass. Origami remains `6.0.0-alpha.1`.
