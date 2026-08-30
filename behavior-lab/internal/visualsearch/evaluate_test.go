package visualsearch

import "testing"

func baselineMetrics() Metrics {
	return Metrics{
		SemanticRoundtripRate: 1,
		BootProbePassRate: .90,
		NativeIndexRecoveryRate: .96,
		NativeSemanticAnswerRate: .92,
		RoutingAccuracy: .96,
		VerifiedEvidenceRate: .97,
		TransportPassRate: .80,
		PerceptualRevealRate: 1,
		ContextEfficiency: .70,
		MeanContextTokens: 1500,
		CarrierBytes: 8192,
		RecoverableSemanticUnits: 1000,
		MeanRecognitionMillis: 1200,
		MeanBootstrapSteps: 6,
		MeanDecodeSteps: 12,
		MechanicalDependencyViolations: 0,
		UnverifiedMechanicalClaims: 0,
		FalseExact: 0,
		BudgetViolations: 0,
		UnknownViolations: 0,
		RealModels: 3,
		Trials: 12,
	}
}

func candidate(id string, kind MutationKind, value string) Candidate {
	return Candidate{Schema:SchemaR0+".candidate",ID:id,BaseProfileID:"origami.canonical-aesthetic.r0",Mutations:[]Mutation{{Kind:kind,Target:"profile",Value:value,Experimental:true}}}
}

func improvedMetrics() Metrics {
	m := baselineMetrics()
	m.BootProbePassRate = .98
	m.NativeIndexRecoveryRate = .99
	m.NativeSemanticAnswerRate = .97
	m.TransportPassRate = .95
	m.ContextEfficiency = .80
	m.RecoverableSemanticUnits = 1200
	m.MeanRecognitionMillis = 900
	m.MeanBootstrapSteps = 5
	m.MeanDecodeSteps = 9
	return m
}

func TestNumericPrimeCandidateCanBeRecommendedOnlyFromEvidence(t *testing.T) {
	c := candidate("prime-layout", MutationNumericStructure, "PRIME_DERIVED_SPACING")
	eval, err := Evaluate(c, baselineMetrics(), Evidence{CandidateID:c.ID,Metrics:improvedMetrics()}, DefaultPolicy()); if err != nil { t.Fatal(err) }
	if !eval.PromotionCandidate || eval.Recommendation != "RECOMMEND_TO_ORIGAMI_FOR_PROFILE_VALIDATION" { t.Fatalf("evidence-backed numeric candidate should be recommendable: %+v", eval) }
}

func TestInterestingPatternWithoutRealModelsCannotPromote(t *testing.T) {
	c := candidate("prime-pretty", MutationNumericStructure, "PRIME_DERIVED_SPACING")
	m := improvedMetrics(); m.RealModels=0; m.Trials=100
	eval, err := Evaluate(c, baselineMetrics(), Evidence{CandidateID:c.ID,Metrics:m}, DefaultPolicy()); if err != nil { t.Fatal(err) }
	if eval.PromotionCandidate { t.Fatalf("visual pattern promoted without real-model evidence: %+v", eval) }
}

func TestFalseExactBlocksAnyAestheticCandidate(t *testing.T) {
	c := candidate("color", MutationColorUsage, "COLOR_FOR_STATE_REDUNDANCY")
	m := improvedMetrics(); m.FalseExact=1
	eval, err := Evaluate(c, baselineMetrics(), Evidence{CandidateID:c.ID,Metrics:m}, DefaultPolicy()); if err != nil { t.Fatal(err) }
	if eval.PromotionCandidate { t.Fatalf("FALSE_EXACT regression promoted: %+v", eval) }
}

func TestNativeIndexFailureBlocksCandidate(t *testing.T) {
	c := candidate("pretty-but-unreadable", MutationLayout, "BINARY_FIRST")
	m := improvedMetrics(); m.NativeIndexRecoveryRate=.50
	eval, err := Evaluate(c, baselineMetrics(), Evidence{CandidateID:c.ID,Metrics:m}, DefaultPolicy()); if err != nil { t.Fatal(err) }
	if eval.PromotionCandidate { t.Fatalf("candidate that fails T2 index recovery promoted: %+v", eval) }
}

func TestMechanicalDependencyBlocksCandidate(t *testing.T) {
	c := candidate("decoder-dependent", MutationPrompt, "DECODE_PIXELS_FIRST")
	m := improvedMetrics(); m.MechanicalDependencyViolations=1
	eval, err := Evaluate(c, baselineMetrics(), Evidence{CandidateID:c.ID,Metrics:m}, DefaultPolicy()); if err != nil { t.Fatal(err) }
	if eval.PromotionCandidate { t.Fatalf("semantic navigation requiring mechanical decoder promoted: %+v", eval) }
}

