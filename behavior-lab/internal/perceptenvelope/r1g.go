package perceptenvelope

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/parrotlab"
	"tlaloc.local/behaviorlab/internal/target"
)

// R1-G EVIDENCE-BASED RECOVERY POLICY — final LFM2-VL characterisation stage.
//
// For deterministic LFM2-VL failures, which evidence-based input
// transformation actually causes WRONG -> CORRECT without unacceptable
// CORRECT -> WRONG? Fresh held-out bases, a deliberately adverse baseline
// condition, and a predeclared recovery condition are scored over the full
// frozen denominator (paired). Exact identical retry is NOT re-tested: it
// is imported from R1-F as a proven-inert negative control.

const (
	r1gDatasetSchema = "tlaloc.parrot-perceptual-envelope-r1.r1g-dataset.r1"
	r1gSeed          = Seed
)

// EXACT_IDENTICAL_RETRY status imported from R1-F (no new model calls).
const R1GExactRetryStatus = "PROVEN_INERT_IN_TESTED_REGIMES"

// Frozen recovery-verdict thresholds (protocol §15). Fixed BEFORE inference.
const (
	r1gEarnedDelta      = 0.20
	r1gPromisingDelta   = 0.10
	r1gNoBenefitBand    = 0.10
	r1gMaxDegradation   = 0.05
	r1gEarnedMcNemarSig = 0.05
)

// R1GFamily is a recovery family.
type R1GFamily struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Capability  string   `json:"capability"`
	Instruction string   `json:"instruction"`
	Conditions  []string `json:"conditions"` // [0]=baseline, rest=recovery
	Evidence    string   `json:"motivating_evidence"`
}

// R1GFamilies is the frozen family plan.
var R1GFamilies = []R1GFamily{
	{
		Key: "GA_SCALE", Name: "SCALE RECOVERY", Capability: string(FrozenOpcode), Instruction: FrozenInstruction,
		Conditions: []string{"GA0_LOW_SCALE", "GA1_SAFE_SCALE", "GA2_NOMINAL_SCALE"},
		Evidence:   "R1-B scale curve: 8px=0.27, 12=0.87, 16=0.90, 24=0.93, 32=0.97, 48=0.97; formal_safe_scale=16px, nominal=32px",
	},
	{
		Key: "GB_CONTEXT", Name: "CONTEXT REDUCTION RECOVERY", Capability: string(FrozenOpcode), Instruction: FrozenInstruction,
		Conditions: []string{"GB0_HIGH_CONTEXT", "GB1_LINE_RECOVERY", "GB2_TARGET_RECOVERY"},
		Evidence:   "R1-A1 fixed-scale context: TARGET 1.00, LINE 1.00, BLOCK 0.90, FULL_VIEWPORT 0.80 (mostly contract/verbosity degradation)",
	},
	{
		Key: "GC_ASSOC_REAL", Name: "DISTRACTOR/ASSOCIATION RECOVERY (REAL_INTERVENTION_REUSE)", Capability: R1DAssocOpcode, Instruction: R1DAssocInstruction,
		Conditions: []string{"GC_REAL_0", "GC_REAL_1", "GC_REAL_2"},
		Evidence:   "R1-D exploratory: K0=1.00, K1=0.64, K2=0.27; R1-E: READ_ASSOCIATED_NUMBER is genuinely visual. No unused real label/value bases remain.",
	},
	{
		Key: "GC_ASSOC_SYN", Name: "DISTRACTOR/ASSOCIATION RECOVERY (SYNTHETIC_REALISTIC_HELDOUT)", Capability: R1DAssocOpcode, Instruction: R1DAssocInstruction,
		Conditions: []string{"GC_SYN_0", "GC_SYN_1", "GC_SYN_2"},
		Evidence:   "canonical fresh mechanistic distractor-recovery test; frozen R1-C glyph-bank synthetic renderer; never pooled with REAL_DOCUMENT",
	},
	{
		Key: "GD_CUE", Name: "CUE / CROP ARTIFACT PROBE", Capability: string(FrozenOpcode), Instruction: FrozenInstruction,
		Conditions: []string{"GD0_TIGHT_VALUE_CUE", "GD1_PADDED_VALUE_CUE", "GD2_NO_VALUE_CUE"},
		Evidence:   "R1-D D0V tight value cue truncated short integers (32->3, 64->4, 350->50); appears to be a renderer/cue artifact, not a core failure",
	},
}

