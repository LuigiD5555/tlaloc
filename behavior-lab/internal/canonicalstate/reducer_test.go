package canonicalstate

import "testing"

type allGood struct{}

func (allGood) Verify(a, c string) (bool, error) { return a != "", nil }

func TestReducerDeterministicMergeAndConflict(t *testing.T) {
	base := Candidate{Schema: CandidateSchema, ID: "a", Role: "semantic", Claim: Claim{Subject: "BLT", Predicate: "uses", Object: "dynamic patches", Polarity: true}, Evidence: []EvidenceRef{{Address: "ohf://x/p/1"}}, Confidence: .9}
	b := base
	b.ID = "b"
	b.Confidence = .8
	r := Reducer{Verifier: allGood{}}
	s1, err := r.Reduce([]Candidate{b, base})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := r.Reduce([]Candidate{base, b})
	if err != nil {
		t.Fatal(err)
	}
	if s1.StateHash != s2.StateHash {
		t.Fatalf("state must be candidate-order independent: %s != %s", s1.StateHash, s2.StateHash)
	}
	if len(s1.Claims) != 1 || s1.Claims[0].Status != "VERIFIED" {
		t.Fatalf("unexpected merge: %+v", s1)
	}
	n := base
	n.ID = "n"
	n.Claim.Polarity = false
	s3, err := r.Reduce([]Candidate{base, n})
	if err != nil {
		t.Fatal(err)
	}
	if len(s3.Conflicts) != 1 || s3.Claims[0].Status != "UNRESOLVED" {
		t.Fatalf("conflict missing: %+v", s3)
	}
}

func TestReducerRejectsEvidenceFreeCandidates(t *testing.T) {
	c := Candidate{ID: "x", Claim: Claim{Subject: "a", Predicate: "b", Object: "c", Polarity: true}, Confidence: .5}
	s, err := (Reducer{Verifier: allGood{}}).Reduce([]Candidate{c})
	if err != nil {
		t.Fatal(err)
	}
	if s.Metrics.Rejected != 1 || len(s.Claims) != 0 {
		t.Fatalf("unexpected state %+v", s)
	}
}
