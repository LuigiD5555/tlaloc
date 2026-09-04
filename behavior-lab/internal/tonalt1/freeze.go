package tonalt1

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Artifact file names (protocol section 20). Written under the output dir.
const (
	FileSelectorManifest   = "TONAL_T1_D3_SELECTOR_MANIFEST.json"
	FileCandidatesAll      = "TONAL_T1_D3_CANDIDATES_ALL.json"
	FileCandidatesEligible = "TONAL_T1_D3_CANDIDATES_ELIGIBLE.json"
	FileExclusions         = "TONAL_T1_D3_EXCLUSIONS.json"
	FileRejections         = "TONAL_T1_D3_REJECTIONS.json"
	FileInventory          = "TONAL_T1_D3_PRIOR_INVENTORY.json"
	FileStats              = "TONAL_T1_D3_STATS.json"
	FileFreeze             = "TONAL_T1_D3_FREEZE.json"
)

// D3Result is the in-memory outcome of a full D3 run.
type D3Result struct {
	Scan       ScanResult
	PriorIndex *PriorUseIndex
	Stats      Stats

	SelectorManifest SelectorManifest
	Inventory        PriorInventory
	Exclusions       ExclusionsArtifact
	Rejections       RejectionsArtifact
	Freeze           FreezeManifest

	Eligible []Candidate
}

// SelectorManifest records the frozen selector configuration.
type SelectorManifest struct {
	Schema          string `json:"schema"`
	ExperimentID    string `json:"experiment_id"`
	SelectorVersion string `json:"selector_version"`
	Seed            string `json:"seed"`

	CorpusReuse           bool `json:"corpus_reuse"`
	InstanceLevelHeldOut  bool `json:"instance_level_held_out"`
	CrossDocumentHeldOut  bool `json:"cross_document_held_out"`
	SourcePoolR1Exhausted bool `json:"source_pool_r1_exhausted"`
	UsesNewSelector       bool `json:"t1_uses_new_deterministic_selector"`

	Store StoreIdentity `json:"store"`

	SpanNormVersion          string              `json:"span_norm_version"`
	EnvelopeVersion          string              `json:"envelope_version"`
	GeometryRuleVersion      string              `json:"geometry_rule_version"`
	PriorUseInventoryVersion string              `json:"prior_use_inventory_version"`
	RuleAuditVersion         string              `json:"rule_audit_version"`
	RuleProvenance           map[string][]string `json:"rule_provenance"` // class -> [rejection codes]
	EnvelopeRules            []string            `json:"envelope_rules"`
	GeometryRules            []string            `json:"geometry_rules"`
	PaddingPolicy            string              `json:"padding_policy"`
	TokenBoxMethod           string              `json:"token_box_method"`

	ExpectedPrimaryUniqueOperandDemand int    `json:"expected_primary_unique_operand_demand"`
	DemandProvenance                   string `json:"expected_primary_unique_operand_demand_provenance"`

	ModelCalls  int  `json:"model_calls"`
	ScorerCalls int  `json:"scorer_calls"`
	Frozen      bool `json:"frozen"`

	Notes []string `json:"notes"`
}

// StoreIdentity is the canonical store identity block.
type StoreIdentity struct {
	StoreDir        string `json:"store_dir"`
	CarrierID       string `json:"carrier_id"`
	SourcePDFSHA256 string `json:"source_pdf_sha256"`
	StoreRootSHA256 string `json:"store_root_sha256"`
	PageCount       int    `json:"page_count"`
	RegionCount     int    `json:"region_count"`
}

// PriorInventory is the auditable prior-experiment inventory (section 5).
type PriorInventory struct {
	Schema                   string                 `json:"schema"`
	PriorUseInventoryVersion string                 `json:"prior_use_inventory_version"`
	TotalInstances           int                    `json:"total_prior_instances"`
	PageVisualExposure       []int                  `json:"page_visual_exposure_pages"`
	Sources                  []PriorInventorySource `json:"sources"`
	SyntheticExcluded        []string               `json:"synthetic_experiments_excluded_no_real_instances"`
}

// PriorInventorySource is one prior experiment/artifact in the inventory.
type PriorInventorySource struct {
	Experiment        string         `json:"experiment"`
	ArtifactPath      string         `json:"artifact_path"`
	ArtifactSHA256    string         `json:"artifact_sha256"`
	InstanceCount     int            `json:"instance_count"`
	IdentityKeyCounts map[string]int `json:"identity_key_counts"`
}

