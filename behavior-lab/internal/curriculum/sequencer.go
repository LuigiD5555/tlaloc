package curriculum

import "fmt"

// Case is one labeled example at some stage. Input/ExpectedLabel are
// domain-agnostic strings; a domain that needs richer structure carries it
// in Meta. ExpectedAbstain marks a case where the correct behavior is to
// return UNKNOWN/UNSUPPORTED rather than a label (C3/C4/C8/C9 often).
type Case struct {
	Stage           Stage             `json:"stage"`
	Input           string            `json:"input"`
	ExpectedLabel   string            `json:"expected_label,omitempty"`
	ExpectedAbstain bool              `json:"expected_abstain,omitempty"`
	Meta            map[string]string `json:"meta,omitempty"`
}

// Generator turns a base case into concrete cases for a given stage. It is
// domain-specific and deterministic in seed. Returning an empty slice for
// a stage means "not applicable to this specialist" (e.g. a pure
// classifier has nothing to generate for C5_WORKER_FAILURE).
type Generator interface {
	Generate(base Case, stage Stage, seed int64) ([]Case, error)
}

// Curriculum is the full generated set, grouped by stage in ladder order.
type Curriculum struct {
	Stages    []Stage          `json:"stages"`
	ByStage   map[Stage][]Case `json:"by_stage"`
	CaseCount map[Stage]int    `json:"case_count"`
}

// Build walks the ladder from C0 up to and including upTo, asking gen for
// cases at each stage. Deterministic: the same base + seed + upTo produce
// the same Curriculum.
func Build(base Case, gen Generator, upTo Stage, seed int64) (Curriculum, error) {
	limit := upTo.Index()
	if limit < 0 {
		return Curriculum{}, fmt.Errorf("curriculum: %q is not a ladder stage", upTo)
	}
	result := Curriculum{
		ByStage:   map[Stage][]Case{},
		CaseCount: map[Stage]int{},
	}
	for _, stage := range Ladder {
		if stage.Index() > limit {
			break
		}
		cases, err := gen.Generate(base, stage, seed)
		if err != nil {
			return Curriculum{}, fmt.Errorf("curriculum: generating %s: %w", stage, err)
		}
		for index := range cases {
			cases[index].Stage = stage
		}
		result.Stages = append(result.Stages, stage)
		result.ByStage[stage] = cases
		result.CaseCount[stage] = len(cases)
	}
	return result, nil
}

// StageResult is a specialist's measured behavior on one stage.
type StageResult struct {
	Stage    Stage   `json:"stage"`
	Cases    int     `json:"cases"`
	Correct  int     `json:"correct"`
	Accuracy float64 `json:"accuracy"`
}

// CompetenceFrontier is the highest stage a specialist clears at or above
// minAccuracy, plus the first stage it fails. results need not be in order.
// A specialist that fails C0 has frontier "" (below the ladder).
type CompetenceFrontier struct {
	Holds       Stage   `json:"holds_through"`
	FailsAt     Stage   `json:"fails_at,omitempty"`
	MinAccuracy float64 `json:"min_accuracy"`
}

// AssessFrontier finds where a specialist stops being competent: it walks
// the ladder in order and stops at the first stage below minAccuracy.
// A robustness stage (C5-C7) that a pure classifier did not generate cases
// for is skipped, not treated as a failure.
func AssessFrontier(results []StageResult, minAccuracy float64) CompetenceFrontier {
	byStage := map[Stage]StageResult{}
	for _, result := range results {
		byStage[result.Stage] = result
	}
	frontier := CompetenceFrontier{MinAccuracy: minAccuracy}
	for _, stage := range Ladder {
		result, tested := byStage[stage]
		if !tested || result.Cases == 0 {
			continue
		}
		if result.Accuracy < minAccuracy {
			frontier.FailsAt = stage
			return frontier
		}
		frontier.Holds = stage
	}
	return frontier
}
