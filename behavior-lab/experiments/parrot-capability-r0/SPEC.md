# Parrot Capability Lab R0 — experiment specification

Model under test: **Parrot / LFM2-VL 1.6B**. Goal: discover Parrot's real
competence envelope and decide which work it keeps vs. which is externalised to
Tlaloques. Grounding R1 is **not** touched during Phase P.

## 1. Hypothesis

> A very small multimodal model can retain a reduced subset of useful
> capabilities if Tlaloc avoids loading it with mutually-interfering operations
> and progressively externalises the weak ones to tiny specialists.

## 2. Primary experimental question

> What is the maximum amount of simultaneous cognitive work we can assign to
> Parrot before its performance drops significantly?

Secondary: which of its capabilities are the priority externalisation candidates?

## P-1. Instrument validation (inserted before P0)

Before trusting any number, the measuring instrument is fixed and proven:

1. **Correct answer vs. answer universe are separate** — `expected.value` /
   `expected.aliases` (correct answer + its spellings) vs. `choices` (the
   universe). Never conflated; validator enforces `expected.value ∈ choices`.
2. **p95 latency** uses `sorted[int((n-1)*0.95)]`, not a wrapping index.
3. **The cliff is decided by a paired (McNemar exact) test**, not by
   non-overlapping Wilson intervals — OP1..OP5 share the same 40 stimuli.
   Wilson per level stays descriptive.
4. **Operation type is counterbalanced against operation count** — the
   generator rotates which of 5 primitives lands on the final (scored) step
   via a 5×5 Latin square, and records `added_primitive`; aggregation reports
   a per-primitive breakdown and a confound note.
5. **"Wrong" ≠ "hallucinated"** — score carries `semantic_correct`,
   `format_valid`, `contract_success`, `abstained`, `unsupported_assertion`.
6. **Stage-level freeze** — prompt/model frozen once globally; each stage
   dataset frozen separately and never rewritten (`FREEZE.json`).
7. **Uninstrumented cost is null, not 0** — `ram_peak_mb: null`,
   `ram_measured: false`. Real instrumentation is Phase S.
8. **The exact model input is recorded** — `system_prompt_hash`, `user_text`
   + hash, `image_hash` per run record.
9. **P0 and P1 datasets are generated, not hand-written.** P1: fully
   deterministic scene generator + renderer + counterbalanced pipeline
   (`generate --stage instruction_cliff`). P0: stratified page selection +
   ground truth from the Origami source, with a human spot-check.

## 3. Mandatory order

`P0 dataset → P1 instruction cliff → P2 singles → P3 interference pairs →
P4 real coalitions → P5 blackboard influence → P6 report.`

No exhaustive triple sweep. No 55 pairs (only the ~12 that co-occur in an Origami
flow). No new Tlaloques except instrumentation.

## 4. P0 — frozen end-to-end dataset (generated)

One complete document via a built `pdfmemory` store. R0: **10 distinct pages, 30
questions**, 6 locate / 6 entity-or-concept / 6 factual / 6 numeric / 6 synthesis;
**each question emitted as a text variant and an image variant** (so P0 itself
measures what Parrot loses without the visual channel). Pages picked
deterministically with `foldtest.SelectSpacedPages` (skips cover / TOC / index /
references / near-empty). Every question keeps its page, `ohf://` address, page
CID, exact evidence fragment + sha256, and generator method in
`end-to-end.provenance.json`. `ValidateEndToEnd` gates: answer demonstrable from
its own evidence, address + evidence present, no answer leak, no duplicate, no
ambiguous cloze. Human reviews the 30; bad ones regenerated with a new seed. Not
used to train capabilities — it is the common system evaluation reused by T, S, O
(§35). Frozen via `freeze --scope stage --stage end_to_end`.

## 5–14. P1 — Instruction Cliff (priority experiment)

40 base cases × 5 operation depths = 200 prompts. Paired design: OP1..OP5 share
the same stimulus wherever possible, so the drop is attributable to operational
depth, not content. Scene families: 10 visual-simple / 10 visual+text /
10 extraction-comparison / 10 visual+reference-selection. No complex Origami
imagery yet — measure the model first.

Variability: run the 200 once, then a 10-case sentinel subset (`"sentinel": true`)
× 5 levels × 3 repetitions = 150 extra inferences to check stability. If
temperature 0 is byte-identical, full repetition is unnecessary.

Per level: n, correct, accuracy, **Wilson 95% CI**, abstention, format_failure,
hallucination, mean/p95 latency, tokens_out, `accuracy_delta_from_OP1`.

**Cliff definition (frozen, computed not eyeballed):** first level N where the
paired accuracy delta vs. N−1 is **≤ −15 percentage points** **and** the paired
McNemar exact test is significant (p < 0.05, more regressions than gains). Full
curve + all paired transitions kept even when no cliff exists.

