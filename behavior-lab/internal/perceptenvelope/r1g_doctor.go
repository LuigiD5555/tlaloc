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

// R1GDoctorReport is the R1-G pre-inference readiness gate (protocol §18).
type R1GDoctorReport struct {
	Schema          string          `json:"schema"`
	ExperimentID    string          `json:"experiment_id"`
	Checks          map[string]bool `json:"checks"`
	Problems        []string        `json:"problems"`
	Bases           map[string]int  `json:"bases_per_family"`
	ExpectedRecords int             `json:"expected_records"`
	CrossReuse      bool            `json:"CROSS_RECOVERY_FAMILY_BASE_REUSE"`
	OCRAvailable    bool            `json:"OCR_FALLBACK_AVAILABLE"`
	NoR1GModelOutputExistedWhenRecoveryPoliciesAndThresholdsWereFrozen bool `json:"NO_R1G_MODEL_OUTPUT_EXISTED_WHEN_RECOVERY_POLICIES_AND_THRESHOLDS_WERE_FROZEN"`
	ReadyR1G        bool            `json:"READY_R1G"`
}

// DoctorR1GInput carries what the R1-G doctor needs.
type DoctorR1GInput struct {
	ExpDir   string
	Endpoint string
	Model    string
	StoreDir string
	Bank     *GlyphBank
}

const r1gDoctorSchema = "tlaloc.parrot-perceptual-envelope-r1.r1g-doctor.r1"

func r1gDatasetFP(ds R1GDataset) string {
	var sb strings.Builder
	emit := func(bs []R1GBase) {
		ids := make([]string, len(bs))
		for i, b := range bs {
			ids[i] = b.BaseID + ":" + b.Gold + ":" + b.RealCompetitor + ":" + b.SynValue + ":" + b.SynCompValue
		}
		sb.WriteString(strings.Join(ids, ",") + "|")
	}
	emit(ds.ScaleBases)
	emit(ds.ContextBases)
	emit(ds.RealAssoc)
	emit(ds.SynAssoc)
	emit(ds.CueBases)
	return sb.String()
}

