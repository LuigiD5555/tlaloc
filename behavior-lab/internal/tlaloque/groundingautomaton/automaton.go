package groundingautomaton

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var numberPattern = regexp.MustCompile(`[-+]?\d+(?:[.,]\d+)?%?`)

var stopwords = map[string]struct{}{
	"a": {}, "al": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "by": {},
	"con": {}, "de": {}, "del": {}, "dentro": {}, "el": {}, "en": {}, "es": {}, "esta": {},
	"este": {}, "for": {}, "from": {}, "in": {}, "is": {}, "la": {}, "las": {}, "los": {},
	"mediante": {}, "no": {}, "of": {}, "on": {}, "o": {}, "or": {}, "para": {}, "por": {},
	"que": {}, "se": {}, "su": {}, "sus": {}, "the": {}, "to": {}, "un": {}, "una": {}, "y": {},
}

var negativeMarkers = []string{
	" no ", " not ", " never ", " nunca ", " ninguno ", " ninguna ", " without ", " sin ",
	" does not ", " do not ", " did not ", " isn t ", " arent ", " aren t ", " wasn t ",
}

const (
	quantifierNone = iota
	quantifierAll
	quantifierSome
	quantifierNever
	quantifierAlways
)

var antonymPairs = [][2]string{
	{"increase", "decrease"}, {"increases", "decreases"}, {"increased", "decreased"},
	{"aumenta", "disminuye"}, {"aumentar", "disminuir"},
	{"before", "after"}, {"antes", "despues"}, {"antes", "después"},
	{"enable", "disable"}, {"enabled", "disabled"},
	{"habilitado", "deshabilitado"}, {"habilitada", "deshabilitada"},
	{"present", "absent"}, {"presente", "ausente"},
	{"true", "false"}, {"verdadero", "falso"}, {"verdadera", "falsa"},
	{"distributed", "centralized"}, {"distribuido", "centralizado"}, {"distribuida", "centralizada"},
	// R1 additions
	{"centralized", "decentralized"}, {"centralised", "decentralised"},
	{"central", "decentralized"}, {"central", "decentralised"},
	{"required", "optional"}, {"mandatory", "optional"}, {"obligatorio", "opcional"},
	{"up", "down"}, {"higher", "lower"}, {"more", "fewer"}, {"more", "less"},
	{"synchronous", "asynchronous"}, {"sincrono", "asincrono"},
	{"include", "exclude"}, {"includes", "excludes"}, {"incluye", "excluye"},
	{"toward", "away"}, {"towards", "away"}, {"hacia", "lejos"},
	{"static", "dynamic"}, {"estatico", "dinamico"},
	{"stateless", "stateful"}, {"mutable", "immutable"}, {"safe", "unsafe"},
	{"valid", "invalid"}, {"success", "failure"}, {"allow", "deny"}, {"allowed", "denied"},
}

