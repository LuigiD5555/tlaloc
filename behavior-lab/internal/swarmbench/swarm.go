package swarmbench

import (
	"context"
	"encoding/json"
	"fmt"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// Capability names for the five real Tlaloques.
const (
	CapabilityDetectIntent      = "DETECT_INTENT"
	CapabilityExtractEntity     = "EXTRACT_ENTITY"
	CapabilityResolveDateAmount = "RESOLVE_DATE_AMOUNT"
	CapabilityRoute             = "ROUTE"
	CapabilityVerify            = "VERIFY"
)

// DAG node ids for BuildFanInPlan. tlaloque.SwarmRunner keys a node's
// CapabilityRequest.Context by these — by dependency node id, not by
// capability name — so routeWorker and verifyWorker below read dependency
// output through these same constants rather than through duplicated string
// literals that could silently drift out of sync with the plan.
const (
	nodeIntent     = "intent"
	nodeEntity     = "entity"
	nodeDateNumber = "date-number"
	nodeRoute      = "route"
	nodeVerify     = FanInTerminalNode
)

// taskInput is what Execute marshals as every node's task input: the
// document text plus the reference date needed to resolve relative
// expressions. It is deliberately the only thing every worker receives
// besides its declared dependency context.
type taskInput struct {
	Text          string `json:"text"`
	ReferenceDate string `json:"reference_date"`
}

func decodeTaskInput(raw json.RawMessage) (taskInput, error) {
	var input taskInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return taskInput{}, fmt.Errorf("task input: %w", err)
	}
	return input, nil
}

// inProcessWorker adapts one pure worker function into a tlaloque.CapabilityWorker
// without any subprocess or HTTP round trip. It exists so the swarm's
// end-to-end behavior can be tested at the speed of a function call — the
// standalone PROCESS/HTTP_JSON binaries call the exact same swarmbench
// functions, so this is not a second implementation to drift from the real one.
type inProcessWorker struct {
	desc tlaloque.CapabilityDescriptor
	run  func(context.Context, tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error)
}

func (w inProcessWorker) Descriptor() tlaloque.CapabilityDescriptor { return w.desc }
func (w inProcessWorker) Execute(ctx context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	return w.run(ctx, req)
}

// IntentLogicFunc and EntityLogicFunc let a caller swap in a different
// classifier behind the same transport/descriptor — the exhaustive
// heuristic (IntentWorkerLogic/EntityWorkerLogic) or a profile calibrated
// against a real local model's observed failures (see lfm2vl_proxy.go).
type IntentLogicFunc func(text string) (intent string, confidence float64)
type EntityLogicFunc func(text string) (organization string, confidence float64)

func intentWorker(id string, parameterCount int64, logic IntentLogicFunc) tlaloque.CapabilityWorker {
	if logic == nil {
		logic = IntentWorkerLogic
	}
	desc := tlaloque.CapabilityDescriptor{
		ID: id, Capability: CapabilityDetectIntent, Scope: tlaloque.ScopeGeneral,
		Engine: tlaloque.EngineModel, InputSchema: "tlaloc.text.r0", OutputSchema: "tlaloc.intent.r0",
		ParameterCount: parameterCount, MaxConcurrency: 4,
	}
	return inProcessWorker{desc: desc, run: func(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
		input, err := decodeTaskInput(req.Input)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		// A missed lexicon cue is a wrong answer, not a transport failure — a
		// real small classifier would emit some (possibly wrong) label rather
		// than crash. Failing this node here would incorrectly block every
		// downstream node, including ones that never needed intent at all.
		intent, confidence := logic(input.Text)
		output, err := json.Marshal(struct {
			Intent string `json:"intent"`
		}{Intent: intent})
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		return tlaloque.CapabilityResponse{WorkerID: id, Output: output, Confidence: confidence}, nil
	}}
}

