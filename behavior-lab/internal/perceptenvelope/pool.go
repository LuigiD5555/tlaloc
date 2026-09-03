// Package perceptenvelope implements the Parrot Perceptual Envelope R1
// experiment (parrot-perceptual-envelope-r1). It characterises LFM2-VL
// 1.6B's CONTROLLED_ATOMIC_VISUAL_READING competence for EXTRACT_NUMBER
// under varying visual context (R1-A) and scale (R1-B).
//
// Every stimulus is selected deterministically from the reconstructed
// pdfmemory store BEFORE any model output exists (protocol addendum
// R1_PROTOCOL_ADDENDUM_00). No function in this package reads a model
// output, a scorer verdict, or an expected answer when building or
// allocating the pool. Ground truth is the store's digital text layer.
package perceptenvelope

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

// SelectionAlgorithmVersion is bumped whenever the deterministic scan /
// filter / allocation logic changes in a way that could move which
// candidates are selected. Frozen artifacts record it.
const SelectionAlgorithmVersion = "r1.0.0"

// Seed is the fixed partition seed (protocol section 6).
const Seed = "20260903"

// primaryTarget matches an admissible R1-A/R1-B primary numeric target:
// a plain 2-4 digit integer, no separators / sign / decimal.
var primaryTarget = regexp.MustCompile(`^[0-9]{2,4}$`)

// anyDigit detects a digit-bearing token (used for the one-number-per-line
// admission rule).
var anyDigit = regexp.MustCompile(`[0-9]`)

// runningHeaderCaps is an ALL-CAPS running header with an embedded page
// number, e.g. "LINEAR NEURAL NETWORKS 173".
var runningHeaderCaps = regexp.MustCompile(`^[A-Z][A-Z .,':;&/-]{6,}\s+[0-9]{1,4}$`)

// rawIntegerToken is the admissible verbatim shape of a primary target
// token: exactly 2-4 bare digits. Anything with a bracket, comma, slash,
// colon, trailing period (list/exercise/section numbers, sentence-final
// numbers) or interior punctuation (citations, dates in "(2022)", ratios,
// ranges) is rejected before it can become a candidate.
var rawIntegerToken = regexp.MustCompile(`^[0-9]{2,4}$`)

// bibliographyCue marks a reference / citation line.
var bibliographyCue = regexp.MustCompile(`(?i)\bet al\b|\bpp\.|\bvol\.|\beds?\.|arxiv|\bdoi\b|https?://|\((?:1[6-9]|20)[0-9]{2}\)`)

// minSentenceFields is the minimum whitespace-token count for a line to
// count as embedded prose (target + >= 3 other words). Bare page numbers,
// section numbers and equation tags have fewer.
const minSentenceFields = 4

// PageDims is a store page's intrinsic coordinate space.
type PageDims struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// LineRef locates the target's containing layout line and its neighbours in
// reading order.
type LineRef struct {
	RegionID     string            `json:"region_id"`
	Kind         string            `json:"kind"`
	ReadingOrder int               `json:"reading_order"`
	Index        int               `json:"region_index"`
	Text         string            `json:"text"`
	BBox         canonicaldoc.BBox `json:"bbox"`
	FontSize     int               `json:"font_size"`
}

// Candidate is one deterministically discovered numeric-reading target.
type Candidate struct {
	CandidateID string `json:"candidate_id"`

	Page       int     `json:"page"`
	PageWidth  float64 `json:"page_width"`
	PageHeight float64 `json:"page_height"`

	TargetToken      string `json:"target_token"`      // verbatim from the line
	NormalizedTarget string `json:"normalized_target"` // punctuation-stripped
	CharOffsetStart  int    `json:"char_offset_start"` // rune offset in Line.Text
	CharOffsetEnd    int    `json:"char_offset_end"`

	Line LineRef `json:"line"`

	// TokenBBoxStore is the frozen proportional token-box estimate in store
	// coordinates, already padded by the fixed policy and clamped to the
	// page. It is the cue geometry for every context/scale variant.
	TokenBBoxStore canonicaldoc.BBox `json:"token_bbox_store"`

	NeighborRegionIDs []string `json:"neighbor_region_ids"` // reading-order [-3..+3] excl. target

	Provenance CandidateProvenance `json:"provenance"`
}

// CandidateProvenance records exactly how the candidate was derived.
type CandidateProvenance struct {
	SourcePDFSHA256  string `json:"source_pdf_sha256"`
	StoreRootSHA256  string `json:"store_root_sha256"`
	StoreCarrierID   string `json:"store_carrier_id"`
	LayoutPath       string `json:"layout_path"`
	AlgorithmVersion string `json:"algorithm_version"`
	PaddingPolicy    string `json:"padding_policy"`
	TokenBoxMethod   string `json:"token_box_method"`
}

