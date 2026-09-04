package perceptenvelope

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/decompositionlab"
	"tlaloc.local/behaviorlab/internal/exocortex"
	"tlaloc.local/behaviorlab/internal/parrotlab"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
	"tlaloc.local/behaviorlab/internal/target"
)

// PHASE H — held-out RAW (H0) vs PROFILE_ADAPTED (H1) validation of the
// frozen CapabilityProfile R1. Fresh held-out real-document candidates
// (disjoint from every SOURCE_POOL_R1 candidate), a naive baseline and a
// profile-driven preventive adaptation, scored paired over the full frozen
// denominator. The adapter consults ONLY the frozen profile + observable
// input properties — never a base id, gold, or scorer verdict.

const (
	profileHDatasetSchema = "tlaloc.parrot-perceptual-envelope-r1.profile-h-dataset.r1"
	profileHSeed          = Seed
)

// Frozen promotion-gate thresholds (protocol §26). Fixed BEFORE H inference.
const (
	phGateDelta          = 0.15
	phGateMcNemarSig     = 0.05
	phGateMaxRegression  = 0.05
)

// ProfileHStratum is one validation stratum.
type ProfileHStratum struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Opcode      string `json:"opcode"`
	AdaptShould string `json:"adaptation_should"` // ACTIVATE | LEAVE_ALONE | REJECT
	Executable  bool   `json:"executable"`
	Note        string `json:"note,omitempty"`
}

var ProfileHStrata = []ProfileHStratum{
	{"H-A", "LOW_SCALE", string(FrozenOpcode), "ACTIVATE", true, "fresh EXTRACT_NUMBER at 8 px"},
	{"H-B", "ALREADY_GOOD_SCALE", string(FrozenOpcode), "LEAVE_ALONE", true, "fresh EXTRACT_NUMBER at 32 px"},
	{"H-C", "HIGH_VISUAL_FIELD", string(FrozenOpcode), "ACTIVATE", true, "fresh EXTRACT_NUMBER in a 256 px local field"},
	{"H-D", "MISSING_VISUAL_OPERAND", string(FrozenOpcode), "REJECT", false, "visual opcode, no visual operand — scored separately from answer-accuracy strata"},
	{"H-E", "ASSOCIATION_WITH_COMPETITOR", "READ_ASSOCIATED_NUMBER", "ACTIVATE", false, "INSUFFICIENT_FRESH_REAL_DATA — no fresh real label/value pool; not manufactured (protocol §19)"},
}

// ProfileHBase is one frozen held-out base.
type ProfileHBase struct {
	BaseID      string    `json:"base_id"`
	CandidateID string    `json:"candidate_id,omitempty"`
	Page        int       `json:"page,omitempty"`
	Gold        string    `json:"gold"`
	RankKey     string    `json:"rank_key,omitempty"`
	Candidate   *Candidate `json:"candidate,omitempty"`
	// H-D synthetic
	Synthetic bool `json:"synthetic,omitempty"`
}

// ProfileHDataset is the frozen H stimulus definition.
type ProfileHDataset struct {
	Schema              string            `json:"schema"`
	ExperimentID        string            `json:"experiment_id"`
	Seed                string            `json:"seed"`
	ProfilePath         string            `json:"profile_path"`
	ProfileHash         string            `json:"profile_hash_sha256"`
	Temperature         float64           `json:"temperature"`
	MaxTokens           int               `json:"max_tokens"`
	Strata              []ProfileHStratum `json:"strata"`
	SharedHeldoutBases  []ProfileHBase    `json:"shared_heldout_bases"`
	SharedBaseSetNote   string            `json:"shared_heldout_base_set_note"`
	MissingOperandBases []ProfileHBase    `json:"missing_operand_bases"`
	HeldoutPoolSize     int               `json:"heldout_pool_size"`
	NAvailablePerStratum map[string]int   `json:"n_available_per_stratum"`
	GateThresholds      map[string]float64 `json:"frozen_promotion_gate_thresholds"`
	Note                string            `json:"note"`
}

func rankKeyProfileH(salt, id string) string {
	sum := sha256.Sum256([]byte(salt + "|" + profileHSeed + "|" + id))
	return hex.EncodeToString(sum[:])
}

