# Model and agent guidance — R0

Tlaloc distinguishes three different ideas that must not be conflated.

## 1. Project-agent guidance (implemented)

The repository includes `CLAUDE.md` and `.claude/skills/<name>/SKILL.md` files to help Claude Code and similar coding agents work on the project without rediscovering architecture and release rules.

Bundled skills:

- `tlaloc-project` — architecture, naming, ownership and version boundaries;
- `tlaloc-behavior` — BehaviorSpec/PromptIR/compiler/evaluation workflow;
- `tlaloc-tlaloque` — bounded-agent design and promotion constraints;
- `origami-semantics` — Origami ownership and coherent-state integration discipline;
- `tlaloc-release` — documentation, installer, retro-uninstall and release gates.

These files are operational documentation. They do not override `BehaviorSpec`, Origami semantics, tests or current architecture documents.

## 2. Target-model adapters (partially implemented)

R0 includes an OpenAI-compatible transport. It does not include a native Anthropic Messages/API adapter and does not optimize prompts specifically for Claude, GPT, Qwen or LFM families.

## 3. Generated target skills (not implemented)

The intended future design may compile model-independent behavior into `SkillIR` and then into target-family skills/instruction packages. That is different from the checked-in project skills above.

```text
BehaviorSpec + invariants
        -> PromptIR / future SkillIR
        -> target-family compiler
        -> prompt / skill / tool contract / other artifact
```

Until SkillIR exists, do not describe the checked-in `.claude/skills` directory as evidence that Tlaloc can generate Claude Skills.

## Claude Code layout

The checked-in Claude integration uses:

```text
CLAUDE.md
.claude/
└── skills/
    └── <skill-name>/
        └── SKILL.md
```

Every bundled `SKILL.md` carries YAML frontmatter with a `name`, `description`, and skill version. Release tests verify the directory/name relationship and required metadata.

The installer keeps a copy inside the installed Tlaloc version and exposes it through:

```bash
tlaloc skills-path
```

It deliberately does not write to `~/.claude`, so installing Tlaloc cannot silently alter a user's global Claude configuration.
