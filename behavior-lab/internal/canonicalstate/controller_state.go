package canonicalstate

type ControllerState string
type ControllerSignal string

const (
	ControllerReducing  ControllerState = "REDUCING"
	ControllerVerifying ControllerState = "VERIFYING"
	ControllerExpanding ControllerState = "EXPANDING"
	ControllerComplete  ControllerState = "COMPLETE"

	SignalConflict    ControllerSignal = "CONFLICT"
	SignalUncertainty ControllerSignal = "UNCERTAINTY"
	SignalSatisfied   ControllerSignal = "SATISFIED"
)

var controllerTransitions = map[ControllerState]map[ControllerSignal]ControllerState{
	ControllerReducing: {
		SignalConflict:    ControllerVerifying,
		SignalUncertainty: ControllerExpanding,
		SignalSatisfied:   ControllerComplete,
	},
}

var controllerActions = map[ControllerState]string{
	ControllerVerifying: "VERIFY_CONFLICTS",
	ControllerExpanding: "EXPAND_EVIDENCE",
	ControllerComplete:  "REDUCE_COMPLETE",
}

type controllerRule struct {
	Signal  ControllerSignal
	Matches func(State) bool
}

var reducingRules = []controllerRule{
	{Signal: SignalConflict, Matches: func(s State) bool { return len(s.Conflicts) > 0 }},
	{Signal: SignalUncertainty, Matches: func(s State) bool { return s.Metrics.Uncertainty > .35 }},
	{Signal: SignalSatisfied, Matches: func(State) bool { return true }},
}

func nextControllerState(current ControllerState, s State) ControllerState {
	transitions, ok := controllerTransitions[current]
	if !ok {
		return current
	}
	for _, rule := range reducingRules {
		if rule.Matches(s) {
			if next, exists := transitions[rule.Signal]; exists {
				return next
			}
		}
	}
	return current
}
