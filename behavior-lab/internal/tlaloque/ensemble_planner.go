package tlaloque

import (
	"fmt"
	"sort"
	"strings"
)

const DefaultEnsembleContextPrefix = "ensemble.member."

type EnsembleGoal struct {
	ID                        string `json:"id,omitempty"`
	Capability                string `json:"capability"`
	ScopeHint                 string `json:"scope_hint,omitempty"`
	DomainHint                string `json:"domain_hint,omitempty"`
	PreferDeterministic       bool   `json:"prefer_deterministic,omitempty"`
	MaxParameters             int64  `json:"max_parameters,omitempty"`
	Members                   int    `json:"members"`
	FusionCapability          string `json:"fusion_capability"`
	FusionWorkerID            string `json:"fusion_worker_id,omitempty"`
	FusionPreferDeterministic bool   `json:"fusion_prefer_deterministic,omitempty"`
	JoinMode                  string `json:"join_mode,omitempty"`
	MinMembers                int    `json:"min_members,omitempty"`
	FusionNodeID              string `json:"fusion_node_id,omitempty"`
	ContextPrefix             string `json:"context_prefix,omitempty"`
}

type PlannedEnsemble struct {
	Goal    EnsembleGoal           `json:"goal"`
	Plan    SwarmPlan              `json:"plan"`
	Members []CapabilityDescriptor `json:"members"`
	Fusion  CapabilityDescriptor   `json:"fusion"`
}

