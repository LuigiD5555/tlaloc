package exocortex

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// RegionLocateTlaloqueID / RegionCropTlaloqueID are the two thin adapters
// over the same deterministic Region Tlaloque logic (section 12: one
// Tlaloque, two capabilities). Neither ever calls a generative model.
const (
	RegionLocateTlaloqueID = "region-locate-tlaloque"
	RegionCropTlaloqueID   = "region-crop-tlaloque"
)

// Locate modes. T0-A (oracle) and T0-B (real) share this one Tlaloque; only
// the mode differs, so no second Region Tlaloque is created.
const (
	LocateModeOracle = "ORACLE"
	LocateModeReal   = "REAL"
)

// RegionLocateResult is the deterministic locator's output contract
// (section 23): selected address, bbox, ranking score, source method,
// provenance, candidate count.
type RegionLocateResult struct {
	SelectedAddress string             `json:"selected_address"`
	RegionAddress   string             `json:"region_address,omitempty"`
	DocID           string             `json:"doc_id"`
	Page            int                `json:"page"`
	BBox            *canonicaldoc.BBox `json:"bbox,omitempty"`
	PageWidth       float64            `json:"page_width,omitempty"`
	PageHeight      float64            `json:"page_height,omitempty"`
	RankingScore    float64            `json:"ranking_score"`
	SourceMethod    string             `json:"source_method"`
	Provenance      map[string]string  `json:"provenance,omitempty"`
	CandidateCount  int                `json:"candidate_count"`
}

// RegionLocateInput is the LOCATE_REGION opcode's input contract.
type RegionLocateInput struct {
	Mode     string `json:"mode"` // ORACLE | REAL
	Question string `json:"question,omitempty"`
	StoreDir string `json:"store_dir,omitempty"`
	Limit    int    `json:"limit,omitempty"`

	// ORACLE-only fields. Ground-truth addresses/bbox may only ever be used
	// to select geometry; OracleLocate never receives or forwards an
	// expected answer, required facts, or hidden labels (section 17).
	OracleAddress string             `json:"oracle_address,omitempty"`
	OracleDocID   string             `json:"oracle_doc_id,omitempty"`
	OraclePage    int                `json:"oracle_page,omitempty"`
	OracleBBox    *canonicaldoc.BBox `json:"oracle_bbox,omitempty"`
	OraclePageW   float64            `json:"oracle_page_width,omitempty"`
	OraclePageH   float64            `json:"oracle_page_height,omitempty"`
}

// RegionLocateTlaloque implements LOCATE_REGION. In ORACLE mode it uses a
// frozen P0 evidence address to select geometry only (T0-A). In REAL mode
// it uses Origami's existing pdfmemory address/index/search primitives —
// never ground truth, never another LLM (T0-B, section 22-23).
type RegionLocateTlaloque struct{}

func (RegionLocateTlaloque) Descriptor() tlaloque.CapabilityDescriptor {
	d, _ := tlaloque.CapabilityDescriptor{
		ID: RegionLocateTlaloqueID, Capability: OpLocateRegion, Engine: tlaloque.EngineDeterministic,
		InputSchema: "exocortex.region-locate-input.r0", OutputSchema: "exocortex.region-locate-result.r0",
		Deterministic: true, MaxConcurrency: 8,
	}.Normalize()
	return d
}

func (RegionLocateTlaloque) Execute(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in RegionLocateInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("region locate tlaloque: decode input: %w", err)
	}
	var result RegionLocateResult
	var err error
	switch strings.ToUpper(strings.TrimSpace(in.Mode)) {
	case LocateModeOracle:
		result, err = OracleLocate(in)
	case LocateModeReal:
		result, err = RealLocate(in)
	default:
		return tlaloque.CapabilityResponse{}, fmt.Errorf("region locate tlaloque: unsupported mode %q", in.Mode)
	}
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	body, err := json.Marshal(result)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{
		WorkerID: RegionLocateTlaloqueID, Output: body, Confidence: 1,
		Observations: []blackboard.Observation{{Key: req.NodeID, Value: body, Confidence: 1, Provenance: map[string]string{"source": RegionLocateTlaloqueID, "source_method": result.SourceMethod}}},
	}, nil
}

// OracleLocate selects geometry from a frozen P0 evidence address. It
// deliberately has no way to receive or forward an expected answer: its
// input contract carries only address/bbox/page geometry (section 17).
func OracleLocate(in RegionLocateInput) (RegionLocateResult, error) {
	if strings.TrimSpace(in.OracleAddress) == "" || strings.TrimSpace(in.OracleDocID) == "" {
		return RegionLocateResult{}, fmt.Errorf("oracle locate: oracle_address and oracle_doc_id are required")
	}
	if in.OraclePage <= 0 {
		return RegionLocateResult{}, fmt.Errorf("oracle locate: oracle_page must be positive")
	}
	return RegionLocateResult{
		SelectedAddress: in.OracleAddress,
		RegionAddress:   in.OracleAddress,
		DocID:           in.OracleDocID,
		Page:            in.OraclePage,
		BBox:            in.OracleBBox,
		PageWidth:       in.OraclePageW,
		PageHeight:      in.OraclePageH,
		RankingScore:    1,
		SourceMethod:    "ORACLE_GROUND_TRUTH_ADDRESS",
		Provenance:      map[string]string{"evidence_address": in.OracleAddress},
		CandidateCount:  1,
	}, nil
}

