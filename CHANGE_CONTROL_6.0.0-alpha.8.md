# Change control — Tlaloc 6.0.0-alpha.8

Date: 2026-08-28
Status: `PROMOTED_RELEASE_HYGIENE_FIX`

## Component changed

Release/version coherence in the independent Tlaloc repository.

## Before

Root `VERSION` and the active README/capability documentation declared `6.0.0-alpha.7`, but `install.sh` still hard-coded `6.0.0-alpha.6`. A clone of `main` therefore installed a managed directory and marker for the wrong release. The existing independent-install test checked lifecycle success but not release identity.

## After

- Tlaloc advances to `6.0.0-alpha.8`.
- `install.sh` reads the release identifier from root `VERSION`; it no longer duplicates a hard-coded release string.
- installation tests require the managed target, managed-version marker and `tlaloc version` output to equal root `VERSION`.
- the release skill explicitly requires this invariant.
- active documentation is synchronized to alpha.8 and current Origami `6.0.0-alpha.3`;
- the `origami-semantics` project skill records that OHF is a nested Origami research track, while the perceptual semantic contract remains the alpha.2 contract.

No BehaviorSpec, PromptIR, generated prompt, evaluator/reference semantics, Tlaloque behavior or Origami semantic contract changed.

## Tests

- Bash syntax for installer/uninstaller/tools/tests.
- `tests/test-version-coherence.sh`.
- `tests/test-current-terminology.sh`.
- `tests/test-skills.sh`.
- `tests/test-independent-install.sh`.
- `go test ./...`, `go vet ./...`, `go test -race ./...` in `behavior-lab`.
- generated prompt hash comparison against `GENERATED_ARTIFACTS.sha256`.

## Downstream impact

Managed installations now use the repository-declared version. Existing alpha.2+ uninstall compatibility and legacy Origami/OHF/VCL cleanup remain unchanged.

## Promotion decision

Promote after all listed gates pass.
