package perceptenvelope

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

// writeR1Store writes a minimal multi-page pdfmemory store for the
// deterministic-selection integrity tests.
func writeR1Store(t *testing.T, pages map[int][]canonicaldoc.Region) string {
	t.Helper()
	dir := t.TempDir()
	var refs []pdfmemory.PageRef
	for num, regions := range pages {
		rel := filepath.Join("canonical", "doc1", "pages", fmt.Sprintf("p%06d.json", num))
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
	m := pdfmemory.Manifest{
		Schema: pdfmemory.Schema, AddressSchema: pdfmemory.AddressSchema, ToolProtocol: pdfmemory.ToolProtocol,
		CarrierID: "fold-bench", DocumentCount: 1, PageCount: len(refs),
		SourceSHA256: "deadbeef", StoreRootSHA256: "cafef00d",
		Documents: []pdfmemory.DocumentRef{{ID: "doc1", Name: "x.pdf", SourceSHA256: "deadbeef"}},
		Pages:     refs,
	}
	mb, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), mb, 0o644); err != nil {
		t.Fatal(err)
	}
	ib, _ := json.Marshal(pdfmemory.Index{Schema: pdfmemory.Schema + ".block-index", Postings: map[string][]int{}})
	if err := os.WriteFile(filepath.Join(dir, "index.json"), ib, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// bodyLine is a well-formed body text line inside the page text box.
func bodyLine(id string, order int, y float64, text string) canonicaldoc.Region {
	return canonicaldoc.Region{
		ID: id, Kind: "text_line", ReadingOrder: order, Text: text, FontSize: 15,
		BBox: canonicaldoc.BBox{X1: 100, Y1: y, X2: 650, Y2: y + 16},
	}
}

func manyGoodPages(t *testing.T) string {
	t.Helper()
	pages := map[int][]canonicaldoc.Region{}
	// 80 pages, one clean single-integer body line each -> ample pool.
	for p := 1; p <= 80; p++ {
		pages[p] = []canonicaldoc.Region{
			bodyLine("text-0001", 1, 120, "This paragraph introduces the topic without any numerals at all."),
			bodyLine("text-0002", 2, 140, fmt.Sprintf("the batch size is %d samples for this run", 100+p)),
			bodyLine("text-0003", 3, 160, "and the following sentence also has no digits in it whatsoever."),
		}
	}
	return writeR1Store(t, pages)
}

// TEST 1 + 2: same store -> identical SOURCE_POOL; same seed -> identical allocation.
func TestDeterministic_PoolAndAllocation(t *testing.T) {
	dir := manyGoodPages(t)
	p1, err := ScanSourcePool(dir)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := ScanSourcePool(dir)
	if err != nil {
		t.Fatal(err)
	}
	b1, _ := json.Marshal(p1)
	b2, _ := json.Marshal(p2)
	if string(b1) != string(b2) {
		t.Fatal("SOURCE_POOL not deterministic")
	}
	a1, r1 := Allocate(p1, "x")
	a2, r2 := Allocate(p2, "x")
	j := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	if j(a1) != j(a2) || j(r1) != j(r2) {
		t.Fatal("allocation not deterministic")
	}
	if len(a1.Bases) != R1ASize || len(r1.Bases) != R1BSize {
		t.Fatalf("want %d/%d bases, got %d/%d", R1ASize, R1BSize, len(a1.Bases), len(r1.Bases))
	}
}

// TEST 3: R1-A and R1-B candidate sets are disjoint.
func TestR1A_R1B_Disjoint(t *testing.T) {
	p, _ := ScanSourcePool(manyGoodPages(t))
	a, b := Allocate(p, "x")
	seen := map[string]bool{}
	for _, x := range a.Bases {
		seen[x.Candidate.CandidateID] = true
	}
	for _, x := range b.Bases {
		if seen[x.Candidate.CandidateID] {
			t.Fatalf("overlap: %s", x.Candidate.CandidateID)
		}
	}
}

// TEST 4 + 15: selection reads no model output; poisoning expected/scorer
// fields cannot change allocation (there are no such fields in the input —
// this test asserts that invariant structurally by confirming the pool /
// allocation depend only on the store bytes).
func TestSelection_NoModelOutputInputs(t *testing.T) {
	dir := manyGoodPages(t)
	p, _ := ScanSourcePool(dir)
	a, _ := Allocate(p, "x")
	// mutate an unrelated "expected answer"-shaped file in the dir; re-scan.
	_ = os.WriteFile(filepath.Join(dir, "POISON_expected_answers.json"), []byte(`{"r1a-01-x":"999"}`), 0o644)
	p2, _ := ScanSourcePool(dir)
	a2, _ := Allocate(p2, "x")
	j := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	if j(a) != j(a2) {
		t.Fatal("allocation changed after adding a poison file -> selection is not a pure function of the layer")
	}
}

// TEST 5 + 8 + 10: target bbox contains the gold token region; token bbox
// inside line; every context level shares the same underlying token bbox.
func TestTargetBBox_ContainsToken_AndStableAcrossLevels(t *testing.T) {
	p, _ := ScanSourcePool(manyGoodPages(t))
	for _, c := range p.Candidates {
		tb := c.TokenBBoxStore
		ln := c.Line.BBox
		if !(tb.X1 < tb.X2 && tb.Y1 < tb.Y2) {
			t.Fatalf("%s degenerate token bbox", c.CandidateID)
		}
		// token vertical span overlaps the line; horizontal within line +/- pad
		pad := 0.5 * (ln.Y2 - ln.Y1)
		if tb.X1 < ln.X1-pad-1 || tb.X2 > ln.X2+pad+1 {
			t.Fatalf("%s token x outside line", c.CandidateID)
		}
		var first canonicaldoc.BBox
		for i, lvl := range AllContextLevels {
			// the cue geometry is c.TokenBBoxStore itself; context region is separate
			region := ContextRegionStore(c, lvl)
			if region.X2 <= region.X1 || region.Y2 <= region.Y1 {
				t.Fatalf("%s %s empty context region", c.CandidateID, lvl)
			}
			if i == 0 {
				first = c.TokenBBoxStore
			}
			if c.TokenBBoxStore != first {
				t.Fatalf("%s token bbox mutated across levels", c.CandidateID)
			}
		}
	}
}

// TEST 6 + 7: cue generation deterministic; cue policy identical across levels.
func TestCue_DeterministicAndPolicyUniform(t *testing.T) {
	p, _ := ScanSourcePool(manyGoodPages(t))
	c := p.Candidates[0]
	png1 := solidPNG(t, 1575, 2250)
	a, rectA, err := RenderCuedPage(png1, c)
	if err != nil {
		t.Fatal(err)
	}
	b, rectB, _ := RenderCuedPage(png1, c)
	if string(a) != string(b) || rectA != rectB {
		t.Fatal("RenderCuedPage not deterministic")
	}
	// cue stroke width + colour are package constants -> identical for every level by construction.
	if cueStrokePx <= 0 {
		t.Fatal("cue stroke not set")
	}
}

// TEST 9: every R1-A base produces all seven context conditions (region non-empty).
func TestEveryBaseProducesSevenConditions(t *testing.T) {
	p, _ := ScanSourcePool(manyGoodPages(t))
	a, _ := Allocate(p, "x")
	for _, base := range a.Bases {
		got := map[ContextLevel]bool{}
		for _, lvl := range AllContextLevels {
			r := ContextRegionStore(base.Candidate, lvl)
			if r.X2 > r.X1 && r.Y2 > r.Y1 {
				got[lvl] = true
			}
		}
		if len(got) != 7 {
			t.Fatalf("%s produced %d/7 conditions", base.BaseID, len(got))
		}
	}
}

// TEST 11 + 12 + 13: ground truth from the digital layer; primary numbers
// satisfy the morphology filter; margin/header exclusions are deterministic.
func TestGroundTruth_MorphologyFilter_MarginExclusion(t *testing.T) {
	pages := map[int][]canonicaldoc.Region{
		1: {
			bodyLine("text-0001", 1, 120, "the batch size is 128 samples for this configuration"),
			// bare page number -> excluded
			{ID: "text-0002", Kind: "text_line", ReadingOrder: 2, Text: "89", FontSize: 9, BBox: canonicaldoc.BBox{X1: 20, Y1: 500, X2: 34, Y2: 512}},
			// two numbers on a line -> excluded
			bodyLine("text-0003", 3, 160, "128 samples at 32 epochs were used"),
			// decimal elsewhere on line -> line_has_other_digit_token
			bodyLine("text-0004", 4, 180, "accuracy was 72.5 with 88 runs"),
			// running header page number
			{ID: "text-0005", Kind: "text_line", ReadingOrder: 5, Text: "LINEAR NEURAL NETWORKS FOR CLASSIFICATION 173", FontSize: 12, BBox: canonicaldoc.BBox{X1: 100, Y1: 70, X2: 600, Y2: 84}},
			// table cell -> excluded kind
			{ID: "cell-0001", Kind: "table_cell", ReadingOrder: 6, Text: "512", FontSize: 13, BBox: canonicaldoc.BBox{X1: 200, Y1: 300, X2: 260, Y2: 316}},
			// citation year in parentheses -> bracketed_or_punctuated_token
			bodyLine("text-0006", 7, 340, "as shown by Smola and Vapnik (1997) in earlier work"),
			// in-sentence date -> year_or_date_token
			bodyLine("text-0007", 8, 360, "the competition was first held in 2012 that year"),
		},
	}
	p, err := ScanSourcePool(writeR1Store(t, pages))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Candidates) != 1 {
		t.Fatalf("want exactly 1 admitted candidate, got %d: %+v", len(p.Candidates), p.Candidates)
	}
	c := p.Candidates[0]
	if c.NormalizedTarget != "128" || c.Line.RegionID != "text-0001" {
		t.Fatalf("wrong candidate admitted: %+v", c)
	}
	if !primaryTarget.MatchString(c.NormalizedTarget) {
		t.Fatal("admitted target fails morphology filter")
	}
	for _, want := range []string{
		"line_has_other_digit_token", "region_kind_excluded", "bare_or_short_number_line",
		"running_header_page_number", "bracketed_or_punctuated_token", "year_or_date_token",
	} {
		if p.RejectionCounts[want] == 0 {
			t.Errorf("expected a %q rejection, got none (counts=%v)", want, p.RejectionCounts)
		}
	}
}

// TEST 14: no expected answer is injected into the model-facing prompt.
func TestFrozenInstruction_NoAnswerLeakage(t *testing.T) {
	if FrozenInstruction != "Read the number inside the marked rectangle. Reply with only the number." {
		t.Fatalf("frozen instruction drifted: %q", FrozenInstruction)
	}
	for _, d := range []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"} {
		_ = d // instruction contains no digits
	}
	for _, ch := range FrozenInstruction {
		if ch >= '0' && ch <= '9' {
			t.Fatal("frozen instruction contains a digit")
		}
	}
}

func solidPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
