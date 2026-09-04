// Command tlaloc-tonalt1-singledigit-qual runs the isolated SINGLE_DIGIT
// capability qualification (TONAL T1 corpus-resolution, protocol sections
// 8-15).
//
// R1-C classified SINGLE_DIGIT as FRAGILE (real n=12, 0.92, CI 0.65-0.99).
// This re-measures a fresh, deterministic, physically held-out SINGLE_DIGIT
// sample (n>=60) under the SAME frozen presentation core / prompt / model /
// generation settings, and applies the SAME frozen R1-C verdict thresholds
// (perceptenvelope.verdictFor) to decide promotion.
//
// This is NOT T1 Arm A/B/C inference. It is an independent capability
// extension experiment. The dataset, renderer, prompt, scoring and
// promotion criteria are frozen BEFORE inference.
//
//	tlaloc-tonalt1-singledigit-qual \
//	  -root . -n 60 \
//	  -store experiments/exocortex-decomposition-r0/stores/fold-bench-reconstructed-r0 \
//	  -out experiments/tonal-t1/singledigit-qual \
//	  -endpoint http://127.0.0.1:1234 -model lfm2-vl-1.6b
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/perceptenvelope"
	"tlaloc.local/behaviorlab/internal/tonalt1"
)

// frozenCriteria is the predeclared promotion rule, identical to
// perceptenvelope.verdictFor (R1-C protocol section 21). Written to disk
// BEFORE the held-out dataset is enumerated and BEFORE any inference.
var frozenCriteria = map[string]any{
	"schema":       "tonal.t1.singledigit-qual.criteria.r1",
	"metric":       "R1-C value_correct (exact canonical numeric match), one model call per base",
	"presentation": "perceptenvelope.RenderR1BScale at R1CLineHeightPx=32, A1C0_TARGET context, 512 canvas, bilinear inverse map, magenta token cue — identical to the R1-C real strata",
	"prompt":       perceptenvelope.FrozenInstruction,
	"generation":   map[string]any{"temperature": 0.0, "max_output_tokens": 32},
	"verdict_thresholds": map[string]any{
		"RELIABLE":                "accuracy >= 0.95 AND ci95_low >= 0.80 AND surface_accuracy >= 0.90",
		"USABLE_WITH_CONSTRAINTS": "accuracy >= 0.90 AND ci95_low >= 0.70",
		"FRAGILE":                 "accuracy >= 0.60",
		"DO_NOT_DEPLOY":           "accuracy < 0.60",
		"min_real_n":              6,
	},
	"promotion_requires": "verdict in {USABLE_WITH_CONSTRAINTS, RELIABLE}",
	"r1c_prior_verdict":  "FRAGILE (real n=12, value 0.92, CI 0.65-0.99)",
	"held_out_rule":      "every SINGLE_DIGIT physical instance previously selected or inferred (incl. the 12 R1-C SINGLE_DIGIT bases) is excluded by the D3 v2 prior-use union",
	"no_leakage_rule":    "no T1 workflow result, no arithmetic-usefulness selection, no manual example choice, no failure retry, no post-hoc dataset edit",
}

