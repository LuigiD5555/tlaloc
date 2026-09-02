# questionclass-charcnn-r0

First trained *text* specialist for the Tlaloque swarm — a character-level
CNN (7,721 parameters, trained from scratch, no pretrained backbone, no
tokenizer) that classifies a question's rhetorical shape:
`DEFINITION | COMPARISON | PROCESS | FACTUAL_DETAIL | GENERAL`.

It replaces the rule-based prefix matching in
`internal/foldtest/swarmask.classifyQuestion` at the `classifier` node of
the swarm DAG. That rule-based path stays as an honest fallback (see
"Integration" below).

## Why a char-CNN

The swarm asks questions in Spanish and English, with accents, typos and
varied phrasing. A character model has a ~60-symbol fixed vocabulary, no
out-of-vocabulary problem, and learns the actual cues (`what is`, `¿cómo`,
` vs `, a bare year, `page 12`) without a word tokenizer. The task is
shallow classification of a short string — two 1-D convolutions over
character embeddings are the right size.

## Pipeline (all under `tools/`)

- `questionclass_model.py` — `QuestionTypeClassifier`, `encode()` (shared
  preprocessing), `predict_text()`.
- `questionclass_dataset.py` — pure-Python synthetic generator. Labels are
  free: every question is built from a template whose class is known by
  construction. Bilingual templates × a swarm-domain topic vocabulary ×
  surface noise (casing, leading `¿`, filler prefixes, contractions).
  Disjoint seeds for train/val/test.
- `questionclass_train.py` — CPU Adam / cross-entropy, reports overall and
  per-class val accuracy.
- `questionclass_serve.py` — stdlib HTTP service speaking the unmodified
  `tlaloque.CapabilityRequest/CapabilityResponse` HTTP_JSON contract, same
  shape as `origami/tools/microisa_serve.py`.
- `../models/questionclass-charcnn-r0.pt` — checkpoint (~40 KB).

## Reproduce

```bash
cd tlaloc/behavior-lab/tools
python3 questionclass_dataset.py --seed 1001 --count 10000 --out /tmp/qclass-train.jsonl
python3 questionclass_dataset.py --seed 2002 --count 2000  --out /tmp/qclass-val.jsonl
python3 questionclass_train.py --train /tmp/qclass-train.jsonl --val /tmp/qclass-val.jsonl \
  --out ../models/questionclass-charcnn-r0.pt --epochs 25
python3 questionclass_serve.py --checkpoint ../models/questionclass-charcnn-r0.pt --addr :8792
```

## Results

- **Synthetic held-out** (same generator, different seed, n=2000): 100%
  overall, 100% per class.
- **Hand-written questions not from any template** (n=17, the real
  generalization check): 16/17. The one miss labels
  "What does this page discuss about autonomous systems…" as
  `FACTUAL_DETAIL` rather than `GENERAL` — defensible, the phrase
  "this page" is a locator cue.

**Load-bearing caveat** (same as microisa-cnn-r0): train / val / held-out
are all from the *same* synthetic generator. 100% synthetic accuracy
validates "learned these templates cleanly", not "robust to arbitrary
phrasing." The honest measure is the hand-written 16/17, and even that is a
small sample of questions written by the same person who wrote the
templates.

**Observed failure mode**: on constructions far from the training
templates the model stays *confident while wrong* — e.g. "How do ant
colonies allocate tasks without a leader?" → `COMPARISON` at 0.64 (should
be `PROCESS`). This is why the swarm integration gates the model at
confidence ≥ 0.70 and falls back to the deterministic prefix rules below
that; on that example the rules correctly return `PROCESS` from the
`how ` prefix.

**Observed bias**: any locator word (`page`, `figure`, `página`, …) pulls
the verdict toward `FACTUAL_DETAIL`. Here that is mostly desirable — it
routes "what's on page N" questions to the guidance that tells the parrot to
UNFOLD and verify, which is the behaviour the whole fold-bench effort is
trying to elicit.

## Integration

