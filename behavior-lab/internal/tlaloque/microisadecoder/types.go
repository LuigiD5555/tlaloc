// Package microisadecoder is the Go client side of microisa-cnn-r0, a
// genuinely trained (not prompted, not deterministic) tiny CNN that decodes
// Origami's SAFE_MICRO_ISA glyph carriers (origami/tools/microisa_*.py, a
// separate Go module — Origami owns its own representation, Tlaloc only
// consumes it here). The model itself lives in the origami module and is
// served by origami/tools/microisa_serve.py over HTTP_JSON; this package
// only registers it as a tlaloque.HTTPWorker and offers a thin client call.
package microisadecoder

import "tlaloc.local/behaviorlab/internal/tlaloque"

const (
	Capability = "DECODE_MICROISA_GLYPH"
	WorkerID   = "microisa-cnn-r0"

	inputSchema  = "tlaloc.microisadecoder.r0.glyph-input"
	outputSchema = "tlaloc.microisadecoder.r0.glyph-output"
)

// GlyphInput is the CapabilityRequest.Input payload: a single SAFE_MICRO_ISA
// carrier.png, base64-encoded so the request never assumes a shared
// filesystem between the Go caller and the Python service.
type GlyphInput struct {
	CarrierPNGBase64 string `json:"carrier_png_base64"`
}

// GlyphOutput is the CapabilityResponse.Output payload: the four discrete
// glyph attributes microisa-cnn-r0 was trained to read.
type GlyphOutput struct {
	Shape     int `json:"shape"`
	Holes     int `json:"holes"`
	Direction int `json:"direction"`
	Frames    int `json:"frames"`
}

// MicroISADescriptor is the single source of truth for this worker's
// capability descriptor.
func MicroISADescriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{
		ID:             WorkerID,
		Capability:     Capability,
		Scope:          tlaloque.ScopeGeneral,
		Engine:         tlaloque.EngineModel,
		InputSchema:    inputSchema,
		OutputSchema:   outputSchema,
		Deterministic:  false,
		ParameterCount: 24_480, // microisa-cnn-r0, trained from scratch this session
		MaxConcurrency: 1,
		Tags:           []string{"trained-specialist", "origami-glyph-decoder", "resident"},
	}
}
