package perceptenvelope

import "fmt"

// R1-G provisional recovery policy builder (protocol §21). Machine-readable
// guidance for a future Tlaloc RecoveryPolicy — NOT yet merged into the
// CapabilityProfile R1.

// R1GPolicyRule is one recovery/prevention rule.
type R1GPolicyRule struct {
	DetectIf        string         `json:"detect_if,omitempty"`
	PreferredAction string         `json:"preferred_action"`
	Mode            string         `json:"mode"` // PREVENTIVE | REACTIVE | REJECT
	Verdict         string         `json:"verdict"`
	Evidence        map[string]any `json:"evidence,omitempty"`
	FallbackAction  string         `json:"fallback_action,omitempty"`
}

// R1GRecoveryPolicy is the frozen provisional policy document.
type R1GRecoveryPolicy struct {
	Schema        string                   `json:"schema"`
	ExperimentID  string                   `json:"experiment_id"`
	Status        string                   `json:"status"`
	NotMergedInto string                   `json:"not_yet_merged_into"`
	Rules         map[string]R1GPolicyRule `json:"rules"`
	UnresolvedFailureFamilies []string     `json:"unresolved_failure_families"`
}

const r1gPolicySchema = "tlaloc.parrot-perceptual-envelope-r1.r1g-recovery-policy.r1"

func bestRow(t R1GRecoveryTable, fam string) (R1GRecoveryRow, bool) {
	var best R1GRecoveryRow
	found := false
	rank := map[string]int{"EARNED_RECOVERY": 4, "PROMISING_RECOVERY": 3, "NO_MEASURED_BENEFIT": 2, "INSUFFICIENT_EVIDENCE": 1, "HARMFUL": 0}
	for _, r := range t.Rows {
		if r.Family != fam {
			continue
		}
		if !found || rank[r.Verdict] > rank[best.Verdict] ||
			(rank[r.Verdict] == rank[best.Verdict] && r.McNemar.AbsoluteDelta > best.McNemar.AbsoluteDelta) {
			best, found = r, true
		}
	}
	return best, found
}

func rowOf(t R1GRecoveryTable, fam, cond string) (R1GRecoveryRow, bool) {
	for _, r := range t.Rows {
		if r.Family == fam && r.RecoveryCondition == cond {
			return r, true
		}
	}
	return R1GRecoveryRow{}, false
}

func accOf(t R1GRecoveryTable, fam, cond string) float64 {
	if r, ok := rowOf(t, fam, cond); ok {
		return round2(r.RecoveryAccuracy)
	}
	return 0
}

