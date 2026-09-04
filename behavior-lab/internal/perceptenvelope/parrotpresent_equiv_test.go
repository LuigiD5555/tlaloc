package perceptenvelope_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"tlaloc.local/behaviorlab/internal/parrotpresent"
	"tlaloc.local/behaviorlab/internal/perceptenvelope"
)

// TONAL T1, requirement 2: the reusable internal/parrotpresent primitive
// must reproduce the frozen R1/H EXTRACT_NUMBER presentation semantics
// (perceptenvelope.RenderR1BScale, commit 445b18c) byte-for-byte given the
// same resolved geometry. If this ever drifts, the R1 held-out evidence
// no longer transfers to the runtime path.
func TestParrotPresentReproducesFrozenR1BRenderer(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 400, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 400; x++ {
			page.Set(x, y, color.RGBA{245, 245, 245, 255})
		}
	}
	for y := 280; y < 300; y++ { // dark operand band
		for x := 120; x < 260; x++ {
			page.Set(x, y, color.RGBA{10, 10, 10, 255})
		}
	}
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, page); err != nil {
		t.Fatal(err)
	}

	const storePageWidth = 200.0 // image is 2 px per store unit
	tokenStore := parrotpresent.BBox{X1: 60, Y1: 140, X2: 130, Y2: 150}
	const lineHeightStore = 10.0
	const targetLinePx = 32.0
	scale := targetLinePx / lineHeightStore

	geometry, err := parrotpresent.DeriveGeometry(parrotpresent.Operand{
		TokenBBox: tokenStore, LineHeight: lineHeightStore, PageWidthStore: storePageWidth,
	}, 400, targetLinePx)
	if err != nil {
		t.Fatal(err)
	}

	ours := parrotpresent.RenderFromGeometry(page, geometry, true)

	tcx, tcy := geometry.TargetCenterStore[0], geometry.TargetCenterStore[1]
	cx1, cy1 := parrotpresent.StoreToCanvas(geometry.CueBBoxStore[0], geometry.CueBBoxStore[1], tcx, tcy, scale)
	cx2, cy2 := parrotpresent.StoreToCanvas(geometry.CueBBoxStore[2], geometry.CueBBoxStore[3], tcx, tcy, scale)

	base := perceptenvelope.Base{Candidate: perceptenvelope.Candidate{PageWidth: storePageWidth}}
	frozenGeo := perceptenvelope.R1BGeometry{
		TargetCenterStore: [2]float64{tcx, tcy},
		SourceCropStore:   geometry.SourceCropStore,
	}
	cond := perceptenvelope.R1BCondGeom{
		AffineScale:     scale,
		CueBBoxCanvasPx: [4]float64{cx1, cy1, cx2, cy2},
		CueStrokePx:     parrotpresent.CueStrokePx,
	}
	theirs, _, err := perceptenvelope.RenderR1BScale(pngBytes.Bytes(), base, frozenGeo, cond)
	if err != nil {
		t.Fatalf("RenderR1BScale: %v", err)
	}

	if !bytes.Equal(ours.Pix, theirs.Pix) {
		diff := 0
		for i := range ours.Pix {
			if ours.Pix[i] != theirs.Pix[i] {
				diff++
			}
		}
		t.Fatalf("parrotpresent differs from the frozen R1-B renderer in %d/%d bytes", diff, len(ours.Pix))
	}
}
