package tonalt1

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"tlaloc.local/behaviorlab/internal/parrotlab"
	"tlaloc.local/behaviorlab/internal/parrotpresent"
)

// FrozenWorkflow is one D4 workflow as recorded in
// experiments/tonal-t1/d4/T1_D4_WORKFLOWS.json. It is loaded verbatim; no
// field is recomputed here.
type FrozenWorkflow struct {
	WorkflowID        string          `json:"workflow_id"`
	Shape             string          `json:"shape"`
	NaturalDepth      int             `json:"natural_depth"`
	CriticalPathDepth int             `json:"critical_path_depth"`
	Operands          []FrozenOperand `json:"operands"`
	DistinctPages     []int           `json:"distinct_pages"`
	WorkflowHash      string          `json:"workflow_hash"`
	Gold              *float64        `json:"gold"`
}

// FrozenOperand is one role assignment inside a FrozenWorkflow.
type FrozenOperand struct {
	CandidateID           string  `json:"candidate_id"`
	Role                  string  `json:"role"`
	NumericValue          float64 `json:"numeric_value"`
	MorphologyFamily      string  `json:"morphology_family"`
	Page                  int     `json:"page"`
	RegionID              string  `json:"region_id"`
	OperandHash           string  `json:"operand_hash"`
	EligibleAsDenominator bool    `json:"eligible_as_denominator"`
}

// LoadFrozenWorkflows loads the frozen D4 workflow allocation from disk. It
// performs no selection, allocation or mutation — the 60 workflows and 144
// operand-role assignments are used exactly as recorded.
func LoadFrozenWorkflows(path string) ([]FrozenWorkflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("tonalt1: read frozen workflows %s: %w", path, err)
	}
	var workflows []FrozenWorkflow
	if err := json.Unmarshal(data, &workflows); err != nil {
		return nil, fmt.Errorf("tonalt1: parse frozen workflows: %w", err)
	}
	if len(workflows) != WorkflowCount {
		return nil, fmt.Errorf("tonalt1: expected %d frozen workflows, got %d", WorkflowCount, len(workflows))
	}
	total := 0
	seen := map[string]bool{}
	for _, w := range workflows {
		for _, op := range w.Operands {
			total++
			if seen[op.CandidateID] {
				return nil, fmt.Errorf("tonalt1: duplicate candidate_id %s across frozen workflows (reuse forbidden)", op.CandidateID)
			}
			seen[op.CandidateID] = true
		}
	}
	if total != OperandCount {
		return nil, fmt.Errorf("tonalt1: expected %d frozen operand assignments, got %d", OperandCount, total)
	}
	return workflows, nil
}

// ResolvedOperand is one frozen operand-role assignment joined against its
// full Candidate record from the frozen primary universe.
type ResolvedOperand struct {
	WorkflowID  string
	Shape       string
	Role        string
	CandidateID string
	Candidate   Candidate
}

// ValidatedAllocation is the frozen D4 allocation after candidate
// resolution: every operand joined to its full frozen geometry, plus the
// derived unique page set (computed from the resolved candidates only —
// never from the eligible-page scan or the primary-universe superset).
type ValidatedAllocation struct {
	Operands    []ResolvedOperand
	UniquePages []int
}

