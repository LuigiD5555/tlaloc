# Temporal Native Benchmark R0

This benchmark measures **where** an Origami temporal communication path succeeds or fails. It is not a visual-quality score and it never uses another LLM as judge.

## What the benchmark reads

The evaluator itself does **not** inspect the image pixels. The tested VLM receives the PNG. Tlaloc receives the resulting evidence record:

```text
SPECIMEN ID / SHA / image variant
+ condition used for the trial
+ question ID
+ verbatim model response
```

The benchmark therefore evaluates the model's interaction with the carrier without re-interpreting the image on the model's behalf.

## Evidence pipeline

```text
ORIGAMI PNG
    |
    v
TEST CONDITION
PNG_ONLY | R4_ASSISTED | DEGRADED_*
    |
    v
QUESTION
    |
    v
TARGET VLM
    |
    v
VERBATIM RESPONSE
    |
    v
DETERMINISTIC NORMALIZATION
    |
    +--> P  perception
    +--> R  protocol / ROSETTA
    +--> S  semantic graph
    +--> T  temporal reasoning
    +--> X  exactness / honesty
    |
    v
CONDITION COMPARISON
    |
    v
MODEL REPORT
```

Each stage has a different responsibility.

### Specimen

The specimen identifies exactly which PNG was shown. A campaign must record the carrier SHA-256, size, dimensions and image variant. This prevents results from different renders or degraded copies being silently mixed.

### Condition

The condition controls what help the model is allowed to receive.

- `NATIVE_PNG_ONLY`: PNG + one question only.
- `R4_ASSISTED`: Master Prompt R4 + the same PNG + the same question.
- `DEGRADED_NATIVE`: transformed PNG + one question.
- `DEGRADED_R4_ASSISTED`: transformed PNG + R4 + one question.

Matched trials must keep the question unchanged. The difference in score then measures assistance or image degradation instead of prompt drift.

### Questions

Questions isolate capabilities instead of asking one large ambiguous question.

- Q0, Q6: protocol / ROSETTA understanding.
- Q1, Q2: basic visual-semantic perception.
- Q3: graph/rule semantic recovery.
- Q4, Q5, Q7: temporal navigation and simulation.
- Q8: exactness boundary and `FALSE_EXACT=0`.

### Verbatim response

The raw target-model answer is preserved exactly. Normalization happens only in the evaluator after capture. This is necessary so the benchmark can be replayed when scoring rules improve.

### Deterministic normalizer

Normalization converts case, punctuation and common status spellings into a stable representation. It does not generate new meaning and does not call a model.

### Layer scorers

The layer scorers answer different diagnostic questions.

```text
P_PERCEPTION
Can the model see the declared cells and state labels?

R_PROTOCOL
Did it understand ROSETTA and that the temporal plane is a semantic/generative film rather than literal video?

S_SEMANTIC
Did it recover the declared dependency/transition semantics?

T_TEMPORAL
Can it locate checkpoints and apply only declared transition rules to derive state?

X_EXACTNESS
Does it refuse a hidden exact claim when no exact decoder was actually executed?
```

A model can therefore score high in perception but low in temporal reasoning, or understand the temporal machine while failing the exactness boundary. Those are different failures and must not be collapsed into one opaque number.

## How the parts collaborate

The benchmark is cumulative but not all-or-nothing:

```text
P enables R/S evidence to be interpreted
R tells the receiver how the carrier should be read
S reconstructs the graph/rule meaning
T uses S + declared temporal semantics to derive behavior
X limits what may be claimed beyond semantic evidence
```

The desired path is:

```text
SEE
 -> SELF-BOOTSTRAP
 -> RECOVER STRUCTURE
 -> APPLY DECLARED BEHAVIOR
 -> STOP AT THE EXACTNESS BOUNDARY
```

If P fails, later semantic failures may simply be perceptual. If P passes but R fails, the model saw marks but did not enter the protocol. If R and S pass but T fails, the representation was understood but not executable by that model. If P/R/S/T pass and X fails, the model understood the carrier but hallucinated exact mechanics.

## Cross-condition comparison

The same model should be tested on the same specimen/questions in at least native and R4-assisted conditions.

```text
assistance_gain = score(R4_ASSISTED) - score(NATIVE_PNG_ONLY)
```

A large positive assistance gain means the external handshake is still doing important work. A small gain with strong native scores is evidence that BOOT + ROSETTA are increasingly self-sufficient.

For degradation campaigns:

```text
degradation_delta = score(DEGRADED) - score(PRISTINE)
```

This locates the perception wall for resize, JPEG/WebP conversion, blur, noise, rotation or screenshot transport.

## Evidence classes

Synthetic fixtures validate the evaluator only.

Only verbatim outputs from independently executed target models count as real model evidence.

`COMPUTATIONAL_PASS != NATIVE_VLM_PASS`.
