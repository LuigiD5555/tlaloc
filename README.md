# Tlaloc 6.0.0-alpha.9

**TLALOC — Transformative Latent Adaptive Logic Orchestration Core**

Tlaloc is the work system: behavior compilation, Tlaloque coordination/training, orchestration, verification, promotion control and model-facing execution.

Origami is an independent representation/state-machine language that Tlaloc may use. It is neither the name of Tlaloc nor a mandatory dependency.

## Current R0 implementation

- `behavior-lab/` — bounded behavior-compilation experiment in Go;
- `BehaviorSpec` -> `PromptIR` -> compiled prompt;
- bounded Tlaloque repair proposals with centralized promotion authority;
- deterministic reference-semantics comparison for the current Origami profile;
- OpenAI-compatible transport for local/compatible endpoints;
- an explicit registry for the compatible `origami.quantum-inspired.r0` and executable `origami.relational-core.r0` consumer profiles;
- strict coherent-state evaluation with causal cancellation and complete structured fields;
- byte-identical Origami `6.0.0-alpha.4` EXP-001 fixtures covering fixed point, cycle, contradiction, and budget exhaustion;
- awareness of `origami.perceptual-channels.r0` (runtime generation/evaluation not yet implemented);
- `CLAUDE.md` + `.claude/skills/` as checked-in project guidance for Claude Code and compatible agent workflows.

Tlaloc evaluates the coherent-state profile and consumes the upstream relational-core fixtures through separate registered paths. It does not duplicate Origami's relational Reference Machine. Perceptual operations such as `MOIRE`, `STEREO_BIND`, and `KINETIC_REVEAL` remain explicitly unsupported. Native Claude/Anthropic API support, generated SkillIR, and model-family compiler backends are also not yet implemented.

## Project skills vs generated skills

The `.claude/skills/` directory is a repository-workflow aid. It teaches an agent how to respect Tlaloc's architecture, behavior-compilation rules, Tlaloque boundaries, Origami ownership, and release hygiene.

These files are not a second source of truth and are not output from the Behavior Compiler. The intended future architecture remains `BehaviorSpec -> PromptIR / future SkillIR -> target-specific artifacts`.

## Source-of-truth rule

`BehaviorSpec + invariants` define intended behavior. Prompts and future generated skills are derived artifacts and may be regenerated, replaced or specialized per model family.

## Naming

- **Tlaloc** = complete work system.
- **Tlaloque** = bounded specialist agents coordinated by Tlaloc.
- **Origami** = independent representation/state-machine language.
- **reference semantics** = deterministic expected-state calculation; not an agent.

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
