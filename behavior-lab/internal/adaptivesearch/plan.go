package adaptivesearch

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/visualsearch"
)

const explorationMass = 0.15

var supportedKinds = []visualsearch.MutationKind{
	visualsearch.MutationPrompt,
	visualsearch.MutationChannelRole,
	visualsearch.MutationPrimitive,
	visualsearch.MutationLayout,
	visualsearch.MutationRedundancy,
	visualsearch.MutationColorUsage,
	visualsearch.MutationNumericStructure,
	visualsearch.MutationInterferenceStructure,
	visualsearch.MutationDepthStructure,
	visualsearch.MutationTemporalStructure,
	visualsearch.MutationEmergentStructure,
}

func BuildPlan(root string, events []learningmemory.Event) Plan {
	summary := learningmemory.BuildSummary(root, events)
	plan := Plan{
		Schema:          SchemaR0 + ".plan",
		MemoryRoot:      root,
		NextDebugTarget: summary.NextDebugTarget,
		Guardrails: []string{
			"MEMORY_GUIDES_EXPERIMENT_BUDGET_NOT_PROMOTION_SCORE",
			"FINAL_TOURNAMENT_REMAINS_EVIDENCE_GATED",
			"SYNTHETIC_EVIDENCE_NE_EMPIRICAL_SEARCH_TARGET",
			"EXPLORATION_FLOOR_GT_0",
			"TLALOC_RECOMMENDS_ORIGAMI_DECIDES",
		},
	}
	for _, e := range events {
		if isRealFailure(e) {
			plan.RealFailureEvents++
		}
	}
	for i, p := range summary.TopRealFailurePatterns {
		if i >= 5 {
			break
		}
		focus := PatternFocus{Key: p.Key, Stage: p.Stage, FailureCode: p.FailureCode, ScoreLayer: p.ScoreLayer, Count: p.Count, Models: append([]string(nil), p.Models...), Specimens: append([]string(nil), p.Specimens...), Questions: append([]string(nil), p.Questions...), SuggestedTarget: p.SuggestedTarget}
		plan.FailurePatterns = append(plan.FailurePatterns, focus)
		if i == 0 {
			cp := focus
			plan.PrimaryPattern = &cp
		}
	}
	plan.Adaptive = plan.PrimaryPattern != nil && plan.RealFailureEvents > 0
	if plan.Adaptive {
		plan.ParentEvidenceIDs = parentEvidenceIDs(events, *plan.PrimaryPattern)
	}

	raw := map[visualsearch.MutationKind]float64{}
	for _, k := range supportedKinds {
		raw[k] = 0
	}
	if plan.Adaptive {
		for _, p := range plan.FailurePatterns {
			base := float64(p.Count) * diversityFactor(len(p.Models), len(p.Specimens))
			for rank, kind := range targetMutationKinds(p.SuggestedTarget) {
				factor := []float64{1.0, 0.8, 0.6, 0.4}
				f := 0.25
				if rank < len(factor) {
					f = factor[rank]
				}
				raw[kind] += base * f
			}
		}
	} else {
		for _, k := range supportedKinds {
			raw[k] = 1
		}
	}

	history := historicalSignals(events)
	plan.HistoricalSignals = history
	for _, h := range history {
		if raw[h.MutationKind] <= 0 {
			continue
		}
		raw[h.MutationKind] *= 1 + h.Adjustment
	}

	weights := normalizedWithExploration(raw)
	priorities := make([]MutationPriority, 0, len(supportedKinds))
	for _, kind := range supportedKinds {
		reason := "exploration floor keeps this mutation family testable"
		target := plan.NextDebugTarget
		if plan.Adaptive && containsKind(targetMutationKinds(target), kind) {
			reason = fmt.Sprintf("targets observed real-model failure frontier %s", target)
		}
		for _, h := range history {
			if h.MutationKind == kind && h.Outcomes > 0 {
				reason += fmt.Sprintf("; historical mean outcome delta %.4f across %d linked outcomes", h.MeanDelta, h.Outcomes)
			}
		}
		priorities = append(priorities, MutationPriority{Kind: kind, Weight: weights[kind], Target: target, Reason: reason, ExplorationFloor: true})
	}
	sort.Slice(priorities, func(i, j int) bool {
		if priorities[i].Weight == priorities[j].Weight {
			return priorities[i].Kind < priorities[j].Kind
		}
		return priorities[i].Weight > priorities[j].Weight
	})
	for i := range priorities {
		priorities[i].Rank = i + 1
	}
	plan.MutationPriorities = priorities
	for i, p := range priorities {
		if i >= 4 {
			break
		}
		plan.SuggestedMutations = append(plan.SuggestedMutations, suggestion(plan.NextDebugTarget, p.Kind))
	}
	return plan
}

