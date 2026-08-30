# Tlaloc 6.0.0-alpha.11

**TLALOC — Transformative Latent Adaptive Logic Orchestration Core**

Tlaloc is the work system: behavior compilation, Tlaloque coordination/training, orchestration, verification, promotion control and model-facing execution.

Origami is an independent representation/state-machine language that Tlaloc may use. It is neither the name of Tlaloc nor a mandatory dependency.

## Current R0 implementation

R2 canonical-memory work adds a layout-preserving Canonical Document IR, evidence-backed Tlaloque candidate protocol, deterministic CanonicalState reducer, uncertainty-driven verification queues, Merkle/CID exact plane and External Recursive Attention over a bounded active context.

- `behavior-lab/` — bounded behavior-compilation experiment in Go;
- `BehaviorSpec` -> `PromptIR` -> compiled prompt;
- bounded Tlaloque repair proposals with centralized promotion authority;
- deterministic reference-semantics comparison for the current Origami profile;
- OpenAI-compatible transport for local/compatible endpoints;
- `origami.quantum-inspired.r0` as the first bundled executable consumer profile;
- awareness of Origami through `6.0.0-alpha.5`, including `origami.perceptual-channels.r0` and experimental `origami.fixed-carrier.r2`; perceptual-channel runtime generation/evaluation remains incomplete;
- experimental `origami-hybrid-receiver-r0` profile and `internal/distill` reference path for turning successful swarm traces into simple deterministic receiver-rule candidates;
- receiver-candidate fitness gates for bootstrap, carrier-local Rosetta use, navigation, correctness, evidence, UNKNOWN, active-context budget, contamination and false exactness;
- `CLAUDE.md` + five `.claude/skills/` entries as checked-in guidance specific to Tlaloc/Origami development.
- `tlaloc-origami` Fixed Carrier R2 PDF memory plane: deterministic PDF ingest, page/CID addressing, Merkle verification, generated Master Prompt, fixed-carrier compilation, `QUERY/EXPAND/VERIFY`, and OpenAI-compatible multimodal tool/text-bridge execution;

The Hybrid Receiver work is **experimental and not yet a promoted compatibility claim**. Its purpose is to use rich Tlaloc swarm/Tlaloque behavior at search time, then distill the externally relevant behavior into a much simpler candidate receiver package: a small universal bootstrap prompt plus deterministic micro-agent rules. Origami remains the authority that validates and stores any promoted receiver artifact.

The current Origami evaluator, reference engine, Tlaloque and original curriculum are still specialized for the coherent-state profile. Tlaloc recognizes the Origami perceptual-channel contract (introduced in Origami alpha.2 and preserved by alpha.3) (`MOIRE`, `STEREO_BIND`, `KINETIC_REVEAL`, temporal integration, etc.) but does **not** yet execute or evaluate those operations. Native Claude/Anthropic API support, generated Claude Skills/SkillIR, and GPT/Qwen/LFM-specific compiler backends are also **not yet implemented**.

## Project skills vs generated skills

The `.claude/skills/` directory contains five checked-in workflow skills owned by Tlaloc/Origami development:

- `tlaloc-project`
- `tlaloc-behavior`
- `tlaloc-tlaloque`
- `origami-semantics`
- `tlaloc-release`

Use `tlaloc skills list`, `tlaloc skills path`, and `tlaloc skills install <name>` for these Tlaloc-owned skills. Installation refuses to overwrite a differing local skill unless `--force` is explicit.

Project-agnostic `repo-flow` and `gatekeeper` skills are owned by **Tonal** (`LuigiD5555/tonal`), not Tlaloc. This keeps one authority for Git/repository workflow and project-wide provenance policy.

These checked-in project skills are not a second semantic source of truth and are not output from the Behavior Compiler. The intended architecture now has two related compiled/distilled paths:

```text
BehaviorSpec -> PromptIR / future SkillIR -> target-specific behavior artifacts

Origami receiver contract -> swarm search -> receiver candidate
  -> prompt + BOOT/Rosetta strategy + MicroAgent IR
  -> Origami validation/promotion
```

## Project Gatekeeper

Tlaloc follows Tonal's project-wide Gatekeeper R0. `gatekeeper.json` is a local CI mirror and `GATEKEEPER.md` explains the component behavior.

- owner PR (`LuigiD5555` from this canonical repository): normal Tlaloc verification still runs; explicit owner promotion override is allowed;
- external PR: normal verification still runs and an `APPROVED` review from `LuigiD5555` is mandatory; external override/auto-promotion is denied.

The policy controls promotion authority, not assumptions about code quality.

## Source-of-truth rule

`BehaviorSpec + invariants` define intended Tlaloc behavior. Prompts and future generated skills are derived artifacts and may be regenerated, replaced or specialized per model family.

For Origami receiver work, Origami's semantic/receiver contract remains the upstream source of truth. Tlaloc may search and distill *how* a target model follows that contract; it does not become the semantic or artifact-storage authority for Origami.

## Naming

- **Tlaloc** = complete work system.
- **Tlaloque** = bounded specialist agents coordinated by Tlaloc.
- **Origami** = independent representation/state-machine language.
- **reference semantics** = deterministic expected-state calculation; not an agent.
- **receiver distillation** = conversion of successful rich swarm behavior into simpler prompt/micro-agent candidate behavior for Origami validation.
- **Tonal** = independent composition/distribution layer for tested Tlaloc + Origami revisions and shared stack-level workflow/promotion policy.

See `docs/NOMENCLATURE.md`.

## Read first

- `GATEKEEPER.md`
- `docs/NOMENCLATURE.md`
- `docs/ARCHITECTURE.md`
- `docs/CAPABILITY_STATUS.md`
- `docs/MODEL_AGENT_GUIDANCE.md`
- `docs/BEHAVIOR_COMPILATION.md`
- `docs/ORIGAMI_INTEGRATION_CONTRACT.md`

## Install from source

Requirements: Linux/macOS shell, Go toolchain, and standard POSIX userland tools. The installer is user-scoped and does not require `sudo`.

```bash
git clone git@github.com:LuigiD5555/tlaloc.git
cd tlaloc
./install.sh
```

Tlaloc installs under `~/.local/share/tlaloc/versions/<version>` and links its CLI under `~/.local/bin` by default.

Origami is **not installed by this repository**. It remains an independent optional project at `LuigiD5555/origami`.

Before removing legacy Origami/OHF/VCL residues, inspect them first:

```bash
./legacy-cleanup.sh --scan
```

Tlaloc itself can be removed with:

```bash
tlaloc-uninstall --yes
```

The retrocompatible legacy cleaner preserves BPFW/PipeCraft and `.me/origami` workspaces by design.
