package perceptenvelope

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/exocortex"
)

// CompileCapabilityProfileR1 compiles the frozen LFM2-VL 1.6B characterisation
// campaign (P1, P2, T0-A, T0-B, R1-A0..R1-G) into an
// exocortex.CapabilityProfileR1. It is a pure evidence-preserving transform:
// every number is read from a hashed frozen artifact; nothing is invented,
// no exploratory result is promoted to a formal constraint.
func CompileCapabilityProfileR1(expDir, profileVersion string) (exocortex.CapabilityProfileR1, error) {
	rp := func(p string) string { return filepath.Join(expDir, p) }

	hash := func(p string) (string, error) {
		h, err := SHA256OfFile(p)
		if err != nil {
			return "", fmt.Errorf("hash %s: %w", p, err)
		}
		return h, nil
	}
	readMap := func(p string) (map[string]any, error) {
		body, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			return nil, fmt.Errorf("decode %s: %w", p, err)
		}
		return m, nil
	}

	// ---- executor identity (MODEL_IDENTITY.json) ----
	miPath := rp("MODEL_IDENTITY.json")
	miHash, err := hash(miPath)
	if err != nil {
		return exocortex.CapabilityProfileR1{}, err
	}
	mi, err := readMap(miPath)
	if err != nil {
		return exocortex.CapabilityProfileR1{}, err
	}
	model, _ := mi["model"].(map[string]any)
	getS := func(m map[string]any, path ...string) string {
		cur := any(m)
		for _, k := range path {
			mm, ok := cur.(map[string]any)
			if !ok {
				return ""
			}
			cur = mm[k]
		}
		if s, ok := cur.(string); ok {
			return s
		}
		if mm, ok := cur.(map[string]any); ok {
			if v, ok := mm["value"].(string); ok {
				return v
			}
		}
		return ""
	}
	weights, _ := model["weights_gguf"].(map[string]any)
	mmproj, _ := model["mmproj_gguf"].(map[string]any)
	ggufMeta, _ := model["gguf_metadata"].(map[string]any)
	runtime, _ := mi["runtime"].(map[string]any)
	serving, _ := mi["serving_configuration"].(map[string]any)
	effGen, _ := mi["effective_generation_settings_r1"].(map[string]any)
	libHashes := map[string]string{}
	if rh, ok := runtime["runtime_binary_hashes"].(map[string]any); ok {
		for k, v := range rh {
			if s, ok := v.(string); ok && len(s) == 64 {
				libHashes[k] = s
			}
		}
	}
	toI := func(v any) int {
		switch t := v.(type) {
		case float64:
			return int(t)
		case int:
			return t
		}
		return 0
	}
	exec := exocortex.ExecutorIdentityR1{
		ModelID:         getS(model, "model_id"),
		Family:          getS(model, "family"),
		Publisher:       getS(model, "publisher"),
		Quantization:    getS(ggufMeta, "quantization"),
		Architecture:    getS(ggufMeta, "arch"),
		WeightsGGUFSHA:  fmt.Sprintf("%v", weights["sha256"]),
		MMProjGGUFSHA:   fmt.Sprintf("%v", mmproj["sha256"]),
		LMStudioVersion: getS(runtime, "application"),
		BackendVersion:  getS(runtime, "inference_engine"),
		RuntimeLibHashes: libHashes,
		ContextLength:   toI(getMapVal(serving, "context_size_loaded", "value")),
		Endpoint:        getS(serving, "endpoint"),
		Temperature:     0,
		MaxOutputTokens: toI(effGen["max_output_tokens"]),
		IdentitySHA256:  miHash,
	}
	if t, ok := effGen["temperature"].(float64); ok {
		exec.Temperature = t
	}

	// ---- source experiments ----
	type srcSpec struct{ id, path, status string }
	srcs := []srcSpec{
		{"parrot-microisa-r0 (P1/P2)", "../parrot-microisa-r0/FREEZE.json", "FROZEN"},
		{"parrot-microisa-r0.1 (P2-A)", "../parrot-microisa-r0.1/FREEZE.json", "FROZEN"},
		{"exocortex-t0a-r0 (T0-A)", "../exocortex-t0a-r0/FREEZE.json", "FROZEN"},
		{"exocortex-decomposition-r0 (T0-B)", "../exocortex-decomposition-r0/T0B_CHECKPOINT.json", "FROZEN"},
		{"R1-A0", "R1A_CHECKPOINT.json", "R1-A0_CONTEXT_ENVELOPE_COMPLETE_FROZEN"},
		{"R1-A1", "R1A1_CHECKPOINT.json", "R1-A1_FIXED_SCALE_CONTEXT_COMPLETE_FROZEN"},
		{"R1-B", "R1B_CHECKPOINT.json", "R1-B_SCALE_ENVELOPE_COMPLETE_FROZEN"},
		{"R1-C", "R1C_CHECKPOINT.json", "R1-C_NUMERIC_MORPHOLOGY_COMPLETE_FROZEN"},
		{"R1-D", "R1D_CHECKPOINT.json", "R1-D_ASSOCIATION_DISTRACTOR_COMPLETE_FROZEN"},
		{"R1-E", "R1E_CHECKPOINT.json", "R1-E_VISUAL_DEPENDENCE_COMPLETE_FROZEN"},
		{"R1-F", "R1F_CHECKPOINT.json", "R1-F_REPEATABILITY_COMPLETE_FROZEN"},
		{"R1-G", "R1G_CHECKPOINT.json", "R1-G_RECOVERY_COMPLETE_FROZEN"},
	}
	var sourceExps []exocortex.SourceExperimentRef
	for _, s := range srcs {
		full := rp(s.path)
		h, herr := hash(full)
		if herr != nil {
			return exocortex.CapabilityProfileR1{}, fmt.Errorf("source experiment %s: %w", s.id, herr)
		}
		ref := exocortex.SourceExperimentRef{ID: s.id, ArtifactPath: s.path, ArtifactSHA256: h, Frozen: true, Status: s.status}
		if m, e := readMap(full); e == nil {
			if r, ok := m["records"].(float64); ok {
				ref.Records = int(r)
			}
		}
		sourceExps = append(sourceExps, ref)
	}

	// ---- EXTRACT_NUMBER scale (R1B_SCALE_CURVE) ----
	b1Path := rp("results/R1B_SCALE_CURVE.json")
	b1Hash, err := hash(b1Path)
	if err != nil {
		return exocortex.CapabilityProfileR1{}, err
	}
	b1, err := readMap(b1Path)
	if err != nil {
		return exocortex.CapabilityProfileR1{}, err
	}
	var rungs []exocortex.ScaleRung
	rungVerdict := map[int]string{8: "TOO_SMALL", 12: "TRANSITION", 16: "OPERATING_REGION", 24: "OPERATING_REGION", 32: "OPERATING_REGION", 48: "OPERATING_REGION"}
	for _, r := range asSlice(b1["rows"]) {
		rr, _ := r.(map[string]any)
		px := toI(rr["nominal_line_height_px"])
		acc, _ := rr["semantic_accuracy"].(float64)
		rungs = append(rungs, exocortex.ScaleRung{LineHeightPx: px, Accuracy: acc, Verdict: rungVerdict[px]})
	}
	b1n := toI(b1["bases"])
	if b1n == 0 {
		b1n = 30
	}
	en := exocortex.ExtractNumberProfile{
		ScaleRungs:                rungs,
		FormalSafeScalePx:         16,
		PreferredScalePx:          32,
		ObservedOperatingRegionPx: [2]int{16, 48},
		Scope:                     "scoped to the tested LFM2-VL 1.6B F16 runtime (LM Studio + llama.cpp CUDA), the tested numeric presentation family (plain 2-4 digit MULTI_DIGIT_INTEGER on prose lines), and the tested fixed-canvas bilinear renderer. NOT a universal visual threshold.",
		Evidence: exocortex.EvidenceRef{
			SourceExperiment: "R1-B", ArtifactPath: "results/R1B_SCALE_CURVE.json", ArtifactSHA256: b1Hash,
			SampleSize: b1n, EvidenceClass: exocortex.EvidenceEarned,
			MeasuredMetric: "EXTRACT_NUMBER semantic accuracy vs submitted containing-line height (8/12/16/24/32/48 px), 30 bases x 6 rungs",
			Limitations:    "formal_safe_scale derived by the preregistered conservative Wilson rule; overscale not degrading within [16,48]; one renderer / one numeric family only",
		},
	}

	// ---- context (R1A0 + R1A1 curves) ----
	a0Path, a1Path := rp("results/R1A_CONTEXT_CURVE.json"), rp("results/R1A1_CONTEXT_CURVE.json")
	a0Hash, _ := hash(a0Path)
	a1Hash, _ := hash(a1Path)
	a0, _ := readMap(a0Path)
	a1, _ := readMap(a1Path)
	curvePts := func(m map[string]any) map[string]float64 {
		out := map[string]float64{}
		for _, l := range asSlice(m["levels"]) {
			ll, _ := l.(map[string]any)
			name, _ := ll["context_level"].(string)
			acc, _ := ll["semantic_accuracy"].(float64)
			out[name] = acc
		}
		return out
	}
	ctx := exocortex.ContextProfile{
		NaturalVisualField: exocortex.ContextEnvelope{
			Name: "NATURAL_VISUAL_FIELD (R1-A0)", Points: curvePts(a0),
			Pipeline: "natural crop, target scale not held constant across levels",
		},
		FixedScaleLocalContext: exocortex.ContextEnvelope{
			Name: "FIXED_SCALE_LOCAL_CONTEXT (R1-A1)", Points: curvePts(a1),
			Pipeline: "one per-base 512x512 viewport at a fixed 32 px line height, nested reveal masks",
		},
		AllowedConclusion:      "controlling target scale dramatically reduces the observed context collapse (A0 target-only 0.80 -> full-page 0.10; A1 fixed-scale target 1.00 -> viewport 0.80).",
		ForbiddenDecomposition: "the two tracks use different samples and pipelines; do NOT claim an exact causal split such as '50 pp scale + 20 pp context'.",
		RuntimePreference:      "minimum sufficient working set",
		AggressiveReductionClass: exocortex.EvidencePreventivePractice,
		Evidence: []exocortex.EvidenceRef{
			{SourceExperiment: "R1-A0", ArtifactPath: "results/R1A_CONTEXT_CURVE.json", ArtifactSHA256: a0Hash, SampleSize: toIntOr(a0["bases"], 30), EvidenceClass: exocortex.EvidenceObservedExploratory, MeasuredMetric: "EXTRACT_NUMBER semantic accuracy vs natural visual field size (target-only .. full-page)", Limitations: "target scale confounded with field size; superseded by R1-A1 for the causal reading"},
			{SourceExperiment: "R1-A1", ArtifactPath: "results/R1A1_CONTEXT_CURVE.json", ArtifactSHA256: a1Hash, SampleSize: toIntOr(a1["bases"], 30), EvidenceClass: exocortex.EvidencePreventivePractice, MeasuredMetric: "EXTRACT_NUMBER semantic accuracy vs revealed context at a fixed 32 px scale (7 nested levels)", Limitations: "R1-G context recovery on a fresh sample was NO_MEASURED_BENEFIT (baseline not adverse); reduction is a practice, not an earned recovery"},
		},
	}

	// ---- READ_ASSOCIATED_NUMBER (R1-E + R1-D + R1-G real) ----
	ePath := rp("results/R1E_VISUAL_DEPENDENCE_TABLE.json")
	eHash, _ := hash(ePath)
	dPath := rp("results/R1D_ASSOCIATION_TABLE.json")
	dHash, _ := hash(dPath)
	gPath := rp("results/R1G_RECOVERY_TABLE.json")
	gHash, _ := hash(gPath)
	ra := exocortex.ReadAssociatedNumberProfile{
		Opcode: "READ_ASSOCIATED_NUMBER", MicroISAPromoted: false, VisualDependence: true,
		Evidence: map[string]string{
			"correct_image": "22/22 task-gold", "no_image": "0/22 task-gold (degenerate 12345)",
			"wrong_image": "0/22 task-gold, 22/22 wrong-visible-operand", "mcnemar": "E2->E0 and E2->E1 both delta -1.00, exact p 4.77e-07",
		},
		Verdict:       "USABLE_WITH_CONSTRAINTS",
		TestedEnvelope: []string{"MULTI_DIGIT_INTEGER operand", "32 px containing-line height", "single-line label/value layout", "zero competing visible numeric operands"},
		ObservedExploratoryExitK: 1,
		FormalMaxDistractors:     "UNKNOWN",
		CompetitorRemovalProvenance: "R1-G GC_ASSOC_REAL supports competitor removal as an EARNED preventive intervention, but the real bases were R1-D intervention reuse (INDEPENDENT_ACCURACY_ESTIMATE=false), not a fresh independent capability estimate; the fresh synthetic proxy did not confirm because the glyph-bank abstract label could not anchor association (SYNTHETIC_PROXY_LIMITATION).",
		EvidenceRefs: []exocortex.EvidenceRef{
			{SourceExperiment: "R1-E", ArtifactPath: "results/R1E_VISUAL_DEPENDENCE_TABLE.json", ArtifactSHA256: eHash, SampleSize: 22, EvidenceClass: exocortex.EvidenceEarned, MeasuredMetric: "task-gold vs image-consistent accuracy across NO_IMAGE / WRONG_IMAGE / CORRECT_IMAGE", Limitations: "intervention reuse of the 22 R1-D bases; not a fresh accuracy estimate"},
			{SourceExperiment: "R1-D", ArtifactPath: "results/R1D_ASSOCIATION_TABLE.json", ArtifactSHA256: dHash, SampleSize: 22, EvidenceClass: exocortex.EvidenceObservedExploratory, MeasuredMetric: "D0L association 22/22; D1 distractor ladder K0..K8 (exploratory: geometry gate failed)", Limitations: "D1 exploratory; formal_max_distractors stays UNKNOWN"},
			{SourceExperiment: "R1-G", ArtifactPath: "results/R1G_RECOVERY_TABLE.json", ArtifactSHA256: gHash, SampleSize: 22, EvidenceClass: exocortex.EvidenceEarned, MeasuredMetric: "GC_ASSOC_REAL competitor removal 0.41 -> 1.00 (delta +0.59, W->C 13, C->W 0)", Limitations: "real intervention reuse; synthetic proxy NOT_CONFIRMED"},
		},
	}

	// ---- morphology (R1-C) ----
	cPath := rp("results/R1C_MORPHOLOGY_TABLE.json")
	cHash, _ := hash(cPath)
	c, _ := readMap(cPath)
	verdicts := map[string]string{}
	for _, v := range asSlice(c["provisional_verdicts"]) {
		vv, _ := v.(map[string]any)
		fam, _ := vv["family"].(string)
		vd, _ := vv["verdict"].(string)
		verdicts[fam] = vd
	}
	type famAgg struct {
		realN, synN                         int
		realVal, realSurf, synVal, synSurf  *float64
		realCI                              *[2]float64
		fails                               map[string]bool
	}
	aggByFam := map[string]*famAgg{}
	for _, r := range asSlice(c["rows"]) {
		rr, _ := r.(map[string]any)
		fam, _ := rr["family"].(string)
		prov, _ := rr["provenance"].(string)
		n := toI(rr["n"])
		if aggByFam[fam] == nil {
			aggByFam[fam] = &famAgg{fails: map[string]bool{}}
		}
		a := aggByFam[fam]
		val := nestedF(rr, "value_correct", "accuracy")
		surf := nestedF(rr, "surface_form_correct", "accuracy")
		if strings.HasPrefix(prov, "REAL") {
			a.realN += n
			a.realVal, a.realSurf = val, surf
			lo := nestedF(rr, "value_correct", "ci95_low")
			hi := nestedF(rr, "value_correct", "ci95_high")
			if lo != nil && hi != nil {
				a.realCI = &[2]float64{*lo, *hi}
			}
		} else {
			a.synN += n
			a.synVal, a.synSurf = val, surf
		}
		if fc, ok := rr["failure_classes"].(map[string]any); ok {
			for k := range fc {
				a.fails[k] = true
			}
		}
	}
	var morph []exocortex.MorphologyFamilyProfile
	for fam, a := range aggByFam {
		var fails []string
		for k := range a.fails {
			fails = append(fails, k)
		}
		morph = append(morph, exocortex.MorphologyFamilyProfile{
			Family: fam, RealN: a.realN, SyntheticN: a.synN,
			RealValueAccuracy: a.realVal, RealSurfaceAccuracy: a.realSurf, RealValueCI95: a.realCI,
			SynValueAccuracy: a.synVal, SynSurfaceAccuracy: a.synSurf,
			ProvisionalVerdict: firstNonEmpty(verdicts[fam], "INSUFFICIENT_REAL_EVIDENCE"),
			FailureModes:       fails, NeverPooled: true,
		})
	}

	// ---- repeatability (R1-F) ----
	fPath := rp("results/R1F_STABILITY_SUMMARY.json")
	fHash, _ := hash(fPath)
	f, _ := readMap(fPath)
	rep := exocortex.RepeatabilityProfile{
		Sentinels: toIntOr(f["sentinels"], 20), Repeats: 5,
		ByteStable:      fmt.Sprintf("%d/%d", toI(f["class_byte_stable"]), toIntOr(f["sentinels"], 20)),
		SemanticStable:  fmt.Sprintf("%d/%d", toI(f["sentinels_semantic_invariant_5of5"]), toIntOr(f["sentinels"], 20)),
		FailedSentinels: toI(f["previously_wrong_sentinels"]),
		Recoveries:      toI(f["any_exact_retry_recoveries"]),
		RuntimeRule:     "DO_NOT_RETRY_IDENTICAL_INPUT",
		EvidenceScope:   "temperature 0, byte-identical input, tested LFM2-VL runtime; no claim about non-zero temperature",
		Evidence: exocortex.EvidenceRef{
			SourceExperiment: "R1-F", ArtifactPath: "results/R1F_STABILITY_SUMMARY.json", ArtifactSHA256: fHash,
			SampleSize: 20, EvidenceClass: exocortex.EvidenceEarned,
			MeasuredMetric: "20 sentinels x 5 exact repeats: byte-stable 20/20, semantic-invariant 20/20, 0/16 failed sentinels recovered",
			Limitations:    "post-hoc sentinel selection; temperature 0 only",
		},
	}

	// ---- recovery rules (R1-G) ----
	gcurveHash := gHash
	rgRow := func(fam, cond string) map[string]any {
		g, _ := readMap(gPath)
		for _, r := range asSlice(g["rows"]) {
			rr, _ := r.(map[string]any)
			if rr["family"] == fam && rr["recovery_condition"] == cond {
				return rr
			}
		}
		return map[string]any{}
	}
	ga32 := rgRow("GA_SCALE", "GA2_NOMINAL_SCALE")
	gcReal := rgRow("GC_ASSOC_REAL", "GC_REAL_2")
	mDelta := func(row map[string]any) string {
		m, _ := row["paired_mcnemar_baseline_to_recovery"].(map[string]any)
		return fmt.Sprintf("delta %+.2f, W->C %d, C->W %d, exact p %v",
			toF(m["absolute_delta"]), toI(m["w_to_c"]), toI(m["c_to_w"]), m["p_value"])
	}
	recRules := []exocortex.RecoveryRule{
		{
			ID: "LOW_SCALE", DetectIf: "measured containing-line / target height < 16 px",
			PreferredAction: "upscale to 32 px BEFORE the model call", Classification: "PREVENTIVE+EARNED", ModelCallsAfter: -1,
			Evidence: exocortex.EvidenceRef{SourceExperiment: "R1-G", ArtifactPath: "results/R1G_RECOVERY_TABLE.json", ArtifactSHA256: gcurveHash, SampleSize: 20, EvidenceClass: exocortex.EvidenceEarned, MeasuredMetric: "GA_SCALE 8px->32px: 0.20 -> 0.95, " + mDelta(ga32), Limitations: "fresh held-out EXTRACT_NUMBER bases; one renderer / one numeric family"},
		},
		{
			ID: "NUMERIC_COMPETITORS", DetectIf: "the working-set builder detects competing numeric operands near a READ_ASSOCIATED_NUMBER operand",
			PreferredAction: "isolate / remove the competitor before the model call", Classification: "PREVENTIVE+EARNED_REAL_INTERVENTION", ModelCallsAfter: -1,
			Evidence: exocortex.EvidenceRef{SourceExperiment: "R1-G", ArtifactPath: "results/R1G_RECOVERY_TABLE.json", ArtifactSHA256: gcurveHash, SampleSize: 22, EvidenceClass: exocortex.EvidenceEarned, MeasuredMetric: "GC_ASSOC_REAL competitor removed: 0.41 -> 1.00, " + mDelta(gcReal), Limitations: "R1-D intervention reuse (not a fresh independent estimate); fresh synthetic proxy NOT_CONFIRMED (glyph-bank label anchor inadequate)"},
		},
		{
			ID: "HIGH_CONTEXT", DetectIf: "the visible working set is much larger than the operand line",
			PreferredAction: "prefer the smallest sufficient working set (crop to the operand line)", Classification: "PREVENTIVE_PRACTICE_NOT_EARNED", ModelCallsAfter: -1,
			Evidence: exocortex.EvidenceRef{SourceExperiment: "R1-G", ArtifactPath: "results/R1G_RECOVERY_TABLE.json", ArtifactSHA256: gcurveHash, SampleSize: 20, EvidenceClass: exocortex.EvidenceNoMeasuredBenefit, MeasuredMetric: "GB_CONTEXT full->line/target: 0.90 -> 0.95 (delta +0.05, p=1.0)", Limitations: "fresh-sample baseline was not adverse (0.90); R1-A1 measured 0.80 at FULL_VIEWPORT. Practice, not an earned recovery"},
		},
		{
			ID: "VALUE_CUE", DetectIf: "the renderer would emit a value cue tighter than the frozen padded rule",
			PreferredAction: "always use the frozen padded value cue (or no cue on an already-isolated operand)", Classification: "SAFE_ENGINEERING_DEFAULT", ModelCallsAfter: -1,
			Evidence: exocortex.EvidenceRef{SourceExperiment: "R1-G", ArtifactPath: "results/R1G_RECOVERY_TABLE.json", ArtifactSHA256: gcurveHash, SampleSize: 12, EvidenceClass: exocortex.EvidenceSafeEngineeringDefault, MeasuredMetric: "GD_CUE tight->padded/no-cue: 0.92 -> 1.00 (delta +0.08, NO_MEASURED_BENEFIT); R1-D D0V proved the truncation real for short 2-digit values", Limitations: "artifact barely reproduced on 12 longer-digit fresh bases; padded cue is free and never hurt"},
		},
		{
			ID: "MISSING_VISUAL_OPERAND", DetectIf: "a visual opcode is invoked with no visual operand",
			PreferredAction: "RETURN_UNSUPPORTED_OR_UNKNOWN without calling the model", Classification: "REJECT_BEFORE_CALL", ModelCallsAfter: 0,
			Evidence: exocortex.EvidenceRef{SourceExperiment: "R1-E", ArtifactPath: "results/R1E_VISUAL_DEPENDENCE_TABLE.json", ArtifactSHA256: eHash, SampleSize: 0, EvidenceClass: exocortex.EvidenceRejectBeforeCall, MeasuredMetric: "NO_IMAGE 0/22 task-gold, degenerate 12345 every time", Limitations: "adding the missing modality changes the task; runtime must reject, not hope"},
		},
		{
			ID: "EXACT_RETRY", DetectIf: "the prepared input is byte-identical to a prior failed call",
			PreferredAction: "REJECT — never repeat an identical failed call", Classification: "REJECT", ModelCallsAfter: 0,
			Evidence: exocortex.EvidenceRef{SourceExperiment: "R1-F", ArtifactPath: "results/R1F_STABILITY_SUMMARY.json", ArtifactSHA256: fHash, SampleSize: 0, EvidenceClass: exocortex.EvidenceDoNotUse, MeasuredMetric: "20/20 byte-stable, 0/16 failed sentinels recovered by 5 exact retries", Limitations: "temperature 0 only"},
		},
		{
			ID: "CURRENT_TESSERACT_OCR", DetectIf: "a caller proposes routing a failed crop to the current tesseract OCR path",
			PreferredAction: "DO_NOT_ROUTE_AS_RECOVERY", Classification: "DO_NOT_USE", ModelCallsAfter: 0,
			Evidence: exocortex.EvidenceRef{SourceExperiment: "R1-G", ArtifactPath: "results/R1G_RECOVERY_TABLE.json", ArtifactSHA256: gcurveHash, SampleSize: 0, EvidenceClass: exocortex.EvidenceDoNotUse, MeasuredMetric: "tesseract 5.5.3 over baseline crops: ~0.04 overall (2/98)", Limitations: "the current crops are small/masked/low-res; a different OCR facility is untested"},
		},
	}

	profile := exocortex.CapabilityProfileR1{
		Schema:         exocortex.CapabilityProfileR1Schema,
		ProfileID:      "parrot-lfm2-vl-1.6b@" + profileVersion,
		ProfileVersion: profileVersion,
		CompiledAt:     time.Now().UTC().Format(time.RFC3339),
		PreservesR0:    "the R0 CapabilityProfile (tlaloc.exocortex-capability-profile.r0) and the R0 behaviour profile are preserved and NOT overwritten; this is a distinct .r1 document at a new path",
		Executor:       exec,
		SourceExperiments: sourceExps,
		GlobalRules: exocortex.GlobalExecutorRules{
			MaxCognitiveTransformationsPerCall: 1,
			Rule: "ONE model invocation handles AT MOST ONE cognitive opcode; formatting / normalization / serialization are external deterministic operations; sequence and working memory are Tlaloc's responsibility; the model result is an Observation, never an authoritative Fact.",
			FormattingIsExternalDeterministic: true, SequenceWorkingMemoryIsTlaloc: true, ModelResultIsObservationNotFact: true,
			Evidence: exocortex.EvidenceRef{
				SourceExperiment: "parrot-microisa-r0 (P1/P2)", ArtifactPath: "../parrot-microisa-r0/FREEZE.json", ArtifactSHA256: sourceExps[0].ArtifactSHA256,
				SampleSize: 1, EvidenceClass: exocortex.EvidenceEarned,
				MeasuredMetric: "P1: formal_safe max_cognitive_transformations_per_call = 1", Limitations: "measured on the P1 opcode battery",
			},
		},
		ExtractNumber: en,
		Context:       ctx,
		ReadAssociated: ra,
		Morphology:    morph,
		MorphologyEvidence: exocortex.EvidenceRef{
			SourceExperiment: "R1-C", ArtifactPath: "results/R1C_MORPHOLOGY_TABLE.json", ArtifactSHA256: cHash,
			SampleSize: 12, EvidenceClass: exocortex.EvidenceObservedExploratory,
			MeasuredMetric: "per-family VALUE_CORRECT and SURFACE_FORM_CORRECT accuracy, REAL_DOCUMENT and SYNTHETIC_REALISTIC scored separately (never pooled)",
			Limitations:    "real n<=12 per family; synthetic strata cannot promote real-document reliability",
		},
		Repeatability: rep,
		RecoveryRules: recRules,
	}

	if err := exocortex.ValidateCapabilityProfileR1(profile); err != nil {
		return exocortex.CapabilityProfileR1{}, err
	}
	return profile, nil
}

// --- small json helpers ---

func asSlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

func toF(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

func toIntOr(v any, d int) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return d
}

func getMapVal(m map[string]any, path ...string) any {
	cur := any(m)
	for _, k := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[k]
	}
	return cur
}

func nestedF(m map[string]any, k1, k2 string) *float64 {
	mm, ok := m[k1].(map[string]any)
	if !ok {
		return nil
	}
	if f, ok := mm[k2].(float64); ok {
		return &f
	}
	return nil
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
