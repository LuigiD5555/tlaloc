package perceptenvelope

import "math"

// RenderDPI is the pdftoppm rasterisation density used everywhere in this
// experiment (matches parrotlab.pdfMemoryProvider.RenderPNG).
const RenderDPI = 150.0

// storeDPI is the pdfmemory store's own coordinate density (PDF points).
const storeDPI = 72.0

// renderScale maps store (page-point) coordinates to rendered-image pixels.
const renderScale = RenderDPI / storeDPI

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
	BaseID       string          `json:"base_id"`
	Page         int             `json:"page"`
	TargetToken  string          `json:"target_token"`
	StorePageW   float64         `json:"store_page_width"`
	StorePageH   float64         `json:"store_page_height"`
	RenderScale  float64         `json:"render_scale_store_to_image"`
	LineFontSize int             `json:"line_font_size"`
	Levels       []LevelGeometry `json:"levels"`
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
		"LM Studio server log 2026-09-03 during R1-A run: 'Evaluated N tokens for image' varied N in {72,96,112,232,256,368,...,991} across the 210 differently-sized crops -> image-token budget (hence effective resolution) scales with the input PNG size",
		"A0 crops are ~38x41 px and are upscaled to at least one 256 tile (~6x target magnification); A6 full pages are ~1575x2362 px and are downscaled/sliced under the token budget (target shrinks). Effective target glyph scale is NOT held constant across A0..A6.",
	},
}

// ScaleAudit computes the per-level image geometry table for the given bases.
func ScaleAudit(bases []Base) ScaleAuditReport {
	rep := ScaleAuditReport{
		Schema: "tlaloc.parrot-perceptual-envelope-r1.scale-audit.r1", ExperimentID: ExperimentID,
		RenderDPI: RenderDPI, VisionPreproc: KnownVisionPreproc,
	}
	for _, base := range bases {
		c := base.Candidate
		ba := BaseScaleAudit{
			BaseID: base.BaseID, Page: c.Page, TargetToken: c.NormalizedTarget,
			StorePageW: c.PageWidth, StorePageH: c.PageHeight, RenderScale: renderScale,
			LineFontSize: c.Line.FontSize,
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
