package learningcycle

import (
	"testing"

	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/promptgenome"
)

func boolp(v bool)*bool{return &v}

func TestBuildPlanTargetsExecutionPolicy(t *testing.T){
	events:=[]learningmemory.Event{{Schema:learningmemory.EventSchema,EventID:"e1",EventType:learningmemory.EventObservation,EvidenceClass:learningmemory.EvidenceRealModel,BenchmarkID:"b",TrialID:"t",QuestionID:"q",Pass:boolp(false),LastCompletedStage:"T2_RULE_MICROGRAMMAR",FailureCode:"TEMPORAL_EXECUTION_INCOMPLETE",ScoreLayer:"T_TEMPORAL"}}
	g:=promptgenome.Genome{Schema:promptgenome.GenomeSchemaR1,ID:"g",Version:1,Modules:[]promptgenome.Module{{ID:"TEMPORAL_GRAMMAR",Version:1,Text:"rules",Priority:10,Protected:true,Maturity:"PROVISIONAL_WIN"},{ID:"EXECUTION_POLICY",Version:1,Text:"execute",Priority:9}}}
	p:=BuildPlanWithGenome("memory",events,g,"baseline","program","payload",3)
	if p.Intent.MutableModule!="EXECUTION_POLICY"{t.Fatalf("mutable=%q",p.Intent.MutableModule)}
	if len(p.Candidates)!=1{t.Fatalf("only negotiated materializable candidate expected, got=%d",len(p.Candidates))}
	if err:=ValidatePlan(p);err!=nil{t.Fatal(err)}
	if p.Candidates[0].Mutations[0].Value!="EXECUTE_VISIBLE_RULES_TO_STABLE_R1"{t.Fatalf("mutation=%#v",p.Candidates[0].Mutations[0])}
	found:=false;for _,x:=range p.Intent.Preserve{if x=="TEMPORAL_GRAMMAR"{found=true}}
	if !found{t.Fatalf("protected prior win missing from preserve: %#v",p.Intent.Preserve)}
}
