package learningpolicy

import (
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/learningmemory"
)

func Derive(events []learningmemory.Event) Policy {
	summary := learningmemory.BuildSummary("", events)
	p := Policy{
		Schema:          SchemaR1,
		Target:          summary.NextDebugTarget,
		Rules:           []Rule{},
		Invariants:      []LearnedInvariant{},
		AntiPatterns:    []AntiPattern{},
		Guardrails:      []string{"ONE_PRIMARY_MUTATION", "FALSE_EXACT_ZERO", "INVALID_SPECIMEN_DOES_NOT_PENALIZE_MODEL", "SEMANTIC_PARITY_BEFORE_REAL_MODEL", "VISIBLE_TEXT_FIDELITY_BEFORE_REAL_MODEL", "REGRESSION_PRECHECK_BEFORE_REAL_MODEL", "CROSS_MODEL_NON_REGRESSION", "MODEL_PANEL_BEFORE_PROMOTION", "TLALOC_RECOMMENDS_ORIGAMI_DECIDES"},
		Authority:       "EXPERIMENT_PLANNING_ONLY",
	}
	if len(summary.TopRealFailurePatterns) > 0 {
		f := summary.TopRealFailurePatterns[0]
		p.FailureFrontier = f.FailureCode
		p.ParentEvidenceIDs = evidenceIDsForPattern(events, f.Stage, f.FailureCode, f.ScoreLayer)
	}
	if p.Target != "" {
		p.Rules = append(p.Rules, Rule{Kind: RuleMutable, Target: p.Target, Reason: "current real-model failure frontier"})
	}

	candidateModules := modulesByCandidate(events)
	for _, out := range summary.CandidateOutcomes {
		targets := candidateModules[out.CandidateID]
		if len(targets)==0 { targets=[]string{out.CandidateID} }
		for _, target := range targets {
			if out.Outcomes > 0 && out.MeanDelta > 0 {
				p.Rules = append(p.Rules, Rule{Kind: RulePreserve, Target: target, Reason: "historical positive outcome", Confidence: confidenceForOutcome(out.Outcomes)})
				p.Invariants = append(p.Invariants, LearnedInvariant{
					ID: "preserve-" + sanitize(target), Scope: "prompt-module", Maturity: maturityForOutcome(out.Outcomes),
					Preserve: []string{target}, Reason: "positive outcome must not be changed incidentally", Protected: true,
				})
			}
			if out.Outcomes > 0 && out.MeanDelta < 0 {
				p.Rules = append(p.Rules, Rule{Kind: RuleAvoid, Target: target, Reason: "historical negative outcome", Confidence: confidenceForOutcome(out.Outcomes)})
			}
		}
	}

	// A model that already passes a real-model baseline becomes a protected
	// compatibility invariant. Future candidates may improve other panel members,
	// but they may not trade away this success.
	passModels := map[string][]string{}
	passEvidence := map[string][]string{}
	for _, e := range events {
		if e.EventType != learningmemory.EventObservation || e.EvidenceClass != learningmemory.EvidenceRealModel || e.Pass == nil || !*e.Pass || strings.TrimSpace(e.ModelID) == "" { continue }
		key := strings.TrimSpace(e.CandidateID)
		if key == "" { key = strings.TrimSpace(e.SpecimenID) }
		if key == "" { continue }
		passModels[key] = append(passModels[key], e.ModelID)
		if e.EventID != "" { passEvidence[key] = append(passEvidence[key], e.EventID) }
	}
	for baseline, models := range passModels {
		models = uniqueSorted(models)
		evidence := uniqueSorted(passEvidence[baseline])
		p.Invariants = append(p.Invariants, LearnedInvariant{
			ID: "cross-model-preserve-" + sanitize(baseline), Scope: "model-compatibility", Maturity: MaturityObservedWin,
			Preserve: []string{"REAL_MODEL_PASS"}, Reason: "models that pass the baseline must remain passing on promoted candidates", EvidenceIDs: evidence, Models: models, Protected: true,
		})
	}

	invalidIDs := []string{}
	for _, e := range events {
		if strings.EqualFold(e.FailureCode, "ARTIFACT_GENERATION_REGRESSION") || hasTag(e.Tags, "invalid-specimen") || hasTag(e.Tags, "semantic-drift") {
			invalidIDs = append(invalidIDs, e.EventID)
		}
	}
	if len(invalidIDs) > 0 {
		sort.Strings(invalidIDs)
		p.AntiPatterns = append(p.AntiPatterns, AntiPattern{
			ID: "GENERATIVE_REWRITE_OF_EXACT_SEMANTICS",
			Trigger: "candidate contains canonical program semantics",
			Failure: "free-form rendering may alter rules, states or transitions",
			Policy: "exact semantic elements must be authored from structured IR and pass semantic parity before VLM testing",
			EvidenceIDs: invalidIDs,
		})
	}

	// Integrity gates are permanent system requirements. Memory explains why they
	// exist, but a fresh/rotated memory store must never silently disable them.
	for _, target := range []string{"SEMANTIC_PARITY_GATE", "VISIBLE_TEXT_FIDELITY_GATE", "REGRESSION_PRECHECK", "CROSS_MODEL_COMPATIBILITY_GATE", "PROGRAM_SHA", "PAYLOAD_SHA", "PROVENANCE", "RAW_RESPONSE_IMMUTABILITY"} {
		p.Rules = append(p.Rules, Rule{Kind: RuleRequire, Target: target, Reason: "experimental integrity"})
	}
	return dedupe(p)
}

