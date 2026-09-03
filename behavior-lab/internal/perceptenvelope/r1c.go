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

	"tlaloc.local/behaviorlab/internal/parrotlab"
	"tlaloc.local/behaviorlab/internal/target"
)

// R1-C NUMERIC MORPHOLOGY — presentation policy, allocation, runner.
//
// Presentation is frozen INSIDE the R1-A1 / R1-B operating envelope and
// held identical for every condition; R1-C varies MORPHOLOGY only
// (protocol R1-C section 2).

// R1CLineHeightPx is the frozen R1-C containing-line height: the R1-B
// nominal high-reliability presentation point.
const R1CLineHeightPx = 32.0

// R1CContextLevel is the frozen R1-C context policy (R1-B context policy).
const R1CContextLevel = "A1C0_TARGET"

// R1CSeed is the frozen deterministic selection seed.
const R1CSeed = Seed

// R1C strata / provenance labels.
const (
	StratumLexical  = "LEXICAL_MORPHOLOGY"
	StratumLayout   = "PRESENTATION_LAYOUT"
	ProvReal        = "REAL_DOCUMENT"
	ProvRealSmallN  = "REAL_DOCUMENT_SMALL_N"
	ProvSynthetic   = "SYNTHETIC_REALISTIC"
	BandFull12      = "FULL_12"
	BandRealSmallN  = "REAL_DOCUMENT_SMALL_N"
	BandDescriptive = "DESCRIPTIVE_ONLY_PLUS_SYNTHETIC"
	BandUnsupported = "UNSUPPORTED_BY_AVAILABLE_CORPUS_PLUS_SYNTHETIC"
)

// r1cLexicalFamilies is the frozen ordered lexical morphology family list
// (protocol section 7, C0..C9).
var r1cLexicalFamilies = []string{
	FamSingleDigit, FamMultiDigit, FamThousands, FamDecimal, FamPercentage,
	FamSigned, FamRange, FamScientific, FamCoordTuple, FamEquation,
}

// r1cLayoutFamilies is the frozen PRESENTATION_LAYOUT_STRATUM family list
// (TABLE_CELL only; PAGE_OR_SECTION_NUMBER had zero real evidence).
var r1cLayoutFamilies = []string{FamTableCell}

// r1cSyntheticFamilies get a 12-case SYNTHETIC_REALISTIC stratum
// (insufficient real-document evidence).
var r1cSyntheticFamilies = map[string]bool{
	FamScientific: true, FamEquation: true, FamCoordTuple: true,
}

// r1cSyntheticTargets are the frozen synthetic target strings, declared in
// code BEFORE any R1-C model output existed.
var r1cSyntheticTargets = map[string][]string{
	FamScientific: {"3.14e-4", "1e6", "2.5e3", "6.02e23", "1.6e-19", "9e9", "4.2e-2", "5e0", "7.1e5", "3e-7", "8.8e4", "1.0e-3"},
	FamEquation:   {"x = 128", "x = 64", "x = 256", "x = 512", "x = 10", "x = 42", "x = 7", "x = 1000", "x = 3", "x = 96", "x = 24", "x = 300"},
	FamCoordTuple: {"(512, 256)", "[3, 7]", "(1, 1)", "(0, 0)", "(28, 28)", "(224, 224)", "[10, 20, 30]", "(2, 3)", "(100, 200)", "[1, 2]", "(7, 7, 7)", "(16, 16)"},
}

// R1CBase is one allocated R1-C stimulus.
type R1CBase struct {
	BaseID           string          `json:"base_id"`
	Family           string          `json:"family"`
	Provenance       string          `json:"provenance"`
	Stratum          string          `json:"stratum"`
	DigitLenSubgroup string          `json:"digit_length_subgroup,omitempty"`
	GoldSurface      string          `json:"gold_surface"`
	RankKey          string          `json:"rank_key"`
	Candidate        *MorphCandidate `json:"candidate,omitempty"`
	CharOffsetStart  int             `json:"char_offset_start,omitempty"`
	CharOffsetEnd    int             `json:"char_offset_end,omitempty"`
	SyntheticTarget  string          `json:"synthetic_target,omitempty"`
}

