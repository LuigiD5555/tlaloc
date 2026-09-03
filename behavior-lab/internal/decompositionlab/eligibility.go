package decompositionlab

import (
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/exocortex"
)

// T0-B ELIGIBILITY AUDIT (T0 protocol sections 17-18, 35).
//
// Deterministic classification of every frozen P0 IMAGE base_id into
// ELIGIBLE_R0 / NOT_APPLICABLE_R0, decided BEFORE any model execution and
// derived ONLY from (a) the P0 task structure and (b) the compiled P2-A
// CapabilityProfile. It never consults whether Parrot answered a case
// correctly, whether the expected answer is "easy", or whether including a
// case would improve aggregate accuracy.
//
// Coverage and accuracy are kept on separate axes (section 35): this file
// only answers "can R0 represent this task?", never "can R0 solve it?".

// Eligibility verdicts.
const (
	EligibleR0      = "ELIGIBLE_R0"
	NotApplicableR0 = "NOT_APPLICABLE_R0"
)

// CaseEligibility is one P0 base_id's frozen classification.
type CaseEligibility struct {
	BaseID                    string       `json:"base_id"`
	Category                  string       `json:"category"`
	TaskFamily                string       `json:"task_family"`
	RequiredEvidenceCount     int          `json:"required_evidence_count"`
	RequiredLogicalOps        []string     `json:"required_logical_ops"`
	RequiredOperandTypes      []string     `json:"required_operand_types"`
	RequiredParrotCapability  string       `json:"required_parrot_capability"`
	RequiredDeterministicCaps []string     `json:"required_deterministic_capabilities"`
	RecipeSupportedByR0       bool         `json:"recipe_supported_by_r0"`
	Eligibility               string       `json:"eligibility"`
	Reason                    string       `json:"reason"`
	Recipe                    []AtomicStep `json:"recipe,omitempty"`
}

// EligibilityAudit is the frozen T0-B eligibility artifact.
type EligibilityAudit struct {
	Schema             string            `json:"schema"`
	ProfileID          string            `json:"profile_id"`
	ProfileSourceHash  string            `json:"profile_source_artifact_hash_sha256"`
	P0DatasetSHA256    string            `json:"p0_dataset_sha256"`
	TotalP0Cases       int               `json:"total_p0_cases"`
	EligibleR0         int               `json:"eligible_r0"`
	NotApplicableR0    int               `json:"not_applicable_r0"`
	R0TaskCoverage     float64           `json:"r0_task_coverage"`
	ReasonFamilyCounts map[string]int    `json:"reason_family_counts"`
	Cases              []CaseEligibility `json:"cases"`
}

const EligibilityAuditSchemaR0 = "tlaloc.exocortex-t0b.eligibility.r0"

// parrotDeployable reports whether the compiled profile lets Parrot carry
// this opcode at all (DEPLOY or DEPLOY_WITH_CONSTRAINTS). An EXTERNALIZE /
// DO_NOT_DEPLOY / missing opcode is not deployable.
func parrotDeployable(profile exocortex.CapabilityProfile, opcode string) bool {
	entry, ok := profile.Entry(opcode)
	if !ok {
		return false
	}
	return entry.DeploymentRecommendation == exocortex.DeploymentDeploy ||
		entry.DeploymentRecommendation == exocortex.DeploymentDeployConstrained
}

func choiceWidthLimit(profile exocortex.CapabilityProfile) int {
	entry, ok := profile.Entry(exocortex.OpSelectOne)
	if !ok {
		return 0
	}
	return entry.Constraints.FormalMaxChoiceWidth
}

