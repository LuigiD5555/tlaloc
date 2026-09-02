package groundingautomaton

import (
	"strconv"
	"strings"
)

// numCtx is a number mention plus the content word next to it (its unit or
// object) and its class. Two mentions are "about the same quantity" when
// they share a context word.
type numCtx struct {
	value   float64
	raw     string
	percent bool
	year    bool
	context map[string]struct{} // 1-2 content words adjacent to the number
}

func numberContexts(text string) []numCtx {
	norm := normalize(text)
	fields := strings.Fields(norm)
	var out []numCtx
	for i, field := range fields {
		raw := strings.TrimSuffix(field, "%")
		raw = strings.ReplaceAll(raw, ",", ".")
		value, err := strconv.ParseFloat(strings.Trim(raw, "-"), 64)
		if err != nil {
			continue
		}
		ctx := numCtx{
			value:   value,
			raw:     field,
			percent: strings.HasSuffix(field, "%"),
			year:    value >= 1800 && value <= 2100 && value == float64(int(value)),
			context: map[string]struct{}{},
		}
		for _, j := range []int{i - 2, i - 1, i + 1, i + 2} {
			if j < 0 || j >= len(fields) {
				continue
			}
			w := stem(fields[j])
			if len(w) < 2 {
				continue
			}
			if _, stop := stopwords[w]; stop {
				continue
			}
			if _, err := strconv.ParseFloat(strings.Trim(strings.TrimSuffix(w, "%"), "-"), 64); err == nil {
				continue
			}
			ctx.context[lemma(w)] = struct{}{}
		}
		out = append(out, ctx)
	}
	return out
}

func sameClass(a, b numCtx) bool { return a.percent == b.percent && a.year == b.year }

func sharedContext(a, b numCtx) bool {
	for w := range a.context {
		if _, ok := b.context[w]; ok {
			return true
		}
	}
	return false
}

const numberTolerance = 1e-6

// numericContradictionR1 fires when a claim number and an evidence number
// are the same class AND share an adjacent context word (same unit/object)
// AND their values differ. Context-sharing is what keeps "3 types" vs "5
// categories" from being a contradiction. Falls back to R0's
// equal-cardinality rule when no context word is available on either side.
func numericContradictionR1(claim, evidence string) (string, bool) {
	claimNumbers := numberContexts(claim)
	evidenceNumbers := numberContexts(evidence)
	if len(claimNumbers) == 0 || len(evidenceNumbers) == 0 {
		return "", false
	}
	for _, cn := range claimNumbers {
		for _, en := range evidenceNumbers {
			if !sameClass(cn, en) {
				continue
			}
			if !sharedContext(cn, en) {
				continue
			}
			if abs64(cn.value-en.value) > numberTolerance*max64(1, abs64(en.value)) {
				return "claim=" + cn.raw + "; evidence=" + en.raw + " (shared context)", true
			}
		}
	}
	return numericContradiction(claim, evidence)
}

// claimNumbersConsistent blocks a SUPPORTED verdict when a number in the
// claim has no equal value anywhere in the evidence span. A claim that
// states a figure the evidence never states cannot be "explicitly
// supported" by it — at best it is PARTIAL/UNKNOWN.
func claimNumbersConsistent(claim, evidence string) bool {
	claimNumbers := numberContexts(claim)
	if len(claimNumbers) == 0 {
		return true
	}
	evidenceNumbers := numberContexts(evidence)
	for _, cn := range claimNumbers {
		found := false
		for _, en := range evidenceNumbers {
			if sameClass(cn, en) && abs64(cn.value-en.value) <= numberTolerance*max64(1, abs64(en.value)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func abs64(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