func modulesByCandidate(events []learningmemory.Event) map[string][]string {
	sets:=map[string]map[string]bool{}
	for _,e:=range events{
		if e.EventType!=learningmemory.EventChange||e.CandidateID==""{continue}
		for _,tag:=range e.Tags{
			lower:=strings.ToLower(tag)
			if !strings.HasPrefix(lower,"module:"){continue}
			m:=strings.TrimSpace(tag[len("module:"):]);if m==""{continue}
			if sets[e.CandidateID]==nil{sets[e.CandidateID]=map[string]bool{}}
			sets[e.CandidateID][m]=true
		}
	}
	out:=map[string][]string{}
	for id,set:=range sets{for m:=range set{out[id]=append(out[id],m)};sort.Strings(out[id])}
	return out
}

func dedupe(p Policy) Policy {
	seen:=map[string]bool{};rules:=make([]Rule,0,len(p.Rules))
	for _,r:=range p.Rules{key:=r.Kind+"|"+r.Target;if seen[key]{continue};seen[key]=true;rules=append(rules,r)}
	p.Rules=rules
	return p
}

func evidenceIDsForPattern(events []learningmemory.Event, stage, failure, layer string) []string {
	out := []string{}
	for _, e := range events {
		if e.EventType != learningmemory.EventObservation || e.EvidenceClass != learningmemory.EvidenceRealModel || e.Pass == nil || *e.Pass { continue }
		if strings.EqualFold(e.LastCompletedStage, stage) && strings.EqualFold(e.FailureCode, failure) && strings.EqualFold(e.ScoreLayer, layer) { out = append(out, e.EventID) }
	}
	sort.Strings(out)
	return out
}

func uniqueSorted(in []string) []string { set:=map[string]bool{};for _,v:=range in{v=strings.TrimSpace(v);if v!=""{set[v]=true}};out:=make([]string,0,len(set));for v:=range set{out=append(out,v)};sort.Strings(out);return out }
func hasTag(tags []string, want string) bool { for _, t := range tags { if strings.EqualFold(t, want) { return true } }; return false }
func confidenceForOutcome(n int) string { switch { case n >= 9: return "HIGH"; case n >= 3: return "MEDIUM"; default: return "LOW" } }
func maturityForOutcome(n int) string { switch { case n >= 9: return MaturityReplicatedWin; case n >= 3: return MaturityProvisionalWin; default: return MaturityObservedWin } }
func sanitize(s string) string { s=strings.ToLower(strings.TrimSpace(s));r:=strings.NewReplacer("/","-"," ","-","_","-",":","-");return r.Replace(s) }