// Rejection is one candidate-shaped location that failed an admission rule.
type Rejection struct {
	Page     int    `json:"page"`
	RegionID string `json:"region_id"`
	Token    string `json:"token"`
	Reason   string `json:"reason"`
}

// SourcePool is the frozen global candidate pool (SOURCE_POOL_R1.json).
type SourcePool struct {
	Schema            string         `json:"schema"`
	ExperimentID      string         `json:"experiment_id"`
	SourcePDFSHA256   string         `json:"source_pdf_sha256"`
	StoreRootSHA256   string         `json:"store_root_sha256"`
	StoreDir          string         `json:"store_dir"`
	StoreCarrierID    string         `json:"store_carrier_id"`
	AlgorithmVersion  string         `json:"selection_algorithm_version"`
	Seed              string         `json:"seed"`
	PaddingPolicy     string         `json:"padding_policy"`
	Filters           []string       `json:"candidate_filters"`
	PagesScanned      int            `json:"pages_scanned"`
	RegionsScanned    int            `json:"regions_scanned"`
	DigitTokensSeen   int            `json:"digit_tokens_seen"`
	PrimaryCandidates int            `json:"primary_candidate_count"`
	RejectionCounts   map[string]int `json:"rejection_counts"`
	Rejections        []Rejection    `json:"rejections"`
	Candidates        []Candidate    `json:"candidates"`
}

const poolSchema = "tlaloc.parrot-perceptual-envelope-r1.source-pool.r1"

// paddingPolicy is the single frozen cue-padding rule (protocol
// "TARGET CUE GEOMETRY" steps 3-4): the proportional token interval is
// taken over the full line height, then expanded by 0.5 * line-height on
// every side, then clamped to the page.
const paddingPolicy = "pad = 0.5 * line_bbox_height on each side (x and y); token x-interval = proportional rune-offset split of line bbox; y-interval = full line bbox; then clamp to page bounds"

const tokenBoxMethod = "PROPORTIONAL_RUNE_OFFSET_SPLIT_OF_LINE_BBOX (no per-word bbox in store; estimate only, not a Tlaloc perception capability)"

// filterList is the frozen, ordered set of admission filters, recorded
// verbatim in the artifact.
var filterList = []string{
	"target token matches ^[0-9]{2,4}$ after stripping surrounding .,;:()[]%",
	"containing line has EXACTLY ONE digit-bearing token (any kind) — reject decimals/percentages/ranges/section numbers/equation terms/citations/dates/other integers/alphanumeric ids elsewhere on the line",
	"verbatim token matches ^[0-9]{2,4}$ exactly — reject bracketed/comma/slash/colon/trailing-period tokens such as (2022)., 2014), 3/4, 10:1, '15.'",
	"reject 4-digit tokens in [1500,2099] as year/date-like",
	"containing line has >= 4 whitespace tokens (embedded prose) — reject bare page/section numbers and equation tags",
	"reject number-leading lines (fields[0] == target) — TOC/section headings, numbered list items, wrapped edge fragments",
	"reject bibliography/citation lines (et al, pp., vol., eds., arXiv, doi, http, (YYYY))",
	"region kind in {text_line, list_item} — reject table_cell/heading/subheading/figure/equation_or_code",
	"line font_size >= 10 (margin line numbers are ~9)",
	"line not a running-header page number (ALLCAPS + trailing number)",
	"line not in page margin: line bbox fully inside [0.10w,0.90w] x [0.09h,0.91h]",
	"line width >= 0.30w (prose line, not a stray short numeric line)",
	"target token located uniquely by substring in the line text (exactly one occurrence)",
	"estimated token bbox non-empty and inside the line bbox in x before padding",
	"padded token bbox not clipped by page bounds (fully strictly inside the page)",
	"line geometry well-formed: 0 <= x1 < x2 <= w, 0 <= y1 < y2 <= h",
	"cue not implausibly large: estimated token width <= 0.85 * line width when line has > 2 whitespace tokens",
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func mustAtoi(s string) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return -1
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

// stripEdgePunct removes surrounding punctuation used to test primaryTarget.
func stripEdgePunct(tok string) string {
	return strings.Trim(tok, ".,;:()[]%\"'")
}

// ScanSourcePool walks every page of the store and returns the frozen
// primary candidate pool plus full rejection accounting. Deterministic:
// same store bytes -> identical SourcePool.
func ScanSourcePool(storeDir string) (SourcePool, error) {
	manifest, _, err := pdfmemory.Load(storeDir)
	if err != nil {
		return SourcePool{}, fmt.Errorf("load store %s: %w", storeDir, err)
	}
	srcSHA := manifest.SourceSHA256
	if srcSHA == "" && len(manifest.Documents) > 0 {
		srcSHA = manifest.Documents[0].SourceSHA256
	}

	pool := SourcePool{
		Schema:           poolSchema,
		ExperimentID:     ExperimentID,
		SourcePDFSHA256:  srcSHA,
		StoreRootSHA256:  manifest.StoreRootSHA256,
		StoreDir:         storeDir,
		StoreCarrierID:   manifest.CarrierID,
		AlgorithmVersion: SelectionAlgorithmVersion,
		Seed:             Seed,
		PaddingPolicy:    paddingPolicy,
		Filters:          filterList,
		RejectionCounts:  map[string]int{},
	}

	pages := append([]pdfmemory.PageRef(nil), manifest.Pages...)
	sort.Slice(pages, func(i, j int) bool { return pages[i].Number < pages[j].Number })

	for _, pref := range pages {
		if strings.TrimSpace(pref.LayoutPath) == "" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(pref.LayoutPath)))
		if err != nil {
			return SourcePool{}, fmt.Errorf("read layout %s: %w", pref.LayoutPath, err)
		}
		var page canonicaldoc.Page
		if err := json.Unmarshal(body, &page); err != nil {
			return SourcePool{}, fmt.Errorf("decode layout %s: %w", pref.LayoutPath, err)
		}
		pool.PagesScanned++
		pool.RegionsScanned += len(page.Regions)
		scanPage(&pool, pref, page, srcSHA, manifest)
	}

	sort.Slice(pool.Candidates, func(i, j int) bool {
		return pool.Candidates[i].CandidateID < pool.Candidates[j].CandidateID
	})
	sort.Slice(pool.Rejections, func(i, j int) bool {
		if pool.Rejections[i].Page != pool.Rejections[j].Page {
			return pool.Rejections[i].Page < pool.Rejections[j].Page
		}
		if pool.Rejections[i].RegionID != pool.Rejections[j].RegionID {
			return pool.Rejections[i].RegionID < pool.Rejections[j].RegionID
		}
		return pool.Rejections[i].Reason < pool.Rejections[j].Reason
	})
	pool.PrimaryCandidates = len(pool.Candidates)
	return pool, nil
}

