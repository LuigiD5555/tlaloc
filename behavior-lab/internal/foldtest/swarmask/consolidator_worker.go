package swarmask

import (
	"context"
	"encoding/json"
	"fmt"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/foldtest"
	"tlaloc.local/behaviorlab/internal/tlaloque"
	"tlaloc.local/behaviorlab/internal/tlaloque/answerscore"
	"tlaloc.local/behaviorlab/internal/tlaloque/calibration"
	"tlaloc.local/behaviorlab/internal/tlaloque/groundingautomaton"
	"tlaloc.local/behaviorlab/internal/tlaloque/groundingscore"
)

// ConsolidatorWorker closes the loop: it checks whether the parrot's answer
// is actually supported by the real content of the page PageScoutWorker
// suggested — something none of the other nodes can do, since they only ever
// see the cover and the question, never real page content.
//
// Judge order, each independent of the parrot model that wrote the answer:
//  1. grounding-automaton-r0, the deterministic verifier. When it returns
//     SUPPORTED or CONTRADICTED that verdict is authoritative — no model
//     second opinion. UNKNOWN / INSUFFICIENT are explicit abstentions and
//     fall through.
//  2. groundingscore-distilled-r0 (only when GroundingRegistry is set): the
//     trained MLP judge. When GroundingProfile is set, calibration.Verdict
//     gates whether its score is trustworthy for this input.
//  3. answerscore (chat judge -> embedding -> keyword), the final fallback.
//
// Provenance on the blackboard records which judge produced the verdict and
// whether it was independent of the answering model.
type ConsolidatorWorker struct {
	GroundingRegistry *tlaloque.Registry
	GroundingProfile  *calibration.CalibrationProfile
	// DisableAutomaton turns off the deterministic first pass (default: on).
	// Used to measure the learned judges in isolation.
	DisableAutomaton bool
}

func (w ConsolidatorWorker) Descriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{
		ID:             ConsolidatorWorkerID,
		Capability:     ConsolidatorCapability,
		Scope:          tlaloque.ScopeGeneral,
		Engine:         tlaloque.EngineDeterministic,
		InputSchema:    inputSchema,
		OutputSchema:   consolidationOutputSchema,
		Deterministic:  false, // deterministic first; learned scorers handle abstentions
		MaxConcurrency: 1,
		Dependencies:   []string{AnswerCapability},
		Tags:           []string{"consolidator", "grounding-check", "deterministic-first"},
	}
}

func (w ConsolidatorWorker) Execute(ctx context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in AskInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, err
	}

	answerOut := answerFromContext(req)

	page, found := suggestedPageFromBlackboard(req)
	out := ConsolidationOutput{Answer: answerOut.Answer}
	if !found {
		raw, err := json.Marshal(out)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		return tlaloque.CapabilityResponse{WorkerID: ConsolidatorWorkerID, Output: raw}, nil
	}

	// A page that can't be read, or a grounding judge that errors, isn't
	// fatal to the run — it just means this answer couldn't be verified:
	// degrade gracefully rather than failing the whole swarm run.
	pageContent, err := foldtest.ExtractPageContent(in.StoreDir, in.Manifest, page)
	if err != nil {
		raw, marshalErr := json.Marshal(out)
		if marshalErr != nil {
			return tlaloque.CapabilityResponse{}, marshalErr
		}
		return tlaloque.CapabilityResponse{WorkerID: ConsolidatorWorkerID, Output: raw}, nil
	}

	grounded, err := w.evaluateGrounding(ctx, in.Model, in.BaseURL, in.Question, answerOut.Answer, pageContent)
	if err != nil {
		raw, marshalErr := json.Marshal(out)
		if marshalErr != nil {
			return tlaloque.CapabilityResponse{}, marshalErr
		}
		return tlaloque.CapabilityResponse{WorkerID: ConsolidatorWorkerID, Output: raw}, nil
	}
	out = grounded
	out.Answer = answerOut.Answer
	out.VerifiedAgainstPage = page

	value, err := json.Marshal(answerGroundedObservation{Page: page, Score: out.Score, ScoredBy: out.ScoredBy})
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}

	method := "answerscore"
	switch out.ScoredBy {
	case groundingautomaton.WorkerID:
		method = "grounding-automaton-r0"
	case groundingscore.WorkerID:
		method = "groundingscore-distilled"
	}

	// Blackboard confidence describes trust in the observation, not the
	// probability the answer is supported. A deterministic contradiction has
	// score 0 but full confidence.
	observationConfidence := out.Score
	if out.ScoredBy == groundingautomaton.WorkerID && !out.Grounded {
		observationConfidence = 1.0
	}

	observations := []blackboard.Observation{{
		Key:        answerGroundedKey,
		Value:      value,
		Confidence: observationConfidence,
		Provenance: map[string]string{
			"source":            ConsolidatorWorkerID,
			"method":            method,
			"scored_by":         out.ScoredBy,
			"judge_independent": fmt.Sprintf("%t", out.JudgeIndependent),
		},
	}}

	raw, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: ConsolidatorWorkerID, Output: raw, Observations: observations}, nil
}

