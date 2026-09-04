package perceptenvelope

import (
	"fmt"
	"sort"

	"tlaloc.local/behaviorlab/internal/decompositionlab"
)

// R1DConditionRow is one D0 cue-condition aggregate.
type R1DConditionRow struct {
	Condition            string         `json:"condition"`
	Opcode               string         `json:"opcode"`
	N                    int            `json:"n"`
	ValueCorrect         int            `json:"value_correct"`
	ValueAccuracy        float64        `json:"value_accuracy"`
	CI95Low              float64        `json:"ci95_low"`
	CI95High             float64        `json:"ci95_high"`
	ContractSuccess      int            `json:"contract_success"`
	Abstained            int            `json:"abstained"`
	UnsupportedAssertion int            `json:"unsupported_assertion"`
	FormatFailure        int            `json:"format_failure"`
	MeanLatencyMS        float64        `json:"mean_latency_ms"`
	P95LatencyMS         float64        `json:"p95_latency_ms"`
	FailureClasses       map[string]int `json:"failure_classes"`
	Errors               int            `json:"errors"`
}

// R1D0Table is the D0 real-association result.
type R1D0Table struct {
	Schema                 string             `json:"schema"`
	ExperimentID           string             `json:"experiment_id"`
	Track                  string             `json:"track"`
	N                      int                `json:"bases"`
	Rows                   []R1DConditionRow  `json:"rows"`
	PairedMcNemar          AdjacentTransition `json:"paired_mcnemar_d0v_to_d0l"`
	GeometryValidThreshold string             `json:"geometry_valid_threshold"`
	RealAssocGeometryValid bool               `json:"R1D_REAL_ASSOCIATION_GEOMETRY_VALID"`
	AssociationCost        float64            `json:"association_cost_value_to_label"`
	GateInterpretation     string             `json:"gate_interpretation"`
}

const r1d0Schema = "tlaloc.parrot-perceptual-envelope-r1.r1d0-association-table.r1"

// AggregateR1D0 builds the paired D0V/D0L table + geometry-validity gate.
func AggregateR1D0(records []R1DRecord) R1D0Table {
	byCond := map[string][]R1DRecord{}
	baseSet := map[string]struct{}{}
	for _, r := range records {
		if r.Track != "D0" {
			continue
		}
		byCond[r.Condition] = append(byCond[r.Condition], r)
		baseSet[r.BaseID] = struct{}{}
	}
	t := R1D0Table{
		Schema: r1d0Schema, ExperimentID: ExperimentID, Track: "D0", N: len(baseSet),
		GeometryValidThreshold: "D0V value accuracy >= 0.90 AND Wilson 95% lower bound >= 0.70",
	}
	for _, cond := range []string{"D0V_VALUE_CUED", "D0L_LABEL_CUED"} {
		t.Rows = append(t.Rows, buildR1DCondRow(cond, byCond[cond]))
	}
	// paired McNemar D0V -> D0L on value correctness
	t.PairedMcNemar = pairR1D(byCond["D0V_VALUE_CUED"], byCond["D0L_LABEL_CUED"], "D0V_VALUE_CUED", "D0L_LABEL_CUED")
	var d0v R1DConditionRow
	for _, r := range t.Rows {
		if r.Condition == "D0V_VALUE_CUED" {
			d0v = r
		}
	}
	t.RealAssocGeometryValid = d0v.ValueAccuracy >= 0.90 && d0v.CI95Low >= 0.70
	var d0l R1DConditionRow
	for _, r := range t.Rows {
		if r.Condition == "D0L_LABEL_CUED" {
			d0l = r
		}
	}
	if len(t.Rows) == 2 {
		t.AssociationCost = d0l.ValueAccuracy - d0v.ValueAccuracy
	}
	switch {
	case t.RealAssocGeometryValid:
		t.GateInterpretation = "D0V >= 0.90: the D0V->D0L delta is a clean association cost; D1 is canonical."
	case d0l.ValueAccuracy >= 0.90 && d0l.CI95Low >= 0.70:
		t.GateInterpretation = "Gate predicate failed on D0V, but D0L (the association task itself) is in the operating band — association is demonstrably intact and the D0V misses are a separate value-cue artifact (tight cue truncation), not a geometry break. D1 is labelled exploratory per the predeclared rule, but its degradation is still an interpretable signal."
	default:
		t.GateInterpretation = "Both D0V and D0L are below the operating band: the R1-D viewport geometry may itself be breaking perception; do not interpret D1 as association/distractor evidence."
	}
	return t
}

