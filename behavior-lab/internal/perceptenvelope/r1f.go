package perceptenvelope

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/target"
)

// R1-F EXACT-INPUT REPEATABILITY / STABILITY.
//
// At temperature 0, when LFM2-VL receives byte-identical pixels + prompt +
// settings + runtime repeatedly, how stable is its output? R1-F samples
// five frozen behaviour strata (post-hoc, deliberately conditioned on the
// frozen prior results) and repeats each exact input five new times.
//
// SENTINEL_POSTHOC_SELECTION_FOR_STABILITY = true — R1-F accuracy is NOT
// an independent capability estimate; its endpoint is REPEATABILITY.

const (
	r1fSentinelsSchema = "tlaloc.parrot-perceptual-envelope-r1.r1f-sentinels.r1"
	r1fDatasetSchema   = "tlaloc.parrot-perceptual-envelope-r1.r1f-dataset.r1"
	r1fRepeatsPerSentinel = 5
	r1fSentinelsPerStratum = 4
)

// Frozen blind-retry decision-rule thresholds (protocol §13). These are
// fixed BEFORE any R1-F repeat output exists and must not be redefined.
const (
	r1fWrongStayWrongThreshold    = 0.90
	r1fSemanticInvariantThreshold = 0.90
)

// R1FStratum is one frozen behaviour stratum.
type R1FStratum struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	SourceStage string `json:"source_stage"`
	Purpose     string `json:"purpose"`
}

// R1FStrata is the frozen five-stratum plan.
var R1FStrata = []R1FStratum{
	{"A", "GOOD_OPERATING_REGION", "R1-B", "stability of successful atomic reading (32 px / B4, previously correct)"},
	{"B", "LOW_SCALE_FAILURE", "R1-B", "do low-scale misreads randomly recover on retry (8 px / B0, previously wrong)"},
	{"C", "HIGH_CONTEXT_FAILURE", "R1-A1", "do context-driven contract/verbosity failures vary (A1C6 full viewport, previously wrong)"},
	{"D", "DISTRACTOR_ASSOCIATION_FAILURE", "R1-D", "is distractor capture / hallucination stable (D1 K1/K2, previously wrong)"},
	{"E", "NO_IMAGE_DEGENERATE", "R1-E", "is the no-image fallback collapse itself stable (E0_NO_IMAGE, previous output \"12345\")"},
}

// R1FSentinel is one frozen exact-input sentinel.
type R1FSentinel struct {
	SentinelID          string   `json:"sentinel_id"`
	Stratum             string   `json:"stratum"`
	StratumName         string   `json:"stratum_name"`
	SourceStage         string   `json:"source_stage"`
	SourceCondition     string   `json:"source_condition"`
	BaseID              string   `json:"base_id"`
	Capability          string   `json:"capability"`
	Opcode              string   `json:"opcode"`
	Instruction         string   `json:"instruction"`
	PromptSHA256        string   `json:"prompt_sha256"`
	HasImage            bool     `json:"has_image"`
	ImagePath           string   `json:"image_path,omitempty"`
	ImageSHA256         string   `json:"image_sha256"` // "NO_IMAGE" when has_image=false
	Gold                string   `json:"gold"`
	VisibleNumbers      []string `json:"visible_numbers,omitempty"`
	Distractors         []string `json:"distractors,omitempty"`
	PrevRawOutput       string   `json:"previous_raw_output"`
	PrevSemanticCorrect bool     `json:"previous_semantic_correct"`
	PrevContractSuccess bool     `json:"previous_contract_success"`
	PrevFailureClass    string   `json:"previous_failure_class,omitempty"`
	SourceCropPath      string   `json:"source_crop_path,omitempty"`
	RankKey             string   `json:"rank_key"`
}

