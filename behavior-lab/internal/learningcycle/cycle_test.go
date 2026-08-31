package learningcycle

import (
	"testing"

	"tlaloc.local/behaviorlab/internal/experimentpolicy"
	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/promptgenome"
)

func boolp(v bool)*bool{return &v}

func TestBuildPlanTargetsExecutionPolicyCompliance(t *testing.T){
	events:=[]learningmemory.Event{{Schema:learningmemory.EventSchema,EventID:"e1",EventType:learningmemory.EventObservation,EvidenceClass:learningmemory.EvidenceRealModel,BenchmarkID:"b",TrialID:"t",QuestionID:"q",Pass:boolp(false),LastCompletedStage:"T2_RULE_MICROGRAMMAR",FailureCode:"TEMPORAL_EXECUTION_INCOMPLETE",ScoreLayer:"T_TEMPORAL"}}
	g:=promptgenome.Genome{Schema:promptgenome.GenomeSchemaR1,ID:"g",Version:1,Modules:[]promptgenome.Module{{ID:"TEMPORAL_GRAMMAR",Version:1,Text:"rules",Priority:10,Protected:true,Maturity:"PROVISIONAL_WIN"},{ID:"EXECUTION_POLICY",Version:1,Text:"execute",Priority:9}}}
	p:=BuildPlanWithGenome("memory",events,g,"baseline","program","payload",3)
	if p.Intent.MutableModule!="EXECUTION_POLICY_COMPLIANCE"{t.Fatalf("mutable=%q",p.Intent.MutableModule)}
	if len(p.Candidates)!=1{t.Fatalf("only negotiated materializable candidate expected, got=%d",len(p.Candidates))}
	if err:=ValidatePlan(p);err!=nil{t.Fatal(err)}
	if p.Candidates[0].Mutations[0].Value!="EXECUTE_DONT_SUMMARIZE_TO_STABLE_R1"{t.Fatalf("mutation=%#v",p.Candidates[0].Mutations[0])}
	found:=false;for _,x:=range p.Intent.Preserve{if x=="TEMPORAL_GRAMMAR"{found=true}}
	if !found{t.Fatalf("protected prior win missing from preserve: %#v",p.Intent.Preserve)}
}

func TestBuildPlanTargetsSynchronousExecutionFidelity(t *testing.T){
	baseline:="execution-policy-compliance-cross-model-r1"
	events:=[]learningmemory.Event{
		{Schema:learningmemory.EventSchema,EventID:"qwen-r6-fail",EventType:learningmemory.EventObservation,EvidenceClass:learningmemory.EvidenceRealModel,ModelID:"qwen-unspecified",SpecimenID:baseline,Pass:boolp(false),LastCompletedStage:"TEMPORAL_EXECUTION",FailureCode:"RULE_FIRING_PRECONDITION_VIOLATION",ScoreLayer:"T_TEMPORAL"},
		{Schema:learningmemory.EventSchema,EventID:"deepseek-r6-fail",EventType:learningmemory.EventObservation,EvidenceClass:learningmemory.EvidenceRealModel,ModelID:"deepseek-unspecified",SpecimenID:baseline,Pass:boolp(false),LastCompletedStage:"TEMPORAL_EXECUTION",FailureCode:"EXECUTION_SEMANTICS_CONTRADICTION",ScoreLayer:"T_TEMPORAL"},
	}
	p:=BuildPlan("memory",events,baseline,"program","",1)
	if p.Intent.MutableModule!="SYNCHRONOUS_EXECUTION_FIDELITY"{t.Fatalf("intent=%#v",p.Intent)}
	if len(p.Candidates)!=1{t.Fatalf("candidates=%#v",p.Candidates)}
	if err:=ValidatePlan(p);err!=nil{t.Fatal(err)}
	c:=p.Candidates[0]
	if c.ID!="synchronous-execution-fidelity-cross-model-r1"{t.Fatalf("candidate=%#v",c)}
	if c.Mutations[0].Target!="SYNCHRONOUS_EXECUTION_FIDELITY"||c.Mutations[0].Value!="FREEZE_SELECT_APPLY_TOGETHER_R1"{t.Fatalf("mutation=%#v",c.Mutations[0])}
	for _,want:=range []string{"RULE_ROLE_BINDING","EXECUTION_POLICY_COMPLIANCE","PROGRAM_SEMANTICS","PAYLOAD"}{if !contains(c.PreservedModules,want){t.Fatalf("missing preserve %s: %#v",want,c.PreservedModules)}}
	if len(c.CompatibilityPanel)!=2{t.Fatalf("panel=%#v",c.CompatibilityPanel)}
	for _,r:=range c.CompatibilityPanel{if r.Mode!=experimentpolicy.ModelCompatibilityImproveToPass||r.BaselinePass{t.Fatalf("R7 panel requirement=%#v",r)}}
}

func TestBuildPlanCarriesCrossModelPanel(t *testing.T){
	baseline:="rule-role-binding-unseen-rules-r1"
	events:=[]learningmemory.Event{
		{Schema:learningmemory.EventSchema,EventID:"deepseek-r5-pass",EventType:learningmemory.EventObservation,EvidenceClass:learningmemory.EvidenceRealModel,ModelID:"deepseek-unspecified",SpecimenID:baseline,Pass:boolp(true),LastCompletedStage:"TEMPORAL_EXECUTION",ScoreLayer:"T_TEMPORAL"},
		{Schema:learningmemory.EventSchema,EventID:"qwen-r5-fail",EventType:learningmemory.EventObservation,EvidenceClass:learningmemory.EvidenceRealModel,ModelID:"qwen-unspecified",SpecimenID:baseline+"-fixed-title",Pass:boolp(false),LastCompletedStage:"TEMPORAL_EXECUTION",FailureCode:"TEMPORAL_EXECUTION_INCOMPLETE",ScoreLayer:"T_TEMPORAL"},
	}
	p:=BuildPlan("memory",events,baseline,"program","",1)
	if p.Intent.MutableModule!="EXECUTION_POLICY_COMPLIANCE"{t.Fatalf("intent=%#v",p.Intent)}
	if len(p.Intent.CompatibilityPanel)!=2{t.Fatalf("panel=%#v",p.Intent.CompatibilityPanel)}
	if len(p.Candidates)!=1||len(p.Candidates[0].CompatibilityPanel)!=2{t.Fatalf("candidate panel=%#v",p.Candidates)}
	byModel:=map[string]experimentpolicy.ModelCompatibilityRequirement{};for _,r:=range p.Intent.CompatibilityPanel{byModel[r.ModelID]=r}
	if byModel["deepseek-unspecified"].Mode!=experimentpolicy.ModelCompatibilityPreservePass||!byModel["deepseek-unspecified"].BaselinePass{t.Fatalf("deepseek requirement=%#v",byModel["deepseek-unspecified"])}
	if byModel["qwen-unspecified"].Mode!=experimentpolicy.ModelCompatibilityImproveToPass||byModel["qwen-unspecified"].BaselinePass{t.Fatalf("qwen requirement=%#v",byModel["qwen-unspecified"])}
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
