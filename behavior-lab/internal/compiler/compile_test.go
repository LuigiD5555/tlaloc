package compiler

import (
	"strings"
	"testing"

	"tlaloc.local/behaviorlab/internal/spec"
)

func TestCompileUsesBehaviorSpecWithoutHardcodedProfileLaw(t *testing.T) {
	s := spec.BehaviorSpec{Version: "0.1", ID: "generic.example", Description: "generic behavior", Identity: "Follow this contract.", StateKinds: []spec.StateKind{"custom"}, Operations: []spec.Operation{"STEP"}, Rules: []spec.Rule{{Code: "CUSTOM_RULE", Description: "Preserve custom semantics.", Priority: 95}}, Invariants: []spec.Invariant{{Code: "CUSTOM_INVARIANT", Description: "must hold", Severity: 5}}, Output: spec.OutputSpec{Format: "json", Schema: "Return one JSON object."}}
	ir, err := BuildIR(s, "test")
	if err != nil { t.Fatal(err) }
	p := Render(ir)
	for _, want := range []string{"TLALOC BEHAVIOR COMPILED PROMPT", "generic.example", "CUSTOM_RULE", "CUSTOM_INVARIANT", "Return one JSON object."} { if !strings.Contains(p, want) { t.Fatalf("missing %q", want) } }
	if strings.Contains(p, "Only an explicit OBSERVE") { t.Fatal("compiler leaked Origami-specific authority law into a generic profile") }
}
