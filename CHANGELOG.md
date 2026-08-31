# Tlaloc changelog

## 6.0.0-alpha.20 — Auto Candidate Generation R0

- closes the remaining candidate-provisioning gap in the alpha.19 closed experimental loop;
- adds opt-in automatic conversion of adaptive `SuggestedMutations` into deterministic one-mutation CandidateConfigs;
- adds explicit candidate-builder capability negotiation before spending model trials;
- filters unsupported mutation families rather than approximating target pixels inside Tlaloc;
- derives automatic candidate identity from parent specimen ID + parent PNG SHA-256 + canonical mutation;
- requires automatic builders to support the configured parent profile and declare `exact_plane_mutation=false`;
- reuses the alpha.19 `TLALOC_*` parent-aware build-hook contract and all existing clean-trial, diagnostic, learning-memory, non-regression and incumbent-advancement gates;
- preserves manual candidate banks and per-candidate build commands when `auto_candidates=false`;
- adds `tlaloc.auto-candidate-generation.r0`, docs and CI contract gates;
- adds a synthetic fake-VLM/fake-builder end-to-end regression proving generation -> build -> evaluation -> memory -> experimental incumbent orchestration without claiming real-model evidence;
- preserves authority: Tlaloc generates mutation intent and experimental order; target-owned builders compile pixels; canonical Origami promotion remains Origami-owned.

## 6.0.0-alpha.19 — Incumbent Closed Experimental Loop R0

- makes the best improved non-regressing candidate the next **experimental incumbent** rather than resetting every generation to the original baseline;
- recalculates the active failure frontier from the current incumbent run so resolved failures stop dominating current search while remaining in regression history;
- adds per-question non-regression checks, missing-question guards and invented-exact-claim guards before incumbent advancement;
- adds `parent_specimen_id` candidate DAG semantics so staged candidates become eligible only when their parent is the active incumbent;
- passes current parent specimen ID/PNG to external candidate build hooks;
- keeps historical outcomes as bounded adaptive-search signal without making them promotion score;
- allows cross-run retesting while suppressing duplicate candidate execution within one closed-loop run;
- adds multi-generation regression coverage for incumbent advance and moving failure frontiers;
- preserves `EXPERIMENTAL_INCUMBENT != CANONICAL_ORIGAMI_PROFILE`.

## 6.0.0-alpha.18 — Closed Experimental Loop R0

- adds config-driven `tlaloc-closed-loop` execution against OpenAI-compatible multimodal endpoints;
- connects clean incumbent trials, deterministic Temporal Native Benchmark scoring, targeted diagnostic retries, persistent learning memory, adaptive candidate ordering, candidate trials and before/after outcome linkage;
- separates transport/execution failures from BOOT/ROSETTA/T2/model-semantic evidence;
- retries only failed questions in diagnostic mode and excludes diagnostics from clean Native/R4 scores;
- adds explicit external candidate `build_command` support while keeping candidate rendering outside Tlaloc authority;
- stops safely when no clean incumbent trial completes or no eligible candidate remains;
- writes per-generation campaign/result/plan/queue artifacts plus a top-level closed-loop report;
- keeps canonical Origami promotion external and evidence-gated.

## 6.0.0-alpha.17 — Temporal Learning Memory + Adaptive Search R0

- adds deterministic Tlaloque trace -> Origami automaton/temporal-program distillation;
- adds Temporal Native Benchmark R0 with perception, ROSETTA/protocol, semantic, temporal and exactness/honesty layers;
- adds observable Debug Trace R0 for targeted failed-question retries without requesting private chain-of-thought;
- adds immutable content-addressed Learning Memory R0 for observation, change-attempt and outcome events;
- separates real-model and synthetic evidence so synthetic fixtures cannot silently drive empirical promotion/adaptive focus;
- adds failure-pattern indexing and next-debug-target selection from persistent real-model evidence;
- adds Adaptive Search R0 using failure frontiers plus bounded historical outcome adjustment while preserving an exploration floor;
- adds `tlaloc-temporal-bench`, `tlaloc-learning-memory` and `tlaloc-adaptive-search` managed CLIs;
- formalizes `MEMORY_PRIORITY != PROMOTION_SCORE` and preserves Tlaloc recommendation vs Origami authority.

## 6.0.0-alpha.16 — Origami Protocol Interoperability R0

