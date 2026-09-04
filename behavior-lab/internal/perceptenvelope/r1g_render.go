package perceptenvelope

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"tlaloc.local/behaviorlab/internal/parrotlab"
)

// R1-G condition renderers. Each returns three 512x512 RGBA images in the
// frozen condition order [baseline, recovery-1, recovery-2] for one base.
// All reuse the frozen R1-A1 / R1-B / R1-D renderers; only the single
// intervention variable changes within a family.

// renderScaleConditions (G-A): identical operand/context, only line height
// changes — 8 px (baseline), 16 px, 32 px. Reuses the R1-B scale renderer.
func renderScaleConditions(prov parrotlab.PageProvider, storeDir string, gb R1GBase) ([3]*image.RGBA, error) {
	var out [3]*image.RGBA
	base := gb.asBase()
	geo, err := DeriveR1BGeometry(storeDir, base)
	if err != nil {
		return out, err
	}
	pagePNG, err := prov.RenderPNG(base.Candidate.Page)
	if err != nil {
		return out, fmt.Errorf("render page %d: %w", base.Candidate.Page, err)
	}
	// R1BScaleLadder indices: 0=B0(8), 2=B2(16), 4=B4(32)
	for i, idx := range []int{0, 2, 4} {
		img, _, rerr := RenderR1BScale(pagePNG, base, geo, geo.Conditions[idx])
		if rerr != nil {
			return out, rerr
		}
		out[i] = img
	}
	return out, nil
}

// maskR1A1Level applies one nested reveal mask to a shared viewport,
// returning an in-memory RGBA (the on-disk WriteR1A1Level equivalent).
func maskR1A1Level(vp *image.RGBA, geo R1A1Geometry, level R1A1Level) *image.RGBA {
	rects := geo.RevealRects[string(level)]
	out := image.NewRGBA(image.Rect(0, 0, CanvasPx, CanvasPx))
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
			} else {
				out.SetRGBA(cx, cy, maskBG)
			}
		}
	}
	return out
}

// renderContextConditions (G-B): fixed 32 px scale + centred target, only
// the visible context changes — FULL_VIEWPORT (baseline), LINE, TARGET.
func renderContextConditions(prov parrotlab.PageProvider, storeDir string, gb R1GBase) ([3]*image.RGBA, error) {
	var out [3]*image.RGBA
	base := gb.asBase()
	geo, err := DeriveR1A1Geometry(storeDir, base)
	if err != nil {
		return out, err
	}
	pagePNG, err := prov.RenderPNG(base.Candidate.Page)
	if err != nil {
		return out, fmt.Errorf("render page %d: %w", base.Candidate.Page, err)
	}
	vp, err := BuildR1A1Viewport(pagePNG, storeDir, base, geo)
	if err != nil {
		return out, err
	}
	for i, lvl := range []R1A1Level{A1C6FullViewport, A1C1Line, A1C0Target} {
		out[i] = maskR1A1Level(vp, geo, lvl)
	}
	return out, nil
}

// renderRealAssocConditions (G-C REAL): the frozen R1-D D0L viewport (label
// cued). Baseline adds one fresh competitor sprite (K1); the two recovery
// conditions have no competitor. Because the R1-D viewport already masks
// every non-line pixel, "competitor masked" and "operand isolated" produce
// the same image for the real single-line layout — a built-in determinism
// check; they diverge only for the synthetic two-line scene.
func renderRealAssocConditions(prov parrotlab.PageProvider, bank *GlyphBank, rdBase R1DBase, competitor string) ([3]*image.RGBA, error) {
	var out [3]*image.RGBA
	geo, err := DeriveR1DGeometry(rdBase)
	if err != nil {
		return out, err
	}
	pagePNG, err := prov.RenderPNG(rdBase.Page)
	if err != nil {
		return out, fmt.Errorf("render page %d: %w", rdBase.Page, err)
	}
	vp, err := BuildR1DViewport(pagePNG, rdBase, geo)
	if err != nil {
		return out, err
	}
	labelCued := cloneRGBA(vp)
	drawR1DCue(labelCued, geo.LabelBBoxCanvas)

	withComp, _, derr := placeDistractors(labelCued, bank, geo, []string{competitor}, 1)
	if derr != nil {
		return out, fmt.Errorf("place competitor: %w", derr)
	}
	out[0] = withComp                  // GC_REAL_0 — competitor visible
	out[1] = cloneRGBA(labelCued)      // GC_REAL_1 — competitor masked (== plain D0L)
	out[2] = cloneRGBA(labelCued)      // GC_REAL_2 — operand isolated (== plain D0L)
	return out, nil
}

// r1gGap is the inter-sprite pixel gap in the synthetic assoc scenes.
const r1gGap = 22

func compositeSprite(dst *image.RGBA, spr *image.RGBA, ox, oy int) {
	b := spr.Bounds()
	for sy := 0; sy < b.Dy(); sy++ {
		for sx := 0; sx < b.Dx(); sx++ {
			sc := spr.RGBAAt(sx, sy)
			px, py := ox+sx, oy+sy
			if px < 0 || py < 0 || px >= CanvasPx || py >= CanvasPx {
				continue
			}
			cur := dst.RGBAAt(px, py)
			nv := uint8(min2(int(cur.R), int(sc.R)))
			dst.SetRGBA(px, py, color.RGBA{R: nv, G: nv, B: nv, A: 255})
		}
	}
}

