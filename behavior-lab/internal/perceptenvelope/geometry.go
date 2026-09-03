package perceptenvelope

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/exocortex"
)

// ContextLevel is one of the seven R1-A context-envelope conditions.
type ContextLevel string

const (
	A0TargetOnly        ContextLevel = "A0_TARGET_ONLY"
	A1TargetPlusLine    ContextLevel = "A1_TARGET_PLUS_LINE"
	A2LocalBlock        ContextLevel = "A2_LOCAL_BLOCK"
	A3BlockPlusNeighbor ContextLevel = "A3_BLOCK_PLUS_NEIGHBOR"
	A4QuarterPage       ContextLevel = "A4_QUARTER_PAGE"
	A5HalfPage          ContextLevel = "A5_HALF_PAGE"
	A6FullPage          ContextLevel = "A6_FULL_PAGE"
)

// AllContextLevels is the frozen ordered condition list for R1-A.
var AllContextLevels = []ContextLevel{
	A0TargetOnly, A1TargetPlusLine, A2LocalBlock, A3BlockPlusNeighbor,
	A4QuarterPage, A5HalfPage, A6FullPage,
}

// cueStrokePx is the fixed cue stroke width in image pixels (frozen).
const cueStrokePx = 3

// cueColor is the fixed cue stroke colour (frozen): opaque magenta, chosen
// to be visually distinct from body text/rules without encoding an answer.
var cueColor = color.RGBA{R: 230, G: 0, B: 130, A: 255}

// ContextRegionStore returns the deterministic store-coordinate region for a
// context level around a candidate. For A6 it returns the full page. All
// regions are a pure function of the frozen candidate geometry and the page
// dimensions — never of model output.
func ContextRegionStore(cand Candidate, level ContextLevel) canonicaldoc.BBox {
	w, h := cand.PageWidth, cand.PageHeight
	line := cand.Line.BBox
	tb := cand.TokenBBoxStore
	cx := (tb.X1 + tb.X2) / 2
	cy := (tb.Y1 + tb.Y2) / 2

	switch level {
	case A0TargetOnly:
		return clampBox(tb, w, h)
	case A1TargetPlusLine:
		return clampBox(canonicaldoc.BBox{X1: line.X1, Y1: line.Y1, X2: line.X2, Y2: line.Y2}, w, h)
	case A2LocalBlock:
		// containing block ~= the target line +/- one line-height band,
		// full text column width (line width padded to the page text box).
		band := 2.5 * (line.Y2 - line.Y1)
		return clampBox(canonicaldoc.BBox{
			X1: 0.08 * w, Y1: line.Y1 - band,
			X2: 0.92 * w, Y2: line.Y2 + band,
		}, w, h)
	case A3BlockPlusNeighbor:
		band := 6.0 * (line.Y2 - line.Y1)
		return clampBox(canonicaldoc.BBox{
			X1: 0.06 * w, Y1: line.Y1 - band,
			X2: 0.94 * w, Y2: line.Y2 + band,
		}, w, h)
	case A4QuarterPage:
		// deterministic 2x2 grid cell containing the target centre.
		col := 0.0
		if cx >= w/2 {
			col = 1
		}
		row := 0.0
		if cy >= h/2 {
			row = 1
		}
		return canonicaldoc.BBox{X1: col * w / 2, Y1: row * h / 2, X2: (col + 1) * w / 2, Y2: (row + 1) * h / 2}
	case A5HalfPage:
		if cy < h/2 {
			return canonicaldoc.BBox{X1: 0, Y1: 0, X2: w, Y2: h / 2}
		}
		return canonicaldoc.BBox{X1: 0, Y1: h / 2, X2: w, Y2: h}
	default: // A6FullPage
		return canonicaldoc.BBox{X1: 0, Y1: 0, X2: w, Y2: h}
	}
}

func clampBox(b canonicaldoc.BBox, w, h float64) canonicaldoc.BBox {
	if b.X1 < 0 {
		b.X1 = 0
	}
	if b.Y1 < 0 {
		b.Y1 = 0
	}
	if b.X2 > w {
		b.X2 = w
	}
	if b.Y2 > h {
		b.Y2 = h
	}
	return b
}

