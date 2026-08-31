package learningcycle

import (
	"fmt"
	"strings"

	"tlaloc.local/behaviorlab/internal/adaptivesearch"
	"tlaloc.local/behaviorlab/internal/experimentpolicy"
	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/learningpolicy"
	"tlaloc.local/behaviorlab/internal/promptgenome"
)

func BuildStatus(root string, events []learningmemory.Event) Status { return BuildStatusWithGenome(root,events,promptgenome.Genome{}) }
func BuildStatusWithGenome(root string, events []learningmemory.Event, genome promptgenome.Genome) Status { policy:=learningpolicy.Derive(events);if genome.Schema==promptgenome.GenomeSchemaR1{policy=learningpolicy.ApplyGenomeProtection(policy,genome)};adaptive:=adaptivesearch.BuildPlan(root,events);return Status{Schema:StatusSchemaR1,FailureFrontier:policy.FailureFrontier,NextTarget:policy.Target,Policy:policy,AdaptiveSearch:adaptive,Promotion:"NOT_EVALUATED_BY_LEARNING_CYCLE"} }
func BuildPlan(root string,events []learningmemory.Event,baseline,programSHA,payloadSHA string,budget int)Plan{return BuildPlanWithGenome(root,events,promptgenome.Genome{},baseline,programSHA,payloadSHA,budget)}
func BuildPlanWithGenome(root string,events []learningmemory.Event,genome promptgenome.Genome,baseline,programSHA,payloadSHA string,budget int)Plan{st:=BuildStatusWithGenome(root,events,genome);if budget<=0{budget=3};preserve,avoid,require:=rulesByKind(st.Policy);intent:=experimentpolicy.ExperimentIntent{Schema:experimentpolicy.IntentSchemaR1,ID:"intent-"+normalize(st.NextTarget),Objective:"improve "+st.NextTarget+" without regressing learned invariants",BaselineCandidateID:baseline,FailureFrontier:st.FailureFrontier,MutableModule:st.NextTarget,Preserve:preserve,Avoid:avoid,Require:require,CandidateBudget:budget,TrialsPerModel:3};candidates:=synthesize(intent,st.Policy.ParentEvidenceIDs,programSHA,payloadSHA);if len(candidates)>budget{candidates=candidates[:budget]};return Plan{Schema:PlanSchemaR1,Status:st,Intent:intent,Candidates:candidates}}