// RealLocate never uses expected_answer, required_facts, ground-truth
// addresses, or a human-selected bbox. It ranks candidate blocks with
// Origami's existing deterministic lexical search (pdfmemory.Search,
// reused rather than reimplemented — section 23 preferred strategy #3),
// then attempts to refine to a tighter region bbox by term overlap within
// the selected page's own layout (a small, honest, deterministic
// refinement, not a second search engine). When no region layout is
// resolvable it returns the page-level candidate with BBox == nil rather
// than fabricating a bbox.
func RealLocate(in RegionLocateInput) (RegionLocateResult, error) {
	question := strings.TrimSpace(in.Question)
	if question == "" {
		return RegionLocateResult{}, fmt.Errorf("real locate: question is required")
	}
	if strings.TrimSpace(in.StoreDir) == "" {
		return RegionLocateResult{}, fmt.Errorf("real locate: store_dir is required")
	}
	m, idx, err := pdfmemory.Load(in.StoreDir)
	if err != nil {
		return RegionLocateResult{}, fmt.Errorf("real locate: load pdfmemory store: %w", err)
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 8
	}
	hits := pdfmemory.Search(m, idx, question, limit)
	if len(hits) == 0 {
		return RegionLocateResult{}, fmt.Errorf("real locate: no candidates found for question")
	}
	top := hits[0]
	result := RegionLocateResult{
		SelectedAddress: top.Address,
		DocID:           top.DocID,
		Page:            top.Page,
		RankingScore:    1,
		SourceMethod:    "DETERMINISTIC_LEXICAL_SEARCH(pdfmemory.Search)",
		Provenance:      map[string]string{"block_address": top.Address},
		CandidateCount:  len(hits),
	}

	pageRef, ok := findPageRef(m, top.DocID, top.Page)
	if !ok {
		return result, nil
	}
	page, err := readPageLayout(in.StoreDir, pageRef)
	if err != nil {
		return result, nil // page-level candidate is still a valid, honest result
	}
	result.PageWidth, result.PageHeight = page.Width, page.Height
	region, score, ok := bestRegionByTermOverlap(page, question)
	if !ok {
		return result, nil
	}
	bbox := region.BBox
	result.BBox = &bbox
	result.RegionAddress = region.Address
	result.RankingScore = score
	result.SourceMethod += "+PAGE_LAYOUT_TERM_OVERLAP"
	return result, nil
}

func findPageRef(m pdfmemory.Manifest, docID string, page int) (pdfmemory.PageRef, bool) {
	for _, p := range m.Pages {
		if p.DocID == docID && p.Number == page {
			return p, true
		}
	}
	return pdfmemory.PageRef{}, false
}

func readPageLayout(storeDir string, page pdfmemory.PageRef) (canonicaldoc.Page, error) {
	if strings.TrimSpace(page.LayoutPath) == "" {
		return canonicaldoc.Page{}, fmt.Errorf("page has no layout path")
	}
	body, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(page.LayoutPath)))
	if err != nil {
		return canonicaldoc.Page{}, err
	}
	if page.LayoutCID != "" {
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != page.LayoutCID {
			return canonicaldoc.Page{}, fmt.Errorf("layout CID mismatch")
		}
	}
	var out canonicaldoc.Page
	if err := json.Unmarshal(body, &out); err != nil {
		return canonicaldoc.Page{}, err
	}
	return out, nil
}

// bestRegionByTermOverlap is a small, honest, deterministic scorer: it
// counts how many query terms appear in each region's text and returns the
// best-covered region. It is not a second BM25 engine — it only refines
// pdfmemory.Search's page-level result to a tighter region within that one
// page.
func bestRegionByTermOverlap(page canonicaldoc.Page, question string) (canonicaldoc.Region, float64, bool) {
	terms := simpleTokenize(question)
	if len(terms) == 0 || len(page.Regions) == 0 {
		return canonicaldoc.Region{}, 0, false
	}
	type scored struct {
		region canonicaldoc.Region
		score  float64
	}
	best := scored{}
	found := false
	for _, region := range page.Regions {
		regionTerms := map[string]bool{}
		for _, t := range simpleTokenize(region.Text) {
			regionTerms[t] = true
		}
		overlap := 0
		for _, t := range terms {
			if regionTerms[t] {
				overlap++
			}
		}
		if overlap == 0 {
			continue
		}
		score := float64(overlap) / float64(len(terms))
		if !found || score > best.score || (score == best.score && region.Address < best.region.Address) {
			best = scored{region: region, score: score}
			found = true
		}
	}
	return best.region, best.score, found
}

