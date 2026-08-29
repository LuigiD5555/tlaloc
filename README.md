# Tlaloc 6.0.0-alpha.10

**TLALOC — Transformative Latent Adaptive Logic Orchestration Core**

Tlaloc is the work system: behavior compilation, Tlaloque coordination/training, orchestration, verification, promotion control and model-facing execution.

Origami is an independent representation/state-machine language that Tlaloc may use. It is neither the name of Tlaloc nor a mandatory dependency.

## Current R0 implementation

- `behavior-lab/` — bounded behavior-compilation experiment in Go;
- `BehaviorSpec` -> `PromptIR` -> compiled prompt;
- bounded Tlaloque repair proposals with centralized promotion authority;
- deterministic reference-semantics comparison for the current Origami profile;
- OpenAI-compatible transport for local/compatible endpoints;
- `origami.quantum-inspired.r0` as the first bundled executable consumer profile;
- awareness of Origami `6.0.0-alpha.3` and its `origami.perceptual-channels.r0` upstream semantic contract (runtime generation/evaluation not yet implemented);
- `CLAUDE.md` + five `.claude/skills/` entries as checked-in guidance specific to Tlaloc/Origami development.

The current Origami evaluator, reference engine, Tlaloque and curriculum are still specialized for the coherent-state profile. Tlaloc recognizes the Origami perceptual-channel contract (introduced in Origami alpha.2 and preserved by alpha.3) (`MOIRE`, `STEREO_BIND`, `KINETIC_REVEAL`, temporal integration, etc.) but does **not** yet execute or evaluate those operations. Native Claude/Anthropic API support, generated Claude Skills/SkillIR, and GPT/Qwen/LFM-specific compiler backends are also **not yet implemented**.

## Project skills vs generated skills

The `.claude/skills/` directory contains five checked-in workflow skills owned by Tlaloc/Origami development:

- `tlaloc-project`
- `tlaloc-behavior`
- `tlaloc-tlaloque`
- `origami-semantics`
- `tlaloc-release`

Use `tlaloc skills list`, `tlaloc skills path`, and `tlaloc skills install <name>` for these Tlaloc-owned skills. Installation refuses to overwrite a differing local skill unless `--force` is explicit.

The project-agnostic `repo-flow` skill is **not owned or distributed by Tlaloc anymore**. Its canonical source and both `.claude/skills/` and `.agents/skills/` mirrors live in **Tonal** (`LuigiD5555/tonal`). Tlaloc keeps an explicit migration error for `tlaloc skills install repo-flow` so old usage fails clearly instead of silently installing a stale copy.

These checked-in project skills are not a second semantic source of truth and are not output from the Behavior Compiler. The intended future architecture remains `BehaviorSpec -> PromptIR / future SkillIR -> target-specific artifacts`.

## Source-of-truth rule

`BehaviorSpec + invariants` define intended behavior. Prompts and future generated skills are derived artifacts and may be regenerated, replaced or specialized per model family.

## Naming

- **Tlaloc** = complete work system.
- **Tlaloque** = bounded specialist agents coordinated by Tlaloc.
- **Origami** = independent representation/state-machine language.
- **reference semantics** = deterministic expected-state calculation; not an agent.
- **Tonal** = independent composition/distribution layer for tested Tlaloc + Origami revisions and shared stack-level assets such as `repo-flow`.

See `docs/NOMENCLATURE.md`.

## Read first

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