func TestUnverifiedByteClaimsBlockCandidate(t *testing.T) {
	c := candidate("hallucinated-exact", MutationPrompt, "GUESS_ARCHIVE_METADATA")
	m := improvedMetrics(); m.UnverifiedMechanicalClaims=2
	eval, err := Evaluate(c, baselineMetrics(), Evidence{CandidateID:c.ID,Metrics:m}, DefaultPolicy()); if err != nil { t.Fatal(err) }
	if eval.PromotionCandidate { t.Fatalf("candidate with unverified mechanical claims promoted: %+v", eval) }
}

func TestSemanticRoundtripRegressionBlocksCandidateEvenWithBetterPerception(t *testing.T) {
	c := candidate("layout", MutationLayout, "RADIAL_HIERARCHY")
	m := improvedMetrics(); m.SemanticRoundtripRate=.99; m.BootProbePassRate=1; m.TransportPassRate=1
	eval, err := Evaluate(c, baselineMetrics(), Evidence{CandidateID:c.ID,Metrics:m}, DefaultPolicy()); if err != nil { t.Fatal(err) }
	if eval.PromotionCandidate { t.Fatalf("semantic regression promoted: %+v", eval) }
}

func TestMutationMustRemainExperimentalBeforePromotion(t *testing.T) {
	c := candidate("bad", MutationPrimitive, "NEW_SHAPE"); c.Mutations[0].Experimental=false
	_, err := Evaluate(c, baselineMetrics(), Evidence{CandidateID:c.ID,Metrics:improvedMetrics()}, DefaultPolicy()); if err == nil { t.Fatal("candidate claimed canonical status before Origami promotion") }
}

func TestTournamentRanksOnlyGatePassingCandidates(t *testing.T) {
	good:=candidate("good",MutationRedundancy,"TRIPLE_PROBE"); bad:=candidate("bad",MutationColorUsage,"COLOR_ONLY_STATE")
	goodM:=improvedMetrics(); badM:=improvedMetrics(); badM.UnknownViolations=1
	report,err:=Rank("origami.canonical-aesthetic.r0",baselineMetrics(),[]Candidate{bad,good},[]Evidence{{CandidateID:bad.ID,Metrics:badM},{CandidateID:good.ID,Metrics:goodM}},DefaultPolicy());if err!=nil{t.Fatal(err)}
	if report.WinnerID!="good"||report.Recommendation!="RECOMMEND_WINNER_TO_ORIGAMI_FOR_CANONICAL_PROFILE_VALIDATION"{t.Fatalf("unexpected tournament: %+v",report)}
}

func TestMoireCandidateNeedsMeasuredReveal(t *testing.T) {
	c:=candidate("moire-depth",MutationInterferenceStructure,"MOIRE_PHASE_RELATION");m:=improvedMetrics();m.PerceptualRevealRate=.70
	eval,err:=Evaluate(c,baselineMetrics(),Evidence{CandidateID:c.ID,Metrics:m},DefaultPolicy());if err!=nil{t.Fatal(err)};if eval.PromotionCandidate{t.Fatalf("moire candidate promoted without reliable reveal: %+v",eval)}
}

func TestStereoCandidateCanWinWhenRevealAndSemanticsHold(t *testing.T) {
	c:=candidate("stereo",MutationDepthStructure,"STEREO_BIND_DEPTH");m:=improvedMetrics();m.PerceptualRevealRate=.98
	eval,err:=Evaluate(c,baselineMetrics(),Evidence{CandidateID:c.ID,Metrics:m},DefaultPolicy());if err!=nil{t.Fatal(err)};if !eval.PromotionCandidate{t.Fatalf("evidence-backed depth candidate should be recommendable: %+v",eval)}
}

func TestOptimizationCanComeFromSmallerFasterFewerSteps(t *testing.T) {
	c:=candidate("compact",MutationLayout,"COMPACT_ROUTE");m:=baselineMetrics();m.CarrierBytes=6144;m.MeanRecognitionMillis=700;m.MeanBootstrapSteps=4;m.MeanDecodeSteps=7;m.RecoverableSemanticUnits=1100;m.NativeIndexRecoveryRate=.98;m.NativeSemanticAnswerRate=.95
	eval,err:=Evaluate(c,baselineMetrics(),Evidence{CandidateID:c.ID,Metrics:m},DefaultPolicy());if err!=nil{t.Fatal(err)};if !eval.PromotionCandidate{t.Fatalf("measured size/speed/decode improvement should count: %+v",eval)}
}
