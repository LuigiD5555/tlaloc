package experimentalspine

import (
	"math"
	"sort"

	"tlaloc.local/behaviorlab/internal/episode"
)

type Summary struct {
	Schema           string `json:"schema"`
	RunID            string `json:"run_id"`
	ParentRunID      string `json:"parent_run_id,omitempty"`
	SourceExperiment string `json:"source_experiment"`
	PrototypeID      string `json:"prototype_id"`
	PrototypeVersion string `json:"prototype_version,omitempty"`
	ParentVersion    string `json:"parent_version,omitempty"`

	Episodes         int     `json:"episodes"`
	Successful       int     `json:"successful"`
	Failed           int     `json:"failed"`
	SemanticCorrect  int     `json:"semantic_correct"`
	ExactCorrect     int     `json:"exact_correct"`
	SemanticAccuracy float64 `json:"semantic_accuracy"`
	ExactAccuracy    float64 `json:"exact_accuracy"`

	Cost                 CostSummary           `json:"cost"`
	Latency              LatencySummary        `json:"latency"`
	FailureRootCauses    []Count               `json:"failure_root_causes,omitempty"`
	ByArm                []EpisodeBreakdown    `json:"by_arm,omitempty"`
	ByFamily             []EpisodeBreakdown    `json:"by_family,omitempty"`
	ByCapability         []CapabilityBreakdown `json:"by_capability,omitempty"`
	MostFailedCapability string                `json:"most_failed_capability,omitempty"`
	NextDebugTarget      string                `json:"next_debug_target,omitempty"`
}

type CostSummary struct {
	PlannedModelCallSlots int   `json:"planned_model_call_slots,omitempty"`
	CompletedTransports   int   `json:"completed_transports"`
	HTTPRequestAttempts   int   `json:"http_request_attempts"`
	ValidCompletions      int   `json:"valid_completions"`
	TransportFailures     int   `json:"transport_failures"`
	SchemaFailures        int   `json:"schema_failures"`
	ModelContractFailures int   `json:"model_contract_failures"`
	BlockedByDependency   int   `json:"blocked_by_dependency"`
	LatencyMS             int64 `json:"latency_ms"`
}

type LatencySummary struct {
	StepSamples int   `json:"step_samples"`
	P50MS       int64 `json:"p50_ms"`
	P95MS       int64 `json:"p95_ms"`
	MaxMS       int64 `json:"max_ms"`
}

type Count struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type EpisodeBreakdown struct {
	Name             string  `json:"name"`
	Episodes         int     `json:"episodes"`
	Successful       int     `json:"successful"`
	Failed           int     `json:"failed"`
	SemanticAccuracy float64 `json:"semantic_accuracy"`
}

type CapabilityBreakdown struct {
	Capability   string `json:"capability"`
	Steps        int    `json:"steps"`
	FailedSteps  int    `json:"failed_steps"`
	BlockedSteps int    `json:"blocked_steps"`
}

