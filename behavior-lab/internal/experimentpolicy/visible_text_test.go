package experimentpolicy

import "testing"

func TestVisibleTextFidelityR3(t *testing.T){
	candidate:=CandidateManifest{Schema:CandidateSchemaR1,ID:"cell-identity-redundancy-r1",ProgramSHA256:"p",ChangedModules:[]string{"CELL_IDENTITY_ENCODING"},ExpectedSemanticChanges:[]SemanticFact{{Key:"VISIBLE_CELL_ID_A",Value:"A[01]"},{Key:"VISIBLE_CELL_ID_B",Value:"B[02]"},{Key:"VISIBLE_CELL_ID_C",Value:"C[03]"}}}
	semantic:=SemanticManifest{Schema:SemanticSchemaR1,ProgramSHA256:"p",Facts:[]SemanticFact{
		{Key:"CELL.A.INITIAL",Value:"ACTIVE"},{Key:"CELL.B.INITIAL",Value:"IDLE"},{Key:"CELL.C.INITIAL",Value:"IDLE"},
		{Key:"VISIBLE_CELL_ID_A",Value:"A[01]"},{Key:"VISIBLE_CELL_ID_B",Value:"B[02]"},{Key:"VISIBLE_CELL_ID_C",Value:"C[03]"},
		{Key:"TEMPORAL_GRAMMAR",Value:"VISIBLE_RULE_MICROGRAMMAR_R1"},{Key:"EXECUTION_POLICY",Value:"EXECUTE_VISIBLE_RULES_TO_STABLE_R1"},
		{Key:"RULE.r1.TARGET",Value:"B"},{Key:"RULE.r1.FROM",Value:"IDLE"},{Key:"RULE.r1.TO",Value:"ACTIVE"},{Key:"RULE.r1.REQUIRES",Value:"A=ACTIVE"},
	}}
	visible:=VisibleTextManifest{Schema:VisibleTextSchemaR1,ProgramSHA256:"p",Facts:[]SemanticFact{
		{Key:"CELL.A.LABEL",Value:"CELL A[01]"},{Key:"CELL.A.INITIAL_TEXT",Value:"ACTIVE"},
		{Key:"CELL.B.LABEL",Value:"CELL B[02]"},{Key:"CELL.B.INITIAL_TEXT",Value:"IDLE"},
		{Key:"CELL.C.LABEL",Value:"CELL C[03]"},{Key:"CELL.C.INITIAL_TEXT",Value:"IDLE"},
		{Key:"TEMPORAL_GRAMMAR.SYNC_TEXT",Value:synchronousRuleTextR1},
		{Key:"RULE.r1.TEXT",Value:"IF A[01]=ACTIVE => B[02]:IDLE>ACTIVE"},
		{Key:"EXECUTION_POLICY.TEXT",Value:executeVisibleRulesToStableTextR1},
	}}
	r:=CheckVisibleTextFidelity(candidate,semantic,visible);if !r.Pass{t.Fatalf("fidelity=%#v",r)}
}

func TestVisibleTextFidelityRejectsCellConfusion(t *testing.T){
	candidate:=CandidateManifest{Schema:CandidateSchemaR1,ID:"c",ProgramSHA256:"p"}
	semantic:=SemanticManifest{Schema:SemanticSchemaR1,ProgramSHA256:"p",Facts:[]SemanticFact{{Key:"CELL.A.INITIAL",Value:"ACTIVE"},{Key:"VISIBLE_CELL_ID_A",Value:"A[01]"}}}
	visible:=VisibleTextManifest{Schema:VisibleTextSchemaR1,ProgramSHA256:"p",Facts:[]SemanticFact{{Key:"CELL.A.LABEL",Value:"CELL B[02]"},{Key:"CELL.A.INITIAL_TEXT",Value:"ACTIVE"}}}
	r:=CheckVisibleTextFidelity(candidate,semantic,visible);if r.Pass||r.FailureCode!="VISIBLE_TEXT_FIDELITY_FAILED"{t.Fatalf("expected fidelity failure, got %#v",r)}
}

func TestRegressionPrecheckAllowsOnlyDeclaredIdentityChange(t *testing.T){
	candidate:=CandidateManifest{ID:"c",ChangedModules:[]string{"CELL_IDENTITY_ENCODING"},ExpectedSemanticChanges:[]SemanticFact{{Key:"VISIBLE_CELL_ID_A",Value:"A[01]"}}}
	expected:=SemanticManifest{ProgramSHA256:"p",Facts:[]SemanticFact{{Key:"RULE.r1.TARGET",Value:"B"},{Key:"VISIBLE_CELL_ID_A",Value:"A"}}}
	actual:=SemanticManifest{ProgramSHA256:"p",Facts:[]SemanticFact{{Key:"RULE.r1.TARGET",Value:"B"},{Key:"VISIBLE_CELL_ID_A",Value:"A[01]"}}}
	build:=BuildManifest{ProgramSHA256:"p",VisibleSemantics:actual}
	r:=CheckRegressionPreconditions(candidate,expected,build);if !r.Pass{t.Fatalf("regression=%#v",r)}
	build.VisibleSemantics.Facts[0].Value="C"
	r=CheckRegressionPreconditions(candidate,expected,build);if r.Pass{t.Fatal("canonical rule regression must fail")}
}
