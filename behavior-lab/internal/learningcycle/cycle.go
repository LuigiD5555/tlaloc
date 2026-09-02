package learningcycle

import (
	"fmt"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/adaptivesearch"
	"tlaloc.local/behaviorlab/internal/experimentpolicy"
	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/learningpolicy"
	"tlaloc.local/behaviorlab/internal/promptgenome"
)

func BuildStatus(root string, events []learningmemory.Event) Status {
	return BuildStatusWithGenome(root, events, promptgenome.Genome{})
}

func BuildStatusWithGenome(root string, events []learningmemory.Event, genome promptgenome.Genome) Status {
	policy := learningpolicy.Derive(events)
	if genome.Schema == promptgenome.GenomeSchemaR1 {
		policy = learningpolicy.ApplyGenomeProtection(policy, genome)
	}
	adaptive := adaptivesearch.BuildPlan(root, events)
	return Status{Schema: StatusSchemaR1, FailureFrontier: policy.FailureFrontier, NextTarget: policy.Target, Policy: policy, AdaptiveSearch: adaptive, Promotion: "NOT_EVALUATED_BY_LEARNING_CYCLE"}
}

func BuildPlan(root string, events []learningmemory.Event, baseline, programSHA, payloadSHA string, budget int) Plan {
	return BuildPlanWithGenome(root, events, promptgenome.Genome{}, baseline, programSHA, payloadSHA, budget)
}

func BuildPlanWithGenome(root string, events []learningmemory.Event, genome promptgenome.Genome, baseline, programSHA, payloadSHA string, budget int) Plan {
	st := BuildStatusWithGenome(root, events, genome)
	if budget <= 0 {
		budget = 3
	}
	preserve, avoid, require := rulesByKind(st.Policy)
	panel := compatibilityPanel(events, baseline)
	intent := experimentpolicy.ExperimentIntent{
		Schema:              experimentpolicy.IntentSchemaR1,
		ID:                  "intent-" + normalize(st.NextTarget),
		Objective:           "improve " + st.NextTarget + " without regressing learned invariants or passing model-panel members",
		BaselineCandidateID: baseline,
		FailureFrontier:     st.FailureFrontier,
		MutableModule:       st.NextTarget,
		Preserve:            preserve,
		Avoid:               avoid,
		Require:             require,
		CandidateBudget:     budget,
		TrialsPerModel:      3,
		Models:              panelModelIDs(panel),
		CompatibilityPanel:  panel,
	}
	candidates := synthesize(intent, st.Policy.ParentEvidenceIDs, programSHA, payloadSHA)
	if len(candidates) > budget {
		candidates = candidates[:budget]
	}
	return Plan{Schema: PlanSchemaR1, Status: st, Intent: intent, Candidates: candidates}
}

