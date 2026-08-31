package experimentpolicy

import "testing"

func TestCrossModelCompatibilityPreservesWinsAndRequiresImprovement(t *testing.T) {
	candidate := CandidateManifest{ID:"r6",CompatibilityPanel:[]ModelCompatibilityRequirement{
		{ModelID:"deepseek-unspecified",Mode:ModelCompatibilityPreservePass,BaselinePass:true,RequiredCandidatePass:true},
		{ModelID:"qwen-unspecified",Mode:ModelCompatibilityImproveToPass,BaselinePass:false,RequiredCandidatePass:true},
	}}
	report := CheckCrossModelCompatibility(candidate, []ModelTrialOutcome{{ModelID:"deepseek-unspecified",Pass:true},{ModelID:"qwen-unspecified",Pass:true}})
	if !report.Pass { t.Fatalf("expected compatible panel, got %#v", report) }
}

func TestCrossModelCompatibilityRejectsDeepSeekRegression(t *testing.T) {
	candidate := CandidateManifest{ID:"r6",CompatibilityPanel:[]ModelCompatibilityRequirement{
		{ModelID:"deepseek-unspecified",Mode:ModelCompatibilityPreservePass,BaselinePass:true,RequiredCandidatePass:true},
		{ModelID:"qwen-unspecified",Mode:ModelCompatibilityImproveToPass,BaselinePass:false,RequiredCandidatePass:true},
	}}
	report := CheckCrossModelCompatibility(candidate, []ModelTrialOutcome{{ModelID:"deepseek-unspecified",Pass:false},{ModelID:"qwen-unspecified",Pass:true}})
	if report.Pass { t.Fatal("expected DeepSeek regression to reject candidate") }
	if report.FailureCode != "CROSS_MODEL_COMPATIBILITY_FAILED" { t.Fatalf("unexpected failure: %s", report.FailureCode) }
}

func TestCrossModelCompatibilityRequiresEveryPanelMember(t *testing.T) {
	candidate := CandidateManifest{ID:"r6",CompatibilityPanel:[]ModelCompatibilityRequirement{
		{ModelID:"deepseek-unspecified",Mode:ModelCompatibilityPreservePass,BaselinePass:true,RequiredCandidatePass:true},
		{ModelID:"qwen-unspecified",Mode:ModelCompatibilityImproveToPass,BaselinePass:false,RequiredCandidatePass:true},
	}}
	report := CheckCrossModelCompatibility(candidate, []ModelTrialOutcome{{ModelID:"deepseek-unspecified",Pass:true}})
	if report.Pass { t.Fatal("expected missing Qwen outcome to reject candidate") }
}
