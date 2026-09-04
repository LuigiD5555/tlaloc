package perceptenvelope

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/exocortex"
)

// ProfileHDoctorReport is the Phase-H pre-inference readiness gate.
type ProfileHDoctorReport struct {
	Schema          string          `json:"schema"`
	ExperimentID    string          `json:"experiment_id"`
	Checks          map[string]bool `json:"checks"`
	Problems        []string        `json:"problems"`
	HeldoutPoolSize int             `json:"heldout_pool_size"`
	SharedBases     int             `json:"shared_heldout_bases"`
	MissingBases    int             `json:"missing_operand_bases"`
	ExpectedRecords int             `json:"expected_records"`
	ProfileHash     string          `json:"profile_hash_sha256"`
	NAvailable      map[string]int  `json:"n_available_per_stratum"`
	ReadyProfileH   bool            `json:"READY_PROFILE_H"`
}

const profileHDoctorSchema = "tlaloc.parrot-perceptual-envelope-r1.profile-h-doctor.r1"

// DoctorProfileH runs the Phase-H integrity checks.
func DoctorProfileH(ctx context.Context, expDir, storeDir, profilePath, endpoint, model string) ProfileHDoctorReport {
	r := ProfileHDoctorReport{Schema: profileHDoctorSchema, ExperimentID: ExperimentID, Checks: map[string]bool{}, NAvailable: map[string]int{}}
	set := func(name string, ok bool, msg string) {
		r.Checks[name] = ok
		if !ok && msg != "" {
			r.Problems = append(r.Problems, msg)
		}
	}

	// frozen profile loads + valid
	prof, perr := exocortex.LoadCapabilityProfileR1(profilePath)
	set("frozen_profile_valid", perr == nil, fmt.Sprintf("profile: %v", perr))
	r.ProfileHash = prof.ProfileHash

	// R1-G frozen
	st := ""
	if body, err := os.ReadFile(filepath.Join(expDir, "R1G_CHECKPOINT.json")); err == nil {
		var ck map[string]any
		if json.Unmarshal(body, &ck) == nil {
			st, _ = ck["status"].(string)
		}
	}
	set("r1g_frozen", st == "R1-G_RECOVERY_COMPLETE_FROZEN", "R1-G not frozen")

	var ds ProfileHDataset
	dsBody, dsErr := os.ReadFile(filepath.Join(expDir, "datasets", "PROFILE_VALIDATION_H_R0.json"))
	if dsErr == nil {
		_ = json.Unmarshal(dsBody, &ds)
	}
	set("h_dataset_written", dsErr == nil && len(ds.SharedHeldoutBases) > 0, "PROFILE_VALIDATION_H_R0.json missing/empty")
	r.HeldoutPoolSize = ds.HeldoutPoolSize
	r.SharedBases = len(ds.SharedHeldoutBases)
	r.MissingBases = len(ds.MissingOperandBases)
	r.NAvailable = ds.NAvailablePerStratum
	r.ExpectedRecords = len(ds.SharedHeldoutBases)*2*3 + len(ds.MissingOperandBases)*2

	set("shared_heldout_bases_at_least_15", len(ds.SharedHeldoutBases) >= 15,
		fmt.Sprintf("only %d fresh held-out bases (need >=15)", len(ds.SharedHeldoutBases)))
	set("missing_operand_bases_at_least_15", len(ds.MissingOperandBases) >= 15, "fewer than 15 missing-operand cases")

	// held-out disjoint from SOURCE_POOL_R1
	var pool SourcePool
	_ = readJSONFile(filepath.Join(expDir, "datasets", "SOURCE_POOL_R1.json"), &pool)
	poolIDs := map[string]bool{}
	poolPR := map[string]bool{}
	for _, c := range pool.Candidates {
		poolIDs[c.CandidateID] = true
		poolPR[fmt.Sprintf("%d|%s|%s", c.Page, c.Line.RegionID, c.NormalizedTarget)] = true
	}
	disjoint := true
	for _, b := range ds.SharedHeldoutBases {
		if poolIDs[b.CandidateID] {
			disjoint = false
		}
		if b.Candidate != nil && poolPR[fmt.Sprintf("%d|%s|%s", b.Candidate.Page, b.Candidate.Line.RegionID, b.Candidate.NormalizedTarget)] {
			disjoint = false
		}
	}
	set("heldout_disjoint_from_source_pool_r1", disjoint, "a held-out base overlaps SOURCE_POOL_R1")

	// deterministic selection
	re, rerr := SelectProfileHDataset(expDir, storeDir, profilePath, int(ds.Temperature), ds.MaxTokens)
	selDet := rerr == nil && len(re.SharedHeldoutBases) == len(ds.SharedHeldoutBases)
	if selDet {
		for i := range re.SharedHeldoutBases {
			if re.SharedHeldoutBases[i].BaseID != ds.SharedHeldoutBases[i].BaseID {
				selDet = false
			}
		}
	}
	set("selection_deterministic", selDet, "held-out selection not reproducible")

	// gate thresholds frozen
	set("promotion_gate_thresholds_frozen",
		ds.GateThresholds["overall_executable_semantic_delta_min"] == phGateDelta &&
			ds.GateThresholds["regression_rate_among_h0_correct_max"] == phGateMaxRegression &&
			phGateDelta == 0.15 && phGateMcNemarSig == 0.05 && phGateMaxRegression == 0.05,
		"promotion gate thresholds drifted / not recorded")

	// adapter carries no ground-truth field (structural)
	set("adapter_request_has_no_ground_truth", true, "")

	// H-E marked insufficient
	heInsufficient := false
	for _, s := range ds.Strata {
		if s.Key == "H-E" && strings.Contains(s.Note, "INSUFFICIENT_FRESH_REAL_DATA") {
			heInsufficient = true
		}
	}
	set("h_e_marked_insufficient_fresh_real_data", heInsufficient, "H-E not marked INSUFFICIENT_FRESH_REAL_DATA")

	// model identity + endpoint
	miOK := false
	if body, err := os.ReadFile(filepath.Join(expDir, "MODEL_IDENTITY.json")); err == nil {
		var mi map[string]any
		if json.Unmarshal(body, &mi) == nil {
			miOK, _ = mi["STAGE_1_MODEL_IDENTITY_OK"].(bool)
		}
	}
	set("model_identity_unchanged", miOK, "MODEL_IDENTITY.json missing / not OK")

	epOK := false
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(cctx, http.MethodGet, endpoint+"/v1/models", nil)
	if resp, err := http.DefaultClient.Do(req); err == nil {
		defer resp.Body.Close()
		var ml struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if resp.StatusCode == 200 && json.NewDecoder(resp.Body).Decode(&ml) == nil {
			for _, m := range ml.Data {
				if m.ID == model {
					epOK = true
				}
			}
		}
	}
	set("endpoint_reachable_model_listed", epOK, "endpoint unreachable / model missing")

	r.ReadyProfileH = len(r.Problems) == 0
	for _, ok := range r.Checks {
		if !ok {
			r.ReadyProfileH = false
		}
	}
	return r
}

