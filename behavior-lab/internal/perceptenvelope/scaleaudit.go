package perceptenvelope

import (
	"fmt"
	"math"
	"path/filepath"
)

// RenderDPI is the pdftoppm rasterisation density (parrotlab RenderPNG).
const RenderDPI = 150.0

// LevelGeometry is the frozen image geometry of one (base, context level)
// BEFORE the VLM image preprocessor. Everything here is a deterministic
// function of the frozen candidate + the render DPI.
type LevelGeometry struct {
	Level                   string   `json:"context_level"`
	ContextRegionStore      BBoxJSON `json:"context_region_store_px"`
	CropImageWidthPx        int      `json:"crop_image_width_px"`
	CropImageHeightPx       int      `json:"crop_image_height_px"`
	CropImageAreaPx         int      `json:"crop_image_area_px"`
	TargetBBoxWidthPx       float64  `json:"target_cue_bbox_width_px"`
	TargetBBoxHeightPx      float64  `json:"target_cue_bbox_height_px"`
	GlyphHeightPx           float64  `json:"target_glyph_height_px_estimate"`
	TargetAreaOverImage     float64  `json:"target_bbox_area_over_crop_area"`
	CueStrokePx             int      `json:"cue_stroke_px"`
	CueStrokeOverGlyph      float64  `json:"cue_stroke_over_glyph_height"`
	NaturalCropConfoundNote string   `json:"note"`
}

// BBoxJSON is a plain bbox for the audit artifact.
type BBoxJSON struct {
	X1, Y1, X2, Y2 float64
}

// BaseScaleAudit is one base's per-level geometry table.
type BaseScaleAudit struct {
	BaseID            string          `json:"base_id"`
	Page              int             `json:"page"`
	TargetToken       string          `json:"target_token"`
	StorePageW        float64         `json:"store_page_width"`
	StorePageH        float64         `json:"store_page_height"`
	RenderScale       float64         `json:"render_scale_store_to_image"`
	RenderScaleSource string          `json:"render_scale_source"`
	LineFontSize      int             `json:"line_font_size"`
	Levels            []LevelGeometry `json:"levels"`
}

// ScaleAuditReport is SCALE_AUDIT_R1A.json.
type ScaleAuditReport struct {
	Schema        string           `json:"schema"`
	ExperimentID  string           `json:"experiment_id"`
	RenderDPI     float64          `json:"render_dpi"`
	VisionPreproc VisionPreproc    `json:"vision_preprocessing"`
	Bases         []BaseScaleAudit `json:"bases"`
}

// VisionPreproc records what is known about the LFM2-VL image pipeline.
type VisionPreproc struct {
	Model                    string   `json:"model"`
	Encoder                  string   `json:"encoder"`
	BaseTilePx               int      `json:"clip_vision_image_size"`
	PatchPx                  int      `json:"clip_vision_patch_size"`
	ProjectorScaleFactor     int      `json:"clip_vision_projector_scale_factor"`
	TokensPerFullTile        int      `json:"tokens_per_full_256_tile"`
	VariableResolution       bool     `json:"variable_resolution_slicing"`
	EffectiveScaleControlled bool     `json:"effective_target_scale_controlled_in_natural_crop"`
	Evidence                 []string `json:"evidence"`
}

// KnownVisionPreproc is the frozen finding from the runtime audit.
var KnownVisionPreproc = VisionPreproc{
	Model:                    "lfm2-vl-1.6b (LiquidAI), llama.cpp mtmd/clip, LM Studio backend llama.cpp-linux-x86_64-nvidia-cuda-avx2 1.104.2",
	Encoder:                  "SigLIP2-family vision encoder, projector_type=lfm2, variable-resolution square slicing (clip get_slice_instructions)",
	BaseTilePx:               256,
	PatchPx:                  16,
	ProjectorScaleFactor:     2,
	TokensPerFullTile:        64, // (256/16/2)^2
	VariableResolution:       true,
	EffectiveScaleControlled: false,
	Evidence: []string{
		"mmproj-LFM2-VL-1.6B-F16.gguf: clip.vision.image_size=256, patch_size=16, projector.scale_factor=2 -> 64 tokens per full 256x256 tile",
		"llama.cpp backend exports clip get_slice_instructions / 'slice %d: x=%d y=%d size=%dx%d' / 'n_patches_x == n_patches_y && only square images supported' -> square-tile slicing keyed to input dimensions",
		"LM Studio server log 2026-09-03 during the R1-A0 run: per-image clip token counts clustered by context level at ~{75, 83, 99, 115, 163, 235, 371, 762, 991} (30 images each at the top three) -> the image-token budget, hence effective resolution, scales with input PNG size, saturating near 991 for full pages",
		"measured R1-A0 crop dimensions: A0 ~41x41 px, A1 ~681x21, A2 ~881x124, A3 ~924x268, A4 ~525x750, A5 ~1050x750, A6 ~1050x1500 (store->image scale ~1.39). A0's ~41 px crop is upscaled to >=1 256-tile (target magnified); A6's 1050 px page is downscaled under the token budget (target glyph ~9 effective px). Effective target glyph scale is NOT held constant across A0..A6.",
	},
}

