package perceptenvelope

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

	mkFamilyRule := func(fam, detect, action string) {
		row, ok := bestRow(tbl, fam)
		if !ok {
			return
		}
		mode := row.Mode
		if row.Verdict == "HARMFUL" || row.Verdict == "NO_MEASURED_BENEFIT" {
			mode = "DO_NOT_APPLY_REACTIVELY"
		}
		p.Rules[detect] = R1GPolicyRule{
			DetectIf:        detect,
			PreferredAction: action,
			Mode:            mode,
			Verdict:         row.Verdict,
			Evidence: map[string]any{
				"baseline_accuracy":  round2(row.BaselineAccuracy),
				"recovery_accuracy":  round2(row.RecoveryAccuracy),
				"delta":              round2(row.McNemar.AbsoluteDelta),
				"mcnemar_exact_p":    row.McNemar.PValue,
				"w_to_c":             row.McNemar.WrongToCorrect,
				"c_to_w":             row.McNemar.CorrectToWrong,
				"degradation_rate":   round2(row.DegradationRate),
				"conditional_recovery_rate": round2(row.ConditionalRecoveryRate),
				"prevention_rationale": row.PreventionRationale,
			},
			FallbackAction: "RETURN_UNKNOWN",
		}
	}
	mkFamilyRule("GA_SCALE", "line_height_px < 16", "UPSCALE_TO_32PX_BEFORE_CALL")
	mkFamilyRule("GB_CONTEXT", "visual_field_much_larger_than_operand_line", "CROP_TO_OPERAND_LINE_OR_TARGET_BEFORE_CALL")
	mkFamilyRule("GC_ASSOC_SYN", "competing_numbers_visible_near_operand", "ISOLATE_OPERAND_TO_SINGLE_LINE_LABEL_VALUE_BEFORE_CALL")
	mkFamilyRule("GD_CUE", "value_cue_tighter_than_frozen_padded_rule", "USE_PADDED_VALUE_CUE_OR_NO_CUE_ON_ISOLATED_OPERAND")

	// unresolved: any family whose best verdict is NO_MEASURED_BENEFIT / HARMFUL / INSUFFICIENT
	for _, fam := range R1GFamilies {
		row, ok := bestRow(tbl, fam.Key)
		if !ok {
			continue
		}
		if row.Verdict == "NO_MEASURED_BENEFIT" || row.Verdict == "HARMFUL" || row.Verdict == "INSUFFICIENT_EVIDENCE" {
			p.UnresolvedFailureFamilies = append(p.UnresolvedFailureFamilies, fam.Key+" (best recovery "+row.Verdict+"; prefer UNKNOWN over another Parrot call)")
		}
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
