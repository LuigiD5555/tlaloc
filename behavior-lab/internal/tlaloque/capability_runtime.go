package tlaloque

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"tlaloc.local/behaviorlab/internal/blackboard"
)

const (
	CapabilitySchemaR0  = "tlaloc.tlaloque-capability.r0"
	ScopeGeneral        = "GENERAL"
	ScopeSpecific       = "SPECIFIC"
	EngineProcess       = "PROCESS"
	EngineDeterministic = "DETERMINISTIC"
	EngineModel         = "MODEL"
)

// EmpiricalProfileRef identifies the exact model behavior/configuration whose
// Behavior Lab evidence should be used for this Tlaloque. Condition is the
// model-specific protocol/prompt/configuration label and is intentionally not
// assumed to be portable between model families.
type EmpiricalProfileRef struct {
	ModelID   string `json:"model_id"`
	Condition string `json:"condition,omitempty"`
}

func (r EmpiricalProfileRef) Normalize() (EmpiricalProfileRef, error) {
	r.ModelID = strings.TrimSpace(r.ModelID)
	r.Condition = strings.TrimSpace(r.Condition)
	if r.ModelID == "" {
		return EmpiricalProfileRef{}, fmt.Errorf("empirical profile requires model_id")
	}
	return r, nil
}

func (r EmpiricalProfileRef) Key() string {
	normalized, err := r.Normalize()
	if err != nil {
		return ""
	}
	return normalized.ModelID + "\x00" + normalized.Condition
}

type CapabilityDescriptor struct {
	Schema         string               `json:"schema"`
	ID             string               `json:"id"`
	Capability     string               `json:"capability"`
	Scope          string               `json:"scope"`
	Domain         string               `json:"domain,omitempty"`
	Engine         string               `json:"engine"`
	InputSchema    string               `json:"input_schema"`
	OutputSchema   string               `json:"output_schema"`
	Deterministic  bool                 `json:"deterministic"`
	ParameterCount int64                `json:"parameter_count,omitempty"`
	MaxConcurrency int                  `json:"max_concurrency,omitempty"`
	Tags           []string             `json:"tags,omitempty"`
	Dependencies   []string             `json:"dependencies,omitempty"`
	EmpiricalProfile *EmpiricalProfileRef `json:"empirical_profile,omitempty"`

	// Requires and Produces are typed dataflow contracts. Dependencies remains
	// the R0 capability-to-capability compatibility field; the R1 planner can
	// materialize these data contracts into explicit node dependencies.
	Requires []string `json:"requires,omitempty"`
	Produces []string `json:"produces,omitempty"`
}

func (d CapabilityDescriptor) Normalize() (CapabilityDescriptor, error) {
	if d.Schema == "" {
		d.Schema = CapabilitySchemaR0
	}
	if d.Schema != CapabilitySchemaR0 {
		return CapabilityDescriptor{}, fmt.Errorf("unexpected capability schema %q", d.Schema)
	}
	d.ID = strings.TrimSpace(d.ID)
	d.Capability = strings.ToUpper(strings.TrimSpace(d.Capability))
	d.Scope = strings.ToUpper(strings.TrimSpace(d.Scope))
	d.Domain = strings.ToUpper(strings.TrimSpace(d.Domain))
	d.Engine = strings.ToUpper(strings.TrimSpace(d.Engine))
	if d.ID == "" || d.Capability == "" || d.InputSchema == "" || d.OutputSchema == "" {
		return CapabilityDescriptor{}, fmt.Errorf("id, capability, input_schema and output_schema are required")
	}
	if d.Scope == "" {
		d.Scope = ScopeGeneral
	}
	if d.Scope != ScopeGeneral && d.Scope != ScopeSpecific {
		return CapabilityDescriptor{}, fmt.Errorf("unsupported scope %q", d.Scope)
	}
	if d.Scope == ScopeSpecific && d.Domain == "" {
		return CapabilityDescriptor{}, fmt.Errorf("specific worker %q requires domain", d.ID)
	}
	if d.Engine == "" {
		d.Engine = EngineModel
	}
	if d.MaxConcurrency <= 0 {
		d.MaxConcurrency = 1
	}
	if d.EmpiricalProfile != nil {
		profile, err := d.EmpiricalProfile.Normalize()
		if err != nil {
			return CapabilityDescriptor{}, fmt.Errorf("worker %q: %w", d.ID, err)
		}
		d.EmpiricalProfile = &profile
	}
	// R0 Dependencies intentionally retain their original ordering and spelling;
	// existing planners and generated artifacts may depend on that representation.
	d.Requires = normalizeDataContractList(d.Requires)
	d.Produces = normalizeDataContractList(d.Produces)
	return d, nil
}