`internal/tlaloque/questionclass/` is the Go client (`Descriptor`,
`NewRegistry(endpoint)`, `Classify(ctx, registry, question)`), registering
the service as a `tlaloque.HTTPWorker` — same pattern as
`internal/tlaloque/microisadecoder`.

`swarmask.QuestionClassifierWorker` gained an optional `ModelRegistry`
field. With it set (via `swarm-ask -classifier-service <url>` /
`AskInput.ClassifierEndpoint`) the node uses the trained model when it is
reachable and confident, otherwise the rule-based `classifyQuestion`,
recording which path produced the verdict in the blackboard observation's
`provenance.method` (`charcnn-model` vs `rule-based`). With `ModelRegistry`
nil the node is exactly the previous rule-based classifier.

Verified end-to-end against real LM Studio: `swarm-ask` with
`-classifier-service` — 5/5 nodes, the `question_type` blackboard entry
carried `provenance.method = "charcnn-model"`, confidence 0.994, verdict
`COMPARISON`, and the parrot produced a comparison-structured answer that the
consolidator scored `grounded=true, 0.8`.

## Calibration / abstention (the real verdict on this model)

`tools/questionclass_ood.py` builds a genuinely out-of-distribution eval
set — indirect/embedded questions, imperatives, multi-clause, ES/EN
code-switching, dropped accents, and topics *outside* the training
vocabulary. `tools/questionclass_calibrate.py` measures the model on it and
emits `models/questionclass-charcnn-r0.calibration.json` in the
`tlaloc.calibration-profile.r0` schema
(`internal/tlaloque/calibration`).

**Result — the model does not generalize:**

| slice | accuracy | ECE | n |
|---|---|---|---|
| in-distribution (held-out, same generator) | 1.00 | 0.0002 | 2000 |
| **out-of-distribution** | **0.51** | **0.345** | 300 |

Even filtering to confidence ≥ 0.95, OOD covered accuracy is only ~0.55.
No confidence threshold reaches a trustworthy covered accuracy, so the
profile's `confidence_floor` is set to an unreachable `1.01`:
`calibration.Verdict` always returns `LOW_EVIDENCE` and
`AdmitAsActive()` refuses the model (OOD accuracy below the 0.80 floor;
floor not in (0,1)). Regression-guarded by
`internal/tlaloque/questionclass.TestShippedProfile_IsNotAdmissibleAsActive`.

**This is the point of the exercise.** The earlier "16/17 hand-written"
number was misleadingly good because those questions were written in a
style close to the templates. The char-CNN memorized template structure,
not question semantics. `minModelConfidence` (a raw-confidence threshold)
could never have caught this — only measuring held-out competence did.

**Wired in:** `swarmask.QuestionClassifierWorker` takes an optional
`*calibration.CalibrationProfile`. When set, `calibration.Verdict` decides
whether to trust each prediction; a non-`ANSWERED` verdict falls back to
the deterministic rules, recording `model_verdict` and
`model_rejected: calibration:<verdict>` in the blackboard observation's
provenance. CLI: `swarm-ask -classifier-service <url> -classifier-calibration
<profile.json>`. Verified end-to-end: the classifier node emitted
`method=rule-based, model_rejected=calibration:LOW_EVIDENCE,
model_verdict=PROCESS` — the model was consulted, overruled, and its
opinion kept for audit.

**Reproduce the profile:**
```bash
python3 questionclass_ood.py --seed 7007 --count 300 --out /tmp/qclass-ood.jsonl
python3 questionclass_calibrate.py --checkpoint ../models/questionclass-charcnn-r0.pt \
  --in-dist /tmp/qclass-test.jsonl --ood /tmp/qclass-ood.jsonl \
  --out ../models/questionclass-charcnn-r0.calibration.json
```

**Next for this model specifically:** to earn ACTIVE it needs training
data with far more surface variety (real question logs, paraphrase
augmentation, adversarial forms) and re-measurement showing OOD accuracy
≥ 0.80 with a reachable confidence floor. Until then it stays a
demonstration of the calibration layer, not a production specialist.
