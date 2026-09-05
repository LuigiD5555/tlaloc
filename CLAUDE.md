# Tlaloc project instructions — Architecture R2

## Role

Tlaloc is Tonal's capability foundry, Behavior Lab and reusable-machinery lifecycle subsystem.

Tlaloc discovers, constructs, tests, qualifies, packages, promotes, deprecates and studies bounded reusable mechanisms. It does not own Tonal's runtime, scheduler, Blackboard or external general-purpose cognition.

## Shared North Star

The Tonal ecosystem investigates whether complex, reliable behavior can emerge from reusable machinery plus selectively invoked external cognition instead of requiring one increasingly complex general-purpose model to perform every operation.

Tlaloc's contribution is to find the smallest reliable reusable structure that removes recurring uncertainty or repeated work.

## Canonical terms

- **Tonal**: complete heterogeneous runtime and research system.
- **Tlaloc**: capability-development, experimentation, qualification and machinery-lifecycle subsystem.
- **Tlaloque**: bounded, typed, measurable reusable machinery produced or qualified through Tlaloc. It may be deterministic, algorithmic, symbolic, tool-backed, specialized-model-backed or hybrid.
- **Parrot**: Tonal's singular external probabilistic cognition interface. It is **not a Tlaloque** and is not produced or promoted by Tlaloc.
- **Shponglese**: semantic operational IR for primitive and composed behavior.
- **Origami**: independent representation, transport and virtual-memory substrate.

## Parrot relationship

Tlaloc may characterize Parrot empirically, test competence envelopes around Parrot-assisted operations and analyze Episodes containing Parrot calls.

Tlaloc must not treat the external model itself as one of its manufactured Tlaloques. Instead, recurring verified Parrot behavior may become input to a machinery-discovery process:

```text
Parrot-assisted Episodes
      ↓
recurring bounded behavior?
      ↓
candidate Tlaloque / Machine / specialist
      ↓
holdout + ablation + verification
      ↓
promote or reject
```

The goal is not to eliminate Parrot universally. The goal is to stop spending external cognition on behavior that can be made reliably reusable.

## Capability lifecycle

Tlaloc should not assume that successful behavior must ultimately become a prompt.

The preferred output is the smallest reliable reusable representation justified by evidence. Depending on the behavior, that may be a deterministic operation, Machine/state machine, tool wrapper, Shponglese motif/program, prompt/policy artifact, small specialized model, Tlaloque or hybrid mechanism.

Use `BehaviorSpec + invariants` as behavioral source of truth. Treat prompts, models, tools and skills as derived/operational artifacts rather than behavioral authority.

## Invariants

When changing Tlaloc:

- keep Tlaloques bounded and unable to self-promote;
- never classify the external Parrot model itself as a Tlaloque;
- keep representation ownership outside Tlaloc;
- keep Tonal runtime authority outside Tlaloc;
- keep current capability claims aligned with code and evidence;
- preserve deterministic authority where deterministic verification is possible;
- preserve installer/uninstaller safety and retrocompatibility;
- prefer explicit evidence over model prestige or size;
- preserve experiment freeze boundaries.

T1 compatibility code may retain historical `Parrot Tlaloque` terminology where required to describe the frozen R1 contract. Do not silently rewrite frozen artifacts; current R2 documents and new APIs must use the corrected ontology.

Run `go test ./...`, `go vet ./...`, and `go test -race ./...` in `behavior-lab` before promotion.

## Episodes and Cognitive JIT direction

Tonal execution traces may be adapted into Episodes. Episodes are evidence records, not automatic training truth.

Future Tlaloc work may use verified Episode corpora to discover recurring trajectories, candidate abstractions, smaller specialists or deterministic Machines. Recurrence alone is not promotion: holdout reliability, failure analysis, verification and cost/complexity tradeoffs are required.

## Document authority

Read in this order:

1. `CLAUDE.md`
2. `README.md`
3. `docs/CURRENT_STATE.md`
4. `docs/ARCHITECTURE.md`
5. `docs/ROLE_IN_TONAL.md`
6. active Behavior Lab / experiment specification

Historical material under `docs/archive/` is not current authority. Frozen experiment specifications remain authoritative for their experiment.

See `.claude/skills/` for task-specific workflows.