func synthesize(intent experimentpolicy.ExperimentIntent, parents []string, programSHA, payloadSHA string) []experimentpolicy.CandidateManifest {
	target := strings.ToUpper(strings.TrimSpace(intent.MutableModule))
	if target == "CELL_IDENTITY_ENCODING" {
		preserved := appendUnique(append([]string(nil), intent.Preserve...), "TEMPORAL_GRAMMAR", "EXECUTION_POLICY", "PROGRAM_SEMANTICS", "PAYLOAD", "INITIAL_STATES")
		forbidden := appendUnique(append([]string(nil), intent.Avoid...), "RULE_MUTATION", "STATE_MUTATION", "EXECUTION_POLICY_MUTATION", "CHECKPOINT_MUTATION", "PAYLOAD_MUTATION", "GENERATIVE_SEMANTIC_REWRITE", "MULTI_MODULE_MUTATION", "TEMPORAL_RULE_MUTATION")
		return []experimentpolicy.CandidateManifest{attachPanel(experimentpolicy.CandidateManifest{
			Schema: experimentpolicy.CandidateSchemaR1, ID: "cell-identity-redundancy-r1", ParentID: intent.BaselineCandidateID, ProgramSHA256: programSHA, PayloadSHA256: payloadSHA,
			Mutations: []experimentpolicy.Mutation{{Kind: "REDUNDANCY", Target: "CELL_IDENTITY_ENCODING", Value: "VISIBLE_CELL_ID_REDUNDANCY_R1"}}, ChangedModules: []string{"CELL_IDENTITY_ENCODING"}, PreservedModules: preserved, ForbiddenChanges: forbidden,
			ExpectedSemanticChanges: []experimentpolicy.SemanticFact{{Key: "VISIBLE_CELL_ID_A", Value: "A[01]"}, {Key: "VISIBLE_CELL_ID_B", Value: "B[02]"}, {Key: "VISIBLE_CELL_ID_C", Value: "C[03]"}}, ExpectedEffect: "reduce A/B/C confusion while preserving rule recovery and execution", ParentEvidenceIDs: append([]string(nil), parents...),
		}, intent.CompatibilityPanel)}
	}
	if target == "FROM_STATE_PRECONDITION_VISIBILITY" {
		preserved := appendUnique(append([]string(nil), intent.Preserve...), "CELL_IDENTITY_ENCODING", "TEMPORAL_GRAMMAR", "EXECUTION_POLICY", "PROGRAM_SEMANTICS", "PAYLOAD", "INITIAL_STATES")
		forbidden := appendUnique(append([]string(nil), intent.Avoid...), "CELL_IDENTITY_MUTATION", "RULE_MUTATION", "STATE_MUTATION", "EXECUTION_POLICY_MUTATION", "CHECKPOINT_MUTATION", "PAYLOAD_MUTATION", "GENERATIVE_SEMANTIC_REWRITE", "MULTI_MODULE_MUTATION", "TEMPORAL_RULE_MUTATION")
		return []experimentpolicy.CandidateManifest{attachPanel(experimentpolicy.CandidateManifest{
			Schema: experimentpolicy.CandidateSchemaR1, ID: "from-state-precondition-visible-r1", ParentID: intent.BaselineCandidateID, ProgramSHA256: programSHA, PayloadSHA256: payloadSHA,
			Mutations: []experimentpolicy.Mutation{{Kind: "TEMPORAL_STRUCTURE", Target: "FROM_STATE_PRECONDITION_VISIBILITY", Value: "VISIBLE_FROM_STATE_PRECONDITION_R1"}}, ChangedModules: []string{"FROM_STATE_PRECONDITION_VISIBILITY"}, PreservedModules: preserved, ForbiddenChanges: forbidden,
			ExpectedSemanticChanges: []experimentpolicy.SemanticFact{{Key: "FROM_STATE_PRECONDITION_VISIBILITY", Value: "VISIBLE_FROM_STATE_PRECONDITION_R1"}}, ExpectedEffect: "make target FROM state an explicit mandatory firing condition", ParentEvidenceIDs: append([]string(nil), parents...),
		}, intent.CompatibilityPanel)}
	}
	if target == "RULE_ROLE_BINDING" {
		preserved := appendUnique(append([]string(nil), intent.Preserve...), "CELL_IDENTITY_ENCODING", "FROM_STATE_PRECONDITION_VISIBILITY", "TEMPORAL_GRAMMAR", "EXECUTION_POLICY", "PROGRAM_SEMANTICS", "PAYLOAD")
		forbidden := appendUnique(append([]string(nil), intent.Avoid...), "CELL_IDENTITY_MUTATION", "EXECUTION_POLICY_MUTATION", "CHECKPOINT_MUTATION", "PAYLOAD_MUTATION", "GENERATIVE_SEMANTIC_REWRITE", "MULTI_MODULE_MUTATION")
		return []experimentpolicy.CandidateManifest{attachPanel(experimentpolicy.CandidateManifest{
			Schema: experimentpolicy.CandidateSchemaR1, ID: "rule-role-binding-unseen-rules-r1", ParentID: intent.BaselineCandidateID, ProgramSHA256: programSHA, PayloadSHA256: payloadSHA,
			Mutations: []experimentpolicy.Mutation{{Kind: "TEMPORAL_STRUCTURE", Target: "RULE_ROLE_BINDING", Value: "VISIBLE_RULE_ROLE_BINDING_R1"}}, ChangedModules: []string{"RULE_ROLE_BINDING"}, PreservedModules: preserved, ForbiddenChanges: forbidden,
			ExpectedSemanticChanges: []experimentpolicy.SemanticFact{{Key: "RULE_ROLE_BINDING", Value: "VISIBLE_RULE_ROLE_BINDING_R1"}}, ExpectedEffect: "separate WHEN source, TARGET identity, REQUIRE target-from-state and SET target-to-state; evaluate on unseen rules", ParentEvidenceIDs: append([]string(nil), parents...),
		}, intent.CompatibilityPanel)}
	}
	if target == "EXECUTION_POLICY_COMPLIANCE" {
		preserved := appendUnique(append([]string(nil), intent.Preserve...), "CELL_IDENTITY_ENCODING", "FROM_STATE_PRECONDITION_VISIBILITY", "RULE_ROLE_BINDING", "TEMPORAL_GRAMMAR", "EXECUTION_POLICY", "PROGRAM_SEMANTICS", "PAYLOAD", "INITIAL_STATES")
		forbidden := appendUnique(append([]string(nil), intent.Avoid...), "CELL_IDENTITY_MUTATION", "RULE_ROLE_BINDING_MUTATION", "FROM_STATE_PRECONDITION_MUTATION", "RULE_MUTATION", "STATE_MUTATION", "CHECKPOINT_MUTATION", "PAYLOAD_MUTATION", "GENERATIVE_SEMANTIC_REWRITE", "MULTI_MODULE_MUTATION")
		return []experimentpolicy.CandidateManifest{attachPanel(experimentpolicy.CandidateManifest{
			Schema: experimentpolicy.CandidateSchemaR1, ID: "execution-policy-compliance-cross-model-r1", ParentID: intent.BaselineCandidateID, ProgramSHA256: programSHA, PayloadSHA256: payloadSHA,
			Mutations: []experimentpolicy.Mutation{{Kind: "PROMPT", Target: "EXECUTION_POLICY_COMPLIANCE", Value: "EXECUTE_DONT_SUMMARIZE_TO_STABLE_R1"}}, ChangedModules: []string{"EXECUTION_POLICY_COMPLIANCE"}, PreservedModules: preserved, ForbiddenChanges: forbidden,
			ExpectedSemanticChanges: []experimentpolicy.SemanticFact{{Key: "EXECUTION_POLICY_COMPLIANCE", Value: "EXECUTE_DONT_SUMMARIZE_TO_STABLE_R1"}}, ExpectedEffect: "make execution mandatory rather than descriptive while preserving every passing model-panel baseline and improving failing panel members to PASS", ParentEvidenceIDs: append([]string(nil), parents...),
		}, intent.CompatibilityPanel)}
	}
	if target == "SYNCHRONOUS_EXECUTION_FIDELITY" {
		preserved := appendUnique(append([]string(nil), intent.Preserve...), "CELL_IDENTITY_ENCODING", "FROM_STATE_PRECONDITION_VISIBILITY", "RULE_ROLE_BINDING", "TEMPORAL_GRAMMAR", "EXECUTION_POLICY", "EXECUTION_POLICY_COMPLIANCE", "PROGRAM_SEMANTICS", "PAYLOAD", "INITIAL_STATES")
		forbidden := appendUnique(append([]string(nil), intent.Avoid...), "CELL_IDENTITY_MUTATION", "RULE_ROLE_BINDING_MUTATION", "FROM_STATE_PRECONDITION_MUTATION", "RULE_MUTATION", "STATE_MUTATION", "CHECKPOINT_MUTATION", "PAYLOAD_MUTATION", "PROGRAM_SEMANTICS_MUTATION", "GENERATIVE_SEMANTIC_REWRITE", "MULTI_MODULE_MUTATION")
		return []experimentpolicy.CandidateManifest{attachPanel(experimentpolicy.CandidateManifest{
			Schema: experimentpolicy.CandidateSchemaR1, ID: "synchronous-execution-fidelity-cross-model-r1", ParentID: intent.BaselineCandidateID, ProgramSHA256: programSHA, PayloadSHA256: payloadSHA,
			Mutations: []experimentpolicy.Mutation{{Kind: "PROMPT", Target: "SYNCHRONOUS_EXECUTION_FIDELITY", Value: "FREEZE_SELECT_APPLY_TOGETHER_R1"}}, ChangedModules: []string{"SYNCHRONOUS_EXECUTION_FIDELITY"}, PreservedModules: preserved, ForbiddenChanges: forbidden,
			ExpectedSemanticChanges: []experimentpolicy.SemanticFact{{Key: "SYNCHRONOUS_EXECUTION_FIDELITY", Value: "FREEZE_SELECT_APPLY_TOGETHER_R1"}}, ExpectedEffect: "make every step use one frozen snapshot, select all fireable rules, apply all selected SETs together, forbid rule order and forbid within-step cascading across the model panel", ParentEvidenceIDs: append([]string(nil), parents...),
		}, intent.CompatibilityPanel)}
	}

	type h struct{ id, kind, mutTarget, value, semanticKey, semanticValue, effect string }
	hs := []h{}
	switch target {
	case "EXECUTION_POLICY":
		hs = []h{{"execute-to-stable-text-r1", "PROMPT", "EXECUTION_POLICY", "EXECUTE_VISIBLE_RULES_TO_STABLE_R1", "EXECUTION_POLICY", "EXECUTE_VISIBLE_RULES_TO_STABLE_R1", "receiver executes visible rules until no state changes"}}
	case "TEMPORAL_GRAMMAR":
		hs = []h{{"temporal-grammar-visible-r1", "TEMPORAL_STRUCTURE", "T2_SEMANTIC_TEMPORAL_SUPERGRAPH", "VISIBLE_RULE_MICROGRAMMAR_R1", "TEMPORAL_GRAMMAR", "VISIBLE_RULE_MICROGRAMMAR_R1", "make causal rule semantics recoverable"}}
	case "SEMANTIC_PARITY_GATE":
		return nil
	default:
		return nil
	}
	out := make([]experimentpolicy.CandidateManifest, 0, len(hs))
	for _, x := range hs {
		out = append(out, attachPanel(experimentpolicy.CandidateManifest{Schema: experimentpolicy.CandidateSchemaR1, ID: x.id, ParentID: intent.BaselineCandidateID, ProgramSHA256: programSHA, PayloadSHA256: payloadSHA, Mutations: []experimentpolicy.Mutation{{Kind: x.kind, Target: x.mutTarget, Value: x.value}}, ChangedModules: []string{intent.MutableModule}, PreservedModules: append([]string(nil), intent.Preserve...), ForbiddenChanges: append([]string{"PROGRAM_SEMANTICS", "PAYLOAD", "UNRELATED_PROMPT_MODULES"}, intent.Avoid...), ExpectedSemanticChanges: []experimentpolicy.SemanticFact{{Key: x.semanticKey, Value: x.semanticValue}}, ExpectedEffect: x.effect, ParentEvidenceIDs: append([]string(nil), parents...)}, intent.CompatibilityPanel))
	}
	return out
}