// ExclusionsArtifact lists every excluded candidate with its matching
// evidence (section 8).
type ExclusionsArtifact struct {
	Schema       string              `json:"schema"`
	Count        int                 `json:"count"`
	ByExperiment map[string]int      `json:"by_experiment"`
	ByKey        map[string]int      `json:"by_key"`
	Exclusions   []ExcludedCandidate `json:"exclusions"`
}

// ExcludedCandidate is one prior-used candidate.
type ExcludedCandidate struct {
	CandidateID string          `json:"candidate_id"`
	Page        int             `json:"page"`
	RegionID    string          `json:"region_id"`
	NumericRaw  string          `json:"numeric_raw"`
	LineText    string          `json:"line_text"`
	Matches     []PriorUseMatch `json:"matches"`
}

// RejectionsArtifact aggregates non-prior-use rejections (section 16).
type RejectionsArtifact struct {
	Schema          string                `json:"schema"`
	RejectionCounts map[RejectionCode]int `json:"rejection_counts"`
	Count           int                   `json:"rejected_candidate_count"`
	Rejections      []RejectedCandidate   `json:"rejections"`
}

// RejectedCandidate is one non-eligible candidate (may also be prior-used).
type RejectedCandidate struct {
	CandidateID    string          `json:"candidate_id"`
	Page           int             `json:"page"`
	RegionID       string          `json:"region_id"`
	NumericRaw     string          `json:"numeric_raw"`
	RejectionCodes []RejectionCode `json:"rejection_codes"`
}

// FreezeManifest is the D3 freeze record (section 26).
type FreezeManifest struct {
	Schema          string `json:"schema"`
	ExperimentID    string `json:"experiment_id"`
	SelectorVersion string `json:"selector_version"`

	Store StoreIdentity `json:"store"`

	SpanNormVersion          string `json:"span_norm_version"`
	EnvelopeVersion          string `json:"envelope_version"`
	GeometryRuleVersion      string `json:"geometry_rule_version"`
	PriorUseInventoryVersion string `json:"prior_use_inventory_version"`

	PriorInventoryHash string `json:"prior_inventory_definition_hash"`

	ArtifactHashes map[string]string `json:"artifact_hashes"`

	Counts FreezeCounts `json:"counts"`

	HardInvariants map[string]bool `json:"hard_invariants"`

	ModelCalls int  `json:"model_calls"`
	Frozen     bool `json:"frozen"`

	// TONAL_T1_D3_FROZEN is true only when every hard invariant holds. It
	// attests that the deterministic scan / exclusion / eligibility
	// pipeline is sound and hash-frozen — NOT that T1 can be built. See
	// DownstreamAllocationBlocker.
	TONALT1D3Frozen bool `json:"TONAL_T1_D3_FROZEN"`

	// DownstreamAllocationBlocker is non-nil when the frozen eligible
	// universe cannot satisfy the frozen T1 primary-workflow demand. D3 is
	// still validly frozen; D4 is blocked pending review (protocol
	// section 22).
	DownstreamAllocationBlocker *AllocationBlocker `json:"downstream_allocation_blocker,omitempty"`
}

// AllocationBlocker records an insufficient-held-out-operands blocker.
type AllocationBlocker struct {
	Blocker                string                   `json:"blocker"`
	NAvailable             int                      `json:"n_available"`
	NRequired              int                      `json:"n_required"`
	FamilyDerivedNRequired int                      `json:"family_derived_n_required_note"`
	Deficit                int                      `json:"deficit"`
	AvailableByMorphology  map[MorphologyFamily]int `json:"available_by_morphology"`
	AffectedStrata         []string                 `json:"affected_family_depth_strata"`
	WhyInvalidating        string                   `json:"why_continuing_would_invalidate_t1"`
	SafeNextActions        []string                 `json:"safest_next_actions_for_review"`
	MustNotDo              []string                 `json:"must_not_do_without_review"`
}