- adds deterministic READ/WRITE/ROUNDTRIP/MULTIHOP evaluation for Origami Protocol R0;
- adds `tlaloc-protocol-eval` and `behavior-lab/internal/protocoleval` without using another LLM as judge;
- measures declared S2/E2 codec discovery, semantic preservation, invented facts, hop-to-hop/final semantic drift and cross-model read/write success;
- compares entities, relations, hierarchy, evidence and uncertainty through canonical structural atoms;
- refines the alpha.15 Native regression so a self-declared semantic decoder such as `S2` is valid while undeclared external decoder/file/binary dependency remains a failure;
- adds a separate semantic-to-exact escalation violation for unnecessary bit extraction/decompression/exact mechanics on semantic questions;
- adds `ORIGAMI_PROTOCOL_INTEROP_R0`, `CODEC_ROUNDTRIP_R0` and `CROSS_MODEL_COMMUNICATION_R0` development contracts;
- adds synthetic perfect-multihop fixtures strictly for evaluator validation and a separate real-trial template for held-out evidence;
- explicitly keeps real cross-model interoperability evidence pending; synthetic harness success is not promotion evidence;
- preserves Tlaloc's authority boundary: Tlaloc evaluates/recommends, Origami owns protocol/profile promotion, Tonal may later pin a reproducible composition.

## 6.0.0-alpha.15 — Native Semantic Regression R0

- turns a failed real prompt-only/multimodal Origami index trial into a deterministic regression instead of discarding it as anecdotal evidence;
- adds `tlaloc.native-semantic-regression.r0` and a non-LLM `nativeeval` scorer;
- adds `tlaloc-native-eval` for evaluating recorded target output from the command line;
- adds first-class `native_index_recovery_rate` and `native_semantic_answer_rate` to Origami visual/profile search;
- blocks candidate recommendations when semantic navigation requires undeclared binary/file/sandbox/decompression access;
- blocks unverified byte/hash/compression/archive claims in prompt-only semantic trials;
- preserves exact/mechanical capabilities for questions that actually require exact recovery and explicitly declare those capabilities;
- adds Native semantic fitness to candidate scoring so smaller/denser/faster visual changes cannot win by making semantic navigation unusable;
- preserves Prompt-First Distillation: the corrected target behavior must still work at the declared deployment level rather than inheriting hidden development machinery;
- installs/uninstalls `tlaloc-native-eval` with the managed Tlaloc development toolchain;
- does not claim the corrected Origami carrier has already passed held-out real-model trials; Tlaloc measures/recommends and Origami owns adoption.

## 6.0.0-alpha.14 — Prompt-First Distillation R0

- redefines Tlaloc's core as a behavioral development kit rather than an Origami-centered runtime;
- formalizes the Tlaloque swarm as a development/reference laboratory used to discover working procedures;
- makes prompt-only deployment the portable default for target LLMs with no sandbox, tools, Go/Python runtime, file access or Tlaloc installation;
- adds deployment ladder `L0 PROMPT_ONLY -> L1 prompt+context/IR -> L2 prompt+tools -> L3 prompt+runtime -> L4 specialized target`;
- adds deterministic least-demanding-artifact selection under behavioral-fidelity/pass/regression/clean-target gates;
- formalizes `Behavior(candidate) ~= Behavior(reference swarm)` as the distillation objective instead of trace-text similarity;
- requires clean-target evaluation and forbids hidden development dependencies in L0 candidates;
- adds `tlaloc.prompt-first-distillation.r0`, `docs/PROMPT_FIRST_R0.md`, Go reference selection logic and CI gates;
- clarifies Origami as one possible development target and Tonal as an optional multi-tool composition layer.

## 6.0.0-alpha.13 — Origami Visual Evolution R0

- adds evidence-gated search over candidate Origami prompt, primitive, layout, redundancy, color, numeric, interference, depth, temporal and emergent representation changes;
- adds prime/modular/factorization and perceptual-channel candidates without granting them canonical authority;
- measures semantic roundtrip, routing/evidence, transport robustness, perceptual reveal reliability, semantic units per byte, recognition time, bootstrap/decode steps, context and carrier size;
- requires real-model evidence for perceptual candidates and preserves `UNKNOWN` when reveal conditions are not established;
- adds `tlaloc-visual-search` and managed installation for both visual search and perception campaign CLIs;
- preserves the authority boundary: Tlaloc recommends with evidence; Origami decides whether Origami changes.

## 6.0.0-alpha.12 — Perception Promotion Campaign R1