// R1FDataset is the frozen R1-F stimulus definition.
type R1FDataset struct {
	Schema                             string        `json:"schema"`
	ExperimentID                       string        `json:"experiment_id"`
	SentinelPosthocSelectionForStability bool        `json:"SENTINEL_POSTHOC_SELECTION_FOR_STABILITY"`
	Note                               string        `json:"note"`
	RepeatsPerSentinel                 int           `json:"repeats_per_sentinel"`
	RepeatIDs                          []string      `json:"repeat_ids"`
	SamplingSeed                       string        `json:"sampling_seed_status"` // exposed | fixed | unavailable
	Temperature                        float64       `json:"temperature"`
	MaxTokens                          int           `json:"max_tokens"`
	Strata                             []R1FStratum  `json:"strata"`
	Sentinels                          []R1FSentinel `json:"sentinels"`
	DecisionRule                       string        `json:"blind_retry_decision_rule"`
}

func rankKeyR1F(stratum, baseID, sourceCondition string) string {
	sum := sha256.Sum256([]byte("R1F" + stratum + baseID + sourceCondition))
	return hex.EncodeToString(sum[:])
}

// --- source record shapes (read-only) --------------------------------------

type r1fExtractRec struct {
	BaseID          string `json:"base_id"`
	ContextLevel    string `json:"context_level"`
	Gold            string `json:"gold"`
	RawText         string `json:"raw_text"`
	SemanticCorrect bool   `json:"semantic_correct"`
	ContractSuccess bool   `json:"contract_success"`
	FailureClass    string `json:"failure_class"`
	CropPath        string `json:"crop_path"`
}

type r1fDistRec struct {
	BaseID      string   `json:"base_id"`
	Condition   string   `json:"condition"`
	GoldValue   string   `json:"gold_value"`
	Opcode      string   `json:"opcode"`
	VisibleNums []string `json:"visible_numbers"`
	Distractors []string `json:"distractors"`
	RawText     string   `json:"raw_text"`
	CropPath    string   `json:"crop_path"`
	Score       R1DScore `json:"score"`
}

