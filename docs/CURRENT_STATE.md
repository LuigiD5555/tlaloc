# Tlaloc current state — R2 Foundation

**Status:** migration foundation in progress  
**Role:** Tonal capability foundry + Behavior Lab  
**Runtime authority:** Tonal, not Tlaloc

## Current direction

Tlaloc develops reusable capabilities for Tonal through bounded experiments, qualification, evidence and promotion.

The project retains its strongest existing ideas:

- bounded Tlaloques;
- BehaviorSpec + invariants as behavior authority;
- deterministic verification where possible;
- CapabilityProfile/competence evidence;
- Behavior Lab campaigns;
- no self-promotion by workers;
- explicit provenance and failure evidence.

Architecture R2 broadens the target of distillation. Tlaloc no longer assumes that the preferred final artifact is always a portable prompt.

Instead, Tlaloc should prefer the smallest reliable reusable representation justified by evidence. This may be a deterministic operation, Machine/state machine, tool wrapper, Shponglese motif/program, prompt/policy, specialized model, probabilistic Tlaloque or hybrid capability.

## Tonal relationship

Tlaloc produces or qualifies capabilities. Tonal decides when and how to use them at runtime.

```text
Tlaloc
  build / test / qualify / promote
             ↓
      Capability Registry
             ↓
           Tonal
  select / execute / verify / account
```

## Parrot

Parrot is one probabilistic Tlaloque. It is characterized and qualified like other capabilities and has no privileged system role.

## Episode direction

Verified execution traces may be normalized into Episodes. Future Cognitive JIT work may use Episode corpora to discover recurring reliable structure, but recurrence alone is insufficient for promotion.

## Immediate R2 work

1. align current documentation with Tonal Architecture R2;
2. archive superseded nomenclature/architecture material;
3. stabilize Episode and capability contracts;
4. support Tonal's generic Capability/SelectionPolicy runtime;
5. later begin Primitive Swarm / MICRO-ISA research.