func synthesize(intent experimentpolicy.ExperimentIntent,parents []string,programSHA,payloadSHA string)[]experimentpolicy.CandidateManifest{
	target:=strings.ToUpper(strings.TrimSpace(intent.MutableModule))
	if target=="CELL_IDENTITY_ENCODING"{
		preserved:=appendUnique(append([]string(nil),intent.Preserve...),"TEMPORAL_GRAMMAR","EXECUTION_POLICY","PROGRAM_SEMANTICS","PAYLOAD","INITIAL_STATES")
		forbidden:=appendUnique(append([]string(nil),intent.Avoid...),"RULE_MUTATION","STATE_MUTATION","EXECUTION_POLICY_MUTATION","CHECKPOINT_MUTATION","PAYLOAD_MUTATION","GENERATIVE_SEMANTIC_REWRITE","MULTI_MODULE_MUTATION","TEMPORAL_RULE_MUTATION")
		return []experimentpolicy.CandidateManifest{{Schema:experimentpolicy.CandidateSchemaR1,ID:"cell-identity-redundancy-r1",ParentID:intent.BaselineCandidateID,ProgramSHA256:programSHA,PayloadSHA256:payloadSHA,Mutations:[]experimentpolicy.Mutation{{Kind:"REDUNDANCY",Target:"CELL_IDENTITY_ENCODING",Value:"VISIBLE_CELL_ID_REDUNDANCY_R1"}},ChangedModules:[]string{"CELL_IDENTITY_ENCODING"},PreservedModules:preserved,ForbiddenChanges:forbidden,ExpectedSemanticChanges:[]experimentpolicy.SemanticFact{{Key:"VISIBLE_CELL_ID_A",Value:"A[01]"},{Key:"VISIBLE_CELL_ID_B",Value:"B[02]"},{Key:"VISIBLE_CELL_ID_C",Value:"C[03]"}},ExpectedEffect:"reduce A/B/C confusion while preserving rule recovery and execution",ParentEvidenceIDs:append([]string(nil),parents...)}}
	}
	if target=="FROM_STATE_PRECONDITION_VISIBILITY"{
		preserved:=appendUnique(append([]string(nil),intent.Preserve...),"CELL_IDENTITY_ENCODING","TEMPORAL_GRAMMAR","EXECUTION_POLICY","PROGRAM_SEMANTICS","PAYLOAD","INITIAL_STATES")
		forbidden:=appendUnique(append([]string(nil),intent.Avoid...),"CELL_IDENTITY_MUTATION","RULE_MUTATION","STATE_MUTATION","EXECUTION_POLICY_MUTATION","CHECKPOINT_MUTATION","PAYLOAD_MUTATION","GENERATIVE_SEMANTIC_REWRITE","MULTI_MODULE_MUTATION","TEMPORAL_RULE_MUTATION")
		return []experimentpolicy.CandidateManifest{{Schema:experimentpolicy.CandidateSchemaR1,ID:"from-state-precondition-visible-r1",ParentID:intent.BaselineCandidateID,ProgramSHA256:programSHA,PayloadSHA256:payloadSHA,Mutations:[]experimentpolicy.Mutation{{Kind:"TEMPORAL_STRUCTURE",Target:"FROM_STATE_PRECONDITION_VISIBILITY",Value:"VISIBLE_FROM_STATE_PRECONDITION_R1"}},ChangedModules:[]string{"FROM_STATE_PRECONDITION_VISIBILITY"},PreservedModules:preserved,ForbiddenChanges:forbidden,ExpectedSemanticChanges:[]experimentpolicy.SemanticFact{{Key:"FROM_STATE_PRECONDITION_VISIBILITY",Value:"VISIBLE_FROM_STATE_PRECONDITION_R1"}},ExpectedEffect:"make target FROM state an explicit mandatory firing condition",ParentEvidenceIDs:append([]string(nil),parents...)}}
	}
	if target=="RULE_ROLE_BINDING"{
		preserved:=appendUnique(append([]string(nil),intent.Preserve...),"CELL_IDENTITY_ENCODING","FROM_STATE_PRECONDITION_VISIBILITY","TEMPORAL_GRAMMAR","EXECUTION_POLICY","PROGRAM_SEMANTICS","PAYLOAD")
		forbidden:=appendUnique(append([]string(nil),intent.Avoid...),"CELL_IDENTITY_MUTATION","EXECUTION_POLICY_MUTATION","CHECKPOINT_MUTATION","PAYLOAD_MUTATION","GENERATIVE_SEMANTIC_REWRITE","MULTI_MODULE_MUTATION")
		return []experimentpolicy.CandidateManifest{{Schema:experimentpolicy.CandidateSchemaR1,ID:"rule-role-binding-unseen-rules-r1",ParentID:intent.BaselineCandidateID,ProgramSHA256:programSHA,PayloadSHA256:payloadSHA,Mutations:[]experimentpolicy.Mutation{{Kind:"TEMPORAL_STRUCTURE",Target:"RULE_ROLE_BINDING",Value:"VISIBLE_RULE_ROLE_BINDING_R1"}},ChangedModules:[]string{"RULE_ROLE_BINDING"},PreservedModules:preserved,ForbiddenChanges:forbidden,ExpectedSemanticChanges:[]experimentpolicy.SemanticFact{{Key:"RULE_ROLE_BINDING",Value:"VISIBLE_RULE_ROLE_BINDING_R1"}},ExpectedEffect:"separate WHEN source, TARGET identity, REQUIRE target-from-state and SET target-to-state; evaluate on unseen rules",ParentEvidenceIDs:append([]string(nil),parents...)}}
	}
	type h struct{id,kind,mutTarget,value,semanticKey,semanticValue,effect string};hs:=[]h{}
	switch target{case "EXECUTION_POLICY":hs=[]h{{"execute-to-stable-text-r1","PROMPT","EXECUTION_POLICY","EXECUTE_VISIBLE_RULES_TO_STABLE_R1","EXECUTION_POLICY","EXECUTE_VISIBLE_RULES_TO_STABLE_R1","receiver executes visible rules until no state changes"}};case "TEMPORAL_GRAMMAR":hs=[]h{{"temporal-grammar-visible-r1","TEMPORAL_STRUCTURE","T2_SEMANTIC_TEMPORAL_SUPERGRAPH","VISIBLE_RULE_MICROGRAMMAR_R1","TEMPORAL_GRAMMAR","VISIBLE_RULE_MICROGRAMMAR_R1","make causal rule semantics recoverable"}};case "SEMANTIC_PARITY_GATE":return nil;default:return nil}
	out:=make([]experimentpolicy.CandidateManifest,0,len(hs));for _,x:=range hs{out=append(out,experimentpolicy.CandidateManifest{Schema:experimentpolicy.CandidateSchemaR1,ID:x.id,ParentID:intent.BaselineCandidateID,ProgramSHA256:programSHA,PayloadSHA256:payloadSHA,Mutations:[]experimentpolicy.Mutation{{Kind:x.kind,Target:x.mutTarget,Value:x.value}},ChangedModules:[]string{intent.MutableModule},PreservedModules:append([]string(nil),intent.Preserve...),ForbiddenChanges:append([]string{"PROGRAM_SEMANTICS","PAYLOAD","UNRELATED_PROMPT_MODULES"},intent.Avoid...),ExpectedSemanticChanges:[]experimentpolicy.SemanticFact{{Key:x.semanticKey,Value:x.semanticValue}},ExpectedEffect:x.effect,ParentEvidenceIDs:append([]string(nil),parents...)})};return out
}

func rulesByKind(p learningpolicy.Policy)(preserve,avoid,require []string){for _,r:=range p.Rules{switch r.Kind{case learningpolicy.RulePreserve:preserve=append(preserve,r.Target);case learningpolicy.RuleAvoid:avoid=append(avoid,r.Target);case learningpolicy.RuleRequire:require=append(require,r.Target)}};return}
func appendUnique(in []string,values ...string)[]string{seen:=map[string]bool{};out:=make([]string,0,len(in)+len(values));for _,v:=range append(in,values...){if v==""||seen[v]{continue};seen[v]=true;out=append(out,v)};return out}
func normalize(s string)string{r:=strings.NewReplacer(" ","-","_","-","/","-");s=strings.ToLower(strings.TrimSpace(s));if s==""{return "general"};return r.Replace(s)}
func ValidatePlan(p Plan)error{if p.Schema!=PlanSchemaR1{return fmt.Errorf("unexpected plan schema %q",p.Schema)};if p.Intent.MutableModule==""{return fmt.Errorf("mutable module is required")};for _,c:=range p.Candidates{if len(c.ChangedModules)!=1{return fmt.Errorf("candidate %s violates one-primary-mutation policy",c.ID)};if c.ChangedModules[0]!=p.Intent.MutableModule{return fmt.Errorf("candidate %s mutates %s outside intent %s",c.ID,c.ChangedModules[0],p.Intent.MutableModule)};if len(c.ExpectedSemanticChanges)==0{return fmt.Errorf("candidate %s lacks exact expected semantic changes",c.ID)}};return nil}
