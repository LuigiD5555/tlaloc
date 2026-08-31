# Origami alpha.15 / Tlaloc alpha.20 Cross-Repo Integration R0

This gate proves the executable development contract between two independently versioned repositories without turning either project into a hidden runtime dependency of the other.

Pinned composition:

```text
Origami  6.0.0-alpha.15
commit   546f36b05712ffaf3d7324bcbee2b12ddd0d4af7

Tlaloc   6.0.0-alpha.20
```

## What the gate executes

```text
Origami temporal program fixture
  -> real origami-temporal-carrier
  -> parent PNG (640x640 / 8192 bytes)
  -> Tlaloc closed-loop clean trial
  -> SYNTHETIC OpenAI-compatible VLM reports one semantic failure
  -> Tlaloc Learning Memory / plan
  -> automatic one-mutation CandidateConfig
  -> real origami-candidate-build
  -> candidate PNG
  -> candidate clean trial
  -> deterministic non-regression / improvement gate
  -> experimental incumbent advancement
  -> real origami-temporal-carrier decode of parent and candidate
  -> exact TemporalProgram equality
```

The synthetic model exists only to make the experiment outcome deterministic. Its model ID begins with `SYNTHETIC_`, so the benchmark does not classify it as real-model evidence.

## What this proves

- Tlaloc alpha.20 can consume Origami alpha.15 builder capabilities directly;
- the `TLALOC_*` build-hook ABI works across independent repository builds;
- automatic CandidateConfigs can be compiled by the real Origami builder;
- parent and candidate remain exactly 8192-byte PNGs;
- the candidate image changes while the embedded exact TemporalProgram remains equal;
- Tlaloc can score the candidate, link outcome evidence and advance a laboratory incumbent;
- synthetic integration evidence remains distinguishable from empirical VLM evidence.

## What this does not prove

```text
CROSS_REPO_MECHANICAL_PASS != REAL_VLM_SUCCESS
SYNTHETIC_MODEL != EMPIRICAL_MODEL_EVIDENCE
CANDIDATE_ADVANCE != CANONICAL_ORIGAMI_PROMOTION
PINNED_DEV_INTEGRATION != RUNTIME_DEPENDENCY
```

The next evidence phase replaces the synthetic OpenAI-compatible endpoint with a real multimodal model while preserving the same carrier, builder, questions, evaluator and non-regression rules.

## Why the Origami revision is pinned

The workflow checks out an exact Origami commit rather than `main`. This prevents an unrelated future Origami change from silently changing the meaning of a Tlaloc integration result.

A newer pair should receive a new verified composition/pin rather than mutating historical evidence.
