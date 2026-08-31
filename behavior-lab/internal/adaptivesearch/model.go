package adaptivesearch

import "tlaloc.local/behaviorlab/internal/visualsearch"

const SchemaR0 = "tlaloc.adaptive-search.r0"

type PatternFocus struct {
	Key             string   `json:"key"`
	Stage           string   `json:"stage,omitempty"`
	FailureCode     string   `json:"failure_code,omitempty"`
	ScoreLayer      string   `json:"score_layer,omitempty"`
	Count           int      `json:"count"`
	Models          []string `json:"models,omitempty"`
	Specimens       []string `json:"specimens,omitempty"`
	Questions       []string `json:"questions,omitempty"`
	SuggestedTarget string   `json:"suggested_target,omitempty"`
}

type HistoricalSignal struct {
	MutationKind visualsearch.MutationKind `json:"mutation_kind"`
	Outcomes     int                       `json:"outcomes"`
	MeanDelta    float64                   `json:"mean_delta"`
	Adjustment   float64                   `json:"bounded_adjustment"`
}

type MutationPriority struct {
	Rank       int                       `json:"rank"`
	Kind       visualsearch.MutationKind `json:"kind"`
	Weight     float64                   `json:"weight"`
	Target     string                    `json:"target,omitempty"`
	Reason     string                    `json:"reason"`
	ExplorationFloor bool                `json:"exploration_floor"`
}

type SuggestedMutation struct {
	Kind         visualsearch.MutationKind `json:"kind"`
	Target       string                    `json:"target"`
	Value        string                    `json:"value"`
	Rationale    string                    `json:"rationale"`
	Experimental bool                      `json:"experimental"`
}

type Plan struct {
	Schema              string             `json:"schema"`
	MemoryRoot          string             `json:"memory_root"`
	Adaptive            bool               `json:"adaptive"`
	RealFailureEvents   int                `json:"real_failure_events"`
	NextDebugTarget     string             `json:"next_debug_target,omitempty"`
	PrimaryPattern      *PatternFocus      `json:"primary_pattern,omitempty"`
	FailurePatterns     []PatternFocus     `json:"failure_patterns,omitempty"`
	ParentEvidenceIDs   []string           `json:"parent_evidence_ids,omitempty"`
	MutationPriorities  []MutationPriority `json:"mutation_priorities"`
	SuggestedMutations  []SuggestedMutation `json:"suggested_mutations,omitempty"`
	HistoricalSignals   []HistoricalSignal `json:"historical_signals,omitempty"`
	Guardrails          []string           `json:"guardrails"`
}

type CandidatePriority struct {
	Rank          int                       `json:"rank"`
	CandidateID   string                    `json:"candidate_id"`
	PriorityScore float64                   `json:"priority_score"`
	MutationKinds []visualsearch.MutationKind `json:"mutation_kinds"`
	MatchedTarget string                    `json:"matched_target,omitempty"`
	Reason        string                    `json:"reason"`
}

type Queue struct {
	Schema         string              `json:"schema"`
	Plan           Plan                `json:"plan"`
	CandidateOrder []CandidatePriority `json:"candidate_order"`
	Authority      string              `json:"authority"`
}
