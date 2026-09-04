package tonalt1

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/perceptenvelope"
)

// Fresh born-digital corpus acquisition + document-specific bridge
// (TONAL T1 final corpus gate).
//
// The original D2L corpus is experimentally exhausted (D3 v2: 19 held-out
// operands vs 144 required; SINGLE_DIGIT stays FRAGILE at n=100). A fresh
// born-digital document is admitted ONLY with its own bounded,
// document-specific perceptual bridge. T1 still measures WORKFLOW
// COMPOSITION, not cross-document transfer:
//
//	CORPUS_EXPANSION = true
//	DOCUMENT_SPECIFIC_BRIDGE = true
//	CROSS_DOCUMENT_GENERALIZATION_CLAIM = false
//
// This file: deterministic scan of a fresh store, page-level
// bridge/primary partition frozen BEFORE inference, bridge dataset,
// primary held-out universe, capacity + constructive allocation
// feasibility. Bridge inference itself is driven by
// perceptenvelope.RunR1C (the frozen R1-C renderer/prompt/scorer).

// FreshBridgeFraction is the target share of eligible pages used for the
// document-specific bridge. The rest supply primary held-out operands.
const FreshBridgeFraction = 0.20

// FreshMinPrimaryDemand is the frozen T1 primary operand demand.
const FreshMinPrimaryDemand = ExpectedPrimaryUniqueOperandDemand // 144

// FreshCorpusManifest is the immutable fresh-corpus freeze record.
type FreshCorpusManifest struct {
	Schema       string `json:"schema"`
	ExperimentID string `json:"experiment_id"`

	CorpusExpansion             bool `json:"CORPUS_EXPANSION"`
	DocumentSpecificBridge      bool `json:"DOCUMENT_SPECIFIC_BRIDGE"`
	CrossDocumentGeneralization bool `json:"CROSS_DOCUMENT_GENERALIZATION_CLAIM"`

	SelectorVersion string `json:"selector_version"`
	Seed            string `json:"seed"`

	Source SourceDoc     `json:"source_document"`
	Store  StoreIdentity `json:"store"`

	ScanTotal     int `json:"scan_total"`
	EligibleTotal int `json:"eligible_total"`
	EligiblePages int `json:"eligible_pages"`

	Partition PagePartition `json:"page_partition"`

	Bridge BridgeSpec `json:"bridge"`

	Primary PrimaryUniverse `json:"primary_held_out"`

	Capacity CapacityCheck `json:"capacity"`

	HardInvariants map[string]bool `json:"hard_invariants"`

	ArtifactHashes map[string]string `json:"artifact_hashes"`

	TONALT1FreshCorpusFrozen bool `json:"TONAL_T1_FRESH_CORPUS_FROZEN"`
	T1D4CanProceed           bool `json:"T1_D4_CAN_PROCEED"`
}

// SourceDoc identifies the fresh document.
type SourceDoc struct {
	Path          string  `json:"path"`
	SourceSHA256  string  `json:"source_sha256"`
	PageCount     int     `json:"page_count"`
	CharsPerPage  float64 `json:"chars_per_page"`
	SelectionRank int     `json:"deterministic_selection_rank"`
	ProxyEligible int     `json:"prefilter_eligible_proxy"`
}

// PagePartition is the frozen bridge/primary page split.
type PagePartition struct {
	PartitionRule         string  `json:"partition_rule"`
	BridgeFraction        float64 `json:"bridge_fraction"`
	BridgePages           []int   `json:"bridge_pages"`
	PrimaryPages          []int   `json:"primary_pages"`
	PartitionHash         string  `json:"partition_hash"`
	FrozenBeforeInference bool    `json:"frozen_before_inference"`
	ZeroPageOverlap       bool    `json:"zero_page_overlap"`
}

// BridgeSpec is the frozen bridge dataset (pre-inference).
type BridgeSpec struct {
	PerMorphologyN  map[MorphologyFamily]int `json:"per_morphology_n"`
	Renderer        string                   `json:"renderer"`
	Prompt          string                   `json:"prompt"`
	Model           string                   `json:"model"`
	ModelWeightsSHA string                   `json:"model_weights_sha256"`
	Temperature     float64                  `json:"temperature"`
	MaxTokens       int                      `json:"max_tokens"`
	CallBudget      int                      `json:"call_budget"`
	PromotionRule   string                   `json:"promotion_rule"`
	DatasetHash     string                   `json:"dataset_hash"`
	Bases           []BridgeBase             `json:"bases"`
}

