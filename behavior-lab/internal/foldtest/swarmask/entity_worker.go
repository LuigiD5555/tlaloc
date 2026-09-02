package swarmask

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// EntityScoutWorker is a second deterministic "tiny model": it scans the
// question itself (not the cover — page content isn't available to any node
// before the parrot decides to UNFOLD, same constraint the parrot itself has)
// for literal factual anchors — years, other numbers, acronyms — and posts
// them as a distinct observation from PageScoutWorker's page suggestion.
// Silent when the question has none, same as PageScoutWorker.
type EntityScoutWorker struct{}

var (
	yearPattern    = regexp.MustCompile(`\b(1[0-9]{3}|2[0-9]{3})\b`)
	numberPattern  = regexp.MustCompile(`\b\d+(?:\.\d+)?\b`)
	acronymPattern = regexp.MustCompile(`\b[A-Z]{2,}\b`)
)

func (EntityScoutWorker) Descriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{
		ID:             EntityWorkerID,
		Capability:     EntityCapability,
		Scope:          tlaloque.ScopeGeneral,
		Engine:         tlaloque.EngineDeterministic,
		InputSchema:    inputSchema,
		OutputSchema:   entityOutputSchema,
		Deterministic:  true,
		MaxConcurrency: 0, // normalized to 1
		Tags:           []string{"entity-extraction", "question-anchors"},
	}
}

func (EntityScoutWorker) Execute(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in AskInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, err
	}

	out := extractQuestionEntities(in.Question)

	var observations []blackboard.Observation
	if len(out.Years) > 0 || len(out.Numbers) > 0 || len(out.Acronyms) > 0 {
		value, err := json.Marshal(out)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		observations = append(observations, blackboard.Observation{
			Key:        questionEntitiesKey,
			Value:      value,
			Confidence: 1.0, // literal extraction, not a heuristic guess
			Provenance: map[string]string{"source": EntityWorkerID, "method": "regex-extraction"},
		})
	}

	raw, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: EntityWorkerID, Output: raw, Observations: observations}, nil
}

// extractQuestionEntities finds years, other numbers, and acronyms in text.
// A year (1000-2999) is never also reported as a plain number.
func extractQuestionEntities(text string) EntityOutput {
	out := EntityOutput{}

	yearMatches := map[string]bool{}
	for _, m := range yearPattern.FindAllString(text, -1) {
		if year, err := strconv.Atoi(m); err == nil {
			out.Years = append(out.Years, year)
			yearMatches[m] = true
		}
	}

	for _, m := range numberPattern.FindAllString(text, -1) {
		if yearMatches[m] {
			continue
		}
		if n, err := strconv.ParseFloat(m, 64); err == nil {
			out.Numbers = append(out.Numbers, n)
		}
	}

	out.Acronyms = acronymPattern.FindAllString(text, -1)

	return out
}
