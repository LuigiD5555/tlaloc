package perceptenvelope

import (
	"fmt"
	"sort"

	"tlaloc.local/behaviorlab/internal/decompositionlab"
)

// R1-G recovery aggregation (protocol §12-§16).

// R1GRecoveryRow is one paired recovery-condition result over the full
// frozen denominator.
type R1GRecoveryRow struct {
	Family                  string             `json:"family"`
	FamilyName              string             `json:"family_name"`
	Capability              string             `json:"capability"`
	BaselineCondition       string             `json:"baseline_condition"`
	RecoveryCondition       string             `json:"recovery_condition"`
	N                       int                `json:"n"`
	BaselineAccuracy        float64            `json:"baseline_semantic_accuracy"`
	RecoveryAccuracy        float64            `json:"recovery_semantic_accuracy"`
	BaselineCI95            [2]float64         `json:"baseline_ci95"`
	RecoveryCI95            [2]float64         `json:"recovery_ci95"`
	McNemar                 AdjacentTransition `json:"paired_mcnemar_baseline_to_recovery"`
	BaselineContractSuccess int                `json:"baseline_contract_success"`
	RecoveryContractSuccess int                `json:"recovery_contract_success"`
	RecoveryFailureClasses  map[string]int     `json:"recovery_failure_taxonomy"`
	RecoveryMeanLatencyMS   float64            `json:"recovery_mean_latency_ms"`
	RecoveryP95LatencyMS    float64            `json:"recovery_p95_latency_ms"`
	ModelCalls              int                `json:"model_calls"`
	DeterministicPreprocOps int                `json:"deterministic_preprocessing_operations"`
	// conditional recovery (§13)
	BaselineFailures        int     `json:"baseline_failures"`
	RecoveredToCorrect      int     `json:"recovered_to_correct"`
	StillWrong              int     `json:"still_wrong"`
	ChangedToDifferentWrong int     `json:"changed_to_different_wrong"`
	ConditionalRecoveryRate float64 `json:"conditional_recovery_rate"`
	// harmful recovery (§14)
	BaselineCorrect int     `json:"baseline_correct"`
	RemainCorrect   int     `json:"remain_correct"`
	DegradedToWrong int     `json:"degraded_to_wrong"`
	DegradationRate float64 `json:"degradation_rate"`
	// verdict (§15) + recovery-vs-prevention (§16)
	Verdict             string `json:"verdict"`
	Mode                string `json:"mode"`
	PreventionRationale string `json:"prevention_rationale"`
}

// R1GRecoveryTable is the full frozen R1-G recovery result.
type R1GRecoveryTable struct {
	Schema                      string           `json:"schema"`
	ExperimentID                string           `json:"experiment_id"`
	RealSyntheticNeverPooled    bool             `json:"real_synthetic_association_evidence_never_pooled"`
	IndependentAccuracyEstimate map[string]bool  `json:"independent_accuracy_estimate"`
	CrossRecoveryFamilyBaseReuse bool            `json:"CROSS_RECOVERY_FAMILY_BASE_REUSE"`
	ExactRetryNegativeControl   map[string]any   `json:"G_NEGATIVE_CONTROL_EXACT_RETRY"`
	OCR                         R1GOCRSummary    `json:"G_OCR_EXISTING"`
	Rows                        []R1GRecoveryRow `json:"rows"`
}

// R1GOCRSummary is the deterministic-OCR system-fallback comparison.
type R1GOCRSummary struct {
	Available    bool               `json:"OCR_FALLBACK_AVAILABLE"`
	Engine       string             `json:"engine,omitempty"`
	ByFamily     map[string][2]int  `json:"correct_over_baseline_by_family"` // [correct, n]
	OverallAcc   float64            `json:"overall_baseline_crop_accuracy"`
	Note         string             `json:"note"`
}

const r1gRecoveryTableSchema = "tlaloc.parrot-perceptual-envelope-r1.r1g-recovery-table.r1"

var r1gFamilyPreprocOps = map[string]int{
	"GA_SCALE": 1, "GB_CONTEXT": 1, "GC_ASSOC_REAL": 1, "GC_ASSOC_SYN": 1, "GD_CUE": 1,
}

var r1gFamilyPrevention = map[string]string{
	"GA_SCALE":      "line_height_px is measurable from page geometry before the call; upscale to the profile-preferred 32 px BEFORE the first Parrot call rather than calling at a known-adverse scale.",
	"GB_CONTEXT":    "the visible-context size is chosen by the caller before the call; crop to the operand line (or the target) BEFORE the first call rather than submitting a full viewport.",
	"GC_ASSOC_REAL": "competing numbers on the operand line/crop are detectable pre-call (digit-token count); isolate the operand to the clean single-line label/value working set BEFORE the call.",
	"GC_ASSOC_SYN":  "same as GC_ASSOC_REAL: detectable competing numbers -> isolate the operand pre-call.",
	"GD_CUE":        "the cue rectangle geometry is produced by the renderer; never emit a tight value cue that can clip a short integer — use the frozen padded-cue rule (or no cue on an already-isolated operand).",
}

