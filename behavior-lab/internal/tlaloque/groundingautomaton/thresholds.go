package groundingautomaton

const (
	// AlignmentCandidateThreshold is the minimum claim-normalized lexical
	// coverage required before contradiction rules may inspect an evidence span.
	AlignmentCandidateThreshold = 0.45
	// AlignmentSupportThreshold is deliberately stricter: R0 only grants
	// deterministic SUPPORTED authority when most claim content is explicit.
	AlignmentSupportThreshold = 0.70
)
