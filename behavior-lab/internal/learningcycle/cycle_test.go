package learningcycle

import (
	"testing"

	"tlaloc.local/behaviorlab/internal/learningmemory"
)

func boolp(v bool)*bool{return &v}

func TestBuildPlanTargetsExecutionPolicy(t *testing.T){
	events:=[]learningmemory.Event{{Schema:learningmemory.EventSchema,EventID:"e1",EventType:learningmemory.EventObservation,EvidenceClass:learningmemory.EvidenceRealModel,BenchmarkID:"b",TrialID:"t",QuestionID:"q",Pass:boolp(false),LastCompletedStage:"T2_RULE_MICROGRAMMAR",FailureCode:"TEMPORAL_EXECUTION_INCOMPLETE",ScoreLayer:"T_TEMPORAL"}}
	p:=BuildPlan("memory",events,"baseline","program","payload",3)
	if p.Intent.MutableModule!="EXECUTION_POLICY"{t.Fatalf("mutable=%q",p.Intent.MutableModule)}
	if len(p.Candidates)!=3{t.Fatalf("candidates=%d",len(p.Candidates))}
	if err:=ValidatePlan(p);err!=nil{t.Fatal(err)}
	for _,c:=range p.Candidates{if len(c.Mutations)!=1{t.Fatalf("candidate %s mutations=%d",c.ID,len(c.Mutations))}}
}
