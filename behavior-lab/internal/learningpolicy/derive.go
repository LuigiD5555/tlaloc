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
		Guardrails:      []string{"ONE_PRIMARY_MUTATION", "FALSE_EXACT_ZERO", "INVALID_SPECIMEN_DOES_NOT_PENALIZE_MODEL", "TLALOC_RECOMMENDS_ORIGAMI_DECIDES"},
		Authority:       "EXPERIMENT_PLANNING_ONLY",
	}
	if len(summary.TopRealFailurePatterns) > 0 {
		f := summary.TopRealFailurePatterns[0]
		p.FailureFrontier = f.FailureCode
		p.ParentEvidenceIDs = evidenceIDsForPattern(events, f.Stage, f.FailureCode, f.ScoreLayer)
	}

	// Current target is the only mutable development area by default.
	if p.Target != "" {
		p.Rules = append(p.Rules, Rule{Kind: RuleMutable, Target: p.Target, Reason: "current real-model failure frontier"})
	}

	// Successful outcomes are retained as provisional preservation signals.
	for _, out := range summary.CandidateOutcomes {
		if out.Outcomes > 0 && out.MeanDelta > 0 {
			p.Rules = append(p.Rules, Rule{Kind: RulePreserve, Target: out.CandidateID, Reason: "historical positive outcome", Confidence: confidenceForOutcome(out.Outcomes)})
			p.Invariants = append(p.Invariants, LearnedInvariant{
				ID: "preserve-" + sanitize(out.CandidateID), Scope: "candidate", Maturity: maturityForOutcome(out.Outcomes),
				Preserve: []string{out.CandidateID}, Reason: "positive outcome must not be changed incidentally", Protected: true,
			})
		}
		if out.Outcomes > 0 && out.MeanDelta < 0 {
			p.Rules = append(p.Rules, Rule{Kind: RuleAvoid, Target: out.CandidateID, Reason: "historical negative outcome", Confidence: confidenceForOutcome(out.Outcomes)})
		}
	}

	// Process failures create durable anti-patterns. These do not become model failures.
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
		p.Rules = append(p.Rules, Rule{Kind: RuleRequire, Target: "SEMANTIC_PARITY_GATE", Reason: "invalid specimen history", EvidenceIDs: invalidIDs})
	}

	// Core system invariants are always required.
	for _, target := range []string{"PROGRAM_SHA", "PAYLOAD_SHA", "PROVENANCE", "RAW_RESPONSE_IMMUTABILITY"} {
		p.Rules = append(p.Rules, Rule{Kind: RuleRequire, Target: target, Reason: "experimental integrity"})
	}

	return p
}

func evidenceIDsForPattern(events []learningmemory.Event, stage, failure, layer string) []string {
	out := []string{}
	for _, e := range events {
		if e.EventType != learningmemory.EventObservation || e.EvidenceClass != learningmemory.EvidenceRealModel || e.Pass == nil || *e.Pass {
			continue
		}
		if strings.EqualFold(e.LastCompletedStage, stage) && strings.EqualFold(e.FailureCode, failure) && strings.EqualFold(e.ScoreLayer, layer) {
			out = append(out, e.EventID)
		}
	}
	sort.Strings(out)
	return out
}

func hasTag(tags []string, want string) bool {
	for _, t := range tags { if strings.EqualFold(t, want) { return true } }
	return false
}

func confidenceForOutcome(n int) string {
	switch { case n >= 9: return "HIGH"; case n >= 3: return "MEDIUM"; default: return "LOW" }
}

func maturityForOutcome(n int) string {
	switch { case n >= 9: return MaturityReplicatedWin; case n >= 3: return MaturityProvisionalWin; default: return MaturityObservedWin }
}

func sanitize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	r := strings.NewReplacer("/", "-", " ", "-", "_", "-", ":", "-")
	return r.Replace(s)
}
