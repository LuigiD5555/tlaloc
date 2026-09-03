package exocortex

import "strings"

// Micro-ISA R0 opcode vocabulary. Opcodes are model-independent logical
// operations; prompt text is never part of a Step (E2). This is
// deliberately the minimal set T0 needs — no speculative opcodes.
const (
	OpLocateRegion   = "LOCATE_REGION"
	OpCropRegion     = "CROP_REGION"
	OpReadShortText  = "READ_SHORT_TEXT"
	OpReadShortLabel = "READ_SHORT_LABEL"
	OpExtractNumber  = "EXTRACT_NUMBER"
	OpExtractEntity  = "EXTRACT_ENTITY"
	OpSelectOne      = "SELECT_ONE"
	OpCompareNumbers = "COMPARE_NUMBERS"
	OpSameDifferent  = "SAME_DIFFERENT"
	OpNormalize      = "NORMALIZE"
	OpVerify         = "VERIFY"
	OpStore          = "STORE"
)

// Opcodes lists the full R0 vocabulary. Anything outside this set is
// rejected by NormalizeOpcode.
func Opcodes() []string {
	return []string{
		OpLocateRegion, OpCropRegion, OpReadShortText, OpReadShortLabel,
		OpExtractNumber, OpExtractEntity, OpSelectOne, OpCompareNumbers,
		OpSameDifferent, OpNormalize, OpVerify, OpStore,
	}
}

// NormalizeOpcode canonicalizes and validates an opcode string against the
// R0 vocabulary.
func NormalizeOpcode(raw string) (string, error) {
	op := strings.ToUpper(strings.TrimSpace(raw))
	for _, known := range Opcodes() {
		if op == known {
			return op, nil
		}
	}
	return "", &UnknownOpcodeError{Opcode: raw}
}

// UnknownOpcodeError reports an opcode outside the R0 Micro-ISA vocabulary.
type UnknownOpcodeError struct{ Opcode string }

func (e *UnknownOpcodeError) Error() string {
	return "exocortex: unknown opcode " + e.Opcode
}

// Address is a logical reference into Origami's addressing space (a region,
// an observation, or a fact), e.g. "region://page176/r3" or
// "observation://fashion_mnist_count". Steps only ever carry Addresses,
// never resolved payloads (E0.8, E3B).
type Address string

// Step is the Exocortex's model-independent unit of work. It intentionally
// mirrors tlaloque.SwarmNode's shape (ID/DependsOn) rather than introducing
// a second DAG representation (E0.15, E2): StepsToPlan below folds a slice
// of Steps directly into a tlaloque.SwarmPlan.
//
// Convention: tlaloque.CapabilityRequest carries no explicit "output
// address" field, so every R0 Tlaloque derives the blackboard Observation
// key it writes from CapabilityRequest.NodeID (the same convention already
// used by internal/lfm2boundary). A Step's ID should therefore equal the
// key portion of its own Output address (e.g. Step{ID: "fashion_mnist_count",
// Output: "observation://fashion_mnist_count"}), and any Step consuming it
// must declare that same string as one of its Inputs.
type Step struct {
	ID           string    `json:"id"`
	Opcode       string    `json:"opcode"`
	Inputs       []Address `json:"inputs"`
	Output       Address   `json:"output"`
	Dependencies []string  `json:"dependencies,omitempty"`
	// DomainHint/ScopeHint/PreferDeterministic/MaxParameters carry straight
	// through to the underlying SwarmNode/SelectionRequest so opcode
	// resolution can prefer a deterministic Tlaloque (E0.9).
	DomainHint          string `json:"domain_hint,omitempty"`
	PreferDeterministic bool   `json:"prefer_deterministic,omitempty"`
	MaxParameters       int64  `json:"max_parameters,omitempty"`
}

// Normalize validates a Step and canonicalizes its opcode.
func (s Step) Normalize() (Step, error) {
	s.ID = strings.TrimSpace(s.ID)
	if s.ID == "" {
		return Step{}, &InvalidStepError{Reason: "id is required"}
	}
	op, err := NormalizeOpcode(s.Opcode)
	if err != nil {
		return Step{}, err
	}
	s.Opcode = op
	if s.Output == "" {
		return Step{}, &InvalidStepError{Reason: "output address is required", StepID: s.ID}
	}
	return s, nil
}

// InvalidStepError reports a structurally invalid Step.
type InvalidStepError struct {
	StepID string
	Reason string
}

func (e *InvalidStepError) Error() string {
	if e.StepID == "" {
		return "exocortex: invalid step: " + e.Reason
	}
	return "exocortex: invalid step " + e.StepID + ": " + e.Reason
}
