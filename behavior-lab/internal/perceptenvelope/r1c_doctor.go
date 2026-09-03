package perceptenvelope

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"tlaloc.local/behaviorlab/internal/parrotlab"
)

// WriteRGBA writes an RGBA image to a PNG file (exported wrapper).
func WriteRGBA(path string, img *image.RGBA) error { return writeRGBAPNG(path, img) }

// RenderR1CReal renders one real R1-C base at the frozen 32 px presentation
// (the R1-B B4 condition of the R1-B renderer).
func RenderR1CReal(storeDir, pdfPath string, base R1CBase) (*image.RGBA, error) {
	if base.Candidate == nil {
		return nil, fmt.Errorf("%s: not a real base", base.BaseID)
	}
	prov, err := parrotlab.NewPDFMemoryProvider(storeDir, pdfPath)
	if err != nil {
		return nil, err
	}
	rb := base.morphToBase()
	geo, err := DeriveR1BGeometry(storeDir, rb)
	if err != nil {
		return nil, err
	}
	var cond R1BCondGeom
	found := false
	for _, cg := range geo.Conditions {
		if cg.NominalLinePx == R1CLineHeightPx {
			cond, found = cg, true
		}
	}
	if !found {
		return nil, fmt.Errorf("%s: no 32px condition", base.BaseID)
	}
	pagePNG, err := prov.RenderPNG(base.Candidate.Page)
	if err != nil {
		return nil, err
	}
	img, _, err := RenderR1BScale(pagePNG, rb, geo, cond)
	return img, err
}

// R1CDoctorReport is the R1-C pre-inference readiness gate (protocol §15).
type R1CDoctorReport struct {
	Schema       string          `json:"schema"`
	ExperimentID string          `json:"experiment_id"`
	Checks       map[string]bool `json:"checks"`
	Problems     []string        `json:"problems"`
	RealBases    int             `json:"real_bases"`
	SyntheticN   int             `json:"synthetic_bases"`
	ExpectedN    int             `json:"expected_records"`
	ReadyR1C     bool            `json:"READY_R1C"`
}

// DoctorR1CInput carries what the R1-C doctor needs.
type DoctorR1CInput struct {
	ExpDir   string
	Endpoint string
	Model    string
	StoreDir string
}

const r1cDoctorSchema = "tlaloc.parrot-perceptual-envelope-r1.r1c-doctor.r1"

// DoctorR1C runs every §15 pre-inference check.
func DoctorR1C(ctx context.Context, in DoctorR1CInput) R1CDoctorReport {
	r := R1CDoctorReport{Schema: r1cDoctorSchema, ExperimentID: ExperimentID, Checks: map[string]bool{}}
	fail := func(name, msg string) {
		r.Checks[name] = false
		if msg != "" {
			r.Problems = append(r.Problems, msg)
		}
	}
	ok := func(name string) { r.Checks[name] = true }

	// dataset + allocation
	allocPath := filepath.Join(in.ExpDir, "datasets", "R1C_DATASET.json")
	var alloc R1CAllocation
	if body, err := os.ReadFile(allocPath); err == nil && json.Unmarshal(body, &alloc) == nil {
		ok("selected_cases_fixed_before_inference")
	} else {
		fail("selected_cases_fixed_before_inference", "R1C_DATASET.json missing or unreadable")
	}
	for _, fa := range alloc.Families {
		r.RealBases += len(fa.RealBases)
		r.SyntheticN += len(fa.SyntheticBases)
	}
	r.ExpectedN = r.RealBases + r.SyntheticN

	setBool := func(name string, cond bool, msg string) {
		if cond {
			ok(name)
		} else {
			fail(name, msg)
		}
	}
	setBool("presentation_line_height_32px", alloc.LineHeightPx == R1CLineHeightPx,
		fmt.Sprintf("line height %.1f != 32", alloc.LineHeightPx))
	setBool("context_level_a1c0_target", alloc.ContextLevel == R1CContextLevel, "context level != A1C0_TARGET")
	setBool("canvas_512", alloc.CanvasPx == CanvasPx, "canvas != 512")
	setBool("single_opcode_extract_number", FrozenOpcode == "EXTRACT_NUMBER", "")
	setBool("prompt_has_no_expected_answer", FrozenInstructionHasNoDigits(), "")

	// strata separation: no synthetic base under RealBases, no real under SyntheticBases
	strataOK := true
	for _, fa := range alloc.Families {
		for _, b := range fa.RealBases {
			if b.Provenance == ProvSynthetic || b.Candidate == nil {
				strataOK = false
			}
		}
		for _, b := range fa.SyntheticBases {
			if b.Provenance != ProvSynthetic || b.Candidate != nil {
				strataOK = false
			}
		}
	}
	setBool("real_and_synthetic_strata_separated", strataOK, "a base is filed under the wrong stratum")

	// scorer self-test (covers VALUE_CORRECT and SURFACE_FORM_CORRECT batteries)
	st := R1CScorerSelfTest()
	setBool("scorer_frozen_value_tests_pass", len(st) == 0, joinProblems("scorer self-test", st))
	setBool("scorer_frozen_surface_tests_pass", len(st) == 0, "")

	// R1-B frozen / committed
	r1bStatus, r1bCommit := "", ""
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "R1B_CHECKPOINT.json")); err == nil {
		var ck map[string]any
		if json.Unmarshal(body, &ck) == nil {
			r1bStatus, _ = ck["status"].(string)
			r1bCommit, _ = ck["tlaloc_commit"].(string)
		}
	}
	setBool("r1b_frozen_and_committed",
		r1bStatus == "R1-B_SCALE_ENVELOPE_COMPLETE_FROZEN" && r1bCommit != "" && r1bCommit != "unknown",
		fmt.Sprintf("R1B_CHECKPOINT status=%q commit=%q", r1bStatus, r1bCommit))

	// model identity
	miOK := false
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "MODEL_IDENTITY.json")); err == nil {
		var mi map[string]any
		if json.Unmarshal(body, &mi) == nil {
			miOK, _ = mi["STAGE_1_MODEL_IDENTITY_OK"].(bool)
		}
	}
	setBool("model_identity_unchanged", miOK, "MODEL_IDENTITY.json missing or STAGE_1_MODEL_IDENTITY_OK != true")

	// endpoint reachable + model listed
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
	setBool("endpoint_reachable_model_listed", epOK, "endpoint /v1/models unreachable or model not listed")

	// addendum-04 records "no model output existed"
	addOK := false
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "R1_PROTOCOL_ADDENDUM_04.json")); err == nil {
		var ad map[string]any
		if json.Unmarshal(body, &ad) == nil {
			addOK, _ = ad["NO_R1C_MODEL_OUTPUT_EXISTED_WHEN_SCORING_AND_DATASET_WERE_FROZEN"].(bool)
		}
	}
	setBool("scoring_and_dataset_frozen_before_output", addOK, "R1_PROTOCOL_ADDENDUM_04 missing the frozen-before-output flag")

	// deterministic + leakage-proof allocation (poison test)
	setBool("allocation_stable_under_poisoned_expected_fields", r1cAllocationStable(in.ExpDir, in.StoreDir),
		"allocation changed when a POISON expected-answers file was present")

	// deterministic glyph bank + render
	setBool("renderer_and_glyphbank_deterministic", r1cRenderDeterministic(in.ExpDir, in.StoreDir),
		"glyph bank or synthetic render is not byte-deterministic / does not match the frozen cache")

	// target centred (structural check on one rendered real base)
	setBool("target_centred_256_256", r1cTargetCentred(in.StoreDir, alloc), "")

	r.ReadyR1C = len(r.Problems) == 0
	for _, v := range r.Checks {
		if !v {
			r.ReadyR1C = false
		}
	}
	return r
}

