package experimentpolicy

import "testing"

func TestVisibleTextFidelityR6ExecutionCompliance(t *testing.T) {
	candidate:=CandidateManifest{ID:"r6",ProgramSHA256:"p"}
	semantic:=SemanticManifest{Schema:SemanticSchemaR1,ProgramSHA256:"p",Facts:[]SemanticFact{{Key:"EXECUTION_POLICY_COMPLIANCE",Value:executeDontSummarizeToStableR1}}}
	visibleFacts:=[]SemanticFact{}
	for k,v:=range executionComplianceVisibleFactsR1{visibleFacts=append(visibleFacts,SemanticFact{Key:k,Value:v})}
	visible:=VisibleTextManifest{Schema:VisibleTextSchemaR1,ProgramSHA256:"p",Facts:visibleFacts}
	report:=CheckVisibleTextFidelity(candidate,semantic,visible)
	if !report.Pass{t.Fatalf("expected R6 visible text to pass: %#v",report)}

	visible.Facts=visible.Facts[:len(visible.Facts)-1]
	report=CheckVisibleTextFidelity(candidate,semantic,visible)
	if report.Pass{t.Fatal("expected missing R6 directive to fail")}
}
