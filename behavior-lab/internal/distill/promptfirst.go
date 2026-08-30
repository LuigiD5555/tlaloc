package distill

import (
	"fmt"
	"sort"
	"strings"
)

const PromptFirstContractID = "tlaloc.prompt-first-distillation.r0"

type DeploymentLevel string

const (
	LevelPromptOnly      DeploymentLevel = "L0_PROMPT_ONLY"
	LevelPromptContext   DeploymentLevel = "L1_PROMPT_PLUS_CONTEXT_OR_IR"
	LevelPromptTools     DeploymentLevel = "L2_PROMPT_PLUS_TOOLS"
	LevelPromptRuntime   DeploymentLevel = "L3_PROMPT_PLUS_RUNTIME"
	LevelSpecialized     DeploymentLevel = "L4_SPECIALIZED_TARGET"
)

type ArtifactCandidate struct {
	ID                 string          `json:"id"`
	Level              DeploymentLevel `json:"level"`
	Prompt             string          `json:"prompt"`
	BehavioralFidelity float64         `json:"behavioral_fidelity"`
	PassRate           float64         `json:"pass_rate"`
	RegressionRate     float64         `json:"regression_rate"`
	PromptTokens       int             `json:"prompt_tokens,omitempty"`
	Dependencies       []string        `json:"dependencies,omitempty"`
	CleanTargetTrials  int             `json:"clean_target_trials"`
	Notes              []string        `json:"notes,omitempty"`
}

type PromptFirstPolicy struct {
	MinBehavioralFidelity float64 `json:"min_behavioral_fidelity"`
	MinPassRate           float64 `json:"min_pass_rate"`
	MaxRegressionRate     float64 `json:"max_regression_rate"`
	MinCleanTargetTrials  int     `json:"min_clean_target_trials"`
}

func DefaultPromptFirstPolicy() PromptFirstPolicy {
	return PromptFirstPolicy{
		MinBehavioralFidelity: 0.95,
		MinPassRate:           0.95,
		MaxRegressionRate:     0.01,
		MinCleanTargetTrials:  3,
	}
}

type ArtifactEvaluation struct {
	ID       string          `json:"id"`
	Level    DeploymentLevel `json:"level"`
	Eligible bool            `json:"eligible"`
	Reasons  []string        `json:"reasons,omitempty"`
}

type PromptFirstDecision struct {
	Schema       string               `json:"schema"`
	SelectedID   string               `json:"selected_id,omitempty"`
	SelectedLevel DeploymentLevel     `json:"selected_level,omitempty"`
	Evaluations  []ArtifactEvaluation `json:"evaluations"`
	Decision     string               `json:"decision"`
}

// SelectPortableArtifact chooses the least demanding deployment level that
// reproduces the reference behavior above policy thresholds. A richer runtime
// cannot win merely by scoring slightly higher when an eligible prompt-only
// artifact already exists.
func SelectPortableArtifact(candidates []ArtifactCandidate, policy PromptFirstPolicy) (PromptFirstDecision, error) {
	policy = normalizePromptFirstPolicy(policy)
	if len(candidates) == 0 {
		return PromptFirstDecision{}, fmt.Errorf("at least one artifact candidate is required")
	}

	seen := map[string]struct{}{}
	evals := make([]ArtifactEvaluation, 0, len(candidates))
	eligible := make([]ArtifactCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.ID) == "" {
			return PromptFirstDecision{}, fmt.Errorf("candidate id is required")
		}
		if _, ok := seen[candidate.ID]; ok {
			return PromptFirstDecision{}, fmt.Errorf("duplicate candidate id %q", candidate.ID)
		}
		seen[candidate.ID] = struct{}{}
		if !validDeploymentLevel(candidate.Level) {
			return PromptFirstDecision{}, fmt.Errorf("candidate %q has unsupported deployment level %q", candidate.ID, candidate.Level)
		}
		reasons := make([]string, 0, 5)
		if strings.TrimSpace(candidate.Prompt) == "" {
			reasons = append(reasons, "prompt is required at every deployment level")
		}
		if candidate.BehavioralFidelity < policy.MinBehavioralFidelity {
			reasons = append(reasons, "behavioral fidelity below threshold")
		}
		if candidate.PassRate < policy.MinPassRate {
			reasons = append(reasons, "pass rate below threshold")
		}
		if candidate.RegressionRate > policy.MaxRegressionRate {
			reasons = append(reasons, "regression rate above threshold")
		}
		if candidate.CleanTargetTrials < policy.MinCleanTargetTrials {
			reasons = append(reasons, "insufficient clean-target trials")
		}
		if candidate.Level == LevelPromptOnly && len(candidate.Dependencies) != 0 {
			reasons = append(reasons, "L0 prompt-only cannot declare runtime/tool dependencies")
		}
		ok := len(reasons) == 0
		evals = append(evals, ArtifactEvaluation{ID: candidate.ID, Level: candidate.Level, Eligible: ok, Reasons: reasons})
		if ok {
			eligible = append(eligible, candidate)
		}
	}

	decision := PromptFirstDecision{
		Schema:      PromptFirstContractID + ".decision",
		Evaluations: evals,
		Decision:    "NO_ARTIFACT_MEETS_BEHAVIORAL_POLICY",
	}
	if len(eligible) == 0 {
		return decision, nil
	}

	sort.SliceStable(eligible, func(i, j int) bool {
		li := deploymentRank(eligible[i].Level)
		lj := deploymentRank(eligible[j].Level)
		if li != lj {
			return li < lj
		}
		if eligible[i].PromptTokens != eligible[j].PromptTokens && eligible[i].PromptTokens > 0 && eligible[j].PromptTokens > 0 {
			return eligible[i].PromptTokens < eligible[j].PromptTokens
		}
		if eligible[i].BehavioralFidelity != eligible[j].BehavioralFidelity {
			return eligible[i].BehavioralFidelity > eligible[j].BehavioralFidelity
		}
		return eligible[i].ID < eligible[j].ID
	})

	winner := eligible[0]
	decision.SelectedID = winner.ID
	decision.SelectedLevel = winner.Level
	decision.Decision = "SELECT_LEAST_DEMANDING_BEHAVIORALLY_VALID_ARTIFACT"
	return decision, nil
}

func normalizePromptFirstPolicy(policy PromptFirstPolicy) PromptFirstPolicy {
	defaults := DefaultPromptFirstPolicy()
	if policy.MinBehavioralFidelity <= 0 { policy.MinBehavioralFidelity = defaults.MinBehavioralFidelity }
	if policy.MinPassRate <= 0 { policy.MinPassRate = defaults.MinPassRate }
	if policy.MaxRegressionRate < 0 { policy.MaxRegressionRate = defaults.MaxRegressionRate }
	if policy.MinCleanTargetTrials <= 0 { policy.MinCleanTargetTrials = defaults.MinCleanTargetTrials }
	return policy
}

func validDeploymentLevel(level DeploymentLevel) bool {
	switch level {
	case LevelPromptOnly, LevelPromptContext, LevelPromptTools, LevelPromptRuntime, LevelSpecialized:
		return true
	default:
		return false
	}
}

func deploymentRank(level DeploymentLevel) int {
	switch level {
	case LevelPromptOnly:
		return 0
	case LevelPromptContext:
		return 1
	case LevelPromptTools:
		return 2
	case LevelPromptRuntime:
		return 3
	case LevelSpecialized:
		return 4
	default:
		return 99
	}
}
