# Manual external real-VLM evidence

This directory preserves real-model observations that were executed outside Tlaloc's reproducible alpha.21 endpoint runner and then supplied for analysis.

These records exist so useful empirical failures are not lost simply because the first observation happened manually.

## Evidence boundary

A manual external observation may be classified as `REAL_MODEL` for Learning Memory when the response genuinely came from a non-synthetic external model. However:

```text
MANUAL_EXTERNAL_REAL_MODEL != ALPHA21_SMOKE
MANUAL_EXTERNAL_REAL_MODEL != ALPHA21_EVIDENCE_PHASE
MANUAL_EXTERNAL_REAL_MODEL != CROSS_MODEL_EVIDENCE
MANUAL_EXTERNAL_REAL_MODEL != PROMOTION_EVIDENCE
```

Missing provenance must remain explicit. Never infer an exact model variant, endpoint, temperature or transport configuration that was not captured at test time.

Each observation should retain, when available:

```text
date
provider/model family
exact model ID or UNSPECIFIED
verbatim prompt
verbatim response
specimen ID + SHA-256
Origami/Tlaloc versions
manual layered assessment
failure frontier
proposed candidate mutation
retest requirements
```

The preferred lifecycle is:

```text
manual external observation
  -> immutable raw record
  -> benchmark-compatible learning record
  -> failure pattern
  -> CHANGE_ATTEMPT
  -> reproducible retest
  -> OUTCOME_LINK
```

Old failures are never rewritten after a correction succeeds.

## Current observations

### 2026-08-30 — DeepSeek / signal-chain-r0

Path:

```text
deepseek-signal-chain-r0-2026-08-30/
```

Observed frontier:

```text
BOOT/ROSETTA perception: strong
initial-state perception: strong
causal-rule recovery: failed
temporal simulation: failed
exact-honesty: mostly strong
```

Dominant failure code:

```text
TEMPORAL_RULE_AMBIGUOUS
```

Suggested target:

```text
TEMPORAL_GRAMMAR
```

Proposed first candidate:

```text
t2-temporal-grammar-visible-r1
TEMPORAL_STRUCTURE -> VISIBLE_RULE_MICROGRAMMAR_R1
```

See `EXPERIMENT.md`, `raw-response.md`, `campaign.json`, `result.json` and `change-proposal.json` for the complete evidence chain.
