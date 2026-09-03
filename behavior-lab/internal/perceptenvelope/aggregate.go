package perceptenvelope

import (
	"sort"

	"tlaloc.local/behaviorlab/internal/decompositionlab"
)

// LevelAggregate is one context level's row in the R1-A context curve.
type LevelAggregate struct {
	Level                string         `json:"context_level"`
	N                    int            `json:"n"`
	SemanticCorrect      int            `json:"semantic_correct"`
	SemanticAccuracy     float64        `json:"semantic_accuracy"`
	SemanticCI95Low      float64        `json:"semantic_ci95_low"`
	SemanticCI95High     float64        `json:"semantic_ci95_high"`
	ContractSuccess      int            `json:"contract_success"`
	ContractAccuracy     float64        `json:"contract_accuracy"`
	Abstained            int            `json:"abstained"`
	UnsupportedAssertion int            `json:"unsupported_assertion"`
	FormatFailure        int            `json:"format_failure"`
	MeanVisualExposure   float64        `json:"mean_visual_exposure_ratio"`
	MeanPixelArea        float64        `json:"mean_pixel_area"`
	MeanLatencyMS        float64        `json:"mean_latency_ms"`
	P95LatencyMS         float64        `json:"p95_latency_ms"`
	FailureClasses       map[string]int `json:"failure_classes"`
}

// AdjacentTransition is a paired McNemar between two adjacent context levels.
type AdjacentTransition struct {
	From             string  `json:"from"`
	To               string  `json:"to"`
	Metric           string  `json:"metric"`
	CorrectToCorrect int     `json:"c_to_c"`
	CorrectToWrong   int     `json:"c_to_w"`
	WrongToCorrect   int     `json:"w_to_c"`
	WrongToWrong     int     `json:"w_to_w"`
	AbsoluteDelta    float64 `json:"absolute_delta"`
	PValue           float64 `json:"p_value"`
}

// ContextCurve is the full R1-A result.
type ContextCurve struct {
	Schema       string               `json:"schema"`
	ExperimentID string               `json:"experiment_id"`
	Stage        string               `json:"stage"`
	Bases        int                  `json:"bases"`
	Records      int                  `json:"records"`
	Levels       []LevelAggregate     `json:"levels"`
	Adjacent     []AdjacentTransition `json:"adjacent_transitions"`
	EndToEnd     []AdjacentTransition `json:"endpoint_transitions"`
}

const curveSchema = "tlaloc.parrot-perceptual-envelope-r1.context-curve.r1"

// AggregateContextCurve builds the R1-A0 curve (7 A0..A6 levels).
func AggregateContextCurve(records []RecordOutcome) ContextCurve {
	order := make([]string, len(AllContextLevels))
	for i, l := range AllContextLevels {
		order[i] = string(l)
	}
	return aggregateCurveByLevels(records, order, "R1-A")
}

// AggregateR1A1Curve builds the R1-A1 fixed-scale curve (7 nested levels).
func AggregateR1A1Curve(records []RecordOutcome) ContextCurve {
	order := make([]string, len(AllR1A1Levels))
	for i, l := range AllR1A1Levels {
		order[i] = string(l)
	}
	return aggregateCurveByLevels(records, order, "R1-A1")
}

func aggregateCurveByLevels(records []RecordOutcome, levelOrder []string, stage string) ContextCurve {
	byLevel := map[string][]RecordOutcome{}
	baseSet := map[string]struct{}{}
	for _, r := range records {
		byLevel[r.Level] = append(byLevel[r.Level], r)
		baseSet[r.BaseID] = struct{}{}
	}
	curve := ContextCurve{
		Schema: curveSchema, ExperimentID: ExperimentID, Stage: stage,
		Bases: len(baseSet), Records: len(records),
	}
	for _, level := range levelOrder {
		rs := byLevel[level]
		if len(rs) == 0 {
			continue
		}
		agg := LevelAggregate{Level: level, N: len(rs), FailureClasses: map[string]int{}}
		var expSum, areaSum, latSum float64
		lat := make([]float64, 0, len(rs))
		for _, r := range rs {
			if r.SemanticCorrect {
				agg.SemanticCorrect++
			}
			if r.ContractSuccess {
				agg.ContractSuccess++
			}
			if r.Abstained {
				agg.Abstained++
			}
			if r.UnsupportedAssertion {
				agg.UnsupportedAssertion++
			}
			if r.FormatFailure {
				agg.FormatFailure++
			}
			if r.FailureClass != "" {
				agg.FailureClasses[r.FailureClass]++
			}
			expSum += r.VisualExposure
			areaSum += float64(r.PixelArea)
			latSum += float64(r.LatencyMS)
			lat = append(lat, float64(r.LatencyMS))
		}
		agg.SemanticAccuracy = ratio(agg.SemanticCorrect, agg.N)
		agg.SemanticCI95Low, agg.SemanticCI95High = decompositionlab.WilsonCI95(agg.SemanticCorrect, agg.N)
		agg.ContractAccuracy = ratio(agg.ContractSuccess, agg.N)
		agg.MeanVisualExposure = expSum / float64(agg.N)
		agg.MeanPixelArea = areaSum / float64(agg.N)
		agg.MeanLatencyMS = latSum / float64(agg.N)
		agg.P95LatencyMS = percentile(lat, 0.95)
		curve.Levels = append(curve.Levels, agg)
	}

	for i := 0; i+1 < len(levelOrder); i++ {
		curve.Adjacent = append(curve.Adjacent,
			pairMcNemar(levelOrder[i], levelOrder[i+1], "semantic", byLevel),
			pairMcNemar(levelOrder[i], levelOrder[i+1], "contract", byLevel))
	}
	if len(levelOrder) >= 3 {
		last := levelOrder[len(levelOrder)-1]
		curve.EndToEnd = append(curve.EndToEnd,
			pairMcNemar(levelOrder[0], last, "semantic", byLevel),
			pairMcNemar(levelOrder[2], last, "semantic", byLevel))
	}
	return curve
}