func Prioritize(plan Plan, candidates []visualsearch.Candidate, limit int) Queue {
	weights := map[visualsearch.MutationKind]float64{}
	for _, p := range plan.MutationPriorities {
		weights[p.Kind] = p.Weight
	}
	items := make([]CandidatePriority, 0, len(candidates))
	for _, c := range candidates {
		seen := map[visualsearch.MutationKind]bool{}
		kinds := []visualsearch.MutationKind{}
		sum := 0.0
		for _, m := range c.Mutations {
			if seen[m.Kind] {
				continue
			}
			seen[m.Kind] = true
			kinds = append(kinds, m.Kind)
			sum += weights[m.Kind]
		}
		sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
		score := 0.0
		if len(kinds) > 0 {
			score = sum / float64(len(kinds))
		}
		if len(c.Mutations) > 1 {
			score -= 0.02 * float64(len(c.Mutations)-1)
		}
		if score < 0 {
			score = 0
		}
		reason := "candidate retained by exploration policy"
		if plan.Adaptive {
			reason = fmt.Sprintf("pre-evidence priority for failure target %s; final tournament score remains evidence-only", plan.NextDebugTarget)
		}
		items = append(items, CandidatePriority{CandidateID: c.ID, PriorityScore: score, MutationKinds: kinds, MatchedTarget: plan.NextDebugTarget, Reason: reason})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].PriorityScore == items[j].PriorityScore {
			return items[i].CandidateID < items[j].CandidateID
		}
		return items[i].PriorityScore > items[j].PriorityScore
	})
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	for i := range items {
		items[i].Rank = i + 1
	}
	return Queue{Schema: SchemaR0 + ".queue", Plan: plan, CandidateOrder: items, Authority: "SEARCH_PRIORITY_ONLY_FINAL_VISUAL_TOURNAMENT_REMAINS_EVIDENCE_GATED"}
}

func ChangeAttemptEvents(queue Queue, candidates []visualsearch.Candidate) []learningmemory.Event {
	if !queue.Plan.Adaptive || len(queue.Plan.ParentEvidenceIDs) == 0 {
		return nil
	}
	byID := map[string]visualsearch.Candidate{}
	for _, c := range candidates {
		byID[c.ID] = c
	}
	out := []learningmemory.Event{}
	for _, item := range queue.CandidateOrder {
		c, ok := byID[item.CandidateID]
		if !ok {
			continue
		}
		tags := []string{"adaptive-search", "target:" + queue.Plan.NextDebugTarget}
		seen := map[string]bool{}
		for _, m := range c.Mutations {
			tag := "mutation:" + string(m.Kind)
			if !seen[tag] {
				tags = append(tags, tag)
				seen[tag] = true
			}
		}
		sort.Strings(tags)
		summary := fmt.Sprintf("Adaptive-search candidate %s selected for target %s before evidence scoring", c.ID, queue.Plan.NextDebugTarget)
		out = append(out, learningmemory.Event{Schema: learningmemory.EventSchema, EventType: learningmemory.EventChange, EvidenceClass: learningmemory.EvidenceManual, CandidateID: c.ID, ParentEventIDs: append([]string(nil), queue.Plan.ParentEvidenceIDs...), ChangeSummary: summary, Tags: tags})
	}
	return out
}

