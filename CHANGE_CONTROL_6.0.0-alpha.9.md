# Change control — Tlaloc 6.0.0-alpha.9

## Decision

Consume Origami `6.0.0-alpha.4` relational-core fixtures through an explicit profile registry while preserving the coherent-state profile as an independent compatible path.

## Changes

- registered coherent and relational profiles with exact ID/version selection;
- unknown profiles and version drift return `UNSUPPORTED`;
- relational fixtures are byte-identical to the upstream Origami artifact;
- profile-specific comparators, curricula, and bounded Tlaloque;
- strict coherent-state decoding and causal `CANCELLED` representation;
- removed unimplemented `FOLD`, `UNFOLD`, and `EVOLVE` claims from the coherent executable profile.

## Gates

Go tests, vet, race tests, deterministic prompt generation, artifact hashes, terminology/version checks, and architecture verification must pass before promotion.