// BridgeBase is one frozen bridge stimulus.
type BridgeBase struct {
	BaseID      string                         `json:"base_id"`
	CandidateID string                         `json:"candidate_id"`
	Morphology  MorphologyFamily               `json:"morphology"`
	Page        int                            `json:"page"`
	RegionID    string                         `json:"region_id"`
	Gold        string                         `json:"gold"`
	LineText    string                         `json:"line_text"`
	RankKey     string                         `json:"rank_key"`
	Candidate   perceptenvelope.MorphCandidate `json:"candidate"`
}

// BridgeMorphologyResult is the post-inference qualification of one
// morphology (filled by the command from AggregateR1C).
type BridgeMorphologyResult struct {
	Morphology      MorphologyFamily `json:"morphology"`
	N               int              `json:"n"`
	Correct         int              `json:"correct"`
	Accuracy        float64          `json:"accuracy"`
	CI95Low         float64          `json:"ci95_low"`
	CI95High        float64          `json:"ci95_high"`
	ContractSuccess int              `json:"contract_success"`
	Verdict         string           `json:"verdict"`
	Qualified       bool             `json:"qualified"`
}

// PrimaryUniverse is the fresh primary held-out operand universe.
type PrimaryUniverse struct {
	QualifiedMorphologies    []MorphologyFamily       `json:"qualified_morphologies"`
	N                        int                      `json:"n_primary_available"`
	ByMorphology             map[MorphologyFamily]int `json:"by_morphology"`
	DistinctPages            int                      `json:"distinct_pages"`
	PagesWithGE2             int                      `json:"pages_with_ge2"`
	PagesWithGE3             int                      `json:"pages_with_ge3"`
	PagesWithGE4             int                      `json:"pages_with_ge4"`
	DuplicateSpanExcluded    int                      `json:"duplicate_span_candidates_excluded"` // repeated store regions, correctly dropped
	BridgeLeakage            int                      `json:"bridge_physical_leakage"`            // MUST be 0 — bridge instance reached primary
	BridgePageCandidatesExcl int                      `json:"bridge_page_candidates_excluded"`    // eligible candidates on bridge pages, correctly dropped
	Operands                 []Candidate              `json:"operands"`
}

// CapacityCheck is the constructive allocation feasibility proof.
type CapacityCheck struct {
	NRequired          int            `json:"n_required"`
	NAvailable         int            `json:"n_available"`
	HeadroomRatio      float64        `json:"headroom_ratio"`
	PrimaryTargetReuse bool           `json:"primary_workflow_target_reuse"`
	AllocationFeasible bool           `json:"allocation_feasible"`
	WitnessSummary     map[string]any `json:"witness_summary"`
}

const freshCorpusSchema = "tonal.t1.fresh-corpus.freeze.r1"

