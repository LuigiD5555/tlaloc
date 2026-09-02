package questiongen

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// TemplateWorker is the deterministic fallback for GENERATE_PAGE_QUESTIONS:
// fixed question templates plus up to two keyword-derived questions from the
// page's first sentences. It never fails and never needs a network round
// trip, so it is always available as a floor under SemanticModelWorker.
type TemplateWorker struct{}

func (TemplateWorker) Descriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{
		ID:             TemplateWorkerID,
		Capability:     Capability,
		Scope:          tlaloque.ScopeGeneral,
		Engine:         tlaloque.EngineDeterministic,
		InputSchema:    inputSchema,
		OutputSchema:   outputSchema,
		Deterministic:  true,
		MaxConcurrency: 0, // normalized to 1
		Tags:           []string{"template", "fallback"},
	}
}

func (w TemplateWorker) Execute(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in GenerateInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, err
	}

	out := GenerateOutput{Questions: templateQuestions(in.PageContent, in.PageNumber)}

	raw, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: TemplateWorkerID, Output: raw}, nil
}

// templateQuestions is the fixed-template question generator moved verbatim
// from the former internal/foldtest.GeneratePageQuestions.
func templateQuestions(pageContent string, pageNumber int) []string {
	questions := []string{
		fmt.Sprintf("¿Cuál es el contenido principal de la página %d?", pageNumber),
		fmt.Sprintf("¿Qué términos técnicos o conceptos clave aparecen en la página %d?", pageNumber),
		fmt.Sprintf("Resumen brevemente los puntos importantes de la página %d", pageNumber),
	}

	// Extract first 3 sentences as potential topics
	sentences := strings.Split(pageContent, ".")
	if len(sentences) > 2 {
		for i := 0; i < 2 && i < len(sentences); i++ {
			keyword := extractKeyword(sentences[i])
			if keyword != "" {
				questions = append(questions, fmt.Sprintf("¿Cómo se relaciona '%s' con el contenido de la página %d?", keyword, pageNumber))
			}
		}
	}

	return questions
}

// extractKeyword finds a meaningful word from text, moved verbatim from the
// former internal/foldtest.extractKeyword.
func extractKeyword(text string) string {
	words := strings.Fields(strings.ToLower(strings.TrimSpace(text)))
	if len(words) == 0 {
		return ""
	}

	stopwords := map[string]bool{"el": true, "la": true, "los": true, "las": true, "un": true, "una": true, "y": true, "o": true, "que": true, "de": true, "en": true, "es": true, "a": true}
	for _, word := range words {
		word = strings.Trim(word, ".,;:!?")
		if !stopwords[word] && len(word) > 3 {
			return word
		}
	}
	return ""
}
