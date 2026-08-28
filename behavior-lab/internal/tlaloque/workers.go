package tlaloque

import (
	"tlaloc.local/behaviorlab/internal/compiler"
	"tlaloc.local/behaviorlab/internal/evaluate"
	"tlaloc.local/behaviorlab/internal/spec"
)

type Guard struct {
	name  string
	code  spec.InvariantCode
	patch compiler.Section
}

func (g Guard) Name() string { return g.name }
func (g Guard) Propose(fs []evaluate.Finding) []compiler.Section {
	for _, f := range fs {
		if f.Code == g.code {
			return []compiler.Section{g.patch}
		}
	}
	return nil
}

func DefaultTlaloque() []Tlaloque {
	return []Tlaloque{
		Guard{"collapse_guard", spec.NoImplicitObservation, compiler.Section{ID: "repair:no-collapse", Priority: 120, Text: "HARD REPAIR: Never convert SUPERPOSED/COUPLED to DETERMINATE or OBSERVED during TRANSFORM, INTERFERE, CONSTRAIN, reasoning, explanation, ranking, or convenience. Preserve the full state until an explicit OBSERVE/FOLD instruction is present."}},
		Guard{"branch_guard", spec.TransformPreservesBranches, compiler.Section{ID: "repair:branch-survival", Priority: 118, Text: "HARD REPAIR: Every valid input branch must contribute to the transformed state unless an explicit CONSTRAIN rule removes it or interference cancels its amplitude exactly. Do not keep only the most probable branch."}},
		Guard{"coupling_guard", spec.CoupledIsJointState, compiler.Section{ID: "repair:joint-state", Priority: 119, Text: "HARD REPAIR: COUPLED is one joint state. Do not answer by assigning independent local states to its members unless an explicit decomposition operation is requested."}},
		Guard{"cancellation_guard", spec.ZeroAmplitudeCancellation, compiler.Section{ID: "repair:cancellation", Priority: 117, Text: "HARD REPAIR: zero total complex amplitude means semantic cancellation. It is a known computed result and MUST NOT be labeled unknown, missing, absent by uncertainty, or unresolved."}},
		Guard{"output_guard", spec.StructuredOutputRequired, compiler.Section{ID: "repair:json-only", Priority: 125, Text: "HARD REPAIR: output one valid JSON object only. No markdown, commentary, code fences, prefixes, or suffixes."}},
	}
}