func compatibilityPanel(events []learningmemory.Event, baseline string) []experimentpolicy.ModelCompatibilityRequirement {
	type state struct {
		pass     bool
		evidence []string
	}
	byModel := map[string]state{}
	for _, e := range events {
		if e.EventType != learningmemory.EventObservation || e.EvidenceClass != learningmemory.EvidenceRealModel || e.Pass == nil || strings.TrimSpace(e.ModelID) == "" {
			continue
		}
		if baseline != "" && !matchesBaseline(e, baseline) {
			continue
		}
		s := byModel[e.ModelID]
		s.pass = *e.Pass
		if e.EventID != "" {
			s.evidence = append(s.evidence, e.EventID)
		}
		byModel[e.ModelID] = s
	}
	ids := make([]string, 0, len(byModel))
	for id := range byModel {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]experimentpolicy.ModelCompatibilityRequirement, 0, len(ids))
	for _, id := range ids {
		s := byModel[id]
		mode := experimentpolicy.ModelCompatibilityImproveToPass
		if s.pass {
			mode = experimentpolicy.ModelCompatibilityPreservePass
		}
		out = append(out, experimentpolicy.ModelCompatibilityRequirement{ModelID: id, Mode: mode, BaselinePass: s.pass, RequiredCandidatePass: true, EvidenceIDs: uniqueStrings(s.evidence)})
	}
	return out
}