func entityWorker(id string, parameterCount int64, logic EntityLogicFunc) tlaloque.CapabilityWorker {
	if logic == nil {
		logic = EntityWorkerLogic
	}
	desc := tlaloque.CapabilityDescriptor{
		ID: id, Capability: CapabilityExtractEntity, Scope: tlaloque.ScopeGeneral,
		Engine: tlaloque.EngineModel, InputSchema: "tlaloc.text.r0", OutputSchema: "tlaloc.entity.r0",
		ParameterCount: parameterCount, MaxConcurrency: 4,
	}
	return inProcessWorker{desc: desc, run: func(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
		input, err := decodeTaskInput(req.Input)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		// Same reasoning as intentWorker: a missed gazetteer entry is a wrong
		// answer the swarm should score, not a failure that blocks route —
		// route never depends on organization in the first place.
		organization, confidence := logic(input.Text)
		output, err := json.Marshal(struct {
			Organization string `json:"organization"`
		}{Organization: organization})
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		return tlaloque.CapabilityResponse{WorkerID: id, Output: output, Confidence: confidence}, nil
	}}
}

func dateNumberWorker(id string) tlaloque.CapabilityWorker {
	desc := tlaloque.CapabilityDescriptor{
		ID: id, Capability: CapabilityResolveDateAmount, Scope: tlaloque.ScopeGeneral,
		Engine: tlaloque.EngineDeterministic, InputSchema: "tlaloc.text.r0", OutputSchema: "tlaloc.date-amount.r0",
		Deterministic: true, MaxConcurrency: 8,
	}
	return inProcessWorker{desc: desc, run: func(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
		input, err := decodeTaskInput(req.Input)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		dateISO, amountCents, err := DateNumberWorkerLogic(input.Text, input.ReferenceDate)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		output, err := json.Marshal(struct {
			DateISO     string `json:"date_iso"`
			AmountCents int64  `json:"amount_cents"`
		}{DateISO: dateISO, AmountCents: amountCents})
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		return tlaloque.CapabilityResponse{WorkerID: id, Output: output, Confidence: 1}, nil
	}}
}

// routeSources names the DAG nodes router reads its four inputs from. A
// single upstream node may serve more than one field (the baseline's
// "date-number" node carries both date_iso and amount_cents in one JSON
// blob); the decomposed plan instead points DateNode and AmountNode at two
// separate atoms. Decoding twice from the same raw bytes when they coincide
// is harmless — each decode only reads the one field it declares.
type routeSources struct {
	IntentNode, OrganizationNode, DateNode, AmountNode string
}

func routeWorker(id string, dependencies []string, sources routeSources) tlaloque.CapabilityWorker {
	desc := tlaloque.CapabilityDescriptor{
		ID: id, Capability: CapabilityRoute, Scope: tlaloque.ScopeGeneral,
		Engine: tlaloque.EngineDeterministic, InputSchema: "tlaloc.swarm-context.r0", OutputSchema: "tlaloc.fields.r0",
		Deterministic: true, MaxConcurrency: 8,
		Dependencies: dependencies,
	}
	return inProcessWorker{desc: desc, run: func(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
		input, err := decodeTaskInput(req.Input)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		var intentOut struct {
			Intent string `json:"intent"`
		}
		var organizationOut struct {
			Organization string `json:"organization"`
		}
		var dateOut struct {
			DateISO string `json:"date_iso"`
		}
		var amountOut struct {
			AmountCents int64 `json:"amount_cents"`
		}
		if raw, ok := req.Context[sources.IntentNode]; ok {
			_ = json.Unmarshal(raw, &intentOut)
		}
		if raw, ok := req.Context[sources.OrganizationNode]; ok {
			_ = json.Unmarshal(raw, &organizationOut)
		}
		if raw, ok := req.Context[sources.DateNode]; ok {
			_ = json.Unmarshal(raw, &dateOut)
		}
		if raw, ok := req.Context[sources.AmountNode]; ok {
			_ = json.Unmarshal(raw, &amountOut)
		}
		fields, err := RouteWorkerLogic(intentOut.Intent, organizationOut.Organization, amountOut.AmountCents, dateOut.DateISO, input.ReferenceDate)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		output, err := json.Marshal(fields)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		return tlaloque.CapabilityResponse{WorkerID: id, Output: output, Confidence: 1}, nil
	}}
}

