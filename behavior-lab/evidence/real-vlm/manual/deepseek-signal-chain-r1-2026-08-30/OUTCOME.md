# DeepSeek manual retest outcome — temporal grammar candidate R1

Status: `REAL_MODEL_MANUAL_SINGLE_MODEL_IMPROVEMENT`

Promotion authority: `NONE`

This record is the post-change observation for candidate `t2-temporal-grammar-visible-r1`. It does not rewrite the original baseline failure. The original evidence remains under `deepseek-signal-chain-r0-2026-08-30/`.

## Provenance boundary

```text
Provider/model family: DeepSeek
Exact model variant: unspecified
Origami version: 6.0.0-alpha.15
Tlaloc version at analysis: 6.0.0-alpha.21
Condition: NATIVE_PNG_ONLY_MANUAL_MATERIALIZED_CANDIDATE
Candidate PNG SHA-256: 94a3e1d8f8e0a97e8daea267f28d1699241c9030b85f44a6e7d3b68aac02879c
Embedded TemporalProgram SHA-256: bc4a5c13a1a0f1707031d8e418fa0720f26605ab626c2210e99a2194f39eff02
```

The candidate image was a manual materialization of the merged Origami candidate geometry over the previously tested parent carrier. It is suitable as exploratory visual evidence, but this record does not claim byte-for-byte identity with an invocation of the Go builder on a Go-generated parent PNG.

## Before / after

| Layer | Baseline | R1 retest | Delta / interpretation |
|---|---:|---:|---|
| P_PERCEPTION | 0.90 | 0.95 | preserved / slightly improved |
| R_PROTOCOL | 0.80 | 0.90 | improved |
| S_SEMANTIC | 0.60 | 0.90 | major improvement; causal rules now visible |
| T_TEMPORAL | 0.30 | 0.725 | major improvement; rule recovery passes, execution still incomplete |
| X_EXACTNESS | 0.80 | 0.80 | preserved; no false exact decode |
| Overall diagnostic | 0.68 | 0.8333333333 | +0.1533333333 |

These are manual diagnostic scores using the same layered rubric as the baseline. They are not canonical alpha.21 Q0-Q8 evaluator output.

## What the mutation fixed

DeepSeek now explicitly recovered all four causal rules:

```text
A=ACTIVE => B: IDLE -> ACTIVE
B=ACTIVE => A: ACTIVE -> DONE
B=ACTIVE => C: IDLE -> ACTIVE
C=ACTIVE => B: ACTIVE -> DONE
```

It also recovered the crucial synchronous semantic statement that all rules for a step are evaluated against the same pre-step snapshot.

Therefore the baseline failure frontier:

```text
TEMPORAL_RULE_AMBIGUOUS
```

is considered **resolved for this single manual DeepSeek trial**.

## What remains wrong

DeepSeek did not actually execute the recovered rules until stability. It never produced the required sequence:

```text
t0: A=ACTIVE  B=IDLE    C=IDLE
t1: A=ACTIVE  B=ACTIVE  C=IDLE
t2: A=DONE    B=ACTIVE  C=ACTIVE
t3: A=DONE    B=DONE    C=ACTIVE
t4: stable
```

and therefore did not recover the final state:

```text
A=DONE  B=DONE  C=ACTIVE
```

It also failed to identify checkpoint times `t0,t2,t4` and introduced unsupported interpretations:

- a repeating cycle `A -> B -> C -> B -> A`;
- checkpoints as possible restart points;
- graph/rules as a virtual machine/interpreter for the payload;
- the exact payload as "not present" even though it is physically embedded but not visually decoded.

No fabricated SHA, CRC or exact payload content was claimed, so `FALSE_EXACT` remains zero.

## Updated learning frontier

The dominant failure has moved from **rule perception** to **rule execution**:

```text
BEFORE: T2_RULE_RECOVERY      -> TEMPORAL_RULE_AMBIGUOUS
AFTER:  T2_RULE_MICROGRAMMAR  -> TEMPORAL_EXECUTION_INCOMPLETE
```

This is important: the first correction worked on the dimension it targeted. The next candidate should not redesign the rule grammar again.

## Next hypothesis

Primary next target:

```text
TEMPORAL_EXECUTION_DIRECTIVE
```

Candidate hypothesis:

> Once the rules are readable, add one compact internal instruction that requires the receiver to apply the visible rules step-by-step from the declared initial state until no rule changes state, and then report the stable state. Keep rule text, payload and TemporalProgram unchanged.

Test this as a single mutation so its causal contribution remains measurable. A separate later candidate should label checkpoint ticks `t0,t2,t4`; do not combine that with the execution directive in the first retest.

## Promotion decision

```text
candidate improved same-model manual score: YES
rule-recovery failure resolved in this trial: YES
final-state execution recovered: NO
repeated trials: NO
cross-model evidence: NO
formal alpha.21 evidence phase: NO
promotion eligible: NO
```

The candidate is retained as a successful **learning step**, not as a canonical Origami promotion.