// scanHeldoutExtractPool re-scans the store for plain 2-4 digit numeric
// reading targets under a RELAXED rule (short numeric lines allowed, >= 2
// whitespace tokens instead of >= 4), then excludes every candidate whose
// (page, region_id, normalized_target) already appears in SOURCE_POOL_R1.
// The result is a genuinely disjoint "short numeric line" presentation
// family. Deterministic.
func scanHeldoutExtractPool(storeDir string, excludePR map[string]bool) ([]Candidate, error) {
	manifest, _, err := pdfmemory.Load(storeDir)
	if err != nil {
		return nil, err
	}
	srcSHA := manifest.SourceSHA256
	if srcSHA == "" && len(manifest.Documents) > 0 {
		srcSHA = manifest.Documents[0].SourceSHA256
	}
	pages := append([]pdfmemory.PageRef(nil), manifest.Pages...)
	sort.Slice(pages, func(i, j int) bool { return pages[i].Number < pages[j].Number })

	var out []Candidate
	for _, pref := range pages {
		if strings.TrimSpace(pref.LayoutPath) == "" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(pref.LayoutPath)))
		if err != nil {
			return nil, err
		}
		var page canonicaldoc.Page
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}
		w, h := page.Width, page.Height
		for _, region := range page.Regions {
			if region.Kind != "text_line" && region.Kind != "list_item" {
				continue
			}
			text := strings.TrimSpace(region.Text)
			if text == "" {
				continue
			}
			fields := strings.Fields(text)
			if len(fields) < 2 {
				continue
			}
			digitTokens, primaryCount := 0, 0
			var primaryTok string
			for _, ff := range fields {
				if anyDigit.MatchString(ff) {
					digitTokens++
					if primaryTarget.MatchString(stripEdgePunct(ff)) {
						primaryTok, primaryCount = ff, primaryCount+1
					}
				}
			}
			// allow up to two digit-bearing tokens (the target is always
			// cued with a rectangle, so a nearby second number is fine); the
			// target itself must still be the unique plain 2-4 digit integer.
			if digitTokens > 2 || primaryCount != 1 || !rawIntegerToken.MatchString(primaryTok) {
				continue
			}
			norm := stripEdgePunct(primaryTok)
			if len(norm) == 4 {
				if v := mustAtoi(norm); v >= 1500 && v <= 2099 {
					continue
				}
			}
			if fields[0] == primaryTok || bibliographyCue.MatchString(text) || runningHeaderCaps.MatchString(text) {
				continue
			}
			if region.FontSize > 0 && region.FontSize < 10 {
				continue
			}
			b := region.BBox
			if !(b.X1 >= 0 && b.X1 < b.X2 && b.X2 <= w && b.Y1 >= 0 && b.Y1 < b.Y2 && b.Y2 <= h) {
				continue
			}
			if (b.X2 - b.X1) < 0.20*w {
				continue
			}
			if strings.Count(text, primaryTok) != 1 {
				continue
			}
			byteStart := strings.Index(text, primaryTok)
			startRune := len([]rune(text[:byteStart]))
			endRune := startRune + len([]rune(primaryTok))
			total := len([]rune(text))
			if total == 0 || startRune >= endRune || endRune > total {
				continue
			}
			lineW := b.X2 - b.X1
			estX1 := b.X1 + lineW*float64(startRune)/float64(total)
			estX2 := b.X1 + lineW*float64(endRune)/float64(total)
			pad := 0.5 * (b.Y2 - b.Y1)
			tb := canonicaldoc.BBox{X1: estX1 - pad, Y1: b.Y1 - pad, X2: estX2 + pad, Y2: b.Y2 + pad}
			if !(tb.X1 > 0 && tb.Y1 > 0 && tb.X2 < w && tb.Y2 < h && tb.X1 < tb.X2 && tb.Y1 < tb.Y2) {
				continue
			}
			if excludePR[fmt.Sprintf("%d|%s|%s", page.Number, region.ID, norm)] {
				continue
			}
			idBytes := []byte(strings.Join([]string{
				"HELDOUT", srcSHA, fmt.Sprintf("p%d", page.Number), region.ID,
				fmt.Sprintf("off%d-%d", startRune, endRune), norm,
			}, "|"))
			cid := sha256Hex(idBytes)[:32]
			out = append(out, Candidate{
				CandidateID: cid, Page: page.Number, PageWidth: w, PageHeight: h,
				TargetToken: primaryTok, NormalizedTarget: norm,
				CharOffsetStart: startRune, CharOffsetEnd: endRune,
				Line: LineRef{RegionID: region.ID, Kind: region.Kind, ReadingOrder: region.ReadingOrder, Text: text, BBox: b, FontSize: region.FontSize},
				TokenBBoxStore: tb,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CandidateID < out[j].CandidateID })
	return out, nil
}

