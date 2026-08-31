# Tlaloc 6.0.0-alpha.18

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
baseline PNG -> clean trials -> diagnostics -> memory -> candidate trials -> outcomes -> next plan
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

## Closed Experimental Loop R0 — alpha.18

Alpha.18 operationalizes the complete experiment cycle through one runner:

```text
baseline Origami PNG
  -> clean Native / R4 trials
  -> deterministic benchmark
  -> retry only failed questions in diagnostic mode
  -> persist real evidence
  -> calculate adaptive failure target
  -> prioritize candidate PNG bank
  -> record CHANGE_ATTEMPT
  -> run selected candidates with the same models/questions
  -> targeted diagnostic retries where needed
  -> persist candidate evidence
  -> link baseline/candidate OUTCOME
  -> recalculate next Adaptive Search plan
  -> repeat within generation budget
```

### CLI

```bash
tlaloc-closed-loop example > closed-loop.json
tlaloc-closed-loop validate -config closed-loop.json
tlaloc-closed-loop run -config closed-loop.json
```

`validate` checks the local configuration, Master Prompt and PNG inputs without running inference. A missing candidate PNG is allowed only when the candidate declares an explicit external `build_command`.

`run` executes the configured experiment generations.

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

If no clean baseline trial completes, the generation stops with:

```text
BASELINE_EXECUTION_UNAVAILABLE
```

and does not compare candidates against a fabricated zero baseline.

A diagnostic retry is admitted into benchmark evidence only when the complete targeted retry succeeds at the transport layer.

### Candidate PNG bank

Tlaloc accepts:

1. pre-rendered experimental Origami PNG candidates; or
2. an explicit `build_command` hook that creates the declared PNG path.

The hook receives:

```text
TLALOC_CANDIDATE_ID
TLALOC_OUTPUT_PNG
TLALOC_MUTATIONS_JSON
```

Invoking a renderer does not make Tlaloc the canonical Origami pixel authority.

### Run output

A closed-loop run writes:

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

Campaign files preserve model responses verbatim. Result files contain deterministic scoring and diagnostic summaries.

### Stopping

A run stops when:

```text
configured generation budget is exhausted
OR candidate bank is exhausted
OR baseline execution is unavailable
```

Stopping never means promotion.

See `behavior-lab/CLOSED_EXPERIMENTAL_LOOP_R0.md` and `behavior-lab/spec/CLOSED_EXPERIMENTAL_LOOP_R0.json`.

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
TLALOC CANDIDATE != CANONICAL ORIGAMI PROFILE
CLOSED LOOP != SELF-MODIFYING CANONICAL ORIGAMI
ORIGAMI IS A TARGET, NOT TLALOC'S IDENTITY
```

## Version

`6.0.0-alpha.18`