// R1GBase is one frozen recovery base.
type R1GBase struct {
	BaseID        string   `json:"base_id"`
	Family        string   `json:"family"`
	CandidateID   string   `json:"candidate_id,omitempty"`
	Page          int      `json:"page,omitempty"`
	Gold          string   `json:"gold"`
	RankKey       string   `json:"rank_key,omitempty"`
	SourceStage   string   `json:"source_stage"`  // FRESH_SOURCE_POOL | R1D_REAL_INTERVENTION_REUSE | SYNTHETIC_HELDOUT
	InterventionReuse bool `json:"intervention_reuse"`
	Candidate     *Candidate `json:"candidate,omitempty"`
	// synthetic-only
	SynLabel      string `json:"syn_label,omitempty"`
	SynValue      string `json:"syn_value,omitempty"`
	SynCompLabel  string `json:"syn_comp_label,omitempty"`
	SynCompValue  string `json:"syn_comp_value,omitempty"`
	// real-assoc reuse
	RealCompetitor string `json:"real_competitor,omitempty"`
	VisibleNumbers []string `json:"visible_numbers,omitempty"`
	R1DBaseIndex   int      `json:"r1d_base_index,omitempty"`
}

// R1GDataset is the frozen R1-G stimulus definition.
type R1GDataset struct {
	Schema       string      `json:"schema"`
	ExperimentID string      `json:"experiment_id"`
	Seed         string      `json:"seed"`
	Temperature  float64     `json:"temperature"`
	MaxTokens    int         `json:"max_tokens"`
	Families     []R1GFamily `json:"families"`
	ScaleBases   []R1GBase   `json:"scale_bases"`
	ContextBases []R1GBase   `json:"context_bases"`
	RealAssoc    []R1GBase   `json:"real_assoc_bases"`
	SynAssoc     []R1GBase   `json:"syn_assoc_bases"`
	CueBases     []R1GBase   `json:"cue_bases"`
	CrossRecoveryFamilyBaseReuse bool `json:"CROSS_RECOVERY_FAMILY_BASE_REUSE"`
	R1DGeometricCoincidenceExcluded int `json:"r1d_geometric_coincidence_excluded"`
	NAvailableFreshPool            int `json:"n_available_fresh_pool"`
	OCRFallbackAvailable           bool   `json:"OCR_FALLBACK_AVAILABLE"`
	OCREngine                      string `json:"ocr_engine,omitempty"`
	ExactRetryImported             string `json:"exact_identical_retry_status_imported_from_r1f"`
	Thresholds                     map[string]float64 `json:"frozen_recovery_verdict_thresholds"`
	Note                           string `json:"note"`
}

func rankKeyR1G(salt, candidateID string) string {
	sum := sha256.Sum256([]byte(salt + "|" + r1gSeed + "|" + candidateID))
	return hex.EncodeToString(sum[:])
}

// --- selection ------------------------------------------------------------

