package tlaloque

import (
	"context"
	"strings"
	"testing"

	"tlaloc.local/behaviorlab/internal/compiler"
	"tlaloc.local/behaviorlab/internal/spec"
)

type fakeModel struct{}

func (fakeModel) Complete(_ context.Context, systemPrompt, user string) (string, error) {
	if strings.Contains(systemPrompt, "repair:no-collapse") || strings.Contains(systemPrompt, "Never convert SUPERPOSED") {
		return `{"kind":"superposed","branches":[{"label":"D","real":0.7071067811865475,"imag":0},{"label":"E","real":0.7071067811865475,"imag":0}]}`, nil
	}
	return `{"kind":"observed","branches":[{"label":"D","real":1,"imag":0}],"observed":"D"}`, nil
}

func TestTrainerCanRepairPrematureCollapse(t *testing.T) {
	s := spec.BehaviorSpec{Version: "0.1", ID: "q", StateKinds: []spec.StateKind{spec.Superposed, spec.Observed}, Operations: []spec.Operation{spec.OpTransform, spec.OpObserve}, Invariants: []spec.Invariant{{Code: spec.NoImplicitObservation, Description: "no implicit observation", Severity: 5}}, Output: spec.OutputSpec{Format: "json"}}
	ir, err := compiler.BuildIR(s, "fake")
	if err != nil {
		t.Fatal(err)
	}
	var reduced []compiler.Section
	for _, sec := range ir.Sections {
		if sec.ID != "authority" {
			reduced = append(reduced, sec)
		}
	}
	ir.Sections = reduced
	cases := []Case{{ID: "preserve", User: "TRANSFORM A+B to D+E", ExpectedRaw: `{"kind":"superposed","branches":[{"label":"D","real":0.7071067811865475,"imag":0},{"label":"E","real":0.7071067811865475,"imag":0}]}`}}
	h, _, err := (Trainer{Model: fakeModel{}, MaxGenerations: 3}).Train(context.Background(), ir, cases)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) < 2 || h[len(h)-1].Failed != 0 {
		t.Fatalf("expected repaired later generation, history=%#v", h)
	}
}
