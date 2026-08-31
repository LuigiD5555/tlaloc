# Closed Experimental Loop R0

Status: `IMPLEMENTED_REAL_MODEL_EVIDENCE_PENDING`

Tlaloc alpha.18 closes the Origami-facing operational experiment loop. It can execute repeated model campaigns, diagnose failures, learn from persistent evidence, prioritize candidate changes, compare them, advance a better experimental incumbent, and continue from the newly exposed failure frontier.

It does **not** promote Origami automatically and it does not make Tlaloc the visual authority.

```text
current experimental incumbent PNG
  -> clean Native/R4 trials
  -> deterministic benchmark
  -> failed question IDs only
  -> diagnostic retry
  -> observable failure frontier
  -> Learning Memory
  -> Adaptive Search plan
  -> eligible candidate priority
  -> candidate PNG build/run
  -> persistent evidence
  -> before/after OUTCOME_LINK
  -> non-regression + minimum-improvement gate
  -> best candidate becomes next experimental incumbent
  -> recalculate active failure frontier
  -> repeat
```

## CLI

```bash
tlaloc-closed-loop example > closed-loop.json
tlaloc-closed-loop validate -config closed-loop.json
tlaloc-closed-loop run -config closed-loop.json
```

R0 executes OpenAI-compatible multimodal endpoints. This includes local LM Studio when its OpenAI-compatible server is enabled.

A minimal model entry is:

```json
{
  "name": "lmstudio-vlm",
  "provider": "OPENAI_COMPAT",
  "base_url": "http://127.0.0.1:1234/v1",
  "model": "REPLACE_WITH_VISION_MODEL",
  "timeout_seconds": 180,
  "transport_retries": 1
}
```

API keys are never stored directly in the config. For remote compatible endpoints use `api_key_env` and keep the secret only in that environment variable.

## Clean trials

`NATIVE_PNG_ONLY` sends an empty system prompt, the benchmark question, and the PNG. No benchmark ground truth, decoder, memory, candidate metadata or failure history is exposed.

`R4_ASSISTED` sends the declared Master Prompt plus the same question and PNG.

These clean trials determine the primary score.

## Diagnostic retry

When `diagnostic_retries` is true, Tlaloc evaluates the clean trial first and extracts only failed question IDs. It repeats only those questions with the test-only `ORIGAMI_DEBUG_R0` instruction.

Diagnostic trials are marked `diagnostic_mode=true`, remain outside Native/R4 comparison scores, and report observable protocol progress only. They are not self-bootstrap evidence and never request private chain-of-thought.

## Transport failures

HTTP errors, timeouts and malformed API responses are written to `execution_errors` and do not become semantic failure observations. A transport failure is not evidence that the model failed BOOT, ROSETTA, T2, semantic decode or temporal reasoning.

## Persistent memory

Every successfully evaluated real-model campaign is imported into `tlaloc.learning-memory.r0`.

For candidate trials the loop records:

```text
parent failure observation(s)
  -> CHANGE_ATTEMPT(candidate)
  -> candidate real-model observations
  -> OUTCOME_LINK(before_score, after_score, delta)
```

Old failures remain as regression history, but **historical observations do not vote as the active frontier forever**. For each generation, active failures come from the current experimental incumbent. Historical `CHANGE_ATTEMPT` and `OUTCOME_LINK` events remain available only as bounded search signal.

This lets Tlaloc move from one bottleneck to the next instead of getting stuck on a failure that has already been fixed experimentally.

## Experimental incumbent

The incumbent is a laboratory reference only. It is never a canonical Origami profile.

A candidate can advance only if all of these are true:

```text
candidate clean trial count >= incumbent clean trial count
no benchmark question score decreases
missing-question count does not increase
invented exact claims do not increase
selected outcome metric improves by >= min_incumbent_improvement
```

`min_incumbent_improvement` defaults to `0.01`.

If more than one candidate passes, the candidate with the highest selected outcome metric becomes the next experimental incumbent. Candidate ID is the deterministic tie-breaker.

Example:

```text
baseline
  failures: T2_NOT_FOUND + TEMPORAL_RULE_AMBIGUOUS
      ↓
layout candidate fixes T2 without regression
      ↓
layout candidate = new experimental incumbent
      ↓
active frontier now becomes TEMPORAL_RULE_AMBIGUOUS
      ↓
temporal candidate is tested next
```

## Candidate DAG

Candidates may declare `parent_specimen_id`.

A parent-bound candidate is eligible only when that parent is the current experimental incumbent. This supports staged compositions:

```text
baseline
  -> t2-layout-fix
      -> temporal-grammar-fix
          -> checkpoint-routing-fix
```

Candidates without `parent_specimen_id` are general alternatives and may challenge any incumbent in the current run.

A candidate parent graph is validated before inference; cycles and unknown parents are rejected.

## Adaptive candidate selection

The current incumbent's failure observations are converted to `tlaloc.adaptive-search.r0.plan`. Persistent historical outcomes can change candidate ordering only through the bounded alpha.17 signal and exploration floor.

Memory controls **experiment budget/order**, not empirical winner score.

A candidate tested in an older closed-loop run is not permanently banned. Historical evidence informs search priority, while duplicate execution is suppressed only within the current run.

## Candidate PNG ownership and builders

Tlaloc accepts either:

1. an already rendered candidate PNG; or
2. a declared `build_command` hook that creates the candidate PNG.

A builder receives:

```text
TLALOC_CANDIDATE_ID
TLALOC_OUTPUT_PNG
TLALOC_MUTATIONS_JSON
TLALOC_PARENT_SPECIMEN_ID
TLALOC_PARENT_PNG
```

The parent variables allow a staged candidate to be rendered from the current incumbent. The command remains an explicit external development hook; Tlaloc does not become Origami pixel authority.

## Run artifacts

A run directory contains:

```text
closed-loop-report.json

generation-001/
  plan-before.json
  candidate-queue.json
  plan-after.json
  <incumbent>/
    campaign.json
    result.json
  <candidate>/
    campaign.json
    result.json
```

The campaign JSON stores model responses verbatim. The result JSON stores deterministic scores and debug-frontier summaries. The top-level report records incumbent-before/incumbent-after, advancement reasons, candidate outcomes and stop reason.

## Stopping

R0 stops when:

- the current incumbent cannot produce clean trials;
- the current incumbent has no active failed benchmark questions, unless `continue_exploration_when_stable=true`;
- no eligible candidate remains for the current incumbent; or
- `max_generations` is reached.

Stopping does not imply canonical promotion.

## Hard boundary

```text
CLOSED LOOP = EXPERIMENT EXECUTION LOOP
EXPERIMENTAL INCUMBENT != CANONICAL ORIGAMI
MEMORY GUIDES SEARCH != PROMOTION SCORE
DIAGNOSTIC RETRY != NATIVE SELF-BOOTSTRAP EVIDENCE
CLOSED LOOP != SELF-MODIFYING CANONICAL ORIGAMI
TLALOC RECOMMENDS
ORIGAMI DECIDES
```

Origami remains responsible for deciding which protocol/profile changes become canonical. Tonal may later pin a verified multi-repository composition.
