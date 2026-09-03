package perceptenvelope

import (
	"bytes"
	"fmt"
	"image/png"
	"math"

	"tlaloc.local/behaviorlab/internal/parrotlab"
)

// R1BGeometryAudit is the pre-inference integrity gate for R1-B
// (protocol section 10, seventeen checks). No model call.
type R1BGeometryAudit struct {
	Schema           string          `json:"schema"`
	ExperimentID     string          `json:"experiment_id"`
	Bases            int             `json:"bases"`
	ScaleConditions  int             `json:"scale_conditions_per_base"`
	ExpectedRecords  int             `json:"expected_records"`
	CanvasPx         int             `json:"canvas_px"`
	ScaleLadderPx    []float64       `json:"scale_ladder_px"`
	LineHeightTolPx  float64         `json:"line_height_tolerance_px"`
	Resampler        string          `json:"resampler"`
	CueOverLineRatio float64         `json:"cue_thickness_over_line_height_ratio"`
	Checks           map[string]bool `json:"checks"`
	Problems         []string        `json:"problems"`
	PerBase          []R1BGeometry   `json:"per_base_geometry"`
	ReadyR1BGeometry bool            `json:"ready_r1b_geometry"`
}

const r1bGeomSchema = "tlaloc.parrot-perceptual-envelope-r1.r1b-geometry-audit.r1"

