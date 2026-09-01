package swarmbench

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/realcampaign"
)

const LMStudioBaselineSchemaR0 = "tlaloc.swarm-bench-lmstudio-baseline.r0"

// singleShotSystemPrompt asks one model to resolve every field in one call —
// the undecomposed condition Phase 5 needs: a single model whose total
// parameter count is comparable to the swarm's sum (128 x 12M individuals
// ~= 1.5B, close to lfm2-vl-1.6b's declared 1.6B), doing the entire task
// end to end instead of five narrow individuals each doing one part of it.
const singleShotSystemPrompt = `Analiza el documento y extrae los siguientes campos exactos. Responde unicamente un objeto JSON con esta forma, sin explicacion:
{"intent":"PAGAR|CANCELAR|CONSULTAR|BUSCAR","organization":"nombre exacto","amount":"monto con simbolo de moneda, ej. $1,200.00","date_expression":"la expresion de fecha tal como aparece en el texto, ej. 'el proximo viernes'"}`

// ResolveFieldsSingleShot asks the model to recover every field the swarm
// splits across five individuals in one call. The route itself is
// deterministically derived from the model's own recovered fields via
// ResolveRoute (the same reference logic the router Tlaloque uses) rather
// than asked of the model directly — this isolates what is actually being
// compared: extraction capability, not whether the model also knows the
// business rule for routing, which no individual in the swarm needs to
// improvise either.
func (c LMStudioClient) ResolveFieldsSingleShot(ctx context.Context, item Item) (fields Fields, raw string, err error) {
	raw, _, _, err = c.complete(ctx, singleShotSystemPrompt, item.Text, 120)
	if err != nil {
		return Fields{}, "", err
	}
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var decoded struct {
		Intent         string `json:"intent"`
		Organization   string `json:"organization"`
		Amount         string `json:"amount"`
		DateExpression string `json:"date_expression"`
	}
	if jsonErr := json.Unmarshal([]byte(cleaned), &decoded); jsonErr != nil {
		// A model that cannot even produce parseable JSON has failed the
		// item outright — score it as a total miss, not a transport error;
		// the swarm's own workers apply the identical "wrong answer is not a
		// crash" rule (see intentWorker/entityWorker in swarm.go).
		return Fields{}, raw, nil
	}
	intent := strings.ToUpper(strings.TrimSpace(decoded.Intent))
	if !knownIntents[intent] {
		intent = ""
	}
	fields.Intent = intent
	fields.Organization = strings.TrimSpace(decoded.Organization)
	if amountCents, amountErr := ParseAmountCents(strings.TrimPrefix(strings.TrimSpace(decoded.Amount), "$")); amountErr == nil {
		fields.AmountCents = amountCents
	}
	if dateISO, dateErr := ResolveDate(decoded.DateExpression, item.ReferenceDate); dateErr == nil {
		fields.DateISO = dateISO
	}
	if fields.Intent != "" && fields.DateISO != "" {
		if route, routeErr := ResolveRoute(fields.Intent, fields.AmountCents, fields.DateISO, item.ReferenceDate); routeErr == nil {
			fields.Route = route
		}
	}
	return fields, raw, nil
}

// BaselineCase is one item's single-shot outcome, mirroring ProbeCase's
// shape for the per-capability probe.
type BaselineCase struct {
	ItemID    string    `json:"item_id"`
	RawOutput string    `json:"raw_output"`
	Got       Fields    `json:"got"`
	ItemScore ItemScore `json:"item_score"`
	Error     string    `json:"error,omitempty"`
	LatencyMS int64     `json:"latency_ms"`
}

// BaselineLog is the single-model, undecomposed-condition counterpart to
// Run (the swarm's own paired Score/Topology record) — same model-tagging
// discipline as ProbeLog, so a baseline result is never read as evidence
// about a model other than the one that actually produced it.
type BaselineLog struct {
	Schema       string                           `json:"schema"`
	Endpoint     string                           `json:"endpoint"`
	Model        string                           `json:"model"`
	ModelInterop realcampaign.ModelInteropProfile `json:"model_interop"`
	StartedAt    time.Time                        `json:"started_at"`
	FinishedAt   time.Time                        `json:"finished_at"`
	Cases        []BaselineCase                   `json:"cases"`
	Score        Score                            `json:"score"`
}

// RunLMStudioBaseline runs the single-shot, undecomposed condition against
// sampleSize items and scores it with the exact same ScoreDataset the swarm
// uses — the two Score values are directly comparable.
func RunLMStudioBaseline(ctx context.Context, client LMStudioClient, dataset Dataset, sampleSize int) BaselineLog {
	log := BaselineLog{
		Schema:       LMStudioBaselineSchemaR0,
		Endpoint:     client.Endpoint,
		Model:        client.Model,
		ModelInterop: realcampaign.BuildModelInteropProfile(client.Model, "openai-chat-completions", "DIRECT"),
		StartedAt:    time.Now().UTC(),
	}
	items := dataset.Items
	if sampleSize > 0 && sampleSize < len(items) {
		items = items[:sampleSize]
	}
	byItem := make(map[string]Fields, len(items))
	for _, item := range items {
		start := time.Now()
		got, raw, err := client.ResolveFieldsSingleShot(ctx, item)
		elapsed := time.Since(start)
		entry := BaselineCase{ItemID: item.ID, RawOutput: raw, Got: got, LatencyMS: elapsed.Milliseconds()}
		if err != nil {
			entry.Error = err.Error()
		} else {
			entry.ItemScore = ScoreItem(item, got)
		}
		log.Cases = append(log.Cases, entry)
		byItem[item.ID] = got
	}
	sampled := Dataset{Schema: dataset.Schema, ID: dataset.ID, Seed: dataset.Seed, Items: items}
	log.Score = ScoreDataset(sampled, func(item Item) Fields { return byItem[item.ID] })
	log.FinishedAt = time.Now().UTC()
	return log
}
