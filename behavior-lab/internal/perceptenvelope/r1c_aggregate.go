package perceptenvelope

import (
	"fmt"
	"sort"

	"tlaloc.local/behaviorlab/internal/decompositionlab"
)

// R1CEndpoint is one correctness endpoint's aggregate.
type R1CEndpoint struct {
	Count    int     `json:"count"`
	N        int     `json:"n"`
	Accuracy float64 `json:"accuracy"`
	CI95Low  float64 `json:"ci95_low"`
	CI95High float64 `json:"ci95_high"`
}

func endpoint(count, n int) R1CEndpoint {
	e := R1CEndpoint{Count: count, N: n, Accuracy: ratio(count, n)}
	e.CI95Low, e.CI95High = decompositionlab.WilsonCI95(count, n)
	return e
}

// R1CRow is one (family, provenance stratum) row of the morphology table.
type R1CRow struct {
	Family               string                 `json:"family"`
	Stratum              string                 `json:"stratum"`
	Provenance           string                 `json:"provenance"`
	N                    int                    `json:"n"`
	Value                R1CEndpoint            `json:"value_correct"`
	Surface              R1CEndpoint            `json:"surface_form_correct"`
	ContractSuccess      int                    `json:"contract_success"`
	Abstained            int                    `json:"abstained"`
	UnsupportedAssertion int                    `json:"unsupported_assertion"`
	FormatFailure        int                    `json:"format_failure"`
	MeanLatencyMS        float64                `json:"mean_latency_ms"`
	P95LatencyMS         float64                `json:"p95_latency_ms"`
	FailureClasses       map[string]int         `json:"failure_classes"`
	ValueAndSurface      int                    `json:"value_and_surface"`
	ValueNotSurface      int                    `json:"value_not_surface"`
	NotValue             int                    `json:"not_value"`
	DigitLenSubgroups    map[string]R1CEndpoint `json:"digit_length_subgroups,omitempty"`
	Errors               int                    `json:"errors"`
}

// R1CFamilyVerdict is the provisional per-family capability verdict.
type R1CFamilyVerdict struct {
	Family  string `json:"family"`
	Verdict string `json:"verdict"`
	Basis   string `json:"basis"`
}

// R1CMorphologyTable is the full R1-C result.
type R1CMorphologyTable struct {
	Schema       string             `json:"schema"`
	ExperimentID string             `json:"experiment_id"`
	Stage        string             `json:"stage"`
	LineHeightPx float64            `json:"line_height_px"`
	ContextLevel string             `json:"context_level"`
	Records      int                `json:"records"`
	Errors       int                `json:"errors"`
	Rows         []R1CRow           `json:"rows"`
	Verdicts     []R1CFamilyVerdict `json:"provisional_verdicts"`
	Answers      map[string]string  `json:"primary_questions"`
}

const r1cTableSchema = "tlaloc.parrot-perceptual-envelope-r1.r1c-morphology-table.r1"

var r1cRowOrder = append(append([]string{}, r1cLexicalFamilies...), r1cLayoutFamilies...)

// AggregateR1C builds the per-(family, stratum) morphology table. REAL and
// SYNTHETIC accuracy are never pooled.
func AggregateR1C(records []R1CRecord) R1CMorphologyTable {
	tbl := R1CMorphologyTable{
		Schema: r1cTableSchema, ExperimentID: ExperimentID, Stage: "R1-C",
		LineHeightPx: R1CLineHeightPx, ContextLevel: R1CContextLevel,
		Records: len(records), Answers: map[string]string{},
	}
	type key struct{ fam, prov string }
	groups := map[key][]R1CRecord{}
	for _, r := range records {
		if r.Error != "" {
			tbl.Errors++
		}
		provTag := "REAL_DOCUMENT"
		if r.Provenance == ProvSynthetic {
			provTag = "SYNTHETIC_REALISTIC"
		}
		groups[key{r.Family, provTag}] = append(groups[key{r.Family, provTag}], r)
	}

	realRows := map[string]R1CRow{}
	for _, fam := range r1cRowOrder {
		for _, provTag := range []string{"REAL_DOCUMENT", "SYNTHETIC_REALISTIC"} {
			rs := groups[key{fam, provTag}]
			if len(rs) == 0 {
				continue
			}
			row := buildR1CRow(fam, provTag, rs)
			tbl.Rows = append(tbl.Rows, row)
			if provTag == "REAL_DOCUMENT" {
				realRows[fam] = row
			}
		}
	}

	for _, fam := range r1cRowOrder {
		tbl.Verdicts = append(tbl.Verdicts, verdictFor(fam, realRows, tbl.Rows))
	}
	tbl.Answers = r1cAnswers(realRows, tbl.Rows)
	return tbl
}

