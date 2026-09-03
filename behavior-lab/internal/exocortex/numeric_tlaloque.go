package exocortex

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// NumericTlaloqueID is the CapabilityDescriptor.ID this Tlaloque registers
// under. If Go can do a comparison or a parse deterministically, it must
// never be sent to Parrot (P1's own conclusion about formatting/arithmetic).
const NumericTlaloqueID = "numeric-tlaloque"

// NumericInput is the input contract for COMPARE_NUMBERS/SAME_DIFFERENT.
type NumericInput struct {
	A string `json:"a"`
	B string `json:"b"`
}

// NumericOutput reports a deterministic comparison result.
type NumericOutput struct {
	A          float64 `json:"a"`
	B          float64 `json:"b"`
	Comparison string  `json:"comparison"` // LESS | EQUAL | GREATER
	Same       bool    `json:"same"`
}

// NumericTlaloque is a purely deterministic Go worker: parse numeric
// values, compare them, and report same/different. It never calls a model.
type NumericTlaloque struct{}

func (NumericTlaloque) Descriptor() tlaloque.CapabilityDescriptor {
	d, _ := tlaloque.CapabilityDescriptor{
		ID: NumericTlaloqueID, Capability: OpCompareNumbers, Engine: tlaloque.EngineDeterministic,
		InputSchema: "exocortex.numeric-input.r0", OutputSchema: "exocortex.numeric-output.r0",
		Deterministic: true, MaxConcurrency: 8,
	}.Normalize()
	return d
}

func (NumericTlaloque) Execute(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in NumericInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("numeric tlaloque: decode input: %w", err)
	}
	a, err := parseNumber(in.A)
	if err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("numeric tlaloque: operand a: %w", err)
	}
	b, err := parseNumber(in.B)
	if err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("numeric tlaloque: operand b: %w", err)
	}
	comparison := "EQUAL"
	switch {
	case a < b:
		comparison = "LESS"
	case a > b:
		comparison = "GREATER"
	}
	out := NumericOutput{A: a, B: b, Comparison: comparison, Same: a == b}
	body, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	obsValue, _ := json.Marshal(out)
	return tlaloque.CapabilityResponse{
		WorkerID: NumericTlaloqueID, Output: body, Confidence: 1,
		Observations: []blackboard.Observation{{Key: string(req.NodeID), Value: obsValue, Confidence: 1, Provenance: map[string]string{"source": NumericTlaloqueID}}},
	}, nil
}

// parseNumber accepts plain-text model output too (trims whitespace and a
// leading currency/unit-free sign) so it can sit directly downstream of a
// Normalize Tlaloque without re-deriving parsing rules.
func parseNumber(raw string) (float64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("empty numeric value")
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("not a number: %q", raw)
	}
	return v, nil
}
