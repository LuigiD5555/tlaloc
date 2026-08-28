package reference

import (
	"errors"
	"fmt"
	"math/rand"

	"tlaloc.local/behaviorlab/internal/spec"
)

type LinearMap map[string][]Branch

func Superpose(branches ...Branch) State {
	kind := spec.Superposed
	if len(canonical(branches)) == 1 {
		kind = spec.Determinate
	}
	return New(kind, branches...).Normalized()
}

func Transform(s State, m LinearMap) (State, error) {
	if s.Kind == spec.Observed {
		return State{}, errors.New("TRANSFORM on observed state requires explicit unfold/restate")
	}
	var out []Branch
	for _, in := range s.Branches {
		targets, ok := m[in.Label]
		if !ok {
			return State{}, fmt.Errorf("missing transform mapping for branch %q", in.Label)
		}
		for _, t := range targets {
			out = append(out, FromComplex(t.Label, in.Complex()*t.Complex()))
		}
	}
	kind := s.Kind
	if kind == spec.Determinate && len(canonical(out)) > 1 {
		kind = spec.Superposed
	}
	return New(kind, out...).Normalized(), nil
}

func Interfere(branches ...Branch) State {
	return Superpose(branches...)
}

func Constrain(s State, allowed map[string]bool) State {
	out := s
	out.Branches = nil
	for _, b := range s.Branches {
		if allowed[b.Label] {
			out.Branches = append(out.Branches, b)
		}
	}
	out.Branches = canonical(out.Branches)
	if len(out.Branches) == 1 && out.Kind != spec.Coupled {
		out.Kind = spec.Determinate
	}
	return out.Normalized()
}

func Couple(members []string, joint ...Branch) State {
	return State{Kind: spec.Coupled, Members: append([]string(nil), members...), Branches: canonical(joint)}.Normalized()
}

func Observe(s State, r *rand.Rand) (State, error) {
	if len(s.Branches) == 0 {
		return State{}, errors.New("cannot observe empty/cancelled state")
	}
	if r == nil {
		r = rand.New(rand.NewSource(1))
	}
	x := r.Float64()
	cum := 0.0
	pick := s.Branches[len(s.Branches)-1].Label
	for _, b := range s.Branches {
		cum += s.Probability(b.Label)
		if x <= cum {
			pick = b.Label
			break
		}
	}
	return State{Kind: spec.Observed, Branches: []Branch{{Label: pick, Real: 1}}, Observed: pick}, nil
}
