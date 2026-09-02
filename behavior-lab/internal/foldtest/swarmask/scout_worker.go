package swarmask

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// PageScoutWorker is a deterministic "tiny model": it reads the whole-book
// cover (one "page N: term1, term2, ..." line per page, see
// foldtest.BuildCoverText) and the question, and suggests the single page
// whose terms overlap the question the most. It never answers the
// question itself and never invents a suggestion when there is no overlap
// at all — silence, not a guess.
type PageScoutWorker struct{}

var coverLinePattern = regexp.MustCompile(`(?m)^\s*page (\d+):\s*(.*)$`)

func (PageScoutWorker) Descriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{
		ID:             ScoutWorkerID,
		Capability:     ScoutCapability,
		Scope:          tlaloque.ScopeGeneral,
		Engine:         tlaloque.EngineDeterministic,
		InputSchema:    inputSchema,
		OutputSchema:   scoutOutputSchema,
		Deterministic:  true,
		MaxConcurrency: 0, // normalized to 1
		Tags:           []string{"page-scout", "cover-term-overlap"},
	}
}

func (PageScoutWorker) Execute(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in AskInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, err
	}

	questionWords := contentWords(in.Question)

	bestPage := 0
	bestScore := 0
	for _, match := range coverLinePattern.FindAllStringSubmatch(in.Cover, -1) {
		page, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		pageWords := contentWords(match[2])
		overlap := 0
		for word := range questionWords {
			if pageWords[word] {
				overlap++
			}
		}
		// Strictly greater: on a tie, the first (lowest-numbered) page found
		// wins, keeping the suggestion deterministic and reproducible.
		if overlap > bestScore {
			bestScore = overlap
			bestPage = page
		}
	}

	out := ScoutOutput{}
	var observations []blackboard.Observation
	if bestScore > 0 {
		score := float64(bestScore) / float64(len(questionWords))
		out.SuggestedPage = bestPage
		out.Score = score

		value, err := json.Marshal(suggestedPageObservation{Page: bestPage, Score: score})
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		observations = append(observations, blackboard.Observation{
			Key:        suggestedPageKey,
			Value:      value,
			Confidence: score,
			Provenance: map[string]string{"source": ScoutWorkerID, "method": "cover-term-overlap"},
		})
	}

	raw, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: ScoutWorkerID, Output: raw, Observations: observations}, nil
}

// contentWords lowercases and filters text into a set of words worth
// matching on (length > 3, a small Spanish/English stopword list removed).
func contentWords(text string) map[string]bool {
	stopwords := map[string]bool{
		"el": true, "la": true, "los": true, "las": true, "un": true, "una": true,
		"y": true, "o": true, "que": true, "de": true, "en": true, "es": true,
		"a": true, "al": true, "del": true, "con": true, "por": true, "para": true,
		"como": true, "pero": true, "cual": true, "cuál": true, "the": true,
		"and": true, "for": true, "are": true, "with": true, "this": true,
	}

	words := map[string]bool{}
	for _, field := range strings.Fields(strings.ToLower(text)) {
		word := strings.Trim(field, ".,;:!?¿()[]{}\"'")
		if len(word) > 3 && !stopwords[word] {
			words[word] = true
		}
	}
	return words
}
