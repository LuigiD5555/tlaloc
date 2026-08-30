# Tlaloc 6.0.0-alpha.15

**TLALOC — Transformative Latent Adaptive Logic Orchestration Core**

Tlaloc is a **development kit for behavioral discovery and distillation**.

Its core job is to take an intention, discover a working procedure by composing many deliberately small Tlaloque actions, verify that the procedure really works, and then compress that demonstrated behavior into the most portable deployable artifact.

The default deployment target is a **prompt**.

```text
INTENT
  -> BehaviorSpec + invariants
  -> bounded Tlaloque swarm
  -> real execution + tests
  -> successful reference behavior
  -> distillation
  -> prompt candidates
  -> clean target-model trials
  -> behavioral-fidelity comparison
  -> portable prompt artifact
```

The swarm is the **playground/reference laboratory**, not the production requirement.

## Prompt-first compatibility

Tlaloc assumes the final target model may have nothing except a text interface:

```text
no sandbox
no Go
no Python
no tools
no file access
no Tlaloc runtime
```

For that reason the deployment hierarchy is:

```text
L0  PROMPT_ONLY
L1  PROMPT + DECLARATIVE CONTEXT/IR
L2  PROMPT + TOOLS
L3  PROMPT + RUNTIME
L4  SPECIALIZED MODEL / TARGET-SPECIFIC SYSTEM
```

Tlaloc chooses the **least demanding level that preserves the required behavior**. A runtime-assisted solution does not beat an adequate prompt-only solution merely because its measured score is slightly higher.

The governing target is:

```text
Behavior(artifact) ~= Behavior(reference swarm)
```

not textual imitation of the swarm trace.

See `docs/PROMPT_FIRST_R0.md` and `behavior-lab/spec/PROMPT_FIRST_DISTILLATION_R0.json`.

## What Tlaloque are for

Tlaloque are deliberately bounded workers. A complex task may be decomposed into small operations such as:

```text
extract one claim
check one condition
compare one pair
follow one relation
verify one invariant
mark UNKNOWN
open one evidence address
```

Complex behavior comes from the composition of those steps: state, ordering, branches, loops, evidence and verification.

A successful swarm gives Tlaloc a **reference procedure that is known to work**. Tlaloc then tries to discover the compact rules/instructions that make a clean target model reproduce that behavior without needing the swarm itself.

## Tlaloc is not an Origami subsystem

Origami is one target that Tlaloc can help develop. It is not Tlaloc's identity.

Tlaloc can be used to develop or optimize, for example:

```text
Origami
calculator behavior
document analysis
classifier behavior
workflow procedures
prompted applications
other software/tool behavior
```

For Origami, Tlaloc can experimentally search Master Prompts, representation rules, perceptual channels, layouts, redundancy and other candidate improvements. Origami remains authoritative for deciding what becomes a canonical Origami release/profile.

For another project, the exact same Tlaloc lifecycle can discover a different behavior entirely.

## Origami-facing development tracks

The existing Origami-specific machinery remains useful, but it is now explicitly a **target-specific development profile** built on top of the general Tlaloc lifecycle.

```text
CANONICAL MEMORY R2
PDF/image source -> Canonical Document IR -> exact/evidence plane
             -> Tlaloque candidates -> CanonicalState -> ERA H0-H5

PERCEPTION PROMOTION R1
carrier -> transport variants -> real-model observations
        -> independent Origami evaluator -> tool-loop/routing evidence

VISUAL EVOLUTION R0
current Origami canonical profile
        -> prompt/shape/layout/color/numeric/moire/depth/temporal candidates
        -> deterministic + real-model evidence
        -> tournament
        -> recommendation to Origami

NATIVE SEMANTIC REGRESSION R0
failed prompt-only/native trial
        -> deterministic failure classification
        -> permanent regression
        -> T2/index + semantic-answer + no-mechanical-dependency gates
        -> visual/prompt candidates must beat the failure
```

These components do not make Tlaloc an Origami runtime dependency.

## Failure-driven Native semantic evaluation — alpha.15

A real external trial asked a model for the index of an Origami book carrier. The model could read the bootstrap but then treated the carrier as a binary archive: it requested file/decoder access and produced unverified byte, compression and hash claims instead of reading the semantic index.

Tlaloc alpha.15 turns that failure into an executable development constraint. `behavior-lab/internal/nativeeval` and `tlaloc-native-eval` score target output without using another LLM as judge.

For semantic queries such as identity, index, overview and topic location, Tlaloc now tracks:

```text
native_index_recovery_rate
native_semantic_answer_rate
mechanical_dependency_violations
unverified_mechanical_claims
```

A candidate Origami profile/prompt cannot be recommended when semantic navigation requires undeclared binary extraction, filesystem/sandbox access or decompression. A visually denser or smaller carrier is not an improvement if a plain multimodal model cannot use its semantic plane.

Reference gates are described in `behavior-lab/spec/NATIVE_SEMANTIC_REGRESSION_R0.json`.

Manual evaluation example:

```bash
./bin/tlaloc-native-eval -in trial.json -out result.json
```

The matching Origami change owns the actual T2/semantic carrier correction; Tlaloc only measures candidates and turns failures into regressions.

## Development complexity vs deployment complexity

Tlaloc is allowed to use expensive machinery while learning:

```text
many Tlaloque
multiple target models
sandboxes
Go utilities
evaluators
tools
large experiment corpora
adversarial trials
```

But those are **development resources**.