// usedByEarlierStages returns the set of SOURCE_POOL_R1 candidate_ids already
// consumed by R1-A0 / R1-A1 / R1-B / R1-C plus any that geometrically
// coincide with a frozen R1-D real label/value base (page + value) — those
// lines were shown to the model in R1-D/R1-E and are not "fresh".
func usedByEarlierStages(expDir string) (map[string]bool, int, error) {
	used := map[string]bool{}
	add := func(cid string) {
		if cid != "" {
			used[cid] = true
		}
	}
	for _, f := range []string{"R1A_BASES.json", "R1A1_BASES.json", "R1B_BASES.json"} {
		var a Allocation
		if err := readJSONFile(filepath.Join(expDir, "datasets", f), &a); err != nil {
			return nil, 0, fmt.Errorf("%s: %w", f, err)
		}
		for _, b := range a.Bases {
			add(b.Candidate.CandidateID)
		}
	}
	var rcRaw any
	if err := readJSONFile(filepath.Join(expDir, "datasets", "R1C_DATASET.json"), &rcRaw); err == nil {
		var walk func(any)
		walk = func(v any) {
			switch t := v.(type) {
			case map[string]any:
				if cid, ok := t["candidate_id"].(string); ok {
					add(cid)
				}
				for _, x := range t {
					walk(x)
				}
			case []any:
				for _, x := range t {
					walk(x)
				}
			}
		}
		walk(rcRaw)
	}
	// R1-D geometric coincidences
	var rd R1DAllocation
	coincide := 0
	if err := readJSONFile(filepath.Join(expDir, "datasets", "R1D_ASSOCIATION_DATASET.json"), &rd); err == nil {
		rdset := map[string]bool{}
		for _, b := range rd.Bases {
			if b.Eligible {
				rdset[fmt.Sprintf("%d|%s", b.Page, b.Value)] = true
			}
		}
		var pool SourcePool
		if err := readJSONFile(filepath.Join(expDir, "datasets", "SOURCE_POOL_R1.json"), &pool); err == nil {
			for _, c := range pool.Candidates {
				if rdset[fmt.Sprintf("%d|%s", c.Page, c.NormalizedTarget)] && !used[c.CandidateID] {
					used[c.CandidateID] = true
					coincide++
				}
			}
		}
	}
	return used, coincide, nil
}

// freshPoolCandidates returns the SOURCE_POOL_R1 candidates unused by
// R1-A0/A1/B/C (and not R1-D-coincident), in candidate_id order.
func freshPoolCandidates(expDir string) ([]Candidate, int, error) {
	var pool SourcePool
	if err := readJSONFile(filepath.Join(expDir, "datasets", "SOURCE_POOL_R1.json"), &pool); err != nil {
		return nil, 0, err
	}
	used, coincide, err := usedByEarlierStages(expDir)
	if err != nil {
		return nil, 0, err
	}
	var out []Candidate
	for _, c := range pool.Candidates {
		if !used[c.CandidateID] {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CandidateID < out[j].CandidateID })
	return out, coincide, nil
}

func pickRanked(cands []Candidate, salt string, n int) []Candidate {
	type rc struct {
		c   Candidate
		key string
	}
	rs := make([]rc, len(cands))
	for i, c := range cands {
		rs[i] = rc{c, rankKeyR1G(salt, c.CandidateID)}
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i].key < rs[j].key })
	var out []Candidate
	for i := 0; i < len(rs) && len(out) < n; i++ {
		out = append(out, rs[i].c)
	}
	return out
}

func candToScaleBase(c Candidate, family, salt string) R1GBase {
	cc := c
	return R1GBase{
		BaseID:      fmt.Sprintf("r1g-%s-%s", strings.ToLower(strings.TrimPrefix(family, "G")), c.CandidateID[:8]),
		Family:      family, CandidateID: c.CandidateID, Page: c.Page, Gold: c.NormalizedTarget,
		RankKey: rankKeyR1G(salt, c.CandidateID), SourceStage: "FRESH_SOURCE_POOL", Candidate: &cc,
	}
}

func (b R1GBase) asBase() Base {
	return Base{BaseID: b.BaseID, Stage: "R1-G", RankKey: b.RankKey, Candidate: *b.Candidate}
}

