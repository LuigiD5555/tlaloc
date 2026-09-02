package groundingautomaton

import "testing"

func TestVerifySupportedParaphrase(t *testing.T) {
	out := Verify(VerifyInput{
		ModelAnswer: "Distributed agents coordinate locally in a swarm.",
		PageContent: "A swarm is a system composed of multiple distributed agents that coordinate their behavior through local interactions.",
	})
	if out.Verdict != VerdictSupported {
		t.Fatalf("expected SUPPORTED, got %s (%+v)", out.Verdict, out)
	}
}

func TestVerifyPolarityContradiction(t *testing.T) {
	out := Verify(VerifyInput{
		ModelAnswer: "The system does not distribute the load between three agents.",
		PageContent: "The system distributes the load between three agents.",
	})
	if out.Verdict != VerdictContradicted {
		t.Fatalf("expected CONTRADICTED, got %s (%+v)", out.Verdict, out)
	}
}

func TestVerifyNumericContradiction(t *testing.T) {
	out := Verify(VerifyInput{ModelAnswer: "The model has 28 million parameters.", PageContent: "The model has 27 million parameters."})
	if out.Verdict != VerdictContradicted {
		t.Fatalf("expected CONTRADICTED, got %s (%+v)", out.Verdict, out)
	}
}

func TestVerifyDecimalNumericContradiction(t *testing.T) {
	out := Verify(VerifyInput{ModelAnswer: "The score is 27.5 percent.", PageContent: "The score is 28.5 percent."})
	if out.Verdict != VerdictContradicted {
		t.Fatalf("expected decimal CONTRADICTED, got %s (%+v)", out.Verdict, out)
	}
}

func TestVerifyQuantifierContradiction(t *testing.T) {
	out := Verify(VerifyInput{ModelAnswer: "All workers require network access.", PageContent: "Some workers require network access."})
	if out.Verdict != VerdictContradicted {
		t.Fatalf("expected CONTRADICTED, got %s (%+v)", out.Verdict, out)
	}
}

func TestVerifyAntonymContradiction(t *testing.T) {
	out := Verify(VerifyInput{ModelAnswer: "The feature is disabled.", PageContent: "The feature is enabled."})
	if out.Verdict != VerdictContradicted {
		t.Fatalf("expected CONTRADICTED, got %s (%+v)", out.Verdict, out)
	}
}

func TestVerifyUnknownWhenEvidenceDoesNotAlign(t *testing.T) {
	out := Verify(VerifyInput{ModelAnswer: "Paris is the capital of France.", PageContent: "A swarm coordinates distributed agents using local interactions."})
	if out.Verdict != VerdictUnknown {
		t.Fatalf("expected UNKNOWN, got %s (%+v)", out.Verdict, out)
	}
}

func TestContradictionHasPriorityAcrossClaims(t *testing.T) {
	out := Verify(VerifyInput{
		ModelAnswer: "The model has 27 million parameters. The model is disabled.",
		PageContent: "The model has 27 million parameters. The model is enabled.",
	})
	if out.Verdict != VerdictContradicted {
		t.Fatalf("expected aggregate CONTRADICTED, got %s (%+v)", out.Verdict, out)
	}
}
