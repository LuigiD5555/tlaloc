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

func BuildStatus(root string, events []learningmemory.Event) Status {
	return BuildStatusWithGenome(root,events,promptgenome.Genome{})
}

func BuildStatusWithGenome(root string, events []learningmemory.Event, genome promptgenome.Genome) Status {
	policy := learningpolicy.Derive(events)
	if genome.Schema==promptgenome.GenomeSchemaR1 { policy=learningpolicy.ApplyGenomeProtection(policy,genome) }
	adaptive := adaptivesearch.BuildPlan(root, events)
	return Status{Schema:StatusSchemaR1, FailureFrontier:policy.FailureFrontier, NextTarget:policy.Target, Policy:policy, AdaptiveSearch:adaptive, Promotion:"NOT_EVALUATED_BY_LEARNING_CYCLE"}
}

func BuildPlan(root string, events []learningmemory.Event, baseline, programSHA, payloadSHA string, budget int) Plan {
	return BuildPlanWithGenome(root,events,promptgenome.Genome{},baseline,programSHA,payloadSHA,budget)
}

func BuildPlanWithGenome(root string, events []learningmemory.Event, genome promptgenome.Genome, baseline, programSHA, payloadSHA string, budget int) Plan {
	st := BuildStatusWithGenome(root, events, genome)
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

// synthesize emits only mutations with a known deterministic materialization
// path. Unsupported ideas remain Adaptive Search suggestions until Origami
// advertises/implements them; they are not silently converted into specimens.
func synthesize(intent experimentpolicy.ExperimentIntent, parents []string, programSHA, payloadSHA string) []experimentpolicy.CandidateManifest {
	target := strings.ToUpper(strings.TrimSpace(intent.MutableModule))
	type h struct{ id, kind, mutTarget, value, effect string }
	hs := []h{}
	switch target {
	case "EXECUTION_POLICY":
		hs = []h{{"execute-to-stable-text-r1","PROMPT","EXECUTION_POLICY","EXECUTE_VISIBLE_RULES_TO_STABLE_R1","receiver executes visible rules until no state changes"}}
	case "TEMPORAL_GRAMMAR":
		hs = []h{{"temporal-grammar-visible-r1","TEMPORAL_STRUCTURE","T2_SEMANTIC_TEMPORAL_SUPERGRAPH","VISIBLE_RULE_MICROGRAMMAR_R1","make causal rule semantics recoverable"}}
	case "SEMANTIC_PARITY_GATE":
		// Validation-only target: no visual specimen should be synthesized.
		return nil
	default:
		// Generic Adaptive Search can rank ideas, but guarded synthesis refuses to
		// invent an Origami renderer contract that has not been negotiated.
		return nil
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
