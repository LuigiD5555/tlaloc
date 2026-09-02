// Package questionclass is the Go client side of questionclass-charcnn-r0, a
// genuinely trained (not prompted, not rule-based) character-level CNN that
// classifies a question's rhetorical shape: DEFINITION, COMPARISON,
// PROCESS, FACTUAL_DETAIL, or GENERAL. The model (tools/questionclass_*.py,
// ~7.7K parameters, trained from scratch on synthetic bilingual templates)
// is served by tools/questionclass_serve.py over HTTP_JSON; this package
// only registers it as a tlaloque.HTTPWorker and offers a thin client call.
// It exists to replace the rule-based prefix matching in
// internal/foldtest/swarmask.classifyQuestion with a learned classifier,
// while that rule-based path stays as an honest fallback.
package questionclass

import "tlaloc.local/behaviorlab/internal/tlaloque"

const (
	Capability = "CLASSIFY_QUESTION_TYPE"
	WorkerID   = "questionclass-charcnn-r0"

	inputSchema  = "tlaloc.questionclass.r0.question-input"
	outputSchema = "tlaloc.questionclass.r0.question-type-output"
)

// Input is the CapabilityRequest.Input payload: a single question string.
type Input struct {
	Question string `json:"question"`
}

// Output is the CapabilityResponse.Output payload: one of the five
// rhetorical-shape labels the model was trained to predict.
type Output struct {
	Type string `json:"type"`
}

// Descriptor is the single source of truth for this worker's capability
// descriptor.
func Descriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{
		ID:             WorkerID,
		Capability:     Capability,
		Scope:          tlaloque.ScopeGeneral,
		Engine:         tlaloque.EngineModel,
		InputSchema:    inputSchema,
		OutputSchema:   outputSchema,
		Deterministic:  false,
		ParameterCount: 7_721, // questionclass-charcnn-r0, trained from scratch this session
		MaxConcurrency: 1,
		Tags:           []string{"trained-specialist", "question-classifier", "char-cnn", "resident"},
	}
}
