package perceptenvelope

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// R1-C NUMERIC MORPHOLOGY — dual-endpoint family-aware scorer.
//
// Frozen before any R1-C model output exists (protocol R1-C sections 9-12).
// Two independent primary correctness bits per record:
//
//	VALUE_CORRECT        — the numeric / structured meaning is right
//	SURFACE_FORM_CORRECT — the visible numeric string was transcribed faithfully
//
// Example: gold "44,000", model "44000" -> VALUE_CORRECT=true,
// SURFACE_FORM_CORRECT=false, FailureClass THOUSANDS_SEPARATOR_DROPPED.
//
// Value comparison is exact (math/big Int / Rat); float equality is never
// used. Surface comparison allows only outer-whitespace trim and one
// documented Unicode normalization: every dash variant
// (U+002D U+2010 U+2011 U+2012 U+2013 U+2014 U+2212) folds to '-'.

// R1CFamily codes (protocol R1-C section 7). The string values reuse the
// morphology.go family constants so classifyMorph output feeds straight in.
const (
	FamSingleDigit = MorphSingleDigit
	FamMultiDigit  = MorphMultiDigitInt
	FamThousands   = MorphThousandsSep
	FamDecimal     = MorphDecimal
	FamPercentage  = MorphPercentage
	FamSigned      = MorphSigned
	FamRange       = MorphRange
	FamScientific  = MorphScientific
	FamCoordTuple  = MorphCoordTuple
	FamEquation    = MorphEquationEmbedded
	FamTableCell   = MorphTableCell
)

// R1CScore is the frozen dual-endpoint outcome for one R1-C record.
type R1CScore struct {
	ValueCorrect         bool   `json:"value_correct"`
	SurfaceFormCorrect   bool   `json:"surface_form_correct"`
	ContractSuccess      bool   `json:"contract_success"`
	Abstained            bool   `json:"abstained"`
	UnsupportedAssertion bool   `json:"unsupported_assertion"`
	FormatFailure        bool   `json:"format_failure"`
	FailureClass         string `json:"failure_class"`
	GoldValueCanonical   string `json:"gold_value_canonical"`
	GotValueCanonical    string `json:"got_value_canonical"`
}

var dashRunes = "-‐‑‒–—−"

