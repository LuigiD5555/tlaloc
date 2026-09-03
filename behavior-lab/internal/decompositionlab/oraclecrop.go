package decompositionlab

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

// ORACLE CROP DERIVATION (T0-B0, protocol addendum sections 3, 5, 9-10).
//
// C1/C2/C3 are ORACLE OPERAND LOCALIZATION interventions: they remove
// spatial search and ask whether Parrot can perform the remaining atomic
// operation once the operand is isolated. The operand geometry is derived
// deterministically from the reconstructed pdfmemory store using ONLY the
// frozen P0 EvidenceRefs (text spans + page). It never reads an expected
// answer, a scorer label, or C0 correctness — DeriveOracleGeometry's inputs
// make that structurally impossible, and DeriveOracleCrop keeps the
// answer-aware sufficiency check strictly separate from the frozen bbox.

// OracleContextBlocks is the fixed, predeclared context-padding policy
// (protocol addendum section 3, step 4): the evidence bounding union is
// expanded by exactly this many layout regions (text lines / blocks) on
// each side in reading order. It is identical for every task family and was
// chosen before any model output existed.
const OracleContextBlocks = 1

// minMatchedChars guards against a spurious one-token match producing a
// nonsense bbox; below it the case is INSUFFICIENT_PROVENANCE.
const minMatchedChars = 12

// wholePageExposureThreshold: at or above this crop/page area ratio the
// "crop" is effectively the whole page and the oracle localization did not
// actually reduce the operand.
const wholePageExposureThreshold = 0.98

// OracleCropSpec is one eligible case's frozen oracle operand crop.
type OracleCropSpec struct {
	BaseID           string            `json:"base_id"`
	Category         string            `json:"category"`
	ModelOpcode      string            `json:"model_opcode"`
	Page             int               `json:"page"`
	EvidenceRefsUsed []string          `json:"evidence_refs_used"`
	MatchedRegionIDs []string          `json:"matched_region_ids"`
	ContextBlocks    int               `json:"context_blocks"`
	SourceMethod     string            `json:"source_method"`
	StorePageWidth   float64           `json:"store_page_width"`
	StorePageHeight  float64           `json:"store_page_height"`
	ImagePageWidth   float64           `json:"image_page_width"`
	ImagePageHeight  float64           `json:"image_page_height"`
	ScaleX           float64           `json:"scale_x"`
	ScaleY           float64           `json:"scale_y"`
	StoreBBox        canonicaldoc.BBox `json:"store_bbox"`
	ImageBBox        canonicaldoc.BBox `json:"image_bbox"`
	VisualExposure   float64           `json:"visual_exposure_ratio_estimate"`
	CropReduction    float64           `json:"crop_reduction_factor_estimate"`
	MatchedChars     int               `json:"matched_chars"`
	Sufficiency      OracleSufficiency `json:"sufficiency"`
	Verdict          string            `json:"verdict"` // COMPATIBLE | INCOMPATIBLE | INSUFFICIENT_PROVENANCE
	Notes            string            `json:"notes,omitempty"`
}

// OracleSufficiency is the answer-aware validation of a derived crop. It is
// computed AFTER the geometry is frozen and is REPORT-ONLY: it never feeds
// back into StoreBBox/ImageBBox.
type OracleSufficiency struct {
	Requirement           string `json:"requirement"`
	OperandContentVisible bool   `json:"operand_content_visible"`
	NotWholePage          bool   `json:"not_whole_page"`
	Detail                string `json:"detail,omitempty"`
}

// Oracle crop verdicts.
const (
	OracleCompatible             = "COMPATIBLE"
	OracleIncompatible           = "INCOMPATIBLE"
	OracleInsufficientProvenance = "INSUFFICIENT_PROVENANCE"
)

var wsRun = regexp.MustCompile(`\s+`)
var hyphenBreak = regexp.MustCompile(`-\s+`)
var nonDigit = regexp.MustCompile(`[^0-9]`)

func normalizeText(s string) string {
	repl := strings.NewReplacer(
		"’", "'", "‘", "'", "“", `"`, "”", `"`,
		"–", "-", "—", "-", "­", "",
	)
	s = repl.Replace(s)
	s = hyphenBreak.ReplaceAllString(s, "")
	s = wsRun.ReplaceAllString(s, " ")
	return strings.ToLower(strings.TrimSpace(s))
}

