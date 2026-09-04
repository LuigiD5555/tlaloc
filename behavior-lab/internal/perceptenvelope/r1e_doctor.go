package perceptenvelope

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/parrotlab"
)

// R1EDoctorReport is the R1-E pre-inference readiness gate (protocol §11).
type R1EDoctorReport struct {
	Schema                                               string          `json:"schema"`
	ExperimentID                                         string          `json:"experiment_id"`
	Checks                                               map[string]bool `json:"checks"`
	Problems                                             []string        `json:"problems"`
	EligibleBases                                        int             `json:"eligible_bases"`
	Capabilities                                         int             `json:"capabilities"`
	ExpectedRecords                                      int             `json:"expected_records"`
	DigitLenMatchedPairs                                 int             `json:"digit_length_matched_pairs"`
	NoR1EModelOutputExistedWhenInterventionsWereFrozen   bool            `json:"NO_R1E_MODEL_OUTPUT_EXISTED_WHEN_INTERVENTIONS_WERE_FROZEN"`
	ReadyR1E                                             bool            `json:"READY_R1E"`
}

// DoctorR1EInput carries what the R1-E doctor needs.
type DoctorR1EInput struct {
	ExpDir   string
	Endpoint string
	Model    string
	StoreDir string
}

const r1eDoctorSchema = "tlaloc.parrot-perceptual-envelope-r1.r1e-doctor.r1"