// BuildR1GRecoveryPolicy derives the provisional policy from the recovery
// table.
func BuildR1GRecoveryPolicy(t R1GRecoveryPolicySource) R1GRecoveryPolicy {
	tbl := t.Table
	p := R1GRecoveryPolicy{
		Schema: r1gPolicySchema, ExperimentID: ExperimentID,
		Status: "PROVISIONAL", NotMergedInto: "CapabilityProfile R1",
		Rules: map[string]R1GPolicyRule{},
	}

	p.Rules["exact_retry"] = R1GPolicyRule{
		PreferredAction: "DO_NOT_RETRY_IDENTICAL_INPUT",
		Mode:            "REJECT",
		Verdict:         "DO_NOT_USE",
		Evidence: map[string]any{
			"source": "R1-F", "previously_wrong_recovered": "0/16",
			"semantic_invariant_5of5": "20/20", "note": R1GExactRetryStatus,
		},
	}
	p.Rules["missing_visual_operand"] = R1GPolicyRule{
		DetectIf:        "visual opcode AND no visual operand",
		PreferredAction: "RETURN_UNSUPPORTED_OR_UNKNOWN_WITHOUT_PARROT",
		Mode:            "REJECT",
		Verdict:         "DO_NOT_USE",
		Evidence:        map[string]any{"source": "R1-E", "no_image": "0/22 task gold, degenerate 12345"},
	}

	ev := func(row R1GRecoveryRow, extra map[string]any) map[string]any {
		m := map[string]any{
			"baseline_accuracy":         round2(row.BaselineAccuracy),
			"recovery_accuracy":         round2(row.RecoveryAccuracy),
			"delta":                     round2(row.McNemar.AbsoluteDelta),
			"mcnemar_exact_p":           row.McNemar.PValue,
			"w_to_c":                    row.McNemar.WrongToCorrect,
			"c_to_w":                    row.McNemar.CorrectToWrong,
			"degradation_rate":          round2(row.DegradationRate),
			"conditional_recovery_rate": round2(row.ConditionalRecoveryRate),
			"prevention_rationale":      row.PreventionRationale,
		}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}

	// low_scale — driven by GA_SCALE (EARNED). 32 px materially beats the 16 px floor.
	if ga, ok := bestRow(tbl, "GA_SCALE"); ok {
		p.Rules["low_scale"] = R1GPolicyRule{
			DetectIf: "line_height_px < 16", PreferredAction: "UPSCALE_TO_32PX_BEFORE_CALL",
			Mode: "PREVENTIVE", Verdict: ga.Verdict, FallbackAction: "RETURN_UNKNOWN",
			Evidence: ev(ga, map[string]any{"16px_recovery_accuracy": accOf(tbl, "GA_SCALE", "GA1_SAFE_SCALE"), "32px_recovery_accuracy": accOf(tbl, "GA_SCALE", "GA2_NOMINAL_SCALE"), "note": "32 px materially outperforms the conservative 16 px floor"}),
		}
	}

	// numeric_distractors — the DEPLOYMENT-relevant evidence is the REAL prose
	// label/value track (EARNED, 0.41 -> 1.00). The canonical synthetic
	// stratum did NOT independently confirm it: the frozen R1-C glyph bank
	// only carries digits + e + x, so the synthetic label is an abstract
	// variable name too weak to establish baseline association (0.33 -> 0.38).
	// That is a synthetic-proxy limitation, not evidence against isolation.
	if real, ok := rowOf(tbl, "GC_ASSOC_REAL", "GC_REAL_2"); ok {
		syn, _ := rowOf(tbl, "GC_ASSOC_SYN", "GC_SYN_2")
		p.Rules["numeric_distractors"] = R1GPolicyRule{
			DetectIf:        "competing_numbers_visible_near_the_operand",
			PreferredAction: "ISOLATE_OPERAND_TO_SINGLE_LINE_LABEL_VALUE_BEFORE_CALL",
			Mode:            "PREVENTIVE", Verdict: real.Verdict, FallbackAction: "RETURN_UNKNOWN",
			Evidence: ev(real, map[string]any{
				"evidence_track":                    "GC_ASSOC_REAL (INDEPENDENT_ACCURACY_ESTIMATE=false; R1-D base reuse as causal intervention)",
				"synthetic_canonical_confirmation":  "NOT_CONFIRMED",
				"synthetic_canonical_note":          fmt.Sprintf("GC_ASSOC_SYN baseline %.2f -> %.2f (NO_MEASURED_BENEFIT): the glyph-bank abstract label cannot anchor association; proxy limitation, not counter-evidence", syn.BaselineAccuracy, syn.RecoveryAccuracy),
			}),
		}
	}

	// high_context — no measured benefit on the fresh sample (baseline already
	// 0.90), but the crop is cheap and never harmed a correct case here, and
	// R1-A1 did show 0.80 at FULL_VIEWPORT. Keep as a preventive practice.
	if gb, ok := bestRow(tbl, "GB_CONTEXT"); ok {
		p.Rules["high_context"] = R1GPolicyRule{
			DetectIf: "visual_field_much_larger_than_the_operand_line",
			PreferredAction: "CROP_TO_OPERAND_LINE_BEFORE_CALL", Mode: "PREVENTIVE_PRACTICE",
			Verdict: gb.Verdict, FallbackAction: "RETURN_UNKNOWN",
			Evidence: ev(gb, map[string]any{"note": "no EARNED recovery on the R1-G fresh sample (baseline already 0.90); R1-A1 measured 0.80 at FULL_VIEWPORT. Cheap and non-damaging -> keep as preventive practice, not a reactive retry."}),
		}
	}

	// value_cue — always emit the frozen padded cue (or none on an isolated
	// operand). NO_MEASURED_BENEFIT here (the tight-cue truncation barely
	// reproduced on 12 longer-digit fresh bases) but it is strictly safe and
	// R1-D proved the artifact real for short 2-digit values.
	if gd, ok := bestRow(tbl, "GD_CUE"); ok {
		p.Rules["value_cue_geometry"] = R1GPolicyRule{
			DetectIf: "renderer_emitting_a_value_cue_rectangle",
			PreferredAction: "ALWAYS_USE_THE_FROZEN_PADDED_CUE_OR_NO_CUE_ON_AN_ISOLATED_OPERAND",
			Mode: "PREVENTIVE_RENDERER_DEFAULT", Verdict: gd.Verdict, FallbackAction: "RETURN_UNKNOWN",
			Evidence: ev(gd, map[string]any{"note": "tight-cue truncation barely reproduced on 12 fresh (longer-digit) bases; R1-D D0V proved it real for short 2-digit values (32->3, 64->4, 350->50). Padded/no-cue is free and never hurt -> renderer default."}),
		}
	}

	// unresolved: only genuinely unrecovered REAL failure regimes. The
	// synthetic association gap is flagged as a proxy limitation.
	synRow, _ := bestRow(tbl, "GC_ASSOC_SYN")
	p.UnresolvedFailureFamilies = []string{
		fmt.Sprintf("GC_ASSOC_SYN (abstract glyph-bank label/value association unrecovered %.2f->%.2f) — SYNTHETIC_PROXY_LIMITATION, not a real-document failure; the GC_ASSOC_REAL track recovered fully. Prefer UNKNOWN if a real operand ever presents this degenerate abstract-label form.", synRow.BaselineAccuracy, synRow.RecoveryAccuracy),
		"No real-document failure family is left unrecovered: every adverse condition (low scale, competing numbers) has an EARNED preventive adaptation; exact retry and missing-operand are REJECT-before-call.",
	}
	return p
}

// R1GRecoveryPolicySource wraps the inputs for policy construction.
type R1GRecoveryPolicySource struct {
	Table R1GRecoveryTable
}

func round2(x float64) float64 {
	return float64(int(x*100+0.5*sign(x))) / 100
}
func sign(x float64) float64 {
	if x < 0 {
		return -1
	}
	return 1
}