- adds deterministic generation of original, 75%, 50% and JPEG-preview visual transports from one canonical Origami carrier;
- adds a strict OpenAI-compatible multimodal perception call that gives target models only the declared prompt, question and image;
- adds structured observation parsing that rejects undeclared evaluator/ground-truth fields;
- bridges every trial to Origami's independent `origami-perception-eval` instead of letting Tlaloc score its own visual ground truth;
- adds campaign aggregation requiring REAL_MODEL evidence, 3 models x 3 original trials by default, transport coverage, real tool-loop success and held-out routing evidence;
- preserves `FALSE_EXACT=0`, zero budget violations and evidence-routing thresholds as hard promotion gates;
- separates `HYBRID_SUPPORTED_CANDIDATE` from `NATIVE_VISUAL_SUPPORTED_CANDIDATE`; Hybrid does not require Native T3;
- reuses the existing alpha.11 `CompleteHybrid` + Fixed Origami executor path for real tool-loop evidence;
- adds `tlaloc-perception-campaign` campaign runner machinery;
- does not fabricate external VLM results.

## 6.0.0-alpha.11 — Origami Fixed Carrier R2 PDF memory plane

- adds deterministic PDF Tlaloque memory ingestion with exact source preservation, 100% page addressing/CIDs, lexical routing, fixed GraphSketch and Merkle root;
- adds `tlaloc-origami compile|boot|query|expand|verify|chat`;
- generates the R1 Master Prompt instead of requiring manual prompt editing;
- binds the supplied `origami.png` to the exact store through the independent `origami-fixed-carrier` decoder;
- adds native OpenAI-compatible function-tool loop support for R1 plus a plain-text `<ORIGAMI_CALL>` fallback;
- treats OCR failure as irrelevant to BOOT and returns explicit capability failures instead of fabricated memory;
- installs `tlaloc-origami` as an independent Tlaloc CLI;
- preserves Tlaloc/Origami ownership separation and `FALSE_EXACT=0`.

## 6.0.0-alpha.10 — Repo-flow ownership migration

- removes the project-agnostic `repo-flow` skill from Tlaloc now that Tonal is its canonical distribution owner;
- keeps the five Tlaloc/Origami-specific project skills under `.claude/skills/`;
- keeps `tlaloc skills list/path/install` for Tlaloc-owned skills only;
- `tlaloc skills install repo-flow` now fails with an explicit migration message pointing to Tonal instead of installing a stale copy;
- replaces the old repo-flow-specific Tlaloc regression with project-skill installation coverage and asserts managed Tlaloc installs no longer contain `repo-flow`;
- adds `.github/workflows/verify.yml` so pull requests and `main` pushes enforce release/version, terminology, skills, isolated install, Go test/vet/race, and generated-artifact hash gates;
- no BehaviorSpec, PromptIR, Tlaloque, reference semantics, Origami contracts, generated prompt, or model-facing runtime behavior changed.

## 6.0.0-alpha.9 — Reusable repository workflow skill

- added the project-agnostic `repo-flow` skill for Git/GitHub repository work: preflight, branch discipline, impact-scoped verification, atomic commits, PR review, conflict resolution, CI gating, merge and post-merge verification;
- codified release/version consistency, changelog/change-control hygiene and multi-repository snapshot/pin rules without making submodules a user-project requirement;
- added `tlaloc skills list`, `tlaloc skills path`, and `tlaloc skills install <name>`;
- `tlaloc skills install repo-flow` installs into the current Git repository root and refuses to overwrite differing local content unless `--force` is explicit;
- added regression coverage for skill discovery, idempotent installation, local-edit protection and explicit forced replacement;
- no BehaviorSpec, PromptIR, Tlaloque, reference semantics, Origami contracts, or model-facing runtime behavior changed.

## 6.0.0-alpha.8 — Release/version coherence

- fixed `install.sh` still installing `6.0.0-alpha.6` while the repository declared a newer release;
- made root `VERSION` the installer version source instead of duplicating a hard-coded value;
- added an installation regression gate that requires the managed version path/marker and `tlaloc version` to match root `VERSION`;
- synchronized active integration guidance with Origami `6.0.0-alpha.3`, which preserves the alpha.2 perceptual contract while clarifying `Origami > OHF`;
- updated active release documentation without changing BehaviorSpec, PromptIR, Tlaloque, reference semantics, or executable Origami integration behavior.

## 6.0.0-alpha.7 — Origami perceptual-contract tracking

