# Learning Policy + Prompt Genome R1

Status: `EXPERIMENTAL_REFERENCE_IMPLEMENTATION`

This layer turns Tlaloc's immutable experimental memory into guarded, cumulative prompt/protocol evolution. It exists to prevent two recurring failures: repeating known mistakes and accidentally destroying behavior that previous real-model experiments already improved.

## Authority boundary

```text
TLALOC LEARNS AND RECOMMENDS
ORIGAMI MATERIALIZES AND DECIDES CANONICAL FORMAT
TONAL FREEZES STABLE COMPOSITIONS LATER
```

Learning policy never promotes an Origami profile by itself.

## End-to-end loop

```text
real VLM evidence
  -> Learning Memory (immutable observations/change/outcome)
  -> failure frontier
  -> Learning Policy
       PRESERVE / AVOID / REQUIRE / MUTABLE
  -> Prompt Genome
       modular versioned protocol source
  -> Experiment Intent
       one primary weak module
  -> Candidate Manifests
  -> Origami deterministic builder
  -> Build Manifest + Visible Semantic Manifest
  -> Semantic Parity Gate
       reject unauthorized semantic drift
  -> regression preconditions
  -> real VLM trials
  -> assertion evaluator + P/R/S/T/X
  -> Outcome Learner
       causal win / no improvement / regression / invalid experiment
  -> OUTCOME_LINK
  -> derived policy rebuilt from memory
  -> next experiment
```

## Learning Policy

Schema: `tlaloc.learning-policy.r1`.

Four rule classes:

- `PRESERVE`: an experimentally useful module/candidate must not be changed incidentally.
- `AVOID`: historical negative outcome or known bad path.
- `REQUIRE`: non-negotiable experimental integrity constraint.
- `MUTABLE`: the single development area selected from the current real-model failure frontier.

The policy also exposes `LearnedInvariant` and `AntiPattern` records. Positive `OUTCOME_LINK` history becomes preservation evidence. Invalid-specimen/semantic-drift history creates the anti-pattern `GENERATIVE_REWRITE_OF_EXACT_SEMANTICS` and requires `SEMANTIC_PARITY_GATE`.

## Prompt Genome

Schema: `tlaloc.prompt-genome.r1`.

The master prompt is source-controlled as independent modules instead of regenerated monolithically:

```text
BOOT
ROSETTA
SEMANTIC_READING
TEMPORAL_GRAMMAR
EXECUTION_POLICY
EXACTNESS_POLICY
OUTPUT_POLICY
...
```

Each module carries version, purpose, priority, full/minimal text, required/protected flags, maturity, evidence references, dependencies and optional model scope.

The default profile is `behavior-lab/profiles/prompt-genome-r1.json`.

### Prompt compilation

`tlaloc-prompt compile` selects required + relevant modules, applies model scope, sorts deterministically by priority, enforces dependencies and fits a character budget using `min_text` only when needed.

The compiled JSON records the exact genome/module versions used. The text output is the model-facing master prompt. The genome is source; compiled prompts are products.

## Experiment Intent

Schema: `tlaloc.experiment-intent.r1`.

An intent binds:

- objective;
- validated baseline candidate;
- current failure frontier;
- exactly one mutable module;
- preserved/avoided/required knowledge;
- candidate budget;
- trials per model;
- target models.

`tlaloc-learn plan` derives this from Learning Memory and generates guarded Candidate Manifests. `-record-attempts` persists each generated experiment as a `CHANGE_ATTEMPT` tagged with `target:`, `module:` and `mutation:` so later outcomes can be attributed to modules rather than filenames.

## Candidate Manifest

Schema: `tlaloc.candidate-manifest.r1`.

A candidate declares before rendering:

- parent/baseline;
- canonical program SHA;
- exact payload SHA;
- genome version;
- exactly one primary mutation;
- changed module;
- preserved modules;
- forbidden changes;
- expected effect;
- motivating evidence IDs.

`learningcycle.ValidatePlan` rejects plans that violate one-primary-mutation policy.

## Semantic and Build Manifests

`tlaloc.semantic-manifest.r1` is a canonical set of semantic facts (`key -> value`) plus program/payload hashes.

Origami returns `tlaloc.build-manifest.r1` containing artifact provenance, applied mutation(s) and `visible_semantics`.

Semantic facts must be generated from structured IR. A free-form image model is not an authority for exact rules, states or transitions.

## Semantic Parity Gate

`tlaloc-learn validate-parity` compares the expected semantic manifest to the Origami build manifest.

Only differences whose fact key equals an explicitly declared mutation target are allowed. Program/payload hash drift is never allowed by this gate.

Any unauthorized difference produces:

```text
UNAUTHORIZED_SEMANTIC_DRIFT
```

and the specimen must not reach a real VLM.

