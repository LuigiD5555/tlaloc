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
	"mediante": {}, "of": {}, "on": {}, "o": {}, "or": {}, "para": {}, "por": {}, "que": {},
	"se": {}, "su": {}, "sus": {}, "the": {}, "to": {}, "un": {}, "una": {}, "y": {},
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
}

func Verify(in VerifyInput) VerifyOutput {
	claims := splitClaims(in.ModelAnswer)
	if len(claims) == 0 {
		return VerifyOutput{Verdict: VerdictInsufficient, Claims: []ClaimTrace{}}
	}

	evidenceSpans := splitClaims(in.PageContent)
	traces := make([]ClaimTrace, 0, len(claims))
	covered, contradicted, supported := 0, 0, 0

	for _, claim := range claims {
		evidence, alignment := bestEvidence(claim, evidenceSpans)
		trace := ClaimTrace{Claim: claim, Evidence: evidence, Alignment: alignment, Verdict: VerdictUnknown}
		if alignment < AlignmentCandidateThreshold || evidence == "" {
			trace.Reasons = append(trace.Reasons, Reason{Code: ReasonLowAlignment, Claim: claim, Evidence: evidence})
			traces = append(traces, trace)
			continue
		}

		covered++
		trace.Reasons = append(trace.Reasons, Reason{Code: ReasonAligned, Claim: claim, Evidence: evidence})

		if detail, ok := numericContradiction(claim, evidence); ok {
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

		if alignment >= AlignmentSupportThreshold {
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
			prevDigit := i > 0 && unicode.IsDigit(runes[i-1])
			nextDigit := i+1 < len(runes) && unicode.IsDigit(runes[i+1])
			boundary = !(prevDigit && nextDigit)
		}
		if boundary {
			flush(i)
			start = i + 1
		}
	}
	flush(len(runes))
	return out
}

func bestEvidence(claim string, spans []string) (string, float64) {
	best, bestScore := "", 0.0
	for _, span := range spans {
		score := claimCoverage(claim, span)
		if score > bestScore {
			best, bestScore = span, score
		}
	}
	return best, bestScore
}

func claimCoverage(claim, evidence string) float64 {
	claimTerms := contentTerms(claim)
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
	norm := normalize(text)
	terms := make(map[string]struct{})
	for _, field := range strings.Fields(norm) {
		field = stem(field)
		if len(field) < 3 {
			continue
		}
		if _, stop := stopwords[field]; stop {
			continue
		}
		terms[field] = struct{}{}
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
	case containsAnyPhrase(norm, "all", "todos", "todas"):
		return quantifierAll
	case containsAnyPhrase(norm, "none", "ninguno", "ninguna"):
		return quantifierNever
	case containsAnyPhrase(norm, "some", "algunos", "algunas"):
		return quantifierSome
	case containsAnyPhrase(norm, "never", "nunca"):
		return quantifierNever
	case containsAnyPhrase(norm, "always", "siempre"):
		return quantifierAlways
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