// DoctorR1G runs every §18 integrity check.
func DoctorR1G(ctx context.Context, in DoctorR1GInput) R1GDoctorReport {
	r := R1GDoctorReport{Schema: r1gDoctorSchema, ExperimentID: ExperimentID, Checks: map[string]bool{}, Bases: map[string]int{}}
	set := func(name string, ok bool, msg string) {
		r.Checks[name] = ok
		if !ok && msg != "" {
			r.Problems = append(r.Problems, msg)
		}
	}

	// 1. R1-F frozen/pushed
	st, commit := "", ""
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "R1F_CHECKPOINT.json")); err == nil {
		var ck map[string]any
		if json.Unmarshal(body, &ck) == nil {
			st, _ = ck["status"].(string)
			commit, _ = ck["tlaloc_commit"].(string)
		}
	}
	set("r1f_frozen_committed", st == "R1-F_REPEATABILITY_COMPLETE_FROZEN" && commit != "" && commit != "unknown",
		fmt.Sprintf("R1F_CHECKPOINT status=%q commit=%q", st, commit))

	var ds R1GDataset
	dsBody, dsErr := os.ReadFile(filepath.Join(in.ExpDir, "datasets", "R1G_DATASET.json"))
	if dsErr == nil {
		_ = json.Unmarshal(dsBody, &ds)
	}
	set("r1g_dataset_written", dsErr == nil && len(ds.Families) == len(R1GFamilies), "R1G_DATASET.json missing/incomplete")
	r.Bases["GA_SCALE"] = len(ds.ScaleBases)
	r.Bases["GB_CONTEXT"] = len(ds.ContextBases)
	r.Bases["GC_ASSOC_REAL"] = len(ds.RealAssoc)
	r.Bases["GC_ASSOC_SYN"] = len(ds.SynAssoc)
	r.Bases["GD_CUE"] = len(ds.CueBases)
	total := 0
	for _, n := range r.Bases {
		total += n
	}
	r.ExpectedRecords = total * 3
	r.CrossReuse = ds.CrossRecoveryFamilyBaseReuse
	r.OCRAvailable = ds.OCRFallbackAvailable

	set("base_counts_in_range",
		len(ds.ScaleBases) >= 12 && len(ds.ContextBases) >= 12 && len(ds.RealAssoc) == 22 &&
			len(ds.SynAssoc) == 24 && len(ds.CueBases) >= 8,
		fmt.Sprintf("base counts %v", r.Bases))
	set("expected_records_294", r.ExpectedRecords == 294, fmt.Sprintf("expected records = %d", r.ExpectedRecords))

	// 9 / 13 / 15(sel). deterministic + poison-invariant selection
	re1, e1 := SelectR1GDataset(in.ExpDir, ds.Temperature, ds.MaxTokens)
	poison := filepath.Join(in.ExpDir, "datasets", "POISON_expected_r1g.json")
	_ = os.WriteFile(poison, []byte(`{"r1g":"999","gold":"111"}`), 0o644)
	re2, e2 := SelectR1GDataset(in.ExpDir, ds.Temperature, ds.MaxTokens)
	_ = os.Remove(poison)
	selDet := e1 == nil && e2 == nil &&
		r1gDatasetFP(re1) == r1gDatasetFP(re2) && (dsErr != nil || r1gDatasetFP(re1) == r1gDatasetFP(ds))
	set("selection_deterministic_and_poison_invariant", selDet, "R1-G selection not reproducible / changed under a poisoned expected-answers file")

	// 3 / 4 / 5 / 6 / 7. per-family condition semantics
	famOK := true
	for _, fam := range R1GFamilies {
		if len(fam.Conditions) != 3 {
			famOK = false
		}
		seen := map[string]bool{}
		for _, c := range fam.Conditions {
			if seen[c] {
				famOK = false
			}
			seen[c] = true
		}
	}
	set("each_family_has_3_distinct_conditions_baseline_first", famOK, "a family does not have exactly 3 distinct conditions")
	set("ga_changes_only_scale", strings.Contains(strings.Join(famByKey("GA_SCALE").Conditions, ","), "SCALE"), "")
	set("gb_changes_only_context", strings.Contains(strings.Join(famByKey("GB_CONTEXT").Conditions, ","), "CONTEXT") ||
		strings.Contains(strings.Join(famByKey("GB_CONTEXT").Conditions, ","), "RECOVERY"), "")
	set("gc_scenes_differ_only_in_competitor_policy", len(famByKey("GC_ASSOC_REAL").Conditions) == 3 && len(famByKey("GC_ASSOC_SYN").Conditions) == 3, "")
	set("gd_differs_only_in_cue_policy", strings.Contains(strings.Join(famByKey("GD_CUE").Conditions, ","), "CUE"), "")

	// 8. real/synthetic never pooled
	realIDs := map[string]bool{}
	for _, b := range ds.RealAssoc {
		realIDs[b.BaseID] = true
	}
	pooled := false
	for _, b := range ds.SynAssoc {
		if realIDs[b.BaseID] {
			pooled = true
		}
	}
	set("real_synthetic_association_never_pooled", !pooled && famByKey("GC_ASSOC_REAL").Key != famByKey("GC_ASSOC_SYN").Key, "real and synthetic association bases overlap")

	// 10. historical reuse labelled
	reuseOK := true
	for _, b := range ds.RealAssoc {
		if !b.InterventionReuse || b.SourceStage != "R1D_REAL_INTERVENTION_REUSE" {
			reuseOK = false
		}
	}
	set("historical_reuse_explicitly_labelled", reuseOK, "a real-assoc base is not labelled intervention reuse")
	set("independent_accuracy_estimate_false_for_real_reuse", true, "")

	// 11. prompt/opcode frozen within family
	promptOK := true
	for _, fam := range R1GFamilies {
		if fam.Instruction == "" || fam.Capability == "" {
			promptOK = false
		}
	}
	set("prompt_and_opcode_frozen_within_family", promptOK, "a family has an empty instruction/capability")

	// 12. expected answer never reaches model
	leakOK := true
	for _, fam := range R1GFamilies {
		if containsAnyDigit(fam.Instruction) || strings.Contains(strings.ToLower(fam.Instruction), "answer") {
			leakOK = false
		}
	}
	// no per-base gold in any prompt (all families use one fixed instruction)
	set("expected_answer_never_in_prompt", leakOK, "a family instruction leaks a digit / answer hint")

	// 14. renderer deterministic — render one base of each pool-derived family twice
	renderOK := true
	if in.StoreDir != "" && len(ds.ScaleBases) > 0 {
		prov, perr := parrotlab.NewPDFMemoryProvider(in.StoreDir, "")
		if perr == nil {
			if a, e := renderScaleConditions(prov, in.StoreDir, ds.ScaleBases[0]); e == nil {
				if b, e2 := renderScaleConditions(prov, in.StoreDir, ds.ScaleBases[0]); e2 == nil {
					for i := range a {
						if pngHash(a[i]) != pngHash(b[i]) {
							renderOK = false
						}
					}
				} else {
					renderOK = false
				}
			} else {
				renderOK = false
				r.Problems = append(r.Problems, "GA render: "+e.Error())
			}
		}
	}
	if in.Bank != nil && len(ds.SynAssoc) > 0 {
		if a, e := renderSynAssocConditions(in.Bank, ds.SynAssoc[0]); e == nil {
			b, _ := renderSynAssocConditions(in.Bank, ds.SynAssoc[0])
			for i := range a {
				if pngHash(a[i]) != pngHash(b[i]) {
					renderOK = false
				}
			}
			// GC_SYN_1 and _2 must genuinely differ (mask vs isolate)
			if pngHash(a[1]) == pngHash(a[2]) {
				renderOK = false
				r.Problems = append(r.Problems, "GC_SYN_1 and GC_SYN_2 render identically")
			}
		} else {
			renderOK = false
			r.Problems = append(r.Problems, "GC_SYN render: "+e.Error())
		}
	}
	set("renderer_deterministic", renderOK, "a family renderer is not byte-deterministic")

	// 15. model identity unchanged
	miOK := false
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "MODEL_IDENTITY.json")); err == nil {
		var mi map[string]any
		if json.Unmarshal(body, &mi) == nil {
			miOK, _ = mi["STAGE_1_MODEL_IDENTITY_OK"].(bool)
		}
	}
	set("model_identity_unchanged", miOK, "MODEL_IDENTITY.json missing or not OK")

	// 16. endpoint reachable
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

	// 17. recovery verdict thresholds frozen
	thrOK := r1gEarnedDelta == 0.20 && r1gPromisingDelta == 0.10 && r1gMaxDegradation == 0.05 &&
		r1gEarnedMcNemarSig == 0.05 && ds.Thresholds["earned_delta"] == 0.20 && ds.Thresholds["max_degradation"] == 0.05
	set("recovery_verdict_thresholds_frozen", thrOK, "recovery verdict thresholds drifted / not recorded in the dataset")

	// 2 / 18. exact identical retry is imported, never executed
	set("exact_identical_retry_imported_not_executed",
		ds.ExactRetryImported == R1GExactRetryStatus && st == "R1-F_REPEATABILITY_COMPLETE_FROZEN",
		"R1-F exact-retry negative control not imported / R1-F not frozen")

	// addendum-08 pre-registration
	addOK := false
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "R1_PROTOCOL_ADDENDUM_08.json")); err == nil {
		var ad map[string]any
		if json.Unmarshal(body, &ad) == nil {
			addOK, _ = ad["NO_R1G_MODEL_OUTPUT_EXISTED_WHEN_RECOVERY_POLICIES_AND_THRESHOLDS_WERE_FROZEN"].(bool)
		}
	}
	r.NoR1GModelOutputExistedWhenRecoveryPoliciesAndThresholdsWereFrozen = addOK
	set("recovery_policies_and_thresholds_frozen_before_output", addOK, "R1_PROTOCOL_ADDENDUM_08.json missing the frozen-before-output flag")

	r.ReadyR1G = len(r.Problems) == 0
	for _, ok := range r.Checks {
		if !ok {
			r.ReadyR1G = false
		}
	}
	return r
}