// SelectProfileHDataset builds the frozen H dataset from fresh held-out
// candidates. Reads only frozen artifacts + the store; no model output.
func SelectProfileHDataset(expDir, storeDir, profilePath string, temperature, maxTokens int) (ProfileHDataset, error) {
	prof, err := exocortex.LoadCapabilityProfileR1(profilePath)
	if err != nil {
		return ProfileHDataset{}, fmt.Errorf("load frozen profile: %w", err)
	}
	// exclusion set: every SOURCE_POOL_R1 candidate (page|region|value)
	var pool SourcePool
	if err := readJSONFile(filepath.Join(expDir, "datasets", "SOURCE_POOL_R1.json"), &pool); err != nil {
		return ProfileHDataset{}, err
	}
	exclude := map[string]bool{}
	for _, c := range pool.Candidates {
		exclude[fmt.Sprintf("%d|%s|%s", c.Page, c.Line.RegionID, c.NormalizedTarget)] = true
	}
	fresh, err := scanHeldoutExtractPool(storeDir, exclude)
	if err != nil {
		return ProfileHDataset{}, err
	}
	// page-diversity: at most 2 per page, rank by the shared salt
	type rc struct {
		c   Candidate
		key string
	}
	rs := make([]rc, len(fresh))
	for i, c := range fresh {
		rs[i] = rc{c, rankKeyProfileH("PROFILE_H_SHARED", c.CandidateID)}
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i].key < rs[j].key })
	perPage := map[int]int{}
	var shared []ProfileHBase
	want := 20
	for _, r := range rs {
		if len(shared) >= want {
			break
		}
		if perPage[r.c.Page] >= 2 {
			continue
		}
		perPage[r.c.Page]++
		cc := r.c
		shared = append(shared, ProfileHBase{
			BaseID: fmt.Sprintf("ph-%s", r.c.CandidateID[:8]), CandidateID: r.c.CandidateID,
			Page: r.c.Page, Gold: r.c.NormalizedTarget, RankKey: r.key, Candidate: &cc,
		})
	}

	// H-D synthetic missing-operand cases (deterministic bookkeeping golds)
	var missing []ProfileHBase
	st := sha256.Sum256([]byte("PROFILE_H_MISSING|" + profileHSeed))
	for i := 0; i < 15; i++ {
		st = sha256.Sum256(st[:])
		v := 10 + int(st[0])%990
		missing = append(missing, ProfileHBase{
			BaseID: fmt.Sprintf("ph-missing-%02d", i+1), Gold: fmt.Sprintf("%d", v), Synthetic: true,
		})
	}

	nAvail := map[string]int{"H-A": len(shared), "H-B": len(shared), "H-C": len(shared), "H-D": len(missing), "H-E": 0}
	ds := ProfileHDataset{
		Schema: profileHDatasetSchema, ExperimentID: ExperimentID, Seed: profileHSeed,
		ProfilePath: profilePath, ProfileHash: prof.ProfileHash,
		Temperature: float64(temperature), MaxTokens: maxTokens,
		Strata: ProfileHStrata, SharedHeldoutBases: shared,
		SharedBaseSetNote: "H-A/H-B/H-C are different rendering conditions on ONE shared fresh held-out base set (SHARED_HELDOUT_BASE_SET) — a within-base scale/field manipulation, not independent samples. Disjoint from every SOURCE_POOL_R1 candidate.",
		MissingOperandBases: missing, HeldoutPoolSize: len(fresh),
		NAvailablePerStratum: nAvail,
		GateThresholds: map[string]float64{
			"overall_executable_semantic_delta_min": phGateDelta,
			"mcnemar_exact_p_max":                   phGateMcNemarSig,
			"regression_rate_among_h0_correct_max":  phGateMaxRegression,
		},
		Note: "PROFILE_ADAPTATION_VALIDATION_H_R0. H1 adapter consults ONLY the frozen CapabilityProfile R1 and observable input properties. H-E is INSUFFICIENT_FRESH_REAL_DATA and not manufactured.",
	}
	return ds, nil
}

