package verify

import (
	"context"

	"tlaloc.local/behaviorlab/internal/executor"
)

// SemanticVerifier is the injected SEMANTIC level: an independent check
// that the evidence supports the claim. A wrapper around
// internal/tlaloque/answerscore is the natural implementation; the spine
// keeps the interface so it can be a different specialist, or absent.
type SemanticVerifier interface {
	AgreesWith(ctx context.Context, claim, evidence string) (agree bool, confidence float64, err error)
}

// Spine runs the three levels. Semantic is optional.
type Spine struct {
	Semantic SemanticVerifier
	// SemanticMinConfidence: an agreement below this is treated as a
	// non-agreement (the specialist is not sure enough to vouch).
	SemanticMinConfidence float64
}

// Input carries whatever the caller has for each level. A level is skipped
// (contributes no checks) when its inputs are absent.
type Input struct {
	// STRUCTURAL
	Output         []byte
	ExpectedHash   string
	RequiredFields []string

	// SEMANTIC
	Claim    string
	Evidence string

	// WORLD
	Execution *executor.Result
}

// Verify runs every level whose inputs are present and folds the checks
// into one verdict.
func (spine Spine) Verify(ctx context.Context, in Input) Report {
	var checks []Check

	if len(in.Output) > 0 {
		checks = append(checks, JSONValid(in.Output))
		if in.ExpectedHash != "" {
			checks = append(checks, HashMatches(in.Output, in.ExpectedHash))
		}
		if len(in.RequiredFields) > 0 {
			checks = append(checks, RequiredFields(in.Output, in.RequiredFields))
		}
	}

	if spine.Semantic != nil && in.Claim != "" {
		check := Check{Level: Semantic, Kind: "independent_agreement"}
		agree, confidence, err := spine.Semantic.AgreesWith(ctx, in.Claim, in.Evidence)
		check.Confidence = confidence
		switch {
		case err != nil:
			check.Passed = false
			check.Detail = "semantic verifier error: " + err.Error()
		case confidence < spine.SemanticMinConfidence:
			check.Passed = false
			check.Detail = "independent specialist not confident enough to agree"
		default:
			check.Passed = agree
			if !agree {
				check.Detail = "independent specialist does not agree the evidence supports the claim"
			}
		}
		checks = append(checks, check)
	}

	if in.Execution != nil {
		checks = append(checks, FromExecution(*in.Execution)...)
	}

	return verdictFrom(checks)
}

// FromExecution adapts an executor.Result into WORLD-level checks: the
// action must have executed, and every postcondition must have passed.
func FromExecution(result executor.Result) []Check {
	checks := []Check{{
		Level:  World,
		Kind:   "action_executed",
		Passed: result.Executed,
		Detail: result.Failure,
	}}
	for _, post := range result.Postconditions {
		checks = append(checks, Check{
			Level:  World,
			Kind:   "postcondition:" + post.Kind,
			Passed: post.Passed,
			Detail: post.Error,
		})
	}
	if result.RolledBack {
		checks = append(checks, Check{
			Level:  World,
			Kind:   "rolled_back",
			Passed: result.RollbackVerified,
			Detail: "action failed verification and was rolled back",
		})
	}
	return checks
}
