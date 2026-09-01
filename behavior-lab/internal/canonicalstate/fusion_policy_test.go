package canonicalstate

import "testing"

func evidencedCandidate(id, subject, predicate, object string, polarity bool) Candidate {
	return Candidate{
		Schema: CandidateSchema,
		ID:     id,
		Role:   "test",
		Claim: Claim{
			Subject:   subject,
			Predicate: predicate,
			Object:    object,
			Polarity:  polarity,
		},
		Evidence:   []EvidenceRef{{Address: "ohf://test/" + id}},
		Confidence: 0.8,
	}
}

func TestClaimPolicyRegistryDefaultsPreserveR0ManySemantics(t *testing.T) {
	r := Reducer{Verifier: allGood{}}
	state, err := r.Reduce([]Candidate{
		evidencedCandidate("a", "doc", "entity", "alice", true),
		evidencedCandidate("b", "doc", "entity", "bob", true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Claims) != 2 {
		t.Fatalf("default MANY policy should preserve independent objects, claims=%d state=%+v", len(state.Claims), state)
	}
	if len(state.Conflicts) != 0 || state.Metrics.Conflicts != 0 {
		t.Fatalf("default R0 semantics should not conflict distinct positive objects: %+v", state.Conflicts)
	}
	if state.PolicyVersion != "" {
		t.Fatalf("unexpected policy version %q without explicit registry", state.PolicyVersion)
	}
}

func TestCardinalityOneConflictsDistinctPositiveValues(t *testing.T) {
	policies, err := NewClaimPolicyRegistry("intent-r1", map[string]ClaimPolicy{
		"intent": {Cardinality: CardinalityOne},
	})
	if err != nil {
		t.Fatal(err)
	}
	state, err := (Reducer{Verifier: allGood{}, Policies: policies}).Reduce([]Candidate{
		evidencedCandidate("buy", "request", "intent", "buy", true),
		evidencedCandidate("cancel", "request", "intent", "cancel", true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if state.PolicyVersion != "intent-r1" {
		t.Fatalf("policy_version=%q", state.PolicyVersion)
	}
	if len(state.Claims) != 1 || state.Claims[0].Status != "UNRESOLVED" {
		t.Fatalf("exclusive values should produce one unresolved canonical claim: %+v", state.Claims)
	}
	if len(state.Conflicts) != 1 {
		t.Fatalf("conflicts=%+v", state.Conflicts)
	}
	conflict := state.Conflicts[0]
	if len(conflict.CandidateIDs) != 2 || len(conflict.Values) != 2 {
		t.Fatalf("conflict should expose candidates and competing values: %+v", conflict)
	}
	if state.Metrics.Conflicts != 1 || state.Metrics.Unresolved != 1 {
		t.Fatalf("metrics=%+v", state.Metrics)
	}
}

func TestCardinalityOneMergesRepeatedSameValue(t *testing.T) {
	policies, err := NewClaimPolicyRegistry("intent-r1", map[string]ClaimPolicy{
		"intent": {Cardinality: CardinalityOne, Fusion: FusionSingleValue},
	})
	if err != nil {
		t.Fatal(err)
	}
	state, err := (Reducer{Verifier: allGood{}, Policies: policies}).Reduce([]Candidate{
		evidencedCandidate("a", "request", "intent", "buy", true),
		evidencedCandidate("b", "request", "intent", "buy", true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Claims) != 1 || state.Claims[0].Status != "VERIFIED" {
		t.Fatalf("same exclusive value should merge: %+v", state.Claims)
	}
	if len(state.Claims[0].CandidateIDs) != 2 {
		t.Fatalf("candidate ids=%v", state.Claims[0].CandidateIDs)
	}
	if state.Metrics.Accepted != 2 || state.Metrics.Conflicts != 0 {
		t.Fatalf("metrics=%+v", state.Metrics)
	}
}

func TestPolicyFusionIsCandidateOrderIndependent(t *testing.T) {
	policies, err := NewClaimPolicyRegistry("intent-r1", map[string]ClaimPolicy{
		"intent": {Cardinality: CardinalityOne},
	})
	if err != nil {
		t.Fatal(err)
	}
	a := evidencedCandidate("a", "request", "intent", "buy", true)
	b := evidencedCandidate("b", "request", "intent", "cancel", true)
	r := Reducer{Verifier: allGood{}, Policies: policies}
	first, err := r.Reduce([]Candidate{a, b})
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.Reduce([]Candidate{b, a})
	if err != nil {
		t.Fatal(err)
	}
	if first.StateHash != second.StateHash {
		t.Fatalf("policy reduction must be order independent: %s != %s", first.StateHash, second.StateHash)
	}
}

func TestReducerAllowsFusionStrategyReplacement(t *testing.T) {
	policies, err := NewClaimPolicyRegistry("custom-r1", map[string]ClaimPolicy{
		"intent": {Cardinality: CardinalityOne, Fusion: FusionSingleValue},
	})
	if err != nil {
		t.Fatal(err)
	}
	called := false
	custom := fusionStrategyFunc(func(groupKey string, rows []verifiedCandidate) (FusionOutcome, error) {
		called = true
		return mergedFusionOutcome(rows), nil
	})
	state, err := (Reducer{
		Verifier: allGood{},
		Policies: policies,
		Fusion: map[FusionMode]FusionStrategy{
			FusionSingleValue: custom,
		},
	}).Reduce([]Candidate{
		evidencedCandidate("a", "request", "intent", "buy", true),
		evidencedCandidate("b", "request", "intent", "cancel", true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("custom fusion strategy was not invoked")
	}
	if len(state.Conflicts) != 0 || len(state.Claims) != 1 || state.Claims[0].Status != "VERIFIED" {
		t.Fatalf("custom strategy result not honored: %+v", state)
	}
}

func TestClaimPolicyRegistryRejectsUnsupportedPolicy(t *testing.T) {
	if _, err := NewClaimPolicyRegistry("bad", map[string]ClaimPolicy{
		"intent": {Cardinality: ClaimCardinality("SOMETIMES")},
	}); err == nil {
		t.Fatal("expected unsupported cardinality to fail")
	}
	if _, err := NewClaimPolicyRegistry("bad", map[string]ClaimPolicy{
		"intent": {Cardinality: CardinalityOne, Fusion: FusionMode("UNKNOWN")},
	}); err == nil {
		t.Fatal("expected unsupported fusion mode to fail")
	}
}

func TestClaimPolicyRegistryNormalizesPredicateKeys(t *testing.T) {
	policies, err := NewClaimPolicyRegistry("normalized-r1", map[string]ClaimPolicy{
		"  Document   Type ": {Cardinality: CardinalityOne},
	})
	if err != nil {
		t.Fatal(err)
	}
	policy := policies.PolicyFor("document type")
	if policy.Cardinality != CardinalityOne || policy.Fusion != FusionSingleValue {
		t.Fatalf("normalized policy=%+v", policy)
	}
}
