package canonicalstate

import (
	"fmt"
	"sort"
	"strings"
)

const (
	ResidualSchemaR1        = "tlaloc.residual.r1"
	RefinementEpochSchemaR1 = "tlaloc.refinement-epoch.r1"
)

type ResidualKind string

type RefinementDecision string

const (
	ResidualConflict    ResidualKind = "CONFLICT"
	ResidualUncertainty ResidualKind = "UNCERTAINTY"

	RefinementContinue  RefinementDecision = "CONTINUE"
	RefinementComplete  RefinementDecision = "COMPLETE"
	RefinementExhausted RefinementDecision = "EXHAUSTED"
)

type Residual struct {
	Schema          string       `json:"schema"`
	ID              string       `json:"id"`
	Kind            ResidualKind `json:"kind"`
	Epoch           int          `json:"epoch"`
	ParentStateHash string       `json:"parent_state_hash_sha256"`
	ConflictID      string       `json:"conflict_id,omitempty"`
	ClaimKey        string       `json:"claim_key,omitempty"`
	CandidateIDs    []string     `json:"candidate_ids,omitempty"`
	Values          []string     `json:"values,omitempty"`
	Evidence        []string     `json:"evidence,omitempty"`
	Action          string       `json:"action"`
	Priority        float64      `json:"priority"`
	ContextBudget   int          `json:"context_budget"`
}

type RefinementPolicy struct {
	MaxEpochs            int     `json:"max_epochs"`
	MaxResidualsPerEpoch int     `json:"max_residuals_per_epoch"`
	ContextBudget        int     `json:"context_budget"`
	UncertaintyThreshold float64 `json:"uncertainty_threshold"`
}

func (p RefinementPolicy) Normalize() RefinementPolicy {
	if p.MaxEpochs <= 0 {
		p.MaxEpochs = 3
	}
	if p.MaxResidualsPerEpoch <= 0 {
		p.MaxResidualsPerEpoch = 16
	}
	if p.ContextBudget <= 0 {
		p.ContextBudget = 4000
	}
	if p.UncertaintyThreshold <= 0 || p.UncertaintyThreshold > 1 {
		p.UncertaintyThreshold = 0.35
	}
	return p
}

type RefinementEpoch struct {
	Schema          string             `json:"schema"`
	Index           int                `json:"index"`
	ParentStateHash string             `json:"parent_state_hash_sha256"`
	Uncertainty     float64            `json:"uncertainty"`
	Decision        RefinementDecision `json:"decision"`
	Reason          string             `json:"reason"`
	Residuals       []Residual         `json:"residuals,omitempty"`
}

// BuildRefinementEpoch turns unresolved canonical-state work into a bounded,
// deterministic set of residual tasks. It does not execute workers and does not
// mutate State; the caller materializes the residuals as a new dataflow fragment
// and reduces the resulting candidates into the next immutable state.
func BuildRefinementEpoch(state State, epochIndex int, policy RefinementPolicy) (RefinementEpoch, error) {
	if epochIndex < 0 {
		return RefinementEpoch{}, fmt.Errorf("refinement epoch index must be non-negative")
	}
	policy = policy.Normalize()
	epoch := RefinementEpoch{
		Schema:          RefinementEpochSchemaR1,
		Index:           epochIndex,
		ParentStateHash: state.StateHash,
		Uncertainty:     state.Metrics.Uncertainty,
	}

	needsConflictWork := len(state.Conflicts) > 0
	needsUncertaintyWork := !needsConflictWork && state.Metrics.Uncertainty > policy.UncertaintyThreshold
	if !needsConflictWork && !needsUncertaintyWork {
		epoch.Decision = RefinementComplete
		epoch.Reason = "STATE_SATISFIED"
		return epoch, nil
	}
	if epochIndex >= policy.MaxEpochs {
		epoch.Decision = RefinementExhausted
		epoch.Reason = "MAX_EPOCHS_REACHED"
		return epoch, nil
	}

	if needsConflictWork {
		verification := BuildVerificationPlan(state, policy.ContextBudget)
		conflicts := make(map[string]Conflict, len(state.Conflicts))
		for _, conflict := range state.Conflicts {
			conflicts[conflict.ID] = conflict
		}
		for _, task := range verification.Tasks {
			conflict, ok := conflicts[task.ConflictID]
			if !ok {
				continue
			}
			epoch.Residuals = append(epoch.Residuals, Residual{
				Schema:          ResidualSchemaR1,
				ID:              fmt.Sprintf("residual:%d:%s", epochIndex, task.ID),
				Kind:            ResidualConflict,
				Epoch:           epochIndex,
				ParentStateHash: state.StateHash,
				ConflictID:      conflict.ID,
				ClaimKey:        conflict.ClaimKey,
				CandidateIDs:    conflictCandidateIDs(conflict),
				Values:          uniqueSortedResidualStrings(conflict.Values),
				Evidence:        uniqueSortedResidualStrings(task.Evidence),
				Action:          task.Action,
				Priority:        task.Priority,
				ContextBudget:   task.ContextBudget,
			})
		}
	} else {
		stateID := strings.TrimSpace(state.StateHash)
		if stateID == "" {
			stateID = "unhashed-state"
		}
		epoch.Residuals = append(epoch.Residuals, Residual{
			Schema:          ResidualSchemaR1,
			ID:              fmt.Sprintf("residual:%d:expand:%s", epochIndex, stateID),
			Kind:            ResidualUncertainty,
			Epoch:           epochIndex,
			ParentStateHash: state.StateHash,
			Action:          "EXPAND_EVIDENCE",
			Priority:        state.Metrics.Uncertainty,
			ContextBudget:   policy.ContextBudget,
		})
	}

	sort.Slice(epoch.Residuals, func(i, j int) bool {
		if epoch.Residuals[i].Priority != epoch.Residuals[j].Priority {
			return epoch.Residuals[i].Priority > epoch.Residuals[j].Priority
		}
		return epoch.Residuals[i].ID < epoch.Residuals[j].ID
	})
	if len(epoch.Residuals) > policy.MaxResidualsPerEpoch {
		epoch.Residuals = append([]Residual(nil), epoch.Residuals[:policy.MaxResidualsPerEpoch]...)
	}
	if len(epoch.Residuals) == 0 {
		epoch.Decision = RefinementExhausted
		epoch.Reason = "NO_ACTIONABLE_RESIDUALS"
		return epoch, nil
	}
	epoch.Decision = RefinementContinue
	epoch.Reason = "RESIDUALS_AVAILABLE"
	return epoch, nil
}

func conflictCandidateIDs(conflict Conflict) []string {
	ids := make([]string, 0, len(conflict.PositiveIDs)+len(conflict.NegativeIDs)+len(conflict.CandidateIDs))
	ids = append(ids, conflict.PositiveIDs...)
	ids = append(ids, conflict.NegativeIDs...)
	ids = append(ids, conflict.CandidateIDs...)
	return uniqueSortedResidualStrings(ids)
}

func uniqueSortedResidualStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