// FreezeCounts is the frozen count block.
type FreezeCounts struct {
	ScanTotal                     int     `json:"scan_total"`
	PagesScanned                  int     `json:"pages_scanned"`
	RegionsScanned                int     `json:"regions_scanned"`
	PriorPhysicalIdentityExcluded int     `json:"prior_physical_identity_excluded"`
	R1EnvelopeRejected            int     `json:"r1_envelope_rejected"`
	GeometryRejected              int     `json:"geometry_rejected"`
	FinalHeldOutAvailable         int     `json:"final_held_out_available"`
	RequiredUniqueOperandDemand   int     `json:"required_unique_operand_demand"`
	HeadroomRatio                 float64 `json:"headroom_ratio"`
	AllocationFeasible            bool    `json:"downstream_allocation_feasible"`
}

const (
	selectorManifestSchema = "tonal.t1.d3.selector-manifest.r1"
	priorInventorySchema   = "tonal.t1.d3.prior-inventory.r1"
	exclusionsSchema       = "tonal.t1.d3.exclusions.r1"
	rejectionsSchema       = "tonal.t1.d3.rejections.r1"
	freezeSchema           = "tonal.t1.d3.freeze.r1"
)

// RunD3 executes the full deterministic D3 pipeline in memory. It performs
// no I/O beyond reading the store and the frozen prior artifacts.
func RunD3(root, storeDir string) (D3Result, error) {
	priorIndex, err := LoadPriorUseIndex(root)
	if err != nil {
		return D3Result{}, err
	}
	scan, err := Scan(storeDir, priorIndex)
	if err != nil {
		return D3Result{}, err
	}
	stats := computeStats(scan, priorIndex)

	var eligible []Candidate
	for _, cand := range scan.Candidates {
		if cand.Eligibility.Eligible {
			eligible = append(eligible, cand)
		}
	}

	result := D3Result{
		Scan:       scan,
		PriorIndex: priorIndex,
		Stats:      stats,
		Eligible:   eligible,
	}

	store := StoreIdentity{
		StoreDir:        storeDir,
		CarrierID:       scan.CarrierID,
		SourcePDFSHA256: scan.SourcePDFSHA256,
		StoreRootSHA256: scan.StoreRootSHA256,
		PageCount:       scan.PagesScanned,
		RegionCount:     scan.RegionsScanned,
	}

	result.SelectorManifest = SelectorManifest{
		Schema:                             selectorManifestSchema,
		ExperimentID:                       ExperimentID,
		SelectorVersion:                    SelectorVersion,
		Seed:                               Seed,
		CorpusReuse:                        true,
		InstanceLevelHeldOut:               true,
		CrossDocumentHeldOut:               false,
		SourcePoolR1Exhausted:              true,
		UsesNewSelector:                    true,
		Store:                              store,
		SpanNormVersion:                    SpanNormVersion,
		EnvelopeVersion:                    EnvelopeVersion,
		GeometryRuleVersion:                GeometryRuleVersion,
		PriorUseInventoryVersion:           PriorUseInventoryVersion,
		RuleAuditVersion:                   RuleAuditVersion,
		RuleProvenance:                     ruleProvenanceTable(),
		EnvelopeRules:                      EnvelopeRuleSummary,
		GeometryRules:                      GeometryRuleSummary,
		PaddingPolicy:                      paddingPolicy,
		TokenBoxMethod:                     tokenBoxMethod,
		ExpectedPrimaryUniqueOperandDemand: ExpectedPrimaryUniqueOperandDemand,
		DemandProvenance:                   "protocol metadata: 12 * (1+2+2+3+4) under PRIMARY_WORKFLOW_TARGET_REUSE=false; D4 re-derives authoritatively from Tonal TaskFamily definitions",
		ModelCalls:                         0,
		ScorerCalls:                        0,
		Frozen:                             true,
		Notes: []string{
			"instance-level held-out only; NO cross-document generalization is claimed",
			"selection universe: fresh deterministic scan of the full 1152-page canonical store, not limited to the frozen R1 selection universe SOURCE_POOL_R1",
			"T1 cue geometry derives from the public LocatedRegion, not historical char-offset target geometry (recorded limitation)",
			"no model / scorer / expected-answer input; no manual visual selection",
		},
	}

	result.Inventory = buildInventory(root, priorIndex)
	result.Exclusions = buildExclusions(scan)
	result.Rejections = buildRejections(scan)
	result.Freeze = buildFreeze(result, store)

	return result, nil
}

