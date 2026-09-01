package tlaloque

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const SwarmSchemaR0 = "tlaloc.tlaloque-swarm.r0"

type SwarmNode struct {
	ID                  string            `json:"id"`
	Capability          string            `json:"capability"`
	WorkerID            string            `json:"worker_id,omitempty"`
	ScopeHint           string            `json:"scope_hint,omitempty"`
	DomainHint          string            `json:"domain_hint,omitempty"`
	DependsOn           []string          `json:"depends_on,omitempty"`
	InputBindings       map[string]string `json:"input_bindings,omitempty"`
	PreferDeterministic bool              `json:"prefer_deterministic,omitempty"`
	MaxParameters       int64             `json:"max_parameters,omitempty"`
	JoinMode            string            `json:"join_mode,omitempty"`
	MinDependencies     int               `json:"min_dependencies,omitempty"`
}

type SwarmPlan struct {
	Schema      string      `json:"schema"`
	ID          string      `json:"id"`
	MaxParallel int         `json:"max_parallel,omitempty"`
	Nodes       []SwarmNode `json:"nodes"`
}

func (p SwarmPlan) Normalize() (SwarmPlan, error) {
	if p.Schema == "" {
		p.Schema = SwarmSchemaR0
	}
	if p.Schema != SwarmSchemaR0 {
		return SwarmPlan{}, fmt.Errorf("unexpected swarm schema %q", p.Schema)
	}
	p.ID = strings.TrimSpace(p.ID)
	if p.ID == "" {
		return SwarmPlan{}, fmt.Errorf("swarm id is required")
	}
	if p.MaxParallel <= 0 {
		p.MaxParallel = 1
	}
	if len(p.Nodes) == 0 {
		return SwarmPlan{}, fmt.Errorf("swarm requires at least one node")
	}

	ids := map[string]bool{}
	for i := range p.Nodes {
		n := &p.Nodes[i]
		n.ID = strings.TrimSpace(n.ID)
		n.Capability = strings.ToUpper(strings.TrimSpace(n.Capability))
		n.ScopeHint = strings.ToUpper(strings.TrimSpace(n.ScopeHint))
		n.DomainHint = strings.ToUpper(strings.TrimSpace(n.DomainHint))
		if n.ID == "" || n.Capability == "" {
			return SwarmPlan{}, fmt.Errorf("node id and capability are required")
		}
		if ids[n.ID] {
			return SwarmPlan{}, fmt.Errorf("duplicate node %q", n.ID)
		}
		ids[n.ID] = true

		if len(n.InputBindings) > 0 {
			normalized := make(map[string]string, len(n.InputBindings))
			for rawProduct, rawProvider := range n.InputBindings {
				product := strings.TrimSpace(rawProduct)
				provider := strings.TrimSpace(rawProvider)
				if product == "" || provider == "" {
					return SwarmPlan{}, fmt.Errorf("node %q input bindings require non-empty product and provider", n.ID)
				}
				if previous, exists := normalized[product]; exists && previous != provider {
					return SwarmPlan{}, fmt.Errorf("node %q product %q has conflicting providers %q and %q", n.ID, product, previous, provider)
				}
				normalized[product] = provider
			}
			n.InputBindings = normalized
		}

		rawJoinMode := strings.TrimSpace(n.JoinMode)
		joinMode, err := normalizeJoinMode(rawJoinMode)
		if err != nil {
			return SwarmPlan{}, fmt.Errorf("node %q: %w", n.ID, err)
		}
		// Preserve R0 serialization: an omitted join mode still means ALL and is
		// left omitted. Explicit R1 modes are canonicalized.
		if rawJoinMode != "" {
			n.JoinMode = string(joinMode)
		}
		if len(n.DependsOn) == 0 && joinMode != JoinAll {
			return SwarmPlan{}, fmt.Errorf("node %q cannot use %s without dependencies", n.ID, joinMode)
		}
		if joinMode == JoinQuorum {
			if n.MinDependencies <= 0 {
				n.MinDependencies = len(n.DependsOn)/2 + 1
			}
			if n.MinDependencies > len(n.DependsOn) {
				return SwarmPlan{}, fmt.Errorf("node %q quorum %d exceeds dependency count %d", n.ID, n.MinDependencies, len(n.DependsOn))
			}
		} else if n.MinDependencies != 0 {
			return SwarmPlan{}, fmt.Errorf("node %q min_dependencies is only valid with QUORUM", n.ID)
		}
	}

	for _, n := range p.Nodes {
		dependencySet := map[string]struct{}{}
		for _, dep := range n.DependsOn {
			if !ids[dep] {
				return SwarmPlan{}, fmt.Errorf("node %q depends on unknown node %q", n.ID, dep)
			}
			if dep == n.ID {
				return SwarmPlan{}, fmt.Errorf("node %q depends on itself", n.ID)
			}
			dependencySet[dep] = struct{}{}
		}
		for product, provider := range n.InputBindings {
			if !ids[provider] {
				return SwarmPlan{}, fmt.Errorf("node %q product %q binds unknown provider %q", n.ID, product, provider)
			}
			if provider == n.ID {
				return SwarmPlan{}, fmt.Errorf("node %q product %q binds itself", n.ID, product)
			}
			if _, ok := dependencySet[provider]; !ok {
				return SwarmPlan{}, fmt.Errorf("node %q product %q provider %q is not a dependency", n.ID, product, provider)
			}
		}
	}
	if err := validateAcyclic(p.Nodes); err != nil {
		return SwarmPlan{}, err
	}
	return p, nil
}

