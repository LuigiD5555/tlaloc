package perceptenvelope

import (
	"fmt"
	"sort"
	"strings"
)

// R1GReportInput carries everything the R1-G report needs.
type R1GReportInput struct {
	Dataset      R1GDataset
	Table        R1GRecoveryTable
	Policy       R1GRecoveryPolicy
	Answers      map[string]string
	Model        string
	RecordsSHA   string
	TableSHA     string
	TransSHA     string
	PolicySHA    string
	DatasetSHA   string
	AddendumSHA  string
	RawTreeSHA   string
	ModelIDSHA   string
	TlalocCommit string
}

// RenderR1GReport builds the frozen R1-G markdown report (protocol §20-§24).
func RenderR1GReport(in R1GReportInput) string {
	var b strings.Builder
	p := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	t := in.Table

	p("# Parrot Perceptual Envelope R1 — Stage R1-G (evidence-based recovery policy)\n\n")
	p("**Status: R1-G_RECOVERY_COMPLETE_FROZEN. Final LFM2-VL characterisation stage. No second executor started.**\n\n")
	p("Model `%s`, temp 0, max 32 tok. Fresh held-out bases + a deliberately adverse baseline + a "+
		"predeclared recovery condition, scored paired over the full frozen denominator. "+
		"`EXACT_IDENTICAL_RETRY = %s` (imported from R1-F, 0/16 recovered, **no new model calls**).\n\n",
		in.Model, R1GExactRetryStatus)
	p("Real and synthetic association evidence are never pooled. GC_ASSOC_REAL is `INDEPENDENT_ACCURACY_ESTIMATE = false` "+
		"(R1-D base reuse as a causal intervention track). `CROSS_RECOVERY_FAMILY_BASE_REUSE = %v`.\n\n---\n\n", t.CrossRecoveryFamilyBaseReuse)

	// §12 recovery table
	p("## 1. Recovery table (paired, full frozen denominator)\n\n")
	p("| family | recovery cond | n | baseline acc | recovery acc | Δ | exact p | W→C | C→W | degrade rate | cond. recovery | verdict |\n")
	p("|---|---|--:|--:|--:|--:|---|--:|--:|--:|--:|---|\n")
	for _, r := range t.Rows {
		p("| %s | %s | %d | %.2f | %.2f | %+.2f | %s | %d | %d | %.2f | %.2f | **%s** |\n",
			r.Family, r.RecoveryCondition, r.N, r.BaselineAccuracy, r.RecoveryAccuracy,
			r.McNemar.AbsoluteDelta, fmtPValue(r.McNemar.PValue), r.McNemar.WrongToCorrect, r.McNemar.CorrectToWrong,
			r.DegradationRate, r.ConditionalRecoveryRate, r.Verdict)
	}
	p("\n")

	// per-family detail
	for _, fam := range R1GFamilies {
		p("### %s — %s\n\n", fam.Key, fam.Name)
		p("Motivating evidence: %s\n\n", fam.Evidence)
		for _, r := range t.Rows {
			if r.Family != fam.Key {
				continue
			}
			p("- **%s → %s** (n=%d): baseline %.2f (CI %.2f–%.2f) → recovery %.2f (CI %.2f–%.2f); "+
				"McNemar Δ %+.2f exact p %s · C→C %d, C→W %d, W→C %d, W→W %d. "+
				"contract %d→%d. conditional recovery %d/%d (%.2f). degradation %d/%d (%.2f). "+
				"recovery mean lat %.0f ms. **%s** · mode `%s`.\n",
				r.BaselineCondition, r.RecoveryCondition, r.N,
				r.BaselineAccuracy, r.BaselineCI95[0], r.BaselineCI95[1],
				r.RecoveryAccuracy, r.RecoveryCI95[0], r.RecoveryCI95[1],
				r.McNemar.AbsoluteDelta, fmtPValue(r.McNemar.PValue),
				r.McNemar.CorrectToCorrect, r.McNemar.CorrectToWrong, r.McNemar.WrongToCorrect, r.McNemar.WrongToWrong,
				r.BaselineContractSuccess, r.RecoveryContractSuccess,
				r.RecoveredToCorrect, r.BaselineFailures, r.ConditionalRecoveryRate,
				r.DegradedToWrong, r.BaselineCorrect, r.DegradationRate,
				r.RecoveryMeanLatencyMS, r.Verdict, r.Mode)
			if len(r.RecoveryFailureClasses) > 0 {
				p("  - residual failure mix: %s\n", fmtClasses(r.RecoveryFailureClasses))
			}
		}
		p("\n")
	}

	// §11 OCR
	p("## 2. G_OCR_EXISTING — deterministic system-fallback comparison\n\n")
	if t.OCR.Available {
		p("`OCR_FALLBACK_AVAILABLE = true` · engine `%s`. %s\n\n", t.OCR.Engine, t.OCR.Note)
		p("| family | correct / baseline crops |\n|---|--:|\n")
		var fams []string
		for k := range t.OCR.ByFamily {
			fams = append(fams, k)
		}
		sort.Strings(fams)
		for _, k := range fams {
			v := t.OCR.ByFamily[k]
			p("| %s | %d / %d |\n", k, v[0], v[1])
		}
		p("\nOverall baseline-crop OCR accuracy: **%.2f**. This is a separate deterministic-fallback data point; it does not drive any recovery verdict.\n\n", t.OCR.OverallAcc)
	} else {
		p("`OCR_FALLBACK_AVAILABLE = false` — no local deterministic OCR facility inventoried. No fallback comparison.\n\n")
	}

	// §9 negative control
	p("## 3. G_NEGATIVE_CONTROL_EXACT_RETRY (imported from R1-F)\n\n")
	p("Exact identical retry: `%s`. Previously-wrong sentinels recovered by 5 byte-identical retries: **0/16**. "+
		"Semantic outcome invariant 5/5 in 20/20 sentinels. **New model calls in R1-G for this control: 0.** Verdict: `DO_NOT_USE`.\n\n",
		R1GExactRetryStatus)

	// §20 answers
	p("## 4. Required scientific answers (§20)\n\n")
	order := []struct{ k, label string }{
		{"A_upscale_recovers_low_scale", "A. Does upscale recover low-scale failures?"},
		{"B_16_sufficient_or_32_better", "B. Is 16 px sufficient, or does 32 px materially outperform it?"},
		{"C_context_reduction_recovers", "C. Does reducing context recover high-context failures?"},
		{"D_line_enough_or_target_needed", "D. Is LINE enough, or is TARGET_ONLY necessary?"},
		{"E_remove_competitor_recovers_assoc", "E. Does removing one competing number recover label/value association?"},
		{"F_isolation_adds_benefit", "F. Does complete operand isolation add further benefit?"},
		{"G_cue_fix_removes_truncation", "G. Does fixing the cue/crop remove the R1-D truncation artifact?"},
		{"H_any_recovery_damages_correct", "H. Does any recovery mechanism damage previously correct cases?"},
		{"I_earned_recovery_interventions", "I. Which interventions qualify as EARNED_RECOVERY?"},
		{"J_preventive_adapter_rules", "J. Which should become PREVENTIVE adapter rules?"},
	}
	for _, o := range order {
		p("- **%s** %s\n", o.label, in.Answers[o.k])
	}
	p("- **K. For which failure families is UNKNOWN preferable to another Parrot call?** No real-document failure "+
		"family is left unrecovered — low scale and competing numbers both have an EARNED preventive adaptation, "+
		"and exact-retry / missing-operand are REJECT-before-call. The only unrecovered regime is the abstract "+
		"synthetic label/value form (GC_ASSOC_SYN 0.33→0.38), a glyph-bank proxy limitation rather than a real "+
		"failure; if a real operand ever presents that degenerate form, return UNKNOWN.\n\n")

	// §16 recovery vs prevention
	p("## 5. Recovery vs prevention (§16)\n\n")
	p("Every adverse condition in R1-A…R1-F is detectable *before* the first Parrot call, so each earned/promising "+
		"intervention is recommended as a **PREVENTIVE adaptation**, not fail-first-then-recover:\n\n")
	for _, r := range t.Rows {
		if r.Verdict == "EARNED_RECOVERY" || r.Verdict == "PROMISING_RECOVERY" {
			p("- `%s / %s` (%s): %s\n", r.Family, r.RecoveryCondition, r.Verdict, r.PreventionRationale)
		}
	}
	p("\n")

	// §21 policy
	p("## 6. Provisional recovery policy (§21 — not merged into CapabilityProfile R1)\n\n")
	p("Written to `results/R1G_RECOVERY_POLICY.json` (`%s`). Rules:\n\n", short(in.PolicySHA))
	var keys []string
	for k := range in.Policy.Rules {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		rule := in.Policy.Rules[k]
		p("- **%s** → `%s` (mode `%s`, verdict `%s`)\n", k, rule.PreferredAction, rule.Mode, rule.Verdict)
	}
	if len(in.Policy.UnresolvedFailureFamilies) > 0 {
		p("\nUnresolved failure families (prefer `UNKNOWN`): %s\n", strings.Join(in.Policy.UnresolvedFailureFamilies, "; "))
	}
	p("\n")

	// §22 freeze
	p("## 7. Freeze (§22)\n\n")
	p("`R1G_RECORDS.json` `%s` · `R1G_RECOVERY_TABLE.json` `%s` · `R1G_FAILURE_TRANSITIONS.json` `%s` · "+
		"`R1G_RECOVERY_POLICY.json` `%s` · dataset `%s` · addendum-08 `%s` · model identity `%s` · raw tree `%s` · commit `%s`.\n\n",
		short(in.RecordsSHA), short(in.TableSHA), short(in.TransSHA), short(in.PolicySHA),
		short(in.DatasetSHA), short(in.AddendumSHA), short(in.ModelIDSHA), short(in.RawTreeSHA), in.TlalocCommit)

	// §24 final stop
	p("## 8. FINAL HARD STOP\n\n")
	p("R1-G is complete and frozen. The LFM2-VL recovery characterisation is done. **STOP.** Do NOT automatically "+
		"begin another executor (LFM2 text, Gemma, Dolphin, Claude, ChatGPT, Qwen, DeepSeek), T1, or Grounding R1. "+
		"Return the recovery table, recovery policy, earned/preventive actions, and unresolved failure families for review.\n")
	return b.String()
}
