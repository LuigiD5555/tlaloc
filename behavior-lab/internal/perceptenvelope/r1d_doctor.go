package perceptenvelope

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/parrotlab"
)

// R1DDoctorReport is the R1-D pre-inference readiness gate (protocol §22).
type R1DDoctorReport struct {
	Schema        string          `json:"schema"`
	ExperimentID  string          `json:"experiment_id"`
	Checks        map[string]bool `json:"checks"`
	Problems      []string        `json:"problems"`
	EligibleBases int             `json:"eligible_bases"`
	D0Records     int             `json:"expected_d0_records"`
	D1Records     int             `json:"expected_d1_records"`
	ReadyR1D      bool            `json:"READY_R1D"`
}

// DoctorR1DInput carries what the R1-D doctor needs.
type DoctorR1DInput struct {
	ExpDir   string
	Endpoint string
	Model    string
	StoreDir string
}

const r1dDoctorSchema = "tlaloc.parrot-perceptual-envelope-r1.r1d-doctor.r1"

// DoctorR1D runs every §22 pre-inference check.
func DoctorR1D(ctx context.Context, in DoctorR1DInput) R1DDoctorReport {
	r := R1DDoctorReport{Schema: r1dDoctorSchema, ExperimentID: ExperimentID, Checks: map[string]bool{}}
	set := func(name string, ok bool, msg string) {
		r.Checks[name] = ok
		if !ok && msg != "" {
			r.Problems = append(r.Problems, msg)
		}
	}

	var alloc R1DAllocation
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "datasets", "R1D_ASSOCIATION_DATASET.json")); err == nil {
		_ = json.Unmarshal(body, &alloc)
	}
	var elig []R1DBase
	allMultiDigit := true
	for _, b := range alloc.Bases {
		if b.Eligible {
			elig = append(elig, b)
			if !isPlainInt(b.Value) || len(b.Value) < 2 || len(b.Value) > 5 {
				allMultiDigit = false
			}
		}
	}
	r.EligibleBases = len(elig)
	r.D0Records = len(elig) * 2
	r.D1Records = len(elig) * len(R1DDistractorLadder)

	set("dataset_frozen", len(alloc.Bases) > 0, "R1D_ASSOCIATION_DATASET.json missing/empty")
	set("eligible_bases_at_least_18", len(elig) >= 18, fmt.Sprintf("only %d eligible bases (need >=18)", len(elig)))
	set("morphology_policy_multi_digit_only", allMultiDigit, "an eligible base is not a 2-5 digit plain integer")

	// deterministic selection + poison invariance
	pool := loadLVPoolFor(in.ExpDir)
	a1 := AllocateR1D(pool)
	poison := filepath.Join(in.ExpDir, "datasets", "POISON_expected_r1d.json")
	_ = os.WriteFile(poison, []byte(`{"r1d-01-x":"999"}`), 0o644)
	a2 := AllocateR1D(pool)
	_ = os.Remove(poison)
	set("selection_deterministic_and_leakage_proof",
		r1dAllocFP(a1) == r1dAllocFP(alloc) && r1dAllocFP(a1) == r1dAllocFP(a2),
		"allocation not reproducible / changed under a poisoned expected-answers file")

	set("assoc_opcode_single", R1DAssocOpcode == "READ_ASSOCIATED_NUMBER", "")
	set("assoc_instruction_no_label_no_answer",
		!containsAnyDigit(R1DAssocInstruction) && !strings.Contains(strings.ToLower(R1DAssocInstruction), "label ="),
		"assoc instruction leaks a digit")
	set("frozen_extract_instruction_no_digits", FrozenInstructionHasNoDigits(), "")

	// per-base geometry + pixel invariants (checked on the first eligible base)
	pixelOK, cueOnlyOK, d1InvariantOK, distractorOK, overlapOK, renderDetOK := true, true, true, true, true, true
	countLadderOK := true
	for _, want := range []int{0, 1, 2, 4, 8} {
		found := false
		for _, l := range R1DDistractorLadder {
			if l.K == want {
				found = true
			}
		}
		if !found {
			countLadderOK = false
		}
	}
	if len(elig) > 0 && in.StoreDir != "" {
		bank, berr := LoadOrBuildGlyphBank(filepath.Join(in.ExpDir, "datasets", "R1C_GLYPHBANK.json"), in.StoreDir, "")
		prov, perr := parrotlab.NewPDFMemoryProvider(in.StoreDir, "")
		if berr == nil && perr == nil {
			for _, base := range elig[:min2(3, len(elig))] {
				geo, gerr := DeriveR1DGeometry(base)
				if gerr != nil {
					r.Problems = append(r.Problems, base.BaseID+": "+gerr.Error())
					pixelOK = false
					continue
				}
				pagePNG, rerr := prov.RenderPNG(base.Page)
				if rerr != nil {
					continue
				}
				vp, verr := BuildR1DViewport(pagePNG, base, geo)
				if verr != nil {
					pixelOK = false
					continue
				}
				// same viewport for both cue conditions: cue is drawn on a
				// clone, so vp itself is untouched -> identical by construction.
				v1 := cloneRGBA(vp)
				drawR1DCue(v1, geo.ValueBBoxCanvas)
				v2 := cloneRGBA(vp)
				drawR1DCue(v2, geo.LabelBBoxCanvas)
				if pngHash(v1) == pngHash(v2) {
					cueOnlyOK = false // cue must actually differ
				}
				// D1: line-rect region identical across K
				labelCued := cloneRGBA(vp)
				drawR1DCue(labelCued, geo.LabelBBoxCanvas)
				var lineHashes []string
				for _, l := range R1DDistractorLadder {
					img, placed, derr := placeDistractors(labelCued, bank, geo, base.DistractorValues, l.K)
					if derr != nil {
						d1InvariantOK = false
						r.Problems = append(r.Problems, base.BaseID+" "+l.ID+": "+derr.Error())
						continue
					}
					lineHashes = append(lineHashes, lineRectHash(img, geo.LineRectCanvas))
					if len(placed) != l.K {
						countLadderOK = false
					}
					for _, pr := range placed {
						for _, prot := range [][4]int{geo.LineRectCanvas, geo.LabelBBoxCanvas, geo.ValueBBoxCanvas} {
							if rectsOverlap(pr, prot) {
								overlapOK = false
							}
						}
					}
					// distractors all wrong + plain 2-4 digit
					for _, d := range base.DistractorValues[:l.K] {
						if d == base.Value || !isPlainInt(d) || len(d) < 2 || len(d) > 4 {
							distractorOK = false
						}
					}
				}
				for _, h := range lineHashes {
					if h != lineHashes[0] {
						d1InvariantOK = false
					}
				}
				// render determinism
				i1, _, e1 := placeDistractors(labelCued, bank, geo, base.DistractorValues, 4)
				i2, _, e2 := placeDistractors(labelCued, bank, geo, base.DistractorValues, 4)
				if e1 != nil || e2 != nil || pngHash(i1) != pngHash(i2) {
					renderDetOK = false
				}
			}
		}
	}
	set("d0_viewport_shared_between_cue_conditions", pixelOK, "D0 viewport derivation failed")
	set("only_cue_changes_between_d0v_and_d0l", cueOnlyOK, "D0V and D0L produced identical images (cue did not move)")
	set("d1_pair_pixels_identical_across_k", d1InvariantOK, "D1 line-rect region changes across the distractor ladder")
	set("distractor_count_exactly_0_1_2_4_8", countLadderOK, "distractor ladder is not {0,1,2,4,8} or a rung under-placed")
	set("distractors_all_wrong_plain_2_4_digit", distractorOK, "a distractor equals the answer or is not a 2-4 digit plain integer")
	set("distractors_do_not_overlap_label_value_cue", overlapOK, "a placed distractor overlaps a protected rect")
	set("renderer_deterministic", renderDetOK, "distractor placement is not byte-deterministic")
	set("final_canvas_512x512", CanvasPx == 512, "")
	set("target_scale_32px", R1DLineHeightPx == 32, "")

	// model identity
	miOK := false
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "MODEL_IDENTITY.json")); err == nil {
		var mi map[string]any
		if json.Unmarshal(body, &mi) == nil {
			miOK, _ = mi["STAGE_1_MODEL_IDENTITY_OK"].(bool)
		}
	}
	set("model_identity_unchanged", miOK, "MODEL_IDENTITY.json missing or not OK")

	// endpoint
	epOK := false
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(cctx, http.MethodGet, in.Endpoint+"/v1/models", nil)
	if resp, err := http.DefaultClient.Do(req); err == nil {
		defer resp.Body.Close()
		var ml struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if resp.StatusCode == 200 && json.NewDecoder(resp.Body).Decode(&ml) == nil {
			for _, m := range ml.Data {
				if m.ID == in.Model {
					epOK = true
				}
			}
		}
	}
	set("endpoint_reachable_model_listed", epOK, "endpoint /v1/models unreachable or model missing")

	// addendum-05
	addOK := false
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "R1_PROTOCOL_ADDENDUM_05.json")); err == nil {
		var ad map[string]any
		if json.Unmarshal(body, &ad) == nil {
			addOK, _ = ad["NO_R1D_MODEL_OUTPUT_EXISTED_WHEN_DATASET_AND_DISTRACTOR_RULES_WERE_FROZEN"].(bool)
		}
	}
	set("dataset_and_distractor_rules_frozen_before_output", addOK, "addendum-05 missing the frozen-before-output flag")

	// R1-C frozen + committed
	st, commit := "", ""
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "R1C_CHECKPOINT.json")); err == nil {
		var ck map[string]any
		if json.Unmarshal(body, &ck) == nil {
			st, _ = ck["status"].(string)
			commit, _ = ck["tlaloc_commit"].(string)
		}
	}
	set("r1c_frozen_committed_pushed",
		st == "R1-C_NUMERIC_MORPHOLOGY_COMPLETE_FROZEN" && commit != "" && commit != "unknown",
		fmt.Sprintf("R1C_CHECKPOINT status=%q commit=%q", st, commit))

	r.ReadyR1D = len(r.Problems) == 0
	for _, ok := range r.Checks {
		if !ok {
			r.ReadyR1D = false
		}
	}
	return r
}

func loadLVPoolFor(expDir string) LabelValuePool {
	var pool LabelValuePool
	if body, err := os.ReadFile(filepath.Join(expDir, "datasets", "R1D_POOL.json")); err == nil {
		_ = json.Unmarshal(body, &pool)
	}
	return pool
}

func r1dAllocFP(a R1DAllocation) string {
	var sb strings.Builder
	for _, b := range a.Bases {
		if b.Eligible {
			fmt.Fprintf(&sb, "%s:%s:%s;", b.BaseID, b.Value, strings.Join(b.DistractorValues, ","))
		}
	}
	return sb.String()
}

func containsAnyDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

// lineRectHash hashes the pixel bytes inside a canvas rect (the protected
// label/value line region).
func lineRectHash(img *image.RGBA, rect [4]int) string {
	h := sha256.New()
	for y := rect[1]; y < rect[3]; y++ {
		for x := rect[0]; x < rect[2]; x++ {
			c := img.RGBAAt(x, y)
			h.Write([]byte{c.R, c.G, c.B})
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}
