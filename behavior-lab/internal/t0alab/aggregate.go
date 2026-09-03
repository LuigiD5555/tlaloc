package t0alab

import (
	"sort"

	"tlaloc.local/behaviorlab/internal/decompositionlab"
)

// ConditionAggregate is one T0-A condition's row (T0 protocol section 31).
type ConditionAggregate struct {
	Condition            Condition `json:"condition"`
	N                    int       `json:"n"`
	Attempted            int       `json:"attempted"`
	ContractSuccess      int       `json:"contract_success"`
	SemanticCorrect      int       `json:"semantic_correct"`
	Abstained            int       `json:"abstained"`
	UnsupportedAssertion int       `json:"unsupported_assertion"`
	FormatFailure        int       `json:"format_failure"`

	SemanticAccuracy   float64 `json:"semantic_accuracy"`
	SemanticCI95Low    float64 `json:"semantic_ci95_low"`
	SemanticCI95High   float64 `json:"semantic_ci95_high"`
	ContractAccuracy   float64 `json:"contract_accuracy"`
	ContractCI95Low    float64 `json:"contract_ci95_low"`
	ContractCI95High   float64 `json:"contract_ci95_high"`
	MeanLatencyMS      float64 `json:"mean_latency_ms"`
	P95LatencyMS       float64 `json:"p95_latency_ms"`
	ModelCalls         int     `json:"model_calls"`
	DeterministicOps   int     `json:"deterministic_ops"`
	MeanWorkflowDepth  float64 `json:"mean_workflow_depth"`
	MaxCognitiveOpsAny int     `json:"max_cognitive_ops_given_to_model_any_step"`
}

// Transition is one paired comparison (D0->D1, D1->D2, D2->D3, D0->D3).
type Transition struct {
	From            Condition                      `json:"from"`
	To              Condition                      `json:"to"`
	PairsN          int                            `json:"pairs_n"`
	SemanticMcNemar decompositionlab.McNemarResult `json:"semantic_mcnemar"`
	ContractMcNemar decompositionlab.McNemarResult `json:"contract_mcnemar"`
	SemanticDelta   float64                        `json:"semantic_delta"`
	ContractDelta   float64                        `json:"contract_delta"`
}

// Results is the full aggregated T0-A artifact.
type Results struct {
	Schema      string                        `json:"schema"`
	DatasetSHA  string                        `json:"dataset_sha256"`
	Conditions  map[string]ConditionAggregate `json:"conditions"`
	Transitions map[string]Transition         `json:"transitions"`
}

const ResultsSchemaR0 = "tlaloc.exocortex-t0a.results.r0"

// AggregateCondition reduces one condition's outcomes.
func AggregateCondition(condition Condition, outcomes []StimulusOutcome) ConditionAggregate {
	agg := ConditionAggregate{Condition: condition, N: len(outcomes)}
	var latencies []float64
	depthSum := 0
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
		agg.ModelCalls += o.ModelCalls
		agg.DeterministicOps += o.DeterministicOps
		depthSum += o.WorkflowDepth
		latencies = append(latencies, float64(o.LatencyMS))
		for _, st := range o.Steps {
			if st.CognitiveOpsGivenToModel > agg.MaxCognitiveOpsAny {
				agg.MaxCognitiveOpsAny = st.CognitiveOpsGivenToModel
			}
		}
	}
	agg.SemanticAccuracy = decompositionlab.Accuracy(agg.SemanticCorrect, agg.N)
	agg.SemanticCI95Low, agg.SemanticCI95High = decompositionlab.WilsonCI95(agg.SemanticCorrect, agg.N)
	agg.ContractAccuracy = decompositionlab.Accuracy(agg.ContractSuccess, agg.N)
	agg.ContractCI95Low, agg.ContractCI95High = decompositionlab.WilsonCI95(agg.ContractSuccess, agg.N)
	agg.MeanLatencyMS = decompositionlab.Mean(latencies)
	agg.P95LatencyMS = decompositionlab.Percentile95(latencies)
	if agg.N > 0 {
		agg.MeanWorkflowDepth = float64(depthSum) / float64(agg.N)
	}
	return agg
}

func indexByID(outcomes []StimulusOutcome) map[string]StimulusOutcome {
	out := make(map[string]StimulusOutcome, len(outcomes))
	for _, o := range outcomes {
		out[o.ID] = o
	}
	return out
}

// PairTransition pairs two conditions' outcomes by stimulus id and runs the
// exact McNemar comparison (T0 protocol section 11).
func PairTransition(from, to Condition, fromOutcomes, toOutcomes []StimulusOutcome) Transition {
	a, b := indexByID(fromOutcomes), indexByID(toOutcomes)
	ids := make([]string, 0, len(a))
	for id := range a {
		if _, ok := b[id]; ok {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	var sem, con []decompositionlab.PairedOutcome
	for _, id := range ids {
		sem = append(sem, decompositionlab.PairedOutcome{CorrectBefore: a[id].SemanticCorrect, CorrectAfter: b[id].SemanticCorrect})
		con = append(con, decompositionlab.PairedOutcome{CorrectBefore: a[id].ContractSuccess, CorrectAfter: b[id].ContractSuccess})
	}
	t := Transition{From: from, To: to, PairsN: len(ids)}
	t.SemanticMcNemar = decompositionlab.McNemarExact(sem)
	t.ContractMcNemar = decompositionlab.McNemarExact(con)
	t.SemanticDelta = t.SemanticMcNemar.AbsoluteDelta
	t.ContractDelta = t.ContractMcNemar.AbsoluteDelta
	return t
}

// Aggregate builds the full Results from every condition's outcomes.
func Aggregate(datasetSHA string, byCondition map[Condition][]StimulusOutcome) Results {
	res := Results{Schema: ResultsSchemaR0, DatasetSHA: datasetSHA, Conditions: map[string]ConditionAggregate{}, Transitions: map[string]Transition{}}
	for _, c := range AllConditions() {
		res.Conditions[string(c)] = AggregateCondition(c, byCondition[c])
	}
	pairs := []struct {
		name     string
		from, to Condition
	}{
		{"D0_to_D1", ConditionD0Direct, ConditionD1ExternalSeq},
		{"D1_to_D2", ConditionD1ExternalSeq, ConditionD2ExternalOp1},
		{"D2_to_D3", ConditionD2ExternalOp1, ConditionD3Verify},
		{"D0_to_D3", ConditionD0Direct, ConditionD3Verify},
	}
	for _, p := range pairs {
		from, fromOK := byCondition[p.from]
		to, toOK := byCondition[p.to]
		if fromOK && toOK {
			res.Transitions[p.name] = PairTransition(p.from, p.to, from, to)
		}
	}
	return res
}
