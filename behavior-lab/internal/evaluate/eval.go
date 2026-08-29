package evaluate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"

	"tlaloc.local/behaviorlab/internal/reference"
	"tlaloc.local/behaviorlab/internal/spec"
)

type Finding struct {
	Code    spec.InvariantCode `json:"code"`
	Message string             `json:"message"`
}
type Result struct {
	Pass     bool      `json:"pass"`
	Score    float64   `json:"score"`
	Findings []Finding `json:"findings"`
}

type strictState struct {
	Kind     *spec.StateKind     `json:"kind"`
	Branches *[]reference.Branch `json:"branches"`
	Members  *[]string           `json:"members"`
	Observed *string             `json:"observed"`
	Unknown  *bool               `json:"unknown"`
	Semantic *string             `json:"semantic"`
	Notes    *[]string           `json:"notes"`
}

func Parse(raw string) (reference.State, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	var wire strictState
	if err := decoder.Decode(&wire); err != nil {
		return reference.State{}, fmt.Errorf("structured output: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return reference.State{}, err
	}
	if wire.Kind == nil || wire.Branches == nil || wire.Members == nil || wire.Observed == nil || wire.Unknown == nil || wire.Semantic == nil || wire.Notes == nil {
		return reference.State{}, fmt.Errorf("structured output: all state fields are required")
	}
	if *wire.Semantic != "PRESENT" && *wire.Semantic != "CANCELLED" {
		return reference.State{}, fmt.Errorf("structured output: unsupported semantic %q", *wire.Semantic)
	}
	return reference.State{Kind: *wire.Kind, Branches: *wire.Branches, Members: *wire.Members, Observed: *wire.Observed, Unknown: *wire.Unknown, Semantic: *wire.Semantic, Notes: *wire.Notes}, nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return fmt.Errorf("structured output: multiple JSON values")
	}
	return fmt.Errorf("structured output: %w", err)
}

func Compare(expected reference.State, raw string) Result {
	actual, err := Parse(raw)
	if err != nil {
		return Result{Pass: false, Findings: []Finding{{Code: spec.StructuredOutputRequired, Message: err.Error()}}}
	}
	var findings []Finding
	if expected.Kind != actual.Kind {
		code := spec.NoImplicitObservation
		if expected.Kind == spec.Coupled {
			code = spec.CoupledIsJointState
		}
		findings = append(findings, Finding{Code: code, Message: fmt.Sprintf("kind expected %s got %s", expected.Kind, actual.Kind)})
	}
	if expected.Kind == spec.Superposed && len(actual.Branches) < len(expected.Branches) {
		findings = append(findings, Finding{Code: spec.TransformPreservesBranches, Message: "valid branches were lost"})
	}
	if !equalStrings(expected.Members, actual.Members) {
		findings = append(findings, Finding{Code: spec.CoupledIsJointState, Message: "coupled members differ from reference semantics"})
	}
	if expected.Observed != actual.Observed {
		findings = append(findings, Finding{Code: spec.ObserveHasAuthority, Message: "observed value differs from reference semantics"})
	}
	if expected.Unknown != actual.Unknown {
		findings = append(findings, Finding{Code: spec.AbsentIsNotUnknown, Message: "unknown status differs from reference semantics"})
	}
	if expected.Semantic != actual.Semantic {
		findings = append(findings, Finding{Code: spec.ZeroAmplitudeCancellation, Message: "semantic value differs from reference semantics"})
	}
	if !equalStrings(expected.Notes, actual.Notes) {
		findings = append(findings, Finding{Code: spec.StructuredOutputRequired, Message: "notes differ from reference evidence"})
	}
	if !sameBranches(expected.Branches, actual.Branches) {
		findings = append(findings, Finding{Code: spec.TransformPreservesBranches, Message: "branch amplitudes differ from reference semantics"})
	}
	score := 1.0
	if len(findings) > 0 {
		score = math.Max(0, 1-float64(len(findings))*.2)
	}
	return Result{Pass: len(findings) == 0, Score: score, Findings: findings}
}

func equalStrings(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}
	return true
}

func sameBranches(first, second []reference.Branch) bool {
	if len(first) != len(second) {
		return false
	}
	left := append([]reference.Branch(nil), first...)
	right := append([]reference.Branch(nil), second...)
	sort.Slice(left, func(firstIndex, secondIndex int) bool { return left[firstIndex].Label < left[secondIndex].Label })
	sort.Slice(right, func(firstIndex, secondIndex int) bool { return right[firstIndex].Label < right[secondIndex].Label })
	for index := range left {
		if left[index].Label != right[index].Label || math.Abs(left[index].Real-right[index].Real) > 1e-6 || math.Abs(left[index].Imag-right[index].Imag) > 1e-6 {
			return false
		}
	}
	return true
}
