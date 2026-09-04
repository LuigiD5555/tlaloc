# tlaloquekit — Tlaloc's public publication surface (R1)

Added for TONAL T1. This is the boundary through which Tlaloc publishes the
Tlaloques it has built and qualified to a consuming system (Tonal's
runtime).

## Why it exists

Before T1 every runtime primitive lived under
`behavior-lab/internal/*` and could only be imported inside the
`tlaloc.local/behaviorlab` module. Tonal's runtime is a separate module and
Go forbids importing another module's `internal/` packages. `tlaloquekit`
is the stable, dependency-free contract Tonal imports instead.

## What it is

Package `tlaloc.local/behaviorlab/tlaloquekit`:

- `Descriptor`, `Candidate`, `Goal`, `PlanNode`, `Resolution`,
  `Observation`, `ExecutionRequest`, `ExecutionResult`, `Usage` — plain Go
  structs and JSON, no internal types.
- `QualifiedRegistry` interface: `Capabilities()`, `Candidates()`,
  `Resolve()`, `Execute()`, `ParrotProfileID()`, `ParrotProfileHash()`.
- `BuildQualifiedRegistry(Config)` — wires the internal `tlaloque.Registry`,
  installs the `exocortex.CapabilityRouter` (deterministic-first,
  smallest-first ranking + CapabilityProfile veto), and registers the
  qualified Tlaloque set.

The internal `Registry`, `CapabilityRouter`, `exocortex` Tlaloques and
`blackboard` remain implementation details behind this facade.

## What it is NOT

Not a workflow engine. It holds no Blackboard, runs no scheduler, keeps no
execution state. Goal intake, the DAG walk, Blackboard ownership, workflow
routing, verification coordination and accounting belong to the consuming
runtime (Tonal).

## Qualified Tlaloque set (T1)

| Capability | Tlaloque | Engine |
|---|---|---|
| `LOCATE_REGION` | `region-locate-tlaloque` | deterministic |
| `CROP_REGION` | `region-crop-tlaloque` | deterministic |
| `NORMALIZE` | `normalize-tlaloque` | deterministic |
| `COMPARE_NUMBERS` | `numeric-tlaloque` | deterministic |
| `ARITHMETIC` | `arithmetic-tlaloque` | deterministic (**new for T1**) |
| `VERIFY` | `verify-tlaloque` | deterministic |
| `EXTRACT_NUMBER` | Parrot Tlaloque (CapabilityProfile R1 + AdapterR1) | generative — *wired in the next T1 increment* |

## ARITHMETIC opcode (new)

Micro-ISA R0 carried `COMPARE_NUMBERS` / `SAME_DIFFERENT` but no
subtraction / ratio / percentage-difference, which the depth-8/12 T1
workflow families need. `OpArithmetic = "ARITHMETIC"` is added to the
opcode vocabulary and served by the deterministic `ArithmeticTlaloque`:

- operations: `SUBTRACT` (a−b), `RATIO` (a/b), `PERCENT_DIFFERENCE`
  (100·(a−b)/b);
- typed input/output, no eval strings, no generative execution;
- division by zero / undefined percentage base → `Status: INVALID_INPUT`,
  `has_result: false` (the workflow branches, it does not crash);
- non-numeric operand or unknown operation → error.

## Frozen artifacts untouched

CapabilityProfile R1 contents and hash (`8acc959b…`), and every P1/P2/T0/R1/
Phase-H result, are unchanged. This change is purely additive: a new public
package, a new opcode, a new deterministic Tlaloque, and their tests.
