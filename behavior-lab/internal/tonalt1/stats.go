package tonalt1

import "sort"

// Stats is the authoritative D3 statistics block (protocol section 21),
// computed from the Go scan — never from a Python approximation.
type Stats struct {
	Schema string `json:"schema"`

	ScanTotal         int `json:"scan_total"` // discovered candidate locations
	PagesScanned      int `json:"pages_scanned"`
	RegionsScanned    int `json:"regions_scanned"`
	DigitTokensSeen   int `json:"digit_tokens_seen"`
	RawNumeric        int `json:"raw_numeric_candidates"` // == ScanTotal (all are numeric by construction)
	PhysicalIDDerived int `json:"physical_identity_derived"`

	PriorPhysicalIdentityExcluded int `json:"prior_physical_identity_excluded"`
	R1EnvelopeRejected            int `json:"r1_envelope_rejected"`
	GeometryRejected              int `json:"geometry_rejected"`
	ParseRejected                 int `json:"parse_rejected"`
	OtherDomainRejected           int `json:"other_domain_rejected"`

	// D3 v2 rule-class rejection accounting.
	CapabilityRejected            int `json:"capability_rejected"`
	PresentationIntegrityRejected int `json:"presentation_integrity_rejected"`
	DomainValidityRejected        int `json:"domain_validity_rejected"`
	AuthoringHeuristicOnly        int `json:"authoring_heuristic_only_would_pass_v2"`

	FinalHeldOutAvailable int `json:"final_held_out_available"`

	RejectionCounts map[RejectionCode]int `json:"rejection_counts"`

	ExclusionsByExperiment map[string]int            `json:"exclusions_by_experiment"`
	ExclusionsByKey        map[string]int            `json:"exclusions_by_key"`
	ExclusionsByExpKey     map[string]map[string]int `json:"exclusions_by_experiment_key"`

	EligibleByPage       map[int]int              `json:"eligible_by_page"`
	EligibleByMorphology map[MorphologyFamily]int `json:"eligible_by_morphology"`
	EligibleByValueClass map[string]int           `json:"eligible_by_value_class"`

	DistinctEligiblePages            int `json:"distinct_eligible_pages"`
	PagesWithGE2Eligible             int `json:"pages_with_ge2_eligible"`
	PagesWithGE3Eligible             int `json:"pages_with_ge3_eligible"`
	PagesWithGE4Eligible             int `json:"pages_with_ge4_eligible"`
	DistinctEligibleNormalizedValues int `json:"distinct_eligible_normalized_values"`

	RequiredUniqueOperandDemand int     `json:"required_unique_operand_demand"`
	AvailableUniqueOperands     int     `json:"available_unique_operands"`
	HeadroomRatio               float64 `json:"headroom_ratio"`
	AllocationFeasible          bool    `json:"downstream_allocation_feasible"`
}

const statsSchema = "tonal.t1.d3.stats.r1"

