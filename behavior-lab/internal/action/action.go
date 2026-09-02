// Package action is the deterministic authorization boundary between
// thought and world. Generative and learned Tlaloque produce an
// ActionCandidate — an untrusted suggestion. Nothing executes a candidate.
// Only action.Compile turns a candidate into an ActionIR, and it does so
// deterministically: it looks the capability up in a Catalog, assigns the
// risk class from the Catalog (never from the candidate), checks the
// arguments against the capability's schema and the Policy's constraints,
// and refuses anything above the Policy's risk ceiling.
//
// Architectural invariant #1: INDETERMINISM MAY PROPOSE. DETERMINISM MUST
// AUTHORIZE. EVIDENCE MUST VERIFY. This package is the "DETERMINISM MUST
// AUTHORIZE" half. A model never gets a shell; it gets to fill in an
// ActionCandidate and hope Compile accepts it.
package action

const Schema = "tlaloc.action-ir.r0"

// RiskClass is the irreversibility/blast-radius ladder. The probabilistic
// side can never move an action to a lower (less restrictive) class — the
// class comes from the Catalog entry for the capability.
type RiskClass int

const (
	R0ReadOnly          RiskClass = iota // observation only, no state change
	R1LocalReversible                    // local write with a defined rollback
	R2LocalIrreversible                  // local write that cannot be undone
	R3ExternalEffect                     // effect leaves the machine (email, network)
	R4Privileged                         // privileged / system-level (kernel, packages)
)

func (risk RiskClass) String() string {
	switch risk {
	case R0ReadOnly:
		return "R0_READ_ONLY"
	case R1LocalReversible:
		return "R1_LOCAL_REVERSIBLE"
	case R2LocalIrreversible:
		return "R2_LOCAL_IRREVERSIBLE"
	case R3ExternalEffect:
		return "R3_EXTERNAL_EFFECT"
	case R4Privileged:
		return "R4_PRIVILEGED"
	default:
		return "R?_UNKNOWN"
	}
}

// Valid reports whether risk is one of the defined classes.
func (risk RiskClass) Valid() bool { return risk >= R0ReadOnly && risk <= R4Privileged }

// ActionCandidate is what a Tlaloque proposes. It is untrusted data: the
// Rationale is free text, the risk it thinks it is does not matter, and it
// is never executed. Only Compile consumes it.
type ActionCandidate struct {
	Capability string            `json:"capability"`
	Arguments  map[string]string `json:"arguments"`
	Rationale  string            `json:"rationale,omitempty"`
	ProposedBy string            `json:"proposed_by,omitempty"`
}

// Precondition is a named check the deterministic executor must confirm
// true before running the action. Kind values are drawn from the
// capability's spec (e.g. "source_exists", "destination_absent",
// "path_inside").
type Precondition struct {
	Kind string `json:"kind"`
	Arg  string `json:"arg,omitempty"`
}

// Postcondition is a named check the executor must confirm true after
// running the action — the "world" level of the Verification Spine.
type Postcondition struct {
	Kind string `json:"kind"`
	Arg  string `json:"arg,omitempty"`
}

// ActionIR is an authorized action: capability, validated arguments, the
// risk class the Catalog assigned, the pre/postconditions to enforce, and
// a rollback action when the capability is reversible. Everything downstream
// of an ActionIR is a deterministic machine.
type ActionIR struct {
	Schema                 string            `json:"schema"`
	ActionID               string            `json:"action_id"`
	Capability             string            `json:"capability"`
	Arguments              map[string]string `json:"arguments"`
	Risk                   RiskClass         `json:"risk"`
	RiskName               string            `json:"risk_name"`
	Preconditions          []Precondition    `json:"preconditions"`
	ExpectedPostconditions []Postcondition   `json:"expected_postconditions"`
	Rollback               *ActionIR         `json:"rollback,omitempty"`
	ProposedBy             string            `json:"proposed_by,omitempty"`
}
