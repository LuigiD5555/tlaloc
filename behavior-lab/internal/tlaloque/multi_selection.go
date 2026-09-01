package tlaloque

import (
	"fmt"
	"strings"
)

// SelectManyResult returns up to limit distinct eligible workers. The limit is
// a maximum, not a quorum requirement: if fewer workers are registered, all
// eligible workers are returned successfully. Composition/quorum semantics live
// above selection in the swarm planner.
func (r *Registry) SelectManyResult(req SelectionRequest, limit int) Result[[]CapabilityWorker] {
	if limit <= 0 {
		return DomainResult[[]CapabilityWorker](ResultInvalidRequest, nil, Diagnostic{
			Code:    "INVALID_SELECTION_LIMIT",
			Message: "selection limit must be greater than zero",
		})
	}

	req.Capability = strings.ToUpper(strings.TrimSpace(req.Capability))
	req.ScopeHint = strings.ToUpper(strings.TrimSpace(req.ScopeHint))
	req.DomainHint = strings.ToUpper(strings.TrimSpace(req.DomainHint))
	if req.ScopeHint != "" && req.ScopeHint != ScopeGeneral && req.ScopeHint != ScopeSpecific {
		return DomainResult[[]CapabilityWorker](ResultInvalidRequest, nil, Diagnostic{
			Code:    "UNSUPPORTED_SCOPE",
			Message: fmt.Sprintf("unsupported scope hint %q", req.ScopeHint),
		})
	}
	if req.ScopeHint == ScopeSpecific && req.DomainHint == "" {
		return DomainResult[[]CapabilityWorker](ResultInvalidRequest, nil, Diagnostic{
			Code:    "MISSING_DOMAIN",
			Message: "SPECIFIC scope requires domain hint",
		})
	}

	// A pinned worker is an explicit request for one concrete implementation;
	// do not silently add peers just because the caller supplied a larger limit.
	if req.WorkerID != "" {
		one := r.SelectResult(req)
		if one.Err != nil {
			return Failure[[]CapabilityWorker](one.Err)
		}
		if one.Code != ResultSuccess {
			return DomainResult[[]CapabilityWorker](one.Code, nil, one.Diagnostics...)
		}
		return Success([]CapabilityWorker{one.Value})
	}

	r.mu.RLock()
	candidates := make([]SelectionCandidate, 0, len(r.workers))
	for _, worker := range r.workers {
		d, err := worker.Descriptor().Normalize()
		if err != nil || !matchesAll(r.specs, d, req) {
			continue
		}
		candidates = append(candidates, SelectionCandidate{Worker: worker, Desc: d})
	}
	strategy := r.selection
	r.mu.RUnlock()
	if strategy == nil {
		strategy = RankedSelectionStrategy{Scoring: DefaultScoringStrategy{}}
	}
	if multi, ok := strategy.(MultiSelectionStrategy); ok {
		return multi.SelectMany(candidates, req, limit)
	}
	return selectManyByRepeatedStrategy(strategy, candidates, req, limit)
}

// SelectMany is the compatibility adapter for callers that still use Go
// errors. New orchestration code should prefer SelectManyResult and branch on
// ResultCode.
func (r *Registry) SelectMany(req SelectionRequest, limit int) ([]CapabilityWorker, error) {
	result := r.SelectManyResult(req, limit)
	if result.Err != nil {
		return nil, result.Err
	}
	if result.Code == ResultSuccess {
		return result.Value, nil
	}
	if len(result.Diagnostics) > 0 && result.Diagnostics[0].Message != "" {
		return result.Value, fmt.Errorf("%s", result.Diagnostics[0].Message)
	}
	return result.Value, fmt.Errorf("multi-worker selection failed: %s", result.Code)
}

func selectManyByRepeatedStrategy(strategy SelectionStrategy, candidates []SelectionCandidate, req SelectionRequest, limit int) Result[[]CapabilityWorker] {
	if len(candidates) == 0 {
		return DomainResult[[]CapabilityWorker](ResultNoCandidate, nil, noEligibleWorkerDiagnostic(req))
	}
	remaining := append([]SelectionCandidate(nil), candidates...)
	selected := make([]CapabilityWorker, 0, limit)
	for len(selected) < limit && len(remaining) > 0 {
		result := strategy.Select(remaining, req)
		if result.Err != nil {
			return Failure[[]CapabilityWorker](result.Err)
		}
		if result.Code != ResultSuccess || result.Value == nil {
			if len(selected) > 0 {
				return DomainResult[[]CapabilityWorker](ResultPartial, selected, result.Diagnostics...)
			}
			return DomainResult[[]CapabilityWorker](result.Code, nil, result.Diagnostics...)
		}
		desc, err := result.Value.Descriptor().Normalize()
		if err != nil {
			return Failure[[]CapabilityWorker](err)
		}
		index := -1
		for i, candidate := range remaining {
			if candidate.Desc.ID == desc.ID {
				index = i
				break
			}
		}
		if index < 0 {
			return Failure[[]CapabilityWorker](fmt.Errorf("selection strategy returned worker %q outside eligible candidate set", desc.ID))
		}
		selected = append(selected, result.Value)
		remaining = append(remaining[:index], remaining[index+1:]...)
	}
	return Success(selected)
}
