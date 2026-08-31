# Real VLM Campaign R0

Real VLM Campaign R0 is the transition from synthetic orchestration tests to reproducible experiments against an actual OpenAI-compatible multimodal model.

It does **not** change the benchmark, Origami carrier, candidate builder or alpha.20 evidence gates. It packages them so a real local/remote VLM campaign can be started without manually constructing closed-loop JSON.

## Flow

```text
real OpenAI-compatible endpoint
  -> /models discovery
  -> select one concrete non-synthetic model
  -> Origami builder capability negotiation
  -> build canonical signal-chain carrier
  -> visual transport probe
  -> write provenance manifest
  -> write closed-loop config
  -> clean benchmark trials
  -> targeted diagnostic retries only on failures
  -> Learning Memory
  -> adaptive SuggestedMutations
  -> real Origami candidate builder
  -> candidate trials
  -> non-regression + improvement
  -> experimental incumbent or reject
```

## Required local ingredients

The campaign intentionally consumes independently installed tools:

```text
tlaloc-real-vlm-campaign
origami-temporal-carrier
origami-candidate-build
```

The benchmark program must be the canonical `signal-chain-r0` TemporalProgram used by Temporal Native Benchmark R0. R0 validates its semantic ground truth before any model scoring.

## 1. Doctor

When exactly one model is loaded, model ID selection is automatic:

```bash
tlaloc-real-vlm-campaign doctor \
  --program /path/to/origami/experiments/temporal-automaton-r0/signal-chain.json
```

Default endpoint:

```text
http://127.0.0.1:1234/v1
```

Doctor performs real external checks:

```text
GET /v1/models
reject SYNTHETIC / placeholder IDs
resolve Origami carrier executable
resolve Origami candidate builder executable
query builder capabilities
require exact_plane_mutation=false
require origami.temporal-carrier.r0.profile-1
build 8192-byte baseline
send Q0 + image through the actual multimodal endpoint
```

A non-empty model answer proves only that the declared multimodal transport path worked. It does not prove the answer is semantically correct.

If the endpoint reports multiple models, choose one explicitly:

```bash
tlaloc-real-vlm-campaign doctor \
  --model MODEL_ID \
  --program /path/to/signal-chain.json
```

## 2. Smoke

Smoke is the smallest real execution:

```bash
tlaloc-real-vlm-campaign run \
  --phase SMOKE \
  --program /path/to/signal-chain.json \
  --out runs/real-vlm/origami-temporal-r0
```

Defaults:

```text
1 real model
1 clean trial/model
1 candidate/generation
1 generation
NATIVE_PNG_ONLY
isolated smoke learning memory
```

Smoke can expose a real semantic failure and can exercise the adaptive/candidate loop, but its manifest always states:

```text
promotion_eligible = false
cross_model_evidence = false
```

The smoke memory is physically separated from evidence-phase memory so a connectivity test cannot silently become promotion evidence.

## 3. Repeated evidence

After smoke succeeds:

```bash
tlaloc-real-vlm-campaign run \
  --phase EVIDENCE \
  --program /path/to/signal-chain.json \
  --out runs/real-vlm/origami-temporal-r0
```

Defaults:

```text
>= 3 clean trials/model
2 candidates/generation
up to 3 generations
NATIVE_PNG_ONLY
+ R4_ASSISTED when --master-prompt is provided
separate evidence learning memory
```

R0 deliberately still sets:

```text
promotion_eligible = false
cross_model_evidence = false
```

Three repeated trials from one VLM are stronger than a smoke test but are not cross-model interoperability evidence. A later campaign/aggregation phase must repeat the benchmark on distinct held-out VLMs before a visual change can be considered broadly supported.

## Provenance

Each phase writes a `manifest.json` containing:

```text
model ID
endpoint
Tlaloc version
expected Origami version
TemporalProgram path + SHA-256
baseline PNG path + SHA-256 + byte count
origami-temporal-carrier path + SHA-256
origami-candidate-build path + SHA-256
builder capability report
closed-loop config path + SHA-256
learning-memory root
evidence policy
promotion/cross-model flags
```

This makes a result traceable even when the local binaries later change.

## Generated layout

```text
<out>/
  smoke/
    baseline.png
    manifest.json
    closed-loop.json
    learning-memory/
    closed-loop/
      closed-loop-report.json
      generation-001/...

  evidence/
    baseline.png
    manifest.json
    closed-loop.json
    learning-memory/
    closed-loop/
      closed-loop-report.json
      generation-001/...
```

## Evidence boundaries

```text
MODEL_DISCOVERED != VISION_SUPPORTED
VISION_TRANSPORT_PASS != SEMANTIC_PASS
SMOKE != PROMOTION EVIDENCE
3 TRIALS ON ONE MODEL != CROSS-MODEL EVIDENCE
CANDIDATE BUILD PASS != MODEL IMPROVEMENT
EXPERIMENTAL INCUMBENT != CANONICAL ORIGAMI
TRANSPORT FAILURE != SEMANTIC FAILURE
FALSE_EXACT = 0
```

Tlaloc owns experiment execution and evidence. Origami remains representation and canonical-profile authority.
