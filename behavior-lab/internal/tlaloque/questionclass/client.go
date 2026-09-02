package questionclass

import (
	"context"
	"encoding/json"
	"fmt"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// Classify is the tool-consultation entry point: given a question, it asks
// questionclass-charcnn-r0 (via registry) for the question's rhetorical
// shape and the model's confidence in that verdict. A caller that also has
// a rule-based classifier (internal/foldtest/swarmask) should treat a
// returned error, or a confidence below its own threshold, as a signal to
// fall back — and report which path actually produced the verdict.
func Classify(ctx context.Context, registry *tlaloque.Registry, question string) (Output, float64, error) {
	worker, ok := registry.Get(WorkerID)
	if !ok {
		return Output{}, 0, fmt.Errorf("questionclass: worker %q not registered", WorkerID)
	}

	inputRaw, err := json.Marshal(Input{Question: question})
	if err != nil {
		return Output{}, 0, err
	}

	resp, err := worker.Execute(ctx, tlaloque.CapabilityRequest{Input: inputRaw})
	if err != nil {
		return Output{}, 0, fmt.Errorf("questionclass: %w", err)
	}

	var out Output
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		return Output{}, 0, fmt.Errorf("questionclass: decoding output: %w", err)
	}
	return out, resp.Confidence, nil
}
