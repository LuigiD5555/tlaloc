// Package parrotpresent is the minimal reusable deterministic presentation
// primitive the R1-aware Parrot Tlaloque needs. It reproduces the frozen
// R1/H EXTRACT_NUMBER presentation semantics (perceptenvelope.RenderR1BScale
// / BuildR1A1Viewport, commit 445b18c) as clean stand-alone code:
//
//   - a fixed 512x512 canvas;
//   - the operand target centred at the canvas centre;
//   - an inverse-mapped bilinear resample from the rendered page image at
//     an affine scale s = target_line_height_px / line_height_store, so the
//     containing line is exactly the requested pixel height;
//   - a neutral mask fill (RGBA 200,200,200) outside a fixed source-crop
//     rectangle and outside the page image;
//   - a magenta cue stroke (RGBA 230,0,130, width 3) around the operand.
//
// Store coordinates and rendered-image pixels are NOT assumed equal:
// k = rendered_image_width / store_page_width maps between them.
//
// It knows nothing about benchmarks, gold answers or scorers. Which
// transformations to apply and the target pixel height are decided upstream
// by exocortex.AdapterR1 against the frozen CapabilityProfile R1.
package parrotpresent

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
)

// Frozen renderer constants — identical to perceptenvelope (445b18c).
const (
	CanvasPx     = 512
	canvasCenter = CanvasPx / 2
	CueStrokePx  = 3
	// PreferredLineHeightPx mirrors perceptenvelope.TargetLineHeightPx; it
	// is the scale the cue/crop pads are normalised against, not a policy
	// decision (AdapterR1 owns the policy).
	PreferredLineHeightPx = 32.0
	c0PadPx               = 10.0
)

var (
	maskBG   = color.RGBA{R: 200, G: 200, B: 200, A: 255}
	cueColor = color.RGBA{R: 230, G: 0, B: 130, A: 255}
)

// BBox is a rectangle in a page's own (store) coordinate space.
type BBox struct{ X1, Y1, X2, Y2 float64 }

func (b BBox) width() float64  { return b.X2 - b.X1 }
func (b BBox) height() float64 { return b.Y2 - b.Y1 }
func (b BBox) center() (float64, float64) {
	return (b.X1 + b.X2) / 2, (b.Y1 + b.Y2) / 2
}

// Operand describes the located operand in STORE coordinates.
type Operand struct {
	// TokenBBox is the operand token box (for T1: the located region box).
	TokenBBox BBox
	// LineHeight is the containing text line's height in store units. For
	// T1 it defaults to TokenBBox height when a separate line is not
	// resolved.
	LineHeight float64
	// LineText, when known, tunes the horizontal cue pad exactly as the
	// frozen R1 policy did (1.5 char widths). Empty falls back to a
	// line-height proportional pad.
	LineText string
	// PageWidthStore is the canonical page width in store units.
	PageWidthStore float64
}

// Plan is the transformation set decided by AdapterR1.
type Plan struct {
	// TargetLineHeightPx is the containing-line height to render at. Zero
	// means "keep the operand's natural resolution" (no upscale).
	TargetLineHeightPx int
	// DrawCue toggles the magenta operand cue. The frozen policy always
	// drew it.
	DrawCue bool
}

// Geometry is the fully resolved transform, emitted for the trace.
type Geometry struct {
	TargetCenterStore [2]float64 `json:"target_center_store"`
	CueBBoxStore      [4]float64 `json:"cue_bbox_store"`
	SourceCropStore   [4]float64 `json:"source_crop_store"`
	AffineScale       float64    `json:"affine_scale_store_to_canvas"`
	LineHeightStore   float64    `json:"line_height_store"`
	KImagePxPerStore  float64    `json:"image_px_per_store_unit"`
}

// Result reports what was produced.
type Result struct {
	Bytes                 []byte
	OutputPath            string
	SubmittedLineHeightPx int
	CanvasWidth           int
	CanvasHeight          int
	SHA256                string
	Upscaled              bool
	Geometry              Geometry
}

