package learningpolicy

const SchemaR1 = "tlaloc.learning-policy.r1"

const (
	RulePreserve = "PRESERVE"
	RuleAvoid    = "AVOID"
	RuleRequire  = "REQUIRE"
	RuleMutable  = "MUTABLE"
)

const (
	MaturityHypothesis      = "HYPOTHESIS"
	MaturityObservedWin     = "OBSERVED_WIN"
	MaturityProvisionalWin  = "PROVISIONAL_WIN"
	MaturityReplicatedWin   = "REPLICATED_WIN"
	MaturityCrossModelWin   = "CROSS_MODEL_WIN"
	MaturityCanonicalCandidate = "CANONICAL_CANDIDATE"
)

type Rule struct {
	Kind       string   `json:"kind"`
	Target     string   `json:"target"`
	Reason     string   `json:"reason"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
	Confidence string   `json:"confidence,omitempty"`
}

type LearnedInvariant struct {
	ID          string   `json:"id"`
	Scope       string   `json:"scope"`
	Maturity    string   `json:"maturity"`
	Preserve    []string `json:"preserve"`
	Reason      string   `json:"reason"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
	Models      []string `json:"models,omitempty"`
	Protected   bool     `json:"protected"`
}

type AntiPattern struct {
	ID          string   `json:"id"`
	Trigger     string   `json:"trigger"`
	Failure     string   `json:"failure"`
	Policy      string   `json:"policy"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
}

type Policy struct {
	Schema            string             `json:"schema"`
	Target            string             `json:"target"`
	FailureFrontier   string             `json:"failure_frontier,omitempty"`
	Rules             []Rule             `json:"rules"`
	Invariants        []LearnedInvariant `json:"learned_invariants,omitempty"`
	AntiPatterns      []AntiPattern      `json:"anti_patterns,omitempty"`
	ParentEvidenceIDs []string           `json:"parent_evidence_ids,omitempty"`
	Guardrails        []string           `json:"guardrails"`
	Authority         string             `json:"authority"`
}
