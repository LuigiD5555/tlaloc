# Grounding Automaton R1 change record

Built on the R0 baseline (origin/main @ 0ab168f) after the R0 experimental
campaign showed R0's bottleneck was lexical alignment, not the contradiction
rules: on 6 realistic hand-crafted cases R0 scored 0/6 (all safe
abstentions), because paraphrased / multi-sentence answers never cleared the
0.45 candidate threshold, so the contradiction rules never ran.

R1 keeps the thresholds (0.45 / 0.70), keeps R0's conservative aggregate,
and holds the hard invariant `false_supported_contradiction = 0`.

Changes:

- **Lemma layer** (`lemma.go`): a small curated EN+ES synonym map applied
  inside term extraction, so "losing"/"loss", "requires"/"needs",
  "group"/"collective" align. A lemma group may never contain two members
  of an antonym pair (guarded by `TestR1_LemmaDoesNotMergeAntonyms`).
- **Predicate-core alignment**: the claim side of `claimCoverage` now drops
  a meta prefix ("According to the passage,"), broad grammatical glue, and
  filler/degree words ("roughly", "generally"), so a long or hedged claim
  is not penalised for words the evidence has no reason to echo. The
  evidence side is only lemmatised, never trimmed.
- **Contradiction-signal tie-break** (`bestEvidence`): among evidence spans
  within 0.15 of the top alignment, prefer one that carries an unambiguous
  conflict signal (opposite negation parity, antonym, or quantifier clash)
  against the claim. Fixes the pre-existing
  `TestContradictionHasPriorityAcrossClaims` failure.
- **Below-threshold contradiction bypass**: a claim under 0.45 is still
  inspected for CONTRADICTION (never support) when the aligned span carries
  a conflict signal AND shares a lemmatised core term.
- **Context-aware numeric contradiction** (`numbers_r1.go`): a claim number
  and an evidence number that share an adjacent content word (unit/object)
  and differ in value contradict, even when the evidence sentence carries
  extra unrelated numbers. `claimNumbersConsistent` also blocks a SUPPORTED
  verdict when a claim states a figure absent from the evidence.
- **Non-answer gate** (`isNonAnswer`): a bare keyword dump (space- or
  comma-joined term list with almost no grammatical glue) or a
  hedge-dominated fragment is INSUFFICIENT.
- Extended `antonymPairs` (centralised/decentralised spellings,
  required/optional, up/down, synchronous/asynchronous, include/exclude,
  toward/away) and `quantifierClass` (frequency adverbs: usually/rarely/...).

Results (locked dev corpora):

|              | R0                       | R1                       |
|--------------|--------------------------|--------------------------|
| core-r0      | acc 1.00, cov 0.90, fsc 0 | acc 1.00, cov 0.90, fsc 0 |
| metamorphic-r0 | acc 1.00, cov 1.00, fsc 0 | acc 1.00, cov 1.00, fsc 0 |
| stress-r1 (6) | acc 0.00, cov 0.00, fsc 0 | acc 1.00, cov 0.83, fsc 0 |

Unit suite: all pass (R0's `TestContradictionHasPriorityAcrossClaims` now
green). Full Tlaloc suite: 46/46 packages pass. Determinism: confirmed.
Benchmark: ~30 us/op (R0 ~20 us) — slower from lemma lookups, still trivial.

CAVEAT: stress-r1 was hand-authored by the same person who wrote R1's rules
and some lemma entries were added for words in those cases. The blind OOD
run (fresh corpus from a document not inspected during development) is the
real test.
