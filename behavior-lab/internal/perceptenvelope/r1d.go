package perceptenvelope

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/parrotlab"
	"tlaloc.local/behaviorlab/internal/target"
)

// R1-D LABEL/VALUE ASSOCIATION + DISTRACTOR DENSITY.
//
// D0 measures whether LFM2-VL can return the numeric VALUE spatially
// associated with a marked textual LABEL, in real single-line document
// layout, with the number's morphology already known readable
// (MULTI_DIGIT_INTEGER only). D1 measures the causal effect of controlled
// numeric distractor density on that association, using a CONTROLLED_
// COMPOSITE track that is never pooled with D0.

// R1DAssocOpcode is the R1-D association operation. It is defined in the
// behaviour lab (not promoted to the T0 Micro-ISA vocabulary yet — that is
// a separate decision recorded in R1_PROTOCOL_ADDENDUM_05).
const R1DAssocOpcode = "READ_ASSOCIATED_NUMBER"

// R1DAssocInstruction is the frozen one-op instruction for the association
// task: it names no label text and no expected value.
const R1DAssocInstruction = "Return only the number associated with the marked text. Reply with only the number."

// R1DLineHeightPx is the frozen R1-D containing-line height (the R1-B
// nominal high-reliability presentation point, matching R1-C).
const R1DLineHeightPx = 32.0

// R1DSeed is the frozen deterministic selection / distractor seed.
const R1DSeed = Seed

// R1DFitBudgetPx is the max label↔value canvas span for a pair to count as
// "both visible" inside the 512 canvas with margin.
const R1DFitBudgetPx = 480.0

// R1DDistractorLadder is the frozen distractor-count ladder for D1.
var R1DDistractorLadder = []struct {
	ID string
	K  int
}{
	{"D1K0", 0}, {"D1K1", 1}, {"D1K2", 2}, {"D1K4", 4}, {"D1K8", 8},
}

var r1dStopwords = map[string]bool{
	"the": true, "a": true, "an": true, "of": true, "is": true, "are": true, "was": true,
	"were": true, "to": true, "in": true, "on": true, "at": true, "by": true, "for": true,
	"with": true, "that": true, "this": true, "and": true, "or": true, "as": true, "it": true,
	"be": true, "we": true, "you": true, "they": true, "its": true,
}

// R1DBase is one allocated label/value stimulus.
type R1DBase struct {
	BaseID           string            `json:"base_id"`
	CandidateID      string            `json:"candidate_id"`
	Page             int               `json:"page"`
	Label            string            `json:"label"`
	Value            string            `json:"value"`
	LineText         string            `json:"line_text"`
	Pattern          string            `json:"pattern"`
	LabelRuneStart   int               `json:"label_rune_start"`
	LabelRuneEnd     int               `json:"label_rune_end"`
	ValueRuneStart   int               `json:"value_rune_start"`
	ValueRuneEnd     int               `json:"value_rune_end"`
	Eligible         bool              `json:"eligible"`
	IneligibleReason string            `json:"ineligible_reason,omitempty"`
	LineBBox         canonicaldoc.BBox `json:"line_bbox"`
	PageWidth        float64           `json:"page_width"`
	PageHeight       float64           `json:"page_height"`
	DistractorValues []string          `json:"distractor_values"`
	RankKey          string            `json:"rank_key"`
}

// R1DGeometry is the frozen per-base transform (canvas pixels).
type R1DGeometry struct {
	BaseID          string     `json:"base_id"`
	AffineScale     float64    `json:"affine_scale_store_to_canvas"`
	LineHeightPx    float64    `json:"line_height_canvas_px"`
	CentreStore     [2]float64 `json:"centre_store"`
	LineRectCanvas  [4]int     `json:"line_rect_canvas_px"`
	LabelBBoxCanvas [4]int     `json:"label_bbox_canvas_px"`
	ValueBBoxCanvas [4]int     `json:"value_bbox_canvas_px"`
	LabelSpanPx     float64    `json:"label_to_value_span_px"`
}