// normSurface applies the only permitted surface normalizations.
func normSurface(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(dashRunes, r) {
			b.WriteByte('-')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// foldDashes maps every dash variant to '-' (used for value parsing only).
func foldDashes(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(dashRunes, r) {
			b.WriteByte('-')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var (
	reIntRun      = regexp.MustCompile(`[0-9]+`)
	reThousandsFF = regexp.MustCompile(`[0-9]{1,3}(?:,[0-9]{3})+`)
	reDecimalFF   = regexp.MustCompile(`[0-9]*\.[0-9]+`)
	rePercentFF   = regexp.MustCompile(`[0-9]+(?:\.[0-9]+)?\s*%`)
	reSignedFF    = regexp.MustCompile(`[+\-][0-9]+(?:\.[0-9]+)?`)
	reRangeFF     = regexp.MustCompile(`[0-9]+(?:\.[0-9]+)?\s*-\s*[0-9]+(?:\.[0-9]+)?`)
	reSciFF       = regexp.MustCompile(`[0-9]+(?:\.[0-9]+)?[eE][+\-]?[0-9]+`)
	reTupleFF     = regexp.MustCompile(`[\(\[]?\s*[0-9]+(?:\s*[,;]\s*[0-9]+)+\s*[\)\]]?`)
	reEquationOp  = regexp.MustCompile(`=\s*([0-9]+)`)
	reProse       = regexp.MustCompile(`(?i)[a-z]{4,}`)
)

// bigIntCanon parses a plain integer run into a canonical decimal string.
func bigIntCanon(s string) (string, bool) {
	m := reIntRun.FindString(s)
	if m == "" {
		return "", false
	}
	n := new(big.Int)
	if _, ok := n.SetString(m, 10); !ok {
		return "", false
	}
	return n.String(), true
}

// bigRatCanon parses a decimal / integer into an exact rational string.
func bigRatCanon(s string) (string, bool) {
	s = strings.TrimSpace(s)
	r := new(big.Rat)
	if _, ok := r.SetString(s); !ok {
		return "", false
	}
	return r.RatString(), true
}

// sciCanon parses "<mantissa>e<exp>" into an exact rational string.
func sciCanon(s string) (string, bool) {
	m := reSciFF.FindString(s)
	if m == "" {
		// exponent dropped: a bare decimal / int still yields a value
		if c, ok := bigRatCanon(strings.TrimSpace(s)); ok {
			return c, true
		}
		return "", false
	}
	lower := strings.ToLower(m)
	parts := strings.SplitN(lower, "e", 2)
	mant := new(big.Rat)
	if _, ok := mant.SetString(parts[0]); !ok {
		return "", false
	}
	expPart := strings.TrimPrefix(parts[1], "+")
	neg := strings.HasPrefix(expPart, "-")
	expPart = strings.TrimPrefix(expPart, "-")
	exp := new(big.Int)
	if _, ok := exp.SetString(expPart, 10); !ok {
		return "", false
	}
	pow := new(big.Int).Exp(big.NewInt(10), exp, nil)
	scale := new(big.Rat).SetInt(pow)
	out := new(big.Rat)
	if neg {
		out.Quo(mant, scale)
	} else {
		out.Mul(mant, scale)
	}
	return out.RatString(), true
}

// tupleCanon parses an ordered numeric tuple into "a,b,c".
func tupleCanon(s string) (string, bool) {
	m := reTupleFF.FindString(s)
	if m == "" {
		return "", false
	}
	m = strings.Trim(m, "()[] \t")
	fields := regexp.MustCompile(`\s*[,;]\s*`).Split(m, -1)
	var parts []string
	for _, f := range fields {
		c, ok := bigIntCanon(f)
		if !ok {
			return "", false
		}
		parts = append(parts, c)
	}
	if len(parts) < 2 {
		return "", false
	}
	return strings.Join(parts, ","), true
}

// rangeCanon parses "a-b" (dashes folded) into "a|b".
func rangeCanon(s string) (string, bool) {
	m := reRangeFF.FindString(foldDashes(s))
	if m == "" {
		return "", false
	}
	idx := strings.Index(m, "-")
	if idx <= 0 {
		return "", false
	}
	a, okA := bigRatCanon(strings.TrimSpace(m[:idx]))
	b, okB := bigRatCanon(strings.TrimSpace(m[idx+1:]))
	if !okA || !okB {
		return "", false
	}
	return a + "|" + b, true
}

// effectiveFamily resolves TABLE_CELL to its lexical sub-morphology.
func effectiveFamily(family, goldSurface string) string {
	if family != FamTableCell {
		return family
	}
	tok := strings.Trim(strings.TrimSpace(goldSurface), "()[]")
	sub := classifyMorph(tok, goldSurface, "text_line")
	if sub == "" || sub == FamTableCell {
		return FamMultiDigit
	}
	return sub
}

// parseFamilyValue returns the canonical exact value representation for a
// family, plus whether the string was structurally well-formed for that
// family (used for CONTRACT_SUCCESS).
func parseFamilyValue(family, s string) (canon string, wellFormed bool) {
	s = strings.TrimSpace(s)
	switch family {
	case FamSingleDigit, FamMultiDigit, FamEquation:
		if family == FamEquation {
			if mm := reEquationOp.FindStringSubmatch(s); mm != nil {
				return bigIntCanon(mm[1])
			}
		}
		return bigIntCanon(s)
	case FamThousands:
		return bigIntCanon(strings.ReplaceAll(s, ",", ""))
	case FamDecimal:
		if m := reDecimalFF.FindString(s); m != "" {
			return bigRatCanon(m)
		}
		return bigRatCanon(s)
	case FamPercentage:
		if m := rePercentFF.FindString(s); m != "" {
			return bigRatCanon(strings.TrimRight(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(m), "%")), " "))
		}
		return "", false // no percent sign -> not well-formed for this family
	case FamSigned:
		if m := reSignedFF.FindString(s); m != "" {
			return bigRatCanon(m)
		}
		return "", false // no explicit sign -> not well-formed
	case FamRange:
		return rangeCanon(s)
	case FamScientific:
		return sciCanon(s)
	case FamCoordTuple:
		return tupleCanon(s)
	}
	return bigIntCanon(s)
}

// ScoreR1C is the frozen dual-endpoint scorer.
func ScoreR1C(family, raw, goldSurface string) R1CScore {
	eff := effectiveFamily(family, goldSurface)
	sc := R1CScore{}

	rawTrim := strings.TrimSpace(raw)
	compact := strings.TrimSpace(strings.Trim(rawTrim, ".,:;"))

	// contract / abstain / commentary
	switch {
	case rawTrim == "":
		sc.Abstained = true
		sc.FailureClass = "ABSTAIN"
	case abstainWords.MatchString(rawTrim) && digitsOnly.ReplaceAllString(rawTrim, "") == "":
		sc.Abstained = true
		sc.FailureClass = "ABSTAIN"
	case len(strings.Fields(rawTrim)) > 3 && reProse.MatchString(rawTrim):
		sc.UnsupportedAssertion = true
		sc.FailureClass = "COMMENTARY_CONTAMINATION"
	}

	goldCanon, goldOK := parseFamilyValue(eff, goldSurface)
	if !goldOK {
		// fall back to a permissive integer/rational read of the gold so
		// the record is still scorable; flagged by GoldValueCanonical="".
		goldCanon, goldOK = bigRatCanon(goldSurface)
	}
	gotCanon, gotWF := parseFamilyValue(eff, rawTrim)
	sc.GoldValueCanonical = goldCanon
	sc.GotValueCanonical = gotCanon

	// contract: a well-formed family structure, or a bare number-like token
	if !sc.Abstained && !sc.UnsupportedAssertion {
		sc.ContractSuccess = gotWF || numberLike.MatchString(compact) || reIntRun.MatchString(rawTrim)
		if !sc.ContractSuccess {
			sc.FormatFailure = true
			if sc.FailureClass == "" {
				sc.FailureClass = "OTHER"
			}
		}
	}

	// SURFACE_FORM_CORRECT
	sc.SurfaceFormCorrect = normSurface(raw) == normSurface(goldSurface)

	// VALUE_CORRECT
	sc.ValueCorrect = goldOK && gotWF && goldCanon != "" && gotCanon == goldCanon

	if sc.ValueCorrect && sc.SurfaceFormCorrect {
		sc.FailureClass = ""
		return sc
	}
	if sc.FailureClass == "" || sc.FailureClass == "OTHER" {
		sc.FailureClass = classifyR1CFailure(eff, rawTrim, goldSurface, sc)
	}
	return sc
}

// classifyR1CFailure assigns the protocol section 12 failure class.
func classifyR1CFailure(family, raw, gold string, sc R1CScore) string {
	nraw := normSurface(raw)
	ngold := normSurface(gold)
	rawDigits := digitsOnly.ReplaceAllString(nraw, "")
	goldDigits := digitsOnly.ReplaceAllString(ngold, "")

	switch family {
	case FamThousands:
		if !strings.Contains(nraw, ",") && sc.ValueCorrect {
			return "THOUSANDS_SEPARATOR_DROPPED"
		}
		if strings.Count(nraw, ",") > strings.Count(ngold, ",") {
			return "THOUSANDS_SEPARATOR_INSERTED"
		}
	case FamDecimal:
		if !strings.Contains(nraw, ".") && strings.Contains(ngold, ".") {
			return "DECIMAL_POINT_DROPPED"
		}
		if strings.Contains(nraw, ".") && !strings.Contains(ngold, ".") {
			return "DECIMAL_POINT_INSERTED"
		}
		if rawDigits == goldDigits && nraw != ngold {
			return "DECIMAL_POINT_MOVED"
		}
	case FamPercentage:
		if !strings.Contains(nraw, "%") {
			return "PERCENT_SIGN_DROPPED"
		}
	case FamSigned:
		rSign := signOf(nraw)
		gSign := signOf(ngold)
		if rSign == "" && gSign != "" {
			return "SIGN_DROPPED"
		}
		if rSign != "" && gSign != "" && rSign != gSign {
			return "SIGN_FLIPPED"
		}
	case FamRange:
		if !strings.ContainsAny(nraw, "-") && strings.ContainsAny(ngold, "-") {
			return "RANGE_SEPARATOR_ERROR"
		}
		if !sc.ValueCorrect {
			return "RANGE_ENDPOINT_ERROR"
		}
	case FamScientific:
		if !strings.ContainsAny(nraw, "eE") && strings.ContainsAny(ngold, "eE") {
			return "EXPONENT_DROPPED"
		}
		if !sc.ValueCorrect {
			if em := reSciFF.FindString(nraw); em != "" {
				return "EXPONENT_VALUE_ERROR"
			}
		}
	case FamCoordTuple:
		rc, _ := tupleCanon(nraw)
		gc, _ := tupleCanon(ngold)
		rn := strings.Count(rc, ",")
		gn := strings.Count(gc, ",")
		if rn != gn {
			return "TUPLE_ARITY_ERROR"
		}
		if rc != gc && sortedTuple(rc) == sortedTuple(gc) {
			return "TUPLE_ORDER_ERROR"
		}
	}

	// generic digit-level taxonomy
	if sc.ValueCorrect && !sc.SurfaceFormCorrect {
		return "SURFACE_FORM_ONLY"
	}
	switch {
	case rawDigits == "":
		return "ABSTAIN"
	case rawDigits == goldDigits:
		return "SEPARATOR_OR_FORMAT_ONLY"
	case len(rawDigits) == len(goldDigits):
		return "DIGIT_SUBSTITUTION"
	case len(rawDigits) < len(goldDigits) && strings.HasSuffix(goldDigits, rawDigits):
		return "PREFIX_TRUNCATION"
	case len(rawDigits) < len(goldDigits) && strings.HasPrefix(goldDigits, rawDigits):
		return "SUFFIX_TRUNCATION"
	case len(rawDigits) < len(goldDigits):
		return "DIGIT_DELETION"
	case len(rawDigits) > len(goldDigits) && strings.Contains(rawDigits, goldDigits):
		return "DIGIT_INSERTION"
	default:
		return "WRONG_NUMBER"
	}
}

func signOf(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "-") {
		return "-"
	}
	if strings.HasPrefix(s, "+") {
		return "+"
	}
	return ""
}

func sortedTuple(csv string) string {
	if csv == "" {
		return ""
	}
	parts := strings.Split(csv, ",")
	// insertion sort by big.Int value
	for i := 1; i < len(parts); i++ {
		for j := i; j > 0; j-- {
			a, _ := new(big.Int).SetString(parts[j-1], 10)
			b, _ := new(big.Int).SetString(parts[j], 10)
			if a != nil && b != nil && a.Cmp(b) > 0 {
				parts[j-1], parts[j] = parts[j], parts[j-1]
			}
		}
	}
	return strings.Join(parts, ",")
}

// R1CScorerSelfTest runs the protocol section 9/10 canonical cases in code
// (used by doctor-r1c). Returns a list of human-readable failures; empty
// slice == pass.
func R1CScorerSelfTest() []string {
	type tc struct {
		family, raw, gold   string
		wantValue, wantSurf bool
		wantClass           string
	}
	cases := []tc{
		{FamThousands, "44000", "44,000", true, false, "THOUSANDS_SEPARATOR_DROPPED"},
		{FamThousands, "44,000", "44,000", true, true, ""},
		{FamPercentage, "98", "98%", false, false, "PERCENT_SIGN_DROPPED"},
		{FamPercentage, "98%", "98%", true, true, ""},
		{FamSigned, "42", "-42", false, false, "SIGN_DROPPED"},
		{FamSigned, "-42", "-42", true, true, ""},
		{FamSigned, "+17", "-17", false, false, "SIGN_FLIPPED"},
		{FamScientific, "1000000", "1e6", true, false, "EXPONENT_DROPPED"},
		{FamScientific, "1e6", "1e6", true, true, ""},
		{FamDecimal, "125", "12.5", false, false, "DECIMAL_POINT_DROPPED"},
		{FamDecimal, "12.5", "12.5", true, true, ""},
		{FamCoordTuple, "(256, 512)", "(512, 256)", false, false, "TUPLE_ORDER_ERROR"},
		{FamCoordTuple, "(512, 256)", "(512, 256)", true, true, ""},
		{FamRange, "3-7", "3–7", true, true, ""}, // dash variants fold (documented normalization)
		{FamRange, "3 to 7", "3–7", false, false, "RANGE_SEPARATOR_ERROR"},
		{FamRange, "3-9", "3–7", false, false, "RANGE_ENDPOINT_ERROR"},
		{FamMultiDigit, "128", "128", true, true, ""},
		{FamEquation, "128", "x = 128", true, false, ""},
	}
	var problems []string
	for i, c := range cases {
		got := ScoreR1C(c.family, c.raw, c.gold)
		if got.ValueCorrect != c.wantValue || got.SurfaceFormCorrect != c.wantSurf {
			problems = append(problems, fmt.Sprintf("case %d (%s %q->%q): value=%v surf=%v want value=%v surf=%v",
				i, c.family, c.raw, c.gold, got.ValueCorrect, got.SurfaceFormCorrect, c.wantValue, c.wantSurf))
		}
		if c.wantClass != "" && got.FailureClass != c.wantClass {
			problems = append(problems, fmt.Sprintf("case %d (%s %q->%q): class=%q want %q",
				i, c.family, c.raw, c.gold, got.FailureClass, c.wantClass))
		}
	}
	return problems
}
