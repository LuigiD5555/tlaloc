package visualsearch

import (
	"fmt"
	"sort"
	"strings"
)

func Evaluate(candidate Candidate, baseline Metrics, evidence Evidence, policy Policy) (Evaluation, error) {
	policy = normalizePolicy(policy)
	if candidate.Schema != "" && candidate.Schema != SchemaR0+".candidate" { return Evaluation{}, fmt.Errorf("candidate schema must be %q", SchemaR0+".candidate") }
	if strings.TrimSpace(candidate.ID) == "" { return Evaluation{}, fmt.Errorf("candidate id is required") }
	if strings.TrimSpace(candidate.BaseProfileID) == "" { return Evaluation{}, fmt.Errorf("base_profile_id is required") }
	if evidence.CandidateID != candidate.ID { return Evaluation{}, fmt.Errorf("candidate/evidence id mismatch") }
	if len(candidate.Mutations) == 0 { return Evaluation{}, fmt.Errorf("candidate must contain at least one mutation") }
	for i, mutation := range candidate.Mutations {
		if !validMutationKind(mutation.Kind) { return Evaluation{}, fmt.Errorf("mutation %d has unsupported kind %q", i, mutation.Kind) }
		if strings.TrimSpace(mutation.Target) == "" || strings.TrimSpace(mutation.Value) == "" { return Evaluation{}, fmt.Errorf("mutation %d requires target and value", i) }
		if !mutation.Experimental { return Evaluation{}, fmt.Errorf("candidate mutation %d must remain experimental until Origami promotion", i) }
	}

	metrics := evidence.Metrics
	baselineScore := score(baseline, baseline, policy)
	candidateScore := score(metrics, baseline, policy)
	improvement := candidateScore - baselineScore

	criticalNonRegression := metrics.SemanticRoundtripRate >= baseline.SemanticRoundtripRate &&
		metrics.NativeIndexRecoveryRate >= baseline.NativeIndexRecoveryRate &&
		metrics.NativeSemanticAnswerRate >= baseline.NativeSemanticAnswerRate &&
		metrics.VerifiedEvidenceRate >= baseline.VerifiedEvidenceRate &&
		metrics.RoutingAccuracy >= baseline.RoutingAccuracy &&
		metrics.MechanicalDependencyViolations <= baseline.MechanicalDependencyViolations &&
		metrics.UnverifiedMechanicalClaims <= baseline.UnverifiedMechanicalClaims &&
		metrics.FalseExact <= baseline.FalseExact &&
		metrics.BudgetViolations <= baseline.BudgetViolations &&
		metrics.UnknownViolations <= baseline.UnknownViolations

	perceptualCandidate := usesAdvancedPerceptualMutation(candidate)
	perceptualPass := !perceptualCandidate || metrics.PerceptualRevealRate >= policy.MinPerceptualRevealRate
	gates := []Gate{
		gate("SEMANTIC_ROUNDTRIP", metrics.SemanticRoundtripRate >= policy.MinSemanticRoundtripRate, "semantic roundtrip must remain complete"),
		gate("NATIVE_T2_INDEX", metrics.NativeIndexRecoveryRate >= policy.MinNativeIndexRecoveryRate, "prompt-only/native T2 index recovery below threshold"),
		gate("NATIVE_SEMANTIC_ANSWER", metrics.NativeSemanticAnswerRate >= policy.MinNativeSemanticAnswerRate, "prompt-only/native semantic answer rate below threshold"),
		gate("NO_MECHANICAL_DEPENDENCY", metrics.MechanicalDependencyViolations == 0, "semantic navigation required undeclared binary/file/sandbox decoding"),
		gate("NO_UNVERIFIED_MECHANICAL_CLAIMS", metrics.UnverifiedMechanicalClaims == 0, "model invented byte counts, hashes, compression or archive claims without verification"),
		gate("FALSE_EXACT_ZERO", metrics.FalseExact == 0, "FALSE_EXACT must remain zero"),
		gate("BUDGET_ZERO", metrics.BudgetViolations == 0, "budget violations must remain zero"),
		gate("UNKNOWN_DISCIPLINE", metrics.UnknownViolations == 0, "UNKNOWN discipline regression detected"),
		gate("CARRIER_SIZE", metrics.CarrierBytes > 0 && metrics.CarrierBytes <= policy.MaxCarrierBytes, "carrier size exceeds current profile ceiling"),
		gate("CONTEXT_BOUND", metrics.MeanContextTokens >= 0 && metrics.MeanContextTokens <= policy.MaxMeanContextTokens, "mean active context exceeds bound"),
		gate("VERIFIED_EVIDENCE", metrics.VerifiedEvidenceRate >= policy.MinVerifiedEvidenceRate, "verified evidence rate below threshold"),
		gate("ROUTING", metrics.RoutingAccuracy >= policy.MinRoutingAccuracy, "routing accuracy below threshold"),
		gate("PERCEPTUAL_REVEAL", perceptualPass, "advanced perceptual channel reveal rate below threshold"),
		gate("REAL_MODELS", metrics.RealModels >= policy.MinRealModelsForPerception, "insufficient real-model replication"),
		gate("TRIALS", metrics.Trials >= policy.MinTrials, "insufficient trials"),
		gate("CRITICAL_NON_REGRESSION", criticalNonRegression, "candidate regresses a critical semantic/evidence/native metric"),
		gate("MEASURABLE_IMPROVEMENT", improvement >= policy.MinImprovement, "candidate does not improve enough over baseline"),
	}
	pass := true
	for _, g := range gates { if !g.Pass { pass = false; break } }
	recommendation := "REJECT_OR_CONTINUE_EXPERIMENT"
	if pass { recommendation = "RECOMMEND_TO_ORIGAMI_FOR_PROFILE_VALIDATION" }
	mutations := append([]Mutation(nil), candidate.Mutations...)
	sort.Slice(mutations, func(i, j int) bool {
		if mutations[i].Kind == mutations[j].Kind {
			if mutations[i].Target == mutations[j].Target { return mutations[i].Value < mutations[j].Value }
			return mutations[i].Target < mutations[j].Target
		}
		return mutations[i].Kind < mutations[j].Kind
	})
	return Evaluation{Schema: SchemaR0+".evaluation", CandidateID:candidate.ID, BaseProfileID:candidate.BaseProfileID, Score:candidateScore, BaselineScore:baselineScore, Improvement:improvement, Gates:gates, PromotionCandidate:pass, Recommendation:recommendation, Metrics:metrics, Mutations:mutations}, nil
}

