# Capability status — Tlaloc 6.0.0-alpha.21

Repository lifecycle: Tlaloc installs/uninstalls independently and can target Origami or unrelated behaviors. This file distinguishes development machinery, reference evidence, deployable artifacts and empirical support.

<!-- BEGIN GENERATED CLAIMS TABLE: do not edit; run python3 tools/claims.py generate -->
| Claim | Statement | Status | Evidence | Version introduced | Last checked | Notes |
|---|---|---|---|---|---|---|
| `TLALOC.BACKEND.MODEL_FAMILY` | Select and execute specialized runtime backends by model family under a common behavior contract. | `designed` | — | `unknown` | 2026-09-01 | Main can detect model families and vary compatibility metadata, but no tested family-specific backend contract exists. |
| `TLALOC.BEHAVIOR_LAB.THREE_LEVEL` | Operate distinct Model Lab, Composition Lab, and Design Lab levels with promotion gates between them. | `designed` | — | `unknown` | 2026-09-01 | The repository contains behavior-lab machinery, but no test demonstrates the complete three-level contract described in the historical architecture. |
| `TLALOC.COMPILER.BEHAVIOR_PIPELINE` | Compile a valid BehaviorSpec into PromptIR and render a deterministic prompt without embedding target-specific laws in the compiler. | `implemented` | `test:behavior-lab/internal/compiler:TestCompileUsesBehaviorSpecWithoutHardcodedProfileLaw` | `6.0.0-alpha.6` | 2026-09-01 | The test exercises BehaviorSpec validation, PromptIR construction, and prompt rendering as one observable compilation contract. |
| `TLALOC.DESIGN.MDP` | Use a Markov decision process with explicit actions and rewards to select runtime or design policies. | `designed` | — | `unknown` | 2026-09-01 | Published main has deterministic and empirical selection policies, not a tested MDP implementation. |
| `TLALOC.IR.SKILL` | Compile behavior into SkillIR and generate portable agent skills from that intermediate representation. | `designed` | — | `unknown` | 2026-09-01 | docs/CAPABILITY_STATUS.md explicitly marks SkillIR and generated skills as not implemented. |
| `TLALOC.PERCEPTION.NATIVE_CHANNEL_EXECUTION` | Execute all seven Origami perceptual-channel operations as native Tlaloc runtime capabilities. | `designed` | — | `6.0.0-alpha.7` | 2026-09-01 | Main contains transport, evaluators, and partial runtime support, but no test demonstrates native execution of all seven declared channel operations. |
| `TLALOC.PERCEPTION.TRANSPORT_AND_EVALUATION` | Run perceptual transport variants and apply deterministic, evidence-gated campaign evaluation without exposing evaluator ground truth to the model. | `implemented` | `test:behavior-lab/internal/promotion:TestBuildTransportVariants`<br>`test:behavior-lab/internal/promotion:TestObservationQuestionDoesNotContainExpectedProbeOrCarrierTruth`<br>`test:behavior-lab/internal/promotion:TestNativeRequiresThreeModelsThreeOriginalTrials`<br>`test:behavior-lab/internal/nativeeval:TestFailedIndexTrialIsRejected` | `6.0.0-alpha.12` | 2026-09-01 | The machinery is implemented; this claim does not assert empirical support for any perceptual channel. |
| `TLALOC.PROFILE.ORIGAMI_QUANTUM_INSPIRED_R0` | Provide the origami.quantum-inspired.r0 behavior profile and execute its deterministic coherent-state reference semantics. | `implemented` | `test:behavior-lab/internal/reference:TestInterferenceCancellation`<br>`test:behavior-lab/internal/reference:TestTransformPreservesAlternatives`<br>`test:behavior-lab/internal/reference:TestCoupledIsJointState` | `6.0.0-alpha.6` | 2026-09-01 | The profile exists at behavior-lab/profiles/origami/quantum-inspired-r0.json; the tests cover its core reference invariants. |
| `TLALOC.PROMOTION.CENTRAL_AUTHORITY` | Keep promotion decisions outside proposal workers and reject promotion based only on mock evidence or an experimental mutation. | `implemented` | `test:behavior-lab/internal/adaptivesearch:TestPrioritizeQueueDoesNotClaimPromotion`<br>`test:behavior-lab/internal/visualsearch:TestMutationMustRemainExperimentalBeforePromotion`<br>`test:behavior-lab/internal/promotion:TestMockCannotSatisfyCrossModelPromotion` | `6.0.0-alpha.12` | 2026-09-01 | Tlaloc may evaluate and recommend; target and stack promotion remain external authority boundaries. |
| `TLALOC.REFERENCE.DETERMINISTIC_SEMANTICS` | Evaluate coherent-state operations against deterministic reference semantics that preserve alternatives, cancellation, and coupled state. | `implemented` | `test:behavior-lab/internal/reference:TestInterferenceCancellation`<br>`test:behavior-lab/internal/reference:TestTransformPreservesAlternatives`<br>`test:behavior-lab/internal/reference:TestCoupledIsJointState` | `6.0.0-alpha.6` | 2026-09-01 | This claim covers deterministic reference behavior, not empirical model capability. |
| `TLALOC.REPAIR.BOUNDED_PROPOSALS` | Use bounded Tlaloque findings to propose and apply prompt repairs across a finite number of generations. | `implemented` | `test:behavior-lab/internal/tlaloque:TestTrainerCanRepairPrematureCollapse` | `6.0.0-alpha.6` | 2026-09-01 | The trainer has an explicit generation bound and repairs a demonstrated premature-collapse failure. |
| `TLALOC.RUNTIME.BLACKBOARD` | Persist immutable run-scoped worker observations, preserve conflicts, and consolidate only when the declared evidence threshold is met. | `implemented` | `test:behavior-lab/internal/blackboard:TestContentIDStableAndAppendIdempotent`<br>`test:behavior-lab/internal/blackboard:TestAppendPublishesOneAtomicEntryUnderConcurrency`<br>`test:behavior-lab/internal/blackboard:TestConsolidateRequiresTwoThirdsAndPreservesConflicts`<br>`test:behavior-lab/internal/blackboard:TestConsolidateTieOrContractViolationIsUnknown` | `6.0.0-alpha.21` | 2026-09-01 | Implemented on published main after the original evidence plan described this area as historical design. |
| `TLALOC.RUNTIME.CANONICAL_STATE_FUSION` | Reduce evidence-bearing candidates into canonical state using explicit cardinality and replaceable deterministic fusion policies while retaining conflicts. | `implemented` | `test:behavior-lab/internal/canonicalstate:TestReducerRejectsEvidenceFreeCandidates`<br>`test:behavior-lab/internal/canonicalstate:TestCardinalityOneConflictsDistinctPositiveValues`<br>`test:behavior-lab/internal/canonicalstate:TestPolicyFusionIsCandidateOrderIndependent`<br>`test:behavior-lab/internal/canonicalstate:TestReducerAllowsFusionStrategyReplacement` | `6.0.0-alpha.21` | 2026-09-01 | Candidate observations remain distinct from canonical facts. |
| `TLALOC.RUNTIME.EMPIRICAL_COMPLEMENTARITY_SELECTION` | Build condition-specific empirical failure profiles and use sufficient shared evidence to select complementary ensemble members deterministically. | `implemented` | `test:behavior-lab/internal/learningmemory:TestBuildEmpiricalProfilesMeasuresFailureComplementarity`<br>`test:behavior-lab/internal/learningmemory:TestEmpiricalProfilesIgnoreSyntheticAndUnaddressableCases`<br>`test:behavior-lab/internal/tlaloque:TestEmpiricalStrategyPrefersComplementaryWeakerMember`<br>`test:behavior-lab/internal/tlaloque:TestEmpiricalStrategyRequiresEnoughSharedEvidence` | `6.0.0-alpha.21` | 2026-09-01 | Selection falls back to the baseline strategy when empirical evidence is insufficient. |
| `TLALOC.RUNTIME.ENSEMBLE_COMPOSITION` | Build and execute pinned multi-worker ensemble plans with an explicit fusion boundary and declared quorum or all-member behavior. | `implemented` | `test:behavior-lab/internal/tlaloque:TestResolveEnsembleBuildsPinnedQuorumPlan`<br>`test:behavior-lab/internal/tlaloque:TestQuorumEnsembleSucceedsWhenOptionalMemberFails`<br>`test:behavior-lab/internal/tlaloque:TestAllEnsembleFailsAtFusionBoundaryWhenMemberFails` | `6.0.0-alpha.21` | 2026-09-01 | This claim covers executable composition, not empirical superiority over a single worker. |
| `TLALOC.RUNTIME.JOIN_AND_FAILURE_POLICIES` | Execute ALL, ANY, and QUORUM dependency joins with explicit node state transitions and tolerated-failure policies. | `implemented` | `test:behavior-lab/internal/tlaloque:TestNodeStateMachineRejectsIllegalTransition`<br>`test:behavior-lab/internal/tlaloque:TestSwarmAnyJoinStartsOnFirstSuccessfulDependency`<br>`test:behavior-lab/internal/tlaloque:TestSwarmQuorumJoinStartsAfterThreshold`<br>`test:behavior-lab/internal/tlaloque:TestSwarmPlanRejectsInvalidQuorum` | `6.0.0-alpha.21` | 2026-09-01 | Join semantics are recorded separately from typed product resolution because either contract can regress independently. |
| `TLALOC.RUNTIME.TYPED_DATAFLOW` | Resolve Requires and Produces contracts into a typed dependency DAG and expose products by contract rather than producer identity. | `implemented` | `test:behavior-lab/internal/tlaloque:TestResolveGoalMaterializesTypedProductDAG`<br>`test:behavior-lab/internal/tlaloque:TestResolveGoalFailsWhenRequiredProductHasNoProducer`<br>`test:behavior-lab/internal/tlaloque:TestRunnerExposesTypedProductInsteadOfProducerIdentity`<br>`test:behavior-lab/internal/tlaloque:TestSwarmPlanRejectsBindingOutsideDependencies` | `6.0.0-alpha.21` | 2026-09-01 | Published main contains executable typed dataflow despite the historical plan classifying Causal Dataflow as designed. |
| `TLALOC.TLALOQUE.AUTO_CREATION` | Automatically design, train, benchmark, and promote a new Tlaloque specialist from an unmet capability goal. | `designed` | — | `unknown` | 2026-09-01 | Main resolves and composes registered workers but does not test the complete auto-creation and promotion lifecycle. |
| `TLALOC.TRANSPORT.ANTHROPIC_NATIVE` | Execute behavior requests through a native Anthropic transport backend. | `designed` | — | `unknown` | 2026-09-01 | No native Anthropic implementation or associated test exists on published main. |
| `TLALOC.TRANSPORT.OPENAI_MULTIMODAL` | Execute declared image and tool-loop requests through OpenAI-compatible multimodal transports. | `implemented` | `test:behavior-lab/internal/target:TestCompletePerceptionUsesLMStudioImageDetailStrategy`<br>`test:behavior-lab/internal/target:TestCompleteHybridUsesLMStudioStrategyExecutesToolAndReturnsAnswer`<br>`test:behavior-lab/internal/target:TestMultimodalCompatibilityStrategies` | `6.0.0-alpha.12` | 2026-09-01 | A successful transport exchange is not a semantic benchmark pass. |
| `TLALOC.TRANSPORT.OPENAI_TEXT` | Execute text requests through an OpenAI-compatible endpoint while preserving the declared behavior input and output boundary. | `implemented` | `test:behavior-lab/internal/target:TestCompleteHybridTextBridge` | `6.0.0-alpha.6` | 2026-09-01 | Transport compatibility is implementation evidence, not evidence that a remote model satisfies a behavior. |
<!-- END GENERATED CLAIMS TABLE -->

