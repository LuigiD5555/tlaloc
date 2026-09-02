package swarmask

import (
	"context"
	"encoding/json"
	"fmt"

	"tlaloc.local/behaviorlab/internal/foldtest"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// ParrotAnswerWorker wraps foldtest.RunSession (the parrot, lfm2-vl-1.6b, doing
// its own UNFOLD/GROUP-driven reading of the whole-book cover) as a
// CapabilityWorker, so it can read whatever other specialists already
// posted to the blackboard — today, PageScoutWorker's "suggested_page" hint
// — before answering. It never trusts a hint blindly: it is offered as
// context in the prompt, not injected as the answer.
type ParrotAnswerWorker struct{}

func (ParrotAnswerWorker) Descriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{
		ID:             AnswerWorkerID,
		Capability:     AnswerCapability,
		Scope:          tlaloque.ScopeGeneral,
		Engine:         tlaloque.EngineModel,
		InputSchema:    inputSchema,
		OutputSchema:   answerOutputSchema,
		Deterministic:  false,
		MaxConcurrency: 1,
		Dependencies:   []string{ScoutCapability, EntityCapability, ClassifierCapability},
		Tags:           []string{"parrot", "lfm2-vl-1.6b", "blackboard-aware"},
	}
}

func (ParrotAnswerWorker) Execute(ctx context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in AskInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, err
	}

	extraContext := blackboardHint(req)

	sessionResult, err := foldtest.RunSession(ctx, foldtest.SessionConfig{
		WorkDir:      in.WorkDir,
		StoreDir:     in.StoreDir,
		Manifest:     in.Manifest,
		Cover:        in.Cover,
		Model:        in.Model,
		BaseURL:      in.BaseURL,
		MaxTurns:     in.MaxTurns,
		Budget:       in.Budget,
		ExtraContext: extraContext,
	}, in.Question)
	if err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("parrot answer: %w", err)
	}

	out := AnswerOutput{
		Answer:                sessionResult.Answer,
		Turns:                 sessionResult.Turns,
		TotalTokensPrompt:     sessionResult.TotalTokensPrompt,
		TotalTokensCompletion: sessionResult.TotalTokensCompletion,
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{
		WorkerID: AnswerWorkerID,
		Output:   raw,
		Usage: &tlaloque.WorkerUsage{
			TokensIn:      sessionResult.TotalTokensPrompt,
			TokensOut:     sessionResult.TotalTokensCompletion,
			UpstreamCalls: sessionResult.Turns,
		},
	}, nil
}

// blackboardHint formats every observation the specialists posted (page
// suggestions, question entities) into a short hint for the system prompt.
// There is normally at most one of each, but this reads all matching
// entries so a future second scout of either kind can contribute without
// code changes here.
func blackboardHint(req tlaloque.CapabilityRequest) string {
	if req.Blackboard == nil {
		return ""
	}
	hint := ""
	for _, entry := range req.Blackboard.Entries {
		switch entry.Key {
		case suggestedPageKey:
			var suggestion suggestedPageObservation
			if err := json.Unmarshal(entry.Value, &suggestion); err != nil {
				continue
			}
			hint += fmt.Sprintf("- A specialist suggests page %d may be relevant (confidence %.2f). You may UNFOLD it directly if it helps.\n", suggestion.Page, suggestion.Score)
		case questionEntitiesKey:
			var entities EntityOutput
			if err := json.Unmarshal(entry.Value, &entities); err != nil {
				continue
			}
			if len(entities.Years) > 0 {
				hint += fmt.Sprintf("- The question specifically mentions year(s): %v.\n", entities.Years)
			}
			if len(entities.Numbers) > 0 {
				hint += fmt.Sprintf("- The question specifically mentions number(s): %v.\n", entities.Numbers)
			}
			if len(entities.Acronyms) > 0 {
				hint += fmt.Sprintf("- The question specifically mentions acronym(s): %v.\n", entities.Acronyms)
			}
		case questionTypeKey:
			var classification QuestionTypeOutput
			if err := json.Unmarshal(entry.Value, &classification); err != nil {
				continue
			}
			hint += questionTypeGuidance(classification.Type)
		}
	}
	return hint
}

// questionTypeGuidance turns a classifier verdict into a short calibration
// line for the parrot's prompt.
func questionTypeGuidance(questionType string) string {
	switch questionType {
	case QuestionTypeDefinition:
		return "- This looks like a general definition question; the cover may already be enough to answer it.\n"
	case QuestionTypeComparison:
		return "- This looks like a comparison question; consider whether more than one page is relevant.\n"
	case QuestionTypeProcess:
		return "- This looks like a how/why process question; a general explanation from the cover may suffice.\n"
	case QuestionTypeFactualDetail:
		return "- This question asks for a specific fact; consider UNFOLDing the suggested page to verify it precisely.\n"
	default:
		return ""
	}
}
