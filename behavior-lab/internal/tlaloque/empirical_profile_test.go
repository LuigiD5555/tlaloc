package tlaloque

import "testing"

func TestCapabilityDescriptorNormalizesEmpiricalProfile(t *testing.T) {
	desc, err := (CapabilityDescriptor{
		ID:           "lfm2-vl",
		Capability:   "PERCEIVE_VISUAL",
		Scope:        ScopeGeneral,
		Engine:       EngineModel,
		InputSchema:  "image",
		OutputSchema: "claims",
		EmpiricalProfile: &EmpiricalProfileRef{
			ModelID:   "  LFM2-VL  ",
			Condition: " protocol-C ",
		},
	}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if desc.EmpiricalProfile == nil {
		t.Fatal("empirical profile missing")
	}
	if desc.EmpiricalProfile.ModelID != "LFM2-VL" || desc.EmpiricalProfile.Condition != "protocol-C" {
		t.Fatalf("profile=%+v", desc.EmpiricalProfile)
	}
	if got := desc.EmpiricalProfile.Key(); got != "LFM2-VL\x00protocol-C" {
		t.Fatalf("key=%q", got)
	}
}

func TestCapabilityDescriptorRejectsEmpiricalProfileWithoutModel(t *testing.T) {
	_, err := (CapabilityDescriptor{
		ID:               "bad",
		Capability:       "CLASSIFY",
		Scope:            ScopeGeneral,
		Engine:           EngineModel,
		InputSchema:      "text",
		OutputSchema:     "class",
		EmpiricalProfile: &EmpiricalProfileRef{Condition: "protocol-A"},
	}).Normalize()
	if err == nil {
		t.Fatal("expected empirical profile without model_id to fail")
	}
}

func TestCapabilityDescriptorWithoutEmpiricalProfileRemainsCompatible(t *testing.T) {
	desc, err := (CapabilityDescriptor{
		ID:           "deterministic-parser",
		Capability:   "PARSE",
		Scope:        ScopeGeneral,
		Engine:       EngineDeterministic,
		InputSchema:  "text",
		OutputSchema: "json",
	}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if desc.EmpiricalProfile != nil {
		t.Fatalf("unexpected profile=%+v", desc.EmpiricalProfile)
	}
}