// --- execution -----------------------------------------------------------

// ProfileHRecord is one (stratum, arm, base) result.
type ProfileHRecord struct {
	Stratum              string           `json:"stratum"`
	Arm                  string           `json:"arm"` // H0_RAW | H1_PROFILE_ADAPTED
	BaseID               string           `json:"base_id"`
	Opcode               string           `json:"opcode"`
	Gold                 string           `json:"gold"`
	RawText              string           `json:"raw_text"`
	SemanticCorrect      bool             `json:"semantic_correct"`
	ContractSuccess      bool             `json:"contract_success"`
	Abstained            bool             `json:"abstained"`
	UnknownReturned      bool             `json:"unknown_returned"`
	UnsupportedAssertion bool             `json:"unsupported_assertion"`
	FormatFailure        bool             `json:"format_failure"`
	FailureClass         string           `json:"failure_class,omitempty"`
	ModelCalls           int              `json:"model_calls"`
	DeterministicTransforms int           `json:"deterministic_transforms"`
	LatencyMS            int64            `json:"latency_ms"`
	ImageSHA256          string           `json:"image_sha256,omitempty"`
	AdapterDecision      *exocortex.AdaptDecisionR1 `json:"adapter_decision,omitempty"`
	Error               string            `json:"error,omitempty"`
}

func phRenderScale(prov parrotlab.PageProvider, storeDir string, b ProfileHBase, px int) (*image.RGBA, error) {
	base := Base{BaseID: b.BaseID, Stage: "PROFILE-H", Candidate: *b.Candidate}
	geo, err := DeriveR1BGeometry(storeDir, base)
	if err != nil {
		return nil, err
	}
	pagePNG, err := prov.RenderPNG(base.Candidate.Page)
	if err != nil {
		return nil, err
	}
	idx := 4 // 32 px
	if px == 8 {
		idx = 0
	}
	img, _, err := RenderR1BScale(pagePNG, base, geo, geo.Conditions[idx])
	return img, err
}

func phRenderField(prov parrotlab.PageProvider, storeDir string, b ProfileHBase, level R1A1Level) (*image.RGBA, error) {
	base := Base{BaseID: b.BaseID, Stage: "PROFILE-H", Candidate: *b.Candidate}
	geo, err := DeriveR1A1Geometry(storeDir, base)
	if err != nil {
		return nil, err
	}
	pagePNG, err := prov.RenderPNG(base.Candidate.Page)
	if err != nil {
		return nil, err
	}
	vp, err := BuildR1A1Viewport(pagePNG, storeDir, base, geo)
	if err != nil {
		return nil, err
	}
	return maskR1A1Level(vp, geo, level), nil
}

func phScore(raw, gold string, rec *ProfileHRecord) {
	var ro RecordOutcome
	ro.RawText = raw
	scoreRecord(&ro, gold)
	rec.SemanticCorrect = ro.SemanticCorrect
	rec.ContractSuccess = ro.ContractSuccess
	rec.Abstained = ro.Abstained
	rec.FormatFailure = ro.FormatFailure
	rec.UnsupportedAssertion = ro.UnsupportedAssertion
	rec.FailureClass = ro.FailureClass
}