// r1gSynBases deterministically generates n synthetic label/value bases
// (abstract variable-name labels, since the frozen R1-C glyph bank only
// carries digits + 'e' + 'x' + punctuation). MULTI_DIGIT_INTEGER values,
// target != competitor, fixed seed, no prompt-answer leakage.
func r1gSynBases(n int) []R1GBase {
	var out []R1GBase
	state := sha256.Sum256([]byte("R1G_SYN|" + r1gSeed))
	next := func() uint64 {
		s := sha256.Sum256(state[:])
		state = s
		var v uint64
		for i := 0; i < 8; i++ {
			v = v<<8 | uint64(s[i])
		}
		return v
	}
	multiDigit := func() string {
		dl := 2 + int(next()%3) // 2..4
		lo := 1
		for i := 1; i < dl; i++ {
			lo *= 10
		}
		return fmt.Sprintf("%d", lo+int(next()%uint64(9*lo)))
	}
	for i := 0; i < n; i++ {
		li := 1 + int(next()%98)  // label index x1..x99
		ci := 1 + int(next()%98)
		for ci == li {
			ci = 1 + int(next()%98)
		}
		val := multiDigit()
		comp := multiDigit()
		for comp == val {
			comp = multiDigit()
		}
		out = append(out, R1GBase{
			BaseID:      fmt.Sprintf("r1g-syn-%02d", i+1),
			Family:      "GC_ASSOC_SYN", Gold: val, SourceStage: "SYNTHETIC_HELDOUT",
			SynLabel: fmt.Sprintf("x%d", li), SynValue: val,
			SynCompLabel: fmt.Sprintf("x%d", ci), SynCompValue: comp,
			VisibleNumbers: []string{val, comp},
		})
	}
	return out
}

// r1gFreshCompetitor deterministically draws one competitor value for a
// real-assoc reuse base: same digit length as the gold where possible,
// different value.
func r1gFreshCompetitor(gold, baseID string) string {
	state := sha256.Sum256([]byte("R1G_REAL_DISTRACTOR|" + r1gSeed + "|" + baseID))
	next := func() uint64 {
		s := sha256.Sum256(state[:])
		state = s
		var v uint64
		for i := 0; i < 8; i++ {
			v = v<<8 | uint64(s[i])
		}
		return v
	}
	dl := len(gold)
	if dl < 2 {
		dl = 2
	}
	if dl > 4 {
		dl = 4
	}
	lo := 1
	for i := 1; i < dl; i++ {
		lo *= 10
	}
	for {
		cand := fmt.Sprintf("%d", lo+int(next()%uint64(9*lo)))
		if cand != gold {
			return cand
		}
	}
}

