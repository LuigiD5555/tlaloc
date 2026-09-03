package perceptenvelope

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/parrotlab"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

// R1-C SYNTHETIC_REALISTIC renderer — glyph bank cut from the real D2L
// corpus (no external font dependency; typography is the corpus's own).
//
// Each needed character is sampled from its first clean occurrence in a
// deterministic page sweep, scaled so the containing line is 32 px (the
// R1-C presentation scale), and stored. RenderSyntheticNumber composites
// those glyphs left-to-right with a channel-wise darken blend over the
// frozen RGB(200,200,200) neutral canvas — no keying threshold, no
// contrast alteration, no typographic tuning.

// R1CGlyphBankVersion is bumped when the extraction algorithm changes.
const R1CGlyphBankVersion = "r1c.glyphbank.1"

// r1cNeededGlyphs is the frozen character set the synthetic strata need.
const r1cNeededGlyphs = "0123456789.,%+-()[]ex= "

// Glyph is one extracted character raster at the R1-C 32 px line scale.
type Glyph struct {
	Char            string  `json:"char"`
	SourcePage      int     `json:"source_page"`
	SourceRegionID  string  `json:"source_region_id"`
	SourceRuneIndex int     `json:"source_rune_index"`
	LineHeightStore float64 `json:"line_height_store"`
	WidthPx         int     `json:"width_px"`
	HeightPx        int     `json:"height_px"`
	AdvancePx       float64 `json:"advance_px"`
	PixelsB64       string  `json:"pixels_b64"` // base64 grayscale W*H, row-major
	pixels          []byte  // decoded cache
}

func (g *Glyph) encode() { g.PixelsB64 = base64.StdEncoding.EncodeToString(g.pixels) }
func (g *Glyph) decode() {
	if g.pixels == nil && g.PixelsB64 != "" {
		g.pixels, _ = base64.StdEncoding.DecodeString(g.PixelsB64)
	}
}

// LoadGlyphBank reads a serialized glyph bank and decodes its rasters.
func LoadGlyphBank(path string) (*GlyphBank, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var bank GlyphBank
	if err := json.Unmarshal(body, &bank); err != nil {
		return nil, err
	}
	for k, g := range bank.Glyphs {
		g.decode()
		bank.Glyphs[k] = g
	}
	return &bank, nil
}

// LoadOrBuildGlyphBank loads the cached bank at path, or builds it from the
// store and writes it if the cache is absent, a different version, or
// missing raster data.
func LoadOrBuildGlyphBank(path, storeDir, pdfPath string) (*GlyphBank, error) {
	if b, err := LoadGlyphBank(path); err == nil && b.Version == R1CGlyphBankVersion && len(b.Glyphs) > 0 && glyphBankHasRasters(b) {
		return b, nil
	}
	b, err := BuildGlyphBank(storeDir, pdfPath)
	if err != nil {
		return nil, err
	}
	if _, err := WriteJSON(path, b); err != nil {
		return nil, err
	}
	return b, nil
}

func glyphBankHasRasters(b *GlyphBank) bool {
	for c, g := range b.Glyphs {
		if c == " " {
			continue
		}
		if len(g.pixels) == 0 {
			return false
		}
	}
	return true
}

// GlyphBank is the frozen synthetic-render glyph set.
type GlyphBank struct {
	Version    string           `json:"version"`
	Seed       string           `json:"seed"`
	CanvasPx   int              `json:"canvas_px"`
	LineHeight float64          `json:"line_height_px"`
	Glyphs     map[string]Glyph `json:"glyphs"`
	SHA256     string           `json:"sha256"`
}

// r1cGlyphBankPageBudget caps the deterministic page sweep so glyph
// extraction stays fast on large stores.
const r1cGlyphBankPageBudget = 220

