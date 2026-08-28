# Tlaloc <-> Origami integration contract — R0

Tlaloc treats Origami as an independent representation provider, not as the orchestration system and not as a mandatory dependency.

## Origami may provide

- canonical state schemas;
- supported operations and invariants;
- a deterministic **reference semantics engine** for valid transitions;
- serialization / projection contracts;
- explicit resolution boundaries such as `OBSERVE` and declared `FOLD` policies.

## Tlaloc builds verification assets

Tlaloc can use Origami's reference semantics engine to construct expected-state fixtures and behavior-evaluation campaigns. Tlaloc owns the evaluation campaign; Origami remains authoritative for the semantics being evaluated.

## Tlaloc supplies

- behavior-compilation lifecycle;
- target-model execution;
- Tlaloque diagnosis and compiled-artifact patch proposals;
- regression campaign coordination;
- promotion/rejection decision records.

## Boundary rule

Tlaloc may learn how to make a model obey Origami semantics. It must not redefine those semantics. Origami may be used independently of Tlaloc, and Tlaloc may operate without Origami.