Operative output: **two** limits (§8), from paired tests on `contract_success`
and on `semantic_correct` separately —
`PARROT_MAX_SAFE_OPS_CONTRACT` and `PARROT_MAX_SAFE_OPS_SEMANTIC`. Semantic >
contract ⇒ "Parrot knows the answer but stops emitting it in the requested form"
⇒ externalise format/control, not decompose. Neither is a hard capability claim.

## 15–21. P2 — Individual capabilities

11 capabilities: `VISUAL_IDENTIFY VISUAL_LOCATE READ_SHORT_TEXT EXTRACT_ENTITY
EXTRACT_NUMBER CLASSIFY_SIMPLE COMPARE_SIMPLE FOLLOW_REFERENCE SELECT_ACTION
USE_BLACKBOARD_HINT ANSWER_FROM_EVIDENCE`.

Not measured in R0: general reasoning, encyclopaedic knowledge, creativity,
coding, long maths, autonomous agency.

Dataset size: n=50 for substitution-deciding capabilities (`VISUAL_LOCATE
READ_SHORT_TEXT EXTRACT_ENTITY EXTRACT_NUMBER SELECT_ACTION USE_BLACKBOARD_HINT
ANSWER_FROM_EVIDENCE`); n=30 for the rest, widened to 50 if near a threshold.

Every accuracy reported with a **Wilson** 95% interval.

**Classification (experimental, frozen thresholds — lower/upper are CI bounds):**

| class    | rule                    |
|----------|-------------------------|
| STRONG   | lower CI ≥ 0.85         |
| USABLE   | lower CI ≥ 0.70         |
| FRAGILE  | CI crosses 0.70         |
| WEAK     | upper CI < 0.70         |
| UNUSABLE | upper CI < 0.50         |

`EXTERNALIZE_CANDIDATE = true` when any of: weak accuracy, high interference,
high latency, high format-failure, low blackboard use, obvious deterministic
alternative exists.

## 22–24. P3 — Interference

~12 co-occurring pairs only, list frozen before running. For pair (A,B):
`interference_A = combined_A − single_A`; `PAIR_INTERFERENCE = mean(iA, iB)`.

| category | PAIR_INTERFERENCE |
|----------|-------------------|
| NEUTRAL  | > −0.05           |
| MILD     | −0.05 … −0.10     |
| MODERATE | −0.10 … −0.20     |
| SEVERE   | < −0.20           |

Special attention: vision+generation, routing+generation, extraction+generation.

## 25–26. P4 — Real coalitions

2–3 deployable configurations only (e.g. `V+E+G`, `V+R+E+G`, `R+E+B+G`). Answers:
even when isolated capabilities are acceptable, can Parrot sustain the combination
we actually want to assign? Produces the first role candidate.

## 27–28. P5 — Blackboard influence

Existing infra only — no Blackboard Consolidator yet. Per case, 4 conditions:
correct hint / no hint / incorrect hint / random hint. ≥ 20 cases per condition.
Metrics: answer_accuracy, grounding_accuracy, fact_recall, hint_follow_rate,
**wrong_hint_acceptance_rate** (the critical one — if Parrot obeys anything in the
blackboard, the blackboard is not yet trustworthy evidence).

## 29–34. Runner

Campaign layer only, reusing `internal/target` (`OpenAICompat.CompletePerception`),
`internal/tlaloque` accounting, `internal/tlaloque/calibration` (Wilson). New:
`internal/parrotlab` + `cmd/tlaloc-parrot-capability-lab`. Responsibilities: load
dataset, load experiment config, run cases, call Parrot, collect accounting,
score, write JSONL runs, produce aggregate JSON. Nothing else. Run/aggregate JSON
shapes: SPEC §31–32 (mirrored by `parrotlab` structs).

## 35–36. Central metric for Phase T (frozen NOW)

Using the P0 frozen end-to-end dataset. X = number of capabilities removed from
Parrot; Y = end-to-end accuracy. Thesis gains evidence iff a systematic
improvement appears. Flat ⇒ orchestration adds nothing. Falling ⇒ orchestration
hurts. **This metric is not changed after seeing results.** Phase T always
reports both local capability accuracy and end-to-end system accuracy.

## 38. Explicitly OUT of Phase P

Grounding R1, new Blackboard Consolidator, Tlaloque package manager, full ONNX
abstraction, llama.cpp adapter, automatic tool generation, 20-capability
recombination test, full Origami seed reduction, swarm concurrency tuning.

## 39. Completion gates

frozen end-to-end dataset · frozen R0 prompt · instruction cliff measured ·
11 singles measured · Wilson CI reported · ~12 relevant pairs measured ·
2–3 real coalitions measured · blackboard influence measured · resource
accounting captured · competence envelope written · externalisation candidates
defined.

## 40. Primary product

`results/PARROT_COMPETENCE_ENVELOPE_R0.json` — not code. Fields:
`model, safe_instruction_depth, strong[], usable[], fragile[], weak[], unusable[],
severe_interference[][], externalize_first[]`.

## 43. Guiding principle

Do not try to make Parrot smarter. Find the lowest cognitive load under which
Parrot delivers maximum value to the system, then build Tlaloc around that
frontier.
