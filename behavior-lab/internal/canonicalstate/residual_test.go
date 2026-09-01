package canonicalstate

import "testing"

func conflictState() State {
	return State{
		StateHash: "state-1",
		Claims: []CanonicalClaim{{
			ID:           "claim-1",
			Status:       "UNRESOLVED",
			CandidateIDs: []string{"b", "a"},
			Evidence: []EvidenceRef{
				{Address: "ohf://evidence/2"},
				{Address: "ohf://evidence/1"},
			},
		}},
		Conflicts: []Conflict{{
			ID:           "intent-conflict",
			ClaimKey:     "request\x00intent",
			CandidateIDs: []string{"b", "a"},
			Values:       []string{"cancel", "buy"},
			Status:       "UNRESOLVED",
		}},
		Metrics: Metrics{Conflicts: 1, Unresolved: 1, Uncertainty: 0.5},
	}
}

func TestVerificationPlanIncludesSingleValueConflictCandidates(t *testing.T) {
	plan := BuildVerificationPlan(conflictState(), 1200)
	if len(plan.Tasks) != 1 {
		t.Fatalf("tasks=%+v", plan.Tasks)
	}
	task := plan.Tasks[0]
	if task.ConflictID != "intent-conflict" || task.ContextBudget != 1200 {
		t.Fatalf("task=%+v", task)
	}
	if len(task.Evidence) != 2 || task.Evidence[0] != "ohf://evidence/1" || task.Evidence[1] != "ohf://evidence/2" {
		t.Fatalf("evidence=%v", task.Evidence)
	}
}

func TestBuildRefinementEpochCreatesFirstClassConflictResidual(t *testing.T) {
	epoch, err := BuildRefinementEpoch(conflictState(), 0, RefinementPolicy{MaxEpochs: 2, ContextBudget: 900})
	if err != nil {
		t.Fatal(err)
	}
	if epoch.Decision != RefinementContinue || epoch.Reason != "RESIDUALS_AVAILABLE" || len(epoch.Residuals) != 1 {
		t.Fatalf("epoch=%+v", epoch)
	}
	residual := epoch.Residuals[0]
	if residual.Schema != ResidualSchemaR1 || residual.Kind != ResidualConflict || residual.Epoch != 0 || residual.ParentStateHash != "state-1" {
		t.Fatalf("residual=%+v", residual)
	}
	if residual.ConflictID != "intent-conflict" || residual.ClaimKey != "request\x00intent" || residual.ContextBudget != 900 {
		t.Fatalf("residual=%+v", residual)
	}
	if len(residual.CandidateIDs) != 2 || residual.CandidateIDs[0] != "a" || residual.CandidateIDs[1] != "b" {
		t.Fatalf("candidate_ids=%v", residual.CandidateIDs)
	}
	if len(residual.Values) != 2 || residual.Values[0] != "buy" || residual.Values[1] != "cancel" {
		t.Fatalf("values=%v", residual.Values)
	}
}

func TestBuildRefinementEpochStopsAtPolicyLimit(t *testing.T) {
	epoch, err := BuildRefinementEpoch(conflictState(), 2, RefinementPolicy{MaxEpochs: 2})
	if err != nil {
		t.Fatal(err)
	}
	if epoch.Decision != RefinementExhausted || epoch.Reason != "MAX_EPOCHS_REACHED" || len(epoch.Residuals) != 0 {
		t.Fatalf("epoch=%+v", epoch)
	}
}

func TestBuildRefinementEpochCapsResidualsDeterministically(t *testing.T) {
	state := State{
		StateHash: "state-many",
		Conflicts: []Conflict{
			{ID: "z", ClaimKey: "z", CandidateIDs: []string{"z1"}},
			{ID: "a", ClaimKey: "a", CandidateIDs: []string{"a1"}},
		},
		Metrics: Metrics{Conflicts: 2, Uncertainty: 1},
	}
	epoch, err := BuildRefinementEpoch(state, 0, RefinementPolicy{MaxResidualsPerEpoch: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(epoch.Residuals) != 1 || epoch.Residuals[0].ConflictID != "a" {
		t.Fatalf("residuals=%+v", epoch.Residuals)
	}
}

func TestBuildRefinementEpochExpandsUncertaintyWithoutConflict(t *testing.T) {
	state := State{StateHash: "state-uncertain", Metrics: Metrics{Uncertainty: 0.6}}
	epoch, err := BuildRefinementEpoch(state, 1, RefinementPolicy{MaxEpochs: 3, ContextBudget: 777, UncertaintyThreshold: 0.35})
	if err != nil {
		t.Fatal(err)
	}
	if epoch.Decision != RefinementContinue || len(epoch.Residuals) != 1 {
		t.Fatalf("epoch=%+v", epoch)
	}
	residual := epoch.Residuals[0]
	if residual.Kind != ResidualUncertainty || residual.Action != "EXPAND_EVIDENCE" || residual.ContextBudget != 777 || residual.Epoch != 1 {
		t.Fatalf("residual=%+v", residual)
	}
}

func TestBuildRefinementEpochCompletesSatisfiedState(t *testing.T) {
	epoch, err := BuildRefinementEpoch(State{StateHash: "done", Metrics: Metrics{Uncertainty: 0.2}}, 0, RefinementPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	if epoch.Decision != RefinementComplete || epoch.Reason != "STATE_SATISFIED" || len(epoch.Residuals) != 0 {
		t.Fatalf("epoch=%+v", epoch)
	}
}

func TestBuildRefinementEpochRejectsNegativeIndex(t *testing.T) {
	if _, err := BuildRefinementEpoch(conflictState(), -1, RefinementPolicy{}); err == nil {
		t.Fatal("expected negative epoch index to fail")
	}
}
