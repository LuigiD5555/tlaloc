# Parrot Capability Lab — R0

Phase P of the Tlaloc reprioritisation: measure the real competence envelope of
Parrot (LFM2-VL 1.6B) before designing any Tlaloque coalition.

Full specification: [SPEC.md](SPEC.md). This file is the operator checklist.

## Execution order (mandatory — SPEC §3)

```
P-1 instrument validation  (scorer / paired stats / stage-freeze / generator — done in code)
 ↓
P0  end-to-end dataset frozen        datasets/end-to-end.jsonl
 ↓
P1  instruction cliff 1→5            datasets/instruction-cliff.jsonl   (deterministically generated)
 ↓
P2  individual capabilities (11)     datasets/singles/*.jsonl
 ↓
P3  interference pairs (~12)         datasets/interference.jsonl
 ↓
P4  real coalitions (2–3)            datasets/coalitions.jsonl
 ↓
P5  blackboard influence             datasets/blackboard.jsonl
 ↓
P6  competence envelope report       results/PARROT_COMPETENCE_ENVELOPE_R0.json
```

## Commands

```sh
LAB="go run ./cmd/tlaloc-parrot-capability-lab"
EXP=experiments/parrot-capability-r0

# 0. confirm the endpoint, fill MODEL.json runtime fields from the output
$LAB doctor --experiment $EXP

# 1. freeze prompt + model config, once, globally
$LAB freeze --experiment $EXP --scope global

# P0: build a pdfmemory store, then generate. On a catalog-structured source the
#     deterministic generators cover locate/synthesis well and entity/factual/
#     numeric thinly; hand-author the balance from the scaffold.
tlaloc-fold-bench build -pdf book.pdf -out /path/to/store
$LAB generate --experiment $EXP --stage end_to_end --store /path/to/store --pdf book.pdf
#   writes: end-to-end.jsonl (auto draft), end-to-end.draft.jsonl (same, keep as backup),
#           end-to-end.provenance.json, end-to-end.authoring-scaffold.md (+ rendered page images)
# → open end-to-end.authoring-scaffold.md, write the missing questions into
#   end-to-end.jsonl per datasets/SCHEMA.md (target 30: 6 per category, text+image each)
$LAB validate --experiment $EXP --stage end_to_end
$LAB freeze   --experiment $EXP --scope stage --stage end_to_end

# 2a. P1: generate the paired 40-scene x 5-depth dataset + images deterministically
$LAB generate --experiment $EXP --stage instruction_cliff --seed 42 --scenes 40

# 2b. freeze that stage's dataset (its hash is recorded and never rewritten)
$LAB freeze --experiment $EXP --scope stage --stage instruction_cliff

# 3. run the full 200 once, then the sentinel subset x3 for stability
$LAB run --experiment $EXP --stage instruction_cliff
$LAB run --experiment $EXP --stage instruction_cliff --sentinel-only --repetitions 3

# 4. aggregate -> results/instruction_cliff.json (cliff verdict via paired McNemar)
$LAB aggregate --experiment $EXP --stage instruction_cliff

# ... later stages author their own dataset, freeze --scope stage, run, aggregate ...

# 6. once every stage has results/, build the envelope
$LAB report --experiment $EXP
```

Add `--allow-unfrozen` (or `--dataset PATH`) to `run` only for throwaway smoke
checks — a real campaign run refuses unless the global config and the stage
dataset are frozen.

## Freeze model (P-1 fix #6)

`FREEZE.json` is the ledger. `--scope global` locks the prompt and MODEL.json
once. `--scope stage` locks one stage's dataset independently and **never
rewrites it** — so P3/P4 datasets can still be authored using what P1/P2
revealed, without ever un-freezing an earlier stage. A `run` verifies the
on-disk dataset still matches its recorded hash.

## No-leakage (SPEC §34, P-1 fix #8)

The runner sends Parrot only `PROMPT.txt` (system) + the case `instruction`
(user) + one image, via `target.OpenAICompat.CompletePerception`, which has no
channel for evaluator data. Each run record stores `system_prompt_hash`, the
exact `user_text` + its hash, and `image_hash`, so no-leakage is auditable from
the record alone.
