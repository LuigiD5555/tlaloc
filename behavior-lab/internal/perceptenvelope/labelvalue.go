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

// LabelValueCandidate is one deterministically identified label->value pair
// for R1-D (protocol Stage R1-D section 8). Unlike R1-A/R1-B these
// intentionally carry a textual label the model must associate with the
// value; NO target cue is used at R1-D.
type LabelValueCandidate struct {
	CandidateID string            `json:"candidate_id"`
	Page        int               `json:"page"`
	RegionID    string            `json:"region_id"`
	RegionKind  string            `json:"region_kind"`
	Label       string            `json:"label"`
	Value       string            `json:"value"`
	LineText    string            `json:"line_text"`
	LineBBox    canonicaldoc.BBox `json:"line_bbox"`
	PageWidth   float64           `json:"page_width"`
	PageHeight  float64           `json:"page_height"`
	Pattern     string            `json:"pattern"`
}

// LabelValuePool is R1D_POOL.json.
type LabelValuePool struct {
	Schema           string                `json:"schema"`
	ExperimentID     string                `json:"experiment_id"`
	SourcePDFSHA256  string                `json:"source_pdf_sha256"`
	StoreRootSHA256  string                `json:"store_root_sha256"`
	AlgorithmVersion string                `json:"selection_algorithm_version"`
	ExcludedIDs      []string              `json:"excluded_candidate_ids"`
	Count            int                   `json:"candidate_count"`
	Candidates       []LabelValueCandidate `json:"candidates"`
	Note             string                `json:"note"`
}

const lvSchema = "tlaloc.parrot-perceptual-envelope-r1.labelvalue-pool.r1"

// labelValuePatterns: "<label> is <int>", "<label>: <int>", "<label> = <int>",
// "<label> of <int>". Label must be >=2 alphabetic words, value a plain int.
var (
	reLVis    = regexp.MustCompile(`^([A-Za-z][A-Za-z '\-]{3,60}?)\s+(?:is|are|was|were|of|:|=)\s+([0-9]{2,5})\b`)
	reLVtrail = regexp.MustCompile(`^([A-Za-z][A-Za-z '\-]{3,60}?)\s+([0-9]{2,5})$`)
)

// ScanLabelValuePool finds label/value lines, excluding R1-A/R1-B candidate ids.
func ScanLabelValuePool(storeDir string, excluded map[string]struct{}) (LabelValuePool, error) {
	manifest, _, err := pdfmemory.Load(storeDir)
	if err != nil {
		return LabelValuePool{}, err
	}
	srcSHA := manifest.SourceSHA256
	if srcSHA == "" && len(manifest.Documents) > 0 {
		srcSHA = manifest.Documents[0].SourceSHA256
	}
	pool := LabelValuePool{
		Schema: lvSchema, ExperimentID: ExperimentID, SourcePDFSHA256: srcSHA,
		StoreRootSHA256: manifest.StoreRootSHA256, AlgorithmVersion: SelectionAlgorithmVersion,
		Note: "Deterministic label/value scan for R1-D. R1-D allocates 24 held-out bases by hash in the R1-D stage. Disjoint from R1-A/R1-B by construction (candidate ids excluded).",
	}
	for id := range excluded {
		pool.ExcludedIDs = append(pool.ExcludedIDs, id)
	}
	sort.Strings(pool.ExcludedIDs)

	pages := append([]pdfmemory.PageRef(nil), manifest.Pages...)
	sort.Slice(pages, func(i, j int) bool { return pages[i].Number < pages[j].Number })
	for _, pref := range pages {
		if pref.LayoutPath == "" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(pref.LayoutPath)))
		if err != nil {
			return LabelValuePool{}, err
		}
		var page canonicaldoc.Page
		if err := json.Unmarshal(body, &page); err != nil {
			return LabelValuePool{}, err
		}
		for _, region := range page.Regions {
			text := strings.TrimSpace(region.Text)
			if text == "" || !anyDigit.MatchString(text) {
				continue
			}
			if region.Kind != "text_line" && region.Kind != "list_item" && region.Kind != "table_cell" {
				continue
			}
			var label, value, pattern string
			if m := reLVis.FindStringSubmatch(text); m != nil {
				label, value, pattern = strings.TrimSpace(m[1]), m[2], "LABEL_REL_VALUE"
			} else if m := reLVtrail.FindStringSubmatch(text); m != nil {
				label, value, pattern = strings.TrimSpace(m[1]), m[2], "LABEL_TRAILING_VALUE"
			} else {
				continue
			}
			// require exactly one number on the line (unambiguous gold)
			if len(regexp.MustCompile(`[0-9]+`).FindAllString(text, -1)) != 1 {
				continue
			}
			cid := sha256Hex([]byte(strings.Join([]string{srcSHA, fmt.Sprintf("p%d", page.Number), region.ID, label, value}, "|")))[:32]
			if _, skip := excluded[cid]; skip {
				continue
			}
			pool.Candidates = append(pool.Candidates, LabelValueCandidate{
				CandidateID: cid, Page: page.Number, RegionID: region.ID, RegionKind: region.Kind,
				Label: label, Value: value, LineText: text, LineBBox: region.BBox,
				PageWidth: page.Width, PageHeight: page.Height, Pattern: pattern,
			})
		}
	}
	sort.Slice(pool.Candidates, func(i, j int) bool { return pool.Candidates[i].CandidateID < pool.Candidates[j].CandidateID })
	pool.Count = len(pool.Candidates)
	return pool, nil
}
