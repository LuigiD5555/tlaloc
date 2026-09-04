// Package parrotpresent is the minimal, reusable, deterministic
// presentation primitive the R1-aware Parrot Tlaloque needs: crop a
// rendered page image to an operand's containing line and, when asked,
// upscale that crop so the line reaches a target pixel height.
//
// It is deliberately generic geometry code. It knows nothing about
// benchmarks, gold answers, scorers, or any particular experiment. Which
// transformations to apply, and the target pixel height, are decided
// upstream by exocortex.AdapterR1 against the frozen CapabilityProfile R1;
// this package only executes the geometry.
package parrotpresent

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
)

// BBox is a rectangle in a page's own coordinate space.
type BBox struct {
	X1, Y1, X2, Y2 float64
}

func (b BBox) width() float64  { return b.X2 - b.X1 }
func (b BBox) height() float64 { return b.Y2 - b.Y1 }

// Region describes the operand's geometry on the page.
type Region struct {
	// Line is the operand's containing text line, in page coordinates.
	Line BBox
	// Target is the operand token bbox in page coordinates. Optional; used
	// only for IsolateOperand.
	Target                *BBox
	PageWidth, PageHeight float64
}

// Plan is the set of transformations to apply, as decided by AdapterR1.
type Plan struct {
	CropToLine         bool
	Upscale            bool
	TargetLineHeightPx int
	IsolateOperand     bool
	// MarginRatio is the padding added on each side, as a fraction of the
	// line height. Zero selects the default (0.5), matching the R1 source
	// pool's padding policy.
	MarginRatio float64
}

// Result reports what was produced.
type Result struct {
	Bytes                 []byte
	OutputPath            string
	SubmittedLineHeightPx int
	CanvasWidth           int
	CanvasHeight          int
	SHA256                string
	AppliedCrop           bool
	AppliedUpscale        bool
	ScaleFactor           float64
}

// Prepare reads the page PNG at pageImagePath, applies plan to region, and
// writes the prepared PNG to outPath (when non-empty). It is deterministic:
// the same inputs always produce byte-identical output.
func Prepare(pageImagePath string, region Region, plan Plan, outPath string) (Result, error) {
	data, err := os.ReadFile(pageImagePath)
	if err != nil {
		return Result{}, fmt.Errorf("parrotpresent: read page image: %w", err)
	}
	source, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return Result{}, fmt.Errorf("parrotpresent: decode page image: %w", err)
	}
	bounds := source.Bounds()
	if region.PageWidth <= 0 || region.PageHeight <= 0 {
		return Result{}, fmt.Errorf("parrotpresent: page dimensions are required")
	}
	scaleX := float64(bounds.Dx()) / region.PageWidth
	scaleY := float64(bounds.Dy()) / region.PageHeight

	marginRatio := plan.MarginRatio
	if marginRatio <= 0 {
		marginRatio = 0.5
	}

	lineHeightPx := region.Line.height() * scaleY
	if lineHeightPx <= 0 {
		return Result{}, fmt.Errorf("parrotpresent: line height resolves to %.2f px", lineHeightPx)
	}
	marginPx := marginRatio * lineHeightPx

	result := Result{ScaleFactor: 1}
	current := image.NewRGBA(bounds)
	draw.Draw(current, bounds, source, bounds.Min, draw.Src)
	cropped := image.Image(current)

	if plan.CropToLine {
		xLeft := region.Line.X1
		xRight := region.Line.X2
		if plan.IsolateOperand && region.Target != nil {
			xLeft = region.Target.X1
			xRight = region.Target.X2
		}
		rect := image.Rect(
			bounds.Min.X+int(math.Floor(xLeft*scaleX-marginPx)),
			bounds.Min.Y+int(math.Floor(region.Line.Y1*scaleY-marginPx)),
			bounds.Min.X+int(math.Ceil(xRight*scaleX+marginPx)),
			bounds.Min.Y+int(math.Ceil(region.Line.Y2*scaleY+marginPx)),
		).Intersect(bounds)
		if rect.Empty() {
			return Result{}, fmt.Errorf("parrotpresent: crop rectangle is empty")
		}
		sub := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
		draw.Draw(sub, sub.Bounds(), current, rect.Min, draw.Src)
		cropped = sub
		result.AppliedCrop = true
	}

	submittedLineHeightPx := lineHeightPx
	if plan.Upscale && plan.TargetLineHeightPx > 0 && lineHeightPx < float64(plan.TargetLineHeightPx) {
		factor := float64(plan.TargetLineHeightPx) / lineHeightPx
		cropped = nearestScale(cropped, factor)
		submittedLineHeightPx = lineHeightPx * factor
		result.AppliedUpscale = true
		result.ScaleFactor = factor
	}

	var buffer bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&buffer, cropped); err != nil {
		return Result{}, fmt.Errorf("parrotpresent: encode: %w", err)
	}
	sum := sha256.Sum256(buffer.Bytes())
	result.Bytes = buffer.Bytes()
	result.SHA256 = hex.EncodeToString(sum[:])
	result.SubmittedLineHeightPx = int(math.Round(submittedLineHeightPx))
	result.CanvasWidth = cropped.Bounds().Dx()
	result.CanvasHeight = cropped.Bounds().Dy()

	if outPath != "" {
		if err := os.WriteFile(outPath, buffer.Bytes(), 0o644); err != nil {
			return Result{}, fmt.Errorf("parrotpresent: write output: %w", err)
		}
		result.OutputPath = outPath
	}
	return result, nil
}

// nearestScale resamples img by factor using nearest-neighbour, which is
// exactly reproducible.
func nearestScale(img image.Image, factor float64) image.Image {
	src := img.Bounds()
	dstW := int(math.Round(float64(src.Dx()) * factor))
	dstH := int(math.Round(float64(src.Dy()) * factor))
	if dstW < 1 {
		dstW = 1
	}
	if dstH < 1 {
		dstH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		sourceY := src.Min.Y + int(float64(y)/factor)
		if sourceY >= src.Max.Y {
			sourceY = src.Max.Y - 1
		}
		for x := 0; x < dstW; x++ {
			sourceX := src.Min.X + int(float64(x)/factor)
			if sourceX >= src.Max.X {
				sourceX = src.Max.X - 1
			}
			dst.Set(x, y, img.At(sourceX, sourceY))
		}
	}
	return dst
}
