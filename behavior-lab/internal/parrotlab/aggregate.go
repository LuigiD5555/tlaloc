package parrotlab

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/tlaloque/calibration"
)

// GroupStat is the metric block for one slice of run records (SPEC §12, §19).
// "Correct" counts contract successes (right content AND requested form);
// SemanticAccuracy forgives the form. Accuracy and its Wilson interval are
// over attempted answers (abstentions excluded), except the "abstain" task
// family where an abstention is the correct behaviour.
type GroupStat struct {
	N                        int     `json:"n"`
	Attempted                int     `json:"attempted"`
	Correct                  int     `json:"correct"`
	Accuracy                 float64 `json:"accuracy"`
	CI95Low                  float64 `json:"ci95_low"`
	CI95High                 float64 `json:"ci95_high"`
	SemanticAccuracy         float64 `json:"semantic_accuracy"`
	AbstentionRate           float64 `json:"abstention_rate"`
	FormatFailureRate        float64 `json:"format_failure_rate"`
	UnsupportedAssertionRate float64 `json:"unsupported_assertion_rate"`
	MeanLatencyMS            int64   `json:"mean_latency_ms"`
	P95LatencyMS             int64   `json:"p95_latency_ms"`
	MeanTokensOut            float64 `json:"mean_tokens_out"`
	CostMeasured             bool    `json:"cost_measured"`
	Errors                   int     `json:"errors"`
}

// StageResult is the aggregate document written to results/<stage>.json.
type StageResult struct {
	ExperimentID string                `json:"experiment_id"`
	Stage        string                `json:"stage"`
	Summary      GroupStat             `json:"summary"`
	Groups       map[string]GroupStat  `json:"groups,omitempty"`
	Cliff        *CliffResult          `json:"instruction_cliff,omitempty"`
	Interference []PairInterference    `json:"pair_interference,omitempty"`
	Blackboard   *BlackboardResult     `json:"blackboard,omitempty"`
	Capability   map[string]CapVerdict `json:"capability_class,omitempty"`
}

// CliffResult is the SPEC §13 verdict. The cliff is decided by the paired
// (McNemar) transition test, not by non-overlapping Wilson intervals: the
// levels share the same 40 stimuli, so a paired test is both correct and
// more powerful (P-1 fix #3). Wilson per-level stays descriptive.
type CliffResult struct {
	// Contract == right content AND requested form. Semantic == right
	// content, form forgiven. The two safe-ops limits (SPEC §8) can differ:
	// a contract cliff with no semantic cliff means "Parrot knows but won't
	// emit the bare answer" -> externalise format/control, not decompose.
	Detected            bool                 `json:"detected"`
	Level               int                  `json:"level,omitempty"`
	MaxSafeOps          int                  `json:"parrot_max_safe_ops"` // contract; kept for compatibility
	MaxSafeOpsContract  int                  `json:"parrot_max_safe_ops_contract"`
	MaxSafeOpsSemantic  int                  `json:"parrot_max_safe_ops_semantic"`
	SemanticDetected    bool                 `json:"semantic_cliff_detected"`
	SemanticLevel       int                  `json:"semantic_cliff_level,omitempty"`
	ByOperationDepth    map[string]LevelStat `json:"by_operation_depth"`
	Transitions         []PairedTransition   `json:"paired_transitions_contract"`
	SemanticTransitions []PairedTransition   `json:"paired_transitions_semantic"`
	ByPrimitive         map[string]GroupStat `json:"by_added_primitive"`
	ByPriorContract     map[string]GroupStat `json:"by_prior_contract"`
	ConfoundNote        string               `json:"confound_note,omitempty"`
}

type LevelStat struct {
	GroupStat
	AccuracyDeltaFromOP1 float64 `json:"accuracy_delta_from_op1"`
}

// PairedTransition is the depth N-1 -> N comparison over the stimuli that
// were run at both depths. b = held at N-1 then failed at N; c = failed at
// N-1 then held at N. PValue is the two-sided exact McNemar (binomial)
// probability; Significant is PValue < 0.05.
type PairedTransition struct {
	From          int     `json:"from"`
	To            int     `json:"to"`
	PairsN        int     `json:"pairs_n"`
	Regressions   int     `json:"regressions"`
	Gains         int     `json:"gains"`
	DeltaAccuracy float64 `json:"delta_accuracy"`
	PValue        float64 `json:"p_value"`
	Significant   bool    `json:"significant"`
}