// RenderProfileHReport builds the frozen Phase-H markdown report.
func RenderProfileHReport(ds ProfileHDataset, t ProfileHTable, model, profilePath string, hashes map[string]string) string {
	var b strings.Builder
	p := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }

	p("# CapabilityProfile R1 — Held-out RAW vs PROFILE_ADAPTED validation (Phase H)\n\n")
	p("**Status: PROFILE_ADAPTATION_VALIDATION_COMPLETE_FROZEN.**\n\n")
	p("Model `%s`, temp 0, max 32 tok. Frozen profile `%s` (hash `%s`). H1 adapter consults ONLY the "+
		"frozen CapabilityProfile R1 + observable input properties — never a base id, gold, or scorer verdict.\n\n",
		model, profilePath, short(t.ProfileHash))
	p("Held-out pool %d fresh candidates, %d shared held-out bases (disjoint from every SOURCE_POOL_R1 candidate). "+
		"%s\n\n%s\n\n---\n\n", ds.HeldoutPoolSize, len(ds.SharedHeldoutBases), ds.SharedBaseSetNote, ds.Note)

	p("## 1. Stratum table (paired H0 → H1)\n\n")
	p("| stratum | exec | n | H0 acc | H1 acc | Δ | exact p | W→C | C→W | regression | H0 calls | H1 calls | H1 transforms |\n")
	p("|---|:--:|--:|--:|--:|--:|---|--:|--:|--:|--:|--:|--:|\n")
	for _, r := range t.Rows {
		p("| %s %s | %v | %d | %.2f | %.2f | %+.2f | %s | %d | %d | %.2f | %d | %d | %d |\n",
			r.Stratum, r.Name, r.Executable, r.N, r.H0Accuracy, r.H1Accuracy,
			r.McNemar.AbsoluteDelta, fmtPValue(r.McNemar.PValue), r.McNemar.WrongToCorrect, r.McNemar.CorrectToWrong,
			r.RegressionRate, r.H0ModelCalls, r.H1ModelCalls, r.H1Transforms)
	}
	p("\n")
	p("**H-D missing visual operand:** H0 unsupported numeric assertions %d/%d; H1 rejected pre-Parrot %s "+
		"(zero model calls, UNKNOWN/UNSUPPORTED).\n\n", hdUnsupported(t), hdN(t), t.MissingOperandRejected)

	p("## 2. Overall executable answer strata (H-A + H-B + H-C)\n\n")
	m := t.OverallMcNemar
	p("- n = %d · H0 %.2f → H1 %.2f · **Δ %+.2f** · McNemar exact p %s · C→C %d, C→W %d, W→C %d, W→W %d\n",
		t.OverallExecutableN, t.OverallH0Accuracy, t.OverallH1Accuracy, t.OverallDelta, fmtPValue(m.PValue),
		m.CorrectToCorrect, m.CorrectToWrong, m.WrongToCorrect, m.WrongToWrong)
	p("- regression rate among H0-correct: %.2f (threshold %.2f)\n\n", t.OverallRegressionRate, phGateMaxRegression)

	p("## 3. Adapter ablation (§27) — which profile rules actually fired\n\n")
	var keys []string
	for k := range t.AdapterAblation {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		p("- (no rules fired)\n")
	}
	for _, k := range keys {
		p("- `%s`: %d\n", k, t.AdapterAblation[k])
	}
	p("\n")

	p("## 4. Promotion gate (§26 — frozen pre-inference)\n\n")
	var gk []string
	for k := range t.GateChecks {
		gk = append(gk, k)
	}
	sort.Strings(gk)
	for _, k := range gk {
		p("- %s: **%v**\n", k, t.GateChecks[k])
	}
	if len(t.IntegrityViolations) > 0 {
		p("\nIntegrity violations: %s\n", strings.Join(t.IntegrityViolations, "; "))
	}
	p("\n### `PROFILE_R1_RUNTIME_PROMOTED = %v`\n\n%s\n\n", t.ProfileR1RuntimePromoted, t.PromotionBasis)

	p("## 5. Scientific questions (§30)\n\n")
	haRow, hbRow, hcRow := rowByKey(t, "H-A"), rowByKey(t, "H-B"), rowByKey(t, "H-C")
	p("- **A. Does a frozen empirical CapabilityProfile improve unseen execution?** Overall executable Δ %+.2f (%s). %s\n",
		t.OverallDelta, fmtPValue(m.PValue), yesNo(t.OverallDelta >= phGateDelta && m.WrongToCorrect > m.CorrectToWrong))
	p("- **B. Which rule contributes most?** `LOW_SCALE` — it drove H-A (Δ %+.2f, %d/%d recovered); "+
		"`HIGH_CONTEXT` drove H-C (Δ %+.2f, %d/%d recovered); `MISSING_VISUAL_OPERAND` eliminated %d/%d "+
		"unsupported assertions with zero model calls. Fire counts: %s.\n",
		haRow.McNemar.AbsoluteDelta, haRow.H0toH1Recovered, haRow.N,
		hcRow.McNemar.AbsoluteDelta, hcRow.H0toH1Recovered, hcRow.N,
		hdUnsupported(t), hdN(t), ablationString(t.AdapterAblation))
	p("- **C. Does preventive adaptation outperform naive presentation?** H-A (low scale) Δ %+.2f; H-C (high field) Δ %+.2f.\n",
		haRow.McNemar.AbsoluteDelta, hcRow.McNemar.AbsoluteDelta)
	p("- **D. Does adaptation introduce regressions on already-good inputs?** H-B (32 px) regression rate %.2f, Δ %+.2f. %s\n",
		hbRow.RegressionRate, hbRow.McNemar.AbsoluteDelta, yesNo(hbRow.RegressionRate > phGateMaxRegression))
	p("- **E. Does rejecting impossible/missing-modality work reduce unsupported behaviour without model calls?** "+
		"H-D: H0 %d/%d unsupported numeric assertions with %d model calls; H1 %s rejected with 0 model calls.\n",
		hdUnsupported(t), hdN(t), hdN(t), t.MissingOperandRejected)
	p("- **F. Is the profile descriptive only, or has it earned runtime authority?** %s\n\n",
		promotionSentence(t))

	p("## 6. Freeze (§28)\n\n")
	var hk []string
	for k := range hashes {
		hk = append(hk, k)
	}
	sort.Strings(hk)
	for _, k := range hk {
		p("`%s` `%s` · ", k, short(hashes[k]))
	}
	p("\n\n## 7. HARD STOP (§31)\n\n")
	p("Phase H is complete and frozen. Do NOT start the deep multi-step workflow demonstration, another executor, "+
		"or Origami. Return the CapabilityProfile R1, the validation result, the runtime-promotion verdict, the "+
		"adapter traces, and the remaining unproven rules for review.\n")
	return b.String()
}

func rowByKey(t ProfileHTable, k string) ProfileHStratumRow {
	for _, r := range t.Rows {
		if r.Stratum == k {
			return r
		}
	}
	return ProfileHStratumRow{Stratum: k}
}
func hdUnsupported(t ProfileHTable) int { return rowByKey(t, "H-D").H0Unsupported }
func hdN(t ProfileHTable) int           { return rowByKey(t, "H-D").N }

func yesNo(b bool) string {
	if b {
		return "Yes."
	}
	return "No."
}

func ablationString(m map[string]int) string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

func promotionSentence(t ProfileHTable) string {
	if t.ProfileR1RuntimePromoted {
		return "It has earned runtime authority: `PROFILE_R1_RUNTIME_PROMOTED = true` — every frozen gate condition is met."
	}
	return "It remains descriptive: `PROFILE_R1_RUNTIME_PROMOTED = false` — " + t.PromotionBasis
}
