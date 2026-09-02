package tlaloque

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

type testWorker struct {
	desc   CapabilityDescriptor
	delay  time.Duration
	fn     func(CapabilityRequest) json.RawMessage
	active *int32
	peak   *int32
}

func (w testWorker) Descriptor() CapabilityDescriptor { return w.desc }
func (w testWorker) Execute(ctx context.Context, req CapabilityRequest) (CapabilityResponse, error) {
	if w.active != nil {
		n := atomic.AddInt32(w.active, 1)
		for {
			p := atomic.LoadInt32(w.peak)
			if n <= p || atomic.CompareAndSwapInt32(w.peak, p, n) {
				break
			}
		}
		defer atomic.AddInt32(w.active, -1)
	}
	if w.delay > 0 {
		select {
		case <-time.After(w.delay):
		case <-ctx.Done():
			return CapabilityResponse{}, ctx.Err()
		}
	}
	out := json.RawMessage(`{"ok":true}`)
	if w.fn != nil {
		out = w.fn(req)
	}
	return CapabilityResponse{WorkerID: w.desc.ID, Output: out, Confidence: 0.9}, nil
}

func TestRegistryPrefersGeneralWithoutDomainAndSpecificWithDomain(t *testing.T) {
	r := NewRegistry()
	general := testWorker{desc: CapabilityDescriptor{ID: "entity-general", Capability: "EXTRACT_ENTITY", Scope: ScopeGeneral, Engine: EngineModel, InputSchema: "text", OutputSchema: "entities", ParameterCount: 20_000_000}}
	specific := testWorker{desc: CapabilityDescriptor{ID: "entity-cfdi", Capability: "EXTRACT_ENTITY", Scope: ScopeSpecific, Domain: "CFDI", Engine: EngineModel, InputSchema: "text", OutputSchema: "entities", ParameterCount: 5_000_000}}
	if err := r.Register(general); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(specific); err != nil {
		t.Fatal(err)
	}
	w, err := r.Select(SelectionRequest{Capability: "EXTRACT_ENTITY"})
	if err != nil {
		t.Fatal(err)
	}
	if w.Descriptor().ID != "entity-general" {
		t.Fatalf("got %s", w.Descriptor().ID)
	}
	w, err = r.Select(SelectionRequest{Capability: "EXTRACT_ENTITY", ScopeHint: ScopeSpecific, DomainHint: "CFDI"})
	if err != nil {
		t.Fatal(err)
	}
	if w.Descriptor().ID != "entity-cfdi" {
		t.Fatalf("got %s", w.Descriptor().ID)
	}
}

func TestSwarmRunsIndependentNodesInParallelAndPassesDependencyContext(t *testing.T) {
	var active, peak int32
	r := NewRegistry()
	mk := func(id, capability string) testWorker {
		return testWorker{desc: CapabilityDescriptor{ID: id, Capability: capability, Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "json", OutputSchema: "json", Deterministic: true, MaxConcurrency: 1}, delay: 20 * time.Millisecond, active: &active, peak: &peak}
	}
	intent := mk("intent", "DETECT_INTENT")
	intent.fn = func(CapabilityRequest) json.RawMessage { return json.RawMessage(`{"intent":"SEARCH"}`) }
	entity := mk("entity", "EXTRACT_ENTITY")
	entity.fn = func(CapabilityRequest) json.RawMessage { return json.RawMessage(`{"entity":"PEMEX"}`) }
	router := mk("router", "ROUTE")
	router.fn = func(req CapabilityRequest) json.RawMessage {
		if len(req.Context["intent"]) == 0 || len(req.Context["entity"]) == 0 {
			panic("missing dependency context")
		}
		return json.RawMessage(`{"route":"documents"}`)
	}
	for _, w := range []CapabilityWorker{intent, entity, router} {
		if err := r.Register(w); err != nil {
			t.Fatal(err)
		}
	}
	plan := SwarmPlan{ID: "document-router", MaxParallel: 2, Nodes: []SwarmNode{
		{ID: "intent", Capability: "DETECT_INTENT", PreferDeterministic: true},
		{ID: "entity", Capability: "EXTRACT_ENTITY", PreferDeterministic: true},
		{ID: "route", Capability: "ROUTE", DependsOn: []string{"intent", "entity"}, PreferDeterministic: true},
	}}
	report, err := (SwarmRunner{Registry: r}).Run(context.Background(), plan, "task-1", json.RawMessage(`{"text":"find Pemex"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !report.Succeeded {
		t.Fatalf("report=%+v", report)
	}
	if report.ExecutedNodes != 3 {
		t.Fatalf("executed=%d", report.ExecutedNodes)
	}
	if report.PeakParallel < 2 || atomic.LoadInt32(&peak) < 2 {
		t.Fatalf("peak=%d workerPeak=%d", report.PeakParallel, peak)
	}
	if string(report.TerminalOutputs["route"]) != `{"route":"documents"}` {
		t.Fatalf("terminal=%s", report.TerminalOutputs["route"])
	}
}

func TestSwarmRejectsCycle(t *testing.T) {
	_, err := (SwarmPlan{ID: "bad", Nodes: []SwarmNode{{ID: "a", Capability: "A", DependsOn: []string{"b"}}, {ID: "b", Capability: "B", DependsOn: []string{"a"}}}}).Normalize()
	if err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestCompositeWorkerActsAsOneCapability(t *testing.T) {
	r := NewRegistry()
	atomic := testWorker{desc: CapabilityDescriptor{ID: "atom", Capability: "ATOM", Scope: ScopeGeneral, Engine: EngineDeterministic, InputSchema: "json", OutputSchema: "json", Deterministic: true}, fn: func(CapabilityRequest) json.RawMessage { return json.RawMessage(`{"value":1}`) }}
	if err := r.Register(atomic); err != nil {
		t.Fatal(err)
	}
	composite := CompositeWorker{Desc: CapabilityDescriptor{ID: "composite", Capability: "COMPLEX", Scope: ScopeGeneral, Engine: "COMPOSITE", InputSchema: "json", OutputSchema: "json", Deterministic: true}, Plan: SwarmPlan{ID: "sub", Nodes: []SwarmNode{{ID: "a", Capability: "ATOM"}}}, Registry: r}
	resp, err := composite.Execute(context.Background(), CapabilityRequest{TaskID: "t", NodeID: "n", Input: json.RawMessage(`{"x":1}`)})
	if err != nil {
		t.Fatal(err)
	}
	if resp.WorkerID != "composite" || !json.Valid(resp.Output) {
		t.Fatalf("resp=%+v", resp)
	}
	if len(resp.Output) == 0 {
		t.Fatal(fmt.Errorf("empty output"))
	}
}
