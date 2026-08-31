# Closed Experimental Loop R0

Status: `EXPERIMENTAL_IMPLEMENTED_REAL_MODEL_EVIDENCE_PENDING`

This layer closes the operational loop built across alpha.15-alpha.17. It does not promote Origami automatically and it does not make Tlaloc the visual authority.

```text
baseline PNG
  -> clean Native/R4 trials
  -> deterministic benchmark
  -> failed question IDs only
  -> diagnostic retry
  -> observable failure frontier
  -> Learning Memory
  -> Adaptive Search plan
  -> candidate bank priority
  -> selected candidate PNG trials
  -> diagnostic retry where needed
  -> persistent evidence
  -> before/after OUTCOME_LINK
  -> next Adaptive Search plan
  -> repeat until budget/candidate bank ends
```

## CLI

```bash
tlaloc-closed-loop example > closed-loop.json
tlaloc-closed-loop validate -config closed-loop.json
tlaloc-closed-loop run -config closed-loop.json
```

R0 executes OpenAI-compatible multimodal endpoints. This includes local LM Studio when its OpenAI-compatible server is enabled.

A minimal local model entry is:

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

API keys are never stored directly in the config. For remote compatible endpoints use `api_key_env` and put the secret only in that environment variable.

## Clean trials

`NATIVE_PNG_ONLY` sends an empty system prompt, the declared benchmark question, and the PNG. No benchmark ground truth, decoder, memory, candidate metadata or failure history is exposed.

`R4_ASSISTED` sends the declared Master Prompt plus the same question and PNG.

These clean trials determine the primary score.

## Diagnostic retry

When `diagnostic_retries` is true, Tlaloc evaluates the clean trial first and extracts only failed question IDs. It then repeats only those questions with the test-only `ORIGAMI_DEBUG_R0` instruction.

Diagnostic trials are marked `diagnostic_mode=true`, remain outside Native/R4 comparison scores, and report observable protocol progress only. They are not self-bootstrap evidence.

## Transport failures

HTTP errors, timeouts and malformed API responses are written to `execution_errors` and do not become semantic failure observations. A transport failure is not evidence that the model failed BOOT, ROSETTA, T2, semantic decode or temporal reasoning.

## Persistent memory

Every successfully evaluated real-model specimen campaign is imported into `tlaloc.learning-memory.r0`.

For candidate trials the loop records:

```text
parent failure observation(s)
  -> CHANGE_ATTEMPT(candidate)
  -> candidate real-model observations
  -> OUTCOME_LINK(before_score, after_score, delta)
```

This provides causal experimental history rather than a flat error log.

## Adaptive candidate selection

After the baseline is evaluated, the current Learning Memory is summarized and converted to `tlaloc.adaptive-search.r0.plan`.

The candidate bank is prioritized before evidence is collected. Historical outcomes may modify experiment order only within the bounded alpha.17 policy. The final empirical result is still determined by the benchmark and normal gates.

Candidates already carrying an `OUTCOME_LINK` in the persistent memory are not automatically retested under the same candidate ID. Use a new candidate ID for a materially new variant or replication campaign.

## Candidate PNG ownership

Tlaloc accepts either:

1. an already rendered candidate PNG; or
2. a `build_command` hook that must create the declared PNG path.

The hook receives:

```text
TLALOC_CANDIDATE_ID
TLALOC_OUTPUT_PNG
TLALOC_MUTATIONS_JSON
```

The command is an explicit external development hook. Tlaloc does not become Origami's pixel authority merely because it invokes the renderer.

## Run artifacts

A run directory contains:

```text
closed-loop-report.json

generation-001/
  plan-before.json
  candidate-queue.json
  plan-after.json
  <baseline>/
    campaign.json
    result.json
  <candidate>/
    campaign.json
    result.json
```

The campaign JSON stores model responses verbatim. The result JSON stores deterministic scores and debug-frontier summaries.

## Stopping

R0 stops when either:

- the configured `max_generations` is reached; or
- the candidate bank is exhausted.

Stopping does not imply promotion. The final `plan-after.json` is the next experimental recommendation, not canonical Origami truth.

## Hard boundary

```text
CLOSED LOOP = EXPERIMENT EXECUTION LOOP
CLOSED LOOP != SELF-MODIFYING CANONICAL ORIGAMI
CLOSED LOOP != AUTOMATIC PROMOTION
```

Origami remains responsible for deciding which protocol/profile changes become canonical. Tonal may later pin a verified multi-repository composition.
