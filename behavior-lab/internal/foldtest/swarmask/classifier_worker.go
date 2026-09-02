package swarmask

import (
	"context"
	"encoding/json"
	"strings"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/tlaloque"
	"tlaloc.local/behaviorlab/internal/tlaloque/questionclass"
)

// minModelConfidence is the floor below which the trained classifier's
// verdict is discarded in favor of the rule-based one. Calibrated against
// real questions: questionclass-charcnn-r0 is ~1.0 confident on phrasings
// close to its synthetic training templates and correct there, but drifts
// to the 0.6s on unfamiliar constructions — and is often wrong when it
// does. Below 0.7 the deterministic prefix rules are the safer bet.
const minModelConfidence = 0.7

// QuestionClassifierWorker classifies the question's rhetorical shape
// (definition, comparison, process, factual-detail, or general),
// independent of what PageScoutWorker (where) and EntityScoutWorker (what
// concrete facts) report — this helps the loro (and the consolidator)
// calibrate how much verification a question needs. Unlike scout/entities,
// it always emits an observation: "this looks like a general question" is
// itself useful information, not a fabrication, since every question has
// *some* shape.
//
// If ModelRegistry is set (a questionclass-charcnn-r0 HTTP service), it uses
// that trained char-CNN and falls back to the rule-based classifyQuestion
// only when the model errors or is not confident enough — reporting which
// path produced the verdict in the observation's provenance. With
// ModelRegistry nil it is purely rule-based.
type QuestionClassifierWorker struct {
	ModelRegistry *tlaloque.Registry
}

func (w QuestionClassifierWorker) Descriptor() tlaloque.CapabilityDescriptor {
	engine, deterministic, tags := tlaloque.EngineDeterministic, true, []string{"question-classifier", "rule-based"}
	if w.ModelRegistry != nil {
		engine, deterministic, tags = tlaloque.EngineModel, false, []string{"question-classifier", "char-cnn", "rule-based-fallback"}
	}
	return tlaloque.CapabilityDescriptor{
		ID:             ClassifierWorkerID,
		Capability:     ClassifierCapability,
		Scope:          tlaloque.ScopeGeneral,
		Engine:         engine,
		InputSchema:    inputSchema,
		OutputSchema:   classifierOutputSchema,
		Deterministic:  deterministic,
		MaxConcurrency: 0, // normalized to 1
		Tags:           tags,
	}
}

func (w QuestionClassifierWorker) Execute(ctx context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in AskInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, err
	}

	questionType, confidence, method := w.classify(ctx, in.Question)
	out := QuestionTypeOutput{Type: questionType}

	value, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	observations := []blackboard.Observation{{
		Key:        questionTypeKey,
		Value:      value,
		Confidence: confidence,
		Provenance: map[string]string{"source": ClassifierWorkerID, "method": method},
	}}

	raw, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: ClassifierWorkerID, Output: raw, Observations: observations}, nil
}

// classify picks the trained char-CNN verdict when a model is wired and
// confident, otherwise the rule-based one — always reporting which via the
// returned method string ("charcnn-model" or "rule-based"), so the
// blackboard never claims a model classified when the fallback did.
func (w QuestionClassifierWorker) classify(ctx context.Context, question string) (questionType string, confidence float64, method string) {
	if w.ModelRegistry != nil {
		if out, modelConfidence, err := questionclass.Classify(ctx, w.ModelRegistry, question); err == nil && modelConfidence >= minModelConfidence {
			return out.Type, modelConfidence, "charcnn-model"
		}
	}
	ruleType, ruleConfidence := classifyQuestion(question)
	return ruleType, ruleConfidence, "rule-based"
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
