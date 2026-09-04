package tonalt1

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
)

// PriorInstance is one physical source instance consumed (selected and/or
// inferred on) by an earlier experiment. Absent identity fields are left
// zero — D3 never invents unavailable precision.
type PriorInstance struct {
	Experiment  string
	Page        int
	RegionID    string
	LineBBox    canonicaldoc.BBox
	HasLineBBox bool
	LineText    string
	CharStart   int
	CharEnd     int
	HasCharSpan bool
	CandidateID string
	// PageLevel marks instances where the earlier experiment exposed the
	// whole page image to the model (T0 full-page Q&A / capability
	// end-to-end image variant): any T1 operand on that page is excluded.
	PageLevel bool
}

// PriorUseIndex answers, for a T1 candidate, every reason it collides with
// a previously consumed physical instance.
type PriorUseIndex struct {
	Version string

	instances []PriorInstance

	byPage       map[int][]PriorInstance
	regionByPage map[int]map[string][]PriorInstance
	pageExposure map[int][]string // page -> experiments that exposed the whole page
	lineIDByPage map[int]map[string][]PriorInstance
	candByID     map[string][]PriorInstance

	// SourceCounts is per-experiment instance counts for the inventory
	// report.
	SourceCounts map[string]int
	// KeyAvailability is per-experiment counts of which identity keys were
	// reconstructable.
	KeyAvailability map[string]map[string]int
}

// pageVisualExposure is the frozen set of store pages whose full-page
// image was shown to Parrot by an earlier experiment (T0 P0 image dataset
// + parrot-capability-r0 end-to-end image variant). Every T1 operand on
// these pages is prior-used.
var pageVisualExposure = []int{43, 50, 66, 176, 200, 209, 275, 310, 641, 689}

// priorArtifact is one frozen file the harvester reads, with the extractor
// that reconstructs its consumed instances.
type priorArtifact struct {
	experiment string
	relPath    string
	extract    func(experiment string, raw []byte) ([]PriorInstance, error)
}

// priorArtifacts is the frozen, ordered inventory of prior experiments and
// artifacts (PriorUseInventoryVersion). Synthetic experiments
// (exocortex-t0a-r0, parrot-microisa-r0/r0.1, R1-C synthetic bases /
// glyphbank) consume NO real-document instance and are intentionally
// absent.
var priorArtifacts = []priorArtifact{
	{"R1-POOL", "experiments/parrot-perceptual-envelope-r1/datasets/SOURCE_POOL_R1.json", extractSourcePool},
	{"R1-A", "experiments/parrot-perceptual-envelope-r1/datasets/R1A_BASES.json", extractAllocationBases},
	{"R1-A1", "experiments/parrot-perceptual-envelope-r1/datasets/R1A1_BASES.json", extractAllocationBases},
	{"R1-B", "experiments/parrot-perceptual-envelope-r1/datasets/R1B_BASES.json", extractAllocationBases},
	{"R1-C", "experiments/parrot-perceptual-envelope-r1/datasets/R1C_REAL_BASES.json", extractStratifiedBases},
	{"R1-D", "experiments/parrot-perceptual-envelope-r1/datasets/R1D_REAL_BASES.json", extractFlatLabelValue},
	{"R1-E", "experiments/parrot-perceptual-envelope-r1/datasets/R1E_BASES.json", extractFlatLabelValue},
	{"R1-D-POOL", "experiments/parrot-perceptual-envelope-r1/datasets/R1D_POOL.json", extractLabelValuePool},
	{"R1-G", "experiments/parrot-perceptual-envelope-r1/datasets/R1G_SCALE_BASES.json", extractGBases},
	{"R1-G", "experiments/parrot-perceptual-envelope-r1/datasets/R1G_CONTEXT_BASES.json", extractGBases},
	{"R1-G", "experiments/parrot-perceptual-envelope-r1/datasets/R1G_CUE_BASES.json", extractGBases},
	{"R1-G", "experiments/parrot-perceptual-envelope-r1/datasets/R1G_REAL_ASSOC_BASES.json", extractGBases},
	{"PROFILE-H", "experiments/parrot-perceptual-envelope-r1/datasets/PROFILE_VALIDATION_H_R0.json", extractProfileH},
	{"T0-B", "experiments/exocortex-decomposition-r0/results/T0B_ORACLE_CROPS_R0.json", extractOracleCrops},
}

