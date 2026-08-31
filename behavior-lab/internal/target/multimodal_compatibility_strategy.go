package target

import (
	"fmt"
	"strings"
)

const (
	CompatibilityOpenAI   = "openai"
	CompatibilityLMStudio = "lm-studio"
	CompatibilityMinimal  = "minimal"
)

// MultimodalCompatibilityStrategy isolates provider-specific differences in
// OpenAI-compatible multimodal payloads. Transport code depends on this
// behavior instead of branching on providers/endpoints itself.
type MultimodalCompatibilityStrategy interface {
	Name() string
	ImageURLPart(dataURL string) map[string]any
}

type OpenAICompatibilityStrategy struct{}

func (OpenAICompatibilityStrategy) Name() string { return CompatibilityOpenAI }
func (OpenAICompatibilityStrategy) ImageURLPart(dataURL string) map[string]any {
	return map[string]any{"url": dataURL, "detail": "auto"}
}

type LMStudioCompatibilityStrategy struct{}

func (LMStudioCompatibilityStrategy) Name() string { return CompatibilityLMStudio }
func (LMStudioCompatibilityStrategy) ImageURLPart(dataURL string) map[string]any {
	// Origami carriers are dense visual artifacts, so request the highest
	// standards-compatible image detail that LM Studio accepts.
	return map[string]any{"url": dataURL, "detail": "high"}
}

type MinimalCompatibilityStrategy struct{}

func (MinimalCompatibilityStrategy) Name() string { return CompatibilityMinimal }
func (MinimalCompatibilityStrategy) ImageURLPart(dataURL string) map[string]any {
	// Use only the required field for servers that reject optional image_url
	// properties while still implementing the OpenAI-compatible shape.
	return map[string]any{"url": dataURL}
}

func ResolveMultimodalCompatibility(name string) (MultimodalCompatibilityStrategy, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", CompatibilityOpenAI:
		return OpenAICompatibilityStrategy{}, nil
	case CompatibilityLMStudio, "lmstudio", "lm_studio":
		return LMStudioCompatibilityStrategy{}, nil
	case CompatibilityMinimal:
		return MinimalCompatibilityStrategy{}, nil
	default:
		return nil, fmt.Errorf("unsupported multimodal compatibility strategy %q", name)
	}
}

func (c OpenAICompat) multimodalCompatibility() MultimodalCompatibilityStrategy {
	if c.Compatibility != nil {
		return c.Compatibility
	}
	return OpenAICompatibilityStrategy{}
}
