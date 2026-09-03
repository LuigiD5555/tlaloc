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

// NormalizeTlaloqueID is the CapabilityDescriptor.ID this Tlaloque
// registers under. P1's own conclusion was that formatting must not be
// carried by Parrot; this Tlaloque is the deterministic Go replacement.
const NormalizeTlaloqueID = "normalize-tlaloque"

// NormalizeInput carries a raw model (or any executor) text output plus
// the target primitive type to convert it to.
type NormalizeInput struct {
	Raw        string `json:"raw"`
	TargetType string `json:"target_type"` // number | text | choice
}

// NormalizeOutput is the deterministic, typed conversion result.
type NormalizeOutput struct {
	Trimmed  string  `json:"trimmed"`
	AsNumber float64 `json:"as_number,omitempty"`
	IsNumber bool    `json:"is_number"`
}

// NormalizeTlaloque trims, canonicalizes and converts plain text to a typed
// primitive deterministically. It never calls a model.
type NormalizeTlaloque struct{}

const (
	TargetTypeNumber = "number"
	TargetTypeText   = "text"
	TargetTypeChoice = "choice"
)

func (NormalizeTlaloque) Descriptor() tlaloque.CapabilityDescriptor {
	d, _ := tlaloque.CapabilityDescriptor{
		ID: NormalizeTlaloqueID, Capability: OpNormalize, Engine: tlaloque.EngineDeterministic,
		InputSchema: "exocortex.normalize-input.r0", OutputSchema: "exocortex.normalize-output.r0",
		Deterministic: true, MaxConcurrency: 8,
	}.Normalize()
	return d
}

func (NormalizeTlaloque) Execute(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in NormalizeInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("normalize tlaloque: decode input: %w", err)
	}
	trimmed := canonicalizeText(in.Raw)
	out := NormalizeOutput{Trimmed: trimmed}
	switch strings.ToLower(strings.TrimSpace(in.TargetType)) {
	case TargetTypeNumber:
		if v, err := strconv.ParseFloat(trimmed, 64); err == nil {
			out.AsNumber = v
			out.IsNumber = true
		}
	case TargetTypeText, TargetTypeChoice, "":
		// trimmed text/choice needs no further conversion.
	default:
		return tlaloque.CapabilityResponse{}, fmt.Errorf("normalize tlaloque: unsupported target_type %q", in.TargetType)
	}
	body, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{
		WorkerID: NormalizeTlaloqueID, Output: body, Confidence: 1,
		Observations: []blackboard.Observation{{Key: req.NodeID, Value: body, Confidence: 1, Provenance: map[string]string{"source": NormalizeTlaloqueID}}},
	}, nil
}

// canonicalizeText trims surrounding whitespace/quotes and collapses
// internal whitespace runs, without altering case or content — the R0
// Normalize Tlaloque is deliberately simple (E0.13: no free-form NLP).
func canonicalizeText(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, "\"'")
	s = strings.TrimSpace(s)
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