// R1DAllocation is the frozen R1-D dataset.
type R1DAllocation struct {
	Schema           string    `json:"schema"`
	ExperimentID     string    `json:"experiment_id"`
	Seed             string    `json:"seed"`
	RankRule         string    `json:"rank_rule"`
	LineHeightPx     float64   `json:"line_height_px"`
	CanvasPx         int       `json:"canvas_px"`
	AssocOpcode      string    `json:"assoc_opcode"`
	AssocInstruction string    `json:"assoc_instruction"`
	PoolCount        int       `json:"pool_candidate_count"`
	EligibleCount    int       `json:"eligible_count"`
	MinRequired      int       `json:"min_required"`
	Proceed          bool      `json:"proceed"`
	Bases            []R1DBase `json:"bases"`
}

const r1dAllocSchema = "tlaloc.parrot-perceptual-envelope-r1.r1d-allocation.r1"

func rankKeyR1D(candidateID string) string {
	sum := sha256.Sum256([]byte(R1DSeed + "|" + candidateID))
	return hex.EncodeToString(sum[:])
}

func runeIdxOf(s string, byteIdx int) int { return len([]rune(s[:byteIdx])) }

// AllocateR1D applies the frozen deterministic selection + §3/§4 eligibility
// filter. No model output is read.
func AllocateR1D(pool LabelValuePool) R1DAllocation {
	alloc := R1DAllocation{
		Schema: r1dAllocSchema, ExperimentID: ExperimentID, Seed: R1DSeed,
		RankRule:     "sha256(seed || candidate_id) ascending",
		LineHeightPx: R1DLineHeightPx, CanvasPx: CanvasPx,
		AssocOpcode: R1DAssocOpcode, AssocInstruction: R1DAssocInstruction,
		PoolCount: len(pool.Candidates), MinRequired: 18,
	}
	cands := append([]LabelValueCandidate(nil), pool.Candidates...)
	sort.Slice(cands, func(i, j int) bool {
		return rankKeyR1D(cands[i].CandidateID) < rankKeyR1D(cands[j].CandidateID)
	})
	for i, c := range cands {
		b := R1DBase{
			BaseID:      fmt.Sprintf("r1d-%02d-%s", i+1, c.CandidateID[:8]),
			CandidateID: c.CandidateID, Page: c.Page, Label: c.Label, Value: c.Value,
			LineText: c.LineText, Pattern: c.Pattern, LineBBox: c.LineBBox,
			PageWidth: c.PageWidth, PageHeight: c.PageHeight, RankKey: rankKeyR1D(c.CandidateID),
		}
		reason := ""
		// value: plain 2-5 digit int
		if !isPlainInt(c.Value) || len(c.Value) < 2 || len(c.Value) > 5 {
			reason = "value not a 2-5 digit plain integer"
		}
		vIdx := strings.Index(c.LineText, c.Value)
		lIdx := strings.Index(c.LineText, c.Label)
		if reason == "" && (vIdx < 0 || lIdx < 0 || lIdx >= vIdx) {
			reason = "label does not precede value in line text"
		}
		if reason == "" && strings.Count(c.LineText, c.Value) != 1 {
			reason = "value not unique in line text"
		}
		if reason == "" && !labelHasContentWord(c.Label) {
			reason = "label has no non-stopword token of >=3 letters"
		}
		if reason == "" {
			b.LabelRuneStart = runeIdxOf(c.LineText, lIdx)
			b.LabelRuneEnd = b.LabelRuneStart + len([]rune(c.Label))
			b.ValueRuneStart = runeIdxOf(c.LineText, vIdx)
			b.ValueRuneEnd = b.ValueRuneStart + len([]rune(c.Value))
			// label↔value canvas span must fit
			lineH := c.LineBBox.Y2 - c.LineBBox.Y1
			if lineH <= 0 {
				reason = "non-positive line height"
			} else {
				s := R1DLineHeightPx / lineH
				nR := float64(len([]rune(c.LineText)))
				lw := c.LineBBox.X2 - c.LineBBox.X1
				lc := c.LineBBox.X1 + lw*float64(b.LabelRuneStart+b.LabelRuneEnd)/(2*nR)
				vc := c.LineBBox.X1 + lw*float64(b.ValueRuneStart+b.ValueRuneEnd)/(2*nR)
				if math.Abs(vc-lc)*s > R1DFitBudgetPx {
					reason = fmt.Sprintf("label-value span %.0f px exceeds %.0f", math.Abs(vc-lc)*s, R1DFitBudgetPx)
				}
			}
		}
		b.Eligible = reason == ""
		b.IneligibleReason = reason
		if b.Eligible {
			alloc.EligibleCount++
		}
		alloc.Bases = append(alloc.Bases, b)
	}
	alloc.Proceed = alloc.EligibleCount >= alloc.MinRequired
	if alloc.Proceed {
		for i := range alloc.Bases {
			if alloc.Bases[i].Eligible {
				alloc.Bases[i].DistractorValues = frozenDistractors(alloc.Bases[i].Value, alloc.Bases[i].CandidateID, 8)
			}
		}
	}
	return alloc
}

