package perceptenvelope

import (
	"fmt"
	"strings"
)

// R1EReportInput carries everything the R1-E report needs.
type R1EReportInput struct {
	Dataset      R1EDataset
	Table        R1EVisualDependenceTable
	Model        string
	RecordsSHA   string
	TableSHA     string
	DatasetSHA   string
	WrongMapSHA  string
	AddendumSHA  string
	RawTreeSHA   string
	TlalocCommit string
}

// RenderR1EReport builds the frozen R1-E markdown report (protocol §13-§15).
func RenderR1EReport(in R1EReportInput) string {
	var b strings.Builder
	p := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }

	p("# Parrot Perceptual Envelope R1 — Stage R1-E (visual dependence / shortcut controls)\n\n")
	p("**Status: R1-E_VISUAL_DEPENDENCE_COMPLETE_FROZEN. R1-F / R1-G not started.**\n\n")
	p("Model `%s`, temp 0, max 32 tok. Three interventions with byte-identical model-facing "+
		"text: `E0_NO_IMAGE` (no visual operand), `E1_WRONG_IMAGE` (a plausible viewport from a "+
		"different eligible base whose visible value differs), `E2_CORRECT_IMAGE` (the frozen "+
		"R1-D crop).\n\n", in.Model)
	p("`INTERVENTION_REUSE_OF_R1D_BASES = true`. %s\n\n---\n\n", in.Dataset.Note)

	p("## 1. Visual-dependence table\n\n")
	p("| capability | role | correct-image | no-image | wrong-image → task gold | wrong-image → visible operand | classification |\n")
	p("|---|---|--:|--:|--:|--:|---|\n")
	for _, c := range in.Table.Capabilities {
		p("| `%s` | %s | %.2f | %.2f | %.2f | %.2f | **%s** |\n",
			c.Capability, c.Role, c.CorrectImageAccuracy, c.NoImageAccuracy,
			c.WrongImageTaskGoldAccuracy, c.WrongImageVisibleOperandAccuracy, c.Classification)
	}
	p("\n")

	for _, c := range in.Table.Capabilities {
		p("### `%s` (%s)\n\n", c.Capability, c.Role)
		p("Bases %d · digit-length-matched wrong pairs %d/%d\n\n", c.Bases, c.DigitLenMatchedPairs, c.PairsTotal)
		p("| condition | n | task-gold acc | Wilson 95%% | image-consistent | contract | abstain | commentary | mean lat ms |\n")
		p("|---|--:|--:|---|--:|--:|--:|--:|--:|\n")
		for _, r := range c.Rows {
			ic := "—"
			if r.Condition == "E1_WRONG_IMAGE" {
				ic = fmt.Sprintf("%.2f", r.ImageConsistentAccuracy)
			}
			p("| %s | %d | %.2f | %.2f–%.2f | %s | %d | %d | %d | %.0f |\n",
				r.Condition, r.N, r.TaskGoldAccuracy, r.CI95Low, r.CI95High, ic,
				r.ContractSuccess, r.Abstained, r.UnsupportedAssertion, r.MeanLatencyMS)
		}
		m1, m2 := c.McNemarCorrectVsNoImage, c.McNemarCorrectVsWrongImage
		p("\n**Paired McNemar** (task gold, exact two-sided binomial on the discordant pairs):\n\n")
		p("- E2_CORRECT → E0_NO_IMAGE: Δ %+.3f, exact p %s · C→C %d, C→W %d, W→C %d, W→W %d\n",
			m1.AbsoluteDelta, fmtPValue(m1.PValue), m1.CorrectToCorrect, m1.CorrectToWrong, m1.WrongToCorrect, m1.WrongToWrong)
		p("- E2_CORRECT → E1_WRONG_IMAGE: Δ %+.3f, exact p %s · C→C %d, C→W %d, W→C %d, W→W %d\n\n",
			m2.AbsoluteDelta, fmtPValue(m2.PValue), m2.CorrectToCorrect, m2.CorrectToWrong, m2.WrongToCorrect, m2.WrongToWrong)
		p("**Classification:** %s — %s\n\n", c.Classification, c.ClassificationBasis)
		for _, r := range c.Rows {
			if len(r.FailureClasses) == 0 {
				continue
			}
			p("- %s failure mix: %s\n", r.Condition, fmtClasses(r.FailureClasses))
		}
		p("\n")
	}

	// §13 READ_ASSOCIATED_NUMBER questions
	var prim R1ECapabilityTable
	for _, c := range in.Table.Capabilities {
		if c.Capability == "READ_ASSOCIATED_NUMBER" {
			prim = c
		}
	}
	disp, why := R1EReadAssocDisposition(in.Table)
	p("## 2. READ_ASSOCIATED_NUMBER — protocol §13\n\n")
	p("- **A. Does the 22/22 R1-D association result collapse with NO_IMAGE?** no-image task-gold accuracy = %.2f (correct-image %.2f; drop %+.2f, McNemar exact p %s). %s\n",
		prim.NoImageAccuracy, prim.CorrectImageAccuracy, prim.CorrectImageAccuracy-prim.NoImageAccuracy,
		fmtPValue(prim.McNemarCorrectVsNoImage.PValue), collapseNote(prim))
	p("- **B. With a plausible WRONG_IMAGE, does the model follow the original task gold or the value in the wrong image?** task-gold %.2f vs visible-operand %.2f (gap %+.2f). %s\n",
		prim.WrongImageTaskGoldAccuracy, prim.WrongImageVisibleOperandAccuracy,
		prim.WrongImageVisibleOperandAccuracy-prim.WrongImageTaskGoldAccuracy, followNote(prim))
	p("- **C. Is READ_ASSOCIATED_NUMBER genuinely visual?** %s\n", genuineNote(prim))
	p("- **D. Evidence of parametric / textual shortcut behaviour?** %s\n", shortcutNote(prim))
	p("- **E. Disposition:** `%s` — %s\n\n", disp, why)

	// §14 SELECT_ONE
	p("## 3. SELECT_ONE — protocol §14\n\n")
	p("SELECT_ONE (and READ_SHORT_LABEL / EXTRACT_ENTITY) have **no frozen suitable stimulus set** "+
		"inside the R1 perceptual-envelope experiment: R1-A…R1-D only ever built EXTRACT_NUMBER and "+
		"READ_ASSOCIATED_NUMBER stimuli. Per §9 (\"do NOT change selection rules merely to manufacture "+
		"fresh cases\") R1-E does not fabricate one. The T0-B SELECT_ONE shortcut check is therefore "+
		"deferred to a dedicated stage with purpose-built matched pages; the positive calibration "+
		"control here is `%s`, whose result is `%s` and bounds the method: %s\n\n",
		frozenOpcodeName(in.Table), controlClass(in.Table), controlNote(in.Table))

	p("## 4. Freeze (§15)\n\n")
	p("`R1E_RECORDS.json` `%s` · `R1E_VISUAL_DEPENDENCE_TABLE.json` `%s` · dataset `%s` · "+
		"wrong-image map `%s` · addendum-06 `%s` · raw tree `%s` · commit `%s`.\n\n",
		short(in.RecordsSHA), short(in.TableSHA), short(in.DatasetSHA), short(in.WrongMapSHA),
		short(in.AddendumSHA), short(in.RawTreeSHA), in.TlalocCommit)

	p("## 5. HARD STOP\n\n")
	p("R1-E is complete and frozen. **Do NOT run R1-F or R1-G.** Return the complete visual-dependence table for review.\n")
	return b.String()
}