// R1CFamilyAllocation is the frozen per-family selection.
type R1CFamilyAllocation struct {
	Family         string    `json:"family"`
	Stratum        string    `json:"stratum"`
	RealAvailable  int       `json:"real_document_n_available"`
	Band           string    `json:"band"`
	RealBases      []R1CBase `json:"real_bases"`
	SyntheticBases []R1CBase `json:"synthetic_bases"`
}

// R1CAllocation is the full frozen R1-C dataset allocation.
type R1CAllocation struct {
	Schema       string                `json:"schema"`
	ExperimentID string                `json:"experiment_id"`
	Seed         string                `json:"seed"`
	RankRule     string                `json:"rank_rule"`
	LineHeightPx float64               `json:"line_height_px"`
	ContextLevel string                `json:"context_level"`
	CanvasPx     int                   `json:"canvas_px"`
	Families     []R1CFamilyAllocation `json:"families"`
}

const r1cAllocSchema = "tlaloc.parrot-perceptual-envelope-r1.r1c-allocation.r1"

func rankKeyR1C(family, candidateID string) string {
	sum := sha256.Sum256([]byte(R1CSeed + "|" + family + "|" + candidateID))
	return hex.EncodeToString(sum[:])
}

func runeOffset(s string, byteIdx int) int {
	return len([]rune(s[:byteIdx]))
}

// digitLen returns the number of digit characters in a token.
func digitLen(tok string) int {
	n := 0
	for _, r := range tok {
		if r >= '0' && r <= '9' {
			n++
		}
	}
	return n
}

// AllocateR1C performs the frozen deterministic per-family selection.
func AllocateR1C(pool MorphologyPool, exclude map[string]struct{}) R1CAllocation {
	alloc := R1CAllocation{
		Schema: r1cAllocSchema, ExperimentID: ExperimentID, Seed: R1CSeed,
		RankRule:     "sha256(seed || family || candidate_id) ascending, <=2 candidates/page",
		LineHeightPx: R1CLineHeightPx, ContextLevel: R1CContextLevel, CanvasPx: CanvasPx,
	}
	fams := append(append([]string{}, r1cLexicalFamilies...), r1cLayoutFamilies...)
	for _, fam := range fams {
		stratum := StratumLexical
		if fam == FamTableCell {
			stratum = StratumLayout
		}
		fa := R1CFamilyAllocation{Family: fam, Stratum: stratum, RealAvailable: pool.FamilyAvailable[fam]}

		cands := append([]MorphCandidate(nil), pool.Families[fam]...)
		filtered := cands[:0]
		for _, c := range cands {
			if _, skip := exclude[c.CandidateID]; skip {
				continue
			}
			filtered = append(filtered, c)
		}
		sort.Slice(filtered, func(i, j int) bool {
			return rankKeyR1C(fam, filtered[i].CandidateID) < rankKeyR1C(fam, filtered[j].CandidateID)
		})

		realN := len(filtered)
		switch {
		case realN >= 12:
			fa.Band = BandFull12
		case realN >= 6:
			fa.Band = BandRealSmallN
		case realN >= 1:
			fa.Band = BandDescriptive
		default:
			fa.Band = BandUnsupported
		}

		take := 12
		if fam == FamMultiDigit {
			fa.RealBases = selectMultiDigitBalanced(fam, stratum, filtered)
		} else {
			fa.RealBases = selectRealBases(fam, stratum, filtered, take)
		}
		for i := range fa.RealBases {
			if realN < 12 && realN >= 6 {
				fa.RealBases[i].Provenance = ProvRealSmallN
			}
		}

		if r1cSyntheticFamilies[fam] {
			for i, targ := range r1cSyntheticTargets[fam] {
				id := fmt.Sprintf("%s-syn-%02d", strings.ToLower(fam), i+1)
				fa.SyntheticBases = append(fa.SyntheticBases, R1CBase{
					BaseID: id, Family: fam, Provenance: ProvSynthetic, Stratum: stratum,
					GoldSurface: syntheticGold(fam, targ), SyntheticTarget: targ,
					RankKey: rankKeyR1C(fam, id),
				})
			}
		}
		alloc.Families = append(alloc.Families, fa)
	}
	return alloc
}

// syntheticGold returns the gold SURFACE string for a synthetic target
// (for EQUATION the operand, else the whole string).
func syntheticGold(family, target string) string {
	if family == FamEquation {
		if mm := reEquationOp.FindStringSubmatch(target); mm != nil {
			return mm[1]
		}
	}
	return target
}

