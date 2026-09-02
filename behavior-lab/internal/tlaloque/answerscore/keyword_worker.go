package answerscore

import (
	"context"
	"encoding/json"
	"strings"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// KeywordOverlapWorker is the deterministic fallback for SCORE_ANSWER_RELEVANCE:
// it scores an answer by keyword overlap with the source page content. It
// never fails and never needs a network round trip, so it is always
// available as a floor under SemanticModelWorker.
type KeywordOverlapWorker struct{}

func (KeywordOverlapWorker) Descriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{
		ID:             KeywordOverlapWorkerID,
		Capability:     Capability,
		Scope:          tlaloque.ScopeGeneral,
		Engine:         tlaloque.EngineDeterministic,
		InputSchema:    inputSchema,
		OutputSchema:   outputSchema,
		Deterministic:  true,
		MaxConcurrency: 0, // normalized to 1
		Tags:           []string{"keyword-overlap", "fallback"},
	}
}

func (w KeywordOverlapWorker) Execute(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in ScoreInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, err
	}

	flexibility := in.FlexibilityScore
	if flexibility == 0.0 {
		flexibility = 0.8
	}

	out := scoreByKeywordOverlap(in.ModelAnswer, in.PageContent, flexibility)
	out.Confidence = estimateConfidence(in.ModelAnswer)

	raw, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: KeywordOverlapWorkerID, Output: raw, Confidence: out.Confidence}, nil
}

// scoreByKeywordOverlap is the keyword-overlap heuristic moved verbatim from
// the former internal/foldtest.ValidateAnswer.
func scoreByKeywordOverlap(modelAnswer, pageContent string, flexibilityScore float64) ScoreOutput {
	out := ScoreOutput{}

	pageKeywords := extractKeywords(pageContent)
	out.KeywordsTotal = len(pageKeywords)

	matchedCount := 0
	for keyword := range pageKeywords {
		if strings.Contains(strings.ToLower(modelAnswer), strings.ToLower(keyword)) {
			matchedCount++
		}
	}
	out.KeywordsMatched = matchedCount

	if out.KeywordsTotal == 0 {
		out.Score = 0.5
		out.Notes = "No keywords found in page content"
		return out
	}

	keywordScore := float64(matchedCount) / float64(out.KeywordsTotal)

	answerLen := len(strings.Fields(modelAnswer))
	contentLen := len(strings.Fields(pageContent))
	var lengthScore float64
	if answerLen == 0 {
		lengthScore = 0.0
	} else if answerLen < contentLen/10 {
		lengthScore = 0.6 // Too short
	} else if answerLen > contentLen*2 {
		lengthScore = 0.7 // Too long/hallucinating
	} else {
		lengthScore = 1.0 // Good length
	}

	semanticScore := 0.8
	if matchedCount == 0 {
		semanticScore = 0.3
	}

	out.Score = keywordScore*0.4 + lengthScore*0.3 + semanticScore*0.3
	out.Score = out.Score*flexibilityScore + (1-flexibilityScore)*0.5

	if out.Score > 1.0 {
		out.Score = 1.0
	}
	if out.Score < 0.0 {
		out.Score = 0.0
	}

	if out.Score >= 0.7 {
		out.Notes = "Answer matches content well"
	} else if out.Score >= 0.5 {
		out.Notes = "Answer partially matches content"
	} else {
		out.Notes = "Answer does not match content"
	}

	return out
}

// estimateConfidence tries to extract confidence from model response,
// moved verbatim from the former internal/foldtest.estimateConfidence.
func estimateConfidence(response string) float64 {
	lower := strings.ToLower(response)

	confidence := 0.7 // Default

	if strings.Contains(lower, "no estoy seguro") || strings.Contains(lower, "no sé") ||
		strings.Contains(lower, "no estoy") && strings.Contains(lower, "seguro") {
		confidence = 0.4
	} else if strings.Contains(lower, "seguro") || strings.Contains(lower, "definitivamente") {
		confidence = 0.9
	} else if strings.Contains(lower, "probablemente") || strings.Contains(lower, "parece") {
		confidence = 0.6
	}

	return confidence
}

// extractKeywords finds meaningful keywords from text, moved verbatim from
// the former internal/foldtest.extractKeywords.
func extractKeywords(text string) map[string]bool {
	words := strings.Fields(strings.ToLower(text))
	stopwords := map[string]bool{
		"el": true, "la": true, "los": true, "las": true, "un": true, "una": true,
		"y": true, "o": true, "que": true, "de": true, "en": true, "es": true,
		"a": true, "al": true, "del": true, "con": true, "por": true, "para": true,
		"este": true, "ese": true, "eso": true, "como": true, "pero": true,
	}

	keywords := make(map[string]bool)
	for _, word := range words {
		word = strings.Trim(word, ".,;:!?()[]{}\"'")
		if len(word) > 4 && !stopwords[word] {
			keywords[word] = true
		}
	}
	return keywords
}
