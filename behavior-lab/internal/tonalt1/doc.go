// Package tonalt1 builds and freezes the TONAL T1 deterministic held-out
// operand universe (experiment step D3).
//
// D3 answers exactly one question: which physical source instances in the
// existing canonical 1152-page D2L store (fold-bench-reconstructed-r0) are
// legitimate NEW held-out operands for TONAL T1, under the already-frozen
// Parrot R1 perceptual envelope?
//
// D3 is entirely deterministic. No function here reads a model output, a
// scorer verdict, an expected workflow answer, or performs any manual
// visual selection. Eligibility is decided solely from frozen code/data
// rules declared before any T1 inference.
//
// D3 does NOT build workflows, assign A/B/a/b roles, compute workflow gold,
// or inspect T1 model output. It emits the frozen candidate universe that
// step D4 later consumes.
//
// Pipeline (protocol section 6):
//
//	canonical store
//	  -> deterministic raw scan            (scan.go)
//	  -> raw candidate universe
//	  -> physical identity derivation      (identity.go)
//	  -> prior-use exclusion (union)       (prioruse.go)
//	  -> frozen R1-envelope eligibility    (envelope.go)
//	  -> geometry / ambiguity checks       (geometry.go)
//	  -> domain validity checks            (eligibility.go)
//	  -> FINAL HELD-OUT ELIGIBLE UNIVERSE
//	  -> freeze                            (freeze.go)
package tonalt1

// SelectorVersion is bumped whenever the deterministic scan / filter /
// identity / eligibility logic changes in a way that could move which
// candidates are selected or how they are identified. Frozen artifacts
// record it and the D3 freeze manifest hashes it.
const SelectorVersion = "tonalt1.d3.selector.r2.0.0"

// RuleAuditVersion versions the D3 v2 rule-provenance classification
// (types.go ruleClassOf): which selector rules are CAPABILITY /
// PRESENTATION_INTEGRITY / DOMAIN_VALIDITY (blocking) vs
// DATASET_AUTHORING_HEURISTIC (advisory only).
const RuleAuditVersion = "tonalt1.ruleaudit.r2.0.0"

// SpanNormVersion versions the normalized source-span canonicalization
// (spanhash.go). Frozen; recorded in every candidate and the manifest.
const SpanNormVersion = "tonalt1.spannorm.r1.0.0"

// EnvelopeVersion versions the frozen R1 perceptual-envelope eligibility
// rule set (envelope.go). Derived from frozen R1 / R1-C real-document
// evidence only.
const EnvelopeVersion = "tonalt1.envelope.r1.0.0"

// GeometryRuleVersion versions the frozen geometry / ambiguity rule set
// (geometry.go).
const GeometryRuleVersion = "tonalt1.geometry.r2.0.0"

// PriorUseInventoryVersion versions the set of prior experiments and the
// extractors that reconstruct their consumed physical instances
// (prioruse.go).
const PriorUseInventoryVersion = "tonalt1.prioruse.r1.0.0"

// ExperimentID is the owning experiment identifier.
const ExperimentID = "tonal-t1-heterogeneous-composition"

// Seed is the fixed deterministic partition seed, shared with R1.
const Seed = "20260903"

// ExpectedPrimaryUniqueOperandDemand is the frozen T1 primary-workflow
// operand demand: 12 * (1 + 2 + 2 + 3 + 4) = 144, under
// PRIMARY_WORKFLOW_TARGET_REUSE = false. D4 authoritatively re-derives this
// from the Tonal TaskFamily definitions; D3 records it as protocol
// metadata with this provenance so the headroom report is meaningful.
const ExpectedPrimaryUniqueOperandDemand = 144
