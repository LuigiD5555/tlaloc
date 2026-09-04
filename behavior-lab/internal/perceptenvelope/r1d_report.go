package perceptenvelope

import (
	"fmt"
	"sort"
	"strings"
)

// R1DReportInput carries everything the R1-D report needs.
type R1DReportInput struct {
	Alloc        R1DAllocation
	D0           R1D0Table
	D1           R1D1Curve
	Verdict      R1DVerdict
	Model        string
	AssocRecSHA  string
	AssocTblSHA  string
	DistRecSHA   string
	DistCurveSHA string
	TaxonomySHA  string
	DatasetSHA   string
	AddendumSHA  string
	RawTreeSHA   string
	TlalocCommit string
}

// RenderR1DReport builds the frozen R1-D markdown report (protocol §25-§27).
func RenderR1DReport(in R1DReportInput) string {
	var b strings.Builder
	p := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	d0, d1 := in.D0, in.D1

	p("# Parrot Perceptual Envelope R1 — Stage R1-D (label/value association + distractor density)\n\n")
	p("**Status: R1-D_ASSOCIATION_DISTRACTOR_COMPLETE_FROZEN. R1-E…R1-G not started.**\n\n")
	p("Model `%s`, temp 0, max 32 tok. Presentation: 32 px line height, 512×512, single-line "+
		"label/value viewport with other-line pixels masked. Primary operand morphology = "+
		"MULTI_DIGIT_INTEGER only. New behaviour-lab opcode `READ_ASSOCIATED_NUMBER` (not yet "+
		"promoted to the T0 Micro-ISA).\n\n", in.Model)
	p("Eligible real bases: **%d / %d** pool candidates (min required 18). "+
		"D0 = REAL_DOCUMENT, D1 = CONTROLLED_COMPOSITE — never pooled.\n\n---\n\n",
		in.Alloc.EligibleCount, in.Alloc.PoolCount)

	p("## 1. D0 — real-document association (paired, same viewport pixels)\n\n")
	p("| condition | opcode | n | VALUE acc | Wilson 95%% | contract | abstain | commentary | mean lat ms |\n")
	p("|---|---|--:|--:|---|--:|--:|--:|--:|\n")
	for _, r := range d0.Rows {
		p("| %s | %s | %d | %.2f | %.2f–%.2f | %d | %d | %d | %.0f |\n",
			r.Condition, r.Opcode, r.N, r.ValueAccuracy, r.CI95Low, r.CI95High,
			r.ContractSuccess, r.Abstained, r.UnsupportedAssertion, r.MeanLatencyMS)
	}
	m := d0.PairedMcNemar
	p("\n**Paired McNemar D0V → D0L (value):** Δ %+.3f, exact p %.4f · C→C %d, C→W %d, W→C %d, W→W %d\n\n",
		m.AbsoluteDelta, m.PValue, m.CorrectToCorrect, m.CorrectToWrong, m.WrongToCorrect, m.WrongToWrong)
	p("**Geometry gate (§24):** %s → `R1D_REAL_ASSOCIATION_GEOMETRY_VALID = %v`. Association cost "+
		"(D0V→D0L) = %+.2f.\n\n%s\n\n", d0.GeometryValidThreshold, d0.RealAssocGeometryValid, d0.AssociationCost, d0.GateInterpretation)

	p("### D0 failure taxonomy\n\n")
	for _, r := range d0.Rows {
		if len(r.FailureClasses) == 0 {
			continue
		}
		p("- **%s**: %s\n", r.Condition, fmtClasses(r.FailureClasses))
	}
	p("\n")

	p("## 2. D1 — controlled distractor density (%s)\n\n", strings.ToUpper(d1.CanonicalNote[:1])+d1.CanonicalNote[1:])
	p("| K | n | VALUE acc | Wilson 95%% | contract | →distractor | →other visible | hallucinated | in band |\n")
	p("|--:|--:|--:|---|--:|--:|--:|--:|:--:|\n")
	for _, r := range d1.Rows {
		p("| %d | %d | %.2f | %.2f–%.2f | %d | %d | %d | %d | %v |\n",
			r.K, r.N, r.ValueAccuracy, r.CI95Low, r.CI95High, r.ContractSuccess,
			r.SelectedDistr, r.SelectedOther, r.Hallucinated, r.InOperatingBand)
	}
	p("\n**Paired McNemar:**\n\n| transition | Δ | p | C→W | W→C |\n|---|--:|--:|--:|--:|\n")
	for _, tr := range d1.Transitions {
		p("| %s → %s | %+.3f | %.4f | %d | %d |\n", tr.From, tr.To, tr.AbsoluteDelta, tr.PValue, tr.CorrectToWrong, tr.WrongToCorrect)
	}
	p("\n")
	if d1.DensityCliffK != nil {
		p("**Density cliff:** first significant drop at K = %d.\n", *d1.DensityCliffK)
	} else {
		p("**Density cliff:** none detected within the tested ladder.\n")
	}
	if d1.OperationalExit != nil {
		p("**Operational exit:** reliability leaves the operating band (acc ≥ 0.90 ∧ Wilson lower ≥ 0.70) at K = %d.\n", *d1.OperationalExit)
	} else {
		p("**Operational exit:** stays in the operating band across all tested K.\n")
	}
	p("\n**Wrong-answer mix (§19):** ")
	var mixKeys []string
	for k := range d1.ResponseMix {
		mixKeys = append(mixKeys, k)
	}
	sort.Strings(mixKeys)
	var mp []string
	for _, k := range mixKeys {
		mp = append(mp, fmt.Sprintf("%s %.0f%%", k, d1.ResponseMix[k]*100))
	}
	p("%s\n\n", strings.Join(mp, " · "))

	p("## 3. Primary scientific questions (§25)\n\n")
	d0v, d0l := d0.Rows[0], d0.Rows[1]
	p("- **A. Atomic value reading still reliable in the association viewport?** D0V value %.2f (CI %.2f–%.2f), n=%d.\n", d0v.ValueAccuracy, d0v.CI95Low, d0v.CI95High, d0v.N)
	p("- **B. Cost of VALUE_CUED → LABEL_CUED?** Δ %+.2f (McNemar p %.4f).\n", d0.AssociationCost, m.PValue)
	p("- **C. One-step label → numeric value association?** D0L value %.2f (CI %.2f–%.2f).\n", d0l.ValueAccuracy, d0l.CI95Low, d0l.CI95High)
	p("- **D. Wrong answers usually nearby real values?** %s\n", nearbyNote(d0l))
	p("- **E–G. Distractor density effect / cliff / operating exit:** %s\n", densityNote(d1))
	p("- **H. Hallucinate vs select visible competitors?** %s\n", mixNote(d1))
	p("- **I. One-op rule adequate for READ_ASSOCIATED_NUMBER?** contract success D0L %d/%d; commentary %d.\n", d0l.ContractSuccess, d0l.N, d0l.UnsupportedAssertion)
	p("\n")

	p("## 4. Provisional capability verdict (§26)\n\n")
	p("`%s` → **%s**\n\n%s\n\nConstraints:\n", in.Verdict.Capability, in.Verdict.Verdict, in.Verdict.Basis)
	for _, cst := range in.Verdict.Constraints {
		p("- %s\n", cst)
	}
	p("\n")

	p("## 5. Freeze (§27)\n\n")
	p("`R1D_ASSOCIATION_RECORDS.json` `%s` · `R1D_ASSOCIATION_TABLE.json` `%s` · "+
		"`R1D_DISTRACTOR_RECORDS.json` `%s` · `R1D_DISTRACTOR_CURVE.json` `%s` · "+
		"`R1D_FAILURE_TAXONOMY.json` `%s` · dataset `%s` · addendum-05 `%s` · raw tree `%s` · commit `%s`.\n\n",
		short(in.AssocRecSHA), short(in.AssocTblSHA), short(in.DistRecSHA), short(in.DistCurveSHA),
		short(in.TaxonomySHA), short(in.DatasetSHA), short(in.AddendumSHA), short(in.RawTreeSHA), in.TlalocCommit)

	p("## 6. HARD STOP\n\n")
	p("R1-D is complete and frozen. **Do NOT run R1-E (visual shortcut controls), R1-F " +
		"(repeatability) or R1-G (recovery).** Return the full R1-D report for review.\n")
	return b.String()
}

