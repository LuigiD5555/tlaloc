package perceptenvelope

import (
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

// Numeric morphology families (protocol Stage R1-C section 4.1 / 7).
const (
	MorphSingleDigit      = "SINGLE_DIGIT"
	MorphMultiDigitInt    = "MULTI_DIGIT_INTEGER"
	MorphThousandsSep     = "THOUSANDS_SEPARATOR"
	MorphDecimal          = "DECIMAL"
	MorphPercentage       = "PERCENTAGE"
	MorphSigned           = "SIGNED_NUMBER"
	MorphRange            = "RANGE"
	MorphScientific       = "SCIENTIFIC_NOTATION"
	MorphCoordTuple       = "COORDINATE_OR_TUPLE"
	MorphEquationEmbedded = "EQUATION_EMBEDDED"
	MorphTableCell        = "TABLE_CELL"
	MorphPageSection      = "PAGE_OR_SECTION_NUMBER"
)

var (
	reThousands  = regexp.MustCompile(`^[0-9]{1,3}(,[0-9]{3})+$`)
	reDecimal    = regexp.MustCompile(`^[0-9]*\.[0-9]+$`)
	rePercent    = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?%$`)
	reSigned     = regexp.MustCompile(`^[+-][0-9]+(\.[0-9]+)?$`)
	reScientific = regexp.MustCompile(`(?i)^[0-9]+(\.[0-9]+)?e[+-]?[0-9]+$`)
	reRange      = regexp.MustCompile(`^[0-9]+[\x{2013}\x{2014}-][0-9]+$`)
	rePlainInt   = regexp.MustCompile(`^[0-9]+$`)
	reTuple      = regexp.MustCompile(`^\(?[0-9]+,\s*[0-9]+\)?$`)
	reEquation   = regexp.MustCompile(`^[A-Za-z]\s*=\s*[0-9]+$`)
)

// MorphCandidate is one classified numeric occurrence for R1-C.
type MorphCandidate struct {
	CandidateID string            `json:"candidate_id"`
	Family      string            `json:"family"`
	Page        int               `json:"page"`
	RegionID    string            `json:"region_id"`
	RegionKind  string            `json:"region_kind"`
	Token       string            `json:"token"`
	LineText    string            `json:"line_text"`
	LineBBox    canonicaldoc.BBox `json:"line_bbox"`
	PageWidth   float64           `json:"page_width"`
	PageHeight  float64           `json:"page_height"`
}

// MorphologyPool is R1C_POOL.json.
type MorphologyPool struct {
	Schema           string                      `json:"schema"`
	ExperimentID     string                      `json:"experiment_id"`
	SourcePDFSHA256  string                      `json:"source_pdf_sha256"`
	StoreRootSHA256  string                      `json:"store_root_sha256"`
	AlgorithmVersion string                      `json:"selection_algorithm_version"`
	ExcludedBaseIDs  []string                    `json:"excluded_candidate_ids"` // R1-A/R1-B primary candidate ids kept out
	FamilyAvailable  map[string]int              `json:"real_document_n_available"`
	Families         map[string][]MorphCandidate `json:"families"`
	Note             string                      `json:"note"`
}

const morphSchema = "tlaloc.parrot-perceptual-envelope-r1.morphology-pool.r1"

func classifyMorph(tok, lineText, kind string) string {
	t := strings.TrimSpace(tok)
	switch {
	case kind == "table_cell":
		return MorphTableCell
	case reScientific.MatchString(t):
		return MorphScientific
	case rePercent.MatchString(t):
		return MorphPercentage
	case reThousands.MatchString(t):
		return MorphThousandsSep
	case reSigned.MatchString(t):
		return MorphSigned
	case reRange.MatchString(t):
		return MorphRange
	case reTuple.MatchString(t):
		return MorphCoordTuple
	case reDecimal.MatchString(t):
		return MorphDecimal
	case reEquation.MatchString(strings.TrimSpace(lineText)):
		return MorphEquationEmbedded
	case rePlainInt.MatchString(t) && len(t) == 1:
		return MorphSingleDigit
	case rePlainInt.MatchString(t):
		return MorphMultiDigitInt
	}
	return ""
}

// ScanMorphologyPool classifies every numeric occurrence in the store into
// the 12 morphology families, excluding the R1-A/R1-B primary candidate ids.
func ScanMorphologyPool(storeDir string, excluded map[string]struct{}) (MorphologyPool, error) {
	manifest, _, err := pdfmemory.Load(storeDir)
	if err != nil {
		return MorphologyPool{}, err
	}
	srcSHA := manifest.SourceSHA256
	if srcSHA == "" && len(manifest.Documents) > 0 {
		srcSHA = manifest.Documents[0].SourceSHA256
	}
	pool := MorphologyPool{
		Schema: morphSchema, ExperimentID: ExperimentID, SourcePDFSHA256: srcSHA,
		StoreRootSHA256: manifest.StoreRootSHA256, AlgorithmVersion: SelectionAlgorithmVersion,
		FamilyAvailable: map[string]int{}, Families: map[string][]MorphCandidate{},
		Note: "Deterministic morphology scan. R1-A/R1-B primary candidates excluded. R1-C allocation (10-12 bases/family by hash) happens in the R1-C stage, not here; families with fewer than 10 record REAL_DOCUMENT_N_AVAILABLE and may be supplemented by a separately-labelled SYNTHETIC_REALISTIC stratum — never pooled.",
	}
	for id := range excluded {
		pool.ExcludedBaseIDs = append(pool.ExcludedBaseIDs, id)
	}
	sort.Strings(pool.ExcludedBaseIDs)

	pages := append([]pdfmemory.PageRef(nil), manifest.Pages...)
	sort.Slice(pages, func(i, j int) bool { return pages[i].Number < pages[j].Number })
	tokenSplit := regexp.MustCompile(`\s+`)
	for _, pref := range pages {
		if pref.LayoutPath == "" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(pref.LayoutPath)))
		if err != nil {
			return MorphologyPool{}, err
		}
		var page canonicaldoc.Page
		if err := json.Unmarshal(body, &page); err != nil {
			return MorphologyPool{}, err
		}
		for _, region := range page.Regions {
			text := strings.TrimSpace(region.Text)
			if text == "" || !anyDigit.MatchString(text) {
				continue
			}
			for _, raw := range tokenSplit.Split(text, -1) {
				tok := strings.Trim(raw, "().,;:")
				if tok == "" || !anyDigit.MatchString(tok) {
					continue
				}
				fam := classifyMorph(tok, text, region.Kind)
				if fam == "" {
					continue
				}
				cid := sha256Hex([]byte(strings.Join([]string{srcSHA, fmt.Sprintf("p%d", page.Number), region.ID, tok, fam}, "|")))[:32]
				if _, skip := excluded[cid]; skip {
					continue
				}
				pool.Families[fam] = append(pool.Families[fam], MorphCandidate{
					CandidateID: cid, Family: fam, Page: page.Number, RegionID: region.ID,
					RegionKind: region.Kind, Token: tok, LineText: text, LineBBox: region.BBox,
					PageWidth: page.Width, PageHeight: page.Height,
				})
			}
		}
	}
	for fam, cs := range pool.Families {
		sort.Slice(cs, func(i, j int) bool { return cs[i].CandidateID < cs[j].CandidateID })
		pool.Families[fam] = cs
		pool.FamilyAvailable[fam] = len(cs)
	}
	for _, fam := range []string{
		MorphSingleDigit, MorphMultiDigitInt, MorphThousandsSep, MorphDecimal, MorphPercentage,
		MorphSigned, MorphRange, MorphScientific, MorphCoordTuple, MorphEquationEmbedded,
		MorphTableCell, MorphPageSection,
	} {
		if _, ok := pool.FamilyAvailable[fam]; !ok {
			pool.FamilyAvailable[fam] = 0
		}
	}
	return pool, nil
}
