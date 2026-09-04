package tonalt1

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

// --- test store helpers ---------------------------------------------------

func writeStore(t *testing.T, pages map[int][]canonicaldoc.Region) string {
	t.Helper()
	dir := t.TempDir()
	var refs []pdfmemory.PageRef
	for num, regions := range pages {
		rel := filepath.Join("canonical", "doc1", "pages", fmt.Sprintf("%06d.json", num))
		layout := canonicaldoc.Page{Number: num, Width: 756, Height: 1080, ExtractionMode: "digital", Regions: regions}
		body, _ := json.MarshalIndent(layout, "", "  ")
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, rel), body, 0o644); err != nil {
			t.Fatal(err)
		}
		refs = append(refs, pdfmemory.PageRef{
			DocID: "doc1", Number: num,
			Address:    fmt.Sprintf("ohf://fold-bench/pages/%06d", num),
			LayoutPath: filepath.ToSlash(rel), ExtractionMode: "digital",
		})
	}
	manifest := pdfmemory.Manifest{
		Schema: pdfmemory.Schema, AddressSchema: pdfmemory.AddressSchema, ToolProtocol: pdfmemory.ToolProtocol,
		CarrierID: "fold-bench", DocumentCount: 1, PageCount: len(refs),
		SourceSHA256: "deadbeef", StoreRootSHA256: "cafef00d",
		Documents: []pdfmemory.DocumentRef{{ID: "doc1", Name: "x.pdf", SourceSHA256: "deadbeef"}},
		Pages:     refs,
	}
	mb, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), mb, 0o644); err != nil {
		t.Fatal(err)
	}
	ib, _ := json.Marshal(pdfmemory.Index{Schema: pdfmemory.Schema + ".block-index", Postings: map[string][]int{}})
	if err := os.WriteFile(filepath.Join(dir, "index.json"), ib, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func proseLine(id string, order int, y float64, text string) canonicaldoc.Region {
	return canonicaldoc.Region{
		ID: id, Kind: "text_line", ReadingOrder: order, Text: text, FontSize: 15,
		BBox: canonicaldoc.BBox{X1: 100, Y1: y, X2: 650, Y2: y + 16},
	}
}

func scanOne(t *testing.T, region canonicaldoc.Region) Candidate {
	t.Helper()
	dir := writeStore(t, map[int][]canonicaldoc.Region{7: {region}})
	result, err := Scan(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("want 1 candidate, got %d", len(result.Candidates))
	}
	return result.Candidates[0]
}

// --- A. scanner determinism --------------------------------------------

func TestScan_Deterministic(t *testing.T) {
	pages := map[int][]canonicaldoc.Region{
		7:  {proseLine("text-1", 1, 200, "the model was trained on 512 tokens per batch here")},
		11: {proseLine("text-2", 1, 300, "a learning rate of 0.01 works well in most practical cases")},
	}
	dir := writeStore(t, pages)
	first, err := Scan(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Scan(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if hashJSON(first.Candidates) != hashJSON(second.Candidates) {
		t.Fatal("scan not deterministic: candidate hash differs between runs")
	}
	if len(first.Candidates) != 2 {
		t.Fatalf("want 2 candidates, got %d", len(first.Candidates))
	}
	for index := 1; index < len(first.Candidates); index++ {
		if first.Candidates[index-1].CandidateID >= first.Candidates[index].CandidateID {
			t.Fatal("candidates not sorted by candidate_id")
		}
	}
}

// --- B. physical identity ---------------------------------------------

func TestCandidateID_StableAcrossProcessOrder(t *testing.T) {
	region := proseLine("text-9", 3, 400, "we set the hidden size to 256 for all layers in the encoder")
	a := scanOne(t, region)
	b := scanOne(t, region)
	if a.CandidateID != b.CandidateID {
		t.Fatalf("candidate id unstable: %s vs %s", a.CandidateID, b.CandidateID)
	}
	if a.CandidateID != deriveCandidateID(a) {
		t.Fatal("candidate id does not re-derive from its own physical identity")
	}
}

func TestBBoxAndSpanMatchers(t *testing.T) {
	box := canonicaldoc.BBox{X1: 10, Y1: 20, X2: 30, Y2: 40}
	if !bboxAlmostEqual(box, canonicaldoc.BBox{X1: 10.4, Y1: 20, X2: 30, Y2: 39.6}, 1.0) {
		t.Fatal("bboxAlmostEqual should tolerate sub-unit drift")
	}
	if bboxAlmostEqual(box, canonicaldoc.BBox{X1: 12, Y1: 20, X2: 30, Y2: 40}, 1.0) {
		t.Fatal("bboxAlmostEqual should reject a 2-unit x shift")
	}
	if !spansOverlap(5, 10, 8, 12) || spansOverlap(5, 10, 10, 12) {
		t.Fatal("spansOverlap boundary handling wrong")
	}
}

// --- C. exclusion union ---------------------------------------------

func TestPriorUse_UnionAnyKeyExcludes(t *testing.T) {
	index := &PriorUseIndex{
		byPage: map[int][]PriorInstance{}, regionByPage: map[int]map[string][]PriorInstance{},
		pageExposure: map[int][]string{}, lineIDByPage: map[int]map[string][]PriorInstance{},
		candByID: map[string][]PriorInstance{}, SourceCounts: map[string]int{}, KeyAvailability: map[string]map[string]int{},
	}
	// One prior instance identified only by (page, region_id).
	index.add(PriorInstance{Experiment: "R1-X", Page: 7, RegionID: "text-1"})
	// A different prior instance on the same page, identical containing line.
	index.add(PriorInstance{Experiment: "R1-Y", Page: 7, LineText: "the model was trained on 512 tokens per batch here",
		LineBBox: canonicaldoc.BBox{X1: 100, Y1: 200, X2: 650, Y2: 216}, HasLineBBox: true})

	dir := writeStore(t, map[int][]canonicaldoc.Region{7: {proseLine("text-1", 1, 200, "the model was trained on 512 tokens per batch here")}})
	result, err := Scan(dir, index)
	if err != nil {
		t.Fatal(err)
	}
	cand := result.Candidates[0]
	if !cand.PriorUse.Excluded {
		t.Fatal("candidate should be prior-used excluded")
	}
	keys := map[string]bool{}
	exps := map[string]bool{}
	for _, match := range cand.PriorUse.Matches {
		keys[match.Key] = true
		exps[match.Experiment] = true
	}
	if !keys["page+region_id"] || !keys["page+bbox"] || !keys["span_hash"] {
		t.Fatalf("expected union of region_id + bbox + span_hash keys, got %v", keys)
	}
	if !exps["R1-X"] || !exps["R1-Y"] {
		t.Fatalf("expected both prior experiments preserved, got %v", exps)
	}
}

func TestPriorUse_PageVisualExposure(t *testing.T) {
	root := repoRoot(t)
	index, err := LoadPriorUseIndex(root)
	if err != nil {
		t.Skipf("prior artifacts unavailable: %v", err)
	}
	cand := Candidate{Corpus: CandidateCorpus{Page: 43}, Identity: PhysicalIdentity{Page: 43}}
	matches := index.Match(cand)
	if len(matches) == 0 || matches[0].Key != "page_visual_exposure" {
		t.Fatalf("page 43 should be excluded by page_visual_exposure, got %v", matches)
	}
	// A page not in any prior set must not match.
	if got := index.Match(Candidate{Corpus: CandidateCorpus{Page: 999999}}); len(got) != 0 {
		t.Fatalf("page 999999 should not match any prior use, got %v", got)
	}
}

// --- D. span normalization ------------------------------------------

func TestSpanNormalization(t *testing.T) {
	if normalizeLineText("  The   QUICK\tbrown  ") != "the quick brown" {
		t.Fatalf("whitespace/case folding wrong: %q", normalizeLineText("  The   QUICK\tbrown  "))
	}
	if normalizeLineText("a–b") != normalizeLineText("a-b") {
		t.Fatal("en-dash should fold to hyphen")
	}
	// Same line text, different pages -> different identity.
	if lineIdentityHash(10, "value is 512") == lineIdentityHash(11, "value is 512") {
		t.Fatal("identical text on different pages must not collide")
	}
	// Same page, same text -> stable.
	if lineIdentityHash(10, "value is 512") != lineIdentityHash(10, "value is 512") {
		t.Fatal("span hash not deterministic")
	}
}

// --- E. envelope ---------------------------------------------------

func TestEnvelope_MorphologyAdmission(t *testing.T) {
	cases := []struct {
		text         string
		wantEligible bool
		wantCode     RejectionCode
	}{
		{"the batch size was set to 512 for the whole training run here", true, ""},
		{"a dropout rate of 0.25 was applied to every hidden layer in turn", true, ""},
		{"the year 1998 was when this particular architecture first appeared", false, RejectYearOrDateToken},
		{"there were only 7 examples left in the held out evaluation split", false, RejectUnsupportedMorphology},
		{"we observed between 5998 and 6008 activations across the whole run", false, RejectMultipleNumericTokens},
		{"the corpus grew by 40% during the second pretraining phase overall", false, RejectUnsupportedMorphology},
	}
	for _, testCase := range cases {
		cand := scanOne(t, proseLine("text-1", 1, 300, testCase.text))
		if cand.Eligibility.Eligible != testCase.wantEligible {
			t.Errorf("%q: eligible=%v want %v (codes %v)", testCase.text, cand.Eligibility.Eligible, testCase.wantEligible, cand.Eligibility.RejectionCodes)
			continue
		}
		if testCase.wantCode != "" && !contains(cand.Eligibility.RejectionCodes, testCase.wantCode) {
			t.Errorf("%q: want code %s, got %v", testCase.text, testCase.wantCode, cand.Eligibility.RejectionCodes)
		}
	}
}

// --- F. geometry / ambiguity --------------------------------------

func TestGeometry_Rules(t *testing.T) {
	// competing numeric tokens on the line.
	multi := scanOne(t, proseLine("text-1", 1, 300, "the layer had 128 units and 256 units in two separate configs here"))
	if !contains(multi.Eligibility.RejectionCodes, RejectMultipleNumericTokens) {
		t.Errorf("multi-number line: want REJECT_MULTIPLE_NUMERIC_TOKENS, got %v", multi.Eligibility.RejectionCodes)
	}
	// number-leading line.
	leading := scanOne(t, proseLine("text-1", 1, 300, "512 hidden units were used across every block of the network here"))
	if !contains(leading.Eligibility.RejectionCodes, RejectNumberLeadingLine) {
		t.Errorf("number-leading: want REJECT_NUMBER_LEADING_LINE, got %v", leading.Eligibility.RejectionCodes)
	}
	// cross-reference.
	xref := scanOne(t, proseLine("text-1", 1, 300, "as we discussed at length in Section 3.6 the model tends to overfit"))
	if !contains(xref.Eligibility.RejectionCodes, RejectCrossReference) {
		t.Errorf("cross-ref: want REJECT_CROSS_REFERENCE, got %v", xref.Eligibility.RejectionCodes)
	}
	// bare / short numeric line.
	bare := scanOne(t, canonicaldoc.Region{ID: "text-1", Kind: "text_line", ReadingOrder: 1, Text: "512", FontSize: 15,
		BBox: canonicaldoc.BBox{X1: 100, Y1: 300, X2: 130, Y2: 316}})
	if bare.Eligibility.Eligible {
		t.Errorf("bare numeric line should be rejected, got %v", bare.Eligibility.RejectionCodes)
	}
	// margin line number (font < 10).
	margin := scanOne(t, canonicaldoc.Region{ID: "text-1", Kind: "text_line", ReadingOrder: 1, Text: "173", FontSize: 9,
		BBox: canonicaldoc.BBox{X1: 60, Y1: 400, X2: 74, Y2: 410}})
	if margin.Eligibility.Eligible {
		t.Errorf("margin line number should be rejected, got %v", margin.Eligibility.RejectionCodes)
	}
	// table cell region kind.
	table := scanOne(t, canonicaldoc.Region{ID: "text-1", Kind: "table_cell", ReadingOrder: 1, Text: "the value 512 appears in this particular cell of the results table", FontSize: 15,
		BBox: canonicaldoc.BBox{X1: 100, Y1: 400, X2: 650, Y2: 416}})
	if !contains(table.Eligibility.RejectionCodes, RejectRegionKind) {
		t.Errorf("table_cell: want REJECT_REGION_KIND, got %v", table.Eligibility.RejectionCodes)
	}
}

// --- G/H. serialization + hashing stability -----------------------

func TestArtifactHashingStable(t *testing.T) {
	pages := map[int][]canonicaldoc.Region{
		7: {proseLine("text-1", 1, 200, "we used a threshold of 0.35 to filter low confidence detections here")},
	}
	dir := writeStore(t, pages)
	scanA, _ := Scan(dir, nil)
	scanB, _ := Scan(dir, nil)
	if hashJSON(scanA.Candidates) != hashJSON(scanB.Candidates) {
		t.Fatal("candidate serialization hash unstable")
	}
	// hashJSON is a pure function of value.
	if hashJSON(scanA) != hashJSON(scanA) {
		t.Fatal("hashJSON not pure")
	}
}

// --- I/J/K. full-pipeline integration on the real corpus ----------

func TestRunD3_RealCorpus(t *testing.T) {
	root := repoRoot(t)
	storeDir := filepath.Join(root, "experiments/exocortex-decomposition-r0/stores/fold-bench-reconstructed-r0")
	if _, err := os.Stat(filepath.Join(storeDir, "manifest.json")); err != nil {
		t.Skipf("canonical store unavailable: %v", err)
	}

	first, err := RunD3(root, storeDir)
	if err != nil {
		t.Fatalf("RunD3: %v", err)
	}
	second, err := RunD3(root, storeDir)
	if err != nil {
		t.Fatalf("RunD3 (2nd): %v", err)
	}

	// K. rerun determinism.
	for name, hash := range first.Freeze.ArtifactHashes {
		if second.Freeze.ArtifactHashes[name] != hash {
			t.Errorf("determinism: %s hash differs between runs", name)
		}
	}
	if hashJSON(first.Freeze) != hashJSON(second.Freeze) {
		t.Error("determinism: freeze manifest differs between runs")
	}

	// I. no duplicate eligible physical instances.
	seen := map[string]bool{}
	for _, cand := range first.Eligible {
		if cand.Identity.NormalizedSpanHash == "" {
			t.Errorf("eligible candidate %s has no span hash", cand.CandidateID)
		}
		if seen[cand.Identity.NormalizedSpanHash] {
			t.Errorf("duplicate eligible physical instance: %s", cand.Identity.NormalizedSpanHash)
		}
		seen[cand.Identity.NormalizedSpanHash] = true
	}

	// J. no prior-used instance in the eligible set + envelope + geometry.
	for _, cand := range first.Eligible {
		if cand.PriorUse.Excluded {
			t.Errorf("prior-used candidate %s leaked into eligible set", cand.CandidateID)
		}
		if !cand.Presentation.R1EnvelopeSupported {
			t.Errorf("eligible candidate %s is not R1-envelope supported", cand.CandidateID)
		}
		if cand.Presentation.CompetingNumericCount != 0 {
			t.Errorf("eligible candidate %s has competing numeric tokens", cand.CandidateID)
		}
		if cand.Presentation.MorphologyFamily != MorphMultiDigitInteger && cand.Presentation.MorphologyFamily != MorphDecimal {
			t.Errorf("eligible candidate %s has non-admissible morphology %s", cand.CandidateID, cand.Presentation.MorphologyFamily)
		}
	}

	// Hard invariants must all hold (scan integrity), independent of
	// downstream allocation feasibility.
	for name, ok := range first.Freeze.HardInvariants {
		if !ok {
			t.Errorf("hard invariant %s failed", name)
		}
	}
	if !first.Freeze.TONALT1D3Frozen {
		t.Error("TONAL_T1_D3_FROZEN should be true when all hard invariants hold")
	}
	if first.Stats.PagesScanned != 1152 {
		t.Errorf("pages scanned = %d, want 1152", first.Stats.PagesScanned)
	}

	t.Logf("D3 real-corpus: scan_total=%d prior_excluded=%d envelope_rejected=%d geometry_rejected=%d eligible=%d demand=%d feasible=%v",
		first.Stats.ScanTotal, first.Stats.PriorPhysicalIdentityExcluded, first.Stats.R1EnvelopeRejected,
		first.Stats.GeometryRejected, first.Stats.FinalHeldOutAvailable, first.Stats.RequiredUniqueOperandDemand,
		first.Stats.AllocationFeasible)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	// tests run in internal/tonalt1; behavior-lab root is two levels up.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
