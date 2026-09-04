package perceptenvelope

import (
	"fmt"
	"strings"
)

// R1FReportInput carries everything the R1-F report needs.
type R1FReportInput struct {
	Dataset      R1FDataset
	Table        R1FSentinelTable
	Summary      R1FStabilitySummary
	Model        string
	RecordsSHA   string
	TableSHA     string
	SummarySHA   string
	SentinelsSHA string
	DatasetSHA   string
	AddendumSHA  string
	RawTreeSHA   string
	ModelIDSHA   string
	TlalocCommit string
}

// RenderR1FReport builds the frozen R1-F markdown report (protocol §8-§20).
func RenderR1FReport(in R1FReportInput) string {
	var b strings.Builder
	p := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	s := in.Summary

	p("# Parrot Perceptual Envelope R1 — Stage R1-F (exact-input repeatability / stability)\n\n")
	p("**Status: R1-F_REPEATABILITY_COMPLETE_FROZEN. R1-G not started.**\n\n")
	p("Model `%s`, temp 0, max 32 tok. Sampling seed: **%s** (the request sends no seed; LM Studio "+
		"exposes no fixed sampling seed — at temp 0 the chain collapses to greedy argmax). %d sentinels "+
		"× 5 new exact repeats = %d calls.\n\n", in.Model, in.Dataset.SamplingSeed, s.Sentinels, s.Calls)
	p("`SENTINEL_POSTHOC_SELECTION_FOR_STABILITY = true`. %s\n\n---\n\n", in.Dataset.Note)

	// §12 global
	p("## 1. Global stability\n\n")
	p("| metric | value |\n|---|--:|\n")
	p("| sentinels with 1 distinct raw output | %d / %d |\n", s.RawDistinct1, s.Sentinels)
	p("| sentinels with 2 distinct raw outputs | %d |\n", s.RawDistinct2)
	p("| sentinels with 3+ distinct raw outputs | %d |\n", s.RawDistinct3Plus)
	p("| semantic outcome invariant 5/5 | %d / %d (%.0f%%) |\n", s.SemanticInvariant5of5, s.Sentinels, s.FracSemanticInvariant*100)
	p("| semantic outcome variable | %d |\n", s.SemanticVariable)
	p("| within-sentinel byte-identical repeat-pair rate | %.3f |\n", s.ByteIdenticalWithinSentinelPairRate)
	p("| BYTE_STABLE / SEMANTICALLY_STABLE / SEMANTICALLY_VARIABLE | %d / %d / %d |\n", s.ByteStable, s.SemanticallyStable, s.SemanticallyVariable)
	p("| any-exact-retry recoveries (source wrong → ≥1 retry correct) | %d |\n", s.AnyExactRetryRecoveries)
	p("| any-exact-retry degradations (source correct → ≥1 retry wrong) | %d |\n\n", s.AnyExactRetryDegradations)

	// §11 stratum table
	p("## 2. Per-stratum aggregates\n\n")
	p("| stratum | name | n | calls | raw-all-identical | mean raw distinct | semantic stability | contract stability | wrong→recovered≥1 | correct→degraded≥1 | mean lat ms |\n")
	p("|---|---|--:|--:|--:|--:|--:|--:|--:|--:|--:|\n")
	for _, a := range s.Strata {
		p("| %s | %s | %d | %d | %.2f | %.2f | %.2f | %.2f | %d | %d | %.0f |\n",
			a.Stratum, a.StratumName, a.Sentinels, a.Calls, a.ExactRawAllIdenticalRate,
			a.MeanRawDistinctOutputs, a.SemanticStabilityRate, a.ContractStabilityRate,
			a.WrongSourceRecoveredAtLeastOnce, a.CorrectSourceDegradedAtLeastOnce, a.MeanLatencyMS)
	}
	p("\n")

	// §8 per-sentinel
	p("## 3. Per-sentinel stability\n\n")
	p("| sentinel | stratum | src cond | base | prev raw | prev sem | raw distinct | raw mode freq | norm distinct | semantic seq | sem flips | H_raw | class | retry |\n")
	p("|---|---|---|---|---|:--:|--:|--:|--:|---|--:|--:|---|---|\n")
	for _, st := range in.Table.Sentinels {
		retry := "—"
		switch {
		case !st.SourceWasCorrect && st.AnyRetryCorrect != nil:
			retry = fmt.Sprintf("any=%v maj=%v allWrong=%v", *st.AnyRetryCorrect, *st.MajorityRetryCorrect, *st.AllRetriesWrong)
		case st.SourceWasCorrect && st.AnyRetryWrong != nil:
			retry = fmt.Sprintf("anyWrong=%v majWrong=%v", *st.AnyRetryWrong, *st.MajorityRetryWrong)
		}
		p("| %s | %s | %s | %s | `%s` | %v | %d | %d | %d | %s | %d | %.2f | %s | %s |\n",
			st.SentinelID, st.Stratum, st.SourceCondition, st.BaseID, truncStr(st.PrevRawOutput, 18),
			st.PrevSemanticCorrect, st.ExactRawDistinctCount, st.ExactRawModeFrequency,
			st.NormalizedOutputDistinctCount, boolSeq(st.SemanticOutcomeSequence), st.SemanticFlipCount,
			st.HRaw, st.OutputStabilityClass, retry)
	}
	p("\n")

	// §13 decision
	p("## 4. Blind-retry decision rule (§13, frozen pre-inference)\n\n")
	p("> %s\n\n", s.DecisionRuleFrozenPreInference)
	p("- previously-wrong sentinels: **%d**; of those, remain wrong on all 5 retries: **%d** (%.0f%%, threshold %.0f%%)\n",
		s.PreviouslyWrongSentinels, s.PreviouslyWrongRemainAllWrong, s.FracWrongRemainWrong*100, r1fWrongStayWrongThreshold*100)
	p("- semantic outcome invariant 5/5: **%d / %d** (%.0f%%, threshold %.0f%%)\n",
		s.SemanticInvariant5of5, s.Sentinels, s.FracSemanticInvariant*100, r1fSemanticInvariantThreshold*100)
	p("\n### `BLIND_RETRY_NOT_USEFUL = %v`\n\n", s.BlindRetryNotUseful)

	// §15 failure-mode questions
	p("## 5. Failure-mode questions (§15)\n\n")
	stratA := stratumAgg(s, "A")
	stratB := stratumAgg(s, "B")
	stratC := stratumAgg(s, "C")
	stratD := stratumAgg(s, "D")
	stratE := stratumAgg(s, "E")
	p("- **A. Are 8 px digit misreads deterministic?** Stratum B: raw-all-identical %.2f, semantic stability %.2f, mean raw distinct %.2f. %s\n",
		stratB.ExactRawAllIdenticalRate, stratB.SemanticStabilityRate, stratB.MeanRawDistinctOutputs, detNote(stratB))
	p("- **B. Is commentary contamination deterministic?** Stratum C: raw-all-identical %.2f, semantic stability %.2f, contract stability %.2f. %s\n",
		stratC.ExactRawAllIdenticalRate, stratC.SemanticStabilityRate, stratC.ContractStabilityRate, detNote(stratC))
	p("- **C. Does distractor capture choose the same wrong number repeatedly?** Stratum D: raw-all-identical %.2f, mean raw distinct %.2f. %s\n",
		stratD.ExactRawAllIdenticalRate, stratD.MeanRawDistinctOutputs, captureNote(in.Table))
	p("- **D. Does hallucination under distractor conditions vary?** Stratum D semantic stability %.2f; %s\n",
		stratD.SemanticStabilityRate, hallucNote(stratD))
	p("- **E. Is the E0 NO_IMAGE \"12345\" collapse byte-stable?** Stratum E: raw-all-identical %.2f, %s\n",
		stratE.ExactRawAllIdenticalRate, e0Note(in.Table))
	p("- **F. Does any exact-input retry provide meaningful recovery?** %d recovery event(s) across %d previously-wrong sentinels; %d degradation event(s) across the %d source-correct sentinels. %s\n\n",
		s.AnyExactRetryRecoveries, s.PreviouslyWrongSentinels, s.AnyExactRetryDegradations, stratA.Sentinels, recoveryNote(s))

	// §20 architectural consequence
	p("## 6. Architectural consequence (§20 — report only, do not build)\n\n")
	if s.BlindRetryNotUseful {
		p("`BLIND_RETRY_NOT_USEFUL = true`. Proposed future Tlaloc rule:\n\n")
		p("> **NEVER repeat an identical failed Parrot call as recovery.** A recovery attempt must change "+
			"something measurable — scale, context, crop, or executor — or return `UNKNOWN`.\n\n")
		p("Blind exact-input retry neither helped (%d/%d recoveries) nor was needed for the successes; "+
			"the model is effectively deterministic under byte-identical input at temp 0.\n\n",
			s.AnyExactRetryRecoveries, s.PreviouslyWrongSentinels)
	} else {
		p("`BLIND_RETRY_NOT_USEFUL = false`. Blind exact-input retry shows some stochastic movement — "+
			"quantify exactly which failure regimes benefit before generalising:\n\n")
		for _, a := range s.Strata {
			if a.WrongSourceRecoveredAtLeastOnce > 0 {
				p("- stratum %s (%s): %d/%d wrong sentinels recovered at least once\n", a.Stratum, a.StratumName, a.WrongSourceRecoveredAtLeastOnce, a.Sentinels)
			}
		}
		p("\nDo not generalise recovery beyond those regimes.\n\n")
	}

	// §19 freeze
	p("## 7. Freeze (§19)\n\n")
	p("`R1F_RECORDS.json` `%s` · `R1F_SENTINEL_TABLE.json` `%s` · `R1F_STABILITY_SUMMARY.json` `%s` · "+
		"sentinels `%s` · dataset `%s` · addendum-07 `%s` · model identity `%s` · raw tree `%s` · commit `%s`.\n\n",
		short(in.RecordsSHA), short(in.TableSHA), short(in.SummarySHA), short(in.SentinelsSHA),
		short(in.DatasetSHA), short(in.AddendumSHA), short(in.ModelIDSHA), short(in.RawTreeSHA), in.TlalocCommit)

	p("## 8. HARD STOP\n\n")
	p("R1-F is complete and frozen. **Do NOT run R1-G.** Return the complete repeatability/stability report for review.\n")
	return b.String()
}

