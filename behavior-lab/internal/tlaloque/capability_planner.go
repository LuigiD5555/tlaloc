package tlaloque

import (
	"fmt"
	"sort"
	"strings"
)

// CapabilityGoal asks Tlaloc for behavior, not for a particular model.
type CapabilityGoal struct {
	Capability          string `json:"capability"`
	ScopeHint           string `json:"scope_hint,omitempty"`
	DomainHint          string `json:"domain_hint,omitempty"`
	PreferDeterministic bool   `json:"prefer_deterministic,omitempty"`
	MaxParameters       int64  `json:"max_parameters,omitempty"`
}

type PlannedSwarm struct {
	Goal     CapabilityGoal         `json:"goal"`
	Plan     SwarmPlan              `json:"plan"`
	Selected []CapabilityDescriptor `json:"selected_workers"`
}

// ResolveGoal recursively follows the selected workers' declared capability
// dependencies and pins each chosen worker into a reproducible DAG.
func (r *Registry) ResolveGoal(goal CapabilityGoal, planID string, maxParallel int) (PlannedSwarm, error) {
	goal.Capability = strings.ToUpper(strings.TrimSpace(goal.Capability))
	goal.ScopeHint = strings.ToUpper(strings.TrimSpace(goal.ScopeHint))
	goal.DomainHint = strings.ToUpper(strings.TrimSpace(goal.DomainHint))
	if goal.Capability == "" { return PlannedSwarm{}, fmt.Errorf("goal capability is required") }
	if strings.TrimSpace(planID) == "" { planID = "auto-" + strings.ToLower(strings.ReplaceAll(goal.Capability, "_", "-")) }
	if maxParallel <= 0 { maxParallel = 1 }

	nodes := map[string]SwarmNode{}
	selected := map[string]CapabilityDescriptor{}
	visiting := map[string]bool{}

	var resolve func(SelectionRequest) (string, error)
	resolve = func(req SelectionRequest) (string, error) {
		worker, err := r.Select(req)
		if err != nil { return "", err }
		desc, err := worker.Descriptor().Normalize()
		if err != nil { return "", err }
		if _, ok := nodes[desc.ID]; ok { return desc.ID, nil }
		if visiting[desc.ID] { return "", fmt.Errorf("capability dependency cycle through worker %q", desc.ID) }
		visiting[desc.ID] = true
		deps := make([]string, 0, len(desc.Dependencies))
		for _, capability := range desc.Dependencies {
			depID, err := resolve(SelectionRequest{
				Capability:          capability,
				DomainHint:          goal.DomainHint,
				PreferDeterministic: goal.PreferDeterministic,
				MaxParameters:       goal.MaxParameters,
			})
			if err != nil { return "", fmt.Errorf("worker %s dependency %s: %w", desc.ID, capability, err) }
			deps = append(deps, depID)
		}
		sort.Strings(deps)
		nodes[desc.ID] = SwarmNode{ID:desc.ID, Capability:desc.Capability, WorkerID:desc.ID, DependsOn:deps, PreferDeterministic:goal.PreferDeterministic, MaxParameters:goal.MaxParameters}
		selected[desc.ID] = desc
		delete(visiting, desc.ID)
		return desc.ID, nil
	}

	_, err := resolve(SelectionRequest{Capability:goal.Capability, ScopeHint:goal.ScopeHint, DomainHint:goal.DomainHint, PreferDeterministic:goal.PreferDeterministic, MaxParameters:goal.MaxParameters})
	if err != nil { return PlannedSwarm{}, err }

	ids := make([]string, 0, len(nodes))
	for id := range nodes { ids = append(ids, id) }
	sort.Strings(ids)
	plan := SwarmPlan{Schema:SwarmSchemaR0, ID:planID, MaxParallel:maxParallel, Nodes:make([]SwarmNode,0,len(ids))}
	for _, id := range ids { plan.Nodes = append(plan.Nodes, nodes[id]) }
	plan, err = plan.Normalize()
	if err != nil { return PlannedSwarm{}, err }
	descs := make([]CapabilityDescriptor,0,len(ids))
	for _, id := range ids { descs = append(descs, selected[id]) }
	return PlannedSwarm{Goal:goal, Plan:plan, Selected:descs}, nil
}
