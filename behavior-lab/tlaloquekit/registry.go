package tlaloquekit

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

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
}

// BuildQualifiedRegistry constructs the registry Tlaloc publishes to a
// consuming runtime. It wires the internal tlaloque.Registry, installs the
// CapabilityProfile-aware CapabilityRouter, and registers the qualified
// Tlaloque set behind the public QualifiedRegistry contract.
func BuildQualifiedRegistry(cfg Config) (QualifiedRegistry, error) {
	registry := tlaloque.NewRegistry()

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
				return nil, fmt.Errorf("tlaloquekit: register %s: %w", worker.Descriptor().ID, err)
			}
		}
	}

	// The router keeps the registry's deterministic-first, smallest-first
	// ranking and adds the CapabilityProfile veto. With no generative
	// Tlaloque registered the Profiles map is empty and the router is a
	// pass-through, but it is installed unconditionally so routing
	// behaviour does not change when Parrot is later added.
	exocortex.CapabilityRouter{Profiles: map[string]exocortex.CapabilityProfile{}}.Install(registry)

	return &kitRegistry{registry: registry}, nil
}

type kitRegistry struct {
	registry *tlaloque.Registry
}

func (k *kitRegistry) ParrotProfileID() string   { return "" }
func (k *kitRegistry) ParrotProfileHash() string { return "" }

