package tlaloque

import (
	"fmt"
	"strings"
)

// ProductSelectionRequest asks for a worker capable of producing one typed
// data product. It intentionally reuses the registry's SelectionStrategy for
// ranking, while product eligibility is expressed as Specifications.
type ProductSelectionRequest struct {
	Product             string
	ScopeHint           string
	DomainHint          string
	PreferDeterministic bool
	MaxParameters       int64
	ExcludeWorkerIDs    []string
}

func (r *Registry) SelectProducerResult(req ProductSelectionRequest) Result[CapabilityWorker] {
	product := strings.TrimSpace(req.Product)
	scopeHint := strings.ToUpper(strings.TrimSpace(req.ScopeHint))
	domainHint := strings.ToUpper(strings.TrimSpace(req.DomainHint))
	if product == "" {
		return DomainResult[CapabilityWorker](ResultInvalidRequest, nil, Diagnostic{Code: "MISSING_PRODUCT", Message: "required product is empty"})
	}
	if scopeHint != "" && scopeHint != ScopeGeneral && scopeHint != ScopeSpecific {
		return DomainResult[CapabilityWorker](ResultInvalidRequest, nil, Diagnostic{Code: "UNSUPPORTED_SCOPE", Message: fmt.Sprintf("unsupported scope hint %q", scopeHint)})
	}
	if scopeHint == ScopeSpecific && domainHint == "" {
		return DomainResult[CapabilityWorker](ResultInvalidRequest, nil, Diagnostic{Code: "MISSING_DOMAIN", Message: "SPECIFIC scope requires domain hint"})
	}

	excluded := map[string]struct{}{}
	for _, id := range req.ExcludeWorkerIDs {
		if id = strings.TrimSpace(id); id != "" {
			excluded[id] = struct{}{}
		}
	}
	selectionReq := SelectionRequest{
		ScopeHint:           scopeHint,
		DomainHint:          domainHint,
		PreferDeterministic: req.PreferDeterministic,
		MaxParameters:       req.MaxParameters,
	}
	specs := productCandidateSpecifications(product, excluded)

	r.mu.RLock()
	candidates := make([]SelectionCandidate, 0, len(r.workers))
	for _, worker := range r.workers {
		desc, err := worker.Descriptor().Normalize()
		if err != nil || !matchesAll(specs, desc, selectionReq) {
			continue
		}
		candidates = append(candidates, SelectionCandidate{Worker: worker, Desc: desc})
	}
	strategy := r.selection
	r.mu.RUnlock()

	if len(candidates) == 0 {
		return DomainResult[CapabilityWorker](ResultNoCandidate, nil, Diagnostic{
			Code:    "NO_PRODUCT_PRODUCER",
			Message: fmt.Sprintf("no worker can produce %q for scope=%s domain=%s", product, scopeHint, domainHint),
			Fields:  map[string]any{"product": product},
		})
	}
	if strategy == nil {
		strategy = RankedSelectionStrategy{Scoring: DefaultScoringStrategy{}}
	}
	return strategy.Select(candidates, selectionReq)
}

func (r *Registry) SelectProducer(req ProductSelectionRequest) (CapabilityWorker, error) {
	result := r.SelectProducerResult(req)
	if result.Err != nil {
		return nil, result.Err
	}
	if result.Code == ResultSuccess {
		return result.Value, nil
	}
	if len(result.Diagnostics) > 0 && result.Diagnostics[0].Message != "" {
		return nil, fmt.Errorf("%s", result.Diagnostics[0].Message)
	}
	return nil, fmt.Errorf("product selection failed: %s", result.Code)
}

func productCandidateSpecifications(product string, excluded map[string]struct{}) []CandidateSpecification {
	return []CandidateSpecification{
		candidateSpecificationFunc(func(desc CapabilityDescriptor, _ SelectionRequest) bool {
			if _, skip := excluded[desc.ID]; skip {
				return false
			}
			for _, produced := range desc.Produces {
				if produced == product {
					return true
				}
			}
			return false
		}),
		candidateSpecificationFunc(func(desc CapabilityDescriptor, req SelectionRequest) bool {
			return req.MaxParameters <= 0 || desc.ParameterCount <= req.MaxParameters
		}),
		candidateSpecificationFunc(func(desc CapabilityDescriptor, req SelectionRequest) bool {
			domainHint := strings.ToUpper(strings.TrimSpace(req.DomainHint))
			return domainHint != "" || desc.Scope != ScopeSpecific
		}),
		candidateSpecificationFunc(func(desc CapabilityDescriptor, req SelectionRequest) bool {
			scopeHint := strings.ToUpper(strings.TrimSpace(req.ScopeHint))
			switch scopeHint {
			case "":
				return true
			case ScopeGeneral:
				return desc.Scope == ScopeGeneral
			case ScopeSpecific:
				return desc.Scope == ScopeSpecific && desc.Domain == strings.ToUpper(strings.TrimSpace(req.DomainHint))
			default:
				return false
			}
		}),
		candidateSpecificationFunc(func(desc CapabilityDescriptor, req SelectionRequest) bool {
			domainHint := strings.ToUpper(strings.TrimSpace(req.DomainHint))
			return domainHint == "" || desc.Scope != ScopeSpecific || desc.Domain == domainHint
		}),
	}
}
