package swarmask

import (
	"context"
	"encoding/json"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/foldtest"
	"tlaloc.local/behaviorlab/internal/tlaloque"
	"tlaloc.local/behaviorlab/internal/tlaloque/answerscore"
)

// ConsolidatorWorker closes the loop: it checks whether the loro's answer
// is actually supported by the real content of the page PageScoutWorker
// suggested — something none of the other nodes can do, since they only
// ever see the cover and the question, never real page content. Reuses
// answerscore (already built and verified this session) rather than
// reimplementing grounding judgment.
type ConsolidatorWorker struct{}

func (ConsolidatorWorker) Descriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{
		ID:             ConsolidatorWorkerID,
		Capability:     ConsolidatorCapability,
		Scope:          tlaloque.ScopeGeneral,
		Engine:         tlaloque.EngineDeterministic,
		InputSchema:    inputSchema,
		OutputSchema:   consolidationOutputSchema,
		Deterministic:  false, // delegates to answerscore's model judge when available
		MaxConcurrency: 1,
		Tags:           []string{"consolidator", "grounding-check"},
	}
}

func (ConsolidatorWorker) Execute(ctx context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in AskInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, err
	}

	// req.Context is keyed by the plan's node ID ("answer"), populated
	// automatically from DependsOn (see swarm_runtime.go's depContext).
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

	// A page that can't be read, or a grounding judge that errors, isn't
	// fatal to the run — it just means this answer couldn't be verified,
	// same as when the scout found no page at all: degrade gracefully
	// rather than failing the whole swarm run over a verification step.
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
		Provenance: map[string]string{"source": ConsolidatorWorkerID, "method": "answerscore"},
	}}

	raw, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: ConsolidatorWorkerID, Output: raw, Observations: observations}, nil
}

// evaluateGrounding is the pure scoring step, separated from Execute's I/O
// (reading the blackboard, extracting page content) so it can be tested
// with literal strings, no store or network required.
func evaluateGrounding(ctx context.Context, model, baseURL, question, answer, pageContent string) (ConsolidationOutput, error) {
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

// suggestedPageFromBlackboard finds PageScoutWorker's suggestion, if any,
// on the shared blackboard snapshot.
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