func buildR1CRow(family, provTag string, rs []R1CRecord) R1CRow {
	row := R1CRow{Family: family, Provenance: provTag, FailureClasses: map[string]int{}}
	if len(rs) > 0 {
		row.Stratum = rs[0].Stratum
	}
	var vC, sC int
	var lat []float64
	smallN := false
	digitBuckets := map[string][2]int{} // subgroup -> [valueCorrect, n]
	for _, r := range rs {
		if r.Error != "" {
			row.Errors++
			continue
		}
		row.N++
		sc := r.Score
		if sc.ValueCorrect {
			vC++
		}
		if sc.SurfaceFormCorrect {
			sC++
		}
		if sc.ContractSuccess {
			row.ContractSuccess++
		}
		if sc.Abstained {
			row.Abstained++
		}
		if sc.UnsupportedAssertion {
			row.UnsupportedAssertion++
		}
		if sc.FormatFailure {
			row.FormatFailure++
		}
		if sc.FailureClass != "" {
			row.FailureClasses[sc.FailureClass]++
		}
		switch {
		case sc.ValueCorrect && sc.SurfaceFormCorrect:
			row.ValueAndSurface++
		case sc.ValueCorrect && !sc.SurfaceFormCorrect:
			row.ValueNotSurface++
		default:
			row.NotValue++
		}
		lat = append(lat, float64(r.LatencyMS))
		if r.Provenance == ProvRealSmallN {
			smallN = true
		}
		if family == FamMultiDigit && r.DigitLenSubgroup != "" {
			b := digitBuckets[r.DigitLenSubgroup]
			if sc.ValueCorrect {
				b[0]++
			}
			b[1]++
			digitBuckets[r.DigitLenSubgroup] = b
		}
	}
	row.Value = endpoint(vC, row.N)
	row.Surface = endpoint(sC, row.N)
	if len(lat) > 0 {
		sum := 0.0
		for _, v := range lat {
			sum += v
		}
		row.MeanLatencyMS = sum / float64(len(lat))
		row.P95LatencyMS = percentile(lat, 0.95)
	}
	if smallN {
		row.Provenance = "REAL_DOCUMENT_SMALL_N"
	}
	if len(digitBuckets) > 0 {
		row.DigitLenSubgroups = map[string]R1CEndpoint{}
		for sg, b := range digitBuckets {
			row.DigitLenSubgroups[sg] = endpoint(b[0], b[1])
		}
	}
	return row
}

// verdictFor applies the protocol section 21 provisional verdict rules.
func verdictFor(family string, realRows map[string]R1CRow, allRows []R1CRow) R1CFamilyVerdict {
	rr, hasReal := realRows[family]
	if hasReal && rr.N >= 6 {
		v := rr.Value
		switch {
		case v.Accuracy >= 0.95 && v.CI95Low >= 0.80 && rr.Surface.Accuracy >= 0.90:
			return R1CFamilyVerdict{family, "RELIABLE", fmt.Sprintf("REAL n=%d value %.2f (CI %.2f-%.2f) surface %.2f", rr.N, v.Accuracy, v.CI95Low, v.CI95High, rr.Surface.Accuracy)}
		case v.Accuracy >= 0.90 && v.CI95Low >= 0.70:
			return R1CFamilyVerdict{family, "USABLE_WITH_CONSTRAINTS", fmt.Sprintf("REAL n=%d value %.2f (CI %.2f-%.2f) surface %.2f", rr.N, v.Accuracy, v.CI95Low, v.CI95High, rr.Surface.Accuracy)}
		case v.Accuracy >= 0.60:
			return R1CFamilyVerdict{family, "FRAGILE", fmt.Sprintf("REAL n=%d value %.2f (CI %.2f-%.2f)", rr.N, v.Accuracy, v.CI95Low, v.CI95High)}
		default:
			return R1CFamilyVerdict{family, "DO_NOT_DEPLOY", fmt.Sprintf("REAL n=%d value %.2f (CI %.2f-%.2f)", rr.N, v.Accuracy, v.CI95Low, v.CI95High)}
		}
	}
	// No sufficient real evidence. Per protocol §21 synthetic evidence can
	// refine mechanism understanding but cannot set a real-document
	// verdict in either direction (the glyph-bank synthetic render is a
	// weaker stimulus than a real page crop), so the verdict stays
	// INSUFFICIENT_REAL_EVIDENCE and the synthetic result is reported as
	// a mechanism note only.
	var realNote string
	if rr, ok := realRows[family]; ok {
		realNote = fmt.Sprintf("real n=%d (descriptive) value %.2f; ", rr.N, rr.Value.Accuracy)
	}
	for _, row := range allRows {
		if row.Family == family && row.Provenance == "SYNTHETIC_REALISTIC" {
			v := row.Value
			return R1CFamilyVerdict{family, "INSUFFICIENT_REAL_EVIDENCE",
				fmt.Sprintf("%sSYNTHETIC_REALISTIC mechanism probe n=%d value %.2f (CI %.2f-%.2f), surface %.2f — glyph-bank stimulus, not real-PDF; not a deployable verdict",
					realNote, row.N, v.Accuracy, v.CI95Low, v.CI95High, row.Surface.Accuracy)}
		}
	}
	return R1CFamilyVerdict{family, "INSUFFICIENT_REAL_EVIDENCE", realNote + "no synthetic stratum"}
}