- tracks Origami `6.0.0-alpha.2` and `origami.perceptual-channels.r0` as an upstream semantic contract;
- updates the Origami project skill with interference/moiré, depth/stereo/parallax, Temporal Latent Image and temporal/emergent terminology;
- distinguishes contract awareness from executable runtime support;
- explicitly marks `MOIRE`, `PHASE_SHIFT`, `STEREO_BIND`, `PARALLAX_RESOLVE`, `KINETIC_REVEAL`, `TEMPORAL_INTEGRATE`, and `TEMPORAL_DECAY` as not yet implemented by Tlaloc evaluators/Tlaloque;
- preserves the current coherent-state behavior profile and generated prompt unchanged.

## 6.0.0-alpha.6 — Independent repository lifecycle

- Tlaloc repository becomes independently installable from source.
- `install.sh` no longer installs or requires Origami.
- `doctor` treats Origami as optional instead of a required installation.
- direct `uninstall.sh` defaults to Tlaloc-only removal; bundle/origami modes remain explicit for retrocompatibility.
- legacy Origami/OHF/VCL cleanup remains available without transferring ownership of Origami to Tlaloc.
- BPFW/PipeCraft and `.me/origami` remain hard-protected.

## 6.0.0-alpha.5 — Agent guidance + lifecycle retrocompatibility audit

- added project `CLAUDE.md` guidance;
- added five project-local Claude Code skills under `.claude/skills/`;
- explicitly separated checked-in project skills from future compiler-generated SkillIR/Claude Skills;
- added `tlaloc skills-path`;
- added release validation for skill metadata and mirrored copies;
- fixed stale active documentation (`Swarm Trainer`, alpha.3 compatibility wording);
- added detection/removal of the obsolete alpha.2 `XDG_STATE_HOME/tlaloc/install-manifest-v1.tsv`;
- expanded conservative legacy scanning to additional user integration/completion artifacts;
- strengthened generation-specific retro-uninstall tests.

Origami remains `6.0.0-alpha.1`; no Origami state semantic law changed.

## 6.0.0-alpha.4 — Tlaloque nomenclature + reference semantics cleanup

### Changed
- renamed the bounded-agent implementation from `internal/swarm` to `internal/tlaloque`;
- introduced `Tlaloque` as the official name for Tlaloc bounded specialist agents;
- renamed deterministic expected-state machinery from `internal/oracle` to `internal/reference`;
- removed Oracle/Adivino/Báculo from current architecture terminology;
- added `docs/NOMENCLATURE.md`;
- added `tlaloc tlaloque` introspection;
- updated active Tlaloc/Origami integration documents to the new naming contract.

### Versioning
Tlaloc advances to `6.0.0-alpha.4`. Origami remains `6.0.0-alpha.1`; its representation semantics are unchanged.

## 6.0.0-alpha.3 — Concept/documentation depuration R0

### Corrected
- separated current architecture docs from historical records;
- removed mandatory-dependency wording between Tlaloc and Origami;
- clarified Origami reference semantics vs Tlaloc evaluation authority;
- removed global `Master Prompt` terminology from current architecture;
- documented actual model support without claiming native Claude/skills support;
- moved the bundled Origami behavior contract into an explicit profile namespace;
- regenerated the stale compiled prompt and artifact hash;
- introduced component-specific managed-install markers with alpha.2 compatibility.

### Versioning
Tlaloc advances to `6.0.0-alpha.3`. Origami remains `6.0.0-alpha.1`; its representation semantics are unchanged.

## 6.0.0-alpha.2 — Managed installation lifecycle

### Added
- complete user-local installer for Tlaloc + current Origami representation package;
- versioned XDG installation roots and `current` links;
- per-version SHA-256 installation manifests;
- `tlaloc` dispatcher and `origami` representation CLI;
- `tlaloc doctor`;
- exact managed uninstall;
- historical Origami/OHF/VCL inventory and cleanup;
- optional shell-startup cleanup with automatic backup;
- explicit BPFW/PipeCraft and `.me/origami` protections.

### Versioning
Tlaloc advances to `6.0.0-alpha.2`. Origami remains `6.0.0-alpha.1` because its representation semantics did not change in this packaging release.

## 6.0.0-alpha.1 — Tlaloc / Origami split + Behavior Compilation R0

Common historical branch point of Tlaloc and Origami.

### Added
- formal Tlaloc project identity;
- `BehaviorSpec -> PromptIR -> compiled prompt` pipeline;
- bounded Go swarm trainer;
- guard workers and central arbiter;
- deterministic regression/evaluation path;
- OpenAI-compatible local target adapter;
- Origami quantum-inspired behavior profile as the first consumer profile.