func Verify(in VerifyInput) VerifyOutput {
	claims := splitClaims(in.ModelAnswer)
	if len(claims) == 0 {
		return VerifyOutput{Verdict: VerdictInsufficient, Claims: []ClaimTrace{}}
	}
	if isNonAnswer(in.ModelAnswer) {
		return VerifyOutput{Verdict: VerdictInsufficient, Claims: []ClaimTrace{}}
	}

	evidenceSpans := splitClaims(in.PageContent)
	traces := make([]ClaimTrace, 0, len(claims))
	covered, contradicted, supported := 0, 0, 0

	for _, claim := range claims {
		evidence, alignment := bestEvidence(claim, evidenceSpans)
		trace := ClaimTrace{Claim: claim, Evidence: evidence, Alignment: alignment, Verdict: VerdictUnknown}

		// A claim below the candidate threshold is still inspected for
		// CONTRADICTION (never support) when:
		//  - the span carries a lexically-anchored conflict (antonym of a
		//    claim term, or a clashing quantifier) and shares a core term, OR
		//  - the span opposes the claim's polarity AND the span is *mostly*
		//    shared core terms with the claim (a short negated sentence that
		//    is entirely about the claim's predicate — "there is no central
		//    scheduler"). Bare polarity parity with only a generic term in
		//    common is not enough: unrelated sentences often clash in
		//    polarity by chance.
		strongConflict := evidence != "" &&
			((anchoredConflictSignal(claim, evidence) && sharesCoreTerm(claim, evidence)) ||
				(hasNegation(claim) != hasNegation(evidence) && sharesStrongCore(claim, evidence)))

		if (alignment < AlignmentCandidateThreshold || evidence == "") && !strongConflict {
			trace.Reasons = append(trace.Reasons, Reason{Code: ReasonLowAlignment, Claim: claim, Evidence: evidence})
			traces = append(traces, trace)
			continue
		}

		covered++
		trace.Reasons = append(trace.Reasons, Reason{Code: ReasonAligned, Claim: claim, Evidence: evidence})

		if detail, ok := numericContradictionR1(claim, evidence); ok {
			trace.Verdict = VerdictContradicted
			trace.Reasons = append(trace.Reasons, Reason{Code: ReasonNumericContradiction, Claim: claim, Evidence: evidence, Detail: detail})
			contradicted++
			traces = append(traces, trace)
			continue
		}
		if detail, ok := boundContradiction(claim, evidence); ok {
			trace.Verdict = VerdictContradicted
			trace.Reasons = append(trace.Reasons, Reason{Code: ReasonNumericContradiction, Claim: claim, Evidence: evidence, Detail: detail})
			contradicted++
			traces = append(traces, trace)
			continue
		}
		if polarityContradiction(claim, evidence) {
			trace.Verdict = VerdictContradicted
			trace.Reasons = append(trace.Reasons, Reason{Code: ReasonPolarityContradiction, Claim: claim, Evidence: evidence})
			contradicted++
			traces = append(traces, trace)
			continue
		}
		if detail, ok := quantifierContradiction(claim, evidence); ok {
			trace.Verdict = VerdictContradicted
			trace.Reasons = append(trace.Reasons, Reason{Code: ReasonQuantifierContradiction, Claim: claim, Evidence: evidence, Detail: detail})
			contradicted++
			traces = append(traces, trace)
			continue
		}
		if detail, ok := antonymContradiction(claim, evidence); ok {
			trace.Verdict = VerdictContradicted
			trace.Reasons = append(trace.Reasons, Reason{Code: ReasonAntonymContradiction, Claim: claim, Evidence: evidence, Detail: detail})
			contradicted++
			traces = append(traces, trace)
			continue
		}

		if alignment >= AlignmentSupportThreshold &&
			len(coreTerms(claim)) >= 2 &&
			claimNumbersConsistent(claim, evidence) &&
			!nearVerbatimDivergence(claim, evidence, alignment) {
			trace.Verdict = VerdictSupported
			supported++
		}
		traces = append(traces, trace)
	}

	coverage := float64(covered) / float64(len(claims))
	out := VerifyOutput{Coverage: coverage, Claims: traces}
	switch {
	case contradicted > 0:
		out.Verdict, out.Confidence = VerdictContradicted, 1.0
	case covered == 0:
		out.Verdict, out.Confidence = VerdictUnknown, 0.0
	case supported == len(claims):
		out.Verdict, out.Confidence = VerdictSupported, averageSupportedAlignment(traces)
	case coverage < 1.0:
		out.Verdict, out.Confidence = VerdictInsufficient, coverage
	default:
		out.Verdict, out.Confidence = VerdictUnknown, averageAlignment(traces)
	}
	return out
}

var knownAbbrev = map[string]struct{}{
	"i.e": {}, "e.g": {}, "etc": {}, "vs": {}, "cf": {}, "al": {}, "no": {},
	"inc": {}, "ltd": {}, "dr": {}, "mr": {}, "ms": {}, "fig": {}, "eq": {},
	"approx": {}, "u.s": {}, "u.k": {}, "a.m": {}, "p.m": {},
}

// tokenBefore returns the lowercased run of letters/digits/'.' ending just
// before index i.
func tokenBefore(runes []rune, i int) string {
	j := i - 1
	for j >= 0 && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '.') {
		j--
	}
	return strings.ToLower(string(runes[j+1 : i]))
}

func splitClaims(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	runes := []rune(text)
	start := 0
	out := make([]string, 0, 4)
	flush := func(end int) {
		part := strings.TrimSpace(string(runes[start:end]))
		if part != "" {
			out = append(out, part)
		}
	}
	for i, r := range runes {
		boundary := r == '!' || r == '?' || r == ';' || r == '\n'
		if r == '.' {
			// Not a sentence break when the period sits inside a token
			// (3.14, crates.io) or right after a known abbreviation
			// (i.e., U.S., etc.).
			prevInWord := i > 0 && (unicode.IsLetter(runes[i-1]) || unicode.IsDigit(runes[i-1]))
			nextInWord := i+1 < len(runes) && (unicode.IsLetter(runes[i+1]) || unicode.IsDigit(runes[i+1]))
			_, isAbbrev := knownAbbrev[tokenBefore(runes, i)]
			boundary = !(prevInWord && nextInWord) && !isAbbrev
		}
		if boundary {
			flush(i)
			start = i + 1
		}
	}
	flush(len(runes))
	return out
}

// tieBreakMargin: a span scoring within this of the top span is treated as
// an equally plausible alignment for tie-breaking purposes.
const tieBreakMargin = 0.15

