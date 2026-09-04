package tonalt1

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

var anyDigit = regexp.MustCompile(`[0-9]`)

// runningHeaderCaps is an ALL-CAPS running header with an embedded page
// number, e.g. "LINEAR NEURAL NETWORKS 173".
var runningHeaderCaps = regexp.MustCompile(`^[A-Z][A-Z .,':;&/-]{6,}\s+[0-9]{1,4}$`)

// bibliographyCue marks a reference / citation line.
var bibliographyCue = regexp.MustCompile(`(?i)\bet al\b|\bpp\.|\bvol\.|\beds?\.|arxiv|\bdoi\b|https?://|\((?:1[6-9]|20)[0-9]{2}\)`)

// crossReferenceCue marks a document cross-reference immediately before
// the numeric token (e.g. "Section 2.4", "Fig. 15.1", "Chapter 12",
// "Equation 3.6"). The number is a locator, not a quantity — the frozen
// R1 pool filters already reject "section numbers" and "equation tags";
// this extends the same principle to decimal N.M references that survive
// the number-leading check.
var crossReferenceCue = regexp.MustCompile(`(?i)^(sec|section|sections|subsection|ch|chap|chapter|chapters|fig|figure|figures|figs|tbl|table|tables|eq|eqn|equation|equations|appendix|app|part|algorithm|alg|theorem|thm|lemma|corollary|proposition|prop|definition|def|example|exercise|problem|listing|step|page|pp|no|№)\.?$`)

// versionCue marks a software / library / format version number: the word
// immediately before the token names a versioned technology. The number is
// an identifier, not a quantity.
var versionCue = regexp.MustCompile(`(?i)^(version|ver|v|release|rev|cuda|cudnn|python|python2|python3|tensorflow|tf|pytorch|torch|keras|mxnet|numpy|scipy|pandas|jax|onnx|java|c\+\+|gcc|clang|glibc|ubuntu|debian|windows|macos|ios|android|opengl|opencl|http|https|api|sdk|bert|gpt|resnet|vgg|yolo|efficientnet|mobilenet|densenet)$`)

// wrappedFragmentFirst marks a containing line whose first whitespace
// token is a broken suffix of a reference word (e.g. "Sec-\ntion 16.2" ->
// line starts "tion 16.2"). Deliberately tiny and specific.
var wrappedFragmentFirst = map[string]bool{
	"tion": true, "tions": true, "ure": true, "ures": true, "dix": true,
	"rithm": true, "rithms": true, "ple": true, "ples": true, "orem": true,
	"ction": true, "ctions": true,
}

// stripEdgePunct removes surrounding punctuation before morphology tests.
func stripEdgePunct(token string) string {
	return strings.Trim(token, ".,;:()[]%\"'")
}

// numericLiteral matches the broad shape of a numeric literal token after
// edge punctuation is stripped: an optional sign, digits, and optional
// interior separators / decimal / exponent / range dash. It is
// deliberately permissive — its job is to decide "this token is a number,
// classify its morphology", not to admit it. "GPT-3", "3rd", "v2" fail.
var numericLiteral = regexp.MustCompile(`^[+-]?[0-9]+(?:[.,][0-9]+)*(?:[eE][+-]?[0-9]+)?$|^[0-9]+[\x{2013}\x{2014}-][0-9]+$`)

// looksNumericToken reports whether a whitespace field is a numeric
// literal (in any morphology, admissible or not).
func looksNumericToken(field string) bool {
	return numericLiteral.MatchString(stripEdgePunct(field))
}

// ScanResult is the complete deterministic D3 scan outcome.
type ScanResult struct {
	StoreDir        string
	SourcePDFSHA256 string
	StoreRootSHA256 string
	CarrierID       string
	PagesScanned    int
	RegionsScanned  int
	DigitTokensSeen int

	// Candidates holds EVERY discovered candidate (eligible and not),
	// sorted by candidate_id. Provenance is preserved for all.
	Candidates []Candidate
}

