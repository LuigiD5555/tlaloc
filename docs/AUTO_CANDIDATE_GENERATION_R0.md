# Auto Candidate Generation R0

Tlaloc alpha.20 closes the candidate-provisioning gap in the alpha.19 closed experimental loop.

Previously the loop could decide which mutation deserved a trial, but the corresponding PNG had to be pre-rendered or supplied through a manually authored `build_command`. R0 adds an opt-in path that converts the current adaptive plan into deterministic CandidateConfigs and delegates the actual PNG construction to an explicit target-owned builder.

```text
current experimental incumbent
  -> clean model trials
  -> failure frontier
  -> learning memory
  -> adaptive SuggestedMutations
  -> query builder capabilities
  -> filter unsupported families
  -> one mutation per candidate
  -> target-owned builder
  -> candidate PNG
  -> held-out trials
  -> non-regression + improvement
  -> next experimental incumbent or reject
```

## Configuration

```json
{
  "auto_candidates": true,
  "candidate_builder": ["origami-candidate-build"],
  "auto_candidate_base_profile_id": "origami.temporal-carrier.r0.profile-1",
  "auto_candidates_per_generation": 4
}
```

`candidate_builder` is an explicit local development dependency. It is not inherited by the target model. Clean Native trials still receive only the declared PNG and question.

## Capability negotiation

Before any model trial is spent on an automatically generated candidate, Tlaloc runs:

```text
<candidate_builder...> capabilities
```

R0 expects `origami.experimental-candidate.r0.capabilities` and requires:

- the configured parent profile is supported;
- at least one mutation family is supported;
- `exact_plane_mutation` is false.

Suggestions outside the builder's advertised mutation families are skipped. Tlaloc does not approximate unsupported visual semantics itself.

## Candidate identity

Each generated candidate contains exactly one adaptive mutation. Its ID is derived deterministically from:

```text
parent specimen ID
+ parent PNG SHA-256
+ canonical mutation
```

This makes the experimental DAG reproducible and prevents two visually different parents from silently sharing one candidate identity.

## Builder boundary

Tlaloc creates a CandidateConfig and invokes the explicit builder through the same environment contract already present in alpha.19:

```text
TLALOC_CANDIDATE_ID
TLALOC_OUTPUT_PNG
TLALOC_MUTATIONS_JSON
TLALOC_PARENT_SPECIMEN_ID
TLALOC_PARENT_PNG
```

For Origami, `origami-candidate-build` owns the visual compilation and independently verifies exact-program preservation.

## Evidence remains authoritative

Automatic generation changes only how experimental candidates enter the queue. It does not change the winner criteria.

The existing alpha.19 checks remain authoritative:

- complete clean trials;
- no increase in invented exact claims;
- no missing benchmark questions;
- no per-question score regression;
- minimum configured incumbent improvement;
- diagnostic retries are excluded from the clean score;
- experimental incumbent advancement is not canonical Origami promotion.

## Backward compatibility

When `auto_candidates` is false, the existing alpha.19 `Run` path is used unchanged. Manual candidate banks and explicit per-candidate build commands remain valid.
