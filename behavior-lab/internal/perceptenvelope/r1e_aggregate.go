package perceptenvelope

import (
	"fmt"
	"sort"

	"tlaloc.local/behaviorlab/internal/decompositionlab"
)

// R1-E visual-dependence aggregation + classification (protocol §7, §8).

// Classification thresholds (frozen).
const (
	r1eHighAcc      = 0.80 // "high" correct-image / shortcut accuracy
	r1eMaterialDrop = 0.30 // "materially" falls (correct_image - no_image)
	r1eMaterialGap  = 0.30 // wrong-image visible operand vs task gold gap
	r1eMinorGap     = 0.10 // any measurable influence
)

// R1EConditionRow is one intervention-condition aggregate for a capability.
type R1EConditionRow struct {
	Condition               string         `json:"condition"`
	N                       int            `json:"n"`
	TaskGoldCorrect         int            `json:"task_gold_correct"`
	TaskGoldAccuracy        float64        `json:"task_gold_accuracy"`
	CI95Low                 float64        `json:"ci95_low"`
	CI95High                float64        `json:"ci95_high"`
	ImageConsistent         int            `json:"image_consistent,omitempty"`
	ImageConsistentAccuracy float64        `json:"image_consistent_accuracy,omitempty"`
	ContractSuccess         int            `json:"contract_success"`
	Abstained               int            `json:"abstained"`
	UnsupportedAssertion    int            `json:"unsupported_assertion"`
	FormatFailure           int            `json:"format_failure"`
	MeanLatencyMS           float64        `json:"mean_latency_ms"`
	FailureClasses          map[string]int `json:"failure_classes"`
	Errors                  int            `json:"errors"`
}

// R1ECapabilityTable is the visual-dependence result for one capability.
type R1ECapabilityTable struct {
	Capability                       string             `json:"capability"`
	Opcode                           string             `json:"opcode"`
	Role                             string             `json:"role"`
	Bases                            int                `json:"bases"`
	Rows                             []R1EConditionRow  `json:"rows"`
	CorrectImageAccuracy             float64            `json:"correct_image_accuracy"`
	NoImageAccuracy                  float64            `json:"no_image_accuracy"`
	WrongImageTaskGoldAccuracy       float64            `json:"wrong_image_task_gold_accuracy"`
	WrongImageVisibleOperandAccuracy float64            `json:"wrong_image_visible_operand_accuracy"`
	McNemarCorrectVsNoImage          AdjacentTransition `json:"paired_mcnemar_correct_vs_no_image"`
	McNemarCorrectVsWrongImage       AdjacentTransition `json:"paired_mcnemar_correct_vs_wrong_image"`
	DigitLenMatchedPairs             int                `json:"digit_length_matched_pairs"`
	PairsTotal                       int                `json:"pairs_total"`
	Classification                   string             `json:"classification"`
	ClassificationBasis              string             `json:"classification_basis"`
}

// R1EVisualDependenceTable is the full frozen R1-E result.
type R1EVisualDependenceTable struct {
	Schema                     string               `json:"schema"`
	ExperimentID               string               `json:"experiment_id"`
	InterventionReuseOfR1DBases bool                 `json:"INTERVENTION_REUSE_OF_R1D_BASES"`
	Note                       string               `json:"note"`
	Capabilities               []R1ECapabilityTable `json:"capabilities"`
}

const r1eTableSchema = "tlaloc.parrot-perceptual-envelope-r1.r1e-visual-dependence-table.r1"

func r1eBuildCondRow(cond string, rs []R1ERecord) R1EConditionRow {
	row := R1EConditionRow{Condition: cond, FailureClasses: map[string]int{}}
	var lat []float64
	for _, r := range rs {
		if r.Error != "" {
			row.Errors++
			continue
		}
		row.N++
		if r.TaskGoldCorrect {
			row.TaskGoldCorrect++
		}
		if r.ImageConsistent {
			row.ImageConsistent++
		}
		if r.ContractSuccess {
			row.ContractSuccess++
		}
		if r.Abstained {
			row.Abstained++
		}
		if r.UnsupportedAssertion {
			row.UnsupportedAssertion++
		}
		if r.FormatFailure {
			row.FormatFailure++
		}
		if r.FailureClass != "" {
			row.FailureClasses[r.FailureClass]++
		}
		lat = append(lat, float64(r.LatencyMS))
	}
	row.TaskGoldAccuracy = ratio(row.TaskGoldCorrect, row.N)
	row.CI95Low, row.CI95High = decompositionlab.WilsonCI95(row.TaskGoldCorrect, row.N)
	if cond == "E1_WRONG_IMAGE" {
		row.ImageConsistentAccuracy = ratio(row.ImageConsistent, row.N)
	}
	if len(lat) > 0 {
		sum := 0.0
		for _, v := range lat {
			sum += v
		}
		row.MeanLatencyMS = sum / float64(len(lat))
	}
	return row
}