// Scan walks every page of the canonical store deterministically and
// returns the full candidate universe with eligibility resolved against
// the prior-use index. Same store bytes + same priorIndex -> identical
// ScanResult (identical candidate_ids and ordering).
func Scan(storeDir string, priorIndex *PriorUseIndex) (ScanResult, error) {
	manifest, _, err := pdfmemory.Load(storeDir)
	if err != nil {
		return ScanResult{}, fmt.Errorf("load store %s: %w", storeDir, err)
	}
	sourceSHA := manifest.SourceSHA256
	if sourceSHA == "" && len(manifest.Documents) > 0 {
		sourceSHA = manifest.Documents[0].SourceSHA256
	}

	result := ScanResult{
		StoreDir:        storeDir,
		SourcePDFSHA256: sourceSHA,
		StoreRootSHA256: manifest.StoreRootSHA256,
		CarrierID:       manifest.CarrierID,
	}

	pages := append([]pdfmemory.PageRef(nil), manifest.Pages...)
	sort.Slice(pages, func(i, j int) bool { return pages[i].Number < pages[j].Number })

	for _, pageRef := range pages {
		if strings.TrimSpace(pageRef.LayoutPath) == "" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(pageRef.LayoutPath)))
		if err != nil {
			return ScanResult{}, fmt.Errorf("read layout %s: %w", pageRef.LayoutPath, err)
		}
		var page canonicaldoc.Page
		if err := json.Unmarshal(body, &page); err != nil {
			return ScanResult{}, fmt.Errorf("decode layout %s: %w", pageRef.LayoutPath, err)
		}
		result.PagesScanned++
		result.RegionsScanned += len(page.Regions)
		scanPage(&result, page, pageRef.LayoutPath, sourceSHA, manifest, priorIndex)
	}

	sort.Slice(result.Candidates, func(i, j int) bool {
		return result.Candidates[i].CandidateID < result.Candidates[j].CandidateID
	})
	return result, nil
}

