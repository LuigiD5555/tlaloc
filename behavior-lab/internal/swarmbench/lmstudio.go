package swarmbench

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/realcampaign"
)

const LMStudioProbeSchemaR0 = "tlaloc.swarm-bench-lmstudio-probe.r0"

// LMStudioClient talks to a locally resident OpenAI-compatible server (LM
// Studio's default endpoint, matching internal/realcampaign's own default).
// It records exactly which model answered every call — the model identity
// carried in every response is not optional, because a result observed
// against one local model says nothing about another: what works for one
// person's downloaded weights will not necessarily work for someone else's.
type LMStudioClient struct {
	Endpoint string // e.g. http://127.0.0.1:1234/v1
	Model    string // exact id as advertised by GET /v1/models
	Client   *http.Client
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatCompletionResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// complete sends one bounded chat completion request and returns the raw
// text the model produced, its self-reported model id (which must match what
// was requested — a silently substituted model invalidates the log), and
// token usage for the record.
func (c LMStudioClient) complete(ctx context.Context, systemPrompt, userContent string, maxTokens int) (text string, respondingModel string, usage struct{ PromptTokens, CompletionTokens int }, err error) {
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	body, marshalErr := json.Marshal(chatCompletionRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
		Temperature: 0,
		MaxTokens:   maxTokens,
	})
	if marshalErr != nil {
		return "", "", usage, marshalErr
	}
	httpReq, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.Endpoint, "/")+"/chat/completions", bytes.NewReader(body))
	if reqErr != nil {
		return "", "", usage, reqErr
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, doErr := client.Do(httpReq)
	if doErr != nil {
		return "", "", usage, fmt.Errorf("lm studio: %w", doErr)
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return "", "", usage, readErr
	}
	if resp.StatusCode/100 != 2 {
		return "", "", usage, fmt.Errorf("lm studio status %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	var decoded chatCompletionResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", "", usage, fmt.Errorf("lm studio: invalid response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return "", "", usage, fmt.Errorf("lm studio: empty choices")
	}
	if decoded.Model != "" && decoded.Model != c.Model {
		return "", "", usage, fmt.Errorf("lm studio: requested model %q, server answered as %q", c.Model, decoded.Model)
	}
	usage.PromptTokens = decoded.Usage.PromptTokens
	usage.CompletionTokens = decoded.Usage.CompletionTokens
	return strings.TrimSpace(decoded.Choices[0].Message.Content), decoded.Model, usage, nil
}

const intentSystemPrompt = "Clasifica la intencion del documento en exactamente una palabra: PAGAR, CANCELAR, CONSULTAR o BUSCAR. Responde unicamente la palabra, sin explicacion."

const entitySystemPrompt = "Extrae el nombre exacto de la organizacion mencionada en el documento. Responde unicamente el nombre, sin explicacion ni puntuacion adicional."

var knownIntents = map[string]bool{IntentPay: true, IntentCancel: true, IntentQuery: true, IntentSearch: true}

// ClassifyIntent asks the resident model for the document's intent. The raw
// text is always returned alongside the parsed label — including when the
// model's answer is not one of the four known intents — so a probe log
// never silently discards how the model actually failed.
func (c LMStudioClient) ClassifyIntent(ctx context.Context, text string) (intent string, raw string, err error) {
	raw, _, _, err = c.complete(ctx, intentSystemPrompt, text, 8)
	if err != nil {
		return "", "", err
	}
	normalized := strings.ToUpper(strings.TrimSpace(raw))
	normalized = strings.Trim(normalized, ".,;: \n\t\"'")
	if knownIntents[normalized] {
		return normalized, raw, nil
	}
	return "", raw, nil
}

// ExtractOrganization asks the resident model for the organization named in
// the document.
func (c LMStudioClient) ExtractOrganization(ctx context.Context, text string) (organization string, raw string, err error) {
	raw, _, _, err = c.complete(ctx, entitySystemPrompt, text, 24)
	if err != nil {
		return "", "", err
	}
	return strings.TrimSpace(raw), raw, nil
}

// ProbeCase is one item's outcome against the real model, for one capability.
type ProbeCase struct {
	ItemID     string `json:"item_id"`
	Capability string `json:"capability"`
	Text       string `json:"text"`
	Expected   string `json:"expected"`
	Got        string `json:"got"`
	RawOutput  string `json:"raw_output"`
	Correct    bool   `json:"correct"`
	Error      string `json:"error,omitempty"`
	LatencyMS  int64  `json:"latency_ms"`
}

// ProbeLog is the persistent, model-tagged record of one probe run. It is
// deliberately schema-stamped and carries the exact model id and its
// realcampaign.ModelInteropProfile so a log from one local model is never
// mistaken for evidence about a different one.
type ProbeLog struct {
	Schema         string                           `json:"schema"`
	Endpoint       string                           `json:"endpoint"`
	Model          string                           `json:"model"`
	ModelInterop   realcampaign.ModelInteropProfile `json:"model_interop"`
	StartedAt      time.Time                        `json:"started_at"`
	FinishedAt     time.Time                        `json:"finished_at"`
	Cases          []ProbeCase                      `json:"cases"`
	IntentAccuracy float64                          `json:"intent_accuracy"`
	EntityAccuracy float64                          `json:"entity_accuracy"`
}

// RunLMStudioProbe runs both capabilities against sampleSize items of
// dataset through the resident model and returns the full, model-tagged log.
// It never aborts on one item's failure — a probe run exists specifically to
// observe failures, not to stop at the first one.
func RunLMStudioProbe(ctx context.Context, client LMStudioClient, dataset Dataset, sampleSize int) ProbeLog {
	log := ProbeLog{
		Schema:       LMStudioProbeSchemaR0,
		Endpoint:     client.Endpoint,
		Model:        client.Model,
		ModelInterop: realcampaign.BuildModelInteropProfile(client.Model, "openai-chat-completions", "DIRECT"),
		StartedAt:    time.Now().UTC(),
	}
	items := dataset.Items
	if sampleSize > 0 && sampleSize < len(items) {
		items = items[:sampleSize]
	}
	intentCorrect, entityCorrect := 0, 0
	for _, item := range items {
		intentCase := probeIntent(ctx, client, item)
		log.Cases = append(log.Cases, intentCase)
		if intentCase.Correct {
			intentCorrect++
		}
		entityCase := probeEntity(ctx, client, item)
		log.Cases = append(log.Cases, entityCase)
		if entityCase.Correct {
			entityCorrect++
		}
	}
	log.FinishedAt = time.Now().UTC()
	if len(items) > 0 {
		log.IntentAccuracy = float64(intentCorrect) / float64(len(items))
		log.EntityAccuracy = float64(entityCorrect) / float64(len(items))
	}
	return log
}

func probeIntent(ctx context.Context, client LMStudioClient, item Item) ProbeCase {
	start := time.Now()
	got, raw, err := client.ClassifyIntent(ctx, item.Text)
	elapsed := time.Since(start)
	entry := ProbeCase{
		ItemID: item.ID, Capability: CapabilityDetectIntent, Text: item.Text,
		Expected: item.Expected.Intent, Got: got, RawOutput: raw, LatencyMS: elapsed.Milliseconds(),
	}
	if err != nil {
		entry.Error = err.Error()
		return entry
	}
	entry.Correct = got == item.Expected.Intent
	return entry
}

func probeEntity(ctx context.Context, client LMStudioClient, item Item) ProbeCase {
	start := time.Now()
	got, raw, err := client.ExtractOrganization(ctx, item.Text)
	elapsed := time.Since(start)
	entry := ProbeCase{
		ItemID: item.ID, Capability: CapabilityExtractEntity, Text: item.Text,
		Expected: item.Expected.Organization, Got: got, RawOutput: raw, LatencyMS: elapsed.Milliseconds(),
	}
	if err != nil {
		entry.Error = err.Error()
		return entry
	}
	entry.Correct = got == item.Expected.Organization
	return entry
}
