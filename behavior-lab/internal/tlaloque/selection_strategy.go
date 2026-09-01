package tlaloque

import (
	"fmt"
	"sort"
	"strings"
)

// SelectionCandidate separates worker eligibility from worker ranking. This is
// the seam used by future empirical/complementarity strategies without growing
// Registry.Select into a large conditional block.
type SelectionCandidate struct {
	Worker CapabilityWorker
	Desc   CapabilityDescriptor
}

// CandidateSpecification answers only whether a worker is eligible. Ranking is
// intentionally delegated to a SelectionStrategy.
type CandidateSpecification interface {
	Matches(CapabilityDescriptor, SelectionRequest) bool
}

type candidateSpecificationFunc func(CapabilityDescriptor, SelectionRequest) bool

func (f candidateSpecificationFunc) Matches(d CapabilityDescriptor, req SelectionRequest) bool {
	return f(d, req)
}

// ScoringStrategy gives an eligible candidate a deterministic score.
type ScoringStrategy interface {
	Score(CapabilityDescriptor, SelectionRequest) int
}

// SelectionStrategy selects one worker from already eligible candidates.
type SelectionStrategy interface {
	Select([]SelectionCandidate, SelectionRequest) Result[CapabilityWorker]
}

type DefaultScoringStrategy struct{}

func (DefaultScoringStrategy) Score(d CapabilityDescriptor, req SelectionRequest) int {
	domainHint := strings.ToUpper(strings.TrimSpace(req.DomainHint))
	rules := []struct {
		points int
		when   func() bool
	}{
		{points: 100, when: func() bool { return req.PreferDeterministic && d.Deterministic }},
		{points: 50, when: func() bool { return domainHint != "" && d.Scope == ScopeSpecific && d.Domain == domainHint }},
		{points: 25, when: func() bool { return d.Scope == ScopeGeneral }},
		{points: 10, when: func() bool { return d.ParameterCount == 0 }},
	}

	score := 0
	for _, rule := range rules {
		if rule.when() {
			score += rule.points
		}
	}
	return score
}

type RankedSelectionStrategy struct {
	Scoring ScoringStrategy
}

func (s RankedSelectionStrategy) Select(candidates []SelectionCandidate, req SelectionRequest) Result[CapabilityWorker] {
	if len(candidates) == 0 {
		return DomainResult[CapabilityWorker](ResultNoCandidate, nil, Diagnostic{
			Code:    "NO_ELIGIBLE_WORKER",
			Message: fmt.Sprintf("no worker satisfies capability=%s scope=%s domain=%s", strings.ToUpper(strings.TrimSpace(req.Capability)), strings.ToUpper(strings.TrimSpace(req.ScopeHint)), strings.ToUpper(strings.TrimSpace(req.DomainHint))),
		})
	}
	if s.Scoring == nil {
		s.Scoring = DefaultScoringStrategy{}
	}

	type ranked struct {
		candidate SelectionCandidate
		score     int
	}
	rows := make([]ranked, 0, len(candidates))
	for _, candidate := range candidates {
		rows = append(rows, ranked{candidate: candidate, score: s.Scoring.Score(candidate.Desc, req)})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].score != rows[j].score {
			return rows[i].score > rows[j].score
		}
		pi, pj := rows[i].candidate.Desc.ParameterCount, rows[j].candidate.Desc.ParameterCount
		if pi != pj {
			if pi == 0 {
				return true
			}
			if pj == 0 {
				return false
			}
			return pi < pj
		}
		return rows[i].candidate.Desc.ID < rows[j].candidate.Desc.ID
	})
	return Success(rows[0].candidate.Worker)
}

func defaultCandidateSpecifications() []CandidateSpecification {
	return []CandidateSpecification{
		candidateSpecificationFunc(func(d CapabilityDescriptor, req SelectionRequest) bool {
			return d.Capability == strings.ToUpper(strings.TrimSpace(req.Capability))
		}),
		candidateSpecificationFunc(func(d CapabilityDescriptor, req SelectionRequest) bool {
			return req.MaxParameters <= 0 || d.ParameterCount <= req.MaxParameters
		}),
		candidateSpecificationFunc(func(d CapabilityDescriptor, req SelectionRequest) bool {
			domainHint := strings.ToUpper(strings.TrimSpace(req.DomainHint))
			return domainHint != "" || d.Scope != ScopeSpecific
		}),
		candidateSpecificationFunc(func(d CapabilityDescriptor, req SelectionRequest) bool {
			scopeHint := strings.ToUpper(strings.TrimSpace(req.ScopeHint))
			switch scopeHint {
			case "":
				return true
			case ScopeGeneral:
				return d.Scope == ScopeGeneral
			case ScopeSpecific:
				return d.Scope == ScopeSpecific && d.Domain == strings.ToUpper(strings.TrimSpace(req.DomainHint))
			default:
				return false
			}
		}),
		candidateSpecificationFunc(func(d CapabilityDescriptor, req SelectionRequest) bool {
			domainHint := strings.ToUpper(strings.TrimSpace(req.DomainHint))
			return domainHint == "" || d.Scope != ScopeSpecific || d.Domain == domainHint
		}),
	}
}

func matchesAll(specs []CandidateSpecification, d CapabilityDescriptor, req SelectionRequest) bool {
	for _, spec := range specs {
		if !spec.Matches(d, req) {
			return false
		}
	}
	return true
}
