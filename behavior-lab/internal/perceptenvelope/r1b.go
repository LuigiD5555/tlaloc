package perceptenvelope

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
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

// R1-B SCALE / RESOLUTION ENVELOPE renderer (protocol R1_PROTOCOL_ADDENDUM_03).
//
// R1-B varies ONE independent variable: the submitted containing-line
// height in canvas pixels. Everything else is held constant with the
// frozen R1-B context policy (A1C0_TARGET crop content), the frozen atomic
// EXTRACT_NUMBER instruction, the frozen LFM2-VL identity, temperature 0
// and a 32-token budget.
//
// For every base a single fixed SOURCE CROP RECTANGLE is derived in store
// coordinates — exactly the R1-A1 A1C0_TARGET reveal region (cue token
// bbox + the frozen 10 px canvas pad, mapped back to store units at the
// R1-A1 32 px scale). Each of the six scale conditions resamples that
// SAME store rectangle at a different affine scale so that the containing
// text line is B px tall, centres the target at canvas (256,256) and fills
// the rest of the fixed 512x512 canvas with the neutral mask background.
// No extra document content is ever revealed as scale grows.

// R1BCondition is one predeclared scale rung.
type R1BCondition struct {
	ID     string  `json:"id"`
	LinePx float64 `json:"target_line_height_px"`
}

// R1BScaleLadder is the frozen six-point submitted-pixel scale ladder
// (protocol section 5). Recorded before any R1-B model output existed.
var R1BScaleLadder = []R1BCondition{
	{ID: "B0", LinePx: 8},
	{ID: "B1", LinePx: 12},
	{ID: "B2", LinePx: 16},
	{ID: "B3", LinePx: 24},
	{ID: "B4", LinePx: 32},
	{ID: "B5", LinePx: 48},
}

// R1BLineHeightTolerancePx is the documented rounding tolerance for the
// actual containing-line height vs the nominal ladder value.
const R1BLineHeightTolerancePx = 0.5

// R1BResampler documents the single deterministic interpolation method
// used for every scale condition (the same bilinear sampler R1-A1 used to
// build its per-base viewport).
const R1BResampler = "go std image bilinear (perceptenvelope.bilinearSample); Go " + "go1.26"

// R1BCueRatio is the frozen cue-thickness / nominal-line-height ratio
// (R1-A1 used a 3 px stroke at 32 px lines).
const R1BCueRatio = float64(cueStrokePx) / TargetLineHeightPx

// R1BExpectedRecords is 30 bases x 6 scales.
const R1BExpectedRecords = R1BSize * 6

// R1BCondGeom is the frozen geometry for one (base, scale) cell, all in
// canvas pixels unless noted.
type R1BCondGeom struct {
	Condition          string     `json:"condition"`
	NominalLinePx      float64    `json:"nominal_line_height_px"`
	AffineScale        float64    `json:"affine_scale_store_to_canvas"`
	LineHeightCanvasPx float64    `json:"line_height_canvas_px"`
	SourceCropCanvasPx [2]float64 `json:"source_crop_canvas_wh_px"`
	CueBBoxCanvasPx    [4]float64 `json:"cue_bbox_canvas_px"`
	TargetBBoxHeightPx float64    `json:"target_bbox_height_canvas_px"`
	CueStrokePx        int        `json:"cue_stroke_px"`
	CueOverGlyphRatio  float64    `json:"cue_thickness_over_line_height_ratio"`
}

// R1BGeometry is the frozen per-base transform: one shared store-space
// source crop, six scale conditions.
type R1BGeometry struct {
	BaseID            string        `json:"base_id"`
	Page              int           `json:"page"`
	LineHeightStore   float64       `json:"line_height_store"`
	TargetCenterStore [2]float64    `json:"target_center_store"`
	CueBBoxStore      [4]float64    `json:"cue_bbox_store"`
	SourceCropStore   [4]float64    `json:"source_crop_store"`
	Conditions        []R1BCondGeom `json:"conditions"`
}

// deriveR1BCue reproduces the frozen R1-A1 tight-token cue bbox in store
// coordinates: line bbox + frozen char offsets, 1.5 char-width x pad,
// 0.15 line-height y pad. Reads only candidate geometry — never gold.
func deriveR1BCue(c Candidate) (tb canonicaldoc.BBox, lineH float64, err error) {
	line := c.Line.BBox
	lineH = line.Y2 - line.Y1
	if lineH <= 0 {
		return canonicaldoc.BBox{}, 0, fmt.Errorf("non-positive line height")
	}
	nRunes := float64(len([]rune(c.Line.Text)))
	if nRunes <= 0 {
		return canonicaldoc.BBox{}, 0, fmt.Errorf("empty line text")
	}
	lw := line.X2 - line.X1
	charW := lw / nRunes
	tokX1 := line.X1 + lw*float64(c.CharOffsetStart)/nRunes
	tokX2 := line.X1 + lw*float64(c.CharOffsetEnd)/nRunes
	xpad := 1.5 * charW
	ypad := 0.15 * lineH
	return canonicaldoc.BBox{X1: tokX1 - xpad, Y1: line.Y1 - ypad, X2: tokX2 + xpad, Y2: line.Y2 + ypad}, lineH, nil
}

