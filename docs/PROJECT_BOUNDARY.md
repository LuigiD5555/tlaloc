# Tlaloc project boundary

Tlaloc is a development kit, not the product it helps build.

```text
TARGET INTENT
    |
    v
TLALOC DEVELOPMENT LAB
BehaviorSpec -> Tlaloque swarm -> tests -> reference behavior
    |
    v
DISTILLATION
prompt candidates -> clean target evaluation
    |
    v
PORTABLE ARTIFACT
prefer L0 prompt-only
```

## Tlaloc owns

- decomposition of a requested behavior into bounded experimental work;
- Tlaloque coordination during development;
- reference execution traces and experiment evidence;
- prompt/PromptIR candidate generation and mutation;
- behavioral-fidelity evaluation;
- clean-target testing;
- prompt-first deployment selection;
- target-specific adapters and laboratories.

## Tlaloc does not own

- the semantics/releases of a target project;
- Origami's canonical profile or Master Prompt release decisions;
- Tonal composition policy;
- hidden runtime dependencies inside an L0 prompt artifact.

## Target examples

Origami is one target. A calculator, classifier, document workflow or other behavior can use the same Tlaloc discovery/distillation lifecycle.

## Deployment boundary

Development resources may be rich. Deployment requirements must be explicit.

```text
L0 prompt only
L1 prompt + declarative context/IR
L2 prompt + tools
L3 prompt + runtime
L4 specialized target
```

Tlaloc prefers the lowest level that satisfies the BehaviorSpec.
