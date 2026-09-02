package groundingautomaton

// ReasonCodes returns the ordered reason codes emitted for a claim trace. It
// is useful for concise provenance without discarding the full trace payload.
func ReasonCodes(trace ClaimTrace) []ReasonCode {
	codes := make([]ReasonCode, 0, len(trace.Reasons))
	for _, reason := range trace.Reasons {
		codes = append(codes, reason.Code)
	}
	return codes
}