// CapVerdict is the SPEC §20 classification.
type CapVerdict struct {
	Class                string   `json:"class"`
	Accuracy             float64  `json:"accuracy"`
	CI95Low              float64  `json:"ci95_low"`
	CI95High             float64  `json:"ci95_high"`
	ExternalizeCandidate bool     `json:"externalize_candidate"`
	Reasons              []string `json:"reasons,omitempty"`
}

// PairInterference is SPEC §23–24 for one capability pair.
type PairInterference struct {
	Pair             [2]string `json:"pair"`
	SingleA          float64   `json:"single_a"`
	SingleB          float64   `json:"single_b"`
	Combined         float64   `json:"combined"`
	InterferenceA    float64   `json:"interference_a"`
	InterferenceB    float64   `json:"interference_b"`
	PairInterference float64   `json:"pair_interference"`
	Category         string    `json:"category"`
}

// BlackboardResult is SPEC §28. wrong_hint_acceptance_rate is approximated
// for R0 as the error rate under the incorrect-hint condition (the model
// produced a wrong answer while a wrong hint was present).
type BlackboardResult struct {
	ByCondition             map[string]GroupStat `json:"by_condition"`
	HintFollowRate          float64              `json:"hint_follow_rate"`
	WrongHintAcceptanceRate float64              `json:"wrong_hint_acceptance_rate"`
	AccuracyLiftFromCorrect float64              `json:"accuracy_lift_from_correct_hint"`
}