func main() {
	root := flagString("-root", ".")
	store := flagString("-store", "experiments/exocortex-decomposition-r0/stores/fold-bench-reconstructed-r0")
	out := flagString("-out", "experiments/tonal-t1/singledigit-qual")
	endpoint := flagString("-endpoint", "http://127.0.0.1:1234")
	model := flagString("-model", "lfm2-vl-1.6b")
	n := flagInt("-n", 60)

	outDir := filepath.Join(*root, *out)
	storeDir := *store
	if !strings.HasPrefix(storeDir, "/") {
		storeDir = filepath.Join(*root, storeDir)
	}
	if err := os.MkdirAll(filepath.Join(outDir, "run"), 0o755); err != nil {
		fail(err)
	}

	// 1. Freeze the promotion criteria FIRST.
	writeJSON(filepath.Join(outDir, "SINGLE_DIGIT_QUAL_CRITERIA.json"), frozenCriteria)

	// 2. Enumerate + freeze the held-out dataset.
	held, err := tonalt1.EnumerateSingleDigitHeldOut(*root, storeDir, *n)
	if err != nil {
		fail(err)
	}
	writeJSON(filepath.Join(outDir, "SINGLE_DIGIT_HELDOUT.json"), held)
	if !held.HeldOutExclusionOK {
		fail(fmt.Errorf("held-out exclusion FAILED: a selected base collides with a prior-used instance"))
	}
	if len(held.Bases) < 6 {
		fail(fmt.Errorf("insufficient held-out SINGLE_DIGIT instances: %d (< min real n 6)", len(held.Bases)))
	}

	// 3. Model / endpoint identity guard.
	identity := loadModelIdentity(*root)
	if err := guardEndpoint(*endpoint, *model); err != nil {
		fail(fmt.Errorf("endpoint identity guard: %w", err))
	}

	// 4. Build the R1-C allocation and run inference.
	alloc := perceptenvelope.R1CAllocation{
		Schema:       "tlaloc.parrot-perceptual-envelope-r1.r1c-allocation.r1",
		ExperimentID: held.ExperimentID,
		Seed:         held.Seed,
		RankRule:     "sha256(seed || candidate_id) ascending",
		LineHeightPx: perceptenvelope.R1CLineHeightPx,
		ContextLevel: perceptenvelope.R1CContextLevel,
		CanvasPx:     512,
		Families: []perceptenvelope.R1CFamilyAllocation{{
			Family:        perceptenvelope.FamSingleDigit,
			Stratum:       perceptenvelope.StratumLexical,
			RealAvailable: held.AvailableN,
			Band:          "SINGLE_DIGIT_QUALIFICATION_FRESH_HELDOUT",
			RealBases:     toR1CBases(held.Bases),
		}},
	}
	writeJSON(filepath.Join(outDir, "SINGLE_DIGIT_ALLOCATION.json"), alloc)

	cfg := perceptenvelope.RunConfig{
		StoreDir: storeDir, Endpoint: *endpoint, Model: *model,
		Temperature: 0.0, MaxTokens: 32,
		RunDir: filepath.Join(outDir, "run"),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	records, err := perceptenvelope.RunR1C(ctx, cfg, alloc, nil)
	if err != nil {
		fail(fmt.Errorf("run: %w", err))
	}
	writeJSON(filepath.Join(outDir, "SINGLE_DIGIT_RECORDS.json"), records)

	// 5. Aggregate with the frozen R1-C thresholds.
	table := perceptenvelope.AggregateR1C(records)
	var verdict perceptenvelope.R1CFamilyVerdict
	for _, v := range table.Verdicts {
		if v.Family == perceptenvelope.FamSingleDigit {
			verdict = v
		}
	}
	var row perceptenvelope.R1CRow
	for _, r := range table.Rows {
		if r.Family == perceptenvelope.FamSingleDigit && r.Provenance == "REAL_DOCUMENT" {
			row = r
		}
	}

	promoted := verdict.Verdict == "USABLE_WITH_CONSTRAINTS" || verdict.Verdict == "RELIABLE"

	result := map[string]any{
		"schema":                "tonal.t1.singledigit-qual.result.r1",
		"experiment_id":         held.ExperimentID,
		"model":                 *model,
		"model_weights_sha256":  identity,
		"endpoint":              *endpoint,
		"held_out_n":            len(held.Bases),
		"per_digit_selected":    held.PerDigitSelected,
		"held_out_exclusion_ok": held.HeldOutExclusionOK,
		"model_calls":           len(records),
		"errors":                table.Errors,
		"value_accuracy":        row.Value.Accuracy,
		"value_ci95_low":        row.Value.CI95Low,
		"value_ci95_high":       row.Value.CI95High,
		"surface_accuracy":      row.Surface.Accuracy,
		"contract_success":      row.ContractSuccess,
		"abstained":             row.Abstained,
		"format_failure":        row.FormatFailure,
		"mean_latency_ms":       row.MeanLatencyMS,
		"failure_classes":       row.FailureClasses,
		"r1c_prior_verdict":     "FRAGILE (n=12, 0.92, CI 0.65-0.99)",
		"qualification_verdict": verdict.Verdict,
		"verdict_basis":         verdict.Basis,
		"SINGLE_DIGIT_PROMOTED": promoted,
	}
	writeJSON(filepath.Join(outDir, "SINGLE_DIGIT_QUALIFICATION_RESULT.json"), result)
	writeJSON(filepath.Join(outDir, "SINGLE_DIGIT_MORPHOLOGY_TABLE.json"), table)

	if promoted {
		ext := map[string]any{
			"schema":                   "tlaloc.capability-profile.r1-ext.singledigit.r1",
			"profile_id":               "parrot-lfm2-vl-1.6b@r1-ext-singledigit",
			"predecessor_profile":      "parrot-lfm2-vl-1.6b@r1",
			"predecessor_profile_hash": "8acc959ba72334e64704c9f5b114bfb5230ca1f58375421c17a956e26b9ba729",
			"model_weights_sha256":     identity,
			"promoted_family":          "SINGLE_DIGIT",
			"promoted_presentation":    "isolated real-document numeric line, magenta token cue, 32 px line height, 512 canvas, bilinear inverse map (RenderR1BScale) — identical to R1-C real strata",
			"new_verdict":              verdict.Verdict,
			"new_evidence": map[string]any{
				"experiment":       held.ExperimentID,
				"real_n":           len(held.Bases),
				"value_accuracy":   row.Value.Accuracy,
				"value_ci95_low":   row.Value.CI95Low,
				"value_ci95_high":  row.Value.CI95High,
				"surface_accuracy": row.Surface.Accuracy,
				"dataset":          "SINGLE_DIGIT_HELDOUT.json",
				"records":          "SINGLE_DIGIT_RECORDS.json",
				"held_out":         "instance-level held-out from R1-A..R1-G, Profile-H, T0; incl. the 12 R1-C SINGLE_DIGIT bases",
			},
			"constraints": []string{
				"single isolated digit token, one numeric token per containing line",
				"same presentation core as R1-C; no distractors; no low-scale regime",
				"instance-level held-out only; NO cross-document generalization",
			},
			"does_not_rewrite": "R1 / R1-C history is immutable; this is an additive extension profile",
		}
		writeJSON(filepath.Join(outDir, "CAPABILITY_PROFILE_R1_EXT.json"), ext)
	}

	report := map[string]any{
		"SINGLE_DIGIT_PROMOTED": promoted,
		"qualification_verdict": verdict.Verdict,
		"held_out_n":            len(held.Bases),
		"per_digit_available":   held.PerDigitAvailable,
		"value_accuracy":        row.Value.Accuracy,
		"value_ci95":            []float64{row.Value.CI95Low, row.Value.CI95High},
		"surface_accuracy":      row.Surface.Accuracy,
		"model_calls":           len(records),
		"errors":                table.Errors,
		"out_dir":               outDir,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(report)
}

func toR1CBases(bases []tonalt1.SingleDigitBase) []perceptenvelope.R1CBase {
	out := make([]perceptenvelope.R1CBase, 0, len(bases))
	for _, b := range bases {
		cc := b.Candidate
		start := 0
		if idx := strings.Index(cc.LineText, cc.Token); idx >= 0 {
			start = len([]rune(cc.LineText[:idx]))
		}
		out = append(out, perceptenvelope.R1CBase{
			BaseID:          b.BaseID,
			Family:          perceptenvelope.FamSingleDigit,
			Provenance:      perceptenvelope.ProvReal,
			Stratum:         perceptenvelope.StratumLexical,
			GoldSurface:     b.Digit,
			RankKey:         b.RankKey,
			Candidate:       &cc,
			CharOffsetStart: start,
			CharOffsetEnd:   start + len([]rune(cc.Token)),
		})
	}
	return out
}

func loadModelIdentity(root string) string {
	body, err := os.ReadFile(filepath.Join(root, "experiments/parrot-perceptual-envelope-r1/MODEL_IDENTITY.json"))
	if err != nil {
		return ""
	}
	var doc struct {
		Model struct {
			WeightsGGUF struct {
				SHA256 string `json:"sha256"`
			} `json:"weights_gguf"`
		} `json:"model"`
	}
	_ = json.Unmarshal(body, &doc)
	return doc.Model.WeightsGGUF.SHA256
}

func guardEndpoint(endpoint, model string) error {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(strings.TrimRight(endpoint, "/") + "/v1/models")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return err
	}
	for _, m := range list.Data {
		if m.ID == model {
			return nil
		}
	}
	return fmt.Errorf("model %q not served by %s (LM Studio JIT should load it on first request; ensure it is available)", model, endpoint)
}

func writeJSON(path string, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		fail(err)
	}
}

func flagString(name, def string) *string {
	value := def
	for i, arg := range os.Args {
		if arg == name && i+1 < len(os.Args) {
			value = os.Args[i+1]
		}
	}
	return &value
}

func flagInt(name string, def int) *int {
	value := def
	for i, arg := range os.Args {
		if arg == name && i+1 < len(os.Args) {
			fmt.Sscanf(os.Args[i+1], "%d", &value)
		}
	}
	return &value
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "tlaloc-tonalt1-singledigit-qual:", err)
	os.Exit(1)
}