They must not silently leak into a supposedly portable artifact. An `L0_PROMPT_ONLY` candidate is invalid if it depends on hidden tools, evaluator state, a sandbox or the Tlaloc runtime.

## Clean-target evaluation

A candidate prompt must be evaluated without access to:

```text
private swarm traces
development sandbox state
evaluator ground truth
undeclared tool state
Tlaloc internal memory
```

The candidate receives only what its declared deployment level permits.

This prevents a prompt from appearing successful merely because the development environment is still doing part of the work.

## Current general capabilities

- BehaviorSpec + invariant validation;
- PromptIR compilation/rendering;
- bounded Tlaloque execution/diagnosis;
- reference behavior evaluation;
- swarm-trace distillation;
- prompt candidate experimentation;
- prompt-first deployment selection;
- clean-target behavioral comparison;
- failure-to-regression conversion;
- deterministic Native semantic response evaluation;
- regression gates;
- OpenAI-compatible model adapters;
- target-specific experiment tracks, including Origami.

## Prompt-first reference implementation

`behavior-lab/internal/distill/promptfirst.go` implements a deterministic artifact-selection rule.

Given multiple candidates that already have evaluation evidence, it chooses the lowest deployment level that satisfies behavioral policy. Within the same level, a smaller valid prompt is preferred before a more expensive equivalent artifact.

Default reference policy currently requires:

```text
behavioral_fidelity >= 0.95
pass_rate >= 0.95
regression_rate <= 0.01
clean_target_trials >= 3
```

Those are reference defaults, not universal scientific constants; a project-specific BehaviorSpec may require stricter thresholds.

## Origami visual evolution remains experimental

Tlaloc alpha.13 added evidence-gated search over Origami visual-profile candidates including:

```text
PROMPT
CHANNEL_ROLE
PRIMITIVE
LAYOUT
REDUNDANCY
COLOR_USAGE
NUMERIC_STRUCTURE
INTERFERENCE_STRUCTURE
DEPTH_STRUCTURE
TEMPORAL_STRUCTURE
EMERGENT_STRUCTURE
```

Alpha.15 adds first-class Native semantic fitness: index recovery and semantic usability without undeclared mechanical decoding must not regress.

Prime/modular patterns, moire/phase, stereo/parallax, temporal and emergent structures remain candidates until real evidence demonstrates improvement.

Tlaloc recommends. **Origami decides whether Origami changes.**

## Tonal relationship

Tonal is optional and sits above development tools when a reproducible multi-tool composition is useful.

Example:

```text
Tonal
├── Tlaloc
├── Blueprint Framework
├── another future development kit
└── exact target-project revisions
```

Tonal can pin, verify and reproduce that toolchain. It does not define Tlaloc behavior and it does not promote Origami semantics.

## Source-of-truth hierarchy

```text
project intent / BehaviorSpec / invariants
        = desired behavior authority

reference swarm + test evidence
        = demonstrated procedure

prompt / PromptIR candidate
        = compressed deployment artifact

Tlaloc evaluator
        = comparison machinery

target project (e.g. Origami)
        = authority over its own releases/contracts
```

A prompt is not authoritative merely because Tlaloc generated it. It becomes useful only after it reproduces the requested behavior under the declared evaluation policy.

## Naming

- **Tlaloc** = behavioral development/discovery/distillation kit.
- **Tlaloque** = deliberately bounded workers used to explore and execute micro-steps.
- **Origami** = independent representation/state-machine/visual language and one possible Tlaloc target.
- **Tonal** = optional multi-tool composition/reproducibility layer.

See `docs/NOMENCLATURE.md`.

## Read first

- `docs/PROMPT_FIRST_R0.md`
- `docs/ARCHITECTURE.md`
- `docs/NOMENCLATURE.md`
- `docs/CAPABILITY_STATUS.md`
- `docs/ORIGAMI_INTEGRATION_CONTRACT.md`
- `docs/ORIGAMI_VISUAL_EVOLUTION_R0.md`
- `behavior-lab/spec/NATIVE_SEMANTIC_REGRESSION_R0.json`
- `GATEKEEPER.md`

## Install from source

```bash
git clone git@github.com:LuigiD5555/tlaloc.git
cd tlaloc
./install.sh
```

Tlaloc installs independently under `~/.local/share/tlaloc/versions/<version>`.

A prompt distilled by Tlaloc does **not** require the recipient model to have Tlaloc installed unless the artifact explicitly declares a higher deployment level.

## Hard boundaries

```text
SWARM = REFERENCE LAB, NOT DEFAULT DEPLOYMENT
PROMPT FIRST FOR PORTABILITY
L0 PROMPT REQUIRES NO TOOLS / SANDBOX / TLALOC RUNTIME
DEVELOPMENT DEPENDENCIES != DEPLOYMENT DEPENDENCIES
BEHAVIORAL FIDELITY != TRACE TEXT SIMILARITY
CLEAN TARGET EVALUATION REQUIRED
FAILED REAL TRIAL -> REGRESSION
SEMANTIC NAVIGATION != UNDECLARED BINARY DECODE
UNVERIFIED MECHANICAL CLAIMS BLOCK RECOMMENDATION
ORIGAMI IS A TARGET, NOT TLALOC'S IDENTITY
TLALOC CANDIDATE != CANONICAL ORIGAMI PROFILE
MOCK != EMPIRICAL EVIDENCE
FALSE_EXACT = 0 WHERE EXACTNESS IS CLAIMED
```

## Version

`6.0.0-alpha.15`