func fillRect(dst *image.RGBA, r [4]int, c color.RGBA) {
	for y := r[1]; y < r[3]; y++ {
		for x := r[0]; x < r[2]; x++ {
			if x >= 0 && y >= 0 && x < CanvasPx && y < CanvasPx {
				dst.SetRGBA(x, y, c)
			}
		}
	}
}

// renderSynAssocConditions (G-C SYN): a genuine two-line synthetic scene.
// Line 1 = "<label> <value>" (label cued). Line 2 = "<compLabel> <comp>".
//
//	GC_SYN_0 — both lines drawn (competitor visible, baseline)
//	GC_SYN_1 — both lines drawn, competitor line then painted with maskBG
//	           (same scene extent, competitor removed)
//	GC_SYN_2 — only line 1, re-centred to canvas centre (operand isolated)
func renderSynAssocConditions(bank *GlyphBank, gb R1GBase) ([3]*image.RGBA, error) {
	var out [3]*image.RGBA
	labelSpr, err := RenderNumberSprite(bank, gb.SynLabel)
	if err != nil {
		return out, fmt.Errorf("label sprite %q: %w", gb.SynLabel, err)
	}
	valueSpr, err := RenderNumberSprite(bank, gb.SynValue)
	if err != nil {
		return out, err
	}
	compLabelSpr, err := RenderNumberSprite(bank, gb.SynCompLabel)
	if err != nil {
		return out, err
	}
	compValSpr, err := RenderNumberSprite(bank, gb.SynCompValue)
	if err != nil {
		return out, err
	}
	lw := labelSpr.Bounds().Dx()
	lh := labelSpr.Bounds().Dy()
	line1W := lw + r1gGap + valueSpr.Bounds().Dx()
	cw := compLabelSpr.Bounds().Dx()
	line2W := cw + r1gGap + compValSpr.Bounds().Dx()
	maxH := lh
	if h := valueSpr.Bounds().Dy(); h > maxH {
		maxH = h
	}

	build := func(mode int) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, CanvasPx, CanvasPx))
		fillRect(img, [4]int{0, 0, CanvasPx, CanvasPx}, maskBG)
		y1 := canvasCenter - 40
		if mode == 2 {
			y1 = canvasCenter - maxH/2 // isolated: centre the operand line
		}
		x1 := canvasCenter - line1W/2
		compositeSprite(img, labelSpr, x1, y1)
		compositeSprite(img, valueSpr, x1+lw+r1gGap, y1)
		cue := clampRect([4]int{x1 - 6, y1 - 6, x1 + lw + 6, y1 + lh + 6})
		if mode != 2 {
			y2 := canvasCenter + 40
			x2 := canvasCenter - line2W/2
			compositeSprite(img, compLabelSpr, x2, y2)
			compositeSprite(img, compValSpr, x2+cw+r1gGap, y2)
			if mode == 1 {
				fillRect(img, clampRect([4]int{x2 - 10, y2 - 10, x2 + line2W + 10, y2 + maxH + 10}), maskBG)
			}
		}
		strokeRect(img, image.Rect(cue[0], cue[1], cue[2], cue[3]), cueColor, cueStrokePx)
		return img
	}
	out[0], out[1], out[2] = build(0), build(1), build(2)
	return out, nil
}

// renderCueConditions (G-D): fixed 32 px target-only crop; only the value
// cue geometry changes — tight (baseline, reproduces the R1-D artifact),
// padded (the currently frozen safer rule), none.
func renderCueConditions(prov parrotlab.PageProvider, storeDir string, gb R1GBase) ([3]*image.RGBA, error) {
	var out [3]*image.RGBA
	base := gb.asBase()
	geo, err := DeriveR1BGeometry(storeDir, base)
	if err != nil {
		return out, err
	}
	pagePNG, err := prov.RenderPNG(base.Candidate.Page)
	if err != nil {
		return out, fmt.Errorf("render page %d: %w", base.Candidate.Page, err)
	}
	b4 := geo.Conditions[4] // 32 px
	if b4.NominalLinePx != 32 {
		return out, fmt.Errorf("expected 32 px at ladder index 4, got %v", b4.NominalLinePx)
	}

	// tight value cue: proportional token box only, no pad, mapped to canvas
	c := base.Candidate
	line := c.Line.BBox
	lineH := line.Y2 - line.Y1
	nR := float64(len([]rune(c.Line.Text)))
	lw := line.X2 - line.X1
	tokX1 := line.X1 + lw*float64(c.CharOffsetStart)/nR
	tokX2 := line.X1 + lw*float64(c.CharOffsetEnd)/nR
	s := b4.AffineScale
	tcx, tcy := geo.TargetCenterStore[0], geo.TargetCenterStore[1]
	tx1, ty1 := storeToCanvas(tokX1, line.Y1, tcx, tcy, s)
	tx2, ty2 := storeToCanvas(tokX2, line.Y2, tcx, tcy, s)
	_ = lineH
	tight := b4
	tight.CueBBoxCanvasPx = [4]float64{math.Floor(tx1), math.Floor(ty1), math.Ceil(tx2), math.Ceil(ty2)}
	padded := b4
	none := b4
	none.CueStrokePx = 0

	for i, cond := range []R1BCondGeom{tight, padded, none} {
		img, _, rerr := RenderR1BScale(pagePNG, base, geo, cond)
		if rerr != nil {
			return out, rerr
		}
		out[i] = img
	}
	return out, nil
}