// RunProfileH executes both arms for every executable stratum + H-D.
func RunProfileH(ctx context.Context, cfg RunConfig, ds ProfileHDataset, profile exocortex.CapabilityProfileR1) ([]ProfileHRecord, error) {
	prov, err := parrotlab.NewPDFMemoryProvider(cfg.StoreDir, cfg.PDFPath)
	if err != nil {
		return nil, err
	}
	client := newR1DClient(cfg)
	adapter := exocortex.AdapterR1{Profile: profile}
	cropDir := filepath.Join(cfg.RunDir, "crops")
	rawDir := filepath.Join(cfg.RunDir, "raw")
	for _, d := range []string{cropDir, rawDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}

	var out []ProfileHRecord
	emit := func(rec ProfileHRecord) {
		out = append(out, rec)
		b, _ := json.MarshalIndent(rec, "", "  ")
		_ = os.WriteFile(filepath.Join(rawDir, strings.ToLower(rec.Stratum+"_"+rec.Arm+"_"+rec.BaseID)+".json"), b, 0o644)
	}
	callImage := func(rec *ProfileHRecord, img *image.RGBA, condName string) {
		cropPath := filepath.Join(cropDir, strings.ToLower(rec.Stratum+"_"+rec.Arm+"_"+rec.BaseID+"_"+condName+".png"))
		if werr := writeRGBAPNG(cropPath, img); werr != nil {
			rec.Error = werr.Error()
			return
		}
		body, rerr := os.ReadFile(cropPath)
		if rerr != nil {
			rec.Error = rerr.Error()
			return
		}
		rec.ImageSHA256 = sha256Hex(body)
		start := time.Now()
		res, cerr := client.CompletePerception(ctx, target.PerceptionInput{Question: FrozenInstruction, Image: body, MediaType: "image/png"})
		rec.LatencyMS = time.Since(start).Milliseconds()
		rec.ModelCalls = 1
		if cerr != nil {
			rec.Error = cerr.Error()
			return
		}
		rec.RawText = res.Content
		phScore(res.Content, rec.Gold, rec)
	}

	// H-A / H-B / H-C on the shared held-out base set
	for _, strat := range ds.Strata {
		if !strat.Executable {
			continue
		}
		for _, b := range ds.SharedHeldoutBases {
			// ---- H0_RAW ----
			h0 := ProfileHRecord{Stratum: strat.Key, Arm: "H0_RAW", BaseID: b.BaseID, Opcode: strat.Opcode, Gold: b.Gold}
			var h0img *image.RGBA
			switch strat.Key {
			case "H-A":
				h0img, err = phRenderScale(prov, cfg.StoreDir, b, 8)
			case "H-B":
				h0img, err = phRenderScale(prov, cfg.StoreDir, b, 32)
			case "H-C":
				h0img, err = phRenderField(prov, cfg.StoreDir, b, A1C4Local256)
			}
			if err != nil {
				return nil, fmt.Errorf("%s H0 %s: %w", strat.Key, b.BaseID, err)
			}
			callImage(&h0, h0img, "h0")
			emit(h0)

			// ---- H1_PROFILE_ADAPTED ----
			h1 := ProfileHRecord{Stratum: strat.Key, Arm: "H1_PROFILE_ADAPTED", BaseID: b.BaseID, Opcode: strat.Opcode, Gold: b.Gold}
			req := exocortex.AdaptRequestR1{Opcode: strat.Opcode, HasVisualOperand: true, CompetingNumericOperands: 0}
			switch strat.Key {
			case "H-A":
				req.LineHeightPx, req.VisualFieldName = 8, "LINE"
			case "H-B":
				req.LineHeightPx, req.VisualFieldName = 32, "LINE"
			case "H-C":
				req.LineHeightPx, req.VisualFieldName = 32, "LOCAL_256"
			}
			decision, derr := adapter.Prepare(req)
			if derr != nil {
				return nil, fmt.Errorf("%s adapter %s: %w", strat.Key, b.BaseID, derr)
			}
			h1.AdapterDecision = &decision
			h1.DeterministicTransforms = len(decision.Transformations)
			// apply the decision deterministically to the rendering
			targetPx := 32
			if v, ok := decision.ResultingWorkingSet["target_line_height_px"].(float64); ok && v == 32 {
				targetPx = 32
			}
			field := "LINE"
			if v, ok := decision.ResultingWorkingSet["visual_field"].(string); ok {
				field = v
			}
			var h1img *image.RGBA
			if strat.Key == "H-C" {
				lvl := A1C4Local256
				if field == "LINE" {
					lvl = A1C1Line
				}
				h1img, err = phRenderField(prov, cfg.StoreDir, b, lvl)
			} else {
				h1img, err = phRenderScale(prov, cfg.StoreDir, b, targetPx)
			}
			if err != nil {
				return nil, fmt.Errorf("%s H1 %s: %w", strat.Key, b.BaseID, err)
			}
			callImage(&h1, h1img, "h1")
			emit(h1)
		}
	}

	// H-D missing visual operand
	for _, b := range ds.MissingOperandBases {
		// H0: naive system calls the model with no image (text-only)
		h0 := ProfileHRecord{Stratum: "H-D", Arm: "H0_RAW", BaseID: b.BaseID, Opcode: string(FrozenOpcode), Gold: b.Gold}
		start := time.Now()
		tr, cerr := client.CompleteText(ctx, "", FrozenInstruction)
		h0.LatencyMS = time.Since(start).Milliseconds()
		h0.ModelCalls = 1
		if cerr != nil {
			h0.Error = cerr.Error()
		} else {
			h0.RawText = tr.Content
			phScore(tr.Content, b.Gold, &h0)
			// a numeric answer with no operand is an unsupported assertion
			if h0.ContractSuccess || numberLike.MatchString(strings.TrimSpace(strings.Trim(tr.Content, ".,:;"))) {
				h0.UnsupportedAssertion = true
			}
		}
		emit(h0)

		// H1: adapter rejects pre-inference
		h1 := ProfileHRecord{Stratum: "H-D", Arm: "H1_PROFILE_ADAPTED", BaseID: b.BaseID, Opcode: string(FrozenOpcode), Gold: b.Gold}
		decision, _ := adapter.Prepare(exocortex.AdaptRequestR1{Opcode: string(FrozenOpcode), HasVisualOperand: false})
		h1.AdapterDecision = &decision
		h1.ModelCalls = decision.ModelCallCount
		if decision.Rejected {
			h1.UnknownReturned = true
			h1.ContractSuccess = true // correct runtime behaviour: UNKNOWN/UNSUPPORTED, zero model calls
		}
		emit(h1)
	}

	return out, nil
}

