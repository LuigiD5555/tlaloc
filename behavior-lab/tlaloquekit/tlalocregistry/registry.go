// Package tlalocregistry wires a concrete Tlaloc-published
// QualifiedRegistry. It is the only tlaloquekit package that imports
// tlaloc.local/behaviorlab/internal/* — the public contract package
// tlaloquekit stays dependency-free so a consuming runtime can import the
// types and interface without pulling in the engine.
package tlalocregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	kit "tlaloc.local/behaviorlab/tlaloquekit"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/exocortex"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// Config selects which qualified Tlaloques the registry publishes. The
// zero value publishes the full deterministic set and no generative
// Tlaloque.
type Config struct {
	// OmitDeterministic, when set, skips registration of the deterministic
	// Tlaloque set. It exists only for tests that want an empty registry.
	OmitDeterministic bool

	// Parrot, when set, registers the R1-aware Parrot Tlaloque. It is the
	// caller's decision to make the generative executor available; there
	// is no implicit capability -> Parrot fallback.
	Parrot *ParrotConfig
}

// ParrotConfig is the frozen-profile-gated configuration for the R1-aware
// Parrot Tlaloque.
type ParrotConfig struct {
	// ProfilePath is the frozen CapabilityProfile R1 document
	// (profiles/parrot-lfm2-vl-1.6b-r1.json).
	ProfilePath string
	// ExpectedProfileHash is the profile hash the caller demands, e.g.
	// "8acc959b...". Loading fails if the on-disk profile does not match.
	ExpectedProfileHash string
	// Endpoint is the OpenAI-compatible transport for the model.
	Endpoint kit.ParrotEndpoint
	// WorkDir is where prepared operand crops are written; defaults to the
	// process temp dir.
	WorkDir string
}

// BuildQualifiedRegistry constructs the registry Tlaloc publishes to a
// consuming runtime. It wires the internal tlaloque.Registry, installs the
// CapabilityProfile-aware CapabilityRouter, and registers the qualified
// Tlaloque set behind the public QualifiedRegistry contract.
func BuildQualifiedRegistry(cfg Config) (kit.QualifiedRegistry, error) {
	registry := tlaloque.NewRegistry()
	out := &kitRegistry{registry: registry, parrotWorkers: map[string]string{}}

	if !cfg.OmitDeterministic {
		deterministic := []tlaloque.CapabilityWorker{
			exocortex.RegionLocateTlaloque{},
			exocortex.RegionCropTlaloque{},
			exocortex.NormalizeTlaloque{},
			exocortex.NumericTlaloque{},
			exocortex.ArithmeticTlaloque{},
			exocortex.VerifyTlaloque{},
		}
		for _, worker := range deterministic {
			if err := registry.Register(worker); err != nil {
				return nil, fmt.Errorf("tlalocregistry: register %s: %w", worker.Descriptor().ID, err)
			}
		}
	}

	if cfg.Parrot != nil {
		workers, profileID, profileHash, err := exocortex.NewParrotTlaloqueR1(
			cfg.Parrot.ProfilePath, cfg.Parrot.ExpectedProfileHash, cfg.Parrot.WorkDir,
			exocortex.ParrotEndpoint{
				BaseURL:     cfg.Parrot.Endpoint.BaseURL,
				Model:       cfg.Parrot.Endpoint.Model,
				Temperature: cfg.Parrot.Endpoint.Temperature,
				MaxTokens:   cfg.Parrot.Endpoint.MaxTokens,
			})
		if err != nil {
			return nil, fmt.Errorf("tlalocregistry: parrot r1: %w", err)
		}
		for _, worker := range workers {
			if err := registry.Register(worker); err != nil {
				return nil, fmt.Errorf("tlalocregistry: register %s: %w", worker.Descriptor().ID, err)
			}
			out.parrotWorkers[worker.Descriptor().ID] = profileID
		}
		out.parrotProfileID = profileID
		out.parrotProfileHash = profileHash
	}

	// The router keeps the registry's deterministic-first, smallest-first
	// ranking. The R1-aware Parrot Tlaloque only registers for opcodes the
	// R1 profile covers, so there is no implicit capability -> Parrot
	// fallback and no lossy R1 -> R0 profile conversion is needed.
	exocortex.CapabilityRouter{Profiles: map[string]exocortex.CapabilityProfile{}}.Install(registry)

	return out, nil
}

type kitRegistry struct {
	registry          *tlaloque.Registry
	parrotWorkers     map[string]string // worker id -> profile id
	parrotProfileID   string
	parrotProfileHash string
}

