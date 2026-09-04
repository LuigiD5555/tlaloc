package exocortex

import (
	"strings"
	"testing"
)

func minimalValidProfileR1() CapabilityProfileR1 {
	ev := func(cls string) EvidenceRef {
		return EvidenceRef{
			SourceExperiment: "R1-X", ArtifactPath: "results/x.json",
			ArtifactSHA256: strings.Repeat("a", 64), SampleSize: 20,
			EvidenceClass: cls, MeasuredMetric: "m", Limitations: "l",
		}
	}
	evZero := func(cls string) EvidenceRef {
		e := ev(cls)
		e.SampleSize = 0
		return e
	}
	p := CapabilityProfileR1{
		Schema: CapabilityProfileR1Schema, ProfileID: "x@r1.0.0", ProfileVersion: "r1.0.0",
		PreservesR0: "R0 preserved and not overwritten",
		Executor: ExecutorIdentityR1{
			ModelID: "lfm2-vl-1.6b", Architecture: "lfm2",
			WeightsGGUFSHA: strings.Repeat("b", 64), MMProjGGUFSHA: strings.Repeat("c", 64),
			IdentitySHA256: strings.Repeat("d", 64),
		},
		GlobalRules: GlobalExecutorRules{
			MaxCognitiveTransformationsPerCall: 1, Rule: "one op",
			FormattingIsExternalDeterministic: true, SequenceWorkingMemoryIsTlaloc: true, ModelResultIsObservationNotFact: true,
			Evidence: ev(EvidenceEarned),
		},
		ExtractNumber: ExtractNumberProfile{
			ScaleRungs: []ScaleRung{{8, 0.27, "TOO_SMALL"}, {12, 0.87, "TRANSITION"}, {16, 0.90, "OPERATING_REGION"}, {24, 0.93, "OPERATING_REGION"}, {32, 0.97, "OPERATING_REGION"}, {48, 0.97, "OPERATING_REGION"}},
			FormalSafeScalePx: 16, PreferredScalePx: 32, ObservedOperatingRegionPx: [2]int{16, 48},
			Scope: "scoped to the tested LFM2-VL runtime", Evidence: ev(EvidenceEarned),
		},
		Context: ContextProfile{
			NaturalVisualField:     ContextEnvelope{Name: "n", Points: map[string]float64{"a": 0.8}},
			FixedScaleLocalContext: ContextEnvelope{Name: "f", Points: map[string]float64{"a": 1.0}},
			AllowedConclusion:      "scale control reduces collapse",
			ForbiddenDecomposition: "no exact causal split",
			RuntimePreference:      "minimum sufficient working set",
			AggressiveReductionClass: EvidencePreventivePractice,
			Evidence:               []EvidenceRef{ev(EvidenceObservedExploratory), ev(EvidencePreventivePractice)},
		},
		ReadAssociated: ReadAssociatedNumberProfile{
			Opcode: "READ_ASSOCIATED_NUMBER", VisualDependence: true, Verdict: "USABLE_WITH_CONSTRAINTS",
			ObservedExploratoryExitK: 1, FormalMaxDistractors: "UNKNOWN",
			CompetitorRemovalProvenance: "R1-D intervention reuse, not a fresh independent estimate",
			EvidenceRefs:                []EvidenceRef{ev(EvidenceEarned)},
		},
		Morphology:         []MorphologyFamilyProfile{{Family: "MULTI_DIGIT_INTEGER", ProvisionalVerdict: "USABLE_WITH_CONSTRAINTS", NeverPooled: true}},
		MorphologyEvidence: ev(EvidenceObservedExploratory),
		Repeatability: RepeatabilityProfile{
			Sentinels: 20, Repeats: 5, ByteStable: "20/20", SemanticStable: "20/20",
			RuntimeRule: "DO_NOT_RETRY_IDENTICAL_INPUT", EvidenceScope: "temperature 0, byte-identical input",
			Evidence: ev(EvidenceEarned),
		},
		RecoveryRules: []RecoveryRule{
			{ID: "LOW_SCALE", DetectIf: "line height < 16 px", PreferredAction: "upscale", Classification: "PREVENTIVE+EARNED", ModelCallsAfter: -1, Evidence: ev(EvidenceEarned)},
			{ID: "NUMERIC_COMPETITORS", DetectIf: "competing numeric operands detected", PreferredAction: "isolate", Classification: "PREVENTIVE+EARNED", ModelCallsAfter: -1, Evidence: ev(EvidenceEarned)},
			{ID: "HIGH_CONTEXT", DetectIf: "field larger than operand line", PreferredAction: "crop", Classification: "PREVENTIVE_PRACTICE", ModelCallsAfter: -1, Evidence: ev(EvidenceNoMeasuredBenefit)},
			{ID: "VALUE_CUE", DetectIf: "cue tighter than padded rule", PreferredAction: "pad", Classification: "SAFE_ENGINEERING_DEFAULT", ModelCallsAfter: -1, Evidence: ev(EvidenceSafeEngineeringDefault)},
			{ID: "MISSING_VISUAL_OPERAND", DetectIf: "visual opcode with no visual operand", PreferredAction: "reject", Classification: "REJECT_BEFORE_CALL", ModelCallsAfter: 0, Evidence: evZero(EvidenceRejectBeforeCall)},
			{ID: "EXACT_RETRY", DetectIf: "input byte-identical to a prior failed call", PreferredAction: "reject", Classification: "REJECT", ModelCallsAfter: 0, Evidence: evZero(EvidenceDoNotUse)},
			{ID: "CURRENT_TESSERACT_OCR", DetectIf: "caller proposes tesseract OCR", PreferredAction: "do not route", Classification: "DO_NOT_USE", ModelCallsAfter: 0, Evidence: evZero(EvidenceDoNotUse)},
		},
	}
	for i := range p.SourceExperiments {
		_ = i
	}
	for i := 0; i < 12; i++ {
		p.SourceExperiments = append(p.SourceExperiments, SourceExperimentRef{ID: "exp", ArtifactSHA256: strings.Repeat("e", 64), Frozen: true})
	}
	return p
}

