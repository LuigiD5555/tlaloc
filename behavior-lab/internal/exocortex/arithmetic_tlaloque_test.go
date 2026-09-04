package exocortex

import (
	"context"
	"encoding/json"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func runArithmetic(t *testing.T, in ArithmeticInput) ArithmeticOutput {
	t.Helper()
	input, _ := json.Marshal(in)
	resp, err := ArithmeticTlaloque{}.Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "op", Input: input})
	if err != nil {
		t.Fatalf("Execute(%+v): %v", in, err)
	}
	var out ArithmeticOutput
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if len(resp.Observations) != 1 || resp.Observations[0].Key != "op" {
		t.Fatalf("expected one observation keyed by node id, got %+v", resp.Observations)
	}
	return out
}

func TestArithmeticTlaloque_Operations(t *testing.T) {
	cases := []struct {
		in   ArithmeticInput
		want float64
	}{
		{ArithmeticInput{Operation: "subtract", A: "10", B: "3"}, 7},
		{ArithmeticInput{Operation: ArithSubtract, A: "3", B: "10"}, -7},
		{ArithmeticInput{Operation: ArithRatio, A: "9", B: "3"}, 3},
		{ArithmeticInput{Operation: ArithPercentDifference, A: "110", B: "100"}, 10},
		{ArithmeticInput{Operation: ArithPercentDifference, A: "90", B: "100"}, -10},
	}
	for _, c := range cases {
		out := runArithmetic(t, c.in)
		if !out.HasResult || out.Status != ArithStatusOK {
			t.Fatalf("%+v: status=%s has_result=%v", c.in, out.Status, out.HasResult)
		}
		if out.Result != c.want {
			t.Fatalf("%+v: result=%v want %v", c.in, out.Result, c.want)
		}
	}
}

func TestArithmeticTlaloque_DivisionByZeroIsInvalidInputNotAPanic(t *testing.T) {
	for _, op := range []string{ArithRatio, ArithPercentDifference} {
		out := runArithmetic(t, ArithmeticInput{Operation: op, A: "5", B: "0"})
		if out.HasResult || out.Status != ArithStatusInvalidInput {
			t.Fatalf("%s by zero: got status=%s has_result=%v, want INVALID_INPUT/false", op, out.Status, out.HasResult)
		}
	}
}

func TestArithmeticTlaloque_RejectsNonNumericOperand(t *testing.T) {
	input, _ := json.Marshal(ArithmeticInput{Operation: ArithSubtract, A: "seven", B: "3"})
	if _, err := (ArithmeticTlaloque{}).Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "op", Input: input}); err == nil {
		t.Fatalf("expected an error for a non-numeric operand")
	}
}

func TestArithmeticTlaloque_RejectsUnknownOperation(t *testing.T) {
	input, _ := json.Marshal(ArithmeticInput{Operation: "MULTIPLY", A: "2", B: "3"})
	if _, err := (ArithmeticTlaloque{}).Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "op", Input: input}); err == nil {
		t.Fatalf("expected an error for an unsupported operation")
	}
}

func TestArithmeticTlaloque_DescriptorIsDeterministicARITHMETIC(t *testing.T) {
	d := ArithmeticTlaloque{}.Descriptor()
	if !d.Deterministic || d.Engine != tlaloque.EngineDeterministic {
		t.Fatalf("descriptor must be deterministic, got %+v", d)
	}
	if d.Capability != OpArithmetic {
		t.Fatalf("capability = %q, want ARITHMETIC", d.Capability)
	}
}

func TestOpcodeVocabularyIncludesArithmetic(t *testing.T) {
	got, err := NormalizeOpcode("arithmetic")
	if err != nil || got != OpArithmetic {
		t.Fatalf("NormalizeOpcode(arithmetic) = %q, %v", got, err)
	}
}