// nestedCandidate is the perceptenvelope.Candidate JSON shape as embedded
// in R1 base artifacts.
type nestedCandidate struct {
	CandidateID     string `json:"candidate_id"`
	Page            int    `json:"page"`
	CharOffsetStart int    `json:"char_offset_start"`
	CharOffsetEnd   int    `json:"char_offset_end"`
	Line            struct {
		RegionID string            `json:"region_id"`
		Text     string            `json:"text"`
		BBox     canonicaldoc.BBox `json:"bbox"`
	} `json:"line"`
}

func (nc nestedCandidate) toInstance(experiment string) PriorInstance {
	inst := PriorInstance{
		Experiment:  experiment,
		Page:        nc.Page,
		RegionID:    nc.Line.RegionID,
		LineText:    nc.Line.Text,
		CandidateID: nc.CandidateID,
	}
	if nc.Line.BBox != (canonicaldoc.BBox{}) {
		inst.LineBBox = nc.Line.BBox
		inst.HasLineBBox = true
	}
	if nc.CharOffsetEnd > nc.CharOffsetStart {
		inst.CharStart = nc.CharOffsetStart
		inst.CharEnd = nc.CharOffsetEnd
		inst.HasCharSpan = true
	}
	return inst
}

func extractSourcePool(experiment string, raw []byte) ([]PriorInstance, error) {
	var doc struct {
		Candidates []nestedCandidate `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	out := make([]PriorInstance, 0, len(doc.Candidates))
	for _, nc := range doc.Candidates {
		out = append(out, nc.toInstance(experiment))
	}
	return out, nil
}

func extractAllocationBases(experiment string, raw []byte) ([]PriorInstance, error) {
	var doc struct {
		Bases []struct {
			Candidate nestedCandidate `json:"candidate"`
		} `json:"bases"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	out := make([]PriorInstance, 0, len(doc.Bases))
	for _, base := range doc.Bases {
		out = append(out, base.Candidate.toInstance(experiment))
	}
	return out, nil
}

func extractStratifiedBases(experiment string, raw []byte) ([]PriorInstance, error) {
	var byStratum map[string][]struct {
		Family    string          `json:"family"`
		Candidate nestedCandidate `json:"candidate"`
	}
	if err := json.Unmarshal(raw, &byStratum); err != nil {
		return nil, err
	}
	strata := make([]string, 0, len(byStratum))
	for name := range byStratum {
		strata = append(strata, name)
	}
	sort.Strings(strata)
	var out []PriorInstance
	for _, name := range strata {
		for _, base := range byStratum[name] {
			inst := base.Candidate.toInstance(experiment)
			if inst.Page == 0 && inst.RegionID == "" && inst.CandidateID == "" {
				continue // e.g. COORDINATE_OR_TUPLE: null
			}
			out = append(out, inst)
		}
	}
	return out, nil
}

// flatLabelValue is the R1D_REAL_BASES / R1E_BASES flat shape.
type flatLabelValue struct {
	CandidateID    string            `json:"candidate_id"`
	Page           int               `json:"page"`
	LineText       string            `json:"line_text"`
	LineBBox       canonicaldoc.BBox `json:"line_bbox"`
	ValueRuneStart int               `json:"value_rune_start"`
	ValueRuneEnd   int               `json:"value_rune_end"`
	RegionID       string            `json:"region_id"`
}

func (f flatLabelValue) toInstance(experiment string) PriorInstance {
	inst := PriorInstance{
		Experiment:  experiment,
		Page:        f.Page,
		RegionID:    f.RegionID,
		LineText:    f.LineText,
		CandidateID: f.CandidateID,
	}
	if f.LineBBox != (canonicaldoc.BBox{}) {
		inst.LineBBox = f.LineBBox
		inst.HasLineBBox = true
	}
	if f.ValueRuneEnd > f.ValueRuneStart {
		inst.CharStart = f.ValueRuneStart
		inst.CharEnd = f.ValueRuneEnd
		inst.HasCharSpan = true
	}
	return inst
}

func extractFlatLabelValue(experiment string, raw []byte) ([]PriorInstance, error) {
	var list []flatLabelValue
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	out := make([]PriorInstance, 0, len(list))
	for _, item := range list {
		out = append(out, item.toInstance(experiment))
	}
	return out, nil
}

func extractLabelValuePool(experiment string, raw []byte) ([]PriorInstance, error) {
	var doc struct {
		Candidates []flatLabelValue `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	out := make([]PriorInstance, 0, len(doc.Candidates))
	for _, item := range doc.Candidates {
		out = append(out, item.toInstance(experiment))
	}
	return out, nil
}

// extractGBases handles every R1-G base file: a flat list whose items each
// carry candidate_id/page and optionally a nested `candidate`.
func extractGBases(experiment string, raw []byte) ([]PriorInstance, error) {
	var list []struct {
		CandidateID string          `json:"candidate_id"`
		Page        int             `json:"page"`
		Candidate   nestedCandidate `json:"candidate"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	out := make([]PriorInstance, 0, len(list))
	for _, item := range list {
		if item.Candidate.Page != 0 || item.Candidate.Line.RegionID != "" {
			out = append(out, item.Candidate.toInstance(experiment))
			continue
		}
		out = append(out, PriorInstance{
			Experiment:  experiment,
			Page:        item.Page,
			CandidateID: item.CandidateID,
		})
	}
	return out, nil
}

func extractProfileH(experiment string, raw []byte) ([]PriorInstance, error) {
	var doc struct {
		SharedHeldoutBases []struct {
			CandidateID string          `json:"candidate_id"`
			Page        int             `json:"page"`
			Candidate   nestedCandidate `json:"candidate"`
		} `json:"shared_heldout_bases"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	out := make([]PriorInstance, 0, len(doc.SharedHeldoutBases))
	for _, base := range doc.SharedHeldoutBases {
		if base.Candidate.Page != 0 || base.Candidate.Line.RegionID != "" {
			out = append(out, base.Candidate.toInstance(experiment))
			continue
		}
		out = append(out, PriorInstance{Experiment: experiment, Page: base.Page, CandidateID: base.CandidateID})
	}
	return out, nil
}

func extractOracleCrops(experiment string, raw []byte) ([]PriorInstance, error) {
	var doc struct {
		Crops []struct {
			Page             int      `json:"page"`
			MatchedRegionIDs []string `json:"matched_region_ids"`
		} `json:"crops"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	var out []PriorInstance
	for _, crop := range doc.Crops {
		for _, regionID := range crop.MatchedRegionIDs {
			out = append(out, PriorInstance{Experiment: experiment, Page: crop.Page, RegionID: regionID})
		}
	}
	return out, nil
}

// LoadPriorUseIndex reads every frozen prior artifact under root and
// builds the index. Fails closed: a missing / unparseable artifact is an
// error, never a silent skip.
func LoadPriorUseIndex(root string) (*PriorUseIndex, error) {
	index := &PriorUseIndex{
		Version:         PriorUseInventoryVersion,
		byPage:          map[int][]PriorInstance{},
		regionByPage:    map[int]map[string][]PriorInstance{},
		pageExposure:    map[int][]string{},
		lineIDByPage:    map[int]map[string][]PriorInstance{},
		candByID:        map[string][]PriorInstance{},
		SourceCounts:    map[string]int{},
		KeyAvailability: map[string]map[string]int{},
	}

	for _, artifact := range priorArtifacts {
		full := filepath.Join(root, filepath.FromSlash(artifact.relPath))
		raw, err := os.ReadFile(full)
		if err != nil {
			return nil, fmt.Errorf("prior artifact %s (%s): %w", artifact.experiment, artifact.relPath, err)
		}
		instances, err := artifact.extract(artifact.experiment, raw)
		if err != nil {
			return nil, fmt.Errorf("prior artifact %s (%s): extract: %w", artifact.experiment, artifact.relPath, err)
		}
		for _, inst := range instances {
			index.add(inst)
		}
	}

	// Page visual exposure (T0 full-page image Q&A + capability end-to-end
	// image variant). One synthetic PageLevel instance per page.
	for _, page := range pageVisualExposure {
		index.add(PriorInstance{Experiment: "T0-P0/CAPABILITY", Page: page, PageLevel: true})
	}

	return index, nil
}

func (index *PriorUseIndex) add(inst PriorInstance) {
	if inst.Page == 0 && inst.CandidateID == "" {
		return
	}
	index.instances = append(index.instances, inst)
	index.SourceCounts[inst.Experiment]++

	keys := index.KeyAvailability[inst.Experiment]
	if keys == nil {
		keys = map[string]int{}
		index.KeyAvailability[inst.Experiment] = keys
	}
	if inst.PageLevel {
		keys["page_visual_exposure"]++
		index.pageExposure[inst.Page] = appendUnique(index.pageExposure[inst.Page], inst.Experiment)
		return
	}
	if inst.RegionID != "" {
		keys["page+region_id"]++
		byRegion := index.regionByPage[inst.Page]
		if byRegion == nil {
			byRegion = map[string][]PriorInstance{}
			index.regionByPage[inst.Page] = byRegion
		}
		byRegion[inst.RegionID] = append(byRegion[inst.RegionID], inst)
	}
	if inst.HasLineBBox {
		keys["page+bbox"]++
	}
	if inst.HasCharSpan {
		keys["page+char_span"]++
	}
	if inst.LineText != "" {
		keys["span_hash"]++
		lineID := lineIdentityHash(inst.Page, inst.LineText)
		byLine := index.lineIDByPage[inst.Page]
		if byLine == nil {
			byLine = map[string][]PriorInstance{}
			index.lineIDByPage[inst.Page] = byLine
		}
		byLine[lineID] = append(byLine[lineID], inst)
	}
	if inst.CandidateID != "" {
		keys["candidate_id"]++
		index.candByID[inst.CandidateID] = append(index.candByID[inst.CandidateID], inst)
	}
	index.byPage[inst.Page] = append(index.byPage[inst.Page], inst)
}

// Match returns every reason cand collides with a prior-used instance,
// deduped by (experiment, key) and deterministically ordered. A match on
// ANY available physical identity key excludes the candidate.
func (index *PriorUseIndex) Match(cand Candidate) []PriorUseMatch {
	seen := map[string]PriorUseMatch{}
	record := func(experiment, key, detail string) {
		compound := experiment + "|" + key
		if _, ok := seen[compound]; !ok {
			seen[compound] = PriorUseMatch{Experiment: experiment, Key: key, Detail: detail}
		}
	}

	page := cand.Corpus.Page

	for _, experiment := range index.pageExposure[page] {
		record(experiment, "page_visual_exposure", fmt.Sprintf("page %d full-image exposure", page))
	}

	if cand.Identity.RegionID != "" {
		for _, inst := range index.regionByPage[page][cand.Identity.RegionID] {
			record(inst.Experiment, "page+region_id", "region "+cand.Identity.RegionID)
		}
	}

	candLineID := lineIdentityHash(page, cand.Source.ContainingLineText)
	for _, inst := range index.lineIDByPage[page][candLineID] {
		if inst.HasCharSpan && cand.Identity.CharEnd > cand.Identity.CharStart {
			if spansOverlap(cand.Identity.CharStart, cand.Identity.CharEnd, inst.CharStart, inst.CharEnd) {
				record(inst.Experiment, "page+char_span", "overlapping token span on identical line")
			} else {
				record(inst.Experiment, "span_hash", "identical containing line, non-overlapping span")
			}
		} else {
			record(inst.Experiment, "span_hash", "identical containing line")
		}
	}

	candBox := cand.Geometry.ContainingLineBBox
	for _, inst := range index.byPage[page] {
		if inst.HasLineBBox && bboxAlmostEqual(candBox, inst.LineBBox, 1.0) {
			record(inst.Experiment, "page+bbox", "identical containing-line bbox "+bboxKey(candBox))
		}
	}

	for _, inst := range index.candByID[cand.CandidateID] {
		record(inst.Experiment, "candidate_id", cand.CandidateID)
	}

	compounds := make([]string, 0, len(seen))
	for compound := range seen {
		compounds = append(compounds, compound)
	}
	sort.Strings(compounds)
	out := make([]PriorUseMatch, 0, len(compounds))
	for _, compound := range compounds {
		out = append(out, seen[compound])
	}
	return out
}

// InstanceCount is the total number of harvested prior instances.
func (index *PriorUseIndex) InstanceCount() int { return len(index.instances) }

// Artifacts returns the frozen ordered list of (experiment, relPath) pairs
// the harvester reads, for the inventory report and hashing.
func Artifacts() [][2]string {
	out := make([][2]string, 0, len(priorArtifacts))
	for _, artifact := range priorArtifacts {
		out = append(out, [2]string{artifact.experiment, artifact.relPath})
	}
	return out
}

// PageVisualExposure returns the frozen page-exposure set.
func PageVisualExposure() []int {
	out := append([]int(nil), pageVisualExposure...)
	sort.Ints(out)
	return out
}

func appendUnique(list []string, value string) []string {
	for _, existing := range list {
		if existing == value {
			return list
		}
	}
	return append(list, value)
}

// PriorInventoryHash is a stable hash of the frozen prior-use inventory
// definition (versions + artifact list + page-exposure set). It does NOT
// depend on the store.
func (index *PriorUseIndex) PriorInventoryHash() string {
	var builder strings.Builder
	builder.WriteString(PriorUseInventoryVersion)
	builder.WriteString(SpanNormVersion)
	for _, artifact := range priorArtifacts {
		builder.WriteString("\n")
		builder.WriteString(artifact.experiment)
		builder.WriteString(" ")
		builder.WriteString(artifact.relPath)
	}
	builder.WriteString("\npages:")
	for _, page := range PageVisualExposure() {
		builder.WriteString(" ")
		builder.WriteString(itoa(page))
	}
	return hashString(builder.String())
}