func validateAcyclic(nodes []SwarmNode) error {
	indegree := map[string]int{}
	children := map[string][]string{}
	for _, n := range nodes {
		indegree[n.ID] = len(n.DependsOn)
		for _, dep := range n.DependsOn {
			children[dep] = append(children[dep], n.ID)
		}
	}
	queue := []string{}
	for id, degree := range indegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	visited := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		visited++
		for _, child := range children[id] {
			indegree[child]--
			if indegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}
	if visited != len(nodes) {
		return fmt.Errorf("swarm plan contains a dependency cycle")
	}
	return nil
}

type NodeExecution struct {
	NodeID     string          `json:"node_id"`
	Capability string          `json:"capability"`
	WorkerID   string          `json:"worker_id"`
	State      NodeState       `json:"state,omitempty"`
	StartedAt  time.Time       `json:"started_at"`
	DurationMS int64           `json:"duration_ms"`
	Confidence float64         `json:"confidence,omitempty"`
	Output     json.RawMessage `json:"output,omitempty"`
	Error      string          `json:"error,omitempty"`
}

type SwarmReport struct {
	Schema            string                     `json:"schema"`
	PlanID            string                     `json:"plan_id"`
	TaskID            string                     `json:"task_id"`
	RunID             string                     `json:"run_id,omitempty"`
	MaxParallel       int                        `json:"max_parallel"`
	RegisteredWorkers int                        `json:"registered_workers"`
	ExecutedNodes     int                        `json:"executed_nodes"`
	PeakParallel      int                        `json:"peak_parallel"`
	DurationMS        int64                      `json:"duration_ms"`
	Succeeded         bool                       `json:"succeeded"`
	Nodes             []NodeExecution            `json:"nodes"`
	TerminalOutputs   map[string]json.RawMessage `json:"terminal_outputs,omitempty"`
}

type SwarmRunner struct {
	Registry   *Registry
	Blackboard *BlackboardRuntime
}

