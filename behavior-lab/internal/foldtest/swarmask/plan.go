package swarmask

import "tlaloc.local/behaviorlab/internal/tlaloque"

// planMaxParallel is the fan-out budget for the classifier/scout/entities
// layer.
const planMaxParallel = 3

// NewPlan builds the swarm DAG by resolving the terminal capability
// (CONSOLIDATE_BLACKBOARD) against registry — the same
// tlaloque.Registry.ResolveGoal the intent planner uses. The dependency
// shape is not hand-wired here; it comes from each worker's descriptor
// Dependencies:
//
//	classifier ─┐
//	scout ──────┼──► answer ──► consolidate   (terminal)
//	entities ───┘
//
// consolidate.Dependencies = [ANSWER_QUESTION]; answer.Dependencies =
// [SUGGEST_RELEVANT_PAGE, EXTRACT_QUESTION_ENTITIES, CLASSIFY_QUESTION_TYPE].
// ResolveGoal pins each producer as a node keyed by its worker ID.
// Blackboard visibility is global per RunID, so consolidate still sees the
// scout's observation even though it only depends on answer.
func NewPlan(registry *tlaloque.Registry) (tlaloque.SwarmPlan, error) {
	resolved, err := registry.ResolveGoal(
		tlaloque.CapabilityGoal{Capability: ConsolidatorCapability},
		"swarmask-r0",
		planMaxParallel,
	)
	if err != nil {
		return tlaloque.SwarmPlan{}, err
	}
	return resolved.Plan, nil
}
