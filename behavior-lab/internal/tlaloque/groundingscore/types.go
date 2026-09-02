// Package groundingscore is the Go client side of groundingscore-distilled-r0:
// a small MLP (a few hundred parameters) trained to imitate a strong
// chat-model grounding judge from cheap features over frozen MiniLM-L6
// embeddings (see tools/grounding_*.py, tools/GROUNDING_RESULTS.md).
//
// It exists so the swarm consolidator (internal/foldtest/swarmask) has a
// grounding judge that is NOT the parrot (lfm2-vl-1.6b) that produced the
// answer — removing the "model grading its own homework" caveat — while
// staying cheap enough to run on every answer. The heavier chat judge in
// internal/tlaloque/answerscore stays as the honest fallback when this
// model errors or its calibration profile says it is out of its depth.
//
// This package only registers the model as a tlaloque.HTTPWorker pointed at
// a running tools/grounding_serve.py instance and offers a thin client call;
// the CapabilityWorker contract (worker-identity check, JSON validity) is
// enforced by http_worker.go unmodified. Same convention as
// internal/tlaloque/questionclass and internal/tlaloque/microisadecoder.
package groundingscore

import "tlaloc.local/behaviorlab/internal/tlaloque"

const (
	// Capability matches answerscore's so this worker produces the same
	// judgement shape; it is selected explicitly by the consolidator, not
	// mixed into answerscore's own candidate order.
	Capability = "SCORE_ANSWER_RELEVANCE"
	WorkerID   = "groundingscore-distilled-r0"

	inputSchema  = "tlaloc.foldtest.r0.answer-score-input"
	outputSchema = "tlaloc.foldtest.r0.answer-score-output"
)

// Input is the CapabilityRequest.Input payload. Field names match
// answerscore.ScoreInput so a caller can reuse one struct for both.
type Input struct {
	Question    string `json:"question"`
	ModelAnswer string `json:"model_answer"`
	PageContent string `json:"page_content"`
}

// Output is the CapabilityResponse.Output payload: the distilled grounding
// score and the model's self-estimated confidence in it.
type Output struct {
	Score      float64 `json:"score"`
	Confidence float64 `json:"confidence"`
	Notes      string  `json:"notes,omitempty"`
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
		ParameterCount: 638, // groundingscore-distilled-r0 MLP head, trained from scratch (12->24->12->2)
		MaxConcurrency: 1,
		Tags:           []string{"trained-specialist", "grounding-judge", "distilled", "minilm-features", "resident"},
	}
}
