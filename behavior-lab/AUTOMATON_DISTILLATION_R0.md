# Tlaloc Automaton Distillation — R0

Status: `EXPERIMENTAL_REFERENCE_IMPLEMENTATION`

Tlaloc may use a large bounded Tlaloque swarm while discovering a behavior, but the portable artifact should contain the smallest demonstrated behavior that still reproduces the task.

R0 adds a new distillation target:

```text
successful Tlaloque action trace
        ↓
canonical ordered transitions
        ↓
repeated local behavior deduplication
        ↓
cells + rules + declared graph edges
        ↓
origami.automaton.r0-compatible IR
```

A development Tlaloque is not automatically a permanent runtime process. The distiller extracts the local transition behavior that was actually demonstrated.

## Input trace

`tlaloc.tlaloque-action-trace.r0` records explicit:

```text
step
tlaloque
from_state
to_state
requires[]
emits_to[]
```

Tlaloc does not infer hidden dependencies. If a dependency matters to the portable automaton, it must be present in the trace.

## Output

The output is intentionally compatible with Origami's `origami.automaton.r0` structure:

```text
cells
rules
edges
source_trace_sha256
```

Tlaloc does not import or redefine Origami's release authority. Origami decides whether the emitted representation becomes part of an Origami release/profile.

## Deduplication

Repeated identical transitions become one rule. R0 reports:

```text
trace_steps
unique_cells
unique_rules
repeated_transitions_removed
distillation_ratio
```

The ratio is a structural-development metric, not proof that a visual carrier is smaller or that a VLM understands it.

## CLI

```bash
tlaloc-automaton-distill \
  -in behavior-lab/testdata/automata/signal-chain-trace.json \
  -out automaton.json
```

## Boundary

```text
SWARM TRACE != DEPLOYMENT ARTIFACT
DISTILLED AUTOMATON != ORIGINAL SWARM
TLALOC != TARGET RUNTIME REQUIREMENT
TARGET PROJECT OWNS ADOPTION
```
