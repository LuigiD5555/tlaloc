package swarmask

import "tlaloc.local/behaviorlab/internal/tlaloque"

// NewPlan returns the 5-node DAG:
//
//	classifier ─┐
//	scout ──────┼──► answer ──► consolidate   (terminal)
//	entities ───┘
//
// scout/entities/classifier run independently (and, with MaxParallel 3,
// concurrently); answer depends on all three so their observations are
// guaranteed present in its blackboard snapshot. consolidate depends only
// on answer — blackboard visibility is global per RunID, not per DAG edge
// (see swarm_runtime.go), and scout has necessarily already run by the time
// answer (and so consolidate) does. SwarmRunner.Run normalizes the plan
// itself, so this can be passed as-is.
func NewPlan() tlaloque.SwarmPlan {
	return tlaloque.SwarmPlan{
		Schema:      tlaloque.SwarmSchemaR0,
		ID:          "swarmask-r0",
		MaxParallel: 3,
		Nodes: []tlaloque.SwarmNode{
			{ID: "scout", Capability: ScoutCapability},
			{ID: "entities", Capability: EntityCapability},
			{ID: "classifier", Capability: ClassifierCapability},
			{ID: "answer", Capability: AnswerCapability, DependsOn: []string{"scout", "entities", "classifier"}},
			{ID: "consolidate", Capability: ConsolidatorCapability, DependsOn: []string{"answer"}},
		},
	}
}