func readJSONFile(path string, v any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// representative low-scale failure classes (stratum B priority).
var r1fRepresentativeLowScale = map[string]bool{
	"DIGIT_SUBSTITUTION": true, "SUFFIX_TRUNCATION": true,
	"PREFIX_TRUNCATION": true, "DIGIT_DELETION": true,
}

// SelectR1FSentinels applies the frozen deterministic post-hoc selection
// (protocol §3, §4): five strata, four sentinels each, ranked by
// sha256("R1F"||stratum||base_id||source_condition) with a stated
// priority key where the stratum definition asks for a representative
// sub-mode. No manual cherry-picking. Reads only frozen prior results.
func SelectR1FSentinels(expDir string) ([]R1FSentinel, error) {
	var extract []r1fExtractRec
	if err := readJSONFile(filepath.Join(expDir, "results", "R1B_RECORDS.json"), &extract); err != nil {
		return nil, fmt.Errorf("R1B_RECORDS: %w", err)
	}
	var a1c6 []r1fExtractRec
	if err := readJSONFile(filepath.Join(expDir, "results", "R1A1_RECORDS.json"), &a1c6); err != nil {
		return nil, fmt.Errorf("R1A1_RECORDS: %w", err)
	}
	var dist []r1fDistRec
	if err := readJSONFile(filepath.Join(expDir, "results", "R1D_DISTRACTOR_RECORDS.json"), &dist); err != nil {
		return nil, fmt.Errorf("R1D_DISTRACTOR_RECORDS: %w", err)
	}
	var e0 []R1ERecord
	if err := readJSONFile(filepath.Join(expDir, "results", "R1E_RECORDS.json"), &e0); err != nil {
		return nil, fmt.Errorf("R1E_RECORDS: %w", err)
	}

	extractInstrSHA := sha256Hex([]byte(FrozenInstruction))
	assocInstrSHA := sha256Hex([]byte(R1DAssocInstruction))

	type ranked struct {
		s        R1FSentinel
		priority int // 0 = preferred sub-mode, 1 = otherwise
	}
	pick := func(stratum string, cands []ranked) []R1FSentinel {
		sort.Slice(cands, func(i, j int) bool {
			if cands[i].priority != cands[j].priority {
				return cands[i].priority < cands[j].priority
			}
			return cands[i].s.RankKey < cands[j].s.RankKey
		})
		var out []R1FSentinel
		for i := 0; i < len(cands) && len(out) < r1fSentinelsPerStratum; i++ {
			out = append(out, cands[i].s)
		}
		return out
	}

	var all []R1FSentinel

	// Stratum A — GOOD_OPERATING_REGION (R1-B B4, previously correct)
	var aCands []ranked
	for _, r := range extract {
		if r.ContextLevel != "B4" || !r.SemanticCorrect {
			continue
		}
		aCands = append(aCands, ranked{s: R1FSentinel{
			Stratum: "A", StratumName: "GOOD_OPERATING_REGION", SourceStage: "R1-B",
			SourceCondition: "B4", BaseID: r.BaseID, Capability: string(FrozenOpcode),
			Opcode: string(FrozenOpcode), Instruction: FrozenInstruction, PromptSHA256: extractInstrSHA,
			HasImage: true, Gold: r.Gold, PrevRawOutput: r.RawText, PrevSemanticCorrect: true,
			PrevContractSuccess: r.ContractSuccess, PrevFailureClass: r.FailureClass,
			SourceCropPath: r.CropPath, RankKey: rankKeyR1F("A", r.BaseID, "B4"),
		}})
	}
	all = append(all, pick("A", aCands)...)

	// Stratum B — LOW_SCALE_FAILURE (R1-B B0, previously wrong)
	var bCands []ranked
	for _, r := range extract {
		if r.ContextLevel != "B0" || r.SemanticCorrect {
			continue
		}
		prio := 1
		if r1fRepresentativeLowScale[r.FailureClass] {
			prio = 0
		}
		bCands = append(bCands, ranked{priority: prio, s: R1FSentinel{
			Stratum: "B", StratumName: "LOW_SCALE_FAILURE", SourceStage: "R1-B",
			SourceCondition: "B0", BaseID: r.BaseID, Capability: string(FrozenOpcode),
			Opcode: string(FrozenOpcode), Instruction: FrozenInstruction, PromptSHA256: extractInstrSHA,
			HasImage: true, Gold: r.Gold, PrevRawOutput: r.RawText, PrevSemanticCorrect: false,
			PrevContractSuccess: r.ContractSuccess, PrevFailureClass: r.FailureClass,
			SourceCropPath: r.CropPath, RankKey: rankKeyR1F("B", r.BaseID, "B0"),
		}})
	}
	all = append(all, pick("B", bCands)...)

	// Stratum C — HIGH_CONTEXT_FAILURE (R1-A1 A1C6_FULL_VIEWPORT, previously wrong;
	// commentary-contamination cases prioritised where available)
	var cCands []ranked
	for _, r := range a1c6 {
		if r.ContextLevel != "A1C6_FULL_VIEWPORT" || r.SemanticCorrect {
			continue
		}
		prio := 1
		if r.FailureClass == "COMMENTARY_CONTAMINATION" {
			prio = 0
		}
		cCands = append(cCands, ranked{priority: prio, s: R1FSentinel{
			Stratum: "C", StratumName: "HIGH_CONTEXT_FAILURE", SourceStage: "R1-A1",
			SourceCondition: "A1C6_FULL_VIEWPORT", BaseID: r.BaseID, Capability: string(FrozenOpcode),
			Opcode: string(FrozenOpcode), Instruction: FrozenInstruction, PromptSHA256: extractInstrSHA,
			HasImage: true, Gold: r.Gold, PrevRawOutput: r.RawText, PrevSemanticCorrect: false,
			PrevContractSuccess: r.ContractSuccess, PrevFailureClass: r.FailureClass,
			SourceCropPath: r.CropPath, RankKey: rankKeyR1F("C", r.BaseID, "A1C6_FULL_VIEWPORT"),
		}})
	}
	all = append(all, pick("C", cCands)...)

	// Stratum D — DISTRACTOR_ASSOCIATION_FAILURE (R1-D D1 K1/K2, previously wrong)
	var dCands []ranked
	for _, r := range dist {
		if (r.Condition != "D1K1" && r.Condition != "D1K2") || r.Score.ValueCorrect {
			continue
		}
		dCands = append(dCands, ranked{s: R1FSentinel{
			Stratum: "D", StratumName: "DISTRACTOR_ASSOCIATION_FAILURE", SourceStage: "R1-D",
			SourceCondition: r.Condition, BaseID: r.BaseID, Capability: R1DAssocOpcode,
			Opcode: R1DAssocOpcode, Instruction: R1DAssocInstruction, PromptSHA256: assocInstrSHA,
			HasImage: true, Gold: r.GoldValue, VisibleNumbers: r.VisibleNums, Distractors: r.Distractors,
			PrevRawOutput: r.RawText, PrevSemanticCorrect: false, PrevContractSuccess: r.Score.ContractSuccess,
			PrevFailureClass: r.Score.FailureClass, SourceCropPath: r.CropPath,
			RankKey: rankKeyR1F("D", r.BaseID, r.Condition),
		}})
	}
	all = append(all, pick("D", dCands)...)

	// Stratum E — NO_IMAGE_DEGENERATE (R1-E E0_NO_IMAGE, READ_ASSOCIATED_NUMBER)
	var eCands []ranked
	for _, r := range e0 {
		if r.Condition != "E0_NO_IMAGE" || r.Capability != "READ_ASSOCIATED_NUMBER" {
			continue
		}
		eCands = append(eCands, ranked{s: R1FSentinel{
			Stratum: "E", StratumName: "NO_IMAGE_DEGENERATE", SourceStage: "R1-E",
			SourceCondition: "E0_NO_IMAGE", BaseID: r.BaseID, Capability: R1DAssocOpcode,
			Opcode: R1DAssocOpcode, Instruction: R1DAssocInstruction, PromptSHA256: assocInstrSHA,
			HasImage: false, ImageSHA256: "NO_IMAGE", Gold: r.TaskGold, PrevRawOutput: r.RawText,
			PrevSemanticCorrect: r.TaskGoldCorrect, PrevContractSuccess: r.ContractSuccess,
			PrevFailureClass: r.FailureClass, RankKey: rankKeyR1F("E", r.BaseID, "E0_NO_IMAGE"),
		}})
	}
	all = append(all, pick("E", eCands)...)

	// assign ids + validate per-stratum counts
	byStratum := map[string]int{}
	for i := range all {
		byStratum[all[i].Stratum]++
		all[i].SentinelID = fmt.Sprintf("r1f-%s-%02d", strings.ToLower(all[i].Stratum), byStratum[all[i].Stratum])
	}
	for _, st := range R1FStrata {
		if byStratum[st.Key] != r1fSentinelsPerStratum {
			return nil, fmt.Errorf("stratum %s: selected %d sentinels, need %d", st.Key, byStratum[st.Key], r1fSentinelsPerStratum)
		}
	}
	return all, nil
}

// FreezeR1FImages copies each image-bearing sentinel's source crop bytes to
// datasets/R1F_images/<sentinel_id>.png (no re-rendering) and fills in the
// frozen image path + sha. Mutates the slice in place.
func FreezeR1FImages(expDir string, sentinels []R1FSentinel) error {
	imgDir := filepath.Join(expDir, "datasets", "R1F_images")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		return err
	}
	for i := range sentinels {
		s := &sentinels[i]
		if !s.HasImage {
			s.ImageSHA256 = "NO_IMAGE"
			continue
		}
		src := s.SourceCropPath
		if _, err := os.Stat(src); err != nil {
			// crop_path may be relative to the behavior-lab root; try expDir-relative fallback
			return fmt.Errorf("%s: source crop %q not found: %w", s.SentinelID, src, err)
		}
		body, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		dst := filepath.Join(imgDir, s.SentinelID+".png")
		if err := os.WriteFile(dst, body, 0o644); err != nil {
			return err
		}
		s.ImagePath = dst
		s.ImageSHA256 = sha256Hex(body)
	}
	return nil
}

