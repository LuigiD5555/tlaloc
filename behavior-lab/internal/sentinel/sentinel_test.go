package sentinel

import (
	"context"
	"strings"
	"testing"

	"tlaloc.local/behaviorlab/internal/action"
	"tlaloc.local/behaviorlab/internal/tlaloque/calibration"
)

func TestPermissionSentinel_IndependentRecheck(t *testing.T) {
	subject := Subject{
		ProposedAction: &action.ActionIR{Capability: "SERVICE.RESTART", Risk: action.R4Privileged},
		Policy:         &action.Policy{MaxRisk: action.R2LocalIrreversible},
	}
	concerns, _ := PermissionSentinel{}.Inspect(context.Background(), subject)
	if len(concerns) != 1 || concerns[0].Severity != Block {
		t.Fatalf("an over-ceiling action must raise a BLOCK concern: %+v", concerns)
	}
}

func TestScopeSentinel_PathOutsideAllowed(t *testing.T) {
	subject := Subject{
		ProposedAction: &action.ActionIR{
			Capability: "FILE.DELETE",
			Arguments:  map[string]string{"path": "/home/u/.ssh/id_rsa"},
		},
		AllowedPaths: []string{"/home/u/Documents"},
	}
	concerns, _ := ScopeSentinel{}.Inspect(context.Background(), subject)
	if len(concerns) != 1 || concerns[0].Severity != Block || concerns[0].Kind != "path_out_of_scope" {
		t.Fatalf("expected a BLOCK path_out_of_scope concern: %+v", concerns)
	}

	// Inside is clean.
	subject.ProposedAction.Arguments["path"] = "/home/u/Documents/old.pdf"
	clean, _ := ScopeSentinel{}.Inspect(context.Background(), subject)
	if len(clean) != 0 {
		t.Errorf("an in-scope path must raise nothing: %+v", clean)
	}
}

func TestConflictSentinel(t *testing.T) {
	subject := Subject{Observations: []Observation{
		{Key: "doc_type", Value: "INVOICE", Source: "classifier-a"},
		{Key: "doc_type", Value: "CONTRACT", Source: "classifier-b"},
		{Key: "page", Value: "7", Source: "scout"},
	}}
	concerns, _ := ConflictSentinel{}.Inspect(context.Background(), subject)
	if len(concerns) != 1 || concerns[0].Kind != "observation_conflict" {
		t.Fatalf("expected one conflict concern for doc_type: %+v", concerns)
	}
}

func TestOODSentinel(t *testing.T) {
	// A profile whose confidence floor is unreachable (the questionclass
	// shape) -> every input is out of distribution.
	profile := calibration.CalibrationProfile{
		Schema:            calibration.Schema,
		ConfidenceFloor:   1.01,
		OutOfDistribution: calibration.EvalSlice{N: 300, Accuracy: 0.51},
	}
	subject := Subject{Calibration: &profile, AnswerConfidence: 0.99}
	concerns, _ := OODSentinel{}.Inspect(context.Background(), subject)
	if len(concerns) != 1 || concerns[0].Kind != "out_of_distribution" {
		t.Fatalf("expected an OOD concern: %+v", concerns)
	}
}

func TestNumericConsistencySentinel(t *testing.T) {
	subject := Subject{
		Answer:   "Swarm research surged around 2019 and again in 2024, with 500 agents tested.",
		Evidence: "The document discusses a 2019 milestone and a simulation of 500 agents.",
	}
	concerns, _ := NumericConsistencySentinel{}.Inspect(context.Background(), subject)
	if len(concerns) != 1 {
		t.Fatalf("expected one concern: %+v", concerns)
	}
	if !strings.Contains(concerns[0].Detail, "2024") {
		t.Errorf("2024 is unsupported and should be flagged: %q", concerns[0].Detail)
	}
}

func TestPanel_Review_BlocksAndAggregates(t *testing.T) {
	panel := DefaultPanel()
	subject := Subject{
		Answer:   "done in 2099",
		Evidence: "no such year here",
		ProposedAction: &action.ActionIR{
			Capability: "FILE.DELETE", Risk: action.R2LocalIrreversible,
			Arguments: map[string]string{"path": "/etc/passwd"},
		},
		Policy:       &action.Policy{MaxRisk: action.R1LocalReversible},
		AllowedPaths: []string{"/home/u/tmp"},
	}
	result := panel.Review(context.Background(), subject)
	if !result.Blocked {
		t.Fatal("this subject should be blocked (risk over ceiling + path out of scope)")
	}
	if len(result.Concerns) < 2 {
		t.Errorf("expected several concerns, got %d: %+v", len(result.Concerns), result.Concerns)
	}
	// The panel result maps into spine checks.
	checks := result.ToChecks()
	for _, check := range checks {
		if check.Passed {
			t.Errorf("a Warn/Block concern must map to a failed check: %+v", check)
		}
	}
}
