package perceptenvelope

import (
	"math"
	"sort"
)

// R1-F stability aggregation (protocol §8-§14).

func empiricalEntropy(items []string) float64 {
	if len(items) == 0 {
		return 0
	}
	counts := map[string]int{}
	for _, it := range items {
		counts[it]++
	}
	n := float64(len(items))
	h := 0.0
	for _, c := range counts {
		p := float64(c) / n
		h -= p * math.Log2(p)
	}
	return h
}

func distinctCount(items []string) int {
	seen := map[string]struct{}{}
	for _, it := range items {
		seen[it] = struct{}{}
	}
	return len(seen)
}

func modeFrequency(items []string) (string, int) {
	counts := map[string]int{}
	for _, it := range items {
		counts[it]++
	}
	best, freq := "", 0
	// deterministic: iterate sorted keys
	var keys []string
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if counts[k] > freq {
			best, freq = k, counts[k]
		}
	}
	return best, freq
}

func flipCount(seq []bool) int {
	n := 0
	for i := 1; i < len(seq); i++ {
		if seq[i] != seq[i-1] {
			n++
		}
	}
	return n
}

func allTrue(seq []bool) bool {
	for _, v := range seq {
		if !v {
			return false
		}
	}
	return len(seq) > 0
}

func allFalse(seq []bool) bool {
	for _, v := range seq {
		if v {
			return false
		}
	}
	return len(seq) > 0
}

func countTrue(seq []bool) int {
	n := 0
	for _, v := range seq {
		if v {
			n++
		}
	}
	return n
}

func boolPtr(b bool) *bool { return &b }

// R1FSentinelStability is the per-sentinel stability record (§8, §10, §14).
type R1FSentinelStability struct {
	SentinelID                            string   `json:"sentinel_id"`
	Stratum                               string   `json:"stratum"`
	StratumName                           string   `json:"stratum_name"`
	SourceStage                           string   `json:"source_stage"`
	SourceCondition                       string   `json:"source_condition"`
	BaseID                                string   `json:"base_id"`
	Capability                            string   `json:"capability"`
	HasImage                              bool     `json:"has_image"`
	ImageSHA256                           string   `json:"image_sha256"`
	PromptSHA256                          string   `json:"prompt_sha256"`
	Gold                                  string   `json:"gold"`
	PrevRawOutput                         string   `json:"previous_raw_output"`
	PrevSemanticCorrect                   bool     `json:"previous_semantic_correct"`
	Repeats                               int      `json:"repeats"`
	RawOutputs                            []string `json:"raw_outputs"`
	NormalizedOutputs                     []string `json:"normalized_outputs"`
	SemanticOutcomeSequence               []bool   `json:"semantic_outcome_sequence"`
	ContractOutcomeSequence               []bool   `json:"contract_outcome_sequence"`
	ExactRawDistinctCount                 int      `json:"exact_raw_distinct_count"`
	ExactRawModeFrequency                 int      `json:"exact_raw_mode_frequency"`
	ExactRawAllIdentical                  bool     `json:"exact_raw_all_identical"`
	NormalizedOutputDistinctCount         int      `json:"normalized_output_distinct_count"`
	SemanticFlipCount                     int      `json:"semantic_flip_count"`
	ContractFlipCount                     int      `json:"contract_flip_count"`
	PreviousSourceOutputMatchesRepeatMode bool     `json:"previous_source_output_matches_repeat_mode"`
	HRaw                                  float64  `json:"h_raw"`
	HSemantic                             float64  `json:"h_semantic"`
	MeanLatencyMS                         float64  `json:"mean_latency_ms"`
	P95LatencyMS                          float64  `json:"p95_latency_ms"`
	SourceWasCorrect                      bool     `json:"source_was_correct"`
	AnyRetryCorrect                       *bool    `json:"any_retry_correct,omitempty"`
	MajorityRetryCorrect                  *bool    `json:"majority_retry_correct,omitempty"`
	AllRetriesWrong                       *bool    `json:"all_retries_wrong,omitempty"`
	AnyRetryWrong                         *bool    `json:"any_retry_wrong,omitempty"`
	MajorityRetryWrong                    *bool    `json:"majority_retry_wrong,omitempty"`
	OutputStabilityClass                  string   `json:"output_stability_class"`
	Errors                                int      `json:"errors"`
}

