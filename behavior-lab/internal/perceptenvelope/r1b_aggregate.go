package perceptenvelope

import (
	"fmt"
	"math"

	"tlaloc.local/behaviorlab/internal/decompositionlab"
)

// R1BScaleRow is one scale rung of the R1-B curve: the shared level
// aggregate plus scale-specific physical measures.
type R1BScaleRow struct {
	LevelAggregate
	NominalLinePx          float64 `json:"nominal_line_height_px"`
	ActualLineHeightPx     float64 `json:"actual_line_height_px"`
	MeanTargetBBoxHeightPx float64 `json:"mean_target_bbox_height_px"`
	MeanPromptTokens       float64 `json:"mean_prompt_tokens"`
	Region                 string  `json:"scale_region"`
}

// R1BScaleCurve is the full R1-B result (protocol sections 14-18).
type R1BScaleCurve struct {
	Schema                string               `json:"schema"`
	ExperimentID          string               `json:"experiment_id"`
	Stage                 string               `json:"stage"`
	ContextPolicy         string               `json:"context_policy_level"`
	Bases                 int                  `json:"bases"`
	Records               int                  `json:"records"`
	Errors                int                  `json:"errors"`
	ScaleLadderPx         []float64            `json:"scale_ladder_px"`
	Rows                  []R1BScaleRow        `json:"rows"`
	Transitions           []AdjacentTransition `json:"paired_transitions"`
	TokenRegimeConstant   bool                 `json:"token_regime_constant"`
	TokenRegimeNote       string               `json:"token_regime_note"`
	StopAuditPreprocess   bool                 `json:"STOP_AND_AUDIT_PREPROCESSING"`
	FormalSafeScalePx     *float64             `json:"formal_safe_scale_px"`
	FormalSafeScaleRule   string               `json:"formal_safe_scale_rule"`
	ObservedOperatingPx   []float64            `json:"observed_operating_region_px"`
	OverscaleDegradation  bool                 `json:"overscale_degradation_observed"`
	R1A0ConsistencyNote   string               `json:"r1a0_full_page_consistency_note"`
	RecommendedR1CScalePx float64              `json:"recommended_scale_policy_for_r1c_px"`
}

const r1bCurveSchema = "tlaloc.parrot-perceptual-envelope-r1.r1b-scale-curve.r1"

// r1bTransitionPairs are the frozen paired comparisons (protocol section 15).
var r1bTransitionPairs = [][2]string{
	{"B0", "B1"}, {"B1", "B2"}, {"B2", "B3"}, {"B3", "B4"}, {"B4", "B5"},
	{"B0", "B4"}, {"B0", "B5"}, {"B2", "B4"}, {"B4", "B5"},
}