## Prompt-First Distillation R0

The canonical development loop is:

```text
intent
 -> BehaviorSpec / invariants
 -> Tlaloque swarm of bounded steps
 -> successful reference execution
 -> distillation
 -> prompt candidates
 -> clean target-model trials
 -> behavioral-fidelity comparison
 -> least demanding valid artifact
```

Deployment ladder:

```text
L0 PROMPT_ONLY
L1 PROMPT_PLUS_DECLARATIVE_CONTEXT_OR_IR
L2 PROMPT_PLUS_TOOLS
L3 PROMPT_PLUS_RUNTIME
L4 SPECIALIZED_MODEL_OR_TARGET_SPECIFIC_SYSTEM
```

The current deterministic selector defaults to behavioral fidelity >= .95, pass rate >= .95, regression rate <= .01 and at least 3 clean-target trials. Project-specific BehaviorSpecs may impose stricter requirements.

A richer runtime does not outrank a valid L0 prompt merely because its fidelity is marginally higher. Tlaloc minimizes deployment requirements subject to the required behavior.

## Native semantic regression R0

The first failure-driven Native regression comes from an external Origami trial that could read BOOT but failed the index question. The response requested binary/file decoding and emitted unverified byte, compression and hash information.

Alpha.15 made that failure testable. Alpha.16 refines the interpretation now that Origami can declare semantic codecs:

```text
DECLARED S2 DECODER = ALLOWED
UNDECLARED EXTERNAL DECODER / FILE / BINARY DEPENDENCY = FAILURE
SEMANTIC QUERY -> UNNECESSARY EXACT/BINARY ROUTE = FAILURE
```

Visual-search candidates retain reference gates for index/semantic recovery, zero dependency/escalation violations, zero unverified mechanical claims and `FALSE_EXACT=0`.

These are development/recommendation gates. They do not claim that a corrected carrier has already passed held-out VLM trials.

## Origami Protocol Interop R0

Alpha.16 adds deterministic protocol evaluation for declared S*/E* read/write behavior, structural semantic preservation and A->B->C multi-hop drift. Synthetic fixtures validate the harness only. Real-model interoperability remains evidence pending.

## Temporal learning loop R0

Alpha.17 adds a persistent feedback cycle:

```text
real trial
 -> layered benchmark
 -> targeted observable debug retry when needed
 -> failure frontier
 -> immutable learning-memory event
 -> real failure-pattern aggregation
 -> next debug target
 -> adaptive search plan
 -> prioritized experimental candidate queue
 -> real trials
 -> ordinary evidence-gated tournament
 -> outcome linked back to memory
```

The memory stores both successes and failures. Fixing a failure does not delete it; the old event remains available for regression testing. Adaptive search may use linked historical outcomes to adjust experiment budget, but memory does not alter final candidate score or promotion gates.