// BuildGlyphBank extracts every needed glyph from the store's rendered
// pages. Deterministic. Errors if any needed glyph cannot be found.
//
// The sweep is the first r1cGlyphBankPageBudget pages (ascending) that
// carry a layout — a fixed, corpus-order subset, never chosen by model
// performance.
func BuildGlyphBank(storeDir, pdfPath string) (*GlyphBank, error) {
	manifest, _, err := pdfmemory.Load(storeDir)
	if err != nil {
		return nil, err
	}
	prov, err := parrotlab.NewPDFMemoryProvider(storeDir, pdfPath)
	if err != nil {
		return nil, err
	}
	allPages := append([]pdfmemory.PageRef(nil), manifest.Pages...)
	sort.Slice(allPages, func(i, j int) bool { return allPages[i].Number < allPages[j].Number })
	var pages []pdfmemory.PageRef
	for _, pr := range allPages {
		if pr.LayoutPath == "" {
			continue
		}
		pages = append(pages, pr)
		if len(pages) >= r1cGlyphBankPageBudget {
			break
		}
	}

	need := map[rune]bool{}
	for _, r := range r1cNeededGlyphs {
		need[r] = true
	}
	bank := &GlyphBank{
		Version: R1CGlyphBankVersion, Seed: R1CSeed, CanvasPx: CanvasPx,
		LineHeight: R1CLineHeightPx, Glyphs: map[string]Glyph{},
	}
	pageImgCache := map[int]image.Image{}

	// priority passes, cleanest source first:
	//   0: very short region (<=2 runes) — near-zero neighbour bleed
	//   1: char flanked by non-letters
	//   2: any occurrence
	type pass struct {
		maxRunes int
		isolated bool
	}
	for _, ps := range []pass{{2, false}, {1 << 30, true}, {1 << 30, false}} {
		requireIsolated := ps.isolated
		maxRunes := ps.maxRunes
		for _, pref := range pages {
			if len(need) == 0 {
				break
			}
			if pref.LayoutPath == "" {
				continue
			}
			page, _, lerr := loadStorePage(storeDir, pref.Number)
			if lerr != nil {
				continue
			}
			var src image.Image
			for _, region := range page.Regions {
				switch region.Kind {
				case "text_line", "list_item", "equation_or_code", "heading":
				default:
					continue
				}
				text := region.Text
				runes := []rune(text)
				if len(runes) == 0 || len(runes) > maxRunes || region.BBox.X2 <= region.BBox.X1 || region.BBox.Y2 <= region.BBox.Y1 {
					continue
				}
				lineH := region.BBox.Y2 - region.BBox.Y1
				lineW := region.BBox.X2 - region.BBox.X1
				charW := lineW / float64(len(runes))
				for i, r := range runes {
					if !need[r] {
						continue
					}
					if requireIsolated && r != ' ' {
						notLetter := func(x rune) bool {
							return !((x >= 'a' && x <= 'z') || (x >= 'A' && x <= 'Z'))
						}
						leftOK := i == 0 || notLetter(runes[i-1])
						rightOK := i == len(runes)-1 || notLetter(runes[i+1])
						if !leftOK || !rightOK {
							continue
						}
					}
					if r == ' ' {
						bank.Glyphs[" "] = Glyph{
							Char: " ", SourcePage: pref.Number, SourceRegionID: region.ID,
							SourceRuneIndex: i, LineHeightStore: lineH,
							AdvancePx: charW * (R1CLineHeightPx / lineH),
						}
						delete(need, r)
						continue
					}
					if src == nil {
						if c, ok := pageImgCache[pref.Number]; ok {
							src = c
						} else {
							b, rerr := prov.RenderPNG(pref.Number)
							if rerr != nil {
								break
							}
							im, derr := png.Decode(bytes.NewReader(b))
							if derr != nil {
								break
							}
							src = im
							pageImgCache[pref.Number] = im
						}
					}
					g, gerr := cutGlyph(src, page.Width, region.BBox, charW, float64(i), lineH, string(r))
					if gerr != nil {
						continue
					}
					g.SourcePage = pref.Number
					g.SourceRegionID = region.ID
					g.SourceRuneIndex = i
					bank.Glyphs[string(r)] = g
					delete(need, r)
				}
			}
		}
	}

	if len(need) > 0 {
		var missing []string
		for r := range need {
			missing = append(missing, fmt.Sprintf("%q", string(r)))
		}
		sort.Strings(missing)
		return nil, fmt.Errorf("glyph bank incomplete: missing %s", strings.Join(missing, " "))
	}
	bank.SHA256 = hashGlyphBank(bank)
	for k, g := range bank.Glyphs {
		g.encode()
		bank.Glyphs[k] = g
	}
	return bank, nil
}

