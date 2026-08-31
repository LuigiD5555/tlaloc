# Closed Experimental Loop R0

Tlaloc alpha.18 closes the Origami-facing experimental cycle without granting Tlaloc authority over canonical Origami releases.

```text
current experimental incumbent PNG
        -> clean Native/R4 trials
        -> deterministic benchmark
        -> retry failed questions only in diagnostic mode
        -> persistent learning memory
        -> active failure frontier
        -> adaptive candidate priority
        -> candidate PNG build/run
        -> before/after outcome link
        -> non-regression + minimum-improvement gate
        -> next experimental incumbent
        -> repeat
```

## Experimental incumbent

The incumbent is only a laboratory reference. It is not a canonical Origami profile and it does not update the Origami repository.

A candidate can advance only when:

- it has at least as many complete clean trials as the incumbent;
- it does not lower any benchmark question score;
- it does not increase missing questions;
- it does not increase invented exact claims;
- its selected outcome metric improves by at least `min_incumbent_improvement` (default `0.01`).

If multiple candidates pass, the highest outcome metric wins; ID is the deterministic tie-breaker.

## Active failure frontier

Old failures remain in memory as regression history, but they must not permanently dominate the current search target.

For each generation, the active frontier is calculated from the current incumbent's observations. Historical `CHANGE_ATTEMPT` and `OUTCOME_LINK` events remain available as bounded search signal.

This means Tlaloc can move from one bottleneck to the next:

```text
T2_NOT_FOUND
  -> candidate fixes T2
  -> candidate becomes experimental incumbent
  -> next run exposes TEMPORAL_RULE_AMBIGUOUS
  -> adaptive search changes focus to temporal grammar
```

## Candidate DAG

Candidates may set `parent_specimen_id`.

A parent-bound candidate is eligible only when that parent is the current experimental incumbent. This allows staged combinations such as:

```text
baseline
  -> t2-layout-fix
      -> temporal-grammar-fix
          -> checkpoint-routing-fix
```

Candidates with no `parent_specimen_id` remain general alternatives and may challenge any incumbent during the run.

## Candidate builders

A candidate can point to a prebuilt PNG or declare `build_command`. A builder receives:

```text
TLALOC_CANDIDATE_ID
TLALOC_OUTPUT_PNG
TLALOC_MUTATIONS_JSON
TLALOC_PARENT_SPECIMEN_ID
TLALOC_PARENT_PNG
```

Tlaloc invokes the builder but does not become Origami pixel authority. The produced PNG is still an experimental Origami artifact.

## Diagnostics

Clean Native evidence remains `PNG + question` only. Diagnostic retries are separate and run only for failed question IDs. Their debug traces never count toward self-bootstrap score.

## Memory and retesting

Persistent history changes search priority, not truth. A candidate tested in an older run is not permanently banned from future runs; only duplicate execution inside the current closed-loop run is suppressed.

## Stop conditions

The loop stops when one of these becomes true:

- the incumbent cannot produce clean trials;
- the incumbent has no active failures (unless continued exploration is explicitly enabled);
- no eligible candidate remains;
- the configured generation limit is reached.

## Authority boundary

```text
CLOSED_LOOP_EXPERIMENTAL_INCUMBENT != CANONICAL_ORIGAMI
MEMORY_GUIDES_SEARCH != PROMOTION_SCORE
TLALOC_RECOMMENDS
ORIGAMI_DECIDES
```

The final profile/release decision remains external to the closed loop and must pass the normal evidence and Origami-owned promotion gates.