func pairMcNemar(from, to string, metric string, byLevel map[string][]RecordOutcome) AdjacentTransition {
	fromByBase := map[string]RecordOutcome{}
	for _, r := range byLevel[from] {
		fromByBase[r.BaseID] = r
	}
	var pairs []decompositionlab.PairedOutcome
	tr := AdjacentTransition{From: from, To: to, Metric: metric}
	toRecs := append([]RecordOutcome(nil), byLevel[to]...)
	sort.Slice(toRecs, func(i, j int) bool { return toRecs[i].BaseID < toRecs[j].BaseID })
	for _, tRec := range toRecs {
		fRec, ok := fromByBase[tRec.BaseID]
		if !ok {
			continue
		}
		before := metricValue(fRec, metric)
		after := metricValue(tRec, metric)
		pairs = append(pairs, decompositionlab.PairedOutcome{CorrectBefore: before, CorrectAfter: after})
		switch {
		case before && after:
			tr.CorrectToCorrect++
		case before && !after:
			tr.CorrectToWrong++
		case !before && after:
			tr.WrongToCorrect++
		default:
			tr.WrongToWrong++
		}
	}
	res := decompositionlab.McNemarExact(pairs)
	tr.AbsoluteDelta = res.AbsoluteDelta
	tr.PValue = res.PValue
	return tr
}

// DiagnosticCell is one (mode, level) accuracy cell of the scale-confound
// diagnostic.
type DiagnosticCell struct {
	Mode            string  `json:"mode"`
	Level           string  `json:"context_level"`
	N               int     `json:"n"`
	SemanticCorrect int     `json:"semantic_correct"`
	SemanticAcc     float64 `json:"semantic_accuracy"`
	MeanCropW       float64 `json:"mean_crop_width_px"`
	MeanCropH       float64 `json:"mean_crop_height_px"`
	MeanExposure    float64 `json:"mean_visual_exposure_ratio"`
}

// DiagnosticCompare is the full scale-confound diagnostic result.
type DiagnosticCompare struct {
	Schema               string               `json:"schema"`
	ExperimentID         string               `json:"experiment_id"`
	Bases                []string             `json:"diagnostic_base_ids"`
	Levels               []string             `json:"levels"`
	Cells                []DiagnosticCell     `json:"cells"`
	NaturalCurve         map[string]float64   `json:"natural_crop_semantic_by_level"`
	FixedCurve           map[string]float64   `json:"fixed_canvas_semantic_by_level"`
	PerLevelDelta        map[string]float64   `json:"fixed_minus_natural_by_level"`
	PairedFixedVsNatural []AdjacentTransition `json:"paired_fixed_vs_natural_by_level"`
	MaxAbsDelta          float64              `json:"max_abs_delta_fixed_minus_natural"`
	Confounded           bool                 `json:"CURRENT_R1A_CONTEXT_IS_SCALE_CONFOUNDED"`
	ConfoundNote         string               `json:"confound_note"`
}

const diagSchema = "tlaloc.parrot-perceptual-envelope-r1.scale-confound-diagnostic.r1"