// BuildR1FDataset assembles the frozen dataset.
func BuildR1FDataset(sentinels []R1FSentinel, temperature float64, maxTokens int) R1FDataset {
	return R1FDataset{
		Schema: r1fDatasetSchema, ExperimentID: ExperimentID,
		SentinelPosthocSelectionForStability: true,
		Note: "R1-F is a REPEATABILITY probe, not a fresh accuracy benchmark. Sentinels are " +
			"deliberately sampled from known operating and failure regimes using the frozen prior results.",
		RepeatsPerSentinel: r1fRepeatsPerSentinel,
		RepeatIDs:          []string{"R0", "R1", "R2", "R3", "R4"},
		SamplingSeed:       "unavailable", // the request sends no seed; LM Studio does not expose a fixed sampling seed. temp 0 => greedy argmax.
		Temperature:        temperature, MaxTokens: maxTokens,
		Strata:    R1FStrata,
		Sentinels: sentinels,
		DecisionRule: fmt.Sprintf(
			"BLIND_RETRY_NOT_USEFUL = true iff (>= %.0f%% of previously-wrong sentinels remain semantically wrong on all 5 exact retries) AND (semantic outcome invariant 5/5 in >= %.0f%% of all sentinels). Frozen pre-inference; not redefinable.",
			r1fWrongStayWrongThreshold*100, r1fSemanticInvariantThreshold*100),
	}
}

