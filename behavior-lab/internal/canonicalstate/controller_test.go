package canonicalstate

import "testing"

func TestControllerQueuesConflicts(t *testing.T) {
	s := State{StateHash: "x", Metrics: Metrics{Uncertainty: .7}, Claims: []CanonicalClaim{{CandidateIDs: []string{"a", "b"}, Evidence: []EvidenceRef{{Address: "ohf://x/p/1"}}}}, Conflicts: []Conflict{{ID: "c", PositiveIDs: []string{"a"}, NegativeIDs: []string{"b"}}}}
	p := BuildVerificationPlan(s, 4000)
	if p.Action != "VERIFY_CONFLICTS" || len(p.Tasks) != 1 {
		t.Fatalf("unexpected plan %+v", p)
	}
}

func TestControllerExpandsEvidenceWhenUncertainWithoutConflict(t *testing.T) {
	p := BuildVerificationPlan(State{StateHash: "x", Metrics: Metrics{Uncertainty: .7}}, 4000)
	if p.Action != "EXPAND_EVIDENCE" {
		t.Fatalf("action=%s", p.Action)
	}
}

func TestControllerCompletesWhenSatisfied(t *testing.T) {
	p := BuildVerificationPlan(State{StateHash: "x", Metrics: Metrics{Uncertainty: .1}}, 4000)
	if p.Action != "REDUCE_COMPLETE" {
		t.Fatalf("action=%s", p.Action)
	}
}

func TestControllerTransitionTableRejectsImplicitUnknownTransition(t *testing.T) {
	if got := nextControllerState(ControllerComplete, State{Metrics: Metrics{Uncertainty: 1}}); got != ControllerComplete {
		t.Fatalf("complete state moved to %s", got)
	}
}
