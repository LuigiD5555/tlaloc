package microisadecoder

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// DecodeCarrier is the tool-consultation entry point: given the raw bytes
// of a SAFE_MICRO_ISA carrier.png, it asks microisa-cnn-r0 (via registry)
// to read the glyph's four attributes. This is what a loro-facing Tlaloque
// (e.g. a future swarmask-style node) would call to consult this model as
// a tool, rather than guessing from the image itself.
func DecodeCarrier(ctx context.Context, registry *tlaloque.Registry, carrierPNG []byte) (GlyphOutput, error) {
	worker, ok := registry.Get(WorkerID)
	if !ok {
		return GlyphOutput{}, fmt.Errorf("microisadecoder: worker %q not registered", WorkerID)
	}

	inputRaw, err := json.Marshal(GlyphInput{CarrierPNGBase64: base64.StdEncoding.EncodeToString(carrierPNG)})
	if err != nil {
		return GlyphOutput{}, err
	}

	resp, err := worker.Execute(ctx, tlaloque.CapabilityRequest{Input: inputRaw})
	if err != nil {
		return GlyphOutput{}, fmt.Errorf("microisadecoder: %w", err)
	}

	var out GlyphOutput
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		return GlyphOutput{}, fmt.Errorf("microisadecoder: decoding output: %w", err)
	}
	return out, nil
}