// LoadRunRecords reads runs/<stage>/<stage>.jsonl.
func LoadRunRecords(exp *Experiment, stage string) ([]RunRecord, error) {
	path := filepath.Join(exp.Root, "runs", stage, stage+".jsonl")
	handle, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	var records []RunRecord
	scanner := bufio.NewScanner(handle)
	scanner.Buffer(make([]byte, 64*1024), 8<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record RunRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, scanner.Err()
}

func computeGroup(records []RunRecord) GroupStat {
	stat := GroupStat{N: len(records)}
	var latencies []int64
	var tokenSum, latencySum int64
	semanticCorrect := 0
	for _, record := range records {
		if record.Error != "" {
			stat.Errors++
			continue
		}
		latencies = append(latencies, record.Resources.WallMS)
		latencySum += record.Resources.WallMS
		tokenSum += int64(record.Resources.TokensOut)
		if record.Resources.RAMMeasured || record.Resources.VRAMMeasured || record.Resources.CPUMeasured {
			stat.CostMeasured = true
		}
		abstainFamily := record.TaskFamily == "abstain"
		switch {
		case record.Score.Abstained && !abstainFamily:
			stat.AbstentionRate++
		default:
			stat.Attempted++
			if record.Score.ContractSuccess {
				stat.Correct++
			}
			if record.Score.SemanticCorrect {
				semanticCorrect++
			}
		}
		if !record.Score.FormatValid {
			stat.FormatFailureRate++
		}
		if record.Score.UnsupportedAssertion {
			stat.UnsupportedAssertionRate++
		}
	}
	scored := stat.N - stat.Errors
	if scored > 0 {
		stat.AbstentionRate /= float64(scored)
		stat.FormatFailureRate /= float64(scored)
		stat.UnsupportedAssertionRate /= float64(scored)
		stat.MeanTokensOut = float64(tokenSum) / float64(scored)
	}
	if stat.Attempted > 0 {
		stat.SemanticAccuracy = float64(semanticCorrect) / float64(stat.Attempted)
	}
	if len(latencies) > 0 {
		stat.MeanLatencyMS = latencySum / int64(len(latencies))
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		index := int(float64(len(latencies)-1) * 0.95)
		stat.P95LatencyMS = latencies[index]
	}
	interval := calibration.WilsonInterval(stat.Correct, stat.Attempted)
	stat.Accuracy = interval.Proportion
	stat.CI95Low = interval.Low
	stat.CI95High = interval.High
	return stat
}

// Aggregate builds the StageResult for a stage from its run records.
func Aggregate(exp *Experiment, stage string) (StageResult, error) {
	records, err := LoadRunRecords(exp, stage)
	if err != nil {
		return StageResult{}, err
	}
	if len(records) == 0 {
		return StageResult{}, fmt.Errorf("no run records for stage %q", stage)
	}
	result := StageResult{
		ExperimentID: exp.Manifest.ExperimentID,
		Stage:        stage,
		Summary:      computeGroup(records),
	}

	switch stage {
	case StageInstructionCliff:
		result.Cliff = buildCliff(records, exp.Manifest.Thresholds.InstructionCliffDropPP)
	case StageSingles:
		result.Groups, result.Capability = buildCapabilities(records, exp.Manifest.Thresholds)
	case StageCoalitions:
		result.Groups = groupBy(records, func(r RunRecord) string {
			if r.BaseID != "" {
				return r.BaseID
			}
			return capabilityKey(r.Capabilities)
		})
	case StageInterference:
		result.Groups = groupBy(records, func(r RunRecord) string { return capabilityKey(r.Capabilities) })
		result.Interference = buildInterference(exp, records)
	case StageBlackboard:
		result.Blackboard = buildBlackboard(records)
		result.Groups = groupBy(records, func(r RunRecord) string { return r.HintCondition })
	case StageEndToEnd:
		result.Groups = groupBy(records, func(r RunRecord) string { return r.TaskFamily })
	}
	return result, nil
}

func groupBy(records []RunRecord, key func(RunRecord) string) map[string]GroupStat {
	buckets := map[string][]RunRecord{}
	for _, record := range records {
		buckets[key(record)] = append(buckets[key(record)], record)
	}
	out := make(map[string]GroupStat, len(buckets))
	for name, bucket := range buckets {
		out[name] = computeGroup(bucket)
	}
	return out
}

func capabilityKey(caps []string) string {
	sorted := append([]string(nil), caps...)
	sort.Strings(sorted)
	return strings.Join(sorted, "+")
}

func buildCliff(records []RunRecord, dropPP float64) *CliffResult {
	byDepth := map[int][]RunRecord{}
	contractByBaseDepth := map[string]map[int]bool{}
	semanticByBaseDepth := map[string]map[int]bool{}
	maxDepth := 0
	accumulate := func(store map[string]map[int]bool, base string, depth int, value bool) {
		if store[base] == nil {
			store[base] = map[int]bool{}
		}
		if prior, seen := store[base][depth]; seen {
			value = prior && value // with repetitions: held iff every rep held
		}
		store[base][depth] = value
	}
	for _, record := range records {
		byDepth[record.Operations] = append(byDepth[record.Operations], record)
		if record.Operations > maxDepth {
			maxDepth = record.Operations
		}
		if record.BaseID == "" {
			continue
		}
		accumulate(contractByBaseDepth, record.BaseID, record.Operations, record.Score.ContractSuccess)
		accumulate(semanticByBaseDepth, record.BaseID, record.Operations, record.Score.SemanticCorrect)
	}

	cliff := &CliffResult{
		ByOperationDepth:   map[string]LevelStat{},
		MaxSafeOps:         maxDepth,
		MaxSafeOpsContract: maxDepth,
		MaxSafeOpsSemantic: maxDepth,
	}
	stats := map[int]GroupStat{}
	for depth, bucket := range byDepth {
		stats[depth] = computeGroup(bucket)
	}
	base := stats[1].Accuracy
	for depth := 1; depth <= maxDepth; depth++ {
		stat, ok := stats[depth]
		if !ok {
			continue
		}
		cliff.ByOperationDepth[fmt.Sprintf("%d", depth)] = LevelStat{
			GroupStat:            stat,
			AccuracyDeltaFromOP1: stat.Accuracy - base,
		}
	}

	// Per-added-primitive breakdown, so a collapse at depth N can be told
	// apart from one specific capability being weak (P-1 fix #4).
	cliff.ByPrimitive = groupBy(records, func(record RunRecord) string {
		if record.AddedPrimitive == "" {
			return "unlabelled"
		}
		return record.AddedPrimitive
	})
	cliff.ByPriorContract = groupBy(records, func(record RunRecord) string {
		if record.PriorContract {
			return "earlier_step_imposed_a_different_output_form"
		}
		return "final_step_form_only"
	})
	cliff.ConfoundNote = primitiveConfoundNote(cliff.ByPrimitive)

	threshold := dropPP / 100.0
	for depth := 2; depth <= maxDepth; depth++ {
		contract := pairedTransition(depth-1, depth, contractByBaseDepth)
		semantic := pairedTransition(depth-1, depth, semanticByBaseDepth)
		cliff.Transitions = append(cliff.Transitions, contract)
		cliff.SemanticTransitions = append(cliff.SemanticTransitions, semantic)
		if !cliff.Detected && -contract.DeltaAccuracy >= threshold && contract.Significant {
			cliff.Detected = true
			cliff.Level = depth
			cliff.MaxSafeOps = depth - 1
			cliff.MaxSafeOpsContract = depth - 1
		}
		if !cliff.SemanticDetected && -semantic.DeltaAccuracy >= threshold && semantic.Significant {
			cliff.SemanticDetected = true
			cliff.SemanticLevel = depth
			cliff.MaxSafeOpsSemantic = depth - 1
		}
	}
	return cliff
}

// pairedTransition runs the McNemar comparison of two depths over the
// stimuli present at both.
func pairedTransition(from, to int, correctByBaseDepth map[string]map[int]bool) PairedTransition {
	transition := PairedTransition{From: from, To: to}
	for _, byDepth := range correctByBaseDepth {
		fromCorrect, hasFrom := byDepth[from]
		toCorrect, hasTo := byDepth[to]
		if !hasFrom || !hasTo {
			continue
		}
		transition.PairsN++
		switch {
		case fromCorrect && !toCorrect:
			transition.Regressions++
		case !fromCorrect && toCorrect:
			transition.Gains++
		}
	}
	if transition.PairsN > 0 {
		transition.DeltaAccuracy = float64(transition.Gains-transition.Regressions) / float64(transition.PairsN)
	}
	transition.PValue = mcnemarExactP(transition.Regressions, transition.Gains)
	transition.Significant = transition.PValue < 0.05 && transition.Regressions > transition.Gains
	return transition
}

// mcnemarExactP is the two-sided exact (binomial) McNemar p-value: under
// H0 the discordant pairs split 50/50, so min(b,c) ~ Binomial(b+c, 0.5).
func mcnemarExactP(regressions, gains int) float64 {
	n := regressions + gains
	if n == 0 {
		return 1
	}
	smaller := regressions
	if gains < smaller {
		smaller = gains
	}
	tail := 0.0
	for k := 0; k <= smaller; k++ {
		tail += binomialPMF(n, k, 0.5)
	}
	p := 2 * tail
	if p > 1 {
		p = 1
	}
	return p
}

func binomialPMF(n, k int, prob float64) float64 {
	logCoeff := lgamma(n+1) - lgamma(k+1) - lgamma(n-k+1)
	return math.Exp(logCoeff + float64(k)*math.Log(prob) + float64(n-k)*math.Log(1-prob))
}

func lgamma(value int) float64 {
	result, _ := math.Lgamma(float64(value))
	return result
}

func primitiveConfoundNote(byPrimitive map[string]GroupStat) string {
	worst, worstName := 1.0, ""
	best := 0.0
	for name, stat := range byPrimitive {
		if stat.Attempted == 0 || name == "unlabelled" {
			continue
		}
		if stat.Accuracy < worst {
			worst, worstName = stat.Accuracy, name
		}
		if stat.Accuracy > best {
			best = stat.Accuracy
		}
	}
	if worstName != "" && best-worst >= 0.25 {
		return fmt.Sprintf("added-primitive accuracy spread is %.2f (worst: %s at %.2f) — part of any depth effect may be a single weak capability, not operation count", best-worst, worstName, worst)
	}
	return ""
}

func buildCapabilities(records []RunRecord, thresholds Thresholds) (map[string]GroupStat, map[string]CapVerdict) {
	groups := map[string]GroupStat{}
	verdicts := map[string]CapVerdict{}
	buckets := map[string][]RunRecord{}
	for _, record := range records {
		if len(record.Capabilities) != 1 {
			continue
		}
		buckets[record.Capabilities[0]] = append(buckets[record.Capabilities[0]], record)
	}
	for capability, bucket := range buckets {
		stat := computeGroup(bucket)
		groups[capability] = stat
		verdicts[capability] = classify(capability, stat, thresholds)
	}
	return groups, verdicts
}

func classify(capability string, stat GroupStat, thresholds Thresholds) CapVerdict {
	class := calibration.WilsonInterval(stat.Correct, stat.Attempted)
	cut := thresholds.CapabilityClass
	verdict := CapVerdict{Accuracy: class.Proportion, CI95Low: class.Low, CI95High: class.High}
	switch {
	case class.Low >= cut.StrongLowerCI:
		verdict.Class = "STRONG"
	case class.Low >= cut.UsableLowerCI:
		verdict.Class = "USABLE"
	case class.High < cut.UnusableUpperCI:
		verdict.Class = "UNUSABLE"
	case class.High < cut.WeakUpperCI:
		verdict.Class = "WEAK"
	default:
		verdict.Class = "FRAGILE"
	}
	if verdict.Class == "WEAK" || verdict.Class == "UNUSABLE" {
		verdict.ExternalizeCandidate = true
		verdict.Reasons = append(verdict.Reasons, "weak accuracy")
	}
	if stat.FormatFailureRate >= 0.15 {
		verdict.ExternalizeCandidate = true
		verdict.Reasons = append(verdict.Reasons, "high format-failure rate")
	}
	if stat.UnsupportedAssertionRate >= 0.20 {
		verdict.ExternalizeCandidate = true
		verdict.Reasons = append(verdict.Reasons, "high unsupported-assertion rate")
	}
	return verdict
}

func buildInterference(exp *Experiment, records []RunRecord) []PairInterference {
	singles := loadSingleAccuracies(exp)
	thresholds := exp.Manifest.Thresholds.PairInterference
	byPair := map[string][]RunRecord{}
	for _, record := range records {
		if len(record.Capabilities) != 2 {
			continue
		}
		byPair[capabilityKey(record.Capabilities)] = append(byPair[capabilityKey(record.Capabilities)], record)
	}
	var out []PairInterference
	for _, bucket := range byPair {
		pair := append([]string(nil), bucket[0].Capabilities...)
		sort.Strings(pair)
		combined := computeGroup(bucket).Accuracy
		singleA, okA := singles[pair[0]]
		singleB, okB := singles[pair[1]]
		if !okA || !okB {
			continue
		}
		entry := PairInterference{
			Pair:          [2]string{pair[0], pair[1]},
			SingleA:       singleA,
			SingleB:       singleB,
			Combined:      combined,
			InterferenceA: combined - singleA,
			InterferenceB: combined - singleB,
		}
		entry.PairInterference = (entry.InterferenceA + entry.InterferenceB) / 2
		entry.Category = interferenceCategory(entry.PairInterference, thresholds)
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PairInterference < out[j].PairInterference })
	return out
}

