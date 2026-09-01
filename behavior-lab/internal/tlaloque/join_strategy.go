package tlaloque

import (
	"fmt"
	"strings"
)

type JoinMode string

const (
	JoinAll    JoinMode = "ALL"
	JoinAny    JoinMode = "ANY"
	JoinQuorum JoinMode = "QUORUM"
)

type JoinProgress struct {
	Total     int
	Finished  int
	Succeeded int
	Required  int
}

type JoinDecision struct {
	Ready      bool
	Impossible bool
}

type JoinStrategy interface {
	Evaluate(JoinProgress) JoinDecision
}

type joinStrategyFunc func(JoinProgress) JoinDecision

func (f joinStrategyFunc) Evaluate(progress JoinProgress) JoinDecision {
	return f(progress)
}

// joinStrategies is deliberately a registry rather than a growing switch in
// SwarmRunner. New dependency semantics become isolated policies.
var joinStrategies = map[JoinMode]JoinStrategy{
	JoinAll: joinStrategyFunc(func(p JoinProgress) JoinDecision {
		if p.Total == 0 {
			return JoinDecision{Ready: true}
		}
		return JoinDecision{
			Ready:      p.Succeeded == p.Total,
			Impossible: p.Finished == p.Total && p.Succeeded < p.Total,
		}
	}),
	JoinAny: joinStrategyFunc(func(p JoinProgress) JoinDecision {
		return JoinDecision{
			Ready:      p.Succeeded >= 1,
			Impossible: p.Finished == p.Total && p.Succeeded == 0,
		}
	}),
	JoinQuorum: joinStrategyFunc(func(p JoinProgress) JoinDecision {
		required := p.Required
		if required <= 0 {
			required = p.Total/2 + 1
		}
		remaining := p.Total - p.Finished
		return JoinDecision{
			Ready:      p.Succeeded >= required,
			Impossible: p.Succeeded+remaining < required,
		}
	}),
}

func normalizeJoinMode(raw string) (JoinMode, error) {
	mode := JoinMode(strings.ToUpper(strings.TrimSpace(raw)))
	if mode == "" {
		mode = JoinAll
	}
	if _, ok := joinStrategies[mode]; !ok {
		return "", fmt.Errorf("unsupported join mode %q", raw)
	}
	return mode, nil
}

func evaluateJoin(node SwarmNode, finished, succeeded int) JoinDecision {
	strategy := joinStrategies[JoinMode(node.JoinMode)]
	if strategy == nil {
		strategy = joinStrategies[JoinAll]
	}
	return strategy.Evaluate(JoinProgress{
		Total:     len(node.DependsOn),
		Finished:  finished,
		Succeeded: succeeded,
		Required:  node.MinDependencies,
	})
}
