# Tlaloc 6.0.0-alpha.11 — Canonical Memory Plane R2

This release candidate closes the document-to-memory orchestration path required by Origami Fixed Carrier R2.

- Canonical Document IR preserves page geometry, reading order, regions and figures, with OCR fallback for image-only pages.
- Exact page/block/region address space is content-addressed and Merkle-bound.
- Tlaloques produce evidence-backed candidates; they do not directly mutate CanonicalState.
- A deterministic Go reducer creates CanonicalState, unresolved Conflict records and a reproducible state hash.
- UncertaintyController produces deterministic verification plans instead of majority voting.
- External Recursive Attention routes H0-H5 under a bounded active context.
- `tlaloc.origami-tools.r2` exposes BOOT / QUERY / EXPAND / VERIFY with native-function and plaintext bridge transports.
- Fixed Carrier R2 T0/T1/T2/T3 bootstrap and visual-probe negotiation are supported.

`FALSE_EXACT=0`, evidence provenance and deterministic replay remain hard invariants.