// --- execution ------------------------------------------------------------

// R1FRecord is one repeated call.
type R1FRecord struct {
	SentinelID           string `json:"sentinel_id"`
	Stratum              string `json:"stratum"`
	BaseID               string `json:"base_id"`
	Capability           string `json:"capability"`
	RepeatID             string `json:"repeat_id"`
	RepeatIndex          int    `json:"repeat_index"`
	HasImage             bool   `json:"has_image"`
	ImageSHA256          string `json:"image_sha256"`
	PromptSHA256         string `json:"prompt_sha256"`
	RawText              string `json:"raw_text"`
	NormalizedValue      string `json:"normalized_value"`
	SemanticCorrect      bool   `json:"semantic_correct"`
	ContractSuccess      bool   `json:"contract_success"`
	Abstained            bool   `json:"abstained"`
	FormatFailure        bool   `json:"format_failure"`
	UnsupportedAssertion bool   `json:"unsupported_assertion"`
	FailureClass         string `json:"failure_class,omitempty"`
	LatencyMS            int64  `json:"latency_ms"`
	PromptTokens         int    `json:"prompt_tokens"`
	CompletionTok        int    `json:"completion_tokens"`
	Error                string `json:"error,omitempty"`
}

func r1fNormalize(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.Trim(s, ".,:;!?\"'()[]{} \t\n")
	return strings.Join(strings.Fields(s), " ")
}