func reject(pool *SourcePool, page int, regionID, token, reason string) {
	pool.RejectionCounts[reason]++
	pool.Rejections = append(pool.Rejections, Rejection{Page: page, RegionID: regionID, Token: token, Reason: reason})
}

func scanPage(pool *SourcePool, pref pdfmemory.PageRef, page canonicaldoc.Page, srcSHA string, manifest pdfmemory.Manifest) {
	w, h := page.Width, page.Height
	for idx, region := range page.Regions {
		text := strings.TrimSpace(region.Text)
		if text == "" {
			continue
		}
		fields := strings.Fields(text)
		// count digit-bearing tokens; find the single primary target
		digitTokens := 0
		var primaryTok string
		primaryCount := 0
		for _, f := range fields {
			if anyDigit.MatchString(f) {
				digitTokens++
				pool.DigitTokensSeen++
				if primaryTarget.MatchString(stripEdgePunct(f)) {
					primaryTok = f
					primaryCount++
				}
			}
		}
		if primaryCount == 0 {
			continue
		}
		norm := stripEdgePunct(primaryTok)

		if region.Kind != "text_line" && region.Kind != "list_item" {
			reject(pool, page.Number, region.ID, primaryTok, "region_kind_excluded")
			continue
		}
		if digitTokens != 1 || primaryCount != 1 {
			reject(pool, page.Number, region.ID, primaryTok, "line_has_other_digit_token")
			continue
		}
		if !rawIntegerToken.MatchString(primaryTok) {
			reject(pool, page.Number, region.ID, primaryTok, "bracketed_or_punctuated_token")
			continue
		}
		if len(norm) == 4 {
			if v := mustAtoi(norm); v >= 1500 && v <= 2099 {
				reject(pool, page.Number, region.ID, primaryTok, "year_or_date_token")
				continue
			}
		}
		if len(fields) < minSentenceFields {
			reject(pool, page.Number, region.ID, primaryTok, "bare_or_short_number_line")
			continue
		}
		if fields[0] == primaryTok {
			// number-leading line: TOC/section heading, numbered list item,
			// or a wrapped fragment where the operand sits at the very edge.
			reject(pool, page.Number, region.ID, primaryTok, "number_leading_line")
			continue
		}
		if bibliographyCue.MatchString(text) {
			reject(pool, page.Number, region.ID, primaryTok, "bibliography_or_citation_line")
			continue
		}
		if region.FontSize > 0 && region.FontSize < 10 {
			reject(pool, page.Number, region.ID, primaryTok, "font_size_below_body")
			continue
		}
		if runningHeaderCaps.MatchString(text) {
			reject(pool, page.Number, region.ID, primaryTok, "running_header_page_number")
			continue
		}
		b := region.BBox
		if !(b.X1 >= 0 && b.X1 < b.X2 && b.X2 <= w && b.Y1 >= 0 && b.Y1 < b.Y2 && b.Y2 <= h) {
			reject(pool, page.Number, region.ID, primaryTok, "line_geometry_malformed")
			continue
		}
		if !(b.X1 >= 0.10*w && b.X2 <= 0.90*w && b.Y1 >= 0.09*h && b.Y2 <= 0.91*h) {
			reject(pool, page.Number, region.ID, primaryTok, "line_in_page_margin")
			continue
		}
		if (b.X2 - b.X1) < 0.30*w {
			reject(pool, page.Number, region.ID, primaryTok, "line_too_narrow_for_prose")
			continue
		}

		// unique substring location of the primary token (use the
		// punctuation-trimmed form's position via the raw token).
		runes := []rune(text)
		occ := strings.Count(text, primaryTok)
		if occ != 1 {
			reject(pool, page.Number, region.ID, primaryTok, "token_offset_not_unique")
			continue
		}
		byteStart := strings.Index(text, primaryTok)
		startRune := len([]rune(text[:byteStart]))
		endRune := startRune + len([]rune(primaryTok))
		total := len(runes)
		if total == 0 || startRune < 0 || endRune > total || startRune >= endRune {
			reject(pool, page.Number, region.ID, primaryTok, "token_offset_invalid")
			continue
		}

		lineW := b.X2 - b.X1
		estX1 := b.X1 + lineW*float64(startRune)/float64(total)
		estX2 := b.X1 + lineW*float64(endRune)/float64(total)
		if !(estX1 >= b.X1 && estX2 <= b.X2 && estX1 < estX2) {
			reject(pool, page.Number, region.ID, primaryTok, "estimated_token_box_outside_line")
			continue
		}
		if len(fields) > 2 && (estX2-estX1) > 0.85*lineW {
			reject(pool, page.Number, region.ID, primaryTok, "cue_covers_implausible_line_fraction")
			continue
		}

		pad := 0.5 * (b.Y2 - b.Y1)
		tb := canonicaldoc.BBox{
			X1: estX1 - pad, Y1: b.Y1 - pad,
			X2: estX2 + pad, Y2: b.Y2 + pad,
		}
		if !(tb.X1 > 0 && tb.Y1 > 0 && tb.X2 < w && tb.Y2 < h && tb.X1 < tb.X2 && tb.Y1 < tb.Y2) {
			reject(pool, page.Number, region.ID, primaryTok, "padded_token_box_clipped_by_page")
			continue
		}

		var neighbors []string
		for off := -3; off <= 3; off++ {
			if off == 0 {
				continue
			}
			ni := idx + off
			if ni < 0 || ni >= len(page.Regions) {
				continue
			}
			neighbors = append(neighbors, page.Regions[ni].ID)
		}

		idBytes := []byte(strings.Join([]string{
			srcSHA,
			fmt.Sprintf("p%d", page.Number),
			region.ID,
			fmt.Sprintf("off%d-%d", startRune, endRune),
			norm,
			fmt.Sprintf("%.2f,%.2f,%.2f,%.2f", tb.X1, tb.Y1, tb.X2, tb.Y2),
		}, "|"))
		cid := sha256Hex(idBytes)[:32]

		pool.Candidates = append(pool.Candidates, Candidate{
			CandidateID:      cid,
			Page:             page.Number,
			PageWidth:        w,
			PageHeight:       h,
			TargetToken:      primaryTok,
			NormalizedTarget: norm,
			CharOffsetStart:  startRune,
			CharOffsetEnd:    endRune,
			Line: LineRef{
				RegionID: region.ID, Kind: region.Kind, ReadingOrder: region.ReadingOrder,
				Index: idx, Text: text, BBox: b, FontSize: region.FontSize,
			},
			TokenBBoxStore:    tb,
			NeighborRegionIDs: neighbors,
			Provenance: CandidateProvenance{
				SourcePDFSHA256:  srcSHA,
				StoreRootSHA256:  manifest.StoreRootSHA256,
				StoreCarrierID:   manifest.CarrierID,
				LayoutPath:       pref.LayoutPath,
				AlgorithmVersion: SelectionAlgorithmVersion,
				PaddingPolicy:    paddingPolicy,
				TokenBoxMethod:   tokenBoxMethod,
			},
		})
	}
}
