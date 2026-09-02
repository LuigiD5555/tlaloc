package groundingautomaton

import "testing"

func TestAlignmentThresholdOrdering(t *testing.T) {
	if AlignmentCandidateThreshold <= 0 || AlignmentSupportThreshold > 1 || AlignmentCandidateThreshold >= AlignmentSupportThreshold {
		t.Fatalf("invalid alignment thresholds: candidate=%f support=%f", AlignmentCandidateThreshold, AlignmentSupportThreshold)
	}
}
