# Adaptive Search R0

Adaptive Search closes the loop between persistent Tlaloc learning memory and experimental candidate selection.

It does **not** change the evidence gates used to recommend an Origami candidate. Memory decides where to spend experiment budget; held-out evidence still decides whether a candidate survives the normal tournament.

```text
real model trials
  -> temporal/native benchmark
  -> observable debug frontier
  -> learning-memory evidence ledger
  -> real failure patterns
  -> adaptive search plan
  -> prioritized experimental mutation families
  -> candidate queue
  -> real trials
  -> ordinary evidence-gated visual tournament
  -> outcome linked back to memory
```

## Why this exists

Without memory guidance, a search can waste equal effort on temporal structure when the dominant failure is still `ROSETTA -> T2`, or repeatedly retry a mutation family that historical evidence showed to be harmful.

Adaptive Search uses durable evidence to focus the next experiment while preserving exploration.

Example:

```text
memory:
  Q3 -> ROSETTA -> T2_NOT_FOUND
  Q4 -> ROSETTA -> T2_NOT_FOUND
  Q7 -> ROSETTA -> T2_NOT_FOUND

next_debug_target:
  T2_NAVIGATION

priority:
  LAYOUT
  PROMPT
  REDUNDANCY
  CHANNEL_ROLE

lower priority, still non-zero:
  TEMPORAL_STRUCTURE
  DEPTH_STRUCTURE
  ...
```

## Evidence hierarchy

`REAL_MODEL` failures drive adaptive focus.

`SYNTHETIC` evidence may test the evaluator, planner or storage implementation but must not become an empirical search target.

Historical outcomes may adjust priorities only within a bounded range. They may not override the current real-model failure frontier.

## Exploration floor

Every supported mutation family keeps a non-zero exploration probability/weight. This avoids permanent lock-in to whichever family happened to work first.

The reference implementation mixes a focused distribution with a fixed exploration mass before normalizing priorities.

## Candidate queue vs final tournament

Adaptive Search produces **pre-evidence priority**.

It answers:

> Which candidate should receive real-model trial budget first?

It does not answer:

> Which candidate should become canonical Origami?

The latter remains the responsibility of the existing evidence-gated tournament and ultimately Origami itself.

```text
memory priority != promotion score
```

## Traceability

When `tlaloc-adaptive-search prioritize -record-attempts` is used, each selected candidate becomes a `CHANGE_ATTEMPT` memory event that references the real failure-event IDs responsible for the plan.

Tags include:

```text
adaptive-search
target:<FAILURE_TARGET>
mutation:<MUTATION_KIND>
```

Later `OUTCOME_LINK` events can connect that attempt to post-change evidence and record whether the candidate helped, hurt or moved the failure frontier.

## CLI

Inspect the current plan:

```bash
tlaloc-adaptive-search plan
```

Prioritize a set of already-declared experimental candidates:

```bash
tlaloc-adaptive-search prioritize \
  -in candidates.json \
  -limit 4
```

Persist those selected experiments into learning memory:

```bash
tlaloc-adaptive-search prioritize \
  -in candidates.json \
  -limit 4 \
  -record-attempts
```

The input schema is:

```json
{
  "schema": "tlaloc.adaptive-search.r0.request",
  "candidates": [
    {
      "schema": "tlaloc.origami-visual-search.r0.candidate",
      "id": "candidate-t2-anchor",
      "base_profile_id": "origami.fixed-carrier.r2.profile-3",
      "mutations": [
        {
          "kind": "LAYOUT",
          "target": "T1_TO_T2_ENTRY_ROUTE",
          "value": "EXPLICIT_DIRECTIONAL_ANCHOR",
          "experimental": true
        }
      ]
    }
  ]
}
```

## Hard boundaries

```text
REAL_MODEL_FAILURES_DRIVE_ADAPTIVE_FOCUS
SYNTHETIC_EVIDENCE != EMPIRICAL_SEARCH_TARGET
MEMORY_GUIDES_EXPERIMENT_BUDGET_NOT_PROMOTION_SCORE
FINAL_TOURNAMENT_REMAINS_EVIDENCE_GATED
EXPLORATION_FLOOR > 0
HISTORICAL_OUTCOME_SIGNAL_IS_BOUNDED
RECORDED_ATTEMPTS_REFERENCE_PARENT_EVIDENCE
MEMORY != CANONICAL ORIGAMI TRUTH
TLALOC RECOMMENDS; ORIGAMI DECIDES
FALSE_EXACT = 0
```
