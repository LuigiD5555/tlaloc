package distill

import "testing"

func TestBuildHybridArtifactSetSeparatesPromptAndCarrierBehavior(t *testing.T) {
	candidate, err := Distill("bootstrap from BOOT and ROSETTA", []SwarmStep{
		{Agent: "a", State: "S0", Token: "BOOT_OK", Action: "advance", NextState: "S1", Emit: "ROSETTA"},
		{Agent: "b", State: "S1", Token: "ROSETTA_OK", Action: "advance", NextState: "S2", Emit: "INDEX"},
	})
	if err != nil {
		t.Fatal(err)
	}
	set, err := BuildHybridArtifactSet(candidate, 4000)
	if err != nil {
		t.Fatal(err)
	}
	if set.Schema != HybridArtifactSchema {
		t.Fatalf("unexpected schema %q", set.Schema)
	}
	if set.UniversalPrompt != candidate.Prompt {
		t.Fatal("universal prompt must remain a separate external artifact")
	}
	if len(set.MicroProgram) != 2 {
		t.Fatalf("expected two distilled micro-rules, got %d", len(set.MicroProgram))
	}
	if set.WorkingWindow != 4000 {
		t.Fatalf("unexpected working window %d", set.WorkingWindow)
	}
}

func TestBuildHybridArtifactSetRejectsIncompleteCandidate(t *testing.T) {
	_, err := BuildHybridArtifactSet(Candidate{Schema: ContractID, ID: "x", SourceTraceSHA256: "hash"}, 4000)
	if err == nil {
		t.Fatal("expected incomplete candidate to be rejected")
	}
}