func pairR1G(baseline, recovery []R1GRecord) (AdjacentTransition, []r1gTransition) {
	bm := map[string]R1GRecord{}
	for _, r := range baseline {
		if r.Error == "" {
			bm[r.BaseID] = r
		}
	}
	tr := AdjacentTransition{From: "BASELINE", To: "RECOVERY", Metric: "semantic"}
	var pairs []decompositionlab.PairedOutcome
	var trans []r1gTransition
	rSorted := append([]R1GRecord(nil), recovery...)
	sort.Slice(rSorted, func(i, j int) bool { return rSorted[i].BaseID < rSorted[j].BaseID })
	for _, rr := range rSorted {
		br, ok := bm[rr.BaseID]
		if !ok || rr.Error != "" {
			continue
		}
		b, a := br.SemanticCorrect, rr.SemanticCorrect
		pairs = append(pairs, decompositionlab.PairedOutcome{CorrectBefore: b, CorrectAfter: a})
		switch {
		case b && a:
			tr.CorrectToCorrect++
		case b && !a:
			tr.CorrectToWrong++
		case !b && a:
			tr.WrongToCorrect++
		default:
			tr.WrongToWrong++
		}
		trans = append(trans, r1gTransition{
			BaseID: rr.BaseID, BaselineCorrect: b, RecoveryCorrect: a,
			BaselineRaw: br.RawText, RecoveryRaw: rr.RawText,
			BaselineNorm: br.NormalizedValue, RecoveryNorm: rr.NormalizedValue,
		})
	}
	res := decompositionlab.McNemarExact(pairs)
	tr.AbsoluteDelta, tr.PValue = res.AbsoluteDelta, res.PValue
	return tr, trans
}

type r1gTransition struct {
	BaseID          string `json:"base_id"`
	BaselineCorrect bool   `json:"baseline_correct"`
	RecoveryCorrect bool   `json:"recovery_correct"`
	BaselineRaw     string `json:"baseline_raw"`
	RecoveryRaw     string `json:"recovery_raw"`
	BaselineNorm    string `json:"baseline_norm"`
	RecoveryNorm    string `json:"recovery_norm"`
}

// R1GFailureTransitions groups the per-base transitions by family/condition.
type R1GFailureTransitions struct {
	Schema       string                       `json:"schema"`
	ExperimentID string                       `json:"experiment_id"`
	Groups       map[string][]r1gTransition   `json:"transitions_by_family_recovery_condition"`
}

func classifyR1GVerdict(row R1GRecoveryRow) string {
	m := row.McNemar
	delta := m.AbsoluteDelta
	switch {
	case row.N < 8:
		return "INSUFFICIENT_EVIDENCE"
	case m.CorrectToWrong > m.WrongToCorrect || delta <= -r1gPromisingDelta:
		return "HARMFUL"
	case delta >= r1gEarnedDelta && m.WrongToCorrect > m.CorrectToWrong && row.DegradationRate <= r1gMaxDegradation && m.PValue < r1gEarnedMcNemarSig:
		return "EARNED_RECOVERY"
	case delta >= r1gPromisingDelta && m.WrongToCorrect > m.CorrectToWrong:
		return "PROMISING_RECOVERY"
	case abs64(delta) < r1gNoBenefitBand:
		return "NO_MEASURED_BENEFIT"
	default:
		return "PROMISING_RECOVERY"
	}
}

