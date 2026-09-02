package evaluate

import (
	"encoding/json"
	"fmt"
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

func Parse(raw string) (reference.State, error) {
	var s reference.State
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return s, fmt.Errorf("structured output: %w", err)
	}
	return s, nil
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
	if expected.Kind == spec.Coupled && len(actual.Members) != len(expected.Members) {
		findings = append(findings, Finding{Code: spec.CoupledIsJointState, Message: "coupled members were decomposed or lost"})
	}
	if len(expected.Branches) == 0 && actual.Unknown {
		findings = append(findings, Finding{Code: spec.ZeroAmplitudeCancellation, Message: "cancelled amplitude was reported as unknown"})
	}
	if !sameBranches(expected.Branches, actual.Branches) {
		findings = append(findings, Finding{Code: spec.TransformPreservesBranches, Message: "branch amplitudes differ from reference semantics"})
	}
	score := 1.0
	if len(findings) > 0 {
		score = math.Max(0, 1-float64(len(findings))*.25)
	}
	return Result{Pass: len(findings) == 0, Score: score, Findings: findings}
}

func sameBranches(a, b []reference.Branch) bool {
	if len(a) != len(b) {
		return false
	}
	ac := append([]reference.Branch(nil), a...)
	bc := append([]reference.Branch(nil), b...)
	sort.Slice(ac, func(i, j int) bool { return ac[i].Label < ac[j].Label })
	sort.Slice(bc, func(i, j int) bool { return bc[i].Label < bc[j].Label })
	for i := range ac {
		if ac[i].Label != bc[i].Label || math.Abs(ac[i].Real-bc[i].Real) > 1e-6 || math.Abs(ac[i].Imag-bc[i].Imag) > 1e-6 {
			return false
		}
	}
	return true
}
