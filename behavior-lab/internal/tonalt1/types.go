package tonalt1

import (
	"sort"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
)

// RejectionCode is a stable enum for why a scanned location is not a
// legitimate held-out T1 operand. Every rejection carries at least one.
type RejectionCode string

const (
	// Raw-scan / morphology.
	RejectParseFailure          RejectionCode = "REJECT_PARSE_FAILURE"
	RejectUnsupportedMorphology RejectionCode = "REJECT_UNSUPPORTED_MORPHOLOGY"
	RejectYearOrDateToken       RejectionCode = "REJECT_YEAR_OR_DATE_TOKEN"
	RejectRegionKind            RejectionCode = "REJECT_REGION_KIND"

	// Geometry / ambiguity.
	RejectMultipleNumericTokens RejectionCode = "REJECT_MULTIPLE_NUMERIC_TOKENS"
	RejectOperandNotIncluded    RejectionCode = "REJECT_OPERAND_NOT_INCLUDED"
	RejectGeometryAmbiguous     RejectionCode = "REJECT_GEOMETRY_AMBIGUOUS"
	RejectGeometryMalformed     RejectionCode = "REJECT_GEOMETRY_MALFORMED"
	RejectLineInPageMargin      RejectionCode = "REJECT_LINE_IN_PAGE_MARGIN"
	RejectLineTooNarrow         RejectionCode = "REJECT_LINE_TOO_NARROW_FOR_PROSE"
	RejectBareOrShortNumberLine RejectionCode = "REJECT_BARE_OR_SHORT_NUMBER_LINE"
	RejectNumberLeadingLine     RejectionCode = "REJECT_NUMBER_LEADING_LINE"
	RejectCrossReference        RejectionCode = "REJECT_CROSS_REFERENCE"
	RejectBibliographyLine      RejectionCode = "REJECT_BIBLIOGRAPHY_OR_CITATION_LINE"
	RejectFontBelowBody         RejectionCode = "REJECT_FONT_SIZE_BELOW_BODY"
	RejectRunningHeader         RejectionCode = "REJECT_RUNNING_HEADER_PAGE_NUMBER"
	RejectCueImplausible        RejectionCode = "REJECT_CUE_COVERS_IMPLAUSIBLE_LINE_FRACTION"
	RejectPaddedBoxClipped      RejectionCode = "REJECT_PADDED_TOKEN_BOX_CLIPPED_BY_PAGE"
	RejectTokenOffsetNotUnique  RejectionCode = "REJECT_TOKEN_OFFSET_NOT_UNIQUE"

	// Prior use.
	RejectPriorUsed RejectionCode = "REJECT_PRIOR_USED"

	// Domain.
	RejectDomainInvalid    RejectionCode = "REJECT_DOMAIN_INVALID"
	RejectVersionString    RejectionCode = "REJECT_VERSION_STRING"
	RejectWrappedFragment  RejectionCode = "REJECT_WRAPPED_REFERENCE_FRAGMENT"
	RejectPageHeaderFooter RejectionCode = "REJECT_PAGE_HEADER_OR_FOOTER"
	RejectLoneNumberLine   RejectionCode = "REJECT_LONE_NUMBER_NO_CONTEXT"
	RejectCaptionOrIndex   RejectionCode = "REJECT_CAPTION_OR_INDEX_ENTRY"
)

// RuleClass classifies why a selector rule exists (D3 v2 audit, protocol
// section 4). Only CAPABILITY, PRESENTATION_INTEGRITY and DOMAIN_VALIDITY
// rules block eligibility; AUTHORING_HEURISTIC rules are recorded as
// advisory tags but do not exclude a candidate.
type RuleClass string

const (
	ClassCapability   RuleClass = "CAPABILITY_EVIDENCE_REQUIRED"
	ClassPresentation RuleClass = "PRESENTATION_INTEGRITY_REQUIRED"
	ClassDomain       RuleClass = "DOMAIN_VALIDITY_REQUIRED"
	ClassAuthoring    RuleClass = "DATASET_AUTHORING_HEURISTIC"
)

