package envelope

import (
	"testing"

	"tlaloc.local/behaviorlab/internal/action"
	"tlaloc.local/behaviorlab/internal/intent"
)

func TestPolicyFor_Ceilings(t *testing.T) {
	// Explicit constraint wins.
	explicit, err := PolicyFor(intent.CompiledIntent{MaxActionRisk: "R1_LOCAL_REVERSIBLE", Risk: intent.RiskProfile{Level: "low"}})
	if err != nil || explicit.MaxRisk != action.R1LocalReversible {
		t.Fatalf("explicit max_action_risk should win: %v %v", explicit.MaxRisk, err)
	}

	// Falls back to the task risk level.
	byLevel, _ := PolicyFor(intent.CompiledIntent{Risk: intent.RiskProfile{Level: "high"}})
	if byLevel.MaxRisk != action.R1LocalReversible {
		t.Errorf("high task risk -> R1 ceiling, got %s", byLevel.MaxRisk)
	}

	// Silence about effects -> no authority to cause them.
	silent, _ := PolicyFor(intent.CompiledIntent{})
	if silent.MaxRisk != action.R0ReadOnly {
		t.Errorf("an intent that says nothing about effects must default to R0, got %s", silent.MaxRisk)
	}

	if _, err := PolicyFor(intent.CompiledIntent{MaxActionRisk: "R9_NONSENSE"}); err == nil {
		t.Error("an unknown max_action_risk must error")
	}
}

// The full second boundary crossing: a compiled intent + a model's action
// candidate -> an authorized (or refused) ActionIR.
func TestAuthorize_EndToEnd(t *testing.T) {
	catalog := action.DefaultCatalog()

	compiled, err := intent.Compile(intent.IntentIR{
		Version:         "1",
		RequiredOutputs: []string{"ORGANIZE_DOCUMENTS"},
		Constraints: []intent.Constraint{
			{Kind: "stay_inside", Value: "/home/u/Documents/Facturas"},
			{Kind: "max_action_risk", Value: "R2_LOCAL_IRREVERSIBLE"},
		},
		Risk: intent.RiskProfile{Level: "medium"},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	// A move inside the sandbox is authorized, with a rollback.
	move, err := Authorize(compiled, catalog, action.ActionCandidate{
		Capability: "FILE.MOVE",
		Arguments: map[string]string{
			"source":      "/home/u/Documents/Facturas/inbox/acme-08.pdf",
			"destination": "/home/u/Documents/Facturas/acme/acme-08.pdf",
		},
		ProposedBy: "invoice-organizer",
	})
	if err != nil {
		t.Fatalf("expected the in-sandbox move to be authorized: %v", err)
	}
	if move.Rollback == nil {
		t.Error("FILE.MOVE authorization must carry a rollback")
	}

	// A delete outside the sandbox is refused even though R2 <= ceiling.
	if _, err := Authorize(compiled, catalog, action.ActionCandidate{
		Capability: "FILE.DELETE",
		Arguments:  map[string]string{"path": "/home/u/.ssh/id_rsa"},
	}); err == nil {
		t.Error("a delete outside stay_inside must be refused")
	}

	// SERVICE.RESTART is R4, above the R2 ceiling -> refused.
	if _, err := Authorize(compiled, catalog, action.ActionCandidate{
		Capability: "SERVICE.RESTART", Arguments: map[string]string{"unit": "nginx"},
	}); err == nil {
		t.Error("an R4 action must be refused under an R2 ceiling")
	}
}
