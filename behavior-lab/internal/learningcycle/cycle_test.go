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

func TestBuildPlanTargetsCellIdentityEncoding(t *testing.T){
	events:=[]learningmemory.Event{{Schema:learningmemory.EventSchema,EventID:"r2-real",EventType:learningmemory.EventObservation,EvidenceClass:learningmemory.EvidenceRealModel,BenchmarkID:"signal-chain",TrialID:"r2",QuestionID:"stable",Pass:boolp(false),LastCompletedStage:"TEMPORAL_EXECUTION",FailureCode:"CELL_IDENTITY_CONFUSION",ScoreLayer:"T_TEMPORAL"}}
	p:=BuildPlan("memory",events,"execute-to-stable-text-r1","program","payload",1)
	if p.Intent.FailureFrontier!="CELL_IDENTITY_CONFUSION"||p.Intent.MutableModule!="CELL_IDENTITY_ENCODING"{t.Fatalf("intent=%#v",p.Intent)}
	if len(p.Candidates)!=1{t.Fatalf("expected one R3 candidate, got %d",len(p.Candidates))}
	if err:=ValidatePlan(p);err!=nil{t.Fatal(err)}
	c:=p.Candidates[0]
	if c.ID!="cell-identity-redundancy-r1"||c.ParentID!="execute-to-stable-text-r1"{t.Fatalf("candidate=%#v",c)}
	if len(c.ChangedModules)!=1||c.ChangedModules[0]!="CELL_IDENTITY_ENCODING"{t.Fatalf("changed=%#v",c.ChangedModules)}
	if len(c.ExpectedSemanticChanges)!=3{t.Fatalf("expected visible A/B/C changes, got %#v",c.ExpectedSemanticChanges)}
	if c.Mutations[0].Kind!="REDUNDANCY"||c.Mutations[0].Value!="VISIBLE_CELL_ID_REDUNDANCY_R1"{t.Fatalf("mutation=%#v",c.Mutations[0])}
	for _,want:=range []string{"TEMPORAL_GRAMMAR","EXECUTION_POLICY","PROGRAM_SEMANTICS","PAYLOAD","INITIAL_STATES"}{if !contains(c.PreservedModules,want){t.Fatalf("missing preserve %s: %#v",want,c.PreservedModules)}}
	for _,want:=range []string{"RULE_MUTATION","STATE_MUTATION","EXECUTION_POLICY_MUTATION","CHECKPOINT_MUTATION","PAYLOAD_MUTATION"}{if !contains(c.ForbiddenChanges,want){t.Fatalf("missing forbidden %s: %#v",want,c.ForbiddenChanges)}}
}

func contains(xs []string,want string)bool{for _,x:=range xs{if x==want{return true}};return false}