// ValidateOperandsInPrimaryUniverse resolves every frozen D4 candidate_id
// against the frozen fresh-corpus primary universe
// (experiments/tonal-t1/fresh-corpus/FRESH_PRIMARY_UNIVERSE.json) and derives
// the authoritative unique page set from those 144 resolved candidates.
func ValidateOperandsInPrimaryUniverse(workflows []FrozenWorkflow) (*ValidatedAllocation, error) {
	universe, err := loadPrimaryUniverse(repoPath("experiments/tonal-t1/fresh-corpus/FRESH_PRIMARY_UNIVERSE.json"))
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Candidate, len(universe))
	for _, c := range universe {
		byID[c.CandidateID] = c
	}

	var resolved []ResolvedOperand
	pageSet := map[int]bool{}
	for _, w := range workflows {
		for _, op := range w.Operands {
			cand, ok := byID[op.CandidateID]
			if !ok {
				return nil, fmt.Errorf("tonalt1: candidate_id %s (workflow %s role %s) not found in frozen primary universe", op.CandidateID, w.WorkflowID, op.Role)
			}
			if cand.Corpus.Page != op.Page {
				return nil, fmt.Errorf("tonalt1: candidate %s page mismatch: D4=%d universe=%d", op.CandidateID, op.Page, cand.Corpus.Page)
			}
			resolved = append(resolved, ResolvedOperand{
				WorkflowID:  w.WorkflowID,
				Shape:       w.Shape,
				Role:        op.Role,
				CandidateID: op.CandidateID,
				Candidate:   cand,
			})
			pageSet[cand.Corpus.Page] = true
		}
	}
	if len(resolved) != OperandCount {
		return nil, fmt.Errorf("tonalt1: resolved %d/%d operands", len(resolved), OperandCount)
	}

	pages := make([]int, 0, len(pageSet))
	for p := range pageSet {
		pages = append(pages, p)
	}
	sort.Ints(pages)

	return &ValidatedAllocation{Operands: resolved, UniquePages: pages}, nil
}

