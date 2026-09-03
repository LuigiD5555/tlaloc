package exocortex

import (
	"fmt"
	"strings"
)

// ContractViolationError is returned instead of ever silently widening a
// CapabilityProfile's measured envelope. A caller that receives this must
// route to the declared fallback rather than retry with the same executor.
type ContractViolationError struct {
	Opcode string
	Reason string
}

func (e *ContractViolationError) Error() string {
	return fmt.Sprintf("CAPABILITY_CONTRACT_VIOLATION: opcode %s: %s", e.Opcode, e.Reason)
}

// Operand describes what a Step actually asks an executor to look at. It is
// deliberately small and declarative so the ModelAdapter can check it
// against a CapabilityProfile without touching any model prompt.
type Operand struct {
	VisualField string `json:"visual_field,omitempty"` // TIGHT_CROP | FULL_PAGE
	CharCount   int    `json:"char_count,omitempty"`   // for READ_SHORT_TEXT/LABEL
	ChoiceWidth int    `json:"choice_width,omitempty"` // for SELECT_ONE
}

// AdapterPlan is what a ModelAdapter hands back for one Step: everything a
// Tlaloque needs to execute, and nothing else (no workflow history, no
// prompt optimization at runtime — templates are fixed, see prompts.go).
type AdapterPlan struct {
	Opcode         string
	Executor       string
	InputType      string
	VisualField    string
	Instruction    string
	OutputContract OutputContract
	Fallback       []string
}

// ModelAdapter turns an (opcode, operand, CapabilityProfile) triple into an
// AdapterPlan, or refuses with a ContractViolationError. It never widens a
// profile's declared limits and never optimizes a prompt at runtime (E3).
type ModelAdapter struct {
	Profile CapabilityProfile
}

// Adapt produces the AdapterPlan for one Step's operand, or rejects it.
func (a ModelAdapter) Adapt(opcode string, operand Operand) (AdapterPlan, error) {
	op, err := NormalizeOpcode(opcode)
	if err != nil {
		return AdapterPlan{}, err
	}
	entry, ok := a.Profile.Entry(op)
	if !ok {
		return AdapterPlan{}, &ContractViolationError{Opcode: op, Reason: fmt.Sprintf("executor %s has no capability entry for this opcode", a.Profile.ExecutorID)}
	}
	if entry.DeploymentRecommendation == DeploymentDoNotDeploy {
		return AdapterPlan{}, &ContractViolationError{Opcode: op, Reason: "deployment_recommendation is DO_NOT_DEPLOY"}
	}
	if entry.DeploymentRecommendation == DeploymentExternalize {
		return AdapterPlan{}, &ContractViolationError{Opcode: op, Reason: "deployment_recommendation is EXTERNALIZE (response collapse or non-transferring capability); route to a deterministic alternative"}
	}

	visualField := strings.ToUpper(strings.TrimSpace(operand.VisualField))
	if visualField != "" && len(entry.Constraints.AllowedVisualField) > 0 && !containsString(entry.Constraints.AllowedVisualField, visualField) {
		return AdapterPlan{}, &ContractViolationError{Opcode: op, Reason: fmt.Sprintf("visual_field %s is outside the profile's allowed field %v", visualField, entry.Constraints.AllowedVisualField)}
	}
	if entry.Constraints.MaxChars > 0 && operand.CharCount > entry.Constraints.MaxChars {
		return AdapterPlan{}, &ContractViolationError{Opcode: op, Reason: fmt.Sprintf("char_count %d exceeds profile max_chars %d", operand.CharCount, entry.Constraints.MaxChars)}
	}
	if entry.Constraints.FormalMaxChoiceWidth > 0 && operand.ChoiceWidth > entry.Constraints.FormalMaxChoiceWidth {
		return AdapterPlan{}, &ContractViolationError{Opcode: op, Reason: fmt.Sprintf("choice_width %d exceeds the formal_max_choice_width %d (the preregistered conservative rung, not the wider observed envelope)", operand.ChoiceWidth, entry.Constraints.FormalMaxChoiceWidth)}
	}

	instruction, err := FixedInstruction(op)
	if err != nil {
		return AdapterPlan{}, err
	}
	if visualField == "" {
		visualField = entry.Constraints.PreferredVisualField
	}
	return AdapterPlan{
		Opcode:         op,
		Executor:       a.Profile.ExecutorID,
		InputType:      entry.PreferredInput.Type,
		VisualField:    visualField,
		Instruction:    instruction,
		OutputContract: entry.OutputContract,
		Fallback:       entry.FallbackSuggestions,
	}, nil
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