func stratumAgg(s R1FStabilitySummary, key string) R1FStratumAgg {
	for _, a := range s.Strata {
		if a.Stratum == key {
			return a
		}
	}
	return R1FStratumAgg{Stratum: key}
}

func detNote(a R1FStratumAgg) string {
	switch {
	case a.SemanticStabilityRate >= 0.99:
		return "Yes — the semantic outcome is invariant across all 5 repeats for every sentinel."
	case a.SemanticStabilityRate >= 0.5:
		return "Mostly — some sentinels flip semantic outcome on repeat."
	default:
		return "No — the semantic outcome varies on repeat for most sentinels."
	}
}

func captureNote(t R1FSentinelTable) string {
	same, total := 0, 0
	for _, st := range t.Sentinels {
		if st.Stratum != "D" {
			continue
		}
		total++
		if st.NormalizedOutputDistinctCount == 1 {
			same++
		}
	}
	return fmt.Sprintf("%d/%d D sentinels return one normalised value across all 5 repeats.", same, total)
}

func hallucNote(a R1FStratumAgg) string {
	if a.MeanRawDistinctOutputs <= 1.01 {
		return "no — the wrong answer is the same string every time (deterministic capture/hallucination)."
	}
	return "yes — some sentinels produce more than one distinct wrong answer on repeat."
}

