package experimentpolicy

import "testing"

func TestToOrigamiSpecMarksMutationsExperimental(t *testing.T){
	c:=CandidateManifest{Schema:CandidateSchemaR1,ID:"execute",Mutations:[]Mutation{{Kind:"PROMPT",Target:"EXECUTION_POLICY",Value:"EXECUTE_VISIBLE_RULES_TO_STABLE_R1"}}}
	s,err:=ToOrigamiSpec(c,"parent-sha");if err!=nil{t.Fatal(err)}
	if s.Schema!=OrigamiCandidateSpecSchemaR0||s.ParentSHA256!="parent-sha"{t.Fatalf("spec=%#v",s)}
	if len(s.Mutations)!=1||!s.Mutations[0].Experimental{t.Fatalf("mutation=%#v",s.Mutations)}
}