func isPlainInt(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func labelHasContentWord(label string) bool {
	for _, tok := range strings.Fields(strings.ToLower(label)) {
		tok = strings.Trim(tok, "'-")
		letters := 0
		for _, r := range tok {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				letters++
			}
		}
		if letters >= 3 && !r1dStopwords[tok] {
			return true
		}
	}
	return false
}

// frozenDistractors deterministically draws n distractor integers for a
// base: plain 2-4 digit, != answer, distinct, balanced across digit
// lengths. Depends only on the seed and the candidate id.
func frozenDistractors(answer, candidateID string, n int) []string {
	h := sha256.Sum256([]byte(R1DSeed + "|distractor|" + candidateID))
	// deterministic PRNG: repeatedly hash the running state
	state := h[:]
	next := func() uint64 {
		s := sha256.Sum256(state)
		state = s[:]
		var v uint64
		for i := 0; i < 8; i++ {
			v = v<<8 | uint64(s[i])
		}
		return v
	}
	digitLens := []int{2, 3, 4}
	used := map[string]bool{answer: true}
	var out []string
	for len(out) < n {
		dl := digitLens[len(out)%3]
		lo := int(math.Pow10(dl - 1))
		hi := int(math.Pow10(dl)) - 1
		v := lo + int(next()%uint64(hi-lo+1))
		str := fmt.Sprintf("%d", v)
		if used[str] {
			continue
		}
		used[str] = true
		out = append(out, str)
	}
	return out
}

// DeriveR1DGeometry computes the frozen viewport transform for one base.
func DeriveR1DGeometry(base R1DBase) (R1DGeometry, error) {
	lb := base.LineBBox
	lineH := lb.Y2 - lb.Y1
	if lineH <= 0 {
		return R1DGeometry{}, fmt.Errorf("%s: non-positive line height", base.BaseID)
	}
	s := R1DLineHeightPx / lineH
	nR := float64(len([]rune(base.LineText)))
	lw := lb.X2 - lb.X1
	tokBox := func(r0, r1 int) canonicaldoc.BBox {
		return canonicaldoc.BBox{
			X1: lb.X1 + lw*float64(r0)/nR, Y1: lb.Y1,
			X2: lb.X1 + lw*float64(r1)/nR, Y2: lb.Y2,
		}
	}
	labelBox := tokBox(base.LabelRuneStart, base.LabelRuneEnd)
	valueBox := tokBox(base.ValueRuneStart, base.ValueRuneEnd)
	lcx := (labelBox.X1 + labelBox.X2) / 2
	vcx := (valueBox.X1 + valueBox.X2) / 2
	tcx := (lcx + vcx) / 2
	tcy := (lb.Y1 + lb.Y2) / 2

	toCanvas := func(b canonicaldoc.BBox) [4]int {
		x1, y1 := storeToCanvas(b.X1, b.Y1, tcx, tcy, s)
		x2, y2 := storeToCanvas(b.X2, b.Y2, tcx, tcy, s)
		return clampRect([4]int{int(math.Floor(x1)), int(math.Floor(y1)), int(math.Ceil(x2)), int(math.Ceil(y2))})
	}
	geo := R1DGeometry{
		BaseID: base.BaseID, AffineScale: s, LineHeightPx: lineH * s,
		CentreStore:     [2]float64{tcx, tcy},
		LineRectCanvas:  toCanvas(lb),
		LabelBBoxCanvas: toCanvas(labelBox),
		ValueBBoxCanvas: toCanvas(valueBox),
		LabelSpanPx:     math.Abs(vcx-lcx) * s,
	}
	return geo, nil
}