func bestEvidence(claim string, spans []string) (string, float64) {
	best, bestScore := "", 0.0
	scores := make([]float64, len(spans))
	for i, span := range spans {
		scores[i] = claimCoverage(claim, span)
		if scores[i] > bestScore {
			best, bestScore = span, scores[i]
		}
	}
	// R1 tie-break: among spans within tieBreakMargin of the top score and
	// above the candidate threshold, prefer one that carries a contradiction
	// signal against the claim (opposite negation parity, or an antonym of a
	// claim term). This fixes multi-claim answers where a claim's
	// lexically-closest span is not the one it actually conflicts with. It
	// can only route toward CONTRADICTED — never toward a false SUPPORTED.
	if bestScore >= AlignmentCandidateThreshold && !carriesContradictionSignal(claim, best) {
		for i, span := range spans {
			if span == best || scores[i] < AlignmentCandidateThreshold || scores[i] < bestScore-tieBreakMargin {
				continue
			}
			if carriesContradictionSignal(claim, span) {
				return span, scores[i]
			}
		}
	}
	return best, bestScore
}

// sharesCoreTerm reports whether the claim and span share at least one
// lemmatised core content term — the guard that keeps the below-threshold
// contradiction bypass from firing on unrelated sentences.
func sharesCoreTerm(claim, span string) bool {
	spanTerms := contentTerms(span)
	for term := range coreTerms(claim) {
		if _, ok := spanTerms[term]; ok {
			return true
		}
	}
	return false
}

func carriesContradictionSignal(claim, span string) bool {
	if span == "" {
		return false
	}
	if hasNegation(claim) != hasNegation(span) {
		return true
	}
	return anchoredConflictSignal(claim, span)
}

// sharesStrongCore reports that the claim and span share at least two
// lemmatised core terms AND those shared terms make up at least half of the
// span's own core — i.e. the span is largely about the same thing the claim
// is, not merely touching a common topic word.
func sharesStrongCore(claim, span string) bool {
	spanTerms := coreTerms(span)
	if len(spanTerms) == 0 {
		return false
	}
	shared := 0
	for term := range coreTerms(claim) {
		if _, ok := spanTerms[term]; ok {
			shared++
		}
	}
	return shared >= 2 && float64(shared)/float64(len(spanTerms)) >= 0.5
}

// anchoredConflictSignal is the subset of contradiction signals that are tied
// to a specific lexical item present on both sides — safe enough to act on
// even when whole-claim alignment is low.
func anchoredConflictSignal(claim, span string) bool {
	if span == "" {
		return false
	}
	if _, ok := antonymContradiction(claim, span); ok {
		return true
	}
	if _, ok := quantifierContradiction(claim, span); ok {
		return true
	}
	return false
}

// claimCoverage is the R1 alignment score: the fraction of the claim's
// *core* content terms (meta-prefix and filler/degree words removed, then
// lemmatised) that appear in the evidence span. Removing filler stops a
// long or hedged claim from being penalised for words the evidence has no
// reason to echo; lemmatising bridges "loss"/"losing", "requires"/"needs".
// The evidence side is NOT trimmed — only lemmatised — so alignment can
// only rise from a real synonym match, never from dropping evidence content.
func claimCoverage(claim, evidence string) float64 {
	claimTerms := coreTerms(claim)
	if len(claimTerms) == 0 {
		return 0
	}
	evidenceTerms := contentTerms(evidence)
	matched := 0
	for term := range claimTerms {
		if _, ok := evidenceTerms[term]; ok {
			matched++
		}
	}
	return float64(matched) / float64(len(claimTerms))
}

func contentTerms(text string) map[string]struct{} {
	return termsOf(text, false)
}

// coreTerms is contentTerms with the meta prefix stripped and filler/degree
// words dropped.
func coreTerms(text string) map[string]struct{} {
	return termsOf(stripMetaPrefix(text), true)
}

func termsOf(text string, core bool) map[string]struct{} {
	norm := normalize(text)
	terms := make(map[string]struct{})
	for _, raw := range strings.Fields(norm) {
		field := stem(raw)
		if len(field) < 2 {
			continue
		}
		if _, stop := stopwords[field]; stop {
			continue
		}
		if core {
			// The claim core also drops broad grammatical glue and
			// filler/degree words so a long or hedged claim is not
			// penalised on alignment for words the evidence needn't echo.
			if _, g := grammarWords[raw]; g {
				continue
			}
			if _, g := grammarWords[field]; g {
				continue
			}
			if _, filler := fillerTerms[field]; filler {
				continue
			}
		}
		terms[lemma(field)] = struct{}{}
	}
	return terms
}