// storePage loads one reconstructed-store page's canonical layout plus its
// own coordinate-space dimensions. It verifies the layout CID.
func storePage(storeDir string, page int) (canonicaldoc.Page, error) {
	manifest, _, err := pdfmemory.Load(storeDir)
	if err != nil {
		return canonicaldoc.Page{}, fmt.Errorf("load store %s: %w", storeDir, err)
	}
	want := fmt.Sprintf("ohf://%s/pages/%06d", manifest.CarrierID, page)
	for _, ref := range manifest.Pages {
		if ref.Address != want && ref.Number != page {
			continue
		}
		if ref.Address != want {
			continue
		}
		if strings.TrimSpace(ref.LayoutPath) == "" {
			return canonicaldoc.Page{}, fmt.Errorf("store page %d has no layout_path", page)
		}
		body, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(ref.LayoutPath)))
		if err != nil {
			return canonicaldoc.Page{}, err
		}
		if ref.LayoutCID != "" && sha256Bytes(body) != ref.LayoutCID {
			return canonicaldoc.Page{}, fmt.Errorf("store page %d layout CID mismatch", page)
		}
		var p canonicaldoc.Page
		if err := json.Unmarshal(body, &p); err != nil {
			return canonicaldoc.Page{}, err
		}
		return p, nil
	}
	return canonicaldoc.Page{}, fmt.Errorf("store has no page %d", page)
}

// regionRange is a half-open [lo, hi) window over a page's regions.
type regionRange struct{ lo, hi int }

// matchEvidenceRegions finds, for one evidence span, the minimal contiguous
// run of page regions (in reading order) whose concatenated text — with the
// same normalization applied to the JOIN (so hyphenated line breaks across
// regions rejoin) — contains it. If no window contains the full span it
// falls back to the window containing the span's 60-char head, then to the
// set of regions whose own text is a substring of the span. It returns the
// covered index range and the count of matched characters.
func matchEvidenceRegions(regionsRaw []string, span string) (regionRange, int, bool) {
	span = normalizeText(span)
	if span == "" {
		return regionRange{}, 0, false
	}
	try := func(needle string) (regionRange, bool) {
		best := regionRange{}
		found := false
		for lo := 0; lo < len(regionsRaw); lo++ {
			var b strings.Builder
			for hi := lo; hi < len(regionsRaw); hi++ {
				if hi > lo {
					b.WriteByte(' ')
				}
				b.WriteString(regionsRaw[hi])
				if strings.Contains(normalizeText(b.String()), needle) {
					if !found || (hi-lo) < (best.hi-best.lo-1) {
						best = regionRange{lo: lo, hi: hi + 1}
						found = true
					}
					break
				}
			}
		}
		return best, found
	}
	if rr, ok := try(span); ok {
		return rr, len(span), true
	}
	head := span
	if len(head) > 60 {
		head = head[:60]
	}
	if rr, ok := try(head); ok {
		return rr, len(head), true
	}
	// Last resort: regions whose own text is a non-trivial substring of the span.
	lo, hi, matched := -1, -1, 0
	for i, rt := range regionsRaw {
		nt := normalizeText(rt)
		if len(nt) >= 4 && strings.Contains(span, nt) {
			if lo < 0 {
				lo = i
			}
			hi = i + 1
			matched += len(nt)
		}
	}
	if lo >= 0 {
		return regionRange{lo: lo, hi: hi}, matched, true
	}
	return regionRange{}, 0, false
}