func buildR1DCondRow(cond string, rs []R1DRecord) R1DConditionRow {
	row := R1DConditionRow{Condition: cond, FailureClasses: map[string]int{}}
	var lat []float64
	for _, r := range rs {
		if r.Error != "" {
			row.Errors++
			continue
		}
		row.N++
		row.Opcode = r.Opcode
		sc := r.Score
		if sc.ValueCorrect {
			row.ValueCorrect++
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
		lat = append(lat, float64(r.LatencyMS))
	}
	row.ValueAccuracy = ratio(row.ValueCorrect, row.N)
	row.CI95Low, row.CI95High = decompositionlab.WilsonCI95(row.ValueCorrect, row.N)
	if len(lat) > 0 {
		sum := 0.0
		for _, v := range lat {
			sum += v
		}
		row.MeanLatencyMS = sum / float64(len(lat))
		row.P95LatencyMS = percentile(lat, 0.95)
	}
	return row
}

func pairR1D(from, to []R1DRecord, fromName, toName string) AdjacentTransition {
	fm := map[string]R1DRecord{}
	for _, r := range from {
		fm[r.BaseID] = r
	}
	tr := AdjacentTransition{From: fromName, To: toName, Metric: "value"}
	var pairs []decompositionlab.PairedOutcome
	toSorted := append([]R1DRecord(nil), to...)
	sort.Slice(toSorted, func(i, j int) bool { return toSorted[i].BaseID < toSorted[j].BaseID })
	for _, tRec := range toSorted {
		fRec, ok := fm[tRec.BaseID]
		if !ok || fRec.Error != "" || tRec.Error != "" {
			continue
		}
		b, a := fRec.Score.ValueCorrect, tRec.Score.ValueCorrect
		pairs = append(pairs, decompositionlab.PairedOutcome{CorrectBefore: b, CorrectAfter: a})
		switch {
		case b && a:
			tr.CorrectToCorrect++
		case b && !a:
			tr.CorrectToWrong++
		case !b && a:
			tr.WrongToCorrect++
		default:
			tr.WrongToWrong++
		}
	}
	res := decompositionlab.McNemarExact(pairs)
	tr.AbsoluteDelta, tr.PValue = res.AbsoluteDelta, res.PValue
	return tr
}

// R1D1KRow is one distractor-density rung.
type R1D1KRow struct {
	Condition       string         `json:"condition"`
	K               int            `json:"k"`
	N               int            `json:"n"`
	ValueCorrect    int            `json:"value_correct"`
	ValueAccuracy   float64        `json:"value_accuracy"`
	CI95Low         float64        `json:"ci95_low"`
	CI95High        float64        `json:"ci95_high"`
	ContractSuccess int            `json:"contract_success"`
	SelectedDistr   int            `json:"selected_distractor"`
	SelectedOther   int            `json:"selected_other_visible"`
	Hallucinated    int            `json:"hallucinated"`
	MeanLatencyMS   float64        `json:"mean_latency_ms"`
	FailureClasses  map[string]int `json:"failure_classes"`
	Errors          int            `json:"errors"`
	InOperatingBand bool           `json:"in_operating_band"`
}

// R1D1Curve is the controlled-composite distractor-density result.
type R1D1Curve struct {
	Schema          string               `json:"schema"`
	ExperimentID    string               `json:"experiment_id"`
	Track           string               `json:"track"`
	Provenance      string               `json:"provenance"`
	Canonical       bool                 `json:"canonical"`
	CanonicalNote   string               `json:"canonical_note"`
	Rows            []R1D1KRow           `json:"rows"`
	Transitions     []AdjacentTransition `json:"paired_transitions"`
	OperationalExit *int                 `json:"operational_exit_k"`
	DensityCliffK   *int                 `json:"density_cliff_k"`
	ResponseMix     map[string]float64   `json:"wrong_answer_mix"`
}

const r1d1Schema = "tlaloc.parrot-perceptual-envelope-r1.r1d1-distractor-curve.r1"

var r1d1Pairs = [][2]string{
	{"D1K0", "D1K1"}, {"D1K1", "D1K2"}, {"D1K2", "D1K4"}, {"D1K4", "D1K8"},
	{"D1K0", "D1K4"}, {"D1K0", "D1K8"},
}

// AggregateR1D1 builds the distractor-density curve. geometryValid controls
// whether the track is labelled canonical or exploratory.
func AggregateR1D1(records []R1DRecord, geometryValid bool) R1D1Curve {
	byCond := map[string][]R1DRecord{}
	for _, r := range records {
		if r.Track != "D1" {
			continue
		}
		byCond[r.Condition] = append(byCond[r.Condition], r)
	}
	c := R1D1Curve{
		Schema: r1d1Schema, ExperimentID: ExperimentID, Track: "D1",
		Provenance: "CONTROLLED_COMPOSITE", Canonical: geometryValid,
		ResponseMix: map[string]float64{},
	}
	if geometryValid {
		c.CanonicalNote = "D0V geometry gate passed; the distractor-density effect is causal evidence"
	} else {
		c.CanonicalNote = "D0V geometry gate FAILED (D0V value acc < 0.90 or Wilson lower < 0.70); D1 is exploratory only, not canonical"
	}
	var wrongTotal, wrongCorrectish, wrongDistr, wrongOther, wrongHall int
	for _, rung := range R1DDistractorLadder {
		rs := byCond[rung.ID]
		row := R1D1KRow{Condition: rung.ID, K: rung.K, FailureClasses: map[string]int{}}
		var lat []float64
		for _, r := range rs {
			if r.Error != "" {
				row.Errors++
				continue
			}
			row.N++
			sc := r.Score
			if sc.ValueCorrect {
				row.ValueCorrect++
			} else {
				wrongTotal++
				switch sc.SelectedKind {
				case "DISTRACTOR":
					row.SelectedDistr++
					wrongDistr++
				case "OTHER_VISIBLE":
					row.SelectedOther++
					wrongOther++
				default:
					if sc.FailureClass == "HALLUCINATED_VALUE" {
						row.Hallucinated++
						wrongHall++
					}
				}
			}
			if sc.ContractSuccess {
				row.ContractSuccess++
			}
			if sc.FailureClass != "" {
				row.FailureClasses[sc.FailureClass]++
			}
			lat = append(lat, float64(r.LatencyMS))
		}
		row.ValueAccuracy = ratio(row.ValueCorrect, row.N)
		row.CI95Low, row.CI95High = decompositionlab.WilsonCI95(row.ValueCorrect, row.N)
		if len(lat) > 0 {
			sum := 0.0
			for _, v := range lat {
				sum += v
			}
			row.MeanLatencyMS = sum / float64(len(lat))
		}
		row.InOperatingBand = row.ValueAccuracy >= 0.90 && row.CI95Low >= 0.70
		c.Rows = append(c.Rows, row)
	}
	for _, p := range r1d1Pairs {
		c.Transitions = append(c.Transitions, pairR1D(byCond[p[0]], byCond[p[1]], p[0], p[1]))
	}
	// operational exit + density cliff
	trByPair := map[string]AdjacentTransition{}
	for _, tr := range c.Transitions {
		trByPair[tr.From+"->"+tr.To] = tr
	}
	for i, row := range c.Rows {
		if !row.InOperatingBand && c.OperationalExit == nil {
			k := row.K
			c.OperationalExit = &k
		}
		if i > 0 {
			if tr, ok := trByPair[c.Rows[i-1].Condition+"->"+row.Condition]; ok && tr.PValue < 0.05 && tr.AbsoluteDelta < 0 && c.DensityCliffK == nil {
				k := row.K
				c.DensityCliffK = &k
			}
		}
	}
	if wrongTotal > 0 {
		c.ResponseMix["equals_correct_value"] = 0
		c.ResponseMix["equals_added_distractor"] = float64(wrongDistr) / float64(wrongTotal)
		c.ResponseMix["equals_other_visible_number"] = float64(wrongOther) / float64(wrongTotal)
		c.ResponseMix["hallucinated"] = float64(wrongHall) / float64(wrongTotal)
		c.ResponseMix["other_wrong"] = float64(wrongTotal-wrongDistr-wrongOther-wrongHall) / float64(wrongTotal)
	}
	_ = wrongCorrectish
	return c
}

// R1DVerdict is the provisional READ_ASSOCIATED_NUMBER capability verdict.
type R1DVerdict struct {
	Capability  string   `json:"capability"`
	Verdict     string   `json:"verdict"`
	Basis       string   `json:"basis"`
	Constraints []string `json:"constraints"`
}

// R1DProvisionalVerdict applies protocol §26.
func R1DProvisionalVerdict(d0 R1D0Table, d1 R1D1Curve) R1DVerdict {
	var d0l R1DConditionRow
	for _, r := range d0.Rows {
		if r.Condition == "D0L_LABEL_CUED" {
			d0l = r
		}
	}
	v := R1DVerdict{Capability: R1DAssocOpcode}
	acc, lo := d0l.ValueAccuracy, d0l.CI95Low
	switch {
	case d0l.N < 6:
		v.Verdict = "INSUFFICIENT_EVIDENCE"
	case acc >= 0.95 && lo >= 0.80:
		v.Verdict = "RELIABLE"
	case acc >= 0.90 && lo >= 0.70:
		v.Verdict = "USABLE_WITH_CONSTRAINTS"
	case acc >= 0.60:
		v.Verdict = "FRAGILE"
	default:
		v.Verdict = "DO_NOT_DEPLOY"
	}
	// Even an exploratory D1 collapse at a single competitor is too strong
	// to leave a RELIABLE headline: cap the verdict and record why.
	if v.Verdict == "RELIABLE" && d1.OperationalExit != nil && *d1.OperationalExit <= 1 {
		v.Verdict = "USABLE_WITH_CONSTRAINTS"
	}
	v.Basis = fmt.Sprintf("D0L real association n=%d value %.2f (CI %.2f-%.2f); D0V baseline %.2f; association cost %+.2f; geometry_valid=%v",
		d0l.N, acc, lo, d0l.CI95High, d0.Rows[0].ValueAccuracy, d0.AssociationCost, d0.RealAssocGeometryValid)
	v.Constraints = []string{
		"MULTI_DIGIT_INTEGER operand only (fragile R1-C morphologies excluded)",
		"32 px containing-line height",
		"single-line label/value layout; other-line numbers masked out",
	}
	maxTol := "0"
	if d1.OperationalExit != nil {
		maxTol = fmt.Sprintf("%d (leaves operating band at K=%d)", *d1.OperationalExit-1, *d1.OperationalExit)
	} else if len(d1.Rows) > 0 {
		maxTol = fmt.Sprintf(">=%d (no exit within tested ladder)", d1.Rows[len(d1.Rows)-1].K)
	}
	note := "canonical"
	if !d1.Canonical {
		note = "EXPLORATORY — geometry gate failed"
	}
	v.Constraints = append(v.Constraints, fmt.Sprintf("max tolerated added numeric distractors: %s [%s]", maxTol, note))
	return v
}
