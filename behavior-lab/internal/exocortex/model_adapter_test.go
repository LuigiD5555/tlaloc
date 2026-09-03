package exocortex

import "testing"

func fixtureProfile(t *testing.T) CapabilityProfile {
	t.Helper()
	path := fixtureArtifact(t)
	profile, err := CompileParrotProfile(path, "parrot-lfm2-vl-1.6b", "lfm2-vl-1.6b", "r0")
	if err != nil {
		t.Fatalf("CompileParrotProfile: %v", err)
	}
	return profile
}

func TestModelAdapter_AcceptsTightCropWithinContract(t *testing.T) {
	adapter := ModelAdapter{Profile: fixtureProfile(t)}
	plan, err := adapter.Adapt(OpExtractNumber, Operand{VisualField: VisualFieldTightCrop})
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	if plan.Instruction == "" {
		t.Fatalf("expected a fixed instruction")
	}
	if plan.VisualField != VisualFieldTightCrop {
		t.Fatalf("visual field = %q, want TIGHT_CROP", plan.VisualField)
	}
}

func TestModelAdapter_RejectsFullPageWhenOnlyTightCropAllowed(t *testing.T) {
	adapter := ModelAdapter{Profile: fixtureProfile(t)}
	_, err := adapter.Adapt(OpExtractNumber, Operand{VisualField: VisualFieldFullPage})
	if err == nil {
		t.Fatalf("expected CAPABILITY_CONTRACT_VIOLATION for full-page EXTRACT_NUMBER")
	}
	if _, ok := err.(*ContractViolationError); !ok {
		t.Fatalf("expected *ContractViolationError, got %T: %v", err, err)
	}
}

func TestModelAdapter_RejectsCharCountAboveMax(t *testing.T) {
	adapter := ModelAdapter{Profile: fixtureProfile(t)}
	_, err := adapter.Adapt(OpReadShortText, Operand{VisualField: VisualFieldTightCrop, CharCount: 64})
	if err == nil {
		t.Fatalf("expected CAPABILITY_CONTRACT_VIOLATION for a 64-char full read against an 8-char envelope")
	}
}

func TestModelAdapter_RejectsChoiceWidthAboveFormalRung(t *testing.T) {
	adapter := ModelAdapter{Profile: fixtureProfile(t)}
	// The fixture's observed envelope is 8, but the formal rung is 2 — the
	// adapter must enforce the formal rung, not the wider observed one.
	_, err := adapter.Adapt(OpSelectOne, Operand{ChoiceWidth: 3})
	if err == nil {
		t.Fatalf("expected rejection at choice width 3 against a formal rung of 2, even though 8 was observed safe")
	}
	if _, err := adapter.Adapt(OpSelectOne, Operand{ChoiceWidth: 2}); err != nil {
		t.Fatalf("choice width 2 (== formal rung) should be accepted: %v", err)
	}
}

func TestModelAdapter_RejectsExternalizeCandidateOutright(t *testing.T) {
	adapter := ModelAdapter{Profile: fixtureProfile(t)}
	_, err := adapter.Adapt("VISUAL_LOCATE", Operand{})
	// VISUAL_LOCATE is not in the R0 opcode vocabulary (E2 keeps the
	// vocabulary minimal); confirm it is rejected as unknown rather than
	// silently accepted.
	if err == nil {
		t.Fatalf("expected VISUAL_LOCATE to be rejected: not part of the R0 Micro-ISA vocabulary")
	}
}

func TestModelAdapter_RejectsUnknownOpcode(t *testing.T) {
	adapter := ModelAdapter{Profile: fixtureProfile(t)}
	if _, err := adapter.Adapt("DO_ANYTHING", Operand{}); err == nil {
		t.Fatalf("expected unknown opcode to be rejected")
	}
}

func TestModelAdapter_RejectsOpcodeWithNoProfileEntry(t *testing.T) {
	adapter := ModelAdapter{Profile: fixtureProfile(t)}
	// COMPARE_NUMBERS is a valid R0 opcode but the fixture profile has no
	// entry for it (it is intended for the deterministic Numeric Tlaloque,
	// never Parrot).
	_, err := adapter.Adapt(OpCompareNumbers, Operand{})
	if err == nil {
		t.Fatalf("expected contract violation for an opcode absent from the profile")
	}
}
