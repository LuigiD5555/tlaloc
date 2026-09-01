package swarmbench

import (
	"fmt"
	"strings"
)

// This file holds the core logic of the five real Tlaloques the
// decomposition experiment runs, decoupled from transport. cmd/tlaloc-swarm-worker
// (PROCESS) and cmd/tlaloc-swarm-model-server (HTTP_JSON) are thin adapters
// over these same functions, so the binaries provably behave exactly like
// what is unit-tested here.

// intentLexicon maps a surface cue to the intent it signals. Longer, more
// specific cues are checked first so "consulta el estado" does not fall
// through to a shorter, coincidentally-matching cue.
var intentLexicon = []struct {
	cue    string
	intent string
}{
	{"consultar", IntentQuery},
	{"consulta", IntentQuery},
	{"estado de cuenta", IntentQuery},
	{"saldo", IntentQuery},
	{"cancelar", IntentCancel},
	{"cancelacion", IntentCancel},
	{"busca", IntentSearch},
	{"localiza", IntentSearch},
	{"pagar", IntentPay},
	{"pago", IntentPay},
}

// IntentWorkerLogic is a heuristic lexical classifier standing in for a
// small trained model (e.g. intent-bert-12m). It is intentionally not the
// deterministic reference: it reads surface text the way a small model
// would, so it can be genuinely wrong on a document ResolveRoute would
// still resolve correctly if fed accurate fields. Confidence reflects how
// many candidate cues matched — a single unambiguous hit scores highest.
func IntentWorkerLogic(text string) (intent string, confidence float64) {
	lower := strings.ToLower(text)
	matches := 0
	found := ""
	for _, entry := range intentLexicon {
		if strings.Contains(lower, entry.cue) {
			matches++
			if found == "" {
				found = entry.intent
			}
		}
	}
	if found == "" {
		return "", 0
	}
	if matches == 1 {
		return found, 0.97
	}
	return found, 0.75
}

// EntityWorkerLogic recovers the organization by gazetteer match. Real small
// NER models lean on exactly this kind of lexicon; the point under test is
// whether the swarm degrades gracefully when this stage is imperfect, not
// whether gazetteer matching itself is a fair opponent for a 12M-parameter
// model.
func EntityWorkerLogic(text string) (organization string, confidence float64) {
	longestMatch := ""
	for _, candidate := range Organizations {
		if strings.Contains(text, candidate) && len(candidate) > len(longestMatch) {
			longestMatch = candidate
		}
	}
	if longestMatch == "" {
		return "", 0
	}
	return longestMatch, 0.99
}

// DateNumberWorkerLogic is the deterministic, zero-parameter Tlaloque: it
// recovers both the resolved date and the monetary amount by exact pattern
// match, never by inference. This is the individual the decomposition
// hypothesis expects to win outright against any learned competitor.
func DateNumberWorkerLogic(text string, referenceDate string) (dateISO string, amountCents int64, err error) {
	dateISO, dateErr := ExtractDate(text, referenceDate)
	amountCents, amountErr := ExtractAmount(text)
	if dateErr != nil && amountErr != nil {
		return "", 0, fmt.Errorf("date-number: %v; %v", dateErr, amountErr)
	}
	if dateErr != nil {
		return "", amountCents, fmt.Errorf("date-number: %w", dateErr)
	}
	if amountErr != nil {
		return dateISO, 0, fmt.Errorf("date-number: %w", amountErr)
	}
	return dateISO, amountCents, nil
}

// RouteWorkerLogic assembles the fields recovered by its three dependencies
// (intent, entity, date-number) and applies the one authoritative routing
// rule. It is deterministic: given the same four inputs it always derives
// the same route, whether or not those inputs happen to be correct.
func RouteWorkerLogic(intent, organization string, amountCents int64, dateISO, referenceDate string) (Fields, error) {
	route, err := ResolveRoute(intent, amountCents, dateISO, referenceDate)
	if err != nil {
		return Fields{}, fmt.Errorf("router: %w", err)
	}
	return Fields{
		Intent:       intent,
		Organization: organization,
		AmountCents:  amountCents,
		DateISO:      dateISO,
		Route:        route,
	}, nil
}

// OrgHeadWorkerLogic and OrgTailWorkerLogic are the genuine decomposition of
// EntityWorkerLogic: instead of one worker recovering the whole organization
// name, two narrower workers each recover one half of the same gazetteer
// match — the way separate NER heads might specialize on different token
// positions. Both independently re-derive which gazetteer entry matched, so
// a join downstream (JoinOrganizationWorkerLogic) never has to trust that
// the two halves agree by construction; they can genuinely disagree if one
// atom fails and the other does not.
func OrgHeadWorkerLogic(text string) (head string, confidence float64) {
	organization, confidence := EntityWorkerLogic(text)
	if organization == "" {
		return "", 0
	}
	words := strings.Fields(organization)
	return words[0], confidence
}

func OrgTailWorkerLogic(text string) (tail string, confidence float64) {
	organization, confidence := EntityWorkerLogic(text)
	if organization == "" {
		return "", 0
	}
	words := strings.Fields(organization)
	return strings.Join(words[1:], " "), confidence
}

// JoinOrganizationWorkerLogic is the deterministic, zero-parameter assembler
// that recombines the two narrow extractors' output. It has no gazetteer of
// its own — it only knows how to concatenate what it is handed, which is
// exactly what keeps a Tlaloque bounded.
func JoinOrganizationWorkerLogic(head, tail string) string {
	if head == "" {
		return ""
	}
	if tail == "" {
		return head
	}
	return head + " " + tail
}

// VerifyWorkerLogic is the fifth Tlaloque: it independently re-derives the
// route from the fields the router already assembled and corrects it on
// mismatch. This is where composition should pay for itself — a cheap
// deterministic check catching a router-stage error introduced upstream by
// a wrong intent or entity guess, without re-running either of them.
func VerifyWorkerLogic(fields Fields, referenceDate string) (corrected Fields, changed bool, err error) {
	recomputed, err := ResolveRoute(fields.Intent, fields.AmountCents, fields.DateISO, referenceDate)
	if err != nil {
		// The router already validated intent/date once; a failure here means
		// the fields it emitted are themselves malformed, which the verifier
		// must surface rather than silently pass through.
		return fields, false, fmt.Errorf("verifier: %w", err)
	}
	if recomputed == fields.Route {
		return fields, false, nil
	}
	fields.Route = recomputed
	return fields, true, nil
}
