package canonicalstate

import "testing"

func TestControllerQueuesConflicts(t *testing.T) {
	s := State{StateHash: "x", Metrics: Metrics{Uncertainty: .7}, Claims: []CanonicalClaim{{CandidateIDs: []string{"a", "b"}, Evidence: []EvidenceRef{{Address: "ohf://x/p/1"}}}}, Conflicts: []Conflict{{ID: "c", PositiveIDs: []string{"a"}, NegativeIDs: []string{"b"}}}}
	p := BuildVerificationPlan(s, 4000)
	if p.Action != "VERIFY_CONFLICTS" || len(p.Tasks) != 1 {
		t.Fatalf("unexpected plan %+v", p)
	}
}
