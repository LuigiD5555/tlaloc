# Experimental Spine R0

Status: experimental infrastructure. This is deliberately small.

## Goal

Make prototype iteration cheap while making results cumulative.

Every prototype may keep its own native/raw evidence format. The common path is only:

```text
prototype-native raw records
        -> thin adapter
        -> Episode[]
        -> experience/manifest.json
        -> experience/episodes/...
        -> experience/summary.json
        -> next iteration/debug target
```

The spine does **not** replace experiment-specific evidence, does not promote candidates, does not call an LLM-as-judge and does not modify a prototype automatically.

## Ownership and public surface

TLALOC owns this development/learning projection. Target projects such as Origami keep their own semantic authority and native evidence. TONAL may emit execution traces, but those traces are adapted rather than redefined here.

Downstream development systems should consume the small public package:

```text
tlaloc.local/behaviorlab/prototypelab
```

It exposes Episode, RunManifest, Summary and `WriteBundle` without exposing any `behavior-lab/internal/*` package. T1-specific adapters remain internal because T1 raw records are Tlaloc experiment machinery.

## Bundle

A completed prototype run can write:

```text
<run>/
  ... native/raw experiment artifacts ...
  experience/
    manifest.json
    episodes/
      YYYY-MM/
        <episode>.json
    summary.json
```

The `experience/` directory is immutable: an existing bundle is never overwritten. Bundle publication is staged in a temporary directory and renamed into place only after manifest, episodes and summary have all been written.

### `manifest.json`

Records why the run exists and which prototype/config produced it:

- run ID and optional parent run ID;
- source experiment;
- prototype ID/version/parent version;
- hypothesis and change summary when explicitly known;
- exact repository revisions when explicitly supplied;
- model requested/reported/endpoint when known;
- config hash;
- run/evidence class;
- start/finish timestamps.

Unknown provenance stays empty. The spine never invents it.

### Episode

`tlaloc.episode.v1` remains a common projection, not the raw scientific source of truth. It now preserves:

- run/arm/family provenance;
- semantic and exact correctness;
- terminal/contract status and failure root cause;
- per-step request index, capability, operation, executor and model;
- input artifact/hash;
- raw and parsed model output;
- transport/schema/contract/general step status;
- latency;
- explicit HTTP-attempt, completion and failure accounting when the source experiment can actually observe those quantities.

No private model chain-of-thought is requested or stored.

### `summary.json`

The summary is deterministic from the manifest + Episodes. It includes:

- run/prototype lineage;
- episode success/failure and semantic/exact accuracy;
- HTTP attempts, valid completions and failure classes when available;
- blocked dependencies;
- p50/p95/max step latency;
- failure-root counts;
- breakdown by arm, family and capability;
- direct failed steps separately from dependency-blocked steps;
- most failed capability;
- a deterministic `next_debug_target`.

Dependency-blocked fan-out never wins the capability-failure vote: a downstream node blocked by an upstream error is recorded as `blocked_steps`, not treated as a new root failure.

`next_debug_target` is a debugging priority, **not** evidence of causality and not a promotion decision.

## Generic downstream use

A prototype outside Tlaloc can write its bundle through the public package:

```go
paths, err := prototypelab.WriteBundle(outDir, manifest, episodes, observedAt)
```

The prototype remains responsible for adapting its native trace into Episodes and for supplying correctness judgments. In particular, execution completion must not silently become semantic correctness.

## T1 integration

T1 native formats remain unchanged:

```text
workflow_records.json
node_call_records.json
run_accounting.json
```

The adapter preserves the rich fields from `NodeCallRecord` instead of throwing them away. Episode HTTP accounting mirrors `tonalt1arms.deriveAccounting` exactly.

Before a T1 experience bundle is written, the spine verifies that Episode-derived dynamic accounting equals the frozen raw `RunAccounting` for:

```text
HTTPRequestAttempts
ValidCompletions
TransportFailures
SchemaFailures
ModelContractFailures
BlockedByDependency
```

The intended primary persistence call is:

```go
experimentalspine.FreezePrimaryT1Run(outDir, manifest, result, observedAt)
```

It performs:

```text
RunResult.Freeze(raw T1 records)
        -> WriteT1Bundle
        -> experience/...
```

If the projection fails, the raw T1 freeze remains untouched so the experience view can be repaired/backfilled without repeating model calls.

### Zero-call backfill

An already-frozen primary run can be projected later with no model calls:

```bash
tlaloc-tonalt1-arms experience \
  -raw tonalt1-arms-run1 \
  -observed-at 2026-09-04T20:00:00Z
```

Optional `-manifest` supplies richer provenance. Without it, the command derives only provenance actually present in the frozen T1 records and leaves unknown repository/timestamp/hypothesis fields empty.

## Development cadence

Do not turn the spine into another long verification project.

Use four practical levels:

```text
DEV
  package-level tests for touched code; seconds

SMOKE REAL
  1-3 real tasks; find out whether the idea breathes

BUILD-TO-LEARN
  ~10, then +20, then 30-50 useful cases; find signal and failure structure

FREEZE / PROMOTION
  full suites, race checks, hashes, doctors, cross-repo verification and formal evidence
```

Heavy promotion gates remain valuable, but they belong at the freeze/promotion boundary rather than every micro-iteration.

## Hard boundaries

```text
RAW EXPERIMENT EVIDENCE REMAINS AUTHORITATIVE
EPISODE IS A COMMON PROJECTION
EXPERIENCE BUNDLES ARE IMMUTABLE
UNKNOWN PROVENANCE IS NOT INVENTED
HTTP ATTEMPTS != SUCCESSFUL COMPLETIONS
EXECUTION SUCCESS != SEMANTIC CORRECTNESS
BLOCKED FANOUT != NEW ROOT FAILURE
MEMORY/DEBUG PRIORITY != PROMOTION SCORE
NO AUTOMATIC SELF-MODIFICATION IN R0
NO LLM-AS-JUDGE IN THE SUMMARY PATH
```