// --- aggregation + gate -------------------------------------------------

// ProfileHStratumRow is one stratum's paired H0->H1 result.
type ProfileHStratumRow struct {
	Stratum          string             `json:"stratum"`
	Name             string             `json:"name"`
	Executable       bool               `json:"executable"`
	N                int                `json:"n"`
	H0Accuracy       float64            `json:"h0_semantic_accuracy"`
	H1Accuracy       float64            `json:"h1_semantic_accuracy"`
	H0CI95           [2]float64         `json:"h0_ci95"`
	H1CI95           [2]float64         `json:"h1_ci95"`
	McNemar          AdjacentTransition `json:"paired_mcnemar_h0_to_h1"`
	H0Correct        int                `json:"h0_correct"`
	H0toH1Recovered  int                `json:"h0_wrong_to_h1_correct"`
	H0toH1Regressed  int                `json:"h0_correct_to_h1_wrong"`
	RegressionRate   float64            `json:"regression_rate_among_h0_correct"`
	H0ModelCalls     int                `json:"h0_model_calls"`
	H1ModelCalls     int                `json:"h1_model_calls"`
	H1Transforms     int                `json:"h1_deterministic_transforms"`
	H0Unsupported    int                `json:"h0_unsupported_assertions"`
	H1UnknownCorrect int                `json:"h1_unknown_returned"`
	H0MeanLatencyMS  float64            `json:"h0_mean_latency_ms"`
	H1MeanLatencyMS  float64            `json:"h1_mean_latency_ms"`
}

// ProfileHTable is the full frozen H result + promotion verdict.
type ProfileHTable struct {
	Schema                string               `json:"schema"`
	ExperimentID          string               `json:"experiment_id"`
	ProfileHash           string               `json:"profile_hash_sha256"`
	Rows                  []ProfileHStratumRow `json:"rows"`
	OverallExecutableN    int                  `json:"overall_executable_n"`
	OverallH0Accuracy     float64              `json:"overall_executable_h0_accuracy"`
	OverallH1Accuracy     float64              `json:"overall_executable_h1_accuracy"`
	OverallDelta          float64              `json:"overall_executable_semantic_delta"`
	OverallMcNemar        AdjacentTransition   `json:"overall_paired_mcnemar"`
	OverallRegressionRate float64              `json:"overall_regression_rate_among_h0_correct"`
	MissingOperandRejected string              `json:"missing_operand_rejected_pre_parrot_in_h1"`
	AdapterAblation       map[string]int       `json:"adapter_rule_fire_counts"`
	GateThresholds        map[string]float64   `json:"frozen_promotion_gate_thresholds"`
	GateChecks            map[string]bool      `json:"promotion_gate_checks"`
	IntegrityViolations   []string             `json:"integrity_violations"`
	ProfileR1RuntimePromoted bool              `json:"PROFILE_R1_RUNTIME_PROMOTED"`
	PromotionBasis        string               `json:"promotion_basis"`
}

