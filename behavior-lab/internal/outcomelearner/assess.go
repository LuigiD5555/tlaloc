package outcomelearner

import "strings"

func Assess(req Request) (Assessment, KnowledgeUpdate) {
	a := Assessment{Schema: AssessmentSchemaR1, HypothesisID: req.HypothesisID, BeforeScore: req.Before.OverallScore, AfterScore: req.After.OverallScore}
	a.Delta = a.AfterScore - a.BeforeScore
	a.CausalConfidence = causalConfidence(len(req.ChangedModules))
	a.ModelPenaltyAllowed = req.After.ValidSpecimen
	if !req.After.ValidSpecimen {
		a.Classification = OutcomeInvalidExperiment
		k := KnowledgeUpdate{Schema: KnowledgeSchemaR1, HypothesisID: req.HypothesisID, Action: "RETAIN_HYPOTHESIS_REJECT_SPECIMEN", Avoid: []string{"INVALID_SPECIMEN_PATH"}, Reason: "invalid specimen cannot be attributed to model or hypothesis"}
		return a, k
	}
	before := assertionMap(req.Before.Assertions)
	after := assertionMap(req.After.Assertions)
	if req.TargetAssertion != "" {
		a.TargetResolved = !before[req.TargetAssertion].Pass && after[req.TargetAssertion].Pass
	}
	for id, b := range before {
		if b.Pass && !after[id].Pass {
			a.Regressions = append(a.Regressions, id)
		}
	}
	a.FrontierMoved = !strings.EqualFold(strings.TrimSpace(req.Before.FailureFrontier), strings.TrimSpace(req.After.FailureFrontier))
	switch {
	case len(a.Regressions) > 0:
		a.Classification = OutcomeRegression
	case a.TargetResolved:
		a.Classification = OutcomeSuccessfulCausalStep
	case a.Delta > 0:
		a.Classification = OutcomeSuccessfulCausalStep
	default:
		a.Classification = OutcomeNoImprovement
	}
	k := KnowledgeUpdate{Schema: KnowledgeSchemaR1, HypothesisID: req.HypothesisID}
	switch a.Classification {
	case OutcomeSuccessfulCausalStep:
		k.Action = "PROMOTE_TO_PROVISIONAL_WIN"
		k.Maturity = "PROVISIONAL_WIN"
		k.Preserve = append([]string(nil), req.ChangedModules...)
		k.Reason = "target improved without material regression"
	case OutcomeRegression:
		k.Action = "RECORD_REGRESSION"
		k.Avoid = append([]string(nil), req.ChangedModules...)
		k.Reason = "candidate regressed previously passing assertions"
	default:
		k.Action = "RECORD_FAILED_HYPOTHESIS"
		k.Avoid = append([]string(nil), req.ChangedModules...)
		k.Reason = "target did not improve"
	}
	return a, k
}

type assertionState struct {
	Pass  bool
	Score float64
}

func assertionMap(in []Assertion) map[string]assertionState {
	m := map[string]assertionState{}
	for _, a := range in {
		m[a.ID] = assertionState{Pass: a.Pass, Score: a.Score}
	}
	return m
}
func causalConfidence(changed int) string {
	switch changed {
	case 0:
		return "UNKNOWN"
	case 1:
		return "HIGH"
	case 2:
		return "MEDIUM"
	default:
		return "LOW"
	}
}