func abs64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func r1gBuildRow(fam R1GFamily, baselineCond, recoveryCond string, baseline, recovery []R1GRecord) (R1GRecoveryRow, []r1gTransition) {
	row := R1GRecoveryRow{
		Family: fam.Key, FamilyName: fam.Name, Capability: fam.Capability,
		BaselineCondition: baselineCond, RecoveryCondition: recoveryCond,
		RecoveryFailureClasses: map[string]int{},
		DeterministicPreprocOps: r1gFamilyPreprocOps[fam.Key],
		Mode: "PREVENTIVE", PreventionRationale: r1gFamilyPrevention[fam.Key],
	}
	bm := map[string]R1GRecord{}
	for _, r := range baseline {
		if r.Error == "" {
			bm[r.BaseID] = r
		}
	}
	var bCorrect, rCorrect int
	var lat []float64
	for _, rr := range recovery {
		if rr.Error != "" {
			continue
		}
		br, ok := bm[rr.BaseID]
		if !ok {
			continue
		}
		row.N++
		row.ModelCalls += 2
		if br.SemanticCorrect {
			bCorrect++
		}
		if rr.SemanticCorrect {
			rCorrect++
		}
		if br.ContractSuccess {
			row.BaselineContractSuccess++
		}
		if rr.ContractSuccess {
			row.RecoveryContractSuccess++
		}
		if rr.FailureClass != "" {
			row.RecoveryFailureClasses[rr.FailureClass]++
		}
		lat = append(lat, float64(rr.LatencyMS))
		if !br.SemanticCorrect {
			row.BaselineFailures++
			switch {
			case rr.SemanticCorrect:
				row.RecoveredToCorrect++
			case rr.NormalizedValue != br.NormalizedValue:
				row.ChangedToDifferentWrong++
				row.StillWrong++
			default:
				row.StillWrong++
			}
		} else {
			row.BaselineCorrect++
			if rr.SemanticCorrect {
				row.RemainCorrect++
			} else {
				row.DegradedToWrong++
			}
		}
	}
	row.BaselineAccuracy = ratio(bCorrect, row.N)
	row.RecoveryAccuracy = ratio(rCorrect, row.N)
	row.BaselineCI95[0], row.BaselineCI95[1] = decompositionlab.WilsonCI95(bCorrect, row.N)
	row.RecoveryCI95[0], row.RecoveryCI95[1] = decompositionlab.WilsonCI95(rCorrect, row.N)
	if row.BaselineFailures > 0 {
		row.ConditionalRecoveryRate = ratio(row.RecoveredToCorrect, row.BaselineFailures)
	}
	if row.BaselineCorrect > 0 {
		row.DegradationRate = ratio(row.DegradedToWrong, row.BaselineCorrect)
	}
	if len(lat) > 0 {
		s := 0.0
		for _, v := range lat {
			s += v
		}
		row.RecoveryMeanLatencyMS = s / float64(len(lat))
		row.RecoveryP95LatencyMS = percentile(lat, 0.95)
	}
	mc, trans := pairR1G(baseline, recovery)
	row.McNemar = mc
	row.Verdict = classifyR1GVerdict(row)
	return row, trans
}

// AggregateR1G builds the recovery table + the failure-transition groups.
func AggregateR1G(records []R1GRecord, ds R1GDataset, ocr []R1GOCRRecord) (R1GRecoveryTable, R1GFailureTransitions) {
	byFC := map[string]map[string][]R1GRecord{} // family -> condition -> records
	for _, r := range records {
		if byFC[r.Family] == nil {
			byFC[r.Family] = map[string][]R1GRecord{}
		}
		byFC[r.Family][r.Condition] = append(byFC[r.Family][r.Condition], r)
	}
	table := R1GRecoveryTable{
		Schema: r1gRecoveryTableSchema, ExperimentID: ExperimentID,
		RealSyntheticNeverPooled: true,
		IndependentAccuracyEstimate: map[string]bool{
			"GA_SCALE": true, "GB_CONTEXT": true, "GC_ASSOC_REAL": false, "GC_ASSOC_SYN": true, "GD_CUE": true,
		},
		CrossRecoveryFamilyBaseReuse: ds.CrossRecoveryFamilyBaseReuse,
		ExactRetryNegativeControl: map[string]any{
			"status":                    ds.ExactRetryImported,
			"imported_from":             "R1-F",
			"previously_wrong_recovered": "0/16",
			"new_model_calls":           0,
			"verdict":                   "DO_NOT_USE",
		},
	}
	ft := R1GFailureTransitions{
		Schema: "tlaloc.parrot-perceptual-envelope-r1.r1g-failure-transitions.r1",
		ExperimentID: ExperimentID, Groups: map[string][]r1gTransition{},
	}
	for _, fam := range R1GFamilies {
		conds := fam.Conditions
		if len(conds) < 2 {
			continue
		}
		baselineCond := conds[0]
		baseline := byFC[fam.Key][baselineCond]
		for _, recoveryCond := range conds[1:] {
			recovery := byFC[fam.Key][recoveryCond]
			row, trans := r1gBuildRow(fam, baselineCond, recoveryCond, baseline, recovery)
			table.Rows = append(table.Rows, row)
			ft.Groups[fam.Key+"|"+recoveryCond] = trans
		}
	}

	// OCR summary
	table.OCR = R1GOCRSummary{
		Available: ds.OCRFallbackAvailable, Engine: ds.OCREngine,
		ByFamily: map[string][2]int{},
		Note:     "Deterministic OCR (tesseract, digit whitelist) over the frozen BASELINE crops only. A SYSTEM_FALLBACK comparison; it does NOT drive the recovery verdicts.",
	}
	var ocrCorrect, ocrN int
	for _, o := range ocr {
		if o.Error != "" {
			continue
		}
		p := table.OCR.ByFamily[o.Family]
		p[1]++
		if o.SemanticCorrect {
			p[0]++
			ocrCorrect++
		}
		ocrN++
		table.OCR.ByFamily[o.Family] = p
	}
	if ocrN > 0 {
		table.OCR.OverallAcc = ratio(ocrCorrect, ocrN)
	}
	return table, ft
}