func (r SwarmRunner) Run(ctx context.Context, plan SwarmPlan, taskID string, input json.RawMessage) (SwarmReport, error) {
	plan, err := plan.Normalize()
	if err != nil {
		return SwarmReport{}, err
	}
	if r.Registry == nil {
		return SwarmReport{}, fmt.Errorf("registry is required")
	}
	if len(input) == 0 || !json.Valid(input) {
		return SwarmReport{}, fmt.Errorf("task input must be valid JSON")
	}
	if strings.TrimSpace(taskID) == "" {
		taskID = plan.ID + "-task"
	}
	bb := newRunBlackboardWriter(r.Blackboard, taskID)
	runID := ""
	if bb != nil {
		runID = bb.runID
	}

	start := time.Now()
	report := SwarmReport{
		Schema:            SwarmSchemaR0 + ".report",
		PlanID:            plan.ID,
		TaskID:            taskID,
		RunID:             runID,
		MaxParallel:       plan.MaxParallel,
		RegisteredWorkers: len(r.Registry.Descriptors()),
	}

	nodes := map[string]SwarmNode{}
	children := map[string][]string{}
	terminal := map[string]bool{}
	states := map[string]NodeState{}
	finishedDeps := map[string]int{}
	succeededDeps := map[string]int{}
	for _, n := range plan.Nodes {
		nodes[n.ID] = n
		terminal[n.ID] = true
		states[n.ID] = NodePending
		for _, dep := range n.DependsOn {
			children[dep] = append(children[dep], n.ID)
			terminal[dep] = false
		}
	}

	outputs := map[string]json.RawMessage{}
	executions := map[string]NodeExecution{}
	globalSem := make(chan struct{}, plan.MaxParallel)
	workerSems := map[string]chan struct{}{}
	var semMu sync.Mutex
	var stateMu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error
	active, peak := 0, 0

	getWorkerSem := func(id string, max int) chan struct{} {
		semMu.Lock()
		defer semMu.Unlock()
		if s, ok := workerSems[id]; ok {
			return s
		}
		if max <= 0 {
			max = 1
		}
		s := make(chan struct{}, max)
		workerSems[id] = s
		return s
	}

	setFirstErr := func(err error) {
		if err == nil {
			return
		}
		stateMu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		stateMu.Unlock()
	}

	transitionLocked := func(nodeID string, event NodeEvent) error {
		next, transitionErr := transitionNode(states[nodeID], event)
		if transitionErr != nil {
			return fmt.Errorf("node %s: %w", nodeID, transitionErr)
		}
		states[nodeID] = next
		return nil
	}

	var launch func(string)
	var finish func(string, NodeExecution, CapabilityResponse, error)

	finish = func(nodeID string, execReport NodeExecution, resp CapabilityResponse, runErr error) {
		stateMu.Lock()
		succeeded := runErr == nil
		event := NodeExecutionFailed
		if succeeded {
			event = NodeSucceeded
		}
		if transitionErr := transitionLocked(nodeID, event); transitionErr != nil {
			if runErr == nil {
				runErr = transitionErr
				succeeded = false
			}
		}
		execReport.State = states[nodeID]
		if succeeded {
			outputs[nodeID] = append(json.RawMessage(nil), resp.Output...)
		} else if execReport.Error == "" && runErr != nil {
			execReport.Error = runErr.Error()
		}
		executions[nodeID] = execReport

		ready := []string{}
		for _, child := range children[nodeID] {
			finishedDeps[child]++
			if succeeded {
				succeededDeps[child]++
			}
			if states[child] != NodePending {
				continue
			}
			decision := evaluateJoin(nodes[child], finishedDeps[child], succeededDeps[child])
			if decision.Ready {
				if transitionErr := transitionLocked(child, NodeDependenciesSatisfied); transitionErr != nil {
					if firstErr == nil {
						firstErr = transitionErr
					}
					continue
				}
				ready = append(ready, child)
			} else if decision.Impossible {
				if transitionErr := transitionLocked(child, NodeDependenciesImpossible); transitionErr == nil {
					executions[child] = NodeExecution{
						NodeID:     child,
						Capability: nodes[child].Capability,
						WorkerID:   "BLOCKED",
						State:      NodeBlocked,
						Error:      "dependency join cannot be satisfied",
					}
				}
			}
		}
		stateMu.Unlock()

		if runErr != nil {
			setFirstErr(fmt.Errorf("node %s: %w", nodeID, runErr))
		}
		sort.Strings(ready)
		for _, child := range ready {
			launch(child)
		}
	}

	launch = func(nodeID string) {
		n := nodes[nodeID]
		wg.Add(1)
		go func() {
			defer wg.Done()

			worker, selectErr := r.Registry.Select(SelectionRequest{
				Capability:          n.Capability,
				WorkerID:            n.WorkerID,
				ScopeHint:           n.ScopeHint,
				DomainHint:          n.DomainHint,
				PreferDeterministic: n.PreferDeterministic,
				MaxParameters:       n.MaxParameters,
			})
			if selectErr != nil {
				ex := NodeExecution{
					NodeID:     n.ID,
					Capability: n.Capability,
					WorkerID:   "UNRESOLVED",
					StartedAt:  time.Now().UTC(),
					Error:      selectErr.Error(),
				}
				if bbErr := bb.RecordNode(ex, nil); bbErr != nil {
					selectErr = fmt.Errorf("%v; blackboard: %w", selectErr, bbErr)
					ex.Error = selectErr.Error()
				}
				finish(n.ID, ex, CapabilityResponse{}, selectErr)
				return
			}
			desc, _ := worker.Descriptor().Normalize()
			wsem := getWorkerSem(desc.ID, desc.MaxConcurrency)

			stateMu.Lock()
			if transitionErr := transitionLocked(n.ID, NodeDispatched); transitionErr != nil {
				stateMu.Unlock()
				setFirstErr(transitionErr)
				return
			}
			stateMu.Unlock()

			select {
			case globalSem <- struct{}{}:
			case <-ctx.Done():
				ex := NodeExecution{NodeID: n.ID, Capability: n.Capability, WorkerID: desc.ID, StartedAt: time.Now().UTC(), Error: ctx.Err().Error()}
				finish(n.ID, ex, CapabilityResponse{}, ctx.Err())
				return
			}
			defer func() { <-globalSem }()
			select {
			case wsem <- struct{}{}:
			case <-ctx.Done():
				ex := NodeExecution{NodeID: n.ID, Capability: n.Capability, WorkerID: desc.ID, StartedAt: time.Now().UTC(), Error: ctx.Err().Error()}
				finish(n.ID, ex, CapabilityResponse{}, ctx.Err())
				return
			}
			defer func() { <-wsem }()

			stateMu.Lock()
			active++
			if active > peak {
				peak = active
			}
			depContext := map[string]json.RawMessage{}
			boundProviders := map[string]struct{}{}
			for product, provider := range n.InputBindings {
				if out, ok := outputs[provider]; ok {
					depContext[product] = append(json.RawMessage(nil), out...)
					boundProviders[provider] = struct{}{}
				}
			}
			for _, dep := range n.DependsOn {
				if _, bound := boundProviders[dep]; bound {
					continue
				}
				if out, ok := outputs[dep]; ok {
					depContext[dep] = append(json.RawMessage(nil), out...)
				}
			}
			stateMu.Unlock()

			nodeStart := time.Now()
			snapshot, snapshotErr := bb.Snapshot()
			var resp CapabilityResponse
			var runErr error
			if snapshotErr != nil {
				runErr = fmt.Errorf("blackboard snapshot: %w", snapshotErr)
			} else {
				resp, runErr = worker.Execute(ctx, CapabilityRequest{
					TaskID:     taskID,
					NodeID:     n.ID,
					Input:      input,
					Context:    depContext,
					Blackboard: snapshot,
				})
			}
			duration := time.Since(nodeStart)

			stateMu.Lock()
			active--
			stateMu.Unlock()

			execReport := NodeExecution{
				NodeID:     n.ID,
				Capability: n.Capability,
				WorkerID:   desc.ID,
				StartedAt:  nodeStart.UTC(),
				DurationMS: duration.Milliseconds(),
			}
			if runErr != nil {
				execReport.Error = runErr.Error()
			} else {
				execReport.Output = append(json.RawMessage(nil), resp.Output...)
				execReport.Confidence = resp.Confidence
			}
			if bbErr := bb.RecordNode(execReport, resp.Observations); bbErr != nil {
				if runErr == nil {
					runErr = fmt.Errorf("blackboard write: %w", bbErr)
				} else {
					runErr = fmt.Errorf("%v; blackboard write: %w", runErr, bbErr)
				}
				execReport.Error = runErr.Error()
			}
			finish(n.ID, execReport, resp, runErr)
		}()
	}

	roots := []string{}
	stateMu.Lock()
	for id, n := range nodes {
		if len(n.DependsOn) != 0 {
			continue
		}
		if transitionErr := transitionLocked(id, NodeDependenciesSatisfied); transitionErr != nil {
			stateMu.Unlock()
			return SwarmReport{}, transitionErr
		}
		roots = append(roots, id)
	}
	stateMu.Unlock()
	sort.Strings(roots)
	for _, root := range roots {
		launch(root)
	}
	wg.Wait()

	stateMu.Lock()
	for id, state := range states {
		if state != NodePending {
			continue
		}
		if transitionErr := transitionLocked(id, NodeDependenciesImpossible); transitionErr == nil {
			executions[id] = NodeExecution{
				NodeID:     id,
				Capability: nodes[id].Capability,
				WorkerID:   "BLOCKED",
				State:      NodeBlocked,
				Error:      "upstream dependencies did not complete",
			}
		}
	}
	stateMu.Unlock()

	report.PeakParallel = peak
	report.DurationMS = time.Since(start).Milliseconds()
	report.TerminalOutputs = map[string]json.RawMessage{}
	ids := make([]string, 0, len(plan.Nodes))
	for _, n := range plan.Nodes {
		ids = append(ids, n.ID)
	}
	sort.Strings(ids)

	allCompleted := true
	for _, id := range ids {
		if ex, ok := executions[id]; ok {
			report.Nodes = append(report.Nodes, ex)
			if !ex.StartedAt.IsZero() {
				report.ExecutedNodes++
			}
		}
		if states[id] != NodeCompleted {
			allCompleted = false
		}
		if terminal[id] {
			if out, ok := outputs[id]; ok {
				report.TerminalOutputs[id] = out
			}
		}
	}
	report.Succeeded = firstErr == nil && allCompleted
	return report, firstErr
}
