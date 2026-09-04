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
)

// R1FDoctorReport is the R1-F pre-inference readiness gate (protocol §17).
type R1FDoctorReport struct {
	Schema          string          `json:"schema"`
	ExperimentID    string          `json:"experiment_id"`
	Checks          map[string]bool `json:"checks"`
	Problems        []string        `json:"problems"`
	Sentinels       int             `json:"sentinels"`
	PerStratum      map[string]int  `json:"per_stratum"`
	RepeatsPer      int             `json:"repeats_per_sentinel"`
	ExpectedRecords int             `json:"expected_records"`
	SamplingSeed    string          `json:"sampling_seed_status"`
	NoR1FRepeatOutputExistedWhenSentinelsAndDecisionRuleWereFrozen bool `json:"NO_R1F_REPEAT_OUTPUT_EXISTED_WHEN_SENTINELS_AND_DECISION_RULE_WERE_FROZEN"`
	ReadyR1F        bool            `json:"READY_R1F"`
}

// DoctorR1FInput carries what the R1-F doctor needs.
type DoctorR1FInput struct {
	ExpDir   string
	Endpoint string
	Model    string
}

const r1fDoctorSchema = "tlaloc.parrot-perceptual-envelope-r1.r1f-doctor.r1"

// DoctorR1F runs every §17 integrity check.
func DoctorR1F(ctx context.Context, in DoctorR1FInput) R1FDoctorReport {
	r := R1FDoctorReport{Schema: r1fDoctorSchema, ExperimentID: ExperimentID, Checks: map[string]bool{}, PerStratum: map[string]int{}}
	set := func(name string, ok bool, msg string) {
		r.Checks[name] = ok
		if !ok && msg != "" {
			r.Problems = append(r.Problems, msg)
		}
	}

	// 1. R1-E frozen and pushed
	st, commit := "", ""
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "R1E_CHECKPOINT.json")); err == nil {
		var ck map[string]any
		if json.Unmarshal(body, &ck) == nil {
			st, _ = ck["status"].(string)
			commit, _ = ck["tlaloc_commit"].(string)
		}
	}
	set("r1e_frozen_committed", st == "R1-E_VISUAL_DEPENDENCE_COMPLETE_FROZEN" && commit != "" && commit != "unknown",
		fmt.Sprintf("R1E_CHECKPOINT status=%q commit=%q", st, commit))

	// dataset on disk
	var ds R1FDataset
	dsBody, dsErr := os.ReadFile(filepath.Join(in.ExpDir, "datasets", "R1F_DATASET.json"))
	if dsErr == nil {
		_ = json.Unmarshal(dsBody, &ds)
	}
	r.Sentinels = len(ds.Sentinels)
	r.RepeatsPer = ds.RepeatsPerSentinel
	r.ExpectedRecords = len(ds.Sentinels) * ds.RepeatsPerSentinel
	r.SamplingSeed = ds.SamplingSeed
	for _, s := range ds.Sentinels {
		r.PerStratum[s.Stratum]++
	}

	set("r1f_dataset_written", dsErr == nil && len(ds.Sentinels) > 0, "R1F_DATASET.json missing/empty")

	// 15. sentinel selection deterministic — reselect and compare ids/keys
	reSel, selErr := SelectR1FSentinels(in.ExpDir)
	selDet := selErr == nil && len(reSel) == len(ds.Sentinels)
	if selDet {
		for i := range reSel {
			if reSel[i].SentinelID != ds.Sentinels[i].SentinelID ||
				reSel[i].BaseID != ds.Sentinels[i].BaseID ||
				reSel[i].RankKey != ds.Sentinels[i].RankKey {
				selDet = false
			}
		}
	}
	set("sentinel_selection_deterministic", selDet, "re-selection does not reproduce the frozen sentinel set")

	// 2. exactly 20 sentinels
	set("exactly_20_sentinels", len(ds.Sentinels) == len(R1FStrata)*r1fSentinelsPerStratum,
		fmt.Sprintf("have %d sentinels, want %d", len(ds.Sentinels), len(R1FStrata)*r1fSentinelsPerStratum))

	// 3. exactly 4 per stratum
	perStratumOK := true
	for _, strat := range R1FStrata {
		if r.PerStratum[strat.Key] != r1fSentinelsPerStratum {
			perStratumOK = false
		}
	}
	set("exactly_4_per_stratum", perStratumOK, fmt.Sprintf("per-stratum counts %v", r.PerStratum))

	// 4 / 5. five repeats + expected total 100
	set("five_repeats_per_sentinel", ds.RepeatsPerSentinel == r1fRepeatsPerSentinel && len(ds.RepeatIDs) == r1fRepeatsPerSentinel, "repeats_per_sentinel != 5")
	set("expected_total_records_100", r.ExpectedRecords == 100, fmt.Sprintf("expected records = %d", r.ExpectedRecords))

	// 6 / 7 / 8. image + prompt + settings constant across repeats (frozen once
	// per sentinel; RunR1F re-verifies the image sha and reuses bytes) — here
	// verify each frozen image exists and hashes to the recorded sha, and each
	// prompt sha matches the instruction.
	imgOK, promptOK := true, true
	for _, s := range ds.Sentinels {
		if s.HasImage {
			b, err := os.ReadFile(s.ImagePath)
			if err != nil || sha256Hex(b) != s.ImageSHA256 {
				imgOK = false
				r.Problems = append(r.Problems, s.SentinelID+": frozen image missing / sha mismatch")
			}
		} else if s.ImageSHA256 != "NO_IMAGE" {
			imgOK = false
		}
		if sha256Hex([]byte(s.Instruction)) != s.PromptSHA256 {
			promptOK = false
		}
	}
	set("frozen_image_sha_constant_and_valid", imgOK, "a frozen image is missing or does not match its recorded sha")
	set("prompt_sha_constant_and_valid", promptOK, "a sentinel prompt sha does not match its instruction")
	set("effective_request_settings_constant", ds.Temperature == 0 && ds.MaxTokens > 0, "temperature != 0 or max_tokens <= 0")

	// 9. no scorer / expected data enters the prompt
	leakOK := true
	for _, s := range ds.Sentinels {
		low := strings.ToLower(s.Instruction)
		if containsAnyDigit(s.Instruction) || strings.Contains(low, "answer") || strings.Contains(low, "gold") || strings.Contains(low, s.Gold) {
			leakOK = false
		}
	}
	set("no_scorer_or_expected_data_in_prompt", leakOK, "an instruction leaks a digit / gold / answer hint")

	// 10. frozen historical SOURCE_REFERENCE not mutated — the source result
	// files still exist and each sentinel's previous_raw_output is still present
	// in its source file for that base/condition.
	sourceOK := true
	var eb, ab, db []byte
	eb, _ = os.ReadFile(filepath.Join(in.ExpDir, "results", "R1B_RECORDS.json"))
	ab, _ = os.ReadFile(filepath.Join(in.ExpDir, "results", "R1A1_RECORDS.json"))
	db, _ = os.ReadFile(filepath.Join(in.ExpDir, "results", "R1D_DISTRACTOR_RECORDS.json"))
	e0b, _ := os.ReadFile(filepath.Join(in.ExpDir, "results", "R1E_RECORDS.json"))
	if len(eb) == 0 || len(ab) == 0 || len(db) == 0 || len(e0b) == 0 {
		sourceOK = false
	} else {
		var extract, a1c6 []r1fExtractRec
		var dist []r1fDistRec
		var e0 []R1ERecord
		_ = json.Unmarshal(eb, &extract)
		_ = json.Unmarshal(ab, &a1c6)
		_ = json.Unmarshal(db, &dist)
		_ = json.Unmarshal(e0b, &e0)
		findExtract := func(pool []r1fExtractRec, base, cond string) (r1fExtractRec, bool) {
			for _, r := range pool {
				if r.BaseID == base && r.ContextLevel == cond {
					return r, true
				}
			}
			return r1fExtractRec{}, false
		}
		for _, s := range ds.Sentinels {
			switch s.Stratum {
			case "A", "B":
				if rec, ok := findExtract(extract, s.BaseID, s.SourceCondition); !ok || rec.RawText != s.PrevRawOutput {
					sourceOK = false
				}
			case "C":
				if rec, ok := findExtract(a1c6, s.BaseID, s.SourceCondition); !ok || rec.RawText != s.PrevRawOutput {
					sourceOK = false
				}
			case "D":
				hit := false
				for _, r := range dist {
					if r.BaseID == s.BaseID && r.Condition == s.SourceCondition && r.RawText == s.PrevRawOutput {
						hit = true
					}
				}
				if !hit {
					sourceOK = false
				}
			case "E":
				hit := false
				for _, r := range e0 {
					if r.BaseID == s.BaseID && r.Condition == "E0_NO_IMAGE" && r.Capability == "READ_ASSOCIATED_NUMBER" && r.RawText == s.PrevRawOutput {
						hit = true
					}
				}
				if !hit {
					sourceOK = false
				}
			}
		}
	}
	set("frozen_source_reference_not_mutated", sourceOK, "a sentinel's previous_raw_output no longer matches its frozen source record")

	// 11. model identity unchanged
	miOK := false
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "MODEL_IDENTITY.json")); err == nil {
		var mi map[string]any
		if json.Unmarshal(body, &mi) == nil {
			miOK, _ = mi["STAGE_1_MODEL_IDENTITY_OK"].(bool)
		}
	}
	set("model_identity_unchanged", miOK, "MODEL_IDENTITY.json missing or not OK")

	// 12. endpoint reachable
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

	// 13. blind-retry decision rule frozen
	ruleOK := strings.Contains(ds.DecisionRule, "BLIND_RETRY_NOT_USEFUL") &&
		strings.Contains(ds.DecisionRule, "not redefinable")
	set("blind_retry_decision_rule_frozen", ruleOK, "R1F_DATASET.json decision rule not frozen / not marked non-redefinable")

	// 14. no new rendering during repetition — enforced structurally (RunR1F
	// loads frozen bytes and never calls a renderer); assert the frozen image
	// dir exists and holds exactly the image-bearing sentinels.
	imgDir := filepath.Join(in.ExpDir, "datasets", "R1F_images")
	wantImgs := 0
	for _, s := range ds.Sentinels {
		if s.HasImage {
			wantImgs++
		}
	}
	haveImgs := 0
	if entries, err := os.ReadDir(imgDir); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".png") {
				haveImgs++
			}
		}
	}
	set("no_new_rendering_frozen_images_present", wantImgs == haveImgs && wantImgs > 0,
		fmt.Sprintf("frozen image dir has %d pngs, want %d", haveImgs, wantImgs))

	// addendum-07 pre-registration
	addOK := false
	if body, err := os.ReadFile(filepath.Join(in.ExpDir, "R1_PROTOCOL_ADDENDUM_07.json")); err == nil {
		var ad map[string]any
		if json.Unmarshal(body, &ad) == nil {
			addOK, _ = ad["NO_R1F_REPEAT_OUTPUT_EXISTED_WHEN_SENTINELS_AND_DECISION_RULE_WERE_FROZEN"].(bool)
		}
	}
	r.NoR1FRepeatOutputExistedWhenSentinelsAndDecisionRuleWereFrozen = addOK
	set("repeats_and_decision_rule_frozen_before_any_output", addOK, "R1_PROTOCOL_ADDENDUM_07.json missing the frozen-before-output flag")

	r.ReadyR1F = len(r.Problems) == 0
	for _, ok := range r.Checks {
		if !ok {
			r.ReadyR1F = false
		}
	}
	return r
}
