# Grounding Automaton R0

`groundingautomaton` is the deterministic first authority for answer grounding.
It is deliberately narrower than general semantic entailment: it decides only
what it can establish from explicit lexical alignment and closed contradiction
checks, and abstains otherwise.

## Contract

Capability: `VERIFY_ANSWER_GROUNDING`

Worker: `grounding-automaton-r0`

Verdicts:

- `SUPPORTED`
- `CONTRADICTED`
- `INSUFFICIENT`
- `UNKNOWN`

The worker emits a claim-level trace containing the selected evidence span,
alignment score, verdict, and reason codes. Contradiction has priority over
support at aggregation time.

## R0 checks

- claim-normalized lexical alignment;
- polarity mismatch;
- numeric mismatch;
- quantifier mismatch;
- a deliberately small bilingual antonym table.

The automaton does not use LM Studio, embeddings, network access, randomness,
or temperature. Learned scorers remain fallback mechanisms for `UNKNOWN` or
`INSUFFICIENT` cases until evaluation demonstrates that they can be retired.

## Local evaluation

From `behavior-lab/`:

```bash
go run ./cmd/tlaloc-grounding-eval \
  -input testdata/grounding/core-r0.jsonl
```

The evaluator reports categorical accuracy, deterministic coverage,
contradiction precision/recall, abstentions, and the critical
`false_supported_contradiction` metric.

R0 intentionally fails closed on contradiction authority: evidence selection
must align first; contradiction checks are never run blindly across an entire
page.
