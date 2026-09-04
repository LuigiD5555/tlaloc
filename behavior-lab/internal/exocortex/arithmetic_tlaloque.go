package exocortex

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// ArithmeticTlaloqueID is the CapabilityDescriptor.ID this Tlaloque
// registers under. Like NumericTlaloque it is purely deterministic Go: if
// the operation can be computed exactly it must never be sent to a model
// (P1's conclusion about arithmetic). It exists because the Micro-ISA R0
// vocabulary carries comparison but no A-B / A/B / percentage-difference
// operation, which the deeper T1 workflow families require.
const ArithmeticTlaloqueID = "arithmetic-tlaloque"

// Arithmetic operations. The set is closed and explicit — there is no eval
// string and no expression parser.
const (
	ArithSubtract          = "SUBTRACT"           // a - b
	ArithRatio             = "RATIO"              // a / b
	ArithPercentDifference = "PERCENT_DIFFERENCE" // 100 * (a - b) / b
)

// ArithmeticInput is the typed input contract. Operands are carried as
// strings so this Tlaloque can sit directly downstream of a Normalize
// Tlaloque without re-deriving parsing rules.
type ArithmeticInput struct {
	Operation string `json:"operation"`
	A         string `json:"a"`
	B         string `json:"b"`
}

// ArithmeticStatus values for the output contract.
const (
	ArithStatusOK           = "OK"
	ArithStatusInvalidInput = "INVALID_INPUT"
)

// ArithmeticOutput is the deterministic typed result. When Status is
// INVALID_INPUT (division by zero, undefined percentage base) HasResult is
// false and Result is left at its zero value; the workflow can branch on
// that rather than crashing.
type ArithmeticOutput struct {
	Operation string  `json:"operation"`
	A         float64 `json:"a"`
	B         float64 `json:"b"`
	Result    float64 `json:"result"`
	HasResult bool    `json:"has_result"`
	Status    string  `json:"status"`
	Detail    string  `json:"detail,omitempty"`
}

// ArithmeticTlaloque implements the deterministic ARITHMETIC opcode. It
// never calls a model.
type ArithmeticTlaloque struct{}

func (ArithmeticTlaloque) Descriptor() tlaloque.CapabilityDescriptor {
	d, _ := tlaloque.CapabilityDescriptor{
		ID: ArithmeticTlaloqueID, Capability: OpArithmetic, Engine: tlaloque.EngineDeterministic,
		InputSchema: "exocortex.arithmetic-input.r0", OutputSchema: "exocortex.arithmetic-output.r0",
		Deterministic: true, MaxConcurrency: 8,
	}.Normalize()
	return d
}

func (ArithmeticTlaloque) Execute(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in ArithmeticInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("arithmetic tlaloque: decode input: %w", err)
	}
	operation := strings.ToUpper(strings.TrimSpace(in.Operation))
	a, err := parseNumber(in.A)
	if err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("arithmetic tlaloque: operand a: %w", err)
	}
	b, err := parseNumber(in.B)
	if err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("arithmetic tlaloque: operand b: %w", err)
	}

	out := ArithmeticOutput{Operation: operation, A: a, B: b, Status: ArithStatusOK}
	switch operation {
	case ArithSubtract:
		out.Result, out.HasResult = a-b, true
	case ArithRatio:
		if b == 0 {
			out.Status, out.Detail = ArithStatusInvalidInput, "division by zero"
		} else {
			out.Result, out.HasResult = a/b, true
		}
	case ArithPercentDifference:
		if b == 0 {
			out.Status, out.Detail = ArithStatusInvalidInput, "percentage base b is zero"
		} else {
			out.Result, out.HasResult = 100*(a-b)/b, true
		}
	default:
		return tlaloque.CapabilityResponse{}, fmt.Errorf("arithmetic tlaloque: unsupported operation %q", in.Operation)
	}
	if out.HasResult && (math.IsNaN(out.Result) || math.IsInf(out.Result, 0)) {
		out.Result, out.HasResult = 0, false
		out.Status, out.Detail = ArithStatusInvalidInput, "result is not a finite number"
	}

	body, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	confidence := 1.0
	if !out.HasResult {
		confidence = 0
	}
	return tlaloque.CapabilityResponse{
		WorkerID: ArithmeticTlaloqueID, Output: body, Confidence: confidence,
		Observations: []blackboard.Observation{{
			Key: req.NodeID, Value: body, Confidence: confidence,
			Provenance: map[string]string{"source": ArithmeticTlaloqueID, "operation": operation, "status": out.Status},
		}},
	}, nil
}
