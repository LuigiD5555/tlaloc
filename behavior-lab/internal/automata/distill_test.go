package automata

import "testing"

func TestDistillRepeatedTransitions(t *testing.T) {
	trace := ActionTrace{Schema: TraceSchema, ID: "chain", Steps: []TraceStep{
		{Step: 0, Tlaloque: "A", FromState: "IDLE", ToState: "ACTIVE", EmitsTo: []string{"B"}},
		{Step: 1, Tlaloque: "B", FromState: "IDLE", ToState: "ACTIVE", Requires: []Predicate{{CellID: "A", State: "ACTIVE"}}, EmitsTo: []string{"C"}},
		{Step: 2, Tlaloque: "B", FromState: "IDLE", ToState: "ACTIVE", Requires: []Predicate{{CellID: "A", State: "ACTIVE"}}, EmitsTo: []string{"C"}},
		{Step: 3, Tlaloque: "C", FromState: "IDLE", ToState: "ACTIVE", Requires: []Predicate{{CellID: "B", State: "ACTIVE"}}},
	}}
	got, err := Distill(trace)
	if err != nil { t.Fatal(err) }
	if got.Automaton.Schema != AutomatonSchema { t.Fatalf("schema %s", got.Automaton.Schema) }
	if got.TemporalProgram.Schema != TemporalProgramSchema { t.Fatalf("temporal schema %s", got.TemporalProgram.Schema) }
	if got.TemporalProgram.Automaton.ID != got.Automaton.ID { t.Fatal("temporal program not bound to distilled automaton") }
	if got.TemporalProgram.MaxSteps != 4 { t.Fatalf("expected max_steps=4, got %d", got.TemporalProgram.MaxSteps) }
	if got.Metrics.TraceSteps != 4 || got.Metrics.TraceMaxStep != 3 || got.Metrics.UniqueRules != 3 || got.Metrics.RepeatedTransitionsRemoved != 1 {
		t.Fatalf("unexpected metrics: %#v", got.Metrics)
	}
	if len(got.Automaton.Cells) != 3 { t.Fatalf("expected 3 cells, got %d", len(got.Automaton.Cells)) }
	if len(got.Automaton.Edges) < 2 { t.Fatalf("expected graph edges, got %#v", got.Automaton.Edges) }
}

func TestDistillOrderInvariantForSameExplicitSteps(t *testing.T) {
	a := ActionTrace{Schema: TraceSchema, ID: "x", Steps: []TraceStep{
		{Step: 1, Tlaloque: "B", FromState: "IDLE", ToState: "ACTIVE"},
		{Step: 0, Tlaloque: "A", FromState: "IDLE", ToState: "ACTIVE"},
	}}
	b := ActionTrace{Schema: TraceSchema, ID: "x", Steps: []TraceStep{a.Steps[1], a.Steps[0]}}
	ra, err := Distill(a); if err != nil { t.Fatal(err) }
	rb, err := Distill(b); if err != nil { t.Fatal(err) }
	if ra.Automaton.SourceTraceSHA256 != rb.Automaton.SourceTraceSHA256 { t.Fatal("canonical trace digest drift") }
	if ra.Metrics.UniqueRules != rb.Metrics.UniqueRules { t.Fatal("rule count drift") }
	if ra.TemporalProgram.MaxSteps != rb.TemporalProgram.MaxSteps { t.Fatal("temporal horizon drift") }
}

func TestDistillRejectsImplicitStep(t *testing.T) {
	_, err := Distill(ActionTrace{Schema: TraceSchema, ID: "bad", Steps: []TraceStep{{Step: -1, Tlaloque: "A", FromState: "IDLE", ToState: "ACTIVE"}}})
	if err == nil { t.Fatal("expected invalid explicit step rejection") }
}