// ClassifyCase applies the fixed R0 rule table to one P0 record. The rules
// are keyed on (category, task_family); every branch is decided before any
// execution and only from the profile + task shape.
func ClassifyCase(record P0Record, profile exocortex.CapabilityProfile) CaseEligibility {
	c := CaseEligibility{
		BaseID: record.BaseID, Category: record.Category, TaskFamily: record.TaskFamily,
		RequiredEvidenceCount: 1,
	}
	key := record.Category + "/" + record.TaskFamily

	switch key {
	case CategoryLocate + "/choice":
		// "Exactly one of these two headings appears on this page — which?"
		// A 2-way selection over the page/crop. Parrot SELECT_ONE, width 2.
		c.RequiredLogicalOps = []string{"SELECT_ONE_OF_2"}
		c.RequiredOperandTypes = []string{"choice"}
		c.RequiredParrotCapability = exocortex.OpSelectOne
		width := 2
		switch {
		case !parrotDeployable(profile, exocortex.OpSelectOne):
			c.Eligibility, c.Reason = NotApplicableR0, "P2-A externalizes SELECT_ONE; no R0 deterministic substitute for a visual heading choice"
		case choiceWidthLimit(profile) < width:
			c.Eligibility, c.Reason = NotApplicableR0, "P2-A formal choice-width limit below the required width 2"
		default:
			c.RecipeSupportedByR0 = true
			c.Eligibility, c.Reason = EligibleR0, "2-way visual heading selection within the SELECT_ONE envelope"
			c.Recipe = []AtomicStep{{
				ID: "select", Opcode: exocortex.OpSelectOne, EvidenceRefs: refIDs(record), OutputKey: "answer", ChoiceWidth: width,
			}}
		}

	case CategoryNumeric + "/numeric":
		// "According to this page, how many X?" — a single number to read.
		c.RequiredLogicalOps = []string{"EXTRACT_NUMBER"}
		c.RequiredOperandTypes = []string{"number"}
		c.RequiredParrotCapability = exocortex.OpExtractNumber
		c.RequiredDeterministicCaps = []string{exocortex.OpNormalize}
		if !parrotDeployable(profile, exocortex.OpExtractNumber) {
			c.Eligibility, c.Reason = NotApplicableR0, "P2-A externalizes EXTRACT_NUMBER"
		} else {
			c.RecipeSupportedByR0 = true
			c.Eligibility, c.Reason = EligibleR0, "single numeric extraction within the EXTRACT_NUMBER envelope (PARTIAL_TRANSFER)"
			c.Recipe = []AtomicStep{
				{ID: "extract", Opcode: exocortex.OpExtractNumber, EvidenceRefs: refIDs(record), OutputKey: "raw"},
				{ID: "normalize", Opcode: exocortex.OpNormalize, OutputKey: "answer", Deterministic: true},
			}
		}

	case CategoryEntity + "/entity":
		c.RequiredLogicalOps = []string{"EXTRACT_ENTITY"}
		c.RequiredOperandTypes = []string{"entity"}
		c.RequiredParrotCapability = exocortex.OpExtractEntity
		c.Eligibility, c.Reason = NotApplicableR0, "EXTRACT_ENTITY is EXTERNALIZE in P2-A (DOES_NOT_TRANSFER on real pages); R0 has no deterministic open-vocabulary entity extractor"

	case CategoryFactual + "/exact":
		c.RequiredLogicalOps = []string{"READ_SHORT_TEXT"}
		c.RequiredOperandTypes = []string{"text"}
		c.RequiredParrotCapability = exocortex.OpReadShortText
		c.Eligibility, c.Reason = NotApplicableR0, "open-vocabulary exact-text answer; READ_SHORT_TEXT / EXTRACT_ENTITY are both EXTERNALIZE in P2-A"

	case CategorySynthesis + "/choice":
		c.RequiredEvidenceCount = 2
		c.RequiredLogicalOps = []string{"EXTRACT x2", "COMPARE"}
		c.RequiredOperandTypes = []string{"number", "number"}
		c.RequiredParrotCapability = exocortex.OpExtractNumber
		c.RequiredDeterministicCaps = []string{exocortex.OpCompareNumbers}
		c.Eligibility, c.Reason = NotApplicableR0, "synthesis choice needs a multi-operand comparison, but frozen P0 evidence is a single page-level span; a bounded R0 recipe cannot address the two operands separately without a planner"

	default:
		c.Eligibility, c.Reason = NotApplicableR0, "no R0 recipe rule for ("+record.Category+", "+record.TaskFamily+")"
	}
	return c
}

func refIDs(record P0Record) []string {
	ids := make([]string, 0, len(record.EvidenceRefs))
	for _, ref := range record.EvidenceRefs {
		ids = append(ids, ref.ID)
	}
	return ids
}

// reasonFamily buckets a free-text reason into a coarse family for the
// section 18 "reasons by unsupported family" report.
func reasonFamily(c CaseEligibility) string {
	if c.Eligibility == EligibleR0 {
		return "eligible"
	}
	r := strings.ToLower(c.Reason)
	switch {
	case strings.Contains(r, "externaliz"):
		return "externalized_capability"
	case strings.Contains(r, "multi-operand") || strings.Contains(r, "planner"):
		return "recipe_not_expressible"
	case strings.Contains(r, "choice-width"):
		return "outside_envelope"
	default:
		return "other"
	}
}

// RunEligibilityAudit classifies every record and assembles the frozen
// audit. The coverage denominator is always the full frozen P0 count
// (section 18), never just the eligible subset.
func RunEligibilityAudit(records []P0Record, profile exocortex.CapabilityProfile, p0DatasetSHA256 string) EligibilityAudit {
	audit := EligibilityAudit{
		Schema: EligibilityAuditSchemaR0, ProfileID: profile.ProfileID,
		ProfileSourceHash: profile.SourceArtifactHash, P0DatasetSHA256: p0DatasetSHA256,
		TotalP0Cases: len(records), ReasonFamilyCounts: map[string]int{},
	}
	sorted := append([]P0Record(nil), records...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].BaseID < sorted[j].BaseID })
	for _, record := range sorted {
		c := ClassifyCase(record, profile)
		if c.Eligibility == EligibleR0 {
			audit.EligibleR0++
		} else {
			audit.NotApplicableR0++
		}
		audit.ReasonFamilyCounts[reasonFamily(c)]++
		audit.Cases = append(audit.Cases, c)
	}
	if audit.TotalP0Cases > 0 {
		audit.R0TaskCoverage = float64(audit.EligibleR0) / float64(audit.TotalP0Cases)
	}
	return audit
}