func scanPage(result *ScanResult, page canonicaldoc.Page, layoutPath, sourceSHA string, manifest pdfmemory.Manifest, priorIndex *PriorUseIndex) {
	for _, region := range page.Regions {
		text := strings.TrimSpace(region.Text)
		if text == "" {
			continue
		}
		fields := strings.Fields(text)

		digitTokens := 0
		var primaryVerbatim string
		numericCount := 0
		for _, field := range fields {
			if !anyDigit.MatchString(field) {
				continue
			}
			digitTokens++
			result.DigitTokensSeen++
			if !looksNumericToken(field) {
				continue // digit-bearing but not a numeric literal (e.g. "GPT-3", "3rd")
			}
			numericCount++
			if primaryVerbatim == "" {
				primaryVerbatim = field
			}
		}
		if numericCount == 0 {
			continue
		}

		stripped := stripEdgePunct(primaryVerbatim)
		family, verbatimAdmissible := classifyMorphology(primaryVerbatim, stripped)

		cand := newCandidate(page, region, layoutPath, sourceSHA, manifest, primaryVerbatim, stripped, family, len(fields))

		var codes []RejectionCode

		// --- morphology / region kind ---
		if region.Kind != "text_line" && region.Kind != "list_item" {
			codes = append(codes, RejectRegionKind)
		}
		if !verbatimAdmissible {
			codes = append(codes, RejectUnsupportedMorphology)
		}
		if family == MorphMultiDigitInteger && isYearLike(stripped) {
			codes = append(codes, RejectYearOrDateToken)
		}

		// --- one-numeric-token-per-line ambiguity ---
		cand.Presentation.NumericTokenCount = numericCount
		cand.Presentation.CompetingNumericCount = numericCount - 1
		if numericCount != 1 || digitTokens != 1 {
			codes = append(codes, RejectMultipleNumericTokens)
		}

		// --- prose / citation / header ---
		if bibliographyCue.MatchString(text) {
			codes = append(codes, RejectBibliographyLine)
		}
		if isCrossReference(fields, primaryVerbatim) {
			codes = append(codes, RejectCrossReference)
		}
		if isVersionString(fields, primaryVerbatim) {
			codes = append(codes, RejectVersionString)
		}
		if len(fields) > 0 && wrappedFragmentFirst[strings.ToLower(stripEdgePunct(fields[0]))] {
			codes = append(codes, RejectWrappedFragment)
		}
		if region.FontSize > 0 && region.FontSize < 10 {
			codes = append(codes, RejectFontBelowBody)
		}
		if runningHeaderCaps.MatchString(text) {
			codes = append(codes, RejectRunningHeader)
		}
		// DOMAIN: a lone number with no textual context is not a
		// contextualised quantity (page/line/figure/equation number).
		if len(fields) < 2 {
			codes = append(codes, RejectLoneNumberLine)
		}
		// DOMAIN: bare number alone in the top or bottom 6% of the page is
		// a running header/footer page number.
		if len(fields) < 3 && page.Height > 0 {
			midY := (region.BBox.Y1 + region.BBox.Y2) / 2
			if midY < 0.06*page.Height || midY > 0.94*page.Height {
				codes = append(codes, RejectPageHeaderFooter)
			}
		}
		// AUTHORING (advisory): the frozen R1-A/R1-B pool prose-context
		// heuristics. R1-C proved the capability without them.
		if len(fields) < 4 {
			codes = append(codes, RejectBareOrShortNumberLine)
		}

		// --- token offsets (rune) ---
		tokenStart, tokenEnd, offsetOK := runeSpan(text, primaryVerbatim)
		if offsetOK {
			cand.Identity.CharStart = tokenStart
			cand.Identity.CharEnd = tokenEnd
			cand.Identity.NormalizedSpanHash = normalizedSpanHash(page.Number, text, tokenStart, tokenEnd)
			cand.Source.QuantityPhrase = quantityPhrase(fields, primaryVerbatim)
		} else {
			codes = append(codes, RejectTokenOffsetNotUnique)
		}
		// The candidate id binds the token rune span, so it must be
		// (re)derived now that the span is known — not at construction time.
		cand.CandidateID = deriveCandidateID(cand)

		// --- geometry / ambiguity audit ---
		if offsetOK {
			verdict := auditGeometry(page, region, primaryVerbatim, tokenStart, tokenEnd, fields)
			codes = append(codes, verdict.rejections...)
			cand.Geometry.OperandBBoxEstimate = verdict.operandBBoxEstimate
			cand.Geometry.CueBBoxStore = verdict.cueBBoxStore
			cand.Presentation.EffectiveScale = verdict.effectiveScale
			cand.Presentation.WhitespaceFieldCount = verdict.whitespaceFields
		}

		// --- domain (only meaningful for an admissible morphology) ---
		if family != "" && verbatimAdmissible {
			domain, domainOK := deriveDomain(stripped, family)
			cand.Domain = domain
			if !domainOK {
				codes = append(codes, RejectDomainInvalid)
			}
		}

		// --- prior-use exclusion (union) ---
		if priorIndex != nil {
			matches := priorIndex.Match(cand)
			if len(matches) > 0 {
				cand.PriorUse.Excluded = true
				cand.PriorUse.Matches = matches
				codes = append(codes, RejectPriorUsed)
			}
		}

		cand.Presentation.R1EnvelopeSupported = family != "" && verbatimAdmissible &&
			!contains(codes, RejectYearOrDateToken) && !contains(codes, RejectRegionKind)

		codes = dedupeSortCodes(codes)
		cand.Eligibility.RejectionCodes = codes
		// D3 v2: only CAPABILITY / PRESENTATION_INTEGRITY / DOMAIN_VALIDITY
		// rules exclude a candidate. AUTHORING_HEURISTIC codes are recorded
		// as advisory tags (see types.go ruleClassOf).
		blocking := 0
		var advisory []RejectionCode
		for _, code := range codes {
			if blocks(code) {
				blocking++
			} else {
				advisory = append(advisory, code)
			}
		}
		cand.Eligibility.AdvisoryTags = advisory
		cand.Eligibility.Eligible = blocking == 0

		result.Candidates = append(result.Candidates, cand)
	}
}

// runeSpan returns the rune [start,end) offsets of the unique occurrence
// of token in text. ok is false if the token does not occur exactly once.
func runeSpan(text, token string) (start, end int, ok bool) {
	if strings.Count(text, token) != 1 {
		return 0, 0, false
	}
	byteStart := strings.Index(text, token)
	start = len([]rune(text[:byteStart]))
	end = start + len([]rune(token))
	if start < 0 || end <= start {
		return 0, 0, false
	}
	return start, end, true
}

// isCrossReference reports whether the numeric token is immediately
// preceded on the line by a reference word ("Section", "Fig.", ...),
// making it a locator rather than a quantity.
func isCrossReference(fields []string, token string) bool {
	return precededByCue(fields, token, crossReferenceCue.MatchString)
}