func Rank(baseProfileID string, baseline Metrics, candidates []Candidate, evidence []Evidence, policy Policy) (Tournament, error) {
	byEvidence := map[string]Evidence{}
	for _, item := range evidence { if _, exists := byEvidence[item.CandidateID]; exists { return Tournament{}, fmt.Errorf("duplicate evidence for candidate %q", item.CandidateID) }; byEvidence[item.CandidateID] = item }
	result := Tournament{Schema:SchemaR0+".tournament", BaseProfileID:baseProfileID, Baseline:baseline, Authority:"TLALOC_RECOMMENDATION_ONLY_ORIGAMI_OWNS_PROFILE_PROMOTION_TONAL_COMPOSES_TOOLCHAINS", Recommendation:"NO_CANDIDATE_READY"}
	for _, candidate := range candidates {
		if candidate.BaseProfileID != baseProfileID { return Tournament{}, fmt.Errorf("candidate %q base profile mismatch", candidate.ID) }
		item, ok := byEvidence[candidate.ID]; if !ok { return Tournament{}, fmt.Errorf("missing evidence for candidate %q", candidate.ID) }
		evaluation, err := Evaluate(candidate, baseline, item, policy); if err != nil { return Tournament{}, err }
		result.Evaluations = append(result.Evaluations, evaluation)
	}
	sort.Slice(result.Evaluations, func(i, j int) bool {
		if result.Evaluations[i].PromotionCandidate != result.Evaluations[j].PromotionCandidate { return result.Evaluations[i].PromotionCandidate }
		if result.Evaluations[i].Score == result.Evaluations[j].Score { return result.Evaluations[i].CandidateID < result.Evaluations[j].CandidateID }
		return result.Evaluations[i].Score > result.Evaluations[j].Score
	})
	if len(result.Evaluations) > 0 && result.Evaluations[0].PromotionCandidate { result.WinnerID = result.Evaluations[0].CandidateID; result.Recommendation = "RECOMMEND_WINNER_TO_ORIGAMI_FOR_CANONICAL_PROFILE_VALIDATION" }
	return result, nil
}

