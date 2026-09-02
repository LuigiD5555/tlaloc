package groundingautomaton

import (
	"context"
	"encoding/json"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

type Worker struct{}

func (Worker) Descriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{
		ID:             WorkerID,
		Capability:     Capability,
		Scope:          tlaloque.ScopeGeneral,
		Engine:         tlaloque.EngineDeterministic,
		InputSchema:    InputSchema,
		OutputSchema:   OutputSchema,
		Deterministic:  true,
		MaxConcurrency: 0,
		Tags:           []string{"grounding", "verification", "lexical", "r0"},
	}
}

func (Worker) Execute(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in VerifyInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	out := Verify(in)
	raw, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{
		WorkerID:   WorkerID,
		Output:     raw,
		Confidence: out.Confidence,
		Notes:      string(out.Verdict),
	}, nil
}
