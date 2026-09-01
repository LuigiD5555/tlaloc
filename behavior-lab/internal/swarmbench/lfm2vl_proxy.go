package swarmbench

import "strings"

// This file calibrates the intent/entity stand-ins against a real probe run
// against lfm2-vl-1.6b, resident locally in LM Studio at
// http://127.0.0.1:1234/v1, recorded in runs/lm-studio/lfm2-vl-1.6b-probe-001.json
// (40 items, seed 2026, dataset "lmstudio-probe", generated 2026-08-31).
//
// These numbers are specific to that exact model. As the user who ran the
// probe put it: what works for one person's local model does not
// necessarily work for someone else's — this profile is not a claim about
// small models in general, only a record of one observed local model's
// behavior on this exact task and prompt.
//
// Observed intent behavior (DETECT_INTENT, zero-shot, temperature 0):
//   39/40 raw outputs were the literal string "CONSULTAR" regardless of the
//   true intent. Accuracy landed at 10/40 = 0.25 — statistically
//   indistinguishable from always answering "CONSULTAR" against this
//   dataset's uniform 4-way intent split (an "always guess the same label"
//   baseline scores ~0.25 here by construction). The model was not
//   discriminating; it collapsed to one output.
//
// Observed entity behavior (EXTRACT_ENTITY): 31/40 = 0.775. Of the 9 misses:
//   6 hijacked the literal substring "CFDI" (present in one of the CANCELAR
//   templates: "...cancelar el CFDI de %ORG%...") instead of the actual
//   organization name; 1 truncated the organization to its last word; 2
//   returned the correct organization with a diacritic the source text
//   itself omits ("Consultoria" -> "Consultoría") — arguably not a real
//   error, so it is deliberately not reproduced here.
//
// Corroboration: runs/lm-studio/lfm2-vl-1.6b-compare-001.json later ran the
// same model live inside the actual five-Tlaloque swarm (via
// IntentLogicLive/EntityLogicLive in lmstudio_live.go — real calls, not this
// stand-in) against a fresh sample of the same dataset/seed. Its decomposed
// condition measured intent=0.250 and organization=0.775 — matching this
// file's calibrated values almost exactly, independently. Treat that as
// confidence in the stand-in below, not as a second, separate calibration.
//
// What that same comparison also surfaced, and what a proxy for one isolated
// capability can never capture by construction: on identical items, the
// SAME model asked to resolve every field in a single call
// (lfm2-vl-1.6b-baseline-001.json / ResolveFieldsSingleShot in
// lmstudio_baseline.go) scored intent=0.450 and organization=0.450. Isolating
// intent into its own narrow one-word-classification prompt made it WORSE
// (0.250 vs 0.450) — the richer single-shot prompt apparently gave the model
// more signal to disambiguate intent than the bare classification prompt did.
// Isolating organization made it BETTER (0.775 vs 0.450) — a dedicated
// extraction prompt outperformed extracting it alongside three other fields.
// So decomposition's effect on a model-backed capability is not uniform: it
// helped one of these two and hurt the other. The swarm's real win on this
// task (route_accuracy 0.550 decomposed vs 0.325 single-shot, same run) came
// almost entirely from replacing amount/date with deterministic, zero-
// parameter extractors (1.000 either way) — not from decomposition alone.

// IntentWorkerLogicLFM2VLProxy reproduces the observed near-total collapse:
// the real model's answer was "CONSULTAR" in 39 of 40 probe cases regardless
// of ground truth, so the faithful stand-in is the constant it actually
// produced, not an invented partial-accuracy heuristic. This is the
// capability where isolating it into its own narrow prompt made the model
// worse, not better — see the corroboration note above before assuming
// decomposition should always help a model-backed individual.
func IntentWorkerLogicLFM2VLProxy(_ string) (intent string, confidence float64) {
	return IntentQuery, 0.25
}

// entityTruncationRate approximates the observed 1-in-40 truncation-to-last-
// word failure. n=40 is a small sample; treat this as an order-of-magnitude
// calibration, not a precise rate.
const entityTruncationRate = 100 / 40

// EntityWorkerLogicLFM2VLProxy reproduces the two generalizable observed
// entity failures — CFDI-literal hijack and occasional truncation to the
// last word — layered on top of the same gazetteer match a correct read
// would produce.
func EntityWorkerLogicLFM2VLProxy(text string) (organization string, confidence float64) {
	if containsWord(text, "CFDI") {
		return "CFDI", 0.6
	}
	organization, confidence = EntityWorkerLogic(text)
	if organization == "" {
		return organization, confidence
	}
	sequence := deterministicSequence{state: seedFromText(text)}
	if sequence.pick(100) < entityTruncationRate {
		words := strings.Fields(organization)
		return words[len(words)-1], confidence
	}
	return organization, confidence
}

func containsWord(text, word string) bool {
	for _, token := range strings.FieldsFunc(text, func(r rune) bool {
		return !('A' <= r && r <= 'Z' || 'a' <= r && r <= 'z' || '0' <= r && r <= '9')
	}) {
		if token == word {
			return true
		}
	}
	return false
}

func seedFromText(text string) uint64 {
	var seed uint64
	for _, character := range text {
		seed = seed*131 + uint64(character)
	}
	return seed
}