func (k *kitRegistry) ParrotProfileID() string   { return k.parrotProfileID }
func (k *kitRegistry) ParrotProfileHash() string { return k.parrotProfileHash }

func (k *kitRegistry) Capabilities() []kit.Descriptor {
	internal := k.registry.Descriptors()
	result := make([]kit.Descriptor, 0, len(internal))
	for _, descriptor := range internal {
		result = append(result, k.toPublicDescriptor(descriptor))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (k *kitRegistry) Candidates(capability string, goal kit.Goal) []kit.Candidate {
	capability = strings.ToUpper(strings.TrimSpace(capability))
	var matching []tlaloque.CapabilityDescriptor
	for _, descriptor := range k.registry.Descriptors() {
		if descriptor.Capability == capability {
			matching = append(matching, descriptor)
		}
	}
	sort.Slice(matching, func(i, j int) bool { return matching[i].ID < matching[j].ID })

	result := k.registry.SelectResult(tlaloque.SelectionRequest{
		Capability:          capability,
		PreferDeterministic: goal.PreferDeterministic,
		MaxParameters:       goal.MaxParameters,
	})
	winnerID, rejection := "", ""
	if result.OK() {
		winnerID = result.Value.Descriptor().ID
	} else if len(result.Diagnostics) > 0 {
		rejection = result.Diagnostics[0].Code + ": " + result.Diagnostics[0].Message
	} else {
		rejection = string(result.Code)
	}

	out := make([]kit.Candidate, 0, len(matching))
	for _, descriptor := range matching {
		candidate := kit.Candidate{Descriptor: k.toPublicDescriptor(descriptor)}
		switch {
		case descriptor.ID == winnerID:
			candidate.Selected = true
			candidate.Reason = "smallest valid executor for " + capability + " (deterministic-first, smallest-parameter-count ranking)"
		case winnerID != "":
			candidate.Reason = "not selected: ranked below " + winnerID
		default:
			candidate.Reason = "not selected: " + rejection
		}
		out = append(out, candidate)
	}
	return out
}

func (k *kitRegistry) Resolve(goal kit.Goal, planID string, maxParallel int) (kit.Resolution, error) {
	planned, err := k.registry.ResolveGoal(tlaloque.CapabilityGoal{
		Capability:          goal.Capability,
		PreferDeterministic: goal.PreferDeterministic,
		MaxParameters:       goal.MaxParameters,
		AvailableProducts:   goal.AvailableProducts,
	}, planID, maxParallel)
	if err != nil {
		return kit.Resolution{}, fmt.Errorf("tlalocregistry: resolve %s: %w", goal.Capability, err)
	}

	resolution := kit.Resolution{Goal: goal, PlanID: planned.Plan.ID, Candidates: map[string][]kit.Candidate{}}
	for _, node := range planned.Plan.Nodes {
		resolution.Nodes = append(resolution.Nodes, kit.PlanNode{
			ID: node.ID, Capability: node.Capability, WorkerID: node.WorkerID,
			DependsOn: append([]string(nil), node.DependsOn...),
		})
		if _, seen := resolution.Candidates[node.Capability]; !seen {
			resolution.Candidates[node.Capability] = k.Candidates(node.Capability, goal)
		}
	}
	for _, descriptor := range planned.Selected {
		resolution.Selected = append(resolution.Selected, k.toPublicDescriptor(descriptor))
	}
	return resolution, nil
}

func (k *kitRegistry) Execute(ctx context.Context, req kit.ExecutionRequest) (kit.ExecutionResult, error) {
	worker, ok := k.registry.Get(req.WorkerID)
	if !ok {
		return kit.ExecutionResult{}, fmt.Errorf("tlalocregistry: worker %q is not registered", req.WorkerID)
	}
	descriptor := worker.Descriptor()
	if capability := strings.ToUpper(strings.TrimSpace(req.Capability)); capability != "" && descriptor.Capability != capability {
		return kit.ExecutionResult{}, fmt.Errorf("tlalocregistry: worker %q serves %s, not %s", req.WorkerID, descriptor.Capability, capability)
	}

	snapshot := snapshotFromObservations(req.TaskID, req.PriorObservations)
	resp, err := worker.Execute(ctx, tlaloque.CapabilityRequest{
		TaskID: req.TaskID, NodeID: req.NodeID, Input: req.Input, Blackboard: snapshot,
	})
	if err != nil {
		return kit.ExecutionResult{}, fmt.Errorf("tlalocregistry: execute %s on %s: %w", req.Capability, req.WorkerID, err)
	}

	out := kit.ExecutionResult{
		WorkerID: resp.WorkerID, Output: resp.Output, Confidence: resp.Confidence, Notes: resp.Notes,
	}
	if resp.Usage != nil {
		out.Usage = &kit.Usage{
			PromptTokens: resp.Usage.TokensIn, CompletionTokens: resp.Usage.TokensOut, ModelCalls: resp.Usage.UpstreamCalls,
		}
	}
	recordedAt := time.Now().UTC().Format(time.RFC3339Nano)
	for _, obs := range resp.Observations {
		out.Observations = append(out.Observations, kit.Observation{
			Producer: resp.WorkerID, Capability: descriptor.Capability, Key: obs.Key, Value: obs.Value,
			Kind:   observationKind(descriptor.Capability, obs.Key),
			Status: factStatus(obs), Confidence: obs.Confidence,
			References: append([]string(nil), obs.References...), Provenance: obs.Provenance,
			ProfileVersion: k.parrotWorkers[resp.WorkerID], RecordedAt: recordedAt,
		})
	}
	return out, nil
}

func snapshotFromObservations(taskID string, prior []kit.Observation) *blackboard.Snapshot {
	if len(prior) == 0 {
		return &blackboard.Snapshot{Schema: blackboard.SnapshotSchema, RunID: taskID}
	}
	entries := make([]blackboard.Entry, 0, len(prior))
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	for index, obs := range prior {
		entryType := blackboard.EntryObservation
		if strings.EqualFold(obs.Kind, "FACT") || strings.HasPrefix(strings.ToLower(obs.Key), "fact.") {
			entryType = blackboard.EntryFact
		}
		producer := strings.TrimSpace(obs.Producer)
		if producer == "" {
			producer = "UNKNOWN"
		}
		recordedAt := strings.TrimSpace(obs.RecordedAt)
		if recordedAt == "" {
			recordedAt = base.Add(time.Duration(index) * time.Second).Format(time.RFC3339Nano)
		}
		entry := blackboard.Entry{
			Schema: blackboard.EntrySchema, Type: entryType, RunID: taskID, TaskID: taskID,
			NodeID:   firstNonEmpty(obs.Provenance["node_id"], obs.Key, "node"),
			WorkerID: producer, Key: obs.Key, Value: obs.Value, Confidence: obs.Confidence,
			References: append([]string(nil), obs.References...), Provenance: obs.Provenance,
		}
		if id, err := blackboard.ContentID(entry); err == nil {
			entry.ID = id
		}
		entry.RecordedAt = recordedAt
		entries = append(entries, entry)
	}
	return &blackboard.Snapshot{Schema: blackboard.SnapshotSchema, RunID: taskID, Entries: entries}
}

func observationKind(capability, key string) string {
	if strings.Contains(strings.ToUpper(capability), "VERIFY") && strings.HasPrefix(strings.ToLower(key), "fact.") {
		return "FACT"
	}
	return "OBSERVATION"
}

func factStatus(obs blackboard.Observation) string {
	if !strings.HasPrefix(strings.ToLower(obs.Key), "fact.") {
		return ""
	}
	var fact blackboard.Fact
	if err := json.Unmarshal(obs.Value, &fact); err != nil {
		return ""
	}
	return fact.Status
}

func (k *kitRegistry) toPublicDescriptor(descriptor tlaloque.CapabilityDescriptor) kit.Descriptor {
	return kit.Descriptor{
		ID: descriptor.ID, Capability: descriptor.Capability, Engine: toEngineKind(descriptor),
		Deterministic: descriptor.Deterministic, ParameterCount: descriptor.ParameterCount,
		InputSchema: descriptor.InputSchema, OutputSchema: descriptor.OutputSchema,
		Dependencies: append([]string(nil), descriptor.Dependencies...),
		ProfileRef:   k.parrotWorkers[descriptor.ID],
	}
}

func toEngineKind(descriptor tlaloque.CapabilityDescriptor) kit.EngineKind {
	switch descriptor.Engine {
	case tlaloque.EngineDeterministic:
		return kit.EngineDeterministic
	case tlaloque.EngineModel:
		return kit.EngineGenerative
	case tlaloque.EngineProcess:
		return kit.EngineSpecialist
	default:
		if descriptor.Deterministic {
			return kit.EngineDeterministic
		}
		return kit.EngineGenerative
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
