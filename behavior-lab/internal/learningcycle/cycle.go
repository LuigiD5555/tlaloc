package learningcycle

import (
	"fmt"
	"strings"

	"tlaloc.local/behaviorlab/internal/adaptivesearch"
	"tlaloc.local/behaviorlab/internal/experimentpolicy"
	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/learningpolicy"
)

func BuildStatus(root string, events []learningmemory.Event) Status {
	policy := learningpolicy.Derive(events)
	adaptive := adaptivesearch.BuildPlan(root, events)
	return Status{Schema:StatusSchemaR1, FailureFrontier:policy.FailureFrontier, NextTarget:policy.Target, Policy:policy, AdaptiveSearch:adaptive, Promotion:"NOT_EVALUATED_BY_LEARNING_CYCLE"}
}

func BuildPlan(root string, events []learningmemory.Event, baseline, programSHA, payloadSHA string, budget int) Plan {
	st := BuildStatus(root, events)
	if budget <= 0 { budget = 3 }
	preserve, avoid, require := rulesByKind(st.Policy)
	intent := experimentpolicy.ExperimentIntent{
		Schema: experimentpolicy.IntentSchemaR1,
		ID: "intent-" + normalize(st.NextTarget),
		Objective: "improve " + st.NextTarget + " without regressing learned invariants",
		BaselineCandidateID: baseline,
		FailureFrontier: st.FailureFrontier,
		MutableModule: st.NextTarget,
		Preserve: preserve,
		Avoid: avoid,
		Require: require,
		CandidateBudget: budget,
		TrialsPerModel: 3,
	}
	candidates := synthesize(intent, st.Policy.ParentEvidenceIDs, programSHA, payloadSHA)
	if len(candidates) > budget { candidates = candidates[:budget] }
	return Plan{Schema:PlanSchemaR1, Status:st, Intent:intent, Candidates:candidates}
}

func synthesize(intent experimentpolicy.ExperimentIntent, parents []string, programSHA, payloadSHA string) []experimentpolicy.CandidateManifest {
	target := strings.ToUpper(strings.TrimSpace(intent.MutableModule))
	type h struct{ id, kind, mutTarget, value, effect string }
	hs := []h{}
	switch target {
	case "EXECUTION_POLICY":
		hs = []h{
			{"execute-to-stable-text-r1","PROMPT","EXECUTION_POLICY","EXECUTE_VISIBLE_RULES_TO_STABLE_R1","receiver executes visible rules until no state changes"},
			{"execute-loop-compact-r1","PROMPT","EXECUTION_POLICY","INIT_APPLY_NEXT_REPEAT_STABLE_COMPACT_R1","compact loop improves execution compliance"},
			{"stop-condition-redundant-r1","REDUNDANCY","EXECUTION_POLICY","REPEAT_UNTIL_UNCHANGED_REPORT_FINAL_R1","redundant stop condition improves stable-state reporting"},
		}
	case "TEMPORAL_GRAMMAR":
		hs = []h{{"temporal-grammar-visible-r1","TEMPORAL_STRUCTURE","TEMPORAL_GRAMMAR","VISIBLE_RULE_MICROGRAMMAR_R1","make causal rule semantics recoverable"}}
	case "SEMANTIC_PARITY_GATE":
		hs = []h{{"semantic-parity-hard-gate-r1","VALIDATION","SEMANTIC_PARITY_GATE","REJECT_UNAUTHORIZED_SEMANTIC_DRIFT","prevent invalid specimens from reaching models"}}
	default:
		hs = []h{
			{"target-prompt-r1","PROMPT",target,"SHORT_EXPLICIT_PROTOCOL_INSTRUCTION","isolate prompt guidance for current frontier"},
			{"target-layout-r1","LAYOUT",target,"LOCALIZE_FAILURE_TARGET_REGION","isolate layout guidance for current frontier"},
			{"target-redundancy-r1","REDUNDANCY",target,"BOUNDED_REDUNDANT_ANCHOR","isolate bounded redundant cue for current frontier"},
		}
	}
	out := make([]experimentpolicy.CandidateManifest,0,len(hs))
	for _, x := range hs {
		out=append(out,experimentpolicy.CandidateManifest{
			Schema:experimentpolicy.CandidateSchemaR1, ID:x.id, ParentID:intent.BaselineCandidateID,
			ProgramSHA256:programSHA, PayloadSHA256:payloadSHA,
			Mutations:[]experimentpolicy.Mutation{{Kind:x.kind,Target:x.mutTarget,Value:x.value}},
			ChangedModules:[]string{intent.MutableModule}, PreservedModules:append([]string(nil),intent.Preserve...),
			ForbiddenChanges:append([]string{"PROGRAM_SEMANTICS","PAYLOAD","UNRELATED_PROMPT_MODULES"},intent.Avoid...),
			ExpectedEffect:x.effect, ParentEvidenceIDs:append([]string(nil),parents...),
		})
	}
	return out
}

func rulesByKind(p learningpolicy.Policy)(preserve,avoid,require []string){
	for _,r:=range p.Rules{
		switch r.Kind{case learningpolicy.RulePreserve:preserve=append(preserve,r.Target);case learningpolicy.RuleAvoid:avoid=append(avoid,r.Target);case learningpolicy.RuleRequire:require=append(require,r.Target)}
	}
	return
}

func normalize(s string)string{r:=strings.NewReplacer(" ","-","_","-","/","-");s=strings.ToLower(strings.TrimSpace(s));if s==""{return "general"};return r.Replace(s)}

func ValidatePlan(p Plan) error {
	if p.Schema!=PlanSchemaR1{return fmt.Errorf("unexpected plan schema %q",p.Schema)}
	if p.Intent.MutableModule==""{return fmt.Errorf("mutable module is required")}
	for _,c:=range p.Candidates{if len(c.ChangedModules)!=1{return fmt.Errorf("candidate %s violates one-primary-mutation policy",c.ID)};if c.ChangedModules[0]!=p.Intent.MutableModule{return fmt.Errorf("candidate %s mutates %s outside intent %s",c.ID,c.ChangedModules[0],p.Intent.MutableModule)}}
	return nil
}