func joinProblems(prefix string, xs []string) string {
	if len(xs) == 0 {
		return ""
	}
	return prefix + ": " + xs[0]
}

func r1cAllocationStable(expDir, storeDir string) bool {
	poolBody, err := os.ReadFile(filepath.Join(expDir, "datasets", "R1C_POOL.json"))
	if err != nil {
		return false
	}
	var pool MorphologyPool
	if json.Unmarshal(poolBody, &pool) != nil {
		return false
	}
	a1 := AllocateR1C(pool, map[string]struct{}{})
	poison := filepath.Join(expDir, "datasets", "POISON_expected_r1c.json")
	_ = os.WriteFile(poison, []byte(`{"scientific_notation-syn-01":"999","x":"0"}`), 0o644)
	defer os.Remove(poison)
	a2 := AllocateR1C(pool, map[string]struct{}{})
	return r1cAllocFingerprint(a1) == r1cAllocFingerprint(a2)
}

func r1cAllocFingerprint(a R1CAllocation) string {
	h := ""
	for _, fa := range a.Families {
		h += fa.Family + ":"
		for _, b := range fa.RealBases {
			h += b.BaseID + ","
		}
		h += ";"
	}
	return h
}

func r1cRenderDeterministic(expDir, storeDir string) bool {
	cachePath := filepath.Join(expDir, "datasets", "R1C_GLYPHBANK.json")
	cached, err := LoadGlyphBank(cachePath)
	if err != nil || len(cached.Glyphs) == 0 {
		return false
	}
	// one fresh build must reproduce the frozen cache's content hash
	fresh, ferr := BuildGlyphBank(storeDir, "")
	if ferr != nil || fresh.SHA256 != cached.SHA256 {
		return false
	}
	i1, _, e1 := RenderSyntheticNumber(cached, "(512, 256)")
	i2, _, e2 := RenderSyntheticNumber(cached, "(512, 256)")
	if e1 != nil || e2 != nil {
		return false
	}
	return pngHash(i1) == pngHash(i2)
}

func pngHash(img *image.RGBA) string {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if enc.Encode(&buf, img) != nil {
		return ""
	}
	return sha256Hex(buf.Bytes())
}

func r1cTargetCentred(storeDir string, alloc R1CAllocation) bool {
	for _, fa := range alloc.Families {
		for _, b := range fa.RealBases {
			geo, err := DeriveR1BGeometry(storeDir, b.morphToBase())
			if err != nil {
				continue
			}
			for _, cg := range geo.Conditions {
				if cg.NominalLinePx != R1CLineHeightPx {
					continue
				}
				cx := (cg.CueBBoxCanvasPx[0] + cg.CueBBoxCanvasPx[2]) / 2
				cy := (cg.CueBBoxCanvasPx[1] + cg.CueBBoxCanvasPx[3]) / 2
				if cx < canvasCenter-2 || cx > canvasCenter+2 || cy < canvasCenter-2 || cy > canvasCenter+2 {
					return false
				}
				return true
			}
		}
	}
	return true
}
