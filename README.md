# Tlaloc 6.0.0-alpha.12

**TLALOC — Transformative Latent Adaptive Logic Orchestration Core**

Tlaloc is the work system: behavior compilation, Tlaloque coordination/training, orchestration, verification, promotion control and model-facing execution.

Origami is an independent representation/state-machine language that Tlaloc may use. It is neither the name of Tlaloc nor a mandatory dependency.

## Current implementation

R2 canonical-memory work provides a layout-preserving Canonical Document IR, evidence-backed Tlaloque candidate protocol, deterministic CanonicalState reducer, uncertainty-driven verification queues, Merkle/CID exact plane and External Recursive Attention over a bounded active context.

Alpha.12 adds the cross-model perception-promotion campaign machinery that had previously existed only as a planned/local second round. It does **not** claim that any external VLM has passed those gates.

- `behavior-lab/` — bounded behavior-compilation and Origami integration laboratory in Go;
- `BehaviorSpec` -> `PromptIR` -> compiled prompt;
- bounded Tlaloque repair proposals with centralized promotion authority;
- deterministic reference-semantics comparison for declared Origami profiles;
- OpenAI-compatible text and multimodal transport for local/compatible endpoints;
- awareness of Origami through `6.0.0-alpha.8`, including Semantic Spine R1, Fixed Carrier R2, Evidence Reduction R0 and Perception Promotion R1;
- experimental `origami-hybrid-receiver-r0` profile and `internal/distill` path for turning successful swarm traces into simple deterministic receiver-rule candidates;
- receiver-candidate fitness gates for bootstrap, Rosetta use, navigation, correctness, evidence, UNKNOWN, active-context budget, contamination and false exactness;
- `tlaloc-origami` Fixed Carrier R2 PDF memory plane: deterministic PDF ingest, page/CID addressing, Merkle verification, generated Master Prompt, fixed-carrier compilation, `QUERY/EXPAND/VERIFY`, and OpenAI-compatible multimodal tool/text-bridge execution;
- perception-campaign transport variants: original PNG, 75% resize, 50% resize and JPEG preview;
- strict real-model observation runner that gives the target model prompt + question + one image and does not leak evaluator ground truth;
- bridge to Origami's independent `origami-perception-eval` per-trial evaluator;
- campaign aggregation requiring real-model trials, transport coverage, real tool loops and held-out routing evidence before producing a Hybrid candidate;
- Native T3 campaign state kept independent from Hybrid support;
- `CLAUDE.md` + five `.claude/skills/` entries as checked-in guidance specific to Tlaloc/Origami development.

The Hybrid Receiver and Perception Promotion work are **experimental and not yet promoted compatibility claims**. Tlaloc may run campaigns and produce promotion recommendations, but Origami remains the authority for its semantic/perceptual contracts and Tonal remains the authority for aggregate stack promotion.

A perfect mock campaign validates machinery only. It can never satisfy empirical promotion. A real successful single trial is still only one trial.

The current perceptual-channel operation family (`MOIRE`, `STEREO_BIND`, `KINETIC_REVEAL`, temporal integration, etc.) remains only partially executable. Native Claude/Anthropic API support, generated Claude Skills/SkillIR, and GPT/Qwen/LFM-specific compiler backends are also **not yet implemented**.

## Project skills vs generated skills

The `.claude/skills/` directory contains five checked-in workflow skills owned by Tlaloc/Origami development:

- `tlaloc-project`
- `tlaloc-behavior`
- `tlaloc-tlaloque`
- `origami-semantics`
- `tlaloc-release`

Use `tlaloc skills list`, `tlaloc skills path`, and `tlaloc skills install <name>` for these Tlaloc-owned skills. Installation refuses to overwrite a differing local skill unless `--force` is explicit.

Project-agnostic `repo-flow` and `gatekeeper` skills are owned by **Tonal** (`LuigiD5555/tonal`), not Tlaloc. This keeps one authority for Git/repository workflow and project-wide provenance policy.

These checked-in project skills are not a second semantic source of truth and are not output from the Behavior Compiler. The intended architecture has related compiled/distilled/search paths:

```text
BehaviorSpec -> PromptIR / future SkillIR -> target-specific behavior artifacts

Origami receiver contract -> swarm search -> receiver candidate
  -> prompt + BOOT/Rosetta strategy + MicroAgent IR
  -> Origami validation/promotion

Origami perception contract -> cross-model campaign
  -> per-trial Origami evaluator reports
  -> routing/tool-loop/transport gates
  -> Tlaloc promotion recommendation
  -> Tonal aggregate promotion
```

## Project Gatekeeper

Tlaloc follows Tonal's project-wide Gatekeeper R0. `gatekeeper.json` is a local CI mirror and `GATEKEEPER.md` explains the component behavior.

- owner PR (`LuigiD5555` from this canonical repository): normal Tlaloc verification still runs; explicit owner promotion override is allowed;
- external PR: normal verification still runs and an `APPROVED` review from `LuigiD5555` is mandatory; external override/auto-promotion is denied.

The policy controls promotion authority, not assumptions about code quality.

## Source-of-truth rule

`BehaviorSpec + invariants` define intended Tlaloc behavior. Prompts and future generated skills are derived artifacts and may be regenerated, replaced or specialized per model family.

For Origami work, Origami's semantic/receiver/visual contracts remain upstream sources of truth. Tlaloc may search, mutate, benchmark and distill *how* target models follow those contracts; it does not become the semantic or canonical-artifact authority for Origami.

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
