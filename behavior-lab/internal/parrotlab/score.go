package parrotlab

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Score is the verdict for one model answer. The taxonomy is deliberately
// wider than "correct/incorrect" (P-1 fix #5): a model that finds the right
// object but replies in prose is a contract failure, not a hallucination;
// a model that names an object outside the answer universe is an
// unsupported assertion.
type Score struct {
	// SemanticCorrect: the content is right, ignoring the requested form
	// (an accepted answer appears in the output).
	SemanticCorrect bool `json:"semantic_correct"`
	// FormatValid: the output is in the requested shape (a bare choice, a
	// number, compact JSON, or UNKNOWN) rather than a narrated sentence.
	FormatValid bool `json:"format_valid"`
	// ContractSuccess: did what was asked, in the form asked. This is what
	// aggregation counts as "correct".
	ContractSuccess bool `json:"contract_success"`
	// Abstained: the model answered UNKNOWN.
	Abstained bool `json:"abstained"`
	// UnsupportedAssertion: the model asserted a value outside the known
	// answer universe (only decidable when the case carries `choices`).
	UnsupportedAssertion bool   `json:"unsupported_assertion"`
	Parsed               string `json:"parsed"`
}

var numberPattern = regexp.MustCompile(`-?\d+(?:[.,]\d+)?`)

// ScoreAnswer grades raw model output against a case.
func ScoreAnswer(item Case, raw string) Score {
	normalised := normaliseAnswer(raw)
	abstained := isUnknown(normalised)
	score := Score{Abstained: abstained, Parsed: normalised}

	if item.TaskFamily == "abstain" {
		score.SemanticCorrect = abstained == item.Expected.Abstain
		score.FormatValid = abstained || isShortAnswer(normalised)
		score.ContractSuccess = score.SemanticCorrect
		return score
	}
	if abstained {
		// Wrongly abstained on an answerable item: not correct, but also
		// not an unsupported assertion and the form is legitimate.
		score.FormatValid = true
		return score
	}

	accepted := acceptedAnswers(item.Expected)

	switch item.TaskFamily {
	case "numeric":
		value, ok := firstNumber(raw)
		// a bare number, optionally with a unit — not a sentence about it.
		score.FormatValid = ok && len(strings.Fields(normalised)) <= 3
		if ok {
			score.Parsed = strconv.FormatFloat(value, 'f', -1, 64)
			if item.Expected.Number != nil {
				score.SemanticCorrect = math.Abs(value-*item.Expected.Number) <= item.Expected.Tolerance
			}
		}
	case "choice":
		score.SemanticCorrect = containsAny(normalised, accepted)
		inUniverse := mapsToChoice(normalised, item.Choices)
		score.FormatValid = inUniverse || (len(item.Choices) == 0 && isShortAnswer(normalised))
		if len(item.Choices) > 0 && !inUniverse {
			score.UnsupportedAssertion = true
		}
	case "entity":
		score.SemanticCorrect = containsAny(normalised, accepted) || tokenSubset(item.Expected.Value, normalised)
		score.FormatValid = isShortAnswer(normalised)
	default: // exact
		score.SemanticCorrect = matchesAny(normalised, accepted) || containsAny(normalised, accepted)
		score.FormatValid = isShortAnswer(normalised)
	}

	score.ContractSuccess = score.SemanticCorrect && score.FormatValid
	return score
}

func normaliseAnswer(raw string) string {
	text := strings.ToLower(strings.TrimSpace(raw))
	text = strings.Trim(text, " \t\r\n.\"'`*")
	for _, prefix := range []string{"answer:", "the answer is", "result:", "final answer:"} {
		text = strings.TrimSpace(strings.TrimPrefix(text, prefix))
	}
	return strings.TrimSpace(text)
}

func isUnknown(normalised string) bool {
	return normalised == "unknown" || normalised == "unknown." || normalised == "n/a"
}

// isShortAnswer treats a reply as form-valid when it is a terse token or
// phrase (or compact JSON) rather than a narrated explanation.
func isShortAnswer(normalised string) bool {
	if normalised == "" {
		return false
	}
	if strings.HasPrefix(normalised, "{") && strings.HasSuffix(normalised, "}") {
		return true
	}
	fields := strings.Fields(normalised)
	// A single whitespace-free token is a bare answer even when long (a
	// 32-char READ_SHORT_TEXT string is one token, not a narration).
	if len(fields) == 1 {
		return len(normalised) <= 64
	}
	return len(fields) <= 6 && len(normalised) <= 48
}

func acceptedAnswers(expected Expected) []string {
	seen := map[string]bool{}
	var out []string
	for _, candidate := range append([]string{expected.Value}, expected.Aliases...) {
		norm := normaliseAnswer(candidate)
		if norm == "" || seen[norm] {
			continue
		}
		seen[norm] = true
		out = append(out, norm)
	}
	return out
}

func matchesAny(normalised string, accepted []string) bool {
	for _, candidate := range accepted {
		if normalised == candidate {
			return true
		}
	}
	return false
}

func containsAny(normalised string, accepted []string) bool {
	fields := tokenSet(normalised)
	for _, candidate := range accepted {
		if normalised == candidate || strings.Contains(normalised, candidate) {
			return true
		}
		if len(strings.Fields(candidate)) == 1 && fields[candidate] {
			return true
		}
	}
	return false
}

// mapsToChoice reports whether the answer resolves to exactly one option of
// the given universe. An empty universe is not a constraint.
func mapsToChoice(normalised string, choices []string) bool {
	if len(choices) == 0 {
		return true
	}
	for _, choice := range choices {
		option := normaliseAnswer(choice)
		if normalised == option || tokenSet(normalised)[option] {
			return true
		}
	}
	return false
}

func tokenSet(text string) map[string]bool {
	fields := strings.Fields(text)
	set := make(map[string]bool, len(fields))
	for _, field := range fields {
		set[strings.Trim(field, ".,;:!?\"'()[]{}")] = true
	}
	return set
}

func tokenSubset(needle, haystack string) bool {
	needle = normaliseAnswer(needle)
	if needle == "" {
		return false
	}
	set := tokenSet(haystack)
	for _, token := range strings.Fields(needle) {
		if !set[token] {
			return false
		}
	}
	return true
}

func firstNumber(raw string) (float64, bool) {
	match := numberPattern.FindString(raw)
	if match == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(strings.Replace(match, ",", ".", 1), 64)
	if err != nil {
		return 0, false
	}
	return value, true
}
