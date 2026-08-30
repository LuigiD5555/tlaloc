# Tlaloc <-> Origami integration contract — R0

Tlaloc treats Origami as an independent representation/state-machine provider, not as the orchestration system and not as a mandatory dependency.

## Origami may provide

- canonical state schemas;
- supported operations and invariants;
- deterministic reference semantics for valid transitions;
- Fold/Unfold, addressability and verification contracts;
- serialization/projection contracts;
- perceptual-channel contracts;
- the self-boot receiver contract `origami.hybrid-receiver.r0`;
- carrier fixtures exposing `BOOT -> ROSETTA -> PROGRAM -> INDEX -> MEMORY -> VERIFICATION`;
- explicit resolution/exactness boundaries.

## Tlaloc builds search and verification assets

Tlaloc can use Origami's reference semantics and receiver fixtures to construct behavior-evaluation campaigns. Tlaloc owns the search/evaluation campaign; Origami remains authoritative for the semantics being evaluated.

For the Hybrid Receiver track, Tlaloc may run richer swarm/Tlaloque behavior to discover a successful reception/navigation strategy and then distill the externally relevant behavior into a candidate package:

```text
swarm behavior
   ↓
trace / findings / successful route
   ↓
receiver distillation
   ├── universal Master Prompt candidate
   ├── BOOT discovery strategy
   ├── Rosetta discovery constraints
   └── deterministic micro-agent candidate rules
   ↓
fitness tournament / cross-model evaluation
   ↓
promotion recommendation
   ↓
Origami semantic validation + storage
```

The current reference contract for this work is `tlaloc.origami-receiver-distillation.r0`.

## Tlaloc supplies

- behavior-compilation lifecycle;
- target-model execution;
- swarm/Tlaloque exploration;
- candidate prompt mutation/repair;
- distillation from rich behavior traces into bounded deterministic candidate transitions;
- receiver-candidate tournaments;
- cross-model regression coordination;
- promotion/rejection recommendations and evidence.

## Origami retains authority

A Tlaloc candidate is never automatically an Origami artifact. Origami must validate that the candidate preserves its semantic contracts before storing/exporting it.

Tlaloc must not:

- assign global meaning to carrier-local glyphs;
- redefine Origami states or transitions;
- collapse UNKNOWN/absence/inhibition/cancellation distinctions;
- silently weaken guards, conditions, exceptions or exactness boundaries;
- promote a contaminated or false-exact candidate merely because its aggregate score is high.

Origami owns promoted receiver artifacts and their provenance. Tlaloc stores experiments/candidates as training evidence, not as the canonical receiver registry.

## Receiver fitness gates

Receiver search should measure at least:

- BOOT discovery;
- carrier-local Rosetta resolution;
- program initialization;
- navigation correctness;
- answer correctness;
- evidence integrity;
- UNKNOWN accuracy;
- peak active model-facing token-equivalent;
- tool/model cost;
- false-exact count;
- contamination.

`FALSE_EXACT != 0`, contamination, or a configured active-window violation makes a candidate ineligible regardless of aggregate score.

## Hybrid division of labor

The preferred target is not a model that performs all decoding by itself and not a runtime that ignores the visual carrier. It is:

```text
model perception
  -> find BOOT / understand carrier-local Rosetta / choose region
Origami deterministic runtime
  -> execute micro-agents / address / compute / verify
model
  -> integrate compact state / choose next bounded access / answer
```

Native-only and Computational-only remain diagnostic baselines.

## Boundary rule

Tlaloc may learn how to make a model bootstrap and operate Origami. It may propose how successful swarm behavior can be compiled into simpler receiver machinery. It must not redefine what Origami means.

Origami may be used independently of Tlaloc, and Tlaloc may operate without Origami.

## Contract tracking

Tlaloc alpha.11 recognizes the experimental Origami Fixed Carrier R2 contract from Origami alpha.5 in addition to earlier contract IDs. This is an implemented local integration path, not a Tonal SUPPORTED composition claim until both component revisions are merged and Tonal pins immutable commits.

## Fixed Carrier R2 integration

For `origami.fixed-carrier.r2`, Tlaloc may provide `tlaloc.origami-tools.r2` as a declared external data/tool plane. Tlaloc must validate the carrier through the independent Origami decoder, match carrier/store/source roots, and verify page CIDs before returning exact data. OCR failure is not a BOOT failure. A missing image returns `ORIGAMI_IMAGE_UNAVAILABLE`; a missing tool plane for external exact data returns `ORIGAMI_TOOL_REQUIRED`.