// SelectR1GDataset builds the frozen R1-G dataset. Reads only frozen prior
// artifacts; no model output.
func SelectR1GDataset(expDir string, temperature float64, maxTokens int) (R1GDataset, error) {
	fresh, coincide, err := freshPoolCandidates(expDir)
	if err != nil {
		return R1GDataset{}, err
	}
	ds := R1GDataset{
		Schema: r1gDatasetSchema, ExperimentID: ExperimentID, Seed: r1gSeed,
		Temperature: temperature, MaxTokens: maxTokens, Families: R1GFamilies,
		R1DGeometricCoincidenceExcluded: coincide, NAvailableFreshPool: len(fresh),
		ExactRetryImported: R1GExactRetryStatus,
		Thresholds: map[string]float64{
			"earned_delta": r1gEarnedDelta, "promising_delta": r1gPromisingDelta,
			"no_benefit_band": r1gNoBenefitBand, "max_degradation": r1gMaxDegradation,
			"earned_mcnemar_p": r1gEarnedMcNemarSig,
		},
		Note: "R1-G characterises RECOVERY for LFM2-VL only. No second executor. Real and synthetic " +
			"association evidence are never pooled. Conditional recovery rate is never the headline.",
	}

	// G-A scale: 20 fresh
	nScale := 20
	if len(fresh) < nScale {
		nScale = len(fresh)
	}
	scaleCands := pickRanked(fresh, "R1G_SCALE", nScale)
	for _, c := range scaleCands {
		ds.ScaleBases = append(ds.ScaleBases, candToScaleBase(c, "GA_SCALE", "R1G_SCALE"))
	}

	// G-B context: prefer disjoint 20; else reuse the G-A set
	scaleSet := map[string]bool{}
	for _, c := range scaleCands {
		scaleSet[c.CandidateID] = true
	}
	var remaining []Candidate
	for _, c := range fresh {
		if !scaleSet[c.CandidateID] {
			remaining = append(remaining, c)
		}
	}
	if len(remaining) >= 20 {
		for _, c := range pickRanked(remaining, "R1G_CONTEXT", 20) {
			ds.ContextBases = append(ds.ContextBases, candToScaleBase(c, "GB_CONTEXT", "R1G_CONTEXT"))
		}
	} else {
		ds.CrossRecoveryFamilyBaseReuse = true
		for _, c := range scaleCands {
			b := candToScaleBase(c, "GB_CONTEXT", "R1G_CONTEXT")
			b.RankKey = rankKeyR1G("R1G_CONTEXT", c.CandidateID)
			ds.ContextBases = append(ds.ContextBases, b)
		}
	}

	// G-C real: the 22 frozen R1-D eligible bases, new competitor realizations
	var rd R1DAllocation
	if err := readJSONFile(filepath.Join(expDir, "datasets", "R1D_ASSOCIATION_DATASET.json"), &rd); err != nil {
		return R1GDataset{}, fmt.Errorf("R1D dataset: %w", err)
	}
	idx := 0
	for i, b := range rd.Bases {
		if !b.Eligible {
			continue
		}
		idx++
		comp := r1gFreshCompetitor(b.Value, b.BaseID)
		ds.RealAssoc = append(ds.RealAssoc, R1GBase{
			BaseID: fmt.Sprintf("r1g-real-%02d-%s", idx, b.CandidateID[:8]), Family: "GC_ASSOC_REAL",
			CandidateID: b.CandidateID, Page: b.Page, Gold: b.Value,
			SourceStage: "R1D_REAL_INTERVENTION_REUSE", InterventionReuse: true,
			RealCompetitor: comp, VisibleNumbers: []string{b.Value, comp}, R1DBaseIndex: i,
		})
	}

	// G-C synthetic: 24 fresh held-out
	ds.SynAssoc = r1gSynBases(24)

	// G-D cue probe: 12 fresh, disjoint from G-A/G-B
	cueSet := map[string]bool{}
	for _, b := range ds.ContextBases {
		cueSet[b.CandidateID] = true
	}
	for _, c := range scaleCands {
		cueSet[c.CandidateID] = true
	}
	var cuePoolCands []Candidate
	for _, c := range fresh {
		if !cueSet[c.CandidateID] {
			cuePoolCands = append(cuePoolCands, c)
		}
	}
	nCue := 12
	if len(cuePoolCands) < nCue {
		nCue = len(cuePoolCands)
	}
	for _, c := range pickRanked(cuePoolCands, "R1G_CUE", nCue) {
		b := candToScaleBase(c, "GD_CUE", "R1G_CUE")
		b.BaseID = fmt.Sprintf("r1g-cue-%s", c.CandidateID[:8])
		ds.CueBases = append(ds.CueBases, b)
	}

	// OCR fallback inventory
	if p, err := exec.LookPath("tesseract"); err == nil && p != "" {
		ds.OCRFallbackAvailable = true
		if out, e := exec.Command("tesseract", "--version").CombinedOutput(); e == nil {
			ds.OCREngine = strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
		} else {
			ds.OCREngine = "tesseract (version unknown)"
		}
	}
	return ds, nil
}

// --- records + execution -------------------------------------------------

// R1GRecord is one (family, base, condition) result.
type R1GRecord struct {
	Family               string `json:"family"`
	BaseID               string `json:"base_id"`
	Condition            string `json:"condition"`
	Role                 string `json:"role"` // BASELINE | RECOVERY
	Capability           string `json:"capability"`
	Gold                 string `json:"gold"`
	RawText              string `json:"raw_text"`
	NormalizedValue      string `json:"normalized_value"`
	SemanticCorrect      bool   `json:"semantic_correct"`
	ContractSuccess      bool   `json:"contract_success"`
	Abstained            bool   `json:"abstained"`
	FormatFailure        bool   `json:"format_failure"`
	UnsupportedAssertion bool   `json:"unsupported_assertion"`
	FailureClass         string `json:"failure_class,omitempty"`
	SelectedKind         string `json:"selected_kind,omitempty"`
	ImageSHA256          string `json:"image_sha256"`
	LatencyMS            int64  `json:"latency_ms"`
	PromptTokens         int    `json:"prompt_tokens"`
	CompletionTok        int    `json:"completion_tokens"`
	CropPath             string `json:"crop_path"`
	Error                string `json:"error,omitempty"`
}

