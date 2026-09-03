package exocortex

import (
	"fmt"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// StepsToPlan folds a fixed, bounded sequence of Steps into a
// tlaloque.SwarmPlan. This reuses the existing SwarmNode/SwarmPlan DAG
// representation rather than introducing a second one (E0.15, E2): opcodes
// become SwarmNode.Capability, and Step.Dependencies become
// SwarmNode.DependsOn. T0 only ever compiles small, fixed recipes (E0.13,
// E0.14) — there is no free-form planner here.
func StepsToPlan(planID string, steps []Step, maxParallel int) (tlaloque.SwarmPlan, error) {
	if len(steps) == 0 {
		return tlaloque.SwarmPlan{}, fmt.Errorf("exocortex: at least one step is required")
	}
	nodes := make([]tlaloque.SwarmNode, 0, len(steps))
	for _, raw := range steps {
		step, err := raw.Normalize()
		if err != nil {
			return tlaloque.SwarmPlan{}, err
		}
		nodes = append(nodes, tlaloque.SwarmNode{
			ID:                  step.ID,
			Capability:          step.Opcode,
			DependsOn:           append([]string(nil), step.Dependencies...),
			DomainHint:          step.DomainHint,
			PreferDeterministic: step.PreferDeterministic,
			MaxParameters:       step.MaxParameters,
		})
	}
	plan := tlaloque.SwarmPlan{ID: planID, MaxParallel: maxParallel, Nodes: nodes}
	return plan.Normalize()
}