// AggregateR1BScaleCurve builds the six-point scale curve, the paired
// transitions, the failure taxonomy by scale, the region classification
// and the formal conservative safe-scale threshold.
func AggregateR1BScaleCurve(records []RecordOutcome, geos []R1BGeometry) R1BScaleCurve {
	byLevel := map[string][]RecordOutcome{}
	baseSet := map[string]struct{}{}
	errCount := 0
	for _, r := range records {
		byLevel[r.Level] = append(byLevel[r.Level], r)
		baseSet[r.BaseID] = struct{}{}
		if r.Error != "" {
			errCount++
		}
	}

	// per-base per-condition geometry lookup
	geoByCond := map[string]map[string]R1BCondGeom{}
	for _, g := range geos {
		m := map[string]R1BCondGeom{}
		for _, cg := range g.Conditions {
			m[cg.Condition] = cg
		}
		geoByCond[g.BaseID] = m
	}

	ladder := make([]float64, len(R1BScaleLadder))
	for i, c := range R1BScaleLadder {
		ladder[i] = c.LinePx
	}
	curve := R1BScaleCurve{
		Schema: r1bCurveSchema, ExperimentID: ExperimentID, Stage: "R1-B",
		ContextPolicy: "A1C0_TARGET", Bases: len(baseSet), Records: len(records),
		Errors: errCount, ScaleLadderPx: ladder,
		FormalSafeScaleRule: "smallest tested line-height with semantic accuracy >= 0.90 AND Wilson 95% lower bound >= 0.70 AND no statistically significant (exact McNemar p < 0.05, negative delta) degradation at the next larger tested scale; null if no rung qualifies",
	}

	for _, cond := range R1BScaleLadder {
		rs := byLevel[cond.ID]
		if len(rs) == 0 {
			continue
		}
		row := R1BScaleRow{LevelAggregate: LevelAggregate{Level: cond.ID, N: len(rs), FailureClasses: map[string]int{}}, NominalLinePx: cond.LinePx}
		var expSum, latSum, tokSum, lhSum, tbSum float64
		lat := make([]float64, 0, len(rs))
		for _, r := range rs {
			if r.SemanticCorrect {
				row.SemanticCorrect++
			}
			if r.ContractSuccess {
				row.ContractSuccess++
			}
			if r.Abstained {
				row.Abstained++
			}
			if r.UnsupportedAssertion {
				row.UnsupportedAssertion++
			}
			if r.FormatFailure {
				row.FormatFailure++
			}
			if r.FailureClass != "" {
				row.FailureClasses[r.FailureClass]++
			}
			expSum += r.VisualExposure
			latSum += float64(r.LatencyMS)
			tokSum += float64(r.PromptTokens)
			lat = append(lat, float64(r.LatencyMS))
			if cg, ok := geoByCond[r.BaseID][r.Level]; ok {
				lhSum += cg.LineHeightCanvasPx
				tbSum += cg.TargetBBoxHeightPx
			}
		}
		n := float64(row.N)
		row.SemanticAccuracy = ratio(row.SemanticCorrect, row.N)
		row.SemanticCI95Low, row.SemanticCI95High = decompositionlab.WilsonCI95(row.SemanticCorrect, row.N)
		row.ContractAccuracy = ratio(row.ContractSuccess, row.N)
		row.MeanVisualExposure = expSum / n
		row.MeanLatencyMS = latSum / n
		row.P95LatencyMS = percentile(lat, 0.95)
		row.MeanPromptTokens = tokSum / n
		row.ActualLineHeightPx = lhSum / n
		row.MeanTargetBBoxHeightPx = tbSum / n
		curve.Rows = append(curve.Rows, row)
	}

	for _, p := range r1bTransitionPairs {
		curve.Transitions = append(curve.Transitions, pairMcNemar(p[0], p[1], "semantic", byLevel))
	}

	// token regime constant: rounded mean prompt tokens equal across rungs
	curve.TokenRegimeConstant = true
	var firstTok float64
	for i, row := range curve.Rows {
		rt := math.Round(row.MeanPromptTokens)
		if i == 0 {
			firstTok = rt
		} else if rt != firstTok {
			curve.TokenRegimeConstant = false
		}
	}
	if curve.TokenRegimeConstant {
		curve.TokenRegimeNote = fmt.Sprintf("API-reported prompt tokens = %.0f (text only; the LM Studio endpoint does not surface clip image tokens via the API) for every scale condition. All 180 images are identical 512x512, so the clip tiling regime is scale-independent by construction; R1-A1 measured 259 image tokens at this exact size from the server log.", firstTok)
	} else {
		curve.TokenRegimeNote = "API-reported prompt tokens vary by scale condition at constant 512x512 dimensions — preprocessing may not be scale-invariant; audit the server log"
		curve.StopAuditPreprocess = true
	}

	classifyR1BRegions(&curve)
	curve.FormalSafeScalePx = formalSafeScale(&curve)
	curve.ObservedOperatingPx = observedOperating(&curve)
	curve.RecommendedR1CScalePx = recommendR1CScale(&curve)
	curve.R1A0ConsistencyNote = r1a0Consistency(&curve)
	return curve
}