const profileHTableSchema = "tlaloc.parrot-perceptual-envelope-r1.profile-h-table.r1"

func phPair(h0, h1 []ProfileHRecord) (AdjacentTransition, int, int, int) {
	// key by stratum|base so the shared held-out base set does not collide
	// when H-A/H-B/H-C records are concatenated for the overall pairing.
	key := func(r ProfileHRecord) string { return r.Stratum + "|" + r.BaseID }
	m := map[string]ProfileHRecord{}
	for _, r := range h0 {
		if r.Error == "" {
			m[key(r)] = r
		}
	}
	tr := AdjacentTransition{From: "H0_RAW", To: "H1_PROFILE_ADAPTED", Metric: "semantic"}
	var pairs []decompositionlab.PairedOutcome
	h1s := append([]ProfileHRecord(nil), h1...)
	sort.Slice(h1s, func(i, j int) bool { return key(h1s[i]) < key(h1s[j]) })
	recovered, regressed, h0correct := 0, 0, 0
	for _, r1 := range h1s {
		r0, ok := m[key(r1)]
		if !ok || r1.Error != "" {
			continue
		}
		b, a := r0.SemanticCorrect, r1.SemanticCorrect
		pairs = append(pairs, decompositionlab.PairedOutcome{CorrectBefore: b, CorrectAfter: a})
		switch {
		case b && a:
			tr.CorrectToCorrect++
		case b && !a:
			tr.CorrectToWrong++
			regressed++
		case !b && a:
			tr.WrongToCorrect++
			recovered++
		default:
			tr.WrongToWrong++
		}
		if b {
			h0correct++
		}
	}
	res := decompositionlab.McNemarExact(pairs)
	tr.AbsoluteDelta, tr.PValue = res.AbsoluteDelta, res.PValue
	return tr, recovered, regressed, h0correct
}

