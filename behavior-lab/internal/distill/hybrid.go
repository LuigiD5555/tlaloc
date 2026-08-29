package distill

import "fmt"

const HybridArtifactSchema = "tlaloc.origami-hybrid-artifact-set.r0"

// HybridArtifactSet is Tlaloc's complete proposal to Origami after a rich
// swarm/Tlaloque behavior has been distilled. The external receiver prompt and
// the carrier-embedded behavior are deliberately separate artifacts.
// Origami remains responsible for semantic validation and promotion.
type HybridArtifactSet struct {
	Schema             string      `json:"schema"`
	CandidateID        string      `json:"candidate_id"`
	UniversalPrompt    string      `json:"universal_prompt"`
	BootStrategy       []string    `json:"boot_strategy"`
	RosettaConstraints []string    `json:"rosetta_constraints"`
	MicroProgram       []MicroRule `json:"micro_program"`
	SourceTraceSHA256  string      `json:"source_trace_sha256"`
	WorkingWindow      int         `json:"working_window_token_eq"`
}

// BuildHybridArtifactSet converts an already distilled candidate into the
// explicit two-target form used by the Hybrid Receiver:
//   1. UniversalPrompt -> external receiver bootstrap.
//   2. Boot/Rosetta/MicroProgram -> carrier/runtime material owned by Origami.
// The function does not promote the proposal and does not assign physical glyph
// meanings; those remain carrier-local Origami concerns.
func BuildHybridArtifactSet(candidate Candidate, workingWindow int) (HybridArtifactSet, error) {
	if candidate.Schema != ContractID {
		return HybridArtifactSet{}, fmt.Errorf("candidate schema must be %q", ContractID)
	}
	if candidate.ID == "" || candidate.SourceTraceSHA256 == "" {
		return HybridArtifactSet{}, fmt.Errorf("candidate id and source trace hash are required")
	}
	if candidate.Prompt == "" {
		return HybridArtifactSet{}, fmt.Errorf("universal receiver prompt is required")
	}
	if len(candidate.BootStrategy) == 0 || len(candidate.RosettaConstraints) == 0 || len(candidate.Program) == 0 {
		return HybridArtifactSet{}, fmt.Errorf("boot strategy, rosetta constraints and micro-program are required")
	}
	if workingWindow <= 0 {
		return HybridArtifactSet{}, fmt.Errorf("working window must be positive")
	}
	return HybridArtifactSet{
		Schema:             HybridArtifactSchema,
		CandidateID:        candidate.ID,
		UniversalPrompt:    candidate.Prompt,
		BootStrategy:       append([]string(nil), candidate.BootStrategy...),
		RosettaConstraints: append([]string(nil), candidate.RosettaConstraints...),
		MicroProgram:       append([]MicroRule(nil), candidate.Program...),
		SourceTraceSHA256:  candidate.SourceTraceSHA256,
		WorkingWindow:      workingWindow,
	}, nil
}
