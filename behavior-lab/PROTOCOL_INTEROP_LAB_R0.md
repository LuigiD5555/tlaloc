# Origami Protocol Interop Lab R0

Status: `EXPERIMENTAL_REFERENCE_IMPLEMENTED_EVIDENCE_PENDING`

This lab evaluates Origami as a communication protocol, not merely as a carrier format.

## Target sequence

```text
semantic ground truth
  -> model A reads/writes Origami through declared S*/E* codecs
  -> model B reads that Origami and writes another compatible message
  -> model C may continue the chain
  -> deterministic structural evaluator measures drift
```

## Evidence classes

### Synthetic fixture

Used only to verify the evaluator itself.

`testdata/protocol/perfect-multihop.json` must score zero drift, but it is **not** interoperability evidence.

### Real held-out trial

A real trial must record:

- exact model/version identity;
- exact Master Prompt / protocol bootstrap identity;
- exact Origami carrier/profile identity;
- raw model output verbatim;
- decoder codec the model says it used;
- encoder codec the model says it used;
- decoded semantic state;
- semantics of the re-encoded Origami after deterministic/reference decode when available;
- whether tools, filesystem, sandbox or exact runtime were available.

Use `testdata/protocol/REAL_TRIAL_TEMPLATE.json` as the starting artifact.

## First read gate

```text
profile-3 PNG
+ Master Prompt R4
+ question: What is the index?

expected route:
T0 -> T1 -> S2 -> T2
```

For this semantic question:

- self-declared `S2` is valid;
- an undeclared external decoder/file dependency is a failure;
- binary extraction/decompression is unnecessary exact escalation;
- invented byte/hash/compression claims are failures;
- exact-plane unavailability is not itself a failure.

## First write gate

Give a model a bounded semantic index and require it to construct an Origami-compatible output through the declared encoder:

```text
Semantic IR -> E2 -> T2 Construction IR
```

If no deterministic image compiler is available, `CONSTRUCTION_SPEC_ONLY` is legitimate. The model must not claim a verified PNG that it did not actually compile.

## Cross-model gate

For A -> B -> C, evaluate:

```text
entities preserved
relations preserved
hierarchy preserved
evidence preserved
uncertainty preserved
invented facts
lost facts
codec discovery
hop-to-hop drift
final drift
external codec dependency
semantic-to-exact escalation
```

Reference gates begin at:

```text
final semantic drift <= 0.05
invented fact rate = 0
undeclared external codec dependencies = 0
semantic-to-exact escalations = 0
FALSE_EXACT = 0
```

These thresholds are development references, not proof that a profile is universally interoperable.

## CLI

```bash
go run ./cmd/tlaloc-protocol-eval \
  -in testdata/protocol/perfect-multihop.json \
  -out -
```

For the legacy Native index regression:

```bash
go run ./cmd/tlaloc-native-eval -in trial.json -out -
```

Alpha.16 changes the interpretation of decoder dependency:

```text
DECLARED S2 DECODER = ALLOWED
UNDECLARED EXTERNAL DECODER = FAILURE
SEMANTIC QUERY -> EXACT/BINARY ROUTE WITHOUT NEED = FAILURE
```

## Promotion boundary

Tlaloc recommends candidates and records evidence. Origami remains the authority for Origami profile/protocol promotion. Tonal may pin a reproducible composition, but composition verification is not Native/interoperability promotion.
