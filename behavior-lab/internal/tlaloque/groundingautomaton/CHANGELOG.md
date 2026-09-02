# Grounding Automaton R0 change record

Prepared for local validation; no local test execution is claimed by this record.

Implemented:

- categorical grounding contract and claim-level trace;
- lexical evidence alignment normalized by claim terms;
- polarity, numeric, conservative quantifier, and bounded antonym contradiction checks;
- explicit UNKNOWN/INSUFFICIENT abstention;
- deterministic Tlaloque worker and registry;
- core, metamorphic, and OOD-reserved corpora;
- evaluator with false-supported-contradiction, coverage, precision and recall metrics;
- swarm-ask deterministic-first cascade with existing answer-score fallback;
- unit, integration, contract, threshold, and benchmark test harnesses.

Deferred until evidence warrants it:

- MiniLM alignment assistance;
- retirement of embedding or semantic fallback workers;
- addition of the previously discussed 24 locked OOD triplets, which were not present in the repository baseline.
