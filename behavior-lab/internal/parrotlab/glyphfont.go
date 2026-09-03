package parrotlab

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
)

// microISAFontScales are every font scale the microisa_visual generator
// actually renders text at (label 8, number 4, text/entity/labels 3, glyph
// key 5, compare 6, filler/distractors 2).
var microISAFontScales = []int{2, 3, 4, 5, 6, 8}

// RenderGlyphAudit renders every allowed synthetic character at every font
// scale P2 uses, for manual collision inspection before freeze. One row per
// character; scales left to right, each labelled.
func RenderGlyphAudit() []byte {
	scales := microISAFontScales
	chars := append(append([]rune{}, letterAlphabet...), digitAlphabet...)
	rowHeight := glyphHeight*maxScale(scales) + 24
	colWidth := 0
	for _, scale := range scales {
		colWidth += glyphWidth*scale + 40
	}
	width := 120 + colWidth
	height := 60 + rowHeight*len(chars)
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	fill(canvas, color.RGBA{255, 255, 255, 255})
	ink := color.RGBA{20, 20, 20, 255}

	drawText(canvas, "GLYPH AUDIT  LETTERS "+string(letterAlphabet)+"  DIGITS "+string(digitAlphabet), image.Point{X: 16, Y: 16}, 2, ink)
	for rowIndex, symbol := range chars {
		top := 50 + rowIndex*rowHeight
		drawText(canvas, string(symbol), image.Point{X: 16, Y: top}, 3, ink)
		cursorX := 110
		for _, scale := range scales {
			drawText(canvas, string(symbol), image.Point{X: cursorX, Y: top}, scale, ink)
			cursorX += glyphWidth*scale + 40
		}
	}
	var out bytes.Buffer
	_ = png.Encode(&out, canvas)
	return out.Bytes()
}

func maxScale(scales []int) int {
	best := 0
	for _, scale := range scales {
		if scale > best {
			best = scale
		}
	}
	return best
}

// A dependency-free 5x7 bitmap font, in the same spirit as scenegen.go's
// hand-rolled A/B glyphs but covering the alphabet the microisa_visual
// generator needs. Only the microisa_visual generator uses it;
// instruction_cliff rendering (frozen) is untouched.
//
// The letter alphabet is the full A-Z: letters never share a rendered string
// with digits in this generator, so O/0-style cross-confusion cannot occur.
// The digit alphabet is deliberately restricted to {2,3,4,5,6,7,9} by the
// generator (0/1/8 dropped) for within-digit legibility; the glyphs for the
// dropped digits still exist here for completeness.

