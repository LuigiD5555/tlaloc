package canonicalstate

import (
	"fmt"
	"strings"
)

type ClaimCardinality string
type FusionMode string

const (
	CardinalityMany ClaimCardinality = "MANY"
	CardinalityOne  ClaimCardinality = "ONE"

	FusionExact       FusionMode = "EXACT"
	FusionSingleValue FusionMode = "SINGLE_VALUE"
)

type ClaimPolicy struct {
	Cardinality ClaimCardinality `json:"cardinality"`
	Fusion      FusionMode       `json:"fusion"`
}

// ClaimPolicyRegistry is a policy object keyed by normalized predicate. An
// absent predicate deliberately uses R0 semantics: multiple object values are
// independent claims and only opposite polarities of the same value conflict.
type ClaimPolicyRegistry struct {
	ID       string                 `json:"id"`
	Policies map[string]ClaimPolicy `json:"policies"`
}

func NewClaimPolicyRegistry(id string, policies map[string]ClaimPolicy) (*ClaimPolicyRegistry, error) {
	registry := &ClaimPolicyRegistry{ID: strings.TrimSpace(id), Policies: map[string]ClaimPolicy{}}
	if registry.ID == "" {
		return nil, fmt.Errorf("claim policy registry id is required")
	}
	for predicate, policy := range policies {
		predicate = norm(predicate)
		if predicate == "" {
			return nil, fmt.Errorf("claim policy predicate is required")
		}
		if policy.Cardinality == "" {
			policy.Cardinality = CardinalityMany
		}
		switch policy.Cardinality {
		case CardinalityMany:
			if policy.Fusion == "" {
				policy.Fusion = FusionExact
			}
		case CardinalityOne:
			if policy.Fusion == "" {
				policy.Fusion = FusionSingleValue
			}
		default:
			return nil, fmt.Errorf("predicate %q has unsupported cardinality %q", predicate, policy.Cardinality)
		}
		if _, ok := defaultFusionStrategies[policy.Fusion]; !ok {
			return nil, fmt.Errorf("predicate %q has unsupported fusion mode %q", predicate, policy.Fusion)
		}
		registry.Policies[predicate] = policy
	}
	return registry, nil
}

func (r *ClaimPolicyRegistry) PolicyFor(predicate string) ClaimPolicy {
	if r != nil {
		if policy, ok := r.Policies[norm(predicate)]; ok {
			return policy
		}
	}
	return ClaimPolicy{Cardinality: CardinalityMany, Fusion: FusionExact}
}

func claimGroupKey(claim Claim, policy ClaimPolicy) string {
	base := norm(claim.Subject) + "\x00" + norm(claim.Predicate)
	if policy.Cardinality == CardinalityOne {
		return base
	}
	return base + "\x00" + norm(claim.Object)
}