// RenderCuedPage decodes a rendered page PNG and draws the frozen target
// cue rectangle at the candidate's token bbox (affine-scaled store->image).
// The cue is drawn once, before any cropping/resampling, so it survives
// identically into every context/scale variant. Returns the cued PNG bytes.
func RenderCuedPage(pagePNG []byte, cand Candidate) ([]byte, image.Rectangle, error) {
	src, err := png.Decode(bytes.NewReader(pagePNG))
	if err != nil {
		return nil, image.Rectangle{}, fmt.Errorf("decode page png: %w", err)
	}
	bnds := src.Bounds()
	rgba := image.NewRGBA(bnds)
	draw.Draw(rgba, bnds, src, bnds.Min, draw.Src)

	sx := float64(bnds.Dx()) / cand.PageWidth
	sy := float64(bnds.Dy()) / cand.PageHeight
	tb := cand.TokenBBoxStore
	rect := image.Rect(
		bnds.Min.X+int(tb.X1*sx), bnds.Min.Y+int(tb.Y1*sy),
		bnds.Min.X+int(tb.X2*sx), bnds.Min.Y+int(tb.Y2*sy),
	).Intersect(bnds)
	if rect.Empty() {
		return nil, image.Rectangle{}, fmt.Errorf("cue rect empty for %s", cand.CandidateID)
	}
	strokeRect(rgba, rect, cueColor, cueStrokePx)

	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, rgba); err != nil {
		return nil, image.Rectangle{}, err
	}
	return buf.Bytes(), rect, nil
}

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

// WriteContextVariant crops the cued page to a context level's store region
// and writes the PNG to outPath. Returns the visual-exposure ratio
// (crop pixels / full page pixels).
func WriteContextVariant(cuedPagePath, outPath string, cand Candidate, level ContextLevel) (float64, error) {
	if level == A6FullPage {
		// full page: the cued page itself is the variant. Copy so every
		// context level has its own on-disk crop artifact.
		data, err := os.ReadFile(cuedPagePath)
		if err != nil {
			return 0, err
		}
		if err := os.WriteFile(outPath, data, 0o644); err != nil {
			return 0, err
		}
		return 1.0, nil
	}
	region := ContextRegionStore(cand, level)
	b := region
	_, exposure, err := exocortex.CropImageToBBox(exocortex.RegionCropInput{
		PageImagePath: cuedPagePath,
		PageWidth:     cand.PageWidth,
		PageHeight:    cand.PageHeight,
		BBox:          &b,
		OutputPath:    outPath,
	})
	return exposure, err
}

// fixedCanvasBG is the frozen neutral fill for pixels outside the visible
// context region in the FIXED_CANVAS diagnostic (mid grey).
var fixedCanvasBG = color.RGBA{R: 200, G: 200, B: 200, A: 255}

// WriteFixedCanvasVariant writes a context variant whose IMAGE DIMENSIONS,
// target position and target pixel scale are identical for every context
// level: the full rendered page canvas, with the cued document pixels kept
// only inside the level's context region and every outside pixel replaced
// by one fixed neutral grey. Only the amount of visible document context
// changes across A0..A6; total image size, target location and target
// glyph scale are constant. Returns visible-region area / canvas area.
func WriteFixedCanvasVariant(cuedPagePath, outPath string, cand Candidate, level ContextLevel) (float64, error) {
	data, err := os.ReadFile(cuedPagePath)
	if err != nil {
		return 0, err
	}
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	b := src.Bounds()
	out := image.NewRGBA(b)
	if level == A6FullPage {
		draw.Draw(out, b, src, b.Min, draw.Src)
	} else {
		draw.Draw(out, b, &image.Uniform{C: fixedCanvasBG}, image.Point{}, draw.Src)
		region := ContextRegionStore(cand, level)
		sx := float64(b.Dx()) / cand.PageWidth
		sy := float64(b.Dy()) / cand.PageHeight
		vis := image.Rect(
			b.Min.X+int(region.X1*sx), b.Min.Y+int(region.Y1*sy),
			b.Min.X+int(region.X2*sx), b.Min.Y+int(region.Y2*sy),
		).Intersect(b)
		draw.Draw(out, vis, src, vis.Min, draw.Src)
	}
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, out); err != nil {
		return 0, err
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		return 0, err
	}
	region := ContextRegionStore(cand, level)
	if level == A6FullPage {
		return 1.0, nil
	}
	return ((region.X2 - region.X1) * (region.Y2 - region.Y1)) / (cand.PageWidth * cand.PageHeight), nil
}

// pageDimsFromPNG reads a PNG's pixel dimensions.
func pageDimsFromPNG(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}
