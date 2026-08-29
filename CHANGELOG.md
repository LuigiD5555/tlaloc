# Tlaloc changelog

## 6.0.0-alpha.10 — Repo-flow ownership migration

- removes the project-agnostic `repo-flow` skill from Tlaloc now that Tonal is its canonical distribution owner;
- keeps the five Tlaloc/Origami-specific project skills under `.claude/skills/`;
- keeps `tlaloc skills list/path/install` for Tlaloc-owned skills only;
- `tlaloc skills install repo-flow` now fails with an explicit migration message pointing to Tonal instead of installing a stale copy;
- replaces the old repo-flow-specific Tlaloc regression with project-skill installation coverage and asserts managed Tlaloc installs no longer contain `repo-flow`;
- adds `.github/workflows/verify.yml` so pull requests and `main` pushes enforce release/version, terminology, skills, isolated install, Go test/vet/race, and generated-artifact hash gates;
- no BehaviorSpec, PromptIR, Tlaloque, reference semantics, Origami contracts, generated prompt, or model-facing runtime behavior changed.

## 6.0.0-alpha.9 — Reusable repository workflow skill

- added the project-agnostic `repo-flow` skill for Git/GitHub repository work: preflight, branch discipline, impact-scoped verification, atomic commits, PR review, conflict resolution, CI gating, merge and post-merge verification;
- codified release/version consistency, changelog/change-control hygiene and multi-repository snapshot/pin rules without making submodules a user-project requirement;
- added `tlaloc skills list`, `tlaloc skills path`, and `tlaloc skills install <name>`;
- `tlaloc skills install repo-flow` installs into the current Git repository root and refuses to overwrite differing local content unless `--force` is explicit;
- added regression coverage for skill discovery, idempotent installation, local-edit protection and explicit forced replacement;
- no BehaviorSpec, PromptIR, Tlaloque, reference semantics, Origami contracts, or model-facing runtime behavior changed.

## 6.0.0-alpha.8 — Release/version coherence

- fixed `install.sh` still installing `6.0.0-alpha.6` while the repository declared a newer release;
- made root `VERSION` the installer version source instead of duplicating a hard-coded value;
- added an installation regression gate that requires the managed version path/marker and `tlaloc version` to match root `VERSION`;
- synchronized active integration guidance with Origami `6.0.0-alpha.3`, which preserves the alpha.2 perceptual contract while clarifying `Origami > OHF`;
- updated active release documentation without changing BehaviorSpec, PromptIR, Tlaloque, reference semantics, or executable Origami integration behavior.

## 6.0.0-alpha.7 — Origami perceptual-contract tracking

- tracks Origami `6.0.0-alpha.2` and `origami.perceptual-channels.r0` as an upstream semantic contract;
- updates the Origami project skill with interference/moiré, depth/stereo/parallax, Temporal Latent Image and temporal/emergent terminology;
- distinguishes contract awareness from executable runtime support;
- explicitly marks `MOIRE`, `PHASE_SHIFT`, `STEREO_BIND`, `PARALLAX_RESOLVE`, `KINETIC_REVEAL`, `TEMPORAL_INTEGRATE`, and `TEMPORAL_DECAY` as not yet implemented by Tlaloc evaluators/Tlaloque;
- preserves the current coherent-state behavior profile and generated prompt unchanged.

## 6.0.0-alpha.6 — Independent repository lifecycle

- Tlaloc repository becomes independently installable from source.
- `install.sh` no longer installs or requires Origami.
- `doctor` treats Origami as optional instead of a required installation.
- direct `uninstall.sh` defaults to Tlaloc-only removal; bundle/origami modes remain explicit for retrocompatibility.
- legacy Origami/OHF/VCL cleanup remains available without transferring ownership of Origami to Tlaloc.
- BPFW/PipeCraft and `.me/origami` remain hard-protected.

## 6.0.0-alpha.5 — Agent guidance + lifecycle retrocompatibility audit

- added project `CLAUDE.md` guidance;
- added five project-local Claude Code skills under `.claude/skills/`;
- explicitly separated checked-in project skills from future compiler-generated SkillIR/Claude Skills;
- added `tlaloc skills-path`;
- added release validation for skill metadata and mirrored copies;
- fixed stale active documentation (`Swarm Trainer`, alpha.3 compatibility wording);
- added detection/removal of the obsolete alpha.2 `XDG_STATE_HOME/tlaloc/install-manifest-v1.tsv`;
- expanded conservative legacy scanning to additional user integration/completion artifacts;
- strengthened generation-specific retro-uninstall tests.

Origami remains `6.0.0-alpha.1`; no Origami state semantic law changed.

## 6.0.0-alpha.4 — Tlaloque nomenclature + reference semantics cleanup

### Changed
- renamed the bounded-agent implementation from `internal/swarm` to `internal/tlaloque`;
- introduced `Tlaloque` as the official name for Tlaloc bounded specialist agents;
- renamed deterministic expected-state machinery from `internal/oracle` to `internal/reference`;
- removed Oracle/Adivino/Báculo from current architecture terminology;
- added `docs/NOMENCLATURE.md`;
- added `tlaloc tlaloque` introspection;
- updated active Tlaloc/Origami integration documents to the new naming contract.

### Versioning
Tlaloc advances to `6.0.0-alpha.4`. Origami remains `6.0.0-alpha.1`; its representation semantics are unchanged.

## 6.0.0-alpha.3 — Concept/documentation depuration R0

### Corrected
- separated current architecture docs from historical records;
- removed mandatory-dependency wording between Tlaloc and Origami;
- clarified Origami reference semantics vs Tlaloc oracle/evaluation authority;
- removed global `Master Prompt` terminology from current architecture;
- documented actual model support without claiming native Claude/skills support;
- moved the bundled Origami behavior contract into an explicit profile namespace;
- regenerated the stale compiled prompt and artifact hash;
- introduced component-specific managed-install markers with alpha.2 compatibility.

### Versioning
Tlaloc advances to `6.0.0-alpha.3`. Origami remains `6.0.0-alpha.1`; its representation semantics are unchanged.

## 6.0.0-alpha.2 — Managed installation lifecycle

### Added
- complete user-local installer for Tlaloc + current Origami representation package;
- versioned XDG installation roots and `current` links;
- per-version SHA-256 installation manifests;
- `tlaloc` dispatcher and `origami` representation CLI;
- `tlaloc doctor`;
- exact managed uninstall;
- historical Origami/OHF/VCL inventory and cleanup;
- optional shell-startup cleanup with automatic backup;
- explicit BPFW/PipeCraft and `.me/origami` protections.

### Versioning
Tlaloc advances to `6.0.0-alpha.2`. Origami remains `6.0.0-alpha.1` because its representation semantics did not change in this packaging release.

## 6.0.0-alpha.1 — Tlaloc / Origami split + Behavior Compilation R0

Common historical branch point of Tlaloc and Origami.

### Added
- formal Tlaloc project identity;
- `BehaviorSpec -> PromptIR -> compiled prompt` pipeline;
- bounded Go swarm trainer;
- guard workers and central arbiter;
- deterministic regression/evaluation path;
- OpenAI-compatible local target adapter;
- Origami quantum-inspired behavior profile as the first consumer profile.

### Naming correction
- `Origami Behavior Lab` -> `Tlaloc Behavior Lab`;
- Go module `origami.local/behaviorlab` -> `tlaloc.local/behaviorlab`;
- compiled prompt ownership moved to Tlaloc.
