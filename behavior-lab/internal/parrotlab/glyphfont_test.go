package parrotlab

import (
	"image"
	"image/color"
	"testing"
)

func TestBitmapFontCoversRequiredAlphabets(t *testing.T) {
	for _, symbol := range letterAlphabet {
		if _, ok := bitmapFont[symbol]; !ok {
			t.Fatalf("letter %q missing from bitmapFont", symbol)
		}
	}
	for _, symbol := range digitAlphabet {
		if _, ok := bitmapFont[symbol]; !ok {
			t.Fatalf("digit %q missing from bitmapFont", symbol)
		}
	}
	for _, dropped := range []rune{'0', '1', '8'} {
		found := false
		for _, kept := range digitAlphabet {
			if kept == dropped {
				found = true
			}
		}
		if found {
			t.Fatalf("digit %q should be excluded from digitAlphabet", dropped)
		}
	}
}

func TestDrawTextIsStableAndBounded(t *testing.T) {
	render := func() *image.RGBA {
		canvas := image.NewRGBA(image.Rect(0, 0, 200, 40))
		drawText(canvas, "AB7", image.Point{X: 4, Y: 4}, 3, color.RGBA{0, 0, 0, 255})
		return canvas
	}
	first, second := render(), render()
	for y := 0; y < 40; y++ {
		for x := 0; x < 200; x++ {
			if first.RGBAAt(x, y) != second.RGBAAt(x, y) {
				t.Fatalf("drawText not deterministic at (%d,%d)", x, y)
			}
		}
	}
	box := drawText(render(), "AB7", image.Point{X: 4, Y: 4}, 3, color.RGBA{0, 0, 0, 255})
	if box.Dx() != textPixelWidth("AB7", 3) || box.Dy() != glyphHeight*3 {
		t.Fatalf("unexpected text box %v", box)
	}
}
