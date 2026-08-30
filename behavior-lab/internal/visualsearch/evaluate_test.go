package visualsearch

import "testing"

func baselineMetrics() Metrics {
	return Metrics{
		SemanticRoundtripRate: 1,
		BootProbePassRate: 0.90,
		RoutingAccuracy: 0.96,
		VerifiedEvidenceRate: 0.97,
		TransportPassRate: 0.80,
		ContextEfficiency: 0.70,
		MeanContextTokens: 1500,
		CarrierBytes: 8192,
		FalseExact: 0,
		BudgetViolations: 0,
		UnknownViolations: 0,
		RealModels: 3,
		Trials: 12,
	}
}

func candidate(id string, kind MutationKind, value string) Candidate {
	return Candidate{
		Schema: SchemaR0 + ".candidate",
		ID: id,
		BaseProfileID: "origami.canonical-aesthetic.r0",
		Mutations: []Mutation{{Kind: kind, Target: "profile", Value: value, Experimental: true}},
	}
}

func improvedMetrics() Metrics {
	m := baselineMetrics()
	m.BootProbePassRate = .98
	m.TransportPassRate = .95
	m.ContextEfficiency = .80
	return m
}

func TestNumericPrimeCandidateCanBeRecommendedOnlyFromEvidence(t *testing.T) {
	c := candidate("prime-layout", MutationNumericStructure, "PRIME_DERIVED_SPACING")
	eval, err := Evaluate(c, baselineMetrics(), Evidence{CandidateID: c.ID, Metrics: improvedMetrics()}, DefaultPolicy())
	if err != nil { t.Fatal(err) }
	if !eval.PromotionCandidate || eval.Recommendation != "RECOMMEND_TO_ORIGAMI_FOR_PROFILE_VALIDATION" {
		t.Fatalf("evidence-backed numeric candidate should be recommendable: %+v", eval)
	}
}

func TestInterestingPatternWithoutRealModelsCannotPromote(t *testing.T) {
	c := candidate("prime-pretty", MutationNumericStructure, "PRIME_DERIVED_SPACING")
	m := improvedMetrics(); m.RealModels = 0; m.Trials = 100
	eval, err := Evaluate(c, baselineMetrics(), Evidence{CandidateID: c.ID, Metrics: m}, DefaultPolicy())
	if err != nil { t.Fatal(err) }
	if eval.PromotionCandidate { t.Fatalf("visual pattern promoted without real-model evidence: %+v", eval) }
}

func TestFalseExactBlocksAnyAestheticCandidate(t *testing.T) {
	c := candidate("color", MutationColorUsage, "COLOR_FOR_STATE_REDUNDANCY")
	m := improvedMetrics(); m.FalseExact = 1
	eval, err := Evaluate(c, baselineMetrics(), Evidence{CandidateID: c.ID, Metrics: m}, DefaultPolicy())
	if err != nil { t.Fatal(err) }
	if eval.PromotionCandidate { t.Fatalf("FALSE_EXACT regression promoted: %+v", eval) }
}

func TestSemanticRoundtripRegressionBlocksCandidateEvenWithBetterPerception(t *testing.T) {
	c := candidate("layout", MutationLayout, "RADIAL_HIERARCHY")
	m := improvedMetrics(); m.SemanticRoundtripRate = .99; m.BootProbePassRate = 1; m.TransportPassRate = 1
	eval, err := Evaluate(c, baselineMetrics(), Evidence{CandidateID: c.ID, Metrics: m}, DefaultPolicy())
	if err != nil { t.Fatal(err) }
	if eval.PromotionCandidate { t.Fatalf("semantic regression promoted: %+v", eval) }
}

func TestMutationMustRemainExperimentalBeforePromotion(t *testing.T) {
	c := candidate("bad", MutationPrimitive, "NEW_SHAPE")
	c.Mutations[0].Experimental = false
	_, err := Evaluate(c, baselineMetrics(), Evidence{CandidateID: c.ID, Metrics: improvedMetrics()}, DefaultPolicy())
	if err == nil { t.Fatal("candidate claimed canonical status before Origami/Tonal promotion") }
}

func TestTournamentRanksOnlyGatePassingCandidates(t *testing.T) {
	good := candidate("good", MutationRedundancy, "TRIPLE_PROBE")
	bad := candidate("bad", MutationColorUsage, "COLOR_ONLY_STATE")
	goodM := improvedMetrics()
	badM := improvedMetrics(); badM.UnknownViolations = 1
	report, err := Rank("origami.canonical-aesthetic.r0", baselineMetrics(), []Candidate{bad, good}, []Evidence{{CandidateID: bad.ID, Metrics: badM}, {CandidateID: good.ID, Metrics: goodM}}, DefaultPolicy())
	if err != nil { t.Fatal(err) }
	if report.WinnerID != "good" || report.Recommendation != "RECOMMEND_WINNER_TO_ORIGAMI_FOR_CANONICAL_PROFILE_VALIDATION" {
		t.Fatalf("unexpected tournament: %+v", report)
	}
	if report.Authority != "TLALOC_RECOMMENDATION_ONLY_ORIGAMI_VALIDATES_TONAL_PROMOTES" {
		t.Fatalf("authority boundary lost: %s", report.Authority)
	}
}
