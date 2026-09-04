package parrotpresent

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

const (
	fixturePage       = "testdata/page.png"
	fixturePageWidth  = 200.0
	fixturePageHeight = 300.0
)

// the operand line in the fixture: y in [100,120], x in [20,180]
var fixtureLine = BBox{X1: 20, Y1: 100, X2: 180, Y2: 120}

func TestPrepare_CropToLine_RetainsTargetAndReportsGeometry(t *testing.T) {
	out := filepath.Join(t.TempDir(), "crop.png")
	result, err := Prepare(fixturePage, Region{Line: fixtureLine, PageWidth: fixturePageWidth, PageHeight: fixturePageHeight},
		Plan{CropToLine: true}, out)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if !result.AppliedCrop || result.AppliedUpscale {
		t.Fatalf("expected crop only, got crop=%v upscale=%v", result.AppliedCrop, result.AppliedUpscale)
	}
	// line is 20 px tall; margin default 0.5 -> 10 px each side -> canvas 40 px tall
	if result.CanvasHeight != 40 {
		t.Fatalf("canvas height = %d, want 40", result.CanvasHeight)
	}
	// submitted line height without upscale is the line's own height
	if result.SubmittedLineHeightPx != 20 {
		t.Fatalf("submitted line height = %d, want 20", result.SubmittedLineHeightPx)
	}
	// the black operand bar must still be present in the crop
	decoded := decodePNG(t, result.Bytes)
	if !hasBlackPixel(decoded) {
		t.Fatalf("cropped image lost the operand bar")
	}
}

func TestPrepare_UpscaleToPreferred_HitsTargetLineHeight(t *testing.T) {
	result, err := Prepare(fixturePage, Region{Line: fixtureLine, PageWidth: fixturePageWidth, PageHeight: fixturePageHeight},
		Plan{CropToLine: true, Upscale: true, TargetLineHeightPx: 32}, "")
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if !result.AppliedUpscale {
		t.Fatal("expected an upscale")
	}
	// line 20 px -> target 32 px, factor 1.6; submitted within 1 px of target
	if result.SubmittedLineHeightPx < 31 || result.SubmittedLineHeightPx > 33 {
		t.Fatalf("submitted line height after upscale = %d, want ~32", result.SubmittedLineHeightPx)
	}
	// canvas: 40 px tall crop * 1.6 = 64
	if result.CanvasHeight != 64 {
		t.Fatalf("canvas height after upscale = %d, want 64", result.CanvasHeight)
	}
}

func TestPrepare_AlreadyAtOrAboveTarget_DoesNotUpscale(t *testing.T) {
	// a 40 px tall line already exceeds a 32 px target
	tallLine := BBox{X1: 20, Y1: 100, X2: 180, Y2: 140}
	result, err := Prepare(fixturePage, Region{Line: tallLine, PageWidth: fixturePageWidth, PageHeight: fixturePageHeight},
		Plan{CropToLine: true, Upscale: true, TargetLineHeightPx: 32}, "")
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if result.AppliedUpscale {
		t.Fatalf("must not upscale a line already >= target; scale factor %v", result.ScaleFactor)
	}
}

func TestPrepare_IsDeterministicByteForByte(t *testing.T) {
	plan := Plan{CropToLine: true, Upscale: true, TargetLineHeightPx: 32}
	region := Region{Line: fixtureLine, PageWidth: fixturePageWidth, PageHeight: fixturePageHeight}
	first, err := Prepare(fixturePage, region, plan, "")
	if err != nil {
		t.Fatalf("Prepare 1: %v", err)
	}
	second, err := Prepare(fixturePage, region, plan, "")
	if err != nil {
		t.Fatalf("Prepare 2: %v", err)
	}
	if first.SHA256 != second.SHA256 || !bytes.Equal(first.Bytes, second.Bytes) {
		t.Fatalf("Prepare is not deterministic: %s != %s", first.SHA256, second.SHA256)
	}
}

func TestPrepare_MissingPage_IsAnErrorNotAPanic(t *testing.T) {
	if _, err := Prepare("testdata/does-not-exist.png", Region{Line: fixtureLine, PageWidth: 1, PageHeight: 1}, Plan{}, ""); err == nil {
		t.Fatal("expected an error for a missing page image")
	}
}

func TestFixtureIsGeometryOnly(t *testing.T) {
	body, err := os.ReadFile("testdata/region.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, banned := range []string{"gold", "expected", "answer", "scorer", "base_id", "benchmark"} {
		if bytes.Contains(bytes.ToLower(body), []byte(banned)) {
			t.Fatalf("fixture region.json must not mention %q", banned)
		}
	}
}

func decodePNG(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	return img
}

func hasBlackPixel(img image.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if r < 0x2000 && g < 0x2000 && bl < 0x2000 {
				return true
			}
		}
	}
	return false
}