This gate exists specifically to prevent a repeat of the invalid R2 DeepSeek specimen where the rendered rule changed from `A=ACTIVE => B...` to `B=ACTIVE => B...`.

## Outcome Learner

Schemas:

- `tlaloc.outcome-assessment.r1`
- `tlaloc.knowledge-update.r1`

Input contains before/after snapshots, target assertion, changed modules and specimen validity.

Classification:

```text
SUCCESSFUL_CAUSAL_STEP
NO_IMPROVEMENT
REGRESSION
INVALID_EXPERIMENT
```

Rules:

- invalid specimen -> no model penalty and hypothesis is not rejected;
- previously passing assertion becomes failing -> regression;
- target assertion changes FAIL -> PASS without regression -> causal success;
- one changed module -> high causal confidence;
- more simultaneous changes reduce causal confidence.

`tlaloc-learn assess-outcome -record` may write the corresponding immutable `OUTCOME_LINK` once the change-event and post-change evidence IDs are supplied.

## Current temporal example

The current accumulated history should lead to:

```text
failure frontier:
TEMPORAL_EXECUTION_INCOMPLETE

MUTABLE:
EXECUTION_POLICY

PRESERVE:
TEMPORAL_GRAMMAR (when linked positive outcome metadata is present)
BOOT / ROSETTA / semantic reading through prompt-genome protection

REQUIRE:
PROGRAM_SHA
PAYLOAD_SHA
PROVENANCE
RAW_RESPONSE_IMMUTABILITY
SEMANTIC_PARITY_GATE after invalid-specimen history

AVOID:
GENERATIVE_REWRITE_OF_EXACT_SEMANTICS
```

Candidate synthesis for `EXECUTION_POLICY` produces isolated alternatives such as `EXECUTE_VISIBLE_RULES_TO_STABLE_R1`. It does not combine checkpoint relabeling or rule-grammar redesign into the same causal experiment.

## CLI

```bash
tlALOC_STORE=/path/to/state

# Inspect cumulative learning state
tlaloc-learn status -store "$TLALOC_STORE"

# Generate next guarded experiment plan and persist attempts
tlaloc-learn plan \
  -store "$TLALOC_STORE" \
  -baseline t2-temporal-grammar-visible-r1 \
  -program-sha <sha256> \
  -payload-sha <sha256> \
  -budget 3 \
  -record-attempts

# Compile master prompt for the model/context
tlaloc-prompt compile \
  -genome behavior-lab/profiles/prompt-genome-r1.json \
  -model deepseek \
  -relevant TEMPORAL_GRAMMAR,EXECUTION_POLICY \
  -max-chars 12000 \
  -out-text master-prompt.txt \
  -out-json master-prompt.json

# After Origami builds a candidate, block semantic drift
tlaloc-learn validate-parity \
  -candidate candidate.json \
  -expected expected-semantics.json \
  -build build-manifest.json

# After benchmark evaluation, classify and optionally persist outcome
tlaloc-learn assess-outcome \
  -request outcome-request.json \
  -record \
  -store "$TLALOC_STORE" \
  -parents <change-event-id>,<post-change-evidence-id>
```

## Model specificity

Prompt modules may declare `model_scopes`. Empty scope means transferable/global. A module with explicit model scopes is compiled only for matching targets. Evidence must still decide whether a model-specific override is useful; one DeepSeek observation cannot become a universal invariant.

## Maturity

Knowledge is represented with progressively stronger labels:

```text
HYPOTHESIS
OBSERVED_WIN
PROVISIONAL_WIN
REPLICATED_WIN
CROSS_MODEL_WIN
CANONICAL_CANDIDATE
```

R1 derives maturity conservatively from repeated linked outcomes. Cross-model/canonical status remains an evidence/promotion concern and is not granted merely by the prompt compiler.

## Hard boundaries

```text
MEMORY_EVENTS_ARE_IMMUTABLE
DERIVED_POLICY_IS_REBUILDABLE
ONE_EXPERIMENT_ONE_PRIMARY_MUTATION
PRESERVE_PRIOR_WINS_BY_DEFAULT
KNOWN_ANTI_PATTERNS_ARE_CHECKED_BEFORE_TRIAL
INVALID_SPECIMEN_NE_MODEL_FAILURE
EXACT_SEMANTICS_FROM_STRUCTURED_IR_ONLY
SEMANTIC_PARITY_BEFORE_REAL_MODEL
PROGRAM_AND_PAYLOAD_SHA_MUST_MATCH
FALSE_EXACT_ZERO
RAW_RESPONSE_RETAINED
MODEL_SPECIFIC_NE_GLOBAL
LEARNING_POLICY_NE_PROMOTION
TLALOC_RECOMMENDS_ORIGAMI_DECIDES
TONAL_FREEZES_ONLY_MATURE_COMPOSITIONS
```
