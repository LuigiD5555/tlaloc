package tlaloque

import (
	"fmt"
	"strings"
)

type NodeFailurePolicy string

const (
	FailureStrict    NodeFailurePolicy = "STRICT"
	FailureTolerated NodeFailurePolicy = "TOLERATED"
)

func normalizeNodeFailurePolicy(raw string) (NodeFailurePolicy, error) {
	value := NodeFailurePolicy(strings.ToUpper(strings.TrimSpace(raw)))
	if value == "" {
		return FailureStrict, nil
	}
	switch value {
	case FailureStrict, FailureTolerated:
		return value, nil
	default:
		return "", fmt.Errorf("unsupported failure policy %q", raw)
	}
}

func toleratesNodeFailure(node SwarmNode) bool {
	policy, err := normalizeNodeFailurePolicy(node.FailurePolicy)
	return err == nil && policy == FailureTolerated
}

func nodeStateSatisfiesRun(node SwarmNode, state NodeState) bool {
	if state == NodeCompleted {
		return true
	}
	if toleratesNodeFailure(node) && (state == NodeFailed || state == NodeBlocked) {
		return true
	}
	return false
}