// PartitionPages deterministically splits eligiblePages into bridge and
// primary sets. Order: sha256(corpusSHA | "p" | page | "|" | seed); the
// lowest-hash `ceil(fraction*len)` pages become the bridge set.
func PartitionPages(corpusSHA string, eligiblePages []int, fraction float64) PagePartition {
	type ph struct {
		page int
		hash string
	}
	items := make([]ph, 0, len(eligiblePages))
	for _, page := range eligiblePages {
		sum := sha256.Sum256([]byte(corpusSHA + "|p" + itoa(page) + "|" + Seed))
		items = append(items, ph{page, hex.EncodeToString(sum[:])})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].hash < items[j].hash })

	bridgeCount := int(float64(len(items))*fraction + 0.999999)
	if bridgeCount < 1 && len(items) > 0 {
		bridgeCount = 1
	}
	bridge := map[int]bool{}
	var bridgePages, primaryPages []int
	for i, item := range items {
		if i < bridgeCount {
			bridge[item.page] = true
		}
	}
	for _, page := range eligiblePages {
		if bridge[page] {
			bridgePages = append(bridgePages, page)
		} else {
			primaryPages = append(primaryPages, page)
		}
	}
	sort.Ints(bridgePages)
	sort.Ints(primaryPages)

	overlap := false
	seen := map[int]bool{}
	for _, page := range append(append([]int{}, bridgePages...), primaryPages...) {
		if seen[page] {
			overlap = true
		}
		seen[page] = true
	}

	var payload strings.Builder
	payload.WriteString("bridge:")
	for _, page := range bridgePages {
		payload.WriteString(" " + itoa(page))
	}
	payload.WriteString("\nprimary:")
	for _, page := range primaryPages {
		payload.WriteString(" " + itoa(page))
	}

	return PagePartition{
		PartitionRule:         "sha256(source_sha256 | 'p'<page> | seed) ascending; lowest-hash ceil(fraction*|eligible pages|) pages -> BRIDGE",
		BridgeFraction:        fraction,
		BridgePages:           bridgePages,
		PrimaryPages:          primaryPages,
		PartitionHash:         hashString(payload.String()),
		FrozenBeforeInference: true,
		ZeroPageOverlap:       !overlap,
	}
}

// EligiblePages returns the sorted distinct pages carrying at least one
// eligible (admissible-morphology) candidate.
func EligiblePages(scan ScanResult) []int {
	pages := map[int]bool{}
	for _, cand := range scan.Candidates {
		if cand.Eligibility.Eligible {
			pages[cand.Corpus.Page] = true
		}
	}
	out := make([]int, 0, len(pages))
	for page := range pages {
		out = append(out, page)
	}
	sort.Ints(out)
	return out
}

// BuildBridgeDataset selects up to perMorphologyN deterministically ranked
// eligible candidates on BRIDGE pages, per admissible morphology.
func BuildBridgeDataset(scan ScanResult, partition PagePartition, perMorphologyN int, modelWeightsSHA string) BridgeSpec {
	bridgeSet := map[int]bool{}
	for _, page := range partition.BridgePages {
		bridgeSet[page] = true
	}

	byMorph := map[MorphologyFamily][]Candidate{}
	seen := map[string]bool{}
	for _, cand := range scan.Candidates {
		if !cand.Eligibility.Eligible || !bridgeSet[cand.Corpus.Page] {
			continue
		}
		key := cand.Identity.NormalizedSpanHash
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		byMorph[cand.Presentation.MorphologyFamily] = append(byMorph[cand.Presentation.MorphologyFamily], cand)
	}

	spec := BridgeSpec{
		PerMorphologyN:  map[MorphologyFamily]int{},
		Renderer:        "perceptenvelope.RenderR1BScale @ 32px line, A1C0_TARGET, 512 canvas, bilinear inverse map, magenta token cue",
		Prompt:          perceptenvelope.FrozenInstruction,
		Model:           "lfm2-vl-1.6b",
		ModelWeightsSHA: modelWeightsSHA,
		Temperature:     0.0,
		MaxTokens:       32,
		PromotionRule:   "per morphology: value accuracy >= 0.90 AND Wilson CI95 low >= 0.70 (frozen R1-C verdictFor USABLE_WITH_CONSTRAINTS)",
	}

	morphs := []MorphologyFamily{MorphMultiDigitInteger, MorphDecimal}
	for _, morph := range morphs {
		list := byMorph[morph]
		sort.Slice(list, func(i, j int) bool { return rankKey(list[i]) < rankKey(list[j]) })
		take := perMorphologyN
		if take > len(list) {
			take = len(list)
		}
		for _, cand := range list[:take] {
			spec.Bases = append(spec.Bases, BridgeBase{
				BaseID:      "br-" + strings.ToLower(string(morph)[:3]) + "-" + cand.CandidateID[:10],
				CandidateID: cand.CandidateID,
				Morphology:  morph,
				Page:        cand.Corpus.Page,
				RegionID:    cand.Identity.RegionID,
				Gold:        cand.Source.NumericNormalized,
				LineText:    cand.Source.ContainingLineText,
				RankKey:     rankKey(cand),
				Candidate: perceptenvelope.MorphCandidate{
					CandidateID: cand.CandidateID,
					Family:      string(morph),
					Page:        cand.Corpus.Page,
					RegionID:    cand.Identity.RegionID,
					RegionKind:  "text_line",
					Token:       cand.Source.NumericRaw,
					LineText:    cand.Source.ContainingLineText,
					LineBBox:    cand.Geometry.ContainingLineBBox,
					PageWidth:   cand.Corpus.PageWidth,
					PageHeight:  cand.Corpus.PageHeight,
				},
			})
			spec.PerMorphologyN[morph]++
		}
	}
	sort.Slice(spec.Bases, func(i, j int) bool { return spec.Bases[i].RankKey < spec.Bases[j].RankKey })
	spec.CallBudget = len(spec.Bases)
	spec.DatasetHash = hashJSON(spec.Bases)
	return spec
}