// isVersionString reports whether the token is preceded by a versioned
// technology name ("CUDA 12.1", "TensorFlow 2.0").
func isVersionString(fields []string, token string) bool {
	return precededByCue(fields, token, versionCue.MatchString)
}

func precededByCue(fields []string, token string, match func(string) bool) bool {
	for index, field := range fields {
		if field != token {
			continue
		}
		if index == 0 {
			return false
		}
		return match(strings.Trim(fields[index-1], ".,;:()[]\"'"))
	}
	return false
}

// quantityPhrase is a deterministic locate hint: up to three whitespace
// tokens immediately preceding the operand on the line (or following it,
// if the operand is line-initial). Never derived from a model.
func quantityPhrase(fields []string, token string) string {
	index := -1
	for i, field := range fields {
		if field == token {
			index = i
			break
		}
	}
	if index < 0 {
		return ""
	}
	lo := index - 3
	if lo < 0 {
		lo = 0
	}
	if lo < index {
		return strings.Join(fields[lo:index], " ")
	}
	hi := index + 4
	if hi > len(fields) {
		hi = len(fields)
	}
	return strings.Join(fields[index+1:hi], " ")
}

// deriveDomain computes deterministic arithmetic-domain properties.
func deriveDomain(normalized string, family MorphologyFamily) (CandidateDomain, bool) {
	value, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return CandidateDomain{}, false
	}
	isInteger := family == MorphMultiDigitInteger || value == float64(int64(value))
	domain := CandidateDomain{
		ParseValid:            true,
		Finite:                true,
		Integer:               isInteger,
		Zero:                  value == 0,
		Sign:                  signOf(value),
		EligibleAsOperand:     true,
		EligibleAsDenominator: value != 0,
	}
	return domain, true
}

func signOf(value float64) int {
	switch {
	case value > 0:
		return 1
	case value < 0:
		return -1
	default:
		return 0
	}
}

func newCandidate(page canonicaldoc.Page, region canonicaldoc.Region, layoutPath, sourceSHA string, manifest pdfmemory.Manifest, verbatim, stripped string, family MorphologyFamily, fieldCount int) Candidate {
	value, _ := strconv.ParseFloat(stripped, 64)
	cand := Candidate{
		SchemaVersion: CandidateSchemaVersion,
		Corpus: CandidateCorpus{
			CorpusID:        manifest.CarrierID,
			SourcePDFSHA256: sourceSHA,
			StoreRootSHA256: manifest.StoreRootSHA256,
			Page:            page.Number,
			PageWidth:       page.Width,
			PageHeight:      page.Height,
			LayoutPath:      layoutPath,
			PageRegionCount: len(page.Regions),
		},
		Identity: PhysicalIdentity{
			Page:       page.Number,
			RegionID:   region.ID,
			SourceBBox: region.BBox,
		},
		Source: CandidateSource{
			ContainingLineText: strings.TrimSpace(region.Text),
			NumericRaw:         verbatim,
			NumericNormalized:  stripped,
			NumericValue:       value,
		},
		Geometry: CandidateGeometry{
			PageWidth:          page.Width,
			PageHeight:         page.Height,
			ContainingLineBBox: region.BBox,
			LocatedRegionBBox:  region.BBox,
		},
		Presentation: CandidatePresentation{
			MorphologyFamily: family,
		},
		Provenance: CandidateProvenance{
			SelectorVersion:          SelectorVersion,
			SpanNormVersion:          SpanNormVersion,
			EnvelopeVersion:          EnvelopeVersion,
			GeometryRuleVersion:      GeometryRuleVersion,
			PriorUseInventoryVersion: PriorUseInventoryVersion,
			TokenBoxMethod:           tokenBoxMethod,
			PaddingPolicy:            paddingPolicy,
		},
	}
	cand.CandidateID = deriveCandidateID(cand)
	return cand
}

func contains(codes []RejectionCode, want RejectionCode) bool {
	for _, code := range codes {
		if code == want {
			return true
		}
	}
	return false
}

func dedupeSortCodes(codes []RejectionCode) []RejectionCode {
	if len(codes) == 0 {
		return nil
	}
	seen := map[RejectionCode]bool{}
	var out []RejectionCode
	for _, code := range codes {
		if !seen[code] {
			seen[code] = true
			out = append(out, code)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