// R1FSentinelTable is the full per-sentinel result.
type R1FSentinelTable struct {
	Schema       string                 `json:"schema"`
	ExperimentID string                 `json:"experiment_id"`
	Sentinels    []R1FSentinelStability `json:"sentinels"`
}

const r1fSentinelTableSchema = "tlaloc.parrot-perceptual-envelope-r1.r1f-sentinel-table.r1"

func buildSentinelStability(s R1FSentinel, recs []R1FRecord) R1FSentinelStability {
	sort.Slice(recs, func(i, j int) bool { return recs[i].RepeatIndex < recs[j].RepeatIndex })
	st := R1FSentinelStability{
		SentinelID: s.SentinelID, Stratum: s.Stratum, StratumName: s.StratumName,
		SourceStage: s.SourceStage, SourceCondition: s.SourceCondition, BaseID: s.BaseID,
		Capability: s.Capability, HasImage: s.HasImage, ImageSHA256: s.ImageSHA256,
		PromptSHA256: s.PromptSHA256, Gold: s.Gold, PrevRawOutput: s.PrevRawOutput,
		PrevSemanticCorrect: s.PrevSemanticCorrect, SourceWasCorrect: s.PrevSemanticCorrect,
		Repeats: len(recs),
	}
	var lat []float64
	for _, r := range recs {
		if r.Error != "" {
			st.Errors++
			continue
		}
		st.RawOutputs = append(st.RawOutputs, r.RawText)
		st.NormalizedOutputs = append(st.NormalizedOutputs, r.NormalizedValue)
		st.SemanticOutcomeSequence = append(st.SemanticOutcomeSequence, r.SemanticCorrect)
		st.ContractOutcomeSequence = append(st.ContractOutcomeSequence, r.ContractSuccess)
		lat = append(lat, float64(r.LatencyMS))
	}
	st.ExactRawDistinctCount = distinctCount(st.RawOutputs)
	mode, freq := modeFrequency(st.RawOutputs)
	st.ExactRawModeFrequency = freq
	st.ExactRawAllIdentical = st.ExactRawDistinctCount == 1 && len(st.RawOutputs) > 0
	st.NormalizedOutputDistinctCount = distinctCount(st.NormalizedOutputs)
	st.SemanticFlipCount = flipCount(st.SemanticOutcomeSequence)
	st.ContractFlipCount = flipCount(st.ContractOutcomeSequence)
	st.PreviousSourceOutputMatchesRepeatMode = mode == s.PrevRawOutput
	st.HRaw = empiricalEntropy(st.RawOutputs)
	st.HSemantic = empiricalEntropy(st.NormalizedOutputs)
	if len(lat) > 0 {
		sum := 0.0
		for _, v := range lat {
			sum += v
		}
		st.MeanLatencyMS = sum / float64(len(lat))
		st.P95LatencyMS = percentile(lat, 0.95)
	}

	// retry metrics (§10)
	if !s.PrevSemanticCorrect {
		any := countTrue(st.SemanticOutcomeSequence) >= 1
		maj := countTrue(st.SemanticOutcomeSequence) >= 3
		allW := allFalse(st.SemanticOutcomeSequence)
		st.AnyRetryCorrect = boolPtr(any)
		st.MajorityRetryCorrect = boolPtr(maj)
		st.AllRetriesWrong = boolPtr(allW)
	} else {
		anyW := countTrue(st.SemanticOutcomeSequence) < len(st.SemanticOutcomeSequence)
		majW := (len(st.SemanticOutcomeSequence) - countTrue(st.SemanticOutcomeSequence)) >= 3
		st.AnyRetryWrong = boolPtr(anyW)
		st.MajorityRetryWrong = boolPtr(majW)
	}

	// output-stability classification (§14)
	switch {
	case st.ExactRawAllIdentical:
		st.OutputStabilityClass = "BYTE_STABLE"
	case st.NormalizedOutputDistinctCount == 1 && flipCount(st.SemanticOutcomeSequence) == 0:
		st.OutputStabilityClass = "SEMANTICALLY_STABLE"
	default:
		st.OutputStabilityClass = "SEMANTICALLY_VARIABLE"
	}
	return st
}

