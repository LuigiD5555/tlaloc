package tonalt1arms

import (
	"encoding/json"
	"math"
	"sort"
)

// ArmSummary is one arm's accuracy/latency/failure summary.
type ArmSummary struct {
	Arm                string
	N                  int
	SemanticCorrect    int
	ExactCorrect       int
	SemanticAccuracy   float64
	SemanticAccuracyCI [2]float64 // Wilson score interval, 95%
	ExactAccuracy      float64
	MeanLatencyMS      float64
	FailureTaxonomy    map[string]int // ContractStatus -> count
}

// PairedComparison is a McNemar test result between two arms over the same
// workflow set.
type PairedComparison struct {
	ArmX, ArmY       string
	Discordant01     int // X wrong, Y right
	Discordant10     int // X right, Y wrong
	McNemarStatistic float64
	Note             string
}

// CallEconomics summarizes RunAccounting into the planned-vs-attempted
// distinction correction G requires.
type CallEconomics struct {
	PlannedModelCallSlots int
	HTTPRequestAttempts   int
	ValidCompletions      int
	TransportFailures     int
	SchemaFailures        int
	ModelContractFailures int
	BlockedByDependency   int
}

// CounterfactualSummary aggregates a slice of BlackboardCounterfactualOutcome.
type CounterfactualSummary struct {
	Total                         int
	TerminalChanged               int
	FailedClosed                  int
	PrimaryObservationUnavailable int
}

// AnalysisReport is the deterministic analyzer's complete output.
type AnalysisReport struct {
	SchemaVersion         string
	ArmSummaries          []ArmSummary // sorted by Arm
	PairedComparisons     []PairedComparison
	CallEconomics         CallEconomics
	CounterfactualSummary CounterfactualSummary
}

// Analyze reads a frozen RunResult (and optionally a counterfactual outcome
// slice) and produces a deterministic AnalysisReport. It makes zero model
// calls and reads no gold-bearing artifact beyond what's already baked into
// WorkflowRecord.SemanticCorrect/ExactCorrect (computed by the scorer at
// execution time, not re-derived here).
func Analyze(result RunResult, counterfactualOutcomes []BlackboardCounterfactualOutcome) AnalysisReport {
	report := AnalysisReport{SchemaVersion: "tonal.t1.analysis.r1"}

	byArm := map[string][]WorkflowRecord{}
	for _, rec := range result.WorkflowRecords {
		byArm[rec.Arm] = append(byArm[rec.Arm], rec)
	}
	arms := make([]string, 0, len(byArm))
	for arm := range byArm {
		arms = append(arms, arm)
	}
	sort.Strings(arms)

	for _, arm := range arms {
		report.ArmSummaries = append(report.ArmSummaries, summarizeArm(arm, byArm[arm]))
	}

	pairs := [][2]string{{"A", "B"}, {"A", "C"}, {"B", "C"}}
	for _, pair := range pairs {
		x, ok1 := byArm[pair[0]]
		y, ok2 := byArm[pair[1]]
		if !ok1 || !ok2 {
			continue
		}
		report.PairedComparisons = append(report.PairedComparisons, pairedMcNemar(pair[0], x, pair[1], y))
	}

	report.CallEconomics = CallEconomics{
		PlannedModelCallSlots: result.Accounting.PlannedModelCallSlots,
		HTTPRequestAttempts:   result.Accounting.HTTPRequestAttempts,
		ValidCompletions:      result.Accounting.ValidCompletions,
		TransportFailures:     result.Accounting.TransportFailures,
		SchemaFailures:        result.Accounting.SchemaFailures,
		ModelContractFailures: result.Accounting.ModelContractFailures,
		BlockedByDependency:   result.Accounting.BlockedByDependency,
	}

	for _, outcome := range counterfactualOutcomes {
		report.CounterfactualSummary.Total++
		if outcome.PrimaryObservationUnavailable {
			report.CounterfactualSummary.PrimaryObservationUnavailable++
			continue
		}
		if outcome.FailedClosed {
			report.CounterfactualSummary.FailedClosed++
		}
		if outcome.TerminalChanged {
			report.CounterfactualSummary.TerminalChanged++
		}
	}

	return report
}

func summarizeArm(arm string, records []WorkflowRecord) ArmSummary {
	summary := ArmSummary{Arm: arm, N: len(records), FailureTaxonomy: map[string]int{}}
	var totalLatency int64
	for _, rec := range records {
		if rec.SemanticCorrect {
			summary.SemanticCorrect++
		}
		if rec.ExactCorrect {
			summary.ExactCorrect++
		}
		if rec.ContractStatus != "OK" {
			summary.FailureTaxonomy[rec.ContractStatus]++
		}
		totalLatency += rec.LatencyMS
	}
	if summary.N > 0 {
		summary.SemanticAccuracy = float64(summary.SemanticCorrect) / float64(summary.N)
		summary.ExactAccuracy = float64(summary.ExactCorrect) / float64(summary.N)
		summary.MeanLatencyMS = float64(totalLatency) / float64(summary.N)
		summary.SemanticAccuracyCI = wilsonScoreInterval(summary.SemanticCorrect, summary.N)
	}
	return summary
}

// wilsonScoreInterval computes a 95% Wilson score confidence interval for a
// binomial proportion successes/n -- a standard, stdlib-only, deterministic
// closed-form calculation (no iterative/random method).
func wilsonScoreInterval(successes, n int) [2]float64 {
	if n == 0 {
		return [2]float64{0, 0}
	}
	const z = 1.959963984540054 // z-score for 95% CI
	p := float64(successes) / float64(n)
	denom := 1 + z*z/float64(n)
	center := p + z*z/(2*float64(n))
	margin := z * math.Sqrt(p*(1-p)/float64(n)+z*z/(4*float64(n)*float64(n)))
	lower := (center - margin) / denom
	upper := (center + margin) / denom
	if lower < 0 {
		lower = 0
	}
	if upper > 1 {
		upper = 1
	}
	return [2]float64{lower, upper}
}

// pairedMcNemar computes McNemar's test statistic for paired binary outcomes
// (SemanticCorrect) between two arms over the same workflow IDs.
func pairedMcNemar(armX string, x []WorkflowRecord, armY string, y []WorkflowRecord) PairedComparison {
	yByWorkflow := make(map[string]bool, len(y))
	for _, rec := range y {
		yByWorkflow[rec.WorkflowID] = rec.SemanticCorrect
	}

	var b, c int // b: X wrong Y right (discordant "01"); c: X right Y wrong (discordant "10")
	matched := 0
	for _, xRec := range x {
		yCorrect, ok := yByWorkflow[xRec.WorkflowID]
		if !ok {
			continue
		}
		matched++
		switch {
		case !xRec.SemanticCorrect && yCorrect:
			b++
		case xRec.SemanticCorrect && !yCorrect:
			c++
		}
	}

	comparison := PairedComparison{ArmX: armX, ArmY: armY, Discordant01: b, Discordant10: c}
	if b+c == 0 {
		comparison.Note = "no discordant pairs"
		return comparison
	}
	comparison.McNemarStatistic = math.Pow(float64(b-c), 2) / float64(b+c)
	return comparison
}

// MarshalDeterministicJSON marshals the report with sorted map keys
// (encoding/json already sorts map keys) and stable field order (Go struct
// field order is fixed), so calling this twice on the same AnalysisReport
// produces byte-identical output -- the required determinism proof.
func (r AnalysisReport) MarshalDeterministicJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
