// Package swarmbench holds the measurement instrument for the decomposition
// experiment: a compositional task with exact ground truth, a scorer, and the
// topology metrics the cost model is fitted against.
package swarmbench

import (
	"fmt"
	"strings"
	"time"
)

const (
	DatasetSchemaR0 = "tlaloc.swarm-bench-dataset.r0"
	ScoreSchemaR0   = "tlaloc.swarm-bench-score.r0"
)

// Intents one document may express.
const (
	IntentPay    = "PAGAR"
	IntentCancel = "CANCELAR"
	IntentQuery  = "CONSULTAR"
	IntentSearch = "BUSCAR"
)

// Routing destinations.
const (
	RouteTreasury   = "TESORERIA"
	RouteCompliance = "CUMPLIMIENTO"
	RouteArchive    = "ARCHIVO"
	RouteSupport    = "SOPORTE"
)

// TreasuryThresholdCents is the amount at or above which a scheduled payment
// is escalated to treasury.
const TreasuryThresholdCents int64 = 500_000

// Fields is everything one document contributes. Each field is owned by a
// different capability, which is what makes the task decomposable at all.
type Fields struct {
	Intent       string `json:"intent"`
	Organization string `json:"organization"`
	AmountCents  int64  `json:"amount_cents"`
	DateISO      string `json:"date_iso"`
	Route        string `json:"route"`
}

// Item is one scored document. ReferenceDate is carried explicitly so that
// relative date expressions resolve deterministically on any machine and on
// any day the benchmark is replayed.
type Item struct {
	ID             string `json:"id"`
	Text           string `json:"text"`
	ReferenceDate  string `json:"reference_date"`
	DateExpression string `json:"date_expression"`
	Expected       Fields `json:"expected"`
}

type Dataset struct {
	Schema string `json:"schema"`
	ID     string `json:"id"`
	Seed   int64  `json:"seed"`
	Items  []Item `json:"items"`
}

// ResolveRoute derives the routing decision from the recovered fields. It is
// the single source of truth: the dataset generator and the router Tlaloque
// both call it, so a swarm can only route correctly by recovering every
// upstream field correctly. Field errors therefore propagate into the route
// exactly the way the depth term of the cost model predicts.
func ResolveRoute(intent string, amountCents int64, dateISO string, referenceISO string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(intent)) {
	case IntentCancel:
		return RouteCompliance, nil
	case IntentQuery, IntentSearch:
		return RouteArchive, nil
	case IntentPay:
		due, err := time.Parse(dateLayout, strings.TrimSpace(dateISO))
		if err != nil {
			return "", fmt.Errorf("route: unparsable date %q: %w", dateISO, err)
		}
		reference, err := time.Parse(dateLayout, strings.TrimSpace(referenceISO))
		if err != nil {
			return "", fmt.Errorf("route: unparsable reference %q: %w", referenceISO, err)
		}
		// An overdue payment is a compliance matter regardless of size.
		if due.Before(reference) {
			return RouteCompliance, nil
		}
		if amountCents >= TreasuryThresholdCents {
			return RouteTreasury, nil
		}
		return RouteSupport, nil
	default:
		return "", fmt.Errorf("route: unknown intent %q", intent)
	}
}
