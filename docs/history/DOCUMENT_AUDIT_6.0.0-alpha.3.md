# Documentation/concept audit — 6.0.0-alpha.3

Date: 2026-08-28

## Corrected

- stale generated prompt identified as Origami-owned -> regenerated as Tlaloc-owned artifact;
- `PROJECT_SPLIT.md` dependency wording -> Origami is optional/independent representation provider;
- `oracle` ownership ambiguity -> Origami provides reference semantics; Tlaloc owns oracle/evaluation role;
- `Master Prompt` wording -> replaced with compiled/promoted behavioral artifact;
- `target-specific compiler` overclaim -> R0 truthfully documented as generic prompt compilation + OpenAI-compatible transport only;
- Behavior Lab `model-independent` overclaim -> lifecycle is general, while current evaluator/guards/curriculum remain Origami-profile-specific;
- installation ownership markers -> Tlaloc and Origami now have component-specific markers; legacy alpha.2 markers remain recognized for migration/uninstall;
- uninstall entry points -> project-specific uninstallers now remove only their own project by default; bundle removal is explicit;
- active/historical documentation -> old change-control and split records moved under `docs/history/`;
- profile placement -> Origami quantum-inspired BehaviorSpec moved from generic `examples/` to `profiles/origami/`;
- temporary-path prompt hash -> replaced by relative generated-artifact manifest.

## Kept as current

- Tlaloc acronym and work-system definition;
- independent Tlaloc/Origami versioning from common split point `6.0.0-alpha.1`;
- Origami ownership of state representation, dynamics and semantic contracts;
- conservative legacy cleanup with BPFW/PipeCraft and `.me/origami` hard exclusions.

## Explicitly not implemented yet

- native Claude/Anthropic adapter;
- Claude Skills generation;
- GPT/Qwen/LFM family-specific prompt compilers;
- generic pluggable evaluator/oracle profile interface;
- behavioral distillation into model weights.

These remain roadmap items and must not be presented as current capabilities.

## File-by-file disposition

| File / family | Disposition | Reason |
|---|---|---|
| bundle `README.md` | REWRITTEN / CURRENT | Clarifies optional Origami relationship and real model-support status. |
| `VERSION_MANIFEST.json` | REWRITTEN / CURRENT | Records independent component versions and explicit support status. |
| `docs/INSTALLATION_LAYOUT.md` | REWRITTEN / CURRENT | Removes Tlaloc ownership markers from new Origami installs. |
| `docs/LEGACY_INSTALLATION_MAP.md` | UPDATED / CURRENT | Retains cleanup history and alpha.2 marker migration compatibility. |
| Tlaloc `README.md` | REWRITTEN / CURRENT | Uses current source-of-truth and capability boundaries. |
| Tlaloc `CHANGELOG.md` | KEEP / CURRENT+HISTORY | Chronological history remains useful. |
| Tlaloc old change-control files | ARCHIVED | Moved to `docs/history/` and marked superseded. |
| old `PROJECT_SPLIT.md` | ARCHIVED + ANNOTATED | Historical split is useful; mandatory-dependency wording was superseded. |
| `docs/ARCHITECTURE.md` | NEW / AUTHORITATIVE | Current project boundary and authority hierarchy. |
| `docs/CAPABILITY_STATUS.md` | NEW / AUTHORITATIVE | Prevents planned Claude/skills/model-profile work from being mistaken as implemented. |
| `docs/BEHAVIOR_COMPILATION.md` | REWRITTEN / CURRENT | Stops claiming R0 already has family-specific compiler backends. |
| `docs/ORIGAMI_INTEGRATION_CONTRACT.md` | REWRITTEN / CURRENT | Separates Origami reference semantics from Tlaloc oracle role. |
| Behavior Lab `README.md` | REWRITTEN / CURRENT | Removes `Master Prompt` and false fully-model-independent wording. |
| `examples/quantum_behavior.json` | MOVED | Now `profiles/origami/quantum-inspired-r0.json`; it is a consumer profile, not a generic example. |
| stale `generated/BEHAVIOR_PROMPT.md` | DELETED | Contained old Origami ownership heading. |
| generated Origami prompt | REGENERATED | Now Tlaloc-owned compiled artifact using PromptIR 0.2. |
| `PROMPT_SHA256.txt` | DELETED | Embedded a temporary `/tmp/...` path and was not a durable artifact manifest. |
| `GENERATED_ARTIFACTS.sha256` | NEW | Uses a package-relative generated artifact path. |
| Origami `README.md` | KEEP / CURRENT | Correctly defines Origami as representation/state-machine language. |
| Origami `PROJECT_BOUNDARY.md` | CORRECTED / CURRENT | Replaces "state oracle" ownership with reference semantics engine. |
| Origami `STATE_SEMANTICS_R0.md` | CORRECTED / CURRENT | Clarifies quantum-inspired semantics are one profile, not all of Origami. |
| Origami changelog/change control | KEEP / CURRENT HISTORY | Correct ownership and semantic split remain valid. |