func matchesBaseline(e learningmemory.Event, baseline string) bool {
	if e.CandidateID == baseline || e.SpecimenID == baseline {
		return true
	}
	return e.SpecimenID != "" && strings.HasPrefix(e.SpecimenID, baseline+"-")
}

func panelModelIDs(panel []experimentpolicy.ModelCompatibilityRequirement) []string {
	out := make([]string, 0, len(panel))
	for _, p := range panel {
		if p.ModelID != "" {
			out = append(out, p.ModelID)
		}
	}
	sort.Strings(out)
	return out
}
func attachPanel(c experimentpolicy.CandidateManifest, panel []experimentpolicy.ModelCompatibilityRequirement) experimentpolicy.CandidateManifest {
	c.CompatibilityPanel = append([]experimentpolicy.ModelCompatibilityRequirement(nil), panel...)
	return c
}
func uniqueStrings(in []string) []string {
	set := map[string]bool{}
	for _, v := range in {
		if v != "" {
			set[v] = true
		}
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
func rulesByKind(p learningpolicy.Policy) (preserve, avoid, require []string) {
	for _, r := range p.Rules {
		switch r.Kind {
		case learningpolicy.RulePreserve:
			preserve = append(preserve, r.Target)
		case learningpolicy.RuleAvoid:
			avoid = append(avoid, r.Target)
		case learningpolicy.RuleRequire:
			require = append(require, r.Target)
		}
	}
	return
}
func appendUnique(in []string, values ...string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in)+len(values))
	for _, v := range append(in, values...) {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
func normalize(s string) string {
	r := strings.NewReplacer(" ", "-", "_", "-", "/", "-")
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "general"
	}
	return r.Replace(s)
}
func ValidatePlan(p Plan) error {
	if p.Schema != PlanSchemaR1 {
		return fmt.Errorf("unexpected plan schema %q", p.Schema)
	}
	if p.Intent.MutableModule == "" {
		return fmt.Errorf("mutable module is required")
	}
	for _, c := range p.Candidates {
		if len(c.ChangedModules) != 1 {
			return fmt.Errorf("candidate %s violates one-primary-mutation policy", c.ID)
		}
		if c.ChangedModules[0] != p.Intent.MutableModule {
			return fmt.Errorf("candidate %s mutates %s outside intent %s", c.ID, p.Intent.MutableModule, c.ChangedModules[0])
		}
		if len(c.ExpectedSemanticChanges) == 0 {
			return fmt.Errorf("candidate %s lacks exact expected semantic changes", c.ID)
		}
	}
	return nil
}
