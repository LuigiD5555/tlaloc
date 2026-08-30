# Tlaloc 6.0.0-alpha.13

**TLALOC — Transformative Latent Adaptive Logic Orchestration Core**

Tlaloc is the work system: behavior compilation, Tlaloque coordination/training, orchestration, verification, experimentation, promotion recommendations and model-facing execution.

Origami is an independent representation/state-machine/visual language. Tlaloc may use it and experimentally search for better ways to make its canonical representation readable, compact and robust, but Tlaloc does not own Origami semantics or directly mutate its canonical profile.

## Current implementation

Tlaloc currently combines three major Origami-facing paths:

```text
CANONICAL MEMORY R2
PDF/image source -> Canonical Document IR -> exact/evidence plane
             -> Tlaloque candidates -> CanonicalState -> ERA H0-H5

PERCEPTION PROMOTION R1
carrier -> transport variants -> real-model observations
        -> independent Origami evaluator -> tool-loop/routing gates

VISUAL EVOLUTION R0
current Origami canonical profile
        -> candidate prompt/shape/layout/color/numeric/temporal mutations
        -> deterministic + real-model evidence
        -> tournament
        -> recommendation to Origami only
```

Alpha.13 adds **Origami Visual Evolution R0**. Its object of search is a candidate next canonical Origami profile, not a new aesthetic for every document.

### Existing R2 / alpha.12 machinery

- layout-preserving Canonical Document IR with OCR fallback and exact source preservation;
- stable page/block/region addresses, CIDs and Merkle exact plane;
- proposal-only Tlaloque candidate generation, deterministic CanonicalState and conflict/uncertainty control;
- External Recursive Attention H0-H5 under bounded active context;
- `tlaloc.origami-tools.r2` BOOT/QUERY/EXPAND/VERIFY bridge;
- OpenAI-compatible multimodal function/tool loop and text fallback;
- cross-model Perception Promotion Campaign machinery;
- original, 75%, 50% and JPEG-preview transport variants;
- REAL_MODEL-only aggregate gates, default 3 models x 3 original trials;
- real tool-loop and held-out routing gates;
- Hybrid and Native T3 candidate states kept separate.

### New Visual Evolution R0 machinery

Tlaloc can represent experimental profile mutations as:

```text
PROMPT
CHANNEL_ROLE
PRIMITIVE
LAYOUT
REDUNDANCY
COLOR_USAGE
NUMERIC_STRUCTURE
TEMPORAL_STRUCTURE
```

Examples of `NUMERIC_STRUCTURE` candidates include prime-derived spacing, modular patterns, factorization structures and periodic density. These are experimental encodings, not privileged or automatically useful.

The reference tournament measures:

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

A candidate cannot be recommended when it regresses semantic roundtrip, evidence/routing discipline, UNKNOWN behavior, budget, carrier-size limits or `FALSE_EXACT=0`.

The reference output authority is deliberately:

```text
TLALOC_RECOMMENDATION_ONLY_ORIGAMI_VALIDATES_TONAL_PROMOTES
```

Tlaloc never turns the winning experiment directly into canonical Origami.

## One Origami aesthetic, versioned evolution

Tlaloc now tracks the intended Origami rule:

```text
ONE CANONICAL ORIGAMI AESTHETIC PER PROFILE VERSION
```

ROSETTA remains mandatory for profile/version, active dimensions and concrete bindings, but it does not mean that each PDF may invent a private visual dialect.

The intended evolution loop is:

```text
Origami canonical profile N
        |
        v
Tlaloc experiments
 prompt / geometry / layout / redundancy /
 color / numeric structure / temporal structure
        |
        v
candidate + evidence
        |
        v
Origami semantic/visual validation
        |
        v
Tonal composition/promotion gate
        |
        v
Origami canonical profile N+1
```

Old carriers remain readable through their embedded profile/version and ROSETTA.

## Prompt ownership

Origami's universal Master Prompt is the canonical READ/WRITE behavior contract.

Tlaloc may:

- generate operational prompts for a specific experiment/runtime;
- mutate prompt candidates experimentally;
- benchmark prompts across real models;
- distill successful behavior into candidates.

It may not silently redefine what Origami means. Prompt mutations are evaluated against the same semantic and visual contracts as visual mutations.

The Fixed Carrier R2 prompt used by `tlaloc-origami` is therefore an operational receiver bridge for the current profile, not a second semantic source of truth.

## Visual evolution CLI

From `behavior-lab`:

```bash
go run ./cmd/tlaloc-visual-search \
  -in experiment.json \
  -out tournament.json
```

The input contains one baseline profile metrics set plus candidate mutations and their evidence bundles. The output ranks candidates and may recommend one for **Origami validation**.

A recommendation is not a profile promotion.

## Perception campaign CLI

The alpha.12 campaign machinery remains available through:

```text
tlaloc-perception-campaign
```

It executes frozen carrier/prompt trials against OpenAI-compatible image-capable models, calls the independent Origami evaluator and aggregates real-model/tool-loop/routing evidence.

No repository test or mock run is treated as evidence that external VLMs actually support Hybrid or Native Origami.

## Project skills vs generated skills

The `.claude/skills/` directory contains five checked-in workflow skills owned by Tlaloc/Origami development:

- `tlaloc-project`
- `tlaloc-behavior`
- `tlaloc-tlaloque`
- `origami-semantics`
- `tlaloc-release`

Use `tlaloc skills list/path/install` for these Tlaloc-owned project skills. Project-agnostic `repo-flow` and `gatekeeper` remain Tonal-owned.

Checked-in project skills are not compiler output and are not a second semantic authority.

## Source-of-truth rule

`BehaviorSpec + invariants` define intended Tlaloc behavior.

For Origami integration:

```text
Origami contracts = semantic / visual / carrier authority
Tlaloc             = search / execution / experimentation / recommendation
Tonal              = composition / aggregate promotion authority
```

## Naming

- **Tlaloc** = complete work/orchestration/search system.
- **Tlaloque** = bounded specialist agents coordinated by Tlaloc.
- **Origami** = independent representation/state-machine/visual language.
- **Tonal** = independent composition/distribution and stack promotion layer.

See `docs/NOMENCLATURE.md`.

## Read first

- `GATEKEEPER.md`
- `docs/NOMENCLATURE.md`
- `docs/ARCHITECTURE.md`
- `docs/CAPABILITY_STATUS.md`
- `docs/ORIGAMI_INTEGRATION_CONTRACT.md`
- `docs/ORIGAMI_VISUAL_EVOLUTION_R0.md`

## Install from source

```bash
git clone git@github.com:LuigiD5555/tlaloc.git
cd tlaloc
./install.sh
```

Tlaloc installs user-locally under `~/.local/share/tlaloc/versions/<version>` and does not install Origami as a mandatory dependency.

## Hard boundaries

```text
MOCK != EMPIRICAL EVIDENCE
ONE REAL TRIAL != SUPPORTED
HYBRID_SUPPORTED != NATIVE_VISUAL_SUPPORTED
ONE DOCUMENT != ONE NEW ORIGAMI AESTHETIC
TLALOC CANDIDATE != CANONICAL ORIGAMI PROFILE
SEMANTIC ROUNDTRIP MUST NOT REGRESS
FALSE_EXACT = 0
UNKNOWN DISCIPLINE MUST NOT REGRESS
TLALOC RECOMMENDS / ORIGAMI VALIDATES / TONAL PROMOTES
```

## Version

`6.0.0-alpha.13`