func r1cAnswers(realRows map[string]R1CRow, allRows []R1CRow) map[string]string {
	get := func(fam string) (R1CRow, bool) { r, ok := realRows[fam]; return r, ok }
	synth := func(fam string) (R1CRow, bool) {
		for _, r := range allRows {
			if r.Family == fam && r.Provenance == "SYNTHETIC_REALISTIC" {
				return r, true
			}
		}
		return R1CRow{}, false
	}
	fmtRow := func(r R1CRow, ok bool) string {
		if !ok {
			return "no data"
		}
		return fmt.Sprintf("value %.2f (CI %.2f-%.2f), surface %.2f, n=%d", r.Value.Accuracy, r.Value.CI95Low, r.Value.CI95High, r.Surface.Accuracy, r.N)
	}
	ans := map[string]string{}
	si, siOK := get(FamSingleDigit)
	mi, miOK := get(FamMultiDigit)
	ans["A_plain_integers_reliable_at_32px"] = "SINGLE_DIGIT " + fmtRow(si, siOK) + " · MULTI_DIGIT_INTEGER " + fmtRow(mi, miOK)
	if miOK && mi.DigitLenSubgroups != nil {
		b := mi.DigitLenSubgroups
		ans["B_digit_length_still_matters"] = fmt.Sprintf("2-digit %.2f (n=%d), 3-digit %.2f (n=%d), 4-digit %.2f (n=%d)",
			b["2"].Accuracy, b["2"].N, b["3"].Accuracy, b["3"].N, b["4"].Accuracy, b["4"].N)
	}
	tr, trOK := get(FamThousands)
	ans["C_thousands_separators_fragile"] = fmtRow(tr, trOK) + fmt.Sprintf(" · value-retained-surface-lost = %d/%d", rowVNS(tr, trOK), rowN(tr, trOK))
	dr, drOK := get(FamDecimal)
	ans["D_decimal_points_fragile"] = fmtRow(dr, drOK) + fmt.Sprintf(" · value-retained-surface-lost = %d/%d", rowVNS(dr, drOK), rowN(dr, drOK))
	sr, srOK := get(FamSigned)
	ans["E_signs_preserved"] = fmtRow(sr, srOK)
	pr, prOK := get(FamPercentage)
	ans["F_percent_signs_preserved"] = fmtRow(pr, prOK)
	rr, rrOK := get(FamRange)
	ans["G_ranges_read_as_structured"] = fmtRow(rr, rrOK)
	sc, scOK := synth(FamScientific)
	ans["H_scientific_notation_reliable"] = "SYNTHETIC " + fmtRow(sc, scOK)
	tp, tpOK := synth(FamCoordTuple)
	ans["I_tuples_order_arity_intact"] = "SYNTHETIC " + fmtRow(tp, tpOK)
	var vns, tot int
	for _, r := range allRows {
		vns += r.ValueNotSurface
		tot += r.N
	}
	ans["J_value_retained_surface_lost"] = fmt.Sprintf("%d / %d records across all families", vns, tot)
	var fragile []string
	for _, fam := range r1cRowOrder {
		if r, ok := realRows[fam]; ok && r.N >= 6 && r.Value.Accuracy < 0.90 {
			fragile = append(fragile, fmt.Sprintf("%s(%.2f)", fam, r.Value.Accuracy))
		}
	}
	if len(fragile) == 0 {
		ans["K_families_failing_after_context_and_scale_controlled"] = "none among families with sufficient real evidence"
	} else {
		ans["K_families_failing_after_context_and_scale_controlled"] = joinStr(fragile)
	}
	return ans
}

func rowVNS(r R1CRow, ok bool) int {
	if !ok {
		return 0
	}
	return r.ValueNotSurface
}
func rowN(r R1CRow, ok bool) int {
	if !ok {
		return 0
	}
	return r.N
}
func joinStr(xs []string) string {
	sort.Strings(xs)
	out := ""
	for i, x := range xs {
		if i > 0 {
			out += ", "
		}
		out += x
	}
	return out
}