// evaluateGrounding is the pure scoring step, separated from Execute's I/O so
// it can be tested with literal strings, no store or network required. See
// the ConsolidatorWorker doc for the judge order.
func (w ConsolidatorWorker) evaluateGrounding(ctx context.Context, model, baseURL, question, answer, pageContent string) (ConsolidationOutput, error) {
	if !w.DisableAutomaton {
		verdict := groundingautomaton.Verify(groundingautomaton.VerifyInput{
			Question:    question,
			ModelAnswer: answer,
			PageContent: pageContent,
		})
		if groundingautomaton.HasAuthority(verdict) {
			grounded := verdict.Verdict == groundingautomaton.VerdictSupported
			score := verdict.Confidence
			if !grounded {
				score = 0.0
			}
			return ConsolidationOutput{
				Grounded:         grounded,
				Score:            score,
				ScoredBy:         groundingautomaton.WorkerID,
				JudgeIndependent: true,
			}, nil
		}
	}

	if w.GroundingRegistry != nil {
		scored, confidence, err := groundingscore.Score(ctx, w.GroundingRegistry, groundingscore.Input{
			Question:    question,
			ModelAnswer: answer,
			PageContent: pageContent,
		})
		if err == nil && w.groundingTrusted(confidence) {
			return ConsolidationOutput{
				Grounded:         scored.Score >= 0.5,
				Score:            scored.Score,
				ScoredBy:         groundingscore.WorkerID,
				JudgeIndependent: true,
			}, nil
		}
	}

	registry := answerscore.NewRegistry(model, baseURL, "")
	scored, workerID, err := answerscore.ScoreAnswer(ctx, registry, answerscore.ScoreInput{
		Question:    question,
		ModelAnswer: answer,
		PageContent: pageContent,
	})
	if err != nil {
		return ConsolidationOutput{}, err
	}
	return ConsolidationOutput{
		Grounded:         scored.Score >= 0.5,
		Score:            scored.Score,
		ScoredBy:         workerID,
		JudgeIndependent: workerID != answerscore.SemanticModelWorkerID,
	}, nil
}

// groundingTrusted reports whether the distilled score should be used as-is.
// With a CalibrationProfile it defers to calibration.Verdict (measured
// competence); without one it always trusts the model, matching the "no
// profile = old behavior" contract used by the classifier node.
func (w ConsolidatorWorker) groundingTrusted(confidence float64) bool {
	if w.GroundingProfile == nil {
		return true
	}
	return w.GroundingProfile.Verdict(calibration.Query{Confidence: confidence}) == calibration.Answered
}

// answerFromContext pulls the parrot's AnswerOutput out of req.Context. The
// runner keys dependency outputs by the provider's plan node ID, which is
// "answer" in the hand-written plan and the parrot's worker ID in a
// capability-resolved plan — so try the known key first, then fall back to
// whichever context entry actually parses as a non-empty AnswerOutput.
func answerFromContext(req tlaloque.CapabilityRequest) AnswerOutput {
	var out AnswerOutput
	if raw, ok := req.Context["answer"]; ok {
		if json.Unmarshal(raw, &out) == nil && out.Answer != "" {
			return out
		}
	}
	if raw, ok := req.Context[AnswerWorkerID]; ok {
		if json.Unmarshal(raw, &out) == nil && out.Answer != "" {
			return out
		}
	}
	for _, raw := range req.Context {
		var candidate AnswerOutput
		if json.Unmarshal(raw, &candidate) == nil && candidate.Answer != "" {
			return candidate
		}
	}
	return AnswerOutput{}
}

// suggestedPageFromBlackboard finds PageScoutWorker's suggestion, if any, on
// the shared blackboard snapshot.
func suggestedPageFromBlackboard(req tlaloque.CapabilityRequest) (int, bool) {
	if req.Blackboard == nil {
		return 0, false
	}
	for _, entry := range req.Blackboard.Entries {
		if entry.Key != suggestedPageKey {
			continue
		}
		var suggestion suggestedPageObservation
		if err := json.Unmarshal(entry.Value, &suggestion); err != nil {
			continue
		}
		return suggestion.Page, true
	}
	return 0, false
}
