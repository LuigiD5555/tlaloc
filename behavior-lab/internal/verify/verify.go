// Package verify is the Verification Spine: the place Tlaloc decides
// whether a result may be trusted. It has three levels, from cheapest and
// most certain to most expensive and least:
//
//   - STRUCTURAL — deterministic: valid JSON, expected fields present, a
//     hash matches, a number is in range, a value is one of an allowed set.
//   - SEMANTIC — does the evidence actually support the claim? does an
//     independent specialist agree? (delegated to an injected verifier).
//   - WORLD — after an action ran, did it actually happen? (adapted from an
//     executor.Result's postcondition checks).
//
// A claim is VERIFIED only when every level that was asked to run passed.
// The spine never upgrades a verdict on confidence alone.
package verify

const Schema = "tlaloc.verification-spine.r0"

// Level is one tier of the spine.
type Level string

const (
	Structural Level = "STRUCTURAL"
	Semantic   Level = "SEMANTIC"
	World      Level = "WORLD"
)

// Check is one evaluation at some level.
type Check struct {
	Level      Level   `json:"level"`
	Kind       string  `json:"kind"`
	Passed     bool    `json:"passed"`
	Detail     string  `json:"detail,omitempty"`
	Confidence float64 `json:"confidence,omitempty"` // semantic checks only
}

// Verdict is the spine's overall conclusion.
type Verdict string

const (
	// Verified: every check that ran passed.
	Verified Verdict = "VERIFIED"
	// Unverified: at least one check failed.
	Unverified Verdict = "UNVERIFIED"
	// Inconclusive: nothing could be checked (no structural inputs, no
	// semantic verifier, no execution) — never treat as VERIFIED.
	Inconclusive Verdict = "INCONCLUSIVE"
)

// Report is the full verification account.
type Report struct {
	Schema       string  `json:"schema"`
	Verdict      Verdict `json:"verdict"`
	Checks       []Check `json:"checks"`
	FailedLevels []Level `json:"failed_levels,omitempty"`
}

// verdictFrom folds a set of checks into a Report.
func verdictFrom(checks []Check) Report {
	report := Report{Schema: Schema, Checks: checks}
	if len(checks) == 0 {
		report.Verdict = Inconclusive
		return report
	}
	failed := map[Level]bool{}
	for _, check := range checks {
		if !check.Passed {
			failed[check.Level] = true
		}
	}
	if len(failed) == 0 {
		report.Verdict = Verified
		return report
	}
	report.Verdict = Unverified
	for _, level := range []Level{Structural, Semantic, World} {
		if failed[level] {
			report.FailedLevels = append(report.FailedLevels, level)
		}
	}
	return report
}