// R1FStratumAgg is one stratum aggregate (§11).
type R1FStratumAgg struct {
	Stratum                          string  `json:"stratum"`
	StratumName                      string  `json:"stratum_name"`
	Sentinels                        int     `json:"sentinels"`
	Calls                            int     `json:"calls"`
	ExactRawAllIdenticalRate         float64 `json:"exact_raw_all_identical_rate"`
	MeanRawDistinctOutputs           float64 `json:"mean_raw_distinct_outputs"`
	SemanticStabilityRate            float64 `json:"semantic_stability_rate"`
	ContractStabilityRate            float64 `json:"contract_stability_rate"`
	WrongSourceRecoveredAtLeastOnce  int     `json:"wrong_source_recovered_at_least_once"`
	CorrectSourceDegradedAtLeastOnce int     `json:"correct_source_degraded_at_least_once"`
	MeanLatencyMS                    float64 `json:"mean_latency_ms"`
	P95LatencyMS                     float64 `json:"p95_latency_ms"`
}

// R1FStabilitySummary is the global stability + decision result (§12, §13).
type R1FStabilitySummary struct {
	Schema                       string          `json:"schema"`
	ExperimentID                 string          `json:"experiment_id"`
	SentinelPosthocSelectionForStability bool     `json:"SENTINEL_POSTHOC_SELECTION_FOR_STABILITY"`
	Sentinels                    int             `json:"sentinels"`
	Calls                        int             `json:"calls"`
	Strata                       []R1FStratumAgg `json:"strata"`
	RawDistinct1                 int             `json:"sentinels_with_1_distinct_raw"`
	RawDistinct2                 int             `json:"sentinels_with_2_distinct_raw"`
	RawDistinct3Plus             int             `json:"sentinels_with_3plus_distinct_raw"`
	SemanticInvariant5of5        int             `json:"sentinels_semantic_invariant_5of5"`
	SemanticVariable             int             `json:"sentinels_semantic_variable"`
	ByteIdenticalWithinSentinelPairRate float64  `json:"byte_identical_within_sentinel_pair_rate"`
	ByteStable                   int             `json:"class_byte_stable"`
	SemanticallyStable           int             `json:"class_semantically_stable"`
	SemanticallyVariable         int             `json:"class_semantically_variable"`
	PreviouslyWrongSentinels     int             `json:"previously_wrong_sentinels"`
	PreviouslyWrongRemainAllWrong int            `json:"previously_wrong_remain_all_wrong"`
	FracWrongRemainWrong         float64         `json:"frac_previously_wrong_remain_all_wrong"`
	FracSemanticInvariant        float64         `json:"frac_semantic_invariant_5of5"`
	AnyExactRetryRecoveries      int             `json:"any_exact_retry_recoveries"`
	AnyExactRetryDegradations    int             `json:"any_exact_retry_degradations"`
	DecisionRuleFrozenPreInference string        `json:"decision_rule_frozen_pre_inference"`
	BlindRetryNotUseful          bool            `json:"BLIND_RETRY_NOT_USEFUL"`
}

const r1fStabilitySummarySchema = "tlaloc.parrot-perceptual-envelope-r1.r1f-stability-summary.r1"