// DeriveR1BGeometry computes the frozen source crop + six scale transforms
// for one base. Deterministic; reads no model output and no gold field.
func DeriveR1BGeometry(storeDir string, base Base) (R1BGeometry, error) {
	c := base.Candidate
	if _, _, err := loadStorePage(storeDir, c.Page); err != nil {
		return R1BGeometry{}, err
	}
	tb, lineH, err := deriveR1BCue(c)
	if err != nil {
		return R1BGeometry{}, fmt.Errorf("%s: %w", base.BaseID, err)
	}
	tcx := (tb.X1 + tb.X2) / 2
	tcy := (tb.Y1 + tb.Y2) / 2

	// Fixed source crop == R1-A1 A1C0_TARGET reveal region, expressed in
	// store units: cue bbox + (10 canvas px / 32-px-scale) pad per side.
	padStore := c0PadPx * lineH / TargetLineHeightPx
	crop := canonicaldoc.BBox{X1: tb.X1 - padStore, Y1: tb.Y1 - padStore, X2: tb.X2 + padStore, Y2: tb.Y2 + padStore}

	geo := R1BGeometry{
		BaseID: base.BaseID, Page: c.Page,
		LineHeightStore:   lineH,
		TargetCenterStore: [2]float64{tcx, tcy},
		CueBBoxStore:      [4]float64{tb.X1, tb.Y1, tb.X2, tb.Y2},
		SourceCropStore:   [4]float64{crop.X1, crop.Y1, crop.X2, crop.Y2},
	}
	for _, cond := range R1BScaleLadder {
		s := cond.LinePx / lineH
		cx1, cy1 := storeToCanvas(tb.X1, tb.Y1, tcx, tcy, s)
		cx2, cy2 := storeToCanvas(tb.X2, tb.Y2, tcx, tcy, s)
		stroke := int(math.Round(RRatio(cond.LinePx)))
		if stroke < 1 {
			stroke = 1
		}
		geo.Conditions = append(geo.Conditions, R1BCondGeom{
			Condition:          cond.ID,
			NominalLinePx:      cond.LinePx,
			AffineScale:        s,
			LineHeightCanvasPx: lineH * s,
			SourceCropCanvasPx: [2]float64{(crop.X2 - crop.X1) * s, (crop.Y2 - crop.Y1) * s},
			CueBBoxCanvasPx:    [4]float64{cx1, cy1, cx2, cy2},
			TargetBBoxHeightPx: (tb.Y2 - tb.Y1) * s,
			CueStrokePx:        stroke,
			CueOverGlyphRatio:  float64(stroke) / (lineH * s),
		})
	}
	return geo, nil
}

// RRatio returns the frozen cue stroke width in canvas pixels for a
// nominal line height, keeping cue-thickness / line-height constant.
func RRatio(linePx float64) float64 { return R1BCueRatio * linePx }

// RenderR1BScale renders one (base, scale) 512x512 condition image from the
// rendered page PNG. Returns the image and the visible (non-mask) fraction.
func RenderR1BScale(pagePNG []byte, base Base, geo R1BGeometry, cond R1BCondGeom) (*image.RGBA, float64, error) {
	src, err := png.Decode(bytes.NewReader(pagePNG))
	if err != nil {
		return nil, 0, err
	}
	sb := src.Bounds()
	c := base.Candidate
	k := float64(sb.Dx()) / c.PageWidth // rendered-page px per store unit
	s := cond.AffineScale
	tcx, tcy := geo.TargetCenterStore[0], geo.TargetCenterStore[1]
	crop := geo.SourceCropStore

	out := image.NewRGBA(image.Rect(0, 0, CanvasPx, CanvasPx))
	visible := 0
	for cy := 0; cy < CanvasPx; cy++ {
		for cx := 0; cx < CanvasPx; cx++ {
			storeX := tcx + (float64(cx)-canvasCenter)/s
			storeY := tcy + (float64(cy)-canvasCenter)/s
			if storeX < crop[0] || storeX > crop[2] || storeY < crop[1] || storeY > crop[3] {
				out.SetRGBA(cx, cy, maskBG)
				continue
			}
			px := float64(sb.Min.X) + storeX*k
			py := float64(sb.Min.Y) + storeY*k
			if col, ok := bilinearSample(src, px, py); ok {
				out.SetRGBA(cx, cy, col)
				visible++
			} else {
				out.SetRGBA(cx, cy, maskBG)
			}
		}
	}
	cb := cond.CueBBoxCanvasPx
	strokeRect(out, image.Rect(int(cb[0]), int(cb[1]), int(cb[2]), int(cb[3])), cueColor, cond.CueStrokePx)
	return out, float64(visible) / float64(CanvasPx*CanvasPx), nil
}