// BuildR1DViewport renders the D0 viewport: the containing line's real
// pixels, everything else neutral. No cue.
func BuildR1DViewport(pagePNG []byte, base R1DBase, geo R1DGeometry) (*image.RGBA, error) {
	src, err := png.Decode(bytes.NewReader(pagePNG))
	if err != nil {
		return nil, err
	}
	sb := src.Bounds()
	k := float64(sb.Dx()) / base.PageWidth
	s := geo.AffineScale
	tcx, tcy := geo.CentreStore[0], geo.CentreStore[1]
	lr := geo.LineRectCanvas
	out := image.NewRGBA(image.Rect(0, 0, CanvasPx, CanvasPx))
	for cy := 0; cy < CanvasPx; cy++ {
		for cx := 0; cx < CanvasPx; cx++ {
			if cx < lr[0] || cx >= lr[2] || cy < lr[1] || cy >= lr[3] {
				out.SetRGBA(cx, cy, maskBG)
				continue
			}
			storeX := tcx + (float64(cx)-canvasCenter)/s
			storeY := tcy + (float64(cy)-canvasCenter)/s
			if col, ok := bilinearSample(src, float64(sb.Min.X)+storeX*k, float64(sb.Min.Y)+storeY*k); ok {
				out.SetRGBA(cx, cy, col)
			} else {
				out.SetRGBA(cx, cy, maskBG)
			}
		}
	}
	return out, nil
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}

func drawR1DCue(img *image.RGBA, rect [4]int) {
	strokeRect(img, image.Rect(rect[0], rect[1], rect[2], rect[3]), cueColor, cueStrokePx)
}

// R1DRecord is one D0 or D1 result.
type R1DRecord struct {
	BaseID        string   `json:"base_id"`
	Track         string   `json:"track"` // D0 | D1
	Condition     string   `json:"condition"`
	Provenance    string   `json:"provenance"` // REAL_DOCUMENT | CONTROLLED_COMPOSITE
	K             int      `json:"k,omitempty"`
	Page          int      `json:"page"`
	Label         string   `json:"label"`
	GoldValue     string   `json:"gold_value"`
	Opcode        string   `json:"opcode"`
	VisibleNums   []string `json:"visible_numbers"`
	Distractors   []string `json:"distractors,omitempty"`
	RawText       string   `json:"raw_text"`
	Score         R1DScore `json:"score"`
	LatencyMS     int64    `json:"latency_ms"`
	PromptTokens  int      `json:"prompt_tokens"`
	CompletionTok int      `json:"completion_tokens"`
	CropPath      string   `json:"crop_path"`
	Error         string   `json:"error,omitempty"`
}

func writeRawR1D(dir string, rec R1DRecord) {
	body, _ := json.MarshalIndent(rec, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, rec.BaseID+"_"+strings.ToLower(rec.Condition)+".json"), body, 0o644)
}

func newR1DClient(cfg RunConfig) target.OpenAICompat {
	baseURL := strings.TrimRight(cfg.Endpoint, "/")
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL += "/v1"
	}
	return target.OpenAICompat{BaseURL: baseURL, Model: cfg.Model, Temperature: cfg.Temperature, MaxTokens: cfg.MaxTokens}
}

