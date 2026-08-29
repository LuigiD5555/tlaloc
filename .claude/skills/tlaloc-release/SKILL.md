---
name: tlaloc-release
description: This skill should be used when preparing a Tlaloc or Tlaloc+Origami release, updating README/docs/version manifests, changing install or uninstall behavior, auditing legacy cleanup, or validating backward-compatible removal of older project generations.
version: 0.2.0
---

# Tlaloc release and lifecycle discipline

Treat release hygiene as part of correctness.

## Documentation

- Update root README, component README, architecture, nomenclature, capability status, changelog and change-control records together when affected.
- Keep historical documents immutable except for relocation/indexing; do not rewrite history to look current.
- Scan active docs for stale component names and capability overclaims.

## Install lifecycle

- Treat root `VERSION` as the single release-version source; installers must read it rather than duplicate a hard-coded version.
- Add a regression test that fails when the installed managed-version marker differs from root `VERSION`.
- Keep user-local installation manifest-driven and versioned.
- Preserve component-specific ownership markers.
- Keep Tlaloc and Origami independently uninstallable.
- Do not modify BPFW/PipeCraft.
- Do not delete `.me/origami` project workspaces.

## Retrocompatibility

Exercise known generations explicitly:

- pre-managed Origami/VCL/OHF paths through conservative legacy scanning;
- Tlaloc/Origami `6.0.0-alpha.2` ownership-marker migration;
- alpha.2 state-manifest residue cleanup;
- alpha.3+ component-specific markers;
- current managed versions and `--all-managed-versions` behavior.

Use medium-confidence deletion only behind an explicit aggressive flag. Back up shell startup files before removing legacy environment lines.

## Release gates

Run syntax checks, Go test/vet/race, skill validation, terminology/capability scans, isolated-HOME install/uninstall tests, legacy cleanup tests, archive integrity checks and SHA-256 regeneration.