```text
MEMORY PRIORITY != PROMOTION SCORE
```

## Closed Experimental Loop R0

Alpha.18 introduced the config-driven operational runner. Alpha.19 closes the inter-generation loop. Alpha.20 removes the remaining manual candidate-provisioning requirement when an explicit target-owned builder supports the requested mutation family.

```text
current experimental incumbent PNG
 -> clean Native/R4 VLM trials
 -> deterministic scoring
 -> retry only failed questions with Debug Trace R0
 -> persist real evidence
 -> derive active failure frontier from current incumbent
 -> Adaptive Search SuggestedMutations
 -> query target-owned candidate builder capabilities
 -> filter unsupported mutation families
 -> generate deterministic one-mutation CandidateConfigs
 -> delegate PNG construction to target-owned builder
 -> run selected candidates under the same models/questions
 -> persist candidate evidence
 -> link before/after outcome
 -> require no per-question/exactness regression + minimum improvement
 -> best passing candidate becomes next experimental incumbent
 -> recalculate the newly exposed failure frontier
 -> repeat
```

Transport failures are reported separately and are not inserted into semantic learning memory. A failed transport does not advance the incumbent.

Automatic candidate generation is opt-in. Manual candidate banks and explicit per-candidate `build_command` hooks remain supported. Unsupported builder capabilities are filtered before spending model trials, and Tlaloc never substitutes its own pixel approximation.

The experimental incumbent has no canonical authority. The alpha.20 end-to-end fixture proves orchestration only; real VLM campaign evidence remains pending until the runner is executed against actual endpoints.

## Development vs deployment

Tlaloc may use large swarms, sandboxes, Go utilities, tools, evaluators and many models during development. Those resources are not automatically inherited by the distilled artifact.

An L0 prompt cannot claim portability if its success depends on:

```text
private swarm trace
development sandbox
undeclared tools
evaluator ground truth
Tlaloc runtime state
```

## Origami as one target

Existing Origami work remains intact:

```text
Canonical Memory R2
Perception Promotion Campaign R1
Origami Visual Evolution R0
Native Semantic Regression R0
Origami Protocol Interop R0
Temporal Native Benchmark R0
Adaptive Search R0
Closed Experimental Loop R0
Auto Candidate Generation R0
prompt/representation experiments
```

These are target-specific development tracks. The same Tlaloc core can instead be used for a calculator, classifier, document workflow or other behavior.

For Origami:

```text
Tlaloc experiments + evidence
 -> recommendation
 -> Origami validates/decides its own next version
```

Tonal may optionally record a reproducible multi-tool development composition afterward.

No document should claim that a successful swarm automatically proves a prompt, that a tool-assisted trial proves L0 portability, that a synthetic interop fixture proves real cross-model interoperability, that adaptive memory changes promotion scoring, that a transport failure proves semantic failure, that an experimental incumbent is canonical, that a generated candidate owns target pixels, or that a Tlaloc recommendation changes a target project's canonical release.