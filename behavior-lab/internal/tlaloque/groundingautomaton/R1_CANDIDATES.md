# grounding-automaton R1 candidates

Comparison of the canonical R0 (`internal/tlaloque/groundingautomaton/automaton.go`,
merged from `origin/main` @ `0ab168f`) against a parallel implementation
built independently this session from the R0 campaign spec (kept on branch
`grounding-automaton-r0`, commit `de3957f`). Both converged on the same
contract: verdicts `SUPPORTED / CONTRADICTED / UNKNOWN / INSUFFICIENT`,
thresholds `0.45` candidate / `0.70` support, "authoritative only when
provable".

The parallel branch is NOT proposed as a replacement — R0 is the canonical
base with the better structural test scaffolding (contract/gate/threshold/
trace/registry tests). These are the rule-coverage gaps it fills, as R1
input, to be adopted one at a time with their own tests and re-measured
against the locked OOD corpus.

## Confirmed R0 gaps

1. **Whole-span negation parity is fragile.** `polarityContradiction` is
   `hasNegation(claim) != hasNegation(evidence)` over the entire span. An
   evidence sentence like "trained from scratch, with no pretrained backbone
   and no tokenizer dependency" carries two negations about different
   objects; a claim "trained from scratch with no pretrained backbone" then
   reads as opposite parity and false-contradicts.
   *Parallel fix:* count negation only in the ~4 tokens before each shared
   anchor term; compare per-anchor parity.
   Regression guard: `TestContradictionHasPriorityAcrossClaims` currently
   FAILS on R0 (a related alignment/antonym miss).

2. **No unit families.** "The checkpoint is about 3 GB" vs "...about 3 MB"
   is not a contradiction under R0 (`normalizedNumbers` extracts `[3]` and
   `[3]`, equal). *Parallel fix:* a small unit table (bytes / seconds /
   grams / metres) normalises to a base magnitude before comparison.

3. **Magnitude words rely on coincidence.** "3 million" vs "3,000,000" pass
   only because the number regex grabs `3` from "3 million" and
   `3000000 -> 3` reduction happens to match. "3.0M" vs "3 hundred" would
   not be caught. *Parallel fix:* expand `k / m / b / thousand / million /
   billion` into the numeric value at extraction time.

4. **No bound-direction rule.** "the retry count is at least 3" vs "...at
   most 3" is not flagged. *Parallel fix:* parse `up to / at least / at
   most / fewer than / more than` into a bound and treat opposite bounds on
   the same value as a contradiction.

5. **Single-valued antonym map, unfolded.** R0's `antonymPairs` is a fixed
   17-pair list matched as space-padded substrings; a word maps to one
   opposite only ("optional" -> can't resolve to both "required" and
   "mandatory") and "synchronously" won't match the "synchronous/asynchronous"
   pair. *Parallel fix:* `map[string][]string`, suffix-folded keys and
   values, ~50 pairs incl. up/down, reduce/increase, toward/away,
   include/exclude, required/optional, synchronous/asynchronous.

6. **Abbreviations break claim splitting.** `splitClaims` treats every "."
   not between digits as a boundary, so "i.e." / "e.g." / "vs." create
   spurious sub-claims. *Parallel fix:* fold known abbreviations before
   splitting.

7. **No structural INSUFFICIENT gate.** A bare comma-joined keyword dump of
   the passage ("swarm, agents, autonomous, local, ...") is scored
   claim-by-claim and can reach SUPPORTED. R0 only emits INSUFFICIENT for
   empty input or partial coverage. *Parallel fix:* pre-checks for
   keyword-list shape (function-word fraction < 0.08 + >=3 commas) and
   hedge-dominated answers -> INSUFFICIENT.

8. **INSUFFICIENT overloaded for partial coverage.** R0 returns INSUFFICIENT
   when `coverage < 1.0` (some claims below candidate alignment). That reads
   as "the answer is degenerate" when it actually means "I couldn't align
   part of it". *Parallel choice:* reserve INSUFFICIENT for structural
   non-answers; route partial coverage to UNKNOWN (an abstention, which is
   what the consolidator should treat it as anyway).

## Corpora

R0 ships `core-r0` (10) and `metamorphic-r0` (6). The parallel branch has
`core-r0` (50, by family) and `metamorphic-r0` (32, paired one-property
mutations). Worth merging the extra cases in as locked development corpus
(NOT into the OOD exam).

## Not gaps — where R0 is better

- Cleaner file/test separation (`gate.go`, `thresholds.go`, `trace.go`,
  `registry.go` each with a focused test).
- `VERIFY_ANSWER_GROUNDING` as its own capability rather than reusing
  `SCORE_ANSWER_RELEVANCE`.
- Precision-biased numeric rule (same-cardinality requirement) is a
  defensible conservative choice.
