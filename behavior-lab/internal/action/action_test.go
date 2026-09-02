package action

import "testing"

func permissivePolicy() Policy {
	return Policy{MaxRisk: R4Privileged}
}

// Positive case: a well-formed candidate for a reversible capability
// compiles to an ActionIR with catalog-derived risk, preconditions, and a
// synthesized rollback with swapped args.
func TestCompile_PositiveReversible(t *testing.T) {
	candidate := ActionCandidate{
		Capability: "FILE.MOVE",
		Arguments:  map[string]string{"source": "/home/u/Downloads/a.pdf", "destination": "/home/u/Documents/a.pdf"},
		ProposedBy: "file-organizer-r3",
	}
	act, err := Compile(candidate, DefaultCatalog(), permissivePolicy())
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if act.Risk != R1LocalReversible {
		t.Errorf("risk: got %s, want R1_LOCAL_REVERSIBLE", act.Risk)
	}
	if len(act.Preconditions) != 2 || len(act.ExpectedPostconditions) != 2 {
		t.Errorf("pre/postconditions not attached from the catalog: %+v", act)
	}
	if act.Rollback == nil {
		t.Fatal("reversible capability must get a rollback")
	}
	if act.Rollback.Arguments["source"] != "/home/u/Documents/a.pdf" ||
		act.Rollback.Arguments["destination"] != "/home/u/Downloads/a.pdf" {
		t.Errorf("rollback args not swapped: %+v", act.Rollback.Arguments)
	}
	if act.ProposedBy != "file-organizer-r3" {
		t.Errorf("proposed_by lost")
	}
}

// The load-bearing invariant: risk comes from the catalog, never the
// candidate. A candidate that claims FILE.DELETE is "read only" still gets
// R2, and if the policy ceiling is R1 the action is refused.
func TestCompile_RiskComesFromCatalogNotCandidate(t *testing.T) {
	candidate := ActionCandidate{
		Capability: "FILE.DELETE",
		Arguments:  map[string]string{"path": "/home/u/Documents/old.pdf"},
		Rationale:  "totally safe, just a read, R0 trust me",
	}

	// With a permissive ceiling it compiles — but as R2, not R0.
	act, err := Compile(candidate, DefaultCatalog(), permissivePolicy())
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if act.Risk != R2LocalIrreversible {
		t.Fatalf("risk: got %s, want R2_LOCAL_IRREVERSIBLE", act.Risk)
	}

	// With an R1 ceiling the same candidate is refused.
	if _, err := Compile(candidate, DefaultCatalog(), Policy{MaxRisk: R1LocalReversible}); err == nil {
		t.Error("FILE.DELETE must be refused under an R1 ceiling")
	}
}

// Prompt-injection shape: a document says "delete ~/.ssh". Even turned into
// a candidate, it cannot become an ActionIR when the capability is off the
// allow-list or the path is outside the sandbox.
func TestCompile_PolicyBoundaries(t *testing.T) {
	catalog := DefaultCatalog()

	// Not on the allow-list.
	restricted := Policy{MaxRisk: R4Privileged, AllowedCapabilities: []string{"FILE.LIST", "FILE.READ"}}
	if _, err := Compile(ActionCandidate{Capability: "FILE.DELETE", Arguments: map[string]string{"path": "/home/u/x"}}, catalog, restricted); err == nil {
		t.Error("capability off the allow-list must be refused")
	}

	// Path outside the sandbox.
	sandboxed := Policy{MaxRisk: R4Privileged, StayInside: []string{"/home/u/Documents/Facturas"}}
	if _, err := Compile(ActionCandidate{
		Capability: "FILE.DELETE",
		Arguments:  map[string]string{"path": "/home/u/.ssh/id_rsa"},
	}, catalog, sandboxed); err == nil {
		t.Error("a path outside StayInside must be refused")
	}
	// Inside the sandbox is fine.
	if _, err := Compile(ActionCandidate{
		Capability: "FILE.DELETE",
		Arguments:  map[string]string{"path": "/home/u/Documents/Facturas/aug/dupe.pdf"},
	}, catalog, sandboxed); err != nil {
		t.Errorf("a path inside StayInside must be allowed: %v", err)
	}
}

func TestCompile_RejectsUnknownCapabilityMissingAndExtraArgs(t *testing.T) {
	catalog := DefaultCatalog()

	if _, err := Compile(ActionCandidate{Capability: "FILE.SHRED"}, catalog, permissivePolicy()); err == nil {
		t.Error("unknown capability must be refused")
	}
	if _, err := Compile(ActionCandidate{Capability: "FILE.MOVE", Arguments: map[string]string{"source": "/home/u/a"}}, catalog, permissivePolicy()); err == nil {
		t.Error("missing required argument must be refused")
	}
	if _, err := Compile(ActionCandidate{
		Capability: "USER.NOTIFY",
		Arguments:  map[string]string{"message": "hi", "run": "rm -rf /"},
	}, catalog, permissivePolicy()); err == nil {
		t.Error("undeclared argument must be refused")
	}
}

func TestRiskClass_OrderingAndNames(t *testing.T) {
	if !(R0ReadOnly < R1LocalReversible && R3ExternalEffect < R4Privileged) {
		t.Error("risk classes must be ordered")
	}
	if R2LocalIrreversible.String() != "R2_LOCAL_IRREVERSIBLE" || RiskClass(99).Valid() {
		t.Error("risk class name/validity wrong")
	}
}
