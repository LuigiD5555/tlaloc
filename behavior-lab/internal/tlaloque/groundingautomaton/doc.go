// Package groundingautomaton implements a deterministic, claim-oriented
// grounding verifier. It is intentionally conservative: explicit lexical
// evidence can produce SUPPORTED or CONTRADICTED, while ambiguous cases
// abstain as UNKNOWN or INSUFFICIENT for downstream learned fallbacks.
package groundingautomaton