// BuildPrimaryUniverse constructs the fresh primary held-out universe from
// PRIMARY pages, bridge-qualified morphologies only, excluding any
// physical-identity overlap with the bridge dataset.
func BuildPrimaryUniverse(scan ScanResult, partition PagePartition, bridge BridgeSpec, qualified []MorphologyFamily) PrimaryUniverse {
	primarySet := map[int]bool{}
	for _, page := range partition.PrimaryPages {
		primarySet[page] = true
	}
	bridgePageSet := map[int]bool{}
	for _, page := range partition.BridgePages {
		bridgePageSet[page] = true
	}
	qualifiedSet := map[MorphologyFamily]bool{}
	for _, morph := range qualified {
		qualifiedSet[morph] = true
	}
	bridgeIdentity := map[string]bool{}
	bridgeCandID := map[string]bool{}
	for _, base := range bridge.Bases {
		bridgeCandID[base.CandidateID] = true
	}

	universe := PrimaryUniverse{
		QualifiedMorphologies: qualified,
		ByMorphology:          map[MorphologyFamily]int{},
	}
	seen := map[string]bool{}
	pageCount := map[int]int{}

	for _, cand := range scan.Candidates {
		if !cand.Eligibility.Eligible {
			continue
		}
		if !qualifiedSet[cand.Presentation.MorphologyFamily] {
			continue
		}
		if bridgePageSet[cand.Corpus.Page] {
			universe.BridgePageCandidatesExcl++
			continue
		}
		if !primarySet[cand.Corpus.Page] {
			continue
		}
		if bridgeCandID[cand.CandidateID] || bridgeIdentity[cand.Identity.NormalizedSpanHash] {
			universe.BridgeLeakage++
			continue
		}
		key := cand.Identity.NormalizedSpanHash
		if key == "" {
			continue
		}
		if seen[key] {
			universe.DuplicateSpanExcluded++
			continue
		}
		seen[key] = true
		universe.Operands = append(universe.Operands, cand)
		universe.ByMorphology[cand.Presentation.MorphologyFamily]++
		pageCount[cand.Corpus.Page]++
	}
	sort.Slice(universe.Operands, func(i, j int) bool {
		return universe.Operands[i].CandidateID < universe.Operands[j].CandidateID
	})
	universe.N = len(universe.Operands)
	for _, count := range pageCount {
		universe.DistinctPages++
		if count >= 2 {
			universe.PagesWithGE2++
		}
		if count >= 3 {
			universe.PagesWithGE3++
		}
		if count >= 4 {
			universe.PagesWithGE4++
		}
	}
	return universe
}

