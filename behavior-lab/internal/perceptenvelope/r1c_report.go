package perceptenvelope

import (
	"fmt"
	"sort"
	"strings"
)

// R1CReportInput carries everything the R1-C report needs.
type R1CReportInput struct {
	Table        R1CMorphologyTable
	Alloc        R1CAllocation
	Model        string
	GlyphBankSHA string
	DatasetSHA   string
	ScorerNote   string
	RecordsSHA   string
	TableSHA     string
	TaxonomySHA  string
	RawTreeSHA   string
	AddendumSHA  string
	TlalocCommit string
}

// RenderR1CReport builds the frozen R1-C markdown report (protocol §18-§22).
func RenderR1CReport(in R1CReportInput) string {
	t := in.Table
	var b strings.Builder
	p := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }

	p("# Parrot Perceptual Envelope R1 — Stage R1-C (numeric morphology)\n\n")
	p("**Status: R1-C_NUMERIC_MORPHOLOGY_COMPLETE_FROZEN. R1-D…R1-G not started.**\n\n")
	p("Model `%s`, temp 0, max 32 tok. Presentation frozen inside the R1-A1/R1-B envelope: "+
		"line height **%.0f px**, context **%s**, canvas 512×512, cue spans the operand, one atomic "+
		"`EXTRACT_NUMBER` opcode. R1-C varies morphology only.\n\n", in.Model, t.LineHeightPx, t.ContextLevel)
	p("Dual endpoints (frozen before inference): **VALUE_CORRECT** (numeric/structured meaning) "+
		"and **SURFACE_FORM_CORRECT** (faithful transcription of the visible string). Exact "+
		"`math/big` comparison; no float equality. REAL_DOCUMENT and SYNTHETIC_REALISTIC never pooled. "+
		"Synthetic = glyph bank cut from the real corpus (`%s`).\n\n", short(in.GlyphBankSHA))
	p("`%d records, %d errors` · dataset `%s` · scorer self-test: %s\n\n---\n\n", t.Records, t.Errors, short(in.DatasetSHA), in.ScorerNote)

	p("## 1. Per-family × per-stratum table (§18)\n\n")
	p("| family | stratum | prov | n | VALUE acc | VALUE CI95 | SURFACE acc | SURFACE CI95 | contract | abstain | commentary | V&S | V&!S | !V |\n")
	p("|---|---|---|--:|--:|---|--:|---|--:|--:|--:|--:|--:|--:|\n")
	for _, r := range t.Rows {
		p("| %s | %s | %s | %d | %.2f | %.2f–%.2f | %.2f | %.2f–%.2f | %d | %d | %d | %d | %d | %d |\n",
			r.Family, shortStratum(r.Stratum), shortProv(r.Provenance), r.N,
			r.Value.Accuracy, r.Value.CI95Low, r.Value.CI95High,
			r.Surface.Accuracy, r.Surface.CI95Low, r.Surface.CI95High,
			r.ContractSuccess, r.Abstained, r.UnsupportedAssertion,
			r.ValueAndSurface, r.ValueNotSurface, r.NotValue)
	}
	p("\n")

	// digit-length subgroups
	for _, r := range t.Rows {
		if r.Family == FamMultiDigit && r.DigitLenSubgroups != nil && r.Provenance != "SYNTHETIC_REALISTIC" {
			p("### MULTI_DIGIT_INTEGER digit-length subgroups (§13, descriptive)\n\n")
			var sg []string
			for k := range r.DigitLenSubgroups {
				sg = append(sg, k)
			}
			sort.Strings(sg)
			p("| digits | n | VALUE acc | CI95 |\n|---|--:|--:|---|\n")
			for _, k := range sg {
				e := r.DigitLenSubgroups[k]
				p("| %s | %d | %.2f | %.2f–%.2f |\n", k, e.N, e.Accuracy, e.CI95Low, e.CI95High)
			}
			p("\n")
		}
	}

	p("## 2. Failure taxonomy by family (§12)\n\n")
	for _, r := range t.Rows {
		if len(r.FailureClasses) == 0 {
			continue
		}
		var cls []string
		for k := range r.FailureClasses {
			cls = append(cls, k)
		}
		sort.Strings(cls)
		var parts []string
		for _, c := range cls {
			parts = append(parts, fmt.Sprintf("%s=%d", c, r.FailureClasses[c]))
		}
		p("- **%s** (%s): %s\n", r.Family, shortProv(r.Provenance), strings.Join(parts, ", "))
	}
	p("\n")

	p("## 3. Cross-metric analysis (§19)\n\n")
	p("The **V&!S** column above is the key R1-C finding: numeric meaning retained, visible " +
		"morphology not faithfully transcribed. Families with V&!S > 0:\n\n")
	for _, r := range t.Rows {
		if r.ValueNotSurface > 0 {
			p("- %s (%s): %d/%d — %s\n", r.Family, shortProv(r.Provenance), r.ValueNotSurface, r.N, dominantVNSClass(r))
		}
	}
	p("\n")

	p("## 4. Primary scientific questions (§20)\n\n")
	keys := make([]string, 0, len(t.Answers))
	for k := range t.Answers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p("- **%s** — %s\n", k, t.Answers[k])
	}
	p("\n")

	p("## 5. Provisional capability verdicts (§21)\n\n")
	p("| family | verdict | basis |\n|---|---|---|\n")
	for _, v := range t.Verdicts {
		p("| %s | **%s** | %s |\n", v.Family, v.Verdict, v.Basis)
	}
	p("\nSynthetic evidence never promotes a family to real-document RELIABLE.\n\n")

	p("## 6. Freeze (§22)\n\n")
	p("`R1C_RECORDS.json` `%s` · `R1C_MORPHOLOGY_TABLE.json` `%s` · `R1C_FAILURE_TAXONOMY.json` `%s` · "+
		"raw tree `%s` · `R1_PROTOCOL_ADDENDUM_04.json` `%s` · glyph bank `%s` · tlaloc commit `%s`.\n\n",
		short(in.RecordsSHA), short(in.TableSHA), short(in.TaxonomySHA), short(in.RawTreeSHA),
		short(in.AddendumSHA), short(in.GlyphBankSHA), in.TlalocCommit)

	p("## 7. HARD STOP\n\n")
	p("R1-C is complete and frozen. **Do NOT run R1-D (distractor / label-value association), " +
		"R1-E (shortcut controls), R1-F (repeatability) or R1-G (recovery).** Return the " +
		"morphology table for review.\n")

	return b.String()
}

func shortStratum(s string) string {
	if s == StratumLayout {
		return "layout"
	}
	return "lexical"
}

func shortProv(s string) string {
	switch s {
	case "SYNTHETIC_REALISTIC":
		return "synth"
	case "REAL_DOCUMENT_SMALL_N":
		return "real·smallN"
	default:
		return "real"
	}
}

func dominantVNSClass(r R1CRow) string {
	best, bestN := "", 0
	for c, n := range r.FailureClasses {
		if n > bestN {
			best, bestN = c, n
		}
	}
	if best == "" {
		return "surface-only differences"
	}
	return best
}