// RunR1D0 executes the paired real-association track: per eligible base,
// D0V_VALUE_CUED then D0L_LABEL_CUED over the SAME viewport pixels.
func RunR1D0(ctx context.Context, cfg RunConfig, alloc R1DAllocation) ([]R1DRecord, error) {
	prov, err := parrotlab.NewPDFMemoryProvider(cfg.StoreDir, cfg.PDFPath)
	if err != nil {
		return nil, err
	}
	client := newR1DClient(cfg)
	cropDir := filepath.Join(cfg.RunDir, "d0", "crops")
	rawDir := filepath.Join(cfg.RunDir, "d0", "raw")
	for _, d := range []string{cropDir, rawDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	var out []R1DRecord
	for _, base := range alloc.Bases {
		if !base.Eligible {
			continue
		}
		geo, gerr := DeriveR1DGeometry(base)
		if gerr != nil {
			return nil, gerr
		}
		pagePNG, perr := prov.RenderPNG(base.Page)
		if perr != nil {
			return nil, fmt.Errorf("render page %d: %w", base.Page, perr)
		}
		vp, verr := BuildR1DViewport(pagePNG, base, geo)
		if verr != nil {
			return nil, verr
		}
		conds := []struct {
			id, opcode, instruction string
			cue                     [4]int
		}{
			{"D0V_VALUE_CUED", FrozenOpcode, FrozenInstruction, geo.ValueBBoxCanvas},
			{"D0L_LABEL_CUED", R1DAssocOpcode, R1DAssocInstruction, geo.LabelBBoxCanvas},
		}
		for _, cd := range conds {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			default:
			}
			img := cloneRGBA(vp)
			drawR1DCue(img, cd.cue)
			cropPath := filepath.Join(cropDir, base.BaseID+"_"+strings.ToLower(cd.id)+".png")
			if werr := writeRGBAPNG(cropPath, img); werr != nil {
				return nil, werr
			}
			rec := R1DRecord{
				BaseID: base.BaseID, Track: "D0", Condition: cd.id, Provenance: "REAL_DOCUMENT",
				Page: base.Page, Label: base.Label, GoldValue: base.Value, Opcode: cd.opcode,
				VisibleNums: []string{base.Value}, CropPath: cropPath,
			}
			raw, ierr := os.ReadFile(cropPath)
			if ierr != nil {
				rec.Error = ierr.Error()
				out = append(out, rec)
				writeRawR1D(rawDir, rec)
				continue
			}
			start := time.Now()
			res, cerr := client.CompletePerception(ctx, target.PerceptionInput{Question: cd.instruction, Image: raw, MediaType: "image/png"})
			rec.LatencyMS = time.Since(start).Milliseconds()
			if cerr != nil {
				rec.Error = cerr.Error()
				out = append(out, rec)
				writeRawR1D(rawDir, rec)
				continue
			}
			rec.RawText = res.Content
			rec.PromptTokens = res.PromptTokensReported
			rec.CompletionTok = res.CompletionTokensReported
			rec.Score = ScoreR1DAssoc(res.Content, base.Value, rec.VisibleNums, nil)
			out = append(out, rec)
			writeRawR1D(rawDir, rec)
		}
	}
	return out, nil
}

// distractorSlots is the frozen ring of candidate placement slots (canvas
// coords, centre of the sprite), ordered deterministically. Two distance
// bands: near (comparable to a label→value gap) and far.
func distractorSlots() [][2]int {
	c := canvasCenter
	near, far := 96, 180
	return [][2]int{
		{c, c - near}, {c, c + near}, {c - far, c}, {c + far, c},
		{c - far, c - near}, {c + far, c - near}, {c - far, c + near}, {c + far, c + near},
		{c, c - far}, {c, c + far}, {c - near, c - far}, {c + near, c + far},
	}
}

