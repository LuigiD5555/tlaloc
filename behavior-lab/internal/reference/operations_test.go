package reference

import (
	"math"
	"testing"

	"tlaloc.local/behaviorlab/internal/spec"
)

func TestInterferenceCancellation(t *testing.T) {
	s := Interfere(Branch{Label: "C", Real: .5}, Branch{Label: "C", Real: -.5})
	if len(s.Branches) != 0 {
		t.Fatalf("expected cancellation, got %#v", s.Branches)
	}
	if s.Unknown {
		t.Fatal("cancellation must not become UNKNOWN")
	}
}

func TestTransformPreservesAlternatives(t *testing.T) {
	s := Superpose(Branch{Label: "A", Real: 1}, Branch{Label: "B", Real: 1})
	out, err := Transform(s, LinearMap{
		"A": {{Label: "D", Real: 1}},
		"B": {{Label: "E", Real: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Kind != spec.Superposed || len(out.Branches) != 2 {
		t.Fatalf("unexpected transform result: %#v", out)
	}
	if math.Abs(out.Probability("D")-.5) > 1e-9 {
		t.Fatalf("bad probability: %v", out.Probability("D"))
	}
}

func TestCoupledIsJointState(t *testing.T) {
	s := Couple([]string{"A", "B"}, Branch{Label: "00", Real: 1}, Branch{Label: "11", Real: 1})
	if s.Kind != spec.Coupled || len(s.Members) != 2 {
		t.Fatalf("bad coupled state: %#v", s)
	}
}
