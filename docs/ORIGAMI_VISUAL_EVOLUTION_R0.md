# Origami Visual Evolution R0

Status: `EXPERIMENTAL_SEARCH_MACHINERY`

Tlaloc may search for better ways to encode Origami visually, but it does not create a different aesthetic for each document.

The object being optimized is a **candidate next canonical Origami profile**.

## Lifecycle

```text
Origami canonical profile N
        |
        v
Tlaloc candidate mutations
        |
        +-- Master Prompt
        +-- channel roles
        +-- visual primitives
        +-- layout
        +-- redundancy
        +-- color
        +-- numeric / mathematical structure
        +-- temporal structure
        |
        v
real-model + deterministic experiments
        |
        v
Tlaloc tournament
        |
        v
PROMOTION RECOMMENDATION ONLY
        |
        v
Origami validation
        |
        v
Tonal stack/profile promotion
        |
        v
Origami canonical profile N+1
```

## Why this belongs in Tlaloc

Tlaloc already owns search, Tlaloque coordination, model campaigns, prompt experimentation and promotion recommendations.

That makes it the right place to test questions such as:

- does color reduce VLM state-confusion when retained redundantly with shape/contrast?;
- does a radial or hierarchical layout improve routing?;
- does another primitive survive resize/JPEG better?;
- does more or less repetition improve perceptual reliability per byte?;
- can mathematical structures encode useful relationships more compactly?;
- does a modified Master Prompt improve BOOT/ROSETTA decoding across models?;
- can a temporal/phase channel add capacity without making static fallback unsafe?

The answer must come from evidence rather than visual intuition.

## Prime / numeric example

A prime-derived pattern is represented as an experimental mutation such as:

```json
{
  "kind": "NUMERIC_STRUCTURE",
  "target": "profile",
  "value": "PRIME_DERIVED_SPACING",
  "experimental": true
}
```

It is not privileged. A modular, factorization, periodic or other mathematical strategy is treated the same way.

A candidate only becomes recommendable when it preserves semantic roundtrip and evidence discipline while measurably improving the evaluated baseline.

## Candidate evidence

The reference metrics include:

```text
semantic_roundtrip_rate
boot_probe_pass_rate
routing_accuracy
verified_evidence_rate
transport_pass_rate
context_efficiency
mean_context_tokens
carrier_bytes
false_exact
budget_violations
unknown_violations
real_models
trials
```

## Hard gates

Default gates require:

```text
semantic_roundtrip_rate = 1.0
FALSE_EXACT = 0
budget_violations = 0
unknown_violations = 0
carrier_bytes <= 512000
mean_context_tokens <= 4000
verified_evidence_rate >= 0.95
routing_accuracy >= 0.95
real_models >= 3
trials >= 9
critical semantic/evidence non-regression
measurable weighted improvement over baseline
```

The exact policy is experimental and may be made stricter, but Tlaloc must not weaken Origami's hard semantic/exactness invariants.

## Weighted score

After hard gates, the deterministic reference score combines:

```text
semantic roundtrip
BOOT/probe perception
routing
verified evidence
transport robustness
context efficiency
carrier-size efficiency
```

The score is only a tournament ranking mechanism. Passing it does not make the candidate canonical.

## Authority

The output authority string is intentionally:

```text
TLALOC_RECOMMENDATION_ONLY_ORIGAMI_VALIDATES_TONAL_PROMOTES
```

Tlaloc never writes a candidate directly into Origami's canonical profile registry as a consequence of this tournament.

## Prompt evolution

`PROMPT` is a first-class candidate mutation.

This means Tlaloc may run experiments against modifications of the universal Master Prompt alongside visual changes.

Examples:

```text
more explicit ROSETTA instruction
less text before visual probe
alternative WRITE-mode decomposition
model-family-specific wording as an experimental candidate
additional error/fail-closed instruction
```

A prompt mutation must be evaluated against the same canonical carrier/profile semantics. It cannot secretly change what the carrier means.

## Current baseline

The baseline profile under this contract is intended to be:

```text
origami.canonical-aesthetic.r0
```

with high-contrast geometry/topology/position/enclosure/scale/repetition/density and limited text.

Color, numeric structures and temporal channels are candidate capabilities until promoted.

## CLI

```bash
cd behavior-lab
go run ./cmd/tlaloc-visual-search -in experiment.json -out tournament.json
```

The input contains the baseline metrics, candidate mutations and evidence bundles. The output ranks candidates and optionally recommends one to Origami for validation.
