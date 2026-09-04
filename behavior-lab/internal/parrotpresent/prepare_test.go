package parrotpresent

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// fixtureOperand: page.png is 200x300 px. Callers choose PageWidthStore to
// exercise store!=image coordinate spaces. The operand bar is at image
// y in [100,120], x in [20,180].
func fixtureOperand(pageWidthStore float64) Operand {
	k := 200.0 / pageWidthStore // image px per store unit
	return Operand{
		TokenBBox:      BBox{X1: 20 / k, Y1: 100 / k, X2: 180 / k, Y2: 120 / k},
		LineHeight:     20 / k,
		PageWidthStore: pageWidthStore,
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

func countMagenta(img image.Image) int {
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if r>>8 == 230 && g>>8 == 0 && bl>>8 == 130 {
				n++
			}
		}
	}
	return n
}

func hasBlackNear(img image.Image, cx, cy, radius int) bool {
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if r>>8 < 0x20 && g>>8 < 0x20 && bl>>8 < 0x20 {
				return true
			}
		}
	}
	return false
}

func TestPrepare_Canvas512_Deterministic_TargetCentredAndCued(t *testing.T) {
	out := filepath.Join(t.TempDir(), "p.png")
	first, err := Prepare("testdata/page.png", fixtureOperand(200), Plan{TargetLineHeightPx: 32, DrawCue: true}, out)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if first.CanvasWidth != 512 || first.CanvasHeight != 512 {
		t.Fatalf("canvas = %dx%d, want 512x512", first.CanvasWidth, first.CanvasHeight)
	}
	if first.SubmittedLineHeightPx != 32 {
		t.Fatalf("submitted line height = %d, want 32", first.SubmittedLineHeightPx)
	}
	img := decodePNG(t, first.Bytes)
	if !hasBlackNear(img, 256, 256, 40) {
		t.Fatalf("operand is not near the canvas centre")
	}
	if countMagenta(img) == 0 {
		t.Fatalf("no cue stroke drawn")
	}
	// determinism
	second, err := Prepare("testdata/page.png", fixtureOperand(200), Plan{TargetLineHeightPx: 32, DrawCue: true}, "")
	if err != nil {
		t.Fatalf("Prepare 2: %v", err)
	}
	if first.SHA256 != second.SHA256 {
		t.Fatalf("not deterministic: %s != %s", first.SHA256, second.SHA256)
	}
}

// Requirement 3: store coordinates are NOT assumed equal to image pixels.
func TestPrepare_StoreToImageCoordinateMapping_UnequalSpaces(t *testing.T) {
	// store page is 100 units wide; the 200 px image is 2 px per store unit
	operand := fixtureOperand(100)
	result, err := Prepare("testdata/page.png", operand, Plan{TargetLineHeightPx: 32, DrawCue: true}, "")
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	// line height in store units is 10; target 32 -> affine scale 3.2
	if got := result.Geometry.AffineScale; got < 3.19 || got > 3.21 {
		t.Fatalf("affine scale = %v, want ~3.2 (target 32 / store line height 10)", got)
	}
	if got := result.Geometry.KImagePxPerStore; got < 1.99 || got > 2.01 {
		t.Fatalf("k = %v, want 2 (image 200 px / store 100)", got)
	}
	if result.SubmittedLineHeightPx != 32 {
		t.Fatalf("submitted line height = %d, want 32", result.SubmittedLineHeightPx)
	}
	img := decodePNG(t, result.Bytes)
	if !hasBlackNear(img, 256, 256, 60) {
		t.Fatalf("operand lost when store and image coordinate spaces differ")
	}
	if countMagenta(img) == 0 {
		t.Fatalf("cue not drawn under unequal coordinate spaces")
	}
}

func TestPrepare_NaturalResolution_WhenNoTarget(t *testing.T) {
	result, err := Prepare("testdata/page.png", fixtureOperand(200), Plan{DrawCue: true}, "")
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	// with k == 1 and no target, affine scale == k == 1, line stays 20 px
	if result.SubmittedLineHeightPx != 20 {
		t.Fatalf("submitted line height = %d, want 20 (natural)", result.SubmittedLineHeightPx)
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
