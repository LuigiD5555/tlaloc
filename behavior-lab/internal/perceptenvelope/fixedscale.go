package perceptenvelope

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

// R1-A1 FIXED-SCALE LOCAL CONTEXT ENVELOPE renderer.
//
// Every R1-A1 condition for every base is a 512x512 RGB image in which the
// target's effective scale, position and cue geometry are IDENTICAL; only
// the set of visible document pixels changes, as a nested sequence of
// reveal masks over ONE per-base transformed viewport (protocol
// R1_PROTOCOL_ADDENDUM_02 sections 4-8).

// CanvasPx is the fixed R1-A1 image side.
const CanvasPx = 512

// canvasCenter is the fixed target centre.
const canvasCenter = CanvasPx / 2

// TargetLineHeightPx is the frozen normalised containing-line height:
// per base, one affine scale s is chosen so the containing text line is
// exactly this tall in canvas pixels, and s is used for ALL seven
// conditions of that base.
const TargetLineHeightPx = 32.0

// maskBG is the frozen neutral fill for hidden pixels and out-of-page area.
var maskBG = color.RGBA{R: 200, G: 200, B: 200, A: 255}

// R1A1Level is one nested fixed-scale context condition.
type R1A1Level string

const (
	A1C0Target            R1A1Level = "A1C0_TARGET"
	A1C1Line              R1A1Level = "A1C1_LINE"
	A1C2Block             R1A1Level = "A1C2_BLOCK"
	A1C3BlockPlusNeighbor R1A1Level = "A1C3_BLOCK_PLUS_NEIGHBOR"
	A1C4Local256          R1A1Level = "A1C4_LOCAL_256"
	A1C5Local384          R1A1Level = "A1C5_LOCAL_384"
	A1C6FullViewport      R1A1Level = "A1C6_FULL_VIEWPORT"
)

// AllR1A1Levels is the frozen ordered nested ladder.
var AllR1A1Levels = []R1A1Level{
	A1C0Target, A1C1Line, A1C2Block, A1C3BlockPlusNeighbor,
	A1C4Local256, A1C5Local384, A1C6FullViewport,
}

// c0PadPx is the single fixed small padding around the cue bbox for A1C0.
const c0PadPx = 10

// R1A1Geometry is the frozen per-base transform + per-level reveal
// rectangles, all in canvas pixels. It is a pure function of the frozen
// candidate + the store page layout.
type R1A1Geometry struct {
	BaseID            string              `json:"base_id"`
	Page              int                 `json:"page"`
	AffineScale       float64             `json:"affine_scale_store_to_canvas"`
	LineHeightStore   float64             `json:"line_height_store"`
	LineHeightCanvas  float64             `json:"line_height_canvas_px"`
	TargetCenterStore [2]float64          `json:"target_center_store"`
	CueBBoxCanvas     [4]float64          `json:"cue_bbox_canvas_px"`
	TargetBBoxCanvas  [4]float64          `json:"target_bbox_canvas_px"`
	RevealRects       map[string][][4]int `json:"reveal_rects_canvas_px"`
}

// loadStorePage reads one reconstructed-store page's layout.
func loadStorePage(storeDir string, page int) (canonicaldoc.Page, string, error) {
	manifest, _, err := pdfmemory.Load(storeDir)
	if err != nil {
		return canonicaldoc.Page{}, "", err
	}
	for _, pref := range manifest.Pages {
		if pref.Number != page || pref.LayoutPath == "" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(pref.LayoutPath)))
		if err != nil {
			return canonicaldoc.Page{}, "", err
		}
		var p canonicaldoc.Page
		if err := json.Unmarshal(body, &p); err != nil {
			return canonicaldoc.Page{}, "", err
		}
		return p, pref.LayoutPath, nil
	}
	return canonicaldoc.Page{}, "", fmt.Errorf("store page %d not found", page)
}

// storeToCanvas maps a store-space point to canvas pixels for scale s and
// target centre tc.
func storeToCanvas(x, y, tcx, tcy, s float64) (float64, float64) {
	return (x-tcx)*s + canvasCenter, (y-tcy)*s + canvasCenter
}

func rectFromStoreBBox(b canonicaldoc.BBox, tcx, tcy, s float64) [4]int {
	x1, y1 := storeToCanvas(b.X1, b.Y1, tcx, tcy, s)
	x2, y2 := storeToCanvas(b.X2, b.Y2, tcx, tcy, s)
	return clampRect([4]int{int(math.Floor(x1)), int(math.Floor(y1)), int(math.Ceil(x2)), int(math.Ceil(y2))})
}

func clampRect(r [4]int) [4]int {
	if r[0] < 0 {
		r[0] = 0
	}
	if r[1] < 0 {
		r[1] = 0
	}
	if r[2] > CanvasPx {
		r[2] = CanvasPx
	}
	if r[3] > CanvasPx {
		r[3] = CanvasPx
	}
	if r[2] < r[0] {
		r[2] = r[0]
	}
	if r[3] < r[1] {
		r[3] = r[1]
	}
	return r
}

