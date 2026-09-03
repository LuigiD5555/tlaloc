# Tlaloc Exocortex — R0

## Status

`EXPERIMENTAL_NOT_PROMOTED`. This document specifies the E0–E6 vertical
slice used by the T0 decomposition experiment. It does not change Tonal
policy, does not promote any Origami artifact, and does not alter P0, P1,
or P2-A.

## Purpose

P1 established `PARROT_MAX_SAFE_OPS_SEMANTIC/CONTRACT = 1`: one model
invocation should execute at most one cognitive opcode. P2-A profiled which
opcodes Parrot (LFM2-VL 1.6B) can carry at all, and under which visual and
output constraints.

Tlaloc does not try to make a small model into a general agent. Instead,
Tlaloc builds an external executive system around the model, derived
directly from the model's empirically measured capability profile. The
model becomes an executor; Tlaloc owns everything the model is bad at:
memory, sequencing, routing, input reduction, verification, and recovery.

This is the **Exocortex** hypothesis. T0 is its first falsification test.

## Architectural invariants (E0.1–E0.15)

| # | Invariant |
|---|---|
| E0.1 | Model != Agent. |
| E0.2 | A model invocation executes at most one cognitive opcode unless its empirical CapabilityProfile explicitly permits more. |
| E0.3 | Working memory belongs to Tlaloc (the Blackboard), never to the model's context. |
| E0.4 | Workflow sequence belongs to Tlaloc (the SwarmPlan DAG), never to the model. |
| E0.5 | Formatting and serialization are deterministic whenever a deterministic Tlaloque can do it. |
| E0.6 | A model result becomes an `Observation`, never automatically a `Fact`. |
| E0.7 | Every non-deterministic executor must have an empirical `CapabilityProfile`. |
| E0.8 | Tlaloc exposes only the minimum working set required for the current opcode (`WorkingSetBuilder`). |
| E0.9 | The smallest reliable executor wins: deterministic code > specialized deterministic worker > tiny specialized model > Parrot > larger general model, provided the candidate satisfies the capability contract. |
| E0.10 | Failure uses an explicit recovery policy (`FailurePolicy`, `JoinStrategy`). Never a blind retry loop. |
| E0.11 | Capability profiles are immutable runtime evidence artifacts; runtime execution cannot rewrite its own profile. |
| E0.12 | No model may promote an Observation to Fact by itself. Only a Verify Tlaloque, through the Blackboard, may do so. |
| E0.13 | No free-form planner is introduced in R0. |
| E0.14 | Goal compilation in R0 is recipe-based (fixed T0-A/T0-B conditions) and bounded. |
| E0.15 | The architecture reuses existing Tlaloc components wherever an equivalent exists. No duplicate registries, schedulers, blackboards, worker contracts, DAG representations, or verification systems. |

## Responsibility boundaries

```
ORIGAMI            what exists, where it exists, addressing, selective
                    unfolding, future recipes. Origami is the pixel/profile
                    and document-index authority (internal/pdfmemory).

TLALOC              what needs to happen, sequence, dependencies, memory,
                    routing, execution, recovery. (internal/tlaloque,
                    internal/blackboard, this internal/exocortex slice.)

TONAL               what is allowed: accepted profiles, executor limits,
                    mandatory verification, policy. Not implemented or
                    enforced in R0 (see docs/POLICY_AUTHORITY.md); the
                    Exocortex records the shape Tonal will later gate.

CAPABILITY PROFILE  what an executor can do, under which conditions,
                    measured limits. Immutable evidence, compiled from a
                    frozen experiment artifact (P2-A for Parrot).

MODEL ADAPTER       how an abstract opcode is presented to a specific
                    model: prepared input, fixed instruction text, output
                    contract, fallback — or CAPABILITY_CONTRACT_VIOLATION.

TLALOQUES           atomic operations: Region, Parrot, Numeric, Normalize,
                    Verify. Bounded, cannot self-promote (CLAUDE.md).

PARROT              one narrow cognitive operation per invocation.
```

