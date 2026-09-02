// Package sentinel is Tlaloque species D: workers that do not try to solve
// the task, only to find why another worker's answer or a proposed action
// might be wrong. A sentinel emits Concerns — arguments for doubt, not
// verdicts. A Panel aggregates them; a Block-severity concern must stop an
// answer from being trusted or an action from auto-executing.
//
// Sentinels are deliberately deterministic here: an independent
// probabilistic check is possible but a rule that can only ever raise a
// flag (never lower one) is the safe default for the OS-facing path.
package sentinel

import (
	"context"

	"tlaloc.local/behaviorlab/internal/action"
	"tlaloc.local/behaviorlab/internal/tlaloque/calibration"
	"tlaloc.local/behaviorlab/internal/verify"
)

// Severity ranks a concern.
type Severity string

const (
	Info  Severity = "INFO"  // worth recording, does not change any decision
	Warn  Severity = "WARN"  // the answer/action is suspect; a human should see this
	Block Severity = "BLOCK" // must not auto-execute / must not be treated as VERIFIED
)

// Concern is one sentinel's objection.
type Concern struct {
	Sentinel string   `json:"sentinel"`
	Kind     string   `json:"kind"`
	Severity Severity `json:"severity"`
	Detail   string   `json:"detail"`
}

// Observation is a typed blackboard claim, enough for the conflict check.
type Observation struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Source string `json:"source"`
}

// Subject is everything a sentinel may inspect. Fields are optional; a
// sentinel that needs one it does not have simply returns no concerns.
type Subject struct {
	Question         string
	Answer           string
	AnswerConfidence float64
	Evidence         string

	ProposedAction *action.ActionIR
	Policy         *action.Policy
	AllowedPaths   []string

	Observations []Observation
	Calibration  *calibration.CalibrationProfile
}

// Sentinel inspects a Subject and returns whatever it finds wrong.
type Sentinel interface {
	Name() string
	Inspect(ctx context.Context, subject Subject) ([]Concern, error)
}

// Panel runs a set of sentinels and aggregates their concerns.
type Panel struct {
	Sentinels []Sentinel
}

// DefaultPanel is the standard set of deterministic sentinels.
func DefaultPanel() Panel {
	return Panel{Sentinels: []Sentinel{
		PermissionSentinel{},
		ScopeSentinel{},
		ConflictSentinel{},
		OODSentinel{},
		NumericConsistencySentinel{},
	}}
}

// PanelResult is the aggregate.
type PanelResult struct {
	Concerns []Concern `json:"concerns"`
	Blocked  bool      `json:"blocked"`
}

// Review runs every sentinel. A sentinel error is itself a Block concern —
// a check that could not run is not a check that passed.
func (panel Panel) Review(ctx context.Context, subject Subject) PanelResult {
	result := PanelResult{}
	for _, worker := range panel.Sentinels {
		concerns, err := worker.Inspect(ctx, subject)
		if err != nil {
			concerns = append(concerns, Concern{
				Sentinel: worker.Name(), Kind: "sentinel_error", Severity: Block,
				Detail: "sentinel could not run: " + err.Error(),
			})
		}
		for _, concern := range concerns {
			if concern.Sentinel == "" {
				concern.Sentinel = worker.Name()
			}
			result.Concerns = append(result.Concerns, concern)
			if concern.Severity == Block {
				result.Blocked = true
			}
		}
	}
	return result
}

// ToChecks turns concerns into verify.Check entries so the Verification
// Spine can carry them: a Block or Warn concern is a failed SEMANTIC check.
func (result PanelResult) ToChecks() []verify.Check {
	checks := make([]verify.Check, 0, len(result.Concerns))
	for _, concern := range result.Concerns {
		checks = append(checks, verify.Check{
			Level:  verify.Semantic,
			Kind:   "sentinel:" + concern.Kind,
			Passed: concern.Severity == Info,
			Detail: concern.Detail,
		})
	}
	return checks
}
