package swarmask

import (
	"context"
	"encoding/json"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/foldtest"
	"tlaloc.local/behaviorlab/internal/tlaloque"
	"tlaloc.local/behaviorlab/internal/tlaloque/answerscore"
	"tlaloc.local/behaviorlab/internal/tlaloque/groundingautomaton"
)

// ConsolidatorWorker closes the loop: it checks whether the loro's answer
// is actually supported by the real content of the page PageScoutWorker
// suggested. Grounding now gives the deterministic automaton first authority;
// learned/heuristic answer scorers are consulted only when R0 abstains.
type ConsolidatorWorker struct{}

func (ConsolidatorWorker) Descriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{
		ID:             ConsolidatorWorkerID,
		Capability:     ConsolidatorCapability,
		Scope:          tlaloque.ScopeGeneral,
		Engine:         tlaloque.EngineDeterministic,
		InputSchema:    inputSchema,
		OutputSchema:   consolidationOutputSchema,
		Deterministic:  false, // deterministic first; learned scorer may handle abstentions
		MaxConcurrency: 1,
		Tags:           []string{"consolidator", "grounding-check", "deterministic-first"},
	}
}

func (ConsolidatorWorker) Execute(ctx context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in AskInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, err
	}

	var answerOut AnswerOutput
	if raw, ok := req.Context["answer"]; ok {
		_ = json.Unmarshal(raw, &answerOut)
	}

	page, found := suggestedPageFromBlackboard(req)
	out := ConsolidationOutput{Answer: answerOut.Answer}
	if !found {
		raw, err := json.Marshal(out)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		return tlaloque.CapabilityResponse{WorkerID: ConsolidatorWorkerID, Output: raw}, nil
	}

	pageContent, err := foldtest.ExtractPageContent(in.StoreDir, in.Manifest, page)
	if err != nil {
		raw, marshalErr := json.Marshal(out)
		if marshalErr != nil {
			return tlaloque.CapabilityResponse{}, marshalErr
		}
		return tlaloque.CapabilityResponse{WorkerID: ConsolidatorWorkerID, Output: raw}, nil
	}

	grounded, err := evaluateGrounding(ctx, in.Model, in.BaseURL, in.Question, answerOut.Answer, pageContent)
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
	observations := []blackboard.Observation{{
		Key:        answerGroundedKey,
		Value:      value,
		Confidence: out.Score,
		Provenance: map[string]string{"source": ConsolidatorWorkerID, "method": out.ScoredBy},
	}}

	raw, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: ConsolidatorWorkerID, Output: raw, Observations: observations}, nil
}

// evaluateGrounding is the pure scoring step. The deterministic automaton is
// authoritative only for SUPPORTED/CONTRADICTED. UNKNOWN/INSUFFICIENT are
// explicit abstentions and continue through the existing answer-score cascade.
func evaluateGrounding(ctx context.Context, model, baseURL, question, answer, pageContent string) (ConsolidationOutput, error) {
	automaton := groundingautomaton.Verify(groundingautomaton.VerifyInput{
		Question:    question,
		ModelAnswer: answer,
		PageContent: pageContent,
	})
	if groundingautomaton.HasAuthority(automaton) {
		grounded := automaton.Verdict == groundingautomaton.VerdictSupported
		score := automaton.Confidence
		if automaton.Verdict == groundingautomaton.VerdictContradicted {
			score = 0.0
		}
		return ConsolidationOutput{
			Grounded: grounded,
			Score:    score,
			ScoredBy: groundingautomaton.WorkerID,
		}, nil
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
		Grounded: scored.Score >= 0.5,
		Score:    scored.Score,
		ScoredBy: workerID,
	}, nil
}

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