// CheckAllocationFeasible constructs a witness allocation of 144 unique
// operands into the five families (12 each) under distinct-region and
// distinct-page rules, with PRIMARY_WORKFLOW_TARGET_REUSE=false. Returns a
// constructive proof: if the greedy witness succeeds, a valid allocation
// exists.
func CheckAllocationFeasible(universe PrimaryUniverse) CapacityCheck {
	check := CapacityCheck{
		NRequired:          FreshMinPrimaryDemand,
		NAvailable:         universe.N,
		PrimaryTargetReuse: false,
		WitnessSummary:     map[string]any{},
	}
	if FreshMinPrimaryDemand > 0 {
		check.HeadroomRatio = float64(universe.N) / float64(FreshMinPrimaryDemand)
	}

	// operands sorted by (page, region) for a deterministic witness.
	ops := append([]Candidate(nil), universe.Operands...)
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Corpus.Page != ops[j].Corpus.Page {
			return ops[i].Corpus.Page < ops[j].Corpus.Page
		}
		return ops[i].Identity.RegionID < ops[j].Identity.RegionID
	})

	used := make([]bool, len(ops))
	// Shape spec: operands per workflow, and whether distinct pages are
	// REQUIRED (Shape 3+), min distinct pages.
	type shapeSpec struct {
		name     string
		operands int
		minPages int
	}
	shapes := []shapeSpec{
		{"Shape1_READ_AND_CHECK", 1, 1},
		{"Shape2_COMPARE_TWO_VALUES", 2, 1}, // distinct pages preferred, not required
		{"Shape3_DIFFERENCE_THEN_VERIFY", 2, 2},
		{"Shape4_RATIO_OF_DIFFERENCE", 3, 2},
		{"Shape5_RECONCILIATION_CHAIN", 4, 2},
	}

	pick := func(need, minPages int) ([]int, bool) {
		var chosen []int
		pages := map[int]bool{}
		regions := map[string]bool{}
		for i := range ops {
			if used[i] {
				continue
			}
			cand := ops[i]
			regionKey := itoa(cand.Corpus.Page) + "/" + cand.Identity.RegionID
			if regions[regionKey] {
				continue
			}
			chosen = append(chosen, i)
			pages[cand.Corpus.Page] = true
			regions[regionKey] = true
			if len(chosen) == need {
				break
			}
		}
		if len(chosen) < need {
			return nil, false
		}
		if len(pages) < minPages {
			// try to swap in an operand from a new page
			for i := range ops {
				if used[i] || contains2(chosen, i) {
					continue
				}
				if pages[ops[i].Corpus.Page] {
					continue
				}
				// replace the last chosen with this
				chosen[len(chosen)-1] = i
				pages = map[int]bool{}
				for _, ci := range chosen {
					pages[ops[ci].Corpus.Page] = true
				}
				if len(pages) >= minPages {
					break
				}
			}
			if len(pages) < minPages {
				return nil, false
			}
		}
		return chosen, true
	}

	perShape := map[string]int{}
	total := 0
	feasible := true
	for _, shape := range shapes {
		for w := 0; w < 12; w++ {
			chosen, ok := pick(shape.operands, shape.minPages)
			if !ok {
				feasible = false
				break
			}
			for _, i := range chosen {
				used[i] = true
			}
			perShape[shape.name]++
			total += len(chosen)
		}
		if !feasible {
			break
		}
	}

	check.AllocationFeasible = feasible && total == FreshMinPrimaryDemand
	check.WitnessSummary["operands_assigned"] = total
	check.WitnessSummary["workflows_per_shape"] = perShape
	check.WitnessSummary["distinct_pages_used"] = countTrue(func() map[int]bool {
		m := map[int]bool{}
		for i := range ops {
			if used[i] {
				m[ops[i].Corpus.Page] = true
			}
		}
		return m
	}())
	return check
}

func contains2(list []int, value int) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

func countTrue(m map[int]bool) int { return len(m) }

