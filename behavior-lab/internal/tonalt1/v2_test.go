package tonalt1

import (
	"testing"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
)

// D3 v2: authoring-heuristic rules are advisory, not blocking.
func TestV2_AuthoringHeuristicsAdvisory(t *testing.T) {
	// A short (2-field) prose line with a bare multi-digit integer in the
	// left margin at small font: v1 rejected on 4 authoring rules; v2
	// keeps it eligible and records the tags.
	region := canonicaldoc.Region{
		ID: "text-1", Kind: "text_line", ReadingOrder: 1, FontSize: 9,
		Text: "about 512", // number not leading, 2 fields
		BBox: canonicaldoc.BBox{X1: 60, Y1: 400, X2: 130, Y2: 409},
	}
	cand := scanOne(t, region)
	if !cand.Eligibility.Eligible {
		t.Fatalf("v2 should keep this eligible; blocking codes present: %v", cand.Eligibility.RejectionCodes)
	}
	tagSet := map[RejectionCode]bool{}
	for _, tag := range cand.Eligibility.AdvisoryTags {
		tagSet[tag] = true
	}
	for _, want := range []RejectionCode{RejectFontBelowBody, RejectLineInPageMargin, RejectLineTooNarrow, RejectBareOrShortNumberLine} {
		if !tagSet[want] {
			t.Errorf("expected advisory tag %s, tags=%v", want, cand.Eligibility.AdvisoryTags)
		}
		if blocks(want) {
			t.Errorf("%s must not be a blocking rule in v2", want)
		}
	}
}

// D3 v2: DOMAIN rules still block.
func TestV2_DomainRulesBlock(t *testing.T) {
	cases := []struct {
		text string
		want RejectionCode
	}{
		{"we used the CUDA 11 toolkit for every training run in this chapter", RejectVersionString},
		{"512 hidden units were used across every layer of the whole network", RejectNumberLeadingLine},
		{"tion 12 introduced the encoder decoder attention mechanism in detail", RejectWrappedFragment},
	}
	for _, testCase := range cases {
		cand := scanOne(t, proseLine("text-1", 1, 300, testCase.text))
		if cand.Eligibility.Eligible {
			t.Errorf("%q: should be blocked by %s", testCase.text, testCase.want)
		}
		if !contains(cand.Eligibility.RejectionCodes, testCase.want) {
			t.Errorf("%q: want %s, got %v", testCase.text, testCase.want, cand.Eligibility.RejectionCodes)
		}
		if !blocks(testCase.want) {
			t.Errorf("%s must be a blocking DOMAIN rule", testCase.want)
		}
	}
}

// D3 v2: a genuine number-leading quantity sentence is still blocked
// (DOMAIN — the corpus's number-leading lines are overwhelmingly TOC /
// heading section numbers, and there is no structural way to tell the two
// apart), and a lone bare number line is blocked.
func TestV2_LoneAndPageArtifact(t *testing.T) {
	lone := scanOne(t, canonicaldoc.Region{
		ID: "text-1", Kind: "text_line", ReadingOrder: 1, FontSize: 15,
		Text: "512", BBox: canonicaldoc.BBox{X1: 100, Y1: 500, X2: 130, Y2: 515},
	})
	if lone.Eligibility.Eligible || !contains(lone.Eligibility.RejectionCodes, RejectLoneNumberLine) {
		t.Errorf("lone number line: want REJECT_LONE_NUMBER_NO_CONTEXT, got %v", lone.Eligibility.RejectionCodes)
	}
	header := scanOne(t, canonicaldoc.Region{
		ID: "text-1", Kind: "text_line", ReadingOrder: 1, FontSize: 10,
		Text: "173 chapter", BBox: canonicaldoc.BBox{X1: 100, Y1: 20, X2: 200, Y2: 32},
	})
	if !contains(header.Eligibility.RejectionCodes, RejectPageHeaderFooter) {
		t.Errorf("top-of-page short line: want REJECT_PAGE_HEADER_OR_FOOTER, got %v", header.Eligibility.RejectionCodes)
	}
}

// Rule provenance table covers every rejection code exactly once.
func TestV2_RuleProvenanceComplete(t *testing.T) {
	all := []RejectionCode{
		RejectParseFailure, RejectUnsupportedMorphology, RejectYearOrDateToken, RejectRegionKind,
		RejectMultipleNumericTokens, RejectOperandNotIncluded, RejectGeometryAmbiguous, RejectGeometryMalformed,
		RejectLineInPageMargin, RejectLineTooNarrow, RejectBareOrShortNumberLine, RejectNumberLeadingLine,
		RejectCrossReference, RejectBibliographyLine, RejectFontBelowBody, RejectRunningHeader,
		RejectCueImplausible, RejectPaddedBoxClipped, RejectTokenOffsetNotUnique, RejectPriorUsed,
		RejectDomainInvalid, RejectVersionString, RejectWrappedFragment, RejectPageHeaderFooter, RejectLoneNumberLine,
	}
	for _, code := range all {
		if _, ok := ruleClassOf[code]; !ok {
			t.Errorf("rejection code %s has no rule-class classification", code)
		}
	}
}

// SINGLE_DIGIT held-out enumeration: nothing leaks, deterministic, digits
// stratified.
func TestV2_SingleDigitHeldOut(t *testing.T) {
	root := repoRoot(t)
	storeDir := root + "/experiments/exocortex-decomposition-r0/stores/fold-bench-reconstructed-r0"
	first, err := EnumerateSingleDigitHeldOut(root, storeDir, 60)
	if err != nil {
		t.Skipf("store unavailable: %v", err)
	}
	if !first.HeldOutExclusionOK {
		t.Fatal("held-out exclusion failed: a selected SINGLE_DIGIT base collides with a prior-used instance")
	}
	if first.AvailableN < 60 {
		t.Logf("note: only %d held-out SINGLE_DIGIT available", first.AvailableN)
	}
	if len(first.Bases) < 6 {
		t.Fatalf("only %d SINGLE_DIGIT bases; need >= 6", len(first.Bases))
	}
	second, _ := EnumerateSingleDigitHeldOut(root, storeDir, 60)
	if len(first.Bases) != len(second.Bases) {
		t.Fatal("SINGLE_DIGIT enumeration not deterministic")
	}
	for i := range first.Bases {
		if first.Bases[i].CandidateID != second.Bases[i].CandidateID {
			t.Fatal("SINGLE_DIGIT enumeration order not deterministic")
		}
	}
	// admitSingleDigit must be reset.
	if admitSingleDigit {
		t.Fatal("admitSingleDigit left true after enumeration")
	}
	// The default scan must still exclude SINGLE_DIGIT.
	scan, _ := Scan(storeDir, nil)
	for _, cand := range scan.Candidates {
		if cand.Presentation.MorphologyFamily == MorphSingleDigit {
			t.Fatal("default D3 scan admitted SINGLE_DIGIT")
		}
	}
}
