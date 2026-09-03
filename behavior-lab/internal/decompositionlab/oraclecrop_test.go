package decompositionlab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

// writeMiniStore writes the minimal pdfmemory store pdfmemory.Load +
// oraclecrop.storePage need: a manifest with one page pointing at a
// canonical layout file, and an (empty) index.
func writeMiniStore(t *testing.T, page int, w, h float64, regions []canonicaldoc.Region) string {
	t.Helper()
	dir := t.TempDir()
	layoutRel := filepath.Join("canonical", "doc1", "pages", "p.json")
	layout := canonicaldoc.Page{Number: page, Address: "", Width: w, Height: h, ExtractionMode: "digital", Regions: regions, TextChars: 100}
	body, err := json.MarshalIndent(layout, "", "  ")
	if err != nil {
		t.Fatalf("marshal layout: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(layoutRel)), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, layoutRel), body, 0o644); err != nil {
		t.Fatalf("write layout: %v", err)
	}
	manifest := pdfmemory.Manifest{
		Schema: pdfmemory.Schema, AddressSchema: pdfmemory.AddressSchema, ToolProtocol: pdfmemory.ToolProtocol,
		CarrierID: "fold-bench", DocumentCount: 1, PageCount: 1,
		Pages: []pdfmemory.PageRef{{
			DocID: "doc1", Number: page,
			Address:    fmt.Sprintf("ohf://fold-bench/pages/%06d", page),
			LayoutPath: filepath.ToSlash(layoutRel), ExtractionMode: "digital",
		}},
	}
	mb, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), mb, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	ib, _ := json.Marshal(pdfmemory.Index{Schema: pdfmemory.Schema + ".block-index", Postings: map[string][]int{}})
	if err := os.WriteFile(filepath.Join(dir, "index.json"), ib, 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	return dir
}

func numericFixtureRegions() []canonicaldoc.Region {
	return []canonicaldoc.Region{
		{ID: "text-0001", Kind: "text_line", ReadingOrder: 1, Text: "Some earlier unrelated line of body text on the page.", BBox: canonicaldoc.BBox{X1: 40, Y1: 40, X2: 700, Y2: 60}},
		{ID: "text-0002", Kind: "text_line", ReadingOrder: 2, Text: "Every second, the microphone will col-", BBox: canonicaldoc.BBox{X1: 40, Y1: 100, X2: 500, Y2: 120}},
		{ID: "text-0003", Kind: "text_line", ReadingOrder: 3, Text: "lect roughly 44,000 samples.", BBox: canonicaldoc.BBox{X1: 40, Y1: 122, X2: 380, Y2: 142}},
		{ID: "text-0004", Kind: "text_line", ReadingOrder: 4, Text: "A later unrelated paragraph continues here for a while.", BBox: canonicaldoc.BBox{X1: 40, Y1: 200, X2: 720, Y2: 220}},
	}
}

func numericFixtureRecord(store string) P0Record {
	return P0Record{
		BaseID: "e2e-numeric-01", Category: CategoryNumeric, TaskFamily: "numeric",
		DocID: "fold-bench", Page: 43, PageImagePath: filepath.Join(store, "img.png"),
		PageWidth: 1050, PageHeight: 1500, ExpectedAnswer: "44,000",
		EvidenceAddress: "ohf://fold-bench/pages/000043",
		EvidenceRefs: []EvidenceRef{{
			ID: "ev1", DocID: "fold-bench", Page: 43,
			TextSpan: "Every second, the microphone will col- lect roughly 44,000 samples.",
		}},
		Recipe: []AtomicStep{
			{ID: "extract", Opcode: "EXTRACT_NUMBER", EvidenceRefs: []string{"ev1"}, OutputKey: "raw"},
			{ID: "normalize", Opcode: "NORMALIZE", OutputKey: "answer", Deterministic: true},
		},
	}
}

func TestDeriveOracleGeometry_Deterministic(t *testing.T) {
	store := writeMiniStore(t, 43, 756, 1080, numericFixtureRegions())
	rec := numericFixtureRecord(store)
	a, err := DeriveOracleGeometry(store, rec.Page, rec.EvidenceRefs, rec.PageWidth, rec.PageHeight)
	if err != nil {
		t.Fatalf("derive a: %v", err)
	}
	b, err := DeriveOracleGeometry(store, rec.Page, rec.EvidenceRefs, rec.PageWidth, rec.PageHeight)
	if err != nil {
		t.Fatalf("derive b: %v", err)
	}
	if a.StoreBBox != b.StoreBBox || a.ImageBBox != b.ImageBBox {
		t.Fatalf("non-deterministic geometry: %+v vs %+v", a, b)
	}
}

func TestDeriveOracleCrop_IndependentOfExpectedAnswer(t *testing.T) {
	store := writeMiniStore(t, 43, 756, 1080, numericFixtureRegions())
	rec := numericFixtureRecord(store)
	clean, err := DeriveOracleCrop(store, rec)
	if err != nil {
		t.Fatalf("clean: %v", err)
	}
	poisoned := rec
	poisoned.ExpectedAnswer = "999999999"
	got, err := DeriveOracleCrop(store, poisoned)
	if err != nil {
		t.Fatalf("poisoned: %v", err)
	}
	if got.StoreBBox != clean.StoreBBox || got.ImageBBox != clean.ImageBBox {
		t.Fatalf("poisoning expected_answer changed the frozen oracle bbox: %+v vs %+v", clean.ImageBBox, got.ImageBBox)
	}
}

func TestDeriveOracleGeometry_ScaleAndBounds(t *testing.T) {
	store := writeMiniStore(t, 43, 756, 1080, numericFixtureRegions())
	rec := numericFixtureRecord(store)
	spec, err := DeriveOracleGeometry(store, rec.Page, rec.EvidenceRefs, rec.PageWidth, rec.PageHeight)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if spec.ScaleX <= 1.0 || spec.ScaleY <= 1.0 {
		t.Fatalf("expected upscale store 756x1080 -> image 1050x1500, got sx=%v sy=%v", spec.ScaleX, spec.ScaleY)
	}
	b := spec.ImageBBox
	if b.X1 < 0 || b.Y1 < 0 || b.X2 > rec.PageWidth || b.Y2 > rec.PageHeight || b.X2 <= b.X1 || b.Y2 <= b.Y1 {
		t.Fatalf("image bbox out of bounds: %+v (image %vx%v)", b, rec.PageWidth, rec.PageHeight)
	}
	// The evidence line (store y ~100-142) must remain inside the scaled crop.
	evYtop := 100 * spec.ScaleY
	evYbot := 142 * spec.ScaleY
	if evYtop < b.Y1 || evYbot > b.Y2 {
		t.Fatalf("scaled crop [%v,%v] no longer contains the evidence rows [%v,%v]", b.Y1, b.Y2, evYtop, evYbot)
	}
	full := rec.PageWidth * rec.PageHeight
	area := (b.X2 - b.X1) * (b.Y2 - b.Y1)
	if area/full >= wholePageExposureThreshold {
		t.Fatalf("derived oracle crop is effectively the whole page: exposure %v", area/full)
	}
	if spec.Verdict != OracleCompatible {
		t.Fatalf("verdict = %s, want COMPATIBLE; notes=%s", spec.Verdict, spec.Notes)
	}
}

func TestDeriveOracleGeometry_InsufficientProvenance(t *testing.T) {
	store := writeMiniStore(t, 43, 756, 1080, numericFixtureRegions())
	rec := numericFixtureRecord(store)
	rec.EvidenceRefs = []EvidenceRef{{ID: "ev1", DocID: "fold-bench", Page: 43, TextSpan: "a sentence that does not appear anywhere on this page at all"}}
	spec, err := DeriveOracleGeometry(store, rec.Page, rec.EvidenceRefs, rec.PageWidth, rec.PageHeight)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if spec.Verdict != OracleInsufficientProvenance {
		t.Fatalf("verdict = %s, want INSUFFICIENT_PROVENANCE", spec.Verdict)
	}
}

func TestModelStep_RecipeSourcedAndGuarded(t *testing.T) {
	rec := numericFixtureRecord(t.TempDir())
	step, err := rec.ModelStep()
	if err != nil || step.Opcode != "EXTRACT_NUMBER" {
		t.Fatalf("want EXTRACT_NUMBER from recipe, got %q err=%v", step.Opcode, err)
	}

	zero := rec
	zero.Recipe = []AtomicStep{{ID: "n", Opcode: "NORMALIZE", OutputKey: "a", Deterministic: true}}
	if _, err := zero.ModelStep(); err == nil {
		t.Fatalf("expected an error for a recipe with zero model steps")
	}

	two := rec
	two.Recipe = []AtomicStep{
		{ID: "a", Opcode: "EXTRACT_NUMBER", OutputKey: "x"},
		{ID: "b", Opcode: "SELECT_ONE", OutputKey: "y"},
	}
	if _, err := two.ModelStep(); err == nil {
		t.Fatalf("expected an error for a recipe with two model steps (no planner)")
	}
}
