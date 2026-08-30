# Change Control — Tlaloc 6.0.0-alpha.16

## Change

Add deterministic development/evaluation support for Origami Protocol R0 read/write interoperability and refine Native semantic decoder-dependency semantics.

## Before

Alpha.15 correctly turned a failed Origami index trial into a regression, but its evaluator treated decoder dependency broadly. It did not yet distinguish a self-declared Origami semantic decoder from an undeclared external decoder, and it had no structural evaluator for encoder discovery, semantic roundtrip or A -> B -> C communication drift.

## After

Alpha.16 adds:

- declared semantic decoder/encoder awareness (`S2`/`E2` reference pair);
- deterministic READ/WRITE/ROUNDTRIP/MULTIHOP protocol evaluation;
- structural preservation metrics for entities, relations, hierarchy, evidence and uncertainty;
- invented-fact and semantic-drift measurement;
- cross-model read/write success and hop-to-hop/final drift;
- separate violations for undeclared external-codec dependency and unnecessary semantic-to-exact escalation;
- `tlaloc-protocol-eval`;
- synthetic evaluator fixtures plus a separate held-out real-trial template.

## Interpretation correction

```text
SELF_DECLARED_SEMANTIC_DECODER = ALLOWED
UNDECLARED_EXTERNAL_DECODER_DEPENDENCY = FAILURE
SEMANTIC_TO_EXACT_ESCALATION_WITHOUT_NEED = FAILURE
```

The old `mechanical_dependency_violation` result remains a compatibility aggregate for the two failure classes; it is no longer interpreted as “any decoder use is invalid”.

## Evidence boundary

Synthetic perfect trials validate the evaluator implementation only. They do not demonstrate that two real held-out models can communicate through Origami.

Real cross-model evidence remains pending and must include verbatim target outputs and explicit model/prompt/carrier identities.

## Authority boundary

Tlaloc evaluates and recommends. Origami owns Origami protocol/profile semantics and promotion. Tonal may later pin an exact verified composition, but composition verification does not promote model capability.

## Gates

- `go test ./...`
- `go vet ./...`
- `go test -race ./...`
- version/terminology/skill/gatekeeper coherence
- isolated managed install lifecycle
- generated artifact hashes
- `FALSE_EXACT=0` wherever exactness is claimed

## Release

`6.0.0-alpha.16`
