package learningcycle

import (
	"tlaloc.local/behaviorlab/internal/adaptivesearch"
	"tlaloc.local/behaviorlab/internal/experimentpolicy"
	"tlaloc.local/behaviorlab/internal/learningpolicy"
)

const (
	StatusSchemaR1 = "tlaloc.learning-cycle.r1.status"
	PlanSchemaR1   = "tlaloc.learning-cycle.r1.plan"
)

type Status struct {
	Schema          string                       `json:"schema"`
	FailureFrontier string                       `json:"failure_frontier,omitempty"`
	NextTarget      string                       `json:"next_target,omitempty"`
	Policy          learningpolicy.Policy        `json:"policy"`
	AdaptiveSearch  adaptivesearch.Plan          `json:"adaptive_search"`
	Promotion       string                       `json:"promotion"`
}

type Plan struct {
	Schema     string                            `json:"schema"`
	Status     Status                            `json:"status"`
	Intent     experimentpolicy.ExperimentIntent `json:"intent"`
	Candidates []experimentpolicy.CandidateManifest `json:"candidates"`
}
