# Tlaloc 6.0.0-alpha.20

**TLALOC — Transformative Latent Adaptive Logic Orchestration Core**

Tlaloc is a development kit for behavioral discovery, verification, distillation and evidence-driven experimentation.

Its general objective is:

```text
INTENT
  -> BehaviorSpec + invariants
  -> bounded Tlaloque workers
  -> demonstrated reference behavior
  -> distillation
  -> portable candidate artifact
  -> clean target-model trials
  -> deterministic evaluation
  -> evidence-backed recommendation
```

The Tlaloque swarm is a development/reference laboratory. It is not the default production runtime.

## Prompt-first portability

Tlaloc assumes a final target model may have only a text interface:

```text
no sandbox
no Go
no Python
no tools
no file access
no Tlaloc runtime
```

Deployment levels are explicit:

```text
L0  PROMPT_ONLY
L1  PROMPT + DECLARATIVE CONTEXT / IR
L2  PROMPT + TOOLS
L3  PROMPT + RUNTIME
L4  SPECIALIZED MODEL / TARGET-SPECIFIC SYSTEM
```

The least demanding artifact that preserves the required behavior is preferred.

```text
Behavior(candidate) ~= Behavior(reference behavior)
```

Tlaloc optimizes behavior, not textual imitation of a swarm trace.

## What Tlaloque are for

Tlaloque are deliberately bounded workers used to discover and test small steps such as:

```text
extract one claim
check one condition
compare one pair
follow one relation
verify one invariant
mark UNKNOWN
open one evidence address
```

Complex behavior is produced by composing those steps through state, ordering, branches, loops, evidence and verification. A successful execution can then be distilled into a smaller portable artifact.

## Origami relationship

Origami is an independent visual/computational protocol and one possible Tlaloc target. Tlaloc does not own Origami semantics or pixels.

For Origami, Tlaloc can experimentally develop and measure:

```text
Master Prompt behavior
BOOT / ROSETTA navigation
semantic codecs
visual layouts
redundancy
channel roles
temporal structures
cross-model read/write behavior
failure regressions
candidate profile changes
```

The authority boundary remains:

```text
Tlaloc experiments + evidence
        -> recommendation
Origami validates and decides canonical protocol/profile changes
Tonal may later pin a reproducible multi-repository composition
```

## Origami-facing development tracks

```text
CANONICAL MEMORY
source -> canonical state -> evidence / exact plane

PERCEPTION PROMOTION
carrier -> transport variants -> real-model observations -> deterministic evaluation

VISUAL EVOLUTION
canonical profile -> isolated experimental mutations -> real evidence -> tournament

NATIVE SEMANTIC REGRESSION
real failed trial -> deterministic failure classification -> permanent regression gate

PROTOCOL INTEROP
semantic state -> E* -> Origami -> S* -> another model -> structural drift measurement

TEMPORAL LEARNING
Tlaloque trace -> automaton/temporal program -> PNG benchmark -> debug frontier -> memory

ADAPTIVE SEARCH
persistent real failures -> mutation priorities -> candidate queue -> real evidence

CLOSED EXPERIMENTAL LOOP
current experimental incumbent PNG -> clean trials -> diagnostics -> memory
  -> adaptive candidate trials -> evidence-gated incumbent advance -> next active frontier

AUTO CANDIDATE GENERATION
current failure plan -> SuggestedMutations -> builder capability negotiation
  -> deterministic one-mutation CandidateConfigs -> target-owned PNG builder
  -> held-out candidate trials -> evidence
```

These tracks are development machinery. They do not make Tlaloc an Origami runtime dependency.

## Native semantic failure rule

A self-declared semantic codec is valid. An undeclared external decoder/file/binary dependency is not.

```text
DECLARED SEMANTIC DECODER SUCH AS S2 = VALID
UNDECLARED EXTERNAL DECODER / FILE / BINARY DEPENDENCY = FAILURE
SEMANTIC QUERY -> UNNECESSARY EXACT/BINARY ESCALATION = FAILURE
```

Tlaloc keeps `FALSE_EXACT=0` and `UNKNOWN > INVENTION` as core experimental discipline.