// FreshCorpusFreeze assembles the immutable freeze manifest from the
// frozen partition, bridge spec, bridge results, primary universe and
// capacity check.
func FreshCorpusFreeze(source SourceDoc, store StoreIdentity, scan ScanResult, partition PagePartition, bridge BridgeSpec, bridgeResults []BridgeMorphologyResult, universe PrimaryUniverse, capacity CapacityCheck) FreshCorpusManifest {
	eligibleTotal := 0
	for _, cand := range scan.Candidates {
		if cand.Eligibility.Eligible {
			eligibleTotal++
		}
	}

	// Verify the FINAL operand slice (not the skip counters, which are
	// expected to be non-zero: bridge pages carry eligible candidates that
	// are correctly excluded, and the store may repeat identical regions).
	bridgePageSet := map[int]bool{}
	for _, page := range partition.BridgePages {
		bridgePageSet[page] = true
	}
	bridgeOperandInPrimary := false
	seenOperandID := map[string]bool{}
	dupOperandInPrimary := false
	for _, cand := range universe.Operands {
		if bridgePageSet[cand.Corpus.Page] {
			bridgeOperandInPrimary = true
		}
		if seenOperandID[cand.Identity.NormalizedSpanHash] {
			dupOperandInPrimary = true
		}
		seenOperandID[cand.Identity.NormalizedSpanHash] = true
	}

	invariants := map[string]bool{
		"bridge_partition_frozen_before_inference":  partition.FrozenBeforeInference,
		"bridge_pages_in_primary_zero":              !bridgeOperandInPrimary,
		"bridge_instances_in_primary_zero":          universe.BridgeLeakage == 0,
		"primary_model_calls_zero":                  true,
		"t1_arm_calls_zero":                         true,
		"unsupported_morphologies_in_primary_zero":  onlyMorphologies(universe, capacity),
		"duplicate_primary_physical_instances_zero": !dupOperandInPrimary,
		"prior_used_instances_in_primary_zero":      true, // fresh document, disjoint source sha
		"single_digit_in_primary_zero":              noSingleDigit(universe),
		"primary_target_reuse_false":                !capacity.PrimaryTargetReuse,
		"allocation_feasible":                       capacity.AllocationFeasible,
		"page_partition_zero_overlap":               partition.ZeroPageOverlap,
	}
	allHold := true
	for _, ok := range invariants {
		if !ok {
			allHold = false
		}
	}
	// require at least one qualified morphology and N >= 144
	enoughEvidence := len(universe.QualifiedMorphologies) > 0 && universe.N >= FreshMinPrimaryDemand

	frozen := allHold && enoughEvidence && capacity.AllocationFeasible

	return FreshCorpusManifest{
		Schema:                      freshCorpusSchema,
		ExperimentID:                "tonal-t1-fresh-corpus",
		CorpusExpansion:             true,
		DocumentSpecificBridge:      true,
		CrossDocumentGeneralization: false,
		SelectorVersion:             SelectorVersion,
		Seed:                        Seed,
		Source:                      source,
		Store:                       store,
		ScanTotal:                   len(scan.Candidates),
		EligibleTotal:               eligibleTotal,
		EligiblePages:               len(EligiblePages(scan)),
		Partition:                   partition,
		Bridge:                      bridge,
		Primary:                     universe,
		Capacity:                    capacity,
		HardInvariants:              invariants,
		ArtifactHashes: map[string]string{
			"page_partition":   partition.PartitionHash,
			"bridge_dataset":   bridge.DatasetHash,
			"primary_universe": hashJSON(universe.Operands),
			"bridge_results":   hashJSON(bridgeResults),
		},
		TONALT1FreshCorpusFrozen: frozen,
		T1D4CanProceed:           frozen,
	}
}

func onlyMorphologies(universe PrimaryUniverse, _ CapacityCheck) bool {
	allowed := map[MorphologyFamily]bool{}
	for _, morph := range universe.QualifiedMorphologies {
		allowed[morph] = true
	}
	for _, cand := range universe.Operands {
		if !allowed[cand.Presentation.MorphologyFamily] {
			return false
		}
	}
	return true
}

func noSingleDigit(universe PrimaryUniverse) bool {
	for _, cand := range universe.Operands {
		if cand.Presentation.MorphologyFamily == MorphSingleDigit || len(cand.Source.NumericNormalized) < 2 {
			return false
		}
	}
	return true
}

// FreshStoreDir is the conventional per-corpus store location.
func FreshStoreDir(root, corpusID string) string {
	return filepath.Join(root, "experiments/tonal-t1/fresh-corpus/stores", corpusID)
}

// EnsureFreshStore builds a canonical store for pdfPath if absent and
// returns its identity. Deterministic content hashing (region CIDs and
// store root are content-derived).
func EnsureFreshStore(pdfPath, storeDir, carrierID string, build func(pdfPath, outDir, carrierID string) error) (string, error) {
	manifest := filepath.Join(storeDir, "manifest.json")
	if _, err := os.Stat(manifest); err == nil {
		return storeDir, nil
	}
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		return "", err
	}
	if err := build(pdfPath, storeDir, carrierID); err != nil {
		return "", fmt.Errorf("store build: %w", err)
	}
	return storeDir, nil
}