// fmtPValue renders an exact p-value without collapsing a tiny-but-nonzero
// value to "0.0000". For 22 one-directional discordant pairs the exact
// two-sided binomial value is 2 * 0.5^22 ≈ 4.77e-07, not zero.
func fmtPValue(p float64) string {
	switch {
	case p <= 0:
		return "0"
	case p >= 1e-4:
		return fmt.Sprintf("%.4f", p)
	default:
		return fmt.Sprintf("%.2e", p)
	}
}

func collapseNote(c R1ECapabilityTable) string {
	switch {
	case c.CorrectImageAccuracy-c.NoImageAccuracy >= r1eMaterialDrop:
		return "Yes — removing the image materially collapses the result."
	case c.CorrectImageAccuracy-c.NoImageAccuracy >= r1eMinorGap:
		return "Partly — the result degrades but does not fully collapse."
	default:
		return "No — the headline largely survives with no image (shortcut/prior signal)."
	}
}

func followNote(c R1ECapabilityTable) string {
	switch {
	case c.WrongImageVisibleOperandAccuracy-c.WrongImageTaskGoldAccuracy >= r1eMaterialGap:
		return "It follows the wrong image's visible operand — strong evidence the image drives the answer."
	case c.WrongImageTaskGoldAccuracy >= r1eHighAcc:
		return "It keeps returning the original task gold despite the wrong image — shortcut/prior behaviour."
	default:
		return "Mixed: neither the wrong operand nor the original gold dominates (often abstention/other)."
	}
}

func genuineNote(c R1ECapabilityTable) string {
	switch c.Classification {
	case "VISUALLY_DEPENDENT":
		return "Yes — classified VISUALLY_DEPENDENT."
	case "MIXED_VISUAL_AND_PRIOR":
		return "Partially — classified MIXED_VISUAL_AND_PRIOR; the image matters but is not the sole driver."
	case "SHORTCUT_DOMINATED":
		return "No — classified SHORTCUT_DOMINATED."
	default:
		return "Undetermined — INSUFFICIENT_EVIDENCE."
	}
}

func shortcutNote(c R1ECapabilityTable) string {
	if c.NoImageAccuracy >= r1eMinorGap || c.WrongImageTaskGoldAccuracy >= r1eMinorGap {
		return fmt.Sprintf("Yes, some: no-image still scores %.2f and wrong-image still returns the task gold %.2f of the time (many R1-D golds are the common value \"10\", a plausible prior).",
			c.NoImageAccuracy, c.WrongImageTaskGoldAccuracy)
	}
	return "No measurable prior: no-image and wrong-image task-gold accuracy are both near zero."
}

func frozenOpcodeName(t R1EVisualDependenceTable) string {
	for _, c := range t.Capabilities {
		if c.Role == "POSITIVE_CALIBRATION_CONTROL" {
			return c.Capability
		}
	}
	return "EXTRACT_NUMBER"
}

func controlClass(t R1EVisualDependenceTable) string {
	for _, c := range t.Capabilities {
		if c.Role == "POSITIVE_CALIBRATION_CONTROL" {
			return c.Classification
		}
	}
	return "INSUFFICIENT_EVIDENCE"
}

func controlNote(t R1EVisualDependenceTable) string {
	for _, c := range t.Capabilities {
		if c.Role == "POSITIVE_CALIBRATION_CONTROL" {
			return c.ClassificationBasis
		}
	}
	return "no control rows"
}
