# Change control — Tlaloc 6.0.0-alpha.10

Date: 2026-08-28  
Status: `CANDIDATE_REPO_FLOW_OWNERSHIP_MIGRATION`

## Component changed

Project-skill ownership, Tlaloc CLI skill-distribution boundary, and repository verification gates.

## Before

Tlaloc `6.0.0-alpha.9` introduced the project-agnostic `repo-flow` skill and distributed it from `.claude/skills/repo-flow`. Tonal `0.1.0-alpha.2` subsequently established a neutral canonical `skills/repo-flow/SKILL.md` plus byte-identical `.claude/skills/` and `.agents/skills/` mirrors. Keeping a second active Tlaloc copy would create two authorities for the same shared workflow asset. Tlaloc also had release tests but no repository-owned GitHub Actions workflow enforcing them on pull requests.

## After

- Tlaloc advances to `6.0.0-alpha.10`.
- `.claude/skills/repo-flow/` is removed from Tlaloc.
- the five Tlaloc/Origami-specific project skills remain owned and distributed by Tlaloc;
- `tlaloc skills list/path/install` continues to work for those Tlaloc-owned skills;
- `tlaloc skills install repo-flow` fails deliberately with a migration message pointing to Tonal;
- the obsolete `tests/test-repo-flow-install.sh` is replaced by `tests/test-project-skill-install.sh`;
- isolated managed-install coverage verifies that installed Tlaloc no longer contains or lists `repo-flow`;
- `.github/workflows/verify.yml` now enforces release/version, terminology, project-skill, isolated-install, Go test/vet/race, and generated-artifact hash gates on pull requests and pushes to `main`.

## Ownership

Tonal owns the canonical project-agnostic `repo-flow` workflow and its multi-agent mirrors. Tlaloc owns only its architecture-specific checked-in project skills. This does not transfer BehaviorSpec, PromptIR, Tlaloque, reference semantics, Origami contracts, or Tlaloc release policy to Tonal.

Historical `6.0.0-alpha.9` records remain unchanged because they accurately describe that release at the time it was published.

## Impact closure / gates

The Tlaloc GitHub Actions workflow must pass:

- `tests/test-version-coherence.sh`;
- `tests/test-current-terminology.sh`;
- `tests/test-skills.sh`;
- `tests/test-project-skill-install.sh`;
- isolated managed install -> skill checks -> doctor -> uninstall via `tests/test-independent-install.sh`;
- `go test ./...`, `go vet ./...`, and `go test -race ./...` in Behavior Lab;
- `sha256sum -c GENERATED_ARTIFACTS.sha256`.

Downstream Tonal CI must then fetch the promoted Tlaloc alpha.10 exact commit and pass the composition gates before Tonal updates its released lock.

## Regression risk

Low. No Go/model-facing runtime source changes. The intentional compatibility break is limited to the old `tlaloc skills install repo-flow` path, which now produces an explicit migration error instead of silently keeping a duplicate authority.

## Downstream impact

Tonal must publish a new composition revision after this Tlaloc release is promoted so its immutable lock can reference the exact alpha.10 commit. Origami requires no change.

## Promotion decision

Promote only after the Tlaloc pull-request workflow passes. Tonal composition promotion is a separate downstream gate.
