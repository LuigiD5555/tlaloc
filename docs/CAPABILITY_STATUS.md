# Capability status — Tlaloc 6.0.0-alpha.8

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
| Origami perceptual-channels contract | **contract-known / runtime not implemented** | Tracks current Origami `6.0.0-alpha.3`; `origami.perceptual-channels.r0` was introduced in alpha.2 and is unchanged in alpha.3; no renderer, detector, reference evaluator or Tlaloque curriculum for MOIRE/STEREO_BIND/KINETIC_REVEAL yet. |
| Origami reference evaluator/guards | R0 implemented | Reference semantics + evaluator remain profile-specific; not yet a generic plugin interface. |
| OpenAI-compatible transport | R0 implemented | Intended for LM Studio and compatible endpoints. |
| Project `CLAUDE.md` guidance | **R0 implemented** | Checked-in instructions for working on this codebase. |
| Project-local Claude Code skills | **R0 implemented** | Five checked-in `.claude/skills/*/SKILL.md` workflow skills; not compiler-generated. |
| Skills validation | **R0 implemented** | Release tests validate required skill structure/frontmatter and mirrored installed copies. |
| Native Anthropic/Claude adapter | **not implemented** | Planned target-family adapter. |
| SkillIR / generated Claude Skills | **not implemented** | Planned output of future model-profile layer. Do not confuse with checked-in project skills. |
| Native OpenAI model profile | **not implemented** | Transport compatibility is not the same as target-specific prompt compilation. |
| Qwen/LFM target profiles | **not implemented** | Planned. |
| Target-specific compiler optimization | **not implemented** | `target` is currently metadata/transport selection, not a specialized compiler backend. |
| Behavioral distillation / weight training | **not implemented** | Future stage after prompt-level behavior is measurable and stable. |
| Managed installer/uninstaller | R0 implemented | Versioned user-local install with independent Tlaloc/Origami lifecycle. |
| Legacy Origami/VCL/OHF cleanup | R0 implemented | Conservative scan/removal with hard exclusions. |
| alpha.2 managed-layout migration | **implemented** | Accepts old Origami Tlaloc-named markers and removes obsolete alpha.2 installer-state manifest. |

No document should claim planned capabilities are already available.
