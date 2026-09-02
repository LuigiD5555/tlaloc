# Grounding evaluation corpora

- `core-r0.jsonl`: hand-written bilingual support/contradiction/abstention cases.
- `metamorphic-r0.jsonl`: paired deterministic mutations where one semantic fact is flipped.
- `ood-r0.jsonl`: reserved for locked out-of-distribution cases. The current file contains one abstention sentinel only; the previously discussed 24 OOD triplets were not present in the repository baseline and must be added here without tuning the automaton against them.

Keep evaluation data separate from implementation tuning. In particular, do not change an OOD expected label to make a regression pass.
