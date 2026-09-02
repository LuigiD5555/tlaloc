package groundingautomaton

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var sentenceSplit = regexp.MustCompile(`[.!?;\n]+`)
var numberPattern = regexp.MustCompile(`[-+]?\d+(?:[.,]\d+)?%?`)

var stopwords = map[string]struct{}{
	"a": {}, "al": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "by": {},
	"con": {}, "de": {}, "del": {}, "el": {}, "en": {}, "es": {}, "for": {}, "from": {},
	"in": {}, "is": {}, "la": {}, "las": {}, "los": {}, "of": {}, "on": {}, "o": {}, "or": {},
	"para": {}, "por": {}, "que": {}, "the": {}, "to": {}, "un": {}, "una": {}, "y": {},
}

var negativeMarkers = []string{
	" no ", " not ", " never ", " nunca ", " ninguno ", " ninguna ", " without ", " sin ",
	"does not", "do not", "did not", "isn't", "isnt", "aren't", "arent", "wasn't", "wasnt",
}

var quantifierGroups = [][]string{
	{"all", "todos", "todas", "always", "siempre"},
	{"none", "ninguno", "ninguna", "never", "nunca"},
	{"some", "algunos", "algunas"},
	{"at least", "al menos"},
	{"at most", "como maximo", "como máximo"},
	{"more than", "mas de", "más de"},
	{"less than", "menos de"},
}

var antonymPairs = [][2]string{
	{"increase", "decrease"},
	{"increases", "decreases"},
	{"increased", "decreased"},
	{"aumenta", "disminuye"},
	{"aumentar", "disminuir"},
	{"before", "after"},
	{"antes", "despues"},
	{"antes", "después"},
	{"enable", "disable"},
	{"enabled", "disabled"},
	{"habilitado", "deshabilitado"},
	{"present", "absent"},
	{"presente", "ausente"},
	{"true", "false"},
	{"verdadero", "falso"},
	{"distributed", "centralized"},
	{"distribuido", "centralizado"},
}

func Verify(in VerifyInput) VerifyOutput {
	claims := splitClaims(in.ModelAnswer)
	if len(claims) == 0 {
		return VerifyOutput{Verdict: VerdictInsufficient, Claims: []ClaimTrace{}}
	}

	evidenceSpans := splitClaims(in.PageContent)
	traces := make([]ClaimTrace, 0, len(claims))
	covered := 0
	contradicted := 0
	supported := 0

	for _, claim := range claims {
		evidence, alignment := bestEvidence(claim, evidenceSpans)
		trace := ClaimTrace{Claim: claim, Evidence: evidence, Alignment: alignment, Verdict: VerdictUnknown}

		if alignment < 0.45 || evidence == "" {
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

		if alignment >= 0.70 {
			trace.Verdict = VerdictSupported
			supported++
		}
		traces = append(traces, trace)
	}

	coverage := float64(covered) / float64(len(claims))
	out := VerifyOutput{Coverage: coverage, Claims: traces}

	switch {
	case contradicted > 0:
		out.Verdict = VerdictContradicted
		out.Confidence = 1.0
	case covered == 0:
		out.Verdict = VerdictUnknown
		out.Confidence = 0.0
	case supported == len(claims):
		out.Verdict = VerdictSupported
		out.Confidence = averageSupportedAlignment(traces)
	case coverage < 1.0:
		out.Verdict = VerdictInsufficient
		out.Confidence = coverage
	default:
		out.Verdict = VerdictUnknown
		out.Confidence = averageAlignment(traces)
	}

	return out
}

func splitClaims(text string) []string {
	parts := sentenceSplit.Split(strings.TrimSpace(text), -1)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func bestEvidence(claim string, spans []string) (string, float64) {
	best := ""
	bestScore := 0.0
	for _, span := range spans {
		score := claimCoverage(claim, span)
		if score > bestScore {
			best = span
			bestScore = score
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
	fields := strings.Fields(norm)
	terms := make(map[string]struct{}, len(fields))
	for _, field := range fields {
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
	b.Grow(len(text))
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
	for _, suffix := range []string{"ing", "ed", "es", "s"} {
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

func numericContradiction(claim, evidence string) (string, bool) {
	claimNumbers := normalizedNumbers(claim)
	evidenceNumbers := normalizedNumbers(evidence)
	if len(claimNumbers) == 0 || len(evidenceNumbers) == 0 {
		return "", false
	}
	if equalStrings(claimNumbers, evidenceNumbers) {
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
	claimGroup := quantifierGroup(claim)
	evidenceGroup := quantifierGroup(evidence)
	if claimGroup == -1 || evidenceGroup == -1 || claimGroup == evidenceGroup {
		return "", false
	}
	return "claim_group=" + strconv.Itoa(claimGroup) + "; evidence_group=" + strconv.Itoa(evidenceGroup), true
}

func quantifierGroup(text string) int {
	norm := " " + normalize(text) + " "
	for i, group := range quantifierGroups {
		for _, term := range group {
			if strings.Contains(norm, " "+term+" ") {
				return i
			}
	}
	}
	return -1
}

func antonymContradiction(claim, evidence string) (string, bool) {
	claimNorm := " " + normalize(claim) + " "
	evidenceNorm := " " + normalize(evidence) + " "
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
	total := 0.0
	count := 0
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