var bitmapFont = map[rune][7]string{
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'B': {"11110", "10001", "11110", "10001", "10001", "10001", "11110"},
	'C': {"01110", "10001", "10000", "10000", "10000", "10001", "01110"},
	'D': {"11100", "10010", "10001", "10001", "10001", "10010", "11100"},
	'E': {"11111", "10000", "11110", "10000", "10000", "10000", "11111"},
	'F': {"11111", "10000", "11110", "10000", "10000", "10000", "10000"},
	'G': {"01110", "10001", "10000", "10111", "10001", "10001", "01111"},
	'H': {"10001", "10001", "11111", "10001", "10001", "10001", "10001"},
	'I': {"11111", "00100", "00100", "00100", "00100", "00100", "11111"},
	'J': {"00111", "00010", "00010", "00010", "10010", "10010", "01100"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N': {"10001", "11001", "10101", "10011", "10001", "10001", "10001"},
	'O': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'Q': {"01110", "10001", "10001", "10001", "10101", "10010", "01101"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V': {"10001", "10001", "10001", "10001", "10001", "01010", "00100"},
	'W': {"10001", "10001", "10001", "10101", "10101", "11011", "10001"},
	'X': {"10001", "10001", "01010", "00100", "01010", "10001", "10001"},
	'Y': {"10001", "10001", "01010", "00100", "00100", "00100", "00100"},
	'Z': {"11111", "00001", "00010", "00100", "01000", "10000", "11111"},
	'0': {"01110", "10011", "10011", "10101", "11001", "11001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11111", "00010", "00100", "00010", "00001", "10001", "01110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "11110", "00001", "00001", "10001", "01110"},
	'6': {"00110", "01000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00010", "01100"},
	' ': {"00000", "00000", "00000", "00000", "00000", "00000", "00000"},
	'-': {"00000", "00000", "00000", "11111", "00000", "00000", "00000"},
	'.': {"00000", "00000", "00000", "00000", "00000", "01100", "01100"},
	',': {"00000", "00000", "00000", "00000", "00000", "00100", "01000"},
	'?': {"01110", "10001", "00010", "00100", "00100", "00000", "00100"},
	'!': {"00100", "00100", "00100", "00100", "00100", "00000", "00100"},
	':': {"00000", "01100", "01100", "00000", "01100", "01100", "00000"},
}

const (
	glyphWidth  = 5
	glyphHeight = 7
)

// letterAlphabet and digitAlphabet are the CONSERVATIVE, visually
// unambiguous character sets the synthetic generator samples from for
// READ_SHORT_LABEL / READ_SHORT_TEXT strings (parrot-microisa-r0.1, after
// the r0 abort for SYNTHETIC_GLYPH_AMBIGUITY). Every excluded letter/digit
// was dropped either as a known confusable pair (O/0, I/1/l, B/8, S/5,
// G/6/9, Z/2) or after the glyph-audit sheet inspection. Letters and digits
// are never mixed within one synthetic reading string.
var (
	letterAlphabet = []rune("ACDEFHKMNPRTUXY")
	digitAlphabet  = []rune("234679")
)

// textPixelWidth is the inked width of s at the given scale (no trailing
// advance after the last glyph).
func textPixelWidth(text string, scale int) int {
	runes := []rune(text)
	if len(runes) == 0 {
		return 0
	}
	return len(runes)*glyphWidth*scale + (len(runes)-1)*scale
}

// drawText renders text with its top-left corner at origin and returns the
// bounding rectangle of the drawn cell area (origin to origin + size).
// Unknown runes render as blank cells.
func drawText(canvas *image.RGBA, text string, origin image.Point, scale int, shade color.RGBA) image.Rectangle {
	if scale < 1 {
		scale = 1
	}
	cursorX := origin.X
	for _, symbol := range strings.ToUpper(text) {
		rows, ok := bitmapFont[symbol]
		if !ok {
			rows = bitmapFont[' ']
		}
		for rowIndex, row := range rows {
			for columnIndex, cell := range row {
				if cell != '1' {
					continue
				}
				for py := 0; py < scale; py++ {
					for px := 0; px < scale; px++ {
						canvas.SetRGBA(cursorX+columnIndex*scale+px, origin.Y+rowIndex*scale+py, shade)
					}
				}
			}
		}
		cursorX += (glyphWidth + 1) * scale
	}
	width := textPixelWidth(text, scale)
	return image.Rect(origin.X, origin.Y, origin.X+width, origin.Y+glyphHeight*scale)
}

// drawRectOutline strokes a `thickness`-pixel border along the edge of rect.
func drawRectOutline(canvas *image.RGBA, rect image.Rectangle, thickness int, shade color.RGBA) {
	if thickness < 1 {
		thickness = 1
	}
	for offset := 0; offset < thickness; offset++ {
		for x := rect.Min.X - offset; x <= rect.Max.X+offset; x++ {
			canvas.SetRGBA(x, rect.Min.Y-offset, shade)
			canvas.SetRGBA(x, rect.Max.Y+offset, shade)
		}
		for y := rect.Min.Y - offset; y <= rect.Max.Y+offset; y++ {
			canvas.SetRGBA(rect.Min.X-offset, y, shade)
			canvas.SetRGBA(rect.Max.X+offset, y, shade)
		}
	}
}

// drawLine strokes a 1px line between two points (Bresenham).
func drawLine(canvas *image.RGBA, from, to image.Point, shade color.RGBA) {
	deltaX := absInt(to.X - from.X)
	deltaY := -absInt(to.Y - from.Y)
	stepX := signInt(to.X - from.X)
	stepY := signInt(to.Y - from.Y)
	err := deltaX + deltaY
	x, y := from.X, from.Y
	for {
		canvas.SetRGBA(x, y, shade)
		if x == to.X && y == to.Y {
			return
		}
		doubled := 2 * err
		if doubled >= deltaY {
			err += deltaY
			x += stepX
		}
		if doubled <= deltaX {
			err += deltaX
			y += stepY
		}
	}
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func signInt(value int) int {
	switch {
	case value > 0:
		return 1
	case value < 0:
		return -1
	default:
		return 0
	}
}