// loadPrimaryUniverse loads the frozen fresh-corpus candidate universe.
func loadPrimaryUniverse(path string) ([]Candidate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("tonalt1: read primary universe %s: %w", path, err)
	}
	var doc struct {
		Operands []Candidate `json:"operands"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("tonalt1: parse primary universe: %w", err)
	}
	return doc.Operands, nil
}

// RasterizedPage is one page rendered fresh from the source PDF at the
// frozen 150 DPI.
type RasterizedPage struct {
	Page   int
	PNG    []byte
	SHA256 string
}

// RasterizePages renders exactly the given pages fresh from the frozen
// source PDF via the real production rasteriser
// (parrotlab.NewPDFMemoryProvider -> RenderPNG, which shells to
// `pdftoppm -png -r 150`). No page outside this set is rendered; no cached
// bridge/R1 page PNG is reused.
func RasterizePages(pages []int) (map[int]RasterizedPage, error) {
	provider, err := parrotlab.NewPDFMemoryProvider(repoPath(FreshCorpusStoreDir), SourcePDFPath)
	if err != nil {
		return nil, fmt.Errorf("tonalt1: open page provider: %w", err)
	}

	out := make(map[int]RasterizedPage, len(pages))
	for _, page := range pages {
		data, err := provider.RenderPNG(page)
		if err != nil {
			return nil, fmt.Errorf("tonalt1: rasterize page %d: %w", page, err)
		}
		sum := sha256.Sum256(data)
		out[page] = RasterizedPage{Page: page, PNG: data, SHA256: hex.EncodeToString(sum[:])}
	}
	if len(out) != len(pages) {
		return nil, fmt.Errorf("tonalt1: rasterized %d/%d pages", len(out), len(pages))
	}
	return out, nil
}

// OperandImageRecord is the full provenance chain for one materialized
// operand image: source PDF -> page raster -> frozen token bbox ->
// parrotpresent.Prepare -> prepared PNG.
type OperandImageRecord struct {
	WorkflowID          string     `json:"workflow_id"`
	Role                string     `json:"role"`
	CandidateID         string     `json:"candidate_id"`
	Page                int        `json:"page"`
	RegionID            string     `json:"region_id"`
	OperandBBoxEstimate [4]float64 `json:"operand_bbox_estimate"`
	PageSHA256          string     `json:"page_sha256"`
	PreparedSHA256      string     `json:"prepared_sha256"`
	PreparedBytes       int        `json:"prepared_bytes"`
	Width               int        `json:"width"`
	Height              int        `json:"height"`
	RendererPath        string     `json:"renderer_path"`
}

// GenerateOperandPresentations renders all 144 frozen operands through the
// real production parrotpresent.Prepare, cropping from the actual page
// rasters at each candidate's frozen geometry.operand_bbox_estimate. It
// writes each page raster to a temp file (Prepare reads from a path) and
// invokes the unmodified renderer — no reimplementation.
func GenerateOperandPresentations(allocation *ValidatedAllocation, pages map[int]RasterizedPage) (map[string][]byte, []OperandImageRecord, error) {
	tmpDir, err := os.MkdirTemp("", "tonalt1-pageraster-")
	if err != nil {
		return nil, nil, fmt.Errorf("tonalt1: temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	pagePaths := make(map[int]string, len(pages))
	for page, raster := range pages {
		path := fmt.Sprintf("%s/page-%d.png", tmpDir, page)
		if err := os.WriteFile(path, raster.PNG, 0o644); err != nil {
			return nil, nil, fmt.Errorf("tonalt1: write page raster %d: %w", page, err)
		}
		pagePaths[page] = path
	}

	images := make(map[string][]byte, len(allocation.Operands))
	var records []OperandImageRecord

	for _, resolved := range allocation.Operands {
		cand := resolved.Candidate
		raster, ok := pages[cand.Corpus.Page]
		if !ok {
			return nil, nil, fmt.Errorf("tonalt1: no rasterized page %d for candidate %s", cand.Corpus.Page, cand.CandidateID)
		}
		pagePath := pagePaths[cand.Corpus.Page]

		bbox := cand.Geometry.OperandBBoxEstimate
		lineBBox := cand.Geometry.ContainingLineBBox
		lineHeight := lineBBox.Y2 - lineBBox.Y1

		op := parrotpresent.Operand{
			TokenBBox:      parrotpresent.BBox{X1: bbox.X1, Y1: bbox.Y1, X2: bbox.X2, Y2: bbox.Y2},
			LineHeight:     lineHeight,
			LineText:       cand.Source.ContainingLineText,
			PageWidthStore: cand.Geometry.PageWidth,
		}
		plan := parrotpresent.Plan{TargetLineHeightPx: TargetLineHeightPx, DrawCue: DrawCue}

		result, err := parrotpresent.Prepare(pagePath, op, plan, "")
		if err != nil {
			return nil, nil, fmt.Errorf("tonalt1: parrotpresent.Prepare candidate %s: %w", cand.CandidateID, err)
		}

		// Decode to verify dimensions and reject the known synthetic
		// placeholder signature (uniform gray + magenta rectangle only).
		img, err := png.Decode(bytes.NewReader(result.Bytes))
		if err != nil {
			return nil, nil, fmt.Errorf("tonalt1: decode prepared PNG for %s: %w", cand.CandidateID, err)
		}
		bounds := img.Bounds()
		if bounds.Dx() != CanvasPx || bounds.Dy() != CanvasPx {
			return nil, nil, fmt.Errorf("tonalt1: candidate %s prepared image is %dx%d, want %dx%d", cand.CandidateID, bounds.Dx(), bounds.Dy(), CanvasPx, CanvasPx)
		}
		if isKnownPlaceholderSignature(img) {
			return nil, nil, fmt.Errorf("tonalt1: candidate %s prepared image matches known invalid placeholder signature (uniform gray + magenta rectangle only)", cand.CandidateID)
		}

		images[resolved.WorkflowID+"|"+resolved.Role] = result.Bytes
		records = append(records, OperandImageRecord{
			WorkflowID:          resolved.WorkflowID,
			Role:                resolved.Role,
			CandidateID:         cand.CandidateID,
			Page:                cand.Corpus.Page,
			RegionID:            cand.Identity.RegionID,
			OperandBBoxEstimate: [4]float64{bbox.X1, bbox.Y1, bbox.X2, bbox.Y2},
			PageSHA256:          raster.SHA256,
			PreparedSHA256:      result.SHA256,
			PreparedBytes:       len(result.Bytes),
			Width:               bounds.Dx(),
			Height:              bounds.Dy(),
			RendererPath:        "internal/parrotpresent/prepare.go:Prepare",
		})
	}

	if len(images) != OperandCount {
		return nil, nil, fmt.Errorf("tonalt1: generated %d/%d operand images", len(images), OperandCount)
	}
	return images, records, nil
}

// isKnownPlaceholderSignature detects the specific invalid fixture produced
// by earlier attempts: a canvas filled entirely with the neutral mask
// background color plus a magenta stroke, and nothing else (fewer than 8
// distinct colors total). A real page-derived crop has far more.
func isKnownPlaceholderSignature(img image.Image) bool {
	seen := map[[4]uint32]bool{}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y && len(seen) < 8; y++ {
		for x := bounds.Min.X; x < bounds.Max.X && len(seen) < 8; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			seen[[4]uint32{r, g, b, a}] = true
		}
	}
	return len(seen) < 8
}

// CompositeRecord is the full provenance for one Arm-A composite.
type CompositeRecord struct {
	WorkflowID       string   `json:"workflow_id"`
	OrderedRoles     []string `json:"ordered_roles"`
	OrderedOperandID []string `json:"ordered_candidate_ids"`
	OperandSHA256    []string `json:"operand_sha256"`
	Width            int      `json:"width"`
	Height           int      `json:"height"`
	PaddingPx        int      `json:"padding_px"`
	CompositeSHA256  string   `json:"composite_sha256"`
}

// armAPaddingPx is the frozen Arm-A layout padding. T1_D5_ARM_A_POLICY.json
// specifies "stack_crops_vertically_with_padding" without a pixel value;
// the padding amount is not separately frozen anywhere in the repository.
// GenerateArmAComposites therefore uses zero padding (pure vertical
// concatenation, the only padding value that requires no invention) and
// records this explicitly rather than picking an arbitrary constant.
const armAPaddingPx = 0

// GenerateArmAComposites builds the 60 Arm-A composites by vertically
// stacking each workflow's real prepared operand PNGs (alphabetical role
// order, per T1_D5_ARM_A_POLICY.json's operand_crop_order), pixel-for-pixel,
// with zero rescaling and zero interpolation.
func GenerateArmAComposites(workflows []FrozenWorkflow, operandImages map[string][]byte) (map[string][]byte, []CompositeRecord, error) {
	composites := make(map[string][]byte, len(workflows))
	var records []CompositeRecord

	for _, w := range workflows {
		roles := make([]string, 0, len(w.Operands))
		byRole := map[string]FrozenOperand{}
		for _, op := range w.Operands {
			roles = append(roles, op.Role)
			byRole[op.Role] = op
		}
		sort.Strings(roles) // frozen policy: operand_crop_order = by_role_alphabetical

		var frames []image.Image
		var shas []string
		var candIDs []string
		for _, role := range roles {
			key := w.WorkflowID + "|" + role
			data, ok := operandImages[key]
			if !ok {
				return nil, nil, fmt.Errorf("tonalt1: missing prepared operand image for %s", key)
			}
			img, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				return nil, nil, fmt.Errorf("tonalt1: decode operand %s: %w", key, err)
			}
			frames = append(frames, img)
			sum := sha256.Sum256(data)
			shas = append(shas, hex.EncodeToString(sum[:]))
			candIDs = append(candIDs, byRole[role].CandidateID)
		}

		width := CanvasPx
		height := len(frames)*CanvasPx + (len(frames)-1)*armAPaddingPx
		canvas := image.NewRGBA(image.Rect(0, 0, width, height))
		y := 0
		for _, frame := range frames {
			drawExact(canvas, frame, 0, y)
			y += CanvasPx + armAPaddingPx
		}

		var buf bytes.Buffer
		if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&buf, canvas); err != nil {
			return nil, nil, fmt.Errorf("tonalt1: encode composite %s: %w", w.WorkflowID, err)
		}
		sum := sha256.Sum256(buf.Bytes())

		composites[w.WorkflowID] = buf.Bytes()
		records = append(records, CompositeRecord{
			WorkflowID:       w.WorkflowID,
			OrderedRoles:     roles,
			OrderedOperandID: candIDs,
			OperandSHA256:    shas,
			Width:            width,
			Height:           height,
			PaddingPx:        armAPaddingPx,
			CompositeSHA256:  hex.EncodeToString(sum[:]),
		})
	}

	if len(composites) != WorkflowCount {
		return nil, nil, fmt.Errorf("tonalt1: generated %d/%d composites", len(composites), WorkflowCount)
	}
	return composites, records, nil
}

// drawExact copies src into dst at (x0,y0) with no scaling or interpolation
// — a direct pixel copy, preserving every source byte exactly.
func drawExact(dst *image.RGBA, src image.Image, x0, y0 int) {
	b := src.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x0+(x-b.Min.X), y0+(y-b.Min.Y), src.At(x, y))
		}
	}
}

// VerifyNoLeakage re-checks the resolved 144 candidates (not the full 405
// candidate universe) against bridge/prior-use exclusion and morphology
// constraints.
func VerifyNoLeakage(allocation *ValidatedAllocation) error {
	seenCandidate := map[string]bool{}
	seenIdentity := map[string]bool{}
	for _, r := range allocation.Operands {
		c := r.Candidate
		if seenCandidate[c.CandidateID] {
			return fmt.Errorf("tonalt1: duplicate candidate_id %s in resolved allocation", c.CandidateID)
		}
		seenCandidate[c.CandidateID] = true

		identityKey := fmt.Sprintf("%d|%s", c.Identity.Page, c.Identity.RegionID)
		if seenIdentity[identityKey] {
			return fmt.Errorf("tonalt1: duplicate physical identity %s", identityKey)
		}
		seenIdentity[identityKey] = true

		if c.PriorUse.Excluded {
			return fmt.Errorf("tonalt1: candidate %s has prior_use.excluded=true", c.CandidateID)
		}
		if c.Presentation.MorphologyFamily != MorphMultiDigitInteger && c.Presentation.MorphologyFamily != MorphDecimal {
			return fmt.Errorf("tonalt1: candidate %s morphology %s not in allowed set", c.CandidateID, c.Presentation.MorphologyFamily)
		}
	}
	if len(seenCandidate) != OperandCount {
		return fmt.Errorf("tonalt1: no-leakage audit saw %d/%d unique candidates", len(seenCandidate), OperandCount)
	}
	return nil
}

// CallBudget is the frozen per-arm model-call count, derived mechanically
// from the workflow shapes (Arm A = 1 monolithic call per workflow; Arm C =
// 1 EXTRACT_NUMBER call per operand-role; Arm B is the independently frozen
// per-capability DAG-walk count carried in raster_config.go, not
// recomputed here since it depends on the Arm-B adapter DAG, not on the
// workflow list alone).
type CallBudget struct {
	ArmA  int
	ArmB  int
	ArmC  int
	Total int
}

// DeriveCallBudget derives Arm A (one call per workflow) and Arm C (one
// call per operand-role) mechanically from the actual frozen workflow list,
// and carries the independently frozen Arm B figure.
func DeriveCallBudget(workflows []FrozenWorkflow) (CallBudget, error) {
	armA := len(workflows)
	armC := 0
	for _, w := range workflows {
		armC += len(w.Operands)
	}
	budget := CallBudget{ArmA: armA, ArmB: ArmBExpectedCalls, ArmC: armC, Total: armA + ArmBExpectedCalls + armC}
	if budget.ArmA != ArmAExpectedCalls || budget.ArmC != ArmCExpectedCalls || budget.Total != PrimaryTotalCalls {
		return budget, fmt.Errorf("tonalt1: derived call budget %d/%d/%d/%d does not match frozen 60/492/144/696", budget.ArmA, budget.ArmB, budget.ArmC, budget.Total)
	}
	return budget, nil
}

// RunTransportFailClosedTests runs the real fail-closed transport test
// suite (internal/target/t1_transport_failclosed_test.go) against a local
// httptest server — no LM Studio contact, zero model HTTP calls. It shells
// to `go test` on that one package/pattern and reports the actual pass/fail,
// rather than re-implementing a parallel fake-test suite.
func RunTransportFailClosedTests() error {
	cmd := exec.Command("go", "test", "-run", "^TestT1Transport_", "-v", "./internal/target/...")
	cmd.Dir = moduleRoot()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tonalt1: transport fail-closed tests failed:\n%s", string(output))
	}
	return nil
}

// moduleRoot walks up from the current working directory to the directory
// containing this module's go.mod, so repo-relative paths (frozen JSON
// manifests, the fresh-corpus store) resolve correctly whether the caller
// is `go run`/the built CLI (cwd = module root) or `go test` (cwd = the
// package directory).
func moduleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "." // reached filesystem root without finding go.mod
		}
		dir = parent
	}
}

// RepoPath resolves a path relative to the module root, so frozen artifact
// paths work regardless of the caller's working directory (the built CLI,
// `go test`, or a subpackage).
func RepoPath(relative string) string {
	return filepath.Join(moduleRoot(), relative)
}

// repoPath is the unexported alias used internally in this file.
func repoPath(relative string) string { return RepoPath(relative) }