// cutGlyph samples one character's proportional x-slice from the rendered
// page and rescales it so the containing line is R1CLineHeightPx tall.
func cutGlyph(src image.Image, pageWidth float64, bb canonicaldoc.BBox, charW, runeIdx, lineH float64, ch string) (Glyph, error) {
	sb := src.Bounds()
	k := float64(sb.Dx()) / pageWidth // rendered-page px per store unit
	s := R1CLineHeightPx / lineH      // store -> glyph-canvas scale
	x1 := bb.X1 + charW*runeIdx
	y1 := bb.Y1
	outW := int(math.Ceil(charW * s))
	outH := int(math.Ceil(lineH * s))
	if outW < 1 || outH < 1 || outW > 256 || outH > 256 {
		return Glyph{}, fmt.Errorf("degenerate glyph box %dx%d", outW, outH)
	}
	// oversample the slice a little wider so a glyph that sits slightly
	// off its proportional cell is still fully captured; ink-bbox trim
	// then removes neighbour bleed.
	padX := 0.25 * charW
	full := make([]byte, 0)
	fw := int(math.Ceil((charW + 2*padX) * s))
	fh := outH
	full = make([]byte, fw*fh)
	const inkThresh = 150
	minX, minY, maxX, maxY := fw, fh, -1, -1
	for oy := 0; oy < fh; oy++ {
		for ox := 0; ox < fw; ox++ {
			storeX := x1 - padX + float64(ox)/s
			storeY := y1 + float64(oy)/s
			col, ok := bilinearSample(src, float64(sb.Min.X)+storeX*k, float64(sb.Min.Y)+storeY*k)
			lum := byte(255)
			if ok {
				lum = byte((int(col.R)*299 + int(col.G)*587 + int(col.B)*114) / 1000)
			}
			full[oy*fw+ox] = lum
			if lum < inkThresh {
				if ox < minX {
					minX = ox
				}
				if ox > maxX {
					maxX = ox
				}
				if oy < minY {
					minY = oy
				}
				if oy > maxY {
					maxY = oy
				}
			}
		}
	}
	_ = minX
	_ = maxX
	_ = minY
	_ = maxY
	// column ink profile: isolate the single glyph around the cell centre
	// by walking outward and stopping at a >=3-column empty gap. This
	// drops bled neighbour characters even when the proportional cell
	// estimate is off.
	colInk := make([]bool, fw)
	anyInk := false
	for ox := 0; ox < fw; ox++ {
		for oy := 0; oy < fh; oy++ {
			if full[oy*fw+ox] < inkThresh {
				colInk[ox] = true
				anyInk = true
				break
			}
		}
	}
	if !anyInk {
		if ch == " " {
			return Glyph{Char: ch, LineHeightStore: lineH, AdvancePx: charW * s}, nil
		}
		return Glyph{}, fmt.Errorf("no ink in %q slice", ch)
	}
	center := int(math.Round((padX + charW/2) * s))
	if center >= fw {
		center = fw - 1
	}
	// snap centre to the nearest inked column
	if !colInk[center] {
		for d := 1; d < fw; d++ {
			if center-d >= 0 && colInk[center-d] {
				center -= d
				break
			}
			if center+d < fw && colInk[center+d] {
				center += d
				break
			}
		}
	}
	lo, hi := center, center
	for lo > 0 {
		gap := 0
		for lo-1-gap >= 0 && !colInk[lo-1-gap] {
			gap++
		}
		if gap >= 3 || lo-1-gap < 0 {
			break
		}
		lo = lo - 1 - gap
	}
	for hi < fw-1 {
		gap := 0
		for hi+1+gap < fw && !colInk[hi+1+gap] {
			gap++
		}
		if gap >= 3 || hi+1+gap >= fw {
			break
		}
		hi = hi + 1 + gap
	}
	m := 1
	minX = maxi(0, lo-m)
	maxX = mini(fw-1, hi+m)
	minY, maxY = 0, fh-1
	gw, gh := maxX-minX+1, maxY-minY+1
	pix := make([]byte, gw*gh)
	for gy := 0; gy < gh; gy++ {
		for gx := 0; gx < gw; gx++ {
			pix[gy*gw+gx] = full[(minY+gy)*fw+(minX+gx)]
		}
	}
	tracking := int(math.Round(0.09 * R1CLineHeightPx))
	return Glyph{
		Char: ch, LineHeightStore: lineH,
		WidthPx: gw, HeightPx: gh, AdvancePx: float64(gw + tracking),
		pixels: pix,
	}, nil
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func mini(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func hashGlyphBank(b *GlyphBank) string {
	h := sha256.New()
	var chars []string
	for c := range b.Glyphs {
		chars = append(chars, c)
	}
	sort.Strings(chars)
	for _, c := range chars {
		g := b.Glyphs[c]
		fmt.Fprintf(h, "%s|%d|%s|%d|%d|%d|%.4f\n", g.Char, g.SourcePage, g.SourceRegionID, g.SourceRuneIndex, g.WidthPx, g.HeightPx, g.AdvancePx)
		h.Write(g.pixels)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// RenderSyntheticNumber composites the glyph bank into a 512x512 R1-C
// canvas for one target string. Returns the image and the cue bbox.
func RenderSyntheticNumber(bank *GlyphBank, target string) (*image.RGBA, [4]int, error) {
	if bank == nil {
		return nil, [4]int{}, fmt.Errorf("nil glyph bank")
	}
	runes := []rune(target)
	total := 0.0
	maxH := 0
	for _, r := range runes {
		g, ok := bank.Glyphs[string(r)]
		if !ok {
			return nil, [4]int{}, fmt.Errorf("no glyph for %q in target %q", string(r), target)
		}
		if r == ' ' {
			total += g.AdvancePx
			continue
		}
		if len(g.pixels) != g.WidthPx*g.HeightPx || g.WidthPx == 0 {
			return nil, [4]int{}, fmt.Errorf("glyph %q has no raster (rebuild the glyph bank)", string(r))
		}
		total += g.AdvancePx + 2
		if g.HeightPx > maxH {
			maxH = g.HeightPx
		}
	}
	out := image.NewRGBA(image.Rect(0, 0, CanvasPx, CanvasPx))
	for y := 0; y < CanvasPx; y++ {
		for x := 0; x < CanvasPx; x++ {
			out.SetRGBA(x, y, maskBG)
		}
	}
	startX := float64(canvasCenter) - total/2
	topY := float64(canvasCenter) - float64(maxH)/2
	curX := startX
	for _, r := range runes {
		g := bank.Glyphs[string(r)]
		if r == ' ' {
			curX += g.AdvancePx
			continue
		}
		ox := int(math.Round(curX))
		oy := int(math.Round(topY + float64(maxH-g.HeightPx)/2))
		for gy := 0; gy < g.HeightPx; gy++ {
			for gx := 0; gx < g.WidthPx; gx++ {
				lum := g.pixels[gy*g.WidthPx+gx]
				px, py := ox+gx, oy+gy
				if px < 0 || py < 0 || px >= CanvasPx || py >= CanvasPx {
					continue
				}
				cur := out.RGBAAt(px, py)
				// channel-wise darken blend (black text on any ground)
				nv := uint8(min2(int(cur.R), int(lum)))
				out.SetRGBA(px, py, color.RGBA{R: nv, G: nv, B: nv, A: 255})
			}
		}
		curX += g.AdvancePx + 2
	}
	cue := [4]int{
		int(startX) - 6, int(topY) - 6,
		int(startX+total) + 6, int(topY) + maxH + 6,
	}
	cue = clampRect(cue)
	strokeRect(out, image.Rect(cue[0], cue[1], cue[2], cue[3]), cueColor, cueStrokePx)
	return out, cue, nil
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// EncodeGlyphBankPreview renders every glyph side by side for sanity
// inspection.
func EncodeGlyphBankPreview(bank *GlyphBank) ([]byte, error) {
	var chars []string
	for c := range bank.Glyphs {
		if c == " " {
			continue
		}
		chars = append(chars, c)
	}
	sort.Strings(chars)
	pad := 6
	w, h := pad, 0
	for _, c := range chars {
		g := bank.Glyphs[c]
		w += g.WidthPx + pad
		if g.HeightPx > h {
			h = g.HeightPx
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h+2*pad))
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, maskBG)
		}
	}
	cx := pad
	for _, c := range chars {
		g := bank.Glyphs[c]
		for gy := 0; gy < g.HeightPx; gy++ {
			for gx := 0; gx < g.WidthPx; gx++ {
				nv := g.pixels[gy*g.WidthPx+gx]
				img.SetRGBA(cx+gx, pad+gy, color.RGBA{R: nv, G: nv, B: nv, A: 255})
			}
		}
		cx += g.WidthPx + pad
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
