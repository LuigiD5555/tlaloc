package perceptenvelope

import (
	"fmt"
	"sort"
	"strings"
)

// R1BReportInput carries everything the R1-B report needs.
type R1BReportInput struct {
	Curve        R1BScaleCurve
	Audit        R1BGeometryAudit
	BasesSHA     string
	DatasetSHA   string
	AuditSHA     string
	AddendumSHA  string
	RecordsSHA   string
	CurveSHA     string
	RawTreeSHA   string
	TlalocCommit string
	Model        string
}

// RenderR1BReport builds the frozen R1-B markdown report (protocol §21).
func RenderR1BReport(in R1BReportInput) string {
	c := in.Curve
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }

	p("# Parrot Perceptual Envelope R1 — Stage R1-B (scale / resolution envelope)\n\n")
	p("**Status: R1-B_SCALE_ENVELOPE_COMPLETE_FROZEN. R1-C…R1-G not started.**\n\n")
	p("Model `%s` F16, llama.cpp CUDA backend, temp 0, max 32 tok. Context policy: fixed `A1C0_TARGET` crop content (`R1B_CONTEXT_POLICY.json`).\n\n", in.Model)
	p("---\n\n")

	p("## 1. Frozen R1-B bases\n\n")
	p("`R1B_BASES.json` `%s` — 30 reserved held-out bases, disjoint from R1-A0 and R1-A1 (`r1b_bases_disjoint_from_r1a0_and_r1a1 = %v`). No manual curation.\n\n", short(in.BasesSHA), c8(in.Audit.Checks["r1b_bases_disjoint_from_r1a0_and_r1a1"]))

	p("## 2. Scale geometry proof (`R1B_GEOMETRY_AUDIT.json` `%s`)\n\n", short(in.AuditSHA))
	p("Frozen source crop = R1-A1 `A1C0_TARGET` reveal region mapped to store units (cue token bbox + 10 px / 32 px-scale pad). Every B0…B5 resamples the SAME store rectangle; the rest of the fixed 512×512 canvas is neutral RGB(200,200,200); target centre at (256,256) for every condition.\n\n")
	p("| check | pass |\n|---|---|\n")
	keys := make([]string, 0, len(in.Audit.Checks))
	for k := range in.Audit.Checks {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p("| %s | %v |\n", k, c8(in.Audit.Checks[k]))
	}
	p("\nResampler: %s. No sharpen/denoise/threshold/OCR/contrast. Line-height tolerance ±%.1f px.\n\n", in.Audit.Resampler, R1BLineHeightTolerancePx)

	p("## 3. Constant token-regime proof\n\n")
	p("%s. `token_regime_constant = %v`. `STOP_AND_AUDIT_PREPROCESSING = %v`.\n\n", c.TokenRegimeNote, c.TokenRegimeConstant, c.StopAuditPreprocess)

	p("## 4. Six-point scale curve (`R1B_SCALE_CURVE.json` `%s`, raw tree `%s`, %d records, %d errors)\n\n", short(in.CurveSHA), short(in.RawTreeSHA), c.Records, c.Errors)
	p("| cond | nominal px | actual line px | sem | 95%% CI (Wilson) | contract | abstain | commentary | mean tok | mean lat ms | region |\n|---|--:|--:|--:|---|--:|--:|--:|--:|--:|---|\n")
	for _, r := range c.Rows {
		p("| %s | %.0f | %.2f | %d/%d (%.2f) | %.2f–%.2f | %d/%d | %d | %d | %.0f | %.0f | %s |\n",
			r.Level, r.NominalLinePx, r.ActualLineHeightPx,
			r.SemanticCorrect, r.N, r.SemanticAccuracy, r.SemanticCI95Low, r.SemanticCI95High,
			r.ContractSuccess, r.N, r.Abstained, r.FailureClasses["COMMENTARY_CONTAMINATION"],
			r.MeanPromptTokens, r.MeanLatencyMS, r.Region)
	}
	p("\n")

	p("## 5. Paired McNemar transitions\n\n")
	p("| transition | Δ | exact p | C→C | C→W | W→C | W→W |\n|---|--:|--:|--:|--:|--:|--:|\n")
	for _, t := range c.Transitions {
		p("| %s → %s | %+.3f | %.4f | %d | %d | %d | %d |\n", t.From, t.To, t.AbsoluteDelta, t.PValue, t.CorrectToCorrect, t.CorrectToWrong, t.WrongToCorrect, t.WrongToWrong)
	}
	p("\n")

	p("## 6. Failure taxonomy by scale\n\n")
	classes := map[string]struct{}{}
	for _, r := range c.Rows {
		for k := range r.FailureClasses {
			classes[k] = struct{}{}
		}
	}
	var cl []string
	for k := range classes {
		cl = append(cl, k)
	}
	sort.Strings(cl)
	p("| cond |")
	for _, k := range cl {
		p(" %s |", k)
	}
	p("\n|---|")
	for range cl {
		p("--:|")
	}
	p("\n")
	for _, r := range c.Rows {
		p("| %s |", r.Level)
		for _, k := range cl {
			p(" %d |", r.FailureClasses[k])
		}
		p("\n")
	}
	p("\n")

	p("## 7. Formal safe scale & operating region\n\n")
	if c.FormalSafeScalePx != nil {
		p("`formal_safe_scale_px = %.0f`. ", *c.FormalSafeScalePx)
	} else {
		p("`formal_safe_scale_px = null` (no rung meets the rule). ")
	}
	p("Rule: %s.\n\n", c.FormalSafeScaleRule)
	p("Observed operating region: %v px. Overscale degradation observed: %v.\n\n", c.ObservedOperatingPx, c.OverscaleDegradation)

	p("## 8. Consistency with R1-A0 full-page failure\n\n%s\n\n", c.R1A0ConsistencyNote)

	p("## 9. Recommended scale policy for R1-C\n\n")
	p("Freeze the presentation at **%.0f px** containing-line height (R1-C then varies numeric morphology at this fixed scale + `A1C0_TARGET` context).\n\n", c.RecommendedR1CScalePx)

	p("## 10. Checkpoint\n\n")
	p("`R1B_CHECKPOINT.json` — records `%s`, curve `%s`, geometry audit `%s`, addendum-03 `%s`, raw tree `%s`, tlaloc commit `%s`.\n\n", short(in.RecordsSHA), short(in.CurveSHA), short(in.AuditSHA), short(in.AddendumSHA), short(in.RawTreeSHA), in.TlalocCommit)

	p("## 11. Readiness for R1-C NUMERIC MORPHOLOGY\n\n")
	p("R1-B is complete and frozen. Context (R1-A1) and scale (R1-B) are now both characterised; one presentation policy can be frozen and R1-C can ask which numeric forms themselves break LFM2-VL. **R1-C is NOT started. HARD STOP — do not run R1-C/D/E/F/G. Review R1-B first.**\n")

	return b.String()
}

func short(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

func c8(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}
