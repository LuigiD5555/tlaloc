// Package envelope is the deterministic boundary around planning and
// execution. It translates a validated intent.CompiledIntent into the
// concrete policies the rest of Tlaloc enforces — today, the action.Policy
// that action.Compile checks every proposed ActionIR against. The envelope
// is the one place allowed to turn "the user asked for X, at risk level Y,
// staying inside Z" into a machine-checkable authorization.
package envelope

import (
	"fmt"
	"strings"

	"tlaloc.local/behaviorlab/internal/action"
	"tlaloc.local/behaviorlab/internal/intent"
)

// riskLevelCeiling is the action-risk ceiling implied by an intent's
// task-level risk posture when it does not set max_action_risk explicitly.
// A high-risk task keeps automatic effects reversible; a low-risk one may
// reach external effects but never privileged operations without an
// explicit opt-in.
var riskLevelCeiling = map[string]action.RiskClass{
	"high":   action.R1LocalReversible,
	"medium": action.R2LocalIrreversible,
	"low":    action.R3ExternalEffect,
}

var actionRiskByName = map[string]action.RiskClass{
	"R0_READ_ONLY":          action.R0ReadOnly,
	"R1_LOCAL_REVERSIBLE":   action.R1LocalReversible,
	"R2_LOCAL_IRREVERSIBLE": action.R2LocalIrreversible,
	"R3_EXTERNAL_EFFECT":    action.R3ExternalEffect,
	"R4_PRIVILEGED":         action.R4Privileged,
}

// PolicyFor builds the action.Policy for a compiled intent. An explicit
// max_action_risk constraint wins; otherwise the task risk level sets the
// ceiling; with neither, the ceiling is the most restrictive (R0) — an
// intent that says nothing about effects gets no authority to cause them.
func PolicyFor(compiled intent.CompiledIntent) (action.Policy, error) {
	ceiling := action.R0ReadOnly

	if explicit := strings.ToUpper(strings.TrimSpace(compiled.MaxActionRisk)); explicit != "" {
		risk, ok := actionRiskByName[explicit]
		if !ok {
			return action.Policy{}, fmt.Errorf("envelope: max_action_risk %q is not a known class", compiled.MaxActionRisk)
		}
		ceiling = risk
	} else if level := strings.ToLower(strings.TrimSpace(compiled.Risk.Level)); level != "" {
		risk, ok := riskLevelCeiling[level]
		if !ok {
			return action.Policy{}, fmt.Errorf("envelope: risk level %q has no action ceiling mapping", compiled.Risk.Level)
		}
		ceiling = risk
	}

	allowed := append([]string(nil), compiled.AllowedCapabilities...)
	sandbox := append([]string(nil), compiled.StayInside...)

	return action.Policy{
		MaxRisk:             ceiling,
		AllowedCapabilities: allowed,
		StayInside:          sandbox,
	}, nil
}

// Authorize is the convenience path: build the policy from the intent, then
// compile the candidate against a catalog under it. This is the full
// second deterministic-boundary crossing —
// IntentIR -> CompiledIntent -> Policy -> ActionIR — in one call.
func Authorize(compiled intent.CompiledIntent, catalog action.Catalog, candidate action.ActionCandidate) (action.ActionIR, error) {
	policy, err := PolicyFor(compiled)
	if err != nil {
		return action.ActionIR{}, err
	}
	return action.Compile(candidate, catalog, policy)
}