// RenderNumberSprite renders a digit string as a tight RGBA raster from the
// glyph bank (darken-composited on white), for distractor placement.
func RenderNumberSprite(bank *GlyphBank, s string) (*image.RGBA, error) {
	runes := []rune(s)
	total, maxH := 0, 0
	for _, r := range runes {
		g, ok := bank.Glyphs[string(r)]
		if !ok || len(g.pixels) != g.WidthPx*g.HeightPx || g.WidthPx == 0 {
			return nil, fmt.Errorf("no raster for %q", string(r))
		}
		total += int(math.Round(g.AdvancePx)) + 1
		if g.HeightPx > maxH {
			maxH = g.HeightPx
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, total+2, maxH))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	cx := 0
	for _, r := range runes {
		g := bank.Glyphs[string(r)]
		for gy := 0; gy < g.HeightPx; gy++ {
			for gx := 0; gx < g.WidthPx; gx++ {
				lum := g.pixels[gy*g.WidthPx+gx]
				px, py := cx+gx, gy
				if px < 0 || py < 0 || px >= img.Bounds().Dx() || py >= img.Bounds().Dy() {
					continue
				}
				cur := img.RGBAAt(px, py)
				nv := uint8(min2(int(cur.R), int(lum)))
				img.SetRGBA(px, py, color.RGBA{R: nv, G: nv, B: nv, A: 255})
			}
		}
		cx += int(math.Round(g.AdvancePx)) + 1
	}
	return img, nil
}

func rectsOverlap(a, b [4]int) bool {
	return a[0] < b[2] && b[0] < a[2] && a[1] < b[3] && b[1] < a[3]
}

// placeDistractors composites the first K non-overlapping distractor
// sprites onto a copy of the base image. Deterministic. Returns the image
// and the placed sprite rects.
func placeDistractors(base *image.RGBA, bank *GlyphBank, geo R1DGeometry, distractors []string, k int) (*image.RGBA, [][4]int, error) {
	img := cloneRGBA(base)
	if k == 0 {
		return img, nil, nil
	}
	protected := [][4]int{geo.LineRectCanvas, geo.LabelBBoxCanvas, geo.ValueBBoxCanvas}
	slots := distractorSlots()
	var placed [][4]int
	di := 0
	for _, slot := range slots {
		if len(placed) >= k || di >= len(distractors) {
			break
		}
		sprite, err := RenderNumberSprite(bank, distractors[di])
		if err != nil {
			return nil, nil, err
		}
		w, h := sprite.Bounds().Dx(), sprite.Bounds().Dy()
		x0, y0 := slot[0]-w/2, slot[1]-h/2
		rect := [4]int{x0, y0, x0 + w, y0 + h}
		if rect[0] < 2 || rect[1] < 2 || rect[2] > CanvasPx-2 || rect[3] > CanvasPx-2 {
			continue
		}
		clash := false
		for _, pr := range append(append([][4]int{}, protected...), placed...) {
			if rectsOverlap(rect, pr) {
				clash = true
				break
			}
		}
		if clash {
			continue
		}
		// darken-composite the sprite
		for sy := 0; sy < h; sy++ {
			for sx := 0; sx < w; sx++ {
				sc := sprite.RGBAAt(sx, sy)
				px, py := x0+sx, y0+sy
				cur := img.RGBAAt(px, py)
				nv := uint8(min2(int(cur.R), int(sc.R)))
				img.SetRGBA(px, py, color.RGBA{R: nv, G: nv, B: nv, A: 255})
			}
		}
		placed = append(placed, rect)
		di++
	}
	if len(placed) < k {
		return nil, nil, fmt.Errorf("only placed %d/%d distractors (no free slot)", len(placed), k)
	}
	return img, placed, nil
}

