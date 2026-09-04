package perceptenvelope

import (
	"fmt"
	"regexp"
	"strings"
)

// R1-D association scorer + failure taxonomy (protocol R1-D §10, §19).
// VALUE_CORRECT is an exact integer match; every other outcome is
// classified by whether the wrong answer corresponds to a number actually
// visible in the viewport.

// R1DScore is the frozen outcome for one R1-D record.
type R1DScore struct {
	ValueCorrect         bool   `json:"value_correct"`
	ContractSuccess      bool   `json:"contract_success"`
	Abstained            bool   `json:"abstained"`
	UnsupportedAssertion bool   `json:"unsupported_assertion"`
	FormatFailure        bool   `json:"format_failure"`
	FailureClass         string `json:"failure_class"`
	// SelectedKind is set when the wrong answer matches a visible number:
	// "DISTRACTOR" | "OTHER_VISIBLE" | "" (not a visible number).
	SelectedKind string `json:"selected_kind,omitempty"`
	GotValue     string `json:"got_value"`
}

var reR1DInt = regexp.MustCompile(`-?[0-9][0-9,]*`)

func normInt(s string) string { return strings.ReplaceAll(strings.TrimSpace(s), ",", "") }

// ScoreR1DAssoc scores one association output. distractorValues is nil for
// D0 and the K added distractors for D1.
func ScoreR1DAssoc(raw, goldValue string, visibleNumbers, distractorValues []string) R1DScore {
	sc := R1DScore{}
	rawTrim := strings.TrimSpace(raw)
	compact := strings.TrimSpace(strings.Trim(rawTrim, ".,:;"))

	switch {
	case rawTrim == "":
		sc.Abstained = true
		sc.FailureClass = "ABSTAIN"
		return sc
	case abstainWords.MatchString(rawTrim) && digitsOnly.ReplaceAllString(rawTrim, "") == "":
		sc.Abstained = true
		sc.FailureClass = "ABSTAIN"
		return sc
	}

	got, gotOK := parseFamilyValue(FamMultiDigit, rawTrim)
	sc.GotValue = got
	gold, _ := parseFamilyValue(FamMultiDigit, goldValue)

	// contract / commentary
	if len(strings.Fields(rawTrim)) > 3 && regexp.MustCompile(`(?i)[a-z]{4,}`).MatchString(rawTrim) {
		sc.UnsupportedAssertion = true
		sc.FailureClass = "COMMENTARY_CONTAMINATION"
	}
	sc.ContractSuccess = gotOK || numberLike.MatchString(compact)
	if !sc.ContractSuccess && sc.FailureClass == "" {
		sc.FormatFailure = true
		if !reR1DInt.MatchString(rawTrim) {
			sc.FailureClass = "LABEL_TEXT_ECHO"
		} else {
			sc.FailureClass = "OTHER"
		}
	}

	sc.ValueCorrect = gotOK && got != "" && got == gold
	if sc.ValueCorrect {
		sc.FailureClass = "TARGET_VALUE_CORRECT"
		return sc
	}

	// is the (numeric) output another visible number?
	if gotOK {
		for _, d := range distractorValues {
			if normInt(d) == got {
				sc.SelectedKind = "DISTRACTOR"
				if sc.FailureClass == "" {
					sc.FailureClass = "WRONG_VISIBLE_VALUE"
				}
				return sc
			}
		}
		for _, v := range visibleNumbers {
			if normInt(v) == got && normInt(v) != gold {
				sc.SelectedKind = "OTHER_VISIBLE"
				if sc.FailureClass == "" {
					sc.FailureClass = "WRONG_VISIBLE_VALUE"
				}
				return sc
			}
		}
	}

	if sc.FailureClass != "" && sc.FailureClass != "OTHER" {
		return sc
	}
	// generic digit-level classification against the gold
	rd := digitsOnly.ReplaceAllString(rawTrim, "")
	gd := digitsOnly.ReplaceAllString(goldValue, "")
	switch {
	case rd == "":
		sc.FailureClass = "LABEL_TEXT_ECHO"
	case len(rd) < len(gd) && (strings.HasPrefix(gd, rd) || strings.HasSuffix(gd, rd)):
		sc.FailureClass = "VALUE_TRUNCATION"
	case gotOK:
		sc.FailureClass = "HALLUCINATED_VALUE" // numeric, well-formed, but not on screen
	default:
		sc.FailureClass = "MORPHOLOGY_ERROR"
	}
	return sc
}

// R1DScorerSelfTest exercises the association scorer's canonical cases.
func R1DScorerSelfTest() []string {
	type tc struct {
		raw, gold         string
		visible, distract []string
		wantValue         bool
		wantClass         string
		wantSelected      string
	}
	cases := []tc{
		{"512", "512", []string{"512"}, nil, true, "TARGET_VALUE_CORRECT", ""},
		{"999", "512", []string{"512", "999", "77"}, []string{"999", "77"}, false, "WRONG_VISIBLE_VALUE", "DISTRACTOR"},
		{"640", "512", []string{"512", "999"}, []string{"999"}, false, "HALLUCINATED_VALUE", ""},
		{"", "512", []string{"512"}, nil, false, "ABSTAIN", ""},
		{"the length", "512", []string{"512"}, nil, false, "LABEL_TEXT_ECHO", ""},
		{"51", "512", []string{"512"}, nil, false, "VALUE_TRUNCATION", ""},
		{"The associated number is 512 here", "512", []string{"512"}, nil, true, "TARGET_VALUE_CORRECT", ""},
	}
	var probs []string
	for i, c := range cases {
		got := ScoreR1DAssoc(c.raw, c.gold, c.visible, c.distract)
		if got.ValueCorrect != c.wantValue {
			probs = append(probs, fmt.Sprintf("case %d (%q): value=%v want %v", i, c.raw, got.ValueCorrect, c.wantValue))
		}
		if got.FailureClass != c.wantClass {
			probs = append(probs, fmt.Sprintf("case %d (%q): class=%q want %q", i, c.raw, got.FailureClass, c.wantClass))
		}
		if got.SelectedKind != c.wantSelected {
			probs = append(probs, fmt.Sprintf("case %d (%q): selected=%q want %q", i, c.raw, got.SelectedKind, c.wantSelected))
		}
	}
	return probs
}