// RunR1BScale executes R1-B: 30 bases x 6 scales = 180 calls. Deterministic
// order; one model call per record; raw output preserved.
func RunR1BScale(ctx context.Context, cfg RunConfig, alloc Allocation) ([]RecordOutcome, []R1BGeometry, error) {
	provider, err := parrotlab.NewPDFMemoryProvider(cfg.StoreDir, cfg.PDFPath)
	if err != nil {
		return nil, nil, fmt.Errorf("page provider: %w", err)
	}
	baseURL := strings.TrimRight(cfg.Endpoint, "/")
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL += "/v1"
	}
	client := target.OpenAICompat{BaseURL: baseURL, Model: cfg.Model, Temperature: cfg.Temperature, MaxTokens: cfg.MaxTokens}

	cropDir := filepath.Join(cfg.RunDir, "crops")
	rawDir := filepath.Join(cfg.RunDir, "raw")
	for _, d := range []string{cropDir, rawDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, nil, err
		}
	}

	var out []RecordOutcome
	var geos []R1BGeometry
	for _, base := range alloc.Bases {
		geo, gerr := DeriveR1BGeometry(cfg.StoreDir, base)
		if gerr != nil {
			return nil, nil, fmt.Errorf("geometry %s: %w", base.BaseID, gerr)
		}
		geos = append(geos, geo)
		pagePNG, rerr := provider.RenderPNG(base.Candidate.Page)
		if rerr != nil {
			return nil, nil, fmt.Errorf("render page %d: %w", base.Candidate.Page, rerr)
		}
		for _, cond := range geo.Conditions {
			select {
			case <-ctx.Done():
				return out, geos, ctx.Err()
			default:
			}
			rec := RecordOutcome{
				BaseID: base.BaseID, CandidateID: base.Candidate.CandidateID, Stage: "R1-B",
				Mode: "SCALE", Level: cond.Condition, Page: base.Candidate.Page,
				Gold: base.Candidate.NormalizedTarget,
			}
			img, visFrac, verr := RenderR1BScale(pagePNG, base, geo, cond)
			if verr != nil {
				rec.Error = verr.Error()
				out = append(out, rec)
				writeRaw(rawDir, rec)
				continue
			}
			rec.VisualExposure = visFrac
			cropPath := filepath.Join(cropDir, fmt.Sprintf("%s_%s.png", base.BaseID, strings.ToLower(cond.Condition)))
			if werr := writeRGBAPNG(cropPath, img); werr != nil {
				return nil, nil, werr
			}
			rec.CropPath = cropPath
			rec.CropWidth, rec.CropHeight, rec.PixelArea = CanvasPx, CanvasPx, CanvasPx*CanvasPx
			raw, rerr := os.ReadFile(cropPath)
			if rerr != nil {
				rec.Error = rerr.Error()
				out = append(out, rec)
				writeRaw(rawDir, rec)
				continue
			}
			start := time.Now()
			result, perr := client.CompletePerception(ctx, target.PerceptionInput{
				Question: FrozenInstruction, Image: raw, MediaType: "image/png",
			})
			rec.LatencyMS = time.Since(start).Milliseconds()
			if perr != nil {
				rec.Error = perr.Error()
				out = append(out, rec)
				writeRaw(rawDir, rec)
				continue
			}
			rec.RawText = result.Content
			rec.PromptTokens = result.PromptTokensReported
			rec.CompletionToks = result.CompletionTokensReported
			scoreRecord(&rec, base.Candidate.NormalizedTarget)
			out = append(out, rec)
			writeRaw(rawDir, rec)
		}
	}
	return out, geos, nil
}

// SHA256OfTree returns a deterministic sha256 over every regular file
// under root (sorted by relative path), folding in each path and its bytes.
func SHA256OfTree(root string) (string, int, error) {
	var paths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !info.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return "", 0, err
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, path := range paths {
		rel, _ := filepath.Rel(root, path)
		h.Write([]byte(rel))
		h.Write([]byte{0})
		body, rerr := os.ReadFile(path)
		if rerr != nil {
			return "", 0, rerr
		}
		h.Write(body)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), len(paths), nil
}

// RenderR1BSanity renders (no inference) every scale condition for the
// given bases into <outDir>/crops for visual geometry inspection.
func RenderR1BSanity(storeDir, pdfPath string, bases []Base, outDir string) ([]R1BGeometry, error) {
	prov, err := parrotlab.NewPDFMemoryProvider(storeDir, pdfPath)
	if err != nil {
		return nil, err
	}
	cropDir := filepath.Join(outDir, "crops")
	if err := os.MkdirAll(cropDir, 0o755); err != nil {
		return nil, err
	}
	var geos []R1BGeometry
	for _, base := range bases {
		geo, err := DeriveR1BGeometry(storeDir, base)
		if err != nil {
			return nil, err
		}
		pagePNG, err := prov.RenderPNG(base.Candidate.Page)
		if err != nil {
			return nil, err
		}
		for _, cond := range geo.Conditions {
			img, _, err := RenderR1BScale(pagePNG, base, geo, cond)
			if err != nil {
				return nil, err
			}
			p := filepath.Join(cropDir, fmt.Sprintf("%s_%s.png", base.BaseID, strings.ToLower(cond.Condition)))
			if err := writeRGBAPNG(p, img); err != nil {
				return nil, err
			}
		}
		geos = append(geos, geo)
	}
	return geos, nil
}
