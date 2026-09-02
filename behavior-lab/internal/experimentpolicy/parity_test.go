package experimentpolicy

import "testing"

func TestCheckParityRejectsUnauthorizedRuleDrift(t *testing.T) {
	c := CandidateManifest{Schema: CandidateSchemaR1, ID: "c", ProgramSHA256: "p", Mutations: []Mutation{{Kind: "PROMPT", Target: "EXECUTION_POLICY", Value: "EXECUTE_TO_STABLE"}}, ExpectedSemanticChanges: []SemanticFact{{Key: "EXECUTION_POLICY", Value: "EXECUTE_TO_STABLE"}}}
	exp := SemanticManifest{Schema: SemanticSchemaR1, ProgramSHA256: "p", PayloadSHA256: "x", Facts: []SemanticFact{{Key: "RULE_R1", Value: "IF A=ACTIVE => B:IDLE>ACTIVE"}, {Key: "EXECUTION_POLICY", Value: "NONE"}}}
	act := SemanticManifest{Schema: SemanticSchemaR1, ProgramSHA256: "p", PayloadSHA256: "x", Facts: []SemanticFact{{Key: "RULE_R1", Value: "IF B=ACTIVE => B:IDLE>ACTIVE"}, {Key: "EXECUTION_POLICY", Value: "EXECUTE_TO_STABLE"}}}
	r := CheckParity(c, exp, act)
	if r.Pass {
		t.Fatal("expected drift rejection")
	}
	if r.FailureCode != "UNAUTHORIZED_SEMANTIC_DRIFT" {
		t.Fatalf("failure=%q", r.FailureCode)
	}
}

func TestCheckParityAllowsDeclaredMutationOnly(t *testing.T) {
	c := CandidateManifest{Schema: CandidateSchemaR1, ID: "c", ProgramSHA256: "p", Mutations: []Mutation{{Kind: "PROMPT", Target: "EXECUTION_POLICY", Value: "EXECUTE_TO_STABLE"}}, ExpectedSemanticChanges: []SemanticFact{{Key: "EXECUTION_POLICY", Value: "EXECUTE_TO_STABLE"}}}
	exp := SemanticManifest{Schema: SemanticSchemaR1, ProgramSHA256: "p", PayloadSHA256: "x", Facts: []SemanticFact{{Key: "RULE_R1", Value: "R1"}, {Key: "EXECUTION_POLICY", Value: "NONE"}}}
	act := SemanticManifest{Schema: SemanticSchemaR1, ProgramSHA256: "p", PayloadSHA256: "x", Facts: []SemanticFact{{Key: "RULE_R1", Value: "R1"}, {Key: "EXECUTION_POLICY", Value: "EXECUTE_TO_STABLE"}}}
	r := CheckParity(c, exp, act)
	if !r.Pass {
		t.Fatalf("unexpected reject: %#v", r)
	}
}

func TestCheckParityRejectsWrongValueOnAllowedKey(t *testing.T) {
	c := CandidateManifest{Schema: CandidateSchemaR1, ID: "c", ProgramSHA256: "p", ExpectedSemanticChanges: []SemanticFact{{Key: "EXECUTION_POLICY", Value: "EXECUTE_TO_STABLE"}}}
	exp := SemanticManifest{Schema: SemanticSchemaR1, ProgramSHA256: "p", Facts: []SemanticFact{{Key: "EXECUTION_POLICY", Value: "NONE"}}}
	act := SemanticManifest{Schema: SemanticSchemaR1, ProgramSHA256: "p", Facts: []SemanticFact{{Key: "EXECUTION_POLICY", Value: "SOMETHING_ELSE"}}}
	if r := CheckParity(c, exp, act); r.Pass {
		t.Fatalf("wrong declared value must fail: %#v", r)
	}
}
