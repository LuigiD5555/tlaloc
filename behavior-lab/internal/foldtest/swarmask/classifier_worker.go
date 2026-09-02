package swarmask

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/tlaloque"
	"tlaloc.local/behaviorlab/internal/tlaloque/calibration"
	"tlaloc.local/behaviorlab/internal/tlaloque/questionclass"
)

// minModelConfidence is the raw-confidence floor used ONLY when no
// CalibrationProfile is available for the model — a weak guard. With a
// profile, calibration.Verdict decides instead, and for
// questionclass-charcnn-r0 that verdict is always an abstention (its
// out-of-distribution accuracy is ~0.51, so its measured confidence floor
// is unreachable). See tools/QUESTIONCLASS_RESULTS.md.
const minModelConfidence = 0.7

// QuestionClassifierWorker classifies the question's rhetorical shape
// (definition, comparison, process, factual-detail, or general),
// independent of what PageScoutWorker (where) and EntityScoutWorker (what
// concrete facts) report — this helps the parrot (and the consolidator)
// calibrate how much verification a question needs. Unlike scout/entities,
// it always emits an observation: "this looks like a general question" is
// itself useful information, not a fabrication, since every question has
// *some* shape.
//
// If ModelRegistry is set (a questionclass-charcnn-r0 HTTP service), it uses
// that trained char-CNN and falls back to the rule-based classifyQuestion
// when the model errors, or — when Profile is set — when
// calibration.Verdict says the model's prediction is not trustworthy for
// this input (LOW_EVIDENCE / UNKNOWN / UNSUPPORTED). The observation's
// provenance always records which path produced the verdict, and the
// model's raw verdict even when it was overruled. With ModelRegistry nil
// it is purely rule-based.
type QuestionClassifierWorker struct {
	ModelRegistry *tlaloque.Registry
	Profile       *calibration.CalibrationProfile
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

	questionType, confidence, provenance := w.classify(ctx, in.Question)
	provenance["source"] = ClassifierWorkerID
	out := QuestionTypeOutput{Type: questionType}

	value, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	observations := []blackboard.Observation{{
		Key:        questionTypeKey,
		Value:      value,
		Confidence: confidence,
		Provenance: provenance,
	}}

	raw, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: ClassifierWorkerID, Output: raw, Observations: observations}, nil
}

// classify picks the trained char-CNN verdict only when a model is wired
// AND its prediction clears the trust check for this input; otherwise the
// deterministic rules. The returned provenance map records the path taken
// ("charcnn-model" / "rule-based") and, when the model was consulted but
// overruled, its raw verdict and why it was rejected — so the blackboard
// never claims a model classified when the fallback did, but the model's
// opinion is still auditable.
func (w QuestionClassifierWorker) classify(ctx context.Context, question string) (questionType string, confidence float64, provenance map[string]string) {
	provenance = map[string]string{}

	if w.ModelRegistry != nil {
		out, modelConfidence, err := questionclass.Classify(ctx, w.ModelRegistry, question)
		switch {
		case err != nil:
			provenance["model_rejected"] = "unreachable"
		default:
			provenance["model_verdict"] = out.Type
			if reason := w.rejectModel(modelConfidence); reason != "" {
				provenance["model_rejected"] = reason
			} else {
				provenance["method"] = "charcnn-model"
				return out.Type, modelConfidence, provenance
			}
		}
	}

	ruleType, ruleConfidence := classifyQuestion(question)
	provenance["method"] = "rule-based"
	return ruleType, ruleConfidence, provenance
}

// rejectModel returns a non-empty reason when the model's prediction should
// not be trusted. With a CalibrationProfile it defers to
// calibration.Verdict (measured competence); without one it falls back to
// the weak raw-confidence floor.
func (w QuestionClassifierWorker) rejectModel(modelConfidence float64) string {
	if w.Profile != nil {
		if verdict := w.Profile.Verdict(calibration.Query{Confidence: modelConfidence}); verdict != calibration.Answered {
			return fmt.Sprintf("calibration:%s", verdict)
		}
		return ""
	}
	if modelConfidence < minModelConfidence {
		return "uncalibrated-low-confidence"
	}
	return ""
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
