# DeepSeek manual carrier experiment — 2026-08-30

Status: `REAL_MODEL_MANUAL_EXTERNAL_OBSERVATION`

Promotion authority: `NONE`

This experiment captures the first manually supplied external VLM response against the Origami alpha.15 `signal-chain-r0` temporal carrier. It is retained as learning/debug evidence and regression history. It is not equivalent to an alpha.21 `SMOKE` or `EVIDENCE` run because the exact DeepSeek model ID, endpoint metadata, canonical Q0-Q8 transport and repeated trials were not captured.

## Bound specimen

```text
Origami version: 6.0.0-alpha.15
Tlaloc version at analysis: 6.0.0-alpha.21
Program: signal-chain-r0
Carrier profile: origami.temporal-carrier.r0.profile-1
PNG size: 8192 bytes
Dimensions: 640x640
PNG SHA-256: ac40044663a4524f12097a691bff5d73b7a18fcbfb5eee4f07969ec66c43cd0e
External model family/provider: DeepSeek
Exact model variant: unspecified
Condition: NATIVE_PNG_ONLY_MANUAL
```

The verbatim response is in `raw-response.md`. `campaign.json` and `result.json` use the existing Temporal Native Benchmark / Learning Memory data structures so the observation can be ingested without creating a parallel evidence system.

## Ground truth

The canonical program starts as:

```text
t0: A=ACTIVE  B=IDLE    C=IDLE
```

Declared synchronous rules produce:

```text
t1: A=ACTIVE  B=ACTIVE  C=IDLE
t2: A=DONE    B=ACTIVE  C=ACTIVE
t3: A=DONE    B=DONE    C=ACTIVE
t4: no change -> stable
```

Declared checkpoints for the observed execution are `t0`, `t2`, `t4`.

Compact rule semantics:

```text
R1: A=ACTIVE                         => B: IDLE   -> ACTIVE
R2: B=ACTIVE                         => A: ACTIVE -> DONE
R3: B=ACTIVE                         => C: IDLE   -> ACTIVE
R4: C=ACTIVE                         => B: ACTIVE -> DONE
```

## What DeepSeek recovered

Strong recovery:

- recognized the carrier as a structured temporal/semantic diagram;
- recovered the visible Rosetta mappings `BOX=CELL`, `ARROW=TRANSITION`, `RING=CHECKPOINT`, `X=TIME`;
- identified cells A, B and C;
- recovered initial states `A=ACTIVE`, `B=IDLE`, `C=IDLE`;
- recognized the timeline/checkpoint region;
- recognized the exact payload declaration `ZLIB JSON + SHA256 + CRC`;
- did not fabricate a SHA-256 or claim exact mechanical decode.

Partial/failed recovery:

- did not recover the declared causal rules;
- did not recover graph edges as program semantics;
- did not recover checkpoint times `t0,t2,t4`;
- did not simulate the synchronous transition sequence;
- did not recover final states `A=DONE, B=DONE, C=ACTIVE`;
- guessed that A might become `IDLE`, which contradicts the program;
- described the exact payload as having "no raw data" even though the payload is physically present but not visually/mechanically decoded.

## Manual layered assessment

This is a diagnostic rubric, not the canonical Q0-Q8 evaluator output:

| Layer | Score | Assessment |
|---|---:|---|
| P_PERCEPTION | 0.90 | entities and initial states recovered |
| R_PROTOCOL | 0.80 | Rosetta and temporal-vs-video framing largely recovered |
| S_SEMANTIC | 0.60 | broad structure understood, causal rule semantics missing |
| T_TEMPORAL | 0.30 | temporal existence detected, actual rule execution not recovered |
| X_EXACTNESS | 0.80 | no false exact decode, but payload presence was mischaracterized |
| Mean diagnostic score | 0.68 | useful partial bootstrap, insufficient temporal semantics |

## Learning-memory failure frontier

The result deliberately records two temporal failures and one semantic-layout failure:

```text
TEMPORAL_ROUTE | TEMPORAL_RULE_AMBIGUOUS | T_TEMPORAL  x2
T2_SEMANTIC_PLANE | SEMANTIC_EVIDENCE_INSUFFICIENT | S_SEMANTIC x1
```

This makes the dominant next debug target:

```text
TEMPORAL_GRAMMAR
```

The experiment therefore teaches a specific lesson:

> The current carrier exposes enough visible information for BOOT/ROSETTA and initial-state perception, but not enough visible temporal grammar for a generic VLM to reconstruct the declared rule system and execute it correctly.

## Proposed correction — candidate `t2-temporal-grammar-visible-r1`

Primary hypothesis:

> Make rule preconditions and state transitions explicitly recoverable in the visible semantic plane, while leaving the exact payload and canonical TemporalProgram unchanged.

First candidate family should be a **single `TEMPORAL_STRUCTURE` mutation** so causal effect is measurable. Do not combine prompt/layout/checkpoint changes in the first comparison.

Suggested visible microgrammar:

```text
A.ACTIVE  => B.IDLE>ACTIVE
B.ACTIVE  => A.ACTIVE>DONE
B.ACTIVE  => C.IDLE>ACTIVE
C.ACTIVE  => B.ACTIVE>DONE
```

The final visual design does not need to be literal text. Tlaloc should search for a compact redundant representation that a VLM can infer reliably, but the candidate must make these four facts recoverable without requiring zlib/binary decode.

### Secondary candidates after the primary test

1. `TEMPORAL_STRUCTURE`: label checkpoint ticks explicitly as `t0`, `t2`, `t4` rather than only endpoints.
2. `LAYOUT`: visually separate topology edges from causal-rule dependency arrows.
3. `PROMPT`: add one Rosetta instruction defining rule notation and synchronous update semantics.
4. `REDUNDANCY`: repeat rule identity or state-change cues using shape/position, not only text.

These should be tested independently before combinations so Learning Memory can attribute improvement to a mutation family.

## Non-regression requirements

A corrected candidate must preserve at minimum:

```text
P_PERCEPTION >= baseline
R_PROTOCOL >= baseline
X_EXACTNESS >= baseline
FALSE_EXACT = 0
exact payload unchanged
TemporalProgram SHA-256 unchanged
carrier hard byte limit respected
```

The primary success criterion is a measurable increase in `T_TEMPORAL`, especially exact recovery of rules, checkpoints and final state.

## Next experiment

Retest the baseline and `t2-temporal-grammar-visible-r1` under the same DeepSeek environment if possible, then repeat on additional held-out VLMs.

Expected successful answer must recover:

```text
A,B,C
A=ACTIVE B=IDLE C=IDLE at t0
R1-R4 causal semantics
t0,t2,t4 checkpoints
final A=DONE B=DONE C=ACTIVE
no invented exact payload claim
```

Only after repeated and cross-model evidence should Tlaloc consider a promotion recommendation. This manual observation alone is not promotion evidence.

## Learning Memory ingestion

The bundle is intentionally compatible with the existing importer:

```bash
tlaloc-learning-memory ingest-benchmark \
  -campaign behavior-lab/evidence/real-vlm/manual/deepseek-signal-chain-r0-2026-08-30/campaign.json \
  -result behavior-lab/evidence/real-vlm/manual/deepseek-signal-chain-r0-2026-08-30/result.json \
  -origami-version 6.0.0-alpha.15 \
  -tlaloc-version 6.0.0-alpha.21 \
  -candidate-id origami-alpha15-temporal-baseline
```

The imported failed observations become immutable Learning Memory events. The proposed change can then be recorded with `record-change`, linking the resulting temporal failure event IDs as parents. A later retest must create an `OUTCOME_LINK` instead of overwriting this baseline failure.