// AggregateDiagnostic builds the mode x level comparison and applies the
// decision rule (materially different curves -> confounded).
func AggregateDiagnostic(records []RecordOutcome, levels []ContextLevel) DiagnosticCompare {
	baseSet := map[string]struct{}{}
	byModeLevel := map[string][]RecordOutcome{}
	for _, r := range records {
		baseSet[r.BaseID] = struct{}{}
		byModeLevel[r.Mode+"|"+r.Level] = append(byModeLevel[r.Mode+"|"+r.Level], r)
	}
	dc := DiagnosticCompare{
		Schema: diagSchema, ExperimentID: ExperimentID,
		NaturalCurve: map[string]float64{}, FixedCurve: map[string]float64{}, PerLevelDelta: map[string]float64{},
	}
	for b := range baseSet {
		dc.Bases = append(dc.Bases, b)
	}
	sort.Strings(dc.Bases)
	for _, lvl := range levels {
		dc.Levels = append(dc.Levels, string(lvl))
	}
	for _, mode := range []string{"NATURAL_CROP", "FIXED_CANVAS"} {
		for _, lvl := range levels {
			rs := byModeLevel[mode+"|"+string(lvl)]
			cell := DiagnosticCell{Mode: mode, Level: string(lvl), N: len(rs)}
			var cw, ch, ex float64
			for _, r := range rs {
				if r.SemanticCorrect {
					cell.SemanticCorrect++
				}
				cw += float64(r.CropWidth)
				ch += float64(r.CropHeight)
				ex += r.VisualExposure
			}
			if cell.N > 0 {
				cell.SemanticAcc = float64(cell.SemanticCorrect) / float64(cell.N)
				cell.MeanCropW = cw / float64(cell.N)
				cell.MeanCropH = ch / float64(cell.N)
				cell.MeanExposure = ex / float64(cell.N)
			}
			dc.Cells = append(dc.Cells, cell)
			if mode == "NATURAL_CROP" {
				dc.NaturalCurve[string(lvl)] = cell.SemanticAcc
			} else {
				dc.FixedCurve[string(lvl)] = cell.SemanticAcc
			}
		}
	}
	for _, lvl := range levels {
		d := dc.FixedCurve[string(lvl)] - dc.NaturalCurve[string(lvl)]
		dc.PerLevelDelta[string(lvl)] = d
		if d < 0 {
			d = -d
		}
		if d > dc.MaxAbsDelta {
			dc.MaxAbsDelta = d
		}
		// paired McNemar fixed-vs-natural at this level
		nat := map[string]RecordOutcome{}
		for _, r := range byModeLevel["NATURAL_CROP|"+string(lvl)] {
			nat[r.BaseID] = r
		}
		tr := AdjacentTransition{From: "NATURAL_CROP@" + string(lvl), To: "FIXED_CANVAS@" + string(lvl), Metric: "semantic"}
		var pairs []decompositionlab.PairedOutcome
		fx := append([]RecordOutcome(nil), byModeLevel["FIXED_CANVAS|"+string(lvl)]...)
		sort.Slice(fx, func(i, j int) bool { return fx[i].BaseID < fx[j].BaseID })
		for _, f := range fx {
			n, ok := nat[f.BaseID]
			if !ok {
				continue
			}
			pairs = append(pairs, decompositionlab.PairedOutcome{CorrectBefore: n.SemanticCorrect, CorrectAfter: f.SemanticCorrect})
			switch {
			case n.SemanticCorrect && f.SemanticCorrect:
				tr.CorrectToCorrect++
			case n.SemanticCorrect && !f.SemanticCorrect:
				tr.CorrectToWrong++
			case !n.SemanticCorrect && f.SemanticCorrect:
				tr.WrongToCorrect++
			default:
				tr.WrongToWrong++
			}
		}
		res := decompositionlab.McNemarExact(pairs)
		tr.AbsoluteDelta, tr.PValue = res.AbsoluteDelta, res.PValue
		dc.PairedFixedVsNatural = append(dc.PairedFixedVsNatural, tr)
	}
	dc.Confounded = dc.MaxAbsDelta >= 0.25
	if dc.Confounded {
		dc.ConfoundNote = "FIXED_CANVAS (constant image size / target scale) and NATURAL_CROP context curves differ by >= 0.25 at one or more levels: the R1-A0 natural-crop context decline is at least partly an effective-scale effect. R1-A0 is the NATURAL_VISUAL_FIELD_CURVE, not the isolated context envelope. Build R1-A1_FIXED_SCALE_CONTEXT as canonical."
	} else {
		dc.ConfoundNote = "FIXED_CANVAS and NATURAL_CROP context curves agree within 0.25 at every tested level: context and scale are not materially confounded on this diagnostic set."
	}
	return dc
}

func metricValue(r RecordOutcome, metric string) bool {
	if metric == "contract" {
		return r.ContractSuccess
	}
	return r.SemanticCorrect
}

func ratio(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func percentile(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	idx := int(p * float64(len(s)-1))
	return s[idx]
}
