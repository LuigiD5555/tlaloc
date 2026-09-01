package swarmbench

import (
	"context"
	"encoding/json"
	"fmt"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// This file is the genuine-decomposition counterpart to replication.go: it
// splits EXTRACT_ENTITY into three narrower atoms (org-head, org-tail,
// join-organization) and RESOLVE_DATE_AMOUNT into its two already-independent
// deterministic atoms (date, amount) — raising the real population from 5 to
// 8 individuals by dividing actual work, not by copying the same swarm.

const (
	CapabilityExtractOrgHead   = "EXTRACT_ORG_HEAD"
	CapabilityExtractOrgTail   = "EXTRACT_ORG_TAIL"
	CapabilityJoinOrganization = "JOIN_ORGANIZATION"
	CapabilityResolveDate      = "RESOLVE_DATE"
	CapabilityResolveAmount    = "RESOLVE_AMOUNT"
)

const (
	nodeOrgHead    = "org-head"
	nodeOrgTail    = "org-tail"
	nodeJoinOrg    = "join-organization"
	nodeDateAtom   = "date"
	nodeAmountAtom = "amount"
)

func orgHeadWorker(id string, parameterCount int64) tlaloque.CapabilityWorker {
	desc := tlaloque.CapabilityDescriptor{
		ID: id, Capability: CapabilityExtractOrgHead, Scope: tlaloque.ScopeGeneral,
		Engine: tlaloque.EngineModel, InputSchema: "tlaloc.text.r0", OutputSchema: "tlaloc.org-head.r0",
		ParameterCount: parameterCount, MaxConcurrency: 4,
	}
	return inProcessWorker{desc: desc, run: func(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
		input, err := decodeTaskInput(req.Input)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		head, confidence := OrgHeadWorkerLogic(input.Text)
		output, err := json.Marshal(struct {
			Head string `json:"head"`
		}{Head: head})
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		return tlaloque.CapabilityResponse{WorkerID: id, Output: output, Confidence: confidence}, nil
	}}
}

func orgTailWorker(id string, parameterCount int64) tlaloque.CapabilityWorker {
	desc := tlaloque.CapabilityDescriptor{
		ID: id, Capability: CapabilityExtractOrgTail, Scope: tlaloque.ScopeGeneral,
		Engine: tlaloque.EngineModel, InputSchema: "tlaloc.text.r0", OutputSchema: "tlaloc.org-tail.r0",
		ParameterCount: parameterCount, MaxConcurrency: 4,
	}
	return inProcessWorker{desc: desc, run: func(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
		input, err := decodeTaskInput(req.Input)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		tail, confidence := OrgTailWorkerLogic(input.Text)
		output, err := json.Marshal(struct {
			Tail string `json:"tail"`
		}{Tail: tail})
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		return tlaloque.CapabilityResponse{WorkerID: id, Output: output, Confidence: confidence}, nil
	}}
}

func joinOrganizationWorker(id string) tlaloque.CapabilityWorker {
	desc := tlaloque.CapabilityDescriptor{
		ID: id, Capability: CapabilityJoinOrganization, Scope: tlaloque.ScopeGeneral,
		Engine: tlaloque.EngineDeterministic, InputSchema: "tlaloc.swarm-context.r0", OutputSchema: "tlaloc.entity.r0",
		Deterministic: true, MaxConcurrency: 8,
		Dependencies: []string{CapabilityExtractOrgHead, CapabilityExtractOrgTail},
	}
	return inProcessWorker{desc: desc, run: func(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
		var headOut struct {
			Head string `json:"head"`
		}
		var tailOut struct {
			Tail string `json:"tail"`
		}
		if raw, ok := req.Context[nodeOrgHead]; ok {
			_ = json.Unmarshal(raw, &headOut)
		}
		if raw, ok := req.Context[nodeOrgTail]; ok {
			_ = json.Unmarshal(raw, &tailOut)
		}
		organization := JoinOrganizationWorkerLogic(headOut.Head, tailOut.Tail)
		output, err := json.Marshal(struct {
			Organization string `json:"organization"`
		}{Organization: organization})
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		return tlaloque.CapabilityResponse{WorkerID: id, Output: output, Confidence: 1}, nil
	}}
}

func dateAtomWorker(id string) tlaloque.CapabilityWorker {
	desc := tlaloque.CapabilityDescriptor{
		ID: id, Capability: CapabilityResolveDate, Scope: tlaloque.ScopeGeneral,
		Engine: tlaloque.EngineDeterministic, InputSchema: "tlaloc.text.r0", OutputSchema: "tlaloc.date.r0",
		Deterministic: true, MaxConcurrency: 8,
	}
	return inProcessWorker{desc: desc, run: func(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
		input, err := decodeTaskInput(req.Input)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		dateISO, err := ExtractDate(input.Text, input.ReferenceDate)
		if err != nil {
			return tlaloque.CapabilityResponse{}, fmt.Errorf("date: %w", err)
		}
		output, err := json.Marshal(struct {
			DateISO string `json:"date_iso"`
		}{DateISO: dateISO})
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		return tlaloque.CapabilityResponse{WorkerID: id, Output: output, Confidence: 1}, nil
	}}
}

func amountAtomWorker(id string) tlaloque.CapabilityWorker {
	desc := tlaloque.CapabilityDescriptor{
		ID: id, Capability: CapabilityResolveAmount, Scope: tlaloque.ScopeGeneral,
		Engine: tlaloque.EngineDeterministic, InputSchema: "tlaloc.text.r0", OutputSchema: "tlaloc.amount.r0",
		Deterministic: true, MaxConcurrency: 8,
	}
	return inProcessWorker{desc: desc, run: func(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
		input, err := decodeTaskInput(req.Input)
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		amountCents, err := ExtractAmount(input.Text)
		if err != nil {
			return tlaloque.CapabilityResponse{}, fmt.Errorf("amount: %w", err)
		}
		output, err := json.Marshal(struct {
			AmountCents int64 `json:"amount_cents"`
		}{AmountCents: amountCents})
		if err != nil {
			return tlaloque.CapabilityResponse{}, err
		}
		return tlaloque.CapabilityResponse{WorkerID: id, Output: output, Confidence: 1}, nil
	}}
}

// BuildDecomposedRegistry registers eight real Tlaloques: intent (unchanged),
// org-head + org-tail + join-organization (the genuine decomposition of
// entity), date + amount (the already-independent halves of date-number),
// router (now fed by four upstream atoms instead of three) and verifier.
func BuildDecomposedRegistry(intentParameters, orgHeadParameters, orgTailParameters int64) (*tlaloque.Registry, error) {
	registry := tlaloque.NewRegistry()
	workers := []tlaloque.CapabilityWorker{
		intentWorker("intent-lexicon-r0", intentParameters, nil),
		orgHeadWorker("org-head-r0", orgHeadParameters),
		orgTailWorker("org-tail-r0", orgTailParameters),
		joinOrganizationWorker("join-organization-r0"),
		dateAtomWorker("date-r0"),
		amountAtomWorker("amount-r0"),
		routeWorker("router-decomposed-r0",
			[]string{CapabilityDetectIntent, CapabilityJoinOrganization, CapabilityResolveDate, CapabilityResolveAmount},
			routeSources{IntentNode: nodeIntent, OrganizationNode: nodeJoinOrg, DateNode: nodeDateAtom, AmountNode: nodeAmountAtom},
		),
		verifyWorker("verifier-decomposed-r0"),
	}
	for _, worker := range workers {
		if err := registry.Register(worker); err != nil {
			return nil, fmt.Errorf("build decomposed registry: %w", err)
		}
	}
	return registry, nil
}

// BuildDecomposedPlan wires the eight atoms into a still-wide, still-shallow
// DAG: intent, org-head, org-tail, date and amount all run independently
// (depth 1, width 5); join-organization fans in the two organization halves
// (depth 2); router fans in intent/organization/date/amount (depth 3);
// verifier closes the chain (depth 4). Terminal node id is "verify".
func BuildDecomposedPlan(planID string, maxParallel int) tlaloque.SwarmPlan {
	return tlaloque.SwarmPlan{
		ID: planID, MaxParallel: maxParallel,
		Nodes: []tlaloque.SwarmNode{
			{ID: nodeIntent, Capability: CapabilityDetectIntent, WorkerID: "intent-lexicon-r0"},
			{ID: nodeOrgHead, Capability: CapabilityExtractOrgHead, WorkerID: "org-head-r0"},
			{ID: nodeOrgTail, Capability: CapabilityExtractOrgTail, WorkerID: "org-tail-r0"},
			{ID: nodeJoinOrg, Capability: CapabilityJoinOrganization, WorkerID: "join-organization-r0", DependsOn: []string{nodeOrgHead, nodeOrgTail}},
			{ID: nodeDateAtom, Capability: CapabilityResolveDate, WorkerID: "date-r0"},
			{ID: nodeAmountAtom, Capability: CapabilityResolveAmount, WorkerID: "amount-r0"},
			{ID: nodeRoute, Capability: CapabilityRoute, WorkerID: "router-decomposed-r0", DependsOn: []string{nodeIntent, nodeJoinOrg, nodeDateAtom, nodeAmountAtom}},
			{ID: nodeVerify, Capability: CapabilityVerify, WorkerID: "verifier-decomposed-r0", DependsOn: []string{nodeRoute}},
		},
	}
}
