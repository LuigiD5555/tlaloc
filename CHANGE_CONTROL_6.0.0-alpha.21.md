# Change Control — Tlaloc 6.0.0-alpha.21

## Scope

Alpha.21 introduces **Real VLM Campaign R0**. It packages the alpha.20 automatic closed experimental loop into a reproducible entry point for actual OpenAI-compatible multimodal models while preserving the separation between transport validation, semantic evidence and canonical promotion.

## New managed surface

```text
tlaloc-real-vlm-campaign
```

Commands:

```text
doctor
prepare
run
example
```

Default endpoint:

```text
http://127.0.0.1:1234/v1
```

## Real-model discovery

The campaign queries `/v1/models` before running. It auto-selects a model only when exactly one model is reported and rejects synthetic or placeholder model identifiers.

```text
SYNTHETIC FIXTURE != REAL MODEL EVIDENCE
```

When the endpoint exposes multiple models, the caller must select one explicitly.

## Doctor and visual transport

`doctor` validates the canonical signal-chain program, resolves `origami-temporal-carrier` and `origami-candidate-build`, records executable SHA-256 provenance, negotiates builder capabilities, builds the canonical 640x640 / 8192-byte PNG and performs a real multimodal transport probe.

The transport probe establishes only that the model endpoint accepted and processed the visual request:

```text
VISION TRANSPORT PASS != SEMANTIC BENCHMARK PASS
```

## SMOKE phase

SMOKE is the minimum operational real-model campaign:

```text
1 real model
1 clean trial/model
1 candidate/generation
1 generation
NATIVE_PNG_ONLY
isolated learning-memory root
```

A smoke run may expose genuine model failures and may exercise the adaptive candidate loop, but it is explicitly excluded from promotion evidence.

```text
SMOKE != PROMOTION EVIDENCE
```

## EVIDENCE phase

EVIDENCE requires at least three trials per model. Defaults are two candidates per generation and up to three generations. When a Master Prompt is supplied, `R4_ASSISTED` may be evaluated alongside `NATIVE_PNG_ONLY`.

R0 remains intentionally single-model:

```text
promotion_eligible = false
cross_model_evidence = false
```

Repeated evidence from one model is empirical evidence for that model, not broad interoperability evidence.

## Provenance and isolation

Each phase writes a manifest binding:

```text
model ID + endpoint
Tlaloc version
expected Origami contract version
program path + SHA-256
baseline PNG path + SHA-256 + bytes
origami-temporal-carrier path + SHA-256
origami-candidate-build path + SHA-256
builder capabilities
closed-loop config path + SHA-256
learning-memory root
evidence policy
promotion/cross-model flags
```

SMOKE and EVIDENCE use separate output and learning-memory roots so exploratory failures cannot silently become promotion evidence.

## Reused alpha.20 machinery

Alpha.21 does not replace the closed-loop engine. It invokes the existing:

```text
clean trial
 -> deterministic benchmark
 -> failed-question diagnostic retry
 -> learning memory
 -> adaptive search
 -> target-owned candidate build
 -> candidate trial
 -> per-question non-regression
 -> experimental incumbent advancement
```

Origami remains the pixel/profile authority.

## Verification

Required alpha.21 gates:

- version coherence across `VERSION`, README and capability status;
- Real VLM Campaign R0 formal contract;
- deterministic OpenAI-compatible `httptest` transport fixture;
- synthetic/placeholder model rejection;
- model-discovery ambiguity handling;
- canonical 8192-byte baseline generation;
- builder capability negotiation and exact-plane guard;
- SMOKE/EVIDENCE invariant tests;
- managed install/uninstall lifecycle includes `tlaloc-real-vlm-campaign`;
- `go test ./...`;
- `go vet ./...`;
- `go test -race ./...`;
- Gatekeeper.

CI must never fabricate an external real-model result.

## Evidence status

```text
REAL_VLM_CAMPAIGN_IMPLEMENTED = true
REAL_EXTERNAL_VLM_SMOKE = pending
REAL_EXTERNAL_VLM_EVIDENCE = pending
CROSS_MODEL_EVIDENCE = pending
A_TO_B_TO_C_REAL_EVIDENCE = pending
```

## Authority

```text
Tlaloc executes experiments, records evidence and recommends.
Origami owns canonical representation/profile promotion.
Tonal may later pin a verified cross-project composition.
```
