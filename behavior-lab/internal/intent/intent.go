// Package intent is the layer above Tlaloc's capability planner. A person
// asks for an outcome ("analyze this document, tell me what's happening,
// verify it, don't invent anything"); the capability planner works in
// capabilities and DAGs. IntentIR is the formal object in between, and
// Compile turns it into the capability goals the existing
// tlaloque.Registry.ResolveGoal already understands.
//
// This package does not execute anything and does not change the runtime.
// It produces plans and carries the intent's invariants, evidence
// requirements, budget and risk profile forward so downstream machinery
// (verification, accounting, promotion gates) can enforce them.
package intent

const Schema = "tlaloc.intent-ir.r0"

// TypedInput is one thing the caller already has and is providing.
type TypedInput struct {
	Name string `json:"name"`
	// Kind is a data-contract / product name; when it matches a product a
	// worker Requires, planning treats it as already available.
	Kind string `json:"kind"`
}

// Constraint is a machine-actionable limit translated into planner hints.
// Known kinds: "max_parameters", "prefer_deterministic", "domain",
// "scope", "max_latency_ms". Unknown kinds are preserved but ignored by
// Compile (a downstream consumer may still use them).
type Constraint struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// Invariant is a statement that must hold over the whole run (e.g. "never
// state a fact not grounded in the provided document"). intent does not
// enforce it — it carries it to whatever does.
type Invariant struct {
	ID        string `json:"id"`
	Statement string `json:"statement"`
}

// Criterion is a measurable success condition.
type Criterion struct {
	ID        string  `json:"id"`
	Metric    string  `json:"metric"`
	Threshold float64 `json:"threshold"`
}

// EvidenceLevel is the A/B/C/D ladder: A same-generator, B render/generator
// shift, C perturbation/adversarial/OOD, D real environment. Only D
// authorizes strong claims.
type EvidenceLevel string

const (
	EvidenceA EvidenceLevel = "A"
	EvidenceB EvidenceLevel = "B"
	EvidenceC EvidenceLevel = "C"
	EvidenceD EvidenceLevel = "D"
)

// EvidenceRequirement is the minimum evidence level a given required
// output must have before its result may be trusted.
type EvidenceRequirement struct {
	ForOutput string        `json:"for_output"`
	MinLevel  EvidenceLevel `json:"min_level"`
}

// Budget is the resource ceiling for the whole run.
type Budget struct {
	MaxTokens        int   `json:"max_tokens,omitempty"`
	MaxWallMS        int64 `json:"max_wall_ms,omitempty"`
	MaxUpstreamCalls int   `json:"max_upstream_calls,omitempty"`
}

// RiskProfile shapes how conservative planning and execution should be.
type RiskProfile struct {
	// Level is "low", "medium" or "high".
	Level string `json:"level"`
	// AbstentionRequired means every learned specialist in the plan must
	// be able to return UNKNOWN/UNSUPPORTED/LOW_EVIDENCE (see
	// internal/tlaloque/calibration) rather than always answering.
	AbstentionRequired bool `json:"abstention_required,omitempty"`
}

// IntentIR is the compiled-from-human, pre-prompt representation of what
// the caller wants. It is deliberately upstream of PromptIR: a prompt is
// one operational artifact a plan might need, not the intent itself.
type IntentIR struct {
	Schema  string `json:"schema"`
	Version string `json:"version"`

	Goal            string       `json:"goal"`
	Inputs          []TypedInput `json:"inputs,omitempty"`
	RequiredOutputs []string     `json:"required_outputs"`

	Constraints     []Constraint `json:"constraints,omitempty"`
	Invariants      []Invariant  `json:"invariants,omitempty"`
	SuccessCriteria []Criterion  `json:"success_criteria,omitempty"`

	EvidenceRequirements []EvidenceRequirement `json:"evidence_requirements,omitempty"`
	Budget               Budget                `json:"budget"`
	Risk                 RiskProfile           `json:"risk"`
}
