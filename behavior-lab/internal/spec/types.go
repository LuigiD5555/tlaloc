package spec

type StateKind string

const (
	Determinate StateKind = "determinate"
	Superposed  StateKind = "superposed"
	Coupled     StateKind = "coupled"
	Observed    StateKind = "observed"
)

type Operation string

const (
	OpSuperpose Operation = "SUPERPOSE"
	OpTransform Operation = "TRANSFORM"
	OpInterfere Operation = "INTERFERE"
	OpConstrain Operation = "CONSTRAIN"
	OpCouple    Operation = "COUPLE"
	OpObserve   Operation = "OBSERVE"
	OpFold      Operation = "FOLD"
	OpUnfold    Operation = "UNFOLD"
	OpEvolve    Operation = "EVOLVE"
)

type InvariantCode string

const (
	NoImplicitObservation      InvariantCode = "NO_IMPLICIT_OBSERVATION"
	TransformPreservesBranches InvariantCode = "TRANSFORM_PRESERVES_VALID_BRANCHES"
	AbsentIsNotUnknown         InvariantCode = "ABSENT_IS_NOT_UNKNOWN"
	CoupledIsJointState        InvariantCode = "COUPLED_IS_JOINT_STATE"
	ZeroAmplitudeCancellation  InvariantCode = "ZERO_AMPLITUDE_IS_CANCELLATION"
	ObserveHasAuthority        InvariantCode = "OBSERVE_HAS_RESOLUTION_AUTHORITY"
	StructuredOutputRequired   InvariantCode = "STRUCTURED_OUTPUT_REQUIRED"
)

type Invariant struct { Code InvariantCode `json:"code"`; Description string `json:"description"`; Severity int `json:"severity"` }
type Rule struct { Code string `json:"code"`; Description string `json:"description"`; Priority int `json:"priority"` }
type BehaviorSpec struct { Version string `json:"version"`; ID string `json:"id"`; Description string `json:"description"`; Identity string `json:"identity,omitempty"`; StateKinds []StateKind `json:"state_kinds"`; Operations []Operation `json:"operations"`; Rules []Rule `json:"rules,omitempty"`; Invariants []Invariant `json:"invariants"`; Output OutputSpec `json:"output"` }
type OutputSpec struct { Format string `json:"format"`; Schema string `json:"schema,omitempty"` }