func buildInventory(root string, priorIndex *PriorUseIndex) PriorInventory {
	inventory := PriorInventory{
		Schema:                   priorInventorySchema,
		PriorUseInventoryVersion: PriorUseInventoryVersion,
		TotalInstances:           priorIndex.InstanceCount(),
		PageVisualExposure:       PageVisualExposure(),
		SyntheticExcluded: []string{
			"exocortex-t0a-r0 (synthetic num_compare images)",
			"parrot-microisa-r0 / r0.1 (synthetic glyph / shape stimuli)",
			"parrot-perceptual-envelope-r1 R1-C synthetic bases + glyphbank (synthetic)",
		},
	}
	// Per-artifact instance counts + hashes.
	seen := map[string]bool{}
	for _, artifact := range priorArtifacts {
		full := filepath.Join(root, filepath.FromSlash(artifact.relPath))
		raw, err := os.ReadFile(full)
		sha := ""
		if err == nil {
			sha = hashBytes(raw)
		}
		key := artifact.experiment + "|" + artifact.relPath
		if seen[key] {
			continue
		}
		seen[key] = true
		inventory.Sources = append(inventory.Sources, PriorInventorySource{
			Experiment:        artifact.experiment,
			ArtifactPath:      artifact.relPath,
			ArtifactSHA256:    sha,
			InstanceCount:     priorIndex.SourceCounts[artifact.experiment],
			IdentityKeyCounts: priorIndex.KeyAvailability[artifact.experiment],
		})
	}
	inventory.Sources = append(inventory.Sources, PriorInventorySource{
		Experiment:        "T0-P0/CAPABILITY",
		ArtifactPath:      "experiments/exocortex-decomposition-r0/datasets/T0_P0_IMAGE_DATASET.json + experiments/parrot-capability-r0/datasets/end-to-end.jsonl",
		InstanceCount:     priorIndex.SourceCounts["T0-P0/CAPABILITY"],
		IdentityKeyCounts: priorIndex.KeyAvailability["T0-P0/CAPABILITY"],
	})
	sort.Slice(inventory.Sources, func(i, j int) bool {
		if inventory.Sources[i].Experiment != inventory.Sources[j].Experiment {
			return inventory.Sources[i].Experiment < inventory.Sources[j].Experiment
		}
		return inventory.Sources[i].ArtifactPath < inventory.Sources[j].ArtifactPath
	})
	return inventory
}

func buildExclusions(scan ScanResult) ExclusionsArtifact {
	artifact := ExclusionsArtifact{
		Schema:       exclusionsSchema,
		ByExperiment: map[string]int{},
		ByKey:        map[string]int{},
	}
	for _, cand := range scan.Candidates {
		if !cand.PriorUse.Excluded {
			continue
		}
		artifact.Exclusions = append(artifact.Exclusions, ExcludedCandidate{
			CandidateID: cand.CandidateID,
			Page:        cand.Corpus.Page,
			RegionID:    cand.Identity.RegionID,
			NumericRaw:  cand.Source.NumericRaw,
			LineText:    cand.Source.ContainingLineText,
			Matches:     cand.PriorUse.Matches,
		})
		for _, match := range cand.PriorUse.Matches {
			artifact.ByExperiment[match.Experiment]++
			artifact.ByKey[match.Key]++
		}
	}
	artifact.Count = len(artifact.Exclusions)
	sort.Slice(artifact.Exclusions, func(i, j int) bool {
		return artifact.Exclusions[i].CandidateID < artifact.Exclusions[j].CandidateID
	})
	return artifact
}

func buildRejections(scan ScanResult) RejectionsArtifact {
	artifact := RejectionsArtifact{
		Schema:          rejectionsSchema,
		RejectionCounts: map[RejectionCode]int{},
	}
	for _, cand := range scan.Candidates {
		if cand.Eligibility.Eligible {
			continue
		}
		artifact.Rejections = append(artifact.Rejections, RejectedCandidate{
			CandidateID:    cand.CandidateID,
			Page:           cand.Corpus.Page,
			RegionID:       cand.Identity.RegionID,
			NumericRaw:     cand.Source.NumericRaw,
			RejectionCodes: cand.Eligibility.RejectionCodes,
		})
		for _, code := range cand.Eligibility.RejectionCodes {
			artifact.RejectionCounts[code]++
		}
	}
	artifact.Count = len(artifact.Rejections)
	sort.Slice(artifact.Rejections, func(i, j int) bool {
		return artifact.Rejections[i].CandidateID < artifact.Rejections[j].CandidateID
	})
	return artifact
}