// deriveCue reproduces perceptenvelope.deriveR1BCue's pad policy: xpad =
// 1.5 char widths (or 0.75*lineHeight when the line text is unknown), ypad
// = 0.15*lineHeight.
func deriveCue(op Operand) BBox {
	lineH := op.LineHeight
	if lineH <= 0 {
		lineH = op.TokenBBox.height()
	}
	xpad := 0.75 * lineH
	if runes := len([]rune(op.LineText)); runes > 0 {
		charW := op.TokenBBox.width() / float64(runes)
		if charW <= 0 {
			charW = op.TokenBBox.width()
		}
		xpad = 1.5 * charW
	}
	ypad := 0.15 * lineH
	return BBox{
		X1: op.TokenBBox.X1 - xpad, Y1: op.TokenBBox.Y1 - ypad,
		X2: op.TokenBBox.X2 + xpad, Y2: op.TokenBBox.Y2 + ypad,
	}
}

// DeriveGeometry computes the frozen transform. targetLinePx <= 0 keeps the
// operand's natural resolution (affine scale s = k).
func DeriveGeometry(op Operand, imageWidthPx int, targetLinePx float64) (Geometry, error) {
	if op.PageWidthStore <= 0 {
		return Geometry{}, fmt.Errorf("parrotpresent: store page width is required")
	}
	lineH := op.LineHeight
	if lineH <= 0 {
		lineH = op.TokenBBox.height()
	}
	if lineH <= 0 {
		return Geometry{}, fmt.Errorf("parrotpresent: line height resolves to %.2f", lineH)
	}
	k := float64(imageWidthPx) / op.PageWidthStore
	scale := k
	if targetLinePx > 0 {
		scale = targetLinePx / lineH
	}

	cue := deriveCue(op)
	padStore := c0PadPx * lineH / PreferredLineHeightPx
	crop := BBox{X1: cue.X1 - padStore, Y1: cue.Y1 - padStore, X2: cue.X2 + padStore, Y2: cue.Y2 + padStore}
	tcx, tcy := cue.center()

	return Geometry{
		TargetCenterStore: [2]float64{tcx, tcy},
		CueBBoxStore:      [4]float64{cue.X1, cue.Y1, cue.X2, cue.Y2},
		SourceCropStore:   [4]float64{crop.X1, crop.Y1, crop.X2, crop.Y2},
		AffineScale:       scale,
		LineHeightStore:   lineH,
		KImagePxPerStore:  k,
	}, nil
}

// Prepare renders the operand presentation from the page PNG at
// pageImagePath and writes it to outPath (when non-empty). Deterministic:
// the same inputs always produce byte-identical output.
func Prepare(pageImagePath string, op Operand, plan Plan, outPath string) (Result, error) {
	data, err := os.ReadFile(pageImagePath)
	if err != nil {
		return Result{}, fmt.Errorf("parrotpresent: read page image: %w", err)
	}
	source, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return Result{}, fmt.Errorf("parrotpresent: decode page image: %w", err)
	}
	bounds := source.Bounds()

	geometry, err := DeriveGeometry(op, bounds.Dx(), float64(plan.TargetLineHeightPx))
	if err != nil {
		return Result{}, err
	}
	canvas := renderCanvas(source, geometry, plan.DrawCue)

	var buffer bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&buffer, canvas); err != nil {
		return Result{}, fmt.Errorf("parrotpresent: encode: %w", err)
	}
	sum := sha256.Sum256(buffer.Bytes())
	result := Result{
		Bytes:  buffer.Bytes(),
		SHA256: hex.EncodeToString(sum[:]),
		// after the affine scale, the containing line is exactly this tall
		SubmittedLineHeightPx: int(math.Round(geometry.LineHeightStore * geometry.AffineScale)),
		CanvasWidth:           CanvasPx,
		CanvasHeight:          CanvasPx,
		Upscaled:              plan.TargetLineHeightPx > 0 && geometry.AffineScale > geometry.KImagePxPerStore,
		Geometry:              geometry,
	}
	if outPath != "" {
		if err := os.WriteFile(outPath, buffer.Bytes(), 0o644); err != nil {
			return Result{}, fmt.Errorf("parrotpresent: write output: %w", err)
		}
		result.OutputPath = outPath
	}
	return result, nil
}