func verifyWorker(id string) tlaloque.CapabilityWorker {
	desc := tlaloque.CapabilityDescriptor{
		ID: id, Capability: CapabilityVerify, Scope: tlaloque.ScopeGeneral,
		Engine: tlaloque.EngineDeterministic, InputSchema: "tlaloc.fields.r0", OutputSchema: "tlaloc.fields.r0",
		Deterministic: true, MaxConcurrency: 8, Dependencies: []string{CapabilityRoute},
	}
	return inProcessWorker{desc: desc, run: func(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
		input, err := decodeTaskInput(req.Input)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		var fields Fields
		if raw, ok := req.Context[nodeRoute]; ok {
			if err := json.Unmarshal(raw, &fields); err != nil {
				return tlaloque.CapabilityResponse{}, fmt.Errorf("verifier: router output: %w", err)
			}
		}
		corrected, _, err := VerifyWorkerLogic(fields, input.ReferenceDate)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		output, err := json.Marshal(corrected)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		return tlaloque.CapabilityResponse{WorkerID: id, Output: output, Confidence: 1}, nil
	}}
}

// BuildInProcessRegistry registers the five real Tlaloques against their
// in-process implementations, using the exhaustive heuristic lexicon/
// gazetteer for intent and entity. intentParameters and entityParameters let
// a caller declare the size the eventual real models will carry, for the
// --max-parameters planning path, without needing real weights yet.
func BuildInProcessRegistry(intentParameters, entityParameters int64) (*tlaloque.Registry, error) {
	return BuildInProcessRegistryWithLogic(intentParameters, entityParameters, nil, nil)
}

// BuildInProcessRegistryWithLogic is BuildInProcessRegistry with the intent
// and entity classifiers swapped out — e.g. for IntentWorkerLogicLFM2VLProxy
// / EntityWorkerLogicLFM2VLProxy, the profile calibrated against a real
// local model's observed failures (see lfm2vl_proxy.go). Passing nil for
// either keeps the exhaustive default for that capability.
func BuildInProcessRegistryWithLogic(intentParameters, entityParameters int64, intentLogic IntentLogicFunc, entityLogic EntityLogicFunc) (*tlaloque.Registry, error) {
	registry := tlaloque.NewRegistry()
	workers := []tlaloque.CapabilityWorker{
		intentWorker("intent-lexicon-r0", intentParameters, intentLogic),
		entityWorker("entity-gazetteer-r0", entityParameters, entityLogic),
		dateNumberWorker("date-number-r0"),
		routeWorker("router-r0",
			[]string{CapabilityDetectIntent, CapabilityExtractEntity, CapabilityResolveDateAmount},
			routeSources{IntentNode: nodeIntent, OrganizationNode: nodeEntity, DateNode: nodeDateNumber, AmountNode: nodeDateNumber},
		),
		verifyWorker("verifier-r0"),
	}
	for _, worker := range workers {
		if err := registry.Register(worker); err != nil {
			return nil, fmt.Errorf("build in-process registry: %w", err)
		}
	}
	return registry, nil
}

// BuildFanInPlan wires the five Tlaloques into the wide, shallow topology the
// design rule recommends: intent, entity and date-number run independently
// (depth 1), router fans them in (depth 2), verifier closes the chain
// (depth 3). Its own terminal node id is "verify" — pass that to Execute.
func BuildFanInPlan(planID string, maxParallel int) tlaloque.SwarmPlan {
	return tlaloque.SwarmPlan{
		ID: planID, MaxParallel: maxParallel,
		Nodes: []tlaloque.SwarmNode{
			{ID: nodeIntent, Capability: CapabilityDetectIntent, WorkerID: "intent-lexicon-r0"},
			{ID: nodeEntity, Capability: CapabilityExtractEntity, WorkerID: "entity-gazetteer-r0"},
			{ID: nodeDateNumber, Capability: CapabilityResolveDateAmount, WorkerID: "date-number-r0"},
			{ID: nodeRoute, Capability: CapabilityRoute, WorkerID: "router-r0", DependsOn: []string{nodeIntent, nodeEntity, nodeDateNumber}},
			{ID: nodeVerify, Capability: CapabilityVerify, WorkerID: "verifier-r0", DependsOn: []string{nodeRoute}},
		},
	}
}

// FanInTerminalNode is the sink Execute must score against.
const FanInTerminalNode = "verify"
