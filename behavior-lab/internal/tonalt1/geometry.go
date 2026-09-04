package tonalt1

import (
	"strings"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
)

// Frozen geometry / ambiguity rules (GeometryRuleVersion).
//
// T1 cue geometry derives from the PUBLIC LocatedRegion (the containing
// layout line), NOT the historical char-offset target geometry — a
// recorded protocol limitation. Every rule here defends the isolated
// single-operand read against that limitation and is machine-decidable,
// declared before any T1 inference. There is NO manual per-instance
// geometry repair.

// paddingPolicy mirrors the frozen R1 cue-padding rule so the T1 render
// path (parrotpresent / RenderR1BScale) reproduces R1/H presentation.
const paddingPolicy = "pad = 0.5 * line_bbox_height on each side (x and y); token x-interval = proportional rune-offset split of line bbox; y-interval = full line bbox; then clamp to page bounds"

const tokenBoxMethod = "PROPORTIONAL_RUNE_OFFSET_SPLIT_OF_LINE_BBOX (no per-word bbox in store; estimate only, not a Tlaloc perception capability)"

// r1RenderCanvasPx is the frozen R1/H render canvas edge (512x512,
// target-centred, bilinear inverse map).
const r1RenderCanvasPx = 512.0

const (
	marginXLo = 0.10
	marginXHi = 0.90
	marginYLo = 0.09
	marginYHi = 0.91

	minLineWidthFraction = 0.30
	maxCueLineFraction   = 0.85
)

// geometryVerdict is the outcome of the geometry / ambiguity audit for one
// candidate location.
type geometryVerdict struct {
	rejections          []RejectionCode
	operandBBoxEstimate canonicaldoc.BBox
	cueBBoxStore        canonicaldoc.BBox
	effectiveScale      float64
	whitespaceFields    int
}

// auditGeometry runs every frozen geometry / ambiguity rule against a
// containing line and the located token. tokenStart/tokenEnd are rune
// offsets in lineText.
func auditGeometry(page canonicaldoc.Page, line canonicaldoc.Region, verbatimToken string, tokenStart, tokenEnd int, fields []string) geometryVerdict {
	var verdict geometryVerdict
	width, height := page.Width, page.Height
	box := line.BBox
	verdict.whitespaceFields = len(fields)

	add := func(code RejectionCode) { verdict.rejections = append(verdict.rejections, code) }

	// Line geometry well-formed.
	if !(box.X1 >= 0 && box.X1 < box.X2 && box.X2 <= width && box.Y1 >= 0 && box.Y1 < box.Y2 && box.Y2 <= height) {
		add(RejectGeometryMalformed)
		return verdict
	}
	// AUTHORING_HEURISTIC (advisory, not blocking): page-margin and
	// prose-width line preferences from the R1-A/R1-B pool author.
	if !(box.X1 >= marginXLo*width && box.X2 <= marginXHi*width && box.Y1 >= marginYLo*height && box.Y2 <= marginYHi*height) {
		add(RejectLineInPageMargin)
	}
	if (box.X2 - box.X1) < minLineWidthFraction*width {
		add(RejectLineTooNarrow)
	}
	// DOMAIN: number-leading line — overwhelmingly a TOC / section-heading
	// / numbered-list entry where the number is a locator, not a quantity.
	if len(fields) > 0 && fields[0] == verbatimToken {
		add(RejectNumberLeadingLine)
	}

	// Unique substring location of the token.
	if strings.Count(line.Text, verbatimToken) != 1 {
		add(RejectTokenOffsetNotUnique)
	}

	runes := []rune(line.Text)
	total := len(runes)
	if total == 0 || tokenStart < 0 || tokenEnd > total || tokenStart >= tokenEnd {
		add(RejectGeometryMalformed)
		return verdict
	}

	lineWidth := box.X2 - box.X1
	estX1 := box.X1 + lineWidth*float64(tokenStart)/float64(total)
	estX2 := box.X1 + lineWidth*float64(tokenEnd)/float64(total)
	verdict.operandBBoxEstimate = canonicaldoc.BBox{X1: estX1, Y1: box.Y1, X2: estX2, Y2: box.Y2}

	// Operand inclusion: the estimated token box sits strictly inside the
	// located line box in x.
	if !(estX1 >= box.X1 && estX2 <= box.X2 && estX1 < estX2) {
		add(RejectOperandNotIncluded)
	}
	// Cue not implausibly large relative to the line.
	if len(fields) > 2 && (estX2-estX1) > maxCueLineFraction*lineWidth {
		add(RejectCueImplausible)
	}

	pad := 0.5 * (box.Y2 - box.Y1)
	cue := canonicaldoc.BBox{X1: estX1 - pad, Y1: box.Y1 - pad, X2: estX2 + pad, Y2: box.Y2 + pad}
	verdict.cueBBoxStore = cue
	if !(cue.X1 > 0 && cue.Y1 > 0 && cue.X2 < width && cue.Y2 < height && cue.X1 < cue.X2 && cue.Y1 < cue.Y2) {
		add(RejectPaddedBoxClipped)
	}

	if lineHeight := box.Y2 - box.Y1; lineHeight > 0 {
		verdict.effectiveScale = r1RenderCanvasPx / lineHeight
	}
	return verdict
}

// GeometryRuleSummary is the frozen, human-readable rule list recorded in
// the selector manifest.
var GeometryRuleSummary = []string{
	"cue geometry derives from the public LocatedRegion (containing layout line), not historical char-offset target geometry (recorded limitation)",
	"containing line must have EXACTLY ONE digit-bearing whitespace token (competing numeric tokens break a LocatedRegion-derived cue)",
	"line geometry well-formed: 0 <= x1 < x2 <= w, 0 <= y1 < y2 <= h",
	"line bbox fully inside [0.10w,0.90w] x [0.09h,0.91h] (not page margin)",
	"line width >= 0.30w (prose line)",
	"line has >= 4 whitespace tokens (embedded prose)",
	"reject number-leading lines (fields[0] == token)",
	"reject cross-references: token immediately preceded by section/figure/table/equation/chapter/etc. (locator, not a quantity)",
	"token located uniquely by substring in the line text",
	"estimated proportional token box strictly inside the line box in x (operand inclusion)",
	"estimated token width <= 0.85 * line width when the line has > 2 whitespace tokens",
	"padded cue box (0.5 * line height each side) strictly inside the page",
	"render: 512x512 canvas, target-centred, bilinear inverse map (frozen parrotpresent / RenderR1BScale)",
}
