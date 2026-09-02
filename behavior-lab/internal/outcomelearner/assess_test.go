package outcomelearner

import "testing"

func TestInvalidSpecimenDoesNotPenalizeModel(t *testing.T) {
	a, k := Assess(Request{HypothesisID: "h", ChangedModules: []string{"EXECUTION_POLICY"}, Before: TrialSnapshot{OverallScore: .8, ValidSpecimen: true}, After: TrialSnapshot{OverallScore: .1, ValidSpecimen: false}})
	if a.Classification != OutcomeInvalidExperiment {
		t.Fatalf("class=%q", a.Classification)
	}
	if a.ModelPenaltyAllowed {
		t.Fatal("model penalty must be disabled")
	}
	if k.Action != "RETAIN_HYPOTHESIS_REJECT_SPECIMEN" {
		t.Fatalf("action=%q", k.Action)
	}
}

func TestTargetResolutionBecomesProvisionalWin(t *testing.T) {
	a, k := Assess(Request{HypothesisID: "h", TargetAssertion: "FINAL_STATE", ChangedModules: []string{"EXECUTION_POLICY"}, Before: TrialSnapshot{OverallScore: .7, ValidSpecimen: true, FailureFrontier: "TEMPORAL_EXECUTION_INCOMPLETE", Assertions: []Assertion{{ID: "RULES", Pass: true}, {ID: "FINAL_STATE", Pass: false}}}, After: TrialSnapshot{OverallScore: .9, ValidSpecimen: true, FailureFrontier: "CHECKPOINT_NOT_FOUND", Assertions: []Assertion{{ID: "RULES", Pass: true}, {ID: "FINAL_STATE", Pass: true}}}})
	if !a.TargetResolved || a.Classification != OutcomeSuccessfulCausalStep {
		t.Fatalf("assessment=%#v", a)
	}
	if k.Maturity != "PROVISIONAL_WIN" {
		t.Fatalf("knowledge=%#v", k)
	}
}