func pairR1E(from, to []R1ERecord, fromName, toName string) AdjacentTransition {
	fm := map[string]R1ERecord{}
	for _, r := range from {
		if r.Error == "" {
			fm[r.BaseID] = r
		}
	}
	tr := AdjacentTransition{From: fromName, To: toName, Metric: "task_gold"}
	var pairs []decompositionlab.PairedOutcome
	toSorted := append([]R1ERecord(nil), to...)
	sort.Slice(toSorted, func(i, j int) bool { return toSorted[i].BaseID < toSorted[j].BaseID })
	for _, tRec := range toSorted {
		fRec, ok := fm[tRec.BaseID]
		if !ok || tRec.Error != "" {
			continue
		}
		b, a := fRec.TaskGoldCorrect, tRec.TaskGoldCorrect
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

// AggregateR1E builds the visual-dependence table for every capability.
func AggregateR1E(records []R1ERecord, ds R1EDataset) R1EVisualDependenceTable {
	t := R1EVisualDependenceTable{
		Schema: r1eTableSchema, ExperimentID: ExperimentID,
		InterventionReuseOfR1DBases: true, Note: ds.Note,
	}
	matched := 0
	for _, p := range ds.WrongMap.Pairs {
		if p.DigitLenMatched {
			matched++
		}
	}
	byCap := map[string][]R1ERecord{}
	for _, r := range records {
		byCap[r.Capability] = append(byCap[r.Capability], r)
	}
	for _, capSpec := range ds.Capabilities {
		rs := byCap[capSpec.Capability]
		byCond := map[string][]R1ERecord{}
		baseSet := map[string]struct{}{}
		for _, r := range rs {
			byCond[r.Condition] = append(byCond[r.Condition], r)
			baseSet[r.BaseID] = struct{}{}
		}
		ct := R1ECapabilityTable{
			Capability: capSpec.Capability, Opcode: capSpec.Opcode, Role: capSpec.Role,
			Bases: len(baseSet), DigitLenMatchedPairs: matched, PairsTotal: len(ds.WrongMap.Pairs),
		}
		for _, cond := range R1EConditions {
			ct.Rows = append(ct.Rows, r1eBuildCondRow(cond, byCond[cond]))
		}
		for _, row := range ct.Rows {
			switch row.Condition {
			case "E0_NO_IMAGE":
				ct.NoImageAccuracy = row.TaskGoldAccuracy
			case "E1_WRONG_IMAGE":
				ct.WrongImageTaskGoldAccuracy = row.TaskGoldAccuracy
				ct.WrongImageVisibleOperandAccuracy = row.ImageConsistentAccuracy
			case "E2_CORRECT_IMAGE":
				ct.CorrectImageAccuracy = row.TaskGoldAccuracy
			}
		}
		ct.McNemarCorrectVsNoImage = pairR1E(byCond["E2_CORRECT_IMAGE"], byCond["E0_NO_IMAGE"], "E2_CORRECT_IMAGE", "E0_NO_IMAGE")
		ct.McNemarCorrectVsWrongImage = pairR1E(byCond["E2_CORRECT_IMAGE"], byCond["E1_WRONG_IMAGE"], "E2_CORRECT_IMAGE", "E1_WRONG_IMAGE")
		classifyR1E(&ct)
		t.Capabilities = append(t.Capabilities, ct)
	}
	return t
}

// classifyR1E applies protocol §8. Precedence: insufficient evidence, then
// visual dependence (the strongest positive claim), then shortcut
// domination, then mixed.
func classifyR1E(ct *R1ECapabilityTable) {
	ci := ct.CorrectImageAccuracy
	ni := ct.NoImageAccuracy
	wg := ct.WrongImageTaskGoldAccuracy
	wv := ct.WrongImageVisibleOperandAccuracy
	ct.ClassificationBasis = fmt.Sprintf(
		"correct_image=%.2f no_image=%.2f wrong_image_task_gold=%.2f wrong_image_visible_operand=%.2f; drop=%+.2f operand_gap=%+.2f",
		ci, ni, wg, wv, ci-ni, wv-wg)
	switch {
	case ct.Bases < 6 || ct.PairsTotal < 6:
		ct.Classification = "INSUFFICIENT_EVIDENCE"
	case ci >= r1eHighAcc && (ci-ni) >= r1eMaterialDrop && (wv-wg) >= r1eMaterialGap:
		ct.Classification = "VISUALLY_DEPENDENT"
	case ni >= r1eHighAcc || wg >= r1eHighAcc:
		ct.Classification = "SHORTCUT_DOMINATED"
	case (ci-ni) >= r1eMinorGap || (wv-wg) >= r1eMinorGap:
		ct.Classification = "MIXED_VISUAL_AND_PRIOR"
	default:
		ct.Classification = "INSUFFICIENT_EVIDENCE"
	}
}

// R1EReadAssocDisposition answers protocol §13.E: keep
// READ_ASSOCIATED_NUMBER USABLE_WITH_CONSTRAINTS or downgrade it.
func R1EReadAssocDisposition(t R1EVisualDependenceTable) (string, string) {
	var prim R1ECapabilityTable
	found := false
	for _, c := range t.Capabilities {
		if c.Capability == "READ_ASSOCIATED_NUMBER" {
			prim, found = c, true
		}
	}
	if !found {
		return "INSUFFICIENT_EVIDENCE", "no READ_ASSOCIATED_NUMBER rows in the table"
	}
	switch prim.Classification {
	case "VISUALLY_DEPENDENT":
		return "USABLE_WITH_CONSTRAINTS",
			"R1-D 22/22 is visually grounded: it collapses without the correct operand and tracks the wrong operand when one is substituted. Keep the R1-D constraints; add the explicit visual-grounding evidence."
	case "MIXED_VISUAL_AND_PRIOR":
		return "USABLE_WITH_CONSTRAINTS_PLUS_GROUNDING_GUARD",
			"The image measurably drives the answer but a prior/textual signal also leaks through. Keep USABLE_WITH_CONSTRAINTS only behind a grounding guard (e.g. a wrong-operand canary or an abstain-on-ambiguity check); do not rely on the bare accuracy."
	case "SHORTCUT_DOMINATED":
		return "DOWNGRADE_TO_NOT_VISUALLY_VERIFIED",
			"The R1-D headline reproduces without the correct image and/or with a wrong image: the 22/22 is not evidence of visual association. Downgrade until a shortcut-free stimulus set is built."
	default:
		return "INSUFFICIENT_EVIDENCE",
			"pairing / sample does not support a classification"
	}
}
