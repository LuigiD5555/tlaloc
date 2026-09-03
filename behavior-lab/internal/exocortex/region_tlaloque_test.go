package exocortex

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

func TestOracleLocate_NeverExposesAnswerFields(t *testing.T) {
	in := RegionLocateInput{
		Mode: LocateModeOracle, OracleAddress: "ohf://carrier/docs/d1/pages/000176/regions/0003",
		OracleDocID: "d1", OraclePage: 176, OracleBBox: &canonicaldoc.BBox{X1: 10, Y1: 10, X2: 50, Y2: 30},
	}
	result, err := OracleLocate(in)
	if err != nil {
		t.Fatalf("OracleLocate: %v", err)
	}
	if result.SourceMethod != "ORACLE_GROUND_TRUTH_ADDRESS" {
		t.Fatalf("source_method = %q", result.SourceMethod)
	}
	// RegionLocateInput itself has no field capable of carrying an expected
	// answer or hidden label; this test documents that the oracle path's
	// input/output types only ever carry geometry.
	body, _ := json.Marshal(result)
	if bytes.Contains(body, []byte("answer")) {
		t.Fatalf("oracle locate result must never mention an answer field: %s", body)
	}
}

func TestOracleLocate_RequiresAddressAndDocID(t *testing.T) {
	if _, err := OracleLocate(RegionLocateInput{Mode: LocateModeOracle}); err == nil {
		t.Fatalf("expected error without oracle_address/oracle_doc_id")
	}
}

// buildFixtureStore writes a minimal, real pdfmemory store to disk: one
// document, one page, one block indexed under two terms, and a hand-built
// page layout with two regions of differing term overlap. This exercises
// the real pdfmemory.Search/Load primitives end-to-end without needing a
// PDF parser or any P0 evidence.
func buildFixtureStore(t *testing.T) (dir string, m pdfmemory.Manifest, idx pdfmemory.Index) {
	t.Helper()
	dir = t.TempDir()

	page := canonicaldoc.Page{
		Number: 1, Width: 600, Height: 800,
		Regions: []canonicaldoc.Region{
			{ID: "r1", Address: "ohf://carrier/docs/doc1/pages/000001/regions/0001", Kind: "text", BBox: canonicaldoc.BBox{X1: 10, Y1: 10, X2: 200, Y2: 40}, Text: "fashion mnist dataset overview"},
			{ID: "r2", Address: "ohf://carrier/docs/doc1/pages/000001/regions/0002", Kind: "text", BBox: canonicaldoc.BBox{X1: 10, Y1: 200, X2: 300, Y2: 260}, Text: "total training sample count value"},
		},
	}
	layoutBody, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("marshal layout: %v", err)
	}
	layoutPath := "pages/000001.layout.json"
	if err := os.MkdirAll(filepath.Join(dir, "pages"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, layoutPath), layoutBody, 0o644); err != nil {
		t.Fatalf("write layout: %v", err)
	}
	sum := sha256.Sum256(layoutBody)
	layoutCID := hex.EncodeToString(sum[:])

	m = pdfmemory.Manifest{
		Schema: pdfmemory.Schema, CarrierID: "carrier",
		Documents: []pdfmemory.DocumentRef{{ID: "doc1", Name: "doc1"}},
		Pages:     []pdfmemory.PageRef{{DocID: "doc1", Number: 1, Address: "ohf://carrier/docs/doc1/pages/000001", LayoutPath: layoutPath, LayoutCID: layoutCID}},
		Blocks: []pdfmemory.BlockRef{
			{DocID: "doc1", Page: 1, Number: 1, Address: "ohf://carrier/docs/doc1/pages/000001/blocks/0001", Kind: "text"},
		},
	}
	idx = pdfmemory.Index{Schema: "tlaloc.pdf-memory.r2.index", Postings: map[string][]int{
		"sample": {0}, "count": {0}, "fashion": {0}, "mnist": {0},
	}}
	return dir, m, idx
}