func fmtClasses(m map[string]int) string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	var parts []string
	for _, k := range ks {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	return strings.Join(parts, ", ")
}

func nearbyNote(d0l R1DConditionRow) string {
	wv := d0l.FailureClasses["WRONG_VISIBLE_VALUE"]
	hall := d0l.FailureClasses["HALLUCINATED_VALUE"]
	return fmt.Sprintf("D0 has only the target visible, so nearby-value confusion is a D1 phenomenon; D0L wrong-visible=%d, hallucinated=%d, echo=%d, truncation=%d",
		wv, hall, d0l.FailureClasses["LABEL_TEXT_ECHO"], d0l.FailureClasses["VALUE_TRUNCATION"])
}

func densityNote(d1 R1D1Curve) string {
	if len(d1.Rows) == 0 {
		return "no D1 data"
	}
	first, last := d1.Rows[0], d1.Rows[len(d1.Rows)-1]
	cliff := "no cliff"
	if d1.DensityCliffK != nil {
		cliff = fmt.Sprintf("cliff at K=%d", *d1.DensityCliffK)
	}
	exit := "no operating-band exit"
	if d1.OperationalExit != nil {
		exit = fmt.Sprintf("exits operating band at K=%d", *d1.OperationalExit)
	}
	return fmt.Sprintf("K0 %.2f → K%d %.2f; %s; %s", first.ValueAccuracy, last.K, last.ValueAccuracy, cliff, exit)
}

func mixNote(d1 R1D1Curve) string {
	if len(d1.ResponseMix) == 0 {
		return "no wrong answers to analyse"
	}
	return fmt.Sprintf("of wrong answers: %.0f%% an added distractor, %.0f%% another visible number, %.0f%% hallucinated",
		d1.ResponseMix["equals_added_distractor"]*100, d1.ResponseMix["equals_other_visible_number"]*100, d1.ResponseMix["hallucinated"]*100)
}
