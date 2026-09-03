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

// AggregateContextCurve builds the curve from R1-A record outcomes.
func AggregateContextCurve(records []RecordOutcome) ContextCurve {
	byLevel := map[string][]RecordOutcome{}
	baseSet := map[string]struct{}{}
	for _, r := range records {
		byLevel[r.Level] = append(byLevel[r.Level], r)
		baseSet[r.BaseID] = struct{}{}
	}
	curve := ContextCurve{
		Schema: curveSchema, ExperimentID: ExperimentID, Stage: "R1-A",
		Bases: len(baseSet), Records: len(records),
	}
	for _, level := range AllContextLevels {
		rs := byLevel[string(level)]
		if len(rs) == 0 {
			continue
		}
		agg := LevelAggregate{Level: string(level), N: len(rs), FailureClasses: map[string]int{}}
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

	adj := func(from, to ContextLevel) {
		curve.Adjacent = append(curve.Adjacent,
			pairMcNemar(from, to, "semantic", byLevel),
			pairMcNemar(from, to, "contract", byLevel))
	}
	for i := 0; i+1 < len(AllContextLevels); i++ {
		adj(AllContextLevels[i], AllContextLevels[i+1])
	}
	curve.EndToEnd = append(curve.EndToEnd,
		pairMcNemar(A0TargetOnly, A6FullPage, "semantic", byLevel),
		pairMcNemar(A1TargetPlusLine, A6FullPage, "semantic", byLevel),
		pairMcNemar(A2LocalBlock, A6FullPage, "semantic", byLevel))
	return curve
}

func pairMcNemar(from, to ContextLevel, metric string, byLevel map[string][]RecordOutcome) AdjacentTransition {
	fromByBase := map[string]RecordOutcome{}
	for _, r := range byLevel[string(from)] {
		fromByBase[r.BaseID] = r
	}
	var pairs []decompositionlab.PairedOutcome
	tr := AdjacentTransition{From: string(from), To: string(to), Metric: metric}
	toRecs := append([]RecordOutcome(nil), byLevel[string(to)]...)
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
