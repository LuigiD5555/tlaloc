package groundingautomaton

import (
	"context"
	"encoding/json"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func TestWorkerDescriptorIsDeterministic(t *testing.T) {
	d, err := Worker{}.Descriptor().Normalize()
	if err != nil {
		t.Fatalf("normalize descriptor: %v", err)
	}
	if d.Capability != Capability || d.Engine != tlaloque.EngineDeterministic || !d.Deterministic {
		t.Fatalf("unexpected descriptor: %+v", d)
	}
}

func TestWorkerEmitsTrace(t *testing.T) {
	input, err := json.Marshal(VerifyInput{ModelAnswer: "The model has 28 million parameters.", PageContent: "The model has 27 million parameters."})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := Worker{}.Execute(context.Background(), tlaloque.CapabilityRequest{Input: input})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	var out VerifyOutput
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if out.Verdict != VerdictContradicted || len(out.Claims) != 1 || len(out.Claims[0].Reasons) == 0 {
		t.Fatalf("expected contradiction trace, got %+v", out)
	}
}
