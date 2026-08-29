package evaluate

import (
	"testing"
	"tlaloc.local/behaviorlab/internal/reference"
	"tlaloc.local/behaviorlab/internal/spec"
)

func expectedState() reference.State {
	return reference.State{Kind: spec.Determinate, Branches: []reference.Branch{{Label: "A", Real: 1}}, Members: []string{}, Observed: "", Unknown: false, Semantic: "PRESENT", Notes: []string{}}
}
func TestCompareStrictState(t *testing.T) {
	raw := `{"kind":"determinate","branches":[{"label":"A","real":1,"imag":0}],"members":[],"observed":"","unknown":false,"semantic":"PRESENT","notes":[]}`
	if result := Compare(expectedState(), raw); !result.Pass {
		t.Fatalf("unexpected findings: %#v", result.Findings)
	}
}
func TestParseRejectsMissingAndUnknownFields(t *testing.T) {
	for _, raw := range []string{`{"kind":"determinate"}`, `{"kind":"determinate","branches":[],"members":[],"observed":"","unknown":false,"semantic":"PRESENT","notes":[],"extra":true}`} {
		if _, err := Parse(raw); err == nil {
			t.Fatalf("expected error for %s", raw)
		}
	}
}
func TestCompareChecksMembersAndObserved(t *testing.T) {
	expected := expectedState()
	expected.Members = []string{"A"}
	expected.Observed = "A"
	raw := `{"kind":"determinate","branches":[{"label":"A","real":1,"imag":0}],"members":[],"observed":"","unknown":false,"semantic":"PRESENT","notes":[]}`
	if result := Compare(expected, raw); result.Pass {
		t.Fatal("expected strict mismatch")
	}
}