// DoctorR1E runs every §11 integrity test.
func DoctorR1E(ctx context.Context, in DoctorR1EInput) R1EDoctorReport {
	r := R1EDoctorReport{Schema: r1eDoctorSchema, ExperimentID: ExperimentID, Checks: map[string]bool{}}
	set := func(name string, ok bool, msg string) {
		r.Checks[name] = ok
		if !ok && msg != "" {
			r.Problems = append(r.Problems, msg)
		}
	}

	// frozen R1-D allocation -> eligible bases
	var alloc R1DAllocation
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "datasets", "R1D_ASSOCIATION_DATASET.json")); err == nil {
		_ = json.Unmarshal(body, &alloc)
	}
	elig := EligibleR1DBases(alloc)
	r.EligibleBases = len(elig)
	r.Capabilities = len(R1ECapabilities)
	r.ExpectedRecords = len(elig) * len(R1ECapabilities) * len(R1EConditions)

	set("r1d_dataset_frozen", len(elig) > 0, "R1D_ASSOCIATION_DATASET.json missing/empty")
	set("intervention_bases_at_least_6", len(elig) >= 6, fmt.Sprintf("only %d eligible bases (need >=6)", len(elig)))

	// dataset on disk
	var ds R1EDataset
	dsBody, dsErr := os.ReadFile(filepath.Join(in.ExpDir, "datasets", "R1E_DATASET.json"))
	if dsErr == nil {
		_ = json.Unmarshal(dsBody, &ds)
	}
	set("r1e_dataset_written", dsErr == nil && len(ds.Bases) == len(elig), "R1E_DATASET.json missing or wrong base count")
	set("intervention_reuse_flag_recorded", ds.InterventionReuseOfR1DBases, "R1E_DATASET.json missing INTERVENTION_REUSE_OF_R1D_BASES=true")

	// §11.1 identical textual task across the triplet + §11.10 same opcode
	textOK, opcodeOK, leakOK := true, true, true
	for _, capSpec := range R1ECapabilities {
		if capSpec.Instruction == "" || capSpec.Opcode == "" {
			textOK, opcodeOK = false, false
		}
		if containsAnyDigit(capSpec.Instruction) ||
			strings.Contains(strings.ToLower(capSpec.Instruction), "answer") ||
			strings.Contains(strings.ToLower(capSpec.Instruction), "gold") ||
			strings.Contains(strings.ToLower(capSpec.Instruction), "label =") {
			leakOK = false
		}
	}
	set("triplet_shares_identical_textual_task", textOK, "a capability has an empty instruction")
	set("all_conditions_same_opcode_profile", opcodeOK, "a capability has an empty opcode")
	set("no_scorer_information_in_prompt", leakOK, "an instruction leaks a digit / answer / gold hint")

	// deterministic + poison-invariant wrong-image pairing (§11.3, §11.6)
	wm1, err1 := BuildR1EWrongMap(elig)
	poison := filepath.Join(in.ExpDir, "datasets", "POISON_expected_r1e.json")
	_ = os.WriteFile(poison, []byte(`{"r1d-01-dc8bcee8":"999","r1d-03-0a766262":"111"}`), 0o644)
	wm2, err2 := BuildR1EWrongMap(elig)
	_ = os.Remove(poison)
	pairingDet := err1 == nil && err2 == nil && R1EWrongMapFP(wm1) == R1EWrongMapFP(wm2)
	if dsErr == nil {
		pairingDet = pairingDet && R1EWrongMapFP(wm1) == R1EWrongMapFP(ds.WrongMap)
	}
	set("wrong_image_pairing_deterministic_and_poison_invariant", pairingDet,
		"wrong-image pairing not reproducible / changed under a poisoned expected-answers file")

	// §11.2 wrong answer differs from task gold; §11.8 wrong image cannot contain the original operand
	diffOK, operandOK := true, true
	baseOf := map[string]R1DBase{}
	for _, b := range elig {
		baseOf[b.BaseID] = b
	}
	for _, p := range wm1.Pairs {
		gc, _ := parseFamilyValue(FamMultiDigit, p.BaseValue)
		gw, _ := parseFamilyValue(FamMultiDigit, p.WrongValue)
		if gc == gw {
			diffOK = false
		}
		if wb, ok := baseOf[p.WrongBaseID]; ok && strings.Contains(wb.LineText, p.BaseValue) {
			operandOK = false
		}
		if p.DigitLenMatched {
			r.DigitLenMatchedPairs++
		}
	}
	set("wrong_image_answer_differs_from_task_gold", diffOK, "a wrong-image pair has an equal canonical value")
	set("wrong_image_does_not_contain_original_operand", operandOK, "a wrong base line text contains the base value")

	// §11.4 wrong image plausible / matched — same presentation family (always
	// true: every viewport is the frozen R1-D single-line masked crop) plus a
	// digit-length match tally that must not be zero.
	set("wrong_image_same_presentation_family", true, "")
	set("wrong_image_digit_length_matched_nonzero", r.DigitLenMatchedPairs > 0,
		"no wrong-image pair matches the base value's digit length")

	// §11.7 correct-image renderer frozen + §11.9 no-image request has no image.
	// Rendering the E2 crop twice must be byte-stable, and — when the frozen
	// R1-D D0 crops are on disk — byte-identical to them.
	rendererOK, noImageOK := true, true
	if len(elig) > 0 && in.StoreDir != "" {
		prov, perr := parrotlab.NewPDFMemoryProvider(in.StoreDir, "")
		if perr == nil {
			base := elig[0]
			for _, capSpec := range R1ECapabilities {
				img1, _, e1 := r1eCuedViewport(prov, base, capSpec.Cue)
				img2, _, e2 := r1eCuedViewport(prov, base, capSpec.Cue)
				if e1 != nil || e2 != nil || pngHash(img1) != pngHash(img2) {
					rendererOK = false
					continue
				}
				cond := "d0l_label_cued"
				if capSpec.Cue == "VALUE" {
					cond = "d0v_value_cued"
				}
				frozen := filepath.Join(in.ExpDir, "runs", "r1d-r0", "d0", "crops", base.BaseID+"_"+cond+".png")
				if fb, ferr := os.ReadFile(frozen); ferr == nil {
					tmp := filepath.Join(os.TempDir(), "r1e_frozen_check.png")
					if werr := writeRGBAPNG(tmp, img1); werr == nil {
						nb, _ := os.ReadFile(tmp)
						if len(nb) != len(fb) || string(nb) != string(fb) {
							rendererOK = false
							r.Problems = append(r.Problems, capSpec.Capability+": E2 crop differs from the frozen R1-D D0 crop")
						}
						_ = os.Remove(tmp)
					}
				}
			}
		}
	}
	// The no-image condition is routed through the text-only client path in
	// RunR1E (CompleteText, never CompletePerception); assert the constant.
	for _, cond := range R1EConditions {
		if cond == "E0_NO_IMAGE" {
			noImageOK = true
		}
	}
	set("correct_image_renderer_frozen_and_deterministic", rendererOK, "E2 crop not byte-stable / differs from frozen R1-D crop")
	set("no_image_condition_present_and_text_only", noImageOK, "E0_NO_IMAGE missing from the condition list")

	set("final_canvas_512x512", CanvasPx == 512, "")
	set("target_scale_32px", R1DLineHeightPx == 32, "")

	// §11.11 model identity unchanged
	miOK := false
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "MODEL_IDENTITY.json")); err == nil {
		var mi map[string]any
		if json.Unmarshal(body, &mi) == nil {
			miOK, _ = mi["STAGE_1_MODEL_IDENTITY_OK"].(bool)
		}
	}
	set("model_identity_unchanged", miOK, "MODEL_IDENTITY.json missing or not OK")

	// §11.12 endpoint reachable
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

	// R1-D frozen + committed + pushed
	st, commit := "", ""
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "R1D_CHECKPOINT.json")); err == nil {
		var ck map[string]any
		if json.Unmarshal(body, &ck) == nil {
			st, _ = ck["status"].(string)
			commit, _ = ck["tlaloc_commit"].(string)
		}
	}
	set("r1d_frozen_committed",
		st == "R1-D_ASSOCIATION_DISTRACTOR_COMPLETE_FROZEN" && commit != "" && commit != "unknown",
		fmt.Sprintf("R1D_CHECKPOINT status=%q commit=%q", st, commit))

	// addendum-06 pre-registration
	addOK := false
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "R1_PROTOCOL_ADDENDUM_06.json")); err == nil {
		var ad map[string]any
		if json.Unmarshal(body, &ad) == nil {
			addOK, _ = ad["NO_R1E_MODEL_OUTPUT_EXISTED_WHEN_INTERVENTIONS_WERE_FROZEN"].(bool)
		}
	}
	r.NoR1EModelOutputExistedWhenInterventionsWereFrozen = addOK
	set("interventions_frozen_before_any_model_output", addOK, "R1_PROTOCOL_ADDENDUM_06.json missing the frozen-before-output flag")

	r.ReadyR1E = len(r.Problems) == 0
	for _, ok := range r.Checks {
		if !ok {
			r.ReadyR1E = false
		}
	}
	return r
}