func e0Note(t R1FSentinelTable) string {
	allByte, val12345, total := 0, 0, 0
	for _, st := range t.Sentinels {
		if st.Stratum != "E" {
			continue
		}
		total++
		if st.ExactRawAllIdentical {
			allByte++
		}
		if len(st.RawOutputs) > 0 && strings.TrimSpace(st.RawOutputs[0]) == "12345" {
			val12345++
		}
	}
	return fmt.Sprintf("%d/%d E sentinels are byte-stable; %d/%d still emit exactly \"12345\".", allByte, total, val12345, total)
}

func recoveryNote(s R1FStabilitySummary) string {
	if s.AnyExactRetryRecoveries == 0 && s.AnyExactRetryDegradations == 0 {
		return "No — exact-input retry is inert; blind retry cannot help or hurt."
	}
	if s.AnyExactRetryRecoveries > 0 {
		return "Some stochastic recovery exists — see the per-stratum breakdown; do not generalise beyond those regimes."
	}
	return "Retry did not recover any failure; it did move some source-correct sentinels."
}

func truncStr(s string, n int) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "|", "/")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func boolSeq(seq []bool) string {
	var parts []string
	for _, v := range seq {
		if v {
			parts = append(parts, "C")
		} else {
			parts = append(parts, "w")
		}
	}
	return strings.Join(parts, "")
}
