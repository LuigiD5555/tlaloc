package swarmbench

import (
	"fmt"
	"strings"
)

// deterministicSequence is an explicit splitmix64 generator. The standard
// library makes no promise that its algorithms stay stable across Go
// versions, and the dataset must rebuild byte-identically from its seed.
type deterministicSequence struct{ state uint64 }

func (sequence *deterministicSequence) next() uint64 {
	sequence.state += 0x9E3779B97F4A7C15
	value := sequence.state
	value = (value ^ (value >> 30)) * 0xBF58476D1CE4E5B9
	value = (value ^ (value >> 27)) * 0x94D049BB133111EB
	return value ^ (value >> 31)
}

func (sequence *deterministicSequence) pick(count int) int {
	if count <= 0 {
		return 0
	}
	return int(sequence.next() % uint64(count))
}

type documentTemplate struct {
	intent string
	format string
}

// Templates keep every intent expressed in several surface forms so that a
// classifier cannot win on a single lexical cue.
var documentTemplates = []documentTemplate{
	{IntentPay, "Necesito pagar la factura de %ORG% por %AMOUNT% %DATE%"},
	{IntentPay, "Favor de programar el pago a %ORG% de %AMOUNT% para %DATE%"},
	{IntentCancel, "Solicito cancelar el CFDI de %ORG% por %AMOUNT% emitido %DATE%"},
	{IntentCancel, "Hay que cancelar el comprobante de %ORG% de %AMOUNT% con fecha %DATE%"},
	{IntentQuery, "Pueden consultar el saldo de %ORG%, son %AMOUNT% con vencimiento %DATE%"},
	{IntentQuery, "Consulta el estado de cuenta de %ORG% por %AMOUNT% al corte %DATE%"},
	{IntentSearch, "Busca los comprobantes de %ORG% por %AMOUNT% de %DATE%"},
	{IntentSearch, "Localiza la documentacion de %ORG% de %AMOUNT% emitida %DATE%"},
}

// Organizations is the fixed gazetteer both the dataset generator and the
// deterministic entity Tlaloque draw from. Sharing one list is not
// leaking ground truth per item — a real NER system built on a gazetteer
// works the same way: it knows the vocabulary, not the answer to any one
// document.
var Organizations = []string{
	"ACME Servicios",
	"Grupo Textil del Norte",
	"Distribuidora Peninsular",
	"Servicios Integrales Anahuac",
	"Comercializadora Bajio",
	"Transportes Union",
	"Refacciones Industriales Lerma",
	"Consultoria Zapopan",
}

// Amounts straddle TreasuryThresholdCents so routing genuinely depends on the
// recovered amount rather than on a constant.
var amountsCents = []int64{
	89_000,
	156_075,
	320_050,
	499_900,
	500_000,
	780_025,
	1_245_000,
	4_500_000,
}

// Expressions straddle the reference date so routing genuinely depends on the
// resolved date as well.
var dateExpressions = []string{
	"el proximo viernes",
	"el martes pasado",
	"en 30 dias",
	"hace 10 dias",
	"a fin de mes",
	"la semana pasada",
	"hoy",
	"ayer",
	"manana",
}

var referenceDates = []string{
	"2026-03-10",
	"2026-06-15",
	"2026-09-01",
	"2025-11-20",
}

// Generate builds a reproducible dataset. The same seed and count always
// produce byte-identical items.
func Generate(datasetID string, seed int64, count int) (Dataset, error) {
	datasetID = strings.TrimSpace(datasetID)
	if datasetID == "" {
		return Dataset{}, fmt.Errorf("dataset id is required")
	}
	if count <= 0 {
		return Dataset{}, fmt.Errorf("dataset requires at least one item")
	}
	sequence := deterministicSequence{state: uint64(seed)}
	dataset := Dataset{Schema: DatasetSchemaR0, ID: datasetID, Seed: seed, Items: make([]Item, 0, count)}

	for index := 0; index < count; index++ {
		template := documentTemplates[sequence.pick(len(documentTemplates))]
		organization := Organizations[sequence.pick(len(Organizations))]
		amountCents := amountsCents[sequence.pick(len(amountsCents))]
		dateExpression := dateExpressions[sequence.pick(len(dateExpressions))]
		referenceDate := referenceDates[sequence.pick(len(referenceDates))]

		dateISO, err := ResolveDate(dateExpression, referenceDate)
		if err != nil {
			return Dataset{}, fmt.Errorf("item %d: %w", index, err)
		}
		route, err := ResolveRoute(template.intent, amountCents, dateISO, referenceDate)
		if err != nil {
			return Dataset{}, fmt.Errorf("item %d: %w", index, err)
		}

		text := template.format
		text = strings.ReplaceAll(text, "%ORG%", organization)
		text = strings.ReplaceAll(text, "%AMOUNT%", FormatAmount(amountCents))
		text = strings.ReplaceAll(text, "%DATE%", dateExpression)

		dataset.Items = append(dataset.Items, Item{
			ID:             fmt.Sprintf("%s-%04d", datasetID, index),
			Text:           text,
			ReferenceDate:  referenceDate,
			DateExpression: dateExpression,
			Expected: Fields{
				Intent:       template.intent,
				Organization: organization,
				AmountCents:  amountCents,
				DateISO:      dateISO,
				Route:        route,
			},
		})
	}
	return dataset, nil
}

// Validate re-derives every dependent field so a corrupted or hand-edited
// dataset cannot silently become the ground truth.
func (dataset Dataset) Validate() error {
	if dataset.Schema != DatasetSchemaR0 {
		return fmt.Errorf("unexpected dataset schema %q", dataset.Schema)
	}
	if len(dataset.Items) == 0 {
		return fmt.Errorf("dataset has no items")
	}
	seen := map[string]bool{}
	for _, item := range dataset.Items {
		if seen[item.ID] {
			return fmt.Errorf("duplicate item id %q", item.ID)
		}
		seen[item.ID] = true

		dateISO, err := ResolveDate(item.DateExpression, item.ReferenceDate)
		if err != nil {
			return fmt.Errorf("item %s: %w", item.ID, err)
		}
		if dateISO != item.Expected.DateISO {
			return fmt.Errorf("item %s: expected date %s, derived %s", item.ID, item.Expected.DateISO, dateISO)
		}
		route, err := ResolveRoute(item.Expected.Intent, item.Expected.AmountCents, dateISO, item.ReferenceDate)
		if err != nil {
			return fmt.Errorf("item %s: %w", item.ID, err)
		}
		if route != item.Expected.Route {
			return fmt.Errorf("item %s: expected route %s, derived %s", item.ID, item.Expected.Route, route)
		}
		if !strings.Contains(item.Text, item.Expected.Organization) {
			return fmt.Errorf("item %s: organization %q absent from text", item.ID, item.Expected.Organization)
		}
		amountCents, err := ExtractAmount(item.Text)
		if err != nil {
			return fmt.Errorf("item %s: %w", item.ID, err)
		}
		if amountCents != item.Expected.AmountCents {
			return fmt.Errorf("item %s: expected amount %d, text carries %d", item.ID, item.Expected.AmountCents, amountCents)
		}
	}
	return nil
}
