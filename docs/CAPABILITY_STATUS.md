# Capability status — Tlaloc 6.0.0-alpha.18

Repository lifecycle: Tlaloc installs/uninstalls independently and can target Origami or unrelated behaviors. This file distinguishes development machinery, reference evidence, deployable artifacts and empirical support.

| Capability | Status | Notes |
|---|---|---|
| BehaviorSpec validation | R0 implemented | Declares desired behavior and invariants. |
| PromptIR | R0 implemented | Deterministic ordering/rendering for prompt artifacts. |
| Generic behavior lifecycle | R0 implemented | intent -> swarm/reference behavior -> distill -> target evaluation. |
| Tlaloque bounded-worker layer | R0 implemented | Small workers are experimental/reference components, not final deployment requirements. |
| Prompt-First Distillation R0 contract | **alpha.14 implemented** | `PROMPT_ONLY` is the portable default; richer deployment classes are explicit fallbacks. |
| Prompt-first artifact selector | **alpha.14 implemented** | Chooses the least demanding behaviorally valid deployment level. |
| Clean-target requirement | **alpha.14 contract + tests** | L0 evaluation may not inherit swarm traces, hidden sandbox/tool state or Tlaloc runtime state. |
| L0 no-tool/no-sandbox compatibility | **alpha.14 hard invariant** | Prompt-only means target LLM text interface only. |
| Behavioral fidelity vs trace similarity | **alpha.14 formalized** | Distillation targets behavior, not textual reproduction of a swarm trace. |
| Failure-to-regression conversion | **alpha.15 implemented for Native semantic trial** | Failed real behavior can become a deterministic regression instead of anecdotal evidence. |
| Native semantic response evaluator | **alpha.16 codec-aware** | Index/semantic response scoring now distinguishes a declared semantic decoder from an undeclared external decoder/file/binary dependency. |
| Native T2 index recovery gate | **alpha.15 implemented** | Visual/prompt candidates require >=.95 reference recovery under configured policy. |
| Declared semantic decoder rule | **alpha.16 implemented** | A self-declared codec such as S2 is valid; hidden/external decoder dependency remains a failure. |
| Semantic-to-exact escalation gate | **alpha.16 implemented** | Semantic questions fail when they unnecessarily route through bit extraction/decompression/exact mechanics. |
| `tlaloc-native-eval` CLI | **alpha.16 refined** | Evaluates recorded target output without an LLM judge and records declared codec discovery. |
| Origami Protocol interoperability evaluator | **alpha.16 implemented / real evidence pending** | Deterministic READ/WRITE/ROUNDTRIP/MULTIHOP structural evaluation. |
| S2/E2 codec discovery metrics | **alpha.16 implemented** | Measures whether target outputs identify the required semantic decoder/encoder. |
| Semantic preservation/drift evaluator | **alpha.16 implemented** | Tracks entities, relations, hierarchy, evidence, uncertainty, invented atoms and Jaccard-style structural drift. |
| Cross-model A→B→C evaluator | **alpha.16 implemented / real evidence pending** | Measures hop-to-hop/final drift and read/write success; synthetic fixtures validate only the evaluator. |
| `tlaloc-protocol-eval` CLI | **alpha.16 implemented** | Evaluates recorded protocol trials deterministically without another LLM as judge. |
| Tlaloque trace -> automaton distillation | **alpha.17 implemented** | Converts ordered Tlaloque action traces into deterministic portable `origami.automaton.r0` and `origami.temporal-program.r0` artifacts. |
| Temporal Native Benchmark R0 | **alpha.17 implemented / real evidence pending** | Scores perception, ROSETTA/protocol, semantic, temporal and exactness/honesty layers without an LLM judge. |
| Observable debug trace R0 | **alpha.17 implemented** | Targeted retries report only observable protocol checkpoints and failure frontiers, never chain-of-thought. |
| Targeted diagnostic retry | **alpha.17 implemented** | `diagnostic_question_ids` repeats only failed probes and excludes diagnostic trials from primary Native/R4 scores. |
| Persistent learning memory R0 | **alpha.17 implemented** | Immutable content-addressed evidence ledger under XDG state; preserves failures, successes, change attempts and outcomes across runs/upgrades. |
| Real/synthetic evidence separation | **alpha.17 hard invariant** | Synthetic evidence may validate machinery but cannot drive empirical promotion or adaptive focus. |
| Failure-pattern index / next debug target | **alpha.17 implemented** | Rebuildable aggregation identifies dominant real-model frontier and suggested next debugging target. |
| Experiment-history linkage | **alpha.17 implemented** | Links parent evidence -> candidate change -> post-change evidence/outcome without deleting old failures. |
| Adaptive Search R0 | **alpha.17 implemented** | Uses real failure frontiers to prioritize mutation families and candidate trial order while leaving final promotion scoring untouched. |
| Bounded historical search signal | **alpha.17 implemented** | Linked outcomes can adjust mutation priority only within +/-0.25 and cannot override the current real failure frontier. |
| Exploration floor | **alpha.17 hard invariant** | Every supported mutation family retains non-zero weight to avoid permanent search lock-in. |
| `tlaloc-temporal-bench` CLI | **alpha.17 implemented** | Runs layered benchmark, debug parsing and automatic persistence of real evidence unless explicitly disabled. |
| `tlaloc-learning-memory` CLI | **alpha.17 implemented** | Ingests/summarizes evidence and records change/outcome links. |
| `tlaloc-adaptive-search` CLI | **alpha.17 implemented** | Builds memory-guided plans and prioritizes/records pre-evidence candidate attempts. |
| Closed Experimental Loop R0 | **alpha.18 implemented / real-model run pending** | Executes baseline -> benchmark -> diagnostic retry -> memory -> adaptive queue -> candidate trials -> outcome links -> next plan. |
| `tlaloc-closed-loop` CLI | **alpha.18 implemented** | `validate`, `run`, and `example` commands; OpenAI-compatible multimodal endpoints including LM Studio. |
| Transport/semantic failure separation | **alpha.18 hard invariant** | HTTP/timeouts/malformed API responses are execution errors, not BOOT/ROSETTA/T2/model-semantic evidence. |
| Automatic targeted diagnostic execution | **alpha.18 implemented** | Only failed clean questions are retried, and only complete diagnostic retries become diagnostic benchmark evidence. |
| Closed-loop before/after outcome linkage | **alpha.18 implemented** | Baseline and candidate scores are linked to CHANGE_ATTEMPT and post-change observations in persistent memory. |
| Multi-generation candidate-bank execution | **alpha.18 implemented** | Recalculates the adaptive plan each generation and stops on budget or candidate-bank exhaustion. |
| External candidate build hook | **alpha.18 implemented** | Explicit `build_command` may render a candidate PNG; Tlaloc remains non-authoritative for Origami pixels. |
| Receiver swarm-trace distillation | experimental R0 implemented | Existing Origami receiver-specific distillation remains a target-specific implementation. |
| Receiver candidate tournament | experimental R0 implemented | Existing target-specific tournament retained. |
| Project-local Claude Code skills | R0 implemented | Development assets; not portable behavior output. |
| Tonal-owned `repo-flow` distribution | external / not Tlaloc-owned | Tonal may distribute cross-project workflow skills. |
| SkillIR / generated Claude Skills | not implemented | Explicit future capability. |
| OpenAI-compatible text transport | R0 implemented | LM Studio and compatible endpoints. |
| OpenAI-compatible multimodal transport | **alpha.18 closed-loop reused** | Sends only declared prompt/question/PNG for clean trials. |
| OpenAI-compatible Hybrid multimodal/tool loop | experimental implemented | Higher-level development/deployment machinery, not L0 baseline. |
| General model-weight training | not implemented | Current distillation/search operates on explicit behavior artifacts/traces. |
| Managed installer/uninstaller | R0 implemented | Tlaloc development environment installs independently; learning memory is deliberately preserved. |
| Origami Semantic Spine awareness | contract-known | Target-specific integration; Tlaloc does not redefine Origami. |
| Origami canonical visual profile awareness | alpha.13 contract-known | One canonical Origami aesthetic per profile version. |
| Origami Writer awareness | alpha.13 contract-known | Tlaloc can develop/test behaviors that feed Writer, but is not pixel authority. |
| Origami Protocol R0 awareness | **alpha.16 contract-known** | Tlaloc evaluates declared S*/E* behavior but does not own Origami protocol/profile semantics. |
| Origami temporal automaton/program awareness | **alpha.17 contract-known** | Tlaloc can distill/test temporal automata while Origami owns their canonical representation. |
| Origami perceptual channels | contract-known / runtime partial | Moire/phase, stereo/parallax, temporal and emergent candidates remain evidence gated. |
| Origami Fixed Carrier R2 PDF memory plane | experimental R1 implemented | Target-specific development/runtime support. |
| Perception transport variants | alpha.12 implemented | Original PNG, 75%, 50% and JPEG preview. |
| Cross-model perception campaign aggregation | alpha.12 implemented | Development evidence machinery, not portable baseline. |
| Origami Visual Evolution R0 | alpha.13 implemented | Searches evidence-backed profile/prompt candidates and recommends only. |
| Native semantic fitness in visual search | **alpha.15 implemented** | Adds native index/answer rates and zero undeclared mechanical/unverified-claim gates. |
| Memory-guided pre-evidence visual search | **alpha.17 implemented** | Orders what to test first; existing evidence-gated `tlaloc-visual-search` ranking remains final on the Tlaloc side. |
| Prime/modular/factorization visual search | candidate family | No canonical authority without Origami adoption. |
| Moire/phase/depth/temporal visual search | candidate families | Reveal reliability and UNKNOWN discipline required. |
| Canonical Origami profile promotion | Origami-owned external authority | Tlaloc can recommend; Origami decides. |
| Tonal multi-tool composition | external / optional | May combine Tlaloc, Blueprint Framework and other development tools; not part of prompt deployment. |

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

Alpha.18 operationalizes the alpha.17 pieces through one runner:

```text
baseline PNG
 -> clean Native/R4 VLM trials
 -> deterministic scoring
 -> retry only failed questions with Debug Trace R0
 -> persist real evidence
 -> calculate adaptive failure target
 -> prioritize candidate PNG bank
 -> record change attempts
 -> run selected candidates under the same models/questions
 -> persist candidate evidence
 -> link before/after outcome
 -> calculate next plan
 -> repeat within configured generation budget
```

The loop intentionally stops if no clean baseline trial can be completed. Transport failures are reported separately and are not inserted into semantic learning memory. Candidate diagnostic evidence is admitted only when the complete targeted retry succeeds at the transport layer.

Candidate rendering remains outside Tlaloc authority. A candidate may be pre-rendered or produced through an explicit external `build_command` hook. The hook does not make its output canonical Origami.

Real VLM campaign evidence is still pending until the runner is executed against actual endpoints.

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

No document should claim that a successful swarm automatically proves a prompt, that a tool-assisted trial proves L0 portability, that a synthetic interop fixture proves real cross-model interoperability, that adaptive memory changes promotion scoring, that a transport failure proves semantic failure, or that a Tlaloc recommendation changes a target project's canonical release.
