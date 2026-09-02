package outcomelearner

const (
	AssessmentSchemaR1 = "tlaloc.outcome-assessment.r1"
	KnowledgeSchemaR1  = "tlaloc.knowledge-update.r1"
)

const (
	OutcomeSuccessfulCausalStep = "SUCCESSFUL_CAUSAL_STEP"
	OutcomeNoImprovement        = "NO_IMPROVEMENT"
	OutcomeRegression           = "REGRESSION"
	OutcomeInvalidExperiment    = "INVALID_EXPERIMENT"
)

type Assertion struct {
	ID    string  `json:"id"`
	Pass  bool    `json:"pass"`
	Score float64 `json:"score,omitempty"`
}

type TrialSnapshot struct {
	CandidateID     string      `json:"candidate_id"`
	ValidSpecimen   bool        `json:"valid_specimen"`
	OverallScore    float64     `json:"overall_score"`
	FailureFrontier string      `json:"failure_frontier,omitempty"`
	Assertions      []Assertion `json:"assertions"`
}

type Request struct {
	HypothesisID    string        `json:"hypothesis_id"`
	TargetAssertion string        `json:"target_assertion"`
	ChangedModules  []string      `json:"changed_modules"`
	Before          TrialSnapshot `json:"before"`
	After           TrialSnapshot `json:"after"`
}

type Assessment struct {
	Schema              string   `json:"schema"`
	HypothesisID        string   `json:"hypothesis_id"`
	Classification      string   `json:"classification"`
	TargetResolved      bool     `json:"target_resolved"`
	BeforeScore         float64  `json:"before_score"`
	AfterScore          float64  `json:"after_score"`
	Delta               float64  `json:"delta"`
	FrontierMoved       bool     `json:"frontier_moved"`
	Regressions         []string `json:"regressions,omitempty"`
	CausalConfidence    string   `json:"causal_confidence"`
	ModelPenaltyAllowed bool     `json:"model_penalty_allowed"`
}

type KnowledgeUpdate struct {
	Schema       string   `json:"schema"`
	HypothesisID string   `json:"hypothesis_id"`
	Action       string   `json:"action"`
	Maturity     string   `json:"maturity,omitempty"`
	Preserve     []string `json:"preserve,omitempty"`
	Avoid        []string `json:"avoid,omitempty"`
	Reason       string   `json:"reason"`
}