func computeStats(scan ScanResult, priorIndex *PriorUseIndex) Stats {
	stats := Stats{
		Schema:                      statsSchema,
		ScanTotal:                   len(scan.Candidates),
		PagesScanned:                scan.PagesScanned,
		RegionsScanned:              scan.RegionsScanned,
		DigitTokensSeen:             scan.DigitTokensSeen,
		RawNumeric:                  len(scan.Candidates),
		RejectionCounts:             map[RejectionCode]int{},
		ExclusionsByExperiment:      map[string]int{},
		ExclusionsByKey:             map[string]int{},
		ExclusionsByExpKey:          map[string]map[string]int{},
		EligibleByPage:              map[int]int{},
		EligibleByMorphology:        map[MorphologyFamily]int{},
		EligibleByValueClass:        map[string]int{},
		RequiredUniqueOperandDemand: ExpectedPrimaryUniqueOperandDemand,
	}

	normValues := map[string]bool{}

	for _, cand := range scan.Candidates {
		if cand.Identity.NormalizedSpanHash != "" {
			stats.PhysicalIDDerived++
		}
		for _, code := range cand.Eligibility.RejectionCodes {
			stats.RejectionCounts[code]++
		}
		switch {
		case cand.PriorUse.Excluded:
			stats.PriorPhysicalIdentityExcluded++
		}
		if contains(cand.Eligibility.RejectionCodes, RejectUnsupportedMorphology) ||
			contains(cand.Eligibility.RejectionCodes, RejectYearOrDateToken) ||
			contains(cand.Eligibility.RejectionCodes, RejectRegionKind) {
			stats.R1EnvelopeRejected++
		}
		if hasAnyGeometryReject(cand.Eligibility.RejectionCodes) {
			stats.GeometryRejected++
		}
		if contains(cand.Eligibility.RejectionCodes, RejectParseFailure) {
			stats.ParseRejected++
		}
		if contains(cand.Eligibility.RejectionCodes, RejectDomainInvalid) {
			stats.OtherDomainRejected++
		}
		// Rule-class accounting.
		var anyCap, anyPres, anyDomain, anyAuthoring bool
		for _, code := range cand.Eligibility.RejectionCodes {
			switch ruleClassOf[code] {
			case ClassCapability:
				anyCap = true
			case ClassPresentation:
				anyPres = true
			case ClassDomain:
				anyDomain = true
			case ClassAuthoring:
				anyAuthoring = true
			}
		}
		if anyCap {
			stats.CapabilityRejected++
		}
		if anyPres {
			stats.PresentationIntegrityRejected++
		}
		if anyDomain {
			stats.DomainValidityRejected++
		}
		if anyAuthoring && !anyCap && !anyPres && !anyDomain {
			stats.AuthoringHeuristicOnly++
		}

		for _, match := range cand.PriorUse.Matches {
			stats.ExclusionsByExperiment[match.Experiment]++
			stats.ExclusionsByKey[match.Key]++
			byKey := stats.ExclusionsByExpKey[match.Experiment]
			if byKey == nil {
				byKey = map[string]int{}
				stats.ExclusionsByExpKey[match.Experiment] = byKey
			}
			byKey[match.Key]++
		}

		if cand.Eligibility.Eligible {
			stats.FinalHeldOutAvailable++
			stats.EligibleByPage[cand.Corpus.Page]++
			stats.EligibleByMorphology[cand.Presentation.MorphologyFamily]++
			stats.EligibleByValueClass[valueClass(cand)]++
			normValues[cand.Source.NumericNormalized] = true
		}
	}

	for _, count := range stats.EligibleByPage {
		stats.DistinctEligiblePages++
		if count >= 2 {
			stats.PagesWithGE2Eligible++
		}
		if count >= 3 {
			stats.PagesWithGE3Eligible++
		}
		if count >= 4 {
			stats.PagesWithGE4Eligible++
		}
	}
	stats.DistinctEligibleNormalizedValues = len(normValues)
	stats.AvailableUniqueOperands = stats.FinalHeldOutAvailable
	if stats.RequiredUniqueOperandDemand > 0 {
		stats.HeadroomRatio = float64(stats.AvailableUniqueOperands) / float64(stats.RequiredUniqueOperandDemand)
	}
	stats.AllocationFeasible = stats.AvailableUniqueOperands >= stats.RequiredUniqueOperandDemand

	return stats
}

// hasBlockingPresentationReject reports whether any code is a
// PRESENTATION_INTEGRITY rule (these always block; used by the
// eligible-set integrity invariant).
func hasBlockingPresentationReject(codes []RejectionCode) bool {
	for _, code := range codes {
		if ruleClassOf[code] == ClassPresentation {
			return true
		}
	}
	return false
}

// hasAnyGeometryReject retains its name for the stats bucket: any
// PRESENTATION_INTEGRITY or geometry-shaped DOMAIN reject.
func hasAnyGeometryReject(codes []RejectionCode) bool {
	for _, code := range codes {
		switch ruleClassOf[code] {
		case ClassPresentation:
			return true
		}
		switch code {
		case RejectNumberLeadingLine, RejectLoneNumberLine, RejectPageHeaderFooter, RejectRunningHeader, RejectCaptionOrIndex:
			return true
		}
	}
	return false
}

func valueClass(cand Candidate) string {
	if cand.Presentation.MorphologyFamily == MorphDecimal {
		return "decimal"
	}
	switch len(cand.Source.NumericNormalized) {
	case 2:
		return "int_2digit"
	case 3:
		return "int_3digit"
	case 4:
		return "int_4digit"
	default:
		return "int_other"
	}
}

// SortedRejectionCounts returns rejection counts as an ordered slice for
// stable reporting.
func (stats Stats) SortedRejectionCounts() [][2]any {
	codes := make([]string, 0, len(stats.RejectionCounts))
	for code := range stats.RejectionCounts {
		codes = append(codes, string(code))
	}
	sort.Strings(codes)
	out := make([][2]any, 0, len(codes))
	for _, code := range codes {
		out = append(out, [2]any{code, stats.RejectionCounts[RejectionCode(code)]})
	}
	return out
}
