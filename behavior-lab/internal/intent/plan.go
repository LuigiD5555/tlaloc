package intent

import (
	"fmt"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// IntentPlan is the DAG that satisfies a CompiledIntent, plus the workers
// pinned into it and any advisories the planner could not turn into hard
// selection (a downstream gate should act on Warnings before execution).
type IntentPlan struct {
	Plan     tlaloque.SwarmPlan
	Selected []tlaloque.CapabilityDescriptor
	Warnings []string
}

// PlanFor resolves every goal in compiled against registry and merges the
// results into one SwarmPlan (a producer shared by two goals appears
// once). It calls only the existing tlaloque.Registry.ResolveGoal — it
// does not touch the runtime. Risk and evidence requirements that cannot
// be enforced by worker selection today are reported as Warnings.
func PlanFor(registry *tlaloque.Registry, compiled CompiledIntent, planID string, maxParallel int) (IntentPlan, error) {
	if registry == nil {
		return IntentPlan{}, fmt.Errorf("intent: registry is required")
	}
	if strings.TrimSpace(planID) == "" {
		planID = "intent-plan"
	}
	if maxParallel <= 0 {
		maxParallel = 1
	}

	merged := map[string]tlaloque.SwarmNode{}
	selected := map[string]tlaloque.CapabilityDescriptor{}
	for _, goal := range compiled.Goals {
		resolved, err := registry.ResolveGoal(goal, planID+"-"+strings.ToLower(goal.Capability), maxParallel)
		if err != nil {
			return IntentPlan{}, fmt.Errorf("intent: resolving %s: %w", goal.Capability, err)
		}
		for _, node := range resolved.Plan.Nodes {
			if _, ok := merged[node.ID]; !ok {
				merged[node.ID] = node
			}
		}
		for _, desc := range resolved.Selected {
			selected[desc.ID] = desc
		}
	}

	ids := make([]string, 0, len(merged))
	for id := range merged {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	plan := tlaloque.SwarmPlan{Schema: tlaloque.SwarmSchemaR0, ID: planID, MaxParallel: maxParallel}
	descs := make([]tlaloque.CapabilityDescriptor, 0, len(ids))
	for _, id := range ids {
		plan.Nodes = append(plan.Nodes, merged[id])
		descs = append(descs, selected[id])
	}
	normalized, err := plan.Normalize()
	if err != nil {
		return IntentPlan{}, fmt.Errorf("intent: merged plan does not normalize: %w", err)
	}

	warnings := advisories(compiled, descs)
	return IntentPlan{Plan: normalized, Selected: descs, Warnings: warnings}, nil
}

// advisories flags intent requirements that plan-time worker selection
// cannot yet guarantee, so a caller decides whether to proceed.
func advisories(compiled CompiledIntent, selected []tlaloque.CapabilityDescriptor) []string {
	var warnings []string

	if compiled.Risk.AbstentionRequired {
		for _, desc := range selected {
			if desc.Engine == tlaloque.EngineModel && !desc.Deterministic {
				warnings = append(warnings, fmt.Sprintf(
					"risk.abstention_required: worker %q is a probabilistic model; its CalibrationProfile must be checked before execution", desc.ID))
			}
		}
	}

	if len(compiled.EvidenceRequirements) > 0 {
		warnings = append(warnings,
			"evidence_requirements are declared but not verifiable at plan time; the run's evidence must be checked against them after execution")
	}

	for _, desc := range selected {
		if desc.ParameterCount > 1_000_000_000 {
			warnings = append(warnings, fmt.Sprintf(
				"worker %q has %d parameters — large for a bounded-specialist swarm; confirm this is intended", desc.ID, desc.ParameterCount))
		}
	}

	sort.Strings(warnings)
	return warnings
}
