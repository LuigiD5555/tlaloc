# Change Control — Tlaloc 6.0.0-alpha.18

## Scope

Alpha.18 introduces **Closed Experimental Loop R0**. It operationalizes the alpha.17 Temporal Native Benchmark, Debug Trace, Learning Memory and Adaptive Search as one bounded experiment runner.

## New executable path

```text
baseline Origami PNG
  -> clean Native/R4 real-model trials
  -> deterministic benchmark
  -> targeted diagnostic retry for failed questions only
  -> persistent Learning Memory
  -> Adaptive Search plan
  -> prioritized candidate PNG bank
  -> CHANGE_ATTEMPT
  -> selected candidate trials
  -> candidate diagnostic retries
  -> persistent post-change observations
  -> OUTCOME_LINK(before, after, delta)
  -> next Adaptive Search plan
```

Managed CLI:

```text
tlaloc-closed-loop
```

Commands:

```text
example
validate
run
```

## Transport

R0 supports OpenAI-compatible multimodal endpoints, including a local LM Studio server.

Transport errors are isolated from model-semantic evidence:

```text
HTTP / timeout / malformed response != BOOT / ROSETTA / T2 / temporal failure
```

If no clean baseline trial completes, the run stops with `BASELINE_EXECUTION_UNAVAILABLE` instead of comparing candidates against a fabricated zero score.

## Diagnostics

Only failed clean question IDs are retried. Diagnostic retries use the existing observable `tlaloc.origami-debug-trace.r0` contract and remain excluded from primary Native/R4 scoring.

An incomplete diagnostic retry caused by transport failure is not admitted as diagnostic benchmark evidence.

## Memory and outcomes

Real successfully evaluated campaigns are persisted through `tlaloc.learning-memory.r0`.

Candidate experiments link:

```text
parent real failure observations
 -> CHANGE_ATTEMPT
 -> post-change real observations
 -> OUTCOME_LINK
```

Old failures remain immutable regression history after a candidate improves them.

## Adaptive search

Adaptive Search selects experiment order and budget only. Memory cannot add promotion score to a candidate.

```text
MEMORY PRIORITY != PROMOTION SCORE
```

Candidate IDs with existing outcome evidence are not automatically reused as new experiments. Materially new variants or explicit replications use a new candidate ID.

## Candidate PNG authority

Candidates may be pre-rendered or created through an explicit external `build_command`. The hook receives candidate ID, output path and mutation JSON.

Tlaloc invoking a renderer does not make Tlaloc the canonical Origami pixel authority.

## Verification

Required alpha.18 gates:

- Closed Experimental Loop R0 contract gate;
- deterministic fake OpenAI-compatible VLM end-to-end test;
- baseline Q3 failure -> targeted `T2_NOT_FOUND` diagnostic -> adaptive candidate -> positive linked outcome regression;
- HTTP 503 transport failure does not create semantic observations;
- managed install/uninstall lifecycle includes `tlaloc-closed-loop`;
- Learning Memory survives uninstall;
- `go test ./...`;
- `go vet ./...`;
- `go test -race ./...`;
- Gatekeeper and version coherence.

## Evidence status

Implementation tests prove the orchestration machinery. They do **not** prove Native temporal interoperability in real VLMs.

```text
CLOSED_LOOP_IMPLEMENTED = true
REAL_VLM_EVIDENCE = pending
```

## Authority

```text
Tlaloc executes experiments and recommends.
Origami owns canonical protocol/profile promotion.
Tonal may later pin a verified composition.
```
