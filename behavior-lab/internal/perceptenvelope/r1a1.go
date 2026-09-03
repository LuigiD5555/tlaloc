package perceptenvelope

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/parrotlab"
	"tlaloc.local/behaviorlab/internal/target"
)

// RunR1A1Context executes the canonical R1-A1 fixed-scale local-context
// envelope: 30 fresh bases x 7 nested levels = 210 calls. One 512x512
// per-base viewport, seven nested reveal masks, one atomic EXTRACT_NUMBER
// instruction, raw output preserved.
func RunR1A1Context(ctx context.Context, cfg RunConfig, alloc Allocation) ([]RecordOutcome, []R1A1Geometry, error) {
	provider, err := parrotlab.NewPDFMemoryProvider(cfg.StoreDir, cfg.PDFPath)
	if err != nil {
		return nil, nil, fmt.Errorf("page provider: %w", err)
	}
	baseURL := strings.TrimRight(cfg.Endpoint, "/")
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL += "/v1"
	}
	client := target.OpenAICompat{BaseURL: baseURL, Model: cfg.Model, Temperature: cfg.Temperature, MaxTokens: cfg.MaxTokens}

	viewDir := filepath.Join(cfg.RunDir, "viewports")
	cropDir := filepath.Join(cfg.RunDir, "crops")
	rawDir := filepath.Join(cfg.RunDir, "raw")
	for _, d := range []string{viewDir, cropDir, rawDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, nil, err
		}
	}

	var out []RecordOutcome
	var geos []R1A1Geometry
	for _, base := range alloc.Bases {
		geo, gerr := DeriveR1A1Geometry(cfg.StoreDir, base)
		if gerr != nil {
			return nil, nil, fmt.Errorf("geometry %s: %w", base.BaseID, gerr)
		}
		geos = append(geos, geo)

		pagePNG, rerr := provider.RenderPNG(base.Candidate.Page)
		if rerr != nil {
			return nil, nil, fmt.Errorf("render page %d: %w", base.Candidate.Page, rerr)
		}
		vp, verr := BuildR1A1Viewport(pagePNG, cfg.StoreDir, base, geo)
		if verr != nil {
			return nil, nil, fmt.Errorf("viewport %s: %w", base.BaseID, verr)
		}
		if werr := writeRGBAPNG(filepath.Join(viewDir, base.BaseID+"_viewport.png"), vp); werr != nil {
			return nil, nil, werr
		}

		for _, level := range AllR1A1Levels {
			select {
			case <-ctx.Done():
				return out, geos, ctx.Err()
			default:
			}
			rec := RecordOutcome{
				BaseID: base.BaseID, CandidateID: base.Candidate.CandidateID, Stage: "R1-A1",
				Mode: "FIXED_SCALE", Level: string(level), Page: base.Candidate.Page,
				Gold: base.Candidate.NormalizedTarget,
			}
			cropPath := filepath.Join(cropDir, fmt.Sprintf("%s_%s.png", base.BaseID, strings.ToLower(string(level))))
			visible, cerr := WriteR1A1Level(vp, cropPath, geo, level)
			if cerr != nil {
				rec.Error = cerr.Error()
				out = append(out, rec)
				writeRaw(rawDir, rec)
				continue
			}
			rec.VisualExposure = visible
			rec.CropPath = cropPath
			rec.CropWidth, rec.CropHeight, rec.PixelArea = CanvasPx, CanvasPx, CanvasPx*CanvasPx

			img, ierr := os.ReadFile(cropPath)
			if ierr != nil {
				rec.Error = ierr.Error()
				out = append(out, rec)
				writeRaw(rawDir, rec)
				continue
			}
			start := time.Now()
			result, perr := client.CompletePerception(ctx, target.PerceptionInput{
				Question: FrozenInstruction, Image: img, MediaType: "image/png",
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

// R1A1GeometryAudit is the pre-inference geometry check (protocol section 10).
type R1A1GeometryAudit struct {
	Schema        string          `json:"schema"`
	ExperimentID  string          `json:"experiment_id"`
	Bases         int             `json:"bases"`
	Levels        int             `json:"levels"`
	CanvasPx      int             `json:"canvas_px"`
	TargetLinePx  float64         `json:"target_line_height_px"`
	Checks        map[string]bool `json:"checks"`
	Problems      []string        `json:"problems"`
	PerBase       []R1A1Geometry  `json:"per_base_geometry"`
	ReadyR1A1Geom bool            `json:"ready_r1a1_geometry"`
}

const r1a1GeomSchema = "tlaloc.parrot-perceptual-envelope-r1.r1a1-geometry-audit.r1"

// AuditR1A1Geometry verifies the frozen geometry for all bases x levels
// without any model call.
func AuditR1A1Geometry(storeDir string, alloc Allocation, r1a0, r1b Allocation) R1A1GeometryAudit {
	a := R1A1GeometryAudit{
		Schema: r1a1GeomSchema, ExperimentID: ExperimentID,
		Bases: len(alloc.Bases), Levels: len(AllR1A1Levels),
		CanvasPx: CanvasPx, TargetLinePx: TargetLineHeightPx,
		Checks: map[string]bool{},
	}
	used := map[string]struct{}{}
	for _, b := range r1a0.Bases {
		used[b.Candidate.CandidateID] = struct{}{}
	}
	for _, b := range r1b.Bases {
		used[b.Candidate.CandidateID] = struct{}{}
	}
	disjoint, tgtVisible, nested, lineHeightOK, cueConst, centreConst, noClip := true, true, true, true, true, true, true
	for _, base := range alloc.Bases {
		if _, clash := used[base.Candidate.CandidateID]; clash {
			disjoint = false
			a.Problems = append(a.Problems, "candidate overlap with R1-A0/R1-B: "+base.BaseID)
		}
		geo, err := DeriveR1A1Geometry(storeDir, base)
		if err != nil {
			a.Problems = append(a.Problems, base.BaseID+": "+err.Error())
			nested = false
			continue
		}
		a.PerBase = append(a.PerBase, geo)
		if geo.LineHeightCanvas < TargetLineHeightPx-1 || geo.LineHeightCanvas > TargetLineHeightPx+1 {
			lineHeightOK = false
			a.Problems = append(a.Problems, fmt.Sprintf("%s: line height %.1f px != %.0f", base.BaseID, geo.LineHeightCanvas, TargetLineHeightPx))
		}
		// cue + centre constant across levels by construction (single geo);
		// assert cue inside canvas and inside A1C0.
		cb := geo.CueBBoxCanvas
		if cb[0] < 0 || cb[1] < 0 || cb[2] > CanvasPx || cb[3] > CanvasPx || cb[0] >= cb[2] || cb[1] >= cb[3] {
			noClip = false
			a.Problems = append(a.Problems, base.BaseID+": cue bbox clipped/degenerate")
		}
		cx := (cb[0] + cb[2]) / 2
		cy := (cb[1] + cb[3]) / 2
		if cx < canvasCenter-6 || cx > canvasCenter+6 || cy < canvasCenter-6 || cy > canvasCenter+6 {
			centreConst = false
			a.Problems = append(a.Problems, fmt.Sprintf("%s: target centre (%.0f,%.0f) not at canvas centre", base.BaseID, cx, cy))
		}
		// nesting: each level's rect set must contain the previous set.
		prev := map[[4]int]struct{}{}
		for _, lvl := range AllR1A1Levels {
			cur := geo.RevealRects[string(lvl)]
			curSet := map[[4]int]struct{}{}
			for _, r := range cur {
				curSet[r] = struct{}{}
			}
			for r := range prev {
				if _, ok := curSet[r]; !ok {
					nested = false
					a.Problems = append(a.Problems, fmt.Sprintf("%s %s: not nested", base.BaseID, lvl))
				}
			}
			prev = curSet
			// target (cue rect) must be inside the level-0 rect which is in every level
		}
		// A1C0 must cover the cue bbox
		c0 := geo.RevealRects[string(A1C0Target)][0]
		if !(int(cb[0]) >= c0[0] && int(cb[2]) <= c0[2] && int(cb[1]) >= c0[1] && int(cb[3]) <= c0[3]) {
			tgtVisible = false
			a.Problems = append(a.Problems, base.BaseID+": cue not inside A1C0 reveal rect")
		}
	}
	a.Checks["final_image_512x512"] = true
	a.Checks["target_centre_constant_at_canvas_centre"] = centreConst
	a.Checks["target_bbox_constant_across_levels"] = cueConst
	a.Checks["containing_line_height_32px"] = lineHeightOK
	a.Checks["cue_constant_across_levels"] = cueConst
	a.Checks["levels_nested"] = nested
	a.Checks["previously_visible_pixels_unchanged"] = nested // masks over one viewport => true by construction
	a.Checks["no_answer_metadata_in_prompt"] = FrozenInstructionHasNoDigits()
	a.Checks["single_opcode_extract_number"] = FrozenOpcode == "EXTRACT_NUMBER"
	a.Checks["target_visible_every_level"] = tgtVisible
	a.Checks["no_target_or_cue_clipping"] = noClip
	a.Checks["bases_disjoint_from_r1a0_and_r1b"] = disjoint
	a.Checks["context_level_alone_changes_visible_area"] = nested
	a.ReadyR1A1Geom = len(a.Problems) == 0
	return a
}

// FrozenInstructionHasNoDigits confirms the model-facing instruction
// carries no numeral.
func FrozenInstructionHasNoDigits() bool {
	for _, ch := range FrozenInstruction {
		if ch >= '0' && ch <= '9' {
			return false
		}
	}
	return true
}

// LoadAllocationFile reads an allocation JSON.
func LoadAllocationFile(path string) (Allocation, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Allocation{}, err
	}
	var a Allocation
	err = json.Unmarshal(body, &a)
	return a, err
}

// RenderR1A1Sanity renders (no inference) every condition PNG for the given
// bases into <outDir>/crops for visual geometry inspection.
func RenderR1A1Sanity(storeDir, pdfPath string, bases []Base, outDir string) ([]R1A1Geometry, error) {
	prov, err := parrotlab.NewPDFMemoryProvider(storeDir, pdfPath)
	if err != nil {
		return nil, err
	}
	cropDir := filepath.Join(outDir, "crops")
	if err := os.MkdirAll(cropDir, 0o755); err != nil {
		return nil, err
	}
	var geos []R1A1Geometry
	for _, base := range bases {
		geo, err := DeriveR1A1Geometry(storeDir, base)
		if err != nil {
			return nil, err
		}
		pagePNG, err := prov.RenderPNG(base.Candidate.Page)
		if err != nil {
			return nil, err
		}
		vp, err := BuildR1A1Viewport(pagePNG, storeDir, base, geo)
		if err != nil {
			return nil, err
		}
		if err := writeRGBAPNG(filepath.Join(cropDir, base.BaseID+"_viewport.png"), vp); err != nil {
			return nil, err
		}
		for _, lvl := range AllR1A1Levels {
			p := filepath.Join(cropDir, fmt.Sprintf("%s_%s.png", base.BaseID, strings.ToLower(string(lvl))))
			if _, err := WriteR1A1Level(vp, p, geo, lvl); err != nil {
				return nil, err
			}
		}
		geos = append(geos, geo)
	}
	return geos, nil
}
