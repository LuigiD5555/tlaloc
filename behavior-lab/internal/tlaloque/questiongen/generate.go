package questiongen

import (
	"context"
	"encoding/json"
	"fmt"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// GenerateQuestions selects the GENERATE_PAGE_QUESTIONS worker via the
// registry, preferring the semantic model generator. If that worker errors
// (LM Studio unreachable, malformed response, too few questions), it falls
// back to the deterministic template worker and reports that explicitly in
// workerID — the caller must never be told the model generated the
// questions when it was really the deterministic fallback.
func GenerateQuestions(ctx context.Context, registry *tlaloque.Registry, in GenerateInput) (out GenerateOutput, workerID string, err error) {
	inputRaw, err := json.Marshal(in)
	if err != nil {
		return GenerateOutput{}, "", err
	}
	req := tlaloque.CapabilityRequest{Input: inputRaw}

	if modelWorker, ok := registry.Get(SemanticModelWorkerID); ok {
		if resp, execErr := modelWorker.Execute(ctx, req); execErr == nil {
			var generated GenerateOutput
			if err := json.Unmarshal(resp.Output, &generated); err == nil {
				return generated, resp.WorkerID, nil
			}
		}
	}

	fallbackWorker, ok := registry.Get(TemplateWorkerID)
	if !ok {
		return GenerateOutput{}, "", fmt.Errorf("questiongen: no %q worker registered", TemplateWorkerID)
	}
	resp, err := fallbackWorker.Execute(ctx, req)
	if err != nil {
		return GenerateOutput{}, "", err
	}
	var generated GenerateOutput
	if err := json.Unmarshal(resp.Output, &generated); err != nil {
		return GenerateOutput{}, "", err
	}
	return generated, resp.WorkerID, nil
}