// ruleClassOf is the frozen rule-provenance classification (D3 v2). The
// four AUTHORING_HEURISTIC rules were introduced by the R1-A/R1-B pool
// author to build a clean prose-context dataset; R1-C then demonstrated
// MULTI_DIGIT_INTEGER / DECIMAL extraction is USABLE_WITH_CONSTRAINTS on
// bare-number, margin, small-font and heading lines under the same
// presentation core — so they are not capability requirements.
var ruleClassOf = map[RejectionCode]RuleClass{
	RejectUnsupportedMorphology: ClassCapability,
	RejectParseFailure:          ClassCapability,

	RejectMultipleNumericTokens: ClassPresentation,
	RejectOperandNotIncluded:    ClassPresentation,
	RejectGeometryAmbiguous:     ClassPresentation,
	RejectGeometryMalformed:     ClassPresentation,
	RejectCueImplausible:        ClassPresentation,
	RejectPaddedBoxClipped:      ClassPresentation,
	RejectTokenOffsetNotUnique:  ClassPresentation,
	RejectRegionKind:            ClassPresentation,

	RejectPriorUsed:         ClassDomain,
	RejectDomainInvalid:     ClassDomain,
	RejectCrossReference:    ClassDomain,
	RejectBibliographyLine:  ClassDomain,
	RejectRunningHeader:     ClassDomain,
	RejectYearOrDateToken:   ClassDomain,
	RejectNumberLeadingLine: ClassDomain,
	RejectVersionString:     ClassDomain,
	RejectWrappedFragment:   ClassDomain,
	RejectPageHeaderFooter:  ClassDomain,
	RejectLoneNumberLine:    ClassDomain,
	RejectCaptionOrIndex:    ClassDomain,

	RejectLineInPageMargin:      ClassAuthoring,
	RejectLineTooNarrow:         ClassAuthoring,
	RejectBareOrShortNumberLine: ClassAuthoring,
	RejectFontBelowBody:         ClassAuthoring,
}

// blocks reports whether a rejection code excludes a candidate from the
// eligible universe (everything except AUTHORING_HEURISTIC).
func blocks(code RejectionCode) bool {
	return ruleClassOf[code] != ClassAuthoring
}

// ruleProvenanceTable groups every rejection code by its audited class for
// the selector manifest.
func ruleProvenanceTable() map[string][]string {
	table := map[string][]string{}
	for code, class := range ruleClassOf {
		table[string(class)] = append(table[string(class)], string(code))
	}
	for class := range table {
		sort.Strings(table[class])
	}
	return table
}

// MorphologyFamily is the frozen presentation family a candidate token
// belongs to. Only families with sufficient frozen real-document R1/R1-C
// support are admissible for the T1 primary benchmark.
type MorphologyFamily string

const (
	MorphMultiDigitInteger MorphologyFamily = "MULTI_DIGIT_INTEGER"
	MorphDecimal           MorphologyFamily = "DECIMAL"
)

// PhysicalIdentity captures every reconstructable physical-identity
// component of a source instance. Absent components are left zero: D3
// never invents unavailable precision.
type PhysicalIdentity struct {
	Page               int               `json:"page"`
	RegionID           string            `json:"region_id,omitempty"`
	SourceBBox         canonicaldoc.BBox `json:"source_bbox"`          // containing-line bbox, store coords
	CharStart          int               `json:"char_start"`           // rune offset of token in line text
	CharEnd            int               `json:"char_end"`             // exclusive
	NormalizedSpanHash string            `json:"normalized_span_hash"` // page+line identity, spanhash.go
}

// PriorUseMatch records one reason a candidate collides with an instance
// consumed by an earlier experiment. All matches are preserved.
type PriorUseMatch struct {
	Experiment string `json:"experiment"` // e.g. "R1-C", "PROFILE-H", "T0-B"
	Key        string `json:"key"`        // e.g. "page+region_id", "page+bbox", "page+char_span", "span_hash", "page_visual_exposure"
	Detail     string `json:"detail,omitempty"`
}

// Candidate is one deterministically discovered numeric-reading target
// with full provenance for auditability.
type Candidate struct {
	SchemaVersion string `json:"schema_version"`
	CandidateID   string `json:"candidate_id"` // deterministic from stable source identity

	Corpus CandidateCorpus `json:"corpus"`

	Identity PhysicalIdentity `json:"physical_identity"`

	Source CandidateSource `json:"source"`

	Geometry CandidateGeometry `json:"geometry"`

	Presentation CandidatePresentation `json:"presentation"`

	Domain CandidateDomain `json:"domain"`

	PriorUse CandidatePriorUse `json:"prior_use"`

	Eligibility CandidateEligibility `json:"eligibility"`

	Provenance CandidateProvenance `json:"provenance"`
}

