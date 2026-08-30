# Tlaloc Canonical Memory Plane R2

Tlaloc R2 converts heterogeneous documents into a canonical, addressable and verifiable memory universe before Origami represents its control plane.

## Pipeline

```text
PDF digital ---- native layout extraction --\
scan/image ----- raster + OCR fallback -------> Canonical Document IR
                                              -> exact plane
                                              -> structural plane
                                              -> Tlaloque candidates
                                              -> deterministic Go reducer
                                              -> CanonicalState
                                              -> Merkle/CID store
                                              -> External Recursive Attention runtime
```

### Canonical Document IR

Pages preserve dimensions, reading order, text regions, bounding boxes, headings/code/equation/table-cell heuristics and figure regions. Pages with insufficient native text use the OCR path. Stable `ohf://` addresses and CIDs are assigned before semantic reduction.

### Models propose; protocol decides

Tlaloques emit `Candidate` objects with explicit evidence. They never write CanonicalState. The Go reducer normalizes candidates, verifies evidence, merges compatible claims, records contradictions as unresolved conflicts, and produces a deterministic state hash. Same candidates + evidence + reducer version must produce the same CanonicalState.

### Uncertainty controller

Conflicts or insufficient evidence produce a deterministic verification queue rather than a blind vote. More evidence/graph depth is allocated only where uncertainty requires it.

### External Recursive Attention

`origami_query` navigates H0 intent -> H1 documents -> H2 graph -> H3 candidate objects -> H4 selective unfold -> H5 exact verified evidence. Active context remains bounded while total memory may grow.

### Exact/semantic separation

Semantic claims point to exact page/block/region evidence. Exact results are verified by CID and Merkle proof against the store root bound into Origami Fixed Carrier R2.

Tool ABI: `origami_boot`, `origami_query`, `origami_expand`, `origami_verify`.