func r1gScore(fam R1GFamily, base R1GBase, raw string, rec *R1GRecord) {
	if fam.Capability == R1DAssocOpcode {
		sc := ScoreR1DAssoc(raw, base.Gold, base.VisibleNumbers, nil)
		rec.SemanticCorrect = sc.ValueCorrect
		rec.ContractSuccess = sc.ContractSuccess
		rec.Abstained = sc.Abstained
		rec.FormatFailure = sc.FormatFailure
		rec.UnsupportedAssertion = sc.UnsupportedAssertion
		rec.FailureClass = sc.FailureClass
		rec.SelectedKind = sc.SelectedKind
		rec.NormalizedValue = sc.GotValue
		return
	}
	var ro RecordOutcome
	ro.RawText = raw
	scoreRecord(&ro, base.Gold)
	rec.SemanticCorrect = ro.SemanticCorrect
	rec.ContractSuccess = ro.ContractSuccess
	rec.Abstained = ro.Abstained
	rec.FormatFailure = ro.FormatFailure
	rec.UnsupportedAssertion = ro.UnsupportedAssertion
	rec.FailureClass = ro.FailureClass
	rec.NormalizedValue = digitsOnly.ReplaceAllString(strings.TrimSpace(raw), "")
}

func famByKey(key string) R1GFamily {
	for _, f := range R1GFamilies {
		if f.Key == key {
			return f
		}
	}
	return R1GFamily{}
}

// RunR1G renders every frozen condition once and executes exactly one model
// call per (family, base, condition). Deterministic order. No identical
// retry. bank is the frozen R1-C glyph bank; rd is the frozen R1-D
// allocation (for the real-assoc reuse geometry).
func RunR1G(ctx context.Context, cfg RunConfig, ds R1GDataset, bank *GlyphBank, rd R1DAllocation) ([]R1GRecord, error) {
	prov, err := parrotlab.NewPDFMemoryProvider(cfg.StoreDir, cfg.PDFPath)
	if err != nil {
		return nil, err
	}
	client := newR1DClient(cfg)
	cropDir := filepath.Join(cfg.RunDir, "crops")
	rawDir := filepath.Join(cfg.RunDir, "raw")
	for _, d := range []string{cropDir, rawDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}

	var out []R1GRecord
	call := func(fam R1GFamily, base R1GBase, condID, role string, img *image.RGBA) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		cropPath := filepath.Join(cropDir, strings.ToLower(base.BaseID+"_"+condID+".png"))
		if err := writeRGBAPNG(cropPath, img); err != nil {
			return err
		}
		body, err := os.ReadFile(cropPath)
		if err != nil {
			return err
		}
		rec := R1GRecord{
			Family: fam.Key, BaseID: base.BaseID, Condition: condID, Role: role,
			Capability: fam.Capability, Gold: base.Gold, ImageSHA256: sha256Hex(body), CropPath: cropPath,
		}
		start := time.Now()
		res, cerr := client.CompletePerception(ctx, target.PerceptionInput{Question: fam.Instruction, Image: body, MediaType: "image/png"})
		rec.LatencyMS = time.Since(start).Milliseconds()
		if cerr != nil {
			rec.Error = cerr.Error()
		} else {
			rec.RawText = res.Content
			rec.PromptTokens = res.PromptTokensReported
			rec.CompletionTok = res.CompletionTokensReported
			r1gScore(fam, base, res.Content, &rec)
		}
		out = append(out, rec)
		b, _ := json.MarshalIndent(rec, "", "  ")
		_ = os.WriteFile(filepath.Join(rawDir, strings.ToLower(base.BaseID+"_"+condID)+".json"), b, 0o644)
		return nil
	}

	// G-A SCALE
	famGA := famByKey("GA_SCALE")
	for _, base := range ds.ScaleBases {
		imgs, err := renderScaleConditions(prov, cfg.StoreDir, base)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", base.BaseID, err)
		}
		for ci, condID := range famGA.Conditions {
			role := "RECOVERY"
			if ci == 0 {
				role = "BASELINE"
			}
			if err := call(famGA, base, condID, role, imgs[ci]); err != nil {
				return out, err
			}
		}
	}

	// G-B CONTEXT
	famGB := famByKey("GB_CONTEXT")
	for _, base := range ds.ContextBases {
		imgs, err := renderContextConditions(prov, cfg.StoreDir, base)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", base.BaseID, err)
		}
		for ci, condID := range famGB.Conditions {
			role := "RECOVERY"
			if ci == 0 {
				role = "BASELINE"
			}
			if err := call(famGB, base, condID, role, imgs[ci]); err != nil {
				return out, err
			}
		}
	}

	// G-C REAL
	famGCR := famByKey("GC_ASSOC_REAL")
	for _, base := range ds.RealAssoc {
		rdBase := rd.Bases[base.R1DBaseIndex]
		imgs, err := renderRealAssocConditions(prov, bank, rdBase, base.RealCompetitor)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", base.BaseID, err)
		}
		for ci, condID := range famGCR.Conditions {
			role := "RECOVERY"
			if ci == 0 {
				role = "BASELINE"
			}
			if err := call(famGCR, base, condID, role, imgs[ci]); err != nil {
				return out, err
			}
		}
	}

	// G-C SYN
	famGCS := famByKey("GC_ASSOC_SYN")
	for _, base := range ds.SynAssoc {
		imgs, err := renderSynAssocConditions(bank, base)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", base.BaseID, err)
		}
		for ci, condID := range famGCS.Conditions {
			role := "RECOVERY"
			if ci == 0 {
				role = "BASELINE"
			}
			if err := call(famGCS, base, condID, role, imgs[ci]); err != nil {
				return out, err
			}
		}
	}

	// G-D CUE
	famGD := famByKey("GD_CUE")
	for _, base := range ds.CueBases {
		imgs, err := renderCueConditions(prov, cfg.StoreDir, base)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", base.BaseID, err)
		}
		for ci, condID := range famGD.Conditions {
			role := "RECOVERY"
			if ci == 0 {
				role = "BASELINE"
			}
			if err := call(famGD, base, condID, role, imgs[ci]); err != nil {
				return out, err
			}
		}
	}

	return out, nil
}