// classifyR1BRegions labels each rung without assuming monotonicity
// (protocol section 16).
func classifyR1BRegions(curve *R1BScaleCurve) {
	rows := curve.Rows
	// peak accuracy
	peak := 0.0
	for _, r := range rows {
		if r.SemanticAccuracy > peak {
			peak = r.SemanticAccuracy
		}
	}
	sig := map[string]AdjacentTransition{}
	for _, t := range curve.Transitions {
		sig[t.From+"->"+t.To] = t
	}
	for i := range rows {
		acc := rows[i].SemanticAccuracy
		switch {
		case acc < 0.50:
			rows[i].Region = "TOO_SMALL"
		case acc < 0.90 || acc < peak-0.10:
			rows[i].Region = "TRANSITION_REGION"
		default:
			rows[i].Region = "OPERATING_REGION"
		}
		// overscale degradation: a significant drop from a smaller rung
		if i > 0 {
			if t, ok := sig[rows[i-1].Level+"->"+rows[i].Level]; ok && t.PValue < 0.05 && t.AbsoluteDelta < 0 && rows[i-1].Region == "OPERATING_REGION" {
				rows[i].Region = "OVERSCALE_DEGRADATION"
				curve.OverscaleDegradation = true
			}
		}
	}
}

func formalSafeScale(curve *R1BScaleCurve) *float64 {
	rows := curve.Rows
	trByPair := map[string]AdjacentTransition{}
	for _, t := range curve.Transitions {
		trByPair[t.From+"->"+t.To] = t
	}
	for i := range rows {
		if rows[i].SemanticAccuracy < 0.90 || rows[i].SemanticCI95Low < 0.70 {
			continue
		}
		degraded := false
		if i+1 < len(rows) {
			if t, ok := trByPair[rows[i].Level+"->"+rows[i+1].Level]; ok && t.PValue < 0.05 && t.AbsoluteDelta < 0 {
				degraded = true
			}
		}
		if degraded {
			continue
		}
		v := rows[i].NominalLinePx
		return &v
	}
	return nil
}

// recommendR1CScale picks a comfortable presentation point for R1-C: the
// smallest rung with semantic accuracy >= 0.95 and Wilson lower bound
// >= 0.80, but never below the formal safe scale. Falls back to the
// formal safe scale, then to the largest tested rung.
func recommendR1CScale(curve *R1BScaleCurve) float64 {
	var floor float64
	if curve.FormalSafeScalePx != nil {
		floor = *curve.FormalSafeScalePx
	}
	for _, r := range curve.Rows {
		if r.NominalLinePx < floor {
			continue
		}
		if r.SemanticAccuracy >= 0.95 && r.SemanticCI95Low >= 0.80 {
			return r.NominalLinePx
		}
	}
	if floor > 0 {
		return floor
	}
	if len(curve.Rows) > 0 {
		return curve.Rows[len(curve.Rows)-1].NominalLinePx
	}
	return 0
}

func observedOperating(curve *R1BScaleCurve) []float64 {
	var out []float64
	for _, r := range curve.Rows {
		if r.Region == "OPERATING_REGION" {
			out = append(out, r.NominalLinePx)
		}
	}
	return out
}

func r1a0Consistency(curve *R1BScaleCurve) string {
	// descriptive only: does the measured low-scale region plausibly
	// explain the R1-A0 full-page 0.10 EXTRACT_NUMBER accuracy?
	var low *R1BScaleRow
	for i := range curve.Rows {
		if low == nil || curve.Rows[i].NominalLinePx < low.NominalLinePx {
			low = &curve.Rows[i]
		}
	}
	if low == nil {
		return "no R1-B rows"
	}
	return fmt.Sprintf("R1-A0 full-page EXTRACT_NUMBER accuracy was 0.10 with target glyphs reduced to a few effective pixels. R1-B's smallest rung (%.0f px line height) scored %.2f (CI %.2f-%.2f). If the R1-A0 full-page effective target line height falls in R1-B's TOO_SMALL/TRANSITION region, the scale curve is consistent with that collapse. This is a descriptive consistency check; R1-A0 is not re-scored or retrofitted.",
		low.NominalLinePx, low.SemanticAccuracy, low.SemanticCI95Low, low.SemanticCI95High)
}