// R1GScientificAnswers renders the §20 required answers from the table.
func R1GScientificAnswers(t R1GRecoveryTable) map[string]string {
	get := func(fam, rec string) (R1GRecoveryRow, bool) {
		for _, r := range t.Rows {
			if r.Family == fam && r.RecoveryCondition == rec {
				return r, true
			}
		}
		return R1GRecoveryRow{}, false
	}
	ans := map[string]string{}
	ga16, _ := get("GA_SCALE", "GA1_SAFE_SCALE")
	ga32, _ := get("GA_SCALE", "GA2_NOMINAL_SCALE")
	ans["A_upscale_recovers_low_scale"] = fmt.Sprintf("8->16px: Δ %+.2f (%s); 8->32px: Δ %+.2f (%s).",
		ga16.McNemar.AbsoluteDelta, ga16.Verdict, ga32.McNemar.AbsoluteDelta, ga32.Verdict)
	ans["B_16_sufficient_or_32_better"] = fmt.Sprintf("16px recovery acc %.2f vs 32px recovery acc %.2f (gap %+.2f).",
		ga16.RecoveryAccuracy, ga32.RecoveryAccuracy, ga32.RecoveryAccuracy-ga16.RecoveryAccuracy)
	gbLine, _ := get("GB_CONTEXT", "GB1_LINE_RECOVERY")
	gbTgt, _ := get("GB_CONTEXT", "GB2_TARGET_RECOVERY")
	ans["C_context_reduction_recovers"] = fmt.Sprintf("full->line: Δ %+.2f (%s); full->target: Δ %+.2f (%s).",
		gbLine.McNemar.AbsoluteDelta, gbLine.Verdict, gbTgt.McNemar.AbsoluteDelta, gbTgt.Verdict)
	ans["D_line_enough_or_target_needed"] = fmt.Sprintf("LINE recovery acc %.2f vs TARGET recovery acc %.2f (gap %+.2f).",
		gbLine.RecoveryAccuracy, gbTgt.RecoveryAccuracy, gbTgt.RecoveryAccuracy-gbLine.RecoveryAccuracy)
	syn1, _ := get("GC_ASSOC_SYN", "GC_SYN_1")
	syn2, _ := get("GC_ASSOC_SYN", "GC_SYN_2")
	real1, _ := get("GC_ASSOC_REAL", "GC_REAL_1")
	ans["E_remove_competitor_recovers_assoc"] = fmt.Sprintf("SYN mask-competitor Δ %+.2f (%s); REAL(reuse) mask-competitor Δ %+.2f (%s).",
		syn1.McNemar.AbsoluteDelta, syn1.Verdict, real1.McNemar.AbsoluteDelta, real1.Verdict)
	ans["F_isolation_adds_benefit"] = fmt.Sprintf("SYN isolate vs mask gap %+.2f (isolate acc %.2f).",
		syn2.RecoveryAccuracy-syn1.RecoveryAccuracy, syn2.RecoveryAccuracy)
	gdPad, _ := get("GD_CUE", "GD1_PADDED_VALUE_CUE")
	gdNo, _ := get("GD_CUE", "GD2_NO_VALUE_CUE")
	ans["G_cue_fix_removes_truncation"] = fmt.Sprintf("tight->padded Δ %+.2f (%s); tight->no-cue Δ %+.2f (%s).",
		gdPad.McNemar.AbsoluteDelta, gdPad.Verdict, gdNo.McNemar.AbsoluteDelta, gdNo.Verdict)
	worst := 0.0
	for _, r := range t.Rows {
		if r.DegradationRate > worst {
			worst = r.DegradationRate
		}
	}
	ans["H_any_recovery_damages_correct"] = fmt.Sprintf("max degradation rate across all recovery conditions = %.2f (threshold %.2f).", worst, r1gMaxDegradation)
	var earned, preventive []string
	for _, r := range t.Rows {
		if r.Verdict == "EARNED_RECOVERY" {
			earned = append(earned, r.Family+"/"+r.RecoveryCondition)
		}
		if r.Verdict == "EARNED_RECOVERY" || r.Verdict == "PROMISING_RECOVERY" {
			preventive = append(preventive, r.Family+"/"+r.RecoveryCondition+" ("+r.Mode+")")
		}
	}
	ans["I_earned_recovery_interventions"] = joinOr(earned, "none")
	ans["J_preventive_adapter_rules"] = joinOr(preventive, "none")
	return ans
}

func joinOr(xs []string, empty string) string {
	if len(xs) == 0 {
		return empty
	}
	out := xs[0]
	for _, x := range xs[1:] {
		out += "; " + x
	}
	return out
}