func interferenceCategory(value float64, thresholds PairInterferenceCuts) string {
	switch {
	case value > thresholds.Neutral:
		return "NEUTRAL"
	case value > thresholds.Mild:
		return "MILD"
	case value > thresholds.Moderate:
		return "MODERATE"
	default:
		return "SEVERE"
	}
}

func loadSingleAccuracies(exp *Experiment) map[string]float64 {
	path := filepath.Join(exp.Root, "results", "singles.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return map[string]float64{}
	}
	var result StageResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return map[string]float64{}
	}
	out := map[string]float64{}
	for capability, stat := range result.Groups {
		out[capability] = stat.Accuracy
	}
	return out
}

func buildBlackboard(records []RunRecord) *BlackboardResult {
	byCondition := groupBy(records, func(r RunRecord) string { return r.HintCondition })
	result := &BlackboardResult{ByCondition: byCondition}
	correct := byCondition["correct"]
	none := byCondition["none"]
	incorrect := byCondition["incorrect"]
	result.HintFollowRate = correct.Accuracy
	result.AccuracyLiftFromCorrect = correct.Accuracy - none.Accuracy
	if incorrect.Attempted > 0 {
		result.WrongHintAcceptanceRate = 1 - incorrect.Accuracy
	}
	return result
}

// WriteStageResult persists a StageResult to results/<stage>.json.
func WriteStageResult(exp *Experiment, result StageResult) (string, error) {
	dir := filepath.Join(exp.Root, "results")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, result.Stage+".json")
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, append(raw, '\n'), 0o644)
}
