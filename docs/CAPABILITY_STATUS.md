# Capability status — Tlaloc 6.0.0-alpha.12

Repository lifecycle: Tlaloc installs and uninstalls independently; Origami is optional. Legacy Origami/OHF/VCL cleanup is retained for migration only.

This file distinguishes implemented machinery from empirical or promoted capability.

| Capability | Status | Notes |
|---|---|---|
| BehaviorSpec validation | R0 implemented | Compiler accepts a formal profile contract. |
| PromptIR | R0 implemented | Deterministic ordering/rendering. |
| Generic behavior lifecycle | R0 implemented | compile -> execute -> evaluate -> Tlaloque repair proposals -> promote candidate. |
| Tlaloque bounded-agent layer | R0 implemented | Rule-based specialist agents under `internal/tlaloque`; centralized promotion authority. |
| Tlaloque introspection | R0 implemented | `tlaloc tlaloque` lists current built-in specialists. |
| Origami coherent-state profile | R0 implemented | First consumer profile and historical test curriculum. |
| Origami Semantic Spine R1 awareness | **contract-known** | Tracks first-class state/context/rule semantics, observation separation, semantic Fold/Unfold and source/semantic boundaries from Origami alpha.7+. Tlaloc does not redefine those semantics. |
| Origami perceptual-channels contract | **contract-known / runtime partial** | Tracks Origami perceptual-channel semantics; no general runtime for every MOIRE/STEREO_BIND/KINETIC_REVEAL operation yet. |
| Origami Hybrid Receiver contract | **experimental contract-known** | Model-facing promotion evidence is still external and gate-driven. |
| Receiver swarm-trace distillation | **experimental R0 implemented** | `internal/distill` converts externally relevant successful semantic transitions into deterministic MicroRule candidates, deduplicates equivalent transitions, rejects conflicts and retains SHA-256 trace provenance. This is behavioral artifact distillation, not model-weight training. |
| Hybrid artifact-set projection | **experimental R0 implemented** | Distilled candidates can be projected into `tlaloc.origami-hybrid-artifact-set.r0` containing the universal prompt plus BOOT/Rosetta/micro-program proposal for explicit Origami import/validation. |
| Receiver candidate tournament | **experimental R0 implemented** | Scores bootstrap/Rosetta/navigation/correctness/evidence/UNKNOWN and hard-rejects contamination, false exactness and active-window violations. |
| Receiver distillation CLI | **experimental R0 implemented** | `behaviorlab receiver-distill` and `receiver-rank` create/rank Tlaloc candidates; they do not promote/write Origami's canonical receiver registry. |
| OpenAI-compatible text transport | R0 implemented | Text Chat Completions transport for LM Studio and compatible endpoints. |
| OpenAI-compatible Hybrid multimodal/tool loop | **experimental R0 implemented** | `CompleteHybrid` sends prompt + carrier image, exposes declared tools, executes tool calls through an external executor and feeds tool results back until final answer/turn limit. Mock-server tests verify loop mechanics. |
| Origami CLI tool bridge | **experimental R0 implemented** | External executors map declared receiver functions to independent Origami processes; Tlaloc does not import Origami runtime code. |
| Origami Fixed Carrier R2 PDF memory plane | **experimental R1 implemented** | `internal/pdfmemory` preserves source PDF bytes, canonical layout/text/CIDs, routing structures and Merkle store root; `tlaloc-origami compile` emits the generated Master Prompt and asks independent Origami tooling to create the fixed image. |
| Fixed Carrier R2 tool bridge | **experimental R1 implemented** | `origami_boot/query/expand/verify` bind actual `origami.png` to the matching store. OCR is not used as BOOT. |
| Plain-text Origami tool fallback | **experimental R1 implemented** | Models without function calling may emit explicit `<ORIGAMI_CALL>` envelopes under Tlaloc's multimodal text bridge. Tool outputs are never model-fabricated. |
| Perception transport variants | **experimental alpha.12 implemented** | Generates original PNG, 75% PNG, 50% PNG and JPEG-preview variants from one canonical carrier. |
| Strict multimodal perception trial runner | **experimental alpha.12 implemented** | Sends only declared prompt/question/image to OpenAI-compatible endpoints and parses a structured observation without leaking evaluator ground truth. |
| Origami per-trial evaluator bridge | **experimental alpha.12 implemented** | Invokes independent `origami-perception-eval` so Tlaloc does not score its own visual ground truth. |
| Cross-model perception campaign aggregation | **experimental alpha.12 implemented** | Requires REAL_MODEL evidence, 3 models x 3 original trials by default, transport coverage, real tool loops and held-out routing evidence before producing a Hybrid candidate. |
| Hybrid vs Native T3 separation | **implemented gate semantics** | Hybrid candidate does not require Native T3. Native T3 is an additional independent cross-model gate. |
| Real held-out Hybrid VLM evidence | **not yet established** | Campaign machinery exists, but external VLM runs are still required. Mock or synthetic reports cannot satisfy promotion. |
| Cross-model Hybrid replication | **not yet established** | Requires real 3+ model campaign evidence under the frozen prompt/carrier policy. |
| Native T3 cross-model evidence | **not yet established** | Requires real models to reproduce T3 under the strict Native boundary. |
| Origami reference evaluator/guards | R0 implemented | Reference semantics remain external/upstream authority; Tlaloc does not redefine Origami truth. |
| Project `CLAUDE.md` guidance | **R0 implemented** | Checked-in instructions for working on this codebase. |
| Project-local Claude Code skills | **R0 implemented** | Five checked-in `.claude/skills/*/SKILL.md` workflow skills specific to Tlaloc/Origami development; not compiler-generated. |
| Tlaloc-owned project skill installation | **R0 implemented** | `tlaloc skills list/path/install` operates on Tlaloc-owned project skills and protects differing local copies by default. |
| Tonal-owned `repo-flow` distribution | **external / not Tlaloc-owned** | Canonical `repo-flow` plus `.claude/skills` and `.agents/skills` mirrors live in Tonal. |
| Skills validation | **R0 implemented** | Release tests validate required Tlaloc-owned skill structure/frontmatter and installed copies. |
| Native Anthropic/Claude adapter | **not implemented** | Planned target-family adapter. |
| SkillIR / generated Claude Skills | **not implemented** | Planned output of future model-profile layer. Do not confuse with checked-in project skills. |
| Native OpenAI model profile | **not implemented** | OpenAI-compatible transport/Hybrid loop is not the same as target-specific prompt compilation. |
| Qwen/LFM target profiles | **not implemented** | Planned. |
| Target-specific compiler optimization | **not implemented** | `target` is currently metadata/transport selection, not a specialized compiler backend. |
| General behavioral distillation / weight training | **not implemented** | Receiver R0 distills explicit external transition traces only. General behavior compression and model-weight training remain future work. |
| Automated cross-repository receiver promotion | **not implemented** | Tlaloc emits recommendations/evidence; Origami/Tonal promotion remains explicit and gate-driven. |
| Managed installer/uninstaller | R0 implemented | Versioned user-local install with independent Tlaloc/Origami lifecycle. |
| Legacy Origami/VCL/OHF cleanup | R0 implemented | Conservative scan/removal with hard exclusions. |
| alpha.2 managed-layout migration | **implemented** | Accepts old Origami Tlaloc-named markers and removes obsolete alpha.2 installer-state manifest. |

No document should claim planned or merely mechanically exercised capabilities are already empirically promoted.

## Canonical Memory Plane R2

Implemented reference: canonical PDF/layout extraction with OCR fallback, stable page/block/region addresses, exact page/block plane, candidate generation, deterministic Go reducer, conflict/uncertainty controller, state ledger, Merkle verification, External Recursive Attention query trace, R2 tool ABI and Fixed Carrier R2 bootstrap integration.

## Perception Promotion Campaign R1

Implemented campaign machinery:

```text
canonical carrier
 -> original / 75% / 50% / JPEG transports
 -> real OpenAI-compatible VLM trial
 -> structured observation
 -> independent Origami evaluator
 -> real tool-loop evidence
 -> held-out routing evidence
 -> campaign gates
```

The default aggregate policy requires at least three real models with three original Hybrid-eligible trials each, required transport coverage, real tool-loop success across at least three models, held-out multi-document routing thresholds, zero budget violations and `FALSE_EXACT=0`.

A Native T3 candidate additionally requires three models with three original Native-T3-eligible trials each. Hybrid support is intentionally allowed to mature before Native T3.