func (k *kitRegistry) Capabilities() []Descriptor {
	internal := k.registry.Descriptors()
	out := make([]Descriptor, 0, len(internal))
	for _, d := range internal {
		out = append(out, toPublicDescriptor(d))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (k *kitRegistry) Candidates(capability string, goal Goal) []Candidate {
	capability = strings.ToUpper(strings.TrimSpace(capability))
	var matching []tlaloque.CapabilityDescriptor
	for _, d := range k.registry.Descriptors() {
		if d.Capability == capability {
			matching = append(matching, d)
		}
	}
	sort.Slice(matching, func(i, j int) bool { return matching[i].ID < matching[j].ID })

	req := tlaloque.SelectionRequest{
		Capability:          capability,
		PreferDeterministic: goal.PreferDeterministic,
		MaxParameters:       goal.MaxParameters,
	}
	result := k.registry.SelectResult(req)
	winnerID := ""
	rejection := ""
	if result.OK() {
		winnerID = result.Value.Descriptor().ID
	} else if len(result.Diagnostics) > 0 {
		rejection = result.Diagnostics[0].Code + ": " + result.Diagnostics[0].Message
	} else {
		rejection = string(result.Code)
	}

	out := make([]Candidate, 0, len(matching))
	for _, d := range matching {
		c := Candidate{Descriptor: toPublicDescriptor(d)}
		switch {
		case d.ID == winnerID:
			c.Selected = true
			c.Reason = "smallest valid executor for " + capability + " (deterministic-first, smallest-parameter-count ranking)"
		case winnerID != "":
			c.Reason = "not selected: ranked below " + winnerID
		default:
			c.Reason = "not selected: " + rejection
		}
		out = append(out, c)
	}
	return out
}

func (k *kitRegistry) Resolve(goal Goal, planID string, maxParallel int) (Resolution, error) {
	planned, err := k.registry.ResolveGoal(tlaloque.CapabilityGoal{
		Capability:          goal.Capability,
		PreferDeterministic: goal.PreferDeterministic,
		MaxParameters:       goal.MaxParameters,
		AvailableProducts:   goal.AvailableProducts,
	}, planID, maxParallel)
	if err != nil {
		return Resolution{}, fmt.Errorf("tlaloquekit: resolve %s: %w", goal.Capability, err)
	}

	resolution := Resolution{
		Goal:       goal,
		PlanID:     planned.Plan.ID,
		Candidates: map[string][]Candidate{},
	}
	for _, node := range planned.Plan.Nodes {
		resolution.Nodes = append(resolution.Nodes, PlanNode{
			ID:         node.ID,
			Capability: node.Capability,
			WorkerID:   node.WorkerID,
			DependsOn:  append([]string(nil), node.DependsOn...),
		})
		if _, seen := resolution.Candidates[node.Capability]; !seen {
			resolution.Candidates[node.Capability] = k.Candidates(node.Capability, goal)
		}
	}
	for _, d := range planned.Selected {
		resolution.Selected = append(resolution.Selected, toPublicDescriptor(d))
	}
	return resolution, nil
}

func (k *kitRegistry) Execute(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	worker, ok := k.registry.Get(req.WorkerID)
	if !ok {
		return ExecutionResult{}, fmt.Errorf("tlaloquekit: worker %q is not registered", req.WorkerID)
	}
	desc := worker.Descriptor()
	if capability := strings.ToUpper(strings.TrimSpace(req.Capability)); capability != "" && desc.Capability != capability {
		return ExecutionResult{}, fmt.Errorf("tlaloquekit: worker %q serves %s, not %s", req.WorkerID, desc.Capability, capability)
	}

	snapshot := snapshotFromObservations(req.TaskID, req.PriorObservations)
	resp, err := worker.Execute(ctx, tlaloque.CapabilityRequest{
		TaskID:     req.TaskID,
		NodeID:     req.NodeID,
		Input:      req.Input,
		Blackboard: snapshot,
	})
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("tlaloquekit: execute %s on %s: %w", req.Capability, req.WorkerID, err)
	}

	out := ExecutionResult{
		WorkerID:   resp.WorkerID,
		Output:     resp.Output,
		Confidence: resp.Confidence,
		Notes:      resp.Notes,
	}
	if resp.Usage != nil {
		out.Usage = &Usage{
			PromptTokens:     resp.Usage.TokensIn,
			CompletionTokens: resp.Usage.TokensOut,
			ModelCalls:       resp.Usage.UpstreamCalls,
		}
	}
	recordedAt := time.Now().UTC().Format(time.RFC3339Nano)
	for _, obs := range resp.Observations {
		out.Observations = append(out.Observations, Observation{
			Producer:   resp.WorkerID,
			Capability: desc.Capability,
			Key:        obs.Key,
			Value:      obs.Value,
			Kind:       observationKind(desc.Capability, obs.Key),
			Status:     factStatus(obs),
			Confidence: obs.Confidence,
			References: append([]string(nil), obs.References...),
			Provenance: obs.Provenance,
			RecordedAt: recordedAt,
		})
	}
	return out, nil
}

// snapshotFromObservations rebuilds a blackboard.Snapshot from the prior
// observations the consuming runtime forwarded, so a stateful Tlaloque
// (Verify) can read them. Each entry gets its real content id and an
// ordered RecordedAt, matching how the internal store would have persisted
// it.
func snapshotFromObservations(taskID string, prior []Observation) *blackboard.Snapshot {
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
			Schema:     blackboard.EntrySchema,
			Type:       entryType,
			RunID:      taskID,
			TaskID:     taskID,
			NodeID:     strings.TrimSpace(firstNonEmpty(obs.Provenance["node_id"], obs.Key, "node")),
			WorkerID:   producer,
			Key:        obs.Key,
			Value:      obs.Value,
			Confidence: obs.Confidence,
			References: append([]string(nil), obs.References...),
			Provenance: obs.Provenance,
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

func toPublicDescriptor(d tlaloque.CapabilityDescriptor) Descriptor {
	return Descriptor{
		ID:             d.ID,
		Capability:     d.Capability,
		Engine:         toEngineKind(d),
		Deterministic:  d.Deterministic,
		ParameterCount: d.ParameterCount,
		InputSchema:    d.InputSchema,
		OutputSchema:   d.OutputSchema,
		Dependencies:   append([]string(nil), d.Dependencies...),
	}
}

func toEngineKind(d tlaloque.CapabilityDescriptor) EngineKind {
	switch d.Engine {
	case tlaloque.EngineDeterministic:
		return EngineDeterministic
	case tlaloque.EngineModel:
		return EngineGenerative
	case tlaloque.EngineProcess:
		return EngineSpecialist
	default:
		if d.Deterministic {
			return EngineDeterministic
		}
		return EngineGenerative
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