// RunR1D1 executes the CONTROLLED_COMPOSITE distractor-density track.
func RunR1D1(ctx context.Context, cfg RunConfig, alloc R1DAllocation, bank *GlyphBank) ([]R1DRecord, error) {
	prov, err := parrotlab.NewPDFMemoryProvider(cfg.StoreDir, cfg.PDFPath)
	if err != nil {
		return nil, err
	}
	client := newR1DClient(cfg)
	cropDir := filepath.Join(cfg.RunDir, "d1", "crops")
	rawDir := filepath.Join(cfg.RunDir, "d1", "raw")
	for _, d := range []string{cropDir, rawDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	var out []R1DRecord
	for _, base := range alloc.Bases {
		if !base.Eligible {
			continue
		}
		geo, gerr := DeriveR1DGeometry(base)
		if gerr != nil {
			return nil, gerr
		}
		pagePNG, perr := prov.RenderPNG(base.Page)
		if perr != nil {
			return nil, fmt.Errorf("render page %d: %w", base.Page, perr)
		}
		vp, verr := BuildR1DViewport(pagePNG, base, geo)
		if verr != nil {
			return nil, verr
		}
		labelCued := cloneRGBA(vp)
		drawR1DCue(labelCued, geo.LabelBBoxCanvas)
		for _, rung := range R1DDistractorLadder {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			default:
			}
			img, _, derr := placeDistractors(labelCued, bank, geo, base.DistractorValues, rung.K)
			rec := R1DRecord{
				BaseID: base.BaseID, Track: "D1", Condition: rung.ID, Provenance: "CONTROLLED_COMPOSITE",
				K: rung.K, Page: base.Page, Label: base.Label, GoldValue: base.Value, Opcode: R1DAssocOpcode,
				Distractors: base.DistractorValues[:rung.K],
			}
			rec.VisibleNums = append([]string{base.Value}, base.DistractorValues[:rung.K]...)
			if derr != nil {
				rec.Error = derr.Error()
				out = append(out, rec)
				writeRawR1D(rawDir, rec)
				continue
			}
			cropPath := filepath.Join(cropDir, base.BaseID+"_"+strings.ToLower(rung.ID)+".png")
			if werr := writeRGBAPNG(cropPath, img); werr != nil {
				return nil, werr
			}
			rec.CropPath = cropPath
			raw, ierr := os.ReadFile(cropPath)
			if ierr != nil {
				rec.Error = ierr.Error()
				out = append(out, rec)
				writeRawR1D(rawDir, rec)
				continue
			}
			start := time.Now()
			res, cerr := client.CompletePerception(ctx, target.PerceptionInput{Question: R1DAssocInstruction, Image: raw, MediaType: "image/png"})
			rec.LatencyMS = time.Since(start).Milliseconds()
			if cerr != nil {
				rec.Error = cerr.Error()
				out = append(out, rec)
				writeRawR1D(rawDir, rec)
				continue
			}
			rec.RawText = res.Content
			rec.PromptTokens = res.PromptTokensReported
			rec.CompletionTok = res.CompletionTokensReported
			rec.Score = ScoreR1DAssoc(res.Content, base.Value, rec.VisibleNums, base.DistractorValues[:rung.K])
			out = append(out, rec)
			writeRawR1D(rawDir, rec)
		}
	}
	return out, nil
}

// RenderR1DSanity renders the inspection set for one eligible base:
// D0V/D0L (same viewport, cue moved) and D1 K0/K2/K8.
func RenderR1DSanity(storeDir, pdfPath string, base R1DBase, bank *GlyphBank) (map[string]*image.RGBA, error) {
	prov, err := parrotlab.NewPDFMemoryProvider(storeDir, pdfPath)
	if err != nil {
		return nil, err
	}
	geo, err := DeriveR1DGeometry(base)
	if err != nil {
		return nil, err
	}
	pagePNG, err := prov.RenderPNG(base.Page)
	if err != nil {
		return nil, err
	}
	vp, err := BuildR1DViewport(pagePNG, base, geo)
	if err != nil {
		return nil, err
	}
	out := map[string]*image.RGBA{}
	d0v := cloneRGBA(vp)
	drawR1DCue(d0v, geo.ValueBBoxCanvas)
	out["d0v_value_cued"] = d0v
	d0l := cloneRGBA(vp)
	drawR1DCue(d0l, geo.LabelBBoxCanvas)
	out["d0l_label_cued"] = d0l
	labelCued := cloneRGBA(vp)
	drawR1DCue(labelCued, geo.LabelBBoxCanvas)
	for _, k := range []int{0, 2, 8} {
		img, _, derr := placeDistractors(labelCued, bank, geo, base.DistractorValues, k)
		if derr != nil {
			return nil, derr
		}
		out[fmt.Sprintf("d1_k%d", k)] = img
	}
	return out, nil
}