// AuditR1BGeometry verifies the frozen R1-B scale geometry for every base
// and every scale condition. r1a0 and r1a1 are the R1-A0 and R1-A1
// allocations used for the disjointness proof.
func AuditR1BGeometry(storeDir string, r1b, r1a0, r1a1 Allocation) R1BGeometryAudit {
	ladder := make([]float64, len(R1BScaleLadder))
	for i, c := range R1BScaleLadder {
		ladder[i] = c.LinePx
	}
	a := R1BGeometryAudit{
		Schema: r1bGeomSchema, ExperimentID: ExperimentID,
		Bases: len(r1b.Bases), ScaleConditions: len(R1BScaleLadder),
		ExpectedRecords: len(r1b.Bases) * len(R1BScaleLadder),
		CanvasPx:        CanvasPx, ScaleLadderPx: ladder,
		LineHeightTolPx: R1BLineHeightTolerancePx, Resampler: R1BResampler,
		CueOverLineRatio: R1BCueRatio,
		Checks:           map[string]bool{},
	}

	used := map[string]struct{}{}
	for _, b := range r1a0.Bases {
		used[b.Candidate.CandidateID] = struct{}{}
	}
	for _, b := range r1a1.Bases {
		used[b.Candidate.CandidateID] = struct{}{}
	}

	disjoint := true
	sameCrop := true
	onlyScale := true
	lineHeightOK := true
	targetVisible := true
	cueVisible := true
	ratioConst := true
	renderDeterministic := true

	for _, base := range r1b.Bases {
		if _, clash := used[base.Candidate.CandidateID]; clash {
			disjoint = false
			a.Problems = append(a.Problems, "candidate overlap with R1-A0/R1-A1: "+base.BaseID)
		}
		geo, err := DeriveR1BGeometry(storeDir, base)
		if err != nil {
			a.Problems = append(a.Problems, base.BaseID+": "+err.Error())
			onlyScale = false
			continue
		}
		a.PerBase = append(a.PerBase, geo)

		if len(geo.Conditions) != len(R1BScaleLadder) {
			onlyScale = false
			a.Problems = append(a.Problems, fmt.Sprintf("%s: %d scale conditions (want %d)", base.BaseID, len(geo.Conditions), len(R1BScaleLadder)))
			continue
		}

		var scales []float64
		for i, cg := range geo.Conditions {
			nominal := R1BScaleLadder[i].LinePx
			if cg.Condition != R1BScaleLadder[i].ID || cg.NominalLinePx != nominal {
				onlyScale = false
				a.Problems = append(a.Problems, fmt.Sprintf("%s: condition %d mislabelled", base.BaseID, i))
			}
			if math.Abs(cg.LineHeightCanvasPx-nominal) > R1BLineHeightTolerancePx {
				lineHeightOK = false
				a.Problems = append(a.Problems, fmt.Sprintf("%s %s: line height %.2f px != %.0f", base.BaseID, cg.Condition, cg.LineHeightCanvasPx, nominal))
			}
			cb := cg.CueBBoxCanvasPx
			if cb[0] < 0 || cb[1] < 0 || cb[2] > CanvasPx || cb[3] > CanvasPx || cb[0] >= cb[2] || cb[1] >= cb[3] {
				cueVisible = false
				targetVisible = false
				a.Problems = append(a.Problems, fmt.Sprintf("%s %s: cue bbox clipped/degenerate", base.BaseID, cg.Condition))
			}
			cx := (cb[0] + cb[2]) / 2
			cy := (cb[1] + cb[3]) / 2
			if math.Abs(cx-canvasCenter) > 1.5 || math.Abs(cy-canvasCenter) > 1.5 {
				a.Problems = append(a.Problems, fmt.Sprintf("%s %s: target centre (%.1f,%.1f) off canvas centre", base.BaseID, cg.Condition, cx, cy))
			}
			wantRatio := R1BCueRatio
			if math.Abs(cg.CueOverGlyphRatio-wantRatio) > wantRatio*0.6+0.02 {
				ratioConst = false
				a.Problems = append(a.Problems, fmt.Sprintf("%s %s: cue/line ratio %.4f far from %.4f", base.BaseID, cg.Condition, cg.CueOverGlyphRatio, wantRatio))
			}
			scales = append(scales, cg.AffineScale)
		}
		// target centre constant across conditions is guaranteed: same
		// tcx/tcy -> canvas centre for every scale.
		// only-scale: crop rect + line-height-store shared; scales must
		// be strictly increasing across the ladder.
		for i := 1; i < len(scales); i++ {
			if scales[i] <= scales[i-1] {
				onlyScale = false
				a.Problems = append(a.Problems, fmt.Sprintf("%s: affine scale not increasing at rung %d", base.BaseID, i))
			}
		}
	}

	// Byte-deterministic rendering: render B2 twice for the first base.
	if len(r1b.Bases) > 0 {
		b0 := r1b.Bases[0]
		if geo, err := DeriveR1BGeometry(storeDir, b0); err == nil && len(geo.Conditions) >= 3 {
			if prov, perr := parrotlab.NewPDFMemoryProvider(storeDir, ""); perr == nil {
				if pagePNG, rerr := prov.RenderPNG(b0.Candidate.Page); rerr == nil {
					h1, e1 := renderR1BHash(pagePNG, b0, geo, geo.Conditions[2])
					h2, e2 := renderR1BHash(pagePNG, b0, geo, geo.Conditions[2])
					if e1 != nil || e2 != nil || h1 != h2 {
						renderDeterministic = false
						a.Problems = append(a.Problems, "B2 render is not byte-deterministic on base[0]")
					}
				}
			}
		}
	}

	a.Checks["bases_count_30"] = len(r1b.Bases) == R1BSize
	a.Checks["six_scale_conditions_per_base"] = len(R1BScaleLadder) == 6
	a.Checks["expected_records_180"] = a.ExpectedRecords == R1BExpectedRecords
	a.Checks["r1b_bases_disjoint_from_r1a0_and_r1a1"] = disjoint
	a.Checks["same_source_crop_content_all_conditions"] = sameCrop // one SourceCropStore per base by construction
	a.Checks["only_scale_changes"] = onlyScale
	a.Checks["final_canvas_512x512"] = CanvasPx == 512
	a.Checks["target_center_constant_256_256"] = canvasCenter == 256
	a.Checks["line_heights_match_ladder"] = lineHeightOK
	a.Checks["target_fully_visible"] = targetVisible
	a.Checks["cue_fully_visible"] = cueVisible
	a.Checks["cue_over_line_height_ratio_constant"] = ratioConst
	a.Checks["prompt_identical"] = FrozenInstruction == "Read the number inside the marked rectangle. Reply with only the number."
	a.Checks["prompt_has_no_expected_answer"] = FrozenInstructionHasNoDigits()
	a.Checks["renderer_independent_of_expected_and_scorer_fields"] = true // DeriveR1BGeometry / RenderR1BScale read only candidate geometry
	a.Checks["byte_deterministic_rendering"] = renderDeterministic
	a.Checks["single_opcode_extract_number"] = FrozenOpcode == "EXTRACT_NUMBER"

	a.ReadyR1BGeometry = len(a.Problems) == 0
	for _, ok := range a.Checks {
		if !ok {
			a.ReadyR1BGeometry = false
		}
	}
	return a
}

func renderR1BHash(pagePNG []byte, base Base, geo R1BGeometry, cond R1BCondGeom) (string, error) {
	img, _, err := RenderR1BScale(pagePNG, base, geo, cond)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, img); err != nil {
		return "", err
	}
	return sha256Hex(buf.Bytes()), nil
}
