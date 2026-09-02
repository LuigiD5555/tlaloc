# Dataset schema — parrot-capability-r0

One JSON object per line (JSONL). One `Case` struct, shared by every stage;
`stage` selects how it is scored and aggregated.

```jsonc
{
  "case_id": "cliff-017-op3",          // unique within the file
  "stage": "instruction_cliff",        // end_to_end | instruction_cliff | singles | interference | coalitions | blackboard
  "capabilities": ["VISUAL_LOCATE", "COMPARE_SIMPLE", "FOLLOW_REFERENCE"],
  "operations": 3,                      // instruction_cliff only: 1..5
  "base_id": "cliff-017",              // instruction_cliff: shared by the 5 depths of one stimulus
  "sentinel": false,                   // instruction_cliff: part of the repeat-3x stability subset
  "task_family": "choice",             // choice | exact | entity | numeric | abstain
  "added_primitive": "compare",        // instruction_cliff: the operation this depth adds vs. the previous
  "choices": ["circle", "square"],     // choice family ONLY: the universe of valid answers
  "instruction": "…",                  // the ONLY text sent to Parrot as the user turn
  "image_path": "instruction-cliff/images/cliff-017.png", // relative to the dataset file's directory
  "blackboard_hint": "…",             // blackboard stage: text prepended to the instruction
  "hint_condition": "correct",         // blackboard stage: correct | none | incorrect | random
  "expected": {
    "value": "square",                 // THE single canonical correct answer
    "aliases": ["the square"],          // other spellings of that same answer — NEVER the answer universe
    "number": 42.0,                     // numeric family
    "tolerance": 0.0,                   // numeric family
    "abstain": false                    // abstain family: true means the correct answer is UNKNOWN
  },

  // end_to_end stage only:
  "variant": "text",                   // "text" (prepends evidence_text to the turn) | "image" (sends image_path)
  "evidence_text": "…full page text…", // text variant only — the page content Parrot sees
  "evidence_cid": "sha256…",           // provenance, never sent
  "source_method": "numeric-cloze-r0", // which generator produced it
  "page_refs": [154],
  "required_facts": ["…exact evidence fragment…"],
  "ground_truth_addresses": ["ohf://…"]
}
```

## end_to_end (P0) — generated, not authored

`tlaloc-parrot-capability-lab generate --stage end_to_end --store <pdfmemory store>`
selects usable pages deterministically (`foldtest.SelectSpacedPages`, skips
cover / TOC / index / references / near-empty), generates candidate Q/A whose
answer is demonstrable from the page's own extracted text, and emits **two
cases per question** sharing `base_id`: a `text` variant and an `image`
variant. `end-to-end.provenance.json` records every question's page, address,
CID, exact evidence fragment + its sha256, and generator method. Categories &
quota: locate / entity / factual / numeric / synthesis, 6 each (§4). Shortfalls
are reported, never fabricated; a human reviews the 30 and bad ones are
regenerated with a new `--seed`. `ValidateEndToEnd` enforces: answer
demonstrable from evidence, evidence + address present, no answer leak in the
question, no duplicates, no ambiguous cloze.

## `expected.value` / `expected.aliases` vs `choices` (P-1 fix #1)

- `expected.value` — the one right answer. `expected.aliases` — equivalent
  spellings of *that same right answer* (`"B"` / `"block b"`).
- `choices` — the whole set of answers the question admits (`["circle","square"]`).
  Used only to tell a wrong-but-valid answer from an answer outside the universe.

They are never mixed. Putting `["warm","cool"]` in `aliases` (so "cool" scores as
correct for a "warm" question) is the exact bug P-1 exists to prevent — the
validator rejects a choice case whose `expected.value` is not in `choices`.

## Score taxonomy (P-1 fix #5)

| field                  | meaning                                                                 |
|------------------------|------------------------------------------------------------------------|
| `semantic_correct`     | content is right, ignoring the requested form                          |
| `format_valid`         | reply is in the requested shape (bare choice / number / JSON / UNKNOWN) |
| `contract_success`     | `semantic_correct && format_valid` — what aggregation counts as correct |
| `abstained`            | answered UNKNOWN                                                        |
| `unsupported_assertion`| asserted a value outside `choices` (only decidable when `choices` is set)|

"Found the right object but replied in a sentence" → `semantic_correct` true,
`format_valid` false, not an unsupported assertion.

## Per-stage required fields

- **end_to_end**: `task_family` ∈ {exact, entity, numeric}, `page_refs`, `expected`.
- **instruction_cliff**: `base_id`, `operations` (1..5), `image_path`, `expected`;
  `choices` for `task_family: choice`; `added_primitive` for depth ≥ 2.
  Groups must be complete OP1..OP5. Prefer the deterministic generator
  (`tlaloc-parrot-capability-lab generate`) over hand authoring.
- **singles**: exactly one `capabilities` entry.
- **interference**: exactly two `capabilities` entries; matching `singles` cases
  must exist so deltas can be computed.
- **coalitions**: `base_id` names the coalition (`K1`/`K2`/`K3`).
- **blackboard**: `blackboard_hint`, `hint_condition`; 4 cases per `base_id`.