func simpleTokenize(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	sort.Strings(fields)
	return fields
}

// RegionCropInput is the CROP_REGION opcode's input contract: a rendered
// page image plus the bbox (in the page's own coordinate space) to crop,
// with an expansion margin so the crop still contains the evidence even
// under bbox imprecision.
type RegionCropInput struct {
	PageImagePath string             `json:"page_image_path"`
	PageWidth     float64            `json:"page_width"`
	PageHeight    float64            `json:"page_height"`
	BBox          *canonicaldoc.BBox `json:"bbox,omitempty"`
	MarginRatio   float64            `json:"margin_ratio,omitempty"`
	OutputPath    string             `json:"output_path"`
}

// RegionCropTlaloque deterministically crops a rendered page PNG to a bbox
// (or passes the full page through when BBox is nil — the honest REAL
// locate fallback). It never calls a generative model.
type RegionCropTlaloque struct{}

func (RegionCropTlaloque) Descriptor() tlaloque.CapabilityDescriptor {
	d, _ := tlaloque.CapabilityDescriptor{
		ID: RegionCropTlaloqueID, Capability: OpCropRegion, Engine: tlaloque.EngineDeterministic,
		InputSchema: "exocortex.region-crop-input.r0", OutputSchema: "exocortex.region-crop-output.r0",
		Deterministic: true, MaxConcurrency: 8,
	}.Normalize()
	return d
}

func (RegionCropTlaloque) Execute(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in RegionCropInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("region crop tlaloque: decode input: %w", err)
	}
	out, exposure, err := CropImageToBBox(in)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	value, _ := json.Marshal(map[string]any{"crop_path": in.OutputPath, "visual_exposure_ratio": exposure})
	body, _ := json.Marshal(map[string]any{"crop_path": in.OutputPath, "bytes": len(out), "visual_exposure_ratio": exposure})
	return tlaloque.CapabilityResponse{
		WorkerID: RegionCropTlaloqueID, Output: body, Confidence: 1,
		Observations: []blackboard.Observation{{Key: req.NodeID, Value: value, Confidence: 1, Provenance: map[string]string{"source": RegionCropTlaloqueID}}},
	}, nil
}

// CropImageToBBox crops pageImage (a PNG) to bbox expanded by marginRatio
// on each side, writes the crop to in.OutputPath, and returns the crop
// bytes plus visual_exposure_ratio = crop_pixels / full_page_pixels
// (section 19). A nil BBox passes the full page through untouched with
// exposure ratio 1.0 — the honest outcome when REAL locate could not
// resolve a tighter region.
func CropImageToBBox(in RegionCropInput) ([]byte, float64, error) {
	data, err := os.ReadFile(in.PageImagePath)
	if err != nil {
		return nil, 0, fmt.Errorf("crop region: read page image: %w", err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, fmt.Errorf("crop region: decode page image: %w", err)
	}
	bounds := img.Bounds()
	fullPixels := float64(bounds.Dx()) * float64(bounds.Dy())
	if in.BBox == nil || in.PageWidth <= 0 || in.PageHeight <= 0 {
		return data, 1.0, nil
	}
	margin := in.MarginRatio
	if margin < 0 {
		margin = 0
	}
	sx := float64(bounds.Dx()) / in.PageWidth
	sy := float64(bounds.Dy()) / in.PageHeight
	w := in.BBox.X2 - in.BBox.X1
	h := in.BBox.Y2 - in.BBox.Y1
	x1 := in.BBox.X1 - w*margin
	y1 := in.BBox.Y1 - h*margin
	x2 := in.BBox.X2 + w*margin
	y2 := in.BBox.Y2 + h*margin
	rect := image.Rect(
		bounds.Min.X+int(x1*sx), bounds.Min.Y+int(y1*sy),
		bounds.Min.X+int(x2*sx), bounds.Min.Y+int(y2*sy),
	).Intersect(bounds)
	if rect.Empty() {
		return nil, 0, fmt.Errorf("crop region: computed crop rectangle is empty")
	}
	cropped := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(cropped, cropped.Bounds(), img, rect.Min, draw.Src)
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, cropped); err != nil {
		return nil, 0, err
	}
	if in.OutputPath != "" {
		if err := os.WriteFile(in.OutputPath, buf.Bytes(), 0o644); err != nil {
			return nil, 0, fmt.Errorf("crop region: write output: %w", err)
		}
	}
	cropPixels := float64(rect.Dx()) * float64(rect.Dy())
	return buf.Bytes(), cropPixels / fullPixels, nil
}
