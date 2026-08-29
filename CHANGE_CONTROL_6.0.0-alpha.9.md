# Change control — Tlaloc 6.0.0-alpha.9

Date: 2026-08-28
Status: `PROMOTED_REUSABLE_SKILL_R0`

## Component changed

Checked-in project skills and the Tlaloc CLI skill-distribution workflow.

## Before

Tlaloc shipped project-local skills for its own architecture and exposed only `tlaloc skills-path`. Reusing a workflow skill in another repository required manual path discovery/copying, and there was no generic skill encoding the branch/commit/PR/CI/merge discipline used during Tlaloc and Origami maintenance.

## After

- Tlaloc advances to `6.0.0-alpha.9`.
- `.claude/skills/repo-flow/SKILL.md` defines a project-agnostic repository workflow covering preflight, scope/impact analysis, branches, diffs, verification, commits, PRs, semantic conflict resolution, CI, guarded merge, post-merge verification, release/version coherence and multi-repository snapshots.
- `tlaloc skills list` enumerates checked-in skills.
- `tlaloc skills path` reports their installed source directory.
- `tlaloc skills install <name>` locates the target Git repository root and copies a skill into `.claude/skills/<name>`.
- existing differing project copies are protected by default; replacement requires explicit `--force`.
- legacy `tlaloc skills-path` remains available for compatibility.

## Ownership and boundaries

`repo-flow` is workflow guidance, not generated SkillIR and not a replacement for a repository's own `CONTRIBUTING.md`, `CLAUDE.md`, release policy, branch protection or CI requirements. Project-local rules take precedence where they are more specific.

The change does not alter BehaviorSpec, PromptIR, Tlaloque behavior, deterministic reference semantics, Origami semantics, installer ownership or legacy cleanup boundaries.

## Impact closure / tests

Required gates for this change:

- Bash syntax for `tools/tlaloc` and affected shell tests;
- `tests/test-skills.sh`;
- `tests/test-repo-flow-install.sh`;
- `tests/test-version-coherence.sh`;
- `tests/test-current-terminology.sh`;
- existing Go behavior-lab tests/vet/race as release regression confirmation;
- generated prompt hash remains unchanged.

## Regression risks

- a CLI parsing regression could break existing Tlaloc commands;
- careless skill installation could overwrite project-local guidance; mitigated by repository-root resolution and default overwrite refusal;
- the generic skill could overrule project-specific conventions; mitigated by explicit instruction to read and obey repository-local policy.

## Downstream impact

No change is required in Origami. Any Git repository may opt in by installing `repo-flow`; installation does not create a runtime dependency on Tlaloc inside that repository.

## Evidence

All declared gates passed locally on the release candidate:

- shell syntax: PASS;
- `tests/test-skills.sh`: PASS;
- `tests/test-repo-flow-install.sh`: PASS, including idempotence, local-edit refusal and explicit `--force`;
- `tests/test-version-coherence.sh`: PASS;
- `tests/test-current-terminology.sh`: PASS;
- `tests/test-independent-install.sh`: PASS for isolated-HOME install -> doctor -> uninstall at `6.0.0-alpha.9`;
- `go test ./...`: PASS;
- `go vet ./...`: PASS;
- `go test -race ./...`: PASS;
- generated prompt SHA-256 remains `5f46e56e72f793d214053d87b30906cfe924a5fcb652450311e258519d981504`.

## Promotion decision

Promoted as `PROMOTED_REUSABLE_SKILL_R0`.
