package learningmemory

import (
	"sort"
	"strings"
)

func BuildSummary(root string, events []Event) Summary {
	summary := Summary{
		Schema:      "tlaloc.learning-memory.r0.summary",
		StoreRoot:   root,
		TotalEvents: len(events),
	}

	type failureAccumulator struct {
		pattern           FailurePattern
		models, specimens map[string]bool
		questions         map[string]bool
	}
	patterns := map[string]*failureAccumulator{}

	type outcomeAccumulator struct {
		count            int
		sum, best, worst float64
		initialized      bool
	}
	outcomes := map[string]*outcomeAccumulator{}

	for _, event := range events {
		switch event.EventType {
		case EventObservation:
			summary.ObservationEvents++
			if event.EvidenceClass == EvidenceRealModel {
				summary.RealModelObservations++
			}
			if event.EvidenceClass == EvidenceSynthetic {
				summary.SyntheticObservations++
			}
			if event.Pass != nil && *event.Pass {
				summary.PassedObservations++
			} else {
				summary.FailedObservations++
			}
			if event.EvidenceClass != EvidenceRealModel || event.Pass == nil || *event.Pass {
				continue
			}

			stage := strings.TrimSpace(event.LastCompletedStage)
			if stage == "" {
				stage = "UNKNOWN_STAGE"
			}
			failure := strings.TrimSpace(event.FailureCode)
			if failure == "" {
				failure = "BENCHMARK_ASSERTION_FAILED"
			}
			layer := strings.TrimSpace(event.ScoreLayer)
			if layer == "" {
				layer = "UNKNOWN_LAYER"
			}
			key := stage + "|" + failure + "|" + layer
			accumulator := patterns[key]
			if accumulator == nil {
				accumulator = &failureAccumulator{
					pattern: FailurePattern{
						Key:             key,
						Stage:           stage,
						FailureCode:     failure,
						ScoreLayer:      layer,
						SuggestedTarget: suggestedTarget(failure, stage),
					},
					models:    map[string]bool{},
					specimens: map[string]bool{},
					questions: map[string]bool{},
				}
				patterns[key] = accumulator
			}
			accumulator.pattern.Count++
			if event.ModelID != "" {
				accumulator.models[event.ModelID] = true
			}
			if event.SpecimenID != "" {
				accumulator.specimens[event.SpecimenID] = true
			}
			if event.QuestionID != "" {
				accumulator.questions[event.QuestionID] = true
			}

		case EventChange:
			summary.ChangeAttempts++

		case EventOutcome:
			summary.OutcomeLinks++
			if event.Delta == nil && event.BeforeScore != nil && event.AfterScore != nil {
				delta := *event.AfterScore - *event.BeforeScore
				event.Delta = &delta
			}
			if event.Delta == nil || event.CandidateID == "" {
				continue
			}
			accumulator := outcomes[event.CandidateID]
			if accumulator == nil {
				accumulator = &outcomeAccumulator{}
				outcomes[event.CandidateID] = accumulator
			}
			value := *event.Delta
			accumulator.count++
			accumulator.sum += value
			if !accumulator.initialized || value > accumulator.best {
				accumulator.best = value
			}
			if !accumulator.initialized || value < accumulator.worst {
				accumulator.worst = value
			}
			accumulator.initialized = true
		}
	}

	for _, accumulator := range patterns {
		accumulator.pattern.Models = sortedSet(accumulator.models)
		accumulator.pattern.Specimens = sortedSet(accumulator.specimens)
		accumulator.pattern.Questions = sortedSet(accumulator.questions)
		summary.TopRealFailurePatterns = append(summary.TopRealFailurePatterns, accumulator.pattern)
	}
	sort.Slice(summary.TopRealFailurePatterns, func(i, j int) bool {
		if summary.TopRealFailurePatterns[i].Count != summary.TopRealFailurePatterns[j].Count {
			return summary.TopRealFailurePatterns[i].Count > summary.TopRealFailurePatterns[j].Count
		}
		return summary.TopRealFailurePatterns[i].Key < summary.TopRealFailurePatterns[j].Key
	})
	if len(summary.TopRealFailurePatterns) > 0 {
		summary.NextDebugTarget = summary.TopRealFailurePatterns[0].SuggestedTarget
	}

	for candidateID, accumulator := range outcomes {
		summary.CandidateOutcomes = append(summary.CandidateOutcomes, CandidateOutcome{
			CandidateID: candidateID,
			Outcomes:    accumulator.count,
			MeanDelta:   accumulator.sum / float64(accumulator.count),
			BestDelta:   accumulator.best,
			WorstDelta:  accumulator.worst,
		})
	}
	sort.Slice(summary.CandidateOutcomes, func(i, j int) bool {
		return summary.CandidateOutcomes[i].CandidateID < summary.CandidateOutcomes[j].CandidateID
	})

	summary.ModelProfiles, summary.PairwiseProfiles = BuildEmpiricalProfiles(events)
	return summary
}

func suggestedTarget(failure, stage string) string {
	switch strings.ToUpper(failure) {
	case "NO_VISUAL_SIGNAL":
		return "VISUAL_SIGNAL"
	case "BOOT_NOT_FOUND":
		return "BOOT"
	case "ROSETTA_NOT_FOUND":
		return "ROSETTA"
	case "CODEC_NOT_FOUND":
		return "CODEC_REGISTRY"
	case "CAPABILITY_MISMATCH":
		return "CAPABILITY_FALLBACK"
	case "T2_NOT_FOUND":
		return "T2_NAVIGATION"
	case "SEMANTIC_EVIDENCE_INSUFFICIENT":
		return "SEMANTIC_LAYOUT"
	case "TEMPORAL_RULE_AMBIGUOUS":
		return "TEMPORAL_GRAMMAR"
	case "TEMPORAL_EXECUTION_INCOMPLETE":
		return "EXECUTION_POLICY_COMPLIANCE"
	case "RULE_FIRING_PRECONDITION_VIOLATION", "EXECUTION_SEMANTICS_CONTRADICTION", "CROSS_MODEL_EXECUTION_FIDELITY_FAILED":
		return "SYNCHRONOUS_EXECUTION_FIDELITY"
	case "CELL_IDENTITY_CONFUSION":
		return "CELL_IDENTITY_ENCODING"
	case "FROM_STATE_PRECONDITION_CONFUSION":
		return "FROM_STATE_PRECONDITION_VISIBILITY"
	case "CONDITION_TARGET_BINDING_CONFUSION":
		return "RULE_ROLE_BINDING"
	case "CHECKPOINT_NOT_FOUND":
		return "TEMPORAL_ROUTING"
	case "ARTIFACT_GENERATION_REGRESSION", "UNAUTHORIZED_SEMANTIC_DRIFT":
		return "SEMANTIC_PARITY_GATE"
	case "UNSUPPORTED_OPERATION":
		return "CAPABILITY_PROFILE"
	}
	if strings.EqualFold(stage, "ROSETTA") {
		return "ROSETTA_TO_T2_ROUTE"
	}
	if strings.EqualFold(stage, "TEMPORAL_ROUTE") || strings.EqualFold(stage, "TEMPORAL_STEP") {
		return "TEMPORAL_PROGRAM"
	}
	return "BENCHMARK_FAILURE_ANALYSIS"
}

func sortedSet(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
