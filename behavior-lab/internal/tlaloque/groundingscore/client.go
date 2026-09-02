package groundingscore

import (
	"context"
	"encoding/json"
	"fmt"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// Score asks groundingscore-distilled-r0 (via registry) how well an answer
// is grounded in a page passage. It returns the distilled score, the
// model's self-estimated confidence, and any error. A caller that also has
// a heavier judge (the swarm consolidator falls back to answerscore) should
// treat a returned error — or a confidence its calibration profile deems
// untrustworthy — as a signal to fall back, and record which path produced
// the verdict.
func Score(ctx context.Context, registry *tlaloque.Registry, in Input) (Output, float64, error) {
	worker, ok := registry.Get(WorkerID)
	if !ok {
		return Output{}, 0, fmt.Errorf("groundingscore: worker %q not registered", WorkerID)
	}

	inputRaw, err := json.Marshal(in)
	if err != nil {
		return Output{}, 0, err
	}

	resp, err := worker.Execute(ctx, tlaloque.CapabilityRequest{Input: inputRaw})
	if err != nil {
		return Output{}, 0, fmt.Errorf("groundingscore: %w", err)
	}

	var out Output
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		return Output{}, 0, fmt.Errorf("groundingscore: decoding output: %w", err)
	}
	confidence := resp.Confidence
	if confidence == 0 {
		confidence = out.Confidence
	}
	return out, confidence, nil
}
