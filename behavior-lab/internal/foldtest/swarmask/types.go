// Package swarmask wires a deterministic "page scout" Tlaloque and the parrot
// (lfm2-vl-1.6b, via foldtest.RunSession) into a real tlaloque.SwarmRunner
// DAG, so they share state through the blackboard instead of the ad-hoc
// in-process fallback chains used by answerscore/questiongen. The scout
// runs first and, if it finds a candidate page, posts a cheap "suggested
// page" observation; the parrot node reads it from the blackboard before
// answering.
package swarmask

import "tlaloc.local/behaviorlab/internal/pdfmemory"

const (
	ScoutCapability        = "SUGGEST_RELEVANT_PAGE"
	EntityCapability       = "EXTRACT_QUESTION_ENTITIES"
	ClassifierCapability   = "CLASSIFY_QUESTION_TYPE"
	AnswerCapability       = "ANSWER_QUESTION"
	ConsolidatorCapability = "CONSOLIDATE_BLACKBOARD"

	ScoutWorkerID        = "swarmask-page-scout"
	EntityWorkerID       = "swarmask-entity-scout"
	ClassifierWorkerID   = "swarmask-question-classifier"
	AnswerWorkerID       = "swarmask-parrot-answer"
	ConsolidatorWorkerID = "swarmask-consolidator"

	inputSchema               = "tlaloc.foldtest.r0.swarmask-input"
	scoutOutputSchema         = "tlaloc.foldtest.r0.swarmask-scout-output"
	entityOutputSchema        = "tlaloc.foldtest.r0.swarmask-entity-output"
	classifierOutputSchema    = "tlaloc.foldtest.r0.swarmask-classifier-output"
	answerOutputSchema        = "tlaloc.foldtest.r0.swarmask-answer-output"
	consolidationOutputSchema = "tlaloc.foldtest.r0.swarmask-consolidation-output"

	suggestedPageKey    = "suggested_page"
	questionEntitiesKey = "question_entities"
	questionTypeKey     = "question_type"
	answerGroundedKey   = "decision.answer_grounded"

	// Question type classifications.
	QuestionTypeDefinition    = "DEFINITION"
	QuestionTypeComparison    = "COMPARISON"
	QuestionTypeProcess       = "PROCESS"
	QuestionTypeFactualDetail = "FACTUAL_DETAIL"
	QuestionTypeGeneral       = "GENERAL"
)

// AskInput is the CapabilityRequest.Input payload shared by every node in
// the plan (per tlaloque.SwarmRunner.Run: the same task input is passed to
// every node — each worker only reads the fields it needs).
type AskInput struct {
	Question string             `json:"question"`
	Cover    string             `json:"cover"`
	WorkDir  string             `json:"work_dir"`
	StoreDir string             `json:"store_dir"`
	Manifest pdfmemory.Manifest `json:"manifest"`
	Model    string             `json:"model"`
	BaseURL  string             `json:"base_url"`
	MaxTurns int                `json:"max_turns"`
	Budget   int                `json:"budget"`

	// ClassifierEndpoint, when set, points the question-type classifier
	// node at a running questionclass-charcnn-r0 HTTP service; empty keeps
	// that node on its rule-based path.
	ClassifierEndpoint string `json:"classifier_endpoint,omitempty"`

	// ClassifierCalibrationPath, when set, is the CalibrationProfile JSON
	// the classifier node consults before trusting the model's prediction
	// (tools/questionclass_calibrate.py output). Requires ClassifierEndpoint.
	ClassifierCalibrationPath string `json:"classifier_calibration_path,omitempty"`

	// GroundingEndpoint, when set, points the consolidator node at a running
	// groundingscore-distilled-r0 HTTP service (tools/grounding_serve.py) —
	// an independent grounding judge distinct from the parrot that produced
	// the answer. Empty keeps the consolidator on the answerscore judge.
	GroundingEndpoint string `json:"grounding_endpoint,omitempty"`

	// GroundingCalibrationPath, when set, is the CalibrationProfile JSON the
	// consolidator consults before trusting the distilled score instead of
	// falling back to answerscore (tools/grounding_calibrate.py output).
	// Requires GroundingEndpoint.
	GroundingCalibrationPath string `json:"grounding_calibration_path,omitempty"`
}

// ScoutOutput is the scout worker's CapabilityResponse.Output payload.
type ScoutOutput struct {
	SuggestedPage int     `json:"suggested_page,omitempty"`
	Score         float64 `json:"score,omitempty"`
}

// AnswerOutput is the parrot worker's CapabilityResponse.Output payload.
type AnswerOutput struct {
	Answer                string `json:"answer"`
	Turns                 int    `json:"turns"`
	TotalTokensPrompt     int    `json:"total_tokens_prompt"`
	TotalTokensCompletion int    `json:"total_tokens_completion"`
}

// suggestedPageObservation is the JSON shape stored in a "suggested_page"
// blackboard Observation's Value.
type suggestedPageObservation struct {
	Page  int     `json:"page"`
	Score float64 `json:"score"`
}

// EntityOutput is the entity worker's CapabilityResponse.Output payload —
// also reused as-is for the "question_entities" blackboard Observation's
// Value, since both shapes are identical.
type EntityOutput struct {
	Years    []int     `json:"years,omitempty"`
	Numbers  []float64 `json:"numbers,omitempty"`
	Acronyms []string  `json:"acronyms,omitempty"`
}

// QuestionTypeOutput is the classifier worker's CapabilityResponse.Output
// payload — also reused as-is for the "question_type" blackboard
// Observation's Value.
type QuestionTypeOutput struct {
	Type string `json:"type"`
}

// ConsolidationOutput is the consolidator worker's CapabilityResponse.Output
// payload, and the final terminal output of the plan: it carries the parrot's
// answer text through plus a grounding verdict checked against the real
// page content.
type ConsolidationOutput struct {
	Answer              string  `json:"answer"`
	Grounded            bool    `json:"grounded"`
	Score               float64 `json:"score"`
	ScoredBy            string  `json:"scored_by"`
	VerifiedAgainstPage int     `json:"verified_against_page,omitempty"`
	// JudgeIndependent is true when the grounding score came from a judge
	// other than the parrot model that produced the answer (the distilled
	// groundingscore worker, or answerscore's embedding/keyword workers) —
	// false when it fell back to the chat judge running the same model.
	JudgeIndependent bool `json:"judge_independent"`
}

// answerGroundedObservation is the JSON shape stored in a
// "decision.answer_grounded" blackboard Observation's Value.
type answerGroundedObservation struct {
	Page     int     `json:"page"`
	Score    float64 `json:"score"`
	ScoredBy string  `json:"scored_by"`
}
