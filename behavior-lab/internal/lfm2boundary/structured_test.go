package lfm2boundary

import (
	"encoding/json"
	"fmt"
	"testing"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/temporalbench"
)

func TestSimulateObservedRulesUsesSynchronousSnapshots(t *testing.T) {
	cells := CellsObservation{Cells: []ObservedCell{{ID: "A", InitialState: "ACTIVE"}, {ID: "B", InitialState: "IDLE"}, {ID: "C", InitialState: "IDLE"}}}
	rules := TransitionsObservation{Rules: []ObservedRule{
		{Requires: []ObservedRequirement{{CellID: "A", State: "ACTIVE"}}, TargetCell: "B", FromState: "IDLE", ToState: "ACTIVE"},
		{Requires: []ObservedRequirement{{CellID: "B", State: "ACTIVE"}}, TargetCell: "A", FromState: "ACTIVE", ToState: "DONE"},
		{Requires: []ObservedRequirement{{CellID: "B", State: "ACTIVE"}}, TargetCell: "C", FromState: "IDLE", ToState: "ACTIVE"},
		{Requires: []ObservedRequirement{{CellID: "C", State: "ACTIVE"}}, TargetCell: "B", FromState: "ACTIVE", ToState: "DONE"},
	}}
	states, err := SimulateObservedRules(cells, rules)
	if err != nil {
		t.Fatal(err)
	}
	if states["A"] != "DONE" || states["B"] != "DONE" || states["C"] != "ACTIVE" {
		t.Fatalf("states=%v", states)
	}
}

func TestSimulateObservedRulesRejectsConflictingWrites(t *testing.T) {
	cells := CellsObservation{Cells: []ObservedCell{{ID: "A", InitialState: "ACTIVE"}, {ID: "B", InitialState: "IDLE"}}}
	rules := TransitionsObservation{Rules: []ObservedRule{{Requires: []ObservedRequirement{{CellID: "A", State: "ACTIVE"}}, TargetCell: "B", FromState: "IDLE", ToState: "ACTIVE"}, {Requires: []ObservedRequirement{{CellID: "A", State: "ACTIVE"}}, TargetCell: "B", FromState: "IDLE", ToState: "DONE"}}}
	if _, err := SimulateObservedRules(cells, rules); err == nil {
		t.Fatal("expected conflicting write error")
	}
}

func TestStructuredObservationKeepsMalformedEvidenceForUnknown(t *testing.T) {
	raw := StructuredObservation(RoleCells, "not json")
	if !json.Valid(raw) {
		t.Fatalf("not valid JSON: %s", raw)
	}
	if roleContract(RoleCells)(raw) {
		t.Fatal("malformed evidence unexpectedly satisfied contract")
	}
}

func TestSynthesizeBlackboardResponsesUsesTwoThirdsQuorumAndPassesTemporalBenchmark(t *testing.T) {
	root := blackboard.New(t.TempDir())
	run := "r"
	values := map[string]string{
		RoleRosetta:     `{"box":"CELL","arrow":"TRANSITION","ring":"CHECKPOINT","x_time":"TIME","semantic_film_not_video":true}`,
		RoleCells:       `{"cells":[{"id":"A","initial_state":"ACTIVE"},{"id":"B","initial_state":"IDLE"},{"id":"C","initial_state":"IDLE"}]}`,
		RoleTransitions: `{"rules":[{"requires":[{"cell_id":"A","state":"ACTIVE"}],"target_cell":"B","from_state":"IDLE","to_state":"ACTIVE"},{"requires":[{"cell_id":"B","state":"ACTIVE"}],"target_cell":"A","from_state":"ACTIVE","to_state":"DONE"},{"requires":[{"cell_id":"B","state":"ACTIVE"}],"target_cell":"C","from_state":"IDLE","to_state":"ACTIVE"},{"requires":[{"cell_id":"C","state":"ACTIVE"}],"target_cell":"B","from_state":"ACTIVE","to_state":"DONE"}]}`,
		RoleTimeline:    `{"checkpoints":["T0","T2","T4"]}`,
	}
	for role, value := range values {
		for i := 0; i < 3; i++ {
			v := value
			if i == 2 && role == RoleCells {
				v = `{"cells":[{"id":"A","initial_state":"IDLE"}]}`
			}
			_, _, err := root.Append(blackboard.Entry{Type: blackboard.EntryObservation, RunID: run, TaskID: "t", NodeID: fmt.Sprintf("%s-r%02d", role, i+1), WorkerID: fmt.Sprintf("w-%02d", i+1), Key: role, Value: json.RawMessage(v)})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	snap, err := root.Snapshot(run)
	if err != nil {
		t.Fatal(err)
	}
	out, err := SynthesizeBlackboardResponses(snap)
	if err != nil {
		t.Fatal(err)
	}
	responses := []temporalbench.Response{}
	for _, q := range temporalbench.CanonicalQuestions() {
		responses = append(responses, temporalbench.Response{QuestionID: q.ID, Text: out.Responses[q.ID]})
	}
	result := temporalbench.EvaluateTrial(temporalbench.Trial{ID: "x", ModelID: RequiredModel, Condition: "BLACKBOARD_CROPPED", Specimen: temporalbench.Specimen{ID: "fixture"}, Responses: responses})
	if result.OverallScore < 8.0/9.0 {
		t.Fatalf("score=%f responses=%v questions=%+v", result.OverallScore, out.Responses, result.Questions)
	}
	if result.TemporalReasoning <= 0 {
		t.Fatalf("temporal=%f", result.TemporalReasoning)
	}
	if result.InventedExactClaims != 0 {
		t.Fatalf("invented exact=%d", result.InventedExactClaims)
	}
	if out.Responses["Q7"] != "A DONE; B DONE; C ACTIVE" {
		t.Fatalf("Q7=%q", out.Responses["Q7"])
	}
}

func TestSynthesizeBlackboardOutOfContractReplicaForcesUnknown(t *testing.T) {
	store := blackboard.New(t.TempDir())
	for i, v := range []string{`{"cells":[{"id":"A","initial_state":"ACTIVE"}]}`, `{"cells":[{"id":"A","initial_state":"ACTIVE"}]}`, `"bad"`} {
		_, _, err := store.Append(blackboard.Entry{Type: blackboard.EntryObservation, RunID: "r", TaskID: "t", NodeID: string(rune('a' + i)), WorkerID: "w", Key: RoleCells, Value: json.RawMessage(v)})
		if err != nil {
			t.Fatal(err)
		}
	}
	snap, err := store.Snapshot("r")
	if err != nil {
		t.Fatal(err)
	}
	out, err := SynthesizeBlackboardResponses(snap)
	if err != nil {
		t.Fatal(err)
	}
	if out.Consensus[RoleCells] != blackboard.ConsensusUnknown || out.Responses["Q1"] != "UNKNOWN" {
		t.Fatalf("out=%+v", out)
	}
}
