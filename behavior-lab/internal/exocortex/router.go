package exocortex

import (
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// CapabilityRouter is the E4 model-adaptive decision layer. It is installed
// on the existing tlaloque.Registry via SetSelectionStrategy rather than
// introducing a second registry or router (E0.15): Registry.ResolveGoal and
// Registry.Select/SelectResult keep working exactly as before, they simply
// now consult CapabilityProfiles when more than one candidate offers the
// same capability.
//
// Routing rule (E0.9, E4): the smallest candidate that (1) supports the
// opcode, (2) is not vetoed by its own CapabilityProfile for that opcode,
// wins, ranked by the registry's existing deterministic-first,
// smallest-parameter-count-first ordering. There is no learned router and
// no routing LLM.
type CapabilityRouter struct {
	// Base ranks the surviving candidates. Defaults to the registry's
	// ordinary RankedSelectionStrategy/DefaultScoringStrategy, which
	// already encodes "deterministic beats model, smaller beats larger".
	Base tlaloque.SelectionStrategy

	// Profiles maps a worker's CapabilityDescriptor.ID to the
	// CapabilityProfile governing it. A worker absent from this map (a
	// purely deterministic Tlaloque with no empirical profile, e.g.
	// Numeric/Normalize) is never vetoed here.
	Profiles map[string]CapabilityProfile
}

func (r CapabilityRouter) base() tlaloque.SelectionStrategy {
	if r.Base != nil {
		return r.Base
	}
	return tlaloque.RankedSelectionStrategy{Scoring: tlaloque.DefaultScoringStrategy{}}
}

// Select implements tlaloque.SelectionStrategy: it removes every candidate
// whose profile marks the requested capability EXTERNALIZE or
// DO_NOT_DEPLOY, then delegates ranking among the survivors. If every
// candidate is vetoed, selection fails explicitly (ResultNoCandidate)
// rather than silently falling through to a collapsed executor.
func (r CapabilityRouter) Select(candidates []tlaloque.SelectionCandidate, req tlaloque.SelectionRequest) tlaloque.Result[tlaloque.CapabilityWorker] {
	eligible, diagnostic := r.filter(candidates, req)
	if len(eligible) == 0 && diagnostic.Code != "" {
		return tlaloque.DomainResult[tlaloque.CapabilityWorker](tlaloque.ResultNoCandidate, nil, diagnostic)
	}
	return r.base().Select(eligible, req)
}

// SelectMany mirrors Select for multi-selection callers, delegating to the
// base strategy's MultiSelectionStrategy when it implements one.
func (r CapabilityRouter) SelectMany(candidates []tlaloque.SelectionCandidate, req tlaloque.SelectionRequest, limit int) tlaloque.Result[[]tlaloque.CapabilityWorker] {
	eligible, diagnostic := r.filter(candidates, req)
	if len(eligible) == 0 && diagnostic.Code != "" {
		return tlaloque.DomainResult[[]tlaloque.CapabilityWorker](tlaloque.ResultNoCandidate, nil, diagnostic)
	}
	if multi, ok := r.base().(tlaloque.MultiSelectionStrategy); ok {
		return multi.SelectMany(eligible, req, limit)
	}
	result := r.base().Select(eligible, req)
	if !result.OK() {
		return tlaloque.DomainResult[[]tlaloque.CapabilityWorker](result.Code, nil, result.Diagnostics...)
	}
	return tlaloque.Success([]tlaloque.CapabilityWorker{result.Value})
}

func (r CapabilityRouter) filter(candidates []tlaloque.SelectionCandidate, req tlaloque.SelectionRequest) ([]tlaloque.SelectionCandidate, tlaloque.Diagnostic) {
	eligible := make([]tlaloque.SelectionCandidate, 0, len(candidates))
	vetoed := 0
	for _, c := range candidates {
		profile, hasProfile := r.Profiles[c.Desc.ID]
		if !hasProfile {
			eligible = append(eligible, c)
			continue
		}
		entry, hasEntry := profile.Entry(req.Capability)
		if !hasEntry {
			eligible = append(eligible, c)
			continue
		}
		if entry.DeploymentRecommendation == DeploymentExternalize || entry.DeploymentRecommendation == DeploymentDoNotDeploy {
			vetoed++
			continue
		}
		eligible = append(eligible, c)
	}
	if len(eligible) == 0 && vetoed > 0 {
		return eligible, tlaloque.Diagnostic{
			Code:    "CAPABILITY_PROFILE_VETO",
			Message: "every candidate for capability " + req.Capability + " is vetoed by its CapabilityProfile (EXTERNALIZE or DO_NOT_DEPLOY); register a deterministic alternative",
		}
	}
	return eligible, tlaloque.Diagnostic{}
}

// Install replaces the Registry's selection strategy with this router.
// The Registry, its workers, and ResolveGoal are all reused unchanged.
func (r CapabilityRouter) Install(registry *tlaloque.Registry) {
	registry.SetSelectionStrategy(r)
}
