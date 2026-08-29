package reference

import (
	"math"
	"math/cmplx"
	"sort"

	"tlaloc.local/behaviorlab/internal/spec"
)

type Branch struct {
	Label string  `json:"label"`
	Real  float64 `json:"real"`
	Imag  float64 `json:"imag"`
}

func (b Branch) Complex() complex128 { return complex(b.Real, b.Imag) }
func FromComplex(label string, z complex128) Branch {
	return Branch{Label: label, Real: real(z), Imag: imag(z)}
}

type State struct {
	Kind     spec.StateKind `json:"kind"`
	Branches []Branch       `json:"branches"`
	Members  []string       `json:"members,omitempty"`
	Observed string         `json:"observed,omitempty"`
	Unknown  bool           `json:"unknown,omitempty"`
	Semantic string         `json:"semantic"`
	Notes    []string       `json:"notes"`
}

func New(kind spec.StateKind, branches ...Branch) State {
	return State{Kind: kind, Branches: canonical(branches), Semantic: "PRESENT", Notes: []string{}}
}

func canonical(in []Branch) []Branch {
	acc := map[string]complex128{}
	for _, b := range in {
		acc[b.Label] += b.Complex()
	}
	labels := make([]string, 0, len(acc))
	for label, z := range acc {
		if cmplx.Abs(z) > 1e-12 {
			labels = append(labels, label)
		}
	}
	sort.Strings(labels)
	out := make([]Branch, 0, len(labels))
	for _, label := range labels {
		out = append(out, FromComplex(label, acc[label]))
	}
	return out
}

func (s State) Probability(label string) float64 {
	var total, hit float64
	for _, b := range s.Branches {
		p := cmplx.Abs(b.Complex())
		p *= p
		total += p
		if b.Label == label {
			hit += p
		}
	}
	if total == 0 {
		return 0
	}
	return hit / total
}

func (s State) Normalized() State {
	var sum float64
	for _, b := range s.Branches {
		a := cmplx.Abs(b.Complex())
		sum += a * a
	}
	if sum == 0 {
		return s
	}
	norm := math.Sqrt(sum)
	out := s
	out.Branches = make([]Branch, 0, len(s.Branches))
	for _, b := range s.Branches {
		out.Branches = append(out.Branches, FromComplex(b.Label, b.Complex()/complex(norm, 0)))
	}
	return out
}
