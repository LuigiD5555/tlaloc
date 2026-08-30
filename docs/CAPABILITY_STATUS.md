# Capability status — Tlaloc 6.0.0-alpha.11

Repository lifecycle: Tlaloc installs and uninstalls independently; Origami is optional. Legacy Origami/OHF/VCL cleanup is retained for migration only.

This file distinguishes implemented behavior from intended architecture.

| Capability | Status | Notes |
|---|---|---|
| BehaviorSpec validation | R0 implemented | Compiler accepts a formal profile contract. |
| PromptIR | R0 implemented | Deterministic ordering/rendering. |
| Generic behavior lifecycle | R0 implemented | compile -> execute -> evaluate -> Tlaloque repair proposals -> promote candidate. |
| Tlaloque bounded-agent layer | R0 implemented | Rule-based specialist agents under `internal/tlaloque`; centralized promotion authority. |
| Tlaloque introspection | R0 implemented | `tlaloc tlaloque` lists current built-in specialists. |
| Origami coherent-state profile | R0 implemented | First consumer profile and current test curriculum. |
| Origami perceptual-channels contract | **contract-known / runtime not implemented** | Tracks Origami perceptual-channel semantics introduced in alpha.2 and preserved through current alpha.5; no renderer, detector, reference evaluator or Tlaloque curriculum for MOIRE/STEREO_BIND/KINETIC_REVEAL yet. |
| Origami Hybrid Receiver contract | **experimental contract-known** | Feature branch recognizes `origami.hybrid-receiver.r0`; model-facing promotion evidence is not yet established. |
| Receiver swarm-trace distillation | **experimental R0 implemented** | `internal/distill` converts externally relevant successful semantic transitions into deterministic MicroRule candidates, deduplicates equivalent transitions, rejects conflicts and retains SHA-256 trace provenance. This is behavioral artifact distillation, not model-weight training. |
| Hybrid artifact-set projection | **experimental R0 implemented** | Distilled candidates can be projected into `tlaloc.origami-hybrid-artifact-set.r0` containing the universal prompt plus BOOT/Rosetta/micro-program proposal for explicit Origami import/validation. |
| Receiver candidate tournament | **experimental R0 implemented** | Scores bootstrap/Rosetta/navigation/correctness/evidence/UNKNOWN and hard-rejects contamination, false exactness and active-window violations. |
| Receiver distillation CLI | **experimental R0 implemented** | `behaviorlab receiver-distill` and `receiver-rank` create/rank Tlaloc candidates; they do not promote/write Origami's canonical receiver registry. |
| OpenAI-compatible text transport | R0 implemented | Existing text Chat Completions transport for LM Studio and compatible endpoints. |
| OpenAI-compatible Hybrid multimodal/tool loop | **experimental R0 implemented** | `CompleteHybrid` and `behaviorlab receiver-run` send prompt + carrier image, expose declared tools, execute tool calls through an external executor and feed tool results back until final answer/turn limit. Mock-server tests verify the loop mechanics. |
| Origami CLI tool bridge | **experimental R0 implemented** | `OrigamiCLIExecutor` maps declared receiver functions to an independent `origami-hybrid-tool` process; Tlaloc does not import Origami runtime code. |
| Origami Fixed Carrier R2 PDF memory plane | **experimental R1 implemented** | `internal/pdfmemory` preserves exact source PDF bytes, canonical page text/CIDs, lexical routing index, fixed GraphSketch and Merkle store root; `tlaloc-origami compile` emits a generated Master Prompt and asks the independent Origami compiler to create the fixed image. |
| Fixed Carrier R2 tool bridge | **experimental R1 implemented** | `origami_boot/query/expand/verify` bind actual `origami.png` to the matching store. OCR is not used as BOOT. |
| Plain-text Origami tool fallback | **experimental R1 implemented** | Models without function calling may emit explicit `<ORIGAMI_CALL>` envelopes under Tlaloc's multimodal text bridge. Tool outputs are never model-fabricated. |
| Real held-out Hybrid VLM evidence | **not yet established** | Requires executing the harness against actual image/tool-capable target models on frozen synthetic carriers, including symbol-permutation and anti-contamination gates. Harness implementation is not evidence that a VLM can self-bootstrap the carrier. |
| Cross-model Hybrid replication | **not yet established** | Requires the same external bootstrap across multiple target models/carrier-local symbol permutations. |
| Origami reference evaluator/guards | R0 implemented | Reference semantics + evaluator remain profile-specific; not yet a generic plugin interface. |
| Project `CLAUDE.md` guidance | **R0 implemented** | Checked-in instructions for working on this codebase. |
| Project-local Claude Code skills | **R0 implemented** | Five checked-in `.claude/skills/*/SKILL.md` workflow skills specific to Tlaloc/Origami development; not compiler-generated. |
| Tlaloc-owned project skill installation | **R0 implemented** | `tlaloc skills list/path/install` operates on Tlaloc-owned project skills and protects differing local copies by default. |
| Tonal-owned `repo-flow` distribution | **external / not Tlaloc-owned** | Canonical `repo-flow` plus `.claude/skills` and `.agents/skills` mirrors live in Tonal; Tlaloc reports a migration message instead of distributing a stale copy. |
| Skills validation | **R0 implemented** | Release tests validate required Tlaloc-owned skill structure/frontmatter and installed copies. |
| Native Anthropic/Claude adapter | **not implemented** | Planned target-family adapter. |
| SkillIR / generated Claude Skills | **not implemented** | Planned output of future model-profile layer. Do not confuse with checked-in project skills. |
| Native OpenAI model profile | **not implemented** | OpenAI-compatible transport/Hybrid loop is not the same as target-specific prompt compilation. |
| Qwen/LFM target profiles | **not implemented** | Planned. |
| Target-specific compiler optimization | **not implemented** | `target` is currently metadata/transport selection, not a specialized compiler backend. |
| General behavioral distillation / weight training | **not implemented** | Receiver R0 distills explicit external transition traces only. General behavior compression and model-weight training remain future work. |
| Automated cross-repository receiver promotion | **not implemented** | Origami currently stores a reference candidate; Tlaloc does not automatically write/promote into Origami. Promotion remains explicit and gate-driven. |
| Managed installer/uninstaller | R0 implemented | Versioned user-local install with independent Tlaloc/Origami lifecycle. |
| Legacy Origami/VCL/OHF cleanup | R0 implemented | Conservative scan/removal with hard exclusions. |
| alpha.2 managed-layout migration | **implemented** | Accepts old Origami Tlaloc-named markers and removes obsolete alpha.2 installer-state manifest. |

No document should claim planned capabilities are already available.


## Canonical Memory Plane R2

Implemented reference: canonical PDF/layout extraction with OCR fallback, stable page/block/region addresses, exact page/block plane, candidate generation, deterministic Go reducer, conflict/uncertainty controller, state ledger, Merkle verification, External Recursive Attention query trace, R2 tool ABI and Fixed Carrier R2 bootstrap integration. Cross-model Native visual interpretation remains unpromoted.
