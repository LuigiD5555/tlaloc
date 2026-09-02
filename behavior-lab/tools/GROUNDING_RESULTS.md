# groundingscore-distilled-r0

Fourth trained specialist for the Tlaloque swarm — a tiny MLP (638
parameters, trained from scratch) that scores how well an answer is
supported by a page passage, from cheap features over **frozen MiniLM-L6
embeddings** plus lexical statistics. No transformer of its own, no
tokenizer.

It gives the swarm `consolidate` node a grounding judge that is **not the
parrot** (`lfm2-vl-1.6b`) that wrote the answer — removing the "model grading
its own homework" caveat that the earlier answerscore-only consolidator had.
The heavier `answerscore` chain (chat judge → embedding → keyword) stays as
the fallback.

## Why a distilled MLP on frozen embeddings

The consolidator runs on every answer, so the judge must be cheap. The
semantic content is already carried by MiniLM-L6 (a real 22M-param
BERT-family model, resident in LM Studio); the head only has to learn how to
weigh ~12 numbers derived from it. It is also the shape the user asked for:
"modelos diminutos … Tlaloc los entrena … un reloj suizo con muchos engranes".

## The 12 features (`tools/grounding_common.py:extract_features`)

Cosines: answer↔passage, question↔passage, question↔answer. Per-answer-
sentence best-passage-chunk support (mean / min / max). Content-word overlap
(answer→passage, question→answer). Verbatim-trigram fraction. Length ratio.
Answer type-token ratio. Answer **function-word fraction** — real prose
carries ~30-45%; a raw keyword dump carries ~0%. That last one is what lets
the head reject the keyword-dump attack that raw cosine falls for (see the
MiniLM finding in `internal/tlaloque/answerscore/embedding_score.go`).

## Pipeline (all under `tools/`)

- `grounding_common.py` — `Embedder` (batched LM Studio `/v1/embeddings`),
  `extract_features()`, the answer-perturbation strategies.
- `grounding_dataset.py` — builds `(question, answer, passage)` triples from
  real fold-bench pages (`tlaloc-fold-bench dump-pages`) × 6 strategies, each
  **labelled by construction** (`STRATEGY_LABEL`): the strategy that built
  the answer fixes its grounding level. Same "labels are free" principle as
  `questionclass_dataset.py`. An optional LM Studio judge (`--judge-model`)
  can be blended in, off by default — the small local judges tried
  (`h2o-danube3.1-4b-chat`, `dolphin-2.9.1-phi-3-kensho-4.5b`) collapsed to a
  near-constant score and `google/gemma-3-4b` / `minicpm3-4b-i1` /
  `phi-4-mini-reasoning` would not load. Distillation from a genuinely strong
  judge remains the r1 goal.
- `grounding_model.py` — `GroundingScorer` (638 params), outputs `score` and
  a self-estimated `confidence` (trained to predict `1 - |score - target|`).
- `grounding_train.py` — CPU Adam, MSE on score + 0.3·MSE on confidence.
- `grounding_ood.py` — 24 hand-authored triples, passages NOT from the book,
  answers NOT from the mechanical perturbations. The real test.
- `grounding_calibrate.py` — emits `models/groundingscore-distilled-r0.calibration.json`
  in the `tlaloc.calibration-profile.r0` schema.
- `grounding_serve.py` — stdlib HTTP service, unmodified
  `tlaloque.CapabilityRequest/CapabilityResponse` contract, port `:8793`.

## Reproduce

```bash
cd tlaloc/behavior-lab
go build -o /tmp/tlaloc-fold-bench ./cmd/tlaloc-fold-bench
/tmp/tlaloc-fold-bench dump-pages -store /tmp/foldstore-swarms -stride 9 -limit 44 > /tmp/gpages.jsonl
cd tools
python3 grounding_dataset.py --pages /tmp/gpages.jsonl \
  --train-out /tmp/grounding-train.jsonl --val-out /tmp/grounding-val.jsonl
python3 grounding_train.py --train /tmp/grounding-train.jsonl --val /tmp/grounding-val.jsonl \
  --out ../models/groundingscore-distilled-r0.pt \
  --metrics-out ../models/groundingscore-distilled-r0.metrics.json --epochs 140
python3 grounding_ood.py --out /tmp/grounding-ood.jsonl
python3 grounding_calibrate.py --checkpoint ../models/groundingscore-distilled-r0.pt \
  --in-dist /tmp/grounding-val.jsonl --ood /tmp/grounding-ood.jsonl \
  --out ../models/groundingscore-distilled-r0.calibration.json
python3 grounding_serve.py --checkpoint ../models/groundingscore-distilled-r0.pt --addr :8793
```

