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
	ID                  string   `json:"id"`
	Capability          string   `json:"capability"`
	WorkerID            string   `json:"worker_id,omitempty"`
	ScopeHint           string   `json:"scope_hint,omitempty"`
	DomainHint          string   `json:"domain_hint,omitempty"`
	DependsOn           []string `json:"depends_on,omitempty"`
	PreferDeterministic bool     `json:"prefer_deterministic,omitempty"`
	MaxParameters       int64    `json:"max_parameters,omitempty"`
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
	}
	for _, n := range p.Nodes {
		for _, dep := range n.DependsOn {
			if !ids[dep] {
				return SwarmPlan{}, fmt.Errorf("node %q depends on unknown node %q", n.ID, dep)
			}
			if dep == n.ID {
				return SwarmPlan{}, fmt.Errorf("node %q depends on itself", n.ID)
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
		if degree == 0 { queue = append(queue, id) }
	}
	visited := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		visited++
		for _, child := range children[id] {
			indegree[child]--
			if indegree[child] == 0 { queue = append(queue, child) }
		}
	}
	if visited != len(nodes) {
		return fmt.Errorf("swarm plan contains a dependency cycle")
	}
	return nil
}

type NodeExecution struct {
	NodeID       string          `json:"node_id"`
	Capability   string          `json:"capability"`
	WorkerID     string          `json:"worker_id"`
	StartedAt    time.Time       `json:"started_at"`
	DurationMS   int64           `json:"duration_ms"`
	Confidence   float64         `json:"confidence,omitempty"`
	Output       json.RawMessage `json:"output,omitempty"`
	Error        string          `json:"error,omitempty"`
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
	if err != nil { return SwarmReport{}, err }
	if r.Registry == nil { return SwarmReport{}, fmt.Errorf("registry is required") }
	if len(input) == 0 || !json.Valid(input) { return SwarmReport{}, fmt.Errorf("task input must be valid JSON") }
	if strings.TrimSpace(taskID) == "" { taskID = plan.ID + "-task" }
	bb := newRunBlackboardWriter(r.Blackboard, taskID)
	runID := ""
	if bb != nil { runID = bb.runID }

	start := time.Now()
	report := SwarmReport{Schema: SwarmSchemaR0 + ".report", PlanID: plan.ID, TaskID: taskID, RunID: runID, MaxParallel: plan.MaxParallel, RegisteredWorkers: len(r.Registry.Descriptors())}

	nodes := map[string]SwarmNode{}
	remaining := map[string]int{}
	children := map[string][]string{}
	terminal := map[string]bool{}
	for _, n := range plan.Nodes {
		nodes[n.ID] = n
		remaining[n.ID] = len(n.DependsOn)
		terminal[n.ID] = true
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
		semMu.Lock(); defer semMu.Unlock()
		if s, ok := workerSems[id]; ok { return s }
		if max <= 0 { max = 1 }
		s := make(chan struct{}, max)
		workerSems[id] = s
		return s
	}

	setFirstErr := func(err error) {
		if err == nil { return }
		stateMu.Lock()
		if firstErr == nil { firstErr = err }
		stateMu.Unlock()
	}

	var launch func(string)
	launch = func(nodeID string) {
		n := nodes[nodeID]
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker, selectErr := r.Registry.Select(SelectionRequest{Capability: n.Capability, WorkerID: n.WorkerID, ScopeHint: n.ScopeHint, DomainHint: n.DomainHint, PreferDeterministic: n.PreferDeterministic, MaxParameters: n.MaxParameters})
			if selectErr != nil {
				ex := NodeExecution{NodeID:n.ID, Capability:n.Capability, WorkerID:"UNRESOLVED", StartedAt:time.Now().UTC(), Error:selectErr.Error()}
				if bbErr := bb.RecordNode(ex, nil); bbErr != nil { selectErr = fmt.Errorf("%v; blackboard: %w", selectErr, bbErr); ex.Error = selectErr.Error() }
				stateMu.Lock(); executions[n.ID] = ex; stateMu.Unlock()
				setFirstErr(fmt.Errorf("node %s: %w", n.ID, selectErr))
				return
			}
			desc, _ := worker.Descriptor().Normalize()
			wsem := getWorkerSem(desc.ID, desc.MaxConcurrency)

			select { case globalSem <- struct{}{}: case <-ctx.Done(): setFirstErr(ctx.Err()); return }
			defer func(){ <-globalSem }()
			select { case wsem <- struct{}{}: case <-ctx.Done(): setFirstErr(ctx.Err()); return }
			defer func(){ <-wsem }()

			stateMu.Lock()
			active++
			if active > peak { peak = active }
			depContext := map[string]json.RawMessage{}
			for _, dep := range n.DependsOn {
				if out, ok := outputs[dep]; ok { depContext[dep] = append(json.RawMessage(nil), out...) }
			}
			stateMu.Unlock()

			nodeStart := time.Now()
			snapshot, snapshotErr := bb.Snapshot()
			var resp CapabilityResponse
			var runErr error
			if snapshotErr != nil {
				runErr = fmt.Errorf("blackboard snapshot: %w", snapshotErr)
			} else {
				resp, runErr = worker.Execute(ctx, CapabilityRequest{TaskID: taskID, NodeID: n.ID, Input: input, Context: depContext, Blackboard: snapshot})
			}
			duration := time.Since(nodeStart)

			stateMu.Lock()
			active--
			stateMu.Unlock()
			execReport := NodeExecution{NodeID: n.ID, Capability: n.Capability, WorkerID: desc.ID, StartedAt: nodeStart.UTC(), DurationMS: duration.Milliseconds()}
			if runErr != nil {
				execReport.Error = runErr.Error()
			} else {
				execReport.Output = append(json.RawMessage(nil), resp.Output...)
				execReport.Confidence = resp.Confidence
			}
			if bbErr := bb.RecordNode(execReport, resp.Observations); bbErr != nil {
				if runErr == nil { runErr = fmt.Errorf("blackboard write: %w", bbErr) } else { runErr = fmt.Errorf("%v; blackboard write: %w", runErr, bbErr) }
				execReport.Error = runErr.Error()
			}

			stateMu.Lock()
			executions[n.ID] = execReport
			ready := []string{}
			if runErr == nil {
				outputs[n.ID] = append(json.RawMessage(nil), resp.Output...)
				for _, child := range children[n.ID] { remaining[child]-- }
				for _, child := range children[n.ID] {
					if remaining[child] == 0 { ready = append(ready, child) }
				}
			} else if firstErr == nil {
				firstErr = fmt.Errorf("node %s worker %s: %w", n.ID, desc.ID, runErr)
			}
			stateMu.Unlock()
			sort.Strings(ready)
			for _, child := range ready { launch(child) }
		}()
	}

	roots := []string{}
	for id, degree := range remaining { if degree == 0 { roots = append(roots, id) } }
	sort.Strings(roots)
	for _, root := range roots { launch(root) }
	wg.Wait()

	report.PeakParallel = peak
	report.DurationMS = time.Since(start).Milliseconds()
	report.TerminalOutputs = map[string]json.RawMessage{}
	ids := make([]string, 0, len(plan.Nodes))
	for _, n := range plan.Nodes { ids = append(ids, n.ID) }
	sort.Strings(ids)
	for _, id := range ids {
		if ex, ok := executions[id]; ok { report.Nodes = append(report.Nodes, ex) }
		if terminal[id] {
			if out, ok := outputs[id]; ok { report.TerminalOutputs[id] = out }
		}
	}
	report.ExecutedNodes = len(report.Nodes)
	report.Succeeded = firstErr == nil && report.ExecutedNodes == len(plan.Nodes)
	return report, firstErr
}
