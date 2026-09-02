package questiongen

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// SemanticModelWorker generates varied, content-grounded questions by asking
// a target model (via the same target.OpenAICompat client the harness
// already uses to talk to LM Studio). It is strictly a generator: it never
// answers its own questions, and any caller must be able to fall back to
// TemplateWorker when it errors or returns too few usable questions.
type SemanticModelWorker struct {
	Client target.OpenAICompat
}

var questionLinePattern = regexp.MustCompile(`(?i)^\s*Q:\s*(.+?)\s*$`)

func (w SemanticModelWorker) Descriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{
		ID:             SemanticModelWorkerID,
		Capability:     Capability,
		Scope:          tlaloque.ScopeGeneral,
		Engine:         tlaloque.EngineModel,
		InputSchema:    inputSchema,
		OutputSchema:   outputSchema,
		Deterministic:  false,
		MaxConcurrency: 1,
		Tags:           []string{"semantic-generator", "lm-studio", w.Client.Model},
	}
}

func (w SemanticModelWorker) Execute(ctx context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in GenerateInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, err
	}

	system := "Eres un generador de preguntas de prueba. Dado el contenido de una página, " +
		"escribe entre 3 y 5 preguntas variadas que solo puedan responderse leyendo ese " +
		"contenido (no preguntes sobre el número de página ni sobre metadatos). " +
		"Responde EXACTAMENTE con una pregunta por línea, cada línea con el formato:\n" +
		"Q: <pregunta>"

	user := fmt.Sprintf("CONTENIDO DE LA PÁGINA %d:\n%s", in.PageNumber, in.PageContent)

	raw, err := w.Client.Complete(ctx, system, user)
	if err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("semantic question generator: %w", err)
	}

	questions := parseQuestions(raw)
	if len(questions) < minModelQuestions {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("semantic question generator: got %d usable questions, want at least %d (raw=%q)", len(questions), minModelQuestions, raw)
	}

	out := GenerateOutput{Questions: questions}
	outputRaw, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: SemanticModelWorkerID, Output: outputRaw}, nil
}

func parseQuestions(raw string) []string {
	var questions []string
	for _, line := range strings.Split(raw, "\n") {
		match := questionLinePattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		question := strings.TrimSpace(match[1])
		if question != "" {
			questions = append(questions, question)
		}
	}
	return questions
}