## Results

| slice | within-±0.2 acc | MAE | Pearson r | ECE | n |
|---|---|---|---|---|---|
| in-distribution val (same generator) | 0.89 | 0.069 | 0.95 | 0.07 | 126 |
| **out-of-distribution** (hand-authored) | **0.46** | ~0.24 | — | 0.51 | 24 |

**What it does well.** Keyword dumps, off-topic answers and generic filler
are reliably scored low (0.05–0.16 on the OOD cases). This is the one thing
raw MiniLM cosine gets wrong, and the reason the scorer exists as a distinct
signal.

**What it does not do (load-bearing caveats).**

1. **Under-scores genuine terse paraphrases.** "What makes a swarm robust?"
   → answer "redundancy, so losing a few agents doesn't change the outcome"
   (a correct paraphrase) scores ~0.37, not ~0.9. The `grounded` training
   answers were long, near-verbatim passage sentences, so the head correlated
   grounding with length and lexical overlap. `terse_grounded` examples
   helped only marginally.
2. **Blind to contradiction.** An on-topic answer that states the *opposite*
   of a passage fact ("foragers deposit pheromone only toward the food") is
   scored high — the feature set has no entailment signal. A `contradiction`
   training strategy was tried and **dropped**: its examples were
   indistinguishable from `grounded` in feature space and only added label
   noise that dragged every prediction down. Catching this needs an NLI
   feature or a cross-encoder — the r1 direction.
3. **Confidence is uninformative on OOD** (~0.95 flat regardless of error),
   so no threshold yields a trustworthy covered accuracy. The profile's
   `confidence_floor` is therefore an unreachable `1.01`.

**Same shape as `questionclass-charcnn-r0` and the MiniLM finding:** the
model learns the synthetic generator cleanly (Pearson 0.95 in-dist) and does
not generalize to arbitrary phrasing. The synthetic val number means "learned
the six strategies", not "judges grounding".

## Integration

`internal/tlaloque/groundingscore/` is the Go client (`Descriptor`,
`NewRegistry(endpoint)`, `Score(ctx, registry, Input)`, `LoadProfile`),
registering the service as a `tlaloque.HTTPWorker` — same pattern as
`internal/tlaloque/questionclass` and `internal/tlaloque/microisadecoder`.
`Input`/`Output` field names match `answerscore.ScoreInput`/`ScoreOutput`.

`swarmask.ConsolidatorWorker` gained optional `GroundingRegistry` and
`GroundingProfile` fields (`swarmask.RegistryConfig`, CLI `swarm-ask
-grounding-service <url> [-grounding-calibration <profile.json>]`):

- **no grounding service** → unchanged: `answerscore.ScoreAnswer` (chat judge
  → embedding → keyword).
- **grounding service, no profile** → distilled score used directly (an
  independent, cheap first pass; opt-in to its known weaknesses).
- **grounding service + profile** → `calibration.Verdict` gates it; with the
  shipped profile (`confidence_floor = 1.01`) the verdict is always an
  abstention, so the consolidator falls back to `answerscore` — the profile
  is doing its job.

The consolidation output carries `scored_by` and a `judge_independent` flag
(true for the distilled worker or answerscore's embedding/keyword workers,
false when it fell back to the chat judge on the parrot's own model). The
blackboard observation's `provenance.method` records `groundingscore-distilled`
vs `answerscore`.

## Next (r1)

Real distillation from a strong judge (a loadable 7B+ or a hosted model),
an NLI / cross-encoder feature to catch contradiction, and `grounded`
training answers that are short paraphrases rather than reworded passage
sentences. Re-measure OOD; target within-±0.2 ≥ 0.8 with a reachable
confidence floor before this earns authority over `answerscore`.