func targetMutationKinds(target string) []visualsearch.MutationKind {
	switch strings.ToUpper(strings.TrimSpace(target)) {
	case "VISUAL_SIGNAL":
		return []visualsearch.MutationKind{visualsearch.MutationPrimitive, visualsearch.MutationRedundancy, visualsearch.MutationLayout, visualsearch.MutationColorUsage}
	case "BOOT":
		return []visualsearch.MutationKind{visualsearch.MutationPrompt, visualsearch.MutationLayout, visualsearch.MutationRedundancy, visualsearch.MutationPrimitive}
	case "ROSETTA":
		return []visualsearch.MutationKind{visualsearch.MutationPrompt, visualsearch.MutationLayout, visualsearch.MutationRedundancy, visualsearch.MutationPrimitive}
	case "CODEC_REGISTRY":
		return []visualsearch.MutationKind{visualsearch.MutationPrompt, visualsearch.MutationLayout, visualsearch.MutationChannelRole, visualsearch.MutationRedundancy}
	case "CAPABILITY_FALLBACK":
		return []visualsearch.MutationKind{visualsearch.MutationPrompt, visualsearch.MutationChannelRole, visualsearch.MutationRedundancy}
	case "T2_NAVIGATION":
		return []visualsearch.MutationKind{visualsearch.MutationLayout, visualsearch.MutationPrompt, visualsearch.MutationRedundancy, visualsearch.MutationChannelRole}
	case "SEMANTIC_LAYOUT":
		return []visualsearch.MutationKind{visualsearch.MutationLayout, visualsearch.MutationPrimitive, visualsearch.MutationRedundancy, visualsearch.MutationChannelRole}
	case "TEMPORAL_GRAMMAR":
		return []visualsearch.MutationKind{visualsearch.MutationTemporalStructure, visualsearch.MutationPrompt, visualsearch.MutationPrimitive, visualsearch.MutationChannelRole}
	case "TEMPORAL_ROUTING":
		return []visualsearch.MutationKind{visualsearch.MutationTemporalStructure, visualsearch.MutationLayout, visualsearch.MutationRedundancy, visualsearch.MutationPrompt}
	case "CAPABILITY_PROFILE":
		return []visualsearch.MutationKind{visualsearch.MutationPrompt, visualsearch.MutationChannelRole, visualsearch.MutationRedundancy}
	case "ROSETTA_TO_T2_ROUTE":
		return []visualsearch.MutationKind{visualsearch.MutationLayout, visualsearch.MutationPrompt, visualsearch.MutationRedundancy, visualsearch.MutationChannelRole}
	case "TEMPORAL_PROGRAM":
		return []visualsearch.MutationKind{visualsearch.MutationTemporalStructure, visualsearch.MutationLayout, visualsearch.MutationPrompt, visualsearch.MutationRedundancy}
	default:
		return []visualsearch.MutationKind{visualsearch.MutationPrompt, visualsearch.MutationLayout, visualsearch.MutationRedundancy, visualsearch.MutationPrimitive}
	}
}

func suggestion(target string, kind visualsearch.MutationKind) SuggestedMutation {
	t := strings.ToUpper(strings.TrimSpace(target))
	if t == "" {
		t = "GENERAL_EXPLORATION"
	}
	targetField := t
	value := "ISOLATED_EXPERIMENTAL_VARIANT"
	switch kind {
	case visualsearch.MutationLayout:
		if t == "T2_NAVIGATION" || t == "ROSETTA_TO_T2_ROUTE" {
			targetField = "T1_TO_T2_ENTRY_ROUTE"
			value = "EXPLICIT_DIRECTIONAL_ANCHOR"
		} else {
			value = "LOCALIZE_FAILURE_TARGET_REGION"
		}
	case visualsearch.MutationPrompt:
		if t == "T2_NAVIGATION" || t == "ROSETTA_TO_T2_ROUTE" {
			targetField = "ROSETTA.S2.READ_SUPERINDEX"
			value = "DECLARE_T2_LOCATION_BEFORE_DECODE"
		} else {
			value = "SHORT_EXPLICIT_PROTOCOL_INSTRUCTION"
		}
	case visualsearch.MutationRedundancy:
		if t == "T2_NAVIGATION" {
			targetField = "T2_ENTRY_MARKER"
			value = "REPEAT_AT_BOOT_AND_ROSETTA"
		} else {
			value = "BOUNDED_REDUNDANT_ANCHOR"
		}
	case visualsearch.MutationChannelRole:
		value = "DEDICATED_SEMANTIC_ROUTING_ROLE"
	case visualsearch.MutationPrimitive:
		value = "DISTINCT_FAILURE_TARGET_PRIMITIVE"
	case visualsearch.MutationTemporalStructure:
		value = "EXPLICIT_PHASE_EVENT_CHECKPOINT_STRUCTURE"
	case visualsearch.MutationColorUsage:
		value = "HIGH_CONTRAST_ROLE_SEPARATION"
	case visualsearch.MutationNumericStructure:
		value = "EXPERIMENTAL_ADDRESS_ORDER_SIGNAL"
	case visualsearch.MutationInterferenceStructure:
		value = "EXPERIMENTAL_INTERFERENCE_SIGNAL"
	case visualsearch.MutationDepthStructure:
		value = "EXPERIMENTAL_DEPTH_SIGNAL"
	case visualsearch.MutationEmergentStructure:
		value = "EXPERIMENTAL_EMERGENT_SIGNAL"
	}
	return SuggestedMutation{Kind: kind, Target: targetField, Value: value, Rationale: "generated from persistent real-model failure memory; must be tested as an isolated candidate before any recommendation", Experimental: true}
}