func normalizeDataContractList(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
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

type CapabilityRequest struct {
	TaskID     string                     `json:"task_id"`
	NodeID     string                     `json:"node_id"`
	Input      json.RawMessage            `json:"input"`
	Context    map[string]json.RawMessage `json:"context,omitempty"`
	Blackboard *blackboard.Snapshot       `json:"blackboard,omitempty"`
}

type CapabilityResponse struct {
	WorkerID     string                   `json:"worker_id"`
	Output       json.RawMessage          `json:"output"`
	Confidence   float64                  `json:"confidence,omitempty"`
	Notes        string                   `json:"notes,omitempty"`
	Observations []blackboard.Observation `json:"observations,omitempty"`
}

type CapabilityWorker interface {
	Descriptor() CapabilityDescriptor
	Execute(context.Context, CapabilityRequest) (CapabilityResponse, error)
}

type SelectionRequest struct {
	Capability          string
	WorkerID            string
	ScopeHint           string
	DomainHint          string
	PreferDeterministic bool
	MaxParameters       int64
}

type Registry struct {
	mu        sync.RWMutex
	workers   map[string]CapabilityWorker
	specs     []CandidateSpecification
	selection SelectionStrategy
}

func NewRegistry() *Registry {
	return &Registry{
		workers:   map[string]CapabilityWorker{},
		specs:     defaultCandidateSpecifications(),
		selection: RankedSelectionStrategy{Scoring: DefaultScoringStrategy{}},
	}
}

func (r *Registry) SetSelectionStrategy(strategy SelectionStrategy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.selection = strategy
}

func (r *Registry) Register(worker CapabilityWorker) error {
	if worker == nil {
		return fmt.Errorf("worker is nil")
	}
	d, err := worker.Descriptor().Normalize()
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.workers[d.ID]; exists {
		return fmt.Errorf("worker %q already registered", d.ID)
	}
	r.workers[d.ID] = worker
	return nil
}

func (r *Registry) Get(id string) (CapabilityWorker, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	w, ok := r.workers[id]
	return w, ok
}

func (r *Registry) Descriptors() []CapabilityDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]CapabilityDescriptor, 0, len(r.workers))
	for _, w := range r.workers {
		d, err := w.Descriptor().Normalize()
		if err == nil {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// SelectResult is the domain-aware selection API. Select remains as the R0
// compatibility adapter for callers that still use (worker, error).
func (r *Registry) SelectResult(req SelectionRequest) Result[CapabilityWorker] {
	capability := strings.ToUpper(strings.TrimSpace(req.Capability))
	scopeHint := strings.ToUpper(strings.TrimSpace(req.ScopeHint))
	domainHint := strings.ToUpper(strings.TrimSpace(req.DomainHint))
	req.Capability = capability
	req.ScopeHint = scopeHint
	req.DomainHint = domainHint

	if scopeHint != "" && scopeHint != ScopeGeneral && scopeHint != ScopeSpecific {
		return DomainResult[CapabilityWorker](ResultInvalidRequest, nil, Diagnostic{Code: "UNSUPPORTED_SCOPE", Message: fmt.Sprintf("unsupported scope hint %q", scopeHint)})
	}
	if scopeHint == ScopeSpecific && domainHint == "" {
		return DomainResult[CapabilityWorker](ResultInvalidRequest, nil, Diagnostic{Code: "MISSING_DOMAIN", Message: "SPECIFIC scope requires domain hint"})
	}

	if req.WorkerID != "" {
		w, ok := r.Get(req.WorkerID)
		if !ok {
			return DomainResult[CapabilityWorker](ResultNoCandidate, nil, Diagnostic{Code: "PINNED_WORKER_NOT_REGISTERED", Message: fmt.Sprintf("pinned worker %q not registered", req.WorkerID)})
		}
		d, err := w.Descriptor().Normalize()
		if err != nil {
			return Failure[CapabilityWorker](err)
		}
		if capability != "" && d.Capability != capability {
			return DomainResult[CapabilityWorker](ResultNoCandidate, nil, Diagnostic{Code: "CAPABILITY_MISMATCH", Message: fmt.Sprintf("worker %q capability=%s, want %s", d.ID, d.Capability, capability)})
		}
		return Success[CapabilityWorker](w)
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
	return strategy.Select(candidates, req)
}

func (r *Registry) Select(req SelectionRequest) (CapabilityWorker, error) {
	result := r.SelectResult(req)
	if result.Err != nil {
		return nil, result.Err
	}
	if result.Code == ResultSuccess {
		return result.Value, nil
	}
	if len(result.Diagnostics) > 0 && result.Diagnostics[0].Message != "" {
		return nil, fmt.Errorf("%s", result.Diagnostics[0].Message)
	}
	return nil, fmt.Errorf("worker selection failed: %s", result.Code)
}