// renderCanvas is the byte-for-byte equivalent of
// perceptenvelope.RenderR1BScale's inner loop (445b18c).
func renderCanvas(source image.Image, geometry Geometry, drawCue bool) *image.RGBA {
	bounds := source.Bounds()
	k := geometry.KImagePxPerStore
	scale := geometry.AffineScale
	tcx, tcy := geometry.TargetCenterStore[0], geometry.TargetCenterStore[1]
	crop := geometry.SourceCropStore

	out := image.NewRGBA(image.Rect(0, 0, CanvasPx, CanvasPx))
	for cy := 0; cy < CanvasPx; cy++ {
		for cx := 0; cx < CanvasPx; cx++ {
			storeX := tcx + (float64(cx)-canvasCenter)/scale
			storeY := tcy + (float64(cy)-canvasCenter)/scale
			if storeX < crop[0] || storeX > crop[2] || storeY < crop[1] || storeY > crop[3] {
				out.SetRGBA(cx, cy, maskBG)
				continue
			}
			px := float64(bounds.Min.X) + storeX*k
			py := float64(bounds.Min.Y) + storeY*k
			if col, ok := bilinearSample(source, px, py); ok {
				out.SetRGBA(cx, cy, col)
			} else {
				out.SetRGBA(cx, cy, maskBG)
			}
		}
	}
	if drawCue {
		cue := geometry.CueBBoxStore
		x1, y1 := storeToCanvas(cue[0], cue[1], tcx, tcy, scale)
		x2, y2 := storeToCanvas(cue[2], cue[3], tcx, tcy, scale)
		strokeRect(out, image.Rect(int(x1), int(y1), int(x2), int(y2)), cueColor, CueStrokePx)
	}
	return out
}

func storeToCanvas(x, y, tcx, tcy, scale float64) (float64, float64) {
	return (x-tcx)*scale + canvasCenter, (y-tcy)*scale + canvasCenter
}

// StoreToCanvas maps a store-space point into the 512x512 canvas. Exported
// for the frozen-renderer equivalence test.
func StoreToCanvas(x, y, tcx, tcy, scale float64) (float64, float64) {
	return storeToCanvas(x, y, tcx, tcy, scale)
}

// RenderFromGeometry renders the canvas from an already-resolved geometry.
// Exported for the frozen-renderer equivalence test; production code uses
// Prepare.
func RenderFromGeometry(source image.Image, geometry Geometry, drawCue bool) *image.RGBA {
	return renderCanvas(source, geometry, drawCue)
}

// bilinearSample is identical to perceptenvelope.bilinearSample (445b18c).
func bilinearSample(img image.Image, fx, fy float64) (color.RGBA, bool) {
	b := img.Bounds()
	x0 := int(math.Floor(fx))
	y0 := int(math.Floor(fy))
	if x0 < b.Min.X || y0 < b.Min.Y || x0+1 >= b.Max.X || y0+1 >= b.Max.Y {
		return color.RGBA{}, false
	}
	dx := fx - float64(x0)
	dy := fy - float64(y0)
	at := func(x, y int) (float64, float64, float64) {
		r, g, bl, _ := img.At(x, y).RGBA()
		return float64(r >> 8), float64(g >> 8), float64(bl >> 8)
	}
	r00, g00, b00 := at(x0, y0)
	r10, g10, b10 := at(x0+1, y0)
	r01, g01, b01 := at(x0, y0+1)
	r11, g11, b11 := at(x0+1, y0+1)
	lerp := func(a, b, t float64) float64 { return a + (b-a)*t }
	r := lerp(lerp(r00, r10, dx), lerp(r01, r11, dx), dy)
	g := lerp(lerp(g00, g10, dx), lerp(g01, g11, dx), dy)
	bb := lerp(lerp(b00, b10, dx), lerp(b01, b11, dx), dy)
	return color.RGBA{R: uint8(r + 0.5), G: uint8(g + 0.5), B: uint8(bb + 0.5), A: 255}, true
}

// strokeRect is identical to perceptenvelope.strokeRect (445b18c).
func strokeRect(img *image.RGBA, r image.Rectangle, c color.Color, width int) {
	for i := 0; i < width; i++ {
		top := image.Rect(r.Min.X-i, r.Min.Y-i, r.Max.X+i, r.Min.Y-i+1)
		bot := image.Rect(r.Min.X-i, r.Max.Y+i-1, r.Max.X+i, r.Max.Y+i)
		lft := image.Rect(r.Min.X-i, r.Min.Y-i, r.Min.X-i+1, r.Max.Y+i)
		rgt := image.Rect(r.Max.X+i-1, r.Min.Y-i, r.Max.X+i, r.Max.Y+i)
		for _, s := range []image.Rectangle{top, bot, lft, rgt} {
			draw.Draw(img, s.Intersect(img.Bounds()), &image.Uniform{C: c}, image.Point{}, draw.Src)
		}
	}
}