// scoreR1F scores one repeat with the scorer that matches the sentinel's
// capability (EXTRACT_NUMBER via the R1-A/B scoreRecord path; READ_
// ASSOCIATED_NUMBER via ScoreR1DAssoc).
func scoreR1F(s R1FSentinel, raw string, rec *R1FRecord) {
	if s.Capability == R1DAssocOpcode {
		sc := ScoreR1DAssoc(raw, s.Gold, s.VisibleNumbers, s.Distractors)
		rec.SemanticCorrect = sc.ValueCorrect
		rec.ContractSuccess = sc.ContractSuccess
		rec.Abstained = sc.Abstained
		rec.FormatFailure = sc.FormatFailure
		rec.UnsupportedAssertion = sc.UnsupportedAssertion
		rec.FailureClass = sc.FailureClass
		rec.NormalizedValue = sc.GotValue
		if rec.NormalizedValue == "" {
			rec.NormalizedValue = r1fNormalize(raw)
		}
		return
	}
	var ro RecordOutcome
	ro.RawText = raw
	scoreRecord(&ro, s.Gold)
	rec.SemanticCorrect = ro.SemanticCorrect
	rec.ContractSuccess = ro.ContractSuccess
	rec.Abstained = ro.Abstained
	rec.FormatFailure = ro.FormatFailure
	rec.UnsupportedAssertion = ro.UnsupportedAssertion
	rec.FailureClass = ro.FailureClass
	rec.NormalizedValue = digitsOnly.ReplaceAllString(strings.TrimSpace(raw), "")
	if rec.NormalizedValue == "" {
		rec.NormalizedValue = r1fNormalize(raw)
	}
}

// RunR1F executes exactly repeats*len(sentinels) calls. For each sentinel
// the frozen image bytes are loaded ONCE and reused across all repeats; no
// crop is rebuilt during the run.
func RunR1F(ctx context.Context, cfg RunConfig, ds R1FDataset) ([]R1FRecord, error) {
	client := newR1DClient(cfg)
	rawDir := filepath.Join(cfg.RunDir, "raw")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		return nil, err
	}
	var out []R1FRecord
	for _, s := range ds.Sentinels {
		var imgBytes []byte
		if s.HasImage {
			b, err := os.ReadFile(s.ImagePath)
			if err != nil {
				return nil, fmt.Errorf("%s: load frozen image: %w", s.SentinelID, err)
			}
			if got := sha256Hex(b); got != s.ImageSHA256 {
				return nil, fmt.Errorf("%s: frozen image sha mismatch (%s != %s)", s.SentinelID, got, s.ImageSHA256)
			}
			imgBytes = b
		}
		for idx, repeatID := range ds.RepeatIDs {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			default:
			}
			rec := R1FRecord{
				SentinelID: s.SentinelID, Stratum: s.Stratum, BaseID: s.BaseID,
				Capability: s.Capability, RepeatID: repeatID, RepeatIndex: idx,
				HasImage: s.HasImage, ImageSHA256: s.ImageSHA256, PromptSHA256: s.PromptSHA256,
			}
			start := time.Now()
			if s.HasImage {
				res, err := client.CompletePerception(ctx, target.PerceptionInput{
					Question: s.Instruction, Image: imgBytes, MediaType: "image/png",
				})
				rec.LatencyMS = time.Since(start).Milliseconds()
				if err != nil {
					rec.Error = err.Error()
				} else {
					rec.RawText = res.Content
					rec.PromptTokens = res.PromptTokensReported
					rec.CompletionTok = res.CompletionTokensReported
				}
			} else {
				tr, err := client.CompleteText(ctx, "", s.Instruction)
				rec.LatencyMS = time.Since(start).Milliseconds()
				if err != nil {
					rec.Error = err.Error()
				} else {
					rec.RawText = tr.Content
					rec.PromptTokens = tr.PromptTokensReported
					rec.CompletionTok = tr.CompletionTokensReported
				}
			}
			if rec.Error == "" {
				scoreR1F(s, rec.RawText, &rec)
			}
			out = append(out, rec)
			body, _ := json.MarshalIndent(rec, "", "  ")
			_ = os.WriteFile(filepath.Join(rawDir, s.SentinelID+"_"+repeatID+".json"), body, 0o644)
		}
	}
	return out, nil
}