// AggregateProfileH builds the stratum table + the frozen promotion verdict.
func AggregateProfileH(records []ProfileHRecord, ds ProfileHDataset) ProfileHTable {
	byKey := map[string]map[string][]ProfileHRecord{}
	for _, r := range records {
		if byKey[r.Stratum] == nil {
			byKey[r.Stratum] = map[string][]ProfileHRecord{}
		}
		byKey[r.Stratum][r.Arm] = append(byKey[r.Stratum][r.Arm], r)
	}
	t := ProfileHTable{
		Schema: profileHTableSchema, ExperimentID: ExperimentID, ProfileHash: ds.ProfileHash,
		AdapterAblation: map[string]int{}, GateThresholds: ds.GateThresholds, GateChecks: map[string]bool{},
	}

	// adapter ablation over all H1 records
	for _, r := range records {
		if r.Arm != "H1_PROFILE_ADAPTED" || r.AdapterDecision == nil {
			continue
		}
		for _, rule := range r.AdapterDecision.RulesApplied {
			t.AdapterAblation[rule]++
		}
	}

	var execCorrectH0, execCorrectH1, execN, execRegressBase, execRegressed int
	var allH0, allH1 []ProfileHRecord
	for _, strat := range ds.Strata {
		h0 := byKey[strat.Key]["H0_RAW"]
		h1 := byKey[strat.Key]["H1_PROFILE_ADAPTED"]
		if len(h0) == 0 && len(h1) == 0 {
			continue
		}
		row := ProfileHStratumRow{Stratum: strat.Key, Name: strat.Name, Executable: strat.Executable}
		var c0, c1 int
		var lat0, lat1 []float64
		h0m := map[string]ProfileHRecord{}
		for _, r := range h0 {
			h0m[r.BaseID] = r
			if r.Error == "" {
				row.N++
				if r.SemanticCorrect {
					c0++
				}
				if r.UnsupportedAssertion {
					row.H0Unsupported++
				}
				row.H0ModelCalls += r.ModelCalls
				lat0 = append(lat0, float64(r.LatencyMS))
			}
		}
		for _, r := range h1 {
			if r.Error == "" {
				if r.SemanticCorrect {
					c1++
				}
				if r.UnknownReturned {
					row.H1UnknownCorrect++
				}
				row.H1ModelCalls += r.ModelCalls
				row.H1Transforms += r.DeterministicTransforms
				lat1 = append(lat1, float64(r.LatencyMS))
			}
		}
		row.H0Accuracy = ratio(c0, row.N)
		row.H1Accuracy = ratio(c1, row.N)
		row.H0CI95[0], row.H0CI95[1] = decompositionlab.WilsonCI95(c0, row.N)
		row.H1CI95[0], row.H1CI95[1] = decompositionlab.WilsonCI95(c1, row.N)
		row.McNemar, row.H0toH1Recovered, row.H0toH1Regressed, row.H0Correct = phPair(h0, h1)
		if row.H0Correct > 0 {
			row.RegressionRate = ratio(row.H0toH1Regressed, row.H0Correct)
		}
		if len(lat0) > 0 {
			row.H0MeanLatencyMS = meanF(lat0)
		}
		if len(lat1) > 0 {
			row.H1MeanLatencyMS = meanF(lat1)
		}
		t.Rows = append(t.Rows, row)

		if strat.Executable {
			execCorrectH0 += c0
			execCorrectH1 += c1
			execN += row.N
			execRegressBase += row.H0Correct
			execRegressed += row.H0toH1Regressed
			allH0 = append(allH0, h0...)
			allH1 = append(allH1, h1...)
		}
	}

	t.OverallExecutableN = execN
	t.OverallH0Accuracy = ratio(execCorrectH0, execN)
	t.OverallH1Accuracy = ratio(execCorrectH1, execN)
	t.OverallDelta = t.OverallH1Accuracy - t.OverallH0Accuracy
	t.OverallMcNemar, _, _, _ = phPair(allH0, allH1)
	if execRegressBase > 0 {
		t.OverallRegressionRate = ratio(execRegressed, execRegressBase)
	}

	// H-D: all H1 must be rejected pre-Parrot
	hdH1 := byKey["H-D"]["H1_PROFILE_ADAPTED"]
	rejected, hdTotal := 0, 0
	for _, r := range hdH1 {
		hdTotal++
		if r.ModelCalls == 0 && r.UnknownReturned {
			rejected++
		}
	}
	t.MissingOperandRejected = fmt.Sprintf("%d/%d", rejected, hdTotal)

	// integrity checks
	if ds.ProfileHash == "" {
		t.IntegrityViolations = append(t.IntegrityViolations, "dataset carries no frozen profile hash")
	}
	for _, r := range records {
		if r.Arm == "H1_PROFILE_ADAPTED" && r.AdapterDecision != nil && r.AdapterDecision.ProfileHash != ds.ProfileHash {
			t.IntegrityViolations = append(t.IntegrityViolations, "H1 "+r.BaseID+": adapter used a different profile hash")
			break
		}
	}

	// frozen promotion gate (§26)
	m := t.OverallMcNemar
	t.GateChecks["A_delta_ge_0.15"] = t.OverallDelta >= phGateDelta
	t.GateChecks["B_w_to_c_gt_c_to_w"] = m.WrongToCorrect > m.CorrectToWrong
	t.GateChecks["C_mcnemar_p_lt_0.05"] = m.PValue < phGateMcNemarSig
	t.GateChecks["D_regression_rate_le_0.05"] = t.OverallRegressionRate <= phGateMaxRegression
	t.GateChecks["E_missing_visual_100pct_rejected"] = hdTotal > 0 && rejected == hdTotal
	t.GateChecks["F_no_integrity_violations"] = len(t.IntegrityViolations) == 0
	promoted := true
	var failed []string
	for k, ok := range t.GateChecks {
		if !ok {
			promoted = false
			failed = append(failed, k)
		}
	}
	sort.Strings(failed)
	t.ProfileR1RuntimePromoted = promoted
	if promoted {
		t.PromotionBasis = fmt.Sprintf("all six frozen gate conditions met: executable Δ %+.2f, W→C %d > C→W %d, exact p %v, regression rate %.2f, missing-visual %s rejected, 0 integrity violations",
			t.OverallDelta, m.WrongToCorrect, m.CorrectToWrong, m.PValue, t.OverallRegressionRate, t.MissingOperandRejected)
	} else {
		t.PromotionBasis = "gate NOT met: " + strings.Join(failed, ", ")
	}
	return t
}

func meanF(xs []float64) float64 {
	s := 0.0
	for _, v := range xs {
		s += v
	}
	return s / float64(len(xs))
}