// ScaleAudit computes the per-level image geometry table for the given
// bases. pagesDir, when non-empty, is the run's rendered-page directory
// (<baseID>_cued.png); the true store->image scale is measured from each
// rendered page rather than assumed from DPI.
func ScaleAudit(bases []Base, pagesDir string) ScaleAuditReport {
	rep := ScaleAuditReport{
		Schema: "tlaloc.parrot-perceptual-envelope-r1.scale-audit.r1", ExperimentID: ExperimentID,
		RenderDPI: RenderDPI, VisionPreproc: KnownVisionPreproc,
	}
	for _, base := range bases {
		c := base.Candidate
		renderScale := RenderDPI / 72.0
		scaleSource := "assumed_from_dpi"
		if pagesDir != "" {
			if w, _, err := pageDimsFromPNG(filepath.Join(pagesDir, fmt.Sprintf("%s_cued.png", base.BaseID))); err == nil && c.PageWidth > 0 {
				renderScale = float64(w) / c.PageWidth
				scaleSource = "measured_from_rendered_page"
			}
		}
		ba := BaseScaleAudit{
			BaseID: base.BaseID, Page: c.Page, TargetToken: c.NormalizedTarget,
			StorePageW: c.PageWidth, StorePageH: c.PageHeight, RenderScale: renderScale,
			RenderScaleSource: scaleSource, LineFontSize: c.Line.FontSize,
		}
		imgW := c.PageWidth * renderScale
		imgH := c.PageHeight * renderScale
		tb := c.TokenBBoxStore
		lineH := c.Line.BBox.Y2 - c.Line.BBox.Y1
		for _, lvl := range AllContextLevels {
			region := ContextRegionStore(c, lvl)
			var cw, ch float64
			if lvl == A6FullPage {
				cw, ch = imgW, imgH
			} else {
				cw = (region.X2 - region.X1) * renderScale
				ch = (region.Y2 - region.Y1) * renderScale
			}
			glyph := lineH * renderScale
			lg := LevelGeometry{
				Level:              string(lvl),
				ContextRegionStore: BBoxJSON{region.X1, region.Y1, region.X2, region.Y2},
				CropImageWidthPx:   int(math.Round(cw)),
				CropImageHeightPx:  int(math.Round(ch)),
				CropImageAreaPx:    int(math.Round(cw * ch)),
				TargetBBoxWidthPx:  (tb.X2 - tb.X1) * renderScale,
				TargetBBoxHeightPx: (tb.Y2 - tb.Y1) * renderScale,
				GlyphHeightPx:      glyph,
				CueStrokePx:        cueStrokePx,
				CueStrokeOverGlyph: float64(cueStrokePx) / glyph,
			}
			if cw*ch > 0 {
				lg.TargetAreaOverImage = ((tb.X2 - tb.X1) * (tb.Y2 - tb.Y1) * renderScale * renderScale) / (cw * ch)
			}
			ba.Levels = append(ba.Levels, lg)
		}
		rep.Bases = append(rep.Bases, ba)
	}
	return rep
}

// DiagnosticBases returns the frozen predeclared diagnostic subset: the
// first n bases of the allocation in base-id order.
func DiagnosticBases(alloc Allocation, n int) []Base {
	bs := append([]Base(nil), alloc.Bases...)
	if n > len(bs) {
		n = len(bs)
	}
	return bs[:n]
}