// DeriveR1A1Geometry computes the frozen transform + nested reveal
// rectangles for one base. Deterministic; reads no model output.
func DeriveR1A1Geometry(storeDir string, base Base) (R1A1Geometry, error) {
	c := base.Candidate
	page, _, err := loadStorePage(storeDir, c.Page)
	if err != nil {
		return R1A1Geometry{}, err
	}
	line := c.Line.BBox
	lineH := line.Y2 - line.Y1
	if lineH <= 0 {
		return R1A1Geometry{}, fmt.Errorf("%s: non-positive line height", base.BaseID)
	}
	s := TargetLineHeightPx / lineH

	// R1-A1 cue: tight token bbox recomputed from the frozen line bbox +
	// frozen character offsets (same frozen-geometry semantics as the pool
	// cue, but bracketing only the digits + a 2 px pad, not padded by half
	// a line height). Deterministic; reads no model output.
	nRunes := float64(len([]rune(c.Line.Text)))
	lw := line.X2 - line.X1
	charW := lw / nRunes
	tokX1 := line.X1 + lw*float64(c.CharOffsetStart)/nRunes
	tokX2 := line.X1 + lw*float64(c.CharOffsetEnd)/nRunes
	// horizontal pad = 1.5 character widths: absorbs the proportional
	// offset estimate's ~1-char error so the digits are always inside the
	// cue. The one-digit-per-line admission rule keeps a loose box
	// unambiguous. Vertical pad is small (0.15 line-heights).
	xpad := 1.5 * charW
	ypad := 0.15 * lineH
	tb := canonicaldoc.BBox{X1: tokX1 - xpad, Y1: line.Y1 - ypad, X2: tokX2 + xpad, Y2: line.Y2 + ypad}
	tcx := (tb.X1 + tb.X2) / 2
	tcy := (tb.Y1 + tb.Y2) / 2

	geo := R1A1Geometry{
		BaseID: base.BaseID, Page: c.Page, AffineScale: s,
		LineHeightStore: lineH, LineHeightCanvas: lineH * s,
		TargetCenterStore: [2]float64{tcx, tcy},
		RevealRects:       map[string][][4]int{},
	}
	cueR := rectFromStoreBBox(tb, tcx, tcy, s)
	geo.CueBBoxCanvas = [4]float64{float64(cueR[0]), float64(cueR[1]), float64(cueR[2]), float64(cueR[3])}
	geo.TargetBBoxCanvas = geo.CueBBoxCanvas

	// locate the target line's region index in the page
	idx := -1
	for i, r := range page.Regions {
		if r.ID == c.Line.RegionID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return R1A1Geometry{}, fmt.Errorf("%s: line region %s not on page %d", base.BaseID, c.Line.RegionID, c.Page)
	}

	// A1C2_BLOCK: the containing paragraph — text_line/list_item regions
	// walking up and down from the target line in vertical order while the
	// inter-line vertical gap stays < 0.8 line-heights, capped at +/-6
	// lines. Headings, table cells, figures and code stop the walk (larger
	// gap or excluded kind). Regions are not spatially ordered by index,
	// so we sort a body-text subset by vertical centre.
	type vreg struct {
		y1, y2 float64
		bb     canonicaldoc.BBox
	}
	var body []vreg
	targetY := (line.Y1 + line.Y2) / 2
	for _, r := range page.Regions {
		if (r.Kind != "text_line" && r.Kind != "list_item") || r.BBox.X2 <= r.BBox.X1 || r.BBox.Y2 <= r.BBox.Y1 {
			continue
		}
		body = append(body, vreg{y1: r.BBox.Y1, y2: r.BBox.Y2, bb: r.BBox})
	}
	sort.Slice(body, func(i, j int) bool {
		return (body[i].y1 + body[i].y2) < (body[j].y1 + body[j].y2)
	})
	ti := -1
	best := math.Inf(1)
	for i, v := range body {
		d := math.Abs((v.y1+v.y2)/2 - targetY)
		if d < best {
			best, ti = d, i
		}
	}
	gapLimit := 0.8 * lineH
	blkLo, blkHi := ti, ti
	for j := ti - 1; j >= 0 && ti-j <= 6; j-- {
		if body[blkLo].y1-body[j].y2 > gapLimit {
			break
		}
		blkLo = j
	}
	for j := ti + 1; j < len(body) && j-ti <= 6; j++ {
		if body[j].y1-body[blkHi].y2 > gapLimit {
			break
		}
		blkHi = j
	}
	unionV := func(lo, hi int) canonicaldoc.BBox {
		u := canonicaldoc.BBox{X1: math.Inf(1), Y1: math.Inf(1), X2: math.Inf(-1), Y2: math.Inf(-1)}
		for i := lo; i <= hi && i < len(body); i++ {
			b := body[i].bb
			u.X1, u.Y1 = math.Min(u.X1, b.X1), math.Min(u.Y1, b.Y1)
			u.X2, u.Y2 = math.Max(u.X2, b.X2), math.Max(u.Y2, b.Y2)
		}
		return u
	}
	nbLo, nbHi := blkLo, blkHi
	if nbLo > 0 {
		nbLo--
	}
	if nbHi < len(body)-1 {
		nbHi++
	}

	c0 := clampRect([4]int{cueR[0] - c0PadPx, cueR[1] - c0PadPx, cueR[2] + c0PadPx, cueR[3] + c0PadPx})
	c1 := rectFromStoreBBox(line, tcx, tcy, s)
	c2 := rectFromStoreBBox(unionV(blkLo, blkHi), tcx, tcy, s)
	c3 := rectFromStoreBBox(unionV(nbLo, nbHi), tcx, tcy, s)
	win := func(half int) [4]int {
		return clampRect([4]int{canvasCenter - half, canvasCenter - half, canvasCenter + half, canvasCenter + half})
	}
	c4, c5 := win(128), win(192)
	c6 := [4]int{0, 0, CanvasPx, CanvasPx}

	// nested: level k reveals the union of def_0..def_k (as a rect list).
	defs := [][4]int{c0, c1, c2, c3, c4, c5, c6}
	for k, lvl := range AllR1A1Levels {
		rects := make([][4]int, 0, k+1)
		for j := 0; j <= k; j++ {
			rects = append(rects, defs[j])
		}
		geo.RevealRects[string(lvl)] = rects
	}
	return geo, nil
}