func buildFreeze(result D3Result, store StoreIdentity) FreezeManifest {
	stats := result.Stats

	hashes := map[string]string{
		FileSelectorManifest:   hashJSON(result.SelectorManifest),
		FileCandidatesAll:      hashJSON(result.Scan.Candidates),
		FileCandidatesEligible: hashJSON(result.Eligible),
		FileExclusions:         hashJSON(result.Exclusions),
		FileRejections:         hashJSON(result.Rejections),
		FileInventory:          hashJSON(result.Inventory),
		FileStats:              hashJSON(stats),
	}

	invariants := hardInvariants(result)
	allHold := true
	for _, ok := range invariants {
		if !ok {
			allHold = false
		}
	}

	var blocker *AllocationBlocker
	if !stats.AllocationFeasible {
		var affected []string
		for _, stratum := range []string{"Shape 1 (natural 2)", "Shape 2 (natural 4)", "Shape 3 (natural 6)", "Shape 4 (natural 8)", "Shape 5 (natural 12)"} {
			affected = append(affected, stratum)
		}
		blocker = &AllocationBlocker{
			Blocker:                "INSUFFICIENT_UNIQUE_HELD_OUT_OPERANDS",
			NAvailable:             stats.AvailableUniqueOperands,
			NRequired:              stats.RequiredUniqueOperandDemand,
			FamilyDerivedNRequired: 144,
			Deficit:                stats.RequiredUniqueOperandDemand - stats.AvailableUniqueOperands,
			AvailableByMorphology:  stats.EligibleByMorphology,
			AffectedStrata:         affected,
			WhyInvalidating:        "T1 requires 60 primary workflows (12 per family) over UNIQUE source operand instances that are instance-level held-out from every prior Parrot experiment and inside the earned R1 envelope (R1-C: MULTI_DIGIT_INTEGER and DECIMAL USABLE_WITH_CONSTRAINTS; SINGLE_DIGIT/THOUSANDS/PERCENTAGE/TABLE_CELL FRAGILE; RANGE/SIGNED DO_NOT_DEPLOY). D3 v2 reclassified the R1-A/R1-B prose-context pool rules (margin / narrow-line / small-font / bare-line-4-fields) as DATASET_AUTHORING_HEURISTIC and stopped blocking on them; even so, after prior-use exclusion and DOMAIN validity (number-leading TOC/heading, cross-reference, version string, page header/footer) the corpus yields only a low-double-digit count of MULTI_DIGIT_INTEGER + DECIMAL quantity operands. Building T1 from this pool would force operand reuse across workflows (PRIMARY_WORKFLOW_TARGET_REUSE=true) or reuse of prior-inferred instances — either destroys the held-out claim and the workflow-composition difficulty axis.",
			SafeNextActions: []string{
				"escalate to protocol review: the frozen T1 design (5 families x 12 x no-reuse, in-envelope, instance-held-out, single canonical document) is not satisfiable by this corpus",
				"option A (needs review): reduce n per family and/or family count / depth strata so demand <= available",
				"option B (needs review): explicitly authorise bounded PRIMARY_WORKFLOW_TARGET_REUSE with a frozen reuse cap and a stated weakened claim",
				"option C (needs review): admit a second born-digital source document (breaks CROSS_DOCUMENT_HELD_OUT=false and the single-corpus design)",
				"option D (needs review): commission fresh Parrot envelope evidence for an additional morphology (e.g. SINGLE_DIGIT is only FRAGILE today)",
				"option E (needs review): relax the isolated-one-numeric-token-per-line rule with a compensating cue mechanism, re-validated against the R1 envelope",
			},
			MustNotDo: []string{
				"do NOT fabricate or synthesise operands",
				"do NOT silently relax PRIMARY_WORKFLOW_TARGET_REUSE",
				"do NOT admit FRAGILE / DO_NOT_DEPLOY morphologies to hit the count",
				"do NOT reuse any prior-inferred physical instance",
				"do NOT manufacture depth via low-scale / adversarial / ambiguous presentation",
			},
		}
	}

	return FreezeManifest{
		Schema:                   freezeSchema,
		ExperimentID:             ExperimentID,
		SelectorVersion:          SelectorVersion,
		Store:                    store,
		SpanNormVersion:          SpanNormVersion,
		EnvelopeVersion:          EnvelopeVersion,
		GeometryRuleVersion:      GeometryRuleVersion,
		PriorUseInventoryVersion: PriorUseInventoryVersion,
		PriorInventoryHash:       result.PriorIndex.PriorInventoryHash(),
		ArtifactHashes:           hashes,
		Counts: FreezeCounts{
			ScanTotal:                     stats.ScanTotal,
			PagesScanned:                  stats.PagesScanned,
			RegionsScanned:                stats.RegionsScanned,
			PriorPhysicalIdentityExcluded: stats.PriorPhysicalIdentityExcluded,
			R1EnvelopeRejected:            stats.R1EnvelopeRejected,
			GeometryRejected:              stats.GeometryRejected,
			FinalHeldOutAvailable:         stats.FinalHeldOutAvailable,
			RequiredUniqueOperandDemand:   stats.RequiredUniqueOperandDemand,
			HeadroomRatio:                 stats.HeadroomRatio,
			AllocationFeasible:            stats.AllocationFeasible,
		},
		HardInvariants:              invariants,
		ModelCalls:                  0,
		Frozen:                      true,
		TONALT1D3Frozen:             allHold,
		DownstreamAllocationBlocker: blocker,
	}
}