func historicalSignals(events []learningmemory.Event) []HistoricalSignal {
	candidateKinds := map[string]map[visualsearch.MutationKind]bool{}
	for _, e := range events {
		if e.EventType != learningmemory.EventChange || e.CandidateID == "" {
			continue
		}
		for _, tag := range e.Tags {
			if !strings.HasPrefix(strings.ToLower(tag), "mutation:") {
				continue
			}
			kind := visualsearch.MutationKind(strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(strings.ToLower(tag), "mutation:"))))
			if !isSupportedKind(kind) {
				continue
			}
			if candidateKinds[e.CandidateID] == nil {
				candidateKinds[e.CandidateID] = map[visualsearch.MutationKind]bool{}
			}
			candidateKinds[e.CandidateID][kind] = true
		}
	}
	type acc struct {
		n   int
		sum float64
	}
	byKind := map[visualsearch.MutationKind]*acc{}
	for _, e := range events {
		if e.EventType != learningmemory.EventOutcome || e.CandidateID == "" || e.Delta == nil {
			continue
		}
		for kind := range candidateKinds[e.CandidateID] {
			a := byKind[kind]
			if a == nil {
				a = &acc{}
				byKind[kind] = a
			}
			a.n++
			a.sum += *e.Delta
		}
	}
	out := []HistoricalSignal{}
	for _, kind := range supportedKinds {
		a := byKind[kind]
		if a == nil || a.n == 0 {
			continue
		}
		mean := a.sum / float64(a.n)
		adj := clamp(mean, -0.25, 0.25)
		out = append(out, HistoricalSignal{MutationKind: kind, Outcomes: a.n, MeanDelta: mean, Adjustment: adj})
	}
	return out
}

func parentEvidenceIDs(events []learningmemory.Event, p PatternFocus) []string {
	ids := []string{}
	for _, e := range events {
		if !isRealFailure(e) {
			continue
		}
		stage := strings.TrimSpace(e.LastCompletedStage)
		if stage == "" {
			stage = "UNKNOWN_STAGE"
		}
		failure := strings.TrimSpace(e.FailureCode)
		if failure == "" {
			failure = "BENCHMARK_ASSERTION_FAILED"
		}
		layer := strings.TrimSpace(e.ScoreLayer)
		if layer == "" {
			layer = "UNKNOWN_LAYER"
		}
		if stage == p.Stage && failure == p.FailureCode && layer == p.ScoreLayer && e.EventID != "" {
			ids = append(ids, e.EventID)
		}
	}
	sort.Strings(ids)
	if len(ids) > 20 {
		ids = ids[:20]
	}
	return ids
}

func normalizedWithExploration(raw map[visualsearch.MutationKind]float64) map[visualsearch.MutationKind]float64 {
	total := 0.0
	for _, k := range supportedKinds {
		if raw[k] > 0 {
			total += raw[k]
		}
	}
	base := map[visualsearch.MutationKind]float64{}
	if total <= 0 {
		for _, k := range supportedKinds {
			base[k] = 1 / float64(len(supportedKinds))
		}
	} else {
		for _, k := range supportedKinds {
			base[k] = math.Max(0, raw[k]) / total
		}
	}
	floor := explorationMass / float64(len(supportedKinds))
	for _, k := range supportedKinds {
		base[k] = (1-explorationMass)*base[k] + floor
	}
	return base
}

func isRealFailure(e learningmemory.Event) bool {
	return e.EventType == learningmemory.EventObservation && e.EvidenceClass == learningmemory.EvidenceRealModel && e.Pass != nil && !*e.Pass
}
func diversityFactor(models, specimens int) float64 {
	return 1 + 0.1*float64(maxInt(0, models-1)) + 0.05*float64(maxInt(0, specimens-1))
}
func containsKind(kinds []visualsearch.MutationKind, want visualsearch.MutationKind) bool {
	for _, k := range kinds {
		if k == want {
			return true
		}
	}
	return false
}
func isSupportedKind(kind visualsearch.MutationKind) bool { return containsKind(supportedKinds, kind) }
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
