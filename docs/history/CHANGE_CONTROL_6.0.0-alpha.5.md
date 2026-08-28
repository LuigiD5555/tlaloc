# Change control — Tlaloc 6.0.0-alpha.5

## Change

Close release-hygiene gaps discovered after alpha.4: project-agent skills were absent, current documentation contained small stale terms, and alpha.2 left a state manifest that was not included in retro cleanup.

## Before

- no `.claude/skills/` existed despite planned Claude-oriented workflows;
- no project `CLAUDE.md` captured Tlaloc/Origami ownership rules;
- capability documentation correctly said generated Claude Skills were not implemented, but did not distinguish project workflow skills from future generated SkillIR;
- `docs/LEGACY_INSTALLATION_MAP.md` still said “Alpha.3 recognizes...”;
- Origami's active changelog still listed `Swarm Trainer` as a current Tlaloc-owned concept name;
- alpha.2's `XDG_STATE_HOME/tlaloc/install-manifest-v1.tsv` could survive upgrades/cleanup.

## After

- project-local Claude Code guidance is bundled and release-tested;
- project skills and future generated skills are explicitly separate concepts;
- `tlaloc skills-path` exposes the installed skill copy without touching global `~/.claude` configuration;
- active docs and README are synchronized with alpha.5 capabilities;
- obsolete alpha.2 state manifest is removed after successful managed install and is recognized by legacy cleanup;
- legacy user-integration scanning includes additional systemd/autostart/completion paths with project-specific prefixes;
- retrocompatibility tests cover pre-managed residue, alpha.2 marker quirks/state residue, alpha.3+ component markers and current independent uninstall behavior.

## Invariants

- Tlaloc remains the complete work system.
- Tlaloque remain bounded and cannot self-promote.
- Origami remains independent and at `6.0.0-alpha.1` in this bundle.
- `BehaviorSpec + invariants` remain the behavioral source of truth; checked-in skills are guidance only.
- Installing Tlaloc must not mutate `~/.claude`.
- BPFW/PipeCraft and `.me/origami` remain hard cleanup exclusions.

## Promotion evidence required

- `go test ./...`
- `go vet ./...`
- `go test -race ./...`
- deterministic prompt rebuild
- skill structure/frontmatter validation
- active terminology/capability scan
- isolated-HOME install lifecycle
- alpha.2 marker + state-manifest migration test
- alpha.3/current component-marker uninstall test
- pre-managed Origami/OHF/VCL cleanup test
- independent Tlaloc/Origami uninstall
- ZIP integrity and SHA-256 manifests
