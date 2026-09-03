package decompositionlab

import (
	"sort"
)

// ConditionAggregate is one condition's row in the section 18 results
// table. It is computed only from RecordOutcome values an actual run
// produced — an empty or partial records slice yields an honestly empty
// aggregate (n=0), never a fabricated number.
type ConditionAggregate struct {
	Condition                Condition `json:"condition"`
	ExternalizedCapabilities int       `json:"externalized_capabilities"`
	N                        int       `json:"n"`
	Attempted                int       `json:"attempted"`
	ContractSuccess          int       `json:"contract_success"`
	SemanticCorrect          int       `json:"semantic_correct"`
	Abstained                int       `json:"abstained"`
	UnsupportedAssertion     int       `json:"unsupported_assertion"`
	FormatFailure            int       `json:"format_failure"`

	ContractAccuracy   float64 `json:"contract_accuracy"`
	ContractAccuracyLo float64 `json:"contract_accuracy_ci95_low"`
	ContractAccuracyHi float64 `json:"contract_accuracy_ci95_high"`
	SemanticAccuracy   float64 `json:"semantic_accuracy"`
	SemanticAccuracyLo float64 `json:"semantic_accuracy_ci95_low"`
	SemanticAccuracyHi float64 `json:"semantic_accuracy_ci95_high"`

	MeanLatencyMS    float64 `json:"mean_latency_ms"`
	P95LatencyMS     float64 `json:"p95_latency_ms"`
	ParrotCalls      int     `json:"parrot_calls"`
	DeterministicOps int     `json:"deterministic_ops"`

	MedianVisualExposure float64 `json:"median_visual_exposure_ratio"`
	MeanVisualExposure   float64 `json:"mean_visual_exposure_ratio"`
	P95VisualExposure    float64 `json:"p95_visual_exposure_ratio"`

	ByCategory map[string]CategoryAggregate `json:"by_category"`
}

// CategoryAggregate is the per-category breakdown section 18 also asks for.
type CategoryAggregate struct {
	N                int     `json:"n"`
	ContractAccuracy float64 `json:"contract_accuracy"`
	SemanticAccuracy float64 `json:"semantic_accuracy"`
}

// AggregateCondition reduces every RecordOutcome for one condition into its
// ConditionAggregate row.
func AggregateCondition(condition Condition, outcomes []RecordOutcome) ConditionAggregate {
	agg := ConditionAggregate{Condition: condition, ExternalizedCapabilities: condition.ExternalizedCapabilities(), N: len(outcomes), ByCategory: map[string]CategoryAggregate{}}
	var latencies, exposures []float64
	byCategory := map[string][]RecordOutcome{}
	for _, o := range outcomes {
		if o.Attempted {
			agg.Attempted++
		}
		if o.ContractSuccess {
			agg.ContractSuccess++
		}
		if o.SemanticCorrect {
			agg.SemanticCorrect++
		}
		if o.Abstained {
			agg.Abstained++
		}
		if o.UnsupportedAssertion {
			agg.UnsupportedAssertion++
		}
		if o.FormatFailure {
			agg.FormatFailure++
		}
		agg.ParrotCalls += o.ParrotCalls
		agg.DeterministicOps += o.DeterministicOps
		latencies = append(latencies, float64(o.LatencyMS))
		exposures = append(exposures, o.VisualExposureRatio)
		byCategory[o.Category] = append(byCategory[o.Category], o)
	}
	agg.ContractAccuracy = Accuracy(agg.ContractSuccess, agg.N)
	agg.ContractAccuracyLo, agg.ContractAccuracyHi = WilsonCI95(agg.ContractSuccess, agg.N)
	agg.SemanticAccuracy = Accuracy(agg.SemanticCorrect, agg.N)
	agg.SemanticAccuracyLo, agg.SemanticAccuracyHi = WilsonCI95(agg.SemanticCorrect, agg.N)
	agg.MeanLatencyMS = Mean(latencies)
	agg.P95LatencyMS = Percentile95(latencies)
	agg.MedianVisualExposure = Median(exposures)
	agg.MeanVisualExposure = Mean(exposures)
	agg.P95VisualExposure = Percentile95(exposures)
	for cat, group := range byCategory {
		contractOK, semanticOK := 0, 0
		for _, o := range group {
			if o.ContractSuccess {
				contractOK++
			}
			if o.SemanticCorrect {
				semanticOK++
			}
		}
		agg.ByCategory[cat] = CategoryAggregate{N: len(group), ContractAccuracy: Accuracy(contractOK, len(group)), SemanticAccuracy: Accuracy(semanticOK, len(group))}
	}
	return agg
}

// Transition is one paired comparison between two conditions (section 18:
// C0->C1, C1->C2, C2->C3, C0->C3; section 26: oracle vs real).
type Transition struct {
	From Condition `json:"from"`
	To   Condition `json:"to"`

	ContractMcNemar McNemarResult `json:"contract_mcnemar"`
	SemanticMcNemar McNemarResult `json:"semantic_mcnemar"`
	ContractDelta   float64       `json:"contract_delta"`
	SemanticDelta   float64       `json:"semantic_delta"`
}

// PairTransition pairs two conditions' RecordOutcomes by BaseID (the T0
// paired denominator, section 16) and computes the McNemar comparison.
// Records missing from either side are excluded from the pairing rather
// than silently treated as a particular outcome.
func PairTransition(from, to Condition, byBaseIDFrom, byBaseIDTo map[string]RecordOutcome) Transition {
	t := Transition{From: from, To: to}
	ids := make([]string, 0, len(byBaseIDFrom))
	for id := range byBaseIDFrom {
		if _, ok := byBaseIDTo[id]; ok {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	var contractPairs, semanticPairs []PairedOutcome
	for _, id := range ids {
		a, b := byBaseIDFrom[id], byBaseIDTo[id]
		contractPairs = append(contractPairs, PairedOutcome{CorrectBefore: a.ContractSuccess, CorrectAfter: b.ContractSuccess})
		semanticPairs = append(semanticPairs, PairedOutcome{CorrectBefore: a.SemanticCorrect, CorrectAfter: b.SemanticCorrect})
	}
	t.ContractMcNemar = McNemarExact(contractPairs)
	t.SemanticMcNemar = McNemarExact(semanticPairs)
	t.ContractDelta = t.ContractMcNemar.AbsoluteDelta
	t.SemanticDelta = t.SemanticMcNemar.AbsoluteDelta
	return t
}

// IndexByBaseID is a small convenience used before pairing two conditions'
// outcome slices.
func IndexByBaseID(outcomes []RecordOutcome) map[string]RecordOutcome {
	out := make(map[string]RecordOutcome, len(outcomes))
	for _, o := range outcomes {
		out[o.BaseID] = o
	}
	return out
}
