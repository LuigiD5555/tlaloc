package tlaloque

import "fmt"

type NodeState string
type NodeEvent string

const (
	NodePending   NodeState = "PENDING"
	NodeReady     NodeState = "READY"
	NodeRunning   NodeState = "RUNNING"
	NodeCompleted NodeState = "COMPLETED"
	NodeFailed    NodeState = "FAILED"
	NodeBlocked   NodeState = "BLOCKED"

	NodeDependenciesSatisfied NodeEvent = "DEPENDENCIES_SATISFIED"
	NodeDispatched            NodeEvent = "DISPATCHED"
	NodeSucceeded             NodeEvent = "SUCCEEDED"
	NodeExecutionFailed       NodeEvent = "FAILED"
	NodeDependenciesImpossible NodeEvent = "DEPENDENCIES_IMPOSSIBLE"
)

var nodeTransitions = map[NodeState]map[NodeEvent]NodeState{
	NodePending: {
		NodeDependenciesSatisfied: NodeReady,
		NodeDependenciesImpossible: NodeBlocked,
	},
	NodeReady: {
		NodeDispatched:      NodeRunning,
		NodeExecutionFailed: NodeFailed,
	},
	NodeRunning: {
		NodeSucceeded:       NodeCompleted,
		NodeExecutionFailed: NodeFailed,
	},
}

func transitionNode(current NodeState, event NodeEvent) (NodeState, error) {
	byEvent, ok := nodeTransitions[current]
	if !ok {
		return current, fmt.Errorf("node state %s is terminal", current)
	}
	next, ok := byEvent[event]
	if !ok {
		return current, fmt.Errorf("illegal node transition %s + %s", current, event)
	}
	return next, nil
}