func selectRealBases(family, stratum string, ranked []MorphCandidate, take int) []R1CBase {
	perPage := map[int]int{}
	var out []R1CBase
	for i := range ranked {
		c := ranked[i]
		if perPage[c.Page] >= 2 {
			continue
		}
		perPage[c.Page]++
		out = append(out, mkRealBase(family, stratum, c))
		if len(out) >= take {
			break
		}
	}
	return out
}

func selectMultiDigitBalanced(family, stratum string, ranked []MorphCandidate) []R1CBase {
	perPage := map[int]int{}
	quota := map[int]int{2: 4, 3: 4, 4: 4}
	got := map[int]int{}
	var out []R1CBase
	// first pass: fill quotas
	for i := range ranked {
		c := ranked[i]
		dl := digitLen(c.Token)
		if quota[dl] == 0 || got[dl] >= quota[dl] || perPage[c.Page] >= 2 {
			continue
		}
		perPage[c.Page]++
		got[dl]++
		out = append(out, mkRealBase(family, stratum, c))
	}
	// second pass: top up to 12 with any remaining eligible
	for i := 0; i < len(ranked) && len(out) < 12; i++ {
		c := ranked[i]
		if perPage[c.Page] >= 2 {
			continue
		}
		already := false
		for _, b := range out {
			if b.Candidate != nil && b.Candidate.CandidateID == c.CandidateID {
				already = true
				break
			}
		}
		if already {
			continue
		}
		perPage[c.Page]++
		out = append(out, mkRealBase(family, stratum, c))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RankKey < out[j].RankKey })
	return out
}

func mkRealBase(family, stratum string, c MorphCandidate) R1CBase {
	cc := c
	b := R1CBase{
		BaseID: fmt.Sprintf("%s-%s", strings.ToLower(family), c.CandidateID[:10]),
		Family: family, Provenance: ProvReal, Stratum: stratum,
		GoldSurface: c.Token, RankKey: rankKeyR1C(family, c.CandidateID),
		Candidate: &cc,
	}
	if idx := strings.Index(c.LineText, c.Token); idx >= 0 {
		b.CharOffsetStart = runeOffset(c.LineText, idx)
		b.CharOffsetEnd = b.CharOffsetStart + len([]rune(c.Token))
	} else {
		b.CharOffsetStart, b.CharOffsetEnd = 0, len([]rune(c.Token))
	}
	if family == FamMultiDigit {
		b.DigitLenSubgroup = fmt.Sprintf("%d", digitLen(c.Token))
	}
	return b
}

// morphToBase adapts a real R1-C base to the R1-B renderer's Base type.
func (b R1CBase) morphToBase() Base {
	c := b.Candidate
	return Base{
		BaseID: b.BaseID, Stage: "R1-C",
		Candidate: Candidate{
			CandidateID:      c.CandidateID,
			Page:             c.Page,
			PageWidth:        c.PageWidth,
			PageHeight:       c.PageHeight,
			TargetToken:      c.Token,
			NormalizedTarget: c.Token,
			CharOffsetStart:  b.CharOffsetStart,
			CharOffsetEnd:    b.CharOffsetEnd,
			Line: LineRef{
				RegionID: c.RegionID, Kind: c.RegionKind, Text: c.LineText, BBox: c.LineBBox,
			},
		},
	}
}

// R1CRecord is one (base) result.
type R1CRecord struct {
	BaseID           string   `json:"base_id"`
	Family           string   `json:"family"`
	Provenance       string   `json:"provenance"`
	Stratum          string   `json:"stratum"`
	DigitLenSubgroup string   `json:"digit_length_subgroup,omitempty"`
	Page             int      `json:"page,omitempty"`
	GoldSurface      string   `json:"gold_surface"`
	RawText          string   `json:"raw_text"`
	Score            R1CScore `json:"score"`
	LatencyMS        int64    `json:"latency_ms"`
	PromptTokens     int      `json:"prompt_tokens"`
	CompletionToks   int      `json:"completion_tokens"`
	CropPath         string   `json:"crop_path"`
	Error            string   `json:"error,omitempty"`
}

