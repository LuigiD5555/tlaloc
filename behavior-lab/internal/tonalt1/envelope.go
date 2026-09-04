package tonalt1

import "regexp"

// Frozen R1 perceptual-envelope morphology rules for TONAL T1.
//
// T1 keeps Parrot perception strictly inside its EARNED frozen R1
// envelope. The only presentation families admitted for the T1 primary
// benchmark are those with sufficient frozen REAL-DOCUMENT evidence in
// R1 / R1-C:
//
//   - MULTI_DIGIT_INTEGER : R1-A/R1-B primary target morphology
//     (^[0-9]{2,4}$, born-digital, isolated-line render). R1-C
//     MULTI_DIGIT_INTEGER stratum (n=12 real bases) classified USABLE.
//   - DECIMAL             : R1-C DECIMAL stratum (n=12 real bases)
//     classified USABLE. Simple form only: 1-3 integer digits, a single
//     '.', 1-4 fraction digits.
//
// Explicitly EXCLUDED from the T1 primary benchmark (insufficient or
// negative frozen real-document support):
//
//   - SINGLE_DIGIT        : tested in R1-C but not affirmatively cleared
//     for deployment; excluded for conservatism.
//   - THOUSANDS_SEPARATOR : R1-C — model drops the comma. DO NOT DEPLOY.
//   - RANGE, SIGNED_NUMBER: R1-C — DO NOT DEPLOY.
//   - PERCENTAGE          : trailing '%' changes the read task; not a
//     bare number.
//   - SCIENTIFIC_NOTATION, EQUATION_EMBEDDED, COORDINATE_OR_TUPLE : n<=2
//     real bases; not characterised.
//   - TABLE_CELL          : region kind excluded (ambiguous line geometry
//     for a LocatedRegion-derived cue).
//
// Every rule below is machine-decidable. "Envelope-compatible" is never a
// subjective judgement.

// admissibleMultiDigit matches a bare 2-4 digit integer after edge
// punctuation has been stripped.
var admissibleMultiDigit = regexp.MustCompile(`^[0-9]{2,4}$`)

// admissibleDecimal matches a simple decimal: 1-3 integer digits, one
// dot, 1-4 fraction digits.
var admissibleDecimal = regexp.MustCompile(`^[0-9]{1,3}\.[0-9]{1,4}$`)

// rawMultiDigitToken is the admissible verbatim shape (no surrounding
// brackets/commas/slashes/colons/trailing period).
var rawMultiDigitToken = regexp.MustCompile(`^[0-9]{2,4}$`)

// rawDecimalToken is the admissible verbatim decimal shape.
var rawDecimalToken = regexp.MustCompile(`^[0-9]{1,3}\.[0-9]{1,4}$`)

// classifyMorphology returns the frozen morphology family for a token
// whose surrounding edge punctuation has already been stripped, plus
// whether the VERBATIM token shape is admissible. An empty family means
// the token is outside the T1 envelope.
func classifyMorphology(verbatim, stripped string) (MorphologyFamily, bool) {
	switch {
	case admissibleMultiDigit.MatchString(stripped):
		return MorphMultiDigitInteger, rawMultiDigitToken.MatchString(verbatim)
	case admissibleDecimal.MatchString(stripped):
		return MorphDecimal, rawDecimalToken.MatchString(verbatim)
	default:
		return "", false
	}
}

// isYearLike reports whether a 4-digit integer normalized token is in the
// [1500,2099] year/date band (frozen R1 rule).
func isYearLike(normalized string) bool {
	if len(normalized) != 4 {
		return false
	}
	value := 0
	for _, ch := range normalized {
		if ch < '0' || ch > '9' {
			return false
		}
		value = value*10 + int(ch-'0')
	}
	return value >= 1500 && value <= 2099
}

// EnvelopeRuleSummary is the frozen, human-readable rule list recorded in
// the selector manifest.
var EnvelopeRuleSummary = []string{
	"admissible morphology families: MULTI_DIGIT_INTEGER (^[0-9]{2,4}$), DECIMAL (^[0-9]{1,3}\\.[0-9]{1,4}$)",
	"MULTI_DIGIT_INTEGER support: R1-A/R1-B primary target morphology; R1-C MULTI_DIGIT_INTEGER real stratum (n=12) USABLE",
	"DECIMAL support: R1-C DECIMAL real stratum (n=12) USABLE; simple form only",
	"reject verbatim tokens with surrounding brackets/comma/slash/colon/trailing-period",
	"reject 4-digit integer tokens in [1500,2099] as year/date-like",
	"excluded families (insufficient or negative frozen real-document support): SINGLE_DIGIT, THOUSANDS_SEPARATOR, RANGE, SIGNED_NUMBER, PERCENTAGE, SCIENTIFIC_NOTATION, EQUATION_EMBEDDED, COORDINATE_OR_TUPLE, TABLE_CELL",
}