// --- OCR system-fallback comparison (no model calls) --------------------

// R1GOCRRecord is one deterministic OCR read of a baseline crop.
type R1GOCRRecord struct {
	Family          string `json:"family"`
	BaseID          string `json:"base_id"`
	Condition       string `json:"condition"`
	Gold            string `json:"gold"`
	OCRText         string `json:"ocr_text"`
	OCRDigits       string `json:"ocr_digits"`
	SemanticCorrect bool   `json:"semantic_correct"`
	Error           string `json:"error,omitempty"`
}

// RunR1GOCRFallback runs `tesseract` over every baseline crop already
// rendered by RunR1G and scores the digit string against the gold. No model
// calls; a clearly separate SYSTEM_FALLBACK comparison (protocol §11).
func RunR1GOCRFallback(records []R1GRecord) []R1GOCRRecord {
	var out []R1GOCRRecord
	for _, r := range records {
		if r.Role != "BASELINE" || r.Error != "" || r.CropPath == "" {
			continue
		}
		rec := R1GOCRRecord{Family: r.Family, BaseID: r.BaseID, Condition: r.Condition, Gold: r.Gold}
		cmd := exec.Command("tesseract", r.CropPath, "stdout", "--psm", "7", "-c", "tessedit_char_whitelist=0123456789")
		body, err := cmd.Output()
		if err != nil {
			// retry without the single-line PSM constraint
			cmd = exec.Command("tesseract", r.CropPath, "stdout", "--psm", "6", "-c", "tessedit_char_whitelist=0123456789")
			body, err = cmd.Output()
		}
		if err != nil {
			rec.Error = err.Error()
			out = append(out, rec)
			continue
		}
		rec.OCRText = strings.TrimSpace(string(body))
		rec.OCRDigits = digitsOnly.ReplaceAllString(rec.OCRText, "")
		g, _ := parseFamilyValue(FamMultiDigit, r.Gold)
		o, _ := parseFamilyValue(FamMultiDigit, rec.OCRDigits)
		rec.SemanticCorrect = o != "" && o == g
		out = append(out, rec)
	}
	return out
}
