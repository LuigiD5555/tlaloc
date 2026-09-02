package distill

import "testing"

func TestDistillCollapsesRepeatedDeterministicSteps(t *testing.T) {
	trace := []SwarmStep{
		{Agent: "mapper", State: "boot", Token: "BOOT_FOUND", Action: "FOLLOW", NextState: "rosetta"},
		{Agent: "mapper-2", State: "boot", Token: "BOOT_FOUND", Action: "FOLLOW", NextState: "rosetta"},
		{Agent: "decoder", State: "rosetta", Token: "ROSETTA_READY", Action: "INIT", NextState: "program"},
		{Agent: "executor", State: "program", Token: "VALUE", Action: "EMIT", NextState: "done", Emit: "AMBER-10593"},
	}
	c, err := Distill("bootstrap prompt", trace)
	if err != nil {
		t.Fatal(err)
	}
	if c.Schema != ContractID {
		t.Fatalf("unexpected schema %q", c.Schema)
	}
	if len(c.Program) != 3 {
		t.Fatalf("expected 3 unique rules, got %d", len(c.Program))
	}
	if c.SourceTraceSHA256 == "" || c.ID == "" {
		t.Fatal("missing provenance identity")
	}
}

func TestDistillRejectsConflictingTransition(t *testing.T) {
	trace := []SwarmStep{
		{State: "s0", Token: "X", Action: "FOLLOW", NextState: "s1"},
		{State: "s0", Token: "X", Action: "STOP", NextState: "done"},
	}
	if _, err := Distill("p", trace); err == nil {
		t.Fatal("expected non-deterministic swarm trace to be rejected")
	}
}

func TestTournamentRejectsFalseExactAndContamination(t *testing.T) {
	good := ScoredCandidate{
		Candidate: Candidate{ID: "good"},
		Fitness:   Fitness{BootstrapSuccess: 1, RosettaSuccess: 1, Navigation: .95, Correctness: .98, Evidence: .95, UnknownAccuracy: 1, PeakActiveTokenEq: 900},
	}
	falseExact := ScoredCandidate{
		Candidate: Candidate{ID: "false-exact"},
		Fitness:   Fitness{BootstrapSuccess: 1, RosettaSuccess: 1, Navigation: 1, Correctness: 1, Evidence: 1, UnknownAccuracy: 1, FalseExact: 1, PeakActiveTokenEq: 100},
	}
	contaminated := ScoredCandidate{
		Candidate: Candidate{ID: "contaminated"},
		Fitness:   Fitness{BootstrapSuccess: 1, RosettaSuccess: 1, Navigation: 1, Correctness: 1, Evidence: 1, UnknownAccuracy: 1, PeakActiveTokenEq: 100, Contaminated: true},
	}
	winner, err := Winner([]ScoredCandidate{falseExact, contaminated, good}, 4000)
	if err != nil {
		t.Fatal(err)
	}
	if winner.Candidate.ID != "good" {
		t.Fatalf("unexpected winner %q", winner.Candidate.ID)
	}
}

func TestTournamentRejectsWindowViolation(t *testing.T) {
	c := ScoredCandidate{
		Candidate: Candidate{ID: "too-wide"},
		Fitness:   Fitness{BootstrapSuccess: 1, RosettaSuccess: 1, Navigation: 1, Correctness: 1, Evidence: 1, UnknownAccuracy: 1, PeakActiveTokenEq: 4001},
	}
	if _, err := Winner([]ScoredCandidate{c}, 4000); err == nil {
		t.Fatal("expected no eligible candidate")
	}
}