// AggregateR1F builds the per-sentinel table + the global stability summary.
func AggregateR1F(records []R1FRecord, ds R1FDataset) (R1FSentinelTable, R1FStabilitySummary) {
	byS := map[string][]R1FRecord{}
	for _, r := range records {
		byS[r.SentinelID] = append(byS[r.SentinelID], r)
	}
	table := R1FSentinelTable{Schema: r1fSentinelTableSchema, ExperimentID: ExperimentID}
	stByID := map[string]R1FSentinelStability{}
	for _, s := range ds.Sentinels {
		st := buildSentinelStability(s, byS[s.SentinelID])
		table.Sentinels = append(table.Sentinels, st)
		stByID[s.SentinelID] = st
	}

	sum := R1FStabilitySummary{
		Schema: r1fStabilitySummarySchema, ExperimentID: ExperimentID,
		SentinelPosthocSelectionForStability: true,
		Sentinels:                            len(ds.Sentinels),
		Calls:                                len(records),
		DecisionRuleFrozenPreInference:       ds.DecisionRule,
	}

	// per stratum
	pairIdentical, pairTotal := 0, 0
	for _, strat := range R1FStrata {
		agg := R1FStratumAgg{Stratum: strat.Key, StratumName: strat.Name}
		var allIdent, semStable, conStable int
		var distinctSum float64
		var lat []float64
		for _, st := range table.Sentinels {
			if st.Stratum != strat.Key {
				continue
			}
			agg.Sentinels++
			agg.Calls += len(st.RawOutputs)
			if st.ExactRawAllIdentical {
				allIdent++
			}
			distinctSum += float64(st.ExactRawDistinctCount)
			if flipCount(st.SemanticOutcomeSequence) == 0 {
				semStable++
			}
			if flipCount(st.ContractOutcomeSequence) == 0 {
				conStable++
			}
			if !st.SourceWasCorrect && st.AnyRetryCorrect != nil && *st.AnyRetryCorrect {
				agg.WrongSourceRecoveredAtLeastOnce++
			}
			if st.SourceWasCorrect && st.AnyRetryWrong != nil && *st.AnyRetryWrong {
				agg.CorrectSourceDegradedAtLeastOnce++
			}
			lat = append(lat, st.MeanLatencyMS)
			// within-sentinel byte-identical pair rate
			for i := 0; i < len(st.RawOutputs); i++ {
				for j := i + 1; j < len(st.RawOutputs); j++ {
					pairTotal++
					if st.RawOutputs[i] == st.RawOutputs[j] {
						pairIdentical++
					}
				}
			}
		}
		if agg.Sentinels > 0 {
			agg.ExactRawAllIdenticalRate = float64(allIdent) / float64(agg.Sentinels)
			agg.MeanRawDistinctOutputs = distinctSum / float64(agg.Sentinels)
			agg.SemanticStabilityRate = float64(semStable) / float64(agg.Sentinels)
			agg.ContractStabilityRate = float64(conStable) / float64(agg.Sentinels)
		}
		if len(lat) > 0 {
			s := 0.0
			for _, v := range lat {
				s += v
			}
			agg.MeanLatencyMS = s / float64(len(lat))
			agg.P95LatencyMS = percentile(lat, 0.95)
		}
		sum.Strata = append(sum.Strata, agg)
	}
	if pairTotal > 0 {
		sum.ByteIdenticalWithinSentinelPairRate = float64(pairIdentical) / float64(pairTotal)
	}

	// global counts
	for _, st := range table.Sentinels {
		switch {
		case st.ExactRawDistinctCount <= 1:
			sum.RawDistinct1++
		case st.ExactRawDistinctCount == 2:
			sum.RawDistinct2++
		default:
			sum.RawDistinct3Plus++
		}
		if flipCount(st.SemanticOutcomeSequence) == 0 {
			sum.SemanticInvariant5of5++
		} else {
			sum.SemanticVariable++
		}
		switch st.OutputStabilityClass {
		case "BYTE_STABLE":
			sum.ByteStable++
		case "SEMANTICALLY_STABLE":
			sum.SemanticallyStable++
		default:
			sum.SemanticallyVariable++
		}
		if !st.SourceWasCorrect {
			sum.PreviouslyWrongSentinels++
			if allFalse(st.SemanticOutcomeSequence) {
				sum.PreviouslyWrongRemainAllWrong++
			}
			if st.AnyRetryCorrect != nil && *st.AnyRetryCorrect {
				sum.AnyExactRetryRecoveries++
			}
		} else {
			if st.AnyRetryWrong != nil && *st.AnyRetryWrong {
				sum.AnyExactRetryDegradations++
			}
		}
	}
	if sum.PreviouslyWrongSentinels > 0 {
		sum.FracWrongRemainWrong = float64(sum.PreviouslyWrongRemainAllWrong) / float64(sum.PreviouslyWrongSentinels)
	}
	if sum.Sentinels > 0 {
		sum.FracSemanticInvariant = float64(sum.SemanticInvariant5of5) / float64(sum.Sentinels)
	}
	sum.BlindRetryNotUseful = sum.FracWrongRemainWrong >= r1fWrongStayWrongThreshold &&
		sum.FracSemanticInvariant >= r1fSemanticInvariantThreshold
	return table, sum
}