// DeriveOracleGeometry computes the frozen oracle crop geometry for one
// record. Its parameters carry ONLY the page number and the frozen
// EvidenceRefs, so it is structurally incapable of reading an expected
// answer. The result is deterministic in the store bytes + refs.
func DeriveOracleGeometry(storeDir string, page int, refs []EvidenceRef, imageW, imageH float64) (OracleCropSpec, error) {
	spec := OracleCropSpec{Page: page, ContextBlocks: OracleContextBlocks, ImagePageWidth: imageW, ImagePageHeight: imageH}
	p, err := storePage(storeDir, page)
	if err != nil {
		return spec, err
	}
	spec.StorePageWidth, spec.StorePageHeight = p.Width, p.Height
	if p.Width <= 0 || p.Height <= 0 || imageW <= 0 || imageH <= 0 {
		return spec, fmt.Errorf("page %d: non-positive page dimensions (store %gx%g image %gx%g)", page, p.Width, p.Height, imageW, imageH)
	}

	regionsRaw := make([]string, len(p.Regions))
	for i, r := range p.Regions {
		regionsRaw[i] = r.Text
	}

	covLo, covHi := -1, -1
	matchedChars := 0
	for _, ref := range refs {
		if strings.TrimSpace(ref.TextSpan) == "" {
			continue
		}
		rr, mc, ok := matchEvidenceRegions(regionsRaw, ref.TextSpan)
		if !ok {
			continue
		}
		spec.EvidenceRefsUsed = append(spec.EvidenceRefsUsed, ref.ID)
		matchedChars += mc
		if covLo < 0 || rr.lo < covLo {
			covLo = rr.lo
		}
		if rr.hi > covHi {
			covHi = rr.hi
		}
	}
	spec.MatchedChars = matchedChars
	if covLo < 0 || matchedChars < minMatchedChars {
		spec.Verdict = OracleInsufficientProvenance
		spec.Notes = "no reconstructed-store region run contained the frozen evidence span"
		return spec, nil
	}

	// One-context-block padding in reading order (fixed policy).
	lo := covLo - OracleContextBlocks
	if lo < 0 {
		lo = 0
	}
	hi := covHi + OracleContextBlocks
	if hi > len(p.Regions) {
		hi = len(p.Regions)
	}

	union := canonicaldoc.BBox{X1: math.Inf(1), Y1: math.Inf(1), X2: math.Inf(-1), Y2: math.Inf(-1)}
	for i := lo; i < hi; i++ {
		b := p.Regions[i].BBox
		if b.X2 <= b.X1 || b.Y2 <= b.Y1 {
			continue
		}
		union.X1 = math.Min(union.X1, b.X1)
		union.Y1 = math.Min(union.Y1, b.Y1)
		union.X2 = math.Max(union.X2, b.X2)
		union.Y2 = math.Max(union.Y2, b.Y2)
		if i >= covLo && i < covHi {
			spec.MatchedRegionIDs = append(spec.MatchedRegionIDs, p.Regions[i].ID)
		}
	}
	if math.IsInf(union.X1, 0) {
		return spec, fmt.Errorf("page %d: matched regions have no usable geometry", page)
	}
	union.X1 = clamp(union.X1, 0, p.Width)
	union.Y1 = clamp(union.Y1, 0, p.Height)
	union.X2 = clamp(union.X2, 0, p.Width)
	union.Y2 = clamp(union.Y2, 0, p.Height)
	spec.StoreBBox = union

	// Deterministic affine scale store -> image, outward rounding.
	sx := imageW / p.Width
	sy := imageH / p.Height
	spec.ScaleX, spec.ScaleY = sx, sy
	img := canonicaldoc.BBox{
		X1: math.Floor(union.X1 * sx),
		Y1: math.Floor(union.Y1 * sy),
		X2: math.Ceil(union.X2 * sx),
		Y2: math.Ceil(union.Y2 * sy),
	}
	img.X1 = clamp(img.X1, 0, imageW)
	img.Y1 = clamp(img.Y1, 0, imageH)
	img.X2 = clamp(img.X2, 0, imageW)
	img.Y2 = clamp(img.Y2, 0, imageH)
	if img.X2 <= img.X1 || img.Y2 <= img.Y1 {
		return spec, fmt.Errorf("page %d: scaled oracle bbox is empty", page)
	}
	spec.ImageBBox = img
	spec.SourceMethod = "RECONSTRUCTED_STORE_LAYOUT_EVIDENCE_UNION+1_CONTEXT_BLOCK+AFFINE_SCALE"

	area := (img.X2 - img.X1) * (img.Y2 - img.Y1)
	full := imageW * imageH
	spec.VisualExposure = area / full
	if spec.VisualExposure > 0 {
		spec.CropReduction = 1 / spec.VisualExposure
	}
	if spec.VisualExposure >= wholePageExposureThreshold {
		spec.Verdict = OracleIncompatible
		spec.Notes = "derived oracle crop covers ~the whole page; oracle localization did not reduce the operand"
		return spec, nil
	}
	spec.Verdict = OracleCompatible
	return spec, nil
}