// CandidateCorpus identifies the source document/page.
type CandidateCorpus struct {
	CorpusID        string  `json:"corpus_id"`
	SourcePDFSHA256 string  `json:"source_pdf_sha256"`
	StoreRootSHA256 string  `json:"store_root_sha256"`
	Page            int     `json:"page"`
	PageWidth       float64 `json:"page_width"`
	PageHeight      float64 `json:"page_height"`
	LayoutPath      string  `json:"layout_path"`
	PageRegionCount int     `json:"page_region_count"`
}

// CandidateSource records the digital-text source semantics.
type CandidateSource struct {
	ContainingLineText string  `json:"containing_line_text"`
	NumericRaw         string  `json:"numeric_raw"`        // verbatim token from the line
	NumericNormalized  string  `json:"numeric_normalized"` // punctuation-stripped
	NumericValue       float64 `json:"numeric_value"`
	QuantityPhrase     string  `json:"quantity_phrase"` // deterministic locate phrase (nearby words)
}

// CandidateGeometry records store-coordinate geometry used for the audit
// and for the T1 LocatedRegion-derived cue.
type CandidateGeometry struct {
	PageWidth           float64           `json:"page_width"`
	PageHeight          float64           `json:"page_height"`
	ContainingLineBBox  canonicaldoc.BBox `json:"containing_line_bbox"`
	OperandBBoxEstimate canonicaldoc.BBox `json:"operand_bbox_estimate"` // proportional rune-offset split
	CueBBoxStore        canonicaldoc.BBox `json:"cue_bbox_store"`        // padded, clamped
	LocatedRegionBBox   canonicaldoc.BBox `json:"located_region_bbox"`   // == containing line bbox for T1
}

// CandidatePresentation records the deterministic presentation audit.
type CandidatePresentation struct {
	MorphologyFamily      MorphologyFamily `json:"morphology_family"`
	NumericTokenCount     int              `json:"numeric_token_count"`     // digit-bearing whitespace tokens on the line
	CompetingNumericCount int              `json:"competing_numeric_count"` // NumericTokenCount - 1
	WhitespaceFieldCount  int              `json:"whitespace_field_count"`
	EffectiveScale        float64          `json:"effective_scale_estimate"` // 512 / line_bbox_height (R1/H target-centred render)
	R1EnvelopeSupported   bool             `json:"r1_envelope_supported"`
}

// CandidateDomain records deterministic arithmetic-domain properties for
// later (D4) family allocation. D3 does not allocate.
type CandidateDomain struct {
	ParseValid            bool `json:"parse_valid"`
	Finite                bool `json:"finite"`
	Integer               bool `json:"integer"`
	Zero                  bool `json:"zero"`
	Sign                  int  `json:"sign"` // -1, 0, +1
	EligibleAsOperand     bool `json:"eligible_as_generic_operand"`
	EligibleAsDenominator bool `json:"eligible_as_denominator"`
}

// CandidatePriorUse records whether the candidate collides with any
// previously consumed physical instance.
type CandidatePriorUse struct {
	Excluded bool            `json:"excluded"`
	Matches  []PriorUseMatch `json:"matches,omitempty"`
}

// CandidateEligibility is the final verdict. RejectionCodes lists every
// rule the candidate tripped; AdvisoryTags is the AUTHORING_HEURISTIC
// subset that does NOT block eligibility (D3 v2).
type CandidateEligibility struct {
	Eligible       bool            `json:"eligible"`
	RejectionCodes []RejectionCode `json:"rejection_codes,omitempty"`
	AdvisoryTags   []RejectionCode `json:"advisory_tags,omitempty"`
}

// CandidateProvenance records exactly how the candidate was derived.
type CandidateProvenance struct {
	SelectorVersion          string `json:"selector_version"`
	SpanNormVersion          string `json:"span_norm_version"`
	EnvelopeVersion          string `json:"envelope_version"`
	GeometryRuleVersion      string `json:"geometry_rule_version"`
	PriorUseInventoryVersion string `json:"prior_use_inventory_version"`
	TokenBoxMethod           string `json:"token_box_method"`
	PaddingPolicy            string `json:"padding_policy"`
}

// CandidateSchemaVersion is the per-candidate schema tag.
const CandidateSchemaVersion = "tonal.t1.d3.candidate.r1"