## Runtime data flow

```
CapabilityGoal / fixed T0 recipe
        |
        v
Registry.ResolveGoal  ---------------->  SwarmPlan (DAG of SwarmNode)
        |                                        (internal/tlaloque)
        v
SwarmRunner.Run
        |
        +--> CapabilityRouter.Select (E4)
        |        picks smallest executor whose CapabilityProfile satisfies
        |        the opcode's contract for this operand
        |
        +--> WorkingSetBuilder (E3B)
        |        reduces Blackboard snapshot + Step to the minimal input
        |        the chosen executor needs (tight crop, one instruction)
        |
        +--> ModelAdapter (E3) [only for non-deterministic executors]
        |        translates (opcode, operand, profile) into a prepared
        |        request, or rejects with CAPABILITY_CONTRACT_VIOLATION
        |
        +--> Tlaloque.Execute -> CapabilityResponse{Output, Observations}
        |
        v
BlackboardRuntime.RecordNode  -->  blackboard.Entry (OBSERVATION, status
                                    UNVERIFIED by construction)
        |
        v
Verify Tlaloque (E6.5) reads Observations from the Blackboard snapshot and
may emit a blackboard.Fact (status VERIFIED / UNSUPPORTED), never the
producing model.
```

## Observation vs. Fact semantics

`internal/blackboard` already defines `Observation` (bounded structure a
worker may return) and `Entry` (immutable, content-addressed, the only
thing `SwarmRunner`/`BlackboardRuntime` may persist). R0 adds `Fact`
(`internal/blackboard/fact.go`) as a second, distinct entry payload:

- An `Observation` is raw executor output. It always starts life with
  effective status `UNVERIFIED`. A model (Parrot) may only ever produce
  Observations (E0.6, E0.12).
- A `Fact` is a typed, verified value: `fact_id`, `value` (typed JSON),
  `status` (`VERIFIED` | `UNSUPPORTED` | `UNKNOWN`), `derived_from`
  (observation IDs), and verification metadata. Only `PromoteFact`,
  called from the Verify Tlaloque, may create one, and it requires at
  least one real `Observation` as input — there is no path that lets a
  Tlaloque fabricate a Fact with no derivation.

Facts are stored as ordinary `blackboard.Entry` values (`Type` extended
with `EntryFact`) so no second store, schema, or snapshot format is
introduced (E0.15).

## CapabilityProfile role (E1)

`internal/exocortex.CapabilityProfile` is a generic, executor-agnostic
runtime contract compiled from an immutable evidence artifact — for Parrot,
`results/PARROT_MICRO_ISA_R0.json` from experiment `parrot-microisa-r0.1`
(P2-A). The compiler (`CompileParrotProfile`) is a pure, hash-checked
transform: it never invents numbers, and it preserves the experiment's own
distinctions rather than collapsing them:

- a formal, preregistered `max_safe_choice_width` (conservative Wilson
  rule) is kept separate from the wider `observed_tested_envelope` P2-A
  actually walked (e.g. SELECT_ONE showed no degradation through width 8);
  the router only ever enforces the formal rung, and the profile keeps the
  observed envelope alongside it so it is never silently reinterpreted as
  "Parrot supports only N choices".
- `RESPONSE_COLLAPSE_CONFIRMED` opcodes (`VISUAL_IDENTIFY`, `VISUAL_LOCATE`,
  `FOLLOW_ONE_REFERENCE`) are carried verbatim as collapse, never smoothed
  into an ordinary accuracy number, and the router treats them as
  `deployment_recommendation: EXTERNALIZE`.

Until a real P2-A artifact is supplied at that path, the compiler has
nothing to compile from and returns an error — R0 does not ship a
profile built from invented numbers.

## ModelAdapter role (E3)