## Origami Protocol interoperability — alpha.16+

Tlaloc can deterministically evaluate:

```text
Semantic State
  -> E2 ENCODE_SUPERINDEX
  -> Origami
  -> S2 READ_SUPERINDEX
  -> Semantic State
```

and multi-hop communication:

```text
Model A -> Origami -> Model B -> Origami -> Model C
```

The evaluator measures codec discovery, semantic preservation, invented facts, hop-to-hop drift, read/write success, undeclared external dependencies and unnecessary exact escalation. Another LLM is not used as judge.

Synthetic fixtures validate the evaluator only. Real interoperability remains an empirical question.

## Temporal benchmark, debug trace, memory and adaptive search — alpha.17

Alpha.17 introduced a persistent experimental feedback cycle:

```text
real PNG trial
  -> layered Temporal Native Benchmark
  -> failed question IDs
  -> targeted diagnostic retry
  -> observable failure frontier
  -> Learning Memory
  -> Adaptive Search plan
  -> prioritized candidate queue
  -> real candidate trials
  -> outcome linked back to memory
```

### Layered benchmark

The temporal benchmark separates:

```text
P  perception
R  protocol / ROSETTA
S  semantic recovery
T  temporal reasoning
X  exactness / honesty
```

This makes a failure location actionable instead of reducing everything to one score.

### Debug Trace R0

Diagnostic retries report only observable protocol checkpoints:

```text
NONE
 -> BOOT
 -> ROSETTA
 -> CODEC_DISCOVERY
 -> T2_NAVIGATION
 -> SEMANTIC_DECODE
 -> TEMPORAL_ROUTE
 -> TEMPORAL_STEP
 -> EXACT_BOUNDARY
 -> ANSWER
```

The trace includes status, last completed stage, selected codec, last/next instruction identifier, failure code, evidence references and confidence. It never requests private reasoning or chain-of-thought.

Only failed questions are retried. Diagnostic trials are excluded from the primary Native/R4 score.

### Learning Memory R0

Real experimental evidence is stored as immutable content-addressed events under XDG state.

```text
OBSERVATION
CHANGE_ATTEMPT
OUTCOME_LINK
```

The memory preserves both successes and failures. Fixing a failure does not delete it; old evidence remains available for regression analysis.

Synthetic evidence is marked separately and cannot silently become empirical promotion evidence.

### Adaptive Search R0

Persistent real failure patterns select where experiment budget should go next.

Examples:

```text
T2_NOT_FOUND
 -> LAYOUT / PROMPT / REDUNDANCY / CHANNEL_ROLE

TEMPORAL_RULE_AMBIGUOUS
 -> TEMPORAL_STRUCTURE / PROMPT / PRIMITIVE / CHANNEL_ROLE
```

Historical outcomes may adjust search priority only within a bounded range, and every supported mutation family retains a non-zero exploration floor.

The critical boundary is:

```text
MEMORY PRIORITY != PROMOTION SCORE
```

Memory decides what to test first. Evidence decides what worked.

## Closed Experimental Loop R0 — alpha.18 / alpha.19 / alpha.20

Alpha.18 introduced the config-driven runner. Alpha.19 closed the inter-generation gap by making the best non-regressing improvement the **experimental incumbent** for the next generation. Alpha.20 removes the remaining requirement to hand-author a candidate bank when an explicit target-owned builder supports the requested mutation family.

The current loop is:

```text
current experimental incumbent Origami PNG
  -> clean Native / R4 trials
  -> deterministic benchmark
  -> retry only failed questions in diagnostic mode
  -> persist real evidence
  -> calculate the incumbent's active failure frontier
  -> Adaptive Search produces SuggestedMutations
  -> query target-owned builder capabilities
  -> filter unsupported mutation families before model inference
  -> derive deterministic one-mutation CandidateConfigs
  -> delegate PNG build to the explicit target-owned builder
  -> run selected candidates with the same models/questions
  -> targeted diagnostic retries where needed
  -> persist candidate evidence
  -> link incumbent/candidate OUTCOME
  -> require per-question non-regression + exactness discipline + minimum improvement
  -> best passing candidate becomes next experimental incumbent
  -> recalculate the newly exposed failure frontier
  -> repeat
```