// hardInvariants (protocol section 25). Note: allocation feasibility
// (N >= 144) is a DOWNSTREAM D4 concern and is NOT a D3 freeze invariant —
// D3 may freeze a scientifically valid scan that yields < 144 and report
// the infeasibility (section 22).
func hardInvariants(result D3Result) map[string]bool {
	scan := result.Scan

	duplicatePhysical := false
	priorInEligible := false
	unsupportedInEligible := false
	ambiguousGeometryInEligible := false
	idsStable := true
	seenIdentity := map[string]bool{}

	for _, cand := range result.Eligible {
		identityKey := cand.Identity.NormalizedSpanHash
		if identityKey == "" {
			idsStable = false
		}
		if seenIdentity[identityKey] {
			duplicatePhysical = true
		}
		seenIdentity[identityKey] = true

		if cand.PriorUse.Excluded {
			priorInEligible = true
		}
		if !cand.Presentation.R1EnvelopeSupported {
			unsupportedInEligible = true
		}
		if hasBlockingPresentationReject(cand.Eligibility.RejectionCodes) {
			ambiguousGeometryInEligible = true
		}
	}

	for _, cand := range scan.Candidates {
		if cand.CandidateID != deriveCandidateID(cand) {
			idsStable = false
		}
	}

	return map[string]bool{
		"model_calls_zero":                             true,
		"scorer_calls_zero":                            true,
		"manual_candidate_overrides_zero":              true,
		"duplicate_eligible_physical_instances_zero":   !duplicatePhysical,
		"prior_used_instances_in_eligible_set_zero":    !priorInEligible,
		"unsupported_r1_morphologies_in_eligible_zero": !unsupportedInEligible,
		"ambiguous_geometry_in_eligible_zero":          !ambiguousGeometryInEligible,
		"candidate_ids_stable":                         idsStable,
	}
}

// Write emits every frozen artifact under outDir. The freeze manifest is
// written last, after its artifact_hashes are computed over the others.
func (result D3Result) Write(outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	writes := []struct {
		name string
		data any
	}{
		{FileSelectorManifest, result.SelectorManifest},
		{FileCandidatesAll, result.Scan.Candidates},
		{FileCandidatesEligible, result.Eligible},
		{FileExclusions, result.Exclusions},
		{FileRejections, result.Rejections},
		{FileInventory, result.Inventory},
		{FileStats, result.Stats},
		{FileFreeze, result.Freeze},
	}
	for _, write := range writes {
		if err := writeJSON(filepath.Join(outDir, write.name), write.data); err != nil {
			return fmt.Errorf("write %s: %w", write.name, err)
		}
	}
	return nil
}

// --- hashing / json helpers ---

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hashString(text string) string { return hashBytes([]byte(text)) }

// hashJSON hashes the canonical (sorted-key, indented) JSON encoding of v.
func hashJSON(value any) string {
	data, err := marshalCanonical(value)
	if err != nil {
		return "MARSHAL_ERROR:" + err.Error()
	}
	return hashBytes(data)
}

func marshalCanonical(value any) ([]byte, error) {
	// json.Marshal already sorts map keys; struct field order is stable.
	return json.MarshalIndent(value, "", "  ")
}

func writeJSON(path string, value any) error {
	data, err := marshalCanonical(value)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
