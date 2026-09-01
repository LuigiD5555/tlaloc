package tlaloque

import (
	"fmt"
	"sort"
	"strings"
)

// CapabilityGoal asks Tlaloc for behavior, not for a particular model.
type CapabilityGoal struct {
	Capability          string   `json:"capability"`
	ScopeHint           string   `json:"scope_hint,omitempty"`
	DomainHint          string   `json:"domain_hint,omitempty"`
	PreferDeterministic bool     `json:"prefer_deterministic,omitempty"`
	MaxParameters       int64    `json:"max_parameters,omitempty"`
	AvailableProducts   []string `json:"available_products,omitempty"`
}

type PlannedSwarm struct {
	Goal     CapabilityGoal         `json:"goal"`
	Plan     SwarmPlan              `json:"plan"`
	Selected []CapabilityDescriptor `json:"selected_workers"`
}

// ResolveGoal recursively follows both R0 capability Dependencies and R1 typed
// data Requires. Every selected producer is pinned into the resulting DAG, so
// the runtime remains reproducible even though planning is capability-driven.
func (r *Registry) ResolveGoal(goal CapabilityGoal, planID string, maxParallel int) (PlannedSwarm, error) {
	goal.Capability = strings.ToUpper(strings.TrimSpace(goal.Capability))
	goal.ScopeHint = strings.ToUpper(strings.TrimSpace(goal.ScopeHint))
	goal.DomainHint = strings.ToUpper(strings.TrimSpace(goal.DomainHint))
	goal.AvailableProducts = normalizeDataContractList(goal.AvailableProducts)
	if goal.Capability == "" {
		return PlannedSwarm{}, fmt.Errorf("goal capability is required")
	}
	if strings.TrimSpace(planID) == "" {
		planID = "auto-" + strings.ToLower(strings.ReplaceAll(goal.Capability, "_", "-"))
	}
	if maxParallel <= 0 {
		maxParallel = 1
	}

	available := map[string]struct{}{}
	for _, product := range goal.AvailableProducts {
		available[product] = struct{}{}
	}
	nodes := map[string]SwarmNode{}
	selected := map[string]CapabilityDescriptor{}
	visiting := map[string]bool{}

	var resolve func(SelectionRequest) (string, error)
	resolve = func(req SelectionRequest) (string, error) {
		worker, err := r.Select(req)
		if err != nil {
			return "", err
		}
		desc, err := worker.Descriptor().Normalize()
		if err != nil {
			return "", err
		}
		if _, ok := nodes[desc.ID]; ok {
			return desc.ID, nil
		}
		if visiting[desc.ID] {
			return "", fmt.Errorf("capability dependency cycle through worker %q", desc.ID)
		}
		visiting[desc.ID] = true
		defer delete(visiting, desc.ID)

		deps := make([]string, 0, len(desc.Dependencies)+len(desc.Requires))
		bindings := map[string]string{}

		for _, capability := range desc.Dependencies {
			depID, err := resolve(SelectionRequest{
				Capability:          capability,
				DomainHint:          goal.DomainHint,
				PreferDeterministic: goal.PreferDeterministic,
				MaxParameters:       goal.MaxParameters,
			})
			if err != nil {
				return "", fmt.Errorf("worker %s dependency %s: %w", desc.ID, capability, err)
			}
			deps = append(deps, depID)
		}

		for _, product := range desc.Requires {
			if _, ok := available[product]; ok {
				continue
			}
			producer, err := r.SelectProducer(ProductSelectionRequest{
				Product:             product,
				DomainHint:          goal.DomainHint,
				PreferDeterministic: goal.PreferDeterministic,
				MaxParameters:       goal.MaxParameters,
				ExcludeWorkerIDs:    visitingWorkerIDs(visiting),
			})
			if err != nil {
				return "", fmt.Errorf("worker %s requires product %s: %w", desc.ID, product, err)
			}
			producerDesc, err := producer.Descriptor().Normalize()
			if err != nil {
				return "", err
			}
			producerID, err := resolve(SelectionRequest{
				Capability:          producerDesc.Capability,
				WorkerID:            producerDesc.ID,
				DomainHint:          goal.DomainHint,
				PreferDeterministic: goal.PreferDeterministic,
				MaxParameters:       goal.MaxParameters,
			})
			if err != nil {
				return "", fmt.Errorf("worker %s product %s producer %s: %w", desc.ID, product, producerDesc.ID, err)
			}
			deps = append(deps, producerID)
			bindings[product] = producerID
		}

		deps = uniqueSortedStrings(deps)
		if len(bindings) == 0 {
			bindings = nil
		}
		nodes[desc.ID] = SwarmNode{
			ID:                  desc.ID,
			Capability:          desc.Capability,
			WorkerID:            desc.ID,
			DependsOn:           deps,
			InputBindings:       bindings,
			PreferDeterministic: goal.PreferDeterministic,
			MaxParameters:       goal.MaxParameters,
		}
		selected[desc.ID] = desc
		return desc.ID, nil
	}

	_, err := resolve(SelectionRequest{
		Capability:          goal.Capability,
		ScopeHint:           goal.ScopeHint,
		DomainHint:          goal.DomainHint,
		PreferDeterministic: goal.PreferDeterministic,
		MaxParameters:       goal.MaxParameters,
	})
	if err != nil {
		return PlannedSwarm{}, err
	}

	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	plan := SwarmPlan{Schema: SwarmSchemaR0, ID: planID, MaxParallel: maxParallel, Nodes: make([]SwarmNode, 0, len(ids))}
	for _, id := range ids {
		plan.Nodes = append(plan.Nodes, nodes[id])
	}
	plan, err = plan.Normalize()
	if err != nil {
		return PlannedSwarm{}, err
	}
	descs := make([]CapabilityDescriptor, 0, len(ids))
	for _, id := range ids {
		descs = append(descs, selected[id])
	}
	return PlannedSwarm{Goal: goal, Plan: plan, Selected: descs}, nil
}

func visitingWorkerIDs(visiting map[string]bool) []string {
	ids := make([]string, 0, len(visiting))
	for id, active := range visiting {
		if active {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
