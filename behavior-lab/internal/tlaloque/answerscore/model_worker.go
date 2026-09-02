package answerscore

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// SemanticModelWorker scores an answer's relevance by asking a target model
// (via the same target.OpenAICompat client the harness already uses to talk
// to LM Studio) to judge whether the answer is supported by the page
// content. It is strictly a judge: it never edits the answer, never
// certifies correctness on its own authority, and any caller must be able to
// fall back to KeywordOverlapWorker when it errors.
type SemanticModelWorker struct {
	Client target.OpenAICompat
}

var scoreLinePattern = regexp.MustCompile(`(?i)SCORE:\s*([01](?:\.\d+)?)`)
var confidenceLinePattern = regexp.MustCompile(`(?i)CONFIDENCE:\s*([01](?:\.\d+)?)`)
var notesLinePattern = regexp.MustCompile(`(?i)NOTES:\s*(.+)`)

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
		Tags:           []string{"semantic-judge", "lm-studio", w.Client.Model},
	}
}

func (w SemanticModelWorker) Execute(ctx context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in ScoreInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, err
	}

	system := "Eres un juez que evalúa si una respuesta está respaldada por el contenido de una página. " +
		"No respondas la pregunta ni corrijas la respuesta: solo evalúa. " +
		"Responde EXACTAMENTE en este formato, una línea por campo:\n" +
		"SCORE: <número entre 0.0 y 1.0, qué tan respaldada está la respuesta por el contenido>\n" +
		"CONFIDENCE: <número entre 0.0 y 1.0, tu confianza en ese juicio>\n" +
		"NOTES: <una frase breve explicando el score>"

	user := fmt.Sprintf("CONTENIDO DE LA PÁGINA:\n%s\n\nPREGUNTA:\n%s\n\nRESPUESTA A EVALUAR:\n%s",
		in.PageContent, in.Question, in.ModelAnswer)

	raw, err := w.Client.Complete(ctx, system, user)
	if err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("semantic scorer: %w", err)
	}

	out, err := parseScoreResponse(raw)
	if err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("semantic scorer: %w", err)
	}

	outputRaw, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: SemanticModelWorkerID, Output: outputRaw, Confidence: out.Confidence, Notes: out.Notes}, nil
}

func parseScoreResponse(raw string) (ScoreOutput, error) {
	scoreMatch := scoreLinePattern.FindStringSubmatch(raw)
	if scoreMatch == nil {
		return ScoreOutput{}, fmt.Errorf("no SCORE line in model response: %q", raw)
	}
	score, err := strconv.ParseFloat(scoreMatch[1], 64)
	if err != nil {
		return ScoreOutput{}, fmt.Errorf("invalid SCORE value %q: %w", scoreMatch[1], err)
	}

	confidence := 0.7
	if confidenceMatch := confidenceLinePattern.FindStringSubmatch(raw); confidenceMatch != nil {
		if parsed, err := strconv.ParseFloat(confidenceMatch[1], 64); err == nil {
			confidence = parsed
		}
	}

	notes := ""
	if notesMatch := notesLinePattern.FindStringSubmatch(raw); notesMatch != nil {
		notes = strings.TrimSpace(notesMatch[1])
	}

	return ScoreOutput{Score: score, Confidence: confidence, Notes: notes}, nil
}