// writeRGBAPNG encodes an RGBA image to path.
func writeRGBAPNG(path string, img *image.RGBA) error {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, img); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// bilinearSample samples img at fractional (fx,fy); out-of-bounds -> ok=false.
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

// BuildR1A1Viewport renders the ONE per-base 512x512 transformed viewport
// (all document content visible, cue drawn) from the rendered page PNG.
// Returns the viewport image; masking to levels is applied by
// WriteR1A1Level.
func BuildR1A1Viewport(pagePNG []byte, storeDir string, base Base, geo R1A1Geometry) (*image.RGBA, error) {
	src, err := png.Decode(bytes.NewReader(pagePNG))
	if err != nil {
		return nil, err
	}
	sb := src.Bounds()
	c := base.Candidate
	k := float64(sb.Dx()) / c.PageWidth // rendered-page px per store unit
	s := geo.AffineScale
	tcx, tcy := geo.TargetCenterStore[0], geo.TargetCenterStore[1]

	vp := image.NewRGBA(image.Rect(0, 0, CanvasPx, CanvasPx))
	for cy := 0; cy < CanvasPx; cy++ {
		for cx := 0; cx < CanvasPx; cx++ {
			// canvas -> store -> rendered-page px
			storeX := tcx + (float64(cx)-canvasCenter)/s
			storeY := tcy + (float64(cy)-canvasCenter)/s
			px := float64(sb.Min.X) + storeX*k
			py := float64(sb.Min.Y) + storeY*k
			if col, ok := bilinearSample(src, px, py); ok {
				vp.SetRGBA(cx, cy, col)
			} else {
				vp.SetRGBA(cx, cy, maskBG)
			}
		}
	}
	// cue rectangle, drawn on the full viewport so it is identical in
	// every masked condition (the cue always sits inside A1C0).
	cr := image.Rect(int(geo.CueBBoxCanvas[0]), int(geo.CueBBoxCanvas[1]), int(geo.CueBBoxCanvas[2]), int(geo.CueBBoxCanvas[3]))
	strokeRect(vp, cr, cueColor, cueStrokePx)
	return vp, nil
}

// WriteR1A1Level writes one nested-mask condition PNG. visible pixels come
// byte-identically from the shared viewport; hidden pixels are maskBG.
func WriteR1A1Level(vp *image.RGBA, outPath string, geo R1A1Geometry, level R1A1Level) (float64, error) {
	rects := geo.RevealRects[string(level)]
	out := image.NewRGBA(image.Rect(0, 0, CanvasPx, CanvasPx))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: maskBG}, image.Point{}, draw.Src)
	visible := 0
	for cy := 0; cy < CanvasPx; cy++ {
		for cx := 0; cx < CanvasPx; cx++ {
			in := false
			for _, r := range rects {
				if cx >= r[0] && cx < r[2] && cy >= r[1] && cy < r[3] {
					in = true
					break
				}
			}
			if in {
				out.SetRGBA(cx, cy, vp.RGBAAt(cx, cy))
				visible++
			}
		}
	}
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, out); err != nil {
		return 0, err
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		return 0, err
	}
	return float64(visible) / float64(CanvasPx*CanvasPx), nil
}