Given `(opcode, operand, CapabilityProfile, ExecutionContext)`, the adapter
produces prepared input, a fixed instruction template, an output contract,
and a fallback — or refuses with `CAPABILITY_CONTRACT_VIOLATION` when the
requested operand (e.g. a 64-char full-page `READ_SHORT_TEXT`) falls
outside the profile's measured envelope for that opcode. No prompt is
optimized or widened at runtime; templates are fixed before T0 execution
(`internal/exocortex/prompts.go`).

## WorkingSetBuilder role (E3B)

Deterministic reduction from `(BlackboardRuntime snapshot, Step)` to the
minimal `CapabilityRequest.Input`/`Context` an executor needs: a tight crop
and one instruction, never workflow history, never unrelated Facts, never
the full page when a crop already exists.

## CapabilityRouter rule (E4)

`internal/exocortex.ProfileAwareScoring` implements `tlaloque.ScoringStrategy`
and is installed on the existing `Registry` via `SetSelectionStrategy` —
no second registry is created. It keeps the existing
`DefaultScoringStrategy` preference (deterministic > small parameter count)
and adds one more gate: a candidate whose `CapabilityProfile` says
`deployment_recommendation: EXTERNALIZE` or `DO_NOT_DEPLOY` for the
requested opcode scores `-1` (ineligible), so `ResolveGoal`/`SwarmRunner`
never silently route a collapsed capability to Parrot.

## Failure / recovery principle

R0 introduces no new failure machinery. `SwarmNode.FailurePolicy`
(`STRICT`/`TOLERATED`) and `JoinMode` (`ALL`/`ANY`/`QUORUM`), both already
in `internal/tlaloque`, are the only recovery vocabulary T0 uses. A
`ModelAdapter` contract rejection and a Verify Tlaloque `UNSUPPORTED`
verdict are both explicit, typed outcomes — never a retry loop.

## Relationship with Origami

Origami remains the pixel/profile and document-index authority
(`internal/pdfmemory`, `internal/canonicaldoc`). The real (non-oracle)
Region Tlaloque in T0-B calls `pdfmemory.Search`/`pdfmemory.ReadRegion`
directly rather than reimplementing lexical ranking or PDF layout
inside Tlaloc (E0.15).

## Relationship with Tonal

Not enforced in R0. `gatekeeper.json` already names `LuigiD5555/tonal` as
policy authority across all three repositories; the Exocortex records
`CapabilityProfile.deployment_recommendation` and
`CapabilityRouter` decisions in a shape Tonal can later read and gate, but
R0 does not call out to Tonal or implement enforcement.

## Relationship with existing Swarm infrastructure

Nothing below is duplicated; R0 adds thin, additive layers on top:

| Exocortex role | Existing type / package |
|---|---|
| CapabilityRouter | `tlaloque.Registry` + `Registry.ResolveGoal` + `Registry.SelectResult`, with a new `ScoringStrategy` installed via `SetSelectionStrategy` |
| Blackboard / working memory | `blackboard.Store` + `tlaloque.BlackboardRuntime` |
| Step / DAG | `tlaloque.SwarmNode` / `tlaloque.SwarmPlan` (opcodes are just `SwarmNode.Capability` strings) |
| Executor (Tlaloque) | `tlaloque.CapabilityWorker` |
| Sequencing / scheduler | `tlaloque.SwarmRunner` |
| Accounting | `runrecord.Record` |
| Verification / consensus primitive | `blackboard.Consolidate` (multi-observation majority), extended with single-path `PromoteFact` for the Verify Tlaloque |
| Model transport (Parrot) | `target.OpenAICompat.CompletePerception` |
| Deterministic locator (T0-B real Region Tlaloque) | `internal/pdfmemory` (`Search`, `Load`, `ReadRegion`) |
| Campaign doctor/prepare/run pattern | `internal/realcampaign` (mirrored, not copied, for the new T0 experiment) |

No new Registry, Blackboard, DAG, worker contract, or verification store is
introduced.