// DeriveOracleCrop runs DeriveOracleGeometry and then attaches the answer-
// aware sufficiency check (report-only). The geometry in the returned spec
// is independent of everything DeriveOracleCrop reads from `rec` beyond
// page + refs + choices/opcode; poisoning rec.ExpectedAnswer cannot change
// StoreBBox/ImageBBox.
func DeriveOracleCrop(storeDir string, rec P0Record) (OracleCropSpec, error) {
	step, err := rec.ModelStep()
	if err != nil {
		return OracleCropSpec{}, err
	}
	spec, err := DeriveOracleGeometry(storeDir, rec.Page, rec.EvidenceRefs, rec.PageWidth, rec.PageHeight)
	if err != nil {
		return OracleCropSpec{}, err
	}
	spec.BaseID = rec.BaseID
	spec.Category = rec.Category
	spec.ModelOpcode = step.Opcode

	spec.Sufficiency.NotWholePage = spec.VisualExposure < wholePageExposureThreshold
	// Recompute the expanded-window normalized text purely for the report.
	p, perr := storePage(storeDir, rec.Page)
	windowText := ""
	if perr == nil {
		windowText = expandedWindowText(p, spec)
	}
	switch step.Opcode {
	case "SELECT_ONE":
		spec.Sufficiency.Requirement = "crop must show the section heading being identified (the present option); the full choice universe stays in the frozen instruction"
		want := normalizeText(rec.ExpectedAnswer)
		spec.Sufficiency.OperandContentVisible = want != "" && strings.Contains(normalizeText(windowText), want)
		spec.Sufficiency.Detail = fmt.Sprintf("present-option heading %q visible in crop window: %v", rec.ExpectedAnswer, spec.Sufficiency.OperandContentVisible)
	case "EXTRACT_NUMBER":
		spec.Sufficiency.Requirement = "crop must contain the target numeric evidence with local label/context"
		wantDigits := nonDigit.ReplaceAllString(rec.ExpectedAnswer, "")
		haveDigits := nonDigit.ReplaceAllString(windowText, "")
		spec.Sufficiency.OperandContentVisible = wantDigits != "" && strings.Contains(haveDigits, wantDigits)
		spec.Sufficiency.Detail = fmt.Sprintf("target digits %q present in crop window: %v", wantDigits, spec.Sufficiency.OperandContentVisible)
	default:
		spec.Sufficiency.Requirement = "unsupported opcode for T0-B R0"
	}

	if spec.Verdict == OracleCompatible && (!spec.Sufficiency.OperandContentVisible || !spec.Sufficiency.NotWholePage) {
		spec.Verdict = OracleIncompatible
		if spec.Notes == "" {
			spec.Notes = "sufficiency check failed: " + spec.Sufficiency.Detail
		}
	}
	return spec, nil
}

// expandedWindowText concatenates the normalized text of the padded region
// window the crop covers, for the report-only sufficiency check.
func expandedWindowText(p canonicaldoc.Page, spec OracleCropSpec) string {
	inBox := func(b canonicaldoc.BBox) bool {
		cx := (b.X1 + b.X2) / 2
		cy := (b.Y1 + b.Y2) / 2
		return cx >= spec.StoreBBox.X1-0.5 && cx <= spec.StoreBBox.X2+0.5 &&
			cy >= spec.StoreBBox.Y1-0.5 && cy <= spec.StoreBBox.Y2+0.5
	}
	var parts []string
	for _, r := range p.Regions {
		if inBox(r.BBox) {
			parts = append(parts, r.Text)
		}
	}
	return strings.Join(parts, " ")
}

// OracleCropReport is the frozen T0-B0 oracle-crop compatibility artifact.
type OracleCropReport struct {
	Schema              string           `json:"schema"`
	StoreDir            string           `json:"store_dir"`
	StoreRootSHA256     string           `json:"store_root_sha256"`
	ContextBlocksPolicy int              `json:"context_blocks_policy"`
	PaddingRule         string           `json:"padding_rule"`
	EligibleCount       int              `json:"eligible_count"`
	CompatibleCount     int              `json:"compatible_count"`
	Crops               []OracleCropSpec `json:"crops"`
}

const OracleCropReportSchemaR0 = "tlaloc.exocortex-t0b.oracle-crops.r0"

// deriveOracleCrops derives the frozen oracle crop for every ELIGIBLE_R0
// record and assembles the compatibility report. It is deterministic and
// makes zero model calls.
func deriveOracleCrops(records []P0Record, audit EligibilityAudit, storeDir string, _ interface{}) (OracleCropReport, error) {
	eligible := map[string]bool{}
	for _, c := range audit.Cases {
		if c.Eligibility == EligibleR0 {
			eligible[c.BaseID] = true
		}
	}
	manifest, _, err := pdfmemory.Load(storeDir)
	if err != nil {
		return OracleCropReport{}, fmt.Errorf("t0b0: load store: %w", err)
	}
	report := OracleCropReport{
		Schema:              OracleCropReportSchemaR0,
		StoreDir:            storeDir,
		StoreRootSHA256:     manifest.StoreRootSHA256,
		ContextBlocksPolicy: OracleContextBlocks,
		PaddingRule:         "evidence region bounding-union expanded by exactly OracleContextBlocks layout regions per side in reading order, clamped to store page bounds, then affine-scaled to the P0 image coordinate space with outward rounding",
	}
	sorted := append([]P0Record(nil), records...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].BaseID < sorted[j].BaseID })
	for _, rec := range sorted {
		if !eligible[rec.BaseID] {
			continue
		}
		report.EligibleCount++
		spec, err := DeriveOracleCrop(storeDir, rec)
		if err != nil {
			spec = OracleCropSpec{BaseID: rec.BaseID, Category: rec.Category, Page: rec.Page, Verdict: OracleInsufficientProvenance, Notes: err.Error()}
		}
		if spec.Verdict == OracleCompatible {
			report.CompatibleCount++
		}
		report.Crops = append(report.Crops, spec)
	}
	return report, nil
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