// Summarize reduces Episodes only. It never calls a model and never uses an
// LLM-as-judge. The result is deterministic for the same episode set.
func Summarize(manifest RunManifest, episodes []episode.Episode) Summary {
	summary := Summary{
		Schema:           SummarySchema,
		RunID:            manifest.RunID,
		ParentRunID:      manifest.ParentRunID,
		SourceExperiment: manifest.SourceExperiment,
		PrototypeID:      manifest.Prototype.ID,
		PrototypeVersion: manifest.Prototype.Version,
		ParentVersion:    manifest.Prototype.ParentVersion,
		Episodes:         len(episodes),
	}

	rootCounts := map[string]int{}
	armCounts := map[string]*EpisodeBreakdown{}
	familyCounts := map[string]*EpisodeBreakdown{}
	capabilityCounts := map[string]*CapabilityBreakdown{}
	var stepLatencies []int64

	for _, ep := range episodes {
		if ep.Success {
			summary.Successful++
		} else {
			summary.Failed++
		}
		if ep.SemanticCorrect {
			summary.SemanticCorrect++
		}
		if ep.ExactCorrect {
			summary.ExactCorrect++
		}
		if ep.FailureRootCause != "" {
			rootCounts[ep.FailureRootCause]++
		}

		accumulateEpisodeBreakdown(armCounts, ep.Arm, ep.Success, ep.SemanticCorrect)
		accumulateEpisodeBreakdown(familyCounts, ep.Family, ep.Success, ep.SemanticCorrect)

		summary.Cost.CompletedTransports += ep.Cost.ModelCalls
		summary.Cost.HTTPRequestAttempts += ep.Cost.HTTPRequestAttempts
		summary.Cost.ValidCompletions += ep.Cost.ValidCompletions
		summary.Cost.TransportFailures += ep.Cost.TransportFailures
		summary.Cost.SchemaFailures += ep.Cost.SchemaFailures
		summary.Cost.ModelContractFailures += ep.Cost.ModelContractFailures
		summary.Cost.BlockedByDependency += ep.Cost.BlockedByDependency
		summary.Cost.LatencyMS += ep.Cost.LatencyMS

		for _, step := range ep.Steps {
			if step.LatencyMS >= 0 {
				stepLatencies = append(stepLatencies, step.LatencyMS)
			}
			if step.SelectedCapability == "" {
				continue
			}
			entry := capabilityCounts[step.SelectedCapability]
			if entry == nil {
				entry = &CapabilityBreakdown{Capability: step.SelectedCapability}
				capabilityCounts[step.SelectedCapability] = entry
			}
			entry.Steps++
			if step.ContractStatus == "BLOCKED_BY_DEPENDENCY" || step.Status == "BLOCKED_BY_DEPENDENCY" {
				entry.BlockedSteps++
			} else if stepFailed(step) {
				entry.FailedSteps++
			}
		}
	}

	if summary.Episodes > 0 {
		summary.SemanticAccuracy = float64(summary.SemanticCorrect) / float64(summary.Episodes)
		summary.ExactAccuracy = float64(summary.ExactCorrect) / float64(summary.Episodes)
	}

	summary.FailureRootCauses = sortedCounts(rootCounts)
	summary.ByArm = sortedEpisodeBreakdowns(armCounts)
	summary.ByFamily = sortedEpisodeBreakdowns(familyCounts)
	summary.ByCapability = sortedCapabilityBreakdowns(capabilityCounts)
	summary.Latency = summarizeLatency(stepLatencies)

	// Debug priority uses direct failures only. Dependency-blocked nodes are
	// consequences and remain visible as BlockedSteps, but they must not win
	// the "most failed capability" vote merely because one upstream failure
	// fanned out through a DAG.
	for _, item := range summary.ByCapability {
		if item.FailedSteps == 0 {
			continue
		}
		if summary.MostFailedCapability == "" {
			summary.MostFailedCapability = item.Capability
			continue
		}
		current := capabilityCounts[summary.MostFailedCapability]
		if item.FailedSteps > current.FailedSteps || (item.FailedSteps == current.FailedSteps && item.Capability < summary.MostFailedCapability) {
			summary.MostFailedCapability = item.Capability
		}
	}

	if summary.MostFailedCapability != "" {
		summary.NextDebugTarget = "capability:" + summary.MostFailedCapability
	} else if len(summary.FailureRootCauses) > 0 {
		best := summary.FailureRootCauses[0]
		for _, item := range summary.FailureRootCauses[1:] {
			if item.Count > best.Count || (item.Count == best.Count && item.Name < best.Name) {
				best = item
			}
		}
		summary.NextDebugTarget = "failure_root:" + best.Name
	}

	return summary
}

func accumulateEpisodeBreakdown(target map[string]*EpisodeBreakdown, name string, success, semanticCorrect bool) {
	if name == "" {
		return
	}
	entry := target[name]
	if entry == nil {
		entry = &EpisodeBreakdown{Name: name}
		target[name] = entry
	}
	entry.Episodes++
	if success {
		entry.Successful++
	} else {
		entry.Failed++
	}
	if semanticCorrect {
		entry.SemanticAccuracy += 1
	}
}

func sortedEpisodeBreakdowns(source map[string]*EpisodeBreakdown) []EpisodeBreakdown {
	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]EpisodeBreakdown, 0, len(keys))
	for _, key := range keys {
		entry := *source[key]
		if entry.Episodes > 0 {
			entry.SemanticAccuracy /= float64(entry.Episodes)
		}
		out = append(out, entry)
	}
	return out
}

func sortedCapabilityBreakdowns(source map[string]*CapabilityBreakdown) []CapabilityBreakdown {
	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]CapabilityBreakdown, 0, len(keys))
	for _, key := range keys {
		out = append(out, *source[key])
	}
	return out
}

func sortedCounts(source map[string]int) []Count {
	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]Count, 0, len(keys))
	for _, key := range keys {
		out = append(out, Count{Name: key, Count: source[key]})
	}
	return out
}

func stepFailed(step episode.Step) bool {
	if step.TransportStatus == "FAILED" || step.SchemaStatus == "FAILED" {
		return true
	}
	if step.ContractStatus != "" && step.ContractStatus != "OK" {
		return true
	}
	if step.Status != "" && step.Status != "OK" {
		return true
	}
	return false
}

func summarizeLatency(values []int64) LatencySummary {
	if len(values) == 0 {
		return LatencySummary{}
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return LatencySummary{
		StepSamples: len(sorted),
		P50MS:       nearestRank(sorted, 0.50),
		P95MS:       nearestRank(sorted, 0.95),
		MaxMS:       sorted[len(sorted)-1],
	}
}

func nearestRank(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(math.Ceil(p*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}
