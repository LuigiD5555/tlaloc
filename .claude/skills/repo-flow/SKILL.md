---
name: repo-flow
description: This skill should be used for repository work involving Git or GitHub: inspecting state, creating branches, making atomic commits, opening or repairing pull requests, resolving conflicts, running CI gates, merging safely, synchronizing versions/changelogs/change-control records, or composing multiple repositories into tested snapshots.
version: 0.1.0
---

# Repo Flow

Use a repository workflow that preserves user work, keeps history understandable, and makes promotion evidence explicit.

The objective is not to maximize Git ceremony. The objective is to make each repository change easy to inspect, test, review, reproduce, merge, and recover from.

## Core rules

1. **Inspect before changing.** Determine repository root, current branch, upstream/base, HEAD, dirty files, untracked files, and project-local contribution/release rules before editing.
2. **Never discard unknown work.** Do not reset, clean, checkout over, stash, amend, rebase, force-push, or delete user changes unless the user explicitly asked for that action and the affected state is understood.
3. **Keep source-of-truth boundaries intact.** A documentation fix must not silently redefine runtime semantics; a packaging change must not silently alter an independently versioned component.
4. **Make the smallest coherent change.** Include everything required for consistency, but exclude unrelated edits.
5. **Test the impact closure.** Run the checks required by the changed component and its downstream effects. Do not substitute a huge test suite for thinking about impact, but obey project-required full gates.
6. **No false green.** Do not describe a PR, release, capability, or merge as passing when required tests or CI are still failing, pending, or unknown.
7. **Resolve conflicts semantically.** `ours`, `theirs`, `current`, `incoming`, and `accept both` are implementation choices, not correctness criteria.
8. **Verify after merge.** Confirm the intended files, version, manifests, hashes, and branch state on the target branch after GitHub reports success.

## Phase 1 — Repository preflight

Before edits, establish a baseline equivalent to:

```text
repository root
current branch
HEAD commit
base/upstream branch
working-tree status
untracked files
recent relevant history
project rules
```

Prefer commands such as:

```bash
git rev-parse --show-toplevel
git status --short --branch
git rev-parse HEAD
git remote -v
git log --oneline --decorate -n 12
```

Read applicable repository guidance before changing files, especially when present:

```text
CLAUDE.md
AGENTS.md
CONTRIBUTING.md
README.md
VERSION
CHANGELOG.md
release/change-control documents
CI workflow files
state/ or manifest files
```

If the working tree contains unrelated modifications, preserve them. Work around them by limiting the edit scope or ask before using a destructive/isolation operation.

## Phase 2 — Define the change

State the change in terms of:

```text
component
before
intended after
reason
expected files
impact closure
tests/gates
version impact
promotion condition
```

For repositories with formal change control, update the project-native record rather than inventing a parallel format.

Do not bump versions automatically merely because a neighboring repository changed. Bump the repository whose public behavior, contract, packaging, or declared release state changed according to that repository's version policy.

## Phase 3 — Branch safely

Normally make changes on a dedicated branch based on the intended target branch.

Use readable names such as:

```text
feature/<topic>
fix/<topic>
docs/<topic>
chore/<topic>
release/<version-or-topic>
```

Before branching from `main`/`master`, ensure the baseline is the intended current base. Do not silently branch from a stale local base when current remote state matters.

If a user already has work on a branch, continue there unless creating a new branch is necessary and safe.

## Phase 4 — Implement without collateral edits

Change only the declared scope and consistency closure.

Examples of consistency closure:

- changing a root version may require README, changelog, installer/manifests and version-coherence tests;
- renaming a component may require active docs, CLI help, tests and machine-readable state;
- moving historical documentation should preserve its content and update indexes instead of deleting history;
- changing a public contract may require downstream compatibility documentation without changing downstream runtime behavior.

After editing, inspect the actual diff rather than trusting intent:

```bash
git diff --stat
git diff --check
git diff
```

Look for accidental generated files, duplicated headings, stale version strings, unresolved conflict markers and unrelated formatting churn.

## Phase 5 — Verification

Run cheap deterministic checks first, then the affected test closure, then required full/release gates.

Typical order:

```text
syntax / parse
static consistency
unit / targeted tests
integration / affected regressions
full required project gates
packaging/install roundtrip when relevant
```

For machine-readable files, parse them. For generated artifacts, compare hashes or regeneration output. For release/version changes, verify every declared version source agrees.

A failed gate blocks promotion until it is fixed, explicitly waived by project policy, or the change is rejected.