func normalize(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	var b strings.Builder
	for _, r := range text {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '%', r == '.', r == ',', r == '-', unicode.IsSpace(r):
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func stem(s string) string {
	for _, suffix := range []string{"mente", "ing", "ed", "es", "s"} {
		if len(s) > len(suffix)+3 && strings.HasSuffix(s, suffix) {
			return strings.TrimSuffix(s, suffix)
		}
	}
	return s
}

func polarityContradiction(claim, evidence string) bool {
	return hasNegation(claim) != hasNegation(evidence)
}

func hasNegation(text string) bool {
	norm := " " + normalize(text) + " "
	for _, marker := range negativeMarkers {
		if strings.Contains(norm, marker) {
			return true
		}
	}
	return false
}

// numericContradiction is intentionally precision-biased. R0 only declares a
// numeric contradiction when both aligned claims express the same cardinality
// of explicit numeric values and those normalized values differ. Extra values
// in either span are treated as ambiguity rather than contradiction.
func numericContradiction(claim, evidence string) (string, bool) {
	claimNumbers := normalizedNumbers(claim)
	evidenceNumbers := normalizedNumbers(evidence)
	if len(claimNumbers) == 0 || len(claimNumbers) != len(evidenceNumbers) || equalStrings(claimNumbers, evidenceNumbers) {
		return "", false
	}
	return "claim=" + strings.Join(claimNumbers, ",") + "; evidence=" + strings.Join(evidenceNumbers, ","), true
}

func normalizedNumbers(text string) []string {
	matches := numberPattern.FindAllString(normalize(text), -1)
	out := make([]string, 0, len(matches))
	for _, raw := range matches {
		percent := strings.HasSuffix(raw, "%")
		raw = strings.TrimSuffix(raw, "%")
		raw = strings.ReplaceAll(raw, ",", ".")
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			continue
		}
		n := strconv.FormatFloat(value, 'f', -1, 64)
		if percent {
			n += "%"
		}
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func quantifierContradiction(claim, evidence string) (string, bool) {
	claimQ, evidenceQ := quantifierClass(claim), quantifierClass(evidence)
	if claimQ == quantifierNone || evidenceQ == quantifierNone || claimQ == evidenceQ {
		return "", false
	}
	contradiction := (claimQ == quantifierAll && evidenceQ == quantifierNever) ||
		(claimQ == quantifierNever && evidenceQ == quantifierAll) ||
		(claimQ == quantifierSome && evidenceQ == quantifierNever) ||
		(claimQ == quantifierNever && evidenceQ == quantifierSome) ||
		(claimQ == quantifierAlways && evidenceQ == quantifierNever) ||
		(claimQ == quantifierNever && evidenceQ == quantifierAlways)
	if !contradiction {
		return "", false
	}
	return "claim_class=" + strconv.Itoa(claimQ) + "; evidence_class=" + strconv.Itoa(evidenceQ), true
}

func quantifierClass(text string) int {
	norm := " " + normalize(text) + " "
	switch {
	case containsAnyPhrase(norm, "all", "every", "todos", "todas", "cada"):
		return quantifierAll
	case containsAnyPhrase(norm, "none", "ninguno", "ninguna"):
		return quantifierNever
	case containsAnyPhrase(norm, "never", "nunca", "rarely", "seldom", "raramente"):
		return quantifierNever
	case containsAnyPhrase(norm, "always", "usually", "typically", "frequently", "siempre", "usualmente"):
		return quantifierAlways
	case containsAnyPhrase(norm, "some", "several", "algunos", "algunas", "varios"):
		return quantifierSome
	default:
		return quantifierNone
	}
}

func containsAnyPhrase(norm string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(norm, " "+value+" ") {
			return true
		}
	}
	return false
}

func antonymContradiction(claim, evidence string) (string, bool) {
	claimNorm, evidenceNorm := " "+normalize(claim)+" ", " "+normalize(evidence)+" "
	for _, pair := range antonymPairs {
		leftClaim := strings.Contains(claimNorm, " "+pair[0]+" ")
		rightClaim := strings.Contains(claimNorm, " "+pair[1]+" ")
		leftEvidence := strings.Contains(evidenceNorm, " "+pair[0]+" ")
		rightEvidence := strings.Contains(evidenceNorm, " "+pair[1]+" ")
		if (leftClaim && rightEvidence) || (rightClaim && leftEvidence) {
			return pair[0] + "<>" + pair[1], true
		}
	}
	return "", false
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func averageSupportedAlignment(traces []ClaimTrace) float64 {
	total, count := 0.0, 0
	for _, trace := range traces {
		if trace.Verdict == VerdictSupported {
			total += trace.Alignment
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func averageAlignment(traces []ClaimTrace) float64 {
	if len(traces) == 0 {
		return 0
	}
	total := 0.0
	for _, trace := range traces {
		total += trace.Alignment
	}
	return total / float64(len(traces))
}