The incumbent is laboratory state only. It is not a canonical Origami profile and it never updates the Origami repository.

### Automatic candidate generation — alpha.20

Automatic candidate generation is opt-in:

```json
{
  "auto_candidates": true,
  "candidate_builder": ["origami-candidate-build"],
  "auto_candidate_base_profile_id": "origami.temporal-carrier.r0.profile-1",
  "auto_candidates_per_generation": 4
}
```

Before spending model trials, Tlaloc asks the builder for its declared capabilities. A builder must support the configured parent profile and declare `exact_plane_mutation=false`. Unsupported mutation families are skipped; Tlaloc does not approximate target pixels itself.

Every automatic candidate contains exactly one mutation so that before/after evidence can be attributed to a specific experimental change. Its ID is deterministic from:

```text
parent specimen ID
+ parent PNG SHA-256
+ canonical mutation
```

The alpha.20 synthetic end-to-end regression uses a fake OpenAI-compatible VLM and fake builder to prove orchestration only: failure detection, adaptive generation, builder invocation, candidate evaluation, memory linkage and experimental-incumbent advancement. It is **not** real-model evidence.

See `docs/AUTO_CANDIDATE_GENERATION_R0.md` and `behavior-lab/spec/AUTO_CANDIDATE_GENERATION_R0.json`.

### CLI

```bash
tlaloc-closed-loop example > closed-loop.json
tlaloc-closed-loop validate -config closed-loop.json
tlaloc-closed-loop run -config closed-loop.json
```

`validate` checks the local configuration, Master Prompt, candidate-parent DAG and PNG inputs without running inference. In automatic mode it also negotiates the explicit builder capability contract.

`run` executes the configured experiment generations. When `auto_candidates=false`, the alpha.19 manual candidate path remains unchanged.

### Local LM Studio

R0 uses OpenAI-compatible multimodal endpoints. A local LM Studio entry can be:

```json
{
  "name": "lmstudio-vlm",
  "provider": "OPENAI_COMPAT",
  "base_url": "http://127.0.0.1:1234/v1",
  "model": "REPLACE_WITH_LOADED_VISION_MODEL",
  "temperature": 0,
  "timeout_seconds": 180,
  "transport_retries": 1
}
```

Remote API keys are referenced by environment-variable name through `api_key_env`; secrets are not embedded in the experiment config.

### Clean conditions

`NATIVE_PNG_ONLY` sends only:

```text
empty system prompt
+ benchmark question
+ PNG
```

It does not expose ground truth, memory, decoder internals, candidate metadata or prior failures.

`R4_ASSISTED` sends the declared Master Prompt plus the same question and PNG.

### Transport isolation

A timeout, HTTP error or malformed compatible API response is an execution/transport error, not evidence that the model failed BOOT, ROSETTA, T2 or temporal reasoning.

If no clean incumbent trial completes, the run stops with:

```text
INCUMBENT_EXECUTION_UNAVAILABLE
```

The incumbent is not advanced and transport failures are not inserted into semantic learning memory.

A diagnostic retry is admitted into benchmark evidence only when the complete targeted retry succeeds at the transport layer.

### Evidence-gated incumbent advancement

A candidate can become the next experimental incumbent only when:

```text
candidate clean trial count >= incumbent clean trial count
no benchmark question score decreases
missing-question count does not increase
invented exact claims do not increase
selected outcome metric improves by >= min_incumbent_improvement
```

The default minimum improvement is `0.01`. If multiple candidates pass, the highest selected outcome metric wins; candidate ID is the deterministic tie-breaker.

### Active failure frontier

Old observations remain in persistent memory as regression history, but they no longer permanently vote as the current failure frontier. Each generation derives active failures from the **current incumbent run**. Historical `CHANGE_ATTEMPT` and `OUTCOME_LINK` events remain available only as bounded search signal.

This allows the loop to move naturally:

