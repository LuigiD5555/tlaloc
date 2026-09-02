package swarmask

import (
	"context"
	"encoding/json"
	"strings"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// QuestionClassifierWorker is a third deterministic "tiny model": it
// classifies the question's grammatical shape (definition, comparison,
// process, factual-detail, or general), independent of what PageScoutWorker
// (where) and EntityScoutWorker (what concrete facts) report — this helps
// the loro (and future consolidators) calibrate how much verification a
// question needs. Unlike scout/entities, it always emits an observation:
// "this looks like a general question" is itself useful information, not a
// fabrication, since every question has *some* shape.
type QuestionClassifierWorker struct{}

func (QuestionClassifierWorker) Descriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{
		ID:             ClassifierWorkerID,
		Capability:     ClassifierCapability,
		Scope:          tlaloque.ScopeGeneral,
		Engine:         tlaloque.EngineDeterministic,
		InputSchema:    inputSchema,
		OutputSchema:   classifierOutputSchema,
		Deterministic:  true,
		MaxConcurrency: 0, // normalized to 1
		Tags:           []string{"question-classifier", "rule-based"},
	}
}

func (QuestionClassifierWorker) Execute(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in AskInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, err
	}

	questionType, confidence := classifyQuestion(in.Question)
	out := QuestionTypeOutput{Type: questionType}

	value, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	observations := []blackboard.Observation{{
		Key:        questionTypeKey,
		Value:      value,
		Confidence: confidence,
		Provenance: map[string]string{"source": ClassifierWorkerID, "method": "rule-based"},
	}}

	raw, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: ClassifierWorkerID, Output: raw, Observations: observations}, nil
}

var (
	definitionPrefixes = []string{"what is", "what are", "qué es", "que es", "define"}
	comparisonWords    = []string{"compare", "comparison", "relationship", "relación", "relacion", " vs ", " vs.", "difference", "diferencia"}
	processPrefixes    = []string{"how ", "cómo", "como ", "why ", "por qué", "por que"}
)

// classifyQuestion returns a QuestionType constant and a confidence for it.
// FACTUAL_DETAIL reuses the same literal year/number detection as
// EntityScoutWorker: a question anchored to a specific figure needs a
// precise answer, not a general one.
func classifyQuestion(question string) (string, float64) {
	lower := strings.ToLower(question)
	lower = strings.TrimLeft(lower, "¿¡\"' ")

	// Comparison is checked before definition: "what is the relationship
	// between X and Y" contains both an ("what is") and a comparison
	// signal ("relationship") — the more specific comparison verdict wins.
	for _, word := range comparisonWords {
		if strings.Contains(lower, word) {
			return QuestionTypeComparison, 0.8
		}
	}
	for _, prefix := range definitionPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return QuestionTypeDefinition, 0.8
		}
	}
	for _, prefix := range processPrefixes {
		if strings.HasPrefix(lower, prefix) || strings.Contains(lower, " "+strings.TrimSpace(prefix)+" ") {
			return QuestionTypeProcess, 0.8
		}
	}
	entities := extractQuestionEntities(question)
	if len(entities.Years) > 0 || len(entities.Numbers) > 0 {
		return QuestionTypeFactualDetail, 0.8
	}

	return QuestionTypeGeneral, 0.3
}