// ResolveEnsemble selects a reproducible set of peer workers and places a real
// fusion Tlaloque behind them. Member failures are tolerated at the run level;
// the fusion node's ALL/ANY/QUORUM join is the authority that decides whether
// enough member evidence exists to continue.
func (r *Registry) ResolveEnsemble(goal EnsembleGoal, maxParallel int) (PlannedEnsemble, error) {
	goal.ID = strings.TrimSpace(goal.ID)
	goal.Capability = strings.ToUpper(strings.TrimSpace(goal.Capability))
	goal.ScopeHint = strings.ToUpper(strings.TrimSpace(goal.ScopeHint))
	goal.DomainHint = strings.ToUpper(strings.TrimSpace(goal.DomainHint))
	goal.FusionCapability = strings.ToUpper(strings.TrimSpace(goal.FusionCapability))
	goal.FusionWorkerID = strings.TrimSpace(goal.FusionWorkerID)
	goal.FusionNodeID = strings.TrimSpace(goal.FusionNodeID)
	goal.ContextPrefix = strings.TrimSpace(goal.ContextPrefix)
	if goal.ID == "" {
		goal.ID = "ensemble-" + strings.ToLower(strings.ReplaceAll(goal.Capability, "_", "-"))
	}
	if goal.Capability == "" {
		return PlannedEnsemble{}, fmt.Errorf("ensemble capability is required")
	}
	if goal.Members < 2 {
		return PlannedEnsemble{}, fmt.Errorf("ensemble requires at least two members")
	}
	if goal.FusionCapability == "" {
		return PlannedEnsemble{}, fmt.Errorf("fusion capability is required")
	}
	if goal.FusionNodeID == "" {
		goal.FusionNodeID = "ensemble-fusion"
	}
	if goal.ContextPrefix == "" {
		goal.ContextPrefix = DefaultEnsembleContextPrefix
	}
	joinMode, err := normalizeJoinMode(goal.JoinMode)
	if err != nil {
		return PlannedEnsemble{}, err
	}
	if strings.TrimSpace(goal.JoinMode) != "" {
		goal.JoinMode = string(joinMode)
	}
	if joinMode == JoinQuorum {
		if goal.MinMembers <= 0 {
			goal.MinMembers = goal.Members/2 + 1
		}
		if goal.MinMembers > goal.Members {
			return PlannedEnsemble{}, fmt.Errorf("ensemble quorum %d exceeds member count %d", goal.MinMembers, goal.Members)
		}
	} else if goal.MinMembers != 0 {
		return PlannedEnsemble{}, fmt.Errorf("min_members is only valid with QUORUM")
	}

	workers, err := r.SelectMany(SelectionRequest{
		Capability:          goal.Capability,
		ScopeHint:           goal.ScopeHint,
		DomainHint:          goal.DomainHint,
		PreferDeterministic: goal.PreferDeterministic,
		MaxParameters:       goal.MaxParameters,
	}, goal.Members)
	if err != nil {
		return PlannedEnsemble{}, err
	}
	if len(workers) != goal.Members {
		return PlannedEnsemble{}, fmt.Errorf("ensemble requested %d members but only %d eligible workers are available", goal.Members, len(workers))
	}

	memberDescs := make([]CapabilityDescriptor, 0, len(workers))
	for _, worker := range workers {
		desc, err := worker.Descriptor().Normalize()
		if err != nil {
			return PlannedEnsemble{}, err
		}
		memberDescs = append(memberDescs, desc)
	}
	sort.Slice(memberDescs, func(i, j int) bool { return memberDescs[i].ID < memberDescs[j].ID })

	fusionWorker, err := r.Select(SelectionRequest{
		Capability:          goal.FusionCapability,
		WorkerID:            goal.FusionWorkerID,
		DomainHint:          goal.DomainHint,
		PreferDeterministic: goal.FusionPreferDeterministic,
	})
	if err != nil {
		return PlannedEnsemble{}, fmt.Errorf("fusion worker: %w", err)
	}
	fusionDesc, err := fusionWorker.Descriptor().Normalize()
	if err != nil {
		return PlannedEnsemble{}, err
	}
	if len(fusionDesc.Dependencies) > 0 || len(fusionDesc.Requires) > 0 {
		return PlannedEnsemble{}, fmt.Errorf("fusion worker %q must be atomic in ensemble R1; dependencies/requires are not supported yet", fusionDesc.ID)
	}

	nodes := make([]SwarmNode, 0, len(memberDescs)+1)
	memberNodeIDs := make([]string, 0, len(memberDescs))
	bindings := make(map[string]string, len(memberDescs))
	for _, desc := range memberDescs {
		nodeID := "ensemble-member-" + desc.ID
		if nodeID == goal.FusionNodeID {
			return PlannedEnsemble{}, fmt.Errorf("fusion node id %q collides with member node", goal.FusionNodeID)
		}
		memberNodeIDs = append(memberNodeIDs, nodeID)
		bindings[goal.ContextPrefix+desc.ID] = nodeID
		nodes = append(nodes, SwarmNode{
			ID:                  nodeID,
			Capability:          desc.Capability,
			WorkerID:            desc.ID,
			PreferDeterministic: goal.PreferDeterministic,
			MaxParameters:       goal.MaxParameters,
			FailurePolicy:       string(FailureTolerated),
		})
	}
	sort.Strings(memberNodeIDs)

	fusionNode := SwarmNode{
		ID:                  goal.FusionNodeID,
		Capability:          fusionDesc.Capability,
		WorkerID:            fusionDesc.ID,
		DependsOn:           memberNodeIDs,
		InputBindings:       bindings,
		PreferDeterministic: goal.FusionPreferDeterministic,
		JoinMode:            goal.JoinMode,
	}
	if joinMode == JoinQuorum {
		fusionNode.MinDependencies = goal.MinMembers
	}
	nodes = append(nodes, fusionNode)

	if maxParallel <= 0 {
		maxParallel = goal.Members
	}
	plan, err := (SwarmPlan{
		Schema:      SwarmSchemaR0,
		ID:          goal.ID,
		MaxParallel: maxParallel,
		Nodes:       nodes,
	}).Normalize()
	if err != nil {
		return PlannedEnsemble{}, err
	}
	return PlannedEnsemble{Goal: goal, Plan: plan, Members: memberDescs, Fusion: fusionDesc}, nil
}
