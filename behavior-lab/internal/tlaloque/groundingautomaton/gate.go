package groundingautomaton

// HasAuthority reports whether the deterministic R0 automaton has enough
// evidence to make the final grounding decision without a learned fallback.
func HasAuthority(out VerifyOutput) bool {
	return out.Verdict == VerdictSupported || out.Verdict == VerdictContradicted
}
