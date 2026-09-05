package tonalt1arms

import "testing"

func loadRealArmBPolicy(t *testing.T) *ArmBPolicy {
	t.Helper()
	policy, err := LoadArmBPolicy(RepoPathHelper("experiments/tonal-t1/d4/T1_D5_ARM_B_POLICY.json"))
	if err != nil {
		t.Fatalf("LoadArmBPolicy: %v", err)
	}
	return policy
}

func loadRealArmCPolicy(t *testing.T) *ArmCPolicy {
	t.Helper()
	policy, err := LoadArmCPolicy(RepoPathHelper("experiments/tonal-t1/d4/T1_D5_ARM_C_POLICY.json"))
	if err != nil {
		t.Fatalf("LoadArmCPolicy: %v", err)
	}
	return policy
}

func TestBuildArmBBindings_GenerativeCapabilitiesUseParrot(t *testing.T) {
	bindings, err := BuildArmBBindings(loadRealArmBPolicy(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, capability := range []string{"EXTRACT_NUMBER", "NORMALIZE", "COMPARE_NUMBERS", "ARITHMETIC"} {
		b, ok := bindings[capability]
		if !ok {
			t.Fatalf("Arm B binding missing for %s", capability)
		}
		if !b.UsesParrot {
			t.Errorf("Arm B %s: UsesParrot = false, want true", capability)
		}
	}
	for _, capability := range []string{"LOCATE_REGION", "CROP_REGION", "VERIFY"} {
		b, ok := bindings[capability]
		if !ok {
			t.Fatalf("Arm B binding missing for %s", capability)
		}
		if b.UsesParrot {
			t.Errorf("Arm B %s: UsesParrot = true, want false (deterministic)", capability)
		}
	}
}

func TestBuildArmCBindings_OnlyExtractNumberUsesParrot(t *testing.T) {
	bindings, err := BuildArmCBindings(loadRealArmCPolicy(t))
	if err != nil {
		t.Fatal(err)
	}
	if !bindings["EXTRACT_NUMBER"].UsesParrot {
		t.Error("Arm C EXTRACT_NUMBER: UsesParrot = false, want true")
	}
	for _, capability := range []string{"NORMALIZE", "COMPARE_NUMBERS", "ARITHMETIC", "LOCATE_REGION", "CROP_REGION", "VERIFY"} {
		b, ok := bindings[capability]
		if !ok {
			t.Fatalf("Arm C binding missing for %s", capability)
		}
		if b.UsesParrot {
			t.Errorf("Arm C %s: UsesParrot = true, want false (deterministic)", capability)
		}
	}
}

// TestArmBArmCBindingsAreIndependentMaps proves mutating a returned binding
// map cannot affect the other arm's table -- not just relying on Go's map
// semantics, but actually exercising the mutation to prove there is no
// shared backing map.
func TestArmBArmCBindingsAreIndependentMaps(t *testing.T) {
	armBBindings, err := BuildArmBBindings(loadRealArmBPolicy(t))
	if err != nil {
		t.Fatal(err)
	}
	armCBindings, err := BuildArmCBindings(loadRealArmCPolicy(t))
	if err != nil {
		t.Fatal(err)
	}

	before := armCBindings["NORMALIZE"]
	// Mutate Arm B's copy of the NORMALIZE binding.
	armBBindings["NORMALIZE"] = Binding{Capability: "NORMALIZE", ExecutorID: "INJECTED", UsesParrot: true}

	after := armCBindings["NORMALIZE"]
	if after != before {
		t.Fatalf("Arm C's NORMALIZE binding changed after mutating Arm B's map: before=%+v after=%+v", before, after)
	}
	if armCBindings["NORMALIZE"].UsesParrot {
		t.Fatal("Arm C NORMALIZE became UsesParrot=true after injecting into Arm B's map -- isolation violated")
	}
}

// TestInjectingArmBAdapterCannotAlterArmC directly exercises the required
// ARM_B_C_ADAPTER_ISOLATION proof: constructing a new Binding and inserting
// it into Arm B's table must have zero effect on a separately-built Arm C
// table, even for the same capability key.
func TestInjectingArmBAdapterCannotAlterArmC(t *testing.T) {
	armBBindings, _ := BuildArmBBindings(loadRealArmBPolicy(t))
	armCBindings, _ := BuildArmCBindings(loadRealArmCPolicy(t))

	armCArithmeticBefore := armCBindings["ARITHMETIC"]
	armBBindings["ARITHMETIC"] = Binding{Capability: "ARITHMETIC", ExecutorID: "hostile-injection", UsesParrot: false}

	if armCBindings["ARITHMETIC"] != armCArithmeticBefore {
		t.Fatal("Arm C's ARITHMETIC binding was altered by an injection into Arm B's table")
	}
	if armCBindings["ARITHMETIC"].UsesParrot {
		t.Fatal("Arm C's ARITHMETIC.UsesParrot became true after injecting a hostile Arm-B binding")
	}
}
