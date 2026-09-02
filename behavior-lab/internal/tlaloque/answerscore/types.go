// Package answerscore provides the SCORE_ANSWER_RELEVANCE Tlaloque: a
// bounded capability that judges whether a model answer is supported by the
// source page content it was asked about.
package answerscore

const (
	Capability = "SCORE_ANSWER_RELEVANCE"

	KeywordOverlapWorkerID = "answerscore-keyword-overlap"
	SemanticModelWorkerID  = "answerscore-semantic-model"

	inputSchema  = "tlaloc.foldtest.r0.answer-score-input"
	outputSchema = "tlaloc.foldtest.r0.answer-score-output"
)

// ScoreInput is the CapabilityRequest.Input payload for SCORE_ANSWER_RELEVANCE.
type ScoreInput struct {
	Question    string `json:"question"`
	ModelAnswer string `json:"model_answer"`
	PageContent string `json:"page_content"`
	// FlexibilityScore only affects KeywordOverlapWorker (0.0-1.0, default
	// 0.8 when zero). It has no effect on SemanticModelWorker.
	FlexibilityScore float64 `json:"flexibility_score,omitempty"`
}

// ScoreOutput is the CapabilityResponse.Output payload for SCORE_ANSWER_RELEVANCE.
type ScoreOutput struct {
	Score           float64 `json:"score"`
	Confidence      float64 `json:"confidence"`
	KeywordsMatched int     `json:"keywords_matched"`
	KeywordsTotal   int     `json:"keywords_total"`
	Notes           string  `json:"notes,omitempty"`
}