## Phase 6 — Commit discipline

Before committing, verify:

```text
only intended files are staged
no conflict markers remain
diff describes one coherent change
required tests have results
```

Prefer atomic commits with messages that describe the effect, for example:

```text
fix: keep installer version aligned with VERSION
feat: add reusable repository workflow skill
docs: reconcile OHF scope under Origami
release: Tlaloc 6.0.0-alpha.X <summary>
```

Do not amend or rewrite already-shared history unless that action is explicitly desired and safe.

## Phase 7 — Pull request

Before opening or updating a PR, compare the branch against its real base and inspect the changed-file set.

A useful PR body should communicate:

```text
why this change exists
what changed
what deliberately did not change
tests/evidence
risk or migration impact
version/release impact when applicable
```

Do not create a misleading PR whose changelog claims code that is not actually in the branch.

When a PR already exists, update the existing PR rather than creating a duplicate unless there is a clear reason to replace it.

## Conflict resolution

For every conflicted file, answer these questions before choosing a resolution button:

1. What valid information exists only on the base side?
2. What valid information exists only on the feature side?
3. Is one side obsolete because the project's authority changed?
4. Is the file historical, cumulative, generated, or authoritative-current?

Useful defaults:

- **current README/architecture authority:** prefer the side matching the current project model, then manually reintroduce still-valid detail;
- **CHANGELOG/history:** usually preserve both valid histories and reorder/deduplicate manually;
- **generated files:** regenerate from the source of truth instead of hand-merging when possible;
- **machine state/manifests:** reconcile schema and facts deliberately; never concatenate JSON/YAML fragments blindly.

Never choose **Accept both** merely because both sides contain useful text. It is correct only when the resulting combined structure is itself valid.

After resolving conflicts, re-read the whole affected file and rerun its gates.

## Phase 8 — CI and merge

A merge candidate requires:

```text
PR diff reviewed
mergeable state known
required CI complete and successful
no unresolved review/blocker
expected head commit unchanged
```

If CI fails, inspect the failing job and fix the cause with a new commit. Do not merge first and repair later unless the repository has an explicit emergency policy.

When tooling permits, guard the merge with the expected PR head SHA so a newly pushed commit cannot be merged without review.

Use the project's configured merge strategy. Do not impose squash/rebase/merge globally when repository policy already defines it.

## Phase 9 — Post-merge verification

After merge, verify the target branch directly:

```text
merge commit/result
VERSION or equivalent
critical changed files
machine-readable state
release/changelog entry
installer/package identity if affected
```

A successful GitHub merge response is not by itself proof that every release invariant is correct.

If another repository depends on the newly merged state, update it only after the upstream state exists on its authoritative branch.

## Multi-repository composition and snapshots

Keep independent repositories independently versioned.

For a tested composition:

```text
repo A exact commit
repo B exact commit
integration gates
      ↓
immutable snapshot / lock manifest
```

Submodules or exact Git pins are useful **inside an integration/snapshot builder** because they record exact component commits. Do not force application repositories to use submodules merely to consume tools when a simpler installed-tool + lockfile model is sufficient.

A snapshot is a distribution artifact, not a new source of truth. Fix bugs in the owning source repository, test the new composition, then publish a new snapshot.

Record both:

```text
compatible range   = combinations expected to work
tested exact set   = combination actually verified
```

Do not silently edit a published snapshot and keep the same identity.

## Release/version invariants

When a repository has a root `VERSION` or equivalent canonical version file:

- prefer it as the single release-version source;
- derive installers/build scripts from it when practical;
- add a regression gate against duplicated hard-coded versions;
- update changelog/change-control documentation in the same coherent release change;
- verify installed/package identity matches the declared release.

Independent repositories do not need synchronized version numbers.

## Stop conditions

Stop and ask or investigate further instead of guessing when:

- unrelated dirty work may be overwritten;
- the correct base branch is unclear;
- a conflict contains two incompatible but apparently valid authorities;
- required CI remains pending/failing;
- a release number or ownership boundary is ambiguous;
- an operation requires force push, destructive reset, cleaning untracked files, or deleting history not explicitly approved;
- a dependency/snapshot points at an unmerged or mutable state when immutability is required.

## Fast path

For a normal clean change, the workflow reduces to:

```text
inspect
→ branch
→ edit
→ diff
→ targeted tests
→ consistency gates
→ atomic commit
→ PR
→ inspect PR diff
→ wait for CI
→ merge
→ verify main
```

Use more ceremony only when the repository risk justifies it.
