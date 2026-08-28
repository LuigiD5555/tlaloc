> **HISTORICAL RECORD — SUPERSEDED.** Preserved for traceability; do not use this file as the current architecture contract. See `../ARCHITECTURE.md` and `../CAPABILITY_STATUS.md`.

# Change control — Tlaloc 6.0.0-alpha.2

## Component
Installation lifecycle / distribution packaging.

## Before
Tlaloc and Origami were distributed as split archives/overlays without a canonical user-local installer manifest. Legacy Origami/VCL/OHF experiments could have left binaries, user data roots, caches, config directories, and shell references with no common ownership record.

## After
- versioned installation roots under XDG user data;
- `current` symlinks for Tlaloc and Origami;
- SHA-256 manifests per installed version;
- canonical `tlaloc`, `origami`, `tlaloc-behavior-lab`, `tlaloc-uninstall`, and `origami-uninstall` entry points;
- exact managed uninstall based on ownership markers;
- legacy inventory/removal for Origami/OHF and optionally VCL;
- hard protection for BPFW/PipeCraft;
- hard protection for `.me/origami` project workspaces;
- optional backed-up shell cleanup;
- `tlaloc doctor` post-install verification.

## Evidence
A fake-HOME lifecycle test installs the bundle, detects/removes simulated historical Origami/OHF/VCL residue, preserves BPFW/PipeCraft and `.me/origami`, cleans shell startup references only when explicitly requested, and then fully removes the managed installation.

## Regression policy
The uninstaller must fail closed when a managed `current` link points outside the expected version root or lacks a managed marker. Generic VCL artifacts are medium-confidence and are not removed without `--aggressive-legacy`.

## Decision
PROMOTED_FOR_ALPHA — installation lifecycle is suitable for the alpha distribution, while legacy detection remains intentionally conservative.