// RunR1C executes every frozen R1-C base once (protocol section 17).
func RunR1C(ctx context.Context, cfg RunConfig, alloc R1CAllocation, bank *GlyphBank) ([]R1CRecord, error) {
	provider, err := parrotlab.NewPDFMemoryProvider(cfg.StoreDir, cfg.PDFPath)
	if err != nil {
		return nil, fmt.Errorf("page provider: %w", err)
	}
	baseURL := strings.TrimRight(cfg.Endpoint, "/")
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL += "/v1"
	}
	client := target.OpenAICompat{BaseURL: baseURL, Model: cfg.Model, Temperature: cfg.Temperature, MaxTokens: cfg.MaxTokens}

	cropDir := filepath.Join(cfg.RunDir, "crops")
	rawDir := filepath.Join(cfg.RunDir, "raw")
	for _, d := range []string{cropDir, rawDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}

	pageCache := map[int][]byte{}
	getPage := func(p int) ([]byte, error) {
		if b, ok := pageCache[p]; ok {
			return b, nil
		}
		b, e := provider.RenderPNG(p)
		if e == nil {
			pageCache[p] = b
		}
		return b, e
	}

	var out []R1CRecord
	for _, fa := range alloc.Families {
		bases := append(append([]R1CBase{}, fa.RealBases...), fa.SyntheticBases...)
		for _, base := range bases {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			default:
			}
			rec := R1CRecord{
				BaseID: base.BaseID, Family: base.Family, Provenance: base.Provenance,
				Stratum: base.Stratum, DigitLenSubgroup: base.DigitLenSubgroup,
				GoldSurface: base.GoldSurface,
			}
			cropPath := filepath.Join(cropDir, base.BaseID+".png")
			var imgBytes []byte
			if base.Provenance == ProvSynthetic {
				img, _, rerr := RenderSyntheticNumber(bank, base.SyntheticTarget)
				if rerr != nil {
					rec.Error = rerr.Error()
					out = append(out, rec)
					writeRawR1C(rawDir, rec)
					continue
				}
				if werr := writeRGBAPNG(cropPath, img); werr != nil {
					return nil, werr
				}
			} else {
				rec.Page = base.Candidate.Page
				geo, gerr := DeriveR1BGeometry(cfg.StoreDir, base.morphToBase())
				if gerr != nil {
					rec.Error = gerr.Error()
					out = append(out, rec)
					writeRawR1C(rawDir, rec)
					continue
				}
				var cond R1BCondGeom
				for _, cg := range geo.Conditions {
					if cg.NominalLinePx == R1CLineHeightPx {
						cond = cg
					}
				}
				pagePNG, perr := getPage(base.Candidate.Page)
				if perr != nil {
					return nil, fmt.Errorf("render page %d: %w", base.Candidate.Page, perr)
				}
				img, _, rerr := RenderR1BScale(pagePNG, base.morphToBase(), geo, cond)
				if rerr != nil {
					rec.Error = rerr.Error()
					out = append(out, rec)
					writeRawR1C(rawDir, rec)
					continue
				}
				if werr := writeRGBAPNG(cropPath, img); werr != nil {
					return nil, werr
				}
			}
			rec.CropPath = cropPath
			b, ierr := os.ReadFile(cropPath)
			if ierr != nil {
				rec.Error = ierr.Error()
				out = append(out, rec)
				writeRawR1C(rawDir, rec)
				continue
			}
			imgBytes = b
			start := time.Now()
			result, cerr := client.CompletePerception(ctx, target.PerceptionInput{
				Question: FrozenInstruction, Image: imgBytes, MediaType: "image/png",
			})
			rec.LatencyMS = time.Since(start).Milliseconds()
			if cerr != nil {
				rec.Error = cerr.Error()
				out = append(out, rec)
				writeRawR1C(rawDir, rec)
				continue
			}
			rec.RawText = result.Content
			rec.PromptTokens = result.PromptTokensReported
			rec.CompletionToks = result.CompletionTokensReported
			rec.Score = ScoreR1C(base.Family, result.Content, base.GoldSurface)
			out = append(out, rec)
			writeRawR1C(rawDir, rec)
		}
	}
	return out, nil
}

func writeRawR1C(rawDir string, rec R1CRecord) {
	body, _ := json.MarshalIndent(rec, "", "  ")
	_ = os.WriteFile(filepath.Join(rawDir, rec.BaseID+".json"), body, 0o644)
}