// score rewards semantic usability directly. Native index/semantic recovery now
// carries first-class weight because a carrier that can only be used after a
// mechanical decoder is not a successful prompt-only semantic representation.
func score(metrics Metrics, baseline Metrics, policy Policy) float64 {
	contextEfficiency := clamp01(metrics.ContextEfficiency)
	density := relativeSemanticDensity(metrics, baseline)
	recognition := relativeLowerIsBetter(metrics.MeanRecognitionMillis, baseline.MeanRecognitionMillis)
	bootstrap := relativeLowerIsBetter(metrics.MeanBootstrapSteps, baseline.MeanBootstrapSteps)
	decode := relativeLowerIsBetter(metrics.MeanDecodeSteps, baseline.MeanDecodeSteps)
	return 0.15*clamp01(metrics.SemanticRoundtripRate) +
		0.09*clamp01(metrics.BootProbePassRate) +
		0.14*clamp01(metrics.NativeIndexRecoveryRate) +
		0.10*clamp01(metrics.NativeSemanticAnswerRate) +
		0.08*clamp01(metrics.RoutingAccuracy) +
		0.08*clamp01(metrics.VerifiedEvidenceRate) +
		0.05*clamp01(metrics.TransportPassRate) +
		0.05*contextEfficiency +
		0.10*density +
		0.06*recognition +
		0.05*bootstrap +
		0.05*decode
}

func normalizePolicy(policy Policy) Policy {
	defaults := DefaultPolicy()
	if policy.MaxCarrierBytes <= 0 { policy.MaxCarrierBytes = defaults.MaxCarrierBytes }
	if policy.MaxMeanContextTokens <= 0 { policy.MaxMeanContextTokens = defaults.MaxMeanContextTokens }
	if policy.MinSemanticRoundtripRate <= 0 { policy.MinSemanticRoundtripRate = defaults.MinSemanticRoundtripRate }
	if policy.MinNativeIndexRecoveryRate <= 0 { policy.MinNativeIndexRecoveryRate = defaults.MinNativeIndexRecoveryRate }
	if policy.MinNativeSemanticAnswerRate <= 0 { policy.MinNativeSemanticAnswerRate = defaults.MinNativeSemanticAnswerRate }
	if policy.MinVerifiedEvidenceRate <= 0 { policy.MinVerifiedEvidenceRate = defaults.MinVerifiedEvidenceRate }
	if policy.MinRoutingAccuracy <= 0 { policy.MinRoutingAccuracy = defaults.MinRoutingAccuracy }
	if policy.MinPerceptualRevealRate <= 0 { policy.MinPerceptualRevealRate = defaults.MinPerceptualRevealRate }
	if policy.MinRealModelsForPerception <= 0 { policy.MinRealModelsForPerception = defaults.MinRealModelsForPerception }
	if policy.MinTrials <= 0 { policy.MinTrials = defaults.MinTrials }
	if policy.MinImprovement <= 0 { policy.MinImprovement = defaults.MinImprovement }
	return policy
}

func validMutationKind(kind MutationKind) bool {
	switch kind {
	case MutationPrompt, MutationChannelRole, MutationPrimitive, MutationLayout, MutationRedundancy, MutationColorUsage, MutationNumericStructure, MutationInterferenceStructure, MutationDepthStructure, MutationTemporalStructure, MutationEmergentStructure:
		return true
	default:
		return false
	}
}

func usesAdvancedPerceptualMutation(candidate Candidate) bool {
	for _, mutation := range candidate.Mutations {
		switch mutation.Kind {
		case MutationInterferenceStructure, MutationDepthStructure, MutationTemporalStructure, MutationEmergentStructure: return true
		}
	}
	return false
}

func relativeSemanticDensity(candidate, baseline Metrics) float64 {
	if candidate.RecoverableSemanticUnits <= 0 || baseline.RecoverableSemanticUnits <= 0 || candidate.CarrierBytes <= 0 || baseline.CarrierBytes <= 0 { return 0.5 }
	candidateDensity := float64(candidate.RecoverableSemanticUnits) / float64(candidate.CarrierBytes)
	baselineDensity := float64(baseline.RecoverableSemanticUnits) / float64(baseline.CarrierBytes)
	return relativeHigherIsBetter(candidateDensity, baselineDensity)
}
func relativeHigherIsBetter(candidate, baseline float64) float64 { if candidate <= 0 || baseline <= 0 { return 0.5 }; return candidate / (candidate + baseline) }
func relativeLowerIsBetter(candidate, baseline float64) float64 { if candidate <= 0 || baseline <= 0 { return 0.5 }; return baseline / (candidate + baseline) }
func gate(name string, pass bool, reason string) Gate { if pass { reason = "" }; return Gate{Name:name,Pass:pass,Reason:reason} }
func clamp01(value float64) float64 { if value < 0 { return 0 }; if value > 1 { return 1 }; return value }