func TestProfileR1_ValidAndHashStable(t *testing.T) {
	p := minimalValidProfileR1()
	if err := ValidateCapabilityProfileR1(p); err != nil {
		t.Fatalf("minimal profile should validate: %v", err)
	}
	h1, err := ComputeProfileR1Hash(p)
	if err != nil {
		t.Fatal(err)
	}
	h2, _ := ComputeProfileR1Hash(p)
	if h1 != h2 || len(h1) != 64 {
		t.Fatalf("hash unstable/short: %q %q", h1, h2)
	}
}

func TestProfileR1_RejectsBadInputs(t *testing.T) {
	cases := map[string]func(*CapabilityProfileR1){
		"exploratory promoted to formal max": func(p *CapabilityProfileR1) { p.ReadAssociated.FormalMaxDistractors = "1" },
		"high context marked earned":         func(p *CapabilityProfileR1) { p.RecoveryRules[2].Evidence.EvidenceClass = EvidenceEarned },
		"low scale not earned":               func(p *CapabilityProfileR1) { p.RecoveryRules[0].Classification = "PROMISING" },
		"exact retry with calls":             func(p *CapabilityProfileR1) { p.RecoveryRules[5].ModelCallsAfter = 1 },
		"missing operand with calls":         func(p *CapabilityProfileR1) { p.RecoveryRules[4].ModelCallsAfter = 1 },
		"global ops != 1":                    func(p *CapabilityProfileR1) { p.GlobalRules.MaxCognitiveTransformationsPerCall = 2 },
		"detect_if references gold":          func(p *CapabilityProfileR1) { p.RecoveryRules[0].DetectIf = "expected answer is known" },
		"impossible accuracy":                func(p *CapabilityProfileR1) { p.ExtractNumber.ScaleRungs[0].Accuracy = 1.5 },
		"safe scale > preferred":             func(p *CapabilityProfileR1) { p.ExtractNumber.PreferredScalePx = 8 },
		"morphology pooled":                  func(p *CapabilityProfileR1) { p.Morphology[0].NeverPooled = false },
		"missing source hash":                func(p *CapabilityProfileR1) { p.SourceExperiments[0].ArtifactSHA256 = "short" },
		"unfrozen source":                    func(p *CapabilityProfileR1) { p.SourceExperiments[0].Frozen = false },
	}
	for name, mut := range cases {
		p := minimalValidProfileR1()
		mut(&p)
		if err := ValidateCapabilityProfileR1(p); err == nil {
			t.Errorf("%s: expected validation failure, got nil", name)
		}
	}
}

func TestAdapterR1_PreventiveTransformsAndRejections(t *testing.T) {
	p := minimalValidProfileR1()
	p.ProfileVersion = "r1.0.0"
	a := AdapterR1{Profile: p}

	// low scale -> upscale
	d, err := a.Prepare(AdaptRequestR1{Opcode: OpExtractNumber, HasVisualOperand: true, LineHeightPx: 8, VisualFieldName: "LINE"})
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(d.RulesApplied, "LOW_SCALE") || d.ModelCallCount != 1 {
		t.Errorf("low scale: rules=%v calls=%d", d.RulesApplied, d.ModelCallCount)
	}
	if v, _ := d.ResultingWorkingSet["target_line_height_px"].(float64); v != 32 {
		t.Errorf("low scale should upscale to 32, got %v", d.ResultingWorkingSet["target_line_height_px"])
	}

	// already good -> no transform
	d, _ = a.Prepare(AdaptRequestR1{Opcode: OpExtractNumber, HasVisualOperand: true, LineHeightPx: 32, VisualFieldName: "LINE"})
	if len(d.Transformations) != 0 {
		t.Errorf("already-good input should get no transforms, got %v", d.Transformations)
	}

	// high field -> crop
	d, _ = a.Prepare(AdaptRequestR1{Opcode: OpExtractNumber, HasVisualOperand: true, LineHeightPx: 32, VisualFieldName: "LOCAL_256"})
	if !containsStr(d.RulesApplied, "HIGH_CONTEXT") {
		t.Errorf("high field should apply HIGH_CONTEXT, got %v", d.RulesApplied)
	}

	// missing operand -> reject, zero calls
	d, _ = a.Prepare(AdaptRequestR1{Opcode: OpExtractNumber, HasVisualOperand: false})
	if !d.Rejected || d.ModelCallCount != 0 || d.FallbackAction != "UNSUPPORTED" {
		t.Errorf("missing operand: rejected=%v calls=%d fallback=%q", d.Rejected, d.ModelCallCount, d.FallbackAction)
	}

	// identical retry -> reject, zero calls
	d, _ = a.Prepare(AdaptRequestR1{Opcode: OpExtractNumber, HasVisualOperand: true, LineHeightPx: 32, IdenticalToAPriorFailedCall: true})
	if !d.Rejected || d.ModelCallCount != 0 {
		t.Errorf("identical retry should be rejected with 0 calls")
	}

	// unknown opcode -> reject
	d, _ = a.Prepare(AdaptRequestR1{Opcode: "SELECT_ONE", HasVisualOperand: true})
	if !d.Rejected {
		t.Errorf("opcode with no R1 evidence should be rejected")
	}
}

func containsStr(xs []string, w string) bool {
	for _, x := range xs {
		if x == w {
			return true
		}
	}
	return false
}
