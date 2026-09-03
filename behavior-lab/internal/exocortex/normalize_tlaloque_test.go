package exocortex

import (
	"context"
	"encoding/json"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func TestNormalizeTlaloque_ConvertsToNumber(t *testing.T) {
	worker := NormalizeTlaloque{}
	input, _ := json.Marshal(NormalizeInput{Raw: "  126 ", TargetType: TargetTypeNumber})
	resp, err := worker.Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "fashion_mnist_count", Input: input})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out NormalizeOutput
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if !out.IsNumber || out.AsNumber != 126 {
		t.Fatalf("got %+v, want as_number=126", out)
	}
	if out.Trimmed != "126" {
		t.Fatalf("trimmed = %q, want \"126\"", out.Trimmed)
	}
}

func TestNormalizeTlaloque_TextPassesThroughTrimmed(t *testing.T) {
	worker := NormalizeTlaloque{}
	input, _ := json.Marshal(NormalizeInput{Raw: "\"  Fashion   MNIST  \"", TargetType: TargetTypeText})
	resp, err := worker.Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "label", Input: input})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out NormalizeOutput
	json.Unmarshal(resp.Output, &out)
	if out.Trimmed != "Fashion MNIST" {
		t.Fatalf("trimmed = %q, want \"Fashion MNIST\"", out.Trimmed)
	}
}

func TestNormalizeTlaloque_NonNumericTargetTypeNumberIsNotNumber(t *testing.T) {
	worker := NormalizeTlaloque{}
	input, _ := json.Marshal(NormalizeInput{Raw: "not a number", TargetType: TargetTypeNumber})
	resp, err := worker.Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "x", Input: input})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out NormalizeOutput
	json.Unmarshal(resp.Output, &out)
	if out.IsNumber {
		t.Fatalf("expected is_number=false for non-numeric raw text")
	}
}
