# Capability status — Tlaloc 6.0.0-alpha.13

Repository lifecycle: Tlaloc installs and uninstalls independently; Origami is optional. Tlaloc is a development kit/work system that can target Origami or other tools. This file distinguishes implemented machinery from empirical or promoted capability.

| Capability | Status | Notes |
|---|---|---|
| BehaviorSpec validation | R0 implemented | Compiler accepts a formal profile contract. |
| PromptIR | R0 implemented | Deterministic ordering/rendering. |
| Generic behavior lifecycle | R0 implemented | compile -> execute -> evaluate -> Tlaloque proposals -> promotion recommendation. |
| Tlaloque bounded-agent layer | R0 implemented | Specialist agents remain proposal/search workers, not canonical truth authority. |
| Project-local Claude Code skills | R0 implemented | Local project skills remain installed and verified independently. |
| Tonal-owned `repo-flow` distribution | external / not Tlaloc-owned | Tonal may distribute project-agnostic workflow skills; Tlaloc does not own the canonical cross-project repo-flow skill. |
| SkillIR / generated Claude Skills | not implemented | Explicit future capability; current skills are maintained artifacts. |
| Origami Semantic Spine R1 awareness | **contract-known** | Tlaloc tracks upstream state/context/rule, observation, Fold/Unfold and evidence semantics without redefining them. |
| Origami canonical visual profile awareness | **alpha.13 contract-known** | Tracks one canonical Origami aesthetic per profile version and mandatory ROSETTA self-description. Per-document private styles are not valid profile evolution. |
| Origami Writer R0 awareness | **alpha.13 contract-known** | Source/PDF semantics are expected to flow through Semantic IR -> visual intent -> canonical grammar -> compiler -> roundtrip verification. Tlaloc remains development/search side, not pixel authority. |
| Origami perceptual channels | **contract-known / runtime partial** | MOIRE/PHASE, STEREO/PARALLAX, kinetic/temporal and emergent channels are explicit experimental families; complete execution remains evidence gated. |
| Receiver swarm-trace distillation | **experimental R0 implemented** | Distills successful trace behavior to bounded receiver-rule candidates with provenance. |
| Receiver candidate tournament | **experimental R0 implemented** | Hard-rejects contamination, false exactness, UNKNOWN and active-window regressions. |
| OpenAI-compatible text transport | R0 implemented | LM Studio and compatible endpoints. |
| OpenAI-compatible Hybrid multimodal/tool loop | **experimental implemented** | Sends prompt + image, executes declared tools externally, returns tool results to the model. |
| Origami Fixed Carrier R2 PDF memory plane | **experimental R1 implemented** | Canonical document ingest, exact source preservation, CIDs/Merkle, generated operational prompt, carrier compilation and QUERY/EXPAND/VERIFY. |
| Fixed Carrier R2 tool bridge | **experimental R1 implemented** | `origami_boot/query/expand/verify` bind the actual carrier to its store. |
| Perception transport variants | **alpha.12 implemented** | Original PNG, 75%, 50% and JPEG preview. |
| Strict multimodal perception trial runner | **alpha.12 implemented** | No private evaluator ground-truth leakage. |
| Origami per-trial evaluator bridge | **alpha.12 implemented** | Invokes independent `origami-perception-eval`. |
| Cross-model perception campaign aggregation | **alpha.12 implemented** | REAL_MODEL-only default 3 models x 3 original trials + transport/tool-loop/routing gates. |
| Hybrid vs Native T3 separation | **implemented gate semantics** | Hybrid candidate does not require Native T3. |
| Real held-out Hybrid VLM evidence | **not yet established** | Machinery is not empirical support. |
| Native T3 cross-model evidence | **not yet established** | Requires external real-model trials. |
| Origami Visual Evolution R0 candidate model | **alpha.13 implemented** | Candidate mutations include PROMPT, CHANNEL_ROLE, PRIMITIVE, LAYOUT, REDUNDANCY, COLOR_USAGE, NUMERIC_STRUCTURE, INTERFERENCE_STRUCTURE, DEPTH_STRUCTURE, TEMPORAL_STRUCTURE and EMERGENT_STRUCTURE. |
| Visual-profile evidence evaluator | **alpha.13 implemented** | Gates semantic roundtrip, evidence/routing, carrier/context budget, UNKNOWN discipline, perceptual reveal, real-model replication and `FALSE_EXACT=0`. |
| Visual-profile deterministic tournament | **alpha.13 implemented** | Ranks evidence-backed candidates and outputs recommendation only. |
| Prime/modular/factorization visual search | **candidate family supported by search schema** | Numeric structures may be tested but have no canonical authority until evidence + Origami profile promotion. |
| Moire/phase/depth/temporal visual search | **candidate families supported by search schema** | Advanced perceptual candidates must demonstrate reveal reliability and preserve UNKNOWN discipline. |
| Master Prompt mutation search | **alpha.13 candidate kind implemented** | Tlaloc may benchmark prompt variants without redefining Origami semantics. |
| Automatic canonical Origami aesthetic mutation | **forbidden/not implemented** | Tlaloc never writes a winning experiment directly into canonical Origami. |
| Canonical Origami profile promotion | **Origami-owned external authority** | Tlaloc recommends with evidence; Origami decides whether a new Origami profile/version becomes canonical. |
| Tonal multi-tool composition | **external / optional** | Tonal may compose Tlaloc, Blueprint Framework and other development tools with exact target revisions. Tonal does not own Origami profile promotion. |
| Native Anthropic/Claude adapter | **not implemented** | Planned target-family adapter. |
| Target-specific compiler optimization | **not implemented** | OpenAI-compatible transport is not a specialized model-family compiler. |
| General model-weight training | **not implemented** | Current distillation/search operates on explicit artifacts/traces, not weights. |
| Managed installer/uninstaller | R0 implemented | Independent user-local lifecycle. |

No document should claim planned, mechanically exercised or visually interesting capabilities are empirically promoted.

## Canonical Memory Plane R2

Implemented reference: canonical PDF/layout extraction with OCR fallback, stable page/block/region addresses, exact page/block plane, candidate generation, deterministic reducer, conflict/uncertainty controller, state ledger, Merkle verification, External Recursive Attention, R2 tool ABI and Fixed Carrier bootstrap integration.

## Perception Promotion Campaign R1

```text
canonical carrier
 -> original / 75% / 50% / JPEG
 -> real VLM observation
 -> independent Origami evaluator
 -> real tool-loop evidence
 -> held-out routing evidence
 -> campaign gates
```

Default aggregation requires at least three real models with three original Hybrid-eligible trials each, required transport coverage, tool-loop success across at least three models, held-out routing thresholds, zero budget violations and `FALSE_EXACT=0`.

## Origami Visual Evolution R0

```text
Origami canonical profile N
 -> Tlaloc development/experimental mutation candidates
 -> deterministic + real-model evidence
 -> hard semantic/evidence/reveal gates
 -> deterministic tournament
 -> recommendation to Origami
 -> Origami decides profile N+1
 -> optional Tonal composition with Tlaloc / Blueprint / other tools
```

The current reference mutation families include prompt changes, primitives, layouts, redundancy, color, numeric/mathematical structures, interference/moire/phase, stereo/parallax/depth, temporal and emergent structures.

The search objective now includes recoverable semantic units per byte, recognition latency, BOOT steps and decode steps in addition to semantic/evidence/routing/transport metrics.

A visually striking prime-derived, moire or stereo pattern can therefore win only if it actually improves measured behavior while preserving all hard invariants. Tlaloc's winner is still only a recommendation to Origami.
