package exocortex

import (
	"context"
	"encoding/json"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func TestNumericTlaloque_ComparesDeterministically(t *testing.T) {
	worker := NumericTlaloque{}
	input, _ := json.Marshal(NumericInput{A: "3", B: "5"})
	resp, err := worker.Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "cmp1", Input: input})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out NumericOutput
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if out.Comparison != "LESS" || out.Same {
		t.Fatalf("got %+v, want LESS/not-same", out)
	}
	if len(resp.Observations) != 1 || resp.Observations[0].Key != "cmp1" {
		t.Fatalf("expected one observation keyed by node id, got %+v", resp.Observations)
	}
}

func TestNumericTlaloque_RejectsNonNumericInput(t *testing.T) {
	worker := NumericTlaloque{}
	input, _ := json.Marshal(NumericInput{A: "not-a-number", B: "5"})
	if _, err := worker.Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "cmp2", Input: input}); err == nil {
		t.Fatalf("expected an error for non-numeric input")
	}
}

func TestNumericTlaloque_Descriptor_IsDeterministic(t *testing.T) {
	d := NumericTlaloque{}.Descriptor()
	if !d.Deterministic {
		t.Fatalf("numeric tlaloque descriptor must be deterministic")
	}
	if d.Engine != tlaloque.EngineDeterministic {
		t.Fatalf("engine = %q, want DETERMINISTIC", d.Engine)
	}
}
