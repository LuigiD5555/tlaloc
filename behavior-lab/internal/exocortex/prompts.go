package exocortex

import (
	"fmt"
	"strings"
)

// fixedInstructions are the one-op, minimal-output prompt templates T0 uses
// for every Parrot-eligible opcode. They are frozen before T0 execution
// (E3): no prompt is optimized, reworded, or widened at runtime, and this
// is the only place they are defined.
var fixedInstructions = map[string]string{
	OpExtractNumber:  "Read the number shown in this image. Reply with only the number, no words.",
	OpExtractEntity:  "Read the single labeled value shown in this image. Reply with only that value.",
	OpReadShortText:  "Read the short text shown in this image. Reply with only that text.",
	OpReadShortLabel: "Read the label shown in this image. Reply with only that label.",
	OpSelectOne:      "Choose exactly one of the shown options. Reply with only your choice.",
	OpSameDifferent:  "Are the two shown values the same or different? Reply with only SAME or DIFFERENT.",
}

// selectOneChoiceTemplate is the frozen SELECT_ONE instruction template used
// when the caller carries the task's explicit choice set (the T0-B locate/
// choice family). The choice universe is part of the frozen P0 task, not
// evidence and not the answer: both options are always presented, in the
// order the frozen task declares them, with no "correct" marker. This
// single template is applied identically to every SELECT_ONE record and to
// C1/C2/C3 alike.
const selectOneChoiceTemplate = "Choose exactly one of these options: %s. Reply with only your choice, copied exactly."

// FixedInstruction returns the frozen prompt for a Parrot-eligible opcode.
func FixedInstruction(opcode string) (string, error) {
	op, err := NormalizeOpcode(opcode)
	if err != nil {
		return "", err
	}
	instruction, ok := fixedInstructions[op]
	if !ok {
		return "", &ContractViolationError{Opcode: op, Reason: "opcode has no fixed one-op instruction template; it is not Parrot-eligible"}
	}
	return instruction, nil
}

// FixedInstructionForOperand returns the frozen prompt for a Parrot-eligible
// opcode, specialised by the operand's declarative shape. Today only
// SELECT_ONE varies: when the operand carries the task's choice labels the
// frozen choice template is filled with them; otherwise the base template
// is returned unchanged. It never consults an expected answer or any
// evidence text.
func FixedInstructionForOperand(opcode string, operand Operand) (string, error) {
	op, err := NormalizeOpcode(opcode)
	if err != nil {
		return "", err
	}
	if op == OpSelectOne && len(operand.Choices) > 0 {
		quoted := make([]string, 0, len(operand.Choices))
		for _, choice := range operand.Choices {
			choice = strings.TrimSpace(choice)
			if choice == "" {
				return "", &ContractViolationError{Opcode: op, Reason: "SELECT_ONE operand carries an empty choice label"}
			}
			quoted = append(quoted, fmt.Sprintf("%q", choice))
		}
		return fmt.Sprintf(selectOneChoiceTemplate, strings.Join(quoted, " or ")), nil
	}
	return FixedInstruction(op)
}