```text
T2_NOT_FOUND
 -> layout candidate fixes T2
 -> layout candidate becomes experimental incumbent
 -> next run exposes TEMPORAL_RULE_AMBIGUOUS
 -> adaptive search moves to temporal grammar
```

### Candidate DAG and build hooks

A candidate may optionally declare `parent_specimen_id`. A parent-bound candidate becomes eligible only when that parent is the current experimental incumbent. Candidates without a parent remain general alternatives.

The parent graph is validated and rejects cycles or unknown parents.

Tlaloc accepts:

1. pre-rendered experimental Origami PNG candidates;
2. an explicit per-candidate `build_command`; or
3. alpha.20 automatic CandidateConfig generation using an explicit target-owned builder.

The builder hook receives:

```text
TLALOC_CANDIDATE_ID
TLALOC_OUTPUT_PNG
TLALOC_MUTATIONS_JSON
TLALOC_PARENT_SPECIMEN_ID
TLALOC_PARENT_PNG
```

Invoking a renderer does not make Tlaloc the canonical Origami pixel authority.

A candidate tested in an older run is not permanently banned. Persistent history can alter its priority; duplicate execution is suppressed only inside the current closed-loop run.

### Run output

A closed-loop run writes:

```text
closed-loop-report.json

generation-001/
  plan-before.json
  candidate-queue.json
  plan-after.json
  auto-candidates/
    <candidate>.png
  <incumbent>/
    campaign.json
    result.json
  <candidate>/
    campaign.json
    result.json
```

Campaign files preserve model responses verbatim. Result files contain deterministic scoring and diagnostic summaries. The top-level report records the incumbent before/after each generation and why advancement did or did not occur.

### Stopping

A run stops when:

```text
current incumbent execution is unavailable
OR current incumbent has no active failed benchmark questions
OR no supported eligible candidate remains for the current incumbent
OR configured generation budget is exhausted
```

`continue_exploration_when_stable=true` can explicitly continue experimentation after a failure-free incumbent, but still grants no canonical authority.

Stopping never means promotion.

See `behavior-lab/CLOSED_EXPERIMENTAL_LOOP_R0.md`, `behavior-lab/spec/CLOSED_EXPERIMENTAL_LOOP_R0.json`, `docs/AUTO_CANDIDATE_GENERATION_R0.md` and `behavior-lab/spec/AUTO_CANDIDATE_GENERATION_R0.json`.

## Current managed CLIs

```text
tlaloc
tlaloc-behavior-lab
tlaloc-origami
tlaloc-perception-campaign
tlaloc-visual-search
tlaloc-native-eval
tlaloc-protocol-eval
tlaloc-automaton-distill
tlaloc-temporal-bench
tlaloc-learning-memory
tlaloc-adaptive-search
tlaloc-closed-loop
tlaloc-uninstall
```

## Development complexity vs deployment complexity

Tlaloc may use expensive development machinery:

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

Those are development resources. They must not leak into a supposedly portable artifact unless the deployment level explicitly permits them.

A clean L0 candidate cannot inherit private swarm traces, development sandbox state, evaluator ground truth, undeclared tools or Tlaloc internal memory.

## Evidence and promotion boundaries

```text
SYNTHETIC FIXTURE != REAL MODEL EVIDENCE
DIAGNOSTIC RETRY != SELF-BOOTSTRAP EVIDENCE
TRANSPORT FAILURE != MODEL SEMANTIC FAILURE
MEMORY != CANONICAL ORIGAMI TRUTH
ADAPTIVE PRIORITY != PROMOTION SCORE
AUTO CANDIDATE GENERATION != PIXEL AUTHORITY
CANDIDATE BUILD SUCCESS != MODEL IMPROVEMENT
EXPERIMENTAL INCUMBENT != CANONICAL ORIGAMI PROFILE
TLALOC RECOMMENDATION != CANONICAL ORIGAMI PROFILE
COMPLETED CLOSED LOOP != AUTOMATIC PROMOTION
```

Tlaloc recommends. Origami decides whether an Origami protocol/profile change becomes canonical.

## Tonal relationship

