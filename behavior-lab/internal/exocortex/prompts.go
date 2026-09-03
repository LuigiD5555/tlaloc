package exocortex

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