func TestRealLocate_RanksAndRefinesToBestOverlapRegion(t *testing.T) {
	dir, m, idx := buildFixtureStore(t)
	manifestBody, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), manifestBody, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	idxBody, _ := json.Marshal(idx)
	if err := os.WriteFile(filepath.Join(dir, "index.json"), idxBody, 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	result, err := RealLocate(RegionLocateInput{Mode: LocateModeReal, Question: "what is the sample count", StoreDir: dir})
	if err != nil {
		t.Fatalf("RealLocate: %v", err)
	}
	if result.DocID != "doc1" || result.Page != 1 {
		t.Fatalf("got doc=%s page=%d, want doc1/page1", result.DocID, result.Page)
	}
	if result.BBox == nil {
		t.Fatalf("expected a refined region bbox, got none")
	}
	if result.RegionAddress != "ohf://carrier/docs/doc1/pages/000001/regions/0002" {
		t.Fatalf("region_address = %q, want the sample-count region (r2), not the fashion-mnist region (r1)", result.RegionAddress)
	}
	if result.CandidateCount == 0 {
		t.Fatalf("candidate_count must be reported")
	}
}

func TestRealLocate_NeverUsesGroundTruth(t *testing.T) {
	// RegionLocateInput's REAL-mode fields (Question/StoreDir/Limit) contain
	// no place to smuggle a ground-truth address; only OracleLocate's
	// distinct Oracle* fields do, and RealLocate never reads them.
	dir, m, idx := buildFixtureStore(t)
	manifestBody, _ := json.Marshal(m)
	os.WriteFile(filepath.Join(dir, "manifest.json"), manifestBody, 0o644)
	idxBody, _ := json.Marshal(idx)
	os.WriteFile(filepath.Join(dir, "index.json"), idxBody, 0o644)

	in := RegionLocateInput{Mode: LocateModeReal, Question: "fashion mnist", StoreDir: dir, OracleAddress: "poisoned-if-used"}
	result, err := RealLocate(in)
	if err != nil {
		t.Fatalf("RealLocate: %v", err)
	}
	if result.SelectedAddress == "poisoned-if-used" {
		t.Fatalf("RealLocate must never read oracle fields")
	}
}

func writeSolidPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write png: %v", err)
	}
}

func TestCropImageToBBox_ReducesVisualExposure(t *testing.T) {
	dir := t.TempDir()
	pagePath := filepath.Join(dir, "page.png")
	writeSolidPNG(t, pagePath, 600, 800)
	outPath := filepath.Join(dir, "crop.png")

	_, exposure, err := CropImageToBBox(RegionCropInput{
		PageImagePath: pagePath, PageWidth: 600, PageHeight: 800,
		BBox: &canonicaldoc.BBox{X1: 10, Y1: 200, X2: 300, Y2: 260}, MarginRatio: 0.1, OutputPath: outPath,
	})
	if err != nil {
		t.Fatalf("CropImageToBBox: %v", err)
	}
	if exposure <= 0 || exposure >= 1 {
		t.Fatalf("visual_exposure_ratio = %v, want strictly between 0 and 1 for a tight crop", exposure)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected crop written to %s: %v", outPath, err)
	}
}

func TestCropImageToBBox_NilBBoxPassesThroughFullPage(t *testing.T) {
	dir := t.TempDir()
	pagePath := filepath.Join(dir, "page.png")
	writeSolidPNG(t, pagePath, 100, 100)
	_, exposure, err := CropImageToBBox(RegionCropInput{PageImagePath: pagePath, OutputPath: filepath.Join(dir, "out.png")})
	if err != nil {
		t.Fatalf("CropImageToBBox: %v", err)
	}
	if exposure != 1.0 {
		t.Fatalf("exposure = %v, want 1.0 (full page fallback) when bbox is nil", exposure)
	}
}