Tonal is optional and can pin exact revisions of Tlaloc, Origami and other development tools after a composition is worth reproducing. Tonal does not define Tlaloc behavior and does not promote Origami semantics.

## Naming

- **Tlaloc** = behavioral discovery/distillation/evaluation development kit.
- **Tlaloque** = deliberately bounded workers used during development/reference execution.
- **Origami** = independent visual/computational representation and communication protocol.
- **Tonal** = optional reproducibility/composition layer.

See `docs/NOMENCLATURE.md`.

## Read first

- `docs/PROMPT_FIRST_R0.md`
- `docs/ARCHITECTURE.md`
- `docs/NOMENCLATURE.md`
- `docs/CAPABILITY_STATUS.md`
- `docs/ORIGAMI_INTEGRATION_CONTRACT.md`
- `docs/ORIGAMI_VISUAL_EVOLUTION_R0.md`
- `docs/TEMPORAL_NATIVE_DEBUG_R0.md`
- `docs/AUTO_CANDIDATE_GENERATION_R0.md`
- `behavior-lab/LEARNING_MEMORY_R0.md`
- `behavior-lab/ADAPTIVE_SEARCH_R0.md`
- `behavior-lab/CLOSED_EXPERIMENTAL_LOOP_R0.md`
- `behavior-lab/PROTOCOL_INTEROP_LAB_R0.md`
- `behavior-lab/AUTOMATON_DISTILLATION_R0.md`
- `behavior-lab/TEMPORAL_NATIVE_BENCHMARK_R0.md`
- `GATEKEEPER.md`

## Install from source

```bash
git clone git@github.com:LuigiD5555/tlaloc.git
cd tlaloc
./install.sh
```

Tlaloc installs independently under:

```text
~/.local/share/tlaloc/versions/<version>
```

Learning memory lives under XDG state and is intentionally preserved across managed upgrade/uninstall.

## Hard boundaries

```text
SWARM = REFERENCE LAB, NOT DEFAULT DEPLOYMENT
PROMPT FIRST FOR PORTABILITY
L0 PROMPT REQUIRES NO TOOLS / SANDBOX / TLALOC RUNTIME
DEVELOPMENT DEPENDENCIES != DEPLOYMENT DEPENDENCIES
BEHAVIORAL FIDELITY != TRACE TEXT SIMILARITY
CLEAN TARGET EVALUATION REQUIRED
FAILED REAL TRIAL -> REGRESSION
DECLARED SEMANTIC CODEC = ALLOWED
UNDECLARED EXTERNAL CODEC DEPENDENCY = FAILURE
SEMANTIC NAVIGATION != UNNECESSARY EXACT/BINARY DECODE
FALSE_EXACT = 0 WHERE EXACTNESS IS CLAIMED
UNKNOWN > INVENTION
SYNTHETIC EVIDENCE != EMPIRICAL PROMOTION EVIDENCE
DIAGNOSTIC RETRY != PRIMARY SCORE
TRANSPORT FAILURE != MODEL SEMANTIC FAILURE
MEMORY != CANONICAL ORIGAMI TRUTH
MEMORY != AUTOMATIC PROMOTION
MEMORY GUIDES EXPERIMENT BUDGET, NOT PROMOTION SCORE
REAL MODEL FAILURES DRIVE ADAPTIVE FOCUS
EXPLORATION FLOOR > 0
AUTO CANDIDATE GENERATION IS OPT-IN
TLALOC GENERATES MUTATION INTENT, NOT CANONICAL PIXELS
UNSUPPORTED BUILDER CAPABILITY -> FILTER BEFORE INFERENCE
ONE AUTOMATIC CANDIDATE = ONE MUTATION
CANDIDATE BUILD SUCCESS != MODEL IMPROVEMENT
EXPERIMENTAL INCUMBENT != CANONICAL ORIGAMI PROFILE
TLALOC CANDIDATE != CANONICAL ORIGAMI PROFILE
CLOSED LOOP != SELF-MODIFYING CANONICAL ORIGAMI
ORIGAMI IS A TARGET, NOT TLALOC'S IDENTITY
```

## Version

`6.0.0-alpha.20`
